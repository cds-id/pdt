package sit

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"testing"
	"time"

	"github.com/cds-id/pdt/backend/internal/services/github"
)

func skipIfNoGitHub(t *testing.T) {
	if getEnv("SIT_GITHUB_TOKEN") == "" {
		t.Skip("SIT_GITHUB_TOKEN not set, skipping GitHub SIT tests")
	}
}

// =============================================================================
// Test 1: Direct GitHub Service — no DB needed
// =============================================================================

func TestGitHubService_ValidateAccess(t *testing.T) {
	skipIfNoGitHub(t)

	client := github.New()
	// Parse repo from env
	owner, name := parseGitHubRepo()
	err := client.ValidateAccess(owner, name, getEnv("SIT_GITHUB_TOKEN"))
	if err != nil {
		t.Fatalf("ValidateAccess failed: %v", err)
	}
	t.Log("GitHub access validated successfully")
}

func TestGitHubService_FetchCommits(t *testing.T) {
	skipIfNoGitHub(t)

	client := github.New()
	since := time.Now().AddDate(0, 0, -30)

	owner, name := parseGitHubRepo()
	commits, err := client.FetchCommits(owner, name, getEnv("SIT_GITHUB_TOKEN"), since)
	if err != nil {
		t.Fatalf("FetchCommits failed: %v", err)
	}

	t.Logf("Fetched %d commits from last 30 days", len(commits))
	for i, c := range commits {
		if i >= 5 {
			t.Logf("  ... and %d more", len(commits)-5)
			break
		}
		jiraKey := "none"
		if k := extractJiraKeyForTest(c.Message); k != "" {
			jiraKey = k
		}
		t.Logf("  [%s] %s — %s (jira: %s, branch: %s)", c.SHA[:8], c.Date.Format("2006-01-02"), truncate(c.Message, 40), jiraKey, c.Branch)
	}
}

// =============================================================================
// Test 2: Full GitHub API Flow — requires MySQL
// =============================================================================

func TestFullGitHubFlow(t *testing.T) {
	skipIfNoGitHub(t)

	router, db, _, _ := setupRouter(t)

	t.Cleanup(func() {
		db.Exec("DELETE FROM commit_card_links WHERE commit_id IN (SELECT id FROM commits WHERE repo_id IN (SELECT id FROM repositories WHERE user_id IN (SELECT id FROM users WHERE email = ?)))", "sit-github@test.local")
		db.Exec("DELETE FROM commits WHERE repo_id IN (SELECT id FROM repositories WHERE user_id IN (SELECT id FROM users WHERE email = ?))", "sit-github@test.local")
		db.Exec("DELETE FROM repositories WHERE user_id IN (SELECT id FROM users WHERE email = ?)", "sit-github@test.local")
		db.Exec("DELETE FROM users WHERE email = ?", "sit-github@test.local")
	})

	var token string

	// --- Step 1: Register ---
	t.Run("1_Register", func(t *testing.T) {
		body := jsonBody(map[string]string{
			"email":    "sit-github@test.local",
			"password": "testpass123",
		})
		w := doRequest(router, "POST", "/api/auth/register", body, "")
		if w.Code != http.StatusCreated {
			t.Fatalf("register failed: %d — %s", w.Code, w.Body.String())
		}

		var resp map[string]interface{}
		json.Unmarshal(w.Body.Bytes(), &resp)
		token = resp["token"].(string)
		t.Logf("Registered, got JWT")
	})

	// --- Step 2: Configure GitHub token ---
	t.Run("2_ConfigureGitHub", func(t *testing.T) {
		body := jsonBody(map[string]string{
			"github_token": getEnv("SIT_GITHUB_TOKEN"),
		})
		w := doRequest(router, "PUT", "/api/user/profile", body, token)
		if w.Code != http.StatusOK {
			t.Fatalf("update profile failed: %d — %s", w.Code, w.Body.String())
		}
		t.Log("GitHub token configured")
	})

	// --- Step 3: Verify profile ---
	t.Run("3_VerifyProfile", func(t *testing.T) {
		w := doRequest(router, "GET", "/api/user/profile", nil, token)
		if w.Code != http.StatusOK {
			t.Fatalf("get profile failed: %d — %s", w.Code, w.Body.String())
		}

		var resp map[string]interface{}
		json.Unmarshal(w.Body.Bytes(), &resp)
		if resp["has_github_token"] != true {
			t.Fatal("expected has_github_token=true")
		}
		t.Logf("Profile OK: has_github=%v", resp["has_github_token"])
	})

	// --- Step 4: Add GitHub repo ---
	var repoID float64
	t.Run("4_AddRepo", func(t *testing.T) {
		body := jsonBody(map[string]string{
			"url": getEnv("SIT_GITHUB_REPO"),
		})
		w := doRequest(router, "POST", "/api/repos", body, token)
		if w.Code != http.StatusCreated {
			t.Fatalf("add repo failed: %d — %s", w.Code, w.Body.String())
		}

		var resp map[string]interface{}
		json.Unmarshal(w.Body.Bytes(), &resp)
		repoID = resp["id"].(float64)
		t.Logf("Added repo: id=%v, owner=%v, name=%v, provider=%v",
			resp["id"], resp["owner"], resp["name"], resp["provider"])
	})

	// --- Step 5: Sync commits ---
	t.Run("5_SyncCommits", func(t *testing.T) {
		w := doRequest(router, "POST", "/api/sync/commits", nil, token)
		if w.Code != http.StatusOK {
			t.Fatalf("sync failed: %d — %s", w.Code, w.Body.String())
		}

		var resp map[string]interface{}
		json.Unmarshal(w.Body.Bytes(), &resp)

		results := resp["results"].([]interface{})
		for _, r := range results {
			result := r.(map[string]interface{})
			t.Logf("Sync result: repo=%v, new=%v, total=%v, error=%v",
				result["repo_name"], result["new_commits"], result["total_fetched"], result["error"])
			if errMsg, ok := result["error"].(string); ok && errMsg != "" {
				t.Fatalf("sync error: %s", errMsg)
			}
		}
	})

	// --- Step 5b: Check sync status ---
	t.Run("5b_SyncStatus", func(t *testing.T) {
		w := doRequest(router, "GET", "/api/sync/status", nil, token)
		if w.Code != http.StatusOK {
			t.Fatalf("sync status failed: %d — %s", w.Code, w.Body.String())
		}

		var resp map[string]interface{}
		json.Unmarshal(w.Body.Bytes(), &resp)
		t.Logf("Sync status: commits=%v, jira=%v", resp["commits"], resp["jira"])
	})

	// --- Step 6: List commits ---
	t.Run("6_ListCommits", func(t *testing.T) {
		w := doRequest(router, "GET", "/api/commits", nil, token)
		if w.Code != http.StatusOK {
			t.Fatalf("list commits failed: %d — %s", w.Code, w.Body.String())
		}

		var commits []map[string]interface{}
		json.Unmarshal(w.Body.Bytes(), &commits)
		t.Logf("Total commits: %d", len(commits))

		linked := 0
		for _, c := range commits {
			if c["has_link"] == true {
				linked++
			}
		}
		t.Logf("With Jira refs: %d, Without: %d", linked, len(commits)-linked)

		for i, c := range commits {
			if i >= 5 {
				t.Logf("  ... and %d more", len(commits)-5)
				break
			}
			t.Logf("  [%s] %s — %s (jira: %v, linked: %v, branch: %v)",
				c["sha"].(string)[:8], c["date"], truncate(fmt.Sprint(c["message"]), 40),
				c["jira_card_key"], c["has_link"], c["branch"])
		}
	})

	// --- Step 7: Missing Jira refs ---
	t.Run("7_MissingJiraRefs", func(t *testing.T) {
		w := doRequest(router, "GET", "/api/commits/missing", nil, token)
		if w.Code != http.StatusOK {
			t.Fatalf("missing failed: %d — %s", w.Code, w.Body.String())
		}

		var commits []map[string]interface{}
		json.Unmarshal(w.Body.Bytes(), &commits)
		t.Logf("Commits missing Jira refs: %d", len(commits))
		for i, c := range commits {
			if i >= 5 {
				t.Logf("  ... and %d more", len(commits)-5)
				break
			}
			t.Logf("  [%s] %s", c["sha"].(string)[:8], truncate(fmt.Sprint(c["message"]), 60))
		}
	})

	// --- Step 8: Filter by repo ---
	t.Run("8_FilterByRepo", func(t *testing.T) {
		url := fmt.Sprintf("/api/commits?repo_id=%d", int(repoID))
		w := doRequest(router, "GET", url, nil, token)
		if w.Code != http.StatusOK {
			t.Fatalf("filter commits failed: %d — %s", w.Code, w.Body.String())
		}

		var commits []map[string]interface{}
		json.Unmarshal(w.Body.Bytes(), &commits)
		t.Logf("Commits for repo %d: %d", int(repoID), len(commits))
	})

	// --- Step 9: Cleanup ---
	t.Run("9_DeleteRepo", func(t *testing.T) {
		url := fmt.Sprintf("/api/repos/%d", int(repoID))
		w := doRequest(router, "DELETE", url, nil, token)
		if w.Code != http.StatusOK {
			t.Fatalf("delete repo failed: %d — %s", w.Code, w.Body.String())
		}
		t.Log("Repo deleted successfully")
	})
}

func parseGitHubRepo() (string, string) {
	repoURL := getEnv("SIT_GITHUB_REPO")
	// https://github.com/cds-id/kendle -> owner=cds-id, name=kendle
	parts := strings.Split(strings.Trim(strings.TrimPrefix(strings.TrimPrefix(repoURL, "https://"), "github.com/"), "/"), "/")
	if len(parts) >= 2 {
		return parts[0], parts[1]
	}
	return "", ""
}

func extractJiraKeyForTest(msg string) string {
	import_re := `([A-Z][A-Z0-9]+-\d+)`
	_ = import_re
	// Simple inline check
	for i := 0; i < len(msg)-2; i++ {
		if msg[i] >= 'A' && msg[i] <= 'Z' {
			j := i + 1
			for j < len(msg) && ((msg[j] >= 'A' && msg[j] <= 'Z') || (msg[j] >= '0' && msg[j] <= '9')) {
				j++
			}
			if j < len(msg) && msg[j] == '-' {
				k := j + 1
				for k < len(msg) && msg[k] >= '0' && msg[k] <= '9' {
					k++
				}
				if k > j+1 {
					return msg[i:k]
				}
			}
		}
	}
	return ""
}

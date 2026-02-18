package sit

import (
	"encoding/json"
	"fmt"
	"net/http"
	"testing"

	"github.com/cds-id/pdt/backend/internal/services/jira"
)

func skipIfNoJira(t *testing.T) {
	if getEnv("SIT_JIRA_TOKEN") == "" {
		t.Skip("SIT_JIRA_TOKEN not set, skipping Jira SIT tests")
	}
}

func jiraClient(t *testing.T) *jira.Client {
	t.Helper()
	return jira.New(
		getEnv("SIT_JIRA_WORKSPACE"),
		getEnv("SIT_JIRA_EMAIL"),
		getEnv("SIT_JIRA_TOKEN"),
	)
}

// =============================================================================
// Test 1: Direct Jira Service — no DB needed
// =============================================================================

func TestJiraService_Validate(t *testing.T) {
	skipIfNoJira(t)

	client := jiraClient(t)
	if err := client.Validate(); err != nil {
		t.Fatalf("Jira validation failed: %v", err)
	}
	t.Log("Jira credentials validated successfully")
}

func TestJiraService_FetchBoards(t *testing.T) {
	skipIfNoJira(t)

	client := jiraClient(t)
	boards, err := client.FetchBoards()
	if err != nil {
		t.Fatalf("FetchBoards failed: %v", err)
	}

	t.Logf("Found %d boards: %v", len(boards), boards)
}

func TestJiraService_FetchSprints(t *testing.T) {
	skipIfNoJira(t)

	client := jiraClient(t)

	boards, err := client.FetchBoards()
	if err != nil {
		t.Fatalf("FetchBoards failed: %v", err)
	}
	if len(boards) == 0 {
		t.Skip("no boards found")
	}

	for _, boardID := range boards {
		sprints, err := client.FetchSprints(boardID)
		if err != nil {
			t.Logf("Board %d: error fetching sprints: %v", boardID, err)
			continue
		}

		t.Logf("Board %d: %d sprints", boardID, len(sprints))
		for _, s := range sprints {
			start := "n/a"
			end := "n/a"
			if s.StartDate != nil {
				start = s.StartDate.Format("2006-01-02")
			}
			if s.EndDate != nil {
				end = s.EndDate.Format("2006-01-02")
			}
			t.Logf("  Sprint %d: %s [%s] (%s → %s)", s.ID, s.Name, s.State, start, end)
		}
	}
}

func TestJiraService_FetchActiveSprintIssues(t *testing.T) {
	skipIfNoJira(t)

	client := jiraClient(t)

	boards, err := client.FetchBoards()
	if err != nil {
		t.Fatalf("FetchBoards failed: %v", err)
	}

	var activeSprintID int
	for _, boardID := range boards {
		sprints, err := client.FetchSprints(boardID)
		if err != nil {
			continue
		}
		for _, s := range sprints {
			if s.State == "active" {
				activeSprintID = s.ID
				t.Logf("Found active sprint: %d — %s", s.ID, s.Name)
				break
			}
		}
		if activeSprintID > 0 {
			break
		}
	}

	if activeSprintID == 0 {
		t.Skip("no active sprint found")
	}

	cards, err := client.FetchSprintIssues(activeSprintID)
	if err != nil {
		t.Fatalf("FetchSprintIssues failed: %v", err)
	}

	t.Logf("Active sprint has %d cards:", len(cards))
	for i, c := range cards {
		if i >= 10 {
			t.Logf("  ... and %d more", len(cards)-10)
			break
		}
		t.Logf("  [%s] %s — %s (assignee: %s)", c.Key, c.Status, truncate(c.Summary, 50), c.Assignee)
	}
}

// =============================================================================
// Test 2: Full Jira API Flow — requires MySQL
// =============================================================================

func TestFullJiraFlow(t *testing.T) {
	skipIfNoJira(t)
	skipIfNoGitLab(t)

	router, db, _, _ := setupRouter(t)

	t.Cleanup(func() {
		db.Exec("DELETE FROM commit_card_links")
		db.Exec("DELETE FROM jira_cards WHERE user_id IN (SELECT id FROM users WHERE email = ?)", "sit-jira@test.local")
		db.Exec("DELETE FROM sprints WHERE user_id IN (SELECT id FROM users WHERE email = ?)", "sit-jira@test.local")
		db.Exec("DELETE FROM commits WHERE repo_id IN (SELECT id FROM repositories WHERE user_id IN (SELECT id FROM users WHERE email = ?))", "sit-jira@test.local")
		db.Exec("DELETE FROM repositories WHERE user_id IN (SELECT id FROM users WHERE email = ?)", "sit-jira@test.local")
		db.Exec("DELETE FROM users WHERE email = ?", "sit-jira@test.local")
	})

	var token string

	// --- Step 1: Register ---
	t.Run("1_Register", func(t *testing.T) {
		body := jsonBody(map[string]string{
			"email":    "sit-jira@test.local",
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

	// --- Step 2: Configure GitLab + Jira credentials ---
	t.Run("2_ConfigureCredentials", func(t *testing.T) {
		body := jsonBody(map[string]string{
			"gitlab_token":  getEnv("SIT_GITLAB_TOKEN"),
			"gitlab_url":    getEnv("SIT_GITLAB_URL"),
			"jira_email":    getEnv("SIT_JIRA_EMAIL"),
			"jira_token":    getEnv("SIT_JIRA_TOKEN"),
			"jira_workspace": getEnv("SIT_JIRA_WORKSPACE"),
			"jira_username":  "Indra Gunanda",
		})
		w := doRequest(router, "PUT", "/api/user/profile", body, token)
		if w.Code != http.StatusOK {
			t.Fatalf("update profile failed: %d — %s", w.Code, w.Body.String())
		}
		t.Log("GitLab + Jira credentials configured")
	})

	// --- Step 3: Verify profile ---
	t.Run("3_VerifyProfile", func(t *testing.T) {
		w := doRequest(router, "GET", "/api/user/profile", nil, token)
		if w.Code != http.StatusOK {
			t.Fatalf("get profile failed: %d — %s", w.Code, w.Body.String())
		}

		var resp map[string]interface{}
		json.Unmarshal(w.Body.Bytes(), &resp)

		if resp["has_gitlab_token"] != true {
			t.Fatal("expected has_gitlab_token=true")
		}
		if resp["has_jira_token"] != true {
			t.Fatal("expected has_jira_token=true")
		}
		t.Logf("Profile OK: gitlab=%v, jira=%v, workspace=%v",
			resp["has_gitlab_token"], resp["has_jira_token"], resp["jira_workspace"])
	})

	// --- Step 4: Add GitLab repo + sync commits ---
	t.Run("4_AddRepoAndSync", func(t *testing.T) {
		body := jsonBody(map[string]string{
			"url": getEnv("SIT_GITLAB_REPO"),
		})
		w := doRequest(router, "POST", "/api/repos", body, token)
		if w.Code != http.StatusCreated {
			t.Fatalf("add repo failed: %d — %s", w.Code, w.Body.String())
		}
		t.Log("Repo added")

		w = doRequest(router, "POST", "/api/sync/commits", nil, token)
		if w.Code != http.StatusOK {
			t.Fatalf("sync failed: %d — %s", w.Code, w.Body.String())
		}

		var resp map[string]interface{}
		json.Unmarshal(w.Body.Bytes(), &resp)
		results := resp["results"].([]interface{})
		result := results[0].(map[string]interface{})
		t.Logf("Synced: %v new commits, %v total", result["new_commits"], result["total_fetched"])
	})

	// --- Step 5: List Jira sprints ---
	t.Run("5_ListSprints", func(t *testing.T) {
		w := doRequest(router, "GET", "/api/jira/sprints", nil, token)
		if w.Code != http.StatusOK {
			t.Fatalf("list sprints failed: %d — %s", w.Code, w.Body.String())
		}

		var sprints []map[string]interface{}
		json.Unmarshal(w.Body.Bytes(), &sprints)
		t.Logf("Found %d sprints:", len(sprints))
		for i, s := range sprints {
			if i >= 5 {
				t.Logf("  ... and %d more", len(sprints)-5)
				break
			}
			t.Logf("  [%v] %s — %s", s["jira_sprint_id"], s["name"], s["state"])
		}
	})

	// --- Step 6: Get active sprint ---
	var activeSprintID float64
	t.Run("6_ActiveSprint", func(t *testing.T) {
		w := doRequest(router, "GET", "/api/jira/active-sprint", nil, token)
		if w.Code == http.StatusNotFound {
			t.Skip("no active sprint found")
		}
		if w.Code != http.StatusOK {
			t.Fatalf("active sprint failed: %d — %s", w.Code, w.Body.String())
		}

		var sprint map[string]interface{}
		json.Unmarshal(w.Body.Bytes(), &sprint)
		activeSprintID = sprint["id"].(float64)
		t.Logf("Active sprint: id=%v, name=%s, state=%s", sprint["id"], sprint["name"], sprint["state"])
	})

	// --- Step 7: List cards in active sprint ---
	t.Run("7_ListCards", func(t *testing.T) {
		if activeSprintID == 0 {
			t.Skip("no active sprint")
		}

		url := fmt.Sprintf("/api/jira/cards?sprint_id=%d", int(activeSprintID))
		w := doRequest(router, "GET", url, nil, token)
		if w.Code != http.StatusOK {
			t.Fatalf("list cards failed: %d — %s", w.Code, w.Body.String())
		}

		var cards []map[string]interface{}
		json.Unmarshal(w.Body.Bytes(), &cards)
		t.Logf("Cards in active sprint: %d", len(cards))
		for i, c := range cards {
			if i >= 10 {
				t.Logf("  ... and %d more", len(cards)-10)
				break
			}
			t.Logf("  [%s] %s — %s (assignee: %s)", c["key"], c["status"], truncate(fmt.Sprint(c["summary"]), 50), c["assignee"])
		}
	})

	// --- Step 8: Get a specific card with parent/subtasks ---
	t.Run("8_CardWithHierarchy", func(t *testing.T) {
		// Find a commit that has a Jira key
		w := doRequest(router, "GET", "/api/commits?has_link=true", nil, token)
		if w.Code != http.StatusOK {
			t.Fatalf("list commits failed: %d — %s", w.Code, w.Body.String())
		}

		var commits []map[string]interface{}
		json.Unmarshal(w.Body.Bytes(), &commits)

		if len(commits) == 0 {
			t.Skip("no linked commits found")
		}

		cardKey := commits[0]["jira_card_key"].(string)
		t.Logf("Looking up card: %s", cardKey)

		w = doRequest(router, "GET", "/api/jira/cards/"+cardKey, nil, token)
		if w.Code != http.StatusOK {
			t.Fatalf("get card failed: %d — %s", w.Code, w.Body.String())
		}

		var resp map[string]interface{}
		json.Unmarshal(w.Body.Bytes(), &resp)

		// Card details from Jira
		if resp["summary"] != nil {
			t.Logf("Card: %s — %s [%s] (type: %s, assignee: %s)",
				cardKey, resp["summary"], resp["status"], resp["issue_type"], resp["assignee"])
		}

		// Parent
		if resp["parent"] != nil {
			parent := resp["parent"].(map[string]interface{})
			t.Logf("Parent: %s — %s [%s] (type: %s)", parent["key"], parent["summary"], parent["status"], parent["type"])
		} else {
			t.Log("Parent: none")
		}

		// Own commits
		cardCommits := resp["commits"].([]interface{})
		t.Logf("Own commits: %d", len(cardCommits))
		for i, c := range cardCommits {
			commit := c.(map[string]interface{})
			if i >= 3 {
				t.Logf("  ... and %d more", len(cardCommits)-3)
				break
			}
			t.Logf("  [%s] %s", commit["sha"].(string)[:8], truncate(fmt.Sprint(commit["message"]), 60))
		}

		// Subtasks
		if resp["subtasks"] != nil {
			subtasks := resp["subtasks"].([]interface{})
			t.Logf("Subtasks: %d", len(subtasks))
			for _, s := range subtasks {
				st := s.(map[string]interface{})
				stCommits := st["commits"].([]interface{})
				t.Logf("  [%s] %s [%s] — %d commits", st["key"], truncate(fmt.Sprint(st["summary"]), 40), st["status"], len(stCommits))
			}
		}

		// Changelog (transition + change history)
		if resp["changelog"] != nil {
			changelog := resp["changelog"].([]interface{})
			t.Logf("Change history: %d entries", len(changelog))
			for _, entry := range changelog {
				h := entry.(map[string]interface{})
				items := h["items"].([]interface{})
				created := truncate(fmt.Sprint(h["created"]), 16)
				for _, item := range items {
					it := item.(map[string]interface{})
					t.Logf("  [%s] %s: %s changed \"%s\" → \"%s\"",
						created, h["author"], it["field"],
						truncate(fmt.Sprint(it["from_string"]), 30),
						truncate(fmt.Sprint(it["to_string"]), 30))
				}
			}
		}
	})

	// --- Step 9: Cross-reference — commits missing Jira refs ---
	t.Run("9_MissingJiraRefs", func(t *testing.T) {
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
}

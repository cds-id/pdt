package sit

import (
	"encoding/json"
	"fmt"
	"net/http"
	"testing"
)

func TestReportYesterday(t *testing.T) {
	skipIfNoGitLab(t)

	router, db, _, _ := setupRouter(t)

	t.Cleanup(func() {
		db.Exec("DELETE FROM reports WHERE user_id IN (SELECT id FROM users WHERE email = ?)", "sit-yesterday@test.local")
		db.Exec("DELETE FROM report_templates WHERE user_id IN (SELECT id FROM users WHERE email = ?)", "sit-yesterday@test.local")
		db.Exec("DELETE FROM commit_card_links WHERE commit_id IN (SELECT id FROM commits WHERE repo_id IN (SELECT id FROM repositories WHERE user_id IN (SELECT id FROM users WHERE email = ?)))", "sit-yesterday@test.local")
		db.Exec("DELETE FROM commits WHERE repo_id IN (SELECT id FROM repositories WHERE user_id IN (SELECT id FROM users WHERE email = ?))", "sit-yesterday@test.local")
		db.Exec("DELETE FROM repositories WHERE user_id IN (SELECT id FROM users WHERE email = ?)", "sit-yesterday@test.local")
		db.Exec("DELETE FROM users WHERE email = ?", "sit-yesterday@test.local")
	})

	var token string

	// --- Setup: register, configure, sync ---
	t.Run("1_Setup", func(t *testing.T) {
		body := jsonBody(map[string]string{"email": "sit-yesterday@test.local", "password": "testpass123"})
		w := doRequest(router, "POST", "/api/auth/register", body, "")
		if w.Code != http.StatusCreated {
			t.Fatalf("register failed: %d — %s", w.Code, w.Body.String())
		}
		var resp map[string]interface{}
		json.Unmarshal(w.Body.Bytes(), &resp)
		token = resp["token"].(string)

		body = jsonBody(map[string]string{
			"gitlab_token":   getEnv("SIT_GITLAB_TOKEN"),
			"gitlab_url":     getEnv("SIT_GITLAB_URL"),
			"jira_email":     getEnv("SIT_JIRA_EMAIL"),
			"jira_token":     getEnv("SIT_JIRA_TOKEN"),
			"jira_workspace": getEnv("SIT_JIRA_WORKSPACE"),
		})
		doRequest(router, "PUT", "/api/user/profile", body, token)

		body = jsonBody(map[string]string{"url": getEnv("SIT_GITLAB_REPO")})
		w = doRequest(router, "POST", "/api/repos", body, token)
		if w.Code != http.StatusCreated {
			t.Fatalf("add repo failed: %d — %s", w.Code, w.Body.String())
		}

		w = doRequest(router, "POST", "/api/sync/commits", nil, token)
		if w.Code != http.StatusOK {
			t.Fatalf("sync failed: %d — %s", w.Code, w.Body.String())
		}

		var syncResp map[string]interface{}
		json.Unmarshal(w.Body.Bytes(), &syncResp)
		results := syncResp["results"].([]interface{})
		for _, r := range results {
			result := r.(map[string]interface{})
			t.Logf("Synced: %v — %v new commits", result["repo_name"], result["new_commits"])
		}
	})

	// --- Generate report for yesterday (2026-02-18) ---
	t.Run("2_GenerateYesterday", func(t *testing.T) {
		body := jsonBody(map[string]string{"date": "2026-02-18"})
		w := doRequest(router, "POST", "/api/reports/generate", body, token)
		if w.Code != http.StatusCreated && w.Code != http.StatusOK {
			t.Fatalf("generate failed: %d — %s", w.Code, w.Body.String())
		}

		var resp map[string]interface{}
		json.Unmarshal(w.Body.Bytes(), &resp)

		t.Logf("Report ID: %v", resp["id"])
		t.Logf("Title: %v", resp["title"])
		t.Logf("File URL: %v", resp["file_url"])
		t.Logf("\n--- FULL REPORT ---\n%s\n--- END ---", resp["content"])
	})

	// --- Also try a date with no commits ---
	t.Run("3_GenerateEmptyDate", func(t *testing.T) {
		body := jsonBody(map[string]string{"date": "2026-01-01"})
		w := doRequest(router, "POST", "/api/reports/generate", body, token)
		if w.Code != http.StatusCreated && w.Code != http.StatusOK {
			t.Fatalf("generate failed: %d — %s", w.Code, w.Body.String())
		}

		var resp map[string]interface{}
		json.Unmarshal(w.Body.Bytes(), &resp)
		content := fmt.Sprint(resp["content"])
		t.Logf("Empty date report (preview): %s", truncate(content, 200))
	})
}

package sit

import (
	"encoding/json"
	"fmt"
	"net/http"
	"testing"
	"time"
)

func TestFullReportFlow(t *testing.T) {
	skipIfNoGitLab(t)

	router, db, _, _ := setupRouter(t)

	t.Cleanup(func() {
		db.Exec("DELETE FROM reports WHERE user_id IN (SELECT id FROM users WHERE email = ?)", "sit-report@test.local")
		db.Exec("DELETE FROM report_templates WHERE user_id IN (SELECT id FROM users WHERE email = ?)", "sit-report@test.local")
		db.Exec("DELETE FROM commit_card_links WHERE commit_id IN (SELECT id FROM commits WHERE repo_id IN (SELECT id FROM repositories WHERE user_id IN (SELECT id FROM users WHERE email = ?)))", "sit-report@test.local")
		db.Exec("DELETE FROM commits WHERE repo_id IN (SELECT id FROM repositories WHERE user_id IN (SELECT id FROM users WHERE email = ?))", "sit-report@test.local")
		db.Exec("DELETE FROM repositories WHERE user_id IN (SELECT id FROM users WHERE email = ?)", "sit-report@test.local")
		db.Exec("DELETE FROM users WHERE email = ?", "sit-report@test.local")
	})

	var token string

	// --- Step 1: Register and setup ---
	t.Run("1_Setup", func(t *testing.T) {
		body := jsonBody(map[string]string{"email": "sit-report@test.local", "password": "testpass123"})
		w := doRequest(router, "POST", "/api/auth/register", body, "")
		if w.Code != http.StatusCreated {
			t.Fatalf("register failed: %d — %s", w.Code, w.Body.String())
		}
		var resp map[string]interface{}
		json.Unmarshal(w.Body.Bytes(), &resp)
		token = resp["token"].(string)

		// Configure GitLab
		body = jsonBody(map[string]string{
			"gitlab_token": getEnv("SIT_GITLAB_TOKEN"),
			"gitlab_url":   getEnv("SIT_GITLAB_URL"),
		})
		w = doRequest(router, "PUT", "/api/user/profile", body, token)
		if w.Code != http.StatusOK {
			t.Fatalf("profile update failed: %d", w.Code)
		}

		// Add repo
		body = jsonBody(map[string]string{"url": getEnv("SIT_GITLAB_REPO")})
		w = doRequest(router, "POST", "/api/repos", body, token)
		if w.Code != http.StatusCreated {
			t.Fatalf("add repo failed: %d — %s", w.Code, w.Body.String())
		}

		// Sync commits
		w = doRequest(router, "POST", "/api/sync/commits", nil, token)
		if w.Code != http.StatusOK {
			t.Fatalf("sync failed: %d — %s", w.Code, w.Body.String())
		}
		t.Log("Setup complete: registered, configured, synced")
	})

	// --- Step 2: Generate report for today ---
	var reportID float64
	t.Run("2_GenerateReport", func(t *testing.T) {
		today := time.Now().Format("2006-01-02")
		body := jsonBody(map[string]string{"date": today})
		w := doRequest(router, "POST", "/api/reports/generate", body, token)
		if w.Code != http.StatusCreated && w.Code != http.StatusOK {
			t.Fatalf("generate report failed: %d — %s", w.Code, w.Body.String())
		}

		var resp map[string]interface{}
		json.Unmarshal(w.Body.Bytes(), &resp)
		reportID = resp["id"].(float64)
		t.Logf("Generated report id=%v, title=%v, file_url=%v", resp["id"], resp["title"], resp["file_url"])
		content := resp["content"].(string)
		if len(content) > 200 {
			t.Logf("Content preview: %s...", content[:200])
		} else {
			t.Logf("Content: %s", content)
		}
	})

	// --- Step 3: List reports ---
	t.Run("3_ListReports", func(t *testing.T) {
		w := doRequest(router, "GET", "/api/reports", nil, token)
		if w.Code != http.StatusOK {
			t.Fatalf("list reports failed: %d — %s", w.Code, w.Body.String())
		}

		var reports []map[string]interface{}
		json.Unmarshal(w.Body.Bytes(), &reports)
		t.Logf("Total reports: %d", len(reports))
	})

	// --- Step 4: Get specific report ---
	t.Run("4_GetReport", func(t *testing.T) {
		url := fmt.Sprintf("/api/reports/%d", int(reportID))
		w := doRequest(router, "GET", url, nil, token)
		if w.Code != http.StatusOK {
			t.Fatalf("get report failed: %d — %s", w.Code, w.Body.String())
		}
		t.Log("Report retrieved successfully")
	})

	// --- Step 5: Create custom template ---
	var templateID float64
	t.Run("5_CreateTemplate", func(t *testing.T) {
		body := jsonBody(map[string]interface{}{
			"name":       "Custom Format",
			"content":    "# {{.DateFormatted}}\n\nTotal: {{.Stats.TotalCommits}} commits on {{.Stats.TotalCards}} cards\n\n{{range .Cards}}\n## {{.Key}}: {{.Summary}}\n{{range .Commits}}\n- {{.Message}}\n{{end}}\n{{end}}",
			"is_default": true,
		})
		w := doRequest(router, "POST", "/api/reports/templates", body, token)
		if w.Code != http.StatusCreated {
			t.Fatalf("create template failed: %d — %s", w.Code, w.Body.String())
		}

		var resp map[string]interface{}
		json.Unmarshal(w.Body.Bytes(), &resp)
		templateID = resp["id"].(float64)
		t.Logf("Created template id=%v, name=%v, is_default=%v", resp["id"], resp["name"], resp["is_default"])
	})

	// --- Step 6: Preview template ---
	t.Run("6_PreviewTemplate", func(t *testing.T) {
		body := jsonBody(map[string]string{
			"content": "Preview: {{.Stats.TotalCommits}} commits by {{.Author}}",
		})
		w := doRequest(router, "POST", "/api/reports/templates/preview", body, token)
		if w.Code != http.StatusOK {
			t.Fatalf("preview failed: %d — %s", w.Code, w.Body.String())
		}

		var resp map[string]interface{}
		json.Unmarshal(w.Body.Bytes(), &resp)
		t.Logf("Preview: %v", resp["rendered"])
	})

	// --- Step 7: Generate with custom template ---
	t.Run("7_GenerateWithCustom", func(t *testing.T) {
		today := time.Now().Format("2006-01-02")
		tid := uint(templateID)
		body := jsonBody(map[string]interface{}{
			"date":        today,
			"template_id": tid,
		})
		w := doRequest(router, "POST", "/api/reports/generate", body, token)
		if w.Code != http.StatusOK && w.Code != http.StatusCreated {
			t.Fatalf("generate with custom failed: %d — %s", w.Code, w.Body.String())
		}

		var resp map[string]interface{}
		json.Unmarshal(w.Body.Bytes(), &resp)
		content := resp["content"].(string)
		t.Logf("Custom report content preview: %s", truncate(content, 200))
	})

	// --- Step 8: Delete template ---
	t.Run("8_DeleteTemplate", func(t *testing.T) {
		url := fmt.Sprintf("/api/reports/templates/%d", int(templateID))
		w := doRequest(router, "DELETE", url, nil, token)
		if w.Code != http.StatusOK {
			t.Fatalf("delete template failed: %d — %s", w.Code, w.Body.String())
		}
		t.Log("Template deleted")
	})

	// --- Step 9: Delete report ---
	t.Run("9_DeleteReport", func(t *testing.T) {
		url := fmt.Sprintf("/api/reports/%d", int(reportID))
		w := doRequest(router, "DELETE", url, nil, token)
		if w.Code != http.StatusOK {
			t.Fatalf("delete report failed: %d — %s", w.Code, w.Body.String())
		}
		t.Log("Report deleted")
	})
}

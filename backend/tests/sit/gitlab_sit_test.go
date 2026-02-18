package sit

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"
	"time"

	"github.com/cds-id/pdt/backend/internal/config"
	"github.com/cds-id/pdt/backend/internal/crypto"
	"github.com/cds-id/pdt/backend/internal/database"
	"github.com/cds-id/pdt/backend/internal/handlers"
	"github.com/cds-id/pdt/backend/internal/middleware"
	"github.com/cds-id/pdt/backend/internal/services/gitlab"
	"github.com/cds-id/pdt/backend/internal/services/report"
	"github.com/cds-id/pdt/backend/internal/services/storage"
	"github.com/cds-id/pdt/backend/internal/worker"
	"github.com/gin-gonic/gin"
	"github.com/joho/godotenv"
	"gorm.io/gorm"
)

func init() {
	_ = godotenv.Load("../../.env")
}

func getEnv(key string) string {
	return os.Getenv(key)
}

func skipIfNoGitLab(t *testing.T) {
	if getEnv("SIT_GITLAB_TOKEN") == "" {
		t.Skip("SIT_GITLAB_TOKEN not set, skipping GitLab SIT tests")
	}
}

// =============================================================================
// Test 1: Direct GitLab Service — no DB needed
// =============================================================================

func TestGitLabService_ValidateAccess(t *testing.T) {
	skipIfNoGitLab(t)

	client := gitlab.New(getEnv("SIT_GITLAB_URL"))
	// For nested paths: owner="siswamedia", repo="frontend/web/v1/siswamedia-lms"
	err := client.ValidateAccess("siswamedia", "frontend/web/v1/siswamedia-lms", getEnv("SIT_GITLAB_TOKEN"))
	if err != nil {
		t.Fatalf("ValidateAccess failed: %v", err)
	}
	t.Log("GitLab access validated successfully")
}

func TestGitLabService_FetchCommits(t *testing.T) {
	skipIfNoGitLab(t)

	client := gitlab.New(getEnv("SIT_GITLAB_URL"))
	since := time.Now().AddDate(0, 0, -30)

	commits, err := client.FetchCommits("siswamedia", "frontend/web/v1/siswamedia-lms", getEnv("SIT_GITLAB_TOKEN"), since)
	if err != nil {
		t.Fatalf("FetchCommits failed: %v", err)
	}

	t.Logf("Fetched %d commits from last 30 days", len(commits))
	for i, c := range commits {
		if i >= 5 {
			t.Logf("  ... and %d more", len(commits)-5)
			break
		}
		t.Logf("  [%s] %s — %s (%s) branch:%s", c.SHA[:8], c.Date.Format("2006-01-02"), truncate(c.Message, 50), c.Author, c.Branch)
	}
}

// =============================================================================
// Test 2: Full API Flow — requires MySQL
// =============================================================================

func setupRouter(t *testing.T) (*gin.Engine, *gorm.DB, *crypto.Encryptor, *config.Config) {
	t.Helper()

	cfg, err := config.Load()
	if err != nil {
		t.Fatalf("failed to load config: %v", err)
	}

	db, err := database.Connect(cfg.DSN())
	if err != nil {
		t.Fatalf("failed to connect to DB: %v\nMake sure MySQL is running and database '%s' exists", err, cfg.DBName)
	}

	if err := database.Migrate(db); err != nil {
		t.Fatalf("failed to migrate: %v", err)
	}

	enc, err := crypto.NewEncryptor(cfg.EncryptionKey)
	if err != nil {
		t.Fatalf("failed to create encryptor: %v", err)
	}

	gin.SetMode(gin.TestMode)
	r := gin.New()

	authHandler := &handlers.AuthHandler{DB: db, JWTSecret: cfg.JWTSecret, JWTExpiryHours: cfg.JWTExpiryHours}
	userHandler := &handlers.UserHandler{DB: db, Encryptor: enc}
	repoHandler := &handlers.RepoHandler{DB: db}
	syncStatus := worker.NewSyncStatus()
	syncHandler := &handlers.SyncHandler{DB: db, Encryptor: enc, Status: syncStatus}
	commitHandler := &handlers.CommitHandler{DB: db}
	jiraHandler := &handlers.JiraHandler{DB: db, Encryptor: enc}
	reportGen := report.NewGenerator(db, enc)
	var r2Client *storage.R2Client
	if id := getEnv("R2_ACCOUNT_ID"); id != "" {
		r2Client = storage.NewR2Client(id, getEnv("R2_ACCESS_KEY_ID"), getEnv("R2_SECRET_ACCESS_KEY"), getEnv("R2_BUCKET_NAME"), getEnv("R2_PUBLIC_DOMAIN"))
	}
	reportHandler := &handlers.ReportHandler{DB: db, Generator: reportGen, R2: r2Client}

	api := r.Group("/api")
	auth := api.Group("/auth")
	auth.POST("/register", authHandler.Register)
	auth.POST("/login", authHandler.Login)

	protected := api.Group("")
	protected.Use(middleware.JWTAuth(cfg.JWTSecret))
	user := protected.Group("/user")
	user.GET("/profile", userHandler.GetProfile)
	user.PUT("/profile", userHandler.UpdateProfile)
	user.POST("/profile/validate", userHandler.ValidateConnections)
	repos := protected.Group("/repos")
	repos.GET("", repoHandler.List)
	repos.POST("", repoHandler.Add)
	repos.DELETE("/:id", repoHandler.Delete)
	protected.POST("/sync/commits", syncHandler.SyncCommits)
	protected.GET("/sync/status", syncHandler.SyncStatus)
	commits := protected.Group("/commits")
	commits.GET("", commitHandler.List)
	commits.GET("/missing", commitHandler.Missing)
	commits.POST("/:sha/link", commitHandler.Link)
	jira := protected.Group("/jira")
	jira.GET("/sprints", jiraHandler.ListSprints)
	jira.GET("/sprints/:id", jiraHandler.GetSprint)
	jira.GET("/active-sprint", jiraHandler.GetActiveSprint)
	jira.GET("/cards", jiraHandler.ListCards)
	jira.GET("/cards/:key", jiraHandler.GetCard)
	reports := protected.Group("/reports")
	reports.POST("/generate", reportHandler.Generate)
	reports.GET("", reportHandler.List)
	reports.GET("/:id", reportHandler.Get)
	reports.DELETE("/:id", reportHandler.Delete)
	reportTemplates := reports.Group("/templates")
	reportTemplates.GET("", reportHandler.ListTemplates)
	reportTemplates.POST("", reportHandler.CreateTemplate)
	reportTemplates.PUT("/:id", reportHandler.UpdateTemplate)
	reportTemplates.DELETE("/:id", reportHandler.DeleteTemplate)
	reportTemplates.POST("/preview", reportHandler.PreviewTemplate)

	return r, db, enc, cfg
}

func TestFullGitLabFlow(t *testing.T) {
	skipIfNoGitLab(t)

	router, db, _, _ := setupRouter(t)

	// Clean up test data after
	t.Cleanup(func() {
		db.Exec("DELETE FROM commit_card_links")
		db.Exec("DELETE FROM commits")
		db.Exec("DELETE FROM repositories WHERE user_id IN (SELECT id FROM users WHERE email = ?)", "sit-gitlab@test.local")
		db.Exec("DELETE FROM users WHERE email = ?", "sit-gitlab@test.local")
	})

	var token string

	// --- Step 1: Register ---
	t.Run("1_Register", func(t *testing.T) {
		body := jsonBody(map[string]string{
			"email":    "sit-gitlab@test.local",
			"password": "testpass123",
		})
		w := doRequest(router, "POST", "/api/auth/register", body, "")
		if w.Code != http.StatusCreated {
			t.Fatalf("register failed: %d — %s", w.Code, w.Body.String())
		}

		var resp map[string]interface{}
		json.Unmarshal(w.Body.Bytes(), &resp)
		token = resp["token"].(string)
		t.Logf("Registered, got JWT token: %s...", token[:20])
	})

	// --- Step 2: Configure GitLab token ---
	t.Run("2_ConfigureGitLab", func(t *testing.T) {
		gitlabToken := getEnv("SIT_GITLAB_TOKEN")
		gitlabURL := getEnv("SIT_GITLAB_URL")

		body := jsonBody(map[string]string{
			"gitlab_token": gitlabToken,
			"gitlab_url":   gitlabURL,
		})
		w := doRequest(router, "PUT", "/api/user/profile", body, token)
		if w.Code != http.StatusOK {
			t.Fatalf("update profile failed: %d — %s", w.Code, w.Body.String())
		}
		t.Log("GitLab token and URL configured")
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
		t.Logf("Profile: has_gitlab=%v, gitlab_url=%s", resp["has_gitlab_token"], resp["gitlab_url"])
	})

	// --- Step 4: Add GitLab repo ---
	var repoID float64
	t.Run("4_AddRepo", func(t *testing.T) {
		body := jsonBody(map[string]string{
			"url": getEnv("SIT_GITLAB_REPO"),
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

	// --- Step 5: List repos ---
	t.Run("5_ListRepos", func(t *testing.T) {
		w := doRequest(router, "GET", "/api/repos", nil, token)
		if w.Code != http.StatusOK {
			t.Fatalf("list repos failed: %d — %s", w.Code, w.Body.String())
		}

		var repos []map[string]interface{}
		json.Unmarshal(w.Body.Bytes(), &repos)
		if len(repos) == 0 {
			t.Fatal("expected at least 1 repo")
		}
		t.Logf("Repos: %d tracked", len(repos))
	})

	// --- Step 6: Sync commits ---
	t.Run("6_SyncCommits", func(t *testing.T) {
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

	// --- Step 7: List all commits ---
	t.Run("7_ListCommits", func(t *testing.T) {
		w := doRequest(router, "GET", "/api/commits", nil, token)
		if w.Code != http.StatusOK {
			t.Fatalf("list commits failed: %d — %s", w.Code, w.Body.String())
		}

		var commits []map[string]interface{}
		json.Unmarshal(w.Body.Bytes(), &commits)
		t.Logf("Total commits: %d", len(commits))

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

	// --- Step 8: Check missing Jira refs ---
	t.Run("8_MissingJiraRefs", func(t *testing.T) {
		w := doRequest(router, "GET", "/api/commits/missing", nil, token)
		if w.Code != http.StatusOK {
			t.Fatalf("missing commits failed: %d — %s", w.Code, w.Body.String())
		}

		var commits []map[string]interface{}
		json.Unmarshal(w.Body.Bytes(), &commits)
		t.Logf("Commits missing Jira reference: %d", len(commits))

		for i, c := range commits {
			if i >= 5 {
				t.Logf("  ... and %d more", len(commits)-5)
				break
			}
			t.Logf("  [%s] %s", c["sha"].(string)[:8], truncate(fmt.Sprint(c["message"]), 60))
		}
	})

	// --- Step 9: Filter commits by repo ---
	t.Run("9_FilterByRepo", func(t *testing.T) {
		url := fmt.Sprintf("/api/commits?repo_id=%d", int(repoID))
		w := doRequest(router, "GET", url, nil, token)
		if w.Code != http.StatusOK {
			t.Fatalf("filter commits failed: %d — %s", w.Code, w.Body.String())
		}

		var commits []map[string]interface{}
		json.Unmarshal(w.Body.Bytes(), &commits)
		t.Logf("Commits for repo %d: %d", int(repoID), len(commits))
	})

	// --- Step 10: Delete repo (cleanup) ---
	t.Run("10_DeleteRepo", func(t *testing.T) {
		url := fmt.Sprintf("/api/repos/%d", int(repoID))
		w := doRequest(router, "DELETE", url, nil, token)
		if w.Code != http.StatusOK {
			t.Fatalf("delete repo failed: %d — %s", w.Code, w.Body.String())
		}
		t.Log("Repo deleted successfully")
	})
}

// =============================================================================
// Helpers
// =============================================================================

func jsonBody(data interface{}) *bytes.Buffer {
	b, _ := json.Marshal(data)
	return bytes.NewBuffer(b)
}

func doRequest(r *gin.Engine, method, path string, body *bytes.Buffer, token string) *httptest.ResponseRecorder {
	var req *http.Request
	if body != nil {
		req = httptest.NewRequest(method, path, body)
		req.Header.Set("Content-Type", "application/json")
	} else {
		req = httptest.NewRequest(method, path, nil)
	}

	if token != "" {
		req.Header.Set("Authorization", "Bearer "+token)
	}

	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	return w
}

func truncate(s string, max int) string {
	// Remove newlines for display
	clean := ""
	for _, c := range s {
		if c == '\n' || c == '\r' {
			clean += " "
		} else {
			clean += string(c)
		}
	}
	if len(clean) > max {
		return clean[:max] + "..."
	}
	return clean
}

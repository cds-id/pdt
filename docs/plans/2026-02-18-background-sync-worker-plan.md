# Background Sync Worker Implementation Plan

> **For Claude:** REQUIRED SUB-SKILL: Use superpowers:executing-plans to implement this plan task-by-task.

**Goal:** Add an embedded background worker that automatically syncs GitHub/GitLab commits and Jira data on configurable intervals.

**Architecture:** Ticker-based goroutines embedded in the existing server binary. Sync logic extracted from handlers into shared functions in `internal/worker/`. Two independent loops (commits + jira) with concurrency guards and in-memory status tracking.

**Tech Stack:** Go stdlib (`time.Ticker`, `sync.RWMutex`, `context.Context`, `sync/atomic`), existing GORM + crypto deps.

---

### Task 1: Add worker config to .env and config loader

**Files:**
- Modify: `backend/internal/config/config.go`
- Modify: `backend/.env`
- Modify: `backend/.env.example`

**Step 1: Add new fields to Config struct**

In `backend/internal/config/config.go`, add these fields to the `Config` struct:

```go
type Config struct {
	// ... existing fields ...
	SyncEnabled          bool
	SyncIntervalCommits  time.Duration
	SyncIntervalJira     time.Duration
}
```

Add `"time"` to the imports.

**Step 2: Parse new env vars in Load()**

Add after the existing config parsing (before the validation block):

```go
	syncEnabled := getEnv("SYNC_ENABLED", "true")
	cfg.SyncEnabled = syncEnabled == "true" || syncEnabled == "1"

	commitInterval, err := time.ParseDuration(getEnv("SYNC_INTERVAL_COMMITS", "15m"))
	if err != nil {
		commitInterval = 15 * time.Minute
	}
	cfg.SyncIntervalCommits = commitInterval

	jiraInterval, err := time.ParseDuration(getEnv("SYNC_INTERVAL_JIRA", "30m"))
	if err != nil {
		jiraInterval = 30 * time.Minute
	}
	cfg.SyncIntervalJira = jiraInterval
```

Note: `err` is already used in this function for `strconv.Atoi`. Shadow it with `:=` — this is fine in Go since these are independent blocks.

**Step 3: Add env vars to .env and .env.example**

Append to both `backend/.env` and `backend/.env.example`:

```
# Worker
SYNC_ENABLED=true
SYNC_INTERVAL_COMMITS=15m
SYNC_INTERVAL_JIRA=30m
```

**Step 4: Verify it compiles**

Run: `cd /home/nst/GolandProjects/pdt/backend && go build ./...`
Expected: No errors.

**Step 5: Commit**

```bash
git add backend/internal/config/config.go backend/.env backend/.env.example
git commit -m "feat: add worker sync config (intervals, enabled flag)"
```

---

### Task 2: Create SyncStatus type with thread-safe access

**Files:**
- Create: `backend/internal/worker/status.go`

**Step 1: Create the status package file**

Create `backend/internal/worker/status.go`:

```go
package worker

import (
	"sync"
	"time"
)

type SyncState string

const (
	StateIdle    SyncState = "idle"
	StateSyncing SyncState = "syncing"
)

type SyncInfo struct {
	LastSync  *time.Time `json:"last_sync"`
	NextSync  *time.Time `json:"next_sync"`
	Status    SyncState  `json:"status"`
	LastError *string    `json:"last_error"`
}

type SyncStatus struct {
	mu      sync.RWMutex
	commits map[uint]*SyncInfo // keyed by userID
	jira    map[uint]*SyncInfo
}

func NewSyncStatus() *SyncStatus {
	return &SyncStatus{
		commits: make(map[uint]*SyncInfo),
		jira:    make(map[uint]*SyncInfo),
	}
}

func (s *SyncStatus) GetCommitStatus(userID uint) SyncInfo {
	s.mu.RLock()
	defer s.mu.RUnlock()
	if info, ok := s.commits[userID]; ok {
		return *info
	}
	return SyncInfo{Status: StateIdle}
}

func (s *SyncStatus) GetJiraStatus(userID uint) SyncInfo {
	s.mu.RLock()
	defer s.mu.RUnlock()
	if info, ok := s.jira[userID]; ok {
		return *info
	}
	return SyncInfo{Status: StateIdle}
}

func (s *SyncStatus) SetCommitSyncing(userID uint) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if _, ok := s.commits[userID]; !ok {
		s.commits[userID] = &SyncInfo{}
	}
	s.commits[userID].Status = StateSyncing
}

func (s *SyncStatus) SetCommitDone(userID uint, nextSync time.Time, syncErr error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if _, ok := s.commits[userID]; !ok {
		s.commits[userID] = &SyncInfo{}
	}
	now := time.Now()
	s.commits[userID].LastSync = &now
	s.commits[userID].NextSync = &nextSync
	s.commits[userID].Status = StateIdle
	if syncErr != nil {
		errStr := syncErr.Error()
		s.commits[userID].LastError = &errStr
	} else {
		s.commits[userID].LastError = nil
	}
}

func (s *SyncStatus) SetJiraSyncing(userID uint) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if _, ok := s.jira[userID]; !ok {
		s.jira[userID] = &SyncInfo{}
	}
	s.jira[userID].Status = StateSyncing
}

func (s *SyncStatus) SetJiraDone(userID uint, nextSync time.Time, syncErr error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if _, ok := s.jira[userID]; !ok {
		s.jira[userID] = &SyncInfo{}
	}
	now := time.Now()
	s.jira[userID].LastSync = &now
	s.jira[userID].NextSync = &nextSync
	s.jira[userID].Status = StateIdle
	if syncErr != nil {
		errStr := syncErr.Error()
		s.jira[userID].LastError = &errStr
	} else {
		s.jira[userID].LastError = nil
	}
}
```

**Step 2: Verify it compiles**

Run: `cd /home/nst/GolandProjects/pdt/backend && go build ./...`
Expected: No errors.

**Step 3: Commit**

```bash
git add backend/internal/worker/status.go
git commit -m "feat: add thread-safe SyncStatus type for worker observability"
```

---

### Task 3: Extract commit sync logic into shared function

**Files:**
- Create: `backend/internal/worker/commits.go`
- Modify: `backend/internal/handlers/sync.go`

**Step 1: Create the shared commit sync function**

Create `backend/internal/worker/commits.go`:

```go
package worker

import (
	"fmt"
	"log"
	"time"

	"github.com/cds-id/pdt/backend/internal/crypto"
	"github.com/cds-id/pdt/backend/internal/models"
	"github.com/cds-id/pdt/backend/internal/services"
	"github.com/cds-id/pdt/backend/internal/services/github"
	"github.com/cds-id/pdt/backend/internal/services/gitlab"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

type CommitSyncResult struct {
	RepoID   uint   `json:"repo_id"`
	RepoName string `json:"repo_name"`
	Provider string `json:"provider"`
	New      int    `json:"new_commits"`
	Total    int    `json:"total_fetched"`
	Error    string `json:"error,omitempty"`
}

// SyncUserCommits syncs commits for all repos of a given user.
// Returns results per repo.
func SyncUserCommits(db *gorm.DB, enc *crypto.Encryptor, userID uint) ([]CommitSyncResult, error) {
	var user models.User
	if err := db.First(&user, userID).Error; err != nil {
		return nil, fmt.Errorf("user not found: %w", err)
	}

	var repos []models.Repository
	if err := db.Where("user_id = ?", userID).Find(&repos).Error; err != nil {
		return nil, fmt.Errorf("failed to fetch repositories: %w", err)
	}

	if len(repos) == 0 {
		return nil, nil
	}

	since := time.Now().AddDate(0, 0, -30)
	var results []CommitSyncResult

	for _, repo := range repos {
		result := CommitSyncResult{
			RepoID:   repo.ID,
			RepoName: fmt.Sprintf("%s/%s", repo.Owner, repo.Name),
			Provider: string(repo.Provider),
		}

		var provider services.CommitProvider
		var token string

		switch repo.Provider {
		case models.ProviderGitHub:
			provider = github.New()
			decrypted, err := enc.Decrypt(user.GithubToken)
			if err != nil {
				result.Error = "failed to decrypt github token"
				results = append(results, result)
				continue
			}
			token = decrypted
		case models.ProviderGitLab:
			provider = gitlab.New(user.GitlabURL)
			decrypted, err := enc.Decrypt(user.GitlabToken)
			if err != nil {
				result.Error = "failed to decrypt gitlab token"
				results = append(results, result)
				continue
			}
			token = decrypted
		}

		if token == "" {
			result.Error = fmt.Sprintf("no %s token configured", repo.Provider)
			results = append(results, result)
			continue
		}

		commits, err := provider.FetchCommits(repo.Owner, repo.Name, token, since)
		if err != nil {
			result.Error = err.Error()
			db.Model(&repo).Update("is_valid", false)
			results = append(results, result)
			continue
		}

		result.Total = len(commits)

		for _, ci := range commits {
			jiraKey := services.ExtractJiraKey(ci.Message)
			commit := models.Commit{
				RepoID:      repo.ID,
				SHA:         ci.SHA,
				Message:     ci.Message,
				Author:      ci.Author,
				AuthorEmail: ci.AuthorEmail,
				Branch:      ci.Branch,
				Date:        ci.Date,
				JiraCardKey: jiraKey,
				HasLink:     jiraKey != "",
			}

			res := db.Clauses(clause.OnConflict{
				Columns:   []clause.Column{{Name: "sha"}},
				DoNothing: true,
			}).Create(&commit)

			if res.RowsAffected > 0 {
				result.New++
			}
		}

		now := time.Now()
		db.Model(&repo).Updates(map[string]interface{}{
			"is_valid":       true,
			"last_synced_at": &now,
		})

		results = append(results, result)
	}

	return results, nil
}

// SyncAllUsersCommits syncs commits for all users who have repos.
func SyncAllUsersCommits(db *gorm.DB, enc *crypto.Encryptor) {
	var userIDs []uint
	db.Model(&models.Repository{}).Distinct("user_id").Pluck("user_id", &userIDs)

	for _, uid := range userIDs {
		results, err := SyncUserCommits(db, enc, uid)
		if err != nil {
			log.Printf("[worker] commit sync failed for user %d: %v", uid, err)
			continue
		}
		for _, r := range results {
			if r.Error != "" {
				log.Printf("[worker] commit sync user=%d repo=%s error=%s", uid, r.RepoName, r.Error)
			} else if r.New > 0 {
				log.Printf("[worker] commit sync user=%d repo=%s new=%d total=%d", uid, r.RepoName, r.New, r.Total)
			}
		}
	}
}
```

**Step 2: Rewrite sync handler to use shared function**

Replace the body of `SyncCommits` in `backend/internal/handlers/sync.go` with:

```go
package handlers

import (
	"net/http"

	"github.com/cds-id/pdt/backend/internal/crypto"
	"github.com/cds-id/pdt/backend/internal/worker"
	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

type SyncHandler struct {
	DB        *gorm.DB
	Encryptor *crypto.Encryptor
}

func (h *SyncHandler) SyncCommits(c *gin.Context) {
	userID := c.GetUint("user_id")

	results, err := worker.SyncUserCommits(h.DB, h.Encryptor, userID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	if results == nil {
		c.JSON(http.StatusOK, gin.H{"message": "no repositories to sync", "results": []worker.CommitSyncResult{}})
		return
	}

	c.JSON(http.StatusOK, gin.H{"results": results})
}
```

**Step 3: Verify it compiles**

Run: `cd /home/nst/GolandProjects/pdt/backend && go build ./...`
Expected: No errors.

**Step 4: Run existing SIT tests to verify no regression**

Run: `cd /home/nst/GolandProjects/pdt/backend && go test -v -run TestFullGitHubFlow ./tests/sit/ -timeout 120s`
Expected: All 9 sub-tests PASS. The sync step should still work identically since we extracted the same logic.

**Step 5: Commit**

```bash
git add backend/internal/worker/commits.go backend/internal/handlers/sync.go
git commit -m "refactor: extract commit sync into shared worker function"
```

---

### Task 4: Add details_json to JiraCard model and extract Jira sync logic

**Files:**
- Modify: `backend/internal/models/jira.go`
- Create: `backend/internal/worker/jira.go`

**Step 1: Add DetailsJSON field to JiraCard model**

In `backend/internal/models/jira.go`, add to the `JiraCard` struct:

```go
type JiraCard struct {
	// ... existing fields ...
	DetailsJSON string    `gorm:"type:text" json:"details_json,omitempty"`
	// ... rest of fields ...
}
```

Add the field after `SprintID` and before `CreatedAt`.

**Step 2: Create the shared Jira sync function**

Create `backend/internal/worker/jira.go`:

```go
package worker

import (
	"encoding/json"
	"fmt"
	"log"
	"strconv"

	"github.com/cds-id/pdt/backend/internal/crypto"
	"github.com/cds-id/pdt/backend/internal/models"
	"github.com/cds-id/pdt/backend/internal/services/jira"
	"gorm.io/gorm"
)

// SyncUserJira syncs sprints, cards, and issue details for a user.
func SyncUserJira(db *gorm.DB, enc *crypto.Encryptor, userID uint) error {
	var user models.User
	if err := db.First(&user, userID).Error; err != nil {
		return fmt.Errorf("user not found: %w", err)
	}

	if user.JiraToken == "" || user.JiraWorkspace == "" || user.JiraEmail == "" {
		return nil // Jira not configured, skip silently
	}

	token, err := enc.Decrypt(user.JiraToken)
	if err != nil {
		return fmt.Errorf("failed to decrypt jira token: %w", err)
	}

	client := jira.New(user.JiraWorkspace, user.JiraEmail, token)

	// 1. Fetch boards
	boards, err := client.FetchBoards()
	if err != nil {
		return fmt.Errorf("failed to fetch boards: %w", err)
	}

	// 2. Fetch sprints from all boards
	for _, boardID := range boards {
		sprints, err := client.FetchSprints(boardID)
		if err != nil {
			log.Printf("[worker] jira sync user=%d board=%d sprint fetch error: %v", userID, boardID, err)
			continue
		}

		for _, s := range sprints {
			sprint := models.Sprint{
				UserID:       userID,
				JiraSprintID: strconv.Itoa(s.ID),
				Name:         s.Name,
				State:        models.SprintState(s.State),
				StartDate:    s.StartDate,
				EndDate:      s.EndDate,
			}
			db.Where("jira_sprint_id = ?", sprint.JiraSprintID).
				Assign(sprint).FirstOrCreate(&sprint)

			// 3. For active sprints, fetch issues
			if s.State == "active" {
				cards, err := client.FetchSprintIssues(s.ID)
				if err != nil {
					log.Printf("[worker] jira sync user=%d sprint=%d issue fetch error: %v", userID, s.ID, err)
					continue
				}

				for _, card := range cards {
					jiraCard := models.JiraCard{
						UserID:   userID,
						Key:      card.Key,
						Summary:  card.Summary,
						Status:   card.Status,
						Assignee: card.Assignee,
						SprintID: &sprint.ID,
					}

					// 4. Fetch issue detail with changelog
					detail, err := client.FetchIssue(card.Key)
					if err == nil && detail != nil {
						detailJSON, _ := json.Marshal(detail)
						jiraCard.DetailsJSON = string(detailJSON)
					}

					db.Where("user_id = ? AND card_key = ?", userID, card.Key).
						Assign(jiraCard).FirstOrCreate(&jiraCard)
				}
			}
		}
	}

	return nil
}

// SyncAllUsersJira syncs Jira data for all users with Jira configured.
func SyncAllUsersJira(db *gorm.DB, enc *crypto.Encryptor) {
	var users []models.User
	db.Where("jira_token != '' AND jira_workspace != '' AND jira_email != ''").Find(&users)

	for _, user := range users {
		if err := SyncUserJira(db, enc, user.ID); err != nil {
			log.Printf("[worker] jira sync failed for user %d: %v", user.ID, err)
		} else {
			log.Printf("[worker] jira sync completed for user %d", user.ID)
		}
	}
}
```

**Step 3: Verify it compiles**

Run: `cd /home/nst/GolandProjects/pdt/backend && go build ./...`
Expected: No errors. (GORM AutoMigrate will add the `details_json` column on next startup.)

**Step 4: Commit**

```bash
git add backend/internal/models/jira.go backend/internal/worker/jira.go
git commit -m "feat: add Jira sync worker with issue detail storage"
```

---

### Task 5: Create the Scheduler with ticker loops and concurrency guards

**Files:**
- Create: `backend/internal/worker/scheduler.go`

**Step 1: Create the scheduler**

Create `backend/internal/worker/scheduler.go`:

```go
package worker

import (
	"context"
	"log"
	"sync/atomic"
	"time"

	"github.com/cds-id/pdt/backend/internal/crypto"
	"gorm.io/gorm"
)

type Scheduler struct {
	DB                  *gorm.DB
	Encryptor           *crypto.Encryptor
	CommitInterval      time.Duration
	JiraInterval        time.Duration
	Status              *SyncStatus
	commitRunning       atomic.Bool
	jiraRunning         atomic.Bool
}

func NewScheduler(db *gorm.DB, enc *crypto.Encryptor, commitInterval, jiraInterval time.Duration) *Scheduler {
	return &Scheduler{
		DB:             db,
		Encryptor:      enc,
		CommitInterval: commitInterval,
		JiraInterval:   jiraInterval,
		Status:         NewSyncStatus(),
	}
}

func (s *Scheduler) Start(ctx context.Context) {
	log.Printf("[worker] starting scheduler: commits=%s, jira=%s", s.CommitInterval, s.JiraInterval)

	// Run immediately on startup, then on interval
	go s.commitSyncLoop(ctx)
	go s.jiraSyncLoop(ctx)
}

func (s *Scheduler) commitSyncLoop(ctx context.Context) {
	// Run once immediately
	s.runCommitSync()

	ticker := time.NewTicker(s.CommitInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			log.Println("[worker] commit sync loop stopped")
			return
		case <-ticker.C:
			s.runCommitSync()
		}
	}
}

func (s *Scheduler) runCommitSync() {
	if !s.commitRunning.CompareAndSwap(false, true) {
		log.Println("[worker] commit sync skipped: previous run still in progress")
		return
	}
	defer s.commitRunning.Store(false)

	log.Println("[worker] commit sync starting")
	SyncAllUsersCommits(s.DB, s.Encryptor)
	log.Println("[worker] commit sync completed")
}

func (s *Scheduler) jiraSyncLoop(ctx context.Context) {
	// Run once immediately
	s.runJiraSync()

	ticker := time.NewTicker(s.JiraInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			log.Println("[worker] jira sync loop stopped")
			return
		case <-ticker.C:
			s.runJiraSync()
		}
	}
}

func (s *Scheduler) runJiraSync() {
	if !s.jiraRunning.CompareAndSwap(false, true) {
		log.Println("[worker] jira sync skipped: previous run still in progress")
		return
	}
	defer s.jiraRunning.Store(false)

	log.Println("[worker] jira sync starting")
	SyncAllUsersJira(s.DB, s.Encryptor)
	log.Println("[worker] jira sync completed")
}
```

**Step 2: Verify it compiles**

Run: `cd /home/nst/GolandProjects/pdt/backend && go build ./...`
Expected: No errors.

**Step 3: Commit**

```bash
git add backend/internal/worker/scheduler.go
git commit -m "feat: add Scheduler with ticker loops and concurrency guards"
```

---

### Task 6: Wire scheduler into main.go and add status endpoint

**Files:**
- Modify: `backend/cmd/server/main.go`
- Modify: `backend/internal/handlers/sync.go`

**Step 1: Add status handler to sync handler**

Add a new method and field to `backend/internal/handlers/sync.go`. Add a `Status` field and a `SyncStatus` method:

```go
package handlers

import (
	"net/http"

	"github.com/cds-id/pdt/backend/internal/crypto"
	"github.com/cds-id/pdt/backend/internal/worker"
	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

type SyncHandler struct {
	DB        *gorm.DB
	Encryptor *crypto.Encryptor
	Status    *worker.SyncStatus
}

func (h *SyncHandler) SyncCommits(c *gin.Context) {
	userID := c.GetUint("user_id")

	results, err := worker.SyncUserCommits(h.DB, h.Encryptor, userID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	if results == nil {
		c.JSON(http.StatusOK, gin.H{"message": "no repositories to sync", "results": []worker.CommitSyncResult{}})
		return
	}

	c.JSON(http.StatusOK, gin.H{"results": results})
}

func (h *SyncHandler) SyncStatus(c *gin.Context) {
	userID := c.GetUint("user_id")

	response := gin.H{
		"commits": h.Status.GetCommitStatus(userID),
		"jira":    h.Status.GetJiraStatus(userID),
	}

	c.JSON(http.StatusOK, response)
}
```

**Step 2: Wire scheduler and status endpoint in main.go**

In `backend/cmd/server/main.go`, add the worker import and scheduler startup. The changes are:

1. Add import: `"github.com/cds-id/pdt/backend/internal/worker"`
2. After the encryptor creation, create and start the scheduler (only if enabled):

```go
	// Worker scheduler
	var scheduler *worker.Scheduler
	if cfg.SyncEnabled {
		scheduler = worker.NewScheduler(db, encryptor, cfg.SyncIntervalCommits, cfg.SyncIntervalJira)
		scheduler.Start(ctx)
	}
```

3. Pass the scheduler's status to the sync handler. Update the syncHandler creation:

```go
	var syncStatus *worker.SyncStatus
	if scheduler != nil {
		syncStatus = scheduler.Status
	} else {
		syncStatus = worker.NewSyncStatus()
	}
	syncHandler := &handlers.SyncHandler{DB: db, Encryptor: encryptor, Status: syncStatus}
```

4. Add the status route after the existing sync route:

```go
		protected.POST("/sync/commits", syncHandler.SyncCommits)
		protected.GET("/sync/status", syncHandler.SyncStatus)
```

**Step 3: Verify it compiles**

Run: `cd /home/nst/GolandProjects/pdt/backend && go build ./...`
Expected: No errors.

**Step 4: Commit**

```bash
git add backend/cmd/server/main.go backend/internal/handlers/sync.go
git commit -m "feat: wire scheduler into server and add sync status endpoint"
```

---

### Task 7: Update SyncStatus tracking in worker loops

**Files:**
- Modify: `backend/internal/worker/scheduler.go`

**Step 1: Update scheduler to track per-user status**

Replace the `runCommitSync` and `runJiraSync` methods to update status:

```go
func (s *Scheduler) runCommitSync() {
	if !s.commitRunning.CompareAndSwap(false, true) {
		log.Println("[worker] commit sync skipped: previous run still in progress")
		return
	}
	defer s.commitRunning.Store(false)

	log.Println("[worker] commit sync starting")

	// Get all user IDs with repos
	var userIDs []uint
	s.DB.Model(&models.Repository{}).Distinct("user_id").Pluck("user_id", &userIDs)

	nextSync := time.Now().Add(s.CommitInterval)

	for _, uid := range userIDs {
		s.Status.SetCommitSyncing(uid)
		results, err := SyncUserCommits(s.DB, s.Encryptor, uid)
		if err != nil {
			s.Status.SetCommitDone(uid, nextSync, err)
			log.Printf("[worker] commit sync failed for user %d: %v", uid, err)
			continue
		}
		for _, r := range results {
			if r.Error != "" {
				log.Printf("[worker] commit sync user=%d repo=%s error=%s", uid, r.RepoName, r.Error)
			} else if r.New > 0 {
				log.Printf("[worker] commit sync user=%d repo=%s new=%d total=%d", uid, r.RepoName, r.New, r.Total)
			}
		}
		s.Status.SetCommitDone(uid, nextSync, nil)
	}

	log.Println("[worker] commit sync completed")
}

func (s *Scheduler) runJiraSync() {
	if !s.jiraRunning.CompareAndSwap(false, true) {
		log.Println("[worker] jira sync skipped: previous run still in progress")
		return
	}
	defer s.jiraRunning.Store(false)

	log.Println("[worker] jira sync starting")

	var users []models.User
	s.DB.Where("jira_token != '' AND jira_workspace != '' AND jira_email != ''").Find(&users)

	nextSync := time.Now().Add(s.JiraInterval)

	for _, user := range users {
		s.Status.SetJiraSyncing(user.ID)
		err := SyncUserJira(s.DB, s.Encryptor, user.ID)
		if err != nil {
			s.Status.SetJiraDone(user.ID, nextSync, err)
			log.Printf("[worker] jira sync failed for user %d: %v", user.ID, err)
		} else {
			s.Status.SetJiraDone(user.ID, nextSync, nil)
			log.Printf("[worker] jira sync completed for user %d", user.ID)
		}
	}

	log.Println("[worker] jira sync completed")
}
```

Also add the models import to scheduler.go:

```go
import (
	"context"
	"log"
	"sync/atomic"
	"time"

	"github.com/cds-id/pdt/backend/internal/crypto"
	"github.com/cds-id/pdt/backend/internal/models"
	"gorm.io/gorm"
)
```

And remove the calls to `SyncAllUsersCommits` and `SyncAllUsersJira` from the scheduler methods since the logic is now inline.

**Step 2: Verify it compiles**

Run: `cd /home/nst/GolandProjects/pdt/backend && go build ./...`
Expected: No errors.

**Step 3: Commit**

```bash
git add backend/internal/worker/scheduler.go
git commit -m "feat: track per-user sync status in worker loops"
```

---

### Task 8: Add sync status route to SIT test router and write status test

**Files:**
- Modify: `backend/tests/sit/gitlab_sit_test.go` (setupRouter function)
- Modify: `backend/tests/sit/github_sit_test.go`

**Step 1: Update setupRouter to include status endpoint**

In `backend/tests/sit/gitlab_sit_test.go`, update the `setupRouter` function:

1. Add import for `"github.com/cds-id/pdt/backend/internal/worker"`
2. Update syncHandler creation to include Status:

```go
	syncStatus := worker.NewSyncStatus()
	syncHandler := &handlers.SyncHandler{DB: db, Encryptor: enc, Status: syncStatus}
```

3. Add the status route after the existing sync route:

```go
	protected.POST("/sync/commits", syncHandler.SyncCommits)
	protected.GET("/sync/status", syncHandler.SyncStatus)
```

**Step 2: Add a sync status test step to the GitHub full flow**

In `backend/tests/sit/github_sit_test.go`, add a new test step after `5_SyncCommits`:

```go
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
```

**Step 3: Verify it compiles**

Run: `cd /home/nst/GolandProjects/pdt/backend && go build ./...`
Expected: No errors.

**Step 4: Run GitHub SIT tests**

Run: `cd /home/nst/GolandProjects/pdt/backend && go test -v -run TestFullGitHubFlow ./tests/sit/ -timeout 120s`
Expected: All tests pass including the new 5b_SyncStatus step.

**Step 5: Run GitLab SIT tests**

Run: `cd /home/nst/GolandProjects/pdt/backend && go test -v -run TestFullGitLabFlow ./tests/sit/ -timeout 120s`
Expected: All tests pass.

**Step 6: Commit**

```bash
git add backend/tests/sit/gitlab_sit_test.go backend/tests/sit/github_sit_test.go
git commit -m "test: add sync status endpoint to SIT test router and GitHub flow"
```

---

### Task 9: Integration test — run the full server with worker

**Files:** None (manual verification)

**Step 1: Start the server with worker enabled**

Run: `cd /home/nst/GolandProjects/pdt/backend && go run cmd/server/main.go`

Expected output should include:
```
[worker] starting scheduler: commits=15m0s, jira=30m0s
[worker] commit sync starting
[worker] jira sync starting
server starting on :8080
```

**Step 2: Verify the worker logs sync activity**

Watch the logs for about 10 seconds. You should see:
- `[worker] commit sync completed` (or errors if no users exist yet)
- `[worker] jira sync completed`

**Step 3: Stop the server with Ctrl+C**

Expected:
```
shutting down...
[worker] commit sync loop stopped
[worker] jira sync loop stopped
server stopped
```

The context cancellation should cleanly stop both ticker loops.

**Step 4: Commit all remaining changes**

If any files were adjusted during integration testing:

```bash
git add -A
git commit -m "feat: background sync worker complete"
```

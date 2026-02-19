# Jira Project Key Scoping Implementation Plan

> **For Claude:** REQUIRED SUB-SKILL: Use superpowers:executing-plans to implement this plan task-by-task.

**Goal:** Allow users to configure Jira project key prefixes to scope which cards are synced and displayed, and fix active sprint selection to use the most recently started sprint.

**Architecture:** Add a `jira_project_keys` field to the User model (comma-separated string). A shared helper filters card keys by prefix. The filter applies at three layers: worker (skip non-matching cards during sync), API handlers (WHERE clause on queries), and frontend (Settings page input). Empty value = no filter for backward compatibility.

**Tech Stack:** Go 1.24, GORM, Gin, React/TypeScript, RTK Query

---

### Task 1: Add FilterByProjectKeys helper with tests

**Files:**
- Create: `backend/internal/helpers/jira.go`
- Create: `backend/internal/helpers/jira_test.go`

**Step 1: Create helper with test file**

Create `backend/internal/helpers/jira_test.go`:

```go
package helpers

import "testing"

func TestFilterByProjectKeys(t *testing.T) {
	tests := []struct {
		name        string
		cardKey     string
		projectKeys string
		want        bool
	}{
		{"empty keys allows all", "PDT-123", "", true},
		{"matching single key", "PDT-123", "PDT", true},
		{"matching first of multiple", "PDT-123", "PDT,CORE", true},
		{"matching second of multiple", "CORE-456", "PDT,CORE", true},
		{"no match", "OTHER-789", "PDT,CORE", false},
		{"partial prefix no match", "PDTX-123", "PDT", false},
		{"whitespace in keys", "CORE-1", " PDT , CORE ", true},
		{"empty card key", "", "PDT", false},
		{"key without dash", "PDT", "PDT", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := FilterByProjectKeys(tt.cardKey, tt.projectKeys)
			if got != tt.want {
				t.Errorf("FilterByProjectKeys(%q, %q) = %v, want %v",
					tt.cardKey, tt.projectKeys, got, tt.want)
			}
		})
	}
}

func TestBuildProjectKeyWhereClauses(t *testing.T) {
	tests := []struct {
		name        string
		projectKeys string
		column      string
		wantClause  string
		wantArgs    int
	}{
		{"empty returns empty", "", "card_key", "", 0},
		{"single key", "PDT", "card_key", "card_key LIKE ?", 1},
		{"multiple keys", "PDT,CORE", "card_key", "(card_key LIKE ? OR card_key LIKE ?)", 2},
		{"whitespace trimmed", " PDT , CORE ", "k", "(k LIKE ? OR k LIKE ?)", 2},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			clause, args := BuildProjectKeyWhereClauses(tt.projectKeys, tt.column)
			if clause != tt.wantClause {
				t.Errorf("clause = %q, want %q", clause, tt.wantClause)
			}
			if len(args) != tt.wantArgs {
				t.Errorf("args len = %d, want %d", len(args), tt.wantArgs)
			}
		})
	}
}
```

**Step 2: Run tests to verify they fail**

Run: `cd /home/nst/GolandProjects/pdt/backend && go test ./internal/helpers/ -v`
Expected: FAIL (package/functions don't exist yet)

**Step 3: Implement the helpers**

Create `backend/internal/helpers/jira.go`:

```go
package helpers

import "strings"

// FilterByProjectKeys checks if a Jira card key matches configured project prefixes.
// Returns true if projectKeys is empty (no filter) or cardKey matches any prefix.
func FilterByProjectKeys(cardKey string, projectKeys string) bool {
	if projectKeys == "" {
		return true
	}
	for _, k := range strings.Split(projectKeys, ",") {
		prefix := strings.TrimSpace(k) + "-"
		if prefix != "-" && strings.HasPrefix(cardKey, prefix) {
			return true
		}
	}
	return false
}

// BuildProjectKeyWhereClauses builds a SQL WHERE clause for filtering by project key prefixes.
// Returns empty string and nil args if projectKeys is empty.
func BuildProjectKeyWhereClauses(projectKeys string, column string) (string, []interface{}) {
	if projectKeys == "" {
		return "", nil
	}
	keys := strings.Split(projectKeys, ",")
	var clauses []string
	var args []interface{}
	for _, k := range keys {
		trimmed := strings.TrimSpace(k)
		if trimmed == "" {
			continue
		}
		clauses = append(clauses, column+" LIKE ?")
		args = append(args, trimmed+"-%")
	}
	if len(clauses) == 0 {
		return "", nil
	}
	if len(clauses) == 1 {
		return clauses[0], args
	}
	return "(" + strings.Join(clauses, " OR ") + ")", args
}
```

**Step 4: Run tests to verify they pass**

Run: `cd /home/nst/GolandProjects/pdt/backend && go test ./internal/helpers/ -v`
Expected: PASS (all tests green)

**Step 5: Commit**

```bash
git add backend/internal/helpers/
git commit -m "feat: add FilterByProjectKeys helper with tests"
```

---

### Task 2: Add JiraProjectKeys field to User model

**Files:**
- Modify: `backend/internal/models/user.go:5-19`

**Step 1: Add the field**

Add after line 16 (`JiraUsername`), before `CreatedAt`:

```go
JiraProjectKeys string    `gorm:"type:varchar(500)" json:"jira_project_keys"`
```

The full User struct becomes:

```go
type User struct {
	ID              uint      `gorm:"primarykey" json:"id"`
	Email           string    `gorm:"type:varchar(255);uniqueIndex;not null" json:"email"`
	PasswordHash    string    `gorm:"-" json:"-"`
	Password        string    `gorm:"column:password_hash;type:varchar(255);not null" json:"-"`
	GithubToken     string    `gorm:"type:text" json:"-"`
	GitlabToken     string    `gorm:"type:text" json:"-"`
	GitlabURL       string    `gorm:"type:varchar(500)" json:"gitlab_url"`
	JiraEmail       string    `gorm:"type:varchar(255)" json:"jira_email"`
	JiraToken       string    `gorm:"type:text" json:"-"`
	JiraWorkspace   string    `gorm:"type:varchar(255)" json:"jira_workspace"`
	JiraUsername    string    `gorm:"type:varchar(255)" json:"jira_username"`
	JiraProjectKeys string    `gorm:"type:varchar(500)" json:"jira_project_keys"`
	CreatedAt       time.Time `json:"created_at"`
	UpdatedAt       time.Time `json:"updated_at"`
}
```

GORM auto-migrates the new column on app start.

**Step 2: Verify it compiles**

Run: `cd /home/nst/GolandProjects/pdt/backend && go build ./...`
Expected: No errors

**Step 3: Commit**

```bash
git add backend/internal/models/user.go
git commit -m "feat: add JiraProjectKeys field to User model"
```

---

### Task 3: Update profile handler to expose and accept JiraProjectKeys

**Files:**
- Modify: `backend/internal/handlers/user.go:17-37`

**Step 1: Update profileResponse struct**

Add to `profileResponse` (after `JiraUsername`, line ~26):

```go
JiraProjectKeys string `json:"jira_project_keys"`
```

**Step 2: Update GetProfile to include the new field**

In `GetProfile` function, add to the `profileResponse` literal (after `JiraUsername`):

```go
JiraProjectKeys: user.JiraProjectKeys,
```

**Step 3: Update updateProfileRequest struct**

Add to `updateProfileRequest`:

```go
JiraProjectKeys *string `json:"jira_project_keys"`
```

**Step 4: Update UpdateProfile to handle the new field**

In `UpdateProfile` function, add before `if len(updates) > 0`:

```go
if req.JiraProjectKeys != nil {
	updates["jira_project_keys"] = *req.JiraProjectKeys
}
```

**Step 5: Verify it compiles**

Run: `cd /home/nst/GolandProjects/pdt/backend && go build ./...`
Expected: No errors

**Step 6: Commit**

```bash
git add backend/internal/handlers/user.go
git commit -m "feat: expose jira_project_keys in profile API"
```

---

### Task 4: Fix active sprint selection (most recent) and filter cards by project keys

**Files:**
- Modify: `backend/internal/handlers/jira.go:96-107` (GetActiveSprint)
- Modify: `backend/internal/handlers/jira.go:109-158` (ListCards)

**Step 1: Fix GetActiveSprint to order by most recent start_date**

In `GetActiveSprint` (around line 100), change:

```go
// FROM:
if err := h.DB.Where("user_id = ? AND state = ?", userID, models.SprintActive).
	Preload("Cards").First(&sprint).Error; err != nil {

// TO:
if err := h.DB.Where("user_id = ? AND state = ?", userID, models.SprintActive).
	Preload("Cards").Order("start_date DESC").First(&sprint).Error; err != nil {
```

**Step 2: Filter cards by project keys in ListCards**

In `ListCards`, after fetching `dbCards` from DB (around line 155), add project key filtering.

Change the DB query from:

```go
h.DB.Where("user_id = ? AND sprint_id = ?", userID, sprint.ID).Find(&dbCards)
```

To:

```go
query := h.DB.Where("user_id = ? AND sprint_id = ?", userID, sprint.ID)

// Apply project key filter
var user models.User
if err := h.DB.First(&user, userID).Error; err == nil && user.JiraProjectKeys != "" {
	clause, args := helpers.BuildProjectKeyWhereClauses(user.JiraProjectKeys, "card_key")
	if clause != "" {
		query = query.Where(clause, args...)
	}
}

query.Find(&dbCards)
```

Add the import at the top of the file:

```go
"github.com/cds-id/pdt/backend/internal/helpers"
```

**Step 3: Also filter cards in GetActiveSprint's Preload**

The `GetActiveSprint` uses `Preload("Cards")` which loads all cards. We need to filter them too.

Change from:

```go
Preload("Cards").Order("start_date DESC").First(&sprint)
```

To:

```go
Order("start_date DESC").First(&sprint)
```

Then after loading the sprint, filter cards separately:

```go
if err := h.DB.Where("user_id = ? AND state = ?", userID, models.SprintActive).
	Order("start_date DESC").First(&sprint).Error; err != nil {
	c.JSON(http.StatusNotFound, gin.H{"error": "no active sprint found"})
	return
}

// Load cards with optional project key filter
cardQuery := h.DB.Where("sprint_id = ?", sprint.ID)
var user models.User
if err := h.DB.First(&user, userID).Error; err == nil && user.JiraProjectKeys != "" {
	clause, args := helpers.BuildProjectKeyWhereClauses(user.JiraProjectKeys, "card_key")
	if clause != "" {
		cardQuery = cardQuery.Where(clause, args...)
	}
}
cardQuery.Find(&sprint.Cards)

c.JSON(http.StatusOK, sprint)
```

**Step 4: Also fix active sprint fallback in ListCards**

The `ListCards` handler also defaults to active sprint (line 127). Fix the same ordering:

```go
// FROM:
if err := h.DB.Where("user_id = ? AND state = ?", userID, models.SprintActive).First(&sprint).Error; err != nil {

// TO:
if err := h.DB.Where("user_id = ? AND state = ?", userID, models.SprintActive).Order("start_date DESC").First(&sprint).Error; err != nil {
```

**Step 5: Verify it compiles**

Run: `cd /home/nst/GolandProjects/pdt/backend && go build ./...`
Expected: No errors

**Step 6: Commit**

```bash
git add backend/internal/handlers/jira.go
git commit -m "feat: filter cards by project keys, fix active sprint ordering"
```

---

### Task 5: Filter cards during worker sync

**Files:**
- Modify: `backend/internal/worker/jira.go:15-87`

**Step 1: Add project key filtering in SyncUserJira**

After line 30 (`client := jira.New(...)`), the user is already loaded. We have `user.JiraProjectKeys` available.

Inside the card loop (lines 63-81), add the filter check. Change:

```go
for _, card := range cards {
	jiraCard := models.JiraCard{
```

To:

```go
for _, card := range cards {
	// Skip cards not matching configured project keys
	if !helpers.FilterByProjectKeys(card.Key, user.JiraProjectKeys) {
		continue
	}

	jiraCard := models.JiraCard{
```

Add the import:

```go
"github.com/cds-id/pdt/backend/internal/helpers"
```

**Step 2: Verify it compiles**

Run: `cd /home/nst/GolandProjects/pdt/backend && go build ./...`
Expected: No errors

**Step 3: Commit**

```bash
git add backend/internal/worker/jira.go
git commit -m "feat: filter jira cards by project keys during sync"
```

---

### Task 6: Update frontend IUser interface and Settings page

**Files:**
- Modify: `frontend/src/domain/user/interfaces/user.interface.ts`
- Modify: `frontend/src/presentation/pages/SettingsPage.tsx`

**Step 1: Add field to IUser interface**

In `frontend/src/domain/user/interfaces/user.interface.ts`, add to `IUser`:

```typescript
jira_project_keys: string
```

**Step 2: Add to Settings page form data**

In `frontend/src/presentation/pages/SettingsPage.tsx`, add to `formData` initial state:

```typescript
jira_project_keys: ''
```

**Step 3: Add input field in the Jira section**

After the Jira `Username` input and before the Jira status badge div, add:

```tsx
<Input
  type="text"
  placeholder="Project keys (e.g., PDT, CORE)"
  value={formData.jira_project_keys}
  onChange={(e) => setFormData({ ...formData, jira_project_keys: e.target.value })}
  className="mb-2 bg-pdt-primary-light border-pdt-background/20 text-pdt-neutral placeholder:text-pdt-neutral/40"
/>
<p className="text-xs text-pdt-neutral/40">
  Comma-separated project key prefixes. Leave empty to show all.
</p>
```

**Step 4: Verify it builds**

Run: `cd /home/nst/GolandProjects/pdt/frontend && npx tsc --noEmit && npx vite build`
Expected: No errors, build succeeds

**Step 5: Commit**

```bash
git add frontend/src/domain/user/interfaces/user.interface.ts frontend/src/presentation/pages/SettingsPage.tsx
git commit -m "feat: add jira project keys input on settings page"
```

---

### Summary

| Task | What | Files |
|------|------|-------|
| 1 | FilterByProjectKeys helper + tests | `helpers/jira.go`, `helpers/jira_test.go` |
| 2 | JiraProjectKeys field on User model | `models/user.go` |
| 3 | Profile API exposes + accepts the field | `handlers/user.go` |
| 4 | Active sprint ordering + handler card filter | `handlers/jira.go` |
| 5 | Worker sync card filter | `worker/jira.go` |
| 6 | Frontend settings input + interface | `user.interface.ts`, `SettingsPage.tsx` |

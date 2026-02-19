# Jira Project Key Scoping

## Problem

The current Jira integration fetches ALL boards, sprints, and cards the user has access to. This causes noise when users only care about specific projects (e.g., PDT, CORE). Additionally, when multiple sprints are active across boards, `GetActiveSprint` returns an unpredictable one.

## Goal

Allow users to configure which Jira project keys to track, and ensure the "active sprint" is always the most recently started one.

## Design

### Data Model

Add one field to the `User` model:

```go
JiraProjectKeys string `gorm:"type:varchar(500)" json:"jira_project_keys"`
```

- Comma-separated project prefixes: `"PDT,CORE,TEAM"`
- Empty string = no filter (backward compatible, show everything)
- GORM auto-migrates the column
- Exposed in `profileResponse` and updatable via `updateProfileRequest`

### Backend Filtering

**Shared helper** (new file `internal/helpers/jira.go`):

```go
func FilterByProjectKeys(cardKey string, projectKeys string) bool {
    if projectKeys == "" {
        return true
    }
    for _, k := range strings.Split(projectKeys, ",") {
        if strings.HasPrefix(cardKey, strings.TrimSpace(k)+"-") {
            return true
        }
    }
    return false
}
```

**Worker** (`worker/jira.go`):
- After `FetchSprintIssues()`, filter cards through `FilterByProjectKeys` before DB storage
- Non-matching cards are skipped entirely

**Handlers** (`handlers/jira.go`):
- `ListCards`: Add WHERE clause filtering card keys by configured prefixes
- `GetActiveSprint`: Change `First()` to `Order("start_date DESC").First()` for most-recent active sprint
- `ListSprints`: No change (sprints don't have project keys)

### Frontend

**Settings page**: Add "Jira Project Keys" input in the Jira section:
- Comma-separated text input with hint text
- New `jira_project_keys` field in form data
- Sent to backend on save

**IUser interface**: Add `jira_project_keys: string`

**Jira page / Dashboard**: No changes needed. Server-side filtering handles scoping transparently.

## Scope

- Backend: User model, profile handler, jira handler, worker
- Frontend: Settings page input, IUser interface
- No new tables, no new endpoints, no breaking changes

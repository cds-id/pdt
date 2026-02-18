# Background Sync Worker Design

**Date:** 2026-02-18
**Status:** Approved

## Problem

Currently all data sync (GitHub/GitLab commits, Jira sprints/cards) is triggered manually via API calls. The user must hit POST /api/sync/commits to get fresh data. This means the dashboard is only as fresh as the last manual sync.

## Solution

An embedded background worker that automatically syncs all three data sources on configurable intervals using Go's built-in concurrency (goroutines + time.Ticker).

## Architecture

```
main.go
  ├── Scheduler.Start(ctx)        ← launches goroutines
  │     ├── commitSyncLoop()      ← ticker for GitHub/GitLab commits
  │     └── jiraSyncLoop()        ← ticker for Jira sprints/cards/details
  ├── HTTP Server (gin)           ← existing API + new status endpoint
  └── <-ctx.Done()                ← signals shutdown to both
```

The Scheduler embeds in the existing server binary. It holds `*gorm.DB` and `*crypto.Encryptor` (same deps as handlers). Shutdown is via the existing `context.Context` — no new lifecycle management needed.

## Sync Logic

### Commit Sync (default: every 15min)

1. Query all users with at least one repository
2. For each user → decrypt token → for each repo → FetchCommits (branch-aware)
3. Upsert commits with ON CONFLICT DO NOTHING
4. Update `repo.LastSyncedAt`
5. On error: log, mark repo `is_valid=false`, continue

### Jira Sync (default: every 30min)

1. Query all users with Jira configured (`jira_token != ''`)
2. For each user → decrypt credentials
3. Fetch boards → sprints → upsert to `sprints` table
4. For active sprints: fetch issues → upsert to `jira_cards` table
5. For each card: fetch issue detail with changelog → store as JSON in `jira_cards.details_json`

### Shared Code

The sync functions are extracted into `worker.SyncUserCommits()` and `worker.SyncUserJira()`. The existing `POST /api/sync/commits` handler calls the same function, so manual trigger and background sync share one code path.

### Concurrency Guard

Each loop skips if the previous run hasn't finished. Prevents pile-up when sync takes longer than the interval.

## Config

New `.env` variables:

```
SYNC_INTERVAL_COMMITS=15m     # GitHub/GitLab commit sync interval
SYNC_INTERVAL_JIRA=30m        # Jira sprint/card/detail sync interval
SYNC_ENABLED=true             # Master switch for background sync
```

Parsed as `time.Duration`. Worker doesn't start when `SYNC_ENABLED=false`.

## API Changes

### New: GET /api/sync/status (authenticated)

Returns per-user sync status:

```json
{
  "commits": {
    "last_sync": "2026-02-18T23:15:00Z",
    "next_sync": "2026-02-18T23:30:00Z",
    "status": "idle",
    "last_error": null
  },
  "jira": {
    "last_sync": "2026-02-18T23:00:00Z",
    "next_sync": "2026-02-18T23:30:00Z",
    "status": "syncing",
    "last_error": null
  }
}
```

### Existing: POST /api/sync/commits

Stays as manual trigger. Calls the same shared `SyncUserCommits()` function.

## DB Changes

- Add `details_json TEXT` column to `jira_cards` table — stores full issue detail (parent, subtasks, changelog) as JSON

## File Structure

```
internal/worker/
  scheduler.go     — Scheduler struct, Start/Stop, ticker loops
  commits.go       — SyncUserCommits() shared function
  jira.go          — SyncUserJira() shared function
  status.go        — SyncStatus type, thread-safe read/write
```

## Decisions

| Decision | Choice | Reason |
|----------|--------|--------|
| Deployment model | Embedded in server | Single binary, simple ops |
| Scheduling | time.Ticker goroutines | Zero deps, fits Go idioms |
| Jira detail storage | DB (details_json) | Survives restarts, queryable |
| Intervals | Configurable via .env | Flexibility without code changes |
| Observability | Status endpoint + manual trigger | Lightweight, sufficient for personal tool |

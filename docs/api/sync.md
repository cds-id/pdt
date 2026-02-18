# Sync API

Manually trigger commit synchronization and check background sync status. All endpoints require authentication.

**Headers (all endpoints):**

| Header | Value | Required |
|--------|-------|----------|
| `Authorization` | `Bearer <token>` | Yes |

## Endpoints

### `POST /api/sync/commits`

Manually trigger a commit sync across all tracked repositories. Fetches commits from the last 30 days from GitHub/GitLab APIs and stores them in the database. Jira card keys are automatically extracted from commit messages using the pattern `[A-Z][A-Z0-9]+-\d+`.

**Request Body:** None

**Response (200 OK) — with results:**

```json
{
  "results": [
    {
      "repo_id": 1,
      "repo_name": "myorg/my-repo",
      "provider": "github",
      "new_commits": 5,
      "total_fetched": 42,
      "error": ""
    },
    {
      "repo_id": 2,
      "repo_name": "myorg/other-repo",
      "provider": "gitlab",
      "new_commits": 0,
      "total_fetched": 10,
      "error": ""
    }
  ]
}
```

**Response (200 OK) — no repositories:**

```json
{
  "message": "no repositories to sync",
  "results": []
}
```

| Field | Type | Description |
|-------|------|-------------|
| `repo_id` | integer | Repository ID |
| `repo_name` | string | `owner/name` format |
| `provider` | string | `github` or `gitlab` |
| `new_commits` | integer | Newly inserted commits (deduped by SHA) |
| `total_fetched` | integer | Total commits fetched from API |
| `error` | string | Error message (empty if successful) |

**Error Responses:**

| Status | Body | Condition |
|--------|------|-----------|
| 500 | `{"error": "..."}` | User not found or system error |

---

### `GET /api/sync/status`

Get the background sync status for the current user. Shows last sync time, next scheduled sync, and current state for both commit and Jira sync workers.

**Response (200 OK):**

```json
{
  "commits": {
    "last_sync": "2026-02-19T00:15:00Z",
    "next_sync": "2026-02-19T00:30:00Z",
    "status": "idle",
    "last_error": null
  },
  "jira": {
    "last_sync": "2026-02-19T00:00:00Z",
    "next_sync": "2026-02-19T00:30:00Z",
    "status": "syncing",
    "last_error": null
  }
}
```

| Field | Type | Description |
|-------|------|-------------|
| `last_sync` | string (ISO 8601) or null | Timestamp of last completed sync |
| `next_sync` | string (ISO 8601) or null | Timestamp of next scheduled sync |
| `status` | string | `idle` or `syncing` |
| `last_error` | string or null | Error from last sync attempt (null if successful) |

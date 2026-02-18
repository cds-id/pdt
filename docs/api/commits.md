# Commits API

Query synced commits and manage their links to Jira cards. All endpoints require authentication.

**Headers (all endpoints):**

| Header | Value | Required |
|--------|-------|----------|
| `Authorization` | `Bearer <token>` | Yes |
| `Content-Type` | `application/json` | For POST |

## Endpoints

### `GET /api/commits`

List all commits across the user's tracked repositories. Supports filtering by repository, Jira card, and link status.

**Query Parameters:**

| Param | Type | Description | Required |
|-------|------|-------------|----------|
| `repo_id` | integer | Filter by repository ID | No |
| `jira_card_key` | string | Filter by Jira card key (e.g., `PROJ-123`) | No |
| `has_link` | string | Filter by link status: `true` or `false` | No |

**Response (200 OK):**

```json
[
  {
    "id": 42,
    "repo_id": 1,
    "sha": "abc123def456789...",
    "message": "PROJ-123: implement user authentication",
    "author": "John Doe",
    "author_email": "john@example.com",
    "branch": "main",
    "date": "2026-02-18T14:30:00Z",
    "jira_card_key": "PROJ-123",
    "has_link": true,
    "created_at": "2026-02-19T00:10:00Z"
  }
]
```

Results are ordered by commit date (newest first). Returns an empty array `[]` if no commits match.

---

### `GET /api/commits/missing`

List commits that are not linked to any Jira card (`has_link = false`). Useful for identifying work that hasn't been tracked in Jira.

**Response (200 OK):**

```json
[
  {
    "id": 55,
    "repo_id": 1,
    "sha": "def789abc123456...",
    "message": "fix login button alignment",
    "author": "Jane Smith",
    "author_email": "jane@example.com",
    "branch": "fix/login-ui",
    "date": "2026-02-18T16:00:00Z",
    "jira_card_key": "",
    "has_link": false,
    "created_at": "2026-02-19T00:10:00Z"
  }
]
```

---

### `POST /api/commits/:sha/link`

Manually link a commit to a Jira card. Creates a `commit_card_link` record and sets `has_link = true` on the commit.

**URL Parameters:**

| Param | Type | Description |
|-------|------|-------------|
| `sha` | string | Full commit SHA |

**Request Body:**

```json
{
  "jira_card_key": "PROJ-456"
}
```

| Field | Type | Validation | Required |
|-------|------|------------|----------|
| `jira_card_key` | string | Non-empty Jira key | Yes |

**Response (201 Created):**

```json
{
  "id": 1,
  "commit_id": 55,
  "jira_card_key": "PROJ-456",
  "linked_at": "2026-02-19T01:00:00Z"
}
```

**Error Responses:**

| Status | Body | Condition |
|--------|------|-----------|
| 400 | `{"error": "..."}` | Missing `jira_card_key` |
| 404 | `{"error": "commit not found"}` | SHA doesn't exist or belongs to another user's repo |
| 500 | `{"error": "failed to create link"}` | Database error |

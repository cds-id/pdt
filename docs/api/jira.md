# Jira API

Access Jira sprints and cards. These endpoints fetch live data from the Jira Cloud API and sync it to the local database. All endpoints require authentication and a configured Jira integration.

**Headers (all endpoints):**

| Header | Value | Required |
|--------|-------|----------|
| `Authorization` | `Bearer <token>` | Yes |

**Prerequisites:** Configure Jira integration via `PUT /api/user/profile` with `jira_email`, `jira_token`, and `jira_workspace`.

## Endpoints

### `GET /api/jira/sprints`

List all sprints across all Jira boards. Fetches live data from the Jira API, syncs to the local database, then returns stored sprints.

**Response (200 OK):**

```json
[
  {
    "id": 1,
    "user_id": 1,
    "jira_sprint_id": "42",
    "name": "Sprint 10 - Q1 Goals",
    "state": "active",
    "start_date": "2026-02-10T00:00:00Z",
    "end_date": "2026-02-24T00:00:00Z",
    "created_at": "2026-02-19T00:00:00Z"
  },
  {
    "id": 2,
    "user_id": 1,
    "jira_sprint_id": "41",
    "name": "Sprint 9",
    "state": "closed",
    "start_date": "2026-01-27T00:00:00Z",
    "end_date": "2026-02-10T00:00:00Z",
    "created_at": "2026-02-19T00:00:00Z"
  }
]
```

| Field | Type | Values |
|-------|------|--------|
| `state` | string | `active`, `closed`, `future` |

Results are ordered by start date (newest first).

**Error Responses:**

| Status | Body | Condition |
|--------|------|-----------|
| 400 | `{"error": "jira not configured"}` | Missing Jira token/workspace/email |
| 500 | `{"error": "failed to fetch boards: ..."}` | Jira API error |

---

### `GET /api/jira/sprints/:id`

Get a specific sprint with its associated cards.

**URL Parameters:**

| Param | Type | Description |
|-------|------|-------------|
| `id` | integer | Local sprint ID (not Jira sprint ID) |

**Response (200 OK):**

```json
{
  "id": 1,
  "user_id": 1,
  "jira_sprint_id": "42",
  "name": "Sprint 10 - Q1 Goals",
  "state": "active",
  "start_date": "2026-02-10T00:00:00Z",
  "end_date": "2026-02-24T00:00:00Z",
  "created_at": "2026-02-19T00:00:00Z",
  "cards": [
    {
      "id": 1,
      "user_id": 1,
      "key": "PROJ-123",
      "summary": "Implement user authentication",
      "status": "In Progress",
      "assignee": "John Doe",
      "sprint_id": 1,
      "created_at": "2026-02-19T00:00:00Z"
    }
  ]
}
```

**Error Responses:**

| Status | Body | Condition |
|--------|------|-----------|
| 404 | `{"error": "sprint not found"}` | ID doesn't exist or belongs to another user |

---

### `GET /api/jira/active-sprint`

Get the currently active sprint with its cards.

**Response (200 OK):**

```json
{
  "id": 1,
  "user_id": 1,
  "jira_sprint_id": "42",
  "name": "Sprint 10 - Q1 Goals",
  "state": "active",
  "start_date": "2026-02-10T00:00:00Z",
  "end_date": "2026-02-24T00:00:00Z",
  "created_at": "2026-02-19T00:00:00Z",
  "cards": [...]
}
```

**Error Responses:**

| Status | Body | Condition |
|--------|------|-----------|
| 404 | `{"error": "no active sprint found"}` | No sprint with `state = "active"` |

---

### `GET /api/jira/cards`

List Jira cards. Fetches live data from the Jira API for the specified or active sprint, syncs to DB, then returns stored cards.

**Query Parameters:**

| Param | Type | Description | Required |
|-------|------|-------------|----------|
| `sprint_id` | integer | Local sprint ID to filter by | No (defaults to active sprint) |

**Response (200 OK):**

```json
[
  {
    "id": 1,
    "user_id": 1,
    "key": "PROJ-123",
    "summary": "Implement user authentication",
    "status": "In Progress",
    "assignee": "John Doe",
    "sprint_id": 1,
    "created_at": "2026-02-19T00:00:00Z"
  }
]
```

**Error Responses:**

| Status | Body | Condition |
|--------|------|-----------|
| 400 | `{"error": "jira not configured"}` | Missing Jira integration |
| 404 | `{"error": "sprint not found"}` | Specified sprint ID not found |
| 404 | `{"error": "no active sprint found"}` | No active sprint (when no `sprint_id` provided) |
| 500 | `{"error": "failed to fetch cards: ..."}` | Jira API error |

---

### `GET /api/jira/cards/:key`

Get detailed information about a Jira card, including linked commits, subtasks with their commits, and changelog. Fetches issue details live from the Jira API.

**URL Parameters:**

| Param | Type | Description |
|-------|------|-------------|
| `key` | string | Jira card key (e.g., `PROJ-123`) |

**Response (200 OK):**

```json
{
  "key": "PROJ-123",
  "summary": "Implement user authentication",
  "status": "In Progress",
  "assignee": "John Doe",
  "issue_type": "Story",
  "parent": {
    "key": "PROJ-100",
    "summary": "Authentication Epic"
  },
  "commits": [
    {
      "id": 42,
      "repo_id": 1,
      "sha": "abc123...",
      "message": "PROJ-123: add login handler",
      "author": "John Doe",
      "author_email": "john@example.com",
      "branch": "feat/auth",
      "date": "2026-02-18T14:30:00Z",
      "jira_card_key": "PROJ-123",
      "has_link": true,
      "created_at": "2026-02-19T00:10:00Z"
    }
  ],
  "subtasks": [
    {
      "key": "PROJ-124",
      "summary": "Add password validation",
      "status": "Done",
      "type": "Sub-task",
      "commits": [...]
    }
  ],
  "changelog": [...]
}
```

**Error Responses:**

| Status | Body | Condition |
|--------|------|-----------|
| 404 | `{"error": "card not found"}` | No card data and no linked commits found |

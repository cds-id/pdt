# User Profile API

Manage user profile and integration settings. All endpoints require authentication.

**Headers (all endpoints):**

| Header | Value | Required |
|--------|-------|----------|
| `Authorization` | `Bearer <token>` | Yes |
| `Content-Type` | `application/json` | For PUT/POST |

## Endpoints

### `GET /api/user/profile`

Get the current user's profile with integration configuration status.

**Response (200 OK):**

```json
{
  "id": 1,
  "email": "user@example.com",
  "has_github_token": true,
  "has_gitlab_token": false,
  "gitlab_url": "",
  "jira_email": "user@company.com",
  "has_jira_token": true,
  "jira_workspace": "myworkspace.atlassian.net",
  "jira_username": "myusername"
}
```

> Note: Actual token values are never returned. Only boolean flags indicate whether tokens are configured.

**Error Responses:**

| Status | Body | Condition |
|--------|------|-----------|
| 401 | `{"error": "..."}` | Missing or invalid JWT |
| 404 | `{"error": "user not found"}` | User deleted |

---

### `PUT /api/user/profile`

Update integration tokens and settings. All fields are optional — only provided fields are updated. Tokens are encrypted with AES-256-GCM before storage.

**Request Body (all fields optional):**

```json
{
  "github_token": "ghp_xxxxxxxxxxxx",
  "gitlab_token": "glpat-xxxxxxxxxxxx",
  "gitlab_url": "https://gitlab.com",
  "jira_email": "user@company.com",
  "jira_token": "ATATT3xxxxxxxxxxx",
  "jira_workspace": "myworkspace.atlassian.net",
  "jira_username": "myusername"
}
```

| Field | Type | Description |
|-------|------|-------------|
| `github_token` | string | GitHub Personal Access Token |
| `gitlab_token` | string | GitLab Personal Access Token |
| `gitlab_url` | string | GitLab instance URL (default: `https://gitlab.com`) |
| `jira_email` | string | Atlassian account email |
| `jira_token` | string | Jira API token |
| `jira_workspace` | string | Jira workspace domain (e.g., `myteam.atlassian.net`) |
| `jira_username` | string | Jira display username |

**Response (200 OK):**

```json
{
  "message": "profile updated"
}
```

**Error Responses:**

| Status | Body | Condition |
|--------|------|-----------|
| 400 | `{"error": "..."}` | Invalid JSON body |
| 404 | `{"error": "user not found"}` | User deleted |
| 500 | `{"error": "failed to encrypt token"}` | Encryption error |

---

### `POST /api/user/profile/validate`

Check which integrations are configured for the current user.

**Request Body:** None

**Response (200 OK):**

```json
{
  "github": {
    "configured": true
  },
  "gitlab": {
    "configured": false
  },
  "jira": {
    "configured": true
  }
}
```

> Jira is considered "configured" when both `jira_token` and `jira_workspace` are set.

**Error Responses:**

| Status | Body | Condition |
|--------|------|-----------|
| 404 | `{"error": "user not found"}` | User deleted |

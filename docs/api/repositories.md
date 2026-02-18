# Repositories API

Manage tracked Git repositories. Repositories are auto-detected as GitHub or GitLab based on the URL. All endpoints require authentication.

**Headers (all endpoints):**

| Header | Value | Required |
|--------|-------|----------|
| `Authorization` | `Bearer <token>` | Yes |
| `Content-Type` | `application/json` | For POST |

## Endpoints

### `GET /api/repos`

List all repositories tracked by the current user.

**Response (200 OK):**

```json
[
  {
    "id": 1,
    "user_id": 1,
    "name": "my-repo",
    "owner": "myorg",
    "provider": "github",
    "url": "https://github.com/myorg/my-repo",
    "is_valid": true,
    "last_synced_at": "2026-02-19T00:15:00Z",
    "created_at": "2026-02-18T10:00:00Z"
  }
]
```

Returns an empty array `[]` if no repositories are tracked.

---

### `POST /api/repos`

Add a new repository to track. The URL is parsed to extract owner, name, and provider (GitHub or GitLab).

**Request Body:**

```json
{
  "url": "https://github.com/myorg/my-repo"
}
```

| Field | Type | Validation | Required |
|-------|------|------------|----------|
| `url` | string | Valid URL, must contain owner/name path | Yes |

**URL Parsing Rules:**
- URLs containing `github.com` are detected as GitHub
- All other URLs are detected as GitLab (supports self-hosted instances)
- `.git` suffix is automatically stripped
- Path must have exactly `owner/name` format

**Response (201 Created):**

```json
{
  "id": 2,
  "user_id": 1,
  "name": "my-repo",
  "owner": "myorg",
  "provider": "github",
  "url": "https://github.com/myorg/my-repo",
  "is_valid": true,
  "last_synced_at": null,
  "created_at": "2026-02-19T00:30:00Z"
}
```

**Error Responses:**

| Status | Body | Condition |
|--------|------|-----------|
| 400 | `{"error": "..."}` | Invalid or unparseable URL |
| 409 | `{"error": "repository already tracked"}` | Duplicate owner/name/provider combo |
| 500 | `{"error": "failed to add repository"}` | Database error |

---

### `DELETE /api/repos/:id`

Delete a tracked repository and all its associated commits.

**URL Parameters:**

| Param | Type | Description |
|-------|------|-------------|
| `id` | integer | Repository ID |

**Response (200 OK):**

```json
{
  "message": "repository removed"
}
```

**Error Responses:**

| Status | Body | Condition |
|--------|------|-----------|
| 404 | `{"error": "repository not found"}` | ID doesn't exist or belongs to another user |

---

### `POST /api/repos/:id/validate`

Check the validation status of a repository. Full validation (API access check) is performed during sync.

**URL Parameters:**

| Param | Type | Description |
|-------|------|-------------|
| `id` | integer | Repository ID |

**Response (200 OK):**

```json
{
  "id": 1,
  "url": "https://github.com/myorg/my-repo",
  "provider": "github",
  "is_valid": true,
  "message": "validation will be performed during sync"
}
```

**Error Responses:**

| Status | Body | Condition |
|--------|------|-----------|
| 404 | `{"error": "repository not found"}` | ID doesn't exist or belongs to another user |

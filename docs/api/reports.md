# Reports API

Generate, manage, and customize daily reports. Reports aggregate commit and Jira card data for a given date and render them using customizable Go templates. All endpoints require authentication.

**Headers (all endpoints):**

| Header | Value | Required |
|--------|-------|----------|
| `Authorization` | `Bearer <token>` | Yes |
| `Content-Type` | `application/json` | For POST/PUT |

## Report Endpoints

### `POST /api/reports/generate`

Generate a daily report for a specific date. Aggregates commits and Jira cards, renders using a template, and optionally uploads to Cloudflare R2. If a report for the same date already exists, it is updated (upsert).

**Request Body:**

```json
{
  "date": "2026-02-18",
  "template_id": 1
}
```

| Field | Type | Description | Required |
|-------|------|-------------|----------|
| `date` | string | Date in `YYYY-MM-DD` format | No (defaults to today) |
| `template_id` | integer | Template ID to use | No (uses default template) |

**Response (201 Created) — new report:**

```json
{
  "id": 1,
  "user_id": 1,
  "template_id": 1,
  "date": "2026-02-18",
  "title": "Daily Report — Wednesday, 18 February 2026",
  "content": "# Daily Report\n\n**Date:** 2026-02-18\n...",
  "file_url": "https://r2domain.com/reports/1/2026-02-18.md",
  "created_at": "2026-02-19T00:30:00Z"
}
```

**Response (200 OK) — updated existing report:**

Same structure as above, with updated content.

> `file_url` is empty if Cloudflare R2 is not configured.

**Error Responses:**

| Status | Body | Condition |
|--------|------|-----------|
| 400 | `{"error": "invalid request body"}` | Malformed JSON |
| 400 | `{"error": "invalid date format, use YYYY-MM-DD"}` | Bad date format |
| 400 | `{"error": "template error: ..."}` | Template rendering failed |
| 500 | `{"error": "..."}` | Data aggregation or DB error |

---

### `GET /api/reports`

List reports with optional date range filtering.

**Query Parameters:**

| Param | Type | Description | Required |
|-------|------|-------------|----------|
| `from` | string | Start date `YYYY-MM-DD` (inclusive) | No |
| `to` | string | End date `YYYY-MM-DD` (inclusive) | No |

**Response (200 OK):**

```json
[
  {
    "id": 1,
    "user_id": 1,
    "template_id": 1,
    "date": "2026-02-18",
    "title": "Daily Report — Wednesday, 18 February 2026",
    "content": "# Daily Report\n...",
    "file_url": "https://r2domain.com/reports/1/2026-02-18.md",
    "created_at": "2026-02-19T00:30:00Z"
  }
]
```

Results are ordered by date (newest first).

---

### `GET /api/reports/:id`

Get a single report by ID.

**URL Parameters:**

| Param | Type | Description |
|-------|------|-------------|
| `id` | integer | Report ID |

**Response (200 OK):**

```json
{
  "id": 1,
  "user_id": 1,
  "template_id": 1,
  "date": "2026-02-18",
  "title": "Daily Report — Wednesday, 18 February 2026",
  "content": "# Daily Report\n...",
  "file_url": "https://r2domain.com/reports/1/2026-02-18.md",
  "created_at": "2026-02-19T00:30:00Z"
}
```

**Error Responses:**

| Status | Body | Condition |
|--------|------|-----------|
| 404 | `{"error": "report not found"}` | ID doesn't exist or belongs to another user |

---

### `DELETE /api/reports/:id`

Delete a report.

**URL Parameters:**

| Param | Type | Description |
|-------|------|-------------|
| `id` | integer | Report ID |

**Response (200 OK):**

```json
{
  "message": "report deleted"
}
```

**Error Responses:**

| Status | Body | Condition |
|--------|------|-----------|
| 404 | `{"error": "report not found"}` | ID doesn't exist or belongs to another user |

---

## Template Endpoints

### `GET /api/reports/templates`

List all report templates for the current user.

**Response (200 OK):**

```json
[
  {
    "id": 1,
    "user_id": 1,
    "name": "Standard Daily Report",
    "content": "# Daily Report — {{.Title}}\n\n...",
    "is_default": true,
    "created_at": "2026-02-18T10:00:00Z",
    "updated_at": "2026-02-18T10:00:00Z"
  }
]
```

Results are ordered by creation date (newest first).

---

### `POST /api/reports/templates`

Create a new report template. Templates use Go `text/template` syntax.

**Request Body:**

```json
{
  "name": "My Custom Template",
  "content": "# {{.Title}}\n\n**Date:** {{.Date}}\n**Author:** {{.Email}}\n\n## Commits\n{{range .Commits}}\n- {{.SHA | slice 0 7}} {{.Message}}\n{{end}}",
  "is_default": true
}
```

| Field | Type | Description | Required |
|-------|------|-------------|----------|
| `name` | string | Template name | Yes |
| `content` | string | Go template content | Yes |
| `is_default` | boolean | Set as default template | No (default: `false`) |

> Setting `is_default: true` will unset any existing default template for this user.

**Response (201 Created):**

```json
{
  "id": 2,
  "user_id": 1,
  "name": "My Custom Template",
  "content": "# {{.Title}}\n...",
  "is_default": true,
  "created_at": "2026-02-19T01:00:00Z",
  "updated_at": "2026-02-19T01:00:00Z"
}
```

**Error Responses:**

| Status | Body | Condition |
|--------|------|-----------|
| 400 | `{"error": "name and content are required"}` | Missing required fields |

---

### `PUT /api/reports/templates/:id`

Update an existing template. All fields are optional.

**URL Parameters:**

| Param | Type | Description |
|-------|------|-------------|
| `id` | integer | Template ID |

**Request Body (all fields optional):**

```json
{
  "name": "Updated Template Name",
  "content": "# Updated content...",
  "is_default": true
}
```

| Field | Type | Description |
|-------|------|-------------|
| `name` | string | New template name |
| `content` | string | New template content |
| `is_default` | boolean | Set as default template |

**Response (200 OK):**

```json
{
  "id": 2,
  "user_id": 1,
  "name": "Updated Template Name",
  "content": "# Updated content...",
  "is_default": true,
  "created_at": "2026-02-19T01:00:00Z",
  "updated_at": "2026-02-19T01:30:00Z"
}
```

**Error Responses:**

| Status | Body | Condition |
|--------|------|-----------|
| 400 | `{"error": "invalid request"}` | Malformed JSON |
| 404 | `{"error": "template not found"}` | ID doesn't exist or belongs to another user |

---

### `DELETE /api/reports/templates/:id`

Delete a report template.

**URL Parameters:**

| Param | Type | Description |
|-------|------|-------------|
| `id` | integer | Template ID |

**Response (200 OK):**

```json
{
  "message": "template deleted"
}
```

**Error Responses:**

| Status | Body | Condition |
|--------|------|-----------|
| 404 | `{"error": "template not found"}` | ID doesn't exist or belongs to another user |

---

### `POST /api/reports/templates/preview`

Preview a template rendering with real data without saving a report.

**Request Body:**

```json
{
  "content": "# {{.Title}}\n\nTotal commits: {{.Stats.TotalCommits}}",
  "date": "2026-02-18"
}
```

| Field | Type | Description | Required |
|-------|------|-------------|----------|
| `content` | string | Template content to preview | Yes |
| `date` | string | Date for data context (`YYYY-MM-DD`) | No (defaults to today) |

**Response (200 OK):**

```json
{
  "rendered": "# Daily Report — Wednesday, 18 February 2026\n\nTotal commits: 15",
  "stats": {
    "total_commits": 15,
    "total_cards": 8
  }
}
```

**Error Responses:**

| Status | Body | Condition |
|--------|------|-----------|
| 400 | `{"error": "content is required"}` | Missing content field |
| 400 | `{"error": "invalid date format"}` | Bad date format |
| 400 | `{"error": "template error: ..."}` | Template syntax error |
| 500 | `{"error": "..."}` | Data aggregation error |

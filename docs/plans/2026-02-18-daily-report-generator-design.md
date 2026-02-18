# Daily Report Generator Design

**Date:** 2026-02-18
**Status:** Approved

## Problem

Users (contractors/freelancers) need to generate formal daily reports to release their fees. Currently they manually compile commit logs and Jira task lists. This is the main feature of PDT — automating that process.

## Solution

A template-based daily report generator that:
- Aggregates commits and Jira cards for a given date
- Renders configurable Markdown/HTML reports using Go `text/template`
- Stores templates and generated reports in DB
- Auto-generates reports daily via the background worker
- Exposes API for on-demand generation and report history

## Data Model

### report_templates

| Column | Type | Description |
|--------|------|-------------|
| id | uint (PK) | |
| user_id | uint (FK, index) | Owner |
| name | varchar(255) | Template name (e.g. "Client A format") |
| content | text | Markdown with Go text/template syntax |
| is_default | bool | User's default template for auto-generation |
| created_at | timestamp | |
| updated_at | timestamp | |

System provides a built-in default template. Users can create custom templates.

### reports

| Column | Type | Description |
|--------|------|-------------|
| id | uint (PK) | |
| user_id | uint (FK, index) | |
| template_id | uint (FK) | Template used |
| date | date (index) | Report date |
| title | varchar(500) | Generated title |
| content | text | Rendered markdown output |
| created_at | timestamp | |

Unique constraint on (user_id, date, template_id) — one report per date per template.

## Template System

### Data Context

Templates receive this data:

```go
type ReportData struct {
    Date            string         // "2026-02-18"
    DateFormatted   string         // "Tuesday, 18 February 2026"
    Author          string         // user email
    Cards           []CardReport   // Jira cards with commits
    UnlinkedCommits []CommitReport // commits without Jira key
    Stats           ReportStats    // summary numbers
}

type CardReport struct {
    Key     string
    Summary string
    Status  string
    Commits []CommitReport
}

type CommitReport struct {
    SHA     string   // short SHA (8 chars)
    Message string   // first line only
    Branch  string
    Repo    string   // owner/name
    Time    string   // "14:30"
}

type ReportStats struct {
    TotalCommits int
    TotalCards   int
    Repos        []string
}
```

### Default Template

```markdown
# Daily Report — {{.DateFormatted}}

**Author:** {{.Author}}

## Summary
- **Commits:** {{.Stats.TotalCommits}}
- **Jira Cards:** {{.Stats.TotalCards}}
- **Repositories:** {{range $i, $r := .Stats.Repos}}{{if $i}}, {{end}}{{$r}}{{end}}

## Work Details
{{range .Cards}}
### {{.Key}} — {{.Summary}}
**Status:** {{.Status}}
{{range .Commits}}
- `{{.SHA}}` {{.Message}} ({{.Branch}}, {{.Time}})
{{end}}
{{end}}
{{if .UnlinkedCommits}}
## Other Commits
{{range .UnlinkedCommits}}
- `{{.SHA}}` {{.Message}} ({{.Repo}}/{{.Branch}}, {{.Time}})
{{end}}
{{end}}
```

## API Endpoints

### Report Generation

| Method | Path | Description |
|--------|------|-------------|
| POST | /api/reports/generate | Generate report for a date |
| GET | /api/reports | List generated reports |
| GET | /api/reports/:id | Get specific report |
| DELETE | /api/reports/:id | Delete a report |

**POST /api/reports/generate** body:
```json
{
  "date": "2026-02-18",
  "template_id": 1
}
```
If `template_id` omitted, uses user's default template. If no custom template exists, uses built-in default.

**GET /api/reports** query params: `?from=2026-02-01&to=2026-02-18`

### Template Management

| Method | Path | Description |
|--------|------|-------------|
| GET | /api/reports/templates | List user's templates |
| POST | /api/reports/templates | Create template |
| PUT | /api/reports/templates/:id | Update template |
| DELETE | /api/reports/templates/:id | Delete template |
| POST | /api/reports/templates/preview | Preview with today's data |

**POST /api/reports/templates** body:
```json
{
  "name": "Client A Format",
  "content": "# Report for {{.DateFormatted}}\n...",
  "is_default": true
}
```

**POST /api/reports/templates/preview** body:
```json
{
  "content": "# Test\n{{.Stats.TotalCommits}} commits"
}
```
Returns rendered output without saving.

## Worker Integration

New ticker in the existing Scheduler:
- Configurable time: `REPORT_AUTO_TIME=23:00` (local time)
- Checks once per minute if it's past the configured time and no report exists for today
- Generates report for each user using their default template
- Skips if report for today already exists

## Config

```
REPORT_AUTO_GENERATE=true
REPORT_AUTO_TIME=23:00
```

## File Structure

```
internal/models/report.go          — ReportTemplate + Report models
internal/services/report/report.go — Report generator (template rendering, data aggregation)
internal/handlers/report.go        — API handlers for reports + templates
internal/worker/reports.go         — Auto-generation logic for scheduler
```

## Decisions

| Decision | Choice | Reason |
|----------|--------|--------|
| Template engine | Go text/template | Zero deps, powerful enough |
| Output format | Markdown (renderable to HTML) | Light, printable, flexible |
| Storage | DB (reports + templates tables) | History, per-user, queryable |
| Auto-generation | Worker ticker at configured time | Users get report ready each morning |
| Default template | Built-in constant in code | Works out of the box, no setup needed |

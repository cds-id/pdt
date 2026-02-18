# PDT API Documentation & README Design

**Date:** 2026-02-19
**Status:** Approved

## Goal

Create comprehensive API documentation, architecture overview, setup guide, and README for the PDT backend. Target audience: both internal team developers and frontend/API consumers.

## File Structure

```
backend/README.md                    -> Project overview, tech stack, quick start, links
docs/
  setup.md                           -> Full onboarding (env, DB, tokens)
  architecture.md                    -> System architecture, data flow, workers
  api/
    auth.md                          -> POST /register, /login
    user.md                          -> GET/PUT /profile, POST /validate
    repositories.md                  -> CRUD repos, validate
    sync.md                          -> Trigger sync, get status
    commits.md                       -> List/filter commits, link to Jira
    jira.md                          -> Sprints, cards, active sprint
    reports.md                       -> Reports CRUD, templates CRUD, preview
```

## API Doc Format (per domain file)

Each API doc uses this consistent structure:

```
# Domain Name

Brief description.

## Endpoints

### `METHOD /api/path`

Description.

**Headers:**
| Header | Value | Required |
|--------|-------|----------|

**Query Parameters:** (if applicable)
| Param | Type | Description | Required |
|-------|------|-------------|----------|

**Request Body:**
JSON example with field descriptions

**Response (status):**
JSON example

**Error Responses:**
| Status | Description |
|--------|-------------|
```

## README.md Content

- Project name + one-line description
- Tech stack table (Go 1.24, Gin, GORM, MySQL, JWT, AES-256-GCM, Cloudflare R2)
- Quick start (clone, .env, run)
- Links to all documentation
- Project structure tree
- License/contributing info

## Architecture Doc Content

- System overview (ASCII diagram)
- Data flow: User -> API -> DB / External APIs
- Background worker flow: Scheduler -> Commit sync / Jira sync / Report gen
- Security model: JWT, AES-256-GCM encryption, bcrypt
- Database ERD (ASCII)
- Integration points: GitHub REST API, GitLab REST API, Jira Cloud REST API

## Setup Guide Content

- Prerequisites (Go 1.24+, MySQL 8.0+)
- Database setup SQL commands
- Environment variables table (name, description, required/optional, default)
- Token acquisition guides (GitHub PAT, GitLab PAT, Jira API token)
- Cloudflare R2 setup (optional)
- Running the server
- Verifying the installation

## Security Considerations for Commit

- `pdt_credentials_20260218_230441.txt` MUST be added to .gitignore
- `.env` already in .gitignore (confirmed)
- `backend/bin/` already in .gitignore (confirmed)
- No hardcoded secrets in source code (tokens encrypted with AES-256-GCM)

## Implementation Tasks

1. Add `pdt_credentials_20260218_230441.txt` to .gitignore
2. Write `docs/setup.md`
3. Write `docs/architecture.md`
4. Write `docs/api/auth.md`
5. Write `docs/api/user.md`
6. Write `docs/api/repositories.md`
7. Write `docs/api/sync.md`
8. Write `docs/api/commits.md`
9. Write `docs/api/jira.md`
10. Write `docs/api/reports.md`
11. Write `backend/README.md`
12. Review all files for sensitive data
13. Commit with semantic message

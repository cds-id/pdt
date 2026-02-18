# Architecture

## System Overview

```
                                ┌──────────────┐
                                │   Client     │
                                │ (Frontend /  │
                                │   Mobile)    │
                                └──────┬───────┘
                                       │ HTTPS
                                       ▼
                              ┌─────────────────┐
                              │   Gin HTTP API   │
                              │   :8080          │
                              ├─────────────────┤
                              │  JWT Middleware  │
                              └────────┬────────┘
                                       │
                 ┌─────────────────────┼─────────────────────┐
                 │                     │                     │
                 ▼                     ▼                     ▼
          ┌─────────────┐     ┌──────────────┐     ┌──────────────┐
          │  Handlers    │     │  Background  │     │   Services   │
          │  (API Logic) │     │  Worker      │     │  (External)  │
          └──────┬───────┘     │  Scheduler   │     ├──────────────┤
                 │             └──────┬───────┘     │ GitHub API   │
                 │                    │             │ GitLab API   │
                 ▼                    │             │ Jira API     │
          ┌─────────────┐            │             │ Cloudflare   │
          │   MySQL      │◄───────────┘             │ R2 Storage   │
          │   (GORM)     │◄────────────────────────┘              │
          └─────────────┘                          └──────────────┘
```

## Request Flow

1. Client sends HTTP request with `Authorization: Bearer <JWT>` header
2. Gin router matches the route
3. JWT middleware validates the token and extracts `user_id`
4. Handler processes the request:
   - Validates request body (JSON binding)
   - Queries database (GORM) scoped to `user_id`
   - Calls external services if needed (GitHub/GitLab/Jira APIs)
   - Returns JSON response

## Background Worker

The scheduler runs three independent loops in goroutines:

```
┌──────────────────────────────────────────────────────────────┐
│                    Background Scheduler                       │
├──────────────────┬──────────────────┬────────────────────────┤
│  Commit Sync     │  Jira Sync       │  Report Auto-Gen       │
│  every 15m       │  every 30m       │  daily at 23:00        │
│                  │                  │                        │
│  For each user:  │  For each user:  │  For each user:        │
│  1. Get repos    │  1. Get boards   │  1. Build report data  │
│  2. Fetch commits│  2. Get sprints  │  2. Render template    │
│     from GitHub/ │  3. Get cards    │  3. Upload to R2       │
│     GitLab API   │  4. Upsert to DB │  4. Save to DB         │
│  3. Extract Jira │                  │                        │
│     keys from    │                  │                        │
│     messages     │                  │                        │
│  4. Upsert to DB │                  │                        │
└──────────────────┴──────────────────┴────────────────────────┘
```

- Each loop has a **concurrency guard** (atomic bool) to prevent overlapping runs
- Worker iterates over all users with configured integrations
- Graceful shutdown via `context.Context` on `SIGINT`/`SIGTERM`

## Security Model

### Authentication

- **JWT (HS256)** — tokens issued on login/register, validated by middleware
- **bcrypt** — password hashing with default cost
- Configurable expiry (default: 72 hours)

### Encryption at Rest

- **AES-256-GCM** — all integration tokens (GitHub, GitLab, Jira) are encrypted before storage
- 64 hex character key (32 bytes) via `ENCRYPTION_KEY` env var
- Random nonce per encryption operation
- Hex-encoded ciphertext stored in database

### Authorization

- **Row-level isolation** — all database queries scoped by `user_id`
- No role-based access control — single user role
- Users can only access their own data

## Database Schema

```
┌──────────────┐       ┌──────────────────┐
│    users     │       │   repositories   │
├──────────────┤       ├──────────────────┤
│ id (PK)      │──┐    │ id (PK)          │
│ email (UQ)   │  │    │ user_id (FK)     │──┐
│ password_hash│  ├───►│ name             │  │
│ github_token │  │    │ owner            │  │
│ gitlab_token │  │    │ provider         │  │
│ gitlab_url   │  │    │ url              │  │
│ jira_email   │  │    │ is_valid         │  │
│ jira_token   │  │    │ last_synced_at   │  │
│ jira_workspace│ │    │ created_at       │  │
│ jira_username│  │    └──────────────────┘  │
│ created_at   │  │                          │
│ updated_at   │  │    ┌──────────────────┐  │
└──────────────┘  │    │    commits       │  │
                  │    ├──────────────────┤  │
                  │    │ id (PK)          │  │
                  │    │ repo_id (FK)     │◄─┘
                  │    │ sha (UQ)         │──┐
                  │    │ message          │  │
                  │    │ author           │  │
                  │    │ author_email     │  │
                  │    │ branch           │  │
                  │    │ date             │  │
                  │    │ jira_card_key    │  │
                  │    │ has_link         │  │
                  │    │ created_at       │  │
                  │    └──────────────────┘  │
                  │                          │
                  │    ┌──────────────────┐  │
                  │    │commit_card_links │  │
                  │    ├──────────────────┤  │
                  │    │ id (PK)          │  │
                  │    │ commit_id (FK)   │◄─┘
                  │    │ jira_card_key    │
                  │    │ linked_at        │
                  │    └──────────────────┘
                  │
                  │    ┌──────────────────┐
                  │    │    sprints       │
                  │    ├──────────────────┤
                  ├───►│ id (PK)          │──┐
                  │    │ user_id (FK)     │  │
                  │    │ jira_sprint_id   │  │
                  │    │ name             │  │
                  │    │ state            │  │
                  │    │ start_date       │  │
                  │    │ end_date         │  │
                  │    │ created_at       │  │
                  │    └──────────────────┘  │
                  │                          │
                  │    ┌──────────────────┐  │
                  │    │   jira_cards     │  │
                  │    ├──────────────────┤  │
                  ├───►│ id (PK)          │  │
                  │    │ user_id (FK)     │  │
                  │    │ card_key         │  │
                  │    │ summary          │  │
                  │    │ status           │  │
                  │    │ assignee         │  │
                  │    │ sprint_id (FK)   │◄─┘
                  │    │ details_json     │
                  │    │ created_at       │
                  │    └──────────────────┘
                  │
                  │    ┌──────────────────┐
                  │    │report_templates  │
                  │    ├──────────────────┤
                  ├───►│ id (PK)          │──┐
                  │    │ user_id (FK)     │  │
                  │    │ name             │  │
                  │    │ content          │  │
                  │    │ is_default       │  │
                  │    │ created_at       │  │
                  │    │ updated_at       │  │
                  │    └──────────────────┘  │
                  │                          │
                  │    ┌──────────────────┐  │
                  │    │    reports       │  │
                  │    ├──────────────────┤  │
                  └───►│ id (PK)          │  │
                       │ user_id (FK)     │  │
                       │ template_id (FK) │◄─┘
                       │ date             │
                       │ title            │
                       │ content          │
                       │ file_url         │
                       │ created_at       │
                       └──────────────────┘
```

## Integration Points

| Service | Protocol | Auth Method | Purpose |
|---------|----------|-------------|---------|
| GitHub | REST API v3 | Personal Access Token (Bearer) | Fetch commits and branches |
| GitLab | REST API v4 | Personal Access Token (PRIVATE-TOKEN header) | Fetch commits and branches |
| Jira Cloud | REST API v3 | Basic Auth (email:token base64) | Fetch boards, sprints, cards |
| Cloudflare R2 | AWS S3 SDK | Access Key + Secret | Upload report files |

## Project Structure

```
backend/
├── cmd/server/main.go              Entry point, router setup, graceful shutdown
├── internal/
│   ├── config/config.go            Environment variable loading
│   ├── crypto/aes.go               AES-256-GCM encryption/decryption
│   ├── database/database.go        MySQL connection and GORM auto-migration
│   ├── handlers/                   HTTP request handlers
│   │   ├── auth.go                 Register, Login
│   │   ├── user.go                 Profile CRUD, validate connections
│   │   ├── repository.go           Repository CRUD
│   │   ├── sync.go                 Manual sync trigger, sync status
│   │   ├── commit.go               List/filter/link commits
│   │   ├── jira.go                 Sprints, cards, active sprint
│   │   └── report.go               Reports CRUD, templates CRUD, preview
│   ├── middleware/auth.go          JWT authentication middleware
│   ├── models/                     GORM database models
│   │   ├── user.go
│   │   ├── repository.go
│   │   ├── commit.go
│   │   ├── jira.go
│   │   └── report.go
│   ├── services/                   External API clients
│   │   ├── provider.go             CommitProvider interface
│   │   ├── github/github.go        GitHub REST API client
│   │   ├── gitlab/gitlab.go        GitLab REST API client
│   │   ├── jira/jira.go            Jira Cloud REST API client
│   │   ├── report/report.go        Report data builder and renderer
│   │   └── storage/r2.go           Cloudflare R2 upload client
│   └── worker/                     Background job scheduler
│       ├── scheduler.go            Main scheduler with sync loops
│       ├── commits.go              Commit sync logic
│       ├── jira.go                 Jira sync logic
│       ├── reports.go              Auto-report generation
│       └── status.go               Per-user sync status tracking
└── tests/sit/                      System integration tests
```

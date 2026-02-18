# PDT — Personal Daily Tracker

A Go backend that tracks your daily developer work across GitHub, GitLab, and Jira, then generates automated daily reports.

## Tech Stack

| Component | Technology |
|-----------|-----------|
| Language | Go 1.24 |
| Web Framework | Gin |
| ORM | GORM |
| Database | MySQL 8.0+ |
| Authentication | JWT (HS256) |
| Encryption | AES-256-GCM |
| External APIs | GitHub REST, GitLab REST, Jira Cloud REST |
| File Storage | Cloudflare R2 (optional) |
| Background Jobs | Go goroutines + time.Ticker |

## Quick Start

```bash
# 1. Clone
git clone git@github.com:cds-id/pdt.git
cd pdt

# 2. Configure
cp .env.example .env
# Edit .env with your database credentials, JWT_SECRET, and ENCRYPTION_KEY

# 3. Run
cd backend
go build -o bin/server ./cmd/server
./bin/server
```

See the [Setup Guide](../docs/setup.md) for detailed instructions including database setup and integration token configuration.

## API Endpoints

| Method | Endpoint | Description | Auth |
|--------|----------|-------------|------|
| `POST` | `/api/auth/register` | Register new user | No |
| `POST` | `/api/auth/login` | Login | No |
| `GET` | `/api/user/profile` | Get profile with integration status | Yes |
| `PUT` | `/api/user/profile` | Update tokens and settings | Yes |
| `POST` | `/api/user/profile/validate` | Check configured integrations | Yes |
| `GET` | `/api/repos` | List tracked repositories | Yes |
| `POST` | `/api/repos` | Add repository by URL | Yes |
| `DELETE` | `/api/repos/:id` | Delete repository and commits | Yes |
| `POST` | `/api/repos/:id/validate` | Check repository status | Yes |
| `POST` | `/api/sync/commits` | Trigger manual commit sync | Yes |
| `GET` | `/api/sync/status` | Get background sync status | Yes |
| `GET` | `/api/commits` | List commits (filterable) | Yes |
| `GET` | `/api/commits/missing` | Commits not linked to Jira | Yes |
| `POST` | `/api/commits/:sha/link` | Link commit to Jira card | Yes |
| `GET` | `/api/jira/sprints` | List all sprints | Yes |
| `GET` | `/api/jira/sprints/:id` | Get sprint with cards | Yes |
| `GET` | `/api/jira/active-sprint` | Get current active sprint | Yes |
| `GET` | `/api/jira/cards` | List cards (by sprint) | Yes |
| `GET` | `/api/jira/cards/:key` | Card details with commits | Yes |
| `POST` | `/api/reports/generate` | Generate report for date | Yes |
| `GET` | `/api/reports` | List reports (date range) | Yes |
| `GET` | `/api/reports/:id` | Get single report | Yes |
| `DELETE` | `/api/reports/:id` | Delete report | Yes |
| `POST` | `/api/reports/templates` | Create report template | Yes |
| `GET` | `/api/reports/templates` | List templates | Yes |
| `PUT` | `/api/reports/templates/:id` | Update template | Yes |
| `DELETE` | `/api/reports/templates/:id` | Delete template | Yes |
| `POST` | `/api/reports/templates/preview` | Preview template rendering | Yes |

## Documentation

| Document | Description |
|----------|-------------|
| [Setup Guide](../docs/setup.md) | Prerequisites, database setup, environment variables, token configuration |
| [Architecture](../docs/architecture.md) | System overview, data flow, background workers, security model, database ERD |
| **API Reference** | |
| [Authentication](../docs/api/auth.md) | Register and login endpoints |
| [User Profile](../docs/api/user.md) | Profile management, integration settings |
| [Repositories](../docs/api/repositories.md) | Repository tracking (GitHub/GitLab) |
| [Sync](../docs/api/sync.md) | Manual sync trigger, background sync status |
| [Commits](../docs/api/commits.md) | Commit queries, Jira linking |
| [Jira](../docs/api/jira.md) | Sprints, cards, active sprint |
| [Reports](../docs/api/reports.md) | Report generation, templates, preview |

## Project Structure

```
backend/
├── cmd/server/main.go              Entry point, router, graceful shutdown
├── internal/
│   ├── config/                     Environment variable loading
│   ├── crypto/                     AES-256-GCM encryption
│   ├── database/                   MySQL connection, GORM migration
│   ├── handlers/                   HTTP request handlers
│   ├── middleware/                  JWT authentication
│   ├── models/                     Database models
│   ├── services/                   External API clients
│   │   ├── github/                 GitHub REST API
│   │   ├── gitlab/                 GitLab REST API
│   │   ├── jira/                   Jira Cloud REST API
│   │   ├── report/                 Report data builder + renderer
│   │   └── storage/                Cloudflare R2 upload
│   └── worker/                     Background sync scheduler
└── tests/sit/                      System integration tests
```

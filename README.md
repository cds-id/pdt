# PDT - Personal Development Tracker

Track your development activity across GitHub, GitLab, and Jira in one dashboard.

## Features

- **Repository tracking** — Monitor commits across GitHub and GitLab repositories
- **Jira integration** — View sprints, cards, and link commits to Jira issues
- **Project key scoping** — Filter Jira cards by project key prefixes (e.g., PDT, CORE)
- **Daily reports** — Auto-generate daily development reports with customizable templates
- **R2 storage** — Upload reports to Cloudflare R2 (optional)

## Tech Stack

- **Backend:** Go 1.24, Gin, GORM, MySQL
- **Frontend:** React 18, TypeScript, Vite, Tailwind CSS, shadcn/ui, RTK Query

## Quick Start

### Prerequisites

- Go 1.24+
- Node.js 18+
- MySQL 8+

### Setup

```bash
# Install dependencies
make install

# Configure environment
cp backend/.env.example backend/.env
# Edit backend/.env with your database and API credentials

# Run development servers (in separate terminals)
make backend-run
make frontend
```

The backend runs on `http://localhost:8080` and frontend on `http://localhost:5173`.

### Docker

```bash
# Build and run with Docker Compose
docker compose up --build
```

## Environment Variables

| Variable | Default | Description |
|----------|---------|-------------|
| `SERVER_PORT` | `8080` | Backend server port |
| `DB_HOST` | `localhost` | MySQL host |
| `DB_PORT` | `3306` | MySQL port |
| `DB_USER` | `pdt` | MySQL user |
| `DB_PASSWORD` | — | MySQL password |
| `DB_NAME` | `pdt` | MySQL database name |
| `JWT_SECRET` | — | **Required.** Secret for JWT tokens |
| `ENCRYPTION_KEY` | — | **Required.** Key for encrypting API tokens |
| `SYNC_ENABLED` | `true` | Enable background sync |
| `SYNC_INTERVAL_COMMITS` | `15m` | Commit sync interval |
| `SYNC_INTERVAL_JIRA` | `30m` | Jira sync interval |
| `REPORT_AUTO_GENERATE` | `true` | Auto-generate daily reports |
| `REPORT_AUTO_TIME` | `23:00` | Time for auto-report generation |
| `R2_ACCOUNT_ID` | — | Cloudflare R2 account ID (optional) |
| `R2_ACCESS_KEY_ID` | — | Cloudflare R2 access key (optional) |
| `R2_SECRET_ACCESS_KEY` | — | Cloudflare R2 secret key (optional) |
| `R2_BUCKET_NAME` | — | Cloudflare R2 bucket (optional) |
| `R2_PUBLIC_DOMAIN` | — | Cloudflare R2 public domain (optional) |

## Project Structure

```
pdt/
├── backend/
│   ├── cmd/server/         # Entry point
│   ├── internal/
│   │   ├── config/         # Environment config
│   │   ├── crypto/         # Encryption utilities
│   │   ├── database/       # DB connection
│   │   ├── handlers/       # HTTP handlers
│   │   ├── helpers/        # Shared utilities
│   │   ├── middleware/      # Auth middleware
│   │   ├── models/         # GORM models
│   │   ├── services/       # External service clients
│   │   └── worker/         # Background sync jobs
│   └── go.mod
├── frontend/
│   ├── src/
│   │   ├── application/    # Redux store
│   │   ├── components/ui/  # shadcn/ui components
│   │   ├── config/         # Navigation config
│   │   ├── domain/         # Interfaces
│   │   ├── infrastructure/ # API services (RTK Query)
│   │   └── presentation/   # Pages, layouts, components
│   └── package.json
├── Makefile
├── Dockerfile
├── docker-compose.yml
└── README.md
```

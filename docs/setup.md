# Setup Guide

## Prerequisites

- **Go** 1.24 or later
- **MySQL** 8.0 or later (or MariaDB 10.5+)
- **Git** for version control

## Database Setup

```sql
CREATE DATABASE pdt CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci;
CREATE USER 'pdt_user'@'localhost' IDENTIFIED BY 'your_secure_password';
GRANT ALL PRIVILEGES ON pdt.* TO 'pdt_user'@'localhost';
FLUSH PRIVILEGES;
```

Tables are auto-migrated by GORM on server startup.

## Environment Variables

Create a `.env` file in the project root (already in `.gitignore`):

```bash
cp .env.example .env
```

| Variable | Description | Required | Default |
|----------|-------------|----------|---------|
| `SERVER_PORT` | HTTP server port | No | `8080` |
| `DB_HOST` | MySQL host | No | `localhost` |
| `DB_PORT` | MySQL port | No | `3306` |
| `DB_USER` | MySQL username | No | `pdt` |
| `DB_PASSWORD` | MySQL password | Yes | — |
| `DB_NAME` | MySQL database name | No | `pdt` |
| `JWT_SECRET` | Secret key for JWT signing (any random string) | **Yes** | — |
| `JWT_EXPIRY_HOURS` | Token expiry time in hours | No | `72` |
| `ENCRYPTION_KEY` | 64 hex character key for AES-256-GCM encryption | **Yes** | — |
| `SYNC_ENABLED` | Enable background sync workers | No | `true` |
| `SYNC_INTERVAL_COMMITS` | Commit sync frequency (Go duration) | No | `15m` |
| `SYNC_INTERVAL_JIRA` | Jira sync frequency (Go duration) | No | `30m` |
| `REPORT_AUTO_GENERATE` | Enable automatic daily report generation | No | `true` |
| `REPORT_AUTO_TIME` | Time to auto-generate reports (24h format) | No | `23:00` |
| `R2_ACCOUNT_ID` | Cloudflare R2 account ID | No | — |
| `R2_ACCESS_KEY_ID` | R2 access key | No | — |
| `R2_SECRET_ACCESS_KEY` | R2 secret key | No | — |
| `R2_BUCKET_NAME` | R2 bucket name | No | — |
| `R2_PUBLIC_DOMAIN` | R2 public domain for file URLs | No | — |

### Generating Required Keys

**JWT Secret** — any random string:

```bash
openssl rand -hex 32
```

**Encryption Key** — must be exactly 64 hex characters (32 bytes):

```bash
openssl rand -hex 32
```

### Example .env

```bash
SERVER_PORT=8080
DB_HOST=localhost
DB_PORT=3306
DB_USER=pdt_user
DB_PASSWORD=your_secure_password
DB_NAME=pdt
JWT_SECRET=your_jwt_secret_here
JWT_EXPIRY_HOURS=72
ENCRYPTION_KEY=your_64_hex_char_encryption_key_here

# Background sync
SYNC_ENABLED=true
SYNC_INTERVAL_COMMITS=15m
SYNC_INTERVAL_JIRA=30m

# Auto reports
REPORT_AUTO_GENERATE=true
REPORT_AUTO_TIME=23:00

# Cloudflare R2 (optional)
# R2_ACCOUNT_ID=
# R2_ACCESS_KEY_ID=
# R2_SECRET_ACCESS_KEY=
# R2_BUCKET_NAME=
# R2_PUBLIC_DOMAIN=
```

## Integration Token Setup

### GitHub Personal Access Token

1. Go to [GitHub Settings > Developer settings > Personal access tokens > Fine-grained tokens](https://github.com/settings/tokens?type=beta)
2. Click **Generate new token**
3. Set a name and expiration
4. Under **Repository access**, select the repositories you want to track
5. Under **Permissions > Repository permissions**, grant **Contents: Read-only**
6. Click **Generate token** and copy it
7. Save the token via the PDT API: `PUT /api/user/profile` with `{"github_token": "your_token"}`

### GitLab Personal Access Token

1. Go to [GitLab > Preferences > Access Tokens](https://gitlab.com/-/user_settings/personal_access_tokens) (or your self-hosted GitLab instance)
2. Click **Add new token**
3. Set a name and expiration
4. Select scope: **read_api**
5. Click **Create personal access token** and copy it
6. Save via the API: `PUT /api/user/profile` with `{"gitlab_token": "your_token", "gitlab_url": "https://gitlab.com"}`

> For self-hosted GitLab, set `gitlab_url` to your instance URL (e.g., `https://gitlab.mycompany.com`).

### Jira API Token

1. Go to [Atlassian API Tokens](https://id.atlassian.com/manage-profile/security/api-tokens)
2. Click **Create API token**
3. Set a label and click **Create**
4. Copy the token
5. Save via the API:
   ```json
   PUT /api/user/profile
   {
     "jira_token": "your_token",
     "jira_email": "your_email@example.com",
     "jira_workspace": "yourworkspace.atlassian.net",
     "jira_username": "your_username"
   }
   ```

## Cloudflare R2 Setup (Optional)

R2 is used to store generated report files. If not configured, reports are still stored in the database but without a downloadable file URL.

1. Log in to [Cloudflare Dashboard](https://dash.cloudflare.com)
2. Go to **R2 Object Storage**
3. Create a bucket (e.g., `pdt-reports`)
4. Create an API token with **Object Read & Write** permissions
5. Set the R2 environment variables in `.env`
6. Optionally connect a custom domain for public URLs

## Running the Server

```bash
# Build
cd backend
go build -o bin/server ./cmd/server

# Run
./bin/server
```

The server will:

1. Load configuration from `.env`
2. Connect to MySQL and auto-migrate tables
3. Start background sync workers (if enabled)
4. Listen on `SERVER_PORT` (default: 8080)

### Verify Installation

```bash
# Health check — register a test user
curl -X POST http://localhost:8080/api/auth/register \
  -H "Content-Type: application/json" \
  -d '{"email": "test@example.com", "password": "testpass123"}'
```

A successful response returns a JWT token and user object.

## Graceful Shutdown

The server handles `SIGINT` and `SIGTERM` for graceful shutdown:

- Stops accepting new requests
- Waits up to 5 seconds for in-flight requests
- Stops background workers
- Closes database connections

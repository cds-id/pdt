# Design: Personal Jira-GitHub/GitLab Commit Tracker

## Overview

A personal dashboard API that connects your GitHub/GitLab commits to Jira cards, showing which commits reference Jira cards and flagging those that don't.

## Scope

- **Sync Window:** Last 30 days of commits
- **Users:** Single user (personal use)
- **Auth:** JWT-based authentication

## Architecture

```
┌─────────────┐     ┌─────────────┐     ┌─────────────┐
│   Web UI    │────▶│  Backend    │────▶│   MySQL     │
│ (Future)    │     │  (Gin)      │     │   (Gorm)    │
└─────────────┘     └──────┬──────┘     └─────────────┘
                           │
         ┌─────────────────┼─────────────────┐
         ▼                 ▼                 ▼
   ┌───────────┐    ┌───────────┐    ┌───────────┐
   │  GitHub   │    │  GitLab   │    │   Jira    │
   │   API     │    │   API     │    │   API     │
   └───────────┘    └───────────┘    └───────────┘
```

## Data Model

### User
| Field | Type | Description |
|-------|------|-------------|
| id | uint | Primary key |
| email | string | User email |
| password_hash | string | Hashed password |
| github_token | string | GitHub PAT (encrypted) |
| gitlab_token | string | GitLab PAT (encrypted) |
| gitlab_url | string | GitLab base URL (for self-hosted) |
| jira_email | string | Jira email |
| jira_token | string | Jira API token (encrypted) |
| jira_workspace | string | Jira workspace (e.g., company.atlassian.net) |
| jira_username | string | Jira username for mapping |
| created_at | timestamp | Created timestamp |
| updated_at | timestamp | Updated timestamp |

### Repository
| Field | Type | Description |
|-------|------|-------------|
| id | uint | Primary key |
| user_id | uint | FK to User |
| name | string | Repository name |
| owner | string | Repo owner/org |
| provider | enum | github / gitlab |
| url | string | Full repo URL |
| is_valid | bool | Whether user has access |
| last_synced_at | timestamp | Last sync time |
| created_at | timestamp | Created timestamp |

### Commit
| Field | Type | Description |
|-------|------|-------------|
| id | uint | Primary key |
| repo_id | uint | FK to Repository |
| sha | string | Commit SHA |
| message | string | Commit message |
| author | string | Commit author |
| author_email | string | Author email |
| date | timestamp | Commit date |
| jira_card_key | string | Detected Jira card (e.g., PROJ-123) |
| has_link | bool | Has manual or auto link |
| created_at | timestamp | Created timestamp |

### CommitCardLink (Manual Links)
| Field | Type | Description |
|-------|------|-------------|
| id | uint | Primary key |
| commit_id | uint | FK to Commit |
| jira_card_key | string | Jira card key |
| linked_at | timestamp | When linked |

### JiraCard
| Field | Type | Description |
|-------|------|-------------|
| id | uint | Primary key |
| user_id | uint | FK to User |
| key | string | Card key (PROJ-123) |
| summary | string | Card title |
| status | string | Card status |
| assignee | string | Assignee |
| sprint_id | uint | FK to Sprint |
| created_at | timestamp | Created timestamp |

### Sprint
| Field | Type | Description |
|-------|------|-------------|
| id | uint | Primary key |
| user_id | uint | FK to User |
| jira_sprint_id | string | Jira sprint ID |
| name | string | Sprint name |
| state | enum | active / closed / future |
| start_date | timestamp | Sprint start |
| end_date | timestamp | Sprint end |
| created_at | timestamp | Created timestamp |

## API Endpoints

### Authentication
| Method | Endpoint | Description |
|--------|----------|-------------|
| POST | /api/auth/register | Register new user |
| POST | /api/auth/login | Login, get JWT |

### User Profile
| Method | Endpoint | Description |
|--------|----------|-------------|
| GET | /api/user/profile | Get user config |
| PUT | /api/user/profile | Update credentials |
| POST | /api/user/profile/validate | Validate all connections |

### Repositories
| Method | Endpoint | Description |
|--------|----------|-------------|
| GET | /api/repos | List tracked repos |
| POST | /api/repos | Add repo |
| DELETE | /api/repos/:id | Remove repo |
| POST | /api/repos/:id/validate | Validate access to repo |

### Sync
| Method | Endpoint | Description |
|--------|----------|-------------|
| POST | /api/sync/commits | Trigger manual commit sync |

### Commits
| Method | Endpoint | Description |
|--------|----------|-------------|
| GET | /api/commits | List commits (filters: repo_id, jira_card_key, has_link) |
| GET | /api/commits/missing | List commits missing Jira refs |
| POST | /api/commits/:sha/link | Manually link commit to card |

### Jira
| Method | Endpoint | Description |
|--------|----------|-------------|
| GET | /api/jira/sprints | List sprints |
| GET | /api/jira/sprints/:id | Get sprint details |
| GET | /api/jira/active-sprint | Get active sprint |
| GET | /api/jira/cards | List cards in sprint |
| GET | /api/jira/cards/:key | Get card with commits |

## Sync Logic

1. For each tracked repository:
   - Fetch commits from last 30 days
   - Extract Jira card references using regex: `([A-Z]+-\d+)`
   - Store commits with detected card keys
   - Mark commits with detected or manual links as `has_link = true`

2. For Jira:
   - Fetch sprints from configured Jira project
   - Cache sprint and card data
   - Identify active sprint

## Security

- All tokens encrypted at rest (AES-256)
- JWT with configurable expiration
- Single-user mode (no multi-tenancy)

## Assumptions

- Jira project key will be detected from card references
- User will provide necessary API tokens with appropriate scopes
- Scheduled sync will be configurable via environment variable

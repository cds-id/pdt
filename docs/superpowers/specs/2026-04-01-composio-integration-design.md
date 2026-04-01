# Composio.dev Integration for PDT Agents

**Date**: 2026-04-01
**Status**: Approved

## Overview

Augment all existing PDT agents with Composio.dev tools (Gmail, Notion, Google Calendar, LinkedIn). Each user manages their own Composio API key and service connections from the PDT dashboard. Agents transparently gain access to connected external services alongside their native tools.

## Architecture

### Composio REST Client

New package: `backend/internal/ai/composio/`

**Files:**
- `client.go` — HTTP client for Composio API v3 (`https://backend.composio.dev/api/v3`)
  - `GetTools(apiKey string, toolkits []string) ([]minimax.Tool, error)` — fetch tool definitions, convert to minimax.Tool format
  - `ExecuteTool(apiKey, toolSlug, connectedAccountID string, args json.RawMessage) (json.RawMessage, error)` — execute a tool via `POST /tools/execute/{slug}`
  - `GetConnectedAccounts(apiKey string) ([]ConnectedAccount, error)` — list user's connected services
  - `InitiateConnection(apiKey, toolkit, redirectURL string) (string, error)` — generate OAuth redirect URL
- `types.go` — Composio API request/response structs
- `converter.go` — Maps Composio `input_parameters` to `minimax.Tool.InputSchema` (mostly 1:1)

**Authentication:** `x-api-key` header with the user's Composio API key on all requests.

### Agent Integration — Decorator Pattern

New struct `ComposioEnhancedAgent` wraps any existing `Agent`:

```go
type ComposioEnhancedAgent struct {
    Inner          agent.Agent
    ComposioClient *composio.Client
    APIKey         string            // user's decrypted Composio API key
    ComposioTools  []minimax.Tool    // fetched once at session start
}
```

- `Name()` / `SystemPrompt()` — delegates to `Inner`
- `Tools()` — returns `Inner.Tools()` + `ComposioTools`
- `ExecuteTool(name, args)` — if name matches a Composio tool slug, route to `ComposioClient.ExecuteTool()`; otherwise delegate to `Inner.ExecuteTool()`

**Wrapping location:** `AgentBuilder` function in `main.go`. If user has a `ComposioConfig` with an API key, wrap each agent. If not, agents are returned unwrapped — zero impact.

**Tool fetching:** Done once when building agents for a session. Fetches only toolkits the user has active connections for.

### No changes to:
- `RunLoop` (loop.go)
- `Orchestrator` (orchestrator.go)
- Any existing agent implementation
- `minimax.Client`

### Database Models

**ComposioConfig:**

| Field     | Type      | Notes                              |
|-----------|-----------|------------------------------------|
| ID        | uint      | PK                                 |
| UserID    | uint      | FK to users, unique                |
| APIKey    | string    | Encrypted (AES-256-GCM)           |
| CreatedAt | time.Time |                                    |
| UpdatedAt | time.Time |                                    |

**ComposioConnection:**

| Field       | Type      | Notes                              |
|-------------|-----------|------------------------------------|
| ID          | uint      | PK                                 |
| UserID      | uint      | FK to users                        |
| Toolkit     | string    | e.g. "gmail", "notion"             |
| AccountID   | string    | Composio's connected_account_id    |
| Status      | string    | "active" or "inactive"             |
| CreatedAt   | time.Time |                                    |
| UpdatedAt   | time.Time |                                    |

Unique constraint on `(UserID, Toolkit)`.

Auto-migrated on startup alongside existing models.

### API Endpoints

New handler: `backend/internal/handlers/composio.go`

All endpoints JWT-protected, scoped to authenticated user.

| Method   | Endpoint                                      | Description                          |
|----------|-----------------------------------------------|--------------------------------------|
| `PUT`    | `/api/composio/config`                        | Save/update Composio API key         |
| `GET`    | `/api/composio/config`                        | Get config status (has key, not key) |
| `DELETE` | `/api/composio/config`                        | Remove API key + all connections     |
| `GET`    | `/api/composio/connections`                   | List connected services + status     |
| `POST`   | `/api/composio/connections/:toolkit/initiate` | Get OAuth redirect URL               |
| `GET`    | `/api/composio/connections/callback`          | OAuth callback, saves connection     |
| `DELETE` | `/api/composio/connections/:toolkit`          | Disconnect a service                 |

### Frontend — Settings Page

New tab "Composio" in the existing SettingsPage.

**API Key Section:**
- Masked input field for Composio API key
- Save / Remove buttons
- Status indicator (configured / not configured)

**Connected Services Section** (visible when API key is saved):
- Grid of 4 cards: Gmail, Notion, Google Calendar, LinkedIn
- Each card: service icon, name, status badge (Connected/Not Connected), Connect/Disconnect button

**OAuth Flow:**
1. User clicks "Connect Gmail"
2. Frontend calls `POST /api/composio/connections/gmail/initiate`
3. Backend calls Composio API, returns OAuth redirect URL
4. Frontend opens URL in popup/new tab
5. User completes OAuth on the service provider
6. Composio redirects to PDT callback endpoint (`GET /api/composio/connections/callback`)
7. Callback saves `ComposioConnection` record, redirects/closes popup
8. Frontend refreshes connection list to show updated status

Uses existing Shadcn UI components: Card, Button, Badge, Input.

## Initial Toolkits

| Toolkit         | Composio Slug      | Example Tools                          |
|-----------------|--------------------|----------------------------------------|
| Gmail           | `gmail`            | Send email, search inbox, read threads |
| Notion          | `notion`           | Create/query pages, update databases   |
| Google Calendar | `googlecalendar`   | Create/list events, check availability |
| LinkedIn        | `linkedin`         | Create posts, get profile info         |

Extensible — adding a new toolkit requires no code changes, just connecting it in the dashboard.

## Key Properties

- **Zero impact** when user has no Composio key
- **Encrypted storage** for API keys (same AES-256-GCM as GitHub/Jira tokens)
- **Per-user multi-tenancy** — each user's Composio account is independent
- **Session-scoped tool fetching** — tool definitions fetched once, not per request
- **Follows existing patterns** — handler structure, encryption, JWT auth, Shadcn UI

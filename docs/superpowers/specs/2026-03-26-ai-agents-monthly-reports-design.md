# AI Agents, Monthly Reports & Jira Bug Fix — Design Spec

**Date:** 2026-03-26
**Status:** Approved

---

## Overview

Add a conversational AI assistant to PDT powered by MiniMax 2.7, using a multi-agent architecture in Go. Users interact via a dedicated chat page over WebSocket, where specialized agents can query and act on their data (commits, Jira cards, reports). Additionally, add monthly report generation (aggregated stats + AI narrative) and fix the Jira card visibility bug.

## Scope

This spec covers 4 workstreams:

1. **AI Agent System** — Orchestrator + 3 specialist agents (Git, Jira, Report)
2. **WebSocket Chat Interface** — Real-time streaming chat page
3. **Monthly Reports** — Aggregation + AI-generated narrative
4. **Jira Bug Fix** — Investigate and fix cards not showing for specific sprints/projects

---

## 1. Agent Architecture

### Orchestrator

The orchestrator is the entry point for all user messages. It:

1. Receives user message via WebSocket
2. Sends message to MiniMax with a router prompt and a `route_to_agent(agent_name, reason)` tool
3. MiniMax picks the best agent (or handles simple greetings directly)
4. Orchestrator delegates to the specialist agent
5. Specialist agent runs its own tool-calling loop
6. Responses streamed back to user via WebSocket

### Agent Interface

```go
type Agent interface {
    Name() string
    SystemPrompt() string
    Tools() []ToolDefinition
    ExecuteTool(ctx context.Context, name string, args json.RawMessage) (any, error)
}
```

### Specialist Agents

| Agent | Purpose | Tools |
|-------|---------|-------|
| **Git Agent** | Commit analysis, repo insights | `search_commits`, `get_commit_detail`, `list_repos`, `get_repo_stats` |
| **Jira Agent** | Sprint/card queries, linking | `get_sprints`, `get_cards`, `get_card_detail`, `search_cards`, `link_commit_to_card` |
| **Report Agent** | Report generation, summaries | `generate_daily_report`, `generate_monthly_report`, `list_reports`, `get_report`, `preview_template` |

### Agent Loop

Shared logic used by all agents:

```
loop:
  1. Send conversation + system prompt + tools to MiniMax
  2. Stream response tokens to WebSocket
  3. If response contains tool_calls -> execute each tool -> append results -> goto 1
  4. If response is final text -> done
```

### Tool Definition Format

```go
type ToolDefinition struct {
    Type     string       `json:"type"`     // "function"
    Function FunctionDef  `json:"function"`
}

type FunctionDef struct {
    Name        string          `json:"name"`
    Description string          `json:"description"`
    Parameters  json.RawMessage `json:"parameters"` // JSON Schema
}
```

---

## 2. WebSocket Layer

### Connection Flow

1. Frontend opens WebSocket at `ws://host/ws/chat?token=<JWT>`
2. Backend authenticates via JWT, upgrades to WebSocket using `gorilla/websocket`
3. Connection scoped to authenticated user — all queries use that user's data
4. Heartbeat ping/pong every 30s

### Message Protocol (JSON)

**Client to Server:**
```json
{
  "type": "message",
  "content": "What did I work on this week?",
  "conversation_id": "uuid"
}
```
- `conversation_id` is optional; omit to start a new conversation.

**Server to Client — Streaming tokens:**
```json
{
  "type": "stream",
  "content": "Based on your commits...",
  "conversation_id": "uuid"
}
```

**Server to Client — Tool execution status:**
```json
{
  "type": "tool_status",
  "tool": "search_commits",
  "status": "executing"
}
```
- `status` values: `executing`, `completed`

**Server to Client — Stream complete:**
```json
{
  "type": "done",
  "conversation_id": "uuid"
}
```

**Server to Client — Error:**
```json
{
  "type": "error",
  "content": "Failed to fetch Jira cards"
}
```

### Conversation Persistence

New database models:

**Conversation:**
- `id` (UUID, primary key)
- `user_id` (FK to users)
- `title` (string, auto-generated from first message)
- `created_at`, `updated_at`

**ChatMessage:**
- `id` (UUID, primary key)
- `conversation_id` (FK to conversations)
- `role` (enum: `user`, `assistant`, `tool`)
- `content` (text)
- `tool_calls` (JSON, nullable — stores tool call data for assistant messages)
- `tool_name` (string, nullable — for tool role messages)
- `created_at`

Context window: send last 20 messages to MiniMax (configurable via `AI_CONTEXT_WINDOW` env var).

---

## 3. MiniMax 2.7 Integration

### API Client

- HTTP client calling MiniMax chat completion API with SSE streaming
- OpenAI-compatible request/response format
- API key in `MINIMAX_API_KEY` env var, group ID in `MINIMAX_GROUP_ID`

### Request Structure

```go
type ChatRequest struct {
    Model       string    `json:"model"`
    Messages    []Message `json:"messages"`
    Tools       []Tool    `json:"tools,omitempty"`
    Stream      bool      `json:"stream"`
    Temperature float64   `json:"temperature"`
}

type Message struct {
    Role       string          `json:"role"`
    Content    string          `json:"content,omitempty"`
    ToolCalls  []ToolCall      `json:"tool_calls,omitempty"`
    ToolCallID string          `json:"tool_call_id,omitempty"`
}
```

### Streaming

MiniMax supports SSE. The client reads chunks, forwards each content delta to WebSocket as `stream` messages. Tool calls are accumulated from the stream, then executed when the stream ends.

### Usage Tracking

New model:

**AIUsage:**
- `id` (primary key)
- `user_id` (FK to users)
- `conversation_id` (FK to conversations)
- `prompt_tokens` (int)
- `completion_tokens` (int)
- `created_at`

Logged per MiniMax API call for monitoring. No hard limits initially.

---

## 4. Monthly Reports

### Generation

New endpoint: `POST /reports/generate-monthly` with `month` and `year` params.
Also accessible via Report Agent in chat.

### Data Aggregation

For a given month, collect:
- All daily reports
- All commits (grouped by week, repo, Jira card)
- All Jira cards worked on (with status transitions)

### Report Structure

**Part 1 — Raw Stats:**
- Total commits, cards completed, cards in progress
- Commits per week (weekly breakdown)
- Repos touched with commit counts
- Top Jira cards by commit activity

**Part 2 — AI Narrative:**
- Aggregated stats fed to MiniMax with a monthly report prompt
- AI generates: executive summary, key accomplishments, trends, areas of focus
- Streamed back if via chat, saved directly if via API

### Model Changes

Add to existing `Report` model:
- `report_type` field: `daily` (default) | `monthly`
- `month` (int, nullable) and `year` (int, nullable) for monthly reports

Monthly report templates stored separately from daily (user can customize both).

### Auto-Generation

Worker generates monthly report on the 1st of each month at `REPORT_MONTHLY_AUTO_TIME` (default `08:00`). Only runs if user had activity that month.

---

## 5. Jira Bug Fix

### Symptoms

Cards not showing for specific sprints/projects.

### Investigation Areas

1. **Project key filtering** — `helpers/jira.go` filters by `JiraProjectKeys`. Cards with unexpected prefixes (sub-tasks, linked issues from other projects) may be silently dropped.

2. **Sprint pagination** — Jira Agile API paginates issues (default 50). If pagination isn't handled, sprints with 50+ cards lose the overflow.

3. **Board selection** — Sync fetches boards then sprints. If a card lives under a different board, it's never fetched.

4. **Sync race condition** — Worker upserts while frontend queries, potentially showing partial data.

### Fix Approach

- Test against the actual Jira instance (credentials in `.env`)
- Verify pagination handling in `services/jira/jira.go`
- Verify project key filter logic in `helpers/jira.go`
- Verify all boards/sprints are traversed
- Add sync logging for traceability

---

## 6. New Database Models Summary

| Model | Fields |
|-------|--------|
| **Conversation** | `id`, `user_id`, `title`, `created_at`, `updated_at` |
| **ChatMessage** | `id`, `conversation_id`, `role`, `content`, `tool_calls`, `tool_name`, `created_at` |
| **AIUsage** | `id`, `user_id`, `conversation_id`, `prompt_tokens`, `completion_tokens`, `created_at` |

Existing model changes:
- **Report**: add `report_type`, `month`, `year` fields

---

## 7. New Environment Variables

| Variable | Default | Purpose |
|----------|---------|---------|
| `MINIMAX_API_KEY` | — | Required for AI features |
| `MINIMAX_GROUP_ID` | — | MiniMax API group ID |
| `AI_CONTEXT_WINDOW` | 20 | Max messages sent to MiniMax per request |
| `REPORT_MONTHLY_AUTO_TIME` | 08:00 | Monthly report auto-generation time |

---

## 8. New API Endpoints

| Method | Path | Purpose |
|--------|------|---------|
| `WS` | `/ws/chat` | WebSocket connection for AI chat |
| `GET` | `/conversations` | List user's conversations |
| `GET` | `/conversations/:id` | Get conversation with messages |
| `DELETE` | `/conversations/:id` | Delete conversation |
| `POST` | `/reports/generate-monthly` | Generate monthly report |

---

## 9. Frontend Changes

### New Page: Assistant (`/assistant`)

- Full-page chat interface
- Left sidebar: conversation history list
- Main area: message thread with streaming text
- Input box at bottom with send button
- Tool execution indicators (e.g., "Searching commits...")
- New conversation button
- Added to sidebar navigation under a new "AI" section

### Reports Page Update

- Add toggle/tab for Daily vs Monthly reports
- Monthly report generation button with month/year picker

---

## 10. File Structure (New)

```
backend/internal/
  ai/
    minimax/
      client.go          # MiniMax API client (streaming SSE)
      types.go           # Request/response types
    agent/
      orchestrator.go    # Routes messages to specialist agents
      loop.go            # Shared agent tool-calling loop
      git.go             # Git Agent
      jira.go            # Jira Agent
      report.go          # Report Agent
      types.go           # Agent interface, ToolDefinition
  handlers/
    chat.go              # WebSocket handler, conversation CRUD
  models/
    conversation.go      # Conversation, ChatMessage models
    ai_usage.go          # AIUsage model

frontend/src/
  presentation/pages/
    AssistantPage.tsx     # Chat page
  presentation/components/
    chat/
      ChatSidebar.tsx     # Conversation list
      ChatMessage.tsx     # Message bubble
      ChatInput.tsx       # Input with send
      ToolStatus.tsx      # Tool execution indicator
  infrastructure/services/
    chat.service.ts       # WebSocket client + conversation API
```

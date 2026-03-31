# Agent Scheduling System Design

## Problem

Agents can only be invoked interactively via chat. There is no way to run agents on a schedule (e.g., morning briefing every weekday at 8am), trigger agents in response to events (e.g., run report after commit sync), or chain agents together. Users have no visibility into automated agent activity.

## Decision

Build a hybrid DB-polled scheduler with a goroutine pool and in-process event bus. The system lives alongside the existing `worker/scheduler.go` (which handles commit/jira sync and auto-reports). Scheduled runs create conversations visible in webchat and send Telegram notifications. A full CRUD frontend page provides schedule management and run history.

## Architecture Overview

```
                    ┌─────────────┐
                    │  Event Bus  │ ← worker emits: commit_synced, jira_synced
                    └──────┬──────┘
                           │ event triggers
    ┌──────────┐    ┌──────▼──────┐    ┌──────────────┐
    │ DB Poll  │───►│   Engine    │───►│ Goroutine    │
    │ (30s)    │    │ (scheduler) │    │ Pool (max 3) │
    └──────────┘    └──────┬──────┘    └──────┬───────┘
                           │                   │
                    ┌──────▼──────┐    ┌──────▼───────┐
                    │  REST API   │    │   Executor   │
                    │  /schedules │    │ (per run)    │
                    └─────────────┘    └──────┬───────┘
                                              │
                              ┌────────────────┼────────────────┐
                              │                │                │
                       ┌──────▼──────┐  ┌──────▼──────┐  ┌─────▼──────┐
                       │ Orchestrator│  │ Conversation │  │  Telegram  │
                       │ / Agent     │  │   (save)     │  │ Notify     │
                       └─────────────┘  └─────────────┘  └────────────┘
```

## Data Model

### agent_schedules

| Column | Type | Description |
|---|---|---|
| id | UUID, PK | |
| user_id | FK → users | Owner |
| name | string | "Morning briefing", "Jira blocker check" |
| agent_name | string | "briefing", "jira", etc. or "" for orchestrator routing |
| prompt | text | Message to send to the agent |
| trigger_type | enum | "cron", "interval", "event" |
| cron_expr | string, nullable | 5-field cron: "0 8 * * 1-5" |
| interval_seconds | int, nullable | e.g., 900 for every 15min |
| event_name | string, nullable | "commit_synced", "jira_synced", etc. |
| chain_config | JSON, nullable | Explicit chain steps |
| enabled | bool, default true | |
| next_run_at | timestamp, nullable | Precomputed next execution time |
| created_at | timestamp | |
| updated_at | timestamp | |

### agent_schedule_runs

| Column | Type | Description |
|---|---|---|
| id | UUID, PK | |
| schedule_id | FK → agent_schedules | |
| user_id | FK → users | |
| conversation_id | FK → conversations, nullable | Linked conversation |
| status | enum | "pending", "running", "completed", "failed" |
| trigger_type | string | "cron", "interval", "event", "manual", "chain" |
| started_at | timestamp | |
| completed_at | timestamp, nullable | |
| result_summary | text, nullable | First 500 chars or first heading of response |
| error | text, nullable | |
| token_usage | JSON, nullable | {prompt_tokens, completion_tokens} |
| created_at | timestamp | |

### agent_schedule_run_steps

| Column | Type | Description |
|---|---|---|
| id | UUID, PK | |
| run_id | FK → agent_schedule_runs | |
| agent_name | string | Which agent executed |
| prompt | text | What was sent |
| response | text | Full response |
| status | enum | "completed", "failed" |
| duration_ms | int | |
| created_at | timestamp | |

## Backend Packages

### `internal/scheduler/engine.go` — Main scheduler loop

- Polls DB every 30 seconds for schedules where `next_run_at <= now AND enabled = true`
- Dispatches to goroutine pool
- After each run, computes and updates `next_run_at`
- On startup, subscribes all enabled event-triggered schedules to the event bus
- Re-subscribes when schedules are created/updated/deleted

### `internal/scheduler/pool.go` — Bounded goroutine pool

- Configurable max concurrency (default 3)
- Jobs queue if pool is full
- Prevents runaway execution

### `internal/scheduler/executor.go` — Single run executor

Execution flow:
1. Create `agent_schedule_runs` entry with status "running"
2. Create new conversation titled "Scheduled: {name} — {date}"
3. Build orchestrator (or target specific agent if `agent_name` set)
4. Run agent with the schedule's prompt
5. Save response to conversation + create run step in `agent_schedule_run_steps`
6. Evaluate explicit chain config — if conditions match, run next agents as additional steps
7. Update run status to "completed" (or "failed") with result summary and token usage
8. Send Telegram notification to user
9. Emit `schedule_completed` event to event bus

### `internal/scheduler/cron.go` — Cron expression handling

- Uses `github.com/robfig/cron/v3` for parsing only (not its scheduler)
- Computes next run time from 5-field cron expression
- Interval schedules compute `next_run_at = now + interval_seconds`

### `internal/scheduler/eventbus/bus.go` — In-process event bus

- Simple pub/sub with Go channels
- `Publish(event string, payload map[string]any)`
- `Subscribe(event string, handler func(payload map[string]any))` returns unsubscribe func
- Engine subscribes event-triggered schedules on startup
- Subscribers filter by `user_id` from payload before enqueuing

### `internal/handlers/schedule.go` — REST API

| Method | Path | Description |
|---|---|---|
| GET | /api/schedules | List user's schedules |
| POST | /api/schedules | Create schedule |
| PUT | /api/schedules/:id | Update schedule |
| DELETE | /api/schedules/:id | Delete schedule |
| POST | /api/schedules/:id/toggle | Enable/disable |
| POST | /api/schedules/:id/run | Trigger manual run |
| GET | /api/schedules/:id/runs | List run history |
| GET | /api/schedules/runs/:runId | Get run detail with steps |

## Event System

### Internal events

| Event | Source | Payload |
|---|---|---|
| commit_synced | worker/commits.go | {user_id, repo_count, new_commits} |
| jira_synced | worker/jira.go | {user_id, workspace, cards_updated} |
| report_generated | Report agent | {user_id, report_id, report_type} |
| schedule_completed | Executor | {user_id, schedule_id, schedule_name, status} |

### Event-triggered flow

1. Worker emits event (e.g., `commit_synced`) on bus after completing sync
2. Engine has pre-registered subscribers for all enabled event-triggered schedules
3. Subscriber filters by `user_id` from payload
4. If match, engine enqueues a run to the pool

### Webhook-ready design

The event bus accepts `Publish()` from any source. Future webhook handler would parse incoming HTTP, map to an event name, and publish. No changes to the scheduler or executor needed.

## Agent Chaining

### Explicit chains (configured in `chain_config`)

```json
[
  {"agent": "jira", "prompt": "List all blockers in current sprint", "condition": "always"},
  {"agent": "whatsapp", "prompt": "Send blocker summary to team lead", "condition": "contains:blocker"}
]
```

After the main agent completes, executor evaluates each chain step:
- `condition` types: `always`, `contains:{keyword}`, `status:completed`, `status:failed`
- Each step creates an `agent_schedule_run_steps` entry
- Chain steps execute sequentially within the same run

### Dynamic chains (agent decides at runtime)

- New tool `trigger_agent` injected by the executor into the agent's tool list during scheduled runs only (not available in interactive chat)
- Tool schema: `{agent: string, prompt: string}`
- Executor intercepts the tool call, runs target agent, returns result as tool output
- Creates an `agent_schedule_run_steps` entry
- Max chain depth: 3 (prevents infinite loops)

## Telegram Notification

When a scheduled run completes, send to user's Telegram chat (looked up from `telegram_whitelist`):

**Success:**
```
📋 Scheduled: Morning Briefing
✅ Completed in 12s

{result_summary — first 500 chars}
```

**Failure:**
```
📋 Scheduled: Morning Briefing
❌ Failed: {error message}
```

Uses the HTML formatter (`formatter.ToTelegramHTML`) for proper formatting.

## Conversation Integration

- Each run creates a new conversation: "Scheduled: {name} — 2026-03-31"
- Full agent response saved as chat messages (user message = prompt, assistant message = response)
- Conversation linked via `conversation_id` on the run record
- Visible in webchat chat history alongside interactive conversations

## Frontend

### New page: `/schedules`

**Schedule list view:**
- Card layout showing all schedules
- Each card: name, agent badge, trigger type badge (cron/interval/event), next run time, enabled toggle, last run status
- Actions: edit, delete, run now
- "Create Schedule" button

**Create/Edit form:**
- Name (text input)
- Agent (dropdown: auto/orchestrator, git, jira, report, proof, briefing, whatsapp)
- Prompt (textarea)
- Trigger type (radio: cron, interval, event)
  - Cron: expression input with presets ("Every weekday 8am", "Every Monday 9am", "Every hour")
  - Interval: number + unit (minutes, hours)
  - Event: dropdown (commit_synced, jira_synced, report_generated, schedule_completed)
- Chain config (optional expandable section): add steps with agent, prompt, condition
- Enabled toggle

**Run history (expandable from card or sub-page):**
- Table: timestamp, trigger type, status badge, duration, summary preview
- Expand row: full response, chain steps with agent outputs, token usage, errors

### Frontend service layer

- `infrastructure/services/schedule-service.ts` — CRUD, toggle, manual run, history
- Follows existing auth/API patterns

## Dependencies

- `github.com/robfig/cron/v3` — cron expression parsing (parser only, not scheduler)

## Files NOT changed

- `worker/scheduler.go` — existing sync loops remain untouched
- Orchestrator routing logic — unchanged
- Existing agent implementations — only addition is the `trigger_agent` tool

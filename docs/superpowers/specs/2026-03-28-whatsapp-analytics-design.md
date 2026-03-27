# WhatsApp Analytics — Design Spec

## Overview

Add WhatsApp analytics to PDT: users pair one or more WhatsApp numbers, register specific groups/contacts as "listeners," and PDT captures incoming messages for AI-powered analysis. An AI agent can search, summarize, and send messages (with user approval). Messages are embedded into Weaviate (using Gemini embeddings) for semantic search. Media files are stored in R2.

## Architecture Decision

**Monolith Extension** — WhatsApp functionality is added directly into the existing PDT backend. whatsmeow connections run as managed goroutines alongside existing sync workers. A new WhatsAppAgent joins the orchestrator. Weaviate client lives in PDT as a service.

Rationale: PDT is a single-user personal tool. Process isolation (sidecar) and microservice splits add operational overhead that isn't justified. The monolith approach reuses existing patterns (services, agents, DB, R2, auth, WebSocket) and allows the WhatsApp agent to cross-reference Jira/Git data natively.

## Data Model

### New MySQL Tables

**wa_numbers** — Paired WhatsApp numbers with whatsmeow session storage.

| Column | Type | Description |
|--------|------|-------------|
| id | uint (PK) | Auto-increment |
| user_id | uint (FK → users) | Owner |
| phone_number | string | Phone number display |
| display_name | string | User-given label (e.g., "Work Phone") |
| device_store | blob | whatsmeow session data, survives restarts |
| status | enum | `pairing`, `connected`, `disconnected` |
| paired_at | timestamp | When pairing completed |
| created_at | timestamp | |
| updated_at | timestamp | |

**wa_listeners** — Whitelisted groups/contacts to capture messages from.

| Column | Type | Description |
|--------|------|-------------|
| id | uint (PK) | Auto-increment |
| wa_number_id | uint (FK → wa_numbers) | Which number this listener belongs to |
| jid | string | WhatsApp JID (group or contact) |
| name | string | User-given label (e.g., "Engineering Chat") |
| type | enum | `group`, `personal` |
| is_active | bool | Whether actively capturing messages |
| created_at | timestamp | |
| updated_at | timestamp | |

**wa_messages** — Captured messages from active listeners.

| Column | Type | Description |
|--------|------|-------------|
| id | uint (PK) | Auto-increment |
| wa_listener_id | uint (FK → wa_listeners) | Source listener |
| message_id | string (unique) | WhatsApp message ID |
| sender_jid | string | Who sent it |
| sender_name | string | Display name of sender |
| content | text | Message text content |
| message_type | enum | `text`, `image`, `document`, `audio`, `video` |
| has_media | bool | Whether message has media attachment |
| timestamp | timestamp | When message was sent in WhatsApp |
| created_at | timestamp | |

**wa_media** — R2 file references for message attachments.

| Column | Type | Description |
|--------|------|-------------|
| id | uint (PK) | Auto-increment |
| wa_message_id | uint (FK → wa_messages) | Parent message |
| file_name | string | Original filename |
| mime_type | string | MIME type |
| file_size | int64 | File size in bytes |
| r2_key | string | R2 object key |
| file_url | string | R2 public/signed URL |
| created_at | timestamp | |

**wa_outbox** — Agent-drafted outgoing messages with approval workflow.

| Column | Type | Description |
|--------|------|-------------|
| id | uint (PK) | Auto-increment |
| wa_number_id | uint (FK → wa_numbers) | Which number to send from |
| target_jid | string | Recipient JID |
| target_name | string | Display name of target |
| content | text | Message text |
| status | enum | `pending`, `approved`, `sent`, `rejected` |
| requested_by | enum | `agent`, `user` |
| context | text | Why agent wants to send (explanation) |
| approved_at | timestamp | |
| sent_at | timestamp | |
| created_at | timestamp | |

### Weaviate Collection

**WaMessageEmbedding** — Vectorized message content for semantic search.

| Property | Type | Description |
|----------|------|-------------|
| message_id | int | Cross-reference to wa_messages.id |
| listener_id | int | For filtering by listener |
| user_id | int | For row-level isolation |
| content | text | Original text (vectorized by Gemini) |
| sender_name | text | Who sent it |
| timestamp | date | When sent |

Vectorizer: `text2vec-palm` module using Gemini embedding API. API key passed via `X-Google-Api-Key` request header.

## Connection Management

### Pairing Flow

1. User clicks "Add Number" in Settings
2. PDT creates `wa_numbers` row with status `pairing`
3. whatsmeow generates QR code
4. QR code streamed to frontend via dedicated WebSocket (`/ws/wa/pair/:number_id`)
5. User scans QR with WhatsApp on phone
6. whatsmeow completes pairing → device_store saved to DB
7. Status updated to `connected`, message listening starts

### Connection Manager

- **Startup**: Load all `wa_numbers` with status=connected → reconnect each via whatsmeow using saved device_store
- **Runtime**: Each number gets its own whatsmeow `Client` instance in a goroutine. Manager holds `map[uint]*Client` keyed by wa_number ID
- **Recovery**: Exponential backoff on disconnect (5s → 10s → 30s → 60s max). After 5 consecutive failures, mark status=disconnected and notify user via WebSocket
- **Shutdown**: Graceful disconnect of all active connections

## Message Flow

### Incoming Messages

1. whatsmeow event handler receives message
2. Check if sender JID exists in `wa_listeners` (active) for this number
3. If not whitelisted → drop
4. If whitelisted → save to `wa_messages`
5. Async (non-blocking):
   - If has_media: download from WhatsApp → upload to R2 → save `wa_media` record
   - Push text content to embedding queue (Go channel)

### Embedding Pipeline

1. Buffered Go channel receives message IDs
2. Embedding worker batches messages (up to 50 or 5s timeout)
3. Upsert to Weaviate with `X-Google-Api-Key` header
4. Weaviate calls Gemini embedding API automatically via `text2vec-palm`
5. On failure: retry up to 3 times, then skip (MySQL is source of truth)
6. Rate limiting: worker respects Gemini API limits, backfill uses slower pacing

### Outgoing Messages

1. AI agent calls `send_message` or `reply_to_message` tool
2. Creates `wa_outbox` row with status `pending` and context explaining why
3. Agent informs user: "I'd like to send this message. Please approve in the Outbox."
4. User reviews in Outbox UI → approve / edit / reject
5. Sender worker (background goroutine, polls every 5s) picks up `approved` messages
6. Sends via whatsmeow → status updated to `sent`

## WhatsApp Agent

New agent (6th) added to the orchestrator. Routes to this agent when user asks about WhatsApp messages, chat summaries, listener activity, or sending messages.

### System Prompt Highlights

- Anti-hallucination: never fabricate message content, only present tool results
- Self-awareness: knows the current user's phone numbers, distinguishes user's own messages from external
- Send discipline: must explain WHY it wants to send (context field), never sends without approval
- Summarization: uses semantic_search to find relevant threads, then builds richer summaries with that context

### Tools (7)

**Read tools (MySQL):**

1. **list_listeners** — List all registered listeners across all numbers. Returns name, type, number, active status, message count.
   - Params: `wa_number_id?` (optional filter)

2. **search_messages** — Keyword search across messages with filters.
   - Params: `query`, `listener_id?`, `sender?`, `start_date?`, `end_date?`, `limit?`

**AI-powered tools (Weaviate / LLM):**

3. **semantic_search** — Vector similarity search via Weaviate. Finds messages by meaning.
   - Params: `query`, `listener_id?`, `start_date?`, `end_date?`, `limit?`

4. **summarize_chat** — Generate AI summary for a listener or across all listeners in a time range. Key topics, action items, decisions.
   - Params: `listener_id?` (null = all), `start_date`, `end_date`

**Write tools (outbox):**

5. **send_message** — Draft an outgoing message for approval.
   - Params: `target_jid`, `content`, `context`

6. **reply_to_message** — Draft a reply to a specific message (quoted reply in WhatsApp).
   - Params: `wa_message_id`, `content`, `context`

**Compound tool:**

7. **full_chat_report** — Single call that runs list_listeners + summarize_chat (all) + semantic_search (recent topics). Returns complete WhatsApp briefing.
   - Params: `start_date`, `end_date`

## API Endpoints

All under `/api/` with JWT auth middleware.

### Numbers

| Method | Path | Description |
|--------|------|-------------|
| GET | `/api/wa/numbers` | List paired numbers |
| POST | `/api/wa/numbers` | Start pairing (returns WS channel for QR) |
| DELETE | `/api/wa/numbers/:id` | Disconnect & remove number |
| PATCH | `/api/wa/numbers/:id` | Update display name |

### Listeners

| Method | Path | Description |
|--------|------|-------------|
| GET | `/api/wa/numbers/:id/listeners` | List listeners for a number |
| POST | `/api/wa/numbers/:id/listeners` | Add listener (JID + name + type) |
| PATCH | `/api/wa/listeners/:id` | Update name, toggle active |
| DELETE | `/api/wa/listeners/:id` | Remove listener |

### Messages

| Method | Path | Description |
|--------|------|-------------|
| GET | `/api/wa/listeners/:id/messages` | Paginated messages for a listener |
| GET | `/api/wa/messages/search` | Search across all listeners |

### Outbox

| Method | Path | Description |
|--------|------|-------------|
| GET | `/api/wa/outbox` | List outbox (filterable by status) |
| PATCH | `/api/wa/outbox/:id` | Approve / reject / edit content |
| DELETE | `/api/wa/outbox/:id` | Delete pending message |

### WebSocket

| Path | Description |
|------|-------------|
| `/ws/wa/pair/:number_id` | QR code stream during pairing |

## Frontend

### Settings Page Extension

- New "WhatsApp Numbers" section in existing Settings page
- Number cards showing phone number, display name, status (connected/disconnected)
- "Add Number" button → opens QR pairing modal
- Per-number "Listeners" button → opens listener management panel
- Listener cards showing name, type, message count, active toggle
- "Add Listener" with JID input or pick from recent chats

### Outbox Page (new)

- Accessible from sidebar navigation
- Pending messages with Send / Edit / Reject actions
- Shows agent's context (reason for sending)
- Sent/rejected history below

### QR Pairing Modal

- Modal overlay when adding a new number
- QR code rendered from WebSocket stream
- Status indicator: waiting → scanning → connected
- Auto-closes on successful pairing

## Weaviate Docker Setup

Added to existing `docker-compose.yml`:

```yaml
services:
  weaviate:
    image: semitechnologies/weaviate:latest
    ports:
      - "8081:8080"
      - "50051:50051"
    environment:
      QUERY_DEFAULTS_LIMIT: 25
      AUTHENTICATION_ANONYMOUS_ACCESS_ENABLED: true
      PERSISTENCE_DATA_PATH: /var/lib/weaviate
      DEFAULT_VECTORIZER_MODULE: text2vec-palm
      ENABLE_MODULES: text2vec-palm
      CLUSTER_HOSTNAME: node1
    volumes:
      - weaviate_data:/var/lib/weaviate
```

No local transformer container — Gemini handles embedding via API. `GEMINI_API_KEY` env var added to PDT config.

## Resilience

- **MySQL is source of truth** — Weaviate is a search index. If Weaviate/Gemini is down, messages still flow into MySQL. Vector search degrades gracefully (agent falls back to keyword search).
- **Async embedding** — message ingestion is never blocked by Weaviate latency. Buffered Go channel decouples the two.
- **Idempotent upserts** — embedding worker uses message_id as deterministic UUID in Weaviate, safe to retry.
- **Weaviate health check** — client pings on startup. If unavailable, logs warning, disables vector features, retries periodically.
- **Connection recovery** — whatsmeow auto-reconnect with exponential backoff. User notified on persistent failure.
- **Backfill** — background job to re-embed all messages from MySQL if Weaviate is rebuilt.

## New Environment Variables

| Variable | Required | Description |
|----------|----------|-------------|
| `GEMINI_API_KEY` | Yes (for vector search) | Google Gemini API key for embeddings |
| `WEAVIATE_URL` | No (default: `http://localhost:8081`) | Weaviate endpoint |

# Memory System & Orchestration Optimization

**Date**: 2026-04-01
**Status**: Approved

## Overview

Add cross-conversation memory to PDT agents using Weaviate, and optimize the orchestrator with memory-aware routing. A background worker processes completed conversations to extract summaries and user facts. Memory is injected into both the router and all agent system prompts, making routing smarter and agents context-aware across conversations.

## Architecture

### Weaviate Collections

**`ConversationSummary`** — one entry per processed conversation:
- `conversation_id` (string) — maps to conversations table
- `user_id` (int) — for filtering
- `content` (text, vectorized) — 1-3 sentence summary of what was discussed and resolved
- `agent_used` (string) — which agent(s) handled the conversation
- `created_at` (date)

Vectorizer: `text2vec-google` (same as existing collections).

**`UserFact`** — persistent extracted facts about the user:
- `user_id` (int) — for filtering
- `content` (text, vectorized) — the fact, e.g. "User prefers responses in Indonesian"
- `category` (string) — "preference", "project", "identity"
- `created_at` (date)
- `updated_at` (date)

Deduplication: before inserting a new fact, semantic search existing UserFact entries for the same user. If distance < 0.15, update the existing fact instead of creating a new one. This keeps the fact list compact (typically < 10 per user).

### Memory Worker

New background worker: `backend/internal/worker/memory.go`

**Schedule**: Configurable interval via `MEMORY_WORKER_INTERVAL` env var (default `5m`).

**Selection criteria** — process conversations that:
- Have at least 2 messages (user + assistant minimum)
- `summarized_at IS NULL` (not yet processed)
- Last message older than 5 minutes (conversation is idle)

**Processing per conversation**:
1. Load all messages from MySQL
2. Call MiniMax LLM (non-streaming `Chat()`, temperature 0.3) with extraction prompt:
   - Input: full conversation messages
   - Output: JSON with `summary` (string) and `facts` (array of `{content, category}`)
3. Store summary in Weaviate `ConversationSummary` collection
4. For each fact: semantic search existing `UserFact` for same user. If similar (distance < 0.15), update. Otherwise create.
5. Set `summarized_at = NOW()` on the conversation row

**Extraction prompt**:
```
Analyze this conversation and extract:
1. A summary (1-3 sentences) of what was discussed and any outcomes.
2. User facts — persistent information about the user (preferences, projects, identity, habits).

Respond in JSON:
{
  "summary": "...",
  "agent_used": "git|jira|report|...",
  "facts": [
    {"content": "User prefers Indonesian language", "category": "preference"},
    {"content": "User works on PDT project", "category": "project"}
  ]
}

Only extract facts that are durable — not transient questions. If no new facts, return empty array.
```

**Config**: Add `MEMORY_WORKER_INTERVAL` to config.go (default `5m`).

### Smart Router with Memory

Modify `orchestrator.go` to inject memory context before the router LLM call.

**Memory fetch at request time** (once per message, before routing):
1. `SearchUserFacts(userID)` — fetch all facts for user (no semantic search needed, just filter by user_id). Typically < 10 entries, ~200 tokens.
2. `SearchConversationSummaries(userID, userMessage, limit=3)` — semantic search summaries using the current user message as query. Returns top 3 most relevant past conversations.

**Inject into router system prompt** (prepended before existing routing instructions):
```
## User Context
Facts:
- User prefers responses in Indonesian
- Main Jira workspace is cds-id.atlassian.net
- Jira username is indra

## Relevant Past Conversations
- 2 days ago: User asked about commit activity on repo PDT for last week, found 23 commits. (agent: git)
- 5 days ago: User set up morning briefing schedule at 8am. (agent: scheduler)
```

**Improved direct response**: Update the router prompt to explicitly allow direct answers when memory + chat history is sufficient:
> "If you can answer the user's question using the conversation history and user context above (e.g., 'what did we discuss yesterday?', 'remind me about X'), respond directly without routing to any agent."

This replaces the current behavior where the router can only handle greetings directly.

### Agent Memory Injection — Decorator Pattern

New decorator: `MemoryEnhancedAgent` (in a `memory` package or `ai/memory/`).

```go
type MemoryEnhancedAgent struct {
    Inner        agent.Agent
    UserFacts    string   // pre-formatted facts block
    Summaries    string   // pre-formatted summaries block
}
```

- `Name()` / `Tools()` / `ExecuteTool()` — delegate to Inner
- `SystemPrompt()` — returns `Inner.SystemPrompt() + "\n\n" + memoryContext`

**Where wrapping happens**: In `AgentBuilder` (main.go) and `HandleWebSocket` (chat.go), same locations as Composio wrapping. Memory context is fetched once per session when agents are built.

**Wrapping order**: Base agent -> Composio wrapper -> Memory wrapper (or reverse, both only affect `SystemPrompt()`/`Tools()`/`ExecuteTool()` independently).

**Shared fetch**: The orchestrator and the agent decorator use the same fetched memory — fetched once, passed to both. No duplicate Weaviate calls.

### Database Changes

Add nullable column to `Conversation` model:

```go
SummarizedAt *time.Time `json:"summarized_at"`
```

GORM AutoMigrate handles the column addition. The memory worker queries:
```sql
WHERE summarized_at IS NULL
  AND updated_at < NOW() - INTERVAL 5 MINUTE
  AND (SELECT COUNT(*) FROM chat_messages WHERE conversation_id = c.id) >= 2
```

No new MySQL tables. Summaries and facts live in Weaviate only. If Weaviate is unavailable, agents work without memory (graceful degradation).

### Weaviate Client Extensions

Add methods to the existing Weaviate client (`backend/internal/services/weaviate/client.go`):

- `EnsureConversationSummaryCollection()` — create collection if not exists (called at startup)
- `EnsureUserFactCollection()` — create collection if not exists
- `UpsertConversationSummary(ctx, conversationID, userID, content, agentUsed)` — store summary
- `UpsertUserFact(ctx, userID, content, category)` — store or update fact (with dedup)
- `SearchConversationSummaries(ctx, query, userID, limit)` — semantic search summaries
- `GetUserFacts(ctx, userID)` — get all facts for a user (filtered, not semantic)
- `SearchUserFacts(ctx, query, userID, limit)` — semantic search facts (for dedup check)

### Data Flow

```
Conversation ends (idle > 5min)
  ↓
Memory Worker picks it up
  ├─ Load messages from MySQL
  ├─ LLM extraction (summary + facts)
  ├─ Store ConversationSummary in Weaviate
  ├─ Upsert UserFacts in Weaviate (dedup by similarity)
  └─ Set summarized_at on conversation

New message arrives
  ↓
ChatHandler / Orchestrator
  ├─ Fetch user facts from Weaviate (~all, ~200 tokens)
  ├─ Semantic search summaries (top 3, ~300 tokens)
  ├─ Inject into router prompt → smarter routing + direct answers
  ├─ Route to agent (or answer directly)
  └─ Agent gets same memory in system prompt → context-aware responses
```

### Configuration

New env vars:
- `MEMORY_WORKER_INTERVAL` — how often the worker runs (default `5m`)

### Performance Impact

- **Memory creation**: Zero impact on request path (background worker)
- **Memory retrieval**: ~50-100ms added (2 Weaviate queries, local)
- **Token overhead**: ~300-500 tokens added per router/agent call
- **LLM cost for extraction**: ~700 tokens per conversation (one-time, background)

### Graceful Degradation

- Weaviate unavailable at startup: no memory collections created, agents work without memory
- Weaviate goes down mid-operation: memory fetch returns empty, agents use original prompts
- Worker fails to process a conversation: `summarized_at` stays NULL, retried next cycle
- LLM extraction fails: conversation skipped, retried next cycle

# Telegram Channel for PDT Assistant

**Date**: 2026-03-30
**Status**: Draft
**Approach**: Direct Integration (mirrors WhatsApp service pattern)

## Overview

Add Telegram as a first-class communication channel to PDT Assistant with full orchestrator parity. Users interact with PDT via a Telegram bot, with access controlled by a Telegram user ID whitelist. The bot supports live thinking status (edited-in-place messages), interactive confirmations via inline keyboard buttons, and long message splitting.

## Database Models

### `telegram_configs`

Stores bot configuration per PDT user.

| Column     | Type      | Notes                                      |
| ---------- | --------- | ------------------------------------------ |
| id         | uint (PK) | auto-increment                             |
| user_id    | uint (FK) | references `users.id`                      |
| bot_token  | string    | encrypted with AES-256-GCM (same as other tokens) |
| enabled    | bool      | toggle bot on/off                          |
| created_at | timestamp |                                            |
| updated_at | timestamp |                                            |

One PDT user has one bot token.

### `telegram_whitelist`

Maps allowed Telegram user IDs to PDT users.

| Column             | Type      | Notes                          |
| ------------------ | --------- | ------------------------------ |
| id                 | uint (PK) | auto-increment                 |
| user_id            | uint (FK) | references `users.id`          |
| telegram_config_id | uint (FK) | references `telegram_configs.id` |
| telegram_user_id   | int64     | Telegram's numeric user ID     |
| display_name       | string    | label for reference            |
| created_at         | timestamp |                                |

A whitelist entry maps a Telegram user ID → PDT user, so the orchestrator knows who's talking.

### Conversation mapping

Each unique Telegram chat ID is mapped to a PDT `Conversation` via a lookup table or by storing `telegram_chat_id` on the `Conversation` model. On first message from a chat, a new conversation is created. Subsequent messages in the same Telegram chat reuse the same conversation (continuous context). Users can start a new conversation by sending `/new`.

## Service Layer — `internal/services/telegram/`

Three files, mirroring the WhatsApp service pattern.

### `bot.go` — Bot Lifecycle

- Wraps `go-telegram-bot-api/telegram-bot-api` v5
- `NewBot(token string, ...) (*Bot, error)` — creates bot instance, validates token
- `Start(ctx context.Context)` — starts long-polling for updates (no webhook; simpler for personal tool)
- `Stop()` — graceful shutdown
- Holds a reference to the update handler

### `handler.go` — Incoming Update Processor

Receives every Telegram update from the bot's polling loop.

**Message flow:**

1. **Whitelist check**: look up `update.Message.From.ID` in `telegram_whitelist` → get `user_id`. If not found, silently ignore (no response to unknown users).
2. Get or create a `Conversation` for this Telegram chat ID.
3. Save user message as `ChatMessage`.
4. Load conversation history (last N messages, same as WebSocket).
5. Build orchestrator with user-scoped agents (same agent set as `ChatHandler`).
6. Call `orchestrator.HandleMessage()` with a `TelegramStreamWriter`.
7. Save assistant response + AI usage.
8. Scan for newly-created pending outbox entries and send follow-up confirmation messages with inline buttons.

**Callback query handler:**

- Parses callback data from inline button presses (`approve:<outboxID>` / `reject:<outboxID>`)
- Validates Telegram user is whitelisted
- Updates outbox status (approved/rejected)
- Edits the confirmation message to show result
- Answers the callback query (removes loading spinner)

### `stream_writer.go` — `agent.StreamWriter` Implementation

```
TelegramStreamWriter
├── WriteThinking(msg)    → edits the "thinking message" in-place with new status text
├── WriteToolStatus(t, s) → edits thinking message: appends tool status line
├── WriteContent(content) → accumulates content into a buffer
├── WriteDone()           → final-edits thinking message, sends final answer as NEW message(s)
└── WriteError(msg)       → sends error as a new message
```

Key difference from `wsStreamWriter`: WebSocket streams content incrementally; Telegram requires batching. Content is accumulated and only sent on `WriteDone()`. The thinking message is edited on each `WriteThinking`/`WriteToolStatus` call.

**Thinking message format (progressive):**

```
⏳ Thinking...

🔧 Using list_commits... running
```

```
⏳ Processing...

✅ list_commits — done
✅ get_sprint — done
🔧 generate_report... running
```

**Final thinking message (after `WriteDone()`):**

```
✅ Done (used: list_commits, get_sprint, generate_report)
```

Then the final answer follows as separate message(s).

## Interactive Confirmations

Confirmations (e.g., WhatsApp outbox approval) use Telegram inline keyboard buttons.

**Flow:**

1. After `WriteDone()`, the handler scans for any newly-created pending outbox entries created during this orchestrator run.
2. For each pending entry, sends a confirmation message with `InlineKeyboardMarkup`:
   - Text: the draft content + target info
   - Buttons: `[Approve]` `[Reject]`
   - Callback data: `approve:<outboxID>` / `reject:<outboxID>`
3. On button press, callback handler updates outbox status and edits the message to show result.

This avoids modifying the orchestrator or agent interfaces — it's purely a transport-layer concern.

**Stale buttons**: If a button is pressed after the entry has already been processed, the handler checks current status and responds with "Already approved" / "Already sent".

## Message Formatting & Splitting

**Formatting**: Telegram MarkdownV2 parse mode.

- Lightweight sanitizer escapes special MarkdownV2 characters not part of formatting.
- Preserves code blocks and bold formatting.
- Falls back to plain text if Telegram rejects the message.

**Splitting** for messages >4096 characters:

1. Split on paragraph boundaries (`\n\n`) first.
2. If a single paragraph exceeds 4096, split on line boundaries (`\n`).
3. If a single line exceeds 4096, hard-split at 4096.
4. Send each chunk sequentially with ~100ms delay to preserve ordering.

## Configuration

**New env vars:**

| Env Var              | Default | Notes                                              |
| -------------------- | ------- | -------------------------------------------------- |
| `TELEGRAM_BOT_TOKEN` | (empty) | Bot token from @BotFather. Bot won't start if empty. |
| `TELEGRAM_WHITELIST` | (empty) | Comma-separated `telegram_user_id:pdt_user_id` pairs, e.g. `123456:1,789012:1` |

**Config struct additions** (`config.go`):

- `TelegramBotToken string`
- `TelegramWhitelist string`

The whitelist env var seeds the `telegram_whitelist` DB table on first boot (only if table is empty). The DB table is the source of truth at runtime.

## Wiring in `main.go`

```go
// Telegram bot (optional)
if cfg.TelegramBotToken != "" && miniMaxClient != nil {
    tgBot := telegram.NewBot(cfg.TelegramBotToken, db, miniMaxClient, encryptor, r2Client, reportGen, waManager, weaviateClient)
    tgBot.SeedWhitelist(cfg.TelegramWhitelist)
    tgBot.Start(ctx)
    defer tgBot.Stop()
}
```

Same dependency injection pattern as WhatsApp. Graceful shutdown calls `tgBot.Stop()` alongside `waManager.Shutdown()`.

## Error Handling & Edge Cases

**Bot connectivity:**
- Long-polling failures retry with exponential backoff (1s → 2s → 4s → ... max 60s).
- Invalid token on startup logs a clear error and skips Telegram initialization (no crash).

**Whitelist misses:**
- Unknown Telegram users are silently ignored. No response reveals the bot's existence.

**Concurrent conversations:**
- Each Telegram chat ID gets its own conversation context.
- Multiple whitelisted users can talk simultaneously — each resolves to their own `user_id`.

**Rate limiting:**
- Thinking message edits throttled to max 1 edit/second (batch status updates) to avoid Telegram `429`.
- Message splitting sends with 100ms delays between chunks.

**Bot restart:**
- Long-polling is stateless. On restart, picks up from latest unprocessed update (Telegram holds updates 24h).
- No pending state to recover; conversations persist in DB.

## New Files Summary

```
backend/
├── internal/
│   ├── config/config.go              (add TelegramBotToken, TelegramWhitelist)
│   ├── models/telegram.go            (TelegramConfig, TelegramWhitelist models)
│   ├── services/telegram/
│   │   ├── bot.go                    (bot lifecycle, long-polling)
│   │   ├── handler.go                (update processor, callback handler)
│   │   └── stream_writer.go          (TelegramStreamWriter)
│   └── database/migrate.go           (add new models to AutoMigrate)
├── cmd/server/main.go                (wire up Telegram bot)
└── go.mod                            (add telegram-bot-api dependency)
```

## Out of Scope

- Webhook mode (long-polling is sufficient for personal tool)
- Telegram management UI in the frontend (manage via env vars / DB for now)
- Media file handling in Telegram responses (text only for v1)
- Telegram-specific agents (reuses existing agent set entirely)

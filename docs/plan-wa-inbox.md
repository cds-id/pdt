# Plan: Fix Executive Summary 404 + WhatsApp Web–style Inbox

## Part 1 — Executive Summary 404 (DONE)

**Cause**: Frontend executive endpoints prefixed `/protected`, but backend `protected := api.Group("")` has empty prefix. Real path `/api/reports/executive/...`. Frontend hit `/api/protected/reports/executive/...` → 404.

**Fix applied** (4 paths, drop `/protected`):
- `frontend/src/infrastructure/services/executiveReport.service.ts` (3)
- `frontend/src/presentation/hooks/useGenerateExecutiveReport.ts` (1)

**Verify**: `grep -rn "/protected" frontend/src` → clean. Rebuild frontend, retry generate.

---

## Part 2 — WhatsApp Inbox (Option A: reuse tables, listener = conversation)

### Design
- whatsmeow already wired (multi-device, QR pair). Reuse.
- Listener row = a conversation (chat JID). Auto-create on first inbound/outbound msg.
- `WaMessage` extended with `from_me` + `chat_jid` so own + peer messages both stored.
- New per-user WebSocket hub broadcasts new messages + receipts → live inbox.
- Send any media: upload from UI → R2 → whatsmeow upload+send. Inbound media already downloads to R2; extend send side to all types.
- Keep existing listener-monitor, outbox, agent features untouched.

### Schema changes (additive, AutoMigrate-safe)
`backend/internal/models/whatsapp.go`:
- `WaListener`: add `LastMessageAt *time.Time`, `LastMessagePreview string`, `UnreadCount int`. Add `Source string` (default `"monitor"`; inbox auto-rows use `"inbox"`) so inbox conversations distinguishable from explicit monitors. `IsActive` for inbox rows = true.
- `WaMessage`: add `FromMe bool` (col `from_me`), `ChatJID string` (col `chat_jid`, index), `Status string` (sent/delivered/read for outgoing).
No table drops. AutoMigrate adds columns.

### Backend tasks

**T1. Models + migrate**
- Add columns above. `database.go` AutoMigrate already lists Wa models — no list change needed.

**T2. Capture-all inbound (handler.go `handleMessage`)**
- Remove "ignore if no listener" + "ignore IsFromMe".
- Compute `chatJID = evt.Info.Chat`.
- Find-or-create listener for (NumberID, chatJID) with `Source="inbox"`, name from contact/group, type personal/group from JID server.
- Save `WaMessage` with `FromMe=evt.Info.IsFromMe`, `ChatJID`.
- Update listener `LastMessageAt`, `LastMessagePreview`, bump `UnreadCount` when `!FromMe`.
- Keep media download → R2 (all types already handled by `DownloadAny`).
- After save (and after media/vision update), broadcast via hub.

**T3. WebSocket hub** (new `backend/internal/services/whatsapp/hub.go`)
- `Hub` keyed by userID → set of conns (gorilla/websocket, mirror chat.go pattern).
- Methods: `Register(userID, conn)`, `Unregister`, `Broadcast(userID, event)`.
- Events: `{type:"message", payload:WaMessageDTO}`, `{type:"receipt", message_id, status}`, `{type:"chat_update", listener}`.
- Manager holds `*Hub`; handler calls it. Resolve userID from NumberID (cache numberID→userID in handler at construction).

**T4. Send all media (manager.go)**
- Generalize `sendImageMessage` → `sendMediaMessage(type, data, mime, caption, filename)` covering image/video/audio(ptt)/document/sticker via correct `waE2E.*Message`.
- New `Manager.SendInboxMessage(ctx, numberID, chatJID, text, mediaURL, mediaType)` → sends, then persists a `WaMessage{FromMe:true,...}`, updates listener preview, broadcasts. Returns saved msg.
- Receipts: `handleReceipt` already updates outbox; also update `WaMessage.Status` by `message_id` and broadcast receipt.

**T5. HTTP endpoints** (`handlers/whatsapp.go` + routes in `main.go` under `wa` group)
- `GET  /wa/numbers/:id/chats` — list listeners (conversations) for number, ordered by `last_message_at desc`, with unread + preview. (Reuse listener table.)
- `GET  /wa/chats/:listenerId/messages` — already exists as ListMessages; ensure returns `from_me`, media. Add ascending order option for chat view.
- `POST /wa/chats/:listenerId/read` — zero `UnreadCount`.
- `POST /wa/numbers/:id/send` — body `{chat_jid, text, media_url?, media_type?}` → `SendInboxMessage`. (Direct live send, bypasses outbox approval.)
- `POST /wa/media/upload` — multipart → R2 → returns `{url, mime, type}`. (For outgoing media from UI.) Reuse `R2Client.Upload`.
- `GET  /wa/ws` — upgrade, register conn in Hub for `user_id`, ping/pong keepalive (mirror chat.go).
- Ownership checks via existing JOIN pattern.

**T6. Wire Manager.Hub + userID resolution in main.go**
- Construct `Hub`, pass to Manager + WhatsAppHandler.
- `connectNumber` + pairing success path: handler already attached; ensure handler has Hub + userID.

### Frontend tasks

**T7. API + types**
- `api.constants.ts` WA: add `CHATS(numberId)`, `CHAT_MESSAGES(listenerId)`, `CHAT_READ(listenerId)`, `SEND(numberId)`, `MEDIA_UPLOAD`, `WS` (ws url builder).
- `whatsapp.interface.ts`: extend `IWaMessage` (`from_me`, `chat_jid`, `status`, media[]), add `IWaChat` (listener + preview/unread).
- `whatsapp.service.ts`: add `listChats`, `getChatMessages`, `markRead`, `sendMessage` (mutation), `uploadMedia` (mutation). Keep existing endpoints.

**T8. Realtime hook**
- `useWhatsAppSocket(numberId)` — opens `/api/wa/ws?token=...` (or Authorization via subprotocol like chat). On `message`/`receipt`/`chat_update`, dispatch RTK Query cache updates (`updateQueryData`) for chats + messages. Reconnect w/ backoff. Mirror existing chat WS client if present.

**T9. Inbox UI** (`presentation/pages/WhatsAppInboxPage.tsx` + `components/whatsapp/inbox/`)
WhatsApp-Web layout, 3-pane:
- Left: number selector (if multiple) + chat list (avatar, name, preview, time, unread badge), search.
- Center: message thread — bubbles left(peer)/right(me), media render (image/video/audio player, doc download, sticker), timestamps, delivery ticks (sent/delivered/read).
- Composer: text input, attach button (image/video/audio/doc), send. Attach → uploadMedia → send with media_url+type. Optimistic append.
- Empty states, loading skeletons, infinite scroll up for history (paginate `getChatMessages`).
- Use shadcn/ui + Tailwind, match existing dashboard style.

**T10. Route + nav**
- `routes/index.tsx`: add `dashboard/whatsapp` → `WhatsAppInboxPage`.
- `config/navigation.ts`: WhatsApp group add `{title:'Inbox', href:'/dashboard/whatsapp', icon: MessageSquare}` above Outbox.

### Testing / verification
- Backend: `cd backend && go build ./... && go test ./internal/services/whatsapp/... ./internal/handlers/...`
- Add handler tests for `/send`, `/chats`, capture-all in handler (extend existing `_test.go` if present).
- Frontend: `cd frontend && bun run build` (tsc) + `bun run test` (vitest) for service/hook.
- Manual: pair number → send/receive text+image+doc → see realtime + R2 URLs + ticks.

### Order of execution
T1 → T2 → T3 → T4 → T5 → T6 (backend, build+test each) → T7 → T8 → T9 → T10 (frontend, build).

### Risks / notes
- Capture-all = every chat stored. Acceptable per Option A. Could add per-number "inbox enabled" toggle later.
- WS auth: gorilla upgrader `CheckOrigin true` already. Token via query param (browsers can't set WS headers) — pass `?token=`; backend JWTAuth middleware reads Authorization only, so add WS-specific token-from-query in `/wa/ws` handler (validate JWT manually, like none exists yet — mirror middleware.JWTAuth logic).
- Voice notes: send as audio `ptt=true`. Stickers send optional (lower priority).
- Existing listener-monitor rows (`Source="monitor"`) coexist; inbox lists all listeners regardless of source, or filter—decide: **inbox shows all chats**, monitor toggle stays in ListenerManager.

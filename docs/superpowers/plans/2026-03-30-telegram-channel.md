# Telegram Channel Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Add Telegram as a first-class communication channel to PDT Assistant with full orchestrator parity, whitelist access control, live thinking messages, inline button confirmations, and message splitting.

**Architecture:** Direct integration mirroring the WhatsApp service pattern. A new `services/telegram/` package handles bot lifecycle, update processing, and a `TelegramStreamWriter` that implements `agent.StreamWriter`. Messages from whitelisted Telegram users are routed through the existing orchestrator with all agents available.

**Tech Stack:** Go, `go-telegram-bot-api/telegram-bot-api` v5, GORM (MySQL), existing MiniMax orchestrator

---

### Task 1: Add `telegram-bot-api` Dependency

**Files:**
- Modify: `backend/go.mod`

- [ ] **Step 1: Add the dependency**

```bash
cd /home/nst/GolandProjects/pdt/backend && go get github.com/go-telegram-bot-api/telegram-bot-api/v5@latest
```

- [ ] **Step 2: Verify it resolves**

```bash
cd /home/nst/GolandProjects/pdt/backend && go mod tidy
```

Expected: Clean exit, `go.sum` updated.

- [ ] **Step 3: Commit**

```bash
cd /home/nst/GolandProjects/pdt/backend && git add go.mod go.sum && git commit -m "feat(telegram): add telegram-bot-api dependency"
```

---

### Task 2: Database Models

**Files:**
- Create: `backend/internal/models/telegram.go`
- Modify: `backend/internal/models/conversation.go` (add `TelegramChatID` field)
- Modify: `backend/internal/database/database.go` (add to AutoMigrate)

- [ ] **Step 1: Create the Telegram models file**

Create `backend/internal/models/telegram.go`:

```go
package models

import "time"

type TelegramConfig struct {
	ID        uint      `gorm:"primarykey" json:"id"`
	UserID    uint      `gorm:"uniqueIndex;not null" json:"user_id"`
	BotToken  string    `gorm:"type:text;not null" json:"-"`
	Enabled   bool      `gorm:"default:true" json:"enabled"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
	User      User      `gorm:"foreignKey:UserID" json:"-"`
}

type TelegramWhitelist struct {
	ID               uint      `gorm:"primarykey" json:"id"`
	UserID           uint      `gorm:"index;not null" json:"user_id"`
	TelegramConfigID uint      `gorm:"index;not null" json:"telegram_config_id"`
	TelegramUserID   int64     `gorm:"index;not null" json:"telegram_user_id"`
	DisplayName      string    `gorm:"type:varchar(200)" json:"display_name"`
	CreatedAt        time.Time `json:"created_at"`
	User             User      `gorm:"foreignKey:UserID" json:"-"`
	TelegramConfig   TelegramConfig `gorm:"foreignKey:TelegramConfigID" json:"-"`
}
```

- [ ] **Step 2: Add `TelegramChatID` to Conversation model**

In `backend/internal/models/conversation.go`, add a `TelegramChatID` field to the `Conversation` struct:

```go
type Conversation struct {
	ID             string    `gorm:"type:varchar(36);primarykey" json:"id"`
	UserID         uint      `gorm:"index;not null" json:"user_id"`
	Title          string    `gorm:"type:varchar(255)" json:"title"`
	TelegramChatID int64     `gorm:"index" json:"telegram_chat_id,omitempty"`
	CreatedAt      time.Time `json:"created_at"`
	UpdatedAt      time.Time `json:"updated_at"`
	User           User      `gorm:"foreignKey:UserID" json:"-"`
	Messages       []ChatMessage `gorm:"foreignKey:ConversationID" json:"messages,omitempty"`
}
```

The only change is adding the `TelegramChatID` field. All other fields remain identical.

- [ ] **Step 3: Register models in AutoMigrate**

In `backend/internal/database/database.go`, add `&models.TelegramConfig{}` and `&models.TelegramWhitelist{}` to the `AutoMigrate` call:

```go
func Migrate(db *gorm.DB) error {
	if err := db.AutoMigrate(
		&models.User{},
		&models.Repository{},
		&models.Commit{},
		&models.CommitCardLink{},
		&models.JiraWorkspaceConfig{},
		&models.Sprint{},
		&models.JiraCard{},
		&models.ReportTemplate{},
		&models.Report{},
		&models.Conversation{},
		&models.ChatMessage{},
		&models.AIUsage{},
		&models.JiraComment{},
		&models.WaNumber{},
		&models.WaListener{},
		&models.WaMessage{},
		&models.WaMedia{},
		&models.WaOutbox{},
		&models.TelegramConfig{},
		&models.TelegramWhitelist{},
	); err != nil {
		return err
	}
```

- [ ] **Step 4: Verify it compiles**

```bash
cd /home/nst/GolandProjects/pdt/backend && go build ./...
```

Expected: Clean build, no errors.

- [ ] **Step 5: Commit**

```bash
git add backend/internal/models/telegram.go backend/internal/models/conversation.go backend/internal/database/database.go
git commit -m "feat(telegram): add database models for Telegram config, whitelist, and chat mapping"
```

---

### Task 3: Configuration

**Files:**
- Modify: `backend/internal/config/config.go`

- [ ] **Step 1: Add Telegram fields to Config struct**

In `backend/internal/config/config.go`, add two fields to the `Config` struct after the WhatsApp/Weaviate section:

```go
	// WhatsApp & Weaviate
	GeminiAPIKey    string
	WeaviateURL     string
	WhatsmeowDBPath string
	// Telegram
	TelegramBotToken  string
	TelegramWhitelist string
```

- [ ] **Step 2: Load the env vars in `Load()`**

In the `Load()` function, after `cfg.WhatsmeowDBPath = getEnv(...)`, add:

```go
	cfg.TelegramBotToken = getEnv("TELEGRAM_BOT_TOKEN", "")
	cfg.TelegramWhitelist = getEnv("TELEGRAM_WHITELIST", "")
```

- [ ] **Step 3: Verify it compiles**

```bash
cd /home/nst/GolandProjects/pdt/backend && go build ./...
```

Expected: Clean build.

- [ ] **Step 4: Commit**

```bash
git add backend/internal/config/config.go
git commit -m "feat(telegram): add Telegram config env vars"
```

---

### Task 4: TelegramStreamWriter

**Files:**
- Create: `backend/internal/services/telegram/stream_writer.go`

- [ ] **Step 1: Create the stream writer**

Create `backend/internal/services/telegram/stream_writer.go`:

```go
package telegram

import (
	"fmt"
	"strings"
	"sync"
	"time"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
)

// streamWriter implements agent.StreamWriter for Telegram.
type streamWriter struct {
	bot    *tgbotapi.BotAPI
	chatID int64

	mu             sync.Mutex
	thinkingMsgID  int
	toolStatuses   []toolStatus
	contentBuf     strings.Builder
	lastEditTime   time.Time
	pendingEdit    bool
}

type toolStatus struct {
	Name   string
	Status string // "executing" or "completed"
}

func newStreamWriter(bot *tgbotapi.BotAPI, chatID int64) *streamWriter {
	return &streamWriter{
		bot:    bot,
		chatID: chatID,
	}
}

func (w *streamWriter) WriteThinking(message string) error {
	w.mu.Lock()
	defer w.mu.Unlock()

	if w.thinkingMsgID == 0 {
		// Send initial thinking message
		msg := tgbotapi.NewMessage(w.chatID, "⏳ "+message)
		sent, err := w.bot.Send(msg)
		if err != nil {
			return fmt.Errorf("send thinking: %w", err)
		}
		w.thinkingMsgID = sent.MessageID
		w.lastEditTime = time.Now()
		return nil
	}

	return w.editThinkingLocked()
}

func (w *streamWriter) WriteToolStatus(toolName, status string) error {
	w.mu.Lock()
	defer w.mu.Unlock()

	// Update or append tool status
	found := false
	for i, ts := range w.toolStatuses {
		if ts.Name == toolName {
			w.toolStatuses[i].Status = status
			found = true
			break
		}
	}
	if !found {
		w.toolStatuses = append(w.toolStatuses, toolStatus{Name: toolName, Status: status})
	}

	// Throttle edits to max 1/second
	if time.Since(w.lastEditTime) < time.Second {
		w.pendingEdit = true
		return nil
	}

	return w.editThinkingLocked()
}

func (w *streamWriter) WriteContent(content string) error {
	w.mu.Lock()
	defer w.mu.Unlock()
	w.contentBuf.WriteString(content)
	return nil
}

func (w *streamWriter) WriteDone() error {
	w.mu.Lock()
	defer w.mu.Unlock()

	// Final edit on thinking message
	if w.thinkingMsgID != 0 {
		names := make([]string, 0, len(w.toolStatuses))
		for _, ts := range w.toolStatuses {
			names = append(names, ts.Name)
		}
		var text string
		if len(names) > 0 {
			text = "✅ Done (used: " + strings.Join(names, ", ") + ")"
		} else {
			text = "✅ Done"
		}
		edit := tgbotapi.NewEditMessageText(w.chatID, w.thinkingMsgID, text)
		w.bot.Send(edit)
	}

	// Send final content as new message(s)
	content := strings.TrimSpace(w.contentBuf.String())
	if content == "" {
		return nil
	}

	chunks := splitMessage(content, 4096)
	for i, chunk := range chunks {
		msg := tgbotapi.NewMessage(w.chatID, chunk)
		msg.ParseMode = "MarkdownV2"
		if _, err := w.bot.Send(msg); err != nil {
			// Retry without markdown on parse failure
			msg.ParseMode = ""
			msg.Text = chunk
			if _, err := w.bot.Send(msg); err != nil {
				return fmt.Errorf("send chunk %d: %w", i, err)
			}
		}
		if i < len(chunks)-1 {
			time.Sleep(100 * time.Millisecond)
		}
	}

	return nil
}

func (w *streamWriter) WriteError(msg string) error {
	errMsg := tgbotapi.NewMessage(w.chatID, "❌ Error: "+msg)
	_, err := w.bot.Send(errMsg)
	return err
}

// editThinkingLocked edits the thinking message with current tool statuses.
// Must be called with w.mu held.
func (w *streamWriter) editThinkingLocked() error {
	if w.thinkingMsgID == 0 {
		return nil
	}

	var sb strings.Builder
	sb.WriteString("⏳ Processing...\n")
	for _, ts := range w.toolStatuses {
		if ts.Status == "completed" {
			sb.WriteString(fmt.Sprintf("\n✅ %s — done", ts.Name))
		} else {
			sb.WriteString(fmt.Sprintf("\n🔧 %s... %s", ts.Name, ts.Status))
		}
	}

	edit := tgbotapi.NewEditMessageText(w.chatID, w.thinkingMsgID, sb.String())
	_, err := w.bot.Send(edit)
	w.lastEditTime = time.Now()
	w.pendingEdit = false
	return err
}

// splitMessage splits text into chunks of at most maxLen characters,
// preferring paragraph then line boundaries.
func splitMessage(text string, maxLen int) []string {
	if len(text) <= maxLen {
		return []string{text}
	}

	var chunks []string
	remaining := text

	for len(remaining) > 0 {
		if len(remaining) <= maxLen {
			chunks = append(chunks, remaining)
			break
		}

		// Try paragraph boundary
		chunk := remaining[:maxLen]
		if idx := strings.LastIndex(chunk, "\n\n"); idx > 0 {
			chunks = append(chunks, remaining[:idx])
			remaining = remaining[idx+2:]
			continue
		}

		// Try line boundary
		if idx := strings.LastIndex(chunk, "\n"); idx > 0 {
			chunks = append(chunks, remaining[:idx])
			remaining = remaining[idx+1:]
			continue
		}

		// Hard split
		chunks = append(chunks, chunk)
		remaining = remaining[maxLen:]
	}

	return chunks
}
```

- [ ] **Step 2: Verify it compiles**

```bash
cd /home/nst/GolandProjects/pdt/backend && go build ./...
```

Expected: Clean build.

- [ ] **Step 3: Commit**

```bash
git add backend/internal/services/telegram/stream_writer.go
git commit -m "feat(telegram): add TelegramStreamWriter implementing agent.StreamWriter"
```

---

### Task 5: Update Handler (Message Processing + Callback Queries)

**Files:**
- Create: `backend/internal/services/telegram/handler.go`

- [ ] **Step 1: Create the handler**

Create `backend/internal/services/telegram/handler.go`:

```go
package telegram

import (
	"context"
	"fmt"
	"log"
	"strconv"
	"strings"
	"time"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
	"gorm.io/gorm"

	"github.com/cds-id/pdt/backend/internal/ai/agent"
	"github.com/cds-id/pdt/backend/internal/ai/minimax"
	"github.com/cds-id/pdt/backend/internal/crypto"
	"github.com/cds-id/pdt/backend/internal/models"
	"github.com/cds-id/pdt/backend/internal/services/report"
	"github.com/cds-id/pdt/backend/internal/services/storage"
	waService "github.com/cds-id/pdt/backend/internal/services/whatsapp"
	wvClient "github.com/cds-id/pdt/backend/internal/services/weaviate"
)

type Handler struct {
	DB              *gorm.DB
	Bot             *tgbotapi.BotAPI
	MiniMaxClient   *minimax.Client
	Encryptor       *crypto.Encryptor
	R2              *storage.R2Client
	ReportGenerator *report.Generator
	ContextWindow   int
	WaManager       *waService.Manager
	WeaviateClient  *wvClient.Client
}

// resolveUser checks the whitelist and returns the PDT user_id for a Telegram user.
// Returns 0 if not whitelisted.
func (h *Handler) resolveUser(telegramUserID int64) uint {
	var entry models.TelegramWhitelist
	if err := h.DB.Where("telegram_user_id = ?", telegramUserID).First(&entry).Error; err != nil {
		return 0
	}
	return entry.UserID
}

// HandleUpdate processes a single Telegram update.
func (h *Handler) HandleUpdate(ctx context.Context, update tgbotapi.Update) {
	if update.CallbackQuery != nil {
		h.handleCallback(ctx, update.CallbackQuery)
		return
	}

	if update.Message == nil {
		return
	}

	// Handle /new command
	if update.Message.IsCommand() && update.Message.Command() == "new" {
		userID := h.resolveUser(update.Message.From.ID)
		if userID == 0 {
			return
		}
		// Create new conversation for this chat
		conv := models.Conversation{
			UserID:         userID,
			Title:          "Telegram conversation",
			TelegramChatID: update.Message.Chat.ID,
		}
		h.DB.Create(&conv)
		msg := tgbotapi.NewMessage(update.Message.Chat.ID, "New conversation started.")
		h.Bot.Send(msg)
		return
	}

	h.handleMessage(ctx, update.Message)
}

func (h *Handler) handleMessage(ctx context.Context, msg *tgbotapi.Message) {
	userID := h.resolveUser(msg.From.ID)
	if userID == 0 {
		return // silently ignore non-whitelisted users
	}

	chatID := msg.Chat.ID
	text := msg.Text
	if text == "" {
		text = msg.Caption // support media captions
	}
	if text == "" {
		return
	}

	// Get or create conversation for this Telegram chat
	var conv models.Conversation
	err := h.DB.Where("telegram_chat_id = ? AND user_id = ?", chatID, userID).
		Order("updated_at desc").
		First(&conv).Error
	if err != nil {
		conv = models.Conversation{
			UserID:         userID,
			Title:          truncate(text, 80),
			TelegramChatID: chatID,
		}
		h.DB.Create(&conv)
	}

	// Save user message
	userMsg := models.ChatMessage{
		ConversationID: conv.ID,
		Role:           "user",
		Content:        text,
	}
	h.DB.Create(&userMsg)

	// Load conversation history
	limit := h.ContextWindow
	if limit <= 0 {
		limit = 20
	}
	var history []models.ChatMessage
	h.DB.Where("conversation_id = ?", conv.ID).
		Order("created_at desc").
		Limit(limit).
		Find(&history)

	// Reverse to chronological order
	for i, j := 0, len(history)-1; i < j; i, j = i+1, j-1 {
		history[i], history[j] = history[j], history[i]
	}

	messages := make([]minimax.Message, 0, len(history))
	for _, m := range history {
		messages = append(messages, minimax.Message{
			Role:    m.Role,
			Content: m.Content,
		})
	}

	// Build orchestrator
	orchestrator := agent.NewOrchestrator(
		h.MiniMaxClient,
		&agent.GitAgent{DB: h.DB, UserID: userID, Encryptor: h.Encryptor, Weaviate: h.WeaviateClient},
		&agent.JiraAgent{DB: h.DB, UserID: userID, Weaviate: h.WeaviateClient},
		&agent.ReportAgent{DB: h.DB, UserID: userID, Generator: h.ReportGenerator, R2: h.R2},
		&agent.ProofAgent{DB: h.DB, UserID: userID},
		&agent.BriefingAgent{DB: h.DB, UserID: userID},
		&agent.WhatsAppAgent{DB: h.DB, UserID: userID, Weaviate: h.WeaviateClient, Manager: h.WaManager},
	)

	// Record outbox count before orchestrator run
	var outboxCountBefore int64
	h.DB.Model(&models.WaOutbox{}).
		Joins("JOIN wa_numbers ON wa_numbers.id = wa_outboxes.wa_number_id").
		Where("wa_numbers.user_id = ? AND wa_outboxes.status = ?", userID, "pending").
		Count(&outboxCountBefore)

	writer := newStreamWriter(h.Bot, chatID)

	result, err := orchestrator.HandleMessage(ctx, messages, writer)
	if err != nil {
		log.Printf("[telegram] orchestrator error: %v", err)
		writer.WriteError(err.Error())
		return
	}

	// Save assistant response
	if result.FullResponse != "" {
		assistantMsg := models.ChatMessage{
			ConversationID: conv.ID,
			Role:           "assistant",
			Content:        result.FullResponse,
		}
		h.DB.Create(&assistantMsg)
	}

	// Log AI usage
	if result.Usage.PromptTokens > 0 || result.Usage.CompletionTokens > 0 {
		usage := models.AIUsage{
			UserID:           userID,
			ConversationID:   conv.ID,
			Provider:         "minimax",
			Model:            "MiniMax-M2.7",
			Feature:          "telegram",
			PromptTokens:     result.Usage.PromptTokens,
			CompletionTokens: result.Usage.CompletionTokens,
		}
		h.DB.Create(&usage)
	}

	h.DB.Model(&conv).Update("updated_at", time.Now())

	// Check for new pending outbox entries and send confirmation buttons
	h.sendOutboxConfirmations(userID, chatID, outboxCountBefore)
}

// sendOutboxConfirmations checks for new pending outbox entries and sends
// inline keyboard confirmation messages for each.
func (h *Handler) sendOutboxConfirmations(userID uint, chatID int64, countBefore int64) {
	var pendingOutbox []models.WaOutbox
	h.DB.Joins("JOIN wa_numbers ON wa_numbers.id = wa_outboxes.wa_number_id").
		Where("wa_numbers.user_id = ? AND wa_outboxes.status = ?", userID, "pending").
		Order("wa_outboxes.created_at desc").
		Find(&pendingOutbox)

	if int64(len(pendingOutbox)) <= countBefore {
		return // no new entries
	}

	// Send confirmation for new entries (newest first, take only new ones)
	newCount := int64(len(pendingOutbox)) - countBefore
	for i := 0; i < int(newCount) && i < len(pendingOutbox); i++ {
		entry := pendingOutbox[i]
		text := fmt.Sprintf("📤 *Outbox Message*\nTo: %s\n\n%s", entry.TargetName, entry.Content)
		if len(text) > 4000 {
			text = text[:4000] + "..."
		}

		keyboard := tgbotapi.NewInlineKeyboardMarkup(
			tgbotapi.NewInlineKeyboardRow(
				tgbotapi.NewInlineKeyboardButtonData("✅ Approve", fmt.Sprintf("approve:%d", entry.ID)),
				tgbotapi.NewInlineKeyboardButtonData("❌ Reject", fmt.Sprintf("reject:%d", entry.ID)),
			),
		)

		msg := tgbotapi.NewMessage(chatID, text)
		msg.ReplyMarkup = keyboard
		h.Bot.Send(msg)
	}
}

// handleCallback processes inline keyboard button presses.
func (h *Handler) handleCallback(ctx context.Context, callback *tgbotapi.CallbackQuery) {
	userID := h.resolveUser(callback.From.ID)
	if userID == 0 {
		return
	}

	data := callback.Data
	parts := strings.SplitN(data, ":", 2)
	if len(parts) != 2 {
		return
	}

	action := parts[0]
	outboxID, err := strconv.ParseUint(parts[1], 10, 64)
	if err != nil {
		return
	}

	// Load outbox entry and verify ownership
	var entry models.WaOutbox
	if err := h.DB.Joins("JOIN wa_numbers ON wa_numbers.id = wa_outboxes.wa_number_id").
		Where("wa_outboxes.id = ? AND wa_numbers.user_id = ?", outboxID, userID).
		First(&entry).Error; err != nil {
		answer := tgbotapi.NewCallback(callback.ID, "Entry not found")
		h.Bot.Send(answer)
		return
	}

	// Handle stale buttons
	if entry.Status != "pending" {
		answer := tgbotapi.NewCallback(callback.ID, "Already "+entry.Status)
		h.Bot.Send(answer)

		edit := tgbotapi.NewEditMessageText(callback.Message.Chat.ID, callback.Message.MessageID,
			fmt.Sprintf("Already %s", entry.Status))
		h.Bot.Send(edit)
		return
	}

	var resultText string
	switch action {
	case "approve":
		now := time.Now()
		h.DB.Model(&entry).Updates(map[string]any{
			"status":      "approved",
			"approved_at": &now,
		})
		resultText = "✅ Approved — sending..."
	case "reject":
		h.DB.Model(&entry).Update("status", "rejected")
		resultText = "❌ Rejected"
	default:
		return
	}

	// Answer callback (removes loading spinner)
	answer := tgbotapi.NewCallback(callback.ID, resultText)
	h.Bot.Send(answer)

	// Edit the original message to show result
	edit := tgbotapi.NewEditMessageText(callback.Message.Chat.ID, callback.Message.MessageID, resultText)
	h.Bot.Send(edit)
}

func truncate(s string, n int) string {
	runes := []rune(s)
	if len(runes) <= n {
		return s
	}
	return string(runes[:n]) + "..."
}
```

- [ ] **Step 2: Verify it compiles**

```bash
cd /home/nst/GolandProjects/pdt/backend && go build ./...
```

Expected: Clean build.

- [ ] **Step 3: Commit**

```bash
git add backend/internal/services/telegram/handler.go
git commit -m "feat(telegram): add update handler with message processing and callback queries"
```

---

### Task 6: Bot Lifecycle

**Files:**
- Create: `backend/internal/services/telegram/bot.go`

- [ ] **Step 1: Create the bot lifecycle manager**

Create `backend/internal/services/telegram/bot.go`:

```go
package telegram

import (
	"context"
	"fmt"
	"log"
	"strconv"
	"strings"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
	"gorm.io/gorm"

	"github.com/cds-id/pdt/backend/internal/ai/minimax"
	"github.com/cds-id/pdt/backend/internal/crypto"
	"github.com/cds-id/pdt/backend/internal/models"
	"github.com/cds-id/pdt/backend/internal/services/report"
	"github.com/cds-id/pdt/backend/internal/services/storage"
	waService "github.com/cds-id/pdt/backend/internal/services/whatsapp"
	wvClient "github.com/cds-id/pdt/backend/internal/services/weaviate"
)

type Bot struct {
	api     *tgbotapi.BotAPI
	handler *Handler
	cancel  context.CancelFunc
}

// NewBot creates a Telegram bot instance and validates the token.
func NewBot(
	token string,
	db *gorm.DB,
	miniMaxClient *minimax.Client,
	encryptor *crypto.Encryptor,
	r2 *storage.R2Client,
	reportGen *report.Generator,
	contextWindow int,
	waManager *waService.Manager,
	weaviateClient *wvClient.Client,
) (*Bot, error) {
	api, err := tgbotapi.NewBotAPI(token)
	if err != nil {
		return nil, err
	}

	log.Printf("[telegram] authorized as @%s", api.Self.UserName)

	handler := &Handler{
		DB:              db,
		Bot:             api,
		MiniMaxClient:   miniMaxClient,
		Encryptor:       encryptor,
		R2:              r2,
		ReportGenerator: reportGen,
		ContextWindow:   contextWindow,
		WaManager:       waManager,
		WeaviateClient:  weaviateClient,
	}

	return &Bot{
		api:     api,
		handler: handler,
	}, nil
}

// SeedWhitelist parses the whitelist string (format: "tgUserID:pdtUserID,...")
// and inserts entries if the table is empty.
func (b *Bot) SeedWhitelist(whitelist string) {
	if whitelist == "" {
		return
	}

	var count int64
	b.handler.DB.Model(&models.TelegramWhitelist{}).Count(&count)
	if count > 0 {
		return // already seeded
	}

	// Ensure a TelegramConfig exists for each PDT user referenced
	configCache := make(map[uint]uint) // pdtUserID -> configID

	pairs := strings.Split(whitelist, ",")
	for _, pair := range pairs {
		parts := strings.SplitN(strings.TrimSpace(pair), ":", 2)
		if len(parts) != 2 {
			log.Printf("[telegram] invalid whitelist entry: %s", pair)
			continue
		}

		tgUserID, err := strconv.ParseInt(parts[0], 10, 64)
		if err != nil {
			log.Printf("[telegram] invalid telegram user ID: %s", parts[0])
			continue
		}

		pdtUserID, err := strconv.ParseUint(parts[1], 10, 64)
		if err != nil {
			log.Printf("[telegram] invalid PDT user ID: %s", parts[1])
			continue
		}

		uid := uint(pdtUserID)

		configID, ok := configCache[uid]
		if !ok {
			// Get or create config for this user
			var cfg models.TelegramConfig
			err := b.handler.DB.Where("user_id = ?", uid).First(&cfg).Error
			if err != nil {
				cfg = models.TelegramConfig{
					UserID:  uid,
					Enabled: true,
				}
				b.handler.DB.Create(&cfg)
			}
			configID = cfg.ID
			configCache[uid] = configID
		}

		entry := models.TelegramWhitelist{
			UserID:           uid,
			TelegramConfigID: configID,
			TelegramUserID:   tgUserID,
			DisplayName:      fmt.Sprintf("TG:%d", tgUserID),
		}
		b.handler.DB.Create(&entry)
		log.Printf("[telegram] whitelist seeded: tg=%d -> pdt=%d", tgUserID, uid)
	}
}

// Start begins long-polling for Telegram updates.
func (b *Bot) Start(ctx context.Context) {
	pollCtx, cancel := context.WithCancel(ctx)
	b.cancel = cancel

	go func() {
		u := tgbotapi.NewUpdate(0)
		u.Timeout = 30

		updates := b.api.GetUpdatesChan(u)

		for {
			select {
			case <-pollCtx.Done():
				log.Println("[telegram] polling stopped")
				return
			case update, ok := <-updates:
				if !ok {
					return
				}
				go b.handler.HandleUpdate(pollCtx, update)
			}
		}
	}()

	log.Println("[telegram] long-polling started")
}

// Stop gracefully stops the bot.
func (b *Bot) Stop() {
	if b.cancel != nil {
		b.cancel()
	}
	b.api.StopReceivingUpdates()
	log.Println("[telegram] bot stopped")
}
```

- [ ] **Step 2: Verify it compiles**

```bash
cd /home/nst/GolandProjects/pdt/backend && go build ./...
```

Expected: Clean build.

- [ ] **Step 4: Commit**

```bash
git add backend/internal/services/telegram/bot.go
git commit -m "feat(telegram): add bot lifecycle with long-polling and whitelist seeding"
```

---

### Task 7: Wire Up in main.go

**Files:**
- Modify: `backend/cmd/server/main.go`

- [ ] **Step 1: Add the import**

In `backend/cmd/server/main.go`, add the Telegram service import alongside the existing WhatsApp import:

```go
	waService "github.com/cds-id/pdt/backend/internal/services/whatsapp"
	tgService "github.com/cds-id/pdt/backend/internal/services/telegram"
	wvService "github.com/cds-id/pdt/backend/internal/services/weaviate"
```

- [ ] **Step 2: Add Telegram bot initialization**

After the `chatHandler` and `waHandler` setup (around line 125, before `// Router`), add:

```go
	// Telegram bot (optional)
	var tgBot *tgService.Bot
	if cfg.TelegramBotToken != "" && miniMaxClient != nil {
		tgBot, err = tgService.NewBot(
			cfg.TelegramBotToken,
			db,
			miniMaxClient,
			encryptor,
			r2Client,
			reportGen,
			cfg.AIContextWindow,
			waManager,
			weaviateClient,
		)
		if err != nil {
			log.Printf("Telegram bot init failed: %v", err)
		} else {
			tgBot.SeedWhitelist(cfg.TelegramWhitelist)
			tgBot.Start(ctx)
		}
	}
```

- [ ] **Step 3: Add graceful shutdown**

In the shutdown section, after `waManager.Shutdown()` (around line 273), add:

```go
	if tgBot != nil {
		tgBot.Stop()
	}
```

- [ ] **Step 4: Verify it compiles**

```bash
cd /home/nst/GolandProjects/pdt/backend && go build ./...
```

Expected: Clean build.

- [ ] **Step 5: Commit**

```bash
git add backend/cmd/server/main.go
git commit -m "feat(telegram): wire up Telegram bot in main.go with graceful shutdown"
```

---

### Task 8: MarkdownV2 Sanitizer

**Files:**
- Modify: `backend/internal/services/telegram/stream_writer.go`

- [ ] **Step 1: Add the sanitizer function**

Add this function to `backend/internal/services/telegram/stream_writer.go`, before the `splitMessage` function:

```go
// escapeMarkdownV2 escapes special characters for Telegram MarkdownV2 parse mode,
// preserving code blocks and bold formatting.
func escapeMarkdownV2(text string) string {
	// First, extract and protect code blocks
	var result strings.Builder
	parts := strings.Split(text, "```")

	for i, part := range parts {
		if i%2 == 1 {
			// Inside code block — don't escape
			result.WriteString("```")
			result.WriteString(part)
			result.WriteString("```")
			continue
		}

		// Outside code block — escape special chars except * (bold)
		specialChars := []string{"_", "[", "]", "(", ")", "~", "`", ">", "#", "+", "-", "=", "|", "{", "}", ".", "!"}
		escaped := part
		for _, ch := range specialChars {
			escaped = strings.ReplaceAll(escaped, ch, "\\"+ch)
		}
		result.WriteString(escaped)
	}

	return result.String()
}
```

- [ ] **Step 2: Use the sanitizer in WriteDone**

In `WriteDone()`, update the message sending to escape content before sending with MarkdownV2:

Replace:
```go
		msg := tgbotapi.NewMessage(w.chatID, chunk)
		msg.ParseMode = "MarkdownV2"
```

With:
```go
		msg := tgbotapi.NewMessage(w.chatID, escapeMarkdownV2(chunk))
		msg.ParseMode = "MarkdownV2"
```

- [ ] **Step 3: Verify it compiles**

```bash
cd /home/nst/GolandProjects/pdt/backend && go build ./...
```

Expected: Clean build.

- [ ] **Step 4: Commit**

```bash
git add backend/internal/services/telegram/stream_writer.go
git commit -m "feat(telegram): add MarkdownV2 sanitizer for Telegram message formatting"
```

---

### Task 9: Flush Pending Edits on WriteDone

**Files:**
- Modify: `backend/internal/services/telegram/stream_writer.go`

- [ ] **Step 1: Flush throttled edits before sending final content**

In `WriteDone()`, add a pending edit flush right before the final thinking message edit. After `w.mu.Lock()` and before the `// Final edit on thinking message` comment, add:

```go
	// Flush any throttled pending edit
	if w.pendingEdit && w.thinkingMsgID != 0 {
		w.editThinkingLocked()
	}
```

- [ ] **Step 2: Verify it compiles**

```bash
cd /home/nst/GolandProjects/pdt/backend && go build ./...
```

Expected: Clean build.

- [ ] **Step 3: Commit**

```bash
git add backend/internal/services/telegram/stream_writer.go
git commit -m "fix(telegram): flush pending throttled edits before sending final content"
```

---

### Task 10: End-to-End Manual Test

**Files:**
- Modify: `backend/.env` (add Telegram env vars)

- [ ] **Step 1: Create a test bot via @BotFather**

Open Telegram, message @BotFather, run `/newbot`, and get the bot token.

- [ ] **Step 2: Get your Telegram user ID**

Message @userinfobot on Telegram to get your numeric user ID.

- [ ] **Step 3: Add env vars**

Add to `backend/.env`:

```
TELEGRAM_BOT_TOKEN=<your-bot-token>
TELEGRAM_WHITELIST=<your-telegram-user-id>:1
```

(Replace `1` with your actual PDT user ID)

- [ ] **Step 4: Start the server**

```bash
cd /home/nst/GolandProjects/pdt/backend && go run ./cmd/server/
```

Expected: Logs show `[telegram] authorized as @YourBotName` and `[telegram] long-polling started`.

- [ ] **Step 5: Test basic message**

Send "what commits did I push today?" to your Telegram bot.

Expected:
- A thinking message appears and updates with tool statuses
- Final thinking message shows "✅ Done (used: ...)"
- A separate response message with the answer follows

- [ ] **Step 6: Test /new command**

Send `/new` to start a fresh conversation, then send a new question.

Expected: New conversation context, no memory of previous messages.

- [ ] **Step 7: Test confirmation buttons**

Send "send a WhatsApp message to [some contact] saying hello" (requires WhatsApp to be set up).

Expected: After the response, an outbox confirmation message appears with Approve/Reject inline buttons.

- [ ] **Step 8: Test unknown user**

Send a message from a non-whitelisted Telegram account.

Expected: No response at all (silent ignore).

- [ ] **Step 9: Commit .env.example update**

Do NOT commit `.env`. Instead, update `.env.example` (if it exists) or document the new env vars:

```bash
git add -A && git commit -m "feat(telegram): complete Telegram channel integration"
```

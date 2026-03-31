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
	"github.com/cds-id/pdt/backend/internal/scheduler"
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
	ScheduleEngine  *scheduler.Engine
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
	log.Printf("[telegram] update received: message=%v callback=%v", update.Message != nil, update.CallbackQuery != nil)

	if update.CallbackQuery != nil {
		h.handleCallback(ctx, update.CallbackQuery)
		return
	}

	if update.Message == nil {
		return
	}

	// Handle commands
	if update.Message.IsCommand() {
		userID := h.resolveUser(update.Message.From.ID)
		if userID == 0 {
			return
		}
		switch update.Message.Command() {
		case "start":
			msg := tgbotapi.NewMessage(update.Message.Chat.ID, "PDT Assistant ready. Send me a message to get started.\n\nUse /new to start a fresh conversation.")
			h.Bot.Send(msg)
			return
		case "new":
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
	}

	h.handleMessage(ctx, update.Message)
}

func (h *Handler) handleMessage(ctx context.Context, msg *tgbotapi.Message) {
	log.Printf("[telegram] received message from user %d: %s", msg.From.ID, msg.Text)

	userID := h.resolveUser(msg.From.ID)
	if userID == 0 {
		log.Printf("[telegram] user %d not whitelisted, ignoring", msg.From.ID)
		return
	}
	log.Printf("[telegram] resolved to PDT user %d", userID)

	chatID := msg.Chat.ID
	text := msg.Text
	if text == "" {
		text = msg.Caption
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
		&agent.SchedulerAgent{DB: h.DB, UserID: userID, Engine: h.ScheduleEngine},
	)

	// Record max outbox ID before orchestrator run
	var maxOutboxIDBefore uint
	var lastOutbox models.WaOutbox
	if err := h.DB.Joins("JOIN wa_numbers ON wa_numbers.id = wa_outboxes.wa_number_id").
		Where("wa_numbers.user_id = ?", userID).
		Order("wa_outboxes.id desc").Limit(1).
		First(&lastOutbox).Error; err == nil {
		maxOutboxIDBefore = lastOutbox.ID
	}

	writer := newStreamWriter(h.Bot, chatID)

	// Send typing indicator repeatedly until orchestrator finishes
	stopTyping := make(chan struct{})
	go func() {
		ticker := time.NewTicker(4 * time.Second)
		defer ticker.Stop()
		// Send immediately
		h.Bot.Send(tgbotapi.NewChatAction(chatID, tgbotapi.ChatTyping))
		for {
			select {
			case <-stopTyping:
				return
			case <-ticker.C:
				h.Bot.Send(tgbotapi.NewChatAction(chatID, tgbotapi.ChatTyping))
			}
		}
	}()

	log.Printf("[telegram] calling orchestrator with %d messages", len(messages))
	result, err := orchestrator.HandleMessage(ctx, messages, writer)
	close(stopTyping)
	if err != nil {
		log.Printf("[telegram] orchestrator error: %v", err)
		writer.WriteError(err.Error())
		return
	}
	log.Printf("[telegram] orchestrator done, response length: %d", len(result.FullResponse))

	// Flush buffered content to Telegram
	if err := writer.WriteDone(); err != nil {
		log.Printf("[telegram] WriteDone error: %v", err)
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
	h.sendOutboxConfirmations(userID, chatID, maxOutboxIDBefore)
}

// sendOutboxConfirmations checks for new pending outbox entries and sends
// inline keyboard confirmation messages for each.
func (h *Handler) sendOutboxConfirmations(userID uint, chatID int64, maxIDBefore uint) {
	var newOutbox []models.WaOutbox
	h.DB.Joins("JOIN wa_numbers ON wa_numbers.id = wa_outboxes.wa_number_id").
		Where("wa_numbers.user_id = ? AND wa_outboxes.status = ? AND wa_outboxes.id > ?", userID, "pending", maxIDBefore).
		Order("wa_outboxes.created_at asc").
		Find(&newOutbox)

	for _, entry := range newOutbox {
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

	var entry models.WaOutbox
	if err := h.DB.Joins("JOIN wa_numbers ON wa_numbers.id = wa_outboxes.wa_number_id").
		Where("wa_outboxes.id = ? AND wa_numbers.user_id = ?", outboxID, userID).
		First(&entry).Error; err != nil {
		answer := tgbotapi.NewCallback(callback.ID, "Entry not found")
		h.Bot.Send(answer)
		return
	}

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

	answer := tgbotapi.NewCallback(callback.ID, resultText)
	h.Bot.Send(answer)
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

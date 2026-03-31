package scheduler

import (
	"fmt"
	"log"

	"github.com/cds-id/pdt/backend/internal/models"
	"github.com/cds-id/pdt/backend/internal/services/telegram/formatter"
	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
	"gorm.io/gorm"
)

type Notifier struct {
	DB  *gorm.DB
	Bot *tgbotapi.BotAPI
}

func (n *Notifier) NotifyRunCompleted(run *models.AgentScheduleRun, scheduleName string) {
	if n == nil || n.Bot == nil {
		log.Printf("[scheduler] notifier skipped: notifier=%v, bot=%v", n != nil, n != nil && n.Bot != nil)
		return
	}

	chatID := n.findTelegramChat(run.UserID)
	if chatID == 0 {
		log.Printf("[scheduler] telegram notification skipped: no chat ID found for user %d", run.UserID)
		return
	}

	var text string
	if run.Status == "completed" {
		summary := run.ResultSummary
		if len([]rune(summary)) > 1500 {
			summary = string([]rune(summary)[:1500]) + "..."
		}
		text = fmt.Sprintf("📋 *Scheduled: %s*\n✅ Completed\n\n%s", scheduleName, summary)
	} else {
		errMsg := run.Error
		if len(errMsg) > 500 {
			errMsg = errMsg[:500] + "..."
		}
		text = fmt.Sprintf("📋 *Scheduled: %s*\n❌ Failed: %s", scheduleName, errMsg)
	}

	n.sendTelegram(chatID, text)
	log.Printf("[scheduler] telegram notification sent to chat %d for schedule %q", chatID, scheduleName)
}

// SendFullResponse sends the complete agent response to Telegram.
func (n *Notifier) SendFullResponse(userID uint, scheduleName, fullResponse string) {
	if n == nil || n.Bot == nil {
		return
	}

	chatID := n.findTelegramChat(userID)
	if chatID == 0 {
		return
	}

	text := fmt.Sprintf("📋 *Scheduled: %s*\n\n%s", scheduleName, fullResponse)

	// Telegram has 4096 char limit, split if needed
	runes := []rune(text)
	for len(runes) > 0 {
		chunk := runes
		if len(chunk) > 4000 {
			chunk = runes[:4000]
			runes = runes[4000:]
		} else {
			runes = nil
		}
		n.sendTelegram(chatID, string(chunk))
	}
}

func (n *Notifier) sendTelegram(chatID int64, text string) {
	htmlContent := formatter.ToTelegramHTML(text)
	msg := tgbotapi.NewMessage(chatID, htmlContent)
	msg.ParseMode = "HTML"
	if _, err := n.Bot.Send(msg); err != nil {
		log.Printf("[scheduler] telegram HTML send failed: %v, falling back to plain text", err)
		msg.ParseMode = ""
		msg.Text = text
		if _, err2 := n.Bot.Send(msg); err2 != nil {
			log.Printf("[scheduler] telegram plain text send also failed: %v", err2)
		}
	}
}

func (n *Notifier) findTelegramChat(userID uint) int64 {
	var whitelist models.TelegramWhitelist
	if err := n.DB.Where("user_id = ?", userID).First(&whitelist).Error; err != nil {
		log.Printf("[scheduler] no telegram whitelist entry for user %d", userID)
		return 0
	}

	// Try to find a conversation with a telegram chat ID
	var conv models.Conversation
	if err := n.DB.Where("user_id = ? AND telegram_chat_id != 0", userID).
		Order("updated_at desc").First(&conv).Error; err != nil {
		// Fallback: use telegram user ID as private chat ID
		log.Printf("[scheduler] using whitelist telegram_user_id %d for user %d", whitelist.TelegramUserID, userID)
		return whitelist.TelegramUserID
	}
	return conv.TelegramChatID
}

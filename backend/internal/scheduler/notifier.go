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
		return
	}

	chatID := n.findTelegramChat(run.UserID)
	if chatID == 0 {
		return
	}

	var text string
	if run.Status == "completed" {
		summary := run.ResultSummary
		if len([]rune(summary)) > 500 {
			summary = string([]rune(summary)[:500]) + "..."
		}
		text = fmt.Sprintf("📋 **Scheduled: %s**\n✅ Completed\n\n%s", scheduleName, summary)
	} else {
		errMsg := run.Error
		if len(errMsg) > 200 {
			errMsg = errMsg[:200] + "..."
		}
		text = fmt.Sprintf("📋 **Scheduled: %s**\n❌ Failed: %s", scheduleName, errMsg)
	}

	htmlContent := formatter.ToTelegramHTML(text)
	msg := tgbotapi.NewMessage(chatID, htmlContent)
	msg.ParseMode = "HTML"
	if _, err := n.Bot.Send(msg); err != nil {
		log.Printf("[scheduler] telegram notification failed: %v", err)
		msg.ParseMode = ""
		msg.Text = text
		n.Bot.Send(msg) //nolint:errcheck
	}
}

func (n *Notifier) findTelegramChat(userID uint) int64 {
	var whitelist models.TelegramWhitelist
	if err := n.DB.Where("user_id = ?", userID).First(&whitelist).Error; err != nil {
		return 0
	}
	var conv models.Conversation
	if err := n.DB.Where("user_id = ? AND telegram_chat_id != 0", userID).
		Order("updated_at desc").First(&conv).Error; err != nil {
		return whitelist.TelegramUserID
	}
	return conv.TelegramChatID
}

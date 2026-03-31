package telegram

import (
	"context"
	"fmt"
	"log"
	"strconv"
	"strings"
	"sync"
	"time"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
	"gorm.io/gorm"

	"github.com/cds-id/pdt/backend/internal/ai/minimax"
	"github.com/cds-id/pdt/backend/internal/crypto"
	"github.com/cds-id/pdt/backend/internal/models"
	"github.com/cds-id/pdt/backend/internal/scheduler"
	"github.com/cds-id/pdt/backend/internal/services/report"
	"github.com/cds-id/pdt/backend/internal/services/storage"
	waService "github.com/cds-id/pdt/backend/internal/services/whatsapp"
	wvClient "github.com/cds-id/pdt/backend/internal/services/weaviate"
)

type Bot struct {
	api       *tgbotapi.BotAPI
	handler   *Handler
	encryptor *crypto.Encryptor
	token     string
	cancel    context.CancelFunc
}

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
		api:       api,
		handler:   handler,
		encryptor: encryptor,
		token:     token,
	}, nil
}

func (b *Bot) SeedWhitelist(whitelist string) {
	if whitelist == "" {
		return
	}

	var count int64
	b.handler.DB.Model(&models.TelegramWhitelist{}).Count(&count)
	if count > 0 {
		return
	}

	configCache := make(map[uint]uint)

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
			var cfg models.TelegramConfig
			err := b.handler.DB.Where("user_id = ?", uid).First(&cfg).Error
			if err != nil {
				encToken, _ := b.encryptor.Encrypt(b.token)
				cfg = models.TelegramConfig{
					UserID:   uid,
					BotToken: encToken,
					Enabled:  true,
				}
				if err := b.handler.DB.Create(&cfg).Error; err != nil {
					log.Printf("[telegram] failed to create config for user %d: %v", uid, err)
					continue
				}
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
		if err := b.handler.DB.Create(&entry).Error; err != nil {
			log.Printf("[telegram] failed to seed whitelist entry tg=%d: %v", tgUserID, err)
			continue
		}
		log.Printf("[telegram] whitelist seeded: tg=%d -> pdt=%d", tgUserID, uid)
	}
}

func (b *Bot) Start(ctx context.Context) {
	pollCtx, cancel := context.WithCancel(ctx)
	b.cancel = cancel

	// Per-chat mutex to serialize message processing within each chat
	chatMu := &sync.Map{} // map[int64]*sync.Mutex

	getChatMu := func(chatID int64) *sync.Mutex {
		val, _ := chatMu.LoadOrStore(chatID, &sync.Mutex{})
		return val.(*sync.Mutex)
	}

	go func() {
		backoff := time.Second
		const maxBackoff = 60 * time.Second
		firstUpdate := true

		for {
			if pollCtx.Err() != nil {
				log.Println("[telegram] polling stopped")
				return
			}

			u := tgbotapi.NewUpdate(0)
			u.Timeout = 30
			updates := b.api.GetUpdatesChan(u)
			firstUpdate = true

			for {
				select {
				case <-pollCtx.Done():
					log.Println("[telegram] polling stopped")
					return
				case update, ok := <-updates:
					if !ok {
						// Stop old goroutine before reconnecting
						b.api.StopReceivingUpdates()
						log.Printf("[telegram] update channel closed, retrying in %v", backoff)
						select {
						case <-time.After(backoff):
						case <-pollCtx.Done():
							return
						}
						if backoff < maxBackoff {
							backoff *= 2
						}
						goto reconnect
					}
					// Reset backoff on first successful update
					if firstUpdate {
						backoff = time.Second
						firstUpdate = false
					}
					// Serialize per chat to prevent history races
					go func(upd tgbotapi.Update) {
						chatID := int64(0)
						if upd.Message != nil {
							chatID = upd.Message.Chat.ID
						} else if upd.CallbackQuery != nil && upd.CallbackQuery.Message != nil {
							chatID = upd.CallbackQuery.Message.Chat.ID
						}
						if chatID != 0 {
							mu := getChatMu(chatID)
							mu.Lock()
							defer mu.Unlock()
						}
						b.handler.HandleUpdate(pollCtx, upd)
					}(update)
				}
			}
		reconnect:
		}
	}()

	log.Println("[telegram] long-polling started")
}

func (b *Bot) API() *tgbotapi.BotAPI {
	return b.api
}

// SetScheduleEngine sets the schedule engine for the scheduler agent.
func (b *Bot) SetScheduleEngine(engine *scheduler.Engine) {
	b.handler.ScheduleEngine = engine
}

func (b *Bot) Stop() {
	if b.cancel != nil {
		b.cancel()
	}
	b.api.StopReceivingUpdates()
	log.Println("[telegram] bot stopped")
}

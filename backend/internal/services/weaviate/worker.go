package weaviate

import (
	"context"
	"log"
	"time"

	"gorm.io/gorm"

	"github.com/cds-id/pdt/backend/internal/models"
)

type EmbedRequest struct {
	MessageID  uint
	ListenerID uint
	UserID     uint
	Content    string
	SenderName string
	Timestamp  time.Time
}

type EmbeddingWorker struct {
	client     *Client
	db         *gorm.DB
	queue      chan EmbedRequest
	maxRetries int
}

func NewEmbeddingWorker(client *Client, db *gorm.DB) *EmbeddingWorker {
	return &EmbeddingWorker{
		client:     client,
		db:         db,
		queue:      make(chan EmbedRequest, 1000),
		maxRetries: 3,
	}
}

func (w *EmbeddingWorker) Enqueue(req EmbedRequest) {
	select {
	case w.queue <- req:
	default:
		log.Printf("[embedding-worker] queue full, dropping message %d", req.MessageID)
	}
}

func (w *EmbeddingWorker) Start(ctx context.Context) {
	go w.run(ctx)
}

func (w *EmbeddingWorker) run(ctx context.Context) {
	batch := make([]EmbedRequest, 0, 50)
	ticker := time.NewTicker(5 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			if len(batch) > 0 {
				w.processBatch(batch)
			}
			return
		case req := <-w.queue:
			batch = append(batch, req)
			if len(batch) >= 50 {
				w.processBatch(batch)
				batch = batch[:0]
			}
		case <-ticker.C:
			if len(batch) > 0 {
				w.processBatch(batch)
				batch = batch[:0]
			}
		}
	}
}

func (w *EmbeddingWorker) processBatch(batch []EmbedRequest) {
	for _, req := range batch {
		var lastErr error
		for attempt := 0; attempt < w.maxRetries; attempt++ {
			err := w.client.Upsert(
				context.Background(),
				int(req.MessageID),
				int(req.ListenerID),
				int(req.UserID),
				req.Content,
				req.SenderName,
				req.Timestamp,
			)
			if err == nil {
				break
			}
			lastErr = err
			time.Sleep(time.Duration(attempt+1) * time.Second)
		}
		if lastErr != nil {
			log.Printf("[embedding-worker] failed to embed message %d after %d retries: %v", req.MessageID, w.maxRetries, lastErr)
		}
	}
}

// Backfill re-embeds all messages from MySQL into Weaviate.
func (w *EmbeddingWorker) Backfill(ctx context.Context, userID uint) {
	var messages []models.WaMessage
	w.db.Joins("JOIN wa_listeners ON wa_listeners.id = wa_messages.wa_listener_id").
		Joins("JOIN wa_numbers ON wa_numbers.id = wa_listeners.wa_number_id").
		Where("wa_numbers.user_id = ?", userID).
		Where("wa_messages.content != ''").
		Find(&messages)

	log.Printf("[embedding-worker] backfill: %d messages for user %d", len(messages), userID)

	for _, msg := range messages {
		select {
		case <-ctx.Done():
			return
		default:
		}
		w.Enqueue(EmbedRequest{
			MessageID:  msg.ID,
			ListenerID: msg.WaListenerID,
			UserID:     userID,
			Content:    msg.Content,
			SenderName: msg.SenderName,
			Timestamp:  msg.Timestamp,
		})
		time.Sleep(100 * time.Millisecond) // Rate limit for backfill
	}
}

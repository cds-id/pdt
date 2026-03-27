package whatsapp

import (
	"context"
	"log"
	"time"

	"gorm.io/gorm"

	"github.com/cds-id/pdt/backend/internal/models"
)

// SenderWorker polls wa_outbox for approved messages and sends them.
type SenderWorker struct {
	DB      *gorm.DB
	Manager *Manager
}

func NewSenderWorker(db *gorm.DB, manager *Manager) *SenderWorker {
	return &SenderWorker{DB: db, Manager: manager}
}

// Start launches the sender goroutine.
func (w *SenderWorker) Start(ctx context.Context) {
	go w.run(ctx)
}

func (w *SenderWorker) run(ctx context.Context) {
	ticker := time.NewTicker(5 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			w.processApproved(ctx)
		}
	}
}

func (w *SenderWorker) processApproved(ctx context.Context) {
	var outboxItems []models.WaOutbox
	result := w.DB.Where("status = ?", "approved").Find(&outboxItems)
	if result.Error != nil {
		log.Printf("[wa-sender] query outbox error: %v", result.Error)
		return
	}

	if len(outboxItems) > 0 {
		log.Printf("[wa-sender] found %d approved messages to send", len(outboxItems))
	}

	for _, item := range outboxItems {
		log.Printf("[wa-sender] sending outbox %d: number=%d target=%s content=%q", item.ID, item.WaNumberID, item.TargetJID, item.Content[:min(len(item.Content), 50)])
		waMsgID, err := w.Manager.SendMessage(ctx, item.WaNumberID, item.TargetJID, item.Content)
		if err != nil {
			log.Printf("[wa-sender] send failed for outbox %d: %v", item.ID, err)
			w.DB.Model(&models.WaOutbox{}).Where("id = ?", item.ID).Update("status", "failed")
			continue
		}

		now := time.Now()
		w.DB.Model(&models.WaOutbox{}).Where("id = ?", item.ID).Updates(map[string]interface{}{
			"status":        "sent",
			"sent_at":       &now,
			"wa_message_id": waMsgID,
		})
		log.Printf("[wa-sender] sent outbox %d to %s", item.ID, item.TargetJID)
	}
}

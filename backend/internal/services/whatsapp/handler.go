package whatsapp

import (
	"context"
	"fmt"
	"log"
	"time"

	"go.mau.fi/whatsmeow/proto/waE2E"
	"go.mau.fi/whatsmeow/types"
	"go.mau.fi/whatsmeow/types/events"
	"gorm.io/gorm"

	"github.com/cds-id/pdt/backend/internal/models"
	"github.com/cds-id/pdt/backend/internal/services/storage"
	wvClient "github.com/cds-id/pdt/backend/internal/services/weaviate"
)

// mediaDownloader is the subset of whatsmeow.Client used for media downloads.
type mediaDownloader interface {
	DownloadAny(ctx context.Context, msg *waE2E.Message) ([]byte, error)
}

// MessageHandler handles incoming WhatsApp events for a specific number.
type MessageHandler struct {
	DB              *gorm.DB
	R2              *storage.R2Client
	EmbeddingWorker *wvClient.EmbeddingWorker
	NumberID        uint
	client          mediaDownloader
}

func NewMessageHandler(db *gorm.DB, r2 *storage.R2Client, ew *wvClient.EmbeddingWorker, numberID uint) *MessageHandler {
	return &MessageHandler{
		DB:              db,
		R2:              r2,
		EmbeddingWorker: ew,
		NumberID:        numberID,
	}
}

// HandleEvent is registered with whatsmeow's AddEventHandler.
func (h *MessageHandler) HandleEvent(evt interface{}) {
	switch v := evt.(type) {
	case *events.Message:
		h.handleMessage(v)
	case *events.Receipt:
		h.handleReceipt(v)
	case *events.Connected:
		log.Printf("[wa-handler] number %d connected", h.NumberID)
		h.DB.Model(&models.WaNumber{}).Where("id = ?", h.NumberID).Update("status", "connected")
	case *events.Disconnected:
		log.Printf("[wa-handler] number %d disconnected", h.NumberID)
		h.DB.Model(&models.WaNumber{}).Where("id = ?", h.NumberID).Update("status", "disconnected")
	}
}

func (h *MessageHandler) handleReceipt(evt *events.Receipt) {
	now := time.Now()
	for _, msgID := range evt.MessageIDs {
		switch evt.Type {
		case events.ReceiptTypeDelivered:
			h.DB.Model(&models.WaOutbox{}).
				Where("wa_message_id = ?", msgID).
				Update("delivered_at", now)
			log.Printf("[wa-handler] message %s delivered", msgID)
		case events.ReceiptTypeRead:
			h.DB.Model(&models.WaOutbox{}).
				Where("wa_message_id = ?", msgID).
				Update("read_at", now)
			log.Printf("[wa-handler] message %s read", msgID)
		}
	}
}

func (h *MessageHandler) handleMessage(evt *events.Message) {
	// Ignore outgoing messages
	if evt.Info.IsFromMe {
		return
	}

	senderJID := evt.Info.Sender.String()
	chatJID := evt.Info.Chat.String()

	// Find an active listener matching either the sender or the chat JID
	var listener models.WaListener
	err := h.DB.Where("wa_number_id = ? AND is_active = ? AND jid IN (?, ?)", h.NumberID, true, senderJID, chatJID).
		First(&listener).Error
	if err != nil {
		// No active listener for this contact/group — ignore
		return
	}

	text := extractText(evt.Message)
	msgType := detectMediaType(evt.Message)
	hasMedia := hasMediaContent(evt.Message)

	msg := models.WaMessage{
		WaListenerID: listener.ID,
		MessageID:    string(evt.Info.ID),
		SenderJID:    senderJID,
		SenderName:   evt.Info.PushName,
		Content:      text,
		MessageType:  msgType,
		HasMedia:     hasMedia,
		Timestamp:    evt.Info.Timestamp,
	}

	if result := h.DB.Create(&msg); result.Error != nil {
		log.Printf("[wa-handler] save message error: %v", result.Error)
		return
	}

	// Enqueue embedding for text messages
	if text != "" {
		h.EmbeddingWorker.Enqueue(wvClient.EmbedRequest{
			MessageID:  msg.ID,
			ListenerID: listener.ID,
			UserID:     0, // populated by the worker via DB join
			Content:    text,
			SenderName: evt.Info.PushName,
			Timestamp:  evt.Info.Timestamp,
		})
	}

	// Async media download
	if hasMedia && h.R2 != nil {
		go h.downloadAndStoreMedia(evt, msg.ID)
	}
}

func (h *MessageHandler) downloadAndStoreMedia(evt *events.Message, msgID uint) {
	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()

	// We need a whatsmeow client to download — obtain via manager via the DB lookup is complex,
	// so we receive it directly when the manager sets it up. For now, we store the client reference
	// as an optional field to be set by the manager after creation.
	if h.client == nil {
		log.Printf("[wa-handler] no client available for media download, message %d", msgID)
		return
	}

	data, err := h.client.DownloadAny(ctx, evt.Message)
	if err != nil {
		log.Printf("[wa-handler] download media error for message %d: %v", msgID, err)
		return
	}

	mimeType := getMimeType(evt.Message)
	fileName := fmt.Sprintf("wa-media/%d-%d", msgID, time.Now().UnixNano())
	url, err := h.R2.Upload(ctx, fileName, data, mimeType)
	if err != nil {
		log.Printf("[wa-handler] r2 upload error for message %d: %v", msgID, err)
		return
	}

	media := models.WaMedia{
		WaMessageID: msgID,
		FileName:    fileName,
		MimeType:    mimeType,
		FileSize:    int64(len(data)),
		R2Key:       fileName,
		FileURL:     url,
	}
	if err := h.DB.Create(&media).Error; err != nil {
		log.Printf("[wa-handler] save media record error: %v", err)
	}
}

// SetClient allows the manager to inject the whatsmeow client for media downloads.
func (h *MessageHandler) SetClient(c mediaDownloader) {
	h.client = c
}

// extractText pulls the text content from a message.
func extractText(msg *waE2E.Message) string {
	if msg == nil {
		return ""
	}
	if msg.GetConversation() != "" {
		return msg.GetConversation()
	}
	if msg.GetExtendedTextMessage() != nil {
		return msg.GetExtendedTextMessage().GetText()
	}
	if msg.GetImageMessage() != nil {
		return msg.GetImageMessage().GetCaption()
	}
	if msg.GetVideoMessage() != nil {
		return msg.GetVideoMessage().GetCaption()
	}
	if msg.GetDocumentMessage() != nil {
		return msg.GetDocumentMessage().GetCaption()
	}
	return ""
}

// hasMediaContent returns true if the message contains media.
func hasMediaContent(msg *waE2E.Message) bool {
	if msg == nil {
		return false
	}
	return msg.GetImageMessage() != nil ||
		msg.GetVideoMessage() != nil ||
		msg.GetAudioMessage() != nil ||
		msg.GetDocumentMessage() != nil ||
		msg.GetStickerMessage() != nil
}

// detectMediaType returns a short media type string.
func detectMediaType(msg *waE2E.Message) string {
	if msg == nil {
		return "text"
	}
	if msg.GetImageMessage() != nil {
		return "image"
	}
	if msg.GetVideoMessage() != nil {
		return "video"
	}
	if msg.GetAudioMessage() != nil {
		return "audio"
	}
	if msg.GetDocumentMessage() != nil {
		return "document"
	}
	if msg.GetStickerMessage() != nil {
		return "sticker"
	}
	return "text"
}

// getMimeType extracts the MIME type from a media message.
func getMimeType(msg *waE2E.Message) string {
	if msg == nil {
		return "application/octet-stream"
	}
	if m := msg.GetImageMessage(); m != nil {
		return m.GetMimetype()
	}
	if m := msg.GetVideoMessage(); m != nil {
		return m.GetMimetype()
	}
	if m := msg.GetAudioMessage(); m != nil {
		return m.GetMimetype()
	}
	if m := msg.GetDocumentMessage(); m != nil {
		return m.GetMimetype()
	}
	return "application/octet-stream"
}

// parseJID parses a JID string using whatsmeow's types package.
func parseJID(jid string) (types.JID, error) {
	parsed, err := types.ParseJID(jid)
	if err != nil {
		return types.JID{}, fmt.Errorf("invalid JID %q: %w", jid, err)
	}
	return parsed, nil
}

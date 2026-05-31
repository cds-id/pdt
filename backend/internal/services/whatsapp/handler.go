package whatsapp

import (
	"context"
	"fmt"
	"log"
	"strings"
	"time"

	"go.mau.fi/whatsmeow"
	"go.mau.fi/whatsmeow/proto/waE2E"
	"go.mau.fi/whatsmeow/types"
	"go.mau.fi/whatsmeow/types/events"
	"gorm.io/gorm"

	"github.com/cds-id/pdt/backend/internal/ai/mistral"
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
	VisionClient    *mistral.VisionClient
	NumberID        uint
	Hub             *Hub
	userID          uint
	client          mediaDownloader
	waClient        *whatsmeow.Client
}

func NewMessageHandler(db *gorm.DB, r2 *storage.R2Client, ew *wvClient.EmbeddingWorker, vc *mistral.VisionClient, numberID uint) *MessageHandler {
	h := &MessageHandler{
		DB:              db,
		R2:              r2,
		EmbeddingWorker: ew,
		VisionClient:    vc,
		NumberID:        numberID,
	}
	// Resolve owning user once for realtime broadcast targeting.
	var num models.WaNumber
	if db != nil {
		if err := db.Select("user_id").First(&num, numberID).Error; err == nil {
			h.userID = num.UserID
		}
	}
	return h
}

// SetHub injects the realtime broadcast hub.
func (h *MessageHandler) SetHub(hub *Hub) { h.Hub = hub }

// broadcast sends an event to the owning user's inbox clients, if a hub is set.
func (h *MessageHandler) broadcast(evtType string, payload interface{}) {
	if h.Hub == nil || h.userID == 0 {
		return
	}
	h.Hub.Broadcast(h.userID, HubEvent{Type: evtType, Payload: payload})
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
		updates := map[string]any{"status": "connected"}
		// Auto-sync device JID if missing
		if h.waClient != nil && h.waClient.Store.ID != nil {
			var num models.WaNumber
			h.DB.Select("device_jid").First(&num, h.NumberID)
			if num.DeviceJID == "" {
				updates["device_jid"] = h.waClient.Store.ID.String()
				log.Printf("[wa-handler] synced device_jid for number %d: %s", h.NumberID, h.waClient.Store.ID.String())
			}
		}
		h.DB.Model(&models.WaNumber{}).Where("id = ?", h.NumberID).Updates(updates)
	case *events.Disconnected:
		log.Printf("[wa-handler] number %d disconnected", h.NumberID)
		h.DB.Model(&models.WaNumber{}).Where("id = ?", h.NumberID).Update("status", "disconnected")
	}
}

func (h *MessageHandler) handleReceipt(evt *events.Receipt) {
	now := time.Now()
	for _, msgID := range evt.MessageIDs {
		var status string
		switch evt.Type {
		case events.ReceiptTypeDelivered:
			status = "delivered"
			h.DB.Model(&models.WaOutbox{}).
				Where("wa_message_id = ?", msgID).
				Update("delivered_at", now)
		case events.ReceiptTypeRead, events.ReceiptTypeReadSelf:
			status = "read"
			h.DB.Model(&models.WaOutbox{}).
				Where("wa_message_id = ?", msgID).
				Update("read_at", now)
		default:
			continue
		}
		// Update our own outgoing inbox message status + notify clients.
		h.DB.Model(&models.WaMessage{}).
			Where("message_id = ? AND from_me = ?", msgID, true).
			Update("status", status)
		h.broadcast("receipt", map[string]interface{}{"message_id": msgID, "status": status})
		log.Printf("[wa-handler] message %s %s", msgID, status)
	}
}

// findOrCreateListener returns the conversation row for a chat JID, creating an
// inbox-sourced listener if none exists.
func (h *MessageHandler) findOrCreateListener(chatJID string, displayName string) (models.WaListener, error) {
	var listener models.WaListener
	err := h.DB.Where("wa_number_id = ? AND jid = ?", h.NumberID, chatJID).First(&listener).Error
	if err == nil {
		return listener, nil
	}
	if err != gorm.ErrRecordNotFound {
		return listener, err
	}

	lisType := "personal"
	if strings.HasSuffix(chatJID, "@g.us") {
		lisType = "group"
	}
	name := displayName
	if name == "" {
		name = chatJIDDisplay(chatJID)
	}
	listener = models.WaListener{
		WaNumberID: h.NumberID,
		JID:        chatJID,
		Name:       name,
		Type:       lisType,
		IsActive:   true,
		Source:     "inbox",
	}
	if err := h.DB.Create(&listener).Error; err != nil {
		// Race: another goroutine created it — re-fetch.
		if e := h.DB.Where("wa_number_id = ? AND jid = ?", h.NumberID, chatJID).First(&listener).Error; e == nil {
			return listener, nil
		}
		return listener, err
	}
	return listener, nil
}

// chatJIDDisplay derives a human label from a JID (the user part).
func chatJIDDisplay(jid string) string {
	if i := strings.IndexByte(jid, '@'); i > 0 {
		return jid[:i]
	}
	return jid
}

func (h *MessageHandler) handleMessage(evt *events.Message) {
	senderJID := evt.Info.Sender.String()
	chatJID := evt.Info.Chat.String()
	fromMe := evt.Info.IsFromMe

	// Inbox is chat-scoped: the conversation key is always the chat JID.
	// Auto-create a listener (conversation) on first sight.
	listener, err := h.findOrCreateListener(chatJID, evt.Info.PushName)
	if err != nil {
		log.Printf("[wa-handler] find/create listener for %s error: %v", chatJID, err)
		return
	}

	rawText := extractText(evt.Message)
	msgType := detectMediaType(evt.Message)
	hasMedia := hasMediaContent(evt.Message)

	// Label media messages that have no text/caption
	text := rawText
	if text == "" && hasMedia {
		text = "[" + msgType + "]"
	}

	senderName := evt.Info.PushName
	if fromMe {
		senderName = "You"
	}

	msg := models.WaMessage{
		WaListenerID: listener.ID,
		MessageID:    string(evt.Info.ID),
		SenderJID:    senderJID,
		SenderName:   senderName,
		Content:      text,
		MessageType:  msgType,
		HasMedia:     hasMedia,
		FromMe:       fromMe,
		ChatJID:      chatJID,
		Timestamp:    evt.Info.Timestamp,
	}

	if result := h.DB.Create(&msg); result.Error != nil {
		if strings.Contains(result.Error.Error(), "Duplicate entry") {
			// WhatsApp delivered the same message twice — ignore
			return
		}
		log.Printf("[wa-handler] save message error: %v", result.Error)
		return
	}

	// Update conversation metadata (preview, time, unread for inbound).
	h.touchListener(listener.ID, text, evt.Info.Timestamp, fromMe)

	// Enqueue embedding only for messages with actual text content.
	if rawText != "" && h.EmbeddingWorker != nil {
		h.EmbeddingWorker.Enqueue(wvClient.EmbedRequest{
			MessageID:  msg.ID,
			ListenerID: listener.ID,
			UserID:     0, // populated by the worker via DB join
			Content:    text,
			SenderName: senderName,
			Timestamp:  evt.Info.Timestamp,
		})
	}

	// Realtime push of the new message to inbox clients.
	h.broadcast("message", msg)

	// Async media download
	if hasMedia && h.R2 != nil {
		go h.downloadAndStoreMedia(evt, msg.ID)
	}
}

// touchListener refreshes conversation ordering metadata and unread count.
func (h *MessageHandler) touchListener(listenerID uint, preview string, ts time.Time, fromMe bool) {
	if len(preview) > 300 {
		preview = preview[:300]
	}
	updates := map[string]interface{}{
		"last_message_at":      ts,
		"last_message_preview": preview,
	}
	if !fromMe {
		updates["unread_count"] = gorm.Expr("unread_count + 1")
	}
	if err := h.DB.Model(&models.WaListener{}).Where("id = ?", listenerID).Updates(updates).Error; err != nil {
		log.Printf("[wa-handler] touch listener %d error: %v", listenerID, err)
		return
	}
	var listener models.WaListener
	if err := h.DB.First(&listener, listenerID).Error; err == nil {
		h.broadcast("chat_update", listener)
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
		return
	}

	// Notify clients the media is now available (re-send message with media attached).
	var withMedia models.WaMessage
	if err := h.DB.Preload("Media").First(&withMedia, msgID).Error; err == nil {
		h.broadcast("message", withMedia)
	}

	// Use Mistral vision to describe images and update message content
	if h.VisionClient != nil && strings.HasPrefix(mimeType, "image/") {
		// Check daily rate limit (20/day)
		var todayCount int64
		todayStart := time.Now().Truncate(24 * time.Hour)
		h.DB.Model(&models.AIUsage{}).
			Where("provider = ? AND feature = ? AND created_at >= ?", "mistral", "wa_vision", todayStart).
			Count(&todayCount)
		if todayCount >= 20 {
			log.Printf("[wa-handler] vision rate limit reached (%d/20 today), skipping message %d", todayCount, msgID)
			return
		}

		h.describeImageAndUpdate(ctx, url, mimeType, msgID)
	}
}

// describeImageAndUpdate calls Mistral vision API to describe an image
// and updates the WaMessage content with the description.
func (h *MessageHandler) describeImageAndUpdate(ctx context.Context, imageURL, mimeType string, msgID uint) {
	result, err := h.VisionClient.DescribeImage(ctx, imageURL, mimeType)
	if err != nil {
		log.Printf("[wa-handler] vision describe error for message %d: %v", msgID, err)
		return
	}

	// Resolve user ID from number for usage tracking
	var waNumber models.WaNumber
	if err := h.DB.Select("user_id").First(&waNumber, h.NumberID).Error; err == nil {
		h.DB.Create(&models.AIUsage{
			UserID:           waNumber.UserID,
			Provider:         "mistral",
			Model:            mistral.VisionModel,
			Feature:          "wa_vision",
			PromptTokens:     result.PromptTokens,
			CompletionTokens: result.CompletionTokens,
		})
	}

	// Read existing content to preserve any caption
	var msg models.WaMessage
	if err := h.DB.Select("content").First(&msg, msgID).Error; err != nil {
		log.Printf("[wa-handler] read message %d error: %v", msgID, err)
		return
	}

	// Build content: keep caption if present, append vision description
	description := result.Description
	var content string
	if msg.Content != "" && msg.Content != "[image]" && msg.Content != "[sticker]" {
		content = msg.Content + "\n\n[Image description: " + description + "]"
	} else {
		content = "[Image description: " + description + "]"
	}

	if err := h.DB.Model(&models.WaMessage{}).Where("id = ?", msgID).Update("content", content).Error; err != nil {
		log.Printf("[wa-handler] update message content error for %d: %v", msgID, err)
		return
	}

	// Enqueue embedding now that we have meaningful content
	if h.EmbeddingWorker != nil {
		var fullMsg models.WaMessage
		if err := h.DB.First(&fullMsg, msgID).Error; err == nil {
			h.EmbeddingWorker.Enqueue(wvClient.EmbedRequest{
				MessageID:  fullMsg.ID,
				ListenerID: fullMsg.WaListenerID,
				UserID:     0,
				Content:    content,
				SenderName: fullMsg.SenderName,
				Timestamp:  fullMsg.Timestamp,
			})
		}
	}

	log.Printf("[wa-handler] vision described message %d: %s", msgID, truncate(description, 80))
}

func truncate(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	return s[:maxLen] + "..."
}

// SetClient allows the manager to inject the whatsmeow client for media downloads.
func (h *MessageHandler) SetClient(c mediaDownloader) {
	h.client = c
	if wc, ok := c.(*whatsmeow.Client); ok {
		h.waClient = wc
	}
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

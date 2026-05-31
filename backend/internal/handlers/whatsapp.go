package handlers

import (
	"context"
	"fmt"
	"io"
	"log"
	"net/http"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/gorilla/websocket"
	"go.mau.fi/whatsmeow"
	waLog "go.mau.fi/whatsmeow/util/log"
	"gorm.io/gorm"

	"github.com/cds-id/pdt/backend/internal/models"
	"github.com/cds-id/pdt/backend/internal/services/storage"
	waService "github.com/cds-id/pdt/backend/internal/services/whatsapp"
)

// WhatsAppHandler handles WhatsApp REST and WebSocket endpoints.
type WhatsAppHandler struct {
	DB      *gorm.DB
	Manager *waService.Manager
	Hub     *waService.Hub
	R2      *storage.R2Client
}

// ─── Request structs ──────────────────────────────────────────────────────────

type addNumberRequest struct {
	PhoneNumber string `json:"phone_number" binding:"required"`
	DisplayName string `json:"display_name"`
}

type updateNumberRequest struct {
	DisplayName string `json:"display_name" binding:"required"`
}

type addListenerRequest struct {
	JID  string `json:"jid"  binding:"required"`
	Name string `json:"name" binding:"required"`
	Type string `json:"type" binding:"required"`
}

type updateListenerRequest struct {
	Name     *string `json:"name"`
	IsActive *bool   `json:"is_active"`
}

type updateOutboxRequest struct {
	Status  string `json:"status"`  // approved | rejected
	Content string `json:"content"` // optional edited content
}

// ─── Numbers ──────────────────────────────────────────────────────────────────

// ListNumbers GET /wa/numbers
func (h *WhatsAppHandler) ListNumbers(c *gin.Context) {
	userID := c.GetUint("user_id")

	var numbers []models.WaNumber
	h.DB.Where("user_id = ?", userID).Order("created_at desc").Find(&numbers)

	c.JSON(http.StatusOK, numbers)
}

// AddNumber POST /wa/numbers
func (h *WhatsAppHandler) AddNumber(c *gin.Context) {
	userID := c.GetUint("user_id")

	var req addNumberRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	number := models.WaNumber{
		UserID:      userID,
		PhoneNumber: req.PhoneNumber,
		DisplayName: req.DisplayName,
		Status:      "pairing",
	}

	if err := h.DB.Create(&number).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to create number"})
		return
	}

	c.JSON(http.StatusCreated, number)
}

// UpdateNumber PATCH /wa/numbers/:id
func (h *WhatsAppHandler) UpdateNumber(c *gin.Context) {
	userID := c.GetUint("user_id")
	id := c.Param("id")

	var number models.WaNumber
	if err := h.DB.Where("id = ? AND user_id = ?", id, userID).First(&number).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "number not found"})
		return
	}

	var req updateNumberRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	if err := h.DB.Model(&number).Update("display_name", req.DisplayName).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to update number"})
		return
	}

	c.JSON(http.StatusOK, number)
}

// DeleteNumber DELETE /wa/numbers/:id
func (h *WhatsAppHandler) DeleteNumber(c *gin.Context) {
	userID := c.GetUint("user_id")
	id := c.Param("id")

	var number models.WaNumber
	if err := h.DB.Where("id = ? AND user_id = ?", id, userID).First(&number).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "number not found"})
		return
	}

	// Disconnect client if running
	if h.Manager != nil {
		h.Manager.RemoveClient(number.ID)
	}

	if err := h.DB.Delete(&number).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to delete number"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "number deleted"})
}

// DisconnectNumber POST /wa/numbers/:id/disconnect
func (h *WhatsAppHandler) DisconnectNumber(c *gin.Context) {
	userID := c.GetUint("user_id")
	id := c.Param("id")

	var number models.WaNumber
	if err := h.DB.Where("id = ? AND user_id = ?", id, userID).First(&number).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "number not found"})
		return
	}

	if h.Manager != nil {
		h.Manager.RemoveClient(number.ID)
	}

	h.DB.Model(&number).Update("status", "disconnected")

	c.JSON(http.StatusOK, gin.H{"message": "number disconnected"})
}

// GetGroups GET /wa/numbers/:id/groups
func (h *WhatsAppHandler) GetGroups(c *gin.Context) {
	userID := c.GetUint("user_id")
	id := c.Param("id")

	var number models.WaNumber
	if err := h.DB.Where("id = ? AND user_id = ?", id, userID).First(&number).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "number not found"})
		return
	}

	if h.Manager == nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{"error": "WhatsApp manager not available"})
		return
	}

	groups, err := h.Manager.GetGroups(c.Request.Context(), number.ID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, groups)
}

// GetContacts GET /wa/numbers/:id/contacts
func (h *WhatsAppHandler) GetContacts(c *gin.Context) {
	userID := c.GetUint("user_id")
	id := c.Param("id")

	var number models.WaNumber
	if err := h.DB.Where("id = ? AND user_id = ?", id, userID).First(&number).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "number not found"})
		return
	}

	if h.Manager == nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{"error": "WhatsApp manager not available"})
		return
	}

	contacts, err := h.Manager.GetContacts(c.Request.Context(), number.ID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, contacts)
}

// ─── QR Pairing ───────────────────────────────────────────────────────────────

// HandlePairing GET /wa/numbers/:id/pair  (WebSocket)
func (h *WhatsAppHandler) HandlePairing(c *gin.Context) {
	userID := c.GetUint("user_id")
	id := c.Param("id")

	// Verify ownership
	var number models.WaNumber
	if err := h.DB.Where("id = ? AND user_id = ?", id, userID).First(&number).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "number not found"})
		return
	}

	conn, err := wsUpgrader.Upgrade(c.Writer, c.Request, nil)
	if err != nil {
		log.Printf("[wa-pair] upgrade error: %v", err)
		return
	}
	defer conn.Close()

	if h.Manager == nil {
		conn.WriteJSON(gin.H{"event": "error", "message": "WhatsApp manager not initialized"})
		return
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
	defer cancel()

	// Create a fresh device
	device := h.Manager.GetContainer().NewDevice()
	client := whatsmeow.NewClient(device, waLog.Noop)

	qrChan, err := client.GetQRChannel(ctx)
	if err != nil {
		log.Printf("[wa-pair] get QR channel error: %v", err)
		conn.WriteJSON(gin.H{"event": "error", "message": err.Error()})
		return
	}

	if err := client.Connect(); err != nil {
		log.Printf("[wa-pair] connect error: %v", err)
		conn.WriteJSON(gin.H{"event": "error", "message": err.Error()})
		return
	}

	// Read from WebSocket in background to detect client disconnect
	done := make(chan struct{})
	go func() {
		defer close(done)
		for {
			if _, _, err := conn.ReadMessage(); err != nil {
				return
			}
		}
	}()

	for item := range qrChan {
		select {
		case <-done:
			log.Printf("[wa-pair] client disconnected during pairing")
			client.Disconnect()
			return
		default:
		}

		switch item.Event {
		case whatsmeow.QRChannelEventCode:
			if err := conn.WriteJSON(gin.H{"event": "code", "code": item.Code}); err != nil {
				log.Printf("[wa-pair] write QR error: %v", err)
				client.Disconnect()
				return
			}

		case "success":
			now := time.Now()
			deviceJID := ""
			if client.Store.ID != nil {
				deviceJID = client.Store.ID.String()
			}
			h.DB.Model(&models.WaNumber{}).Where("id = ?", number.ID).Updates(map[string]interface{}{
				"status":     "connected",
				"paired_at":  &now,
				"device_jid": deviceJID,
			})

			// Wire up message handler for this number
			handler := waService.NewMessageHandler(h.DB, h.R2, nil, nil, number.ID)
			handler.SetHub(h.Hub)
			handler.SetClient(client)
			client.AddEventHandler(handler.HandleEvent)

			h.Manager.RegisterClient(number.ID, client)
			conn.WriteJSON(gin.H{"event": "success"})
			return

		case "timeout":
			conn.WriteJSON(gin.H{"event": "timeout"})
			client.Disconnect()
			return

		case whatsmeow.QRChannelEventError:
			conn.WriteJSON(gin.H{"event": "error", "message": "QR pairing failed"})
			client.Disconnect()
			return

		default:
			conn.WriteJSON(gin.H{"event": item.Event})
		}
	}
}

// ─── Listeners ────────────────────────────────────────────────────────────────

type listenerWithCount struct {
	models.WaListener
	MessageCount int64 `json:"message_count"`
}

// ListListeners GET /wa/numbers/:id/listeners
func (h *WhatsAppHandler) ListListeners(c *gin.Context) {
	userID := c.GetUint("user_id")
	numberID := c.Param("id")

	// Verify ownership
	var number models.WaNumber
	if err := h.DB.Where("id = ? AND user_id = ?", numberID, userID).First(&number).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "number not found"})
		return
	}

	var listeners []models.WaListener
	h.DB.Where("wa_number_id = ?", number.ID).Order("created_at asc").Find(&listeners)

	// Attach message counts
	result := make([]listenerWithCount, 0, len(listeners))
	for _, l := range listeners {
		var count int64
		h.DB.Model(&models.WaMessage{}).Where("wa_listener_id = ?", l.ID).Count(&count)
		result = append(result, listenerWithCount{WaListener: l, MessageCount: count})
	}

	c.JSON(http.StatusOK, result)
}

// AddListener POST /wa/numbers/:id/listeners
func (h *WhatsAppHandler) AddListener(c *gin.Context) {
	userID := c.GetUint("user_id")
	numberID := c.Param("id")

	// Verify ownership
	var number models.WaNumber
	if err := h.DB.Where("id = ? AND user_id = ?", numberID, userID).First(&number).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "number not found"})
		return
	}

	var req addListenerRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	listener := models.WaListener{
		WaNumberID: number.ID,
		JID:        req.JID,
		Name:       req.Name,
		Type:       req.Type,
		IsActive:   true,
	}

	if err := h.DB.Create(&listener).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to create listener"})
		return
	}

	c.JSON(http.StatusCreated, listener)
}

// UpdateListener PATCH /wa/listeners/:id
func (h *WhatsAppHandler) UpdateListener(c *gin.Context) {
	userID := c.GetUint("user_id")
	id := c.Param("id")

	// Verify ownership via JOIN
	var listener models.WaListener
	if err := h.DB.Joins("JOIN wa_numbers ON wa_numbers.id = wa_listeners.wa_number_id").
		Where("wa_listeners.id = ? AND wa_numbers.user_id = ?", id, userID).
		First(&listener).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "listener not found"})
		return
	}

	var req updateListenerRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	updates := map[string]interface{}{}
	if req.Name != nil {
		updates["name"] = *req.Name
	}
	if req.IsActive != nil {
		updates["is_active"] = *req.IsActive
	}

	if len(updates) > 0 {
		if err := h.DB.Model(&listener).Updates(updates).Error; err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to update listener"})
			return
		}
	}

	c.JSON(http.StatusOK, listener)
}

// DeleteListener DELETE /wa/listeners/:id
func (h *WhatsAppHandler) DeleteListener(c *gin.Context) {
	userID := c.GetUint("user_id")
	id := c.Param("id")

	// Verify ownership via JOIN
	var listener models.WaListener
	if err := h.DB.Joins("JOIN wa_numbers ON wa_numbers.id = wa_listeners.wa_number_id").
		Where("wa_listeners.id = ? AND wa_numbers.user_id = ?", id, userID).
		First(&listener).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "listener not found"})
		return
	}

	if err := h.DB.Delete(&listener).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to delete listener"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "listener deleted"})
}

// ─── Messages ─────────────────────────────────────────────────────────────────

// ListMessages GET /wa/listeners/:id/messages
func (h *WhatsAppHandler) ListMessages(c *gin.Context) {
	userID := c.GetUint("user_id")
	id := c.Param("id")

	// Verify ownership via JOIN
	var listener models.WaListener
	if err := h.DB.Joins("JOIN wa_numbers ON wa_numbers.id = wa_listeners.wa_number_id").
		Where("wa_listeners.id = ? AND wa_numbers.user_id = ?", id, userID).
		First(&listener).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "listener not found"})
		return
	}

	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	pageSize, _ := strconv.Atoi(c.DefaultQuery("page_size", "50"))
	if page < 1 {
		page = 1
	}
	if pageSize < 1 || pageSize > 200 {
		pageSize = 50
	}
	offset := (page - 1) * pageSize

	var messages []models.WaMessage
	var total int64

	h.DB.Model(&models.WaMessage{}).Where("wa_listener_id = ?", listener.ID).Count(&total)
	h.DB.Where("wa_listener_id = ?", listener.ID).
		Order("timestamp desc").
		Limit(pageSize).
		Offset(offset).
		Find(&messages)

	c.JSON(http.StatusOK, gin.H{
		"data":      messages,
		"total":     total,
		"page":      page,
		"page_size": pageSize,
	})
}

// SearchMessages GET /wa/messages/search?q=keyword
func (h *WhatsAppHandler) SearchMessages(c *gin.Context) {
	userID := c.GetUint("user_id")
	keyword := c.Query("q")

	if keyword == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "query parameter 'q' is required"})
		return
	}

	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	pageSize, _ := strconv.Atoi(c.DefaultQuery("page_size", "50"))
	if page < 1 {
		page = 1
	}
	if pageSize < 1 || pageSize > 200 {
		pageSize = 50
	}
	offset := (page - 1) * pageSize

	var messages []models.WaMessage
	var total int64

	baseQuery := h.DB.Model(&models.WaMessage{}).
		Joins("JOIN wa_listeners ON wa_listeners.id = wa_messages.wa_listener_id").
		Joins("JOIN wa_numbers ON wa_numbers.id = wa_listeners.wa_number_id").
		Where("wa_numbers.user_id = ? AND wa_messages.content LIKE ?", userID, "%"+keyword+"%")

	baseQuery.Count(&total)
	baseQuery.Order("wa_messages.timestamp desc").
		Limit(pageSize).
		Offset(offset).
		Find(&messages)

	c.JSON(http.StatusOK, gin.H{
		"data":      messages,
		"total":     total,
		"page":      page,
		"page_size": pageSize,
	})
}

// ─── Outbox ───────────────────────────────────────────────────────────────────

// ListOutbox GET /wa/outbox
func (h *WhatsAppHandler) ListOutbox(c *gin.Context) {
	userID := c.GetUint("user_id")
	status := c.Query("status")

	query := h.DB.
		Joins("JOIN wa_numbers ON wa_numbers.id = wa_outboxes.wa_number_id").
		Where("wa_numbers.user_id = ?", userID)

	if status != "" {
		query = query.Where("wa_outboxes.status = ?", status)
	}

	var outbox []models.WaOutbox
	query.Order("wa_outboxes.created_at desc").Find(&outbox)

	c.JSON(http.StatusOK, outbox)
}

// UpdateOutbox PATCH /wa/outbox/:id
func (h *WhatsAppHandler) UpdateOutbox(c *gin.Context) {
	userID := c.GetUint("user_id")
	id := c.Param("id")

	// Verify ownership via JOIN
	var item models.WaOutbox
	if err := h.DB.Joins("JOIN wa_numbers ON wa_numbers.id = wa_outboxes.wa_number_id").
		Where("wa_outboxes.id = ? AND wa_numbers.user_id = ?", id, userID).
		First(&item).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "outbox item not found"})
		return
	}

	var req updateOutboxRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	updates := map[string]interface{}{}
	if req.Status != "" {
		updates["status"] = req.Status
		if req.Status == "approved" {
			now := time.Now()
			updates["approved_at"] = &now
		}
	}
	if req.Content != "" {
		updates["content"] = req.Content
	}

	if len(updates) > 0 {
		if err := h.DB.Model(&item).Updates(updates).Error; err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to update outbox item"})
			return
		}
	}

	c.JSON(http.StatusOK, item)
}

// DeleteOutbox DELETE /wa/outbox/:id  (only pending)
func (h *WhatsAppHandler) DeleteOutbox(c *gin.Context) {
	userID := c.GetUint("user_id")
	id := c.Param("id")

	// Verify ownership via JOIN
	var item models.WaOutbox
	if err := h.DB.Joins("JOIN wa_numbers ON wa_numbers.id = wa_outboxes.wa_number_id").
		Where("wa_outboxes.id = ? AND wa_numbers.user_id = ?", id, userID).
		First(&item).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "outbox item not found"})
		return
	}

	if item.Status != "pending" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "only pending outbox items can be deleted"})
		return
	}

	if err := h.DB.Delete(&item).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to delete outbox item"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "outbox item deleted"})
}

// ─── Inbox (WhatsApp Web–style) ─────────────────────────────────────────────

type chatItem struct {
	models.WaListener
	Status string `json:"number_status"`
}

// ListChats GET /wa/numbers/:id/chats
// Returns conversations (listeners) for a number, newest activity first.
func (h *WhatsAppHandler) ListChats(c *gin.Context) {
	userID := c.GetUint("user_id")
	id := c.Param("id")

	var number models.WaNumber
	if err := h.DB.Where("id = ? AND user_id = ?", id, userID).First(&number).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "number not found"})
		return
	}

	var listeners []models.WaListener
	h.DB.Where("wa_number_id = ?", number.ID).
		Order("last_message_at desc, created_at desc").
		Find(&listeners)

	c.JSON(http.StatusOK, listeners)
}

// GetChatMessages GET /wa/chats/:listenerId/messages
// Paginated message history for a conversation, ascending by time (oldest→newest).
func (h *WhatsAppHandler) GetChatMessages(c *gin.Context) {
	userID := c.GetUint("user_id")
	id := c.Param("listenerId")

	var listener models.WaListener
	if err := h.DB.Joins("JOIN wa_numbers ON wa_numbers.id = wa_listeners.wa_number_id").
		Where("wa_listeners.id = ? AND wa_numbers.user_id = ?", id, userID).
		First(&listener).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "chat not found"})
		return
	}

	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	pageSize, _ := strconv.Atoi(c.DefaultQuery("page_size", "50"))
	if page < 1 {
		page = 1
	}
	if pageSize < 1 || pageSize > 200 {
		pageSize = 50
	}
	offset := (page - 1) * pageSize

	var total int64
	h.DB.Model(&models.WaMessage{}).Where("wa_listener_id = ?", listener.ID).Count(&total)

	// Fetch newest page first (desc) then reverse to ascending for display.
	var rows []models.WaMessage
	h.DB.Preload("Media").
		Where("wa_listener_id = ?", listener.ID).
		Order("timestamp desc").
		Limit(pageSize).Offset(offset).
		Find(&rows)

	// reverse in place → ascending
	for i, j := 0, len(rows)-1; i < j; i, j = i+1, j-1 {
		rows[i], rows[j] = rows[j], rows[i]
	}

	c.JSON(http.StatusOK, gin.H{
		"data":      rows,
		"total":     total,
		"page":      page,
		"page_size": pageSize,
	})
}

// MarkChatRead POST /wa/chats/:listenerId/read
func (h *WhatsAppHandler) MarkChatRead(c *gin.Context) {
	userID := c.GetUint("user_id")
	id := c.Param("listenerId")

	var listener models.WaListener
	if err := h.DB.Joins("JOIN wa_numbers ON wa_numbers.id = wa_listeners.wa_number_id").
		Where("wa_listeners.id = ? AND wa_numbers.user_id = ?", id, userID).
		First(&listener).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "chat not found"})
		return
	}

	h.DB.Model(&models.WaListener{}).Where("id = ?", listener.ID).Update("unread_count", 0)
	c.JSON(http.StatusOK, gin.H{"message": "marked read"})
}

type sendInboxRequest struct {
	ChatJID   string `json:"chat_jid" binding:"required"`
	Text      string `json:"text"`
	MediaURL  string `json:"media_url"`
	MediaType string `json:"media_type"`
}

// SendInbox POST /wa/numbers/:id/send
// Live send (bypasses outbox approval), persists + broadcasts the message.
func (h *WhatsAppHandler) SendInbox(c *gin.Context) {
	userID := c.GetUint("user_id")
	id := c.Param("id")

	var number models.WaNumber
	if err := h.DB.Where("id = ? AND user_id = ?", id, userID).First(&number).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "number not found"})
		return
	}

	var req sendInboxRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	if req.Text == "" && req.MediaURL == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "text or media_url required"})
		return
	}
	if h.Manager == nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{"error": "WhatsApp manager not available"})
		return
	}

	msg, err := h.Manager.SendInboxMessage(c.Request.Context(), number.ID, req.ChatJID, req.Text, req.MediaURL, req.MediaType)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, msg)
}

// UploadMedia POST /wa/media/upload  (multipart form field "file")
// Stores an outgoing attachment in R2 and returns its public URL + type.
func (h *WhatsAppHandler) UploadMedia(c *gin.Context) {
	if h.R2 == nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{"error": "media storage not configured"})
		return
	}

	fileHeader, err := c.FormFile("file")
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "file required"})
		return
	}
	f, err := fileHeader.Open()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "open file failed"})
		return
	}
	defer f.Close()

	data, err := io.ReadAll(f)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "read file failed"})
		return
	}

	mimeType := fileHeader.Header.Get("Content-Type")
	if mimeType == "" {
		mimeType = "application/octet-stream"
	}

	userID := c.GetUint("user_id")
	key := fmt.Sprintf("wa-outgoing/%d/%d-%s", userID, time.Now().UnixNano(), fileHeader.Filename)
	url, err := h.R2.Upload(c.Request.Context(), key, data, mimeType)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"url":        url,
		"mime":       mimeType,
		"media_type": mediaCategory(mimeType),
		"file_name":  fileHeader.Filename,
	})
}

func mediaCategory(mime string) string {
	switch {
	case len(mime) >= 6 && mime[:6] == "image/":
		return "image"
	case len(mime) >= 6 && mime[:6] == "video/":
		return "video"
	case len(mime) >= 6 && mime[:6] == "audio/":
		return "audio"
	default:
		return "document"
	}
}

// HandleInboxSocket GET /wa/ws  (WebSocket)
// Registers the connection with the realtime hub for the authenticated user.
func (h *WhatsAppHandler) HandleInboxSocket(c *gin.Context) {
	userID := c.GetUint("user_id")
	if h.Hub == nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{"error": "realtime hub not available"})
		return
	}

	conn, err := wsUpgrader.Upgrade(c.Writer, c.Request, nil)
	if err != nil {
		log.Printf("[wa-ws] upgrade error: %v", err)
		return
	}
	defer conn.Close()

	hc := h.Hub.Register(userID, conn)
	defer h.Hub.Unregister(userID, hc)

	conn.SetReadDeadline(time.Now().Add(60 * time.Second))
	conn.SetPongHandler(func(string) error {
		conn.SetReadDeadline(time.Now().Add(60 * time.Second))
		return nil
	})

	pingTicker := time.NewTicker(30 * time.Second)
	defer pingTicker.Stop()
	done := make(chan struct{})

	go func() {
		defer close(done)
		for {
			if _, _, err := conn.ReadMessage(); err != nil {
				return
			}
		}
	}()

	for {
		select {
		case <-done:
			return
		case <-pingTicker.C:
			h.Hub.LockWrite(hc)
			err := conn.WriteMessage(websocket.PingMessage, nil)
			h.Hub.UnlockWrite(hc)
			if err != nil {
				return
			}
		}
	}
}

package handlers

import (
	"context"
	"encoding/json"
	"log"
	"net/http"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/gorilla/websocket"
	"gorm.io/gorm"

	"github.com/cds-id/pdt/backend/internal/ai/agent"
	"github.com/cds-id/pdt/backend/internal/ai/minimax"
	"github.com/cds-id/pdt/backend/internal/crypto"
	"github.com/cds-id/pdt/backend/internal/models"
	"github.com/cds-id/pdt/backend/internal/services/report"
	"github.com/cds-id/pdt/backend/internal/services/storage"
)

type ChatHandler struct {
	DB              *gorm.DB
	MiniMaxClient   *minimax.Client
	Encryptor       *crypto.Encryptor
	R2              *storage.R2Client
	ReportGenerator *report.Generator
	ContextWindow   int
}

// wsStreamWriter implements agent.StreamWriter for WebSocket connections.
type wsStreamWriter struct {
	conn           *websocket.Conn
	mu             sync.Mutex
	conversationID string
}

func (w *wsStreamWriter) writeJSON(v any) error {
	w.mu.Lock()
	defer w.mu.Unlock()
	return w.conn.WriteJSON(v)
}

func (w *wsStreamWriter) WriteContent(content string) error {
	return w.writeJSON(map[string]string{
		"type":            "stream",
		"content":         content,
		"conversation_id": w.conversationID,
	})
}

func (w *wsStreamWriter) WriteThinking(message string) error {
	return w.writeJSON(map[string]string{
		"type":            "thinking",
		"content":         message,
		"conversation_id": w.conversationID,
	})
}

func (w *wsStreamWriter) WriteToolStatus(toolName, status string) error {
	return w.writeJSON(map[string]string{
		"type":   "tool_status",
		"tool":   toolName,
		"status": status,
	})
}

func (w *wsStreamWriter) WriteDone() error {
	return w.writeJSON(map[string]string{
		"type":            "done",
		"conversation_id": w.conversationID,
	})
}

func (w *wsStreamWriter) WriteError(msg string) error {
	return w.writeJSON(map[string]string{
		"type":    "error",
		"content": msg,
	})
}

var wsUpgrader = websocket.Upgrader{
	CheckOrigin: func(r *http.Request) bool { return true },
}

func (h *ChatHandler) HandleWebSocket(c *gin.Context) {
	userID := c.GetUint("user_id")

	conn, err := wsUpgrader.Upgrade(c.Writer, c.Request, nil)
	if err != nil {
		log.Printf("[ws] upgrade error: %v", err)
		return
	}
	defer conn.Close()

	// Create writer immediately after upgrade so the ping goroutine can use its mutex.
	writer := &wsStreamWriter{conn: conn}

	// Ping/pong keepalive
	conn.SetReadDeadline(time.Now().Add(60 * time.Second))
	conn.SetPongHandler(func(string) error {
		conn.SetReadDeadline(time.Now().Add(60 * time.Second))
		return nil
	})

	pingTicker := time.NewTicker(30 * time.Second)
	defer pingTicker.Stop()

	go func() {
		for range pingTicker.C {
			writer.mu.Lock()
			err := conn.WriteMessage(websocket.PingMessage, nil)
			writer.mu.Unlock()
			if err != nil {
				return
			}
		}
	}()

	// Build orchestrator with user-scoped agents
	orchestrator := agent.NewOrchestrator(
		h.MiniMaxClient,
		&agent.GitAgent{DB: h.DB, UserID: userID, Encryptor: h.Encryptor},
		&agent.JiraAgent{DB: h.DB, UserID: userID},
		&agent.ReportAgent{DB: h.DB, UserID: userID, Generator: h.ReportGenerator, R2: h.R2},
		&agent.ProofAgent{DB: h.DB, UserID: userID},
	)

	for {
		_, raw, err := conn.ReadMessage()
		if err != nil {
			if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway, websocket.CloseAbnormalClosure) {
				log.Printf("[ws] read error: %v", err)
			}
			break
		}

		conn.SetReadDeadline(time.Now().Add(60 * time.Second))

		var msg struct {
			Type           string `json:"type"`
			Content        string `json:"content"`
			ConversationID string `json:"conversation_id"`
		}
		if err := json.Unmarshal(raw, &msg); err != nil {
			log.Printf("[ws] parse error: %v", err)
			continue
		}

		if msg.Type != "message" {
			continue
		}

		// Get or create conversation
		var conv models.Conversation
		if msg.ConversationID != "" {
			if err := h.DB.Where("id = ? AND user_id = ?", msg.ConversationID, userID).First(&conv).Error; err != nil {
				// Not found or wrong user — create new
				conv = models.Conversation{UserID: userID, Title: truncate(msg.Content, 80)}
				h.DB.Create(&conv)
			}
		} else {
			conv = models.Conversation{UserID: userID, Title: truncate(msg.Content, 80)}
			h.DB.Create(&conv)
		}

		writer.conversationID = conv.ID

		// Save user message
		userMsg := models.ChatMessage{
			ConversationID: conv.ID,
			Role:           "user",
			Content:        msg.Content,
		}
		h.DB.Create(&userMsg)

		// Load conversation history (last ContextWindow messages)
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

		// Convert to minimax.Messages
		messages := make([]minimax.Message, 0, len(history))
		for _, m := range history {
			messages = append(messages, minimax.Message{
				Role:    m.Role,
				Content: m.Content,
			})
		}

		// Run orchestrator
		result, err := orchestrator.HandleMessage(context.Background(), messages, writer)
		if err != nil {
			log.Printf("[ws] orchestrator error: %v", err)
			writer.WriteError(err.Error())
			continue
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
				PromptTokens:     result.Usage.PromptTokens,
				CompletionTokens: result.Usage.CompletionTokens,
			}
			h.DB.Create(&usage)
		}

		// Touch conversation updated_at
		h.DB.Model(&conv).Update("updated_at", time.Now())

		writer.WriteDone()
	}
}

func (h *ChatHandler) ListConversations(c *gin.Context) {
	userID := c.GetUint("user_id")

	var conversations []models.Conversation
	h.DB.Where("user_id = ?", userID).Order("updated_at desc").Find(&conversations)

	c.JSON(http.StatusOK, conversations)
}

func (h *ChatHandler) GetConversation(c *gin.Context) {
	userID := c.GetUint("user_id")
	id := c.Param("id")

	var conv models.Conversation
	if err := h.DB.Where("id = ? AND user_id = ?", id, userID).
		Preload("Messages", func(db *gorm.DB) *gorm.DB {
			return db.Order("created_at asc")
		}).
		First(&conv).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "conversation not found"})
		return
	}

	c.JSON(http.StatusOK, conv)
}

func (h *ChatHandler) DeleteConversation(c *gin.Context) {
	userID := c.GetUint("user_id")
	id := c.Param("id")

	// Verify ownership
	var conv models.Conversation
	if err := h.DB.Where("id = ? AND user_id = ?", id, userID).First(&conv).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "conversation not found"})
		return
	}

	// Delete messages first, then conversation
	h.DB.Where("conversation_id = ?", id).Delete(&models.ChatMessage{})
	h.DB.Delete(&conv)

	c.JSON(http.StatusOK, gin.H{"message": "conversation deleted"})
}

// truncate returns the first n runes of s, appending "..." if truncated.
func truncate(s string, n int) string {
	runes := []rune(s)
	if len(runes) <= n {
		return s
	}
	return string(runes[:n]) + "..."
}

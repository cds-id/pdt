package agent

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/cds-id/pdt/backend/internal/ai/minimax"
	"github.com/cds-id/pdt/backend/internal/models"
	wvClient "github.com/cds-id/pdt/backend/internal/services/weaviate"
	"gorm.io/gorm"
)

type WhatsAppAgent struct {
	DB       *gorm.DB
	UserID   uint
	Weaviate *wvClient.Client
}

func (a *WhatsAppAgent) Name() string { return "whatsapp" }

func (a *WhatsAppAgent) SystemPrompt() string {
	today := time.Now().Format("2006-01-02")

	// Fetch user's phone numbers for self-awareness
	var numbers []models.WaNumber
	a.DB.Where("user_id = ?", a.UserID).Find(&numbers)

	phoneList := make([]string, 0, len(numbers))
	for _, n := range numbers {
		label := n.PhoneNumber
		if n.DisplayName != "" {
			label = fmt.Sprintf("%s (%s)", n.PhoneNumber, n.DisplayName)
		}
		phoneList = append(phoneList, label)
	}
	phoneSummary := "none configured"
	if len(phoneList) > 0 {
		phoneSummary = strings.Join(phoneList, ", ")
	}

	return fmt.Sprintf(`You are a WhatsApp assistant for PDT. Today is %s. You understand Indonesian and English.

YOUR PHONE NUMBERS: %s
These are the numbers that belong to the current user. Messages sent FROM these numbers are the user's own messages.

CRITICAL RULES — NEVER BREAK THESE:
1. NEVER fabricate or hallucinate data. Only present information that comes from tool results.
2. If a tool returns empty results, say "tidak ada data" — do NOT invent messages, senders, or dates.
3. NEVER invent message content, sender names, or conversation summaries not in the tool results.
4. When quoting messages, use the EXACT text from tool results. Do not paraphrase or embellish.

SELF-AWARENESS:
- The numbers listed above belong to the user. Messages from those JIDs/numbers are the user's own messages.
- Messages from other numbers/JIDs are from external contacts.
- Clearly distinguish: "Anda mengirim: ..." vs "[Contact] mengirim: ..."

SEND DISCIPLINE:
- NEVER send a message without user approval.
- When you create an outbox entry (send_message or reply_to_message), ALWAYS explain WHY you are proposing to send it.
- Present the proposed message content and context to the user before it is approved and sent.
- The outbox entry is only a proposal — the user must approve it via the outbox UI.

WORKFLOW:
- Use list_listeners to see available chats/groups.
- Use search_messages for keyword search, semantic_search for concept/topic search.
- Use summarize_chat or full_chat_report for conversation overviews.
- Use send_message or reply_to_message ONLY when the user explicitly asks to send something.

FORMAT (respond in user's language):
- For message lists: show sender, timestamp, and truncated content.
- For summaries: group by listener/chat, highlight key topics.
- For send proposals: clearly show the target, content, and your reasoning.

If no data exists for a request, write "Tidak ada data" — do NOT fill it with made-up content.`, today, phoneSummary)
}

func (a *WhatsAppAgent) Tools() []minimax.Tool {
	return []minimax.Tool{
		{
			Name:        "list_listeners",
			Description: "List all WhatsApp listeners (chats/groups) with their message counts. Use this to see what chats are being monitored.",
			InputSchema: json.RawMessage(`{
				"type": "object",
				"properties": {
					"wa_number_id": {"type": "integer", "description": "Filter by a specific WA number ID (optional)"}
				}
			}`),
		},
		{
			Name:        "search_messages",
			Description: "Search WhatsApp messages using keyword (MySQL LIKE) search. Use for finding specific words, phrases, or sender names.",
			InputSchema: json.RawMessage(`{
				"type": "object",
				"properties": {
					"query": {"type": "string", "description": "Keyword to search in message content"},
					"listener_id": {"type": "integer", "description": "Filter by listener/chat ID (optional)"},
					"sender": {"type": "string", "description": "Filter by sender name or JID (optional)"},
					"start_date": {"type": "string", "description": "Start date filter (YYYY-MM-DD, optional)"},
					"end_date": {"type": "string", "description": "End date filter (YYYY-MM-DD, optional)"},
					"limit": {"type": "integer", "description": "Max results (default 20)"}
				},
				"required": ["query"]
			}`),
		},
		{
			Name:        "semantic_search",
			Description: "Search WhatsApp messages using vector/semantic search via Weaviate. Use for finding messages by topic, meaning, or concept. Falls back gracefully if Weaviate is unavailable.",
			InputSchema: json.RawMessage(`{
				"type": "object",
				"properties": {
					"query": {"type": "string", "description": "Semantic query — describe what you are looking for"},
					"listener_id": {"type": "integer", "description": "Filter by listener/chat ID (optional)"},
					"start_date": {"type": "string", "description": "Start date filter (YYYY-MM-DD, optional)"},
					"end_date": {"type": "string", "description": "End date filter (YYYY-MM-DD, optional)"},
					"limit": {"type": "integer", "description": "Max results (default 10)"}
				},
				"required": ["query"]
			}`),
		},
		{
			Name:        "summarize_chat",
			Description: "Fetch messages in a time range and group them by listener/chat. Returns raw data for LLM summarization. Requires start_date and end_date.",
			InputSchema: json.RawMessage(`{
				"type": "object",
				"properties": {
					"listener_id": {"type": "integer", "description": "Specific listener/chat ID to summarize (null = all listeners)"},
					"start_date": {"type": "string", "description": "Start date (YYYY-MM-DD)"},
					"end_date": {"type": "string", "description": "End date (YYYY-MM-DD)"}
				},
				"required": ["start_date", "end_date"]
			}`),
		},
		{
			Name:        "send_message",
			Description: "Create an outbox entry proposing to send a WhatsApp message to a target JID. This does NOT send immediately — it creates a pending entry for user approval. ALWAYS explain the reason in 'context'.",
			InputSchema: json.RawMessage(`{
				"type": "object",
				"properties": {
					"target_jid": {"type": "string", "description": "Target WhatsApp JID (e.g., '628123456789@s.whatsapp.net' or group JID)"},
					"content": {"type": "string", "description": "Message content to send"},
					"context": {"type": "string", "description": "Why you are proposing this message — reason and background"}
				},
				"required": ["target_jid", "content", "context"]
			}`),
		},
		{
			Name:        "reply_to_message",
			Description: "Create an outbox entry proposing to reply to a specific WhatsApp message. This does NOT send immediately — it creates a pending entry for user approval. ALWAYS explain the reason in 'context'.",
			InputSchema: json.RawMessage(`{
				"type": "object",
				"properties": {
					"wa_message_id": {"type": "integer", "description": "The ID of the WaMessage to reply to"},
					"content": {"type": "string", "description": "Reply content"},
					"context": {"type": "string", "description": "Why you are proposing this reply — reason and background"}
				},
				"required": ["wa_message_id", "content", "context"]
			}`),
		},
		{
			Name:        "full_chat_report",
			Description: "Compound tool: combines list_listeners + summarize_chat + semantic_search to produce a comprehensive WhatsApp activity report. Use for broad overviews. Requires start_date and end_date.",
			InputSchema: json.RawMessage(`{
				"type": "object",
				"properties": {
					"start_date": {"type": "string", "description": "Start date (YYYY-MM-DD)"},
					"end_date": {"type": "string", "description": "End date (YYYY-MM-DD)"}
				},
				"required": ["start_date", "end_date"]
			}`),
		},
	}
}

func (a *WhatsAppAgent) ExecuteTool(ctx context.Context, name string, args json.RawMessage) (any, error) {
	switch name {
	case "list_listeners":
		return a.listListeners(args)
	case "search_messages":
		return a.searchMessages(args)
	case "semantic_search":
		return a.semanticSearch(ctx, args)
	case "summarize_chat":
		return a.summarizeChat(args)
	case "send_message":
		return a.sendMessage(args)
	case "reply_to_message":
		return a.replyToMessage(args)
	case "full_chat_report":
		return a.fullChatReport(ctx, args)
	default:
		return nil, fmt.Errorf("unknown tool: %s", name)
	}
}

// listListeners returns all WaListeners for the user with message counts.
func (a *WhatsAppAgent) listListeners(args json.RawMessage) (any, error) {
	var params struct {
		WaNumberID *uint `json:"wa_number_id"`
	}
	json.Unmarshal(args, &params)

	type listenerEntry struct {
		ID           uint   `json:"id"`
		WaNumberID   uint   `json:"wa_number_id"`
		JID          string `json:"jid"`
		Name         string `json:"name"`
		Type         string `json:"type"`
		IsActive     bool   `json:"is_active"`
		MessageCount int64  `json:"message_count"`
	}

	query := a.DB.Model(&models.WaListener{}).
		Joins("JOIN wa_numbers ON wa_numbers.id = wa_listeners.wa_number_id").
		Where("wa_numbers.user_id = ?", a.UserID)

	if params.WaNumberID != nil {
		query = query.Where("wa_listeners.wa_number_id = ?", *params.WaNumberID)
	}

	var listeners []models.WaListener
	query.Find(&listeners)

	results := make([]listenerEntry, 0, len(listeners))
	for _, l := range listeners {
		var count int64
		a.DB.Model(&models.WaMessage{}).Where("wa_listener_id = ?", l.ID).Count(&count)
		results = append(results, listenerEntry{
			ID:           l.ID,
			WaNumberID:   l.WaNumberID,
			JID:          l.JID,
			Name:         l.Name,
			Type:         l.Type,
			IsActive:     l.IsActive,
			MessageCount: count,
		})
	}

	return map[string]any{
		"total_listeners": len(results),
		"listeners":       results,
	}, nil
}

// searchMessages searches messages by keyword (MySQL LIKE) with optional filters.
func (a *WhatsAppAgent) searchMessages(args json.RawMessage) (any, error) {
	var params struct {
		Query      string  `json:"query"`
		ListenerID *uint   `json:"listener_id"`
		Sender     string  `json:"sender"`
		StartDate  string  `json:"start_date"`
		EndDate    string  `json:"end_date"`
		Limit      int     `json:"limit"`
	}
	json.Unmarshal(args, &params)
	if params.Query == "" {
		return nil, fmt.Errorf("query is required")
	}
	if params.Limit == 0 {
		params.Limit = 20
	}

	query := a.DB.Model(&models.WaMessage{}).
		Joins("JOIN wa_listeners ON wa_listeners.id = wa_messages.wa_listener_id").
		Joins("JOIN wa_numbers ON wa_numbers.id = wa_listeners.wa_number_id").
		Where("wa_numbers.user_id = ?", a.UserID).
		Where("wa_messages.content LIKE ?", "%"+params.Query+"%")

	if params.ListenerID != nil {
		query = query.Where("wa_messages.wa_listener_id = ?", *params.ListenerID)
	}
	if params.Sender != "" {
		query = query.Where("(wa_messages.sender_name LIKE ? OR wa_messages.sender_jid LIKE ?)",
			"%"+params.Sender+"%", "%"+params.Sender+"%")
	}
	if params.StartDate != "" {
		if t, err := time.Parse("2006-01-02", params.StartDate); err == nil {
			query = query.Where("wa_messages.timestamp >= ?", t)
		}
	}
	if params.EndDate != "" {
		if t, err := time.Parse("2006-01-02", params.EndDate); err == nil {
			query = query.Where("wa_messages.timestamp <= ?", t.Add(24*time.Hour))
		}
	}

	var messages []models.WaMessage
	query.Order("wa_messages.timestamp DESC").Limit(params.Limit).Find(&messages)

	type msgEntry struct {
		ID           uint   `json:"id"`
		ListenerID   uint   `json:"listener_id"`
		SenderName   string `json:"sender_name"`
		SenderJID    string `json:"sender_jid"`
		Content      string `json:"content"`
		Timestamp    string `json:"timestamp"`
	}
	results := make([]msgEntry, 0, len(messages))
	for _, m := range messages {
		results = append(results, msgEntry{
			ID:         m.ID,
			ListenerID: m.WaListenerID,
			SenderName: m.SenderName,
			SenderJID:  m.SenderJID,
			Content:    truncateStr(m.Content, 300),
			Timestamp:  m.Timestamp.Format("2006-01-02 15:04"),
		})
	}

	return map[string]any{
		"query":   params.Query,
		"count":   len(results),
		"results": results,
	}, nil
}

// semanticSearch performs vector search via Weaviate, falling back gracefully if unavailable.
func (a *WhatsAppAgent) semanticSearch(ctx context.Context, args json.RawMessage) (any, error) {
	var params struct {
		Query      string `json:"query"`
		ListenerID *int   `json:"listener_id"`
		StartDate  string `json:"start_date"`
		EndDate    string `json:"end_date"`
		Limit      int    `json:"limit"`
	}
	json.Unmarshal(args, &params)
	if params.Query == "" {
		return nil, fmt.Errorf("query is required")
	}
	if params.Limit == 0 {
		params.Limit = 10
	}

	if a.Weaviate == nil || !a.Weaviate.IsAvailable() {
		return map[string]any{
			"available": false,
			"message":   "Weaviate semantic search is not available. Use search_messages for keyword search instead.",
			"results":   []any{},
		}, nil
	}

	var startDate, endDate *time.Time
	if params.StartDate != "" {
		if t, err := time.Parse("2006-01-02", params.StartDate); err == nil {
			startDate = &t
		}
	}
	if params.EndDate != "" {
		if t, err := time.Parse("2006-01-02", params.EndDate); err == nil {
			end := t.Add(24 * time.Hour)
			endDate = &end
		}
	}

	results, err := a.Weaviate.Search(ctx, params.Query, int(a.UserID), params.ListenerID, startDate, endDate, params.Limit)
	if err != nil {
		return map[string]any{
			"available": true,
			"error":     err.Error(),
			"results":   []any{},
		}, nil
	}

	type srEntry struct {
		MessageID  float64 `json:"message_id"`
		ListenerID float64 `json:"listener_id"`
		SenderName string  `json:"sender_name"`
		Content    string  `json:"content"`
		Timestamp  string  `json:"timestamp"`
		Distance   float32 `json:"distance"`
	}
	entries := make([]srEntry, 0, len(results))
	for _, r := range results {
		entries = append(entries, srEntry{
			MessageID:  r.MessageID,
			ListenerID: r.ListenerID,
			SenderName: r.SenderName,
			Content:    truncateStr(r.Content, 300),
			Timestamp:  r.Timestamp,
			Distance:   r.Distance,
		})
	}

	return map[string]any{
		"available": true,
		"query":     params.Query,
		"count":     len(entries),
		"results":   entries,
	}, nil
}

// summarizeChat fetches messages in a time range and groups them by listener.
func (a *WhatsAppAgent) summarizeChat(args json.RawMessage) (any, error) {
	var params struct {
		ListenerID *uint  `json:"listener_id"`
		StartDate  string `json:"start_date"`
		EndDate    string `json:"end_date"`
	}
	json.Unmarshal(args, &params)

	if params.StartDate == "" || params.EndDate == "" {
		return nil, fmt.Errorf("start_date and end_date are required")
	}

	startTime, err := time.Parse("2006-01-02", params.StartDate)
	if err != nil {
		return nil, fmt.Errorf("invalid start_date: %w", err)
	}
	endTime, err := time.Parse("2006-01-02", params.EndDate)
	if err != nil {
		return nil, fmt.Errorf("invalid end_date: %w", err)
	}
	endTime = endTime.Add(24 * time.Hour)

	query := a.DB.Model(&models.WaMessage{}).
		Joins("JOIN wa_listeners ON wa_listeners.id = wa_messages.wa_listener_id").
		Joins("JOIN wa_numbers ON wa_numbers.id = wa_listeners.wa_number_id").
		Where("wa_numbers.user_id = ?", a.UserID).
		Where("wa_messages.timestamp >= ? AND wa_messages.timestamp <= ?", startTime, endTime)

	if params.ListenerID != nil {
		query = query.Where("wa_messages.wa_listener_id = ?", *params.ListenerID)
	}

	var messages []models.WaMessage
	query.Order("wa_messages.wa_listener_id, wa_messages.timestamp ASC").Find(&messages)

	// Group by listener ID
	type msgSummary struct {
		ID         uint   `json:"id"`
		SenderName string `json:"sender_name"`
		SenderJID  string `json:"sender_jid"`
		Content    string `json:"content"`
		Timestamp  string `json:"timestamp"`
	}
	type listenerGroup struct {
		ListenerID   uint         `json:"listener_id"`
		MessageCount int          `json:"message_count"`
		Messages     []msgSummary `json:"messages"`
	}

	groupMap := make(map[uint]*listenerGroup)
	for _, m := range messages {
		g, ok := groupMap[m.WaListenerID]
		if !ok {
			g = &listenerGroup{ListenerID: m.WaListenerID}
			groupMap[m.WaListenerID] = g
		}
		g.MessageCount++
		if len(g.Messages) < 50 { // cap per group to avoid token explosion
			g.Messages = append(g.Messages, msgSummary{
				ID:         m.ID,
				SenderName: m.SenderName,
				SenderJID:  m.SenderJID,
				Content:    truncateStr(m.Content, 200),
				Timestamp:  m.Timestamp.Format("2006-01-02 15:04"),
			})
		}
	}

	groups := make([]*listenerGroup, 0, len(groupMap))
	for _, g := range groupMap {
		groups = append(groups, g)
	}

	return map[string]any{
		"start_date":      params.StartDate,
		"end_date":        params.EndDate,
		"total_messages":  len(messages),
		"listener_groups": groups,
	}, nil
}

// sendMessage creates a WaOutbox pending entry for approval.
func (a *WhatsAppAgent) sendMessage(args json.RawMessage) (any, error) {
	var params struct {
		TargetJID string `json:"target_jid"`
		Content   string `json:"content"`
		Context   string `json:"context"`
	}
	json.Unmarshal(args, &params)

	if params.TargetJID == "" || params.Content == "" || params.Context == "" {
		return nil, fmt.Errorf("target_jid, content, and context are all required")
	}

	// Find the first active WaNumber for this user
	var waNumber models.WaNumber
	if err := a.DB.Where("user_id = ?", a.UserID).First(&waNumber).Error; err != nil {
		return nil, fmt.Errorf("no WA number found for user")
	}

	outbox := models.WaOutbox{
		WaNumberID:  waNumber.ID,
		TargetJID:   params.TargetJID,
		Content:     params.Content,
		Status:      "pending",
		RequestedBy: "agent",
		Context:     params.Context,
	}
	if err := a.DB.Create(&outbox).Error; err != nil {
		return nil, fmt.Errorf("failed to create outbox entry: %w", err)
	}

	return map[string]any{
		"outbox_id":   outbox.ID,
		"target_jid":  params.TargetJID,
		"content":     params.Content,
		"context":     params.Context,
		"status":      "pending",
		"note":        "Message queued for approval. The user must approve it in the outbox before it is sent.",
	}, nil
}

// replyToMessage creates a WaOutbox pending entry as a reply to an existing message.
func (a *WhatsAppAgent) replyToMessage(args json.RawMessage) (any, error) {
	var params struct {
		WaMessageID uint   `json:"wa_message_id"`
		Content     string `json:"content"`
		Context     string `json:"context"`
	}
	json.Unmarshal(args, &params)

	if params.WaMessageID == 0 || params.Content == "" || params.Context == "" {
		return nil, fmt.Errorf("wa_message_id, content, and context are all required")
	}

	// Look up the original message (scoped to user via listener join)
	var original models.WaMessage
	err := a.DB.
		Joins("JOIN wa_listeners ON wa_listeners.id = wa_messages.wa_listener_id").
		Joins("JOIN wa_numbers ON wa_numbers.id = wa_listeners.wa_number_id").
		Where("wa_numbers.user_id = ? AND wa_messages.id = ?", a.UserID, params.WaMessageID).
		First(&original).Error
	if err != nil {
		return nil, fmt.Errorf("message not found: %w", err)
	}

	// Find the first active WaNumber for this user
	var waNumber models.WaNumber
	if err := a.DB.Where("user_id = ?", a.UserID).First(&waNumber).Error; err != nil {
		return nil, fmt.Errorf("no WA number found for user")
	}

	outbox := models.WaOutbox{
		WaNumberID:  waNumber.ID,
		TargetJID:   original.SenderJID,
		TargetName:  original.SenderName,
		Content:     params.Content,
		Status:      "pending",
		RequestedBy: "agent",
		Context:     fmt.Sprintf("[Reply to msg #%d from %s] %s", original.ID, original.SenderName, params.Context),
	}
	if err := a.DB.Create(&outbox).Error; err != nil {
		return nil, fmt.Errorf("failed to create outbox entry: %w", err)
	}

	return map[string]any{
		"outbox_id":        outbox.ID,
		"target_jid":       original.SenderJID,
		"target_name":      original.SenderName,
		"original_content": truncateStr(original.Content, 150),
		"reply_content":    params.Content,
		"context":          params.Context,
		"status":           "pending",
		"note":             "Reply queued for approval. The user must approve it in the outbox before it is sent.",
	}, nil
}

// fullChatReport is a compound tool combining list_listeners + summarize_chat + semantic_search.
func (a *WhatsAppAgent) fullChatReport(ctx context.Context, args json.RawMessage) (any, error) {
	var params struct {
		StartDate string `json:"start_date"`
		EndDate   string `json:"end_date"`
	}
	json.Unmarshal(args, &params)

	if params.StartDate == "" || params.EndDate == "" {
		return nil, fmt.Errorf("start_date and end_date are required")
	}

	// 1. List listeners
	listenersResult, _ := a.listListeners(json.RawMessage(`{}`))

	// 2. Summarize all chats in the range
	summarizeArgs, _ := json.Marshal(map[string]any{
		"start_date": params.StartDate,
		"end_date":   params.EndDate,
	})
	summaryResult, _ := a.summarizeChat(summarizeArgs)

	// 3. Semantic search for general activity (if available)
	var semanticResult any
	if a.Weaviate != nil && a.Weaviate.IsAvailable() {
		semanticArgs, _ := json.Marshal(map[string]any{
			"query":      "important messages discussions updates",
			"start_date": params.StartDate,
			"end_date":   params.EndDate,
			"limit":      20,
		})
		semanticResult, _ = a.semanticSearch(ctx, semanticArgs)
	} else {
		semanticResult = map[string]any{
			"available": false,
			"message":   "Weaviate unavailable — semantic highlights not included.",
		}
	}

	return map[string]any{
		"start_date":      params.StartDate,
		"end_date":        params.EndDate,
		"generated_at":    time.Now().Format("2006-01-02 15:04"),
		"listeners":       listenersResult,
		"chat_summary":    summaryResult,
		"semantic_highlights": semanticResult,
	}, nil
}

// truncateStr truncates a string to n runes, appending "..." if truncated.
func truncateStr(s string, n int) string {
	runes := []rune(s)
	if len(runes) <= n {
		return s
	}
	return string(runes[:n]) + "..."
}

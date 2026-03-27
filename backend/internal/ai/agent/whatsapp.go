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
	waService "github.com/cds-id/pdt/backend/internal/services/whatsapp"
	"gorm.io/gorm"
)

type WhatsAppAgent struct {
	DB       *gorm.DB
	UserID   uint
	Weaviate *wvClient.Client
	Manager  WaManager
}

// WaManager is the interface the WhatsApp agent needs from the manager.
type WaManager interface {
	GetGroups(ctx context.Context, numberID uint) ([]waService.GroupInfo, error)
	GetContacts(ctx context.Context, numberID uint) ([]waService.ContactInfo, error)
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

WHATSAPP MESSAGE FORMATTING (for all outgoing messages in send_message/reply_to_message):
- Bold: use *single asterisks* — NEVER use **double asterisks**
- Italic: use _underscores_
- Strikethrough: use ~tildes~
- Monospace: use triple backticks
- Bullet lists: use * followed by a space
- Numbered lists: use 1. 2. 3.
- Quotes: use > at the start of a line
- NEVER use markdown headers (#, ##, ###) — use *BOLD CAPS* instead
- NEVER use markdown tables — use bullet lists for structured data
- NEVER use horizontal rules (---) — use a line of underscores if needed
- Goal: clean, human-to-human look. Technical but readable on mobile.

WORKFLOW:
- Use list_listeners to see available chats/groups.
- Use search_messages for keyword search, semantic_search for concept/topic search.
- Use summarize_chat or full_chat_report for conversation overviews.
- Use send_message or reply_to_message ONLY when the user explicitly asks to send something.
- After creating an outbox entry, show the user the message content and ask for confirmation.
- When the user confirms (e.g., "yes", "send it", "ok", "kirim"), use approve_outbox to approve it.
- The sender worker will send the approved message within seconds.

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
			Name:        "list_contacts",
			Description: "List WhatsApp contacts and groups available for sending messages. Use this to find the correct JID when the user mentions a contact by name.",
			InputSchema: json.RawMessage(`{
				"type": "object",
				"properties": {
					"search": {"type": "string", "description": "Optional: filter contacts/groups by name"}
				}
			}`),
		},
		{
			Name:        "list_repositories",
			Description: "List all registered Git repositories (GitHub/GitLab) for the user. Shows repo name, owner, provider, and URL.",
			InputSchema: json.RawMessage(`{
				"type": "object",
				"properties": {}
			}`),
		},
		{
			Name:        "list_commits",
			Description: "List recent commits for a repository or all repositories. Can filter by date range and search by message keyword. Results can be sent via WhatsApp.",
			InputSchema: json.RawMessage(`{
				"type": "object",
				"properties": {
					"repo_id": {"type": "integer", "description": "Optional: filter by repository ID"},
					"days_back": {"type": "integer", "description": "Number of days to look back (default 7)"},
					"keyword": {"type": "string", "description": "Optional: search commit messages by keyword"},
					"limit": {"type": "integer", "description": "Max results (default 30)"}
				}
			}`),
		},
		{
			Name:        "send_commits_report",
			Description: "Generate a commits summary for a repository (or all repos) and send it via WhatsApp. Auto-approved for immediate sending.",
			InputSchema: json.RawMessage(`{
				"type": "object",
				"properties": {
					"target_jid": {"type": "string", "description": "Target WhatsApp JID to send the report to"},
					"repo_id": {"type": "integer", "description": "Optional: specific repository ID. 0 = all repos."},
					"days_back": {"type": "integer", "description": "Number of days to look back (default 7)"}
				},
				"required": ["target_jid"]
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
			Description: "Send a WhatsApp message. By default creates a pending outbox entry for user approval. Set auto_approve=true when the user has EXPLICITLY asked to send (e.g., 'kirim pesan ke X', 'send message to Y') — this approves immediately and the message is sent within seconds.",
			InputSchema: json.RawMessage(`{
				"type": "object",
				"properties": {
					"target_jid": {"type": "string", "description": "Target WhatsApp JID (e.g., '628123456789@s.whatsapp.net' or group JID)"},
					"content": {"type": "string", "description": "Message content to send"},
					"context": {"type": "string", "description": "Why you are sending this message — reason and background"},
					"auto_approve": {"type": "boolean", "description": "Set true if user explicitly asked to send. Default false (pending approval)."}
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
			Name:        "approve_outbox",
			Description: "Approve a pending outbox message so it gets sent via WhatsApp. If outbox_id is omitted, approves the most recent pending message. Use after user confirms sending.",
			InputSchema: json.RawMessage(`{
				"type": "object",
				"properties": {
					"outbox_id": {"type": "integer", "description": "Optional: specific outbox ID. If omitted, approves the latest pending message."}
				}
			}`),
		},
		{
			Name:        "send_briefing",
			Description: "Generate a work briefing/summary (sprint cards, commits, blockers) and send it to a WhatsApp contact or group. Use when user asks to share their work summary, standup, or briefing via WhatsApp. The briefing covers active sprint cards, recent commits, and status.",
			InputSchema: json.RawMessage(`{
				"type": "object",
				"properties": {
					"target_jid": {"type": "string", "description": "Target WhatsApp JID to send the briefing to"},
					"days_back": {"type": "integer", "description": "Number of days to look back for commits (default 1)"},
					"assignee": {"type": "string", "description": "Optional: filter by assignee name"}
				},
				"required": ["target_jid"]
			}`),
		},
		{
			Name:        "send_report",
			Description: "Fetch a generated PDT report (daily/monthly) and send it to a WhatsApp contact or group. The report content is formatted for WhatsApp and auto-approved for sending. Use when user asks to share/send a report via WhatsApp.",
			InputSchema: json.RawMessage(`{
				"type": "object",
				"properties": {
					"report_id": {"type": "integer", "description": "The report ID to send. Use 0 to send the latest report."},
					"target_jid": {"type": "string", "description": "Target WhatsApp JID to send the report to"},
					"report_date": {"type": "string", "description": "Optional: find report by date (YYYY-MM-DD) instead of ID"}
				},
				"required": ["target_jid"]
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
	case "list_contacts":
		return a.listContacts(ctx, args)
	case "list_repositories":
		return a.listRepositories()
	case "list_commits":
		return a.listCommits(args)
	case "send_commits_report":
		return a.sendCommitsReport(args)
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
	case "approve_outbox":
		return a.approveOutbox(args)
	case "send_briefing":
		return a.sendBriefing(args)
	case "send_report":
		return a.sendReport(args)
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
func (a *WhatsAppAgent) listContacts(ctx context.Context, args json.RawMessage) (any, error) {
	var params struct {
		Search string `json:"search"`
	}
	json.Unmarshal(args, &params)

	// Get user's first connected number
	var waNumber models.WaNumber
	if err := a.DB.Where("user_id = ? AND status = ?", a.UserID, "connected").First(&waNumber).Error; err != nil {
		return map[string]any{"error": "No connected WhatsApp number"}, nil
	}

	var allContacts []map[string]any

	// Get groups
	if a.Manager != nil {
		groups, err := a.Manager.GetGroups(ctx, waNumber.ID)
		if err == nil {
			for _, g := range groups {
				if params.Search == "" || strings.Contains(strings.ToLower(g.Name), strings.ToLower(params.Search)) {
					allContacts = append(allContacts, map[string]any{
						"jid":  g.JID,
						"name": g.Name,
						"type": "group",
					})
				}
			}
		}

		contacts, err := a.Manager.GetContacts(ctx, waNumber.ID)
		if err == nil {
			for _, c := range contacts {
				if c.Name == "" {
					continue
				}
				if params.Search == "" || strings.Contains(strings.ToLower(c.Name), strings.ToLower(params.Search)) {
					allContacts = append(allContacts, map[string]any{
						"jid":  c.JID,
						"name": c.Name,
						"type": "personal",
					})
				}
			}
		}
	}

	// Also include registered listeners
	var listeners []models.WaListener
	a.DB.Where("wa_number_id = ?", waNumber.ID).Find(&listeners)
	existingJIDs := map[string]bool{}
	for _, c := range allContacts {
		existingJIDs[c["jid"].(string)] = true
	}
	for _, l := range listeners {
		if !existingJIDs[l.JID] {
			allContacts = append(allContacts, map[string]any{
				"jid":        l.JID,
				"name":       l.Name,
				"type":       l.Type,
				"is_listener": true,
			})
		}
	}

	return map[string]any{"contacts": allContacts, "count": len(allContacts)}, nil
}

func (a *WhatsAppAgent) listRepositories() (any, error) {
	var repos []models.Repository
	a.DB.Where("user_id = ?", a.UserID).Order("name asc").Find(&repos)

	type repoInfo struct {
		ID       uint   `json:"id"`
		Name     string `json:"name"`
		Owner    string `json:"owner"`
		Provider string `json:"provider"`
		URL      string `json:"url"`
	}

	var result []repoInfo
	for _, r := range repos {
		result = append(result, repoInfo{
			ID:       r.ID,
			Name:     r.Name,
			Owner:    r.Owner,
			Provider: string(r.Provider),
			URL:      r.URL,
		})
	}

	return map[string]any{"repositories": result, "count": len(result)}, nil
}

func (a *WhatsAppAgent) listCommits(args json.RawMessage) (any, error) {
	var params struct {
		RepoID   uint   `json:"repo_id"`
		DaysBack int    `json:"days_back"`
		Keyword  string `json:"keyword"`
		Limit    int    `json:"limit"`
	}
	json.Unmarshal(args, &params)

	if params.DaysBack <= 0 {
		params.DaysBack = 7
	}
	if params.Limit <= 0 {
		params.Limit = 30
	}

	since := time.Now().AddDate(0, 0, -params.DaysBack)

	query := a.DB.Where("commits.user_id = ? AND commits.date >= ?", a.UserID, since).
		Joins("JOIN repositories ON repositories.id = commits.repo_id")

	if params.RepoID > 0 {
		query = query.Where("commits.repo_id = ?", params.RepoID)
	}
	if params.Keyword != "" {
		query = query.Where("commits.message LIKE ?", "%"+params.Keyword+"%")
	}

	var commits []models.Commit
	query.Order("commits.date desc").Limit(params.Limit).Find(&commits)

	// Group by repo
	type commitInfo struct {
		SHA     string `json:"sha"`
		Message string `json:"message"`
		Author  string `json:"author"`
		Date    string `json:"date"`
		RepoID  uint   `json:"repo_id"`
	}

	var result []commitInfo
	for _, c := range commits {
		result = append(result, commitInfo{
			SHA:     c.SHA[:min(len(c.SHA), 7)],
			Message: c.Message,
			Author:  c.Author,
			Date:    c.Date.Format("2006-01-02 15:04"),
			RepoID:  c.RepoID,
		})
	}

	return map[string]any{"commits": result, "count": len(result), "days_back": params.DaysBack}, nil
}

func (a *WhatsAppAgent) sendCommitsReport(args json.RawMessage) (any, error) {
	var params struct {
		TargetJID string `json:"target_jid"`
		RepoID    uint   `json:"repo_id"`
		DaysBack  int    `json:"days_back"`
	}
	json.Unmarshal(args, &params)

	if params.TargetJID == "" {
		return nil, fmt.Errorf("target_jid is required")
	}
	if params.DaysBack <= 0 {
		params.DaysBack = 7
	}

	since := time.Now().AddDate(0, 0, -params.DaysBack)

	query := a.DB.Where("commits.user_id = ? AND commits.date >= ?", a.UserID, since).
		Joins("JOIN repositories ON repositories.id = commits.repo_id")

	if params.RepoID > 0 {
		query = query.Where("commits.repo_id = ?", params.RepoID)
	}

	var commits []models.Commit
	query.Order("commits.date desc").Limit(50).Preload("Repository").Find(&commits)

	// Group commits by repo
	repoCommits := map[string][]models.Commit{}
	for _, c := range commits {
		repoName := fmt.Sprintf("%s/%s", c.Repository.Owner, c.Repository.Name)
		repoCommits[repoName] = append(repoCommits[repoName], c)
	}

	// Build WhatsApp message
	var sb strings.Builder
	sb.WriteString(fmt.Sprintf("*📝 Commits Report*\n_%s — last %d days_\n", time.Now().Format("2006-01-02"), params.DaysBack))

	totalCommits := 0
	for repoName, rCommits := range repoCommits {
		sb.WriteString(fmt.Sprintf("\n*%s* (%d commits):\n", repoName, len(rCommits)))
		for _, c := range rCommits {
			msg := c.Message
			if len(msg) > 70 {
				msg = msg[:70] + "..."
			}
			sb.WriteString(fmt.Sprintf("* `%s` %s _%s_\n", c.SHA[:7], msg, c.Date.Format("Jan 02")))
			totalCommits++
		}
	}

	if totalCommits == 0 {
		sb.WriteString("\n_No commits found in this period._")
	}

	content := sb.String()

	// Find WA number and target
	var waNumber models.WaNumber
	if err := a.DB.Where("user_id = ?", a.UserID).First(&waNumber).Error; err != nil {
		return map[string]any{"error": "No WhatsApp number configured"}, nil
	}

	targetName := params.TargetJID
	var listener models.WaListener
	if err := a.DB.Where("jid = ? AND wa_number_id = ?", params.TargetJID, waNumber.ID).First(&listener).Error; err == nil {
		targetName = listener.Name
	}

	now := time.Now()
	outbox := models.WaOutbox{
		WaNumberID:  waNumber.ID,
		TargetJID:   params.TargetJID,
		TargetName:  targetName,
		Content:     content,
		Status:      "approved",
		RequestedBy: "agent",
		Context:     fmt.Sprintf("Commits report (%d days) sent via WhatsApp as requested", params.DaysBack),
		ApprovedAt:  &now,
	}
	a.DB.Create(&outbox)

	return map[string]any{
		"status":        "approved",
		"outbox_id":     outbox.ID,
		"target":        targetName,
		"total_commits": totalCommits,
		"repos":         len(repoCommits),
		"message":       fmt.Sprintf("Commits report (%d commits across %d repos) will be sent to %s within seconds.", totalCommits, len(repoCommits), targetName),
	}, nil
}

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
		TargetJID   string `json:"target_jid"`
		Content     string `json:"content"`
		Context     string `json:"context"`
		AutoApprove bool   `json:"auto_approve"`
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

	// Look up target name from listeners
	targetName := params.TargetJID
	var listener models.WaListener
	if err := a.DB.Where("jid = ? AND wa_number_id = ?", params.TargetJID, waNumber.ID).First(&listener).Error; err == nil {
		targetName = listener.Name
	}

	status := "pending"
	var approvedAt *time.Time
	if params.AutoApprove {
		status = "approved"
		now := time.Now()
		approvedAt = &now
	}

	outbox := models.WaOutbox{
		WaNumberID:  waNumber.ID,
		TargetJID:   params.TargetJID,
		TargetName:  targetName,
		Content:     params.Content,
		Status:      status,
		RequestedBy: "agent",
		Context:     params.Context,
		ApprovedAt:  approvedAt,
	}
	if err := a.DB.Create(&outbox).Error; err != nil {
		return nil, fmt.Errorf("failed to create outbox entry: %w", err)
	}

	if params.AutoApprove {
		return map[string]any{
			"outbox_id":  outbox.ID,
			"target":     targetName,
			"content":    params.Content,
			"status":     "approved",
			"note":       "Message approved and will be sent within seconds.",
		}, nil
	}

	return map[string]any{
		"outbox_id":  outbox.ID,
		"target":     targetName,
		"content":    params.Content,
		"status":     "pending",
		"note":       "Message queued for approval. The user must approve it in the outbox or confirm here.",
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
func (a *WhatsAppAgent) approveOutbox(args json.RawMessage) (any, error) {
	var params struct {
		OutboxID uint `json:"outbox_id"`
	}
	json.Unmarshal(args, &params)

	var item models.WaOutbox

	if params.OutboxID > 0 {
		// Approve specific outbox item
		err := a.DB.
			Joins("JOIN wa_numbers ON wa_numbers.id = wa_outboxes.wa_number_id").
			Where("wa_outboxes.id = ? AND wa_numbers.user_id = ? AND wa_outboxes.status = ?", params.OutboxID, a.UserID, "pending").
			First(&item).Error
		if err != nil {
			return map[string]any{"error": "Outbox item not found or not pending"}, nil
		}
	} else {
		// No ID given — approve the most recent pending outbox item
		err := a.DB.
			Joins("JOIN wa_numbers ON wa_numbers.id = wa_outboxes.wa_number_id").
			Where("wa_numbers.user_id = ? AND wa_outboxes.status = ?", a.UserID, "pending").
			Order("wa_outboxes.created_at desc").
			First(&item).Error
		if err != nil {
			return map[string]any{"error": "No pending outbox items found"}, nil
		}
	}

	now := time.Now()
	a.DB.Model(&item).Updates(map[string]any{
		"status":      "approved",
		"approved_at": now,
	})

	return map[string]any{
		"status":    "approved",
		"message":   fmt.Sprintf("Message to %s approved and will be sent shortly.", item.TargetName),
		"outbox_id": item.ID,
	}, nil
}

func (a *WhatsAppAgent) sendBriefing(args json.RawMessage) (any, error) {
	var params struct {
		TargetJID string `json:"target_jid"`
		DaysBack  int    `json:"days_back"`
		Assignee  string `json:"assignee"`
	}
	json.Unmarshal(args, &params)

	if params.TargetJID == "" {
		return nil, fmt.Errorf("target_jid is required")
	}
	if params.DaysBack <= 0 {
		params.DaysBack = 1
	}

	since := time.Now().AddDate(0, 0, -params.DaysBack)

	// Get active sprint cards
	var cards []models.JiraCard
	a.DB.Joins("JOIN sprints ON sprints.id = jira_cards.sprint_id").
		Where("jira_cards.user_id = ? AND sprints.state = ?", a.UserID, "active").
		Find(&cards)

	// Get recent commits
	var commits []models.Commit
	a.DB.Where("user_id = ? AND committed_at >= ?", a.UserID, since).
		Order("committed_at desc").Limit(20).Find(&commits)

	// Build WhatsApp-formatted briefing
	var sb strings.Builder
	sb.WriteString(fmt.Sprintf("*📋 Work Briefing*\n_%s_\n\n", time.Now().Format("2006-01-02 15:04")))

	// Cards by status
	if len(cards) > 0 {
		statusGroups := map[string][]models.JiraCard{}
		for _, c := range cards {
			status := strings.ToLower(c.Status)
			if params.Assignee == "" || strings.EqualFold(c.Assignee, params.Assignee) {
				statusGroups[status] = append(statusGroups[status], c)
			}
		}

		sb.WriteString("*Active Sprint Cards:*\n")
		for status, group := range statusGroups {
			sb.WriteString(fmt.Sprintf("\n_%s_ (%d):\n", status, len(group)))
			for _, c := range group {
				assignee := ""
				if c.Assignee != "" {
					assignee = fmt.Sprintf(" [%s]", c.Assignee)
				}
				sb.WriteString(fmt.Sprintf("* %s — %s%s\n", c.Key, c.Summary, assignee))
			}
		}
	} else {
		sb.WriteString("_No active sprint cards found._\n")
	}

	// Recent commits
	sb.WriteString(fmt.Sprintf("\n*Recent Commits (%d days):*\n", params.DaysBack))
	if len(commits) > 0 {
		for _, c := range commits {
			msg := c.Message
			if len(msg) > 80 {
				msg = msg[:80] + "..."
			}
			sb.WriteString(fmt.Sprintf("* `%s` %s\n", c.SHA[:7], msg))
		}
	} else {
		sb.WriteString("_No recent commits._\n")
	}

	content := sb.String()

	// Find WA number and target name
	var waNumber models.WaNumber
	if err := a.DB.Where("user_id = ?", a.UserID).First(&waNumber).Error; err != nil {
		return map[string]any{"error": "No WhatsApp number configured"}, nil
	}

	targetName := params.TargetJID
	var listener models.WaListener
	if err := a.DB.Where("jid = ? AND wa_number_id = ?", params.TargetJID, waNumber.ID).First(&listener).Error; err == nil {
		targetName = listener.Name
	}

	// Auto-approve
	now := time.Now()
	outbox := models.WaOutbox{
		WaNumberID:  waNumber.ID,
		TargetJID:   params.TargetJID,
		TargetName:  targetName,
		Content:     content,
		Status:      "approved",
		RequestedBy: "agent",
		Context:     "User requested work briefing sent via WhatsApp",
		ApprovedAt:  &now,
	}
	a.DB.Create(&outbox)

	return map[string]any{
		"status":     "approved",
		"outbox_id":  outbox.ID,
		"target":     targetName,
		"cards":      len(cards),
		"commits":    len(commits),
		"message":    fmt.Sprintf("Briefing with %d cards and %d commits will be sent to %s within seconds.", len(cards), len(commits), targetName),
	}, nil
}

func (a *WhatsAppAgent) sendReport(args json.RawMessage) (any, error) {
	var params struct {
		ReportID   uint   `json:"report_id"`
		TargetJID  string `json:"target_jid"`
		ReportDate string `json:"report_date"`
	}
	json.Unmarshal(args, &params)

	if params.TargetJID == "" {
		return nil, fmt.Errorf("target_jid is required")
	}

	// Find the report
	var report models.Report
	if params.ReportID > 0 {
		if err := a.DB.Where("id = ? AND user_id = ?", params.ReportID, a.UserID).First(&report).Error; err != nil {
			return map[string]any{"error": "Report not found"}, nil
		}
	} else if params.ReportDate != "" {
		if err := a.DB.Where("user_id = ? AND date = ?", a.UserID, params.ReportDate).Order("created_at desc").First(&report).Error; err != nil {
			return map[string]any{"error": fmt.Sprintf("No report found for date %s", params.ReportDate)}, nil
		}
	} else {
		// Latest report
		if err := a.DB.Where("user_id = ?", a.UserID).Order("created_at desc").First(&report).Error; err != nil {
			return map[string]any{"error": "No reports found"}, nil
		}
	}

	// Format report content for WhatsApp
	content := fmt.Sprintf("*%s*\n_%s_\n\n%s", report.Title, report.Date, report.Content)

	// Truncate if too long for WhatsApp (max ~65536 chars)
	if len(content) > 10000 {
		content = content[:10000] + "\n\n_... (report truncated, full version available in PDT)_"
	}

	// Find WA number
	var waNumber models.WaNumber
	if err := a.DB.Where("user_id = ?", a.UserID).First(&waNumber).Error; err != nil {
		return map[string]any{"error": "No WhatsApp number configured"}, nil
	}

	// Look up target name
	targetName := params.TargetJID
	var listener models.WaListener
	if err := a.DB.Where("jid = ? AND wa_number_id = ?", params.TargetJID, waNumber.ID).First(&listener).Error; err == nil {
		targetName = listener.Name
	}

	// Create and auto-approve
	now := time.Now()
	outbox := models.WaOutbox{
		WaNumberID:  waNumber.ID,
		TargetJID:   params.TargetJID,
		TargetName:  targetName,
		Content:     content,
		Status:      "approved",
		RequestedBy: "agent",
		Context:     fmt.Sprintf("Sending report '%s' (%s) as requested by user", report.Title, report.Date),
		ApprovedAt:  &now,
	}
	a.DB.Create(&outbox)

	return map[string]any{
		"status":      "approved",
		"outbox_id":   outbox.ID,
		"report_id":   report.ID,
		"report_title": report.Title,
		"report_date": report.Date,
		"target":      targetName,
		"message":     fmt.Sprintf("Report '%s' will be sent to %s within seconds.", report.Title, targetName),
	}, nil
}

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

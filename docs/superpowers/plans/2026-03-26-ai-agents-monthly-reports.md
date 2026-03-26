# AI Agents, Monthly Reports & Jira Bug Fix — Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Add a conversational AI assistant with multi-agent architecture, monthly reports with AI narrative, and fix the Jira card visibility bug.

**Architecture:** Go-native multi-agent system with MiniMax 2.7 as the LLM. Orchestrator routes user messages to specialist agents (Git, Jira, Report) that each have their own tools. WebSocket streams responses in real-time. Monthly reports aggregate daily data and use AI for narrative generation.

**Tech Stack:** Go 1.24 / Gin / GORM / gorilla/websocket / MiniMax API (OpenAI-compatible) / React 18 / TypeScript / RTK Query / Tailwind CSS

---

## File Structure (New & Modified)

### Backend — New Files

```
backend/internal/
  ai/
    minimax/
      client.go          # MiniMax HTTP client with SSE streaming
      types.go           # Request/response types (OpenAI-compatible)
    agent/
      types.go           # Agent interface, ToolDefinition, ToolCall types
      loop.go            # Shared agent tool-calling loop (stream + execute)
      orchestrator.go    # Routes messages to specialist agents
      git.go             # Git Agent (search commits, repo stats)
      jira.go            # Jira Agent (sprints, cards, linking)
      report.go          # Report Agent (generate, list, monthly)
  handlers/
    chat.go              # WebSocket upgrade, conversation CRUD endpoints
  models/
    conversation.go      # Conversation, ChatMessage, AIUsage models
```

### Backend — Modified Files

```
backend/internal/config/config.go        # Add MiniMax + monthly report env vars
backend/internal/database/database.go    # Add new models to Migrate()
backend/internal/models/report.go        # Add report_type, month, year fields
backend/internal/services/jira/jira.go   # Fix pagination in FetchSprintIssues
backend/internal/worker/jira.go          # Fix: sync all sprint states, not just active
backend/internal/worker/scheduler.go     # Add monthly report auto-generation loop
backend/internal/worker/reports.go       # Add monthly report generation logic
backend/internal/services/report/report.go  # Add BuildMonthlyReportData()
backend/internal/handlers/report.go      # Add GenerateMonthly endpoint
backend/cmd/server/main.go              # Register chat handler, WebSocket route, new endpoints
```

### Frontend — New Files

```
frontend/src/
  presentation/pages/
    AssistantPage.tsx                    # Full-page chat interface
  presentation/components/
    chat/
      ChatSidebar.tsx                   # Conversation history list
      ChatMessage.tsx                   # Message bubble (user/assistant/tool)
      ChatInput.tsx                     # Text input with send button
      ToolStatus.tsx                    # Tool execution indicator
  infrastructure/services/
    chat.service.ts                     # Conversation CRUD (RTK Query) + WebSocket hook
  domain/chat/interfaces/
    chat.interface.ts                   # TypeScript interfaces for chat
```

### Frontend — Modified Files

```
frontend/src/config/navigation.ts                    # Add Assistant route
frontend/src/presentation/routes/index.tsx           # Add AssistantPage route
frontend/src/infrastructure/constants/api.constants.ts  # Add chat/conversation endpoints
frontend/src/infrastructure/services/api.ts          # Add 'Conversation' tag type
frontend/src/presentation/pages/ReportsPage.tsx      # Add monthly report tab
frontend/src/infrastructure/services/report.service.ts  # Add monthly report endpoint
```

---

## Task 1: Fix Jira Bug — Pagination & Sprint Filtering

**Files:**
- Modify: `backend/internal/services/jira/jira.go`
- Modify: `backend/internal/worker/jira.go`

This is the independent bug fix. The Jira Agile API paginates sprint issues (default 50 per page). The current code likely doesn't paginate. Also, the sync only processes `active` sprints — cards in `future` or `closed` sprints are never synced.

- [ ] **Step 1: Read the current FetchSprintIssues implementation**

Open `backend/internal/services/jira/jira.go` and find `FetchSprintIssues`. Check if it handles the `startAt` / `maxResults` pagination fields in the Jira response.

- [ ] **Step 2: Fix FetchSprintIssues to paginate**

In `backend/internal/services/jira/jira.go`, update `FetchSprintIssues` to loop through all pages:

```go
func (c *Client) FetchSprintIssues(sprintID int) ([]CardInfo, error) {
	var allCards []CardInfo
	startAt := 0
	maxResults := 50

	for {
		url := fmt.Sprintf("%s/rest/agile/1.0/sprint/%d/issue?startAt=%d&maxResults=%d",
			c.baseURL(), sprintID, startAt, maxResults)

		body, err := c.doRequest(url)
		if err != nil {
			return nil, fmt.Errorf("fetch sprint issues: %w", err)
		}

		var resp struct {
			Issues []struct {
				Key    string `json:"key"`
				Fields struct {
					Summary  string `json:"summary"`
					Status   struct{ Name string } `json:"status"`
					Assignee *struct{ DisplayName string } `json:"assignee"`
				} `json:"fields"`
			} `json:"issues"`
			StartAt    int `json:"startAt"`
			MaxResults int `json:"maxResults"`
			Total      int `json:"total"`
		}

		if err := json.Unmarshal(body, &resp); err != nil {
			return nil, fmt.Errorf("parse sprint issues: %w", err)
		}

		for _, issue := range resp.Issues {
			assignee := ""
			if issue.Fields.Assignee != nil {
				assignee = issue.Fields.Assignee.DisplayName
			}
			allCards = append(allCards, CardInfo{
				Key:      issue.Key,
				Summary:  issue.Fields.Summary,
				Status:   issue.Fields.Status.Name,
				Assignee: assignee,
			})
		}

		startAt += len(resp.Issues)
		if startAt >= resp.Total {
			break
		}
	}

	return allCards, nil
}
```

- [ ] **Step 3: Fix worker to sync all sprint states, not just active**

In `backend/internal/worker/jira.go`, find the `SyncUserJira` function. Change the condition that only syncs cards for `active` sprints to sync cards for `active` and `closed` sprints (users want to see recently completed work too):

```go
// Before: if s.State == "active" {
// After:
if s.State == "active" || s.State == "closed" {
    cards, err := client.FetchSprintIssues(s.ID)
    if err != nil {
        log.Printf("[jira-sync] user=%d sprint=%d fetch cards error: %v", userID, s.ID, err)
        continue
    }
    for _, card := range cards {
        if !helpers.FilterByProjectKeys(card.Key, user.JiraProjectKeys) {
            continue
        }
        // ... rest of upsert logic
    }
}
```

- [ ] **Step 4: Add sync logging for traceability**

In `backend/internal/worker/jira.go`, add log statements to `SyncUserJira`:

```go
log.Printf("[jira-sync] user=%d starting sync", userID)
log.Printf("[jira-sync] user=%d found %d boards", userID, len(boards))
// Inside board loop:
log.Printf("[jira-sync] user=%d board=%d found %d sprints", userID, boardID, len(sprints))
// Inside sprint loop:
log.Printf("[jira-sync] user=%d sprint=%d (%s) found %d cards, %d after filter",
    userID, s.ID, s.State, len(cards), savedCount)
```

- [ ] **Step 5: Test against real Jira instance**

Run the backend with the existing `.env` and trigger a manual sync:

```bash
cd /home/nst/GolandProjects/pdt/backend
go run cmd/server/main.go
```

Then call the sync endpoint:
```bash
curl -X POST http://localhost:8080/api/sync/commits \
  -H "Authorization: Bearer <token>" \
  -H "Content-Type: application/json"
```

Check logs for the `[jira-sync]` output. Verify cards appear for all sprints.

- [ ] **Step 6: Commit**

```bash
git add backend/internal/services/jira/jira.go backend/internal/worker/jira.go
git commit -m "fix: add Jira pagination and sync closed sprints for missing cards"
```

---

## Task 2: Add MiniMax API Client

**Files:**
- Create: `backend/internal/ai/minimax/types.go`
- Create: `backend/internal/ai/minimax/client.go`
- Modify: `backend/internal/config/config.go`

- [ ] **Step 1: Add MiniMax config fields**

In `backend/internal/config/config.go`, add to the `Config` struct:

```go
MiniMaxAPIKey   string
MiniMaxGroupID  string
AIContextWindow int
```

In the `Load()` function, add:

```go
aiContextWindow, _ := strconv.Atoi(getEnv("AI_CONTEXT_WINDOW", "20"))
```

And set:
```go
cfg.MiniMaxAPIKey = getEnv("MINIMAX_API_KEY", "")
cfg.MiniMaxGroupID = getEnv("MINIMAX_GROUP_ID", "")
cfg.AIContextWindow = aiContextWindow
```

- [ ] **Step 2: Create MiniMax types**

Create `backend/internal/ai/minimax/types.go`:

```go
package minimax

import "encoding/json"

type ChatRequest struct {
	Model       string    `json:"model"`
	Messages    []Message `json:"messages"`
	Tools       []Tool    `json:"tools,omitempty"`
	Stream      bool      `json:"stream"`
	Temperature float64   `json:"temperature,omitempty"`
}

type Message struct {
	Role       string     `json:"role"`
	Content    string     `json:"content,omitempty"`
	ToolCalls  []ToolCall `json:"tool_calls,omitempty"`
	ToolCallID string     `json:"tool_call_id,omitempty"`
}

type Tool struct {
	Type     string      `json:"type"`
	Function FunctionDef `json:"function"`
}

type FunctionDef struct {
	Name        string          `json:"name"`
	Description string          `json:"description"`
	Parameters  json.RawMessage `json:"parameters"`
}

type ToolCall struct {
	ID       string       `json:"id"`
	Type     string       `json:"type"`
	Function FunctionCall `json:"function"`
}

type FunctionCall struct {
	Name      string `json:"name"`
	Arguments string `json:"arguments"`
}

type ChatResponse struct {
	ID      string   `json:"id"`
	Choices []Choice `json:"choices"`
	Usage   Usage    `json:"usage"`
}

type Choice struct {
	Delta        Delta  `json:"delta"`
	FinishReason string `json:"finish_reason"`
}

type Delta struct {
	Role      string     `json:"role,omitempty"`
	Content   string     `json:"content,omitempty"`
	ToolCalls []ToolCall `json:"tool_calls,omitempty"`
}

type Usage struct {
	PromptTokens     int `json:"prompt_tokens"`
	CompletionTokens int `json:"completion_tokens"`
	TotalTokens      int `json:"total_tokens"`
}
```

- [ ] **Step 3: Create MiniMax streaming client**

Create `backend/internal/ai/minimax/client.go`:

```go
package minimax

import (
	"bufio"
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
)

const apiURL = "https://api.minimaxi.chat/v1/text/chatcompletion_v2"

type Client struct {
	APIKey  string
	GroupID string
	Model   string
}

func NewClient(apiKey, groupID string) *Client {
	return &Client{
		APIKey:  apiKey,
		GroupID: groupID,
		Model:   "MiniMax-Text-01",
	}
}

// StreamEvent represents a single SSE event from the MiniMax API.
type StreamEvent struct {
	Content      string     // Text content delta
	ToolCalls    []ToolCall // Tool call deltas
	FinishReason string     // "stop", "tool_calls", or ""
	Usage        *Usage     // Only present on final event
	Err          error      // Non-nil if something went wrong
}

// ChatStream sends a streaming chat request and returns a channel of events.
func (c *Client) ChatStream(req ChatRequest) (<-chan StreamEvent, error) {
	req.Stream = true
	if req.Model == "" {
		req.Model = c.Model
	}

	body, err := json.Marshal(req)
	if err != nil {
		return nil, fmt.Errorf("marshal request: %w", err)
	}

	httpReq, err := http.NewRequest("POST", apiURL, bytes.NewReader(body))
	if err != nil {
		return nil, fmt.Errorf("create request: %w", err)
	}
	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("Authorization", "Bearer "+c.APIKey)

	resp, err := http.DefaultClient.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("do request: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		respBody, _ := io.ReadAll(resp.Body)
		resp.Body.Close()
		return nil, fmt.Errorf("minimax API error %d: %s", resp.StatusCode, string(respBody))
	}

	ch := make(chan StreamEvent, 32)
	go func() {
		defer resp.Body.Close()
		defer close(ch)
		c.readSSE(resp.Body, ch)
	}()

	return ch, nil
}

func (c *Client) readSSE(body io.Reader, ch chan<- StreamEvent) {
	scanner := bufio.NewScanner(body)
	// Accumulate tool calls across deltas (they come in pieces)
	toolCallMap := make(map[int]*ToolCall)

	for scanner.Scan() {
		line := scanner.Text()
		if !strings.HasPrefix(line, "data: ") {
			continue
		}
		data := strings.TrimPrefix(line, "data: ")
		if data == "[DONE]" {
			return
		}

		var resp ChatResponse
		if err := json.Unmarshal([]byte(data), &resp); err != nil {
			ch <- StreamEvent{Err: fmt.Errorf("parse SSE: %w", err)}
			return
		}

		if len(resp.Choices) == 0 {
			continue
		}

		choice := resp.Choices[0]
		evt := StreamEvent{
			Content:      choice.Delta.Content,
			FinishReason: choice.FinishReason,
		}

		// Accumulate tool call deltas
		for _, tc := range choice.Delta.ToolCalls {
			idx := 0 // Default index
			if existing, ok := toolCallMap[idx]; ok {
				existing.Function.Arguments += tc.Function.Arguments
			} else {
				copy := tc
				toolCallMap[idx] = &copy
			}
		}

		if resp.Usage.TotalTokens > 0 {
			evt.Usage = &resp.Usage
		}

		// When finish_reason is "tool_calls", emit accumulated tool calls
		if choice.FinishReason == "tool_calls" {
			for _, tc := range toolCallMap {
				evt.ToolCalls = append(evt.ToolCalls, *tc)
			}
			toolCallMap = make(map[int]*ToolCall)
		}

		ch <- evt
	}

	if err := scanner.Err(); err != nil {
		ch <- StreamEvent{Err: fmt.Errorf("read SSE stream: %w", err)}
	}
}

// Chat sends a non-streaming chat request and returns the full response.
func (c *Client) Chat(req ChatRequest) (*ChatResponse, error) {
	req.Stream = false
	if req.Model == "" {
		req.Model = c.Model
	}

	body, err := json.Marshal(req)
	if err != nil {
		return nil, fmt.Errorf("marshal request: %w", err)
	}

	httpReq, err := http.NewRequest("POST", apiURL, bytes.NewReader(body))
	if err != nil {
		return nil, fmt.Errorf("create request: %w", err)
	}
	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("Authorization", "Bearer "+c.APIKey)

	resp, err := http.DefaultClient.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("do request: %w", err)
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("read response: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("minimax API error %d: %s", resp.StatusCode, string(respBody))
	}

	var chatResp ChatResponse
	if err := json.Unmarshal(respBody, &chatResp); err != nil {
		return nil, fmt.Errorf("parse response: %w", err)
	}
	return &chatResp, nil
}
```

- [ ] **Step 4: Commit**

```bash
git add backend/internal/ai/minimax/ backend/internal/config/config.go
git commit -m "feat: add MiniMax 2.7 API client with SSE streaming support"
```

---

## Task 3: Add Conversation Models

**Files:**
- Create: `backend/internal/models/conversation.go`
- Modify: `backend/internal/database/database.go`

- [ ] **Step 1: Create conversation models**

Create `backend/internal/models/conversation.go`:

```go
package models

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

type Conversation struct {
	ID        string    `gorm:"type:varchar(36);primarykey" json:"id"`
	UserID    uint      `gorm:"index;not null" json:"user_id"`
	Title     string    `gorm:"type:varchar(255)" json:"title"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
	User      User      `gorm:"foreignKey:UserID" json:"-"`
	Messages  []ChatMessage `gorm:"foreignKey:ConversationID" json:"messages,omitempty"`
}

func (c *Conversation) BeforeCreate(tx *gorm.DB) error {
	if c.ID == "" {
		c.ID = uuid.New().String()
	}
	return nil
}

type ChatMessage struct {
	ID             string    `gorm:"type:varchar(36);primarykey" json:"id"`
	ConversationID string    `gorm:"type:varchar(36);index;not null" json:"conversation_id"`
	Role           string    `gorm:"type:varchar(20);not null" json:"role"`
	Content        string    `gorm:"type:text" json:"content"`
	ToolCalls      string    `gorm:"type:text" json:"tool_calls,omitempty"`
	ToolName       string    `gorm:"type:varchar(100)" json:"tool_name,omitempty"`
	ToolCallID     string    `gorm:"type:varchar(100)" json:"tool_call_id,omitempty"`
	CreatedAt      time.Time `json:"created_at"`
	Conversation   Conversation `gorm:"foreignKey:ConversationID" json:"-"`
}

func (m *ChatMessage) BeforeCreate(tx *gorm.DB) error {
	if m.ID == "" {
		m.ID = uuid.New().String()
	}
	return nil
}

type AIUsage struct {
	ID               uint      `gorm:"primarykey" json:"id"`
	UserID           uint      `gorm:"index;not null" json:"user_id"`
	ConversationID   string    `gorm:"type:varchar(36);index" json:"conversation_id"`
	PromptTokens     int       `json:"prompt_tokens"`
	CompletionTokens int       `json:"completion_tokens"`
	CreatedAt        time.Time `json:"created_at"`
	User             User      `gorm:"foreignKey:UserID" json:"-"`
}
```

- [ ] **Step 2: Add uuid dependency**

```bash
cd /home/nst/GolandProjects/pdt/backend
go get github.com/google/uuid
```

- [ ] **Step 3: Register models in Migrate()**

In `backend/internal/database/database.go`, add the new models to `AutoMigrate`:

```go
func Migrate(db *gorm.DB) error {
	return db.AutoMigrate(
		&models.User{},
		&models.Repository{},
		&models.Commit{},
		&models.CommitCardLink{},
		&models.Sprint{},
		&models.JiraCard{},
		&models.ReportTemplate{},
		&models.Report{},
		&models.Conversation{},
		&models.ChatMessage{},
		&models.AIUsage{},
	)
}
```

- [ ] **Step 4: Add report_type, month, year to Report model**

In `backend/internal/models/report.go`, add fields to `Report`:

```go
type Report struct {
	ID         uint      `gorm:"primarykey" json:"id"`
	UserID     uint      `gorm:"index;not null" json:"user_id"`
	TemplateID *uint     `gorm:"index" json:"template_id"`
	Date       string    `gorm:"type:varchar(10);index;not null" json:"date"`
	Title      string    `gorm:"type:varchar(500)" json:"title"`
	Content    string    `gorm:"type:text" json:"content"`
	FileURL    string    `gorm:"type:varchar(500)" json:"file_url"`
	ReportType string    `gorm:"type:varchar(10);default:daily" json:"report_type"`
	Month      *int      `json:"month,omitempty"`
	Year       *int      `json:"year,omitempty"`
	CreatedAt  time.Time `json:"created_at"`
	User       User      `gorm:"foreignKey:UserID" json:"-"`
}
```

- [ ] **Step 5: Commit**

```bash
git add backend/internal/models/conversation.go backend/internal/models/report.go backend/internal/database/database.go backend/go.mod backend/go.sum
git commit -m "feat: add conversation, chat message, AI usage models and report type fields"
```

---

## Task 4: Build Agent Framework

**Files:**
- Create: `backend/internal/ai/agent/types.go`
- Create: `backend/internal/ai/agent/loop.go`
- Create: `backend/internal/ai/agent/orchestrator.go`

- [ ] **Step 1: Create agent types**

Create `backend/internal/ai/agent/types.go`:

```go
package agent

import (
	"context"
	"encoding/json"

	"github.com/cds-id/pdt/backend/internal/ai/minimax"
)

// Agent is the interface all specialist agents implement.
type Agent interface {
	Name() string
	SystemPrompt() string
	Tools() []minimax.Tool
	ExecuteTool(ctx context.Context, name string, args json.RawMessage) (any, error)
}

// StreamWriter receives streaming output from the agent loop.
type StreamWriter interface {
	WriteContent(content string) error
	WriteToolStatus(toolName string, status string) error
	WriteDone() error
	WriteError(msg string) error
}

// LoopResult contains the final state after an agent loop completes.
type LoopResult struct {
	FullResponse string
	ToolCalls    []minimax.ToolCall
	Usage        minimax.Usage
}
```

- [ ] **Step 2: Create agent loop**

Create `backend/internal/ai/agent/loop.go`:

```go
package agent

import (
	"context"
	"encoding/json"
	"fmt"
	"log"

	"github.com/cds-id/pdt/backend/internal/ai/minimax"
)

const maxToolRounds = 10

// RunLoop executes the agent tool-calling loop: send messages to MiniMax,
// stream response, execute tool calls, repeat until final text response.
func RunLoop(ctx context.Context, client *minimax.Client, agent Agent, messages []minimax.Message, writer StreamWriter) (*LoopResult, error) {
	// Prepend agent system prompt
	systemMsg := minimax.Message{
		Role:    "system",
		Content: agent.SystemPrompt(),
	}
	conversation := append([]minimax.Message{systemMsg}, messages...)
	tools := agent.Tools()

	var totalUsage minimax.Usage

	for round := 0; round < maxToolRounds; round++ {
		req := minimax.ChatRequest{
			Messages:    conversation,
			Tools:       tools,
			Temperature: 0.7,
		}

		stream, err := client.ChatStream(req)
		if err != nil {
			return nil, fmt.Errorf("chat stream: %w", err)
		}

		var fullContent string
		var toolCalls []minimax.ToolCall
		var usage *minimax.Usage

		for evt := range stream {
			if ctx.Err() != nil {
				return nil, ctx.Err()
			}
			if evt.Err != nil {
				return nil, fmt.Errorf("stream event: %w", evt.Err)
			}
			if evt.Content != "" {
				fullContent += evt.Content
				if err := writer.WriteContent(evt.Content); err != nil {
					return nil, fmt.Errorf("write content: %w", err)
				}
			}
			if len(evt.ToolCalls) > 0 {
				toolCalls = append(toolCalls, evt.ToolCalls...)
			}
			if evt.Usage != nil {
				usage = evt.Usage
			}
		}

		if usage != nil {
			totalUsage.PromptTokens += usage.PromptTokens
			totalUsage.CompletionTokens += usage.CompletionTokens
			totalUsage.TotalTokens += usage.TotalTokens
		}

		// No tool calls — final response
		if len(toolCalls) == 0 {
			return &LoopResult{
				FullResponse: fullContent,
				Usage:        totalUsage,
			}, nil
		}

		// Append assistant message with tool calls
		conversation = append(conversation, minimax.Message{
			Role:      "assistant",
			Content:   fullContent,
			ToolCalls: toolCalls,
		})

		// Execute each tool call and append results
		for _, tc := range toolCalls {
			if err := writer.WriteToolStatus(tc.Function.Name, "executing"); err != nil {
				log.Printf("[agent-loop] write tool status error: %v", err)
			}

			result, err := agent.ExecuteTool(ctx, tc.Function.Name, json.RawMessage(tc.Function.Arguments))
			if err != nil {
				result = map[string]string{"error": err.Error()}
			}

			resultJSON, _ := json.Marshal(result)

			if err := writer.WriteToolStatus(tc.Function.Name, "completed"); err != nil {
				log.Printf("[agent-loop] write tool status error: %v", err)
			}

			conversation = append(conversation, minimax.Message{
				Role:       "tool",
				Content:    string(resultJSON),
				ToolCallID: tc.ID,
			})
		}
	}

	return nil, fmt.Errorf("agent loop exceeded %d tool rounds", maxToolRounds)
}
```

- [ ] **Step 3: Create orchestrator**

Create `backend/internal/ai/agent/orchestrator.go`:

```go
package agent

import (
	"context"
	"encoding/json"
	"fmt"
	"log"

	"github.com/cds-id/pdt/backend/internal/ai/minimax"
)

// Orchestrator routes user messages to the appropriate specialist agent.
type Orchestrator struct {
	Client *minimax.Client
	Agents map[string]Agent
}

func NewOrchestrator(client *minimax.Client, agents ...Agent) *Orchestrator {
	agentMap := make(map[string]Agent)
	for _, a := range agents {
		agentMap[a.Name()] = a
	}
	return &Orchestrator{
		Client: client,
		Agents: agentMap,
	}
}

const routerSystemPrompt = `You are a routing assistant for PDT (Personal Development Tracker). Your job is to determine which specialist agent should handle the user's request.

Available agents:
- "git": Handles questions about commits, repositories, branches, and code activity.
- "jira": Handles questions about Jira sprints, cards, issues, and linking commits to cards.
- "report": Handles report generation (daily/monthly), listing reports, and report templates.

If the user's message is a simple greeting or general question not related to any agent, respond directly without routing.

For all other messages, use the route_to_agent tool to delegate to the appropriate agent.`

var routerTool = minimax.Tool{
	Type: "function",
	Function: minimax.FunctionDef{
		Name:        "route_to_agent",
		Description: "Route the user's message to a specialist agent",
		Parameters: json.RawMessage(`{
			"type": "object",
			"properties": {
				"agent_name": {
					"type": "string",
					"enum": ["git", "jira", "report"],
					"description": "The specialist agent to route to"
				},
				"reason": {
					"type": "string",
					"description": "Brief reason for routing to this agent"
				}
			},
			"required": ["agent_name", "reason"]
		}`),
	},
}

// HandleMessage routes a user message to the appropriate agent and streams the response.
func (o *Orchestrator) HandleMessage(ctx context.Context, messages []minimax.Message, writer StreamWriter) (*LoopResult, error) {
	// Ask the router which agent to use
	routerMessages := append([]minimax.Message{{
		Role:    "system",
		Content: routerSystemPrompt,
	}}, messages...)

	req := minimax.ChatRequest{
		Messages:    routerMessages,
		Tools:       []minimax.Tool{routerTool},
		Temperature: 0.3,
	}

	resp, err := o.Client.Chat(req)
	if err != nil {
		return nil, fmt.Errorf("router call: %w", err)
	}

	if len(resp.Choices) == 0 {
		return nil, fmt.Errorf("router returned no choices")
	}

	choice := resp.Choices[0]

	// If no tool calls, the router wants to respond directly (greeting, etc.)
	if len(choice.Delta.ToolCalls) == 0 {
		content := choice.Delta.Content
		if err := writer.WriteContent(content); err != nil {
			return nil, err
		}
		return &LoopResult{FullResponse: content, Usage: resp.Usage}, nil
	}

	// Parse the routing decision
	tc := choice.Delta.ToolCalls[0]
	var routing struct {
		AgentName string `json:"agent_name"`
		Reason    string `json:"reason"`
	}
	if err := json.Unmarshal([]byte(tc.Function.Arguments), &routing); err != nil {
		return nil, fmt.Errorf("parse routing: %w", err)
	}

	agent, ok := o.Agents[routing.AgentName]
	if !ok {
		return nil, fmt.Errorf("unknown agent: %s", routing.AgentName)
	}

	log.Printf("[orchestrator] routing to %s: %s", routing.AgentName, routing.Reason)

	// Run the specialist agent's tool-calling loop
	return RunLoop(ctx, o.Client, agent, messages, writer)
}
```

- [ ] **Step 4: Commit**

```bash
git add backend/internal/ai/agent/
git commit -m "feat: add agent framework with orchestrator, tool-calling loop, and types"
```

---

## Task 5: Build Specialist Agents

**Files:**
- Create: `backend/internal/ai/agent/git.go`
- Create: `backend/internal/ai/agent/jira.go`
- Create: `backend/internal/ai/agent/report.go`

- [ ] **Step 1: Create Git Agent**

Create `backend/internal/ai/agent/git.go`:

```go
package agent

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/cds-id/pdt/backend/internal/ai/minimax"
	"github.com/cds-id/pdt/backend/internal/models"
	"gorm.io/gorm"
)

type GitAgent struct {
	DB     *gorm.DB
	UserID uint
}

func (a *GitAgent) Name() string { return "git" }

func (a *GitAgent) SystemPrompt() string {
	return `You are a Git assistant for PDT. You help users explore their commit history, repository statistics, and code activity. Use the available tools to fetch data and provide insightful answers. Always be specific with numbers and dates.`
}

func (a *GitAgent) Tools() []minimax.Tool {
	return []minimax.Tool{
		{
			Type: "function",
			Function: minimax.FunctionDef{
				Name:        "search_commits",
				Description: "Search commits by message keyword, author, repo, or date range",
				Parameters: json.RawMessage(`{
					"type": "object",
					"properties": {
						"keyword": {"type": "string", "description": "Search keyword in commit message"},
						"repo": {"type": "string", "description": "Repository name filter"},
						"since": {"type": "string", "description": "Start date (YYYY-MM-DD)"},
						"until": {"type": "string", "description": "End date (YYYY-MM-DD)"},
						"limit": {"type": "integer", "description": "Max results (default 20)"}
					}
				}`),
			},
		},
		{
			Type: "function",
			Function: minimax.FunctionDef{
				Name:        "list_repos",
				Description: "List all tracked repositories for the user",
				Parameters: json.RawMessage(`{"type": "object", "properties": {}}`),
			},
		},
		{
			Type: "function",
			Function: minimax.FunctionDef{
				Name:        "get_repo_stats",
				Description: "Get commit statistics for a specific repository",
				Parameters: json.RawMessage(`{
					"type": "object",
					"properties": {
						"repo": {"type": "string", "description": "Repository name"},
						"days": {"type": "integer", "description": "Number of days to look back (default 30)"}
					},
					"required": ["repo"]
				}`),
			},
		},
		{
			Type: "function",
			Function: minimax.FunctionDef{
				Name:        "get_commit_detail",
				Description: "Get detailed information about a specific commit by SHA",
				Parameters: json.RawMessage(`{
					"type": "object",
					"properties": {
						"sha": {"type": "string", "description": "Commit SHA (full or short)"}
					},
					"required": ["sha"]
				}`),
			},
		},
	}
}

func (a *GitAgent) ExecuteTool(ctx context.Context, name string, args json.RawMessage) (any, error) {
	switch name {
	case "search_commits":
		return a.searchCommits(args)
	case "list_repos":
		return a.listRepos()
	case "get_repo_stats":
		return a.getRepoStats(args)
	case "get_commit_detail":
		return a.getCommitDetail(args)
	default:
		return nil, fmt.Errorf("unknown tool: %s", name)
	}
}

func (a *GitAgent) searchCommits(args json.RawMessage) (any, error) {
	var params struct {
		Keyword string `json:"keyword"`
		Repo    string `json:"repo"`
		Since   string `json:"since"`
		Until   string `json:"until"`
		Limit   int    `json:"limit"`
	}
	json.Unmarshal(args, &params)
	if params.Limit == 0 {
		params.Limit = 20
	}

	query := a.DB.Joins("JOIN repositories ON repositories.id = commits.repo_id").
		Where("repositories.user_id = ?", a.UserID).
		Preload("Repository")

	if params.Keyword != "" {
		query = query.Where("commits.message LIKE ?", "%"+params.Keyword+"%")
	}
	if params.Repo != "" {
		query = query.Where("repositories.name = ?", params.Repo)
	}
	if params.Since != "" {
		if t, err := time.Parse("2006-01-02", params.Since); err == nil {
			query = query.Where("commits.date >= ?", t)
		}
	}
	if params.Until != "" {
		if t, err := time.Parse("2006-01-02", params.Until); err == nil {
			query = query.Where("commits.date < ?", t.Add(24*time.Hour))
		}
	}

	var commits []models.Commit
	query.Order("commits.date desc").Limit(params.Limit).Find(&commits)

	type result struct {
		SHA     string `json:"sha"`
		Message string `json:"message"`
		Author  string `json:"author"`
		Date    string `json:"date"`
		Repo    string `json:"repo"`
		Branch  string `json:"branch"`
		JiraKey string `json:"jira_key,omitempty"`
	}
	var results []result
	for _, c := range commits {
		repoName := ""
		if c.Repository.Name != "" {
			repoName = c.Repository.Owner + "/" + c.Repository.Name
		}
		results = append(results, result{
			SHA:     c.SHA[:8],
			Message: c.Message,
			Author:  c.Author,
			Date:    c.Date.Format("2006-01-02 15:04"),
			Repo:    repoName,
			Branch:  c.Branch,
			JiraKey: c.JiraCardKey,
		})
	}
	return results, nil
}

func (a *GitAgent) listRepos() (any, error) {
	var repos []models.Repository
	a.DB.Where("user_id = ?", a.UserID).Find(&repos)

	type result struct {
		Name     string `json:"name"`
		Owner    string `json:"owner"`
		Provider string `json:"provider"`
		URL      string `json:"url"`
	}
	var results []result
	for _, r := range repos {
		results = append(results, result{
			Name:     r.Name,
			Owner:    r.Owner,
			Provider: r.Provider,
			URL:      r.URL,
		})
	}
	return results, nil
}

func (a *GitAgent) getRepoStats(args json.RawMessage) (any, error) {
	var params struct {
		Repo string `json:"repo"`
		Days int    `json:"days"`
	}
	json.Unmarshal(args, &params)
	if params.Days == 0 {
		params.Days = 30
	}

	since := time.Now().AddDate(0, 0, -params.Days)

	var repo models.Repository
	if err := a.DB.Where("user_id = ? AND name = ?", a.UserID, params.Repo).First(&repo).Error; err != nil {
		return nil, fmt.Errorf("repository not found: %s", params.Repo)
	}

	var totalCommits int64
	a.DB.Model(&models.Commit{}).Where("repo_id = ? AND date >= ?", repo.ID, since).Count(&totalCommits)

	var linkedCommits int64
	a.DB.Model(&models.Commit{}).Where("repo_id = ? AND date >= ? AND has_link = ?", repo.ID, since, true).Count(&linkedCommits)

	type branchStat struct {
		Branch string `json:"branch"`
		Count  int64  `json:"count"`
	}
	var branches []branchStat
	a.DB.Model(&models.Commit{}).
		Select("branch, count(*) as count").
		Where("repo_id = ? AND date >= ?", repo.ID, since).
		Group("branch").Order("count desc").Limit(10).
		Scan(&branches)

	return map[string]any{
		"repo":            params.Repo,
		"period_days":     params.Days,
		"total_commits":   totalCommits,
		"linked_to_jira":  linkedCommits,
		"top_branches":    branches,
	}, nil
}

func (a *GitAgent) getCommitDetail(args json.RawMessage) (any, error) {
	var params struct {
		SHA string `json:"sha"`
	}
	json.Unmarshal(args, &params)

	var commit models.Commit
	if err := a.DB.Joins("JOIN repositories ON repositories.id = commits.repo_id").
		Where("repositories.user_id = ? AND commits.sha LIKE ?", a.UserID, params.SHA+"%").
		Preload("Repository").
		First(&commit).Error; err != nil {
		return nil, fmt.Errorf("commit not found: %s", params.SHA)
	}

	return map[string]any{
		"sha":          commit.SHA,
		"message":      commit.Message,
		"author":       commit.Author,
		"author_email": commit.AuthorEmail,
		"date":         commit.Date.Format("2006-01-02 15:04:05"),
		"branch":       commit.Branch,
		"repo":         commit.Repository.Owner + "/" + commit.Repository.Name,
		"jira_key":     commit.JiraCardKey,
		"has_link":     commit.HasLink,
	}, nil
}
```

- [ ] **Step 2: Create Jira Agent**

Create `backend/internal/ai/agent/jira.go`:

```go
package agent

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/cds-id/pdt/backend/internal/ai/minimax"
	"github.com/cds-id/pdt/backend/internal/helpers"
	"github.com/cds-id/pdt/backend/internal/models"
	"gorm.io/gorm"
)

type JiraAgent struct {
	DB     *gorm.DB
	UserID uint
}

func (a *JiraAgent) Name() string { return "jira" }

func (a *JiraAgent) SystemPrompt() string {
	return `You are a Jira assistant for PDT. You help users explore their Jira sprints, cards, and issues. You can also link commits to Jira cards. Use the available tools to fetch data and provide helpful answers.`
}

func (a *JiraAgent) Tools() []minimax.Tool {
	return []minimax.Tool{
		{
			Type: "function",
			Function: minimax.FunctionDef{
				Name:        "get_sprints",
				Description: "List all synced Jira sprints, optionally filtered by state",
				Parameters: json.RawMessage(`{
					"type": "object",
					"properties": {
						"state": {"type": "string", "enum": ["active", "closed", "future"], "description": "Filter by sprint state"}
					}
				}`),
			},
		},
		{
			Type: "function",
			Function: minimax.FunctionDef{
				Name:        "get_cards",
				Description: "List Jira cards, optionally filtered by sprint or status",
				Parameters: json.RawMessage(`{
					"type": "object",
					"properties": {
						"sprint_id": {"type": "integer", "description": "Filter by sprint ID"},
						"status": {"type": "string", "description": "Filter by card status (e.g., 'Done', 'In Progress')"},
						"keyword": {"type": "string", "description": "Search keyword in card summary"},
						"limit": {"type": "integer", "description": "Max results (default 30)"}
					}
				}`),
			},
		},
		{
			Type: "function",
			Function: minimax.FunctionDef{
				Name:        "get_card_detail",
				Description: "Get detailed information about a specific Jira card by key",
				Parameters: json.RawMessage(`{
					"type": "object",
					"properties": {
						"key": {"type": "string", "description": "Jira card key (e.g., PDT-123)"}
					},
					"required": ["key"]
				}`),
			},
		},
		{
			Type: "function",
			Function: minimax.FunctionDef{
				Name:        "search_cards",
				Description: "Search Jira cards by keyword across all sprints",
				Parameters: json.RawMessage(`{
					"type": "object",
					"properties": {
						"keyword": {"type": "string", "description": "Search keyword"},
						"limit": {"type": "integer", "description": "Max results (default 20)"}
					},
					"required": ["keyword"]
				}`),
			},
		},
		{
			Type: "function",
			Function: minimax.FunctionDef{
				Name:        "link_commit_to_card",
				Description: "Link a commit to a Jira card by SHA and card key",
				Parameters: json.RawMessage(`{
					"type": "object",
					"properties": {
						"sha": {"type": "string", "description": "Commit SHA (full or short)"},
						"card_key": {"type": "string", "description": "Jira card key (e.g., PDT-123)"}
					},
					"required": ["sha", "card_key"]
				}`),
			},
		},
	}
}

func (a *JiraAgent) ExecuteTool(ctx context.Context, name string, args json.RawMessage) (any, error) {
	switch name {
	case "get_sprints":
		return a.getSprints(args)
	case "get_cards":
		return a.getCards(args)
	case "get_card_detail":
		return a.getCardDetail(args)
	case "search_cards":
		return a.searchCards(args)
	case "link_commit_to_card":
		return a.linkCommitToCard(args)
	default:
		return nil, fmt.Errorf("unknown tool: %s", name)
	}
}

func (a *JiraAgent) getSprints(args json.RawMessage) (any, error) {
	var params struct {
		State string `json:"state"`
	}
	json.Unmarshal(args, &params)

	query := a.DB.Where("user_id = ?", a.UserID)
	if params.State != "" {
		query = query.Where("state = ?", params.State)
	}

	var sprints []models.Sprint
	query.Order("start_date desc").Find(&sprints)

	type result struct {
		ID        uint   `json:"id"`
		Name      string `json:"name"`
		State     string `json:"state"`
		StartDate string `json:"start_date,omitempty"`
		EndDate   string `json:"end_date,omitempty"`
		CardCount int64  `json:"card_count"`
	}
	var results []result
	for _, s := range sprints {
		var count int64
		a.DB.Model(&models.JiraCard{}).Where("sprint_id = ?", s.ID).Count(&count)
		r := result{
			ID:        s.ID,
			Name:      s.Name,
			State:     string(s.State),
			CardCount: count,
		}
		if s.StartDate != nil {
			r.StartDate = s.StartDate.Format("2006-01-02")
		}
		if s.EndDate != nil {
			r.EndDate = s.EndDate.Format("2006-01-02")
		}
		results = append(results, r)
	}
	return results, nil
}

func (a *JiraAgent) getCards(args json.RawMessage) (any, error) {
	var params struct {
		SprintID int    `json:"sprint_id"`
		Status   string `json:"status"`
		Keyword  string `json:"keyword"`
		Limit    int    `json:"limit"`
	}
	json.Unmarshal(args, &params)
	if params.Limit == 0 {
		params.Limit = 30
	}

	// Get user for project key filtering
	var user models.User
	a.DB.First(&user, a.UserID)

	query := a.DB.Where("user_id = ?", a.UserID)
	if params.SprintID > 0 {
		query = query.Where("sprint_id = ?", params.SprintID)
	}
	if params.Status != "" {
		query = query.Where("status = ?", params.Status)
	}
	if params.Keyword != "" {
		query = query.Where("summary LIKE ?", "%"+params.Keyword+"%")
	}

	// Apply project key filter
	if clause, filterArgs := helpers.BuildProjectKeyWhereClauses(user.JiraProjectKeys, "card_key"); clause != "" {
		query = query.Where(clause, filterArgs...)
	}

	var cards []models.JiraCard
	query.Order("created_at desc").Limit(params.Limit).Find(&cards)

	type result struct {
		Key      string `json:"key"`
		Summary  string `json:"summary"`
		Status   string `json:"status"`
		Assignee string `json:"assignee"`
	}
	var results []result
	for _, c := range cards {
		results = append(results, result{
			Key:      c.Key,
			Summary:  c.Summary,
			Status:   c.Status,
			Assignee: c.Assignee,
		})
	}
	return results, nil
}

func (a *JiraAgent) getCardDetail(args json.RawMessage) (any, error) {
	var params struct {
		Key string `json:"key"`
	}
	json.Unmarshal(args, &params)

	var card models.JiraCard
	if err := a.DB.Where("user_id = ? AND card_key = ?", a.UserID, params.Key).First(&card).Error; err != nil {
		return nil, fmt.Errorf("card not found: %s", params.Key)
	}

	// Also fetch linked commits
	var commits []models.Commit
	a.DB.Joins("JOIN repositories ON repositories.id = commits.repo_id").
		Where("repositories.user_id = ? AND commits.jira_card_key = ?", a.UserID, params.Key).
		Find(&commits)

	type commitInfo struct {
		SHA     string `json:"sha"`
		Message string `json:"message"`
		Date    string `json:"date"`
	}
	var linkedCommits []commitInfo
	for _, c := range commits {
		linkedCommits = append(linkedCommits, commitInfo{
			SHA:     c.SHA[:8],
			Message: c.Message,
			Date:    c.Date.Format("2006-01-02 15:04"),
		})
	}

	result := map[string]any{
		"key":      card.Key,
		"summary":  card.Summary,
		"status":   card.Status,
		"assignee": card.Assignee,
		"commits":  linkedCommits,
	}
	if card.DetailsJSON != "" {
		var details any
		json.Unmarshal([]byte(card.DetailsJSON), &details)
		result["details"] = details
	}
	return result, nil
}

func (a *JiraAgent) searchCards(args json.RawMessage) (any, error) {
	var params struct {
		Keyword string `json:"keyword"`
		Limit   int    `json:"limit"`
	}
	json.Unmarshal(args, &params)
	if params.Limit == 0 {
		params.Limit = 20
	}

	var cards []models.JiraCard
	a.DB.Where("user_id = ? AND (card_key LIKE ? OR summary LIKE ?)",
		a.UserID, "%"+params.Keyword+"%", "%"+params.Keyword+"%").
		Limit(params.Limit).Find(&cards)

	type result struct {
		Key     string `json:"key"`
		Summary string `json:"summary"`
		Status  string `json:"status"`
	}
	var results []result
	for _, c := range cards {
		results = append(results, result{Key: c.Key, Summary: c.Summary, Status: c.Status})
	}
	return results, nil
}

func (a *JiraAgent) linkCommitToCard(args json.RawMessage) (any, error) {
	var params struct {
		SHA     string `json:"sha"`
		CardKey string `json:"card_key"`
	}
	json.Unmarshal(args, &params)

	var commit models.Commit
	if err := a.DB.Joins("JOIN repositories ON repositories.id = commits.repo_id").
		Where("repositories.user_id = ? AND commits.sha LIKE ?", a.UserID, params.SHA+"%").
		First(&commit).Error; err != nil {
		return nil, fmt.Errorf("commit not found: %s", params.SHA)
	}

	commit.JiraCardKey = params.CardKey
	commit.HasLink = true
	a.DB.Save(&commit)

	return map[string]string{
		"status":  "linked",
		"sha":     commit.SHA[:8],
		"card_key": params.CardKey,
	}, nil
}
```

- [ ] **Step 3: Create Report Agent**

Create `backend/internal/ai/agent/report.go`:

```go
package agent

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/cds-id/pdt/backend/internal/ai/minimax"
	"github.com/cds-id/pdt/backend/internal/models"
	"github.com/cds-id/pdt/backend/internal/services/report"
	"github.com/cds-id/pdt/backend/internal/services/storage"
	"gorm.io/gorm"
)

type ReportAgent struct {
	DB        *gorm.DB
	UserID    uint
	Generator *report.Generator
	R2        *storage.R2Client
}

func (a *ReportAgent) Name() string { return "report" }

func (a *ReportAgent) SystemPrompt() string {
	return `You are a Report assistant for PDT. You help users generate daily and monthly reports, view existing reports, and manage report templates. Use the available tools to fetch and generate reports. When generating reports, confirm the date/month with the user first.`
}

func (a *ReportAgent) Tools() []minimax.Tool {
	return []minimax.Tool{
		{
			Type: "function",
			Function: minimax.FunctionDef{
				Name:        "generate_daily_report",
				Description: "Generate a daily development report for a specific date",
				Parameters: json.RawMessage(`{
					"type": "object",
					"properties": {
						"date": {"type": "string", "description": "Date in YYYY-MM-DD format (default: today)"}
					}
				}`),
			},
		},
		{
			Type: "function",
			Function: minimax.FunctionDef{
				Name:        "generate_monthly_report",
				Description: "Generate a monthly report with aggregated stats and AI narrative",
				Parameters: json.RawMessage(`{
					"type": "object",
					"properties": {
						"month": {"type": "integer", "description": "Month number (1-12)"},
						"year": {"type": "integer", "description": "Year (e.g., 2026)"}
					},
					"required": ["month", "year"]
				}`),
			},
		},
		{
			Type: "function",
			Function: minimax.FunctionDef{
				Name:        "list_reports",
				Description: "List existing reports, optionally filtered by type",
				Parameters: json.RawMessage(`{
					"type": "object",
					"properties": {
						"report_type": {"type": "string", "enum": ["daily", "monthly"], "description": "Filter by report type"},
						"limit": {"type": "integer", "description": "Max results (default 20)"}
					}
				}`),
			},
		},
		{
			Type: "function",
			Function: minimax.FunctionDef{
				Name:        "get_report",
				Description: "Get the full content of a specific report by ID",
				Parameters: json.RawMessage(`{
					"type": "object",
					"properties": {
						"id": {"type": "integer", "description": "Report ID"}
					},
					"required": ["id"]
				}`),
			},
		},
		{
			Type: "function",
			Function: minimax.FunctionDef{
				Name:        "preview_template",
				Description: "Preview a report template rendered with today's data",
				Parameters: json.RawMessage(`{
					"type": "object",
					"properties": {
						"template_id": {"type": "integer", "description": "Template ID to preview"}
					},
					"required": ["template_id"]
				}`),
			},
		},
	}
}

func (a *ReportAgent) ExecuteTool(ctx context.Context, name string, args json.RawMessage) (any, error) {
	switch name {
	case "generate_daily_report":
		return a.generateDaily(args)
	case "generate_monthly_report":
		return a.generateMonthly(args)
	case "list_reports":
		return a.listReports(args)
	case "get_report":
		return a.getReport(args)
	case "preview_template":
		return a.previewTemplate(args)
	default:
		return nil, fmt.Errorf("unknown tool: %s", name)
	}
}

func (a *ReportAgent) generateDaily(args json.RawMessage) (any, error) {
	var params struct {
		Date string `json:"date"`
	}
	json.Unmarshal(args, &params)
	if params.Date == "" {
		params.Date = time.Now().Format("2006-01-02")
	}

	date, err := time.Parse("2006-01-02", params.Date)
	if err != nil {
		return nil, fmt.Errorf("invalid date format: %s", params.Date)
	}

	data, err := a.Generator.BuildReportData(a.UserID, date)
	if err != nil {
		return nil, fmt.Errorf("build report data: %w", err)
	}

	templateContent, templateID := a.Generator.GetTemplateContent(a.UserID, nil)
	rendered, err := a.Generator.Render(templateContent, data)
	if err != nil {
		return nil, fmt.Errorf("render report: %w", err)
	}

	// Upsert report
	var existing models.Report
	rpt := models.Report{
		UserID:     a.UserID,
		TemplateID: templateID,
		Date:       params.Date,
		Title:      fmt.Sprintf("Daily Report — %s", data.DateFormatted),
		Content:    rendered,
		ReportType: "daily",
	}

	if a.DB.Where("user_id = ? AND date = ? AND report_type = ?", a.UserID, params.Date, "daily").First(&existing).Error == nil {
		existing.Content = rendered
		existing.Title = rpt.Title
		a.DB.Save(&existing)
		rpt.ID = existing.ID
	} else {
		a.DB.Create(&rpt)
	}

	return map[string]any{
		"id":      rpt.ID,
		"title":   rpt.Title,
		"date":    params.Date,
		"content": rendered,
		"stats": map[string]any{
			"total_commits": data.Stats.TotalCommits,
			"total_cards":   data.Stats.TotalCards,
		},
	}, nil
}

func (a *ReportAgent) generateMonthly(args json.RawMessage) (any, error) {
	var params struct {
		Month int `json:"month"`
		Year  int `json:"year"`
	}
	json.Unmarshal(args, &params)

	data, err := a.Generator.BuildMonthlyReportData(a.UserID, params.Month, params.Year)
	if err != nil {
		return nil, fmt.Errorf("build monthly report data: %w", err)
	}

	templateContent := a.Generator.GetMonthlyTemplateContent(a.UserID)
	rendered, err := a.Generator.Render(templateContent, data)
	if err != nil {
		return nil, fmt.Errorf("render monthly report: %w", err)
	}

	month := params.Month
	year := params.Year
	dateStr := fmt.Sprintf("%04d-%02d", year, month)
	rpt := models.Report{
		UserID:     a.UserID,
		Date:       dateStr,
		Title:      fmt.Sprintf("Monthly Report — %s %d", time.Month(month).String(), year),
		Content:    rendered,
		ReportType: "monthly",
		Month:      &month,
		Year:       &year,
	}

	var existing models.Report
	if a.DB.Where("user_id = ? AND report_type = ? AND month = ? AND year = ?", a.UserID, "monthly", month, year).First(&existing).Error == nil {
		existing.Content = rendered
		existing.Title = rpt.Title
		a.DB.Save(&existing)
		rpt.ID = existing.ID
	} else {
		a.DB.Create(&rpt)
	}

	return map[string]any{
		"id":      rpt.ID,
		"title":   rpt.Title,
		"month":   month,
		"year":    year,
		"content": rendered,
	}, nil
}

func (a *ReportAgent) listReports(args json.RawMessage) (any, error) {
	var params struct {
		ReportType string `json:"report_type"`
		Limit      int    `json:"limit"`
	}
	json.Unmarshal(args, &params)
	if params.Limit == 0 {
		params.Limit = 20
	}

	query := a.DB.Where("user_id = ?", a.UserID)
	if params.ReportType != "" {
		query = query.Where("report_type = ?", params.ReportType)
	}

	var reports []models.Report
	query.Order("created_at desc").Limit(params.Limit).Find(&reports)

	type result struct {
		ID         uint   `json:"id"`
		Title      string `json:"title"`
		Date       string `json:"date"`
		ReportType string `json:"report_type"`
	}
	var results []result
	for _, r := range reports {
		results = append(results, result{
			ID:         r.ID,
			Title:      r.Title,
			Date:       r.Date,
			ReportType: r.ReportType,
		})
	}
	return results, nil
}

func (a *ReportAgent) getReport(args json.RawMessage) (any, error) {
	var params struct {
		ID uint `json:"id"`
	}
	json.Unmarshal(args, &params)

	var rpt models.Report
	if err := a.DB.Where("id = ? AND user_id = ?", params.ID, a.UserID).First(&rpt).Error; err != nil {
		return nil, fmt.Errorf("report not found: %d", params.ID)
	}
	return map[string]any{
		"id":          rpt.ID,
		"title":       rpt.Title,
		"date":        rpt.Date,
		"report_type": rpt.ReportType,
		"content":     rpt.Content,
	}, nil
}

func (a *ReportAgent) previewTemplate(args json.RawMessage) (any, error) {
	var params struct {
		TemplateID uint `json:"template_id"`
	}
	json.Unmarshal(args, &params)

	var tmpl models.ReportTemplate
	if err := a.DB.Where("id = ? AND user_id = ?", params.TemplateID, a.UserID).First(&tmpl).Error; err != nil {
		return nil, fmt.Errorf("template not found: %d", params.TemplateID)
	}

	data, err := a.Generator.BuildReportData(a.UserID, time.Now())
	if err != nil {
		return nil, fmt.Errorf("build preview data: %w", err)
	}

	rendered, err := a.Generator.Render(tmpl.Content, data)
	if err != nil {
		return nil, fmt.Errorf("render preview: %w", err)
	}

	return map[string]any{
		"template_name": tmpl.Name,
		"preview":       rendered,
	}, nil
}
```

- [ ] **Step 4: Commit**

```bash
git add backend/internal/ai/agent/git.go backend/internal/ai/agent/jira.go backend/internal/ai/agent/report.go
git commit -m "feat: add Git, Jira, and Report specialist agents with tools"
```

---

## Task 6: Add Monthly Report Generation to Report Service

**Files:**
- Modify: `backend/internal/services/report/report.go`

- [ ] **Step 1: Add MonthlyReportData and BuildMonthlyReportData**

In `backend/internal/services/report/report.go`, add the following after the existing `BuildReportData` function:

```go
type MonthlyReportData struct {
	Month          int
	Year           int
	MonthName      string
	Author         string
	TotalCommits   int
	TotalCards     int
	CardsCompleted int
	CardsInProgress int
	WeeklyBreakdown []WeekStats
	RepoBreakdown   []RepoStats
	TopCards        []CardReport
	DailyReports   []DailyReportSummary
}

type WeekStats struct {
	WeekNumber int    `json:"week_number"`
	StartDate  string `json:"start_date"`
	EndDate    string `json:"end_date"`
	Commits    int    `json:"commits"`
	Cards      int    `json:"cards"`
}

type RepoStats struct {
	Repo    string `json:"repo"`
	Commits int    `json:"commits"`
}

type DailyReportSummary struct {
	Date    string `json:"date"`
	Title   string `json:"title"`
	Commits int    `json:"commits"`
	Cards   int    `json:"cards"`
}

func (g *Generator) BuildMonthlyReportData(userID uint, month, year int) (*MonthlyReportData, error) {
	var user models.User
	if err := g.DB.First(&user, userID).Error; err != nil {
		return nil, fmt.Errorf("user not found: %w", err)
	}

	monthStart := time.Date(year, time.Month(month), 1, 0, 0, 0, 0, time.Local)
	monthEnd := monthStart.AddDate(0, 1, 0)

	// Fetch all commits for the month
	var commits []models.Commit
	g.DB.Joins("JOIN repositories ON repositories.id = commits.repo_id").
		Where("repositories.user_id = ? AND commits.date >= ? AND commits.date < ?", userID, monthStart, monthEnd).
		Preload("Repository").
		Order("commits.date asc").
		Find(&commits)

	// Fetch all cards worked on (cards with commits this month)
	cardKeys := make(map[string]bool)
	for _, c := range commits {
		if c.JiraCardKey != "" {
			cardKeys[c.JiraCardKey] = true
		}
	}

	var cards []models.JiraCard
	if len(cardKeys) > 0 {
		keys := make([]string, 0, len(cardKeys))
		for k := range cardKeys {
			keys = append(keys, k)
		}
		g.DB.Where("user_id = ? AND card_key IN ?", userID, keys).Find(&cards)
	}

	// Build weekly breakdown
	var weeklyBreakdown []WeekStats
	current := monthStart
	weekNum := 1
	for current.Before(monthEnd) {
		weekEnd := current.AddDate(0, 0, 7)
		if weekEnd.After(monthEnd) {
			weekEnd = monthEnd
		}
		weekCommits := 0
		weekCards := make(map[string]bool)
		for _, c := range commits {
			if !c.Date.Before(current) && c.Date.Before(weekEnd) {
				weekCommits++
				if c.JiraCardKey != "" {
					weekCards[c.JiraCardKey] = true
				}
			}
		}
		weeklyBreakdown = append(weeklyBreakdown, WeekStats{
			WeekNumber: weekNum,
			StartDate:  current.Format("2006-01-02"),
			EndDate:    weekEnd.AddDate(0, 0, -1).Format("2006-01-02"),
			Commits:    weekCommits,
			Cards:      len(weekCards),
		})
		current = weekEnd
		weekNum++
	}

	// Build repo breakdown
	repoMap := make(map[string]int)
	for _, c := range commits {
		repoName := c.Repository.Owner + "/" + c.Repository.Name
		repoMap[repoName]++
	}
	var repoBreakdown []RepoStats
	for repo, count := range repoMap {
		repoBreakdown = append(repoBreakdown, RepoStats{Repo: repo, Commits: count})
	}

	// Build top cards by commit count
	cardCommitCount := make(map[string]int)
	for _, c := range commits {
		if c.JiraCardKey != "" {
			cardCommitCount[c.JiraCardKey]++
		}
	}
	var topCards []CardReport
	for _, card := range cards {
		topCards = append(topCards, CardReport{
			Key:     card.Key,
			Summary: card.Summary,
			Status:  card.Status,
		})
	}

	// Count completed and in-progress
	completed := 0
	inProgress := 0
	for _, card := range cards {
		if card.Status == "Done" || card.Status == "Closed" {
			completed++
		} else {
			inProgress++
		}
	}

	// Fetch daily reports for the month
	var dailyReports []models.Report
	g.DB.Where("user_id = ? AND report_type = ? AND date >= ? AND date < ?",
		userID, "daily", monthStart.Format("2006-01-02"), monthEnd.Format("2006-01-02")).
		Order("date asc").Find(&dailyReports)

	var dailySummaries []DailyReportSummary
	for _, r := range dailyReports {
		dailySummaries = append(dailySummaries, DailyReportSummary{
			Date:  r.Date,
			Title: r.Title,
		})
	}

	return &MonthlyReportData{
		Month:           month,
		Year:            year,
		MonthName:       time.Month(month).String(),
		Author:          user.Email,
		TotalCommits:    len(commits),
		TotalCards:      len(cards),
		CardsCompleted:  completed,
		CardsInProgress: inProgress,
		WeeklyBreakdown: weeklyBreakdown,
		RepoBreakdown:   repoBreakdown,
		TopCards:        topCards,
		DailyReports:    dailySummaries,
	}, nil
}

const DefaultMonthlyTemplate = `# Monthly Report — {{.MonthName}} {{.Year}}

**Author:** {{.Author}}

## Summary
- **Total Commits:** {{.TotalCommits}}
- **Total Jira Cards Worked On:** {{.TotalCards}}
- **Cards Completed:** {{.CardsCompleted}}
- **Cards In Progress:** {{.CardsInProgress}}

## Weekly Breakdown
{{range .WeeklyBreakdown}}
### Week {{.WeekNumber}} ({{.StartDate}} — {{.EndDate}})
- Commits: {{.Commits}}
- Cards: {{.Cards}}
{{end}}

## Repository Activity
{{range .RepoBreakdown}}
- **{{.Repo}}**: {{.Commits}} commits
{{end}}

## Top Jira Cards
{{range .TopCards}}
- **{{.Key}}** — {{.Summary}} ({{.Status}})
{{end}}
`

func (g *Generator) GetMonthlyTemplateContent(userID uint) string {
	var tmpl models.ReportTemplate
	if g.DB.Where("user_id = ? AND name = ? AND is_default = ?", userID, "Monthly Default", true).First(&tmpl).Error == nil {
		return tmpl.Content
	}
	return DefaultMonthlyTemplate
}
```

- [ ] **Step 2: Add import for `time` if not already present**

Verify `time` is already imported in the file. It should be since `BuildReportData` uses it.

- [ ] **Step 3: Commit**

```bash
git add backend/internal/services/report/report.go
git commit -m "feat: add monthly report data aggregation and default template"
```

---

## Task 7: Add Monthly Report Endpoint and Worker

**Files:**
- Modify: `backend/internal/handlers/report.go`
- Modify: `backend/internal/worker/scheduler.go`
- Modify: `backend/internal/worker/reports.go`
- Modify: `backend/internal/config/config.go`

- [ ] **Step 1: Add GenerateMonthly handler**

In `backend/internal/handlers/report.go`, add:

```go
func (h *ReportHandler) GenerateMonthly(c *gin.Context) {
	userID := c.GetUint("user_id")

	var req struct {
		Month int `json:"month" binding:"required"`
		Year  int `json:"year" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "month and year are required"})
		return
	}

	data, err := h.Generator.BuildMonthlyReportData(userID, req.Month, req.Year)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	templateContent := h.Generator.GetMonthlyTemplateContent(userID)
	rendered, err := h.Generator.Render(templateContent, data)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	dateStr := fmt.Sprintf("%04d-%02d", req.Year, req.Month)
	fileURL := h.uploadToR2(userID, "monthly-"+dateStr, rendered)

	month := req.Month
	year := req.Year
	rpt := models.Report{
		UserID:     userID,
		Date:       dateStr,
		Title:      fmt.Sprintf("Monthly Report — %s %d", time.Month(month).String(), year),
		Content:    rendered,
		FileURL:    fileURL,
		ReportType: "monthly",
		Month:      &month,
		Year:       &year,
	}

	var existing models.Report
	if h.DB.Where("user_id = ? AND report_type = ? AND month = ? AND year = ?", userID, "monthly", month, year).First(&existing).Error == nil {
		existing.Content = rendered
		existing.Title = rpt.Title
		existing.FileURL = fileURL
		h.DB.Save(&existing)
		c.JSON(http.StatusOK, existing)
		return
	}

	if err := h.DB.Create(&rpt).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusCreated, rpt)
}
```

- [ ] **Step 2: Add monthly report auto-generation config**

In `backend/internal/config/config.go`, add to `Config` struct:

```go
ReportMonthlyAutoTime string
```

In `Load()`:

```go
cfg.ReportMonthlyAutoTime = getEnv("REPORT_MONTHLY_AUTO_TIME", "08:00")
```

- [ ] **Step 3: Add monthly report worker logic**

In `backend/internal/worker/reports.go`, add a function for monthly report generation:

```go
func GenerateMonthlyReportForUser(db *gorm.DB, enc *crypto.Encryptor, r2 *storage.R2Client, userID uint, month, year int) error {
	generator := report.NewGenerator(db, enc)
	data, err := generator.BuildMonthlyReportData(userID, month, year)
	if err != nil {
		return fmt.Errorf("build monthly data: %w", err)
	}

	if data.TotalCommits == 0 {
		log.Printf("[report-worker] user=%d no activity in %d-%02d, skipping monthly report", userID, year, month)
		return nil
	}

	templateContent := generator.GetMonthlyTemplateContent(userID)
	rendered, err := generator.Render(templateContent, data)
	if err != nil {
		return fmt.Errorf("render monthly: %w", err)
	}

	m := month
	y := year
	dateStr := fmt.Sprintf("%04d-%02d", year, month)
	rpt := models.Report{
		UserID:     userID,
		Date:       dateStr,
		Title:      fmt.Sprintf("Monthly Report — %s %d", time.Month(month).String(), year),
		Content:    rendered,
		ReportType: "monthly",
		Month:      &m,
		Year:       &y,
	}

	var existing models.Report
	if db.Where("user_id = ? AND report_type = ? AND month = ? AND year = ?", userID, "monthly", month, year).First(&existing).Error == nil {
		existing.Content = rendered
		existing.Title = rpt.Title
		db.Save(&existing)
	} else {
		db.Create(&rpt)
	}

	log.Printf("[report-worker] user=%d monthly report generated for %d-%02d", userID, year, month)
	return nil
}
```

- [ ] **Step 4: Add monthly report loop to scheduler**

In `backend/internal/worker/scheduler.go`, add a `lastMonthlyReportMonth` field and a `monthlyReportLoop` method:

Add to `Scheduler` struct:
```go
ReportMonthlyAutoTime string
lastMonthlyReport     string
monthlyRunning        atomic.Bool
```

Add to `Start()`:
```go
if s.ReportAutoGenerate {
	go s.reportLoop(ctx)
	go s.monthlyReportLoop(ctx)
}
```

Add the loop method:
```go
func (s *Scheduler) monthlyReportLoop(ctx context.Context) {
	ticker := time.NewTicker(1 * time.Minute)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			now := time.Now()
			// Only run on the 1st of the month
			if now.Day() != 1 {
				continue
			}
			currentKey := now.Format("2006-01")
			if currentKey == s.lastMonthlyReport {
				continue
			}
			if now.Format("15:04") < s.ReportMonthlyAutoTime {
				continue
			}
			if !s.monthlyRunning.CompareAndSwap(false, true) {
				continue
			}

			// Generate for previous month
			prevMonth := now.AddDate(0, -1, 0)
			month := int(prevMonth.Month())
			year := prevMonth.Year()

			var userIDs []uint
			s.DB.Model(&models.User{}).Pluck("id", &userIDs)
			for _, uid := range userIDs {
				if err := GenerateMonthlyReportForUser(s.DB, s.Encryptor, s.R2, uid, month, year); err != nil {
					log.Printf("[monthly-report] user=%d error: %v", uid, err)
				}
			}
			s.lastMonthlyReport = currentKey
			s.monthlyRunning.Store(false)
		}
	}
}
```

- [ ] **Step 5: Commit**

```bash
git add backend/internal/handlers/report.go backend/internal/worker/scheduler.go backend/internal/worker/reports.go backend/internal/config/config.go
git commit -m "feat: add monthly report endpoint, worker auto-generation, and config"
```

---

## Task 8: Add WebSocket Chat Handler

**Files:**
- Create: `backend/internal/handlers/chat.go`
- Modify: `backend/cmd/server/main.go`

- [ ] **Step 1: Add gorilla/websocket dependency**

```bash
cd /home/nst/GolandProjects/pdt/backend
go get github.com/gorilla/websocket
```

- [ ] **Step 2: Create chat handler with WebSocket and conversation CRUD**

Create `backend/internal/handlers/chat.go`:

```go
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

var upgrader = websocket.Upgrader{
	CheckOrigin: func(r *http.Request) bool { return true },
}

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

func (w *wsStreamWriter) WriteContent(content string) error {
	w.mu.Lock()
	defer w.mu.Unlock()
	return w.conn.WriteJSON(map[string]string{
		"type":            "stream",
		"content":         content,
		"conversation_id": w.conversationID,
	})
}

func (w *wsStreamWriter) WriteToolStatus(toolName string, status string) error {
	w.mu.Lock()
	defer w.mu.Unlock()
	return w.conn.WriteJSON(map[string]string{
		"type":   "tool_status",
		"tool":   toolName,
		"status": status,
	})
}

func (w *wsStreamWriter) WriteDone() error {
	w.mu.Lock()
	defer w.mu.Unlock()
	return w.conn.WriteJSON(map[string]string{
		"type":            "done",
		"conversation_id": w.conversationID,
	})
}

func (w *wsStreamWriter) WriteError(msg string) error {
	w.mu.Lock()
	defer w.mu.Unlock()
	return w.conn.WriteJSON(map[string]string{
		"type":    "error",
		"content": msg,
	})
}

type wsMessage struct {
	Type           string `json:"type"`
	Content        string `json:"content"`
	ConversationID string `json:"conversation_id"`
}

func (h *ChatHandler) HandleWebSocket(c *gin.Context) {
	userID := c.GetUint("user_id")

	conn, err := upgrader.Upgrade(c.Writer, c.Request, nil)
	if err != nil {
		log.Printf("[chat] websocket upgrade error: %v", err)
		return
	}
	defer conn.Close()

	// Set up ping/pong
	conn.SetReadDeadline(time.Now().Add(60 * time.Second))
	conn.SetPongHandler(func(string) error {
		conn.SetReadDeadline(time.Now().Add(60 * time.Second))
		return nil
	})

	// Ping ticker
	go func() {
		ticker := time.NewTicker(30 * time.Second)
		defer ticker.Stop()
		for range ticker.C {
			if err := conn.WriteMessage(websocket.PingMessage, nil); err != nil {
				return
			}
		}
	}()

	// Build orchestrator with user-scoped agents
	orchestrator := agent.NewOrchestrator(
		h.MiniMaxClient,
		&agent.GitAgent{DB: h.DB, UserID: userID},
		&agent.JiraAgent{DB: h.DB, UserID: userID},
		&agent.ReportAgent{DB: h.DB, UserID: userID, Generator: h.ReportGenerator, R2: h.R2},
	)

	for {
		_, rawMsg, err := conn.ReadMessage()
		if err != nil {
			if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway, websocket.CloseNormalClosure) {
				log.Printf("[chat] read error: %v", err)
			}
			return
		}

		var msg wsMessage
		if err := json.Unmarshal(rawMsg, &msg); err != nil {
			log.Printf("[chat] parse error: %v", err)
			continue
		}

		if msg.Type != "message" || msg.Content == "" {
			continue
		}

		// Get or create conversation
		conversationID := msg.ConversationID
		if conversationID == "" {
			conv := models.Conversation{
				UserID: userID,
				Title:  truncate(msg.Content, 100),
			}
			h.DB.Create(&conv)
			conversationID = conv.ID
		}

		// Save user message
		h.DB.Create(&models.ChatMessage{
			ConversationID: conversationID,
			Role:           "user",
			Content:        msg.Content,
		})

		// Load conversation history
		var chatMessages []models.ChatMessage
		h.DB.Where("conversation_id = ?", conversationID).
			Order("created_at asc").
			Limit(h.ContextWindow).
			Find(&chatMessages)

		// Convert to MiniMax messages
		var messages []minimax.Message
		for _, m := range chatMessages {
			mm := minimax.Message{
				Role:    m.Role,
				Content: m.Content,
			}
			if m.ToolCalls != "" {
				json.Unmarshal([]byte(m.ToolCalls), &mm.ToolCalls)
			}
			if m.ToolCallID != "" {
				mm.ToolCallID = m.ToolCallID
			}
			messages = append(messages, mm)
		}

		writer := &wsStreamWriter{
			conn:           conn,
			conversationID: conversationID,
		}

		// Run orchestrator
		ctx := context.Background()
		result, err := orchestrator.HandleMessage(ctx, messages, writer)
		if err != nil {
			log.Printf("[chat] orchestrator error: %v", err)
			writer.WriteError(err.Error())
			continue
		}

		// Save assistant response
		h.DB.Create(&models.ChatMessage{
			ConversationID: conversationID,
			Role:           "assistant",
			Content:        result.FullResponse,
		})

		// Log usage
		h.DB.Create(&models.AIUsage{
			UserID:           userID,
			ConversationID:   conversationID,
			PromptTokens:     result.Usage.PromptTokens,
			CompletionTokens: result.Usage.CompletionTokens,
		})

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

	var conversation models.Conversation
	if err := h.DB.Where("id = ? AND user_id = ?", id, userID).
		Preload("Messages", func(db *gorm.DB) *gorm.DB {
			return db.Order("created_at asc")
		}).
		First(&conversation).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "conversation not found"})
		return
	}

	c.JSON(http.StatusOK, conversation)
}

func (h *ChatHandler) DeleteConversation(c *gin.Context) {
	userID := c.GetUint("user_id")
	id := c.Param("id")

	result := h.DB.Where("id = ? AND user_id = ?", id, userID).Delete(&models.Conversation{})
	if result.RowsAffected == 0 {
		c.JSON(http.StatusNotFound, gin.H{"error": "conversation not found"})
		return
	}

	// Delete associated messages
	h.DB.Where("conversation_id = ?", id).Delete(&models.ChatMessage{})

	c.JSON(http.StatusOK, gin.H{"message": "conversation deleted"})
}

func truncate(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	return s[:maxLen] + "..."
}
```

- [ ] **Step 3: Register routes in main.go**

In `backend/cmd/server/main.go`, add the ChatHandler initialization and routes. After the existing handler initializations:

```go
// Initialize MiniMax client (only if API key is configured)
var miniMaxClient *minimax.Client
if cfg.MiniMaxAPIKey != "" {
	miniMaxClient = minimax.NewClient(cfg.MiniMaxAPIKey, cfg.MiniMaxGroupID)
}

chatHandler := &handlers.ChatHandler{
	DB:              db,
	MiniMaxClient:   miniMaxClient,
	Encryptor:       encryptor,
	R2:              r2Client,
	ReportGenerator: reportGenerator,
	ContextWindow:   cfg.AIContextWindow,
}
```

Add routes in the protected group:

```go
// Chat / AI Assistant
if miniMaxClient != nil {
	protected.GET("/ws/chat", chatHandler.HandleWebSocket)
	protected.GET("/conversations", chatHandler.ListConversations)
	protected.GET("/conversations/:id", chatHandler.GetConversation)
	protected.DELETE("/conversations/:id", chatHandler.DeleteConversation)
}

// Monthly reports
protected.POST("/reports/generate-monthly", reportHandler.GenerateMonthly)
```

Add the import for the minimax package:

```go
"github.com/cds-id/pdt/backend/internal/ai/minimax"
```

- [ ] **Step 4: Verify it compiles**

```bash
cd /home/nst/GolandProjects/pdt/backend
go build ./cmd/server/
```

Fix any compilation errors.

- [ ] **Step 5: Commit**

```bash
git add backend/internal/handlers/chat.go backend/cmd/server/main.go backend/go.mod backend/go.sum
git commit -m "feat: add WebSocket chat handler, conversation CRUD, and route registration"
```

---

## Task 9: Frontend — Chat TypeScript Interfaces and API Service

**Files:**
- Create: `frontend/src/domain/chat/interfaces/chat.interface.ts`
- Create: `frontend/src/infrastructure/services/chat.service.ts`
- Modify: `frontend/src/infrastructure/constants/api.constants.ts`
- Modify: `frontend/src/infrastructure/services/api.ts`

- [ ] **Step 1: Create chat interfaces**

Create `frontend/src/domain/chat/interfaces/chat.interface.ts`:

```typescript
export interface IConversation {
  id: string
  user_id: number
  title: string
  created_at: string
  updated_at: string
  messages?: IChatMessage[]
}

export interface IChatMessage {
  id: string
  conversation_id: string
  role: 'user' | 'assistant' | 'tool'
  content: string
  tool_calls?: string
  tool_name?: string
  created_at: string
}

export interface IWSMessage {
  type: 'message'
  content: string
  conversation_id?: string
}

export interface IWSResponse {
  type: 'stream' | 'tool_status' | 'done' | 'error'
  content?: string
  conversation_id?: string
  tool?: string
  status?: 'executing' | 'completed'
}
```

- [ ] **Step 2: Add API constants**

In `frontend/src/infrastructure/constants/api.constants.ts`, add to `API_CONSTANTS`:

```typescript
CONVERSATIONS: {
  LIST: '/conversations',
  GET: (id: string) => `/conversations/${id}`,
  DELETE: (id: string) => `/conversations/${id}`,
},
CHAT: {
  WS: '/ws/chat',
},
```

- [ ] **Step 3: Add Conversation tag to base API**

In `frontend/src/infrastructure/services/api.ts`, add `'Conversation'` to the `tagTypes` array.

- [ ] **Step 4: Create chat service**

Create `frontend/src/infrastructure/services/chat.service.ts`:

```typescript
import { api } from './api'
import { API_CONSTANTS } from '../constants/api.constants'
import type { IConversation } from '../../domain/chat/interfaces/chat.interface'

export const chatApi = api.injectEndpoints({
  endpoints: (builder) => ({
    listConversations: builder.query<IConversation[], void>({
      query: () => API_CONSTANTS.CONVERSATIONS.LIST,
      providesTags: (result) =>
        result
          ? [
              ...result.map(({ id }) => ({ type: 'Conversation' as const, id })),
              { type: 'Conversation', id: 'LIST' },
            ]
          : [{ type: 'Conversation', id: 'LIST' }],
    }),
    getConversation: builder.query<IConversation, string>({
      query: (id) => API_CONSTANTS.CONVERSATIONS.GET(id),
      providesTags: (_result, _error, id) => [{ type: 'Conversation', id }],
    }),
    deleteConversation: builder.mutation<void, string>({
      query: (id) => ({
        url: API_CONSTANTS.CONVERSATIONS.DELETE(id),
        method: 'DELETE',
      }),
      invalidatesTags: [{ type: 'Conversation', id: 'LIST' }],
    }),
  }),
})

export const {
  useListConversationsQuery,
  useGetConversationQuery,
  useDeleteConversationMutation,
} = chatApi
```

- [ ] **Step 5: Add monthly report endpoint to report service**

In `frontend/src/infrastructure/services/report.service.ts`, add to the existing endpoints:

```typescript
generateMonthlyReport: builder.mutation<Report, { month: number; year: number }>({
  query: ({ month, year }) => ({
    url: API_CONSTANTS.REPORTS.GENERATE_MONTHLY,
    method: 'POST',
    body: { month, year },
  }),
  invalidatesTags: [{ type: 'Report', id: 'LIST' }],
}),
```

Export the hook:

```typescript
export const {
  // ... existing exports
  useGenerateMonthlyReportMutation,
} = reportApi
```

Add to API constants:

```typescript
GENERATE_MONTHLY: '/reports/generate-monthly',
```

- [ ] **Step 6: Commit**

```bash
git add frontend/src/domain/chat/ frontend/src/infrastructure/services/chat.service.ts frontend/src/infrastructure/constants/api.constants.ts frontend/src/infrastructure/services/api.ts frontend/src/infrastructure/services/report.service.ts
git commit -m "feat: add chat interfaces, conversation API service, and monthly report endpoint"
```

---

## Task 10: Frontend — Assistant Chat Page

**Files:**
- Create: `frontend/src/presentation/components/chat/ChatMessage.tsx`
- Create: `frontend/src/presentation/components/chat/ChatInput.tsx`
- Create: `frontend/src/presentation/components/chat/ChatSidebar.tsx`
- Create: `frontend/src/presentation/components/chat/ToolStatus.tsx`
- Create: `frontend/src/presentation/pages/AssistantPage.tsx`
- Modify: `frontend/src/config/navigation.ts`
- Modify: `frontend/src/presentation/routes/index.tsx`

- [ ] **Step 1: Create ChatMessage component**

Create `frontend/src/presentation/components/chat/ChatMessage.tsx`:

```tsx
import { User, Bot } from 'lucide-react'

interface ChatMessageProps {
  role: 'user' | 'assistant'
  content: string
  isStreaming?: boolean
}

export function ChatMessage({ role, content, isStreaming }: ChatMessageProps) {
  return (
    <div className={`flex gap-3 ${role === 'user' ? 'justify-end' : 'justify-start'}`}>
      {role === 'assistant' && (
        <div className="flex-shrink-0 w-8 h-8 rounded-full bg-pdt-accent/20 flex items-center justify-center">
          <Bot className="w-4 h-4 text-pdt-accent" />
        </div>
      )}
      <div
        className={`max-w-[80%] rounded-lg px-4 py-2 ${
          role === 'user'
            ? 'bg-pdt-accent text-pdt-neutral-900'
            : 'bg-pdt-neutral-800 text-pdt-neutral-100'
        }`}
      >
        <div className="whitespace-pre-wrap text-sm">{content}</div>
        {isStreaming && (
          <span className="inline-block w-2 h-4 bg-current animate-pulse ml-0.5" />
        )}
      </div>
      {role === 'user' && (
        <div className="flex-shrink-0 w-8 h-8 rounded-full bg-pdt-neutral-700 flex items-center justify-center">
          <User className="w-4 h-4 text-pdt-neutral-300" />
        </div>
      )}
    </div>
  )
}
```

- [ ] **Step 2: Create ToolStatus component**

Create `frontend/src/presentation/components/chat/ToolStatus.tsx`:

```tsx
import { Loader2, CheckCircle } from 'lucide-react'

interface ToolStatusProps {
  toolName: string
  status: 'executing' | 'completed'
}

const toolLabels: Record<string, string> = {
  search_commits: 'Searching commits',
  get_commit_detail: 'Getting commit details',
  list_repos: 'Listing repositories',
  get_repo_stats: 'Getting repo statistics',
  get_sprints: 'Fetching sprints',
  get_cards: 'Fetching Jira cards',
  get_card_detail: 'Getting card details',
  search_cards: 'Searching cards',
  link_commit_to_card: 'Linking commit to card',
  generate_daily_report: 'Generating daily report',
  generate_monthly_report: 'Generating monthly report',
  list_reports: 'Listing reports',
  get_report: 'Getting report',
}

export function ToolStatus({ toolName, status }: ToolStatusProps) {
  const label = toolLabels[toolName] || toolName

  return (
    <div className="flex items-center gap-2 text-xs text-pdt-neutral-400 py-1 px-3">
      {status === 'executing' ? (
        <Loader2 className="w-3 h-3 animate-spin" />
      ) : (
        <CheckCircle className="w-3 h-3 text-green-500" />
      )}
      <span>{label}...</span>
    </div>
  )
}
```

- [ ] **Step 3: Create ChatInput component**

Create `frontend/src/presentation/components/chat/ChatInput.tsx`:

```tsx
import { useState, useRef, useEffect } from 'react'
import { Send } from 'lucide-react'

interface ChatInputProps {
  onSend: (message: string) => void
  disabled?: boolean
}

export function ChatInput({ onSend, disabled }: ChatInputProps) {
  const [value, setValue] = useState('')
  const textareaRef = useRef<HTMLTextAreaElement>(null)

  useEffect(() => {
    if (textareaRef.current) {
      textareaRef.current.style.height = 'auto'
      textareaRef.current.style.height = Math.min(textareaRef.current.scrollHeight, 150) + 'px'
    }
  }, [value])

  const handleSubmit = () => {
    const trimmed = value.trim()
    if (!trimmed || disabled) return
    onSend(trimmed)
    setValue('')
  }

  const handleKeyDown = (e: React.KeyboardEvent) => {
    if (e.key === 'Enter' && !e.shiftKey) {
      e.preventDefault()
      handleSubmit()
    }
  }

  return (
    <div className="border-t border-pdt-neutral-700 p-4">
      <div className="flex gap-2 items-end max-w-3xl mx-auto">
        <textarea
          ref={textareaRef}
          value={value}
          onChange={(e) => setValue(e.target.value)}
          onKeyDown={handleKeyDown}
          placeholder="Ask about your commits, Jira cards, or reports..."
          className="flex-1 resize-none bg-pdt-neutral-800 border border-pdt-neutral-600 rounded-lg px-4 py-2 text-sm text-pdt-neutral-100 placeholder-pdt-neutral-500 focus:outline-none focus:border-pdt-accent"
          rows={1}
          disabled={disabled}
        />
        <button
          onClick={handleSubmit}
          disabled={disabled || !value.trim()}
          className="flex-shrink-0 w-10 h-10 rounded-lg bg-pdt-accent text-pdt-neutral-900 flex items-center justify-center hover:bg-pdt-accent/90 disabled:opacity-50 disabled:cursor-not-allowed"
        >
          <Send className="w-4 h-4" />
        </button>
      </div>
    </div>
  )
}
```

- [ ] **Step 4: Create ChatSidebar component**

Create `frontend/src/presentation/components/chat/ChatSidebar.tsx`:

```tsx
import { Plus, MessageSquare, Trash2 } from 'lucide-react'
import type { IConversation } from '../../../domain/chat/interfaces/chat.interface'

interface ChatSidebarProps {
  conversations: IConversation[]
  activeId?: string
  onSelect: (id: string) => void
  onNew: () => void
  onDelete: (id: string) => void
}

export function ChatSidebar({ conversations, activeId, onSelect, onNew, onDelete }: ChatSidebarProps) {
  return (
    <div className="w-64 border-r border-pdt-neutral-700 flex flex-col h-full">
      <div className="p-3">
        <button
          onClick={onNew}
          className="w-full flex items-center gap-2 px-3 py-2 rounded-lg border border-pdt-neutral-600 text-sm text-pdt-neutral-300 hover:bg-pdt-neutral-800"
        >
          <Plus className="w-4 h-4" />
          New Conversation
        </button>
      </div>
      <div className="flex-1 overflow-y-auto px-2">
        {conversations.map((conv) => (
          <div
            key={conv.id}
            className={`group flex items-center gap-2 px-3 py-2 rounded-lg cursor-pointer mb-1 text-sm ${
              activeId === conv.id
                ? 'bg-pdt-accent/20 text-pdt-accent'
                : 'text-pdt-neutral-400 hover:bg-pdt-neutral-800'
            }`}
            onClick={() => onSelect(conv.id)}
          >
            <MessageSquare className="w-4 h-4 flex-shrink-0" />
            <span className="truncate flex-1">{conv.title}</span>
            <button
              onClick={(e) => {
                e.stopPropagation()
                onDelete(conv.id)
              }}
              className="opacity-0 group-hover:opacity-100 text-pdt-neutral-500 hover:text-red-400"
            >
              <Trash2 className="w-3 h-3" />
            </button>
          </div>
        ))}
      </div>
    </div>
  )
}
```

- [ ] **Step 5: Create AssistantPage**

Create `frontend/src/presentation/pages/AssistantPage.tsx`:

```tsx
import { useState, useEffect, useRef, useCallback } from 'react'
import { useAppSelector } from '../../application/hooks/useAppSelector'
import {
  useListConversationsQuery,
  useDeleteConversationMutation,
} from '../../infrastructure/services/chat.service'
import { API_CONSTANTS } from '../../infrastructure/constants/api.constants'
import { PageHeader } from '../components/common/PageHeader'
import { ChatSidebar } from '../components/chat/ChatSidebar'
import { ChatMessage } from '../components/chat/ChatMessage'
import { ChatInput } from '../components/chat/ChatInput'
import { ToolStatus } from '../components/chat/ToolStatus'
import type { IWSResponse } from '../../domain/chat/interfaces/chat.interface'

interface DisplayMessage {
  id: string
  role: 'user' | 'assistant'
  content: string
  isStreaming?: boolean
}

interface ToolStatusItem {
  tool: string
  status: 'executing' | 'completed'
}

export function AssistantPage() {
  const token = useAppSelector((state) => state.auth.token)
  const { data: conversations = [], refetch } = useListConversationsQuery()
  const [deleteConversation] = useDeleteConversationMutation()

  const [activeConversationId, setActiveConversationId] = useState<string | undefined>()
  const [messages, setMessages] = useState<DisplayMessage[]>([])
  const [toolStatuses, setToolStatuses] = useState<ToolStatusItem[]>([])
  const [isStreaming, setIsStreaming] = useState(false)
  const wsRef = useRef<WebSocket | null>(null)
  const messagesEndRef = useRef<HTMLDivElement>(null)
  const streamBufferRef = useRef('')

  const scrollToBottom = useCallback(() => {
    messagesEndRef.current?.scrollIntoView({ behavior: 'smooth' })
  }, [])

  useEffect(() => {
    scrollToBottom()
  }, [messages, toolStatuses, scrollToBottom])

  // Connect WebSocket
  useEffect(() => {
    if (!token) return

    const wsUrl = `${API_CONSTANTS.BASE_URL.replace('http', 'ws')}${API_CONSTANTS.API_PREFIX}${API_CONSTANTS.CHAT.WS}?token=${token}`
    const ws = new WebSocket(wsUrl)

    ws.onopen = () => {
      console.log('[ws] connected')
    }

    ws.onmessage = (event) => {
      const data: IWSResponse = JSON.parse(event.data)

      switch (data.type) {
        case 'stream':
          if (data.conversation_id && !activeConversationId) {
            setActiveConversationId(data.conversation_id)
          }
          streamBufferRef.current += data.content || ''
          setMessages((prev) => {
            const last = prev[prev.length - 1]
            if (last && last.role === 'assistant' && last.isStreaming) {
              return [
                ...prev.slice(0, -1),
                { ...last, content: streamBufferRef.current },
              ]
            }
            return [
              ...prev,
              {
                id: crypto.randomUUID(),
                role: 'assistant',
                content: streamBufferRef.current,
                isStreaming: true,
              },
            ]
          })
          break

        case 'tool_status':
          if (data.tool && data.status) {
            setToolStatuses((prev) => {
              const existing = prev.findIndex((t) => t.tool === data.tool)
              if (existing >= 0) {
                const updated = [...prev]
                updated[existing] = { tool: data.tool!, status: data.status! }
                return updated
              }
              return [...prev, { tool: data.tool!, status: data.status! }]
            })
          }
          break

        case 'done':
          setMessages((prev) => {
            const last = prev[prev.length - 1]
            if (last && last.isStreaming) {
              return [...prev.slice(0, -1), { ...last, isStreaming: false }]
            }
            return prev
          })
          setToolStatuses([])
          setIsStreaming(false)
          streamBufferRef.current = ''
          refetch()
          break

        case 'error':
          setIsStreaming(false)
          setToolStatuses([])
          streamBufferRef.current = ''
          setMessages((prev) => [
            ...prev,
            {
              id: crypto.randomUUID(),
              role: 'assistant',
              content: `Error: ${data.content}`,
            },
          ])
          break
      }
    }

    ws.onclose = () => {
      console.log('[ws] disconnected')
    }

    wsRef.current = ws

    return () => {
      ws.close()
    }
  }, [token])

  const handleSend = (content: string) => {
    if (!wsRef.current || wsRef.current.readyState !== WebSocket.OPEN) return

    setMessages((prev) => [
      ...prev,
      { id: crypto.randomUUID(), role: 'user', content },
    ])
    setIsStreaming(true)
    streamBufferRef.current = ''

    wsRef.current.send(
      JSON.stringify({
        type: 'message',
        content,
        conversation_id: activeConversationId,
      })
    )
  }

  const handleNewConversation = () => {
    setActiveConversationId(undefined)
    setMessages([])
    setToolStatuses([])
  }

  const handleSelectConversation = async (id: string) => {
    setActiveConversationId(id)
    setToolStatuses([])

    // Load messages from API
    try {
      const resp = await fetch(
        `${API_CONSTANTS.BASE_URL}${API_CONSTANTS.API_PREFIX}/conversations/${id}`,
        { headers: { Authorization: `Bearer ${token}` } }
      )
      const data = await resp.json()
      if (data.messages) {
        setMessages(
          data.messages
            .filter((m: { role: string }) => m.role === 'user' || m.role === 'assistant')
            .map((m: { id: string; role: 'user' | 'assistant'; content: string }) => ({
              id: m.id,
              role: m.role,
              content: m.content,
            }))
        )
      }
    } catch (err) {
      console.error('Failed to load conversation:', err)
    }
  }

  const handleDeleteConversation = async (id: string) => {
    await deleteConversation(id).unwrap()
    if (activeConversationId === id) {
      handleNewConversation()
    }
  }

  return (
    <div className="min-w-0 flex flex-col h-[calc(100vh-4rem)]">
      <PageHeader title="AI Assistant" description="Chat with your development data" />
      <div className="flex flex-1 overflow-hidden border border-pdt-neutral-700 rounded-lg mx-4 mb-4">
        <ChatSidebar
          conversations={conversations}
          activeId={activeConversationId}
          onSelect={handleSelectConversation}
          onNew={handleNewConversation}
          onDelete={handleDeleteConversation}
        />
        <div className="flex-1 flex flex-col">
          <div className="flex-1 overflow-y-auto p-4 space-y-4">
            {messages.length === 0 && (
              <div className="flex items-center justify-center h-full text-pdt-neutral-500 text-sm">
                Start a conversation — ask about your commits, Jira cards, or reports.
              </div>
            )}
            {messages.map((msg) => (
              <ChatMessage
                key={msg.id}
                role={msg.role}
                content={msg.content}
                isStreaming={msg.isStreaming}
              />
            ))}
            {toolStatuses.map((ts) => (
              <ToolStatus key={ts.tool} toolName={ts.tool} status={ts.status} />
            ))}
            <div ref={messagesEndRef} />
          </div>
          <ChatInput onSend={handleSend} disabled={isStreaming} />
        </div>
      </div>
    </div>
  )
}
```

- [ ] **Step 6: Add route and navigation**

In `frontend/src/config/navigation.ts`, add to the navigation items (in the appropriate section):

```typescript
{
  label: 'AI Assistant',
  path: '/assistant',
  icon: 'Bot',
}
```

In `frontend/src/presentation/routes/index.tsx`, add the route inside the dashboard layout:

```tsx
import { AssistantPage } from '../pages/AssistantPage'

// Inside the route config:
{ path: 'assistant', element: <AssistantPage /> }
```

- [ ] **Step 7: Commit**

```bash
git add frontend/src/presentation/components/chat/ frontend/src/presentation/pages/AssistantPage.tsx frontend/src/config/navigation.ts frontend/src/presentation/routes/index.tsx
git commit -m "feat: add AI Assistant chat page with WebSocket streaming and conversation management"
```

---

## Task 11: Frontend — Monthly Reports Tab on Reports Page

**Files:**
- Modify: `frontend/src/presentation/pages/ReportsPage.tsx`

- [ ] **Step 1: Add monthly report generation UI**

In `frontend/src/presentation/pages/ReportsPage.tsx`, add a "Monthly" tab alongside the existing "Reports" and "Templates" tabs. The monthly tab should have:

- A month/year picker (two select dropdowns)
- A "Generate Monthly Report" button
- A list of existing monthly reports (filtered by `report_type === 'monthly'`)

Add to the tabs state:
```typescript
const [activeTab, setActiveTab] = useState<'reports' | 'templates' | 'monthly'>('reports')
```

Add the import:
```typescript
import { useGenerateMonthlyReportMutation } from '../../infrastructure/services/report.service'
```

Add the hook:
```typescript
const [generateMonthly, { isLoading: isGeneratingMonthly }] = useGenerateMonthlyReportMutation()
```

Add state for month/year:
```typescript
const [selectedMonth, setSelectedMonth] = useState(new Date().getMonth() + 1)
const [selectedYear, setSelectedYear] = useState(new Date().getFullYear())
```

Add the monthly tab content:
```tsx
{activeTab === 'monthly' && (
  <DataCard title="Monthly Reports">
    <div className="flex gap-4 items-end mb-6">
      <div>
        <label className="block text-xs text-pdt-neutral-400 mb-1">Month</label>
        <select
          value={selectedMonth}
          onChange={(e) => setSelectedMonth(Number(e.target.value))}
          className="bg-pdt-neutral-800 border border-pdt-neutral-600 rounded px-3 py-2 text-sm text-pdt-neutral-100"
        >
          {Array.from({ length: 12 }, (_, i) => (
            <option key={i + 1} value={i + 1}>
              {new Date(2026, i).toLocaleString('default', { month: 'long' })}
            </option>
          ))}
        </select>
      </div>
      <div>
        <label className="block text-xs text-pdt-neutral-400 mb-1">Year</label>
        <select
          value={selectedYear}
          onChange={(e) => setSelectedYear(Number(e.target.value))}
          className="bg-pdt-neutral-800 border border-pdt-neutral-600 rounded px-3 py-2 text-sm text-pdt-neutral-100"
        >
          {[2024, 2025, 2026].map((y) => (
            <option key={y} value={y}>{y}</option>
          ))}
        </select>
      </div>
      <button
        onClick={() => generateMonthly({ month: selectedMonth, year: selectedYear })}
        disabled={isGeneratingMonthly}
        className="px-4 py-2 bg-pdt-accent text-pdt-neutral-900 rounded text-sm font-medium hover:bg-pdt-accent/90 disabled:opacity-50"
      >
        {isGeneratingMonthly ? 'Generating...' : 'Generate Monthly Report'}
      </button>
    </div>
    <div className="space-y-3">
      {(reportsData || [])
        .filter((r) => r.report_type === 'monthly')
        .map((report) => (
          <div key={report.id} className="p-4 rounded-lg bg-pdt-neutral-800 border border-pdt-neutral-700">
            <div className="flex justify-between items-center">
              <div>
                <p className="text-sm font-medium text-pdt-neutral-100">{report.title}</p>
                <p className="text-xs text-pdt-neutral-400">{report.date}</p>
              </div>
            </div>
          </div>
        ))}
    </div>
  </DataCard>
)}
```

Add the "Monthly" tab button alongside the existing tab buttons.

- [ ] **Step 2: Commit**

```bash
git add frontend/src/presentation/pages/ReportsPage.tsx
git commit -m "feat: add monthly reports tab with month/year picker and generation"
```

---

## Task 12: Integration Test — End-to-End Verification

**Files:** None (manual testing)

- [ ] **Step 1: Start the backend**

```bash
cd /home/nst/GolandProjects/pdt/backend
go run cmd/server/main.go
```

Verify:
- No compilation errors
- Database migrations run (check for new tables: `conversations`, `chat_messages`, `ai_usages`)
- Log shows `[jira-sync]` messages with pagination info
- MiniMax client initializes (if API key is in `.env`)

- [ ] **Step 2: Start the frontend**

```bash
cd /home/nst/GolandProjects/pdt/frontend
bun run dev
```

Verify:
- Frontend compiles without errors
- "AI Assistant" appears in sidebar navigation
- `/assistant` page loads with chat interface
- Reports page has "Monthly" tab

- [ ] **Step 3: Test WebSocket chat**

1. Navigate to `/assistant`
2. Type "Hello" — should get a greeting response
3. Type "What commits did I make this week?" — should route to Git Agent, show tool status, stream response
4. Type "Show me my active Jira sprint" — should route to Jira Agent
5. Type "Generate today's report" — should route to Report Agent

- [ ] **Step 4: Test monthly report generation**

1. Navigate to Reports page
2. Click "Monthly" tab
3. Select a month/year with known activity
4. Click "Generate Monthly Report"
5. Verify report appears in the list

- [ ] **Step 5: Verify Jira fix**

1. Trigger a manual sync
2. Check logs for `[jira-sync]` with card counts
3. Navigate to Jira page — verify all cards appear for all sprints
4. Check specific sprints/projects that were previously missing

- [ ] **Step 6: Final commit if any fixes needed**

```bash
git add -A
git commit -m "fix: address integration test findings"
```

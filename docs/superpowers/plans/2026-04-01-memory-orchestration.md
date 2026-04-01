# Memory System & Orchestration Optimization Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Add cross-conversation memory via Weaviate (summaries + user facts) and optimize the orchestrator with memory-aware routing so agents have context from past conversations and routing is smarter.

**Architecture:** Background memory worker processes idle conversations via LLM extraction, storing summaries and facts in two new Weaviate collections. At request time, the orchestrator and all agents receive memory context (user facts + relevant summaries) via a decorator pattern and modified router prompt. The router gains the ability to answer context-aware questions directly.

**Tech Stack:** Go (GORM, Weaviate Go client v4, MiniMax/Anthropic SDK), Weaviate (text2vec-google vectorizer)

---

## File Map

### New Files
- `backend/internal/services/weaviate/memory_embed.go` — Weaviate schema + CRUD for ConversationSummary and UserFact collections
- `backend/internal/worker/memory.go` — Background worker that processes conversations into memory
- `backend/internal/ai/memory/enhanced_agent.go` — Decorator that injects memory into agent system prompts
- `backend/internal/ai/memory/context.go` — Shared memory fetch logic (user facts + summaries)

### Modified Files
- `backend/internal/models/conversation.go` — Add `SummarizedAt` field to Conversation model
- `backend/internal/services/weaviate/client.go` — Call `ensureMemorySchema` at init
- `backend/internal/ai/agent/orchestrator.go` — Inject memory into router prompt, improve direct answers
- `backend/internal/handlers/chat.go` — Fetch memory context, pass to orchestrator, wrap agents with memory decorator
- `backend/cmd/server/main.go` — Start memory worker, wire memory dependencies
- `backend/internal/config/config.go` — Add `MemoryWorkerInterval` config

---

### Task 1: Weaviate Memory Collections

**Files:**
- Create: `backend/internal/services/weaviate/memory_embed.go`
- Modify: `backend/internal/services/weaviate/client.go`

- [ ] **Step 1: Create the memory embed file with schema + CRUD methods**

```go
package weaviate

import (
	"context"
	"crypto/sha256"
	"fmt"
	"time"

	"github.com/weaviate/weaviate-go-client/v4/weaviate/filters"
	"github.com/weaviate/weaviate-go-client/v4/weaviate/graphql"
	"github.com/weaviate/weaviate/entities/models"
)

const (
	summaryCollectionName  = "ConversationSummary"
	userFactCollectionName = "UserFact"
)

func (c *Client) ensureMemorySchema(ctx context.Context) error {
	if err := c.ensureSummarySchema(ctx); err != nil {
		return fmt.Errorf("summary schema: %w", err)
	}
	if err := c.ensureUserFactSchema(ctx); err != nil {
		return fmt.Errorf("user fact schema: %w", err)
	}
	return nil
}

func (c *Client) ensureSummarySchema(ctx context.Context) error {
	_, err := c.client.Schema().ClassGetter().WithClassName(summaryCollectionName).Do(ctx)
	if err == nil {
		return nil
	}

	trueVal := true
	skip := map[string]interface{}{"text2vec-google": map[string]interface{}{"skip": true}}
	noSkip := map[string]interface{}{"text2vec-google": map[string]interface{}{"skip": false}}

	class := &models.Class{
		Class:      summaryCollectionName,
		Vectorizer: "text2vec-google",
		ModuleConfig: map[string]interface{}{
			"text2vec-google": map[string]interface{}{
				"projectId":   "google",
				"apiEndpoint": "generativelanguage.googleapis.com",
				"modelId":     "gemini-embedding-001",
			},
		},
		Properties: []*models.Property{
			{Name: "conversation_id", DataType: []string{"text"}, ModuleConfig: skip},
			{Name: "user_id", DataType: []string{"number"}, ModuleConfig: skip},
			{Name: "content", DataType: []string{"text"}, IndexInverted: &trueVal, Tokenization: models.PropertyTokenizationWord, ModuleConfig: noSkip},
			{Name: "agent_used", DataType: []string{"text"}, ModuleConfig: skip},
			{Name: "created_at", DataType: []string{"date"}, ModuleConfig: skip},
		},
	}
	return c.client.Schema().ClassCreator().WithClass(class).Do(ctx)
}

func (c *Client) ensureUserFactSchema(ctx context.Context) error {
	_, err := c.client.Schema().ClassGetter().WithClassName(userFactCollectionName).Do(ctx)
	if err == nil {
		return nil
	}

	trueVal := true
	skip := map[string]interface{}{"text2vec-google": map[string]interface{}{"skip": true}}
	noSkip := map[string]interface{}{"text2vec-google": map[string]interface{}{"skip": false}}

	class := &models.Class{
		Class:      userFactCollectionName,
		Vectorizer: "text2vec-google",
		ModuleConfig: map[string]interface{}{
			"text2vec-google": map[string]interface{}{
				"projectId":   "google",
				"apiEndpoint": "generativelanguage.googleapis.com",
				"modelId":     "gemini-embedding-001",
			},
		},
		Properties: []*models.Property{
			{Name: "user_id", DataType: []string{"number"}, ModuleConfig: skip},
			{Name: "content", DataType: []string{"text"}, IndexInverted: &trueVal, Tokenization: models.PropertyTokenizationWord, ModuleConfig: noSkip},
			{Name: "category", DataType: []string{"text"}, ModuleConfig: skip},
			{Name: "created_at", DataType: []string{"date"}, ModuleConfig: skip},
			{Name: "updated_at", DataType: []string{"date"}, ModuleConfig: skip},
		},
	}
	return c.client.Schema().ClassCreator().WithClass(class).Do(ctx)
}

func summaryUUID(conversationID string) string {
	data := fmt.Sprintf("conv-summary-%s", conversationID)
	h := sha256.Sum256([]byte(data))
	h[6] = (h[6] & 0x0f) | 0x50
	h[8] = (h[8] & 0x3f) | 0x80
	return fmt.Sprintf("%08x-%04x-%04x-%04x-%012x",
		h[0:4], h[4:6], h[6:8], h[8:10], h[10:16])
}

func userFactUUID(userID int, content string) string {
	data := fmt.Sprintf("user-fact-%d-%s", userID, content)
	h := sha256.Sum256([]byte(data))
	h[6] = (h[6] & 0x0f) | 0x50
	h[8] = (h[8] & 0x3f) | 0x80
	return fmt.Sprintf("%08x-%04x-%04x-%04x-%012x",
		h[0:4], h[4:6], h[6:8], h[8:10], h[10:16])
}

// UpsertConversationSummary stores a conversation summary in Weaviate.
func (c *Client) UpsertConversationSummary(ctx context.Context, conversationID string, userID int, content, agentUsed string) error {
	if !c.available || content == "" {
		return nil
	}

	uuid := summaryUUID(conversationID)
	properties := map[string]interface{}{
		"conversation_id": conversationID,
		"user_id":         float64(userID),
		"content":         content,
		"agent_used":      agentUsed,
		"created_at":      time.Now().UTC().Format(time.RFC3339),
	}

	err := c.client.Data().Updater().
		WithClassName(summaryCollectionName).
		WithID(uuid).
		WithProperties(properties).
		WithMerge().
		Do(ctx)

	if err != nil {
		_, createErr := c.client.Data().Creator().
			WithClassName(summaryCollectionName).
			WithID(uuid).
			WithProperties(properties).
			Do(ctx)
		if createErr != nil {
			return fmt.Errorf("summary upsert failed: %w", createErr)
		}
	}
	return nil
}

// UpsertUserFact stores or updates a user fact in Weaviate.
// Uses deterministic UUID based on user_id + content for idempotent upserts.
func (c *Client) UpsertUserFact(ctx context.Context, userID int, content, category string) error {
	if !c.available || content == "" {
		return nil
	}

	uuid := userFactUUID(userID, content)
	now := time.Now().UTC().Format(time.RFC3339)
	properties := map[string]interface{}{
		"user_id":    float64(userID),
		"content":    content,
		"category":   category,
		"created_at": now,
		"updated_at": now,
	}

	err := c.client.Data().Updater().
		WithClassName(userFactCollectionName).
		WithID(uuid).
		WithProperties(properties).
		WithMerge().
		Do(ctx)

	if err != nil {
		_, createErr := c.client.Data().Creator().
			WithClassName(userFactCollectionName).
			WithID(uuid).
			WithProperties(properties).
			Do(ctx)
		if createErr != nil {
			return fmt.Errorf("user fact upsert failed: %w", createErr)
		}
	}
	return nil
}

// SearchConversationSummaries performs semantic search on past conversation summaries.
func (c *Client) SearchConversationSummaries(ctx context.Context, query string, userID int, limit int) ([]SummarySearchResult, error) {
	if !c.available {
		return nil, nil
	}
	if limit <= 0 {
		limit = 3
	}

	whereFilter := filters.Where().
		WithPath([]string{"user_id"}).
		WithOperator(filters.Equal).
		WithValueNumber(float64(userID))

	nearText := c.client.GraphQL().NearTextArgBuilder().WithConcepts([]string{query})

	fields := []graphql.Field{
		{Name: "conversation_id"},
		{Name: "content"},
		{Name: "agent_used"},
		{Name: "created_at"},
		{Name: "_additional", Fields: []graphql.Field{{Name: "distance"}}},
	}

	result, err := c.client.GraphQL().Get().
		WithClassName(summaryCollectionName).
		WithNearText(nearText).
		WithFields(fields...).
		WithWhere(whereFilter).
		WithLimit(limit).
		Do(ctx)

	if err != nil {
		return nil, fmt.Errorf("summary search failed: %w", err)
	}
	if len(result.Errors) > 0 {
		return nil, fmt.Errorf("summary search error: %v", result.Errors[0].Message)
	}

	return parseSummaryResults(result.Data)
}

// GetUserFacts fetches all facts for a user (filtered, not semantic search).
func (c *Client) GetUserFacts(ctx context.Context, userID int) ([]UserFactResult, error) {
	if !c.available {
		return nil, nil
	}

	whereFilter := filters.Where().
		WithPath([]string{"user_id"}).
		WithOperator(filters.Equal).
		WithValueNumber(float64(userID))

	fields := []graphql.Field{
		{Name: "content"},
		{Name: "category"},
		{Name: "updated_at"},
	}

	result, err := c.client.GraphQL().Get().
		WithClassName(userFactCollectionName).
		WithFields(fields...).
		WithWhere(whereFilter).
		WithLimit(50).
		Do(ctx)

	if err != nil {
		return nil, fmt.Errorf("user facts fetch failed: %w", err)
	}
	if len(result.Errors) > 0 {
		return nil, fmt.Errorf("user facts error: %v", result.Errors[0].Message)
	}

	return parseUserFactResults(result.Data)
}

// SearchUserFacts performs semantic search on user facts (used for dedup).
func (c *Client) SearchUserFacts(ctx context.Context, query string, userID int, limit int) ([]UserFactResult, error) {
	if !c.available {
		return nil, nil
	}
	if limit <= 0 {
		limit = 5
	}

	whereFilter := filters.Where().
		WithPath([]string{"user_id"}).
		WithOperator(filters.Equal).
		WithValueNumber(float64(userID))

	nearText := c.client.GraphQL().NearTextArgBuilder().WithConcepts([]string{query})

	fields := []graphql.Field{
		{Name: "content"},
		{Name: "category"},
		{Name: "updated_at"},
		{Name: "_additional", Fields: []graphql.Field{{Name: "distance"}, {Name: "id"}}},
	}

	result, err := c.client.GraphQL().Get().
		WithClassName(userFactCollectionName).
		WithNearText(nearText).
		WithFields(fields...).
		WithWhere(whereFilter).
		WithLimit(limit).
		Do(ctx)

	if err != nil {
		return nil, fmt.Errorf("user fact search failed: %w", err)
	}
	if len(result.Errors) > 0 {
		return nil, fmt.Errorf("user fact search error: %v", result.Errors[0].Message)
	}

	return parseUserFactResults(result.Data)
}

// Result types

type SummarySearchResult struct {
	ConversationID string  `json:"conversation_id"`
	Content        string  `json:"content"`
	AgentUsed      string  `json:"agent_used"`
	CreatedAt      string  `json:"created_at"`
	Distance       float32 `json:"distance"`
}

type UserFactResult struct {
	Content   string  `json:"content"`
	Category  string  `json:"category"`
	UpdatedAt string  `json:"updated_at"`
	Distance  float32 `json:"distance"`
	ID        string  `json:"id,omitempty"`
}

func parseSummaryResults(data map[string]models.JSONObject) ([]SummarySearchResult, error) {
	raw, err := marshalAndParse(data, summaryCollectionName)
	if err != nil || raw == nil {
		return nil, err
	}

	var results []SummarySearchResult
	for _, obj := range raw {
		r := SummarySearchResult{}
		if v, ok := obj["conversation_id"].(string); ok {
			r.ConversationID = v
		}
		if v, ok := obj["content"].(string); ok {
			r.Content = v
		}
		if v, ok := obj["agent_used"].(string); ok {
			r.AgentUsed = v
		}
		if v, ok := obj["created_at"].(string); ok {
			r.CreatedAt = v
		}
		if additional, ok := obj["_additional"].(map[string]interface{}); ok {
			if d, ok := additional["distance"].(float64); ok {
				r.Distance = float32(d)
			}
		}
		results = append(results, r)
	}
	return results, nil
}

func parseUserFactResults(data map[string]models.JSONObject) ([]UserFactResult, error) {
	raw, err := marshalAndParse(data, userFactCollectionName)
	if err != nil || raw == nil {
		return nil, err
	}

	var results []UserFactResult
	for _, obj := range raw {
		r := UserFactResult{}
		if v, ok := obj["content"].(string); ok {
			r.Content = v
		}
		if v, ok := obj["category"].(string); ok {
			r.Category = v
		}
		if v, ok := obj["updated_at"].(string); ok {
			r.UpdatedAt = v
		}
		if additional, ok := obj["_additional"].(map[string]interface{}); ok {
			if d, ok := additional["distance"].(float64); ok {
				r.Distance = float32(d)
			}
			if id, ok := additional["id"].(string); ok {
				r.ID = id
			}
		}
		results = append(results, r)
	}
	return results, nil
}
```

- [ ] **Step 2: Add ensureMemorySchema call to client.go**

In `backend/internal/services/weaviate/client.go`, after the `ensureCommitSchema` call (line 79-81), add:

```go
if err := c.ensureMemorySchema(context.Background()); err != nil {
	log.Printf("[weaviate] Memory schema setup failed: %v", err)
}
```

- [ ] **Step 3: Verify it compiles**

Run: `cd /home/nst/GolandProjects/pdt/backend && go build ./internal/services/weaviate/...`

- [ ] **Step 4: Commit**

```bash
git add backend/internal/services/weaviate/memory_embed.go backend/internal/services/weaviate/client.go
git commit -m "feat(memory): add Weaviate collections for conversation summaries and user facts"
```

---

### Task 2: Conversation Model Update

**Files:**
- Modify: `backend/internal/models/conversation.go`

- [ ] **Step 1: Add SummarizedAt field to Conversation struct**

In `backend/internal/models/conversation.go`, add the `SummarizedAt` field after `UpdatedAt`:

```go
type Conversation struct {
	ID             string     `gorm:"type:varchar(36);primarykey" json:"id"`
	UserID         uint       `gorm:"index;not null" json:"user_id"`
	Title          string     `gorm:"type:varchar(255)" json:"title"`
	TelegramChatID int64      `gorm:"index" json:"telegram_chat_id,omitempty"`
	CreatedAt      time.Time  `json:"created_at"`
	UpdatedAt      time.Time  `json:"updated_at"`
	SummarizedAt   *time.Time `json:"summarized_at"`
	User           User       `gorm:"foreignKey:UserID" json:"-"`
	Messages       []ChatMessage `gorm:"foreignKey:ConversationID" json:"messages,omitempty"`
}
```

- [ ] **Step 2: Verify it compiles**

Run: `cd /home/nst/GolandProjects/pdt/backend && go build ./...`

- [ ] **Step 3: Commit**

```bash
git add backend/internal/models/conversation.go
git commit -m "feat(memory): add SummarizedAt field to Conversation model"
```

---

### Task 3: Config Update

**Files:**
- Modify: `backend/internal/config/config.go`
- Modify: `backend/.env.example`

- [ ] **Step 1: Add MemoryWorkerInterval to Config struct and Load()**

In `backend/internal/config/config.go`, add the field to the Config struct after `TelegramWhitelist`:

```go
// Memory
MemoryWorkerInterval time.Duration
```

In the `Load()` function, after `cfg.TelegramWhitelist` (line 98), add:

```go
memoryInterval, err := time.ParseDuration(getEnv("MEMORY_WORKER_INTERVAL", "5m"))
if err != nil {
	memoryInterval = 5 * time.Minute
}
cfg.MemoryWorkerInterval = memoryInterval
```

- [ ] **Step 2: Add to .env.example**

In `backend/.env.example`, after the Telegram section, add:

```
# Memory Worker — processes conversations into Weaviate memory
MEMORY_WORKER_INTERVAL=5m
```

- [ ] **Step 3: Verify it compiles**

Run: `cd /home/nst/GolandProjects/pdt/backend && go build ./...`

- [ ] **Step 4: Commit**

```bash
git add backend/internal/config/config.go backend/.env.example
git commit -m "feat(memory): add MEMORY_WORKER_INTERVAL config"
```

---

### Task 4: Memory Context Helper

**Files:**
- Create: `backend/internal/ai/memory/context.go`

- [ ] **Step 1: Create the shared memory fetch and formatting logic**

```go
package memory

import (
	"context"
	"fmt"
	"strings"

	wvClient "github.com/cds-id/pdt/backend/internal/services/weaviate"
)

// MemoryContext holds pre-fetched memory for a user session.
type MemoryContext struct {
	FactsBlock     string // formatted facts for prompt injection
	SummariesBlock string // formatted summaries for prompt injection
}

// FetchMemoryContext fetches user facts and relevant summaries from Weaviate.
// Returns an empty MemoryContext (not nil) if Weaviate is unavailable or has no data.
func FetchMemoryContext(ctx context.Context, wv *wvClient.Client, userID uint, userMessage string) MemoryContext {
	if wv == nil || !wv.IsAvailable() {
		return MemoryContext{}
	}

	mc := MemoryContext{}

	// Fetch all user facts
	facts, err := wv.GetUserFacts(ctx, int(userID))
	if err == nil && len(facts) > 0 {
		var lines []string
		for _, f := range facts {
			lines = append(lines, fmt.Sprintf("- %s", f.Content))
		}
		mc.FactsBlock = "## User Context\nFacts:\n" + strings.Join(lines, "\n")
	}

	// Semantic search for relevant conversation summaries
	if userMessage != "" {
		summaries, err := wv.SearchConversationSummaries(ctx, userMessage, int(userID), 3)
		if err == nil && len(summaries) > 0 {
			var lines []string
			for _, s := range summaries {
				date := s.CreatedAt
				if len(date) >= 10 {
					date = date[:10]
				}
				line := fmt.Sprintf("- %s: %s", date, s.Content)
				if s.AgentUsed != "" {
					line += fmt.Sprintf(" (agent: %s)", s.AgentUsed)
				}
				lines = append(lines, line)
			}
			mc.SummariesBlock = "## Relevant Past Conversations\n" + strings.Join(lines, "\n")
		}
	}

	return mc
}

// FormatForPrompt returns the full memory block for injection into a system prompt.
// Returns empty string if no memory is available.
func (mc MemoryContext) FormatForPrompt() string {
	var parts []string
	if mc.FactsBlock != "" {
		parts = append(parts, mc.FactsBlock)
	}
	if mc.SummariesBlock != "" {
		parts = append(parts, mc.SummariesBlock)
	}
	if len(parts) == 0 {
		return ""
	}
	return "\n\n" + strings.Join(parts, "\n\n")
}
```

- [ ] **Step 2: Verify it compiles**

Run: `cd /home/nst/GolandProjects/pdt/backend && go build ./internal/ai/memory/...`

- [ ] **Step 3: Commit**

```bash
git add backend/internal/ai/memory/context.go
git commit -m "feat(memory): add shared memory context fetch and formatting"
```

---

### Task 5: MemoryEnhancedAgent Decorator

**Files:**
- Create: `backend/internal/ai/memory/enhanced_agent.go`

- [ ] **Step 1: Create the decorator**

```go
package memory

import (
	"context"
	"encoding/json"

	"github.com/cds-id/pdt/backend/internal/ai/agent"
	"github.com/cds-id/pdt/backend/internal/ai/minimax"
)

// EnhancedAgent wraps an existing Agent, appending memory context to its system prompt.
type EnhancedAgent struct {
	Inner       agent.Agent
	MemoryBlock string // pre-formatted memory to append to system prompt
}

// NewEnhancedAgent creates a decorator that injects memory into an agent's system prompt.
func NewEnhancedAgent(inner agent.Agent, mc MemoryContext) *EnhancedAgent {
	return &EnhancedAgent{
		Inner:       inner,
		MemoryBlock: mc.FormatForPrompt(),
	}
}

func (e *EnhancedAgent) Name() string { return e.Inner.Name() }

func (e *EnhancedAgent) SystemPrompt() string {
	base := e.Inner.SystemPrompt()
	if e.MemoryBlock == "" {
		return base
	}
	return base + e.MemoryBlock
}

func (e *EnhancedAgent) Tools() []minimax.Tool {
	return e.Inner.Tools()
}

func (e *EnhancedAgent) ExecuteTool(ctx context.Context, name string, args json.RawMessage) (any, error) {
	return e.Inner.ExecuteTool(ctx, name, args)
}

// WrapAgents wraps all agents with memory context.
// Returns agents unchanged if memory is empty.
func WrapAgents(agents []agent.Agent, mc MemoryContext) []agent.Agent {
	if mc.FormatForPrompt() == "" {
		return agents
	}
	wrapped := make([]agent.Agent, len(agents))
	for i, a := range agents {
		wrapped[i] = NewEnhancedAgent(a, mc)
	}
	return wrapped
}
```

- [ ] **Step 2: Verify it compiles**

Run: `cd /home/nst/GolandProjects/pdt/backend && go build ./internal/ai/memory/...`

- [ ] **Step 3: Commit**

```bash
git add backend/internal/ai/memory/enhanced_agent.go
git commit -m "feat(memory): add MemoryEnhancedAgent decorator"
```

---

### Task 6: Memory Worker

**Files:**
- Create: `backend/internal/worker/memory.go`

- [ ] **Step 1: Create the memory worker**

```go
package worker

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"sync/atomic"
	"time"

	"github.com/cds-id/pdt/backend/internal/ai/minimax"
	"github.com/cds-id/pdt/backend/internal/models"
	wvClient "github.com/cds-id/pdt/backend/internal/services/weaviate"
	"gorm.io/gorm"
)

const memoryExtractionPrompt = `Analyze this conversation and extract:
1. A summary (1-3 sentences) of what was discussed and any outcomes.
2. User facts — persistent information about the user (preferences, projects, identity, habits).

Respond ONLY with valid JSON:
{
  "summary": "...",
  "agent_used": "the primary agent used (git/jira/report/proof/briefing/whatsapp/scheduler) or empty string if general",
  "facts": [
    {"content": "fact about the user", "category": "preference|project|identity"}
  ]
}

Only extract facts that are durable — not transient questions. If no new facts, return empty array.`

type memoryExtraction struct {
	Summary   string `json:"summary"`
	AgentUsed string `json:"agent_used"`
	Facts     []struct {
		Content  string `json:"content"`
		Category string `json:"category"`
	} `json:"facts"`
}

type MemoryWorker struct {
	DB       *gorm.DB
	Client   *minimax.Client
	Weaviate *wvClient.Client
	Interval time.Duration
	running  atomic.Bool
}

func NewMemoryWorker(db *gorm.DB, client *minimax.Client, wv *wvClient.Client, interval time.Duration) *MemoryWorker {
	return &MemoryWorker{
		DB:       db,
		Client:   client,
		Weaviate: wv,
		Interval: interval,
	}
}

func (w *MemoryWorker) Start(ctx context.Context) {
	go w.loop(ctx)
}

func (w *MemoryWorker) loop(ctx context.Context) {
	// Initial delay to let the system stabilize
	time.Sleep(30 * time.Second)

	ticker := time.NewTicker(w.Interval)
	defer ticker.Stop()

	w.processConversations(ctx)

	for {
		select {
		case <-ctx.Done():
			log.Println("[memory-worker] stopped")
			return
		case <-ticker.C:
			w.processConversations(ctx)
		}
	}
}

func (w *MemoryWorker) processConversations(ctx context.Context) {
	if !w.running.CompareAndSwap(false, true) {
		return
	}
	defer w.running.Store(false)

	cutoff := time.Now().Add(-5 * time.Minute)

	// Find conversations that need processing
	var conversations []models.Conversation
	w.DB.Where("summarized_at IS NULL AND updated_at < ?", cutoff).
		Find(&conversations)

	if len(conversations) == 0 {
		return
	}

	log.Printf("[memory-worker] processing %d conversations", len(conversations))

	for _, conv := range conversations {
		select {
		case <-ctx.Done():
			return
		default:
		}

		// Check message count
		var count int64
		w.DB.Model(&models.ChatMessage{}).Where("conversation_id = ?", conv.ID).Count(&count)
		if count < 2 {
			// Mark as summarized to skip next time (too short to be useful)
			now := time.Now()
			w.DB.Model(&conv).Update("summarized_at", &now)
			continue
		}

		if err := w.processConversation(ctx, &conv); err != nil {
			log.Printf("[memory-worker] conversation %s failed: %v", conv.ID, err)
			continue
		}
	}
}

func (w *MemoryWorker) processConversation(ctx context.Context, conv *models.Conversation) error {
	// Load messages
	var messages []models.ChatMessage
	w.DB.Where("conversation_id = ? AND role IN ?", conv.ID, []string{"user", "assistant"}).
		Order("created_at asc").
		Find(&messages)

	if len(messages) == 0 {
		return nil
	}

	// Build conversation text for the LLM
	var conversationText string
	for _, m := range messages {
		conversationText += fmt.Sprintf("%s: %s\n", m.Role, m.Content)
	}

	// Call LLM for extraction
	resp, err := w.Client.Chat(minimax.ChatRequest{
		Messages: []minimax.Message{
			{Role: "system", Content: memoryExtractionPrompt},
			{Role: "user", Content: conversationText},
		},
		Temperature: 0.3,
	})
	if err != nil {
		return fmt.Errorf("LLM extraction: %w", err)
	}

	// Parse response
	content := ""
	if len(resp.Choices) > 0 {
		content = resp.Choices[0].Delta.Content
	}
	if content == "" {
		content = resp.Content
	}

	var extraction memoryExtraction
	if err := json.Unmarshal([]byte(content), &extraction); err != nil {
		return fmt.Errorf("parse extraction JSON: %w (raw: %s)", err, content)
	}

	// Store summary in Weaviate
	if extraction.Summary != "" {
		if err := w.Weaviate.UpsertConversationSummary(ctx, conv.ID, int(conv.UserID), extraction.Summary, extraction.AgentUsed); err != nil {
			log.Printf("[memory-worker] summary upsert failed for %s: %v", conv.ID, err)
		}
	}

	// Store/update user facts with dedup
	for _, fact := range extraction.Facts {
		if fact.Content == "" {
			continue
		}

		// Check for similar existing facts
		existing, err := w.Weaviate.SearchUserFacts(ctx, fact.Content, int(conv.UserID), 1)
		if err == nil && len(existing) > 0 && existing[0].Distance < 0.15 {
			// Similar fact exists — update it by upserting with the existing content's UUID
			// The existing fact is close enough, skip creating a duplicate
			continue
		}

		if err := w.Weaviate.UpsertUserFact(ctx, int(conv.UserID), fact.Content, fact.Category); err != nil {
			log.Printf("[memory-worker] fact upsert failed: %v", err)
		}
	}

	// Mark conversation as summarized
	now := time.Now()
	w.DB.Model(conv).Update("summarized_at", &now)

	log.Printf("[memory-worker] processed conversation %s: summary=%d chars, facts=%d",
		conv.ID, len(extraction.Summary), len(extraction.Facts))

	return nil
}
```

- [ ] **Step 2: Verify it compiles**

Run: `cd /home/nst/GolandProjects/pdt/backend && go build ./internal/worker/...`

- [ ] **Step 3: Commit**

```bash
git add backend/internal/worker/memory.go
git commit -m "feat(memory): add background memory extraction worker"
```

---

### Task 7: Smart Router with Memory

**Files:**
- Modify: `backend/internal/ai/agent/orchestrator.go`

- [ ] **Step 1: Add MemoryBlock field to Orchestrator and inject into router prompt**

Add a `MemoryBlock` field to the `Orchestrator` struct:

```go
type Orchestrator struct {
	Client      *minimax.Client
	Agents      map[string]Agent
	MemoryBlock string // pre-formatted memory context
}
```

Update `NewOrchestrator` to accept an optional memory block:

```go
func NewOrchestrator(client *minimax.Client, memoryBlock string, agents ...Agent) *Orchestrator {
	agentMap := make(map[string]Agent)
	for _, a := range agents {
		agentMap[a.Name()] = a
	}
	return &Orchestrator{
		Client:      client,
		Agents:      agentMap,
		MemoryBlock: memoryBlock,
	}
}
```

In `HandleMessage`, update the router system prompt construction (around line 82-85). Replace:

```go
routerMessages := append([]minimax.Message{{
	Role:    "system",
	Content: fmt.Sprintf("Today is %s.\n\n%s", today, routerSystemPrompt),
}}, messages...)
```

With:

```go
systemContent := fmt.Sprintf("Today is %s.\n\n%s", today, routerSystemPrompt)
if o.MemoryBlock != "" {
	systemContent = o.MemoryBlock + "\n\n" + systemContent
}
routerMessages := append([]minimax.Message{{
	Role:    "system",
	Content: systemContent,
}}, messages...)
```

- [ ] **Step 2: Update the routerSystemPrompt to allow direct memory-based answers**

Replace the last paragraph of `routerSystemPrompt` (the line starting "If the user's message is a simple greeting"):

```go
If the user's message is a simple greeting or general question not related to any agent, respond directly without routing.
```

With:

```go
If you can answer the user's question using the conversation history and user context above (e.g., "what did we discuss yesterday?", "remind me about X", general greetings, or questions about the user's own preferences/setup), respond directly without routing to any agent.
```

- [ ] **Step 3: Verify it compiles**

Run: `cd /home/nst/GolandProjects/pdt/backend && go build ./internal/ai/agent/...`

- [ ] **Step 4: Commit**

```bash
git add backend/internal/ai/agent/orchestrator.go
git commit -m "feat(memory): inject memory context into router and improve direct answers"
```

---

### Task 8: Wire Everything Together

**Files:**
- Modify: `backend/internal/handlers/chat.go`
- Modify: `backend/cmd/server/main.go`

- [ ] **Step 1: Update chat.go to fetch memory and wrap agents**

Add import for the memory package:

```go
"github.com/cds-id/pdt/backend/internal/ai/memory"
```

In `HandleWebSocket`, after building the base agents list and before the Composio wrapping, add memory fetching and wrapping. The section that builds agents and the orchestrator should become:

```go
// Build base agents
agents := []agent.Agent{
	&agent.GitAgent{DB: h.DB, UserID: userID, Encryptor: h.Encryptor, Weaviate: h.WeaviateClient},
	&agent.JiraAgent{DB: h.DB, UserID: userID, Weaviate: h.WeaviateClient},
	&agent.ReportAgent{DB: h.DB, UserID: userID, Generator: h.ReportGenerator, R2: h.R2},
	&agent.ProofAgent{DB: h.DB, UserID: userID},
	&agent.BriefingAgent{DB: h.DB, UserID: userID},
	&agent.WhatsAppAgent{DB: h.DB, UserID: userID, Weaviate: h.WeaviateClient, Manager: h.WaManager},
	&agent.SchedulerAgent{DB: h.DB, UserID: userID, Engine: h.ScheduleEngine},
}

// Wrap with Composio tools if user has it configured
if h.ComposioClient != nil {
	agents = composio.WrapAgents(h.DB, h.Encryptor, h.ComposioClient, userID, agents)
}
```

This stays at the WebSocket connection level (once per connection). But the memory context needs to be fetched per-message since it depends on the user's message content for semantic search. Move the orchestrator creation inside the message loop.

Find the current orchestrator creation and the message loop. The orchestrator should now be created per-message, inside the `for` loop, after `messages` are built. Replace the orchestrator creation with this pattern:

After the `messages` conversion from history (around line 210-218), add:

```go
// Fetch memory context for this message
mc := memory.FetchMemoryContext(context.Background(), h.WeaviateClient, userID, msg.Content)

// Wrap agents with memory
memoryAgents := memory.WrapAgents(agents, mc)

// Create orchestrator with memory block
orchestrator := agent.NewOrchestrator(h.MiniMaxClient, mc.FormatForPrompt(), memoryAgents...)
```

And remove the old `orchestrator` creation that was outside the loop.

**Important:** The orchestrator creation moves from the WebSocket connection setup (once) to inside the message processing loop (per message). The `agents` slice (base + composio) is still built once at connection time. Only the memory wrapping and orchestrator creation happen per message.

- [ ] **Step 2: Update main.go — start memory worker**

After the worker scheduler `Start(ctx)` block (around line 97), add:

```go
// Memory worker
if miniMaxClient != nil && weaviateClient != nil && weaviateClient.IsAvailable() {
	memWorker := worker.NewMemoryWorker(db, miniMaxClient, weaviateClient, cfg.MemoryWorkerInterval)
	memWorker.Start(ctx)
	log.Printf("Memory worker started: interval=%s", cfg.MemoryWorkerInterval)
}
```

Add import if not already present:
```go
// worker package is already imported
```

- [ ] **Step 3: Update main.go — update NewOrchestrator calls**

The `NewOrchestrator` signature changed to include a `memoryBlock` parameter. Update the existing call in the scheduler's `scheduleEngine` setup. The agents registered directly in `NewEngine` (lines 165-171) don't need memory (they're templates). But the `SetAgentBuilder` closure needs updating.

In the `SetAgentBuilder`, add memory context:

```go
scheduleEngine.SetAgentBuilder(func(userID uint) []agent.Agent {
	agents := []agent.Agent{
		&agent.GitAgent{DB: db, UserID: userID, Encryptor: encryptor, Weaviate: weaviateClient},
		&agent.JiraAgent{DB: db, UserID: userID, Weaviate: weaviateClient},
		&agent.ReportAgent{DB: db, UserID: userID, Generator: reportGen, R2: r2Client},
		&agent.ProofAgent{DB: db, UserID: userID},
		&agent.BriefingAgent{DB: db, UserID: userID},
		&agent.WhatsAppAgent{DB: db, UserID: userID, Weaviate: weaviateClient, Manager: waManager},
		&agent.SchedulerAgent{DB: db, UserID: userID, Engine: scheduleEngine},
	}
	agents = composio.WrapAgents(db, encryptor, composioClient, userID, agents)

	// Add memory context to scheduled agent runs
	if weaviateClient != nil && weaviateClient.IsAvailable() {
		mc := memory.FetchMemoryContext(context.Background(), weaviateClient, userID, "")
		agents = memory.WrapAgents(agents, mc)
	}

	return agents
})
```

Add import for memory package in main.go:
```go
"github.com/cds-id/pdt/backend/internal/ai/memory"
```

- [ ] **Step 4: Update any other NewOrchestrator calls**

Search for other `NewOrchestrator` calls. The scheduler engine also creates orchestrators internally. Check `backend/internal/scheduler/engine.go` for `NewOrchestrator` calls and add `""` as the memory block parameter where needed.

In `scheduler/engine.go`, find any `agent.NewOrchestrator(...)` calls and add `""` as the second parameter (scheduled runs get memory via the agent decorator, not the orchestrator).

- [ ] **Step 5: Update Telegram bot if it calls NewOrchestrator**

Check `backend/internal/services/telegram/bot.go` for `NewOrchestrator` calls and add `""` as the second parameter.

- [ ] **Step 6: Verify it compiles**

Run: `cd /home/nst/GolandProjects/pdt/backend && go build ./...`

- [ ] **Step 7: Commit**

```bash
git add backend/internal/handlers/chat.go backend/cmd/server/main.go backend/internal/scheduler/engine.go backend/internal/services/telegram/bot.go
git commit -m "feat(memory): wire memory worker, context injection, and agent wrapping"
```

---

### Task 9: Build Verification

- [ ] **Step 1: Verify backend compiles**

Run: `cd /home/nst/GolandProjects/pdt/backend && go build ./...`
Expected: No errors.

- [ ] **Step 2: Verify no import cycles**

Run: `cd /home/nst/GolandProjects/pdt/backend && go vet ./...`
Expected: No errors. The dependency graph should be:
- `memory` imports from `agent` (for interface) and `weaviate` (for search)
- `agent` does NOT import `memory`
- `handlers` imports `memory` for `FetchMemoryContext` and `WrapAgents`

- [ ] **Step 3: Commit any fixes**

```bash
git add -A
git commit -m "fix(memory): resolve build issues"
```

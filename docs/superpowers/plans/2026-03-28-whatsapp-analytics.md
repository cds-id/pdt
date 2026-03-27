# WhatsApp Analytics Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Add WhatsApp analytics to PDT — multi-number pairing via whatsmeow, whitelist-based listeners, message capture with R2 media storage, Weaviate vector search with Gemini embeddings, a new WhatsApp AI agent, and an outbox approval workflow.

**Architecture:** Monolith extension of the existing PDT Go backend. WhatsApp connections run as managed goroutines. A new WhatsAppAgent (6th) joins the orchestrator. Weaviate client is a new service called by the agent. Frontend extends Settings page and adds an Outbox page.

**Tech Stack:** Go 1.24, whatsmeow (WhatsApp Web bridge), Weaviate Go client v4, Gemini embeddings via text2vec-palm, React 18 + RTK Query, existing MySQL/R2/Gin/WebSocket stack.

**Spec:** `docs/superpowers/specs/2026-03-28-whatsapp-analytics-design.md`

---

## File Structure

### Backend — New Files

| File | Responsibility |
|------|---------------|
| `backend/internal/models/whatsapp.go` | GORM models: WaNumber, WaListener, WaMessage, WaMedia, WaOutbox |
| `backend/internal/services/whatsapp/manager.go` | Connection manager: multi-number whatsmeow lifecycle, reconnect, shutdown |
| `backend/internal/services/whatsapp/handler.go` | Message handler: incoming message processing, listener filtering, media download |
| `backend/internal/services/whatsapp/sender.go` | Outbox sender worker: polls approved messages, sends via whatsmeow |
| `backend/internal/services/weaviate/client.go` | Weaviate client: collection schema, upsert embeddings, semantic search |
| `backend/internal/services/weaviate/worker.go` | Embedding worker: buffered channel, batch upsert, retry logic |
| `backend/internal/ai/agent/whatsapp.go` | WhatsApp AI agent: system prompt, 7 tools, tool handlers |
| `backend/internal/handlers/whatsapp.go` | REST handlers: numbers CRUD, listeners CRUD, messages, outbox, QR pairing WS |
| `frontend/src/infrastructure/services/whatsapp.service.ts` | RTK Query endpoints for WhatsApp API |
| `frontend/src/infrastructure/constants/whatsapp.constants.ts` | API endpoint constants for WhatsApp |
| `frontend/src/domain/whatsapp/interfaces/whatsapp.interface.ts` | TypeScript interfaces for WhatsApp entities |
| `frontend/src/presentation/pages/OutboxPage.tsx` | Outbox approval page |
| `frontend/src/presentation/components/whatsapp/NumberManager.tsx` | Number cards + add number UI |
| `frontend/src/presentation/components/whatsapp/ListenerManager.tsx` | Listener cards per number |
| `frontend/src/presentation/components/whatsapp/QrPairingModal.tsx` | QR code pairing modal |

### Backend — Modified Files

| File | Change |
|------|--------|
| `backend/internal/config/config.go` | Add `GeminiAPIKey`, `WeaviateURL` fields |
| `backend/internal/database/database.go` | Add WA models to `Migrate()` |
| `backend/cmd/server/main.go` | Initialize WA manager, weaviate client, embedding worker, sender worker, WA handler, routes |
| `backend/internal/ai/agent/orchestrator.go` | Add "whatsapp" to router enum and system prompt |
| `backend/internal/handlers/chat.go` | Add WhatsAppAgent to orchestrator construction |
| `backend/go.mod` | Add whatsmeow, weaviate-go-client dependencies |
| `docker-compose.yml` | Add weaviate service |
| `frontend/src/infrastructure/services/api.ts` | Add 'WhatsApp' to tagTypes |
| `frontend/src/infrastructure/constants/api.constants.ts` | Add WA endpoint constants |
| `frontend/src/presentation/routes/index.tsx` | Add outbox route |
| `frontend/src/presentation/pages/SettingsPage.tsx` | Add WhatsApp section |

---

## Task 1: Add Go Dependencies

**Files:**
- Modify: `backend/go.mod`

- [ ] **Step 1: Add whatsmeow and weaviate dependencies**

```bash
cd /home/nst/GolandProjects/pdt/backend && go get go.mau.fi/whatsmeow@latest && go get go.mau.fi/whatsmeow/store/sqlstore@latest && go get github.com/weaviate/weaviate-go-client/v4@latest && go get github.com/mattn/go-sqlite3@latest
```

Note: whatsmeow uses sqlstore for device session persistence. We use MySQL-backed sqlstore (via the `sqlstore` package with a MySQL container). However, whatsmeow's sqlstore natively supports `sqlite3` and `pgx`. For MySQL, we'll store the device session as a serialized blob in our `wa_numbers` table instead.

- [ ] **Step 2: Tidy modules**

```bash
cd /home/nst/GolandProjects/pdt/backend && go mod tidy
```

- [ ] **Step 3: Verify build**

```bash
cd /home/nst/GolandProjects/pdt/backend && go build ./...
```

Expected: Build succeeds with no errors.

- [ ] **Step 4: Commit**

```bash
cd /home/nst/GolandProjects/pdt && git add backend/go.mod backend/go.sum && git commit -m "feat(wa): add whatsmeow and weaviate-go-client dependencies"
```

---

## Task 2: GORM Models

**Files:**
- Create: `backend/internal/models/whatsapp.go`
- Modify: `backend/internal/database/database.go`

- [ ] **Step 1: Create WhatsApp models**

Create `backend/internal/models/whatsapp.go`:

```go
package models

import "time"

type WaNumber struct {
	ID          uint       `gorm:"primarykey" json:"id"`
	UserID      uint       `gorm:"index;not null" json:"user_id"`
	PhoneNumber string     `gorm:"type:varchar(30);not null" json:"phone_number"`
	DisplayName string     `gorm:"type:varchar(100)" json:"display_name"`
	DeviceStore []byte     `gorm:"type:longblob" json:"-"`
	Status      string     `gorm:"type:varchar(20);default:disconnected" json:"status"`
	PairedAt    *time.Time `json:"paired_at,omitempty"`
	CreatedAt   time.Time  `json:"created_at"`
	UpdatedAt   time.Time  `json:"updated_at"`
	User        User       `gorm:"foreignKey:UserID" json:"-"`
}

type WaListener struct {
	ID         uint      `gorm:"primarykey" json:"id"`
	WaNumberID uint      `gorm:"index;not null" json:"wa_number_id"`
	JID        string    `gorm:"type:varchar(100);not null" json:"jid"`
	Name       string    `gorm:"type:varchar(200);not null" json:"name"`
	Type       string    `gorm:"type:varchar(20);not null" json:"type"`
	IsActive   bool      `gorm:"default:true" json:"is_active"`
	CreatedAt  time.Time `json:"created_at"`
	UpdatedAt  time.Time `json:"updated_at"`
	WaNumber   WaNumber  `gorm:"foreignKey:WaNumberID" json:"-"`
}

type WaMessage struct {
	ID           uint      `gorm:"primarykey" json:"id"`
	WaListenerID uint      `gorm:"index;not null" json:"wa_listener_id"`
	MessageID    string    `gorm:"type:varchar(100);uniqueIndex;not null" json:"message_id"`
	SenderJID    string    `gorm:"type:varchar(100);not null" json:"sender_jid"`
	SenderName   string    `gorm:"type:varchar(200)" json:"sender_name"`
	Content      string    `gorm:"type:longtext" json:"content"`
	MessageType  string    `gorm:"type:varchar(20);default:text" json:"message_type"`
	HasMedia     bool      `gorm:"default:false" json:"has_media"`
	Timestamp    time.Time `gorm:"index;not null" json:"timestamp"`
	CreatedAt    time.Time `json:"created_at"`
	WaListener   WaListener `gorm:"foreignKey:WaListenerID" json:"-"`
}

type WaMedia struct {
	ID          uint      `gorm:"primarykey" json:"id"`
	WaMessageID uint      `gorm:"index;not null" json:"wa_message_id"`
	FileName    string    `gorm:"type:varchar(255)" json:"file_name"`
	MimeType    string    `gorm:"type:varchar(100)" json:"mime_type"`
	FileSize    int64     `json:"file_size"`
	R2Key       string    `gorm:"type:varchar(500)" json:"r2_key"`
	FileURL     string    `gorm:"type:varchar(500)" json:"file_url"`
	CreatedAt   time.Time `json:"created_at"`
	WaMessage   WaMessage `gorm:"foreignKey:WaMessageID" json:"-"`
}

type WaOutbox struct {
	ID          uint       `gorm:"primarykey" json:"id"`
	WaNumberID  uint       `gorm:"index;not null" json:"wa_number_id"`
	TargetJID   string     `gorm:"type:varchar(100);not null" json:"target_jid"`
	TargetName  string     `gorm:"type:varchar(200)" json:"target_name"`
	Content     string     `gorm:"type:text;not null" json:"content"`
	Status      string     `gorm:"type:varchar(20);default:pending;index" json:"status"`
	RequestedBy string    `gorm:"type:varchar(20);default:agent" json:"requested_by"`
	Context     string     `gorm:"type:text" json:"context"`
	ApprovedAt  *time.Time `json:"approved_at,omitempty"`
	SentAt      *time.Time `json:"sent_at,omitempty"`
	CreatedAt   time.Time  `json:"created_at"`
	WaNumber    WaNumber   `gorm:"foreignKey:WaNumberID" json:"-"`
}
```

- [ ] **Step 2: Add models to AutoMigrate**

In `backend/internal/database/database.go`, add the 5 new models to the `Migrate` function:

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
		&models.JiraComment{},
		&models.WaNumber{},
		&models.WaListener{},
		&models.WaMessage{},
		&models.WaMedia{},
		&models.WaOutbox{},
	)
}
```

- [ ] **Step 3: Verify build**

```bash
cd /home/nst/GolandProjects/pdt/backend && go build ./...
```

Expected: Build succeeds.

- [ ] **Step 4: Commit**

```bash
cd /home/nst/GolandProjects/pdt && git add backend/internal/models/whatsapp.go backend/internal/database/database.go && git commit -m "feat(wa): add GORM models for WhatsApp tables"
```

---

## Task 3: Configuration

**Files:**
- Modify: `backend/internal/config/config.go`

- [ ] **Step 1: Add WhatsApp and Weaviate config fields**

Add to the `Config` struct after the MiniMax fields:

```go
	// WhatsApp & Weaviate
	GeminiAPIKey string
	WeaviateURL  string
```

Add to the `Load()` function after `cfg.AIContextWindow = aiContextWindow`:

```go
	cfg.GeminiAPIKey = getEnv("GEMINI_API_KEY", "")
	cfg.WeaviateURL = getEnv("WEAVIATE_URL", "http://localhost:8081")
```

- [ ] **Step 2: Verify build**

```bash
cd /home/nst/GolandProjects/pdt/backend && go build ./...
```

- [ ] **Step 3: Commit**

```bash
cd /home/nst/GolandProjects/pdt && git add backend/internal/config/config.go && git commit -m "feat(wa): add Gemini and Weaviate config fields"
```

---

## Task 4: Weaviate Client Service

**Files:**
- Create: `backend/internal/services/weaviate/client.go`

- [ ] **Step 1: Create Weaviate client with schema creation and search**

Create `backend/internal/services/weaviate/client.go`:

```go
package weaviate

import (
	"context"
	"fmt"
	"log"
	"time"

	"github.com/weaviate/weaviate-go-client/v4/weaviate"
	"github.com/weaviate/weaviate-go-client/v4/weaviate/graphql"
	"github.com/weaviate/weaviate/entities/models"
)

const className = "WaMessageEmbedding"

type Client struct {
	client    *weaviate.Client
	apiKey    string
	available bool
}

type SearchResult struct {
	MessageID  float64 `json:"message_id"`
	ListenerID float64 `json:"listener_id"`
	Content    string  `json:"content"`
	SenderName string  `json:"sender_name"`
	Timestamp  string  `json:"timestamp"`
	Distance   float32 `json:"distance"`
}

func NewClient(url, geminiAPIKey string) *Client {
	cfg := weaviate.Config{
		Host:   url,
		Scheme: "http",
		Headers: map[string]string{
			"X-Google-Api-Key": geminiAPIKey,
		},
	}

	client, err := weaviate.NewClient(cfg)
	if err != nil {
		log.Printf("[weaviate] failed to create client: %v", err)
		return &Client{available: false}
	}

	c := &Client{
		client:    client,
		apiKey:    geminiAPIKey,
		available: true,
	}

	if err := c.ensureSchema(context.Background()); err != nil {
		log.Printf("[weaviate] schema setup failed: %v, vector search disabled", err)
		c.available = false
	}

	return c
}

func (c *Client) IsAvailable() bool {
	return c.available
}

func (c *Client) ensureSchema(ctx context.Context) error {
	exists, err := c.client.Schema().ClassExistenceChecker().WithClassName(className).Do(ctx)
	if err != nil {
		return fmt.Errorf("check class existence: %w", err)
	}
	if exists {
		return nil
	}

	class := &models.Class{
		Class:      className,
		Vectorizer: "text2vec-palm",
		ModuleConfig: map[string]any{
			"text2vec-palm": map[string]any{
				"projectId":    "",
				"modelId":      "text-embedding-004",
				"vectorizeClassName": false,
			},
		},
		Properties: []*models.Property{
			{Name: "message_id", DataType: []string{"int"}},
			{Name: "listener_id", DataType: []string{"int"}},
			{Name: "user_id", DataType: []string{"int"}},
			{Name: "content", DataType: []string{"text"}},
			{Name: "sender_name", DataType: []string{"text"}, ModuleConfig: map[string]any{
				"text2vec-palm": map[string]any{"skip": true},
			}},
			{Name: "timestamp", DataType: []string{"date"}},
		},
	}

	return c.client.Schema().ClassCreator().WithClass(class).Do(ctx)
}

func (c *Client) Upsert(ctx context.Context, messageID, listenerID, userID int, content, senderName string, timestamp time.Time) error {
	if !c.available {
		return nil
	}

	id := deterministicUUID(messageID)

	_, err := c.client.Data().Creator().
		WithClassName(className).
		WithID(id).
		WithProperties(map[string]any{
			"message_id":  messageID,
			"listener_id": listenerID,
			"user_id":     userID,
			"content":     content,
			"sender_name": senderName,
			"timestamp":   timestamp.Format(time.RFC3339),
		}).
		Do(ctx)

	return err
}

func (c *Client) Search(ctx context.Context, query string, userID int, listenerID *int, startDate, endDate *time.Time, limit int) ([]SearchResult, error) {
	if !c.available {
		return nil, nil
	}

	if limit <= 0 {
		limit = 10
	}

	fields := []graphql.Field{
		{Name: "message_id"},
		{Name: "listener_id"},
		{Name: "content"},
		{Name: "sender_name"},
		{Name: "timestamp"},
		{Name: "_additional { distance }"},
	}

	where := c.client.GraphQL().NearTextArgBuilder().WithConcepts([]string{query})

	builder := c.client.GraphQL().Get().
		WithClassName(className).
		WithFields(fields...).
		WithNearText(where).
		WithLimit(limit)

	// Add where filter for user_id (and optionally listener_id, dates)
	filters := buildFilters(userID, listenerID, startDate, endDate)
	if filters != nil {
		builder = builder.WithWhere(filters)
	}

	result, err := builder.Do(ctx)
	if err != nil {
		return nil, fmt.Errorf("weaviate search: %w", err)
	}

	return parseSearchResults(result), nil
}

func deterministicUUID(messageID int) string {
	return fmt.Sprintf("%08x-0000-0000-0000-%012x", messageID>>32, messageID&0xFFFFFFFFFFFF)
}

func buildFilters(userID int, listenerID *int, startDate, endDate *time.Time) *graphql.WhereArgumentBuilder {
	operands := []*graphql.WhereArgumentBuilder{
		(&graphql.WhereArgumentBuilder{}).
			WithPath([]string{"user_id"}).
			WithOperator(graphql.Equal).
			WithValueInt(int64(userID)),
	}

	if listenerID != nil {
		operands = append(operands, (&graphql.WhereArgumentBuilder{}).
			WithPath([]string{"listener_id"}).
			WithOperator(graphql.Equal).
			WithValueInt(int64(*listenerID)))
	}

	if startDate != nil {
		operands = append(operands, (&graphql.WhereArgumentBuilder{}).
			WithPath([]string{"timestamp"}).
			WithOperator(graphql.GreaterThanEqual).
			WithValueDate(startDate.Format(time.RFC3339)))
	}

	if endDate != nil {
		operands = append(operands, (&graphql.WhereArgumentBuilder{}).
			WithPath([]string{"timestamp"}).
			WithOperator(graphql.LessThanEqual).
			WithValueDate(endDate.Format(time.RFC3339)))
	}

	if len(operands) == 1 {
		return operands[0]
	}

	return (&graphql.WhereArgumentBuilder{}).
		WithOperator(graphql.And).
		WithOperands(operands)
}

func parseSearchResults(result *models.GraphQLResponse) []SearchResult {
	if result == nil || result.Data == nil {
		return nil
	}

	get, ok := result.Data["Get"].(map[string]any)
	if !ok {
		return nil
	}

	items, ok := get[className].([]any)
	if !ok {
		return nil
	}

	var results []SearchResult
	for _, item := range items {
		m, ok := item.(map[string]any)
		if !ok {
			continue
		}
		r := SearchResult{
			Content:    strVal(m, "content"),
			SenderName: strVal(m, "sender_name"),
			Timestamp:  strVal(m, "timestamp"),
		}
		if v, ok := m["message_id"].(float64); ok {
			r.MessageID = v
		}
		if v, ok := m["listener_id"].(float64); ok {
			r.ListenerID = v
		}
		if add, ok := m["_additional"].(map[string]any); ok {
			if d, ok := add["distance"].(float64); ok {
				r.Distance = float32(d)
			}
		}
		results = append(results, r)
	}
	return results
}

func strVal(m map[string]any, key string) string {
	if v, ok := m[key].(string); ok {
		return v
	}
	return ""
}
```

- [ ] **Step 2: Verify build**

```bash
cd /home/nst/GolandProjects/pdt/backend && go build ./...
```

- [ ] **Step 3: Commit**

```bash
cd /home/nst/GolandProjects/pdt && git add backend/internal/services/weaviate/ && git commit -m "feat(wa): add Weaviate client service with schema setup and semantic search"
```

---

## Task 5: Embedding Worker

**Files:**
- Create: `backend/internal/services/weaviate/worker.go`

- [ ] **Step 1: Create embedding worker with buffered channel and batch processing**

Create `backend/internal/services/weaviate/worker.go`:

```go
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
	client  *Client
	db      *gorm.DB
	queue   chan EmbedRequest
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
```

- [ ] **Step 2: Verify build**

```bash
cd /home/nst/GolandProjects/pdt/backend && go build ./...
```

- [ ] **Step 3: Commit**

```bash
cd /home/nst/GolandProjects/pdt && git add backend/internal/services/weaviate/worker.go && git commit -m "feat(wa): add embedding worker with batch processing and backfill"
```

---

## Task 6: WhatsApp Connection Manager

**Files:**
- Create: `backend/internal/services/whatsapp/manager.go`

- [ ] **Step 1: Create connection manager with multi-number lifecycle**

Create `backend/internal/services/whatsapp/manager.go`:

```go
package whatsapp

import (
	"context"
	"fmt"
	"log"
	"sync"
	"time"

	"go.mau.fi/whatsmeow"
	"go.mau.fi/whatsmeow/store"
	"go.mau.fi/whatsmeow/store/sqlstore"
	waProto "go.mau.fi/whatsmeow/binary/proto"
	waLog "go.mau.fi/whatsmeow/util/log"
	"gorm.io/gorm"

	"github.com/cds-id/pdt/backend/internal/models"
	"github.com/cds-id/pdt/backend/internal/services/storage"
	wvClient "github.com/cds-id/pdt/backend/internal/services/weaviate"
)

type Manager struct {
	DB              *gorm.DB
	R2              *storage.R2Client
	EmbeddingWorker *wvClient.EmbeddingWorker
	clients         map[uint]*whatsmeow.Client
	devices         map[uint]*store.Device
	mu              sync.RWMutex
	container       *sqlstore.Container
}

func NewManager(db *gorm.DB, r2 *storage.R2Client, ew *wvClient.EmbeddingWorker) (*Manager, error) {
	// whatsmeow uses its own SQLite store for device sessions
	container, err := sqlstore.New("sqlite3", "file:whatsmeow.db?_foreign_keys=on", waLog.Noop)
	if err != nil {
		return nil, fmt.Errorf("create sqlstore: %w", err)
	}

	return &Manager{
		DB:              db,
		R2:              r2,
		EmbeddingWorker: ew,
		clients:         make(map[uint]*whatsmeow.Client),
		devices:         make(map[uint]*store.Device),
		container:       container,
	}, nil
}

func (m *Manager) Start(ctx context.Context) {
	var numbers []models.WaNumber
	m.DB.Where("status = ?", "connected").Find(&numbers)

	for _, num := range numbers {
		go m.connectNumber(ctx, num.ID)
	}

	log.Printf("[wa-manager] started with %d numbers", len(numbers))
}

func (m *Manager) Shutdown() {
	m.mu.Lock()
	defer m.mu.Unlock()

	for id, client := range m.clients {
		client.Disconnect()
		log.Printf("[wa-manager] disconnected number %d", id)
	}
	m.clients = make(map[uint]*whatsmeow.Client)
}

func (m *Manager) connectNumber(ctx context.Context, numberID uint) {
	devices, err := m.container.GetAllDevices()
	if err != nil {
		log.Printf("[wa-manager] get devices error: %v", err)
		return
	}

	var device *store.Device
	for _, d := range devices {
		if d.ID != nil {
			// Try to match by stored JID — for reconnection
			device = d
			break
		}
	}

	if device == nil {
		device = m.container.NewDevice()
	}

	client := whatsmeow.NewClient(device, waLog.Noop)
	handler := NewMessageHandler(m.DB, m.R2, m.EmbeddingWorker, numberID)
	client.AddEventHandler(handler.HandleEvent)

	if client.Store.ID == nil {
		// Need pairing — skip auto-connect, will be done via QR
		log.Printf("[wa-manager] number %d needs pairing", numberID)
		m.DB.Model(&models.WaNumber{}).Where("id = ?", numberID).Update("status", "disconnected")
		return
	}

	err = client.Connect()
	if err != nil {
		log.Printf("[wa-manager] connect error for number %d: %v", numberID, err)
		m.reconnectWithBackoff(ctx, numberID, client)
		return
	}

	m.mu.Lock()
	m.clients[numberID] = client
	m.devices[numberID] = device
	m.mu.Unlock()

	log.Printf("[wa-manager] connected number %d", numberID)
}

func (m *Manager) reconnectWithBackoff(ctx context.Context, numberID uint, client *whatsmeow.Client) {
	delays := []time.Duration{5 * time.Second, 10 * time.Second, 30 * time.Second, 60 * time.Second, 60 * time.Second}

	for attempt, delay := range delays {
		select {
		case <-ctx.Done():
			return
		case <-time.After(delay):
		}

		err := client.Connect()
		if err == nil {
			m.mu.Lock()
			m.clients[numberID] = client
			m.mu.Unlock()
			m.DB.Model(&models.WaNumber{}).Where("id = ?", numberID).Update("status", "connected")
			log.Printf("[wa-manager] reconnected number %d after %d attempts", numberID, attempt+1)
			return
		}
		log.Printf("[wa-manager] reconnect attempt %d for number %d failed: %v", attempt+1, numberID, err)
	}

	m.DB.Model(&models.WaNumber{}).Where("id = ?", numberID).Update("status", "disconnected")
	log.Printf("[wa-manager] number %d marked disconnected after max retries", numberID)
}

func (m *Manager) GetClient(numberID uint) (*whatsmeow.Client, bool) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	c, ok := m.clients[numberID]
	return c, ok
}

func (m *Manager) GetContainer() *sqlstore.Container {
	return m.container
}

func (m *Manager) RegisterClient(numberID uint, client *whatsmeow.Client) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.clients[numberID] = client
}

func (m *Manager) RemoveClient(numberID uint) {
	m.mu.Lock()
	defer m.mu.Unlock()
	if client, ok := m.clients[numberID]; ok {
		client.Disconnect()
		delete(m.clients, numberID)
	}
}

func (m *Manager) SendMessage(ctx context.Context, numberID uint, jid string, text string) error {
	client, ok := m.GetClient(numberID)
	if !ok {
		return fmt.Errorf("number %d not connected", numberID)
	}

	targetJID, err := parseJID(jid)
	if err != nil {
		return err
	}

	_, err = client.SendMessage(ctx, targetJID, &waProto.Message{
		Conversation: &text,
	})
	return err
}
```

Note: The `parseJID` helper and `NewMessageHandler` will be in `handler.go` (Task 7). This file may not compile yet until Task 7 is done.

- [ ] **Step 2: Commit (partial — will compile after Task 7)**

```bash
cd /home/nst/GolandProjects/pdt && git add backend/internal/services/whatsapp/manager.go && git commit -m "feat(wa): add WhatsApp connection manager with multi-number lifecycle"
```

---

## Task 7: Message Handler

**Files:**
- Create: `backend/internal/services/whatsapp/handler.go`

- [ ] **Step 1: Create message handler for incoming messages**

Create `backend/internal/services/whatsapp/handler.go`:

```go
package whatsapp

import (
	"context"
	"fmt"
	"log"
	"time"

	"go.mau.fi/whatsmeow"
	"go.mau.fi/whatsmeow/types"
	"go.mau.fi/whatsmeow/types/events"
	"gorm.io/gorm"

	"github.com/cds-id/pdt/backend/internal/models"
	"github.com/cds-id/pdt/backend/internal/services/storage"
	wvClient "github.com/cds-id/pdt/backend/internal/services/weaviate"
)

type MessageHandler struct {
	DB              *gorm.DB
	R2              *storage.R2Client
	EmbeddingWorker *wvClient.EmbeddingWorker
	NumberID        uint
}

func NewMessageHandler(db *gorm.DB, r2 *storage.R2Client, ew *wvClient.EmbeddingWorker, numberID uint) *MessageHandler {
	return &MessageHandler{
		DB:              db,
		R2:              r2,
		EmbeddingWorker: ew,
		NumberID:        numberID,
	}
}

func (h *MessageHandler) HandleEvent(evt interface{}) {
	switch v := evt.(type) {
	case *events.Message:
		h.handleMessage(v)
	case *events.Connected:
		log.Printf("[wa-handler] number %d connected", h.NumberID)
		h.DB.Model(&models.WaNumber{}).Where("id = ?", h.NumberID).Update("status", "connected")
	case *events.Disconnected:
		log.Printf("[wa-handler] number %d disconnected", h.NumberID)
		h.DB.Model(&models.WaNumber{}).Where("id = ?", h.NumberID).Update("status", "disconnected")
	}
}

func (h *MessageHandler) handleMessage(evt *events.Message) {
	senderJID := evt.Info.Sender.String()
	chatJID := evt.Info.Chat.String()

	// Check if chat is a registered active listener
	var listener models.WaListener
	err := h.DB.Where("wa_number_id = ? AND jid = ? AND is_active = ?", h.NumberID, chatJID, true).First(&listener).Error
	if err != nil {
		// Not a registered listener — drop
		return
	}

	// Get user_id for this number
	var number models.WaNumber
	h.DB.Select("user_id").Where("id = ?", h.NumberID).First(&number)

	// Extract text content
	content := extractText(evt.Message)
	if content == "" && !hasMedia(evt) {
		return
	}

	// Determine message type
	msgType := "text"
	hasMediaFlag := false
	if hasMedia(evt) {
		msgType = detectMediaType(evt)
		hasMediaFlag = true
	}

	senderName := evt.Info.PushName

	// Save message
	msg := models.WaMessage{
		WaListenerID: listener.ID,
		MessageID:    evt.Info.ID,
		SenderJID:    senderJID,
		SenderName:   senderName,
		Content:      content,
		MessageType:  msgType,
		HasMedia:     hasMediaFlag,
		Timestamp:    evt.Info.Timestamp,
	}

	if err := h.DB.Create(&msg).Error; err != nil {
		log.Printf("[wa-handler] save message error: %v", err)
		return
	}

	// Async: embed text into Weaviate
	if content != "" && h.EmbeddingWorker != nil {
		h.EmbeddingWorker.Enqueue(wvClient.EmbedRequest{
			MessageID:  msg.ID,
			ListenerID: listener.ID,
			UserID:     number.UserID,
			Content:    content,
			SenderName: senderName,
			Timestamp:  evt.Info.Timestamp,
		})
	}

	// Async: download and upload media to R2
	if hasMediaFlag && h.R2 != nil {
		go h.handleMedia(evt, msg.ID)
	}
}

func (h *MessageHandler) handleMedia(evt *events.Message, messageID uint) {
	// Download media from WhatsApp
	data, err := downloadMedia(evt)
	if err != nil {
		log.Printf("[wa-handler] media download error: %v", err)
		return
	}

	mimeType, fileName := mediaInfo(evt)
	r2Key := fmt.Sprintf("wa-media/%d/%d/%s", h.NumberID, messageID, fileName)

	fileURL, err := h.R2.Upload(context.Background(), r2Key, data, mimeType)
	if err != nil {
		log.Printf("[wa-handler] r2 upload error: %v", err)
		return
	}

	media := models.WaMedia{
		WaMessageID: messageID,
		FileName:    fileName,
		MimeType:    mimeType,
		FileSize:    int64(len(data)),
		R2Key:       r2Key,
		FileURL:     fileURL,
	}
	h.DB.Create(&media)
}

func extractText(msg *waProto.Message) string {
	if msg == nil {
		return ""
	}
	if msg.Conversation != nil {
		return *msg.Conversation
	}
	if msg.ExtendedTextMessage != nil && msg.ExtendedTextMessage.Text != nil {
		return *msg.ExtendedTextMessage.Text
	}
	if msg.ImageMessage != nil && msg.ImageMessage.Caption != nil {
		return *msg.ImageMessage.Caption
	}
	if msg.VideoMessage != nil && msg.VideoMessage.Caption != nil {
		return *msg.VideoMessage.Caption
	}
	if msg.DocumentMessage != nil && msg.DocumentMessage.Caption != nil {
		return *msg.DocumentMessage.Caption
	}
	return ""
}

func hasMedia(evt *events.Message) bool {
	msg := evt.Message
	return msg.ImageMessage != nil || msg.VideoMessage != nil || msg.AudioMessage != nil || msg.DocumentMessage != nil
}

func detectMediaType(evt *events.Message) string {
	msg := evt.Message
	if msg.ImageMessage != nil {
		return "image"
	}
	if msg.VideoMessage != nil {
		return "video"
	}
	if msg.AudioMessage != nil {
		return "audio"
	}
	if msg.DocumentMessage != nil {
		return "document"
	}
	return "text"
}

func downloadMedia(evt *events.Message) ([]byte, error) {
	// whatsmeow provides Download() on media messages
	// The actual download depends on the message type
	msg := evt.Message
	if msg.ImageMessage != nil {
		return whatsmeow.Download(evt.RawMessage, msg.ImageMessage)
	}
	if msg.VideoMessage != nil {
		return whatsmeow.Download(evt.RawMessage, msg.VideoMessage)
	}
	if msg.AudioMessage != nil {
		return whatsmeow.Download(evt.RawMessage, msg.AudioMessage)
	}
	if msg.DocumentMessage != nil {
		return whatsmeow.Download(evt.RawMessage, msg.DocumentMessage)
	}
	return nil, fmt.Errorf("no downloadable media")
}

func mediaInfo(evt *events.Message) (mimeType, fileName string) {
	msg := evt.Message
	if msg.ImageMessage != nil {
		return ptrStr(msg.ImageMessage.Mimetype, "image/jpeg"), fmt.Sprintf("%s.jpg", evt.Info.ID)
	}
	if msg.VideoMessage != nil {
		return ptrStr(msg.VideoMessage.Mimetype, "video/mp4"), fmt.Sprintf("%s.mp4", evt.Info.ID)
	}
	if msg.AudioMessage != nil {
		return ptrStr(msg.AudioMessage.Mimetype, "audio/ogg"), fmt.Sprintf("%s.ogg", evt.Info.ID)
	}
	if msg.DocumentMessage != nil {
		name := fmt.Sprintf("%s", evt.Info.ID)
		if msg.DocumentMessage.FileName != nil {
			name = *msg.DocumentMessage.FileName
		}
		return ptrStr(msg.DocumentMessage.Mimetype, "application/octet-stream"), name
	}
	return "application/octet-stream", evt.Info.ID
}

func ptrStr(p *string, fallback string) string {
	if p != nil {
		return *p
	}
	return fallback
}

func parseJID(jid string) (types.JID, error) {
	parsed, err := types.ParseJID(jid)
	if err != nil {
		return types.JID{}, fmt.Errorf("invalid JID %q: %w", jid, err)
	}
	return parsed, nil
}
```

Note: The `whatsmeow.Download` API may vary by version. The implementer should check the whatsmeow docs for the exact media download API and adapt. The key pattern is: download bytes → upload to R2 → save wa_media record.

- [ ] **Step 2: Verify build**

```bash
cd /home/nst/GolandProjects/pdt/backend && go build ./...
```

Expected: May need adjustments to whatsmeow imports based on the exact version. Fix any import path issues.

- [ ] **Step 3: Commit**

```bash
cd /home/nst/GolandProjects/pdt && git add backend/internal/services/whatsapp/handler.go && git commit -m "feat(wa): add message handler with listener filtering and media upload"
```

---

## Task 8: Outbox Sender Worker

**Files:**
- Create: `backend/internal/services/whatsapp/sender.go`

- [ ] **Step 1: Create sender worker that polls approved outbox messages**

Create `backend/internal/services/whatsapp/sender.go`:

```go
package whatsapp

import (
	"context"
	"log"
	"time"

	"gorm.io/gorm"

	"github.com/cds-id/pdt/backend/internal/models"
)

type SenderWorker struct {
	DB      *gorm.DB
	Manager *Manager
}

func NewSenderWorker(db *gorm.DB, manager *Manager) *SenderWorker {
	return &SenderWorker{DB: db, Manager: manager}
}

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
			w.processApproved()
		}
	}
}

func (w *SenderWorker) processApproved() {
	var outbox []models.WaOutbox
	w.DB.Where("status = ?", "approved").Find(&outbox)

	for _, msg := range outbox {
		err := w.Manager.SendMessage(context.Background(), msg.WaNumberID, msg.TargetJID, msg.Content)
		if err != nil {
			log.Printf("[wa-sender] failed to send outbox %d: %v", msg.ID, err)
			continue
		}

		now := time.Now()
		w.DB.Model(&msg).Updates(map[string]any{
			"status":  "sent",
			"sent_at": now,
		})
		log.Printf("[wa-sender] sent outbox %d to %s", msg.ID, msg.TargetJID)
	}
}
```

- [ ] **Step 2: Verify build**

```bash
cd /home/nst/GolandProjects/pdt/backend && go build ./...
```

- [ ] **Step 3: Commit**

```bash
cd /home/nst/GolandProjects/pdt && git add backend/internal/services/whatsapp/sender.go && git commit -m "feat(wa): add outbox sender worker"
```

---

## Task 9: WhatsApp REST Handlers

**Files:**
- Create: `backend/internal/handlers/whatsapp.go`

- [ ] **Step 1: Create WhatsApp handlers for numbers, listeners, messages, and outbox**

Create `backend/internal/handlers/whatsapp.go`:

```go
package handlers

import (
	"encoding/json"
	"log"
	"net/http"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/gorilla/websocket"
	"gorm.io/gorm"

	"github.com/cds-id/pdt/backend/internal/models"
	"github.com/cds-id/pdt/backend/internal/services/whatsapp"

	"go.mau.fi/whatsmeow"
	waLog "go.mau.fi/whatsmeow/util/log"
)

type WhatsAppHandler struct {
	DB      *gorm.DB
	Manager *whatsapp.Manager
}

// --- Numbers ---

func (h *WhatsAppHandler) ListNumbers(c *gin.Context) {
	userID := c.GetUint("user_id")
	var numbers []models.WaNumber
	h.DB.Where("user_id = ?", userID).Order("created_at desc").Find(&numbers)
	c.JSON(http.StatusOK, numbers)
}

type addNumberRequest struct {
	PhoneNumber string `json:"phone_number" binding:"required"`
	DisplayName string `json:"display_name"`
}

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

func (h *WhatsAppHandler) UpdateNumber(c *gin.Context) {
	userID := c.GetUint("user_id")
	id, _ := strconv.ParseUint(c.Param("id"), 10, 64)

	var number models.WaNumber
	if err := h.DB.Where("id = ? AND user_id = ?", id, userID).First(&number).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "number not found"})
		return
	}

	var req struct {
		DisplayName string `json:"display_name"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	h.DB.Model(&number).Update("display_name", req.DisplayName)
	c.JSON(http.StatusOK, number)
}

func (h *WhatsAppHandler) DeleteNumber(c *gin.Context) {
	userID := c.GetUint("user_id")
	id, _ := strconv.ParseUint(c.Param("id"), 10, 64)

	var number models.WaNumber
	if err := h.DB.Where("id = ? AND user_id = ?", id, userID).First(&number).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "number not found"})
		return
	}

	h.Manager.RemoveClient(uint(id))
	h.DB.Delete(&number)
	c.JSON(http.StatusOK, gin.H{"message": "number disconnected and removed"})
}

// --- QR Pairing WebSocket ---

func (h *WhatsAppHandler) HandlePairing(c *gin.Context) {
	userID := c.GetUint("user_id")
	id, _ := strconv.ParseUint(c.Param("id"), 10, 64)

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

	device := h.Manager.GetContainer().NewDevice()
	client := whatsmeow.NewClient(device, waLog.Noop)

	// Register event handler for this number
	handler := whatsapp.NewMessageHandler(h.DB, nil, nil, uint(id))
	client.AddEventHandler(handler.HandleEvent)

	qrChan, _ := client.GetQRChannel(c.Request.Context())

	err = client.Connect()
	if err != nil {
		conn.WriteJSON(map[string]string{"type": "error", "content": err.Error()})
		return
	}

	for evt := range qrChan {
		switch evt.Event {
		case "code":
			conn.WriteJSON(map[string]string{
				"type":    "qr",
				"content": evt.Code,
			})
		case "success":
			now := time.Now()
			h.DB.Model(&number).Updates(map[string]any{
				"status":    "connected",
				"paired_at": now,
			})
			h.Manager.RegisterClient(uint(id), client)

			conn.WriteJSON(map[string]string{
				"type":    "success",
				"content": "paired",
			})
			return
		case "timeout":
			conn.WriteJSON(map[string]string{
				"type":    "error",
				"content": "QR code timed out",
			})
			client.Disconnect()
			return
		}
	}
}

// --- Listeners ---

func (h *WhatsAppHandler) ListListeners(c *gin.Context) {
	userID := c.GetUint("user_id")
	numberID, _ := strconv.ParseUint(c.Param("id"), 10, 64)

	// Verify ownership
	var number models.WaNumber
	if err := h.DB.Where("id = ? AND user_id = ?", numberID, userID).First(&number).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "number not found"})
		return
	}

	var listeners []models.WaListener
	h.DB.Where("wa_number_id = ?", numberID).Order("created_at desc").Find(&listeners)

	// Attach message counts
	type listenerWithCount struct {
		models.WaListener
		MessageCount int64 `json:"message_count"`
	}
	result := make([]listenerWithCount, len(listeners))
	for i, l := range listeners {
		var count int64
		h.DB.Model(&models.WaMessage{}).Where("wa_listener_id = ?", l.ID).Count(&count)
		result[i] = listenerWithCount{WaListener: l, MessageCount: count}
	}

	c.JSON(http.StatusOK, result)
}

type addListenerRequest struct {
	JID  string `json:"jid" binding:"required"`
	Name string `json:"name" binding:"required"`
	Type string `json:"type" binding:"required"`
}

func (h *WhatsAppHandler) AddListener(c *gin.Context) {
	userID := c.GetUint("user_id")
	numberID, _ := strconv.ParseUint(c.Param("id"), 10, 64)

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
		WaNumberID: uint(numberID),
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

func (h *WhatsAppHandler) UpdateListener(c *gin.Context) {
	userID := c.GetUint("user_id")
	id, _ := strconv.ParseUint(c.Param("id"), 10, 64)

	var listener models.WaListener
	if err := h.DB.Joins("JOIN wa_numbers ON wa_numbers.id = wa_listeners.wa_number_id").
		Where("wa_listeners.id = ? AND wa_numbers.user_id = ?", id, userID).
		First(&listener).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "listener not found"})
		return
	}

	var req struct {
		Name     *string `json:"name"`
		IsActive *bool   `json:"is_active"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	updates := map[string]any{}
	if req.Name != nil {
		updates["name"] = *req.Name
	}
	if req.IsActive != nil {
		updates["is_active"] = *req.IsActive
	}

	h.DB.Model(&listener).Updates(updates)
	c.JSON(http.StatusOK, listener)
}

func (h *WhatsAppHandler) DeleteListener(c *gin.Context) {
	userID := c.GetUint("user_id")
	id, _ := strconv.ParseUint(c.Param("id"), 10, 64)

	var listener models.WaListener
	if err := h.DB.Joins("JOIN wa_numbers ON wa_numbers.id = wa_listeners.wa_number_id").
		Where("wa_listeners.id = ? AND wa_numbers.user_id = ?", id, userID).
		First(&listener).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "listener not found"})
		return
	}

	h.DB.Delete(&listener)
	c.JSON(http.StatusOK, gin.H{"message": "listener removed"})
}

// --- Messages ---

func (h *WhatsAppHandler) ListMessages(c *gin.Context) {
	userID := c.GetUint("user_id")
	listenerID, _ := strconv.ParseUint(c.Param("id"), 10, 64)

	// Verify ownership
	var listener models.WaListener
	if err := h.DB.Joins("JOIN wa_numbers ON wa_numbers.id = wa_listeners.wa_number_id").
		Where("wa_listeners.id = ? AND wa_numbers.user_id = ?", listenerID, userID).
		First(&listener).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "listener not found"})
		return
	}

	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "50"))
	offset := (page - 1) * limit

	var messages []models.WaMessage
	h.DB.Where("wa_listener_id = ?", listenerID).
		Order("timestamp desc").
		Offset(offset).Limit(limit).
		Find(&messages)

	var total int64
	h.DB.Model(&models.WaMessage{}).Where("wa_listener_id = ?", listenerID).Count(&total)

	c.JSON(http.StatusOK, gin.H{
		"messages": messages,
		"total":    total,
		"page":     page,
		"limit":    limit,
	})
}

func (h *WhatsAppHandler) SearchMessages(c *gin.Context) {
	userID := c.GetUint("user_id")
	query := c.Query("q")
	if query == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "query parameter 'q' is required"})
		return
	}

	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "20"))

	var messages []models.WaMessage
	h.DB.Joins("JOIN wa_listeners ON wa_listeners.id = wa_messages.wa_listener_id").
		Joins("JOIN wa_numbers ON wa_numbers.id = wa_listeners.wa_number_id").
		Where("wa_numbers.user_id = ? AND wa_messages.content LIKE ?", userID, "%"+query+"%").
		Order("wa_messages.timestamp desc").
		Limit(limit).
		Find(&messages)

	c.JSON(http.StatusOK, messages)
}

// --- Outbox ---

func (h *WhatsAppHandler) ListOutbox(c *gin.Context) {
	userID := c.GetUint("user_id")
	status := c.Query("status")

	query := h.DB.Joins("JOIN wa_numbers ON wa_numbers.id = wa_outbox.wa_number_id").
		Where("wa_numbers.user_id = ?", userID).
		Order("wa_outbox.created_at desc")

	if status != "" {
		query = query.Where("wa_outbox.status = ?", status)
	}

	var outbox []models.WaOutbox
	query.Find(&outbox)
	c.JSON(http.StatusOK, outbox)
}

func (h *WhatsAppHandler) UpdateOutbox(c *gin.Context) {
	userID := c.GetUint("user_id")
	id, _ := strconv.ParseUint(c.Param("id"), 10, 64)

	var outbox models.WaOutbox
	if err := h.DB.Joins("JOIN wa_numbers ON wa_numbers.id = wa_outbox.wa_number_id").
		Where("wa_outbox.id = ? AND wa_numbers.user_id = ?", id, userID).
		First(&outbox).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "outbox item not found"})
		return
	}

	var req struct {
		Status  *string `json:"status"`
		Content *string `json:"content"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	updates := map[string]any{}
	if req.Content != nil {
		updates["content"] = *req.Content
	}
	if req.Status != nil {
		switch *req.Status {
		case "approved":
			now := time.Now()
			updates["status"] = "approved"
			updates["approved_at"] = now
		case "rejected":
			updates["status"] = "rejected"
		default:
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid status, must be 'approved' or 'rejected'"})
			return
		}
	}

	h.DB.Model(&outbox).Updates(updates)
	h.DB.First(&outbox, id)
	c.JSON(http.StatusOK, outbox)
}

func (h *WhatsAppHandler) DeleteOutbox(c *gin.Context) {
	userID := c.GetUint("user_id")
	id, _ := strconv.ParseUint(c.Param("id"), 10, 64)

	var outbox models.WaOutbox
	if err := h.DB.Joins("JOIN wa_numbers ON wa_numbers.id = wa_outbox.wa_number_id").
		Where("wa_outbox.id = ? AND wa_numbers.user_id = ? AND wa_outbox.status = ?", id, userID, "pending").
		First(&outbox).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "pending outbox item not found"})
		return
	}

	h.DB.Delete(&outbox)
	c.JSON(http.StatusOK, gin.H{"message": "outbox item deleted"})
}
```

- [ ] **Step 2: Verify build**

```bash
cd /home/nst/GolandProjects/pdt/backend && go build ./...
```

- [ ] **Step 3: Commit**

```bash
cd /home/nst/GolandProjects/pdt && git add backend/internal/handlers/whatsapp.go && git commit -m "feat(wa): add REST handlers for numbers, listeners, messages, outbox, and QR pairing"
```

---

## Task 10: WhatsApp AI Agent

**Files:**
- Create: `backend/internal/ai/agent/whatsapp.go`

- [ ] **Step 1: Create WhatsApp agent with 7 tools**

Create `backend/internal/ai/agent/whatsapp.go`:

```go
package agent

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"gorm.io/gorm"

	"github.com/cds-id/pdt/backend/internal/ai/minimax"
	"github.com/cds-id/pdt/backend/internal/models"
	wvClient "github.com/cds-id/pdt/backend/internal/services/weaviate"
)

type WhatsAppAgent struct {
	DB       *gorm.DB
	UserID   uint
	Weaviate *wvClient.Client
}

func (a *WhatsAppAgent) Name() string { return "whatsapp" }

func (a *WhatsAppAgent) SystemPrompt() string {
	// Load user's phone numbers for self-awareness
	var numbers []models.WaNumber
	a.DB.Where("user_id = ?", a.UserID).Find(&numbers)
	var phoneList []string
	for _, n := range numbers {
		phoneList = append(phoneList, n.PhoneNumber+" ("+n.DisplayName+")")
	}
	phones := strings.Join(phoneList, ", ")

	return fmt.Sprintf(`You are a WhatsApp Analytics assistant for PDT (Personal Development Tracker).

Your phone numbers: %s
Current date: %s

You help the user analyze their WhatsApp conversations from registered listeners (groups and personal chats).

CRITICAL RULES:
1. NEVER fabricate message content. Only present data from tool results.
2. You know the user's phone numbers above — distinguish their messages from external ones.
3. When sending messages (send_message, reply_to_message), you MUST explain WHY in the context field. Never send without the user explicitly asking.
4. For summaries, use semantic_search first to find relevant threads, then summarize with that context.
5. The user may write in Indonesian or English. Respond in the same language.
6. When presenting messages, include sender name, timestamp, and content.
7. For the full_chat_report compound tool, present results in a structured briefing format.`, phones, time.Now().Format("2006-01-02"))
}

func (a *WhatsAppAgent) Tools() []minimax.Tool {
	return []minimax.Tool{
		{
			Name:        "list_listeners",
			Description: "List all registered WhatsApp listeners (groups and contacts) with message counts. Use this to see what chat sources are being tracked.",
			InputSchema: json.RawMessage(`{
				"type": "object",
				"properties": {
					"wa_number_id": {"type": "integer", "description": "Optional: filter by specific phone number ID"}
				}
			}`),
		},
		{
			Name:        "search_messages",
			Description: "Search WhatsApp messages by keyword. Filters by listener, sender, and date range.",
			InputSchema: json.RawMessage(`{
				"type": "object",
				"properties": {
					"query": {"type": "string", "description": "Keyword to search for"},
					"listener_id": {"type": "integer", "description": "Optional: filter by listener ID"},
					"sender": {"type": "string", "description": "Optional: filter by sender name"},
					"start_date": {"type": "string", "description": "Optional: start date (YYYY-MM-DD)"},
					"end_date": {"type": "string", "description": "Optional: end date (YYYY-MM-DD)"},
					"limit": {"type": "integer", "description": "Max results (default 20)"}
				},
				"required": ["query"]
			}`),
		},
		{
			Name:        "semantic_search",
			Description: "Find messages by meaning using vector similarity search. Better than keyword search for finding discussions about a topic. Example: 'what did we discuss about the release timeline?'",
			InputSchema: json.RawMessage(`{
				"type": "object",
				"properties": {
					"query": {"type": "string", "description": "Natural language query"},
					"listener_id": {"type": "integer", "description": "Optional: filter by listener ID"},
					"start_date": {"type": "string", "description": "Optional: start date (YYYY-MM-DD)"},
					"end_date": {"type": "string", "description": "Optional: end date (YYYY-MM-DD)"},
					"limit": {"type": "integer", "description": "Max results (default 10)"}
				},
				"required": ["query"]
			}`),
		},
		{
			Name:        "summarize_chat",
			Description: "Generate an AI summary for a specific listener or across all listeners in a time range. Returns key topics, action items, and decisions.",
			InputSchema: json.RawMessage(`{
				"type": "object",
				"properties": {
					"listener_id": {"type": "integer", "description": "Optional: specific listener ID. Null = all listeners."},
					"start_date": {"type": "string", "description": "Start date (YYYY-MM-DD)"},
					"end_date": {"type": "string", "description": "End date (YYYY-MM-DD)"}
				},
				"required": ["start_date", "end_date"]
			}`),
		},
		{
			Name:        "send_message",
			Description: "Draft an outgoing WhatsApp message for user approval. The message goes to the Outbox where the user must approve before it's sent.",
			InputSchema: json.RawMessage(`{
				"type": "object",
				"properties": {
					"target_jid": {"type": "string", "description": "Recipient WhatsApp JID (group or contact)"},
					"content": {"type": "string", "description": "Message text to send"},
					"context": {"type": "string", "description": "REQUIRED: explain WHY you want to send this message"}
				},
				"required": ["target_jid", "content", "context"]
			}`),
		},
		{
			Name:        "reply_to_message",
			Description: "Draft a reply to a specific WhatsApp message for user approval. Creates a quoted reply in WhatsApp.",
			InputSchema: json.RawMessage(`{
				"type": "object",
				"properties": {
					"wa_message_id": {"type": "integer", "description": "ID of the message to reply to"},
					"content": {"type": "string", "description": "Reply text"},
					"context": {"type": "string", "description": "REQUIRED: explain WHY you want to reply"}
				},
				"required": ["wa_message_id", "content", "context"]
			}`),
		},
		{
			Name:        "full_chat_report",
			Description: "Generate a complete WhatsApp briefing report. Combines: list all listeners, summarize all chats, and find recent notable topics. Use this when the user asks for a full WhatsApp overview.",
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

func (a *WhatsAppAgent) listListeners(args json.RawMessage) (any, error) {
	var params struct {
		WaNumberID int `json:"wa_number_id"`
	}
	json.Unmarshal(args, &params)

	query := a.DB.Joins("JOIN wa_numbers ON wa_numbers.id = wa_listeners.wa_number_id").
		Where("wa_numbers.user_id = ?", a.UserID)

	if params.WaNumberID > 0 {
		query = query.Where("wa_listeners.wa_number_id = ?", params.WaNumberID)
	}

	var listeners []models.WaListener
	query.Find(&listeners)

	type result struct {
		ID           uint   `json:"id"`
		Name         string `json:"name"`
		Type         string `json:"type"`
		JID          string `json:"jid"`
		IsActive     bool   `json:"is_active"`
		MessageCount int64  `json:"message_count"`
	}

	var results []result
	for _, l := range listeners {
		var count int64
		a.DB.Model(&models.WaMessage{}).Where("wa_listener_id = ?", l.ID).Count(&count)
		results = append(results, result{
			ID:           l.ID,
			Name:         l.Name,
			Type:         l.Type,
			JID:          l.JID,
			IsActive:     l.IsActive,
			MessageCount: count,
		})
	}

	return map[string]any{"listeners": results, "count": len(results)}, nil
}

func (a *WhatsAppAgent) searchMessages(args json.RawMessage) (any, error) {
	var params struct {
		Query      string `json:"query"`
		ListenerID int    `json:"listener_id"`
		Sender     string `json:"sender"`
		StartDate  string `json:"start_date"`
		EndDate    string `json:"end_date"`
		Limit      int    `json:"limit"`
	}
	json.Unmarshal(args, &params)

	if params.Limit <= 0 {
		params.Limit = 20
	}

	query := a.DB.Joins("JOIN wa_listeners ON wa_listeners.id = wa_messages.wa_listener_id").
		Joins("JOIN wa_numbers ON wa_numbers.id = wa_listeners.wa_number_id").
		Where("wa_numbers.user_id = ?", a.UserID).
		Where("wa_messages.content LIKE ?", "%"+params.Query+"%")

	if params.ListenerID > 0 {
		query = query.Where("wa_messages.wa_listener_id = ?", params.ListenerID)
	}
	if params.Sender != "" {
		query = query.Where("wa_messages.sender_name LIKE ?", "%"+params.Sender+"%")
	}
	if params.StartDate != "" {
		query = query.Where("wa_messages.timestamp >= ?", params.StartDate)
	}
	if params.EndDate != "" {
		query = query.Where("wa_messages.timestamp <= ?", params.EndDate+" 23:59:59")
	}

	var messages []models.WaMessage
	query.Order("wa_messages.timestamp desc").Limit(params.Limit).Find(&messages)

	return map[string]any{"messages": messages, "count": len(messages)}, nil
}

func (a *WhatsAppAgent) semanticSearch(ctx context.Context, args json.RawMessage) (any, error) {
	if a.Weaviate == nil || !a.Weaviate.IsAvailable() {
		return map[string]any{"error": "Vector search is not available. Use search_messages for keyword search instead."}, nil
	}

	var params struct {
		Query      string `json:"query"`
		ListenerID *int   `json:"listener_id"`
		StartDate  string `json:"start_date"`
		EndDate    string `json:"end_date"`
		Limit      int    `json:"limit"`
	}
	json.Unmarshal(args, &params)

	var startDate, endDate *time.Time
	if params.StartDate != "" {
		t, _ := time.Parse("2006-01-02", params.StartDate)
		startDate = &t
	}
	if params.EndDate != "" {
		t, _ := time.Parse("2006-01-02", params.EndDate)
		end := t.Add(24*time.Hour - time.Second)
		endDate = &end
	}

	results, err := a.Weaviate.Search(ctx, params.Query, int(a.UserID), params.ListenerID, startDate, endDate, params.Limit)
	if err != nil {
		return nil, fmt.Errorf("semantic search: %w", err)
	}

	return map[string]any{"results": results, "count": len(results)}, nil
}

func (a *WhatsAppAgent) summarizeChat(args json.RawMessage) (any, error) {
	var params struct {
		ListenerID *int   `json:"listener_id"`
		StartDate  string `json:"start_date"`
		EndDate    string `json:"end_date"`
	}
	json.Unmarshal(args, &params)

	query := a.DB.Joins("JOIN wa_listeners ON wa_listeners.id = wa_messages.wa_listener_id").
		Joins("JOIN wa_numbers ON wa_numbers.id = wa_listeners.wa_number_id").
		Where("wa_numbers.user_id = ?", a.UserID).
		Where("wa_messages.timestamp >= ? AND wa_messages.timestamp <= ?", params.StartDate, params.EndDate+" 23:59:59")

	if params.ListenerID != nil {
		query = query.Where("wa_messages.wa_listener_id = ?", *params.ListenerID)
	}

	var messages []models.WaMessage
	query.Order("wa_messages.timestamp asc").Limit(500).Find(&messages)

	// Group by listener for per-listener summaries
	grouped := make(map[uint][]models.WaMessage)
	for _, m := range messages {
		grouped[m.WaListenerID] = append(grouped[m.WaListenerID], m)
	}

	// Build summary data for the LLM to summarize
	type listenerSummary struct {
		ListenerID   uint   `json:"listener_id"`
		ListenerName string `json:"listener_name"`
		MessageCount int    `json:"message_count"`
		Messages     []struct {
			Sender    string `json:"sender"`
			Content   string `json:"content"`
			Timestamp string `json:"timestamp"`
		} `json:"messages"`
	}

	var summaries []listenerSummary
	for lid, msgs := range grouped {
		var listener models.WaListener
		a.DB.First(&listener, lid)

		ls := listenerSummary{
			ListenerID:   lid,
			ListenerName: listener.Name,
			MessageCount: len(msgs),
		}
		for _, m := range msgs {
			ls.Messages = append(ls.Messages, struct {
				Sender    string `json:"sender"`
				Content   string `json:"content"`
				Timestamp string `json:"timestamp"`
			}{
				Sender:    m.SenderName,
				Content:   m.Content,
				Timestamp: m.Timestamp.Format("2006-01-02 15:04"),
			})
		}
		summaries = append(summaries, ls)
	}

	return map[string]any{
		"period":           params.StartDate + " to " + params.EndDate,
		"total_messages":   len(messages),
		"listener_count":   len(grouped),
		"listener_details": summaries,
		"instruction":      "Summarize each listener's chat: key topics, decisions, action items. Then provide a cross-listener summary.",
	}, nil
}

func (a *WhatsAppAgent) sendMessage(args json.RawMessage) (any, error) {
	var params struct {
		TargetJID string `json:"target_jid"`
		Content   string `json:"content"`
		Context   string `json:"context"`
	}
	json.Unmarshal(args, &params)

	// Find the first connected number for this user
	var number models.WaNumber
	if err := a.DB.Where("user_id = ? AND status = ?", a.UserID, "connected").First(&number).Error; err != nil {
		return map[string]any{"error": "No connected WhatsApp number found"}, nil
	}

	// Look up target name from listeners
	var listener models.WaListener
	targetName := params.TargetJID
	if err := a.DB.Where("jid = ? AND wa_number_id = ?", params.TargetJID, number.ID).First(&listener).Error; err == nil {
		targetName = listener.Name
	}

	outbox := models.WaOutbox{
		WaNumberID:  number.ID,
		TargetJID:   params.TargetJID,
		TargetName:  targetName,
		Content:     params.Content,
		Status:      "pending",
		RequestedBy: "agent",
		Context:     params.Context,
	}
	a.DB.Create(&outbox)

	return map[string]any{
		"status":  "pending_approval",
		"message": fmt.Sprintf("Message drafted to %s. User must approve in the Outbox before it's sent.", targetName),
		"outbox_id": outbox.ID,
	}, nil
}

func (a *WhatsAppAgent) replyToMessage(args json.RawMessage) (any, error) {
	var params struct {
		WaMessageID int    `json:"wa_message_id"`
		Content     string `json:"content"`
		Context     string `json:"context"`
	}
	json.Unmarshal(args, &params)

	// Look up the original message
	var original models.WaMessage
	if err := a.DB.First(&original, params.WaMessageID).Error; err != nil {
		return map[string]any{"error": "Original message not found"}, nil
	}

	// Get listener and number
	var listener models.WaListener
	a.DB.First(&listener, original.WaListenerID)

	var number models.WaNumber
	a.DB.First(&number, listener.WaNumberID)

	outbox := models.WaOutbox{
		WaNumberID:  number.ID,
		TargetJID:   listener.JID,
		TargetName:  listener.Name,
		Content:     params.Content,
		Status:      "pending",
		RequestedBy: "agent",
		Context:     fmt.Sprintf("Reply to %s's message: \"%s\". Reason: %s", original.SenderName, truncateStr(original.Content, 100), params.Context),
	}
	a.DB.Create(&outbox)

	return map[string]any{
		"status":   "pending_approval",
		"message":  fmt.Sprintf("Reply drafted to %s. User must approve in the Outbox.", listener.Name),
		"outbox_id": outbox.ID,
	}, nil
}

func (a *WhatsAppAgent) fullChatReport(ctx context.Context, args json.RawMessage) (any, error) {
	// 1. List listeners
	listeners, _ := a.listListeners(json.RawMessage(`{}`))

	// 2. Summarize all chats
	summary, _ := a.summarizeChat(args)

	// 3. Semantic search for recent notable topics (if available)
	var semanticResults any
	if a.Weaviate != nil && a.Weaviate.IsAvailable() {
		semanticResults, _ = a.semanticSearch(ctx, json.RawMessage(`{"query": "important decisions action items deadlines", "limit": 15}`))
	}

	return map[string]any{
		"listeners":        listeners,
		"summary":          summary,
		"notable_topics":   semanticResults,
		"instruction":      "Present a structured WhatsApp briefing: 1) Active listeners overview, 2) Per-listener summaries with key topics and action items, 3) Cross-listener highlights and notable discussions.",
	}, nil
}

func truncateStr(s string, n int) string {
	runes := []rune(s)
	if len(runes) <= n {
		return s
	}
	return string(runes[:n]) + "..."
}
```

- [ ] **Step 2: Verify build**

```bash
cd /home/nst/GolandProjects/pdt/backend && go build ./...
```

- [ ] **Step 3: Commit**

```bash
cd /home/nst/GolandProjects/pdt && git add backend/internal/ai/agent/whatsapp.go && git commit -m "feat(wa): add WhatsApp AI agent with 7 tools"
```

---

## Task 11: Orchestrator & Chat Handler Integration

**Files:**
- Modify: `backend/internal/ai/agent/orchestrator.go`
- Modify: `backend/internal/handlers/chat.go`

- [ ] **Step 1: Add whatsapp to orchestrator router**

In `backend/internal/ai/agent/orchestrator.go`, update the `routerSystemPrompt` to add the WhatsApp agent:

Add after the "briefing" agent description:
```
- "whatsapp": Handles questions about WhatsApp messages, chat summaries, listener activity, sending messages, and WhatsApp analytics. Use this when users ask about WhatsApp chats, want to search or summarize conversations, send messages, or get a WhatsApp briefing.
```

Add WhatsApp keywords:
```
Keywords that suggest "whatsapp" agent: whatsapp, wa, chat summary, pesan, kirim pesan, ringkasan chat, listener, group chat.
```

Update the `routerTool` enum to include "whatsapp":
```go
"enum": ["git", "jira", "report", "proof", "briefing", "whatsapp"],
```

- [ ] **Step 2: Add WhatsAppAgent to chat handler**

In `backend/internal/handlers/chat.go`, add a `Weaviate` field to `ChatHandler`:

```go
type ChatHandler struct {
	DB              *gorm.DB
	MiniMaxClient   *minimax.Client
	Encryptor       *crypto.Encryptor
	R2              *storage.R2Client
	ReportGenerator *report.Generator
	ContextWindow   int
	WaManager       *whatsapp.Manager
	WeaviateClient  *weaviateClient.Client
}
```

Add the import for whatsapp and weaviate packages, then add the WhatsAppAgent to the orchestrator in `HandleWebSocket`:

```go
orchestrator := agent.NewOrchestrator(
	h.MiniMaxClient,
	&agent.GitAgent{DB: h.DB, UserID: userID, Encryptor: h.Encryptor},
	&agent.JiraAgent{DB: h.DB, UserID: userID},
	&agent.ReportAgent{DB: h.DB, UserID: userID, Generator: h.ReportGenerator, R2: h.R2},
	&agent.ProofAgent{DB: h.DB, UserID: userID},
	&agent.BriefingAgent{DB: h.DB, UserID: userID},
	&agent.WhatsAppAgent{DB: h.DB, UserID: userID, Weaviate: h.WeaviateClient},
)
```

- [ ] **Step 3: Verify build**

```bash
cd /home/nst/GolandProjects/pdt/backend && go build ./...
```

- [ ] **Step 4: Commit**

```bash
cd /home/nst/GolandProjects/pdt && git add backend/internal/ai/agent/orchestrator.go backend/internal/handlers/chat.go && git commit -m "feat(wa): integrate WhatsApp agent into orchestrator and chat handler"
```

---

## Task 12: Main.go Wiring

**Files:**
- Modify: `backend/cmd/server/main.go`
- Modify: `docker-compose.yml`

- [ ] **Step 1: Add Weaviate and WhatsApp initialization to main.go**

Add imports:
```go
	waService "github.com/cds-id/pdt/backend/internal/services/whatsapp"
	wvService "github.com/cds-id/pdt/backend/internal/services/weaviate"
```

After the R2 client setup and before the worker scheduler, add:

```go
	// Weaviate client (optional — for vector search)
	var weaviateClient *wvService.Client
	var embeddingWorker *wvService.EmbeddingWorker
	if cfg.GeminiAPIKey != "" {
		weaviateClient = wvService.NewClient(cfg.WeaviateURL, cfg.GeminiAPIKey)
		if weaviateClient.IsAvailable() {
			embeddingWorker = wvService.NewEmbeddingWorker(weaviateClient, db)
			embeddingWorker.Start(ctx)
			log.Printf("Weaviate connected: %s", cfg.WeaviateURL)
		}
	}

	// WhatsApp connection manager (optional)
	var waManager *waService.Manager
	if true { // Always initialize — connections are optional
		var err error
		waManager, err = waService.NewManager(db, r2Client, embeddingWorker)
		if err != nil {
			log.Printf("WhatsApp manager init failed: %v", err)
		} else {
			waManager.Start(ctx)
			sender := waService.NewSenderWorker(db, waManager)
			sender.Start(ctx)
		}
	}
```

Update the `chatHandler` to include new fields:
```go
	chatHandler := &handlers.ChatHandler{
		DB:              db,
		MiniMaxClient:   miniMaxClient,
		Encryptor:       encryptor,
		R2:              r2Client,
		ReportGenerator: reportGen,
		ContextWindow:   cfg.AIContextWindow,
		WaManager:       waManager,
		WeaviateClient:  weaviateClient,
	}
```

Add WhatsApp handler and routes after the conversations routes:
```go
	waHandler := &handlers.WhatsAppHandler{
		DB:      db,
		Manager: waManager,
	}
```

Add routes inside the `protected` group:
```go
			wa := protected.Group("/wa")
			{
				waNumbers := wa.Group("/numbers")
				{
					waNumbers.GET("", waHandler.ListNumbers)
					waNumbers.POST("", waHandler.AddNumber)
					waNumbers.PATCH("/:id", waHandler.UpdateNumber)
					waNumbers.DELETE("/:id", waHandler.DeleteNumber)
					waNumbers.GET("/:id/listeners", waHandler.ListListeners)
					waNumbers.POST("/:id/listeners", waHandler.AddListener)
				}

				waListeners := wa.Group("/listeners")
				{
					waListeners.PATCH("/:id", waHandler.UpdateListener)
					waListeners.DELETE("/:id", waHandler.DeleteListener)
					waListeners.GET("/:id/messages", waHandler.ListMessages)
				}

				wa.GET("/messages/search", waHandler.SearchMessages)

				waOutbox := wa.Group("/outbox")
				{
					waOutbox.GET("", waHandler.ListOutbox)
					waOutbox.PATCH("/:id", waHandler.UpdateOutbox)
					waOutbox.DELETE("/:id", waHandler.DeleteOutbox)
				}

				wa.GET("/pair/:id", waHandler.HandlePairing)
			}
```

- [ ] **Step 2: Add Weaviate to docker-compose.yml**

Add to the `services` section in `docker-compose.yml`:

```yaml
  weaviate:
    image: semitechnologies/weaviate:latest
    ports:
      - "8081:8080"
      - "50051:50051"
    environment:
      QUERY_DEFAULTS_LIMIT: 25
      AUTHENTICATION_ANONYMOUS_ACCESS_ENABLED: 'true'
      PERSISTENCE_DATA_PATH: /var/lib/weaviate
      DEFAULT_VECTORIZER_MODULE: text2vec-palm
      ENABLE_MODULES: text2vec-palm
      CLUSTER_HOSTNAME: node1
    volumes:
      - weaviate_data:/var/lib/weaviate
    restart: unless-stopped
```

Add `weaviate_data:` to the `volumes` section.

- [ ] **Step 3: Add graceful shutdown for WA manager**

Before `srv.Shutdown(shutdownCtx)` in main.go, add:

```go
	if waManager != nil {
		waManager.Shutdown()
	}
```

- [ ] **Step 4: Verify build**

```bash
cd /home/nst/GolandProjects/pdt/backend && go build ./...
```

- [ ] **Step 5: Commit**

```bash
cd /home/nst/GolandProjects/pdt && git add backend/cmd/server/main.go docker-compose.yml && git commit -m "feat(wa): wire WhatsApp manager, Weaviate, and routes into main.go"
```

---

## Task 13: Frontend — TypeScript Interfaces & API Constants

**Files:**
- Create: `frontend/src/domain/whatsapp/interfaces/whatsapp.interface.ts`
- Create: `frontend/src/infrastructure/constants/whatsapp.constants.ts`
- Modify: `frontend/src/infrastructure/constants/api.constants.ts`
- Modify: `frontend/src/infrastructure/services/api.ts`

- [ ] **Step 1: Create WhatsApp TypeScript interfaces**

Create `frontend/src/domain/whatsapp/interfaces/whatsapp.interface.ts`:

```typescript
export interface IWaNumber {
  id: number
  user_id: number
  phone_number: string
  display_name: string
  status: 'pairing' | 'connected' | 'disconnected'
  paired_at?: string
  created_at: string
  updated_at: string
}

export interface IWaListener {
  id: number
  wa_number_id: number
  jid: string
  name: string
  type: 'group' | 'personal'
  is_active: boolean
  message_count?: number
  created_at: string
  updated_at: string
}

export interface IWaMessage {
  id: number
  wa_listener_id: number
  message_id: string
  sender_jid: string
  sender_name: string
  content: string
  message_type: 'text' | 'image' | 'document' | 'audio' | 'video'
  has_media: boolean
  timestamp: string
  created_at: string
}

export interface IWaOutbox {
  id: number
  wa_number_id: number
  target_jid: string
  target_name: string
  content: string
  status: 'pending' | 'approved' | 'sent' | 'rejected'
  requested_by: 'agent' | 'user'
  context: string
  approved_at?: string
  sent_at?: string
  created_at: string
}

export interface IWaMessagePage {
  messages: IWaMessage[]
  total: number
  page: number
  limit: number
}
```

- [ ] **Step 2: Add WhatsApp API constants**

Add to `frontend/src/infrastructure/constants/api.constants.ts`, inside the `API_CONSTANTS` object:

```typescript
  // WhatsApp Endpoints
  WA: {
    NUMBERS: '/wa/numbers',
    NUMBER: (id: number) => `/wa/numbers/${id}`,
    LISTENERS: (numberId: number) => `/wa/numbers/${numberId}/listeners`,
    LISTENER: (id: number) => `/wa/listeners/${id}`,
    MESSAGES: (listenerId: number) => `/wa/listeners/${listenerId}/messages`,
    SEARCH_MESSAGES: '/wa/messages/search',
    OUTBOX: '/wa/outbox',
    OUTBOX_ITEM: (id: number) => `/wa/outbox/${id}`,
    PAIR: (numberId: number) => `/wa/pair/${numberId}`,
  },
```

- [ ] **Step 3: Add 'WhatsApp' tag to RTK Query api**

In `frontend/src/infrastructure/services/api.ts`, add `'WhatsApp'` to the `tagTypes` array:

```typescript
  tagTypes: [
    'User',
    'Auth',
    'Repo',
    'Sync',
    'Commit',
    'Jira',
    'Report',
    'ReportTemplate',
    'Conversation',
    'WhatsApp'
  ],
```

- [ ] **Step 4: Commit**

```bash
cd /home/nst/GolandProjects/pdt && git add frontend/src/domain/whatsapp/ frontend/src/infrastructure/constants/api.constants.ts frontend/src/infrastructure/services/api.ts && git commit -m "feat(wa): add frontend TypeScript interfaces and API constants"
```

---

## Task 14: Frontend — RTK Query Service

**Files:**
- Create: `frontend/src/infrastructure/services/whatsapp.service.ts`

- [ ] **Step 1: Create WhatsApp RTK Query service**

Create `frontend/src/infrastructure/services/whatsapp.service.ts`:

```typescript
import { api } from './api'
import { API_CONSTANTS } from '../constants/api.constants'
import type {
  IWaNumber,
  IWaListener,
  IWaMessage,
  IWaOutbox,
  IWaMessagePage,
} from '@/domain/whatsapp/interfaces/whatsapp.interface'

export const whatsappApi = api.injectEndpoints({
  endpoints: (builder) => ({
    // Numbers
    listNumbers: builder.query<IWaNumber[], void>({
      query: () => API_CONSTANTS.WA.NUMBERS,
      providesTags: [{ type: 'WhatsApp' as const, id: 'NUMBERS' }],
    }),
    addNumber: builder.mutation<IWaNumber, { phone_number: string; display_name?: string }>({
      query: (data) => ({
        url: API_CONSTANTS.WA.NUMBERS,
        method: 'POST',
        body: data,
      }),
      invalidatesTags: [{ type: 'WhatsApp' as const, id: 'NUMBERS' }],
    }),
    updateNumber: builder.mutation<IWaNumber, { id: number; display_name: string }>({
      query: ({ id, ...data }) => ({
        url: API_CONSTANTS.WA.NUMBER(id),
        method: 'PATCH',
        body: data,
      }),
      invalidatesTags: [{ type: 'WhatsApp' as const, id: 'NUMBERS' }],
    }),
    deleteNumber: builder.mutation<void, number>({
      query: (id) => ({
        url: API_CONSTANTS.WA.NUMBER(id),
        method: 'DELETE',
      }),
      invalidatesTags: [{ type: 'WhatsApp' as const, id: 'NUMBERS' }],
    }),

    // Listeners
    listListeners: builder.query<IWaListener[], number>({
      query: (numberId) => API_CONSTANTS.WA.LISTENERS(numberId),
      providesTags: (_, __, numberId) => [{ type: 'WhatsApp' as const, id: `LISTENERS-${numberId}` }],
    }),
    addListener: builder.mutation<IWaListener, { numberId: number; jid: string; name: string; type: string }>({
      query: ({ numberId, ...data }) => ({
        url: API_CONSTANTS.WA.LISTENERS(numberId),
        method: 'POST',
        body: data,
      }),
      invalidatesTags: (_, __, { numberId }) => [{ type: 'WhatsApp' as const, id: `LISTENERS-${numberId}` }],
    }),
    updateListener: builder.mutation<IWaListener, { id: number; name?: string; is_active?: boolean }>({
      query: ({ id, ...data }) => ({
        url: API_CONSTANTS.WA.LISTENER(id),
        method: 'PATCH',
        body: data,
      }),
      invalidatesTags: [{ type: 'WhatsApp' as const, id: 'NUMBERS' }],
    }),
    deleteListener: builder.mutation<void, number>({
      query: (id) => ({
        url: API_CONSTANTS.WA.LISTENER(id),
        method: 'DELETE',
      }),
      invalidatesTags: [{ type: 'WhatsApp' as const, id: 'NUMBERS' }],
    }),

    // Messages
    listMessages: builder.query<IWaMessagePage, { listenerId: number; page?: number; limit?: number }>({
      query: ({ listenerId, page = 1, limit = 50 }) =>
        `${API_CONSTANTS.WA.MESSAGES(listenerId)}?page=${page}&limit=${limit}`,
    }),
    searchMessages: builder.query<IWaMessage[], { q: string; limit?: number }>({
      query: ({ q, limit = 20 }) =>
        `${API_CONSTANTS.WA.SEARCH_MESSAGES}?q=${encodeURIComponent(q)}&limit=${limit}`,
    }),

    // Outbox
    listOutbox: builder.query<IWaOutbox[], string | void>({
      query: (status) =>
        status ? `${API_CONSTANTS.WA.OUTBOX}?status=${status}` : API_CONSTANTS.WA.OUTBOX,
      providesTags: [{ type: 'WhatsApp' as const, id: 'OUTBOX' }],
    }),
    updateOutbox: builder.mutation<IWaOutbox, { id: number; status?: string; content?: string }>({
      query: ({ id, ...data }) => ({
        url: API_CONSTANTS.WA.OUTBOX_ITEM(id),
        method: 'PATCH',
        body: data,
      }),
      invalidatesTags: [{ type: 'WhatsApp' as const, id: 'OUTBOX' }],
    }),
    deleteOutbox: builder.mutation<void, number>({
      query: (id) => ({
        url: API_CONSTANTS.WA.OUTBOX_ITEM(id),
        method: 'DELETE',
      }),
      invalidatesTags: [{ type: 'WhatsApp' as const, id: 'OUTBOX' }],
    }),
  }),
})

export const {
  useListNumbersQuery,
  useAddNumberMutation,
  useUpdateNumberMutation,
  useDeleteNumberMutation,
  useListListenersQuery,
  useAddListenerMutation,
  useUpdateListenerMutation,
  useDeleteListenerMutation,
  useListMessagesQuery,
  useSearchMessagesQuery,
  useListOutboxQuery,
  useUpdateOutboxMutation,
  useDeleteOutboxMutation,
} = whatsappApi
```

- [ ] **Step 2: Commit**

```bash
cd /home/nst/GolandProjects/pdt && git add frontend/src/infrastructure/services/whatsapp.service.ts && git commit -m "feat(wa): add WhatsApp RTK Query service with all endpoints"
```

---

## Task 15: Frontend — WhatsApp Settings Components

**Files:**
- Create: `frontend/src/presentation/components/whatsapp/NumberManager.tsx`
- Create: `frontend/src/presentation/components/whatsapp/ListenerManager.tsx`
- Create: `frontend/src/presentation/components/whatsapp/QrPairingModal.tsx`
- Modify: `frontend/src/presentation/pages/SettingsPage.tsx`

- [ ] **Step 1: Create QR Pairing Modal**

Create `frontend/src/presentation/components/whatsapp/QrPairingModal.tsx`:

```tsx
import { useState, useEffect, useRef } from 'react'
import { API_CONSTANTS, buildUrl } from '@/infrastructure/constants/api.constants'

interface QrPairingModalProps {
  numberId: number
  token: string
  onClose: () => void
  onSuccess: () => void
}

export function QrPairingModal({ numberId, token, onClose, onSuccess }: QrPairingModalProps) {
  const [qrCode, setQrCode] = useState<string | null>(null)
  const [status, setStatus] = useState<'connecting' | 'waiting' | 'success' | 'error'>('connecting')
  const [error, setError] = useState<string | null>(null)
  const wsRef = useRef<WebSocket | null>(null)

  useEffect(() => {
    const wsUrl = buildUrl(API_CONSTANTS.WA.PAIR(numberId))
      .replace('http://', 'ws://')
      .replace('https://', 'wss://')
      + `?token=${token}`

    const ws = new WebSocket(wsUrl)
    wsRef.current = ws

    ws.onopen = () => setStatus('waiting')

    ws.onmessage = (event) => {
      const data = JSON.parse(event.data)
      switch (data.type) {
        case 'qr':
          setQrCode(data.content)
          setStatus('waiting')
          break
        case 'success':
          setStatus('success')
          setTimeout(onSuccess, 1000)
          break
        case 'error':
          setStatus('error')
          setError(data.content)
          break
      }
    }

    ws.onerror = () => {
      setStatus('error')
      setError('Connection failed')
    }

    return () => ws.close()
  }, [numberId, token, onSuccess])

  return (
    <div className="fixed inset-0 bg-black/60 flex items-center justify-center z-50" onClick={onClose}>
      <div className="bg-pdt-neutral-900 rounded-lg p-6 max-w-sm w-full" onClick={(e) => e.stopPropagation()}>
        <h3 className="text-lg font-semibold text-white mb-4 text-center">Scan QR Code</h3>

        <div className="flex justify-center mb-4">
          {status === 'connecting' && <p className="text-gray-400">Connecting...</p>}
          {status === 'waiting' && qrCode && (
            <img
              src={`https://api.qrserver.com/v1/create-qr-code/?size=200x200&data=${encodeURIComponent(qrCode)}`}
              alt="QR Code"
              className="w-48 h-48 rounded"
            />
          )}
          {status === 'waiting' && !qrCode && <p className="text-gray-400">Generating QR code...</p>}
          {status === 'success' && <p className="text-green-400 text-lg">Paired successfully!</p>}
          {status === 'error' && <p className="text-red-400">{error}</p>}
        </div>

        <p className="text-gray-500 text-sm text-center mb-4">
          Open WhatsApp → Linked Devices → Link a Device
        </p>

        <button
          onClick={onClose}
          className="w-full px-4 py-2 text-sm border border-gray-600 rounded text-gray-300 hover:bg-gray-800"
        >
          Cancel
        </button>
      </div>
    </div>
  )
}
```

Note: This uses a public QR code API for rendering. The implementer may want to use a client-side QR library like `qrcode.react` for offline rendering. Adapt as needed.

- [ ] **Step 2: Create Number Manager**

Create `frontend/src/presentation/components/whatsapp/NumberManager.tsx`:

```tsx
import { useState } from 'react'
import { useListNumbersQuery, useAddNumberMutation, useDeleteNumberMutation } from '@/infrastructure/services/whatsapp.service'
import { useSelector } from 'react-redux'
import { RootState } from '@/application/store'
import { ListenerManager } from './ListenerManager'
import { QrPairingModal } from './QrPairingModal'

export function NumberManager() {
  const { data: numbers, isLoading, refetch } = useListNumbersQuery()
  const [addNumber] = useAddNumberMutation()
  const [deleteNumber] = useDeleteNumberMutation()
  const token = useSelector((state: RootState) => state.auth.token)

  const [showAdd, setShowAdd] = useState(false)
  const [phoneNumber, setPhoneNumber] = useState('')
  const [displayName, setDisplayName] = useState('')
  const [pairingNumberId, setPairingNumberId] = useState<number | null>(null)
  const [expandedNumber, setExpandedNumber] = useState<number | null>(null)

  const handleAdd = async () => {
    if (!phoneNumber) return
    try {
      const result = await addNumber({ phone_number: phoneNumber, display_name: displayName }).unwrap()
      setPhoneNumber('')
      setDisplayName('')
      setShowAdd(false)
      setPairingNumberId(result.id)
    } catch {
      // handle error
    }
  }

  const handleDelete = async (id: number) => {
    if (confirm('Disconnect and remove this number?')) {
      await deleteNumber(id)
    }
  }

  if (isLoading) return <p className="text-gray-400">Loading...</p>

  return (
    <div className="space-y-3">
      {numbers?.map((num) => (
        <div key={num.id}>
          <div className="bg-pdt-neutral-800 rounded-lg p-4 flex items-center justify-between">
            <div>
              <div className="text-pdt-primary-light font-semibold">{num.phone_number}</div>
              <div className="text-gray-500 text-sm">
                {num.display_name || 'No name'} •{' '}
                <span className={num.status === 'connected' ? 'text-green-400' : 'text-red-400'}>
                  {num.status}
                </span>
              </div>
            </div>
            <div className="flex gap-2">
              <button
                onClick={() => setExpandedNumber(expandedNumber === num.id ? null : num.id)}
                className="px-3 py-1.5 text-xs border border-gray-600 rounded text-gray-300 hover:bg-gray-700"
              >
                Listeners
              </button>
              <button
                onClick={() => handleDelete(num.id)}
                className="px-3 py-1.5 text-xs border border-red-600 rounded text-red-400 hover:bg-red-900/20"
              >
                Disconnect
              </button>
            </div>
          </div>
          {expandedNumber === num.id && (
            <div className="ml-4 mt-2">
              <ListenerManager numberId={num.id} />
            </div>
          )}
        </div>
      ))}

      {showAdd ? (
        <div className="bg-pdt-neutral-800 rounded-lg p-4 space-y-3">
          <input
            placeholder="Phone number (e.g., +62812...)"
            value={phoneNumber}
            onChange={(e) => setPhoneNumber(e.target.value)}
            className="w-full px-3 py-2 bg-pdt-neutral-900 border border-gray-700 rounded text-white text-sm"
          />
          <input
            placeholder="Display name (optional)"
            value={displayName}
            onChange={(e) => setDisplayName(e.target.value)}
            className="w-full px-3 py-2 bg-pdt-neutral-900 border border-gray-700 rounded text-white text-sm"
          />
          <div className="flex gap-2">
            <button onClick={handleAdd} className="px-4 py-2 text-sm bg-pdt-primary rounded text-white hover:bg-pdt-primary-dark">
              Add & Pair
            </button>
            <button onClick={() => setShowAdd(false)} className="px-4 py-2 text-sm border border-gray-600 rounded text-gray-300">
              Cancel
            </button>
          </div>
        </div>
      ) : (
        <button
          onClick={() => setShowAdd(true)}
          className="w-full py-2.5 text-sm border border-dashed border-gray-600 rounded text-gray-400 hover:border-gray-400 hover:text-gray-300"
        >
          + Add Number
        </button>
      )}

      {pairingNumberId && token && (
        <QrPairingModal
          numberId={pairingNumberId}
          token={token}
          onClose={() => setPairingNumberId(null)}
          onSuccess={() => {
            setPairingNumberId(null)
            refetch()
          }}
        />
      )}
    </div>
  )
}
```

- [ ] **Step 3: Create Listener Manager**

Create `frontend/src/presentation/components/whatsapp/ListenerManager.tsx`:

```tsx
import { useState } from 'react'
import {
  useListListenersQuery,
  useAddListenerMutation,
  useUpdateListenerMutation,
  useDeleteListenerMutation,
} from '@/infrastructure/services/whatsapp.service'

interface ListenerManagerProps {
  numberId: number
}

export function ListenerManager({ numberId }: ListenerManagerProps) {
  const { data: listeners, isLoading } = useListListenersQuery(numberId)
  const [addListener] = useAddListenerMutation()
  const [updateListener] = useUpdateListenerMutation()
  const [deleteListener] = useDeleteListenerMutation()

  const [showAdd, setShowAdd] = useState(false)
  const [jid, setJid] = useState('')
  const [name, setName] = useState('')
  const [type, setType] = useState<'group' | 'personal'>('group')

  const handleAdd = async () => {
    if (!jid || !name) return
    await addListener({ numberId, jid, name, type })
    setJid('')
    setName('')
    setShowAdd(false)
  }

  const handleToggle = async (id: number, isActive: boolean) => {
    await updateListener({ id, is_active: !isActive })
  }

  if (isLoading) return <p className="text-gray-500 text-sm">Loading listeners...</p>

  return (
    <div className="space-y-2">
      {listeners?.map((l) => (
        <div key={l.id} className="bg-pdt-neutral-850 rounded p-3 flex items-center justify-between">
          <div>
            <div className="text-green-400 font-medium text-sm">{l.name}</div>
            <div className="text-gray-500 text-xs">
              {l.type} • {l.message_count ?? 0} messages • {l.is_active ? 'Active' : 'Paused'}
            </div>
          </div>
          <div className="flex gap-2">
            <button
              onClick={() => handleToggle(l.id, l.is_active)}
              className="px-2 py-1 text-xs border border-gray-600 rounded text-gray-300 hover:bg-gray-700"
            >
              {l.is_active ? 'Pause' : 'Resume'}
            </button>
            <button
              onClick={() => deleteListener(l.id)}
              className="px-2 py-1 text-xs border border-red-600 rounded text-red-400 hover:bg-red-900/20"
            >
              Remove
            </button>
          </div>
        </div>
      ))}

      {showAdd ? (
        <div className="bg-pdt-neutral-850 rounded p-3 space-y-2">
          <input
            placeholder="WhatsApp JID (e.g., 628xx@s.whatsapp.net)"
            value={jid}
            onChange={(e) => setJid(e.target.value)}
            className="w-full px-2 py-1.5 bg-pdt-neutral-900 border border-gray-700 rounded text-white text-xs"
          />
          <input
            placeholder="Display name (e.g., Engineering Chat)"
            value={name}
            onChange={(e) => setName(e.target.value)}
            className="w-full px-2 py-1.5 bg-pdt-neutral-900 border border-gray-700 rounded text-white text-xs"
          />
          <select
            value={type}
            onChange={(e) => setType(e.target.value as 'group' | 'personal')}
            className="w-full px-2 py-1.5 bg-pdt-neutral-900 border border-gray-700 rounded text-white text-xs"
          >
            <option value="group">Group</option>
            <option value="personal">Personal</option>
          </select>
          <div className="flex gap-2">
            <button onClick={handleAdd} className="px-3 py-1 text-xs bg-pdt-primary rounded text-white">
              Add
            </button>
            <button onClick={() => setShowAdd(false)} className="px-3 py-1 text-xs border border-gray-600 rounded text-gray-300">
              Cancel
            </button>
          </div>
        </div>
      ) : (
        <button
          onClick={() => setShowAdd(true)}
          className="w-full py-1.5 text-xs border border-dashed border-gray-700 rounded text-gray-500 hover:text-gray-400"
        >
          + Add Listener
        </button>
      )}
    </div>
  )
}
```

- [ ] **Step 4: Add WhatsApp section to SettingsPage**

In `frontend/src/presentation/pages/SettingsPage.tsx`, import and add the NumberManager component. Add after the last `DataCard`:

```tsx
import { NumberManager } from '../components/whatsapp/NumberManager'

// Inside the return, add after existing DataCard sections:
<DataCard title="WhatsApp Numbers">
  <NumberManager />
</DataCard>
```

- [ ] **Step 5: Commit**

```bash
cd /home/nst/GolandProjects/pdt && git add frontend/src/presentation/components/whatsapp/ frontend/src/presentation/pages/SettingsPage.tsx && git commit -m "feat(wa): add WhatsApp settings UI with number/listener management and QR pairing"
```

---

## Task 16: Frontend — Outbox Page

**Files:**
- Create: `frontend/src/presentation/pages/OutboxPage.tsx`
- Modify: `frontend/src/presentation/routes/index.tsx`

- [ ] **Step 1: Create Outbox approval page**

Create `frontend/src/presentation/pages/OutboxPage.tsx`:

```tsx
import { useState } from 'react'
import { useListOutboxQuery, useUpdateOutboxMutation, useDeleteOutboxMutation } from '@/infrastructure/services/whatsapp.service'

export function OutboxPage() {
  const [filter, setFilter] = useState<string>('')
  const { data: outbox, isLoading } = useListOutboxQuery(filter || undefined)
  const [updateOutbox] = useUpdateOutboxMutation()
  const [deleteOutbox] = useDeleteOutboxMutation()
  const [editingId, setEditingId] = useState<number | null>(null)
  const [editContent, setEditContent] = useState('')

  const handleApprove = async (id: number) => {
    await updateOutbox({ id, status: 'approved' })
  }

  const handleReject = async (id: number) => {
    await updateOutbox({ id, status: 'rejected' })
  }

  const handleEdit = (id: number, content: string) => {
    setEditingId(id)
    setEditContent(content)
  }

  const handleSaveEdit = async (id: number) => {
    await updateOutbox({ id, content: editContent, status: 'approved' })
    setEditingId(null)
  }

  const statusColor = (status: string) => {
    switch (status) {
      case 'pending': return 'bg-yellow-500/20 text-yellow-400'
      case 'approved': return 'bg-blue-500/20 text-blue-400'
      case 'sent': return 'bg-green-500/20 text-green-400'
      case 'rejected': return 'bg-red-500/20 text-red-400'
      default: return 'bg-gray-500/20 text-gray-400'
    }
  }

  return (
    <div className="space-y-4 p-6">
      <div className="flex items-center justify-between">
        <h1 className="text-xl font-semibold text-white">WhatsApp Outbox</h1>
        <select
          value={filter}
          onChange={(e) => setFilter(e.target.value)}
          className="px-3 py-1.5 bg-pdt-neutral-900 border border-gray-700 rounded text-white text-sm"
        >
          <option value="">All</option>
          <option value="pending">Pending</option>
          <option value="sent">Sent</option>
          <option value="rejected">Rejected</option>
        </select>
      </div>

      {isLoading && <p className="text-gray-400">Loading...</p>}

      <div className="space-y-3">
        {outbox?.map((item) => (
          <div key={item.id} className="bg-pdt-neutral-800 rounded-lg p-4">
            <div className="flex items-center justify-between mb-2">
              <span className="text-yellow-400 font-semibold text-sm">
                → {item.target_name}
              </span>
              <span className={`text-xs px-2 py-0.5 rounded ${statusColor(item.status)}`}>
                {item.status}
              </span>
            </div>

            {editingId === item.id ? (
              <textarea
                value={editContent}
                onChange={(e) => setEditContent(e.target.value)}
                className="w-full px-3 py-2 bg-pdt-neutral-900 border border-gray-700 rounded text-white text-sm mb-2"
                rows={3}
              />
            ) : (
              <p className="text-gray-300 text-sm mb-2">"{item.content}"</p>
            )}

            {item.context && (
              <p className="text-gray-500 text-xs mb-3">Agent reason: {item.context}</p>
            )}

            {item.status === 'pending' && (
              <div className="flex gap-2">
                {editingId === item.id ? (
                  <>
                    <button
                      onClick={() => handleSaveEdit(item.id)}
                      className="px-3 py-1.5 text-xs border border-green-600 rounded text-green-400 hover:bg-green-900/20"
                    >
                      Save & Send
                    </button>
                    <button
                      onClick={() => setEditingId(null)}
                      className="px-3 py-1.5 text-xs border border-gray-600 rounded text-gray-300"
                    >
                      Cancel
                    </button>
                  </>
                ) : (
                  <>
                    <button
                      onClick={() => handleApprove(item.id)}
                      className="px-3 py-1.5 text-xs border border-green-600 rounded text-green-400 hover:bg-green-900/20"
                    >
                      Send
                    </button>
                    <button
                      onClick={() => handleEdit(item.id, item.content)}
                      className="px-3 py-1.5 text-xs border border-gray-600 rounded text-gray-300 hover:bg-gray-700"
                    >
                      Edit
                    </button>
                    <button
                      onClick={() => handleReject(item.id)}
                      className="px-3 py-1.5 text-xs border border-red-600 rounded text-red-400 hover:bg-red-900/20"
                    >
                      Reject
                    </button>
                    <button
                      onClick={() => deleteOutbox(item.id)}
                      className="px-3 py-1.5 text-xs border border-gray-700 rounded text-gray-500 hover:bg-gray-800"
                    >
                      Delete
                    </button>
                  </>
                )}
              </div>
            )}

            {item.status === 'sent' && item.sent_at && (
              <p className="text-gray-500 text-xs">Sent {new Date(item.sent_at).toLocaleString()}</p>
            )}
          </div>
        ))}

        {outbox?.length === 0 && (
          <p className="text-gray-500 text-center py-8">No outbox messages</p>
        )}
      </div>
    </div>
  )
}
```

- [ ] **Step 2: Add outbox route**

In `frontend/src/presentation/routes/index.tsx`, add import and route:

```tsx
import { OutboxPage } from '../pages/OutboxPage'

// Inside the DashboardLayout children array:
{ path: 'dashboard/outbox', element: <OutboxPage /> },
```

- [ ] **Step 3: Commit**

```bash
cd /home/nst/GolandProjects/pdt && git add frontend/src/presentation/pages/OutboxPage.tsx frontend/src/presentation/routes/index.tsx && git commit -m "feat(wa): add Outbox approval page with send/edit/reject workflow"
```

---

## Task 17: Docker Compose & Final Verification

**Files:**
- Verify all changes compile and wire together

- [ ] **Step 1: Verify backend builds**

```bash
cd /home/nst/GolandProjects/pdt/backend && go build ./...
```

Expected: Clean build with no errors.

- [ ] **Step 2: Verify frontend builds**

```bash
cd /home/nst/GolandProjects/pdt/frontend && npm run build
```

Expected: Build succeeds.

- [ ] **Step 3: Verify docker-compose is valid**

```bash
cd /home/nst/GolandProjects/pdt && docker compose config --quiet
```

Expected: No errors.

- [ ] **Step 4: Commit any remaining fixes**

If any compilation fixes were needed:

```bash
cd /home/nst/GolandProjects/pdt && git add -A && git commit -m "fix(wa): resolve compilation issues from integration"
```

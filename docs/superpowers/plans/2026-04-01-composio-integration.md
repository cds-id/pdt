# Composio.dev Integration Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Augment all PDT agents with Composio.dev tools (Gmail, Notion, Google Calendar, LinkedIn) via a decorator pattern, with per-user API key and OAuth connection management from the Settings dashboard.

**Architecture:** New `composio` package provides a REST client for Composio API v3. A `ComposioEnhancedAgent` decorator wraps existing agents, injecting Composio tools into their `Tools()` and routing execution via `ExecuteTool()`. Frontend adds a Composio section to SettingsPage with API key management and OAuth connection cards.

**Tech Stack:** Go (Gin, GORM, net/http), React (RTK Query, Shadcn UI, Tailwind CSS), Composio REST API v3

---

## File Map

### Backend — New Files
- `backend/internal/ai/composio/client.go` — HTTP client for Composio REST API
- `backend/internal/ai/composio/types.go` — Request/response structs
- `backend/internal/ai/composio/enhanced_agent.go` — Decorator that wraps Agent interface
- `backend/internal/models/composio.go` — ComposioConfig + ComposioConnection models
- `backend/internal/handlers/composio.go` — API endpoints for config + connections

### Backend — Modified Files
- `backend/internal/database/database.go` — Add models to AutoMigrate
- `backend/internal/handlers/chat.go` — Wrap agents with ComposioEnhancedAgent
- `backend/cmd/server/main.go` — Wire ComposioHandler, pass encryptor to ChatHandler for Composio wrapping
- `backend/internal/scheduler/engine.go` — Wrap agents in AgentBuilder with Composio tools

### Frontend — New Files
- `frontend/src/infrastructure/services/composio.service.ts` — RTK Query endpoints
- `frontend/src/presentation/components/settings/ComposioSettings.tsx` — Settings section component

### Frontend — Modified Files
- `frontend/src/infrastructure/constants/api.constants.ts` — Add COMPOSIO endpoints
- `frontend/src/infrastructure/services/api.ts` — Add 'Composio' tag type
- `frontend/src/presentation/pages/SettingsPage.tsx` — Import and render ComposioSettings

---

### Task 1: Composio API Types

**Files:**
- Create: `backend/internal/ai/composio/types.go`

- [ ] **Step 1: Create the types file**

```go
package composio

import "encoding/json"

// ToolDefinition is the shape returned by GET /api/v3/tools.
type ToolDefinition struct {
	Slug            string          `json:"slug"`
	Name            string          `json:"name"`
	Description     string          `json:"description"`
	InputParameters json.RawMessage `json:"input_parameters"`
	AppName         string          `json:"appName"`
}

// GetToolsResponse is the response from GET /api/v3/tools.
type GetToolsResponse struct {
	Tools []ToolDefinition `json:"tools"`
}

// ExecuteRequest is the body for POST /api/v3/tools/execute/{slug}.
type ExecuteRequest struct {
	ConnectedAccountID string          `json:"connectedAccountId"`
	Arguments          json.RawMessage `json:"input"`
}

// ExecuteResponse is the response from tool execution.
type ExecuteResponse struct {
	Data       json.RawMessage `json:"data"`
	Error      string          `json:"error,omitempty"`
	Successful bool            `json:"successfull"`
}

// ConnectedAccount is a user's connected service account.
type ConnectedAccount struct {
	ID        string `json:"id"`
	AppName   string `json:"appName"`
	Status    string `json:"status"`
	CreatedAt string `json:"createdAt"`
}

// GetConnectedAccountsResponse is the response from GET /api/v3/connected_accounts.
type GetConnectedAccountsResponse struct {
	Items []ConnectedAccount `json:"items"`
}

// InitiateConnectionRequest is the body for POST /api/v3/connected_accounts.
type InitiateConnectionRequest struct {
	IntegrationID string `json:"integrationId"`
	RedirectURI   string `json:"redirectUri"`
	UserID        string `json:"entityId"`
}

// InitiateConnectionResponse is the response with the OAuth redirect URL.
type InitiateConnectionResponse struct {
	ConnectionStatus string `json:"connectionStatus"`
	ConnectedAccountID string `json:"connectedAccountId"`
	RedirectURL      string `json:"redirectUrl"`
}
```

- [ ] **Step 2: Commit**

```bash
git add backend/internal/ai/composio/types.go
git commit -m "feat(composio): add API types for Composio REST client"
```

---

### Task 2: Composio REST Client

**Files:**
- Create: `backend/internal/ai/composio/client.go`

- [ ] **Step 1: Create the client**

```go
package composio

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/cds-id/pdt/backend/internal/ai/minimax"
)

const baseURL = "https://backend.composio.dev/api/v3"

// Client talks to the Composio REST API.
type Client struct {
	http *http.Client
}

// NewClient creates a Composio client.
func NewClient() *Client {
	return &Client{
		http: &http.Client{Timeout: 30 * time.Second},
	}
}

// GetTools fetches tool definitions for the given toolkits and converts them to minimax.Tool format.
func (c *Client) GetTools(apiKey string, toolkits []string) ([]minimax.Tool, error) {
	u, _ := url.Parse(baseURL + "/tools")
	q := u.Query()
	q.Set("toolkit_slug", strings.Join(toolkits, ","))
	u.RawQuery = q.Encode()

	req, err := http.NewRequest("GET", u.String(), nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("x-api-key", apiKey)

	resp, err := c.http.Do(req)
	if err != nil {
		return nil, fmt.Errorf("composio get tools: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("composio get tools: status %d: %s", resp.StatusCode, body)
	}

	var toolsResp GetToolsResponse
	if err := json.NewDecoder(resp.Body).Decode(&toolsResp); err != nil {
		return nil, fmt.Errorf("composio decode tools: %w", err)
	}

	tools := make([]minimax.Tool, 0, len(toolsResp.Tools))
	for _, t := range toolsResp.Tools {
		tools = append(tools, minimax.Tool{
			Name:        t.Slug,
			Description: t.Description,
			InputSchema: t.InputParameters,
		})
	}
	return tools, nil
}

// ExecuteTool calls a Composio tool with the given arguments.
func (c *Client) ExecuteTool(apiKey, toolSlug, connectedAccountID string, args json.RawMessage) (json.RawMessage, error) {
	body, _ := json.Marshal(ExecuteRequest{
		ConnectedAccountID: connectedAccountID,
		Arguments:          args,
	})

	req, err := http.NewRequest("POST", baseURL+"/tools/execute/"+toolSlug, bytes.NewReader(body))
	if err != nil {
		return nil, err
	}
	req.Header.Set("x-api-key", apiKey)
	req.Header.Set("Content-Type", "application/json")

	resp, err := c.http.Do(req)
	if err != nil {
		return nil, fmt.Errorf("composio execute: %w", err)
	}
	defer resp.Body.Close()

	respBody, _ := io.ReadAll(resp.Body)

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("composio execute: status %d: %s", resp.StatusCode, respBody)
	}

	var execResp ExecuteResponse
	if err := json.Unmarshal(respBody, &execResp); err != nil {
		return respBody, nil // return raw if can't parse
	}

	if !execResp.Successful {
		return nil, fmt.Errorf("composio tool error: %s", execResp.Error)
	}

	return execResp.Data, nil
}

// GetConnectedAccounts lists a user's connected service accounts.
func (c *Client) GetConnectedAccounts(apiKey, entityID string) ([]ConnectedAccount, error) {
	u, _ := url.Parse(baseURL + "/connected_accounts")
	q := u.Query()
	q.Set("user_uuid", entityID)
	u.RawQuery = q.Encode()

	req, err := http.NewRequest("GET", u.String(), nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("x-api-key", apiKey)

	resp, err := c.http.Do(req)
	if err != nil {
		return nil, fmt.Errorf("composio connected accounts: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("composio connected accounts: status %d: %s", resp.StatusCode, body)
	}

	var result GetConnectedAccountsResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, err
	}
	return result.Items, nil
}

// InitiateConnection starts an OAuth flow for a toolkit and returns the redirect URL.
func (c *Client) InitiateConnection(apiKey, integrationID, redirectURI, entityID string) (*InitiateConnectionResponse, error) {
	body, _ := json.Marshal(InitiateConnectionRequest{
		IntegrationID: integrationID,
		RedirectURI:   redirectURI,
		UserID:        entityID,
	})

	req, err := http.NewRequest("POST", baseURL+"/connected_accounts", bytes.NewReader(body))
	if err != nil {
		return nil, err
	}
	req.Header.Set("x-api-key", apiKey)
	req.Header.Set("Content-Type", "application/json")

	resp, err := c.http.Do(req)
	if err != nil {
		return nil, fmt.Errorf("composio initiate: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		respBody, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("composio initiate: status %d: %s", resp.StatusCode, respBody)
	}

	var result InitiateConnectionResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, err
	}
	return &result, nil
}
```

- [ ] **Step 2: Commit**

```bash
git add backend/internal/ai/composio/client.go
git commit -m "feat(composio): add REST client for Composio API v3"
```

---

### Task 3: Database Models

**Files:**
- Create: `backend/internal/models/composio.go`
- Modify: `backend/internal/database/database.go`

- [ ] **Step 1: Create the models file**

```go
package models

import "time"

// ComposioConfig stores a user's Composio API key (encrypted).
type ComposioConfig struct {
	ID        uint      `gorm:"primarykey" json:"id"`
	UserID    uint      `gorm:"uniqueIndex;not null" json:"user_id"`
	APIKey    string    `gorm:"type:text;not null" json:"-"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

// ComposioConnection tracks a user's connected Composio service.
type ComposioConnection struct {
	ID              uint      `gorm:"primarykey" json:"id"`
	UserID          uint      `gorm:"not null" json:"user_id"`
	Toolkit         string    `gorm:"type:varchar(100);not null" json:"toolkit"`
	IntegrationID   string    `gorm:"type:varchar(255)" json:"integration_id"`
	AccountID       string    `gorm:"type:varchar(255)" json:"account_id"`
	Status          string    `gorm:"type:varchar(50);default:inactive" json:"status"`
	CreatedAt       time.Time `json:"created_at"`
	UpdatedAt       time.Time `json:"updated_at"`
}
```

- [ ] **Step 2: Add models to AutoMigrate in `backend/internal/database/database.go`**

Add `&models.ComposioConfig{}` and `&models.ComposioConnection{}` to the `db.AutoMigrate(...)` call after `&models.AgentScheduleRunStep{}`:

```go
&models.AgentScheduleRunStep{},
&models.ComposioConfig{},
&models.ComposioConnection{},
```

- [ ] **Step 3: Commit**

```bash
git add backend/internal/models/composio.go backend/internal/database/database.go
git commit -m "feat(composio): add database models for config and connections"
```

---

### Task 4: ComposioEnhancedAgent Decorator

**Files:**
- Create: `backend/internal/ai/composio/enhanced_agent.go`

- [ ] **Step 1: Create the decorator**

```go
package composio

import (
	"context"
	"encoding/json"
	"log"

	"github.com/cds-id/pdt/backend/internal/ai/agent"
	"github.com/cds-id/pdt/backend/internal/ai/minimax"
)

// EnhancedAgent wraps an existing Agent, injecting Composio tools alongside native tools.
type EnhancedAgent struct {
	Inner         agent.Agent
	client        *Client
	apiKey        string
	composioTools []minimax.Tool
	// toolToAccount maps tool slug -> connected account ID for execution
	toolToAccount map[string]string
}

// NewEnhancedAgent creates a decorator that augments an agent with Composio tools.
// composioTools and toolToAccount should be pre-fetched at session start.
func NewEnhancedAgent(inner agent.Agent, client *Client, apiKey string, composioTools []minimax.Tool, toolToAccount map[string]string) *EnhancedAgent {
	return &EnhancedAgent{
		Inner:         inner,
		client:        client,
		apiKey:        apiKey,
		composioTools: composioTools,
		toolToAccount: toolToAccount,
	}
}

func (e *EnhancedAgent) Name() string        { return e.Inner.Name() }
func (e *EnhancedAgent) SystemPrompt() string { return e.Inner.SystemPrompt() }

func (e *EnhancedAgent) Tools() []minimax.Tool {
	native := e.Inner.Tools()
	all := make([]minimax.Tool, 0, len(native)+len(e.composioTools))
	all = append(all, native...)
	all = append(all, e.composioTools...)
	return all
}

func (e *EnhancedAgent) ExecuteTool(ctx context.Context, name string, args json.RawMessage) (any, error) {
	// Check if this is a Composio tool
	if accountID, ok := e.toolToAccount[name]; ok {
		result, err := e.client.ExecuteTool(e.apiKey, name, accountID, args)
		if err != nil {
			log.Printf("[composio] tool %s error: %v", name, err)
			return map[string]string{"error": err.Error()}, nil
		}
		// Return as parsed JSON so it marshals cleanly
		var parsed any
		if json.Unmarshal(result, &parsed) == nil {
			return parsed, nil
		}
		return string(result), nil
	}
	// Delegate to inner agent
	return e.Inner.ExecuteTool(ctx, name, args)
}
```

- [ ] **Step 2: Commit**

```bash
git add backend/internal/ai/composio/enhanced_agent.go
git commit -m "feat(composio): add EnhancedAgent decorator for tool injection"
```

---

### Task 5: Helper to Build Enhanced Agents

**Files:**
- Create: `backend/internal/ai/composio/builder.go`

- [ ] **Step 1: Create the builder helper**

This function handles the "fetch tools once, wrap all agents" logic used by both chat handler and scheduler.

```go
package composio

import (
	"log"

	"github.com/cds-id/pdt/backend/internal/ai/agent"
	"github.com/cds-id/pdt/backend/internal/ai/minimax"
	"github.com/cds-id/pdt/backend/internal/crypto"
	"github.com/cds-id/pdt/backend/internal/models"
	"gorm.io/gorm"
)

// supportedToolkits defines which Composio toolkits map to which connection name.
var supportedToolkits = []string{"gmail", "notion", "googlecalendar", "linkedin"}

// WrapAgents takes a list of agents and returns them wrapped with Composio tools
// if the user has a Composio config. Returns the original agents unchanged if not.
func WrapAgents(db *gorm.DB, encryptor *crypto.Encryptor, client *Client, userID uint, agents []agent.Agent) []agent.Agent {
	// Check if user has Composio configured
	var cfg models.ComposioConfig
	if err := db.Where("user_id = ?", userID).First(&cfg).Error; err != nil {
		return agents // no config, return as-is
	}

	apiKey, err := encryptor.Decrypt(cfg.APIKey)
	if err != nil {
		log.Printf("[composio] decrypt api key for user %d: %v", userID, err)
		return agents
	}

	// Get active connections
	var connections []models.ComposioConnection
	db.Where("user_id = ? AND status = ?", userID, "active").Find(&connections)
	if len(connections) == 0 {
		return agents // no active connections
	}

	// Build toolkit list and account mapping from active connections
	var activeToolkits []string
	accountByToolkit := make(map[string]string)
	for _, conn := range connections {
		activeToolkits = append(activeToolkits, conn.Toolkit)
		accountByToolkit[conn.Toolkit] = conn.AccountID
	}

	// Fetch tools from Composio for active toolkits only
	tools, err := client.GetTools(apiKey, activeToolkits)
	if err != nil {
		log.Printf("[composio] fetch tools for user %d: %v", userID, err)
		return agents
	}

	if len(tools) == 0 {
		return agents
	}

	// Build tool slug -> account ID mapping
	// We need to figure out which toolkit each tool belongs to.
	// Composio tool slugs are prefixed with the app name: GMAIL_SEND_EMAIL, NOTION_CREATE_PAGE, etc.
	toolToAccount := buildToolAccountMap(tools, connections)

	// Wrap each agent
	wrapped := make([]agent.Agent, len(agents))
	for i, a := range agents {
		wrapped[i] = NewEnhancedAgent(a, client, apiKey, tools, toolToAccount)
	}
	return wrapped
}

// buildToolAccountMap maps each tool slug to its connected account ID.
// Composio tool slugs follow the pattern: APPNAME_ACTION (e.g., GMAIL_SEND_EMAIL).
func buildToolAccountMap(tools []minimax.Tool, connections []models.ComposioConnection) map[string]string {
	// Build app name -> account ID from connections
	appToAccount := make(map[string]string)
	for _, conn := range connections {
		appToAccount[conn.Toolkit] = conn.AccountID
	}

	result := make(map[string]string)
	for _, tool := range tools {
		// Try each known toolkit as a prefix
		for toolkit, accountID := range appToAccount {
			prefix := toolkit + "_"
			if len(tool.Name) > len(prefix) && tool.Name[:len(prefix)] == toUpper(prefix) {
				result[tool.Name] = accountID
				break
			}
		}
	}
	return result
}

func toUpper(s string) string {
	b := make([]byte, len(s))
	for i := range s {
		c := s[i]
		if c >= 'a' && c <= 'z' {
			c -= 32
		}
		b[i] = c
	}
	return string(b)
}
```

- [ ] **Step 2: Commit**

```bash
git add backend/internal/ai/composio/builder.go
git commit -m "feat(composio): add WrapAgents builder for session-scoped tool injection"
```

---

### Task 6: Composio API Handler

**Files:**
- Create: `backend/internal/handlers/composio.go`

- [ ] **Step 1: Create the handler**

```go
package handlers

import (
	"fmt"
	"net/http"

	"github.com/cds-id/pdt/backend/internal/ai/composio"
	"github.com/cds-id/pdt/backend/internal/crypto"
	"github.com/cds-id/pdt/backend/internal/models"
	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

type ComposioHandler struct {
	DB              *gorm.DB
	Encryptor       *crypto.Encryptor
	ComposioClient  *composio.Client
}

// GetConfig returns whether the user has a Composio API key configured.
func (h *ComposioHandler) GetConfig(c *gin.Context) {
	userID := c.GetUint("user_id")

	var cfg models.ComposioConfig
	err := h.DB.Where("user_id = ?", userID).First(&cfg).Error
	hasKey := err == nil

	c.JSON(http.StatusOK, gin.H{
		"configured": hasKey,
	})
}

// SaveConfig saves or updates the user's Composio API key.
func (h *ComposioHandler) SaveConfig(c *gin.Context) {
	userID := c.GetUint("user_id")

	var req struct {
		APIKey string `json:"api_key" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Validate the API key by making a test call
	_, err := h.ComposioClient.GetConnectedAccounts(req.APIKey, fmt.Sprintf("pdt-user-%d", userID))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid Composio API key"})
		return
	}

	encrypted, err := h.Encryptor.Encrypt(req.APIKey)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "encryption failed"})
		return
	}

	var cfg models.ComposioConfig
	result := h.DB.Where("user_id = ?", userID).First(&cfg)
	if result.Error != nil {
		// Create new
		cfg = models.ComposioConfig{
			UserID: userID,
			APIKey: encrypted,
		}
		h.DB.Create(&cfg)
	} else {
		// Update
		h.DB.Model(&cfg).Update("api_key", encrypted)
	}

	c.JSON(http.StatusOK, gin.H{"configured": true})
}

// DeleteConfig removes the user's Composio API key and all connections.
func (h *ComposioHandler) DeleteConfig(c *gin.Context) {
	userID := c.GetUint("user_id")

	h.DB.Where("user_id = ?", userID).Delete(&models.ComposioConnection{})
	h.DB.Where("user_id = ?", userID).Delete(&models.ComposioConfig{})

	c.JSON(http.StatusOK, gin.H{"message": "Composio configuration removed"})
}

// ListConnections returns the user's Composio service connections.
func (h *ComposioHandler) ListConnections(c *gin.Context) {
	userID := c.GetUint("user_id")

	var connections []models.ComposioConnection
	h.DB.Where("user_id = ?", userID).Find(&connections)

	c.JSON(http.StatusOK, connections)
}

// InitiateConnection starts the OAuth flow for a toolkit.
func (h *ComposioHandler) InitiateConnection(c *gin.Context) {
	userID := c.GetUint("user_id")
	toolkit := c.Param("toolkit")

	var req struct {
		IntegrationID string `json:"integration_id" binding:"required"`
		RedirectURI   string `json:"redirect_uri" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Get user's API key
	var cfg models.ComposioConfig
	if err := h.DB.Where("user_id = ?", userID).First(&cfg).Error; err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Composio API key not configured"})
		return
	}

	apiKey, err := h.Encryptor.Decrypt(cfg.APIKey)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "decryption failed"})
		return
	}

	entityID := fmt.Sprintf("pdt-user-%d", userID)
	result, err := h.ComposioClient.InitiateConnection(apiKey, req.IntegrationID, req.RedirectURI, entityID)
	if err != nil {
		c.JSON(http.StatusBadGateway, gin.H{"error": err.Error()})
		return
	}

	// Upsert the connection record
	var conn models.ComposioConnection
	dbResult := h.DB.Where("user_id = ? AND toolkit = ?", userID, toolkit).First(&conn)
	if dbResult.Error != nil {
		conn = models.ComposioConnection{
			UserID:        userID,
			Toolkit:       toolkit,
			IntegrationID: req.IntegrationID,
			AccountID:     result.ConnectedAccountID,
			Status:        result.ConnectionStatus,
		}
		h.DB.Create(&conn)
	} else {
		h.DB.Model(&conn).Updates(map[string]any{
			"account_id":     result.ConnectedAccountID,
			"integration_id": req.IntegrationID,
			"status":         result.ConnectionStatus,
		})
	}

	c.JSON(http.StatusOK, gin.H{
		"redirect_url": result.RedirectURL,
		"account_id":   result.ConnectedAccountID,
		"status":       result.ConnectionStatus,
	})
}

// SyncConnections refreshes connection statuses from Composio API.
func (h *ComposioHandler) SyncConnections(c *gin.Context) {
	userID := c.GetUint("user_id")

	var cfg models.ComposioConfig
	if err := h.DB.Where("user_id = ?", userID).First(&cfg).Error; err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Composio API key not configured"})
		return
	}

	apiKey, err := h.Encryptor.Decrypt(cfg.APIKey)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "decryption failed"})
		return
	}

	entityID := fmt.Sprintf("pdt-user-%d", userID)
	accounts, err := h.ComposioClient.GetConnectedAccounts(apiKey, entityID)
	if err != nil {
		c.JSON(http.StatusBadGateway, gin.H{"error": err.Error()})
		return
	}

	// Update local connection records based on Composio's state
	for _, acc := range accounts {
		h.DB.Model(&models.ComposioConnection{}).
			Where("user_id = ? AND account_id = ?", userID, acc.ID).
			Update("status", acc.Status)
	}

	var connections []models.ComposioConnection
	h.DB.Where("user_id = ?", userID).Find(&connections)

	c.JSON(http.StatusOK, connections)
}

// DeleteConnection removes a service connection.
func (h *ComposioHandler) DeleteConnection(c *gin.Context) {
	userID := c.GetUint("user_id")
	toolkit := c.Param("toolkit")

	h.DB.Where("user_id = ? AND toolkit = ?", userID, toolkit).Delete(&models.ComposioConnection{})

	c.JSON(http.StatusOK, gin.H{"message": fmt.Sprintf("%s disconnected", toolkit)})
}
```

- [ ] **Step 2: Commit**

```bash
git add backend/internal/handlers/composio.go
git commit -m "feat(composio): add API handler for config and connection management"
```

---

### Task 7: Wire Backend Routes and Agent Wrapping

**Files:**
- Modify: `backend/cmd/server/main.go`
- Modify: `backend/internal/handlers/chat.go`

- [ ] **Step 1: Add Composio client and handler to `main.go`**

After the `miniMaxClient` initialization (around line 116), add:

```go
composioClient := composio.NewClient()
```

After `scheduleHandler` declaration (around line 193), add:

```go
composioHandler := &handlers.ComposioHandler{
	DB:             db,
	Encryptor:      encryptor,
	ComposioClient: composioClient,
}
```

Add the import for composio package:
```go
"github.com/cds-id/pdt/backend/internal/ai/composio"
```

In the `protected` route group (after the `/schedules` block), add:

```go
comp := protected.Group("/composio")
{
	comp.GET("/config", composioHandler.GetConfig)
	comp.PUT("/config", composioHandler.SaveConfig)
	comp.DELETE("/config", composioHandler.DeleteConfig)
	comp.GET("/connections", composioHandler.ListConnections)
	comp.POST("/connections/:toolkit/initiate", composioHandler.InitiateConnection)
	comp.POST("/connections/sync", composioHandler.SyncConnections)
	comp.DELETE("/connections/:toolkit", composioHandler.DeleteConnection)
}
```

- [ ] **Step 2: Add Composio fields to ChatHandler and wrap agents in `chat.go`**

Add fields to `ChatHandler` struct:

```go
type ChatHandler struct {
	DB              *gorm.DB
	MiniMaxClient   *minimax.Client
	Encryptor       *crypto.Encryptor
	R2              *storage.R2Client
	ReportGenerator *report.Generator
	ContextWindow   int
	WaManager       *waService.Manager
	WeaviateClient  *wvClient.Client
	ScheduleEngine  *scheduler.Engine
	ComposioClient  *composio.Client
}
```

In `HandleWebSocket`, after building the orchestrator's agent list (around line 128-137), wrap agents before passing to orchestrator. Replace the `NewOrchestrator` call:

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

orchestrator := agent.NewOrchestrator(h.MiniMaxClient, agents...)
```

Add the composio import to chat.go:
```go
"github.com/cds-id/pdt/backend/internal/ai/composio"
```

- [ ] **Step 3: Wire ComposioClient into ChatHandler in `main.go`**

Update the `chatHandler` initialization to include `ComposioClient`:

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
	ComposioClient:  composioClient,
}
```

- [ ] **Step 4: Wrap agents in scheduler's AgentBuilder in `main.go`**

Update the `SetAgentBuilder` closure to also wrap with Composio:

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
	return composio.WrapAgents(db, encryptor, composioClient, userID, agents)
})
```

- [ ] **Step 5: Commit**

```bash
git add backend/cmd/server/main.go backend/internal/handlers/chat.go
git commit -m "feat(composio): wire routes, agent wrapping in chat and scheduler"
```

---

### Task 8: Frontend API Constants and Service

**Files:**
- Modify: `frontend/src/infrastructure/constants/api.constants.ts`
- Modify: `frontend/src/infrastructure/services/api.ts`
- Create: `frontend/src/infrastructure/services/composio.service.ts`

- [ ] **Step 1: Add Composio endpoints to `api.constants.ts`**

Add after the `WA` block:

```typescript
// Composio Endpoints
COMPOSIO: {
  CONFIG: '/composio/config',
  CONNECTIONS: '/composio/connections',
  INITIATE: (toolkit: string) => `/composio/connections/${toolkit}/initiate`,
  SYNC: '/composio/connections/sync',
  DISCONNECT: (toolkit: string) => `/composio/connections/${toolkit}`,
},
```

- [ ] **Step 2: Add 'Composio' tag type to `api.ts`**

Add `'Composio'` to the `tagTypes` array:

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
  'WhatsApp',
  'AIUsage',
  'Schedule',
  'Composio'
],
```

- [ ] **Step 3: Create the service file**

```typescript
import { api } from './api'
import { API_CONSTANTS } from '../constants/api.constants'

interface ComposioConfigResponse {
  configured: boolean
}

interface ComposioConnection {
  id: number
  user_id: number
  toolkit: string
  integration_id: string
  account_id: string
  status: string
  created_at: string
  updated_at: string
}

interface InitiateResponse {
  redirect_url: string
  account_id: string
  status: string
}

export const composioApi = api.injectEndpoints({
  endpoints: (builder) => ({
    getComposioConfig: builder.query<ComposioConfigResponse, void>({
      query: () => API_CONSTANTS.COMPOSIO.CONFIG,
      providesTags: [{ type: 'Composio' as const, id: 'CONFIG' }]
    }),
    saveComposioConfig: builder.mutation<ComposioConfigResponse, { api_key: string }>({
      query: (data) => ({
        url: API_CONSTANTS.COMPOSIO.CONFIG,
        method: 'PUT',
        body: data
      }),
      invalidatesTags: [{ type: 'Composio' as const, id: 'CONFIG' }]
    }),
    deleteComposioConfig: builder.mutation<void, void>({
      query: () => ({
        url: API_CONSTANTS.COMPOSIO.CONFIG,
        method: 'DELETE'
      }),
      invalidatesTags: ['Composio']
    }),
    listComposioConnections: builder.query<ComposioConnection[], void>({
      query: () => API_CONSTANTS.COMPOSIO.CONNECTIONS,
      providesTags: [{ type: 'Composio' as const, id: 'CONNECTIONS' }]
    }),
    initiateComposioConnection: builder.mutation<InitiateResponse, { toolkit: string; integration_id: string; redirect_uri: string }>({
      query: ({ toolkit, ...body }) => ({
        url: API_CONSTANTS.COMPOSIO.INITIATE(toolkit),
        method: 'POST',
        body
      }),
      invalidatesTags: [{ type: 'Composio' as const, id: 'CONNECTIONS' }]
    }),
    syncComposioConnections: builder.mutation<ComposioConnection[], void>({
      query: () => ({
        url: API_CONSTANTS.COMPOSIO.SYNC,
        method: 'POST'
      }),
      invalidatesTags: [{ type: 'Composio' as const, id: 'CONNECTIONS' }]
    }),
    deleteComposioConnection: builder.mutation<void, string>({
      query: (toolkit) => ({
        url: API_CONSTANTS.COMPOSIO.DISCONNECT(toolkit),
        method: 'DELETE'
      }),
      invalidatesTags: [{ type: 'Composio' as const, id: 'CONNECTIONS' }]
    })
  })
})

export const {
  useGetComposioConfigQuery,
  useSaveComposioConfigMutation,
  useDeleteComposioConfigMutation,
  useListComposioConnectionsQuery,
  useInitiateComposioConnectionMutation,
  useSyncComposioConnectionsMutation,
  useDeleteComposioConnectionMutation
} = composioApi
```

- [ ] **Step 4: Commit**

```bash
git add frontend/src/infrastructure/constants/api.constants.ts \
       frontend/src/infrastructure/services/api.ts \
       frontend/src/infrastructure/services/composio.service.ts
git commit -m "feat(composio): add frontend API service and constants"
```

---

### Task 9: Frontend ComposioSettings Component

**Files:**
- Create: `frontend/src/presentation/components/settings/ComposioSettings.tsx`

- [ ] **Step 1: Create the component**

```tsx
import { useState } from 'react'
import { Save, Trash2, ExternalLink, RefreshCw } from 'lucide-react'

import {
  useGetComposioConfigQuery,
  useSaveComposioConfigMutation,
  useDeleteComposioConfigMutation,
  useListComposioConnectionsQuery,
  useInitiateComposioConnectionMutation,
  useSyncComposioConnectionsMutation,
  useDeleteComposioConnectionMutation
} from '@/infrastructure/services/composio.service'
import { Button } from '@/components/ui/button'
import { Input } from '@/components/ui/input'
import { DataCard, StatusBadge } from '@/presentation/components/common'

const TOOLKITS = [
  { slug: 'gmail', name: 'Gmail', description: 'Send and read emails' },
  { slug: 'notion', name: 'Notion', description: 'Create and query pages' },
  { slug: 'googlecalendar', name: 'Google Calendar', description: 'Manage events' },
  { slug: 'linkedin', name: 'LinkedIn', description: 'Create posts, view profile' }
]

export function ComposioSettings() {
  const { data: config } = useGetComposioConfigQuery()
  const { data: connections = [] } = useListComposioConnectionsQuery(undefined, {
    skip: !config?.configured
  })
  const [saveConfig, { isLoading: isSaving }] = useSaveComposioConfigMutation()
  const [deleteConfig] = useDeleteComposioConfigMutation()
  const [initiateConnection] = useInitiateComposioConnectionMutation()
  const [syncConnections, { isLoading: isSyncing }] = useSyncComposioConnectionsMutation()
  const [deleteConnection] = useDeleteComposioConnectionMutation()

  const [apiKey, setApiKey] = useState('')
  const [message, setMessage] = useState<{ type: 'success' | 'error'; text: string } | null>(null)

  const handleSaveKey = async () => {
    setMessage(null)
    try {
      await saveConfig({ api_key: apiKey }).unwrap()
      setApiKey('')
      setMessage({ type: 'success', text: 'API key saved and validated!' })
    } catch {
      setMessage({ type: 'error', text: 'Invalid API key or connection failed.' })
    }
  }

  const handleRemoveKey = async () => {
    if (!confirm('Remove Composio API key and all connections?')) return
    await deleteConfig().unwrap()
    setMessage(null)
  }

  const handleConnect = async (toolkit: string) => {
    try {
      const redirectURI = window.location.origin + '/dashboard/settings'
      const result = await initiateConnection({
        toolkit,
        integration_id: toolkit,
        redirect_uri: redirectURI
      }).unwrap()

      if (result.redirect_url) {
        window.open(result.redirect_url, '_blank', 'width=600,height=700')
      }
    } catch (err) {
      console.error('Failed to initiate connection:', err)
    }
  }

  const handleDisconnect = async (toolkit: string) => {
    if (!confirm(`Disconnect ${toolkit}?`)) return
    await deleteConnection(toolkit).unwrap()
  }

  const getConnectionStatus = (toolkit: string) => {
    const conn = connections.find((c) => c.toolkit === toolkit)
    return conn?.status === 'active' ? 'active' : 'inactive'
  }

  return (
    <>
      <DataCard title="Composio — External Tools">
        <p className="mb-3 text-xs text-pdt-neutral/50">
          Connect external services (Gmail, Notion, Calendar, LinkedIn) to use them as AI agent tools.
          Get your API key from{' '}
          <a
            href="https://app.composio.dev/settings"
            target="_blank"
            rel="noopener noreferrer"
            className="text-pdt-accent underline"
          >
            app.composio.dev
          </a>
          .
        </p>

        <div className="flex items-center gap-2">
          <Input
            type="password"
            placeholder="Composio API Key"
            value={apiKey}
            onChange={(e) => setApiKey(e.target.value)}
            className="border-pdt-accent/20 bg-pdt-primary-light text-pdt-neutral placeholder:text-pdt-neutral/40"
          />
          <Button
            type="button"
            variant="pdt"
            size="sm"
            disabled={!apiKey || isSaving}
            onClick={handleSaveKey}
          >
            <Save className="mr-1 size-3" />
            {isSaving ? 'Saving...' : 'Save'}
          </Button>
        </div>

        <div className="mt-2 flex items-center gap-2 text-xs">
          {config?.configured ? (
            <>
              <StatusBadge variant="success">Configured</StatusBadge>
              <Button
                type="button"
                variant="ghost"
                size="sm"
                onClick={handleRemoveKey}
              >
                <Trash2 className="size-3 text-red-400" />
              </Button>
            </>
          ) : (
            <StatusBadge variant="danger">Not configured</StatusBadge>
          )}
        </div>

        {message && (
          <p className={`mt-2 text-sm ${message.type === 'success' ? 'text-green-400' : 'text-red-400'}`}>
            {message.text}
          </p>
        )}
      </DataCard>

      {config?.configured && (
        <DataCard
          title="Connected Services"
        >
          <div className="mb-3 flex items-center justify-between">
            <p className="text-xs text-pdt-neutral/50">
              Connect services to let your AI agents use them.
            </p>
            <Button
              type="button"
              variant="ghost"
              size="sm"
              disabled={isSyncing}
              onClick={() => syncConnections()}
            >
              <RefreshCw className={`size-3 ${isSyncing ? 'animate-spin' : ''}`} />
            </Button>
          </div>

          <div className="grid gap-3 sm:grid-cols-2">
            {TOOLKITS.map((tk) => {
              const status = getConnectionStatus(tk.slug)
              const isActive = status === 'active'

              return (
                <div
                  key={tk.slug}
                  className="flex items-center justify-between rounded-lg border border-pdt-neutral/10 p-3"
                >
                  <div>
                    <p className="text-sm font-medium text-pdt-neutral">{tk.name}</p>
                    <p className="text-xs text-pdt-neutral/50">{tk.description}</p>
                  </div>
                  <div className="flex items-center gap-2">
                    <StatusBadge variant={isActive ? 'success' : 'warning'}>
                      {isActive ? 'Connected' : 'Not Connected'}
                    </StatusBadge>
                    {isActive ? (
                      <Button
                        type="button"
                        variant="ghost"
                        size="sm"
                        onClick={() => handleDisconnect(tk.slug)}
                      >
                        <Trash2 className="size-3 text-red-400" />
                      </Button>
                    ) : (
                      <Button
                        type="button"
                        variant="pdtOutline"
                        size="sm"
                        onClick={() => handleConnect(tk.slug)}
                      >
                        <ExternalLink className="mr-1 size-3" />
                        Connect
                      </Button>
                    )}
                  </div>
                </div>
              )
            })}
          </div>
        </DataCard>
      )}
    </>
  )
}
```

- [ ] **Step 2: Commit**

```bash
git add frontend/src/presentation/components/settings/ComposioSettings.tsx
git commit -m "feat(composio): add ComposioSettings dashboard component"
```

---

### Task 10: Integrate ComposioSettings into SettingsPage

**Files:**
- Modify: `frontend/src/presentation/pages/SettingsPage.tsx`

- [ ] **Step 1: Import ComposioSettings**

Add this import at the top of the file:

```typescript
import { ComposioSettings } from '@/presentation/components/settings/ComposioSettings'
```

- [ ] **Step 2: Render the component**

Add `<ComposioSettings />` after the `</form>` closing tag and before the Jira Workspaces `<DataCard>`. Place it between the form's action buttons and the Jira Workspaces section:

```tsx
</form>

{/* Composio */}
<ComposioSettings />

{/* Jira Workspaces */}
```

- [ ] **Step 3: Commit**

```bash
git add frontend/src/presentation/pages/SettingsPage.tsx
git commit -m "feat(composio): integrate ComposioSettings into SettingsPage"
```

---

### Task 11: Build Verification

- [ ] **Step 1: Verify backend compiles**

Run:
```bash
cd /home/nst/GolandProjects/pdt/backend && go build ./...
```

Expected: No errors. Fix any import issues or type mismatches.

- [ ] **Step 2: Verify frontend compiles**

Run:
```bash
cd /home/nst/GolandProjects/pdt/frontend && npm run build
```

Expected: No errors. Fix any TypeScript issues.

- [ ] **Step 3: Commit any fixes**

```bash
git add -A
git commit -m "fix(composio): resolve build issues"
```

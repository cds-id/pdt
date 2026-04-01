package composio

import (
	"fmt"
	"log"
	"strings"

	"github.com/cds-id/pdt/backend/internal/ai/agent"
	"github.com/cds-id/pdt/backend/internal/ai/minimax"
	"github.com/cds-id/pdt/backend/internal/crypto"
	"github.com/cds-id/pdt/backend/internal/models"
	"gorm.io/gorm"
)

// WrapResult contains the wrapped agents and routing hints for the orchestrator.
type WrapResult struct {
	Agents          []agent.Agent
	ExternalToolHint string
}

// WrapAgents takes a list of agents and returns them wrapped with Composio tools
// if the user has a Composio config. Returns the original agents unchanged if not.
func WrapAgents(db *gorm.DB, encryptor *crypto.Encryptor, client *Client, userID uint, agents []agent.Agent) WrapResult {
	// Check if user has Composio configured
	noChange := WrapResult{Agents: agents}

	var cfg models.ComposioConfig
	if err := db.Where("user_id = ?", userID).First(&cfg).Error; err != nil {
		return noChange // no config, return as-is
	}

	apiKey, err := encryptor.Decrypt(cfg.APIKey)
	if err != nil {
		log.Printf("[composio] decrypt api key for user %d: %v", userID, err)
		return noChange
	}

	// Get active connections
	var connections []models.ComposioConnection
	db.Where("user_id = ? AND status = ?", userID, "active").Find(&connections)
	log.Printf("[composio] user %d: found %d active connections", userID, len(connections))
	if len(connections) == 0 {
		return noChange // no active connections
	}

	// Build toolkit list from active connections
	var activeToolkits []string
	for _, conn := range connections {
		activeToolkits = append(activeToolkits, conn.Toolkit)
	}
	log.Printf("[composio] user %d: active toolkits: %v", userID, activeToolkits)

	// Fetch tools from Composio for active toolkits only
	tools, err := client.GetTools(apiKey, activeToolkits)
	if err != nil {
		log.Printf("[composio] fetch tools for user %d: %v", userID, err)
		return noChange
	}

	log.Printf("[composio] user %d: fetched %d tools", userID, len(tools))
	if len(tools) == 0 {
		return noChange
	}

	// Build tool slug -> account ID mapping
	toolToAccount := buildToolAccountMap(tools, connections)

	// Wrap each agent
	entityID := fmt.Sprintf("pdt-user-%d", userID)
	wrapped := make([]agent.Agent, len(agents))
	for i, a := range agents {
		wrapped[i] = NewEnhancedAgent(a, client, apiKey, entityID, tools, toolToAccount)
	}

	// Build routing hint for the orchestrator
	hint := fmt.Sprintf("The user has connected external services via Composio: %s. ALL agents can handle requests for these services. For example, if the user asks about LinkedIn, route to any agent (e.g. 'git' or 'report') — they all have the external tools available.", strings.Join(activeToolkits, ", "))

	return WrapResult{Agents: wrapped, ExternalToolHint: hint}
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
			prefix := toUpper(toolkit + "_")
			if len(tool.Name) > len(prefix) && tool.Name[:len(prefix)] == prefix {
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

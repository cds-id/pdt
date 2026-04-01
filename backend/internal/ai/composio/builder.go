package composio

import (
	"log"

	"github.com/cds-id/pdt/backend/internal/ai/agent"
	"github.com/cds-id/pdt/backend/internal/ai/minimax"
	"github.com/cds-id/pdt/backend/internal/crypto"
	"github.com/cds-id/pdt/backend/internal/models"
	"gorm.io/gorm"
)

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

	// Build toolkit list from active connections
	var activeToolkits []string
	for _, conn := range connections {
		activeToolkits = append(activeToolkits, conn.Toolkit)
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

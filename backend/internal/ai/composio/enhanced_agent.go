package composio

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"strings"

	"github.com/cds-id/pdt/backend/internal/ai/agent"
	"github.com/cds-id/pdt/backend/internal/ai/minimax"
)

// EnhancedAgent wraps an existing Agent, injecting Composio tools alongside native tools.
type EnhancedAgent struct {
	Inner         agent.Agent
	client        *Client
	apiKey        string
	entityID      string
	composioTools []minimax.Tool
	// toolToAccount maps tool slug -> connected account ID for execution
	toolToAccount map[string]string
}

// NewEnhancedAgent creates a decorator that augments an agent with Composio tools.
func NewEnhancedAgent(inner agent.Agent, client *Client, apiKey, entityID string, composioTools []minimax.Tool, toolToAccount map[string]string) *EnhancedAgent {
	return &EnhancedAgent{
		Inner:         inner,
		client:        client,
		apiKey:        apiKey,
		entityID:      entityID,
		composioTools: composioTools,
		toolToAccount: toolToAccount,
	}
}

func (e *EnhancedAgent) Name() string { return e.Inner.Name() }
func (e *EnhancedAgent) SystemPrompt() string {
	base := e.Inner.SystemPrompt()
	if len(e.composioTools) == 0 {
		return base
	}
	var toolNames []string
	for _, t := range e.composioTools {
		toolNames = append(toolNames, t.Name)
	}
	return base + fmt.Sprintf("\n\nYou also have access to external service tools via Composio: %s. Use these tools when the user asks about those services.", strings.Join(toolNames, ", "))
}

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
		result, err := e.client.ExecuteTool(e.apiKey, name, accountID, e.entityID, args)
		if err != nil {
			log.Printf("[composio] tool %s error: %v", name, err)
			return map[string]string{"error": err.Error()}, nil
		}
		var parsed any
		if json.Unmarshal(result, &parsed) == nil {
			return parsed, nil
		}
		return string(result), nil
	}
	// Delegate to inner agent
	return e.Inner.ExecuteTool(ctx, name, args)
}

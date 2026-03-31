package agent

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/cds-id/pdt/backend/internal/ai/minimax"
)


// TriggerAgentTool allows agents to dynamically trigger other agents during scheduled runs.
type TriggerAgentTool struct {
	Agents   map[string]Agent
	Client   *minimax.Client
	MaxDepth int
	Depth    int
}

type triggerArgs struct {
	Agent  string `json:"agent"`
	Prompt string `json:"prompt"`
}

func (t *TriggerAgentTool) Definition() minimax.Tool {
	return minimax.Tool{
		Name:        "trigger_agent",
		Description: "Trigger another agent with a specific prompt. Use this to delegate tasks to specialist agents.",
		InputSchema: json.RawMessage(`{
			"type": "object",
			"properties": {
				"agent": {
					"type": "string",
					"enum": ["git", "jira", "report", "proof", "briefing", "whatsapp", "scheduler"],
					"description": "The agent to trigger"
				},
				"prompt": {
					"type": "string",
					"description": "The message to send to the agent"
				}
			},
			"required": ["agent", "prompt"]
		}`),
	}
}

func (t *TriggerAgentTool) Execute(ctx context.Context, args json.RawMessage) (any, error) {
	if t.Depth >= t.MaxDepth {
		return map[string]string{"error": "maximum chain depth reached"}, nil
	}

	var a triggerArgs
	if err := json.Unmarshal(args, &a); err != nil {
		return nil, fmt.Errorf("parse trigger_agent args: %w", err)
	}

	target, ok := t.Agents[a.Agent]
	if !ok {
		return map[string]string{"error": fmt.Sprintf("unknown agent: %s", a.Agent)}, nil
	}

	messages := []minimax.Message{{Role: "user", Content: a.Prompt}}
	writer := nopStreamWriter{}
	result, err := RunLoop(ctx, t.Client, target, messages, writer)
	if err != nil {
		return map[string]string{"error": err.Error()}, nil
	}

	return map[string]string{"response": result.FullResponse}, nil
}

type nopStreamWriter struct{}

func (nopStreamWriter) WriteContent(string) error           { return nil }
func (nopStreamWriter) WriteThinking(string) error          { return nil }
func (nopStreamWriter) WriteToolStatus(string, string) error { return nil }
func (nopStreamWriter) WriteDone() error                    { return nil }
func (nopStreamWriter) WriteError(string) error             { return nil }

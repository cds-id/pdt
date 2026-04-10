package agent

import (
	"context"
	"encoding/json"

	"github.com/cds-id/pdt/backend/internal/ai/minimax"
	"github.com/cds-id/pdt/backend/internal/services/executive"
)

// minimaxExecutiveLLM implements ExecutiveLLM using the MiniMax client.
type minimaxExecutiveLLM struct {
	client *minimax.Client
	model  string
}

// NewMinimaxExecutiveLLM returns a production ExecutiveLLM backed by MiniMax/Anthropic.
func NewMinimaxExecutiveLLM(c *minimax.Client, model string) ExecutiveLLM {
	return &minimaxExecutiveLLM{client: c, model: model}
}

// Stream calls the MiniMax streaming API and forwards events to out.
// It always closes out when it returns (deferred).
func (m *minimaxExecutiveLLM) Stream(ctx context.Context, system, user string, out chan<- ExecutiveEvent) {
	defer close(out)

	// Build the emit_suggestion tool following the same pattern as report.go Tools().
	emitSuggestionTool := minimax.Tool{
		Name:        "emit_suggestion",
		Description: "Emit a structured suggestion (gap, stale work, or next step) identified during analysis.",
		InputSchema: json.RawMessage(`{
			"type": "object",
			"properties": {
				"kind":   {"type": "string", "enum": ["gap", "stale", "next_step"], "description": "Category of the suggestion"},
				"title":  {"type": "string", "description": "Short title summarising the suggestion"},
				"detail": {"type": "string", "description": "Full explanation with evidence and recommendation"},
				"refs":   {"type": "array", "items": {"type": "string"}, "description": "Optional list of reference tags, e.g. [jira:KEY] [commit:sha]"}
			},
			"required": ["kind", "title", "detail"]
		}`),
	}

	req := minimax.ChatRequest{
		Model: m.model,
		Messages: []minimax.Message{
			{Role: "system", Content: system},
			{Role: "user", Content: user},
		},
		Tools:  []minimax.Tool{emitSuggestionTool},
		Stream: true,
	}

	eventCh, err := m.client.ChatStream(req)
	if err != nil {
		out <- ExecutiveEvent{Kind: "error", Err: err}
		return
	}

	for {
		select {
		case <-ctx.Done():
			out <- ExecutiveEvent{Kind: "error", Err: ctx.Err()}
			return
		case evt, ok := <-eventCh:
			if !ok {
				// Channel closed — stream finished normally.
				return
			}

			if evt.Err != nil {
				out <- ExecutiveEvent{Kind: "error", Err: evt.Err}
				return
			}

			// Text delta.
			if evt.Content != "" {
				out <- ExecutiveEvent{Kind: "delta", Delta: evt.Content}
			}

			// Tool-use events: FinishReason == "tool_calls" carries accumulated ToolCalls.
			if evt.FinishReason == "tool_calls" {
				for _, tc := range evt.ToolCalls {
					if tc.Function.Name != "emit_suggestion" {
						continue
					}
					var s executive.Suggestion
					if err := json.Unmarshal([]byte(tc.Function.Arguments), &s); err != nil {
						out <- ExecutiveEvent{Kind: "error", Err: err}
						return
					}
					out <- ExecutiveEvent{Kind: "suggestion", Suggestion: &s}
				}
			}
		}
	}
}

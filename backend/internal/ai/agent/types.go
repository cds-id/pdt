package agent

import (
	"context"
	"encoding/json"

	"github.com/cds-id/pdt/backend/internal/ai/minimax"
)

type Agent interface {
	Name() string
	SystemPrompt() string
	Tools() []minimax.Tool
	ExecuteTool(ctx context.Context, name string, args json.RawMessage) (any, error)
}

type StreamWriter interface {
	WriteContent(content string) error
	WriteThinking(message string) error
	WriteToolStatus(toolName string, status string) error
	WriteDone() error
	WriteError(msg string) error
}

type LoopResult struct {
	FullResponse string
	ToolCalls    []minimax.ToolCall
	Usage        minimax.Usage
}

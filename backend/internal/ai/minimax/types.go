package minimax

import "encoding/json"

// These types are used by the agent framework. They provide a stable interface
// that decouples the agents from the underlying LLM SDK (Anthropic SDK via MiniMax).

// Message represents a conversation message used by the agent framework.
type Message struct {
	Role       string     `json:"role"`    // "user", "assistant", "system", "tool"
	Content    string     `json:"content"`
	ToolCalls  []ToolCall `json:"tool_calls,omitempty"`
	ToolCallID string     `json:"tool_call_id,omitempty"`
}

// Tool represents a tool definition for the agent framework.
type Tool struct {
	Name        string          `json:"name"`
	Description string          `json:"description"`
	InputSchema json.RawMessage `json:"input_schema"`
}

// ToolCall represents a tool invocation returned by the model.
type ToolCall struct {
	Index    int          `json:"-"`
	ID       string       `json:"id"`
	Type     string       `json:"type"`
	Function FunctionCall `json:"function"`
}

// FunctionCall holds the tool name and arguments.
type FunctionCall struct {
	Name      string `json:"name"`
	Arguments string `json:"arguments"`
}

// ChatRequest is the internal request format used by the agent framework.
type ChatRequest struct {
	Model       string    `json:"model"`
	Messages    []Message `json:"messages"`
	Tools       []Tool    `json:"tools,omitempty"`
	Stream      bool      `json:"stream"`
	Temperature float64   `json:"temperature,omitempty"`
}

// ChatResponse is the internal response format for non-streaming calls.
type ChatResponse struct {
	ID      string  `json:"id"`
	Content string  `json:"content"`
	Choices []Choice `json:"choices"`
	Usage   Usage   `json:"usage"`
}

// Choice represents a response choice (maps from Anthropic content blocks).
type Choice struct {
	Delta        Delta  `json:"delta"`
	FinishReason string `json:"finish_reason"`
}

// Delta holds response content.
type Delta struct {
	Role      string     `json:"role,omitempty"`
	Content   string     `json:"content,omitempty"`
	ToolCalls []ToolCall `json:"tool_calls,omitempty"`
}

// StreamEvent represents a single event from the streaming API.
type StreamEvent struct {
	Content      string
	ToolCalls    []ToolCall
	FinishReason string
	Usage        *Usage
	Err          error
}

// Usage tracks token consumption.
type Usage struct {
	PromptTokens     int `json:"prompt_tokens"`
	CompletionTokens int `json:"completion_tokens"`
	TotalTokens      int `json:"total_tokens"`
}

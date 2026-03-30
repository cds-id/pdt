package minimax

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"strings"

	"github.com/anthropics/anthropic-sdk-go"
	"github.com/anthropics/anthropic-sdk-go/option"
)

const defaultBaseURL = "https://api.minimax.io/anthropic"

// Client wraps the Anthropic SDK, pointing at MiniMax's Anthropic-compatible endpoint.
type Client struct {
	sdk   anthropic.Client
	Model string
}

// NewClient creates a MiniMax client using the Anthropic SDK.
func NewClient(apiKey, groupID string) *Client {
	sdk := anthropic.NewClient(
		option.WithAPIKey(apiKey),
		option.WithBaseURL(defaultBaseURL),
	)
	return &Client{
		sdk:   sdk,
		Model:   "MiniMax-M2.7",
	}
}

// ChatStream sends a streaming request and returns a channel of events.
func (c *Client) ChatStream(req ChatRequest) (<-chan StreamEvent, error) {
	model := req.Model
	if model == "" {
		model = c.Model
	}

	// Build Anthropic request
	params := c.buildParams(req, model)

	ch := make(chan StreamEvent, 32)

	go func() {
		defer close(ch)

		stream := c.sdk.Messages.NewStreaming(context.Background(), params)
		defer stream.Close()

		// Track tool use blocks being built
		toolUseBlocks := make(map[int]*ToolCall) // index -> ToolCall
		var totalUsage Usage

		for stream.Next() {
			evt := stream.Current()

			switch evt := evt.AsAny().(type) {
			case anthropic.MessageStartEvent:
				totalUsage.PromptTokens = int(evt.Message.Usage.InputTokens)

			case anthropic.ContentBlockStartEvent:
				if evt.ContentBlock.Type == "tool_use" {
					tb := evt.ContentBlock.AsToolUse()
					toolUseBlocks[int(evt.Index)] = &ToolCall{
						Index: int(evt.Index),
						ID:    tb.ID,
						Type:  "function",
						Function: FunctionCall{
							Name:      tb.Name,
							Arguments: "",
						},
					}
				}

			case anthropic.ContentBlockDeltaEvent:
				switch delta := evt.Delta.AsAny().(type) {
				case anthropic.TextDelta:
					ch <- StreamEvent{Content: delta.Text}
				case anthropic.InputJSONDelta:
					if tc, ok := toolUseBlocks[int(evt.Index)]; ok {
						tc.Function.Arguments += delta.PartialJSON
					}
				}

			case anthropic.MessageDeltaEvent:
				totalUsage.CompletionTokens = int(evt.Usage.OutputTokens)
				totalUsage.TotalTokens = totalUsage.PromptTokens + totalUsage.CompletionTokens

				reason := string(evt.Delta.StopReason)
				if reason == "tool_use" {
					var toolCalls []ToolCall
					for _, tc := range toolUseBlocks {
						toolCalls = append(toolCalls, *tc)
					}
					ch <- StreamEvent{
						ToolCalls:    toolCalls,
						FinishReason: "tool_calls",
						Usage:        &totalUsage,
					}
				} else {
					ch <- StreamEvent{
						FinishReason: reason,
						Usage:        &totalUsage,
					}
				}
			}
		}

		if err := stream.Err(); err != nil {
			ch <- StreamEvent{Err: fmt.Errorf("stream error: %w", err)}
		}
	}()

	return ch, nil
}

// Chat sends a non-streaming request and returns the response.
func (c *Client) Chat(req ChatRequest) (*ChatResponse, error) {
	model := req.Model
	if model == "" {
		model = c.Model
	}

	params := c.buildParams(req, model)

	resp, err := c.sdk.Messages.New(context.Background(), params)
	if err != nil {
		return nil, fmt.Errorf("minimax chat: %w", err)
	}

	// Convert response
	result := &ChatResponse{
		ID: resp.ID,
		Usage: Usage{
			PromptTokens:     int(resp.Usage.InputTokens),
			CompletionTokens: int(resp.Usage.OutputTokens),
			TotalTokens:      int(resp.Usage.InputTokens + resp.Usage.OutputTokens),
		},
	}

	// Extract text content and tool calls
	var textContent string
	var toolCalls []ToolCall
	for i, block := range resp.Content {
		switch block.Type {
		case "text":
			textContent += block.Text
		case "tool_use":
			argsJSON, _ := json.Marshal(block.Input)
			toolCalls = append(toolCalls, ToolCall{
				Index: i,
				ID:    block.ID,
				Type:  "function",
				Function: FunctionCall{
					Name:      block.Name,
					Arguments: string(argsJSON),
				},
			})
		}
	}

	finishReason := string(resp.StopReason)
	if finishReason == "tool_use" {
		finishReason = "tool_calls"
	}

	result.Choices = []Choice{{
		Delta: Delta{
			Role:      "assistant",
			Content:   textContent,
			ToolCalls: toolCalls,
		},
		FinishReason: finishReason,
	}}

	return result, nil
}

// buildParams converts our internal request format to Anthropic SDK params.
func (c *Client) buildParams(req ChatRequest, model string) anthropic.MessageNewParams {
	// Separate system message from conversation messages
	var systemPrompt string
	var messages []Message
	for _, m := range req.Messages {
		if m.Role == "system" {
			systemPrompt = m.Content
		} else {
			messages = append(messages, m)
		}
	}

	// Convert messages to Anthropic format
	var anthropicMessages []anthropic.MessageParam
	for _, m := range messages {
		switch m.Role {
		case "user":
			anthropicMessages = append(anthropicMessages, anthropic.NewUserMessage(
				anthropic.NewTextBlock(m.Content),
			))
		case "assistant":
			if len(m.ToolCalls) > 0 {
				// Assistant message with tool calls
				var blocks []anthropic.ContentBlockParamUnion
				if m.Content != "" {
					blocks = append(blocks, anthropic.NewTextBlock(m.Content))
				}
				for _, tc := range m.ToolCalls {
					var input interface{}
					json.Unmarshal([]byte(tc.Function.Arguments), &input)
					if input == nil {
						input = map[string]interface{}{}
					}
					blocks = append(blocks, anthropic.NewToolUseBlock(tc.ID, input, tc.Function.Name))
				}
				anthropicMessages = append(anthropicMessages, anthropic.NewAssistantMessage(blocks...))
			} else {
				anthropicMessages = append(anthropicMessages, anthropic.NewAssistantMessage(
					anthropic.NewTextBlock(m.Content),
				))
			}
		case "tool":
			// Tool result message
			anthropicMessages = append(anthropicMessages, anthropic.NewUserMessage(
				anthropic.NewToolResultBlock(m.ToolCallID, m.Content, false),
			))
		default:
			log.Printf("[minimax] unknown message role: %s", m.Role)
		}
	}

	// Convert tools
	var tools []anthropic.ToolUnionParam
	for _, t := range req.Tools {
		// Parse the full JSON schema to extract properties and required separately
		var schema struct {
			Properties json.RawMessage `json:"properties"`
			Required   []string        `json:"required"`
		}
		json.Unmarshal(t.InputSchema, &schema)

		// Build the proper InputSchema with separated fields
		inputSchema := anthropic.ToolInputSchemaParam{}
		if schema.Properties != nil {
			var props interface{}
			json.Unmarshal(schema.Properties, &props)
			inputSchema.Properties = props
		}
		if len(schema.Required) > 0 {
			inputSchema.Required = schema.Required
		}

		tools = append(tools, anthropic.ToolUnionParam{
			OfTool: &anthropic.ToolParam{
				Name:        t.Name,
				Description: anthropic.String(t.Description),
				InputSchema: inputSchema,
			},
		})
	}

	params := anthropic.MessageNewParams{
		Model:     model,
		MaxTokens: 4096,
		Messages:  anthropicMessages,
	}

	if systemPrompt != "" {
		params.System = []anthropic.TextBlockParam{
			{Text: systemPrompt},
		}
	}

	if len(tools) > 0 {
		params.Tools = tools
	}

	if req.Temperature > 0 {
		params.Temperature = anthropic.Float(req.Temperature)
	}

	return params
}

// mergeToolInputSchema handles the Anthropic SDK's InputSchema format.
// The SDK expects properties to be set on the InputSchema directly.
func mergeToolInputSchema(schema json.RawMessage) json.RawMessage {
	// Parse the JSON schema to extract just the properties
	var s map[string]interface{}
	if err := json.Unmarshal(schema, &s); err != nil {
		return schema
	}

	// If it has "properties", it's already a valid schema
	if _, ok := s["properties"]; ok {
		return schema
	}

	return schema
}

// Helper to check if a string contains a substring (used for error handling).
func contains(s, substr string) bool {
	return strings.Contains(s, substr)
}

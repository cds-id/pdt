package agent

import (
	"context"
	"encoding/json"
	"fmt"
	"log"

	"github.com/cds-id/pdt/backend/internal/ai/minimax"
)

const maxToolRounds = 10

func RunLoop(ctx context.Context, client *minimax.Client, agent Agent, messages []minimax.Message, writer StreamWriter) (*LoopResult, error) {
	systemMsg := minimax.Message{
		Role:    "system",
		Content: agent.SystemPrompt(),
	}
	conversation := append([]minimax.Message{systemMsg}, messages...)
	tools := agent.Tools()

	var totalUsage minimax.Usage

	for round := 0; round < maxToolRounds; round++ {
		req := minimax.ChatRequest{
			Messages:    conversation,
			Tools:       tools,
			Temperature: 0.7,
		}

		stream, err := client.ChatStream(req)
		if err != nil {
			return nil, fmt.Errorf("chat stream: %w", err)
		}

		var fullContent string
		var toolCalls []minimax.ToolCall
		var usage *minimax.Usage

		for evt := range stream {
			if ctx.Err() != nil {
				return nil, ctx.Err()
			}
			if evt.Err != nil {
				return nil, fmt.Errorf("stream event: %w", evt.Err)
			}
			if evt.Content != "" {
				fullContent += evt.Content
				if err := writer.WriteContent(evt.Content); err != nil {
					return nil, fmt.Errorf("write content: %w", err)
				}
			}
			if len(evt.ToolCalls) > 0 {
				toolCalls = append(toolCalls, evt.ToolCalls...)
			}
			if evt.Usage != nil {
				usage = evt.Usage
			}
		}

		if usage != nil {
			totalUsage.PromptTokens += usage.PromptTokens
			totalUsage.CompletionTokens += usage.CompletionTokens
			totalUsage.TotalTokens += usage.TotalTokens
		}

		if len(toolCalls) == 0 {
			return &LoopResult{
				FullResponse: fullContent,
				Usage:        totalUsage,
			}, nil
		}

		// Send thinking indicator before executing tools
		if err := writer.WriteThinking("Analyzing your request..."); err != nil {
			log.Printf("[agent-loop] write thinking error: %v", err)
		}

		conversation = append(conversation, minimax.Message{
			Role:      "assistant",
			Content:   fullContent,
			ToolCalls: toolCalls,
		})

		for _, tc := range toolCalls {
			if err := writer.WriteToolStatus(tc.Function.Name, "executing"); err != nil {
				log.Printf("[agent-loop] write tool status error: %v", err)
			}

			result, err := agent.ExecuteTool(ctx, tc.Function.Name, json.RawMessage(tc.Function.Arguments))
			if err != nil {
				result = map[string]string{"error": err.Error()}
			}

			resultJSON, _ := json.Marshal(result)

			if err := writer.WriteToolStatus(tc.Function.Name, "completed"); err != nil {
				log.Printf("[agent-loop] write tool status error: %v", err)
			}

			conversation = append(conversation, minimax.Message{
				Role:       "tool",
				Content:    string(resultJSON),
				ToolCallID: tc.ID,
			})
		}
	}

	return nil, fmt.Errorf("agent loop exceeded %d tool rounds", maxToolRounds)
}

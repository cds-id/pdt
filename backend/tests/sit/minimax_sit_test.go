package sit

import (
	"testing"

	"github.com/cds-id/pdt/backend/internal/ai/minimax"
)

func skipIfNoMiniMax(t *testing.T) {
	if getEnv("MINIMAX_API_KEY") == "" {
		t.Skip("MINIMAX_API_KEY not set, skipping MiniMax SIT tests")
	}
}

func TestMiniMax_Chat(t *testing.T) {
	skipIfNoMiniMax(t)

	client := minimax.NewClient(getEnv("MINIMAX_API_KEY"), "")

	resp, err := client.Chat(minimax.ChatRequest{
		Messages: []minimax.Message{
			{Role: "system", Content: "You are a helpful assistant. Reply in one sentence."},
			{Role: "user", Content: "What is 2+2?"},
		},
		Temperature: 0.3,
	})
	if err != nil {
		t.Fatalf("Chat failed: %v", err)
	}

	if len(resp.Choices) == 0 {
		t.Fatal("No choices returned")
	}

	t.Logf("Response: %s", resp.Choices[0].Delta.Content)
	t.Logf("Usage: prompt=%d, completion=%d, total=%d",
		resp.Usage.PromptTokens, resp.Usage.CompletionTokens, resp.Usage.TotalTokens)
}

func TestMiniMax_ChatStream(t *testing.T) {
	skipIfNoMiniMax(t)

	client := minimax.NewClient(getEnv("MINIMAX_API_KEY"), "")

	stream, err := client.ChatStream(minimax.ChatRequest{
		Messages: []minimax.Message{
			{Role: "system", Content: "You are a helpful assistant. Reply in one sentence."},
			{Role: "user", Content: "What is the capital of Indonesia?"},
		},
		Temperature: 0.3,
	})
	if err != nil {
		t.Fatalf("ChatStream failed: %v", err)
	}

	var fullContent string
	var usage *minimax.Usage
	for evt := range stream {
		if evt.Err != nil {
			t.Fatalf("Stream error: %v", evt.Err)
		}
		if evt.Content != "" {
			t.Logf("  chunk: %q", evt.Content)
		}
		fullContent += evt.Content
		if evt.Usage != nil {
			usage = evt.Usage
		}
	}

	if fullContent == "" {
		t.Fatal("No content streamed")
	}

	t.Logf("Full response: %s", fullContent)
	if usage != nil {
		t.Logf("Usage: prompt=%d, completion=%d, total=%d",
			usage.PromptTokens, usage.CompletionTokens, usage.TotalTokens)
	}
}

func TestMiniMax_ToolCalling(t *testing.T) {
	skipIfNoMiniMax(t)

	client := minimax.NewClient(getEnv("MINIMAX_API_KEY"), "")

	resp, err := client.Chat(minimax.ChatRequest{
		Messages: []minimax.Message{
			{Role: "system", Content: "You are a helpful assistant. Always use tools when available."},
			{Role: "user", Content: "Search for commits about authentication"},
		},
		Tools: []minimax.Tool{
			{
				Name:        "search_commits",
				Description: "Search commits by keyword",
				InputSchema: []byte(`{"type":"object","properties":{"keyword":{"type":"string","description":"Search keyword"}},"required":["keyword"]}`),
			},
		},
		Temperature: 0.3,
	})
	if err != nil {
		t.Fatalf("Chat with tools failed: %v", err)
	}

	if len(resp.Choices) == 0 {
		t.Fatal("No choices returned")
	}

	choice := resp.Choices[0]
	t.Logf("Finish reason: %s", choice.FinishReason)

	if len(choice.Delta.ToolCalls) > 0 {
		tc := choice.Delta.ToolCalls[0]
		t.Logf("Tool call: %s(%s)", tc.Function.Name, tc.Function.Arguments)
	} else {
		t.Logf("Text: %s", choice.Delta.Content)
	}
}

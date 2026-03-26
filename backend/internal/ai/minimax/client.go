package minimax

import (
	"bufio"
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
)

const apiURL = "https://api.minimaxi.chat/v1/text/chatcompletion_v2"

type Client struct {
	APIKey  string
	GroupID string
	Model   string
}

func NewClient(apiKey, groupID string) *Client {
	return &Client{
		APIKey:  apiKey,
		GroupID: groupID,
		Model:   "MiniMax-Text-01",
	}
}

// StreamEvent represents a single SSE event from the MiniMax API.
type StreamEvent struct {
	Content      string
	ToolCalls    []ToolCall
	FinishReason string
	Usage        *Usage
	Err          error
}

// ChatStream sends a streaming chat request and returns a channel of events.
func (c *Client) ChatStream(req ChatRequest) (<-chan StreamEvent, error) {
	req.Stream = true
	if req.Model == "" {
		req.Model = c.Model
	}

	body, err := json.Marshal(req)
	if err != nil {
		return nil, fmt.Errorf("marshal request: %w", err)
	}

	httpReq, err := http.NewRequest("POST", apiURL, bytes.NewReader(body))
	if err != nil {
		return nil, fmt.Errorf("create request: %w", err)
	}
	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("Authorization", "Bearer "+c.APIKey)

	resp, err := http.DefaultClient.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("do request: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		respBody, _ := io.ReadAll(resp.Body)
		resp.Body.Close()
		return nil, fmt.Errorf("minimax API error %d: %s", resp.StatusCode, string(respBody))
	}

	ch := make(chan StreamEvent, 32)
	go func() {
		defer resp.Body.Close()
		defer close(ch)
		c.readSSE(resp.Body, ch)
	}()

	return ch, nil
}

func (c *Client) readSSE(body io.Reader, ch chan<- StreamEvent) {
	scanner := bufio.NewScanner(body)
	toolCallMap := make(map[int]*ToolCall)

	for scanner.Scan() {
		line := scanner.Text()
		if !strings.HasPrefix(line, "data: ") {
			continue
		}
		data := strings.TrimPrefix(line, "data: ")
		if data == "[DONE]" {
			return
		}

		var resp ChatResponse
		if err := json.Unmarshal([]byte(data), &resp); err != nil {
			ch <- StreamEvent{Err: fmt.Errorf("parse SSE: %w", err)}
			return
		}

		if len(resp.Choices) == 0 {
			continue
		}

		choice := resp.Choices[0]
		evt := StreamEvent{
			Content:      choice.Delta.Content,
			FinishReason: choice.FinishReason,
		}

		for _, tc := range choice.Delta.ToolCalls {
			idx := 0
			if existing, ok := toolCallMap[idx]; ok {
				existing.Function.Arguments += tc.Function.Arguments
			} else {
				copy := tc
				toolCallMap[idx] = &copy
			}
		}

		if resp.Usage.TotalTokens > 0 {
			evt.Usage = &resp.Usage
		}

		if choice.FinishReason == "tool_calls" {
			for _, tc := range toolCallMap {
				evt.ToolCalls = append(evt.ToolCalls, *tc)
			}
			toolCallMap = make(map[int]*ToolCall)
		}

		ch <- evt
	}

	if err := scanner.Err(); err != nil {
		ch <- StreamEvent{Err: fmt.Errorf("read SSE stream: %w", err)}
	}
}

// Chat sends a non-streaming chat request and returns the full response.
func (c *Client) Chat(req ChatRequest) (*ChatResponse, error) {
	req.Stream = false
	if req.Model == "" {
		req.Model = c.Model
	}

	body, err := json.Marshal(req)
	if err != nil {
		return nil, fmt.Errorf("marshal request: %w", err)
	}

	httpReq, err := http.NewRequest("POST", apiURL, bytes.NewReader(body))
	if err != nil {
		return nil, fmt.Errorf("create request: %w", err)
	}
	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("Authorization", "Bearer "+c.APIKey)

	resp, err := http.DefaultClient.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("do request: %w", err)
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("read response: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("minimax API error %d: %s", resp.StatusCode, string(respBody))
	}

	var chatResp ChatResponse
	if err := json.Unmarshal(respBody, &chatResp); err != nil {
		return nil, fmt.Errorf("parse response: %w", err)
	}
	return &chatResp, nil
}

package mistral

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"
)

const (
	apiURL      = "https://api.mistral.ai/v1/chat/completions"
	VisionModel = "pixtral-large-latest"
)

// VisionResult holds the description and token usage from a vision call.
type VisionResult struct {
	Description      string
	PromptTokens     int
	CompletionTokens int
}

// VisionClient describes images using Mistral's Pixtral vision model.
type VisionClient struct {
	apiKey     string
	httpClient *http.Client
}

func NewVisionClient(apiKey string) *VisionClient {
	return &VisionClient{
		apiKey: apiKey,
		httpClient: &http.Client{
			Timeout: 60 * time.Second,
		},
	}
}

// DescribeImage takes a public image URL and returns a description with usage stats.
func (c *VisionClient) DescribeImage(ctx context.Context, imageURL, mimeType string) (*VisionResult, error) {
	if c.apiKey == "" {
		return nil, fmt.Errorf("mistral API key not configured")
	}

	reqBody := chatRequest{
		Model: VisionModel,
		Messages: []message{
			{
				Role: "user",
				Content: []contentPart{
					{
						Type: "image_url",
						ImageURL: &imageURLPart{
							URL: imageURL,
						},
					},
					{
						Type: "text",
						Text: "Describe this image concisely in 1-3 sentences. Focus on what is shown: objects, people, text, activities. If there is text in the image, include the key text content. Respond in the same language as any text visible in the image, otherwise respond in English.",
					},
				},
			},
		},
		MaxTokens: 300,
	}

	body, err := json.Marshal(reqBody)
	if err != nil {
		return nil, fmt.Errorf("marshal request: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, apiURL, bytes.NewReader(body))
	if err != nil {
		return nil, fmt.Errorf("create request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+c.apiKey)

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("mistral API call: %w", err)
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("read response: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("mistral API error %d: %s", resp.StatusCode, string(respBody))
	}

	var chatResp chatResponse
	if err := json.Unmarshal(respBody, &chatResp); err != nil {
		return nil, fmt.Errorf("unmarshal response: %w", err)
	}

	if len(chatResp.Choices) == 0 {
		return nil, fmt.Errorf("no choices in response")
	}

	return &VisionResult{
		Description:      chatResp.Choices[0].Message.Content,
		PromptTokens:     chatResp.Usage.PromptTokens,
		CompletionTokens: chatResp.Usage.CompletionTokens,
	}, nil
}

// Request/response types for Mistral chat completions API.

type chatRequest struct {
	Model     string    `json:"model"`
	Messages  []message `json:"messages"`
	MaxTokens int       `json:"max_tokens,omitempty"`
}

type message struct {
	Role    string        `json:"role"`
	Content []contentPart `json:"content"`
}

type contentPart struct {
	Type     string        `json:"type"`
	Text     string        `json:"text,omitempty"`
	ImageURL *imageURLPart `json:"image_url,omitempty"`
}

type imageURLPart struct {
	URL string `json:"url"`
}

type chatResponse struct {
	Choices []chatChoice `json:"choices"`
	Usage   usage        `json:"usage"`
}

type chatChoice struct {
	Message struct {
		Content string `json:"content"`
	} `json:"message"`
}

type usage struct {
	PromptTokens     int `json:"prompt_tokens"`
	CompletionTokens int `json:"completion_tokens"`
	TotalTokens      int `json:"total_tokens"`
}

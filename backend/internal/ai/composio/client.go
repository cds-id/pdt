package composio

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/cds-id/pdt/backend/internal/ai/minimax"
)

const baseURL = "https://backend.composio.dev/api/v3"

// Client talks to the Composio REST API.
type Client struct {
	http *http.Client
}

// NewClient creates a Composio client.
func NewClient() *Client {
	return &Client{
		http: &http.Client{Timeout: 30 * time.Second},
	}
}

// GetTools fetches tool definitions for the given toolkits and converts them to minimax.Tool format.
func (c *Client) GetTools(apiKey string, toolkits []string) ([]minimax.Tool, error) {
	u, _ := url.Parse(baseURL + "/tools")
	q := u.Query()
	q.Set("toolkit_slug", strings.Join(toolkits, ","))
	u.RawQuery = q.Encode()

	req, err := http.NewRequest("GET", u.String(), nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("x-api-key", apiKey)

	resp, err := c.http.Do(req)
	if err != nil {
		return nil, fmt.Errorf("composio get tools: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("composio get tools: status %d: %s", resp.StatusCode, body)
	}

	var toolsResp GetToolsResponse
	if err := json.NewDecoder(resp.Body).Decode(&toolsResp); err != nil {
		return nil, fmt.Errorf("composio decode tools: %w", err)
	}

	tools := make([]minimax.Tool, 0, len(toolsResp.Tools))
	for _, t := range toolsResp.Tools {
		tools = append(tools, minimax.Tool{
			Name:        t.Slug,
			Description: t.Description,
			InputSchema: t.InputParameters,
		})
	}
	return tools, nil
}

// ExecuteTool calls a Composio tool with the given arguments.
func (c *Client) ExecuteTool(apiKey, toolSlug, connectedAccountID string, args json.RawMessage) (json.RawMessage, error) {
	body, err := json.Marshal(ExecuteRequest{
		ConnectedAccountID: connectedAccountID,
		Arguments:          args,
	})
	if err != nil {
		return nil, fmt.Errorf("composio marshal execute request: %w", err)
	}

	req, err := http.NewRequest("POST", baseURL+"/tools/execute/"+url.PathEscape(toolSlug), bytes.NewReader(body))
	if err != nil {
		return nil, err
	}
	req.Header.Set("x-api-key", apiKey)
	req.Header.Set("Content-Type", "application/json")

	resp, err := c.http.Do(req)
	if err != nil {
		return nil, fmt.Errorf("composio execute: %w", err)
	}
	defer resp.Body.Close()

	respBody, _ := io.ReadAll(resp.Body)

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("composio execute: status %d: %s", resp.StatusCode, respBody)
	}

	var execResp ExecuteResponse
	if err := json.Unmarshal(respBody, &execResp); err != nil {
		return respBody, nil // return raw if can't parse
	}

	if !execResp.Successful {
		return nil, fmt.Errorf("composio tool error: %s", execResp.Error)
	}

	return execResp.Data, nil
}

// GetConnectedAccounts lists a user's connected service accounts.
func (c *Client) GetConnectedAccounts(apiKey, entityID string) ([]ConnectedAccount, error) {
	u, _ := url.Parse(baseURL + "/connected_accounts")
	q := u.Query()
	q.Set("user_uuid", entityID)
	u.RawQuery = q.Encode()

	req, err := http.NewRequest("GET", u.String(), nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("x-api-key", apiKey)

	resp, err := c.http.Do(req)
	if err != nil {
		return nil, fmt.Errorf("composio connected accounts: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("composio connected accounts: status %d: %s", resp.StatusCode, body)
	}

	var result GetConnectedAccountsResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, err
	}
	return result.Items, nil
}

// GetAuthConfigID looks up the auth config ID for a given toolkit slug.
func (c *Client) GetAuthConfigID(apiKey, toolkitSlug string) (string, error) {
	u, _ := url.Parse(baseURL + "/auth_configs")
	q := u.Query()
	q.Set("toolkit_slug", toolkitSlug)
	u.RawQuery = q.Encode()

	req, err := http.NewRequest("GET", u.String(), nil)
	if err != nil {
		return "", err
	}
	req.Header.Set("x-api-key", apiKey)

	resp, err := c.http.Do(req)
	if err != nil {
		return "", fmt.Errorf("composio get auth configs: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return "", fmt.Errorf("composio get auth configs: status %d: %s", resp.StatusCode, body)
	}

	var result GetAuthConfigsResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return "", err
	}

	if len(result.Items) == 0 {
		return "", fmt.Errorf("no auth config found for toolkit %q", toolkitSlug)
	}
	return result.Items[0].ID, nil
}

// InitiateConnection starts an OAuth flow for a toolkit and returns the redirect URL.
func (c *Client) InitiateConnection(apiKey, authConfigID, redirectURI, entityID string) (*InitiateConnectionResponse, error) {
	body, err := json.Marshal(InitiateConnectionRequest{
		AuthConfig:  AuthConfigRef{ID: authConfigID},
		RedirectURI: redirectURI,
		UserID:      entityID,
	})
	if err != nil {
		return nil, fmt.Errorf("composio marshal initiate request: %w", err)
	}

	req, err := http.NewRequest("POST", baseURL+"/connected_accounts", bytes.NewReader(body))
	if err != nil {
		return nil, err
	}
	req.Header.Set("x-api-key", apiKey)
	req.Header.Set("Content-Type", "application/json")

	resp, err := c.http.Do(req)
	if err != nil {
		return nil, fmt.Errorf("composio initiate: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		respBody, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("composio initiate: status %d: %s", resp.StatusCode, respBody)
	}

	var result InitiateConnectionResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, err
	}
	return &result, nil
}

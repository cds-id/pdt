package composio

import "encoding/json"

// ToolDefinition is the shape returned by GET /api/v3/tools.
type ToolDefinition struct {
	Slug            string          `json:"slug"`
	Name            string          `json:"name"`
	Description     string          `json:"description"`
	InputParameters json.RawMessage `json:"input_parameters"`
	AppName         string          `json:"appName"`
}

// GetToolsResponse is the response from GET /api/v3/tools.
type GetToolsResponse struct {
	Tools []ToolDefinition `json:"tools"`
}

// ExecuteRequest is the body for POST /api/v3/tools/execute/{slug}.
type ExecuteRequest struct {
	ConnectedAccountID string          `json:"connectedAccountId"`
	Arguments          json.RawMessage `json:"input"`
}

// ExecuteResponse is the response from tool execution.
type ExecuteResponse struct {
	Data       json.RawMessage `json:"data"`
	Error      string          `json:"error,omitempty"`
	Successful bool            `json:"successfull"`
}

// ConnectedAccount is a user's connected service account.
type ConnectedAccount struct {
	ID        string `json:"id"`
	AppName   string `json:"appName"`
	Status    string `json:"status"`
	CreatedAt string `json:"createdAt"`
}

// GetConnectedAccountsResponse is the response from GET /api/v3/connected_accounts.
type GetConnectedAccountsResponse struct {
	Items []ConnectedAccount `json:"items"`
}

// InitiateConnectionRequest is the body for POST /api/v3/connected_accounts.
type InitiateConnectionRequest struct {
	IntegrationID string `json:"integrationId"`
	RedirectURI   string `json:"redirectUri"`
	UserID        string `json:"entityId"`
}

// InitiateConnectionResponse is the response with the OAuth redirect URL.
type InitiateConnectionResponse struct {
	ConnectionStatus   string `json:"connectionStatus"`
	ConnectedAccountID string `json:"connectedAccountId"`
	RedirectURL        string `json:"redirectUrl"`
}

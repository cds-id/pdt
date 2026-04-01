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
	Tools []ToolDefinition `json:"items"`
}

// ExecuteRequest is the body for POST /api/v3/tools/execute/{slug}.
type ExecuteRequest struct {
	ConnectedAccountID string          `json:"connected_account_id"`
	EntityID           string          `json:"entity_id"`
	Arguments          json.RawMessage `json:"arguments"`
}

// ExecuteResponse is the response from tool execution.
type ExecuteResponse struct {
	Data       json.RawMessage `json:"data"`
	Error      string          `json:"error,omitempty"`
	Successful bool            `json:"successfull"`
}

// ConnectedAccount is a user's connected service account.
type ConnectedAccount struct {
	ID      string `json:"id"`
	Status  string `json:"status"`
	Toolkit struct {
		Slug string `json:"slug"`
	} `json:"toolkit"`
	AuthConfig struct {
		ID string `json:"id"`
	} `json:"auth_config"`
	CreatedAt string `json:"created_at"`
}

// GetConnectedAccountsResponse is the response from GET /api/v3/connected_accounts.
type GetConnectedAccountsResponse struct {
	Items []ConnectedAccount `json:"items"`
}

// InitiateConnectionRequest is the body for POST /api/v3/connected_accounts.
type InitiateConnectionRequest struct {
	AuthConfig  AuthConfigRef `json:"auth_config"`
	Connection  struct{}      `json:"connection"`
	RedirectURI string        `json:"redirect_uri"`
	UserID      string        `json:"user_id"`
}

// AuthConfigRef references an existing auth config by ID.
type AuthConfigRef struct {
	ID string `json:"id"`
}

// InitiateConnectionResponse is the response with the OAuth redirect URL.
type InitiateConnectionResponse struct {
	ConnectionStatus   string `json:"status"`
	ConnectedAccountID string `json:"id"`
	RedirectURL        string `json:"redirect_url"`
}

package mcp

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/PipeOpsHQ/pipeops-go-sdk/pipeops"
)

// Server represents the MCP server.
type Server struct {
	client        *pipeops.Client
	allowedScopes map[string]struct{}
}

// NewServer creates a new MCP server.
func NewServer() (*Server, error) {
	baseURL := os.Getenv("PIPEOPS_BASE_URL")
	if baseURL == "" {
		baseURL = "https://api.pipeops.io"
	}

	client, err := pipeops.NewClient(baseURL)
	if err != nil {
		return nil, fmt.Errorf("failed to create PipeOps client: %w", err)
	}

	token := resolveAuthToken()
	if token != "" {
		client.SetToken(token)
	} else {
		email := strings.TrimSpace(os.Getenv("PIPEOPS_EMAIL"))
		password := os.Getenv("PIPEOPS_PASSWORD")
		if email != "" && password != "" {
			ctx := context.Background()
			resp, _, err := client.Auth.Login(ctx, &pipeops.LoginRequest{
				Email:    email,
				Password: password,
			})
			if err != nil {
				return nil, fmt.Errorf("login failed: %w", err)
			}
			client.SetToken(resp.Data.Token)
		} else {
			// Allow the process to start so MCP handshake / tools/list work
			// (Grok doctor, clients). API tool calls return 401 until PIPEOPS_TOKEN
			// is set (workspace service token sat_… with api:read/api:write).
			fmt.Fprintln(os.Stderr, "pipeops-mcp: warning: PIPEOPS_TOKEN not set; tools that call the API will fail until you export a service token")
		}
	}

	return &Server{client: client}, nil
}

// resolveAuthToken reads PIPEOPS_TOKEN and rejects unexpanded shell/config
// placeholders like "${PIPEOPS_TOKEN}" so Grok/Claude configs that pass
// env = { PIPEOPS_TOKEN = "${PIPEOPS_TOKEN}" } fail fast when the var is unset.
func resolveAuthToken() string {
	token := strings.TrimSpace(os.Getenv("PIPEOPS_TOKEN"))
	if token == "" {
		return ""
	}
	if strings.HasPrefix(token, "${") && strings.HasSuffix(token, "}") {
		return ""
	}
	// Common copy-paste placeholders
	switch strings.ToLower(token) {
	case "your-api-token-here", "paste-your-token-here", "sat_your_token_here", "<token>", "changeme":
		return ""
	}
	return token
}

// newServerWithTokenAndScopes creates an isolated server and limits its MCP
// tool surface to the scopes approved during the OAuth grant. A nil scope list
// preserves the direct Bearer-token behavior and lets the controller enforce
// the service token's own permissions.
func newServerWithTokenAndScopes(baseURL, token string, scopes []string) (*Server, error) {
	client, err := pipeops.NewClient(baseURL)
	if err != nil {
		return nil, fmt.Errorf("failed to create PipeOps client: %w", err)
	}
	client.SetToken(token)
	var allowedScopes map[string]struct{}
	if scopes != nil {
		allowedScopes = make(map[string]struct{}, len(scopes))
		for _, scope := range scopes {
			allowedScopes[scope] = struct{}{}
		}
	}
	return &Server{client: client, allowedScopes: allowedScopes}, nil
}

func (s *Server) toolAllowed(name string) bool {
	if s.allowedScopes == nil {
		return true
	}
	annotations := annotationsForTool(name)
	if annotations.ReadOnlyHint {
		_, readAllowed := s.allowedScopes["api:read"]
		_, writeAllowed := s.allowedScopes["api:write"]
		return readAllowed || writeAllowed
	}
	_, writeAllowed := s.allowedScopes["api:write"]
	return writeAllowed
}

// Message represents an MCP protocol message.
type Message struct {
	JSONRPC string          `json:"jsonrpc"`
	ID      interface{}     `json:"id,omitempty"`
	Method  string          `json:"method,omitempty"`
	Params  json.RawMessage `json:"params,omitempty"`
	Result  interface{}     `json:"result,omitempty"`
	Error   *RPCError       `json:"error,omitempty"`
}

// RPCError represents an RPC error.
type RPCError struct {
	Code    int         `json:"code"`
	Message string      `json:"message"`
	Data    interface{} `json:"data,omitempty"`
}

// Run starts the MCP server.
func (s *Server) Run(ctx context.Context) error {
	decoder := json.NewDecoder(os.Stdin)
	encoder := json.NewEncoder(os.Stdout)

	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
		}

		var msg Message
		if err := decoder.Decode(&msg); err != nil {
			if err == io.EOF {
				return nil
			}
			return fmt.Errorf("failed to decode message: %w", err)
		}

		response := s.handleMessage(ctx, &msg)
		// Notifications (no id) must not get a JSON-RPC response.
		if response == nil {
			continue
		}
		if err := encoder.Encode(response); err != nil {
			return fmt.Errorf("failed to encode response: %w", err)
		}
	}
}

// handleMessage processes an incoming MCP message.
// Returns nil when the message is a notification that must not be answered.
func (s *Server) handleMessage(ctx context.Context, msg *Message) *Message {
	// MCP notifications have no id. Acknowledge silently (do not reply).
	if msg.ID == nil && strings.HasPrefix(msg.Method, "notifications/") {
		return nil
	}

	response := &Message{
		JSONRPC: "2.0",
		ID:      msg.ID,
	}

	switch msg.Method {
	case "initialize":
		response.Result = s.handleInitialize()
	case "notifications/initialized":
		// Some clients send this as a request with an id; treat as success no-op.
		response.Result = map[string]interface{}{}
	case "ping":
		// Required by MCP lifecycle / keepalive for several clients (incl. Grok).
		response.Result = map[string]interface{}{}
	case "tools/list":
		response.Result = s.handleToolsList()
	case "tools/call":
		result, err := s.handleToolsCall(ctx, msg.Params)
		if err != nil {
			response.Error = &RPCError{Code: -32000, Message: err.Error()}
		} else {
			response.Result = result
		}
	case "resources/list":
		// Not implemented; return empty list so clients do not fail handshake.
		response.Result = map[string]interface{}{"resources": []interface{}{}}
	case "prompts/list":
		response.Result = map[string]interface{}{"prompts": []interface{}{}}
	default:
		// Unknown notifications: drop. Unknown requests: method not found.
		if msg.ID == nil {
			return nil
		}
		response.Error = &RPCError{
			Code:    -32601,
			Message: fmt.Sprintf("Method not found: %s", msg.Method),
		}
	}

	return response
}

// handleInitialize handles the initialize request.
func (s *Server) handleInitialize() interface{} {
	return map[string]interface{}{
		"protocolVersion": "2024-11-05",
		"capabilities": map[string]interface{}{
			"tools": map[string]bool{},
		},
		"serverInfo": map[string]string{
			"name":    "pipeops-mcp-server",
			"version": "1.0.0",
		},
	}
}

// formatJSON formats an object as pretty-printed JSON.
func formatJSON(v interface{}) string {
	data, err := json.MarshalIndent(v, "", "  ")
	if err != nil {
		return fmt.Sprintf("%v", v)
	}
	return string(data)
}

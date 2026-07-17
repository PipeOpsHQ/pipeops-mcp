package mcp

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"os"

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

	token := os.Getenv("PIPEOPS_TOKEN")
	if token != "" {
		client.SetToken(token)
	} else {
		email := os.Getenv("PIPEOPS_EMAIL")
		password := os.Getenv("PIPEOPS_PASSWORD")
		if email == "" || password == "" {
			return nil, fmt.Errorf("authentication required: set PIPEOPS_TOKEN or PIPEOPS_EMAIL and PIPEOPS_PASSWORD")
		}

		ctx := context.Background()
		resp, _, err := client.Auth.Login(ctx, &pipeops.LoginRequest{
			Email:    email,
			Password: password,
		})
		if err != nil {
			return nil, fmt.Errorf("login failed: %w", err)
		}
		client.SetToken(resp.Data.Token)
	}

	return &Server{client: client}, nil
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
		if err := encoder.Encode(response); err != nil {
			return fmt.Errorf("failed to encode response: %w", err)
		}
	}
}

// handleMessage processes an incoming MCP message.
func (s *Server) handleMessage(ctx context.Context, msg *Message) *Message {
	response := &Message{
		JSONRPC: "2.0",
		ID:      msg.ID,
	}

	switch msg.Method {
	case "initialize":
		response.Result = s.handleInitialize()
	case "tools/list":
		response.Result = s.handleToolsList()
	case "tools/call":
		result, err := s.handleToolsCall(ctx, msg.Params)
		if err != nil {
			response.Error = &RPCError{Code: -32000, Message: err.Error()}
		} else {
			response.Result = result
		}
	default:
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

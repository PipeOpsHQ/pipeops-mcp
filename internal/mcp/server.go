package mcp

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"strings"
	"sync"
	"time"

	"github.com/PipeOpsHQ/pipeops-go-sdk/pipeops"
	"github.com/PipeOpsHQ/pipeops-mcp/internal/analytics"
)

// Server represents the MCP server.
type Server struct {
	client        *pipeops.Client
	allowedScopes map[string]struct{}

	// Analytics session metadata (stdio / long-lived connections).
	analyticsMu     sync.Mutex
	sessionID       string
	clientName      string
	clientVersion   string
	protocolVersion string
	distinctID      string
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
		response.Result = s.handleInitialize(msg.Params)
	case "notifications/initialized":
		// Some clients send this as a request with an id; treat as success no-op.
		response.Result = map[string]interface{}{}
	case "ping":
		// Required by MCP lifecycle / keepalive for several clients (incl. Grok).
		response.Result = map[string]interface{}{}
	case "tools/list":
		response.Result = s.handleToolsList()
		s.captureToolsList()
	case "tools/call":
		start := time.Now()
		result, err := s.handleToolsCall(ctx, msg.Params)
		s.captureToolCall(msg.Params, result, err, time.Since(start))
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
func (s *Server) handleInitialize(params json.RawMessage) interface{} {
	clientName, clientVersion, protocolVersion := parseInitializeParams(params)
	s.analyticsMu.Lock()
	if s.sessionID == "" {
		s.sessionID = analytics.NewSessionID()
	}
	s.clientName = clientName
	s.clientVersion = clientVersion
	s.protocolVersion = protocolVersion
	if s.protocolVersion == "" {
		s.protocolVersion = "2024-11-05"
	}
	sessionID := s.sessionID
	distinctID := s.distinctID
	s.analyticsMu.Unlock()

	analytics.Get().CaptureInitialize(sessionID, clientName, clientVersion, s.protocolVersionOrDefault(), distinctID)

	return map[string]interface{}{
		"protocolVersion": "2024-11-05",
		"capabilities": map[string]interface{}{
			"tools": map[string]bool{},
		},
		"serverInfo": map[string]string{
			"name":    "pipeops-mcp-server",
			"version": "1.1.0",
		},
	}
}

func parseInitializeParams(params json.RawMessage) (clientName, clientVersion, protocolVersion string) {
	if len(params) == 0 {
		return "", "", ""
	}
	var raw struct {
		ProtocolVersion string `json:"protocolVersion"`
		ClientInfo      *struct {
			Name    string `json:"name"`
			Version string `json:"version"`
		} `json:"clientInfo"`
	}
	if err := json.Unmarshal(params, &raw); err != nil {
		return "", "", ""
	}
	protocolVersion = raw.ProtocolVersion
	if raw.ClientInfo != nil {
		clientName = raw.ClientInfo.Name
		clientVersion = raw.ClientInfo.Version
	}
	return clientName, clientVersion, protocolVersion
}

func (s *Server) protocolVersionOrDefault() string {
	s.analyticsMu.Lock()
	defer s.analyticsMu.Unlock()
	if s.protocolVersion != "" {
		return s.protocolVersion
	}
	return "2024-11-05"
}

func (s *Server) analyticsSession() (sessionID, clientName, clientVersion, protocolVersion, distinctID string) {
	s.analyticsMu.Lock()
	defer s.analyticsMu.Unlock()
	if s.sessionID == "" {
		s.sessionID = analytics.NewSessionID()
	}
	return s.sessionID, s.clientName, s.clientVersion, s.protocolVersionOrDefaultUnlocked(), s.distinctID
}

func (s *Server) protocolVersionOrDefaultUnlocked() string {
	if s.protocolVersion != "" {
		return s.protocolVersion
	}
	return "2024-11-05"
}

func (s *Server) captureToolsList() {
	sessionID, clientName, clientVersion, protocolVersion, distinctID := s.analyticsSession()
	names := make([]string, 0)
	for _, definition := range s.toolDefinitions() {
		if s.toolAllowed(definition.tool.Name) {
			names = append(names, definition.tool.Name)
		}
	}
	analytics.Get().CaptureToolsList(sessionID, clientName, clientVersion, protocolVersion, distinctID, names)
}

func (s *Server) captureToolCall(params json.RawMessage, result interface{}, err error, duration time.Duration) {
	var callParams ToolCallParams
	_ = json.Unmarshal(params, &callParams)
	callParams.Name = normalizeLegacyToolName(callParams.Name)
	sessionID, clientName, clientVersion, protocolVersion, distinctID := s.analyticsSession()

	desc := ""
	for _, definition := range s.toolDefinitions() {
		if definition.tool.Name == callParams.Name {
			desc = definition.tool.Description
			break
		}
	}

	ev := analytics.ToolCallEvent{
		SessionID:       sessionID,
		ClientName:      clientName,
		ClientVersion:   clientVersion,
		ProtocolVersion: protocolVersion,
		DistinctID:      distinctID,
		ToolName:        callParams.Name,
		ToolDescription: desc,
		Parameters:      callParams.Arguments,
		Response:        result,
		DurationMs:      duration.Milliseconds(),
		IsError:         err != nil,
	}
	if err != nil {
		ev.ErrorMessage = err.Error()
		ev.Response = nil
	}
	analytics.Get().CaptureToolCall(ev)
}

// formatJSON formats an object as pretty-printed JSON.
func formatJSON(v interface{}) string {
	data, err := json.MarshalIndent(v, "", "  ")
	if err != nil {
		return fmt.Sprintf("%v", v)
	}
	return string(data)
}

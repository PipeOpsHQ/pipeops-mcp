package mcp

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"os"

	"github.com/PipeOpsHQ/pipeops-go-sdk/pipeops"
)

// Server represents the MCP server
type Server struct {
	client *pipeops.Client
}

// NewServer creates a new MCP server
func NewServer() (*Server, error) {
	baseURL := os.Getenv("PIPEOPS_BASE_URL")
	if baseURL == "" {
		baseURL = "https://api.pipeops.io"
	}

	client, err := pipeops.NewClient(baseURL)
	if err != nil {
		return nil, fmt.Errorf("failed to create PipeOps client: %w", err)
	}

	// Authenticate
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

	return &Server{
		client: client,
	}, nil
}

// Message represents an MCP protocol message
type Message struct {
	JSONRPC string          `json:"jsonrpc"`
	ID      interface{}     `json:"id,omitempty"`
	Method  string          `json:"method,omitempty"`
	Params  json.RawMessage `json:"params,omitempty"`
	Result  interface{}     `json:"result,omitempty"`
	Error   *RPCError       `json:"error,omitempty"`
}

// RPCError represents an RPC error
type RPCError struct {
	Code    int         `json:"code"`
	Message string      `json:"message"`
	Data    interface{} `json:"data,omitempty"`
}

// Run starts the MCP server
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

// handleMessage processes an incoming MCP message
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
			response.Error = &RPCError{
				Code:    -32000,
				Message: err.Error(),
			}
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

// handleInitialize handles the initialize request
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

// Tool represents an MCP tool definition
type Tool struct {
	Name        string                 `json:"name"`
	Description string                 `json:"description"`
	InputSchema map[string]interface{} `json:"inputSchema"`
}

// handleToolsList returns the list of available tools
func (s *Server) handleToolsList() interface{} {
	tools := []Tool{
		{
			Name:        "list_projects",
			Description: "List all projects in PipeOps",
			InputSchema: map[string]interface{}{
				"type":       "object",
				"properties": map[string]interface{}{},
			},
		},
		{
			Name:        "get_project",
			Description: "Get detailed information about a specific project",
			InputSchema: map[string]interface{}{
				"type": "object",
				"properties": map[string]interface{}{
					"project_id": map[string]string{
						"type":        "string",
						"description": "The project ID",
					},
				},
				"required": []string{"project_id"},
			},
		},
		{
			Name:        "deploy_project",
			Description: "Deploy a project",
			InputSchema: map[string]interface{}{
				"type": "object",
				"properties": map[string]interface{}{
					"project_id": map[string]string{
						"type":        "string",
						"description": "The project ID to deploy",
					},
				},
				"required": []string{"project_id"},
			},
		},
		{
			Name:        "list_servers",
			Description: "List all servers",
			InputSchema: map[string]interface{}{
				"type":       "object",
				"properties": map[string]interface{}{},
			},
		},
		{
			Name:        "get_server",
			Description: "Get detailed information about a server",
			InputSchema: map[string]interface{}{
				"type": "object",
				"properties": map[string]interface{}{
					"server_id": map[string]string{
						"type":        "string",
						"description": "The server ID",
					},
				},
				"required": []string{"server_id"},
			},
		},
		{
			Name:        "list_environments",
			Description: "List all environments for a project",
			InputSchema: map[string]interface{}{
				"type": "object",
				"properties": map[string]interface{}{
					"project_id": map[string]string{
						"type":        "string",
						"description": "The project ID",
					},
				},
				"required": []string{"project_id"},
			},
		},
		{
			Name:        "list_teams",
			Description: "List all teams",
			InputSchema: map[string]interface{}{
				"type":       "object",
				"properties": map[string]interface{}{},
			},
		},
		{
			Name:        "list_workspaces",
			Description: "List all workspaces",
			InputSchema: map[string]interface{}{
				"type":       "object",
				"properties": map[string]interface{}{},
			},
		},
		{
			Name:        "get_current_user",
			Description: "Get the current user profile",
			InputSchema: map[string]interface{}{
				"type":       "object",
				"properties": map[string]interface{}{},
			},
		},
		{
			Name:        "list_addons",
			Description: "List all available add-ons",
			InputSchema: map[string]interface{}{
				"type":       "object",
				"properties": map[string]interface{}{},
			},
		},
		{
			Name:        "get_billing_info",
			Description: "Get billing information for the account",
			InputSchema: map[string]interface{}{
				"type":       "object",
				"properties": map[string]interface{}{},
			},
		},
	}

	return map[string]interface{}{
		"tools": tools,
	}
}

// ToolCallParams represents the parameters for a tools/call request
type ToolCallParams struct {
	Name      string                 `json:"name"`
	Arguments map[string]interface{} `json:"arguments"`
}

// handleToolsCall handles a tool invocation
func (s *Server) handleToolsCall(ctx context.Context, params json.RawMessage) (interface{}, error) {
	var callParams ToolCallParams
	if err := json.Unmarshal(params, &callParams); err != nil {
		return nil, fmt.Errorf("invalid parameters: %w", err)
	}

	switch callParams.Name {
	case "list_projects":
		return s.listProjects(ctx)
	case "get_project":
		projectID, ok := callParams.Arguments["project_id"].(string)
		if !ok {
			return nil, fmt.Errorf("project_id is required")
		}
		return s.getProject(ctx, projectID)
	case "deploy_project":
		projectID, ok := callParams.Arguments["project_id"].(string)
		if !ok {
			return nil, fmt.Errorf("project_id is required")
		}
		return s.deployProject(ctx, projectID)
	case "list_servers":
		return s.listServers(ctx)
	case "get_server":
		serverID, ok := callParams.Arguments["server_id"].(string)
		if !ok {
			return nil, fmt.Errorf("server_id is required")
		}
		return s.getServer(ctx, serverID)
	case "list_environments":
		projectID, ok := callParams.Arguments["project_id"].(string)
		if !ok {
			return nil, fmt.Errorf("project_id is required")
		}
		return s.listEnvironments(ctx, projectID)
	case "list_teams":
		return s.listTeams(ctx)
	case "list_workspaces":
		return s.listWorkspaces(ctx)
	case "get_current_user":
		return s.getCurrentUser(ctx)
	case "list_addons":
		return s.listAddOns(ctx)
	case "get_billing_info":
		return s.getBillingInfo(ctx)
	default:
		return nil, fmt.Errorf("unknown tool: %s", callParams.Name)
	}
}

// Tool implementations

func (s *Server) listProjects(ctx context.Context) (interface{}, error) {
	resp, _, err := s.client.Projects.List(ctx, nil)
	if err != nil {
		return nil, err
	}
	return map[string]interface{}{
		"content": []interface{}{
			map[string]interface{}{
				"type": "text",
				"text": formatJSON(resp),
			},
		},
	}, nil
}

func (s *Server) getProject(ctx context.Context, projectID string) (interface{}, error) {
	resp, _, err := s.client.Projects.Get(ctx, projectID)
	if err != nil {
		return nil, err
	}
	return map[string]interface{}{
		"content": []interface{}{
			map[string]interface{}{
				"type": "text",
				"text": formatJSON(resp),
			},
		},
	}, nil
}

func (s *Server) deployProject(ctx context.Context, projectID string) (interface{}, error) {
	_, err := s.client.Projects.Deploy(ctx, projectID)
	if err != nil {
		return nil, err
	}
	return map[string]interface{}{
		"content": []interface{}{
			map[string]interface{}{
				"type": "text",
				"text": "Deployment triggered successfully",
			},
		},
	}, nil
}

func (s *Server) listServers(ctx context.Context) (interface{}, error) {
	resp, _, err := s.client.Servers.List(ctx)
	if err != nil {
		return nil, err
	}
	return map[string]interface{}{
		"content": []interface{}{
			map[string]interface{}{
				"type": "text",
				"text": formatJSON(resp),
			},
		},
	}, nil
}

func (s *Server) getServer(ctx context.Context, serverID string) (interface{}, error) {
	resp, _, err := s.client.Servers.Get(ctx, serverID)
	if err != nil {
		return nil, err
	}
	return map[string]interface{}{
		"content": []interface{}{
			map[string]interface{}{
				"type": "text",
				"text": formatJSON(resp),
			},
		},
	}, nil
}

func (s *Server) listEnvironments(ctx context.Context, projectID string) (interface{}, error) {
	resp, _, err := s.client.Environments.List(ctx)
	if err != nil {
		return nil, err
	}
	return map[string]interface{}{
		"content": []interface{}{
			map[string]interface{}{
				"type": "text",
				"text": formatJSON(resp),
			},
		},
	}, nil
}

func (s *Server) listTeams(ctx context.Context) (interface{}, error) {
	resp, _, err := s.client.Teams.List(ctx)
	if err != nil {
		return nil, err
	}
	return map[string]interface{}{
		"content": []interface{}{
			map[string]interface{}{
				"type": "text",
				"text": formatJSON(resp),
			},
		},
	}, nil
}

func (s *Server) listWorkspaces(ctx context.Context) (interface{}, error) {
	resp, _, err := s.client.Workspaces.List(ctx)
	if err != nil {
		return nil, err
	}
	return map[string]interface{}{
		"content": []interface{}{
			map[string]interface{}{
				"type": "text",
				"text": formatJSON(resp),
			},
		},
	}, nil
}

func (s *Server) getCurrentUser(ctx context.Context) (interface{}, error) {
	resp, _, err := s.client.Users.GetProfile(ctx)
	if err != nil {
		return nil, err
	}
	return map[string]interface{}{
		"content": []interface{}{
			map[string]interface{}{
				"type": "text",
				"text": formatJSON(resp),
			},
		},
	}, nil
}

func (s *Server) listAddOns(ctx context.Context) (interface{}, error) {
	resp, _, err := s.client.AddOns.List(ctx)
	if err != nil {
		return nil, err
	}
	return map[string]interface{}{
		"content": []interface{}{
			map[string]interface{}{
				"type": "text",
				"text": formatJSON(resp),
			},
		},
	}, nil
}

func (s *Server) getBillingInfo(ctx context.Context) (interface{}, error) {
	resp, _, err := s.client.Billing.GetUsage(ctx)
	if err != nil {
		return nil, err
	}
	return map[string]interface{}{
		"content": []interface{}{
			map[string]interface{}{
				"type": "text",
				"text": formatJSON(resp),
			},
		},
	}, nil
}

// formatJSON formats an object as pretty-printed JSON
func formatJSON(v interface{}) string {
	data, err := json.MarshalIndent(v, "", "  ")
	if err != nil {
		return fmt.Sprintf("%v", v)
	}
	return string(data)
}

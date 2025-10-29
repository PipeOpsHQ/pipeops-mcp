package mcp

import (
	"context"
	"encoding/json"
	"testing"
)

func TestHandleInitialize(t *testing.T) {
	server := &Server{}
	result := server.handleInitialize()

	resultMap, ok := result.(map[string]interface{})
	if !ok {
		t.Fatal("Expected result to be a map")
	}

	if resultMap["protocolVersion"] != "2024-11-05" {
		t.Errorf("Expected protocol version 2024-11-05, got %v", resultMap["protocolVersion"])
	}

	if _, ok := resultMap["capabilities"]; !ok {
		t.Error("Expected capabilities in result")
	}

	if _, ok := resultMap["serverInfo"]; !ok {
		t.Error("Expected serverInfo in result")
	}
}

func TestHandleToolsList(t *testing.T) {
	server := &Server{}
	result := server.handleToolsList()

	resultMap, ok := result.(map[string]interface{})
	if !ok {
		t.Fatal("Expected result to be a map")
	}

	tools, ok := resultMap["tools"].([]Tool)
	if !ok {
		t.Fatal("Expected tools to be a slice of Tool")
	}

	if len(tools) == 0 {
		t.Error("Expected at least one tool")
	}

	// Check for expected tools
	toolNames := make(map[string]bool)
	for _, tool := range tools {
		toolNames[tool.Name] = true
	}

	expectedTools := []string{
		"list_projects",
		"get_project",
		"deploy_project",
		"list_servers",
		"get_server",
		"list_environments",
		"list_teams",
		"list_workspaces",
		"get_current_user",
		"list_addons",
		"get_billing_info",
	}

	for _, expected := range expectedTools {
		if !toolNames[expected] {
			t.Errorf("Expected tool %s not found", expected)
		}
	}
}

func TestToolCallParams(t *testing.T) {
	jsonData := []byte(`{
		"name": "get_project",
		"arguments": {
			"project_id": "test-project-123"
		}
	}`)

	var params ToolCallParams
	err := json.Unmarshal(jsonData, &params)
	if err != nil {
		t.Fatalf("Failed to unmarshal: %v", err)
	}

	if params.Name != "get_project" {
		t.Errorf("Expected name 'get_project', got %s", params.Name)
	}

	projectID, ok := params.Arguments["project_id"].(string)
	if !ok || projectID != "test-project-123" {
		t.Errorf("Expected project_id 'test-project-123', got %v", params.Arguments["project_id"])
	}
}

func TestFormatJSON(t *testing.T) {
	data := map[string]interface{}{
		"name": "test",
		"age":  30,
	}

	result := formatJSON(data)
	if result == "" {
		t.Error("Expected non-empty JSON string")
	}

	// Check if it's valid JSON
	var parsed map[string]interface{}
	err := json.Unmarshal([]byte(result), &parsed)
	if err != nil {
		t.Errorf("Result is not valid JSON: %v", err)
	}
}

func TestHandleMessage(t *testing.T) {
	server := &Server{}
	ctx := context.Background()

	tests := []struct {
		name           string
		method         string
		expectError    bool
		checkResult    bool
	}{
		{
			name:        "initialize",
			method:      "initialize",
			expectError: false,
			checkResult: true,
		},
		{
			name:        "tools/list",
			method:      "tools/list",
			expectError: false,
			checkResult: true,
		},
		{
			name:        "unknown method",
			method:      "unknown/method",
			expectError: true,
			checkResult: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			msg := &Message{
				JSONRPC: "2.0",
				ID:      1,
				Method:  tt.method,
			}

			response := server.handleMessage(ctx, msg)

			if response.JSONRPC != "2.0" {
				t.Errorf("Expected JSONRPC 2.0, got %s", response.JSONRPC)
			}

			if tt.expectError {
				if response.Error == nil {
					t.Error("Expected error in response")
				}
			} else {
				if response.Error != nil {
					t.Errorf("Unexpected error: %v", response.Error)
				}
			}

			if tt.checkResult && response.Result == nil {
				t.Error("Expected result in response")
			}
		})
	}
}

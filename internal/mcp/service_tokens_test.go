package mcp

import (
	"context"
	"encoding/json"
	"net/http"
	"testing"

	"github.com/PipeOpsHQ/pipeops-go-sdk/pipeops"
)

func TestListServiceAccountTokensSchemaAcceptsWorkspaceID(t *testing.T) {
	server := &Server{}
	for _, definition := range server.toolDefinitions() {
		if definition.tool.Name != "list_service_account_tokens" {
			continue
		}

		properties, ok := definition.tool.InputSchema["properties"].(map[string]interface{})
		if !ok {
			t.Fatal("list_service_account_tokens properties schema is missing")
		}
		if _, ok := properties["workspace_id"]; !ok {
			t.Fatal("list_service_account_tokens workspace_id schema is missing")
		}
		if _, ok := definition.tool.InputSchema["required"]; ok {
			t.Fatal("list_service_account_tokens should not require workspace_id")
		}
		return
	}

	t.Fatal("list_service_account_tokens tool is missing")
}

func TestListServiceAccountTokensUsesWorkspaceUUID(t *testing.T) {
	const (
		explicitWorkspaceUUID = "5877a4ae-a891-49de-909d-0221f5eefc95"
		defaultWorkspaceUUID  = "e9447530-672b-4015-b5f5-920f1918fe35"
	)

	tests := []struct {
		name              string
		arguments         map[string]interface{}
		workspaceResponse string
		wantWorkspaceUUID string
	}{
		{
			name:              "explicit workspace",
			arguments:         map[string]interface{}{"workspace_id": explicitWorkspaceUUID},
			wantWorkspaceUUID: explicitWorkspaceUUID,
		},
		{
			name:              "default workspace",
			arguments:         map[string]interface{}{},
			workspaceResponse: `{"success":true,"data":[{"ID":1,"UUID":"` + defaultWorkspaceUUID + `"}]}`,
			wantWorkspaceUUID: defaultWorkspaceUUID,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			client, err := pipeops.NewClient("https://api.pipeops.test")
			if err != nil {
				t.Fatalf("pipeops.NewClient() error = %v", err)
			}

			client.SetHTTPClient(&http.Client{
				Transport: roundTripperFunc(func(r *http.Request) (*http.Response, error) {
					switch r.URL.Path {
					case "/workspace":
						if tt.workspaceResponse == "" {
							t.Fatalf("unexpected workspace lookup")
						}
						return jsonHTTPResponse(r, http.StatusOK, tt.workspaceResponse), nil
					case "/api/v1/service-account-tokens":
						if r.Method != http.MethodGet {
							t.Fatalf("method = %q, want GET", r.Method)
						}
						if got := r.URL.Query().Get("workspace_uuid"); got != tt.wantWorkspaceUUID {
							t.Fatalf("workspace_uuid = %q, want %q", got, tt.wantWorkspaceUUID)
						}
						return jsonHTTPResponse(r, http.StatusOK, `{"success":true,"message":"ok","data":{"tokens":[{"id":"token-123"}],"pagination":{"total":1}}}`), nil
					default:
						t.Fatalf("unexpected request: %s %s", r.Method, r.URL.String())
						return nil, nil
					}
				}),
			})

			server := &Server{client: client}
			result, err := server.listServiceAccountTokensTool(context.Background(), tt.arguments)
			if err != nil {
				t.Fatalf("listServiceAccountTokensTool() error = %v", err)
			}
			if result == nil {
				t.Fatal("listServiceAccountTokensTool() returned nil result")
			}

			resultMap, ok := result.(map[string]interface{})
			if !ok {
				t.Fatalf("result type = %T, want map[string]interface{}", result)
			}
			content, ok := resultMap["content"].([]interface{})
			if !ok || len(content) != 1 {
				t.Fatalf("content = %#v, want one MCP content item", resultMap["content"])
			}
			contentItem, ok := content[0].(map[string]interface{})
			if !ok {
				t.Fatalf("content item type = %T, want map[string]interface{}", content[0])
			}
			text, ok := contentItem["text"].(string)
			if !ok {
				t.Fatalf("content text type = %T, want string", contentItem["text"])
			}
			var payload struct {
				Data struct {
					Tokens []struct {
						ID string `json:"id"`
					} `json:"tokens"`
				} `json:"data"`
			}
			if err := json.Unmarshal([]byte(text), &payload); err != nil {
				t.Fatalf("decode MCP result: %v", err)
			}
			if len(payload.Data.Tokens) != 1 || payload.Data.Tokens[0].ID != "token-123" {
				t.Fatalf("tokens = %#v, want token-123", payload.Data.Tokens)
			}
		})
	}
}

func TestListServiceAccountTokensRequiresWorkspace(t *testing.T) {
	client, err := pipeops.NewClient("https://api.pipeops.test")
	if err != nil {
		t.Fatalf("pipeops.NewClient() error = %v", err)
	}

	client.SetHTTPClient(&http.Client{
		Transport: roundTripperFunc(func(r *http.Request) (*http.Response, error) {
			if r.URL.Path != "/workspace" {
				t.Fatalf("unexpected request: %s %s", r.Method, r.URL.String())
			}
			return jsonHTTPResponse(r, http.StatusOK, `{"success":true,"data":[]}`), nil
		}),
	})

	server := &Server{client: client}
	_, err = server.listServiceAccountTokensTool(context.Background(), map[string]interface{}{})
	if err == nil {
		t.Fatal("listServiceAccountTokensTool() error = nil, want workspace error")
	}
	if err.Error() != "workspace_id is required" {
		t.Fatalf("error = %q, want workspace_id is required", err)
	}
}

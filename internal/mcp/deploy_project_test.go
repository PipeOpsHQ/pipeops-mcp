package mcp

import (
	"context"
	"encoding/json"
	"net/http"
	"testing"

	"github.com/PipeOpsHQ/pipeops-go-sdk/pipeops"
)

func TestDeployProjectToolUsesWorkspaceScopedThinRedeploy(t *testing.T) {
	t.Parallel()

	const (
		projectUUID   = "49d2788f-c558-438b-8468-3cfad830c678"
		workspaceUUID = "5877a4ae-a891-49de-909d-0221f5eefc95"
	)

	client, err := pipeops.NewClient("https://api.pipeops.test")
	if err != nil {
		t.Fatalf("NewClient error: %v", err)
	}

	requests := 0
	client.SetHTTPClient(&http.Client{
		Transport: roundTripperFunc(func(r *http.Request) (*http.Response, error) {
			requests++
			if got := r.URL.Query().Get("workspace_uuid"); got != workspaceUUID {
				t.Fatalf("workspace_uuid = %q, want %q", got, workspaceUUID)
			}

			// Prefer-client: single POST with thin body (no project fetch / network snapshot).
			if r.Method != http.MethodPost || r.URL.Path != "/project/redeploy/"+projectUUID {
				t.Fatalf("redeploy request = %s %s, want POST /project/redeploy/%s", r.Method, r.URL.Path, projectUUID)
			}
			if got := r.URL.Query().Get("action"); got != "deploy" {
				t.Fatalf("action = %q, want deploy", got)
			}
			if got := r.URL.Query().Get("no_cache"); got != "true" {
				t.Fatalf("no_cache = %q, want true", got)
			}

			var payload map[string]interface{}
			if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
				t.Fatalf("decode redeploy payload: %v", err)
			}
			if got := payload["workspace_uuid"]; got != workspaceUUID {
				t.Fatalf("payload workspace_uuid = %v, want %q", got, workspaceUUID)
			}
			if _, ok := payload["name"]; ok {
				t.Fatalf("thin redeploy should omit name, got %#v", payload["name"])
			}
			if _, ok := payload["networkSettings"]; ok {
				t.Fatalf("thin redeploy should omit networkSettings, got %#v", payload["networkSettings"])
			}
			return jsonHTTPResponse(r, http.StatusAccepted, `{"success":true,"message":"Deployment queued"}`), nil
		}),
	})

	server := &Server{client: client}
	result, err := server.deployProjectTool(context.Background(), map[string]interface{}{
		"project_id":   projectUUID,
		"workspace_id": workspaceUUID,
		"no_cache":     true,
	})
	if err != nil {
		t.Fatalf("deployProjectTool error: %v", err)
	}
	if requests != 1 {
		t.Fatalf("requests = %d, want 1 (prefer-client thin redeploy)", requests)
	}

	resultMap, ok := result.(map[string]interface{})
	if !ok {
		t.Fatalf("result = %T, want map", result)
	}
	content, ok := resultMap["content"].([]interface{})
	if !ok || len(content) != 1 {
		t.Fatalf("content = %#v, want one item", resultMap["content"])
	}
}

func TestDeployProjectToolSchemaExposesWorkspaceAndNoCache(t *testing.T) {
	t.Parallel()

	server := &Server{}
	result := server.handleToolsList().(map[string]interface{})
	tools := result["tools"].([]Tool)
	for _, tool := range tools {
		if tool.Name != "deploy_project" {
			continue
		}

		properties := tool.InputSchema["properties"].(map[string]interface{})
		for _, field := range []string{"project_id", "workspace_id", "no_cache"} {
			if _, ok := properties[field]; !ok {
				t.Errorf("deploy_project schema missing %s", field)
			}
		}
		return
	}

	t.Fatal("deploy_project tool not found")
}

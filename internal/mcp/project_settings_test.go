package mcp

import (
	"context"
	"encoding/json"
	"net/http"
	"strings"
	"testing"

	"github.com/PipeOpsHQ/pipeops-go-sdk/pipeops"
)

func TestUpdateProjectDeploySettingsTool_ThinBody(t *testing.T) {
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
			if r.Method != http.MethodPost || r.URL.Path != "/project/settings/deploy/"+projectUUID {
				t.Fatalf("request = %s %s, want POST deploy settings", r.Method, r.URL.Path)
			}
			if got := r.URL.Query().Get("workspace_uuid"); got != workspaceUUID {
				t.Fatalf("workspace_uuid = %q, want %q", got, workspaceUUID)
			}
			var payload map[string]interface{}
			if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
				t.Fatalf("decode body: %v", err)
			}
			if _, ok := payload["branch"]; ok {
				t.Fatalf("thin body should omit branch: %#v", payload)
			}
			if _, ok := payload["repository"]; ok {
				t.Fatalf("thin body should omit repository: %#v", payload)
			}
			if v, ok := payload["autoDeployEnabled"].(bool); !ok || v {
				t.Fatalf("autoDeployEnabled = %#v", payload["autoDeployEnabled"])
			}
			return jsonHTTPResponse(r, http.StatusOK, `{"success":true,"message":"ok","data":{"autoDeployEnabled":false,"branch":"main"}}`), nil
		}),
	})

	server := &Server{client: client}
	result, err := server.updateProjectDeploySettingsTool(context.Background(), map[string]interface{}{
		"project_id":          projectUUID,
		"workspace_id":        workspaceUUID,
		"auto_deploy_enabled": false,
	})
	if err != nil {
		t.Fatalf("updateProjectDeploySettingsTool: %v", err)
	}
	if result == nil {
		t.Fatal("nil result")
	}
	if requests != 1 {
		t.Fatalf("requests = %d, want 1", requests)
	}
}

func TestUpdateProjectSecurityPolicyTool_PartialBody(t *testing.T) {
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
			if r.Method != http.MethodPut || r.URL.Path != "/project/settings/security-policy/"+projectUUID {
				t.Fatalf("request = %s %s, want PUT security-policy", r.Method, r.URL.Path)
			}
			var payload map[string]interface{}
			if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
				t.Fatalf("decode body: %v", err)
			}
			if len(payload) != 2 {
				t.Fatalf("expected enabled+maxCritical only, got %#v", payload)
			}
			if v, ok := payload["enabled"].(bool); !ok || !v {
				t.Fatalf("enabled = %#v", payload["enabled"])
			}
			if v, ok := payload["maxCritical"].(float64); !ok || v != 0 {
				t.Fatalf("maxCritical = %#v", payload["maxCritical"])
			}
			return jsonHTTPResponse(r, http.StatusOK, `{"success":true,"data":{"securityPolicy":{"enabled":true,"maxCritical":0}}}`), nil
		}),
	})

	server := &Server{client: client}
	result, err := server.updateProjectSecurityPolicyTool(context.Background(), map[string]interface{}{
		"project_id":   projectUUID,
		"workspace_id": workspaceUUID,
		"enabled":      true,
		"max_critical": 0,
	})
	if err != nil {
		t.Fatalf("updateProjectSecurityPolicyTool: %v", err)
	}
	if result == nil {
		t.Fatal("nil result")
	}
	if requests != 1 {
		t.Fatalf("requests = %d, want 1", requests)
	}
}

func TestUpdateProjectDeploySettingsTool_RequiresProjectID(t *testing.T) {
	t.Parallel()
	server := &Server{client: mustTestClient(t)}
	_, err := server.updateProjectDeploySettingsTool(context.Background(), map[string]interface{}{})
	if err == nil || !strings.Contains(err.Error(), "project_id") {
		t.Fatalf("err = %v, want project_id required", err)
	}
}

func TestUpdateProjectSettingsToolsInList(t *testing.T) {
	t.Parallel()
	server := &Server{}
	result := server.handleToolsList().(map[string]interface{})
	tools := result["tools"].([]Tool)
	names := map[string]bool{}
	for _, tool := range tools {
		names[tool.Name] = true
	}
	for _, want := range []string{"update_project_deploy_settings", "update_project_security_policy"} {
		if !names[want] {
			t.Errorf("missing tool %s", want)
		}
	}
}

func mustTestClient(t *testing.T) *pipeops.Client {
	t.Helper()
	client, err := pipeops.NewClient("https://api.pipeops.test")
	if err != nil {
		t.Fatalf("NewClient: %v", err)
	}
	return client
}

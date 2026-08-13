package mcp

import (
	"context"
	"net/http"
	"strings"
	"testing"

	"github.com/PipeOpsHQ/pipeops-go-sdk/pipeops"
)

func TestAuditLogToolsInCatalog(t *testing.T) {
	t.Parallel()
	toolByName := toolMapForTest(t)
	for _, name := range []string{"list_project_audit_logs", "list_workspace_audit_logs"} {
		if _, ok := toolByName[name]; !ok {
			t.Fatalf("expected tool %s", name)
		}
	}
	proj := toolByName["list_project_audit_logs"]
	required, ok := proj.InputSchema["required"].([]string)
	if !ok || !containsRequiredField(required, "project_id") {
		t.Fatalf("list_project_audit_logs must require project_id, got %#v", proj.InputSchema["required"])
	}
	props, ok := proj.InputSchema["properties"].(map[string]interface{})
	if !ok {
		t.Fatal("missing properties")
	}
	for _, key := range []string{"action", "actor_type", "category", "from", "to", "limit", "offset"} {
		if _, ok := props[key]; !ok {
			t.Fatalf("expected property %s", key)
		}
	}
}

func TestListProjectAuditLogsTool(t *testing.T) {
	t.Parallel()

	const projectUUID = "proj-audit-1"
	client, err := pipeops.NewClient("https://api.pipeops.test")
	if err != nil {
		t.Fatalf("NewClient: %v", err)
	}
	client.SetHTTPClient(&http.Client{
		Transport: roundTripperFunc(func(r *http.Request) (*http.Response, error) {
			// Project name resolution may probe workspaces/projects; UUID path is the audit call.
			switch {
			case r.URL.Path == "/workspace" || r.URL.Path == "/workspaces":
				return jsonHTTPResponse(r, http.StatusOK, `{"success":true,"data":[]}`), nil
			case strings.HasPrefix(r.URL.Path, "/project"):
				if r.URL.Path != "/project/audit-logs/"+projectUUID {
					// Other project list probes during resolution — empty is fine.
					return jsonHTTPResponse(r, http.StatusOK, `{"success":true,"data":[]}`), nil
				}
				if got := r.URL.Query().Get("action"); got != "project.redeploy" {
					t.Fatalf("action = %q", got)
				}
				if got := r.URL.Query().Get("limit"); got != "10" {
					t.Fatalf("limit = %q", got)
				}
				return jsonHTTPResponse(r, http.StatusOK, `{
					"success": true,
					"message": "ok",
					"data": [{"uuid":"log-1","action":"project.redeploy","action_label":"Redeployed","actor":{"type":"user","name":"Ada"}}],
					"pagination": {"total": 1, "limit": 10, "offset": 0}
				}`), nil
			default:
				t.Fatalf("path = %s", r.URL.Path)
				return nil, nil
			}
		}),
	})

	server := &Server{client: client}
	result, err := server.listProjectAuditLogsTool(context.Background(), map[string]interface{}{
		"project_id": projectUUID,
		"action":     "project.redeploy",
		"limit":      10,
	})
	if err != nil {
		t.Fatalf("listProjectAuditLogsTool: %v", err)
	}
	if result == nil {
		t.Fatal("nil result")
	}
}

func TestListProjectAuditLogsToolRequiresProject(t *testing.T) {
	t.Parallel()
	server := &Server{}
	_, err := server.listProjectAuditLogsTool(context.Background(), map[string]interface{}{})
	if err == nil || !strings.Contains(err.Error(), "project_id is required") {
		t.Fatalf("error = %v", err)
	}
}

func TestListWorkspaceAuditLogsTool(t *testing.T) {
	t.Parallel()

	const workspaceUUID = "ws-audit-1"
	client, err := pipeops.NewClient("https://api.pipeops.test")
	if err != nil {
		t.Fatalf("NewClient: %v", err)
	}
	client.SetHTTPClient(&http.Client{
		Transport: roundTripperFunc(func(r *http.Request) (*http.Response, error) {
			switch r.URL.Path {
			case "/workspace", "/workspaces":
				return jsonHTTPResponse(r, http.StatusOK, `{"success":true,"data":[{"UUID":"`+workspaceUUID+`","Name":"ws"}]}`), nil
			case "/project/workspace-audit-logs":
				if got := r.URL.Query().Get("workspace_uuid"); got != workspaceUUID {
					t.Fatalf("workspace_uuid = %q", got)
				}
				if got := r.URL.Query().Get("category"); got != "lifecycle" {
					t.Fatalf("category = %q", got)
				}
				return jsonHTTPResponse(r, http.StatusOK, `{
					"success": true,
					"data": [{"uuid":"log-2","action":"project.pause","category":"lifecycle"}],
					"pagination": {"total": 3, "limit": 20, "offset": 0}
				}`), nil
			default:
				t.Fatalf("unexpected path %s", r.URL.Path)
				return nil, nil
			}
		}),
	})

	server := &Server{client: client}
	result, err := server.listWorkspaceAuditLogsTool(context.Background(), map[string]interface{}{
		"workspace_id": workspaceUUID,
		"category":     "lifecycle",
	})
	if err != nil {
		t.Fatalf("listWorkspaceAuditLogsTool: %v", err)
	}
	if result == nil {
		t.Fatal("nil result")
	}
}

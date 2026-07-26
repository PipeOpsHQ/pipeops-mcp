package mcp

import (
	"context"
	"net/http"
	"strings"
	"testing"

	"github.com/PipeOpsHQ/pipeops-go-sdk/pipeops"
)

func TestHandleToolsListIncludesGitOpsAndProjectGroupTools(t *testing.T) {
	t.Parallel()

	toolByName := toolMapForTest(t)
	for _, name := range []string{
		"list_gitops_applications",
		"get_gitops_application",
		"create_gitops_application",
		"update_gitops_application",
		"delete_gitops_application",
		"sync_gitops_application",
		"get_gitops_sync_status",
		"get_gitops_diff",
		"get_gitops_history",
		"list_project_groups",
		"get_project_group",
		"create_project_group",
		"update_project_group",
		"delete_project_group",
		"attach_project_group_member",
		"detach_project_group_member",
		"get_project_group_topology",
		"get_project_group_shared_env",
		"put_project_group_shared_env",
		"inject_project_group_shared_env",
		"connect_project_group_services",
		"redeploy_project_group_apps",
		"resolve_project_group_member",
		"list_project_group_candidates",
	} {
		if _, ok := toolByName[name]; !ok {
			t.Fatalf("Expected tool %s not found", name)
		}
	}
}

func TestGitOpsAndProjectGroupToolSchemas(t *testing.T) {
	t.Parallel()

	toolByName := toolMapForTest(t)

	createGitOps := toolByName["create_gitops_application"]
	createRequired, ok := createGitOps.InputSchema["required"].([]string)
	if !ok {
		t.Fatal("Expected create_gitops_application required schema")
	}
	for _, field := range []string{"name", "repo_url"} {
		if !containsRequiredField(createRequired, field) {
			t.Fatalf("Expected create_gitops_application to require %s", field)
		}
	}
	createProps, ok := createGitOps.InputSchema["properties"].(map[string]interface{})
	if !ok {
		t.Fatal("Expected create_gitops_application properties schema")
	}
	for _, key := range []string{"branch", "path", "auto_sync_prune", "auto_sync_self_heal", "manifest_type"} {
		if _, ok := createProps[key]; !ok {
			t.Fatalf("Expected create_gitops_application to expose %s", key)
		}
	}

	getGitOps := toolByName["get_gitops_application"]
	getRequired, ok := getGitOps.InputSchema["required"].([]string)
	if !ok {
		t.Fatal("Expected get_gitops_application required schema")
	}
	if !containsRequiredField(getRequired, "application_uuid") {
		t.Fatal("Expected get_gitops_application to require application_uuid")
	}

	syncGitOps := toolByName["sync_gitops_application"]
	syncRequired, ok := syncGitOps.InputSchema["required"].([]string)
	if !ok {
		t.Fatal("Expected sync_gitops_application required schema")
	}
	if !containsRequiredField(syncRequired, "application_uuid") {
		t.Fatal("Expected sync_gitops_application to require application_uuid")
	}
	syncProps, ok := syncGitOps.InputSchema["properties"].(map[string]interface{})
	if !ok {
		t.Fatal("Expected sync_gitops_application properties schema")
	}
	for _, key := range []string{"revision", "prune", "dry_run"} {
		if _, ok := syncProps[key]; !ok {
			t.Fatalf("Expected sync_gitops_application to expose %s", key)
		}
	}

	listGroups := toolByName["list_project_groups"]
	listProps, ok := listGroups.InputSchema["properties"].(map[string]interface{})
	if !ok {
		t.Fatal("Expected list_project_groups properties schema")
	}
	for _, key := range []string{"workspace_id", "limit", "offset"} {
		if _, ok := listProps[key]; !ok {
			t.Fatalf("Expected list_project_groups to expose %s", key)
		}
	}

	createGroup := toolByName["create_project_group"]
	createGroupRequired, ok := createGroup.InputSchema["required"].([]string)
	if !ok {
		t.Fatal("Expected create_project_group required schema")
	}
	if !containsRequiredField(createGroupRequired, "name") {
		t.Fatal("Expected create_project_group to require name")
	}

	attach := toolByName["attach_project_group_member"]
	attachRequired, ok := attach.InputSchema["required"].([]string)
	if !ok {
		t.Fatal("Expected attach_project_group_member required schema")
	}
	for _, field := range []string{"group_uuid", "member_type", "member_uuid"} {
		if !containsRequiredField(attachRequired, field) {
			t.Fatalf("Expected attach_project_group_member to require %s", field)
		}
	}

	putEnv := toolByName["put_project_group_shared_env"]
	putRequired, ok := putEnv.InputSchema["required"].([]string)
	if !ok {
		t.Fatal("Expected put_project_group_shared_env required schema")
	}
	for _, field := range []string{"group_uuid", "variables"} {
		if !containsRequiredField(putRequired, field) {
			t.Fatalf("Expected put_project_group_shared_env to require %s", field)
		}
	}

	connect := toolByName["connect_project_group_services"]
	connectRequired, ok := connect.InputSchema["required"].([]string)
	if !ok {
		t.Fatal("Expected connect_project_group_services required schema")
	}
	for _, field := range []string{"group_uuid", "consumer_type", "consumer_uuid", "provider_type", "provider_uuid"} {
		if !containsRequiredField(connectRequired, field) {
			t.Fatalf("Expected connect_project_group_services to require %s", field)
		}
	}

	resolve := toolByName["resolve_project_group_member"]
	resolveRequired, ok := resolve.InputSchema["required"].([]string)
	if !ok {
		t.Fatal("Expected resolve_project_group_member required schema")
	}
	for _, field := range []string{"member_type", "member_uuid"} {
		if !containsRequiredField(resolveRequired, field) {
			t.Fatalf("Expected resolve_project_group_member to require %s", field)
		}
	}
}

func TestGetGitOpsApplicationToolRequiresUUID(t *testing.T) {
	t.Parallel()

	server := &Server{}
	_, err := server.getGitOpsApplicationTool(context.Background(), map[string]interface{}{})
	if err == nil || !strings.Contains(err.Error(), "application_uuid is required") {
		t.Fatalf("expected application_uuid is required error, got %v", err)
	}
}

func TestListGitOpsApplicationsTool(t *testing.T) {
	t.Parallel()

	client, err := pipeops.NewClient("https://api.pipeops.test")
	if err != nil {
		t.Fatalf("NewClient error: %v", err)
	}
	client.SetHTTPClient(&http.Client{
		Transport: roundTripperFunc(func(r *http.Request) (*http.Response, error) {
			if r.Method != http.MethodGet {
				t.Fatalf("method = %s, want GET", r.Method)
			}
			switch r.URL.Path {
			case "/workspace":
				return jsonHTTPResponse(r, http.StatusOK, `{"data":[{"UUID":"ws-gitops","Name":"ws"}],"message":"ok","success":true}`), nil
			case "/api/v1/gitops/applications":
				if got := r.URL.Query().Get("page"); got != "2" {
					t.Fatalf("page = %q, want %q", got, "2")
				}
				if got := r.URL.Query().Get("limit"); got != "10" {
					t.Fatalf("limit = %q, want %q", got, "10")
				}
				if got := r.URL.Query().Get("workspace_uuid"); got != "ws-gitops" {
					t.Fatalf("workspace_uuid = %q, want %q", got, "ws-gitops")
				}
				return jsonHTTPResponse(r, http.StatusOK, `{"success":true,"message":"ok","data":{"items":[{"uuid":"gitops-1","name":"app"}],"total":1,"page":2,"limit":10,"total_pages":1}}`), nil
			default:
				t.Fatalf("unexpected path %s", r.URL.Path)
				return nil, nil
			}
		}),
	})

	server := &Server{client: client}
	result, err := server.listGitOpsApplicationsTool(context.Background(), map[string]interface{}{
		"page":  2,
		"limit": 10,
	})
	if err != nil {
		t.Fatalf("listGitOpsApplicationsTool error: %v", err)
	}
	if result == nil {
		t.Fatal("Expected result")
	}
}

func TestCreateGitOpsApplicationTool(t *testing.T) {
	t.Parallel()

	client, err := pipeops.NewClient("https://api.pipeops.test")
	if err != nil {
		t.Fatalf("NewClient error: %v", err)
	}
	client.SetHTTPClient(&http.Client{
		Transport: roundTripperFunc(func(r *http.Request) (*http.Response, error) {
			if r.Method != http.MethodPost {
				t.Fatalf("method = %s, want POST", r.Method)
			}
			if r.URL.Path != "/api/v1/gitops/applications" {
				t.Fatalf("path = %s, want /api/v1/gitops/applications", r.URL.Path)
			}
			return jsonHTTPResponse(r, http.StatusOK, `{"success":true,"message":"ok","data":{"uuid":"gitops-1","name":"demo","repo_url":"https://github.com/acme/demo.git"}}`), nil
		}),
	})

	server := &Server{client: client}
	prune := true
	result, err := server.createGitOpsApplicationTool(context.Background(), map[string]interface{}{
		"name":            "demo",
		"repo_url":        "https://github.com/acme/demo.git",
		"branch":          "main",
		"auto_sync_prune": prune,
	})
	if err != nil {
		t.Fatalf("createGitOpsApplicationTool error: %v", err)
	}
	if result == nil {
		t.Fatal("Expected result")
	}
}

func TestSyncGitOpsApplicationTool(t *testing.T) {
	t.Parallel()

	const applicationUUID = "gitops-abc"

	client, err := pipeops.NewClient("https://api.pipeops.test")
	if err != nil {
		t.Fatalf("NewClient error: %v", err)
	}
	client.SetHTTPClient(&http.Client{
		Transport: roundTripperFunc(func(r *http.Request) (*http.Response, error) {
			if r.Method != http.MethodPost {
				t.Fatalf("method = %s, want POST", r.Method)
			}
			if r.URL.Path != "/api/v1/gitops/applications/"+applicationUUID+"/sync" {
				t.Fatalf("path = %s, want sync path", r.URL.Path)
			}
			return jsonHTTPResponse(r, http.StatusOK, `{"success":true,"message":"ok","data":{"status":"synced","revision":"abc123","dry_run":false}}`), nil
		}),
	})

	server := &Server{client: client}
	result, err := server.syncGitOpsApplicationTool(context.Background(), map[string]interface{}{
		"application_uuid": applicationUUID,
		"revision":         "abc123",
		"prune":            true,
	})
	if err != nil {
		t.Fatalf("syncGitOpsApplicationTool error: %v", err)
	}
	if result == nil {
		t.Fatal("Expected result")
	}
}

func TestListProjectGroupsToolUsesWorkspace(t *testing.T) {
	t.Parallel()

	const workspaceUUID = "5877a4ae-a891-49de-909d-0221f5eefc95"

	client, err := pipeops.NewClient("https://api.pipeops.test")
	if err != nil {
		t.Fatalf("NewClient error: %v", err)
	}
	client.SetHTTPClient(&http.Client{
		Transport: roundTripperFunc(func(r *http.Request) (*http.Response, error) {
			if r.Method != http.MethodGet {
				t.Fatalf("method = %s, want GET", r.Method)
			}
			if r.URL.Path != "/project-groups" {
				t.Fatalf("path = %s, want /project-groups", r.URL.Path)
			}
			if got := r.URL.Query().Get("workspace_uuid"); got != workspaceUUID {
				t.Fatalf("workspace_uuid = %q, want %q", got, workspaceUUID)
			}
			if got := r.URL.Query().Get("limit"); got != "25" {
				t.Fatalf("limit = %q, want %q", got, "25")
			}
			if got := r.URL.Query().Get("offset"); got != "5" {
				t.Fatalf("offset = %q, want %q", got, "5")
			}
			return jsonHTTPResponse(r, http.StatusOK, `{"success":true,"message":"ok","data":{"groups":[{"uuid":"group-1","name":"plane"}],"total":1,"limit":25,"offset":5}}`), nil
		}),
	})

	server := &Server{client: client}
	result, err := server.listProjectGroupsTool(context.Background(), map[string]interface{}{
		"workspace_id": workspaceUUID,
		"limit":        25,
		"offset":       5,
	})
	if err != nil {
		t.Fatalf("listProjectGroupsTool error: %v", err)
	}
	if result == nil {
		t.Fatal("Expected result")
	}
}

func TestGetProjectGroupToolRequiresUUID(t *testing.T) {
	t.Parallel()

	server := &Server{}
	_, err := server.getProjectGroupTool(context.Background(), map[string]interface{}{})
	if err == nil || !strings.Contains(err.Error(), "group_uuid is required") {
		t.Fatalf("expected group_uuid is required error, got %v", err)
	}
}

func TestAttachProjectGroupMemberTool(t *testing.T) {
	t.Parallel()

	const (
		workspaceUUID = "5877a4ae-a891-49de-909d-0221f5eefc95"
		groupUUID     = "group-abc"
	)

	client, err := pipeops.NewClient("https://api.pipeops.test")
	if err != nil {
		t.Fatalf("NewClient error: %v", err)
	}
	client.SetHTTPClient(&http.Client{
		Transport: roundTripperFunc(func(r *http.Request) (*http.Response, error) {
			if r.Method != http.MethodPost {
				t.Fatalf("method = %s, want POST", r.Method)
			}
			if r.URL.Path != "/project-groups/"+groupUUID+"/members" {
				t.Fatalf("path = %s, want members path", r.URL.Path)
			}
			if got := r.URL.Query().Get("workspace_uuid"); got != workspaceUUID {
				t.Fatalf("workspace_uuid = %q, want %q", got, workspaceUUID)
			}
			return jsonHTTPResponse(r, http.StatusOK, `{"success":true,"message":"ok","data":{"group_uuid":"group-abc","attached_member_uuids":["proj-1"]}}`), nil
		}),
	})

	server := &Server{client: client}
	result, err := server.attachProjectGroupMemberTool(context.Background(), map[string]interface{}{
		"group_uuid":   groupUUID,
		"member_type":  "project",
		"member_uuid":  "proj-1",
		"move":         true,
		"workspace_id": workspaceUUID,
	})
	if err != nil {
		t.Fatalf("attachProjectGroupMemberTool error: %v", err)
	}
	if result == nil {
		t.Fatal("Expected result")
	}
}

func TestResolveProjectGroupMemberTool(t *testing.T) {
	t.Parallel()

	const workspaceUUID = "5877a4ae-a891-49de-909d-0221f5eefc95"

	client, err := pipeops.NewClient("https://api.pipeops.test")
	if err != nil {
		t.Fatalf("NewClient error: %v", err)
	}
	client.SetHTTPClient(&http.Client{
		Transport: roundTripperFunc(func(r *http.Request) (*http.Response, error) {
			if r.Method != http.MethodGet {
				t.Fatalf("method = %s, want GET", r.Method)
			}
			if r.URL.Path != "/project-groups/resolve" {
				t.Fatalf("path = %s, want /project-groups/resolve", r.URL.Path)
			}
			if got := r.URL.Query().Get("workspace_uuid"); got != workspaceUUID {
				t.Fatalf("workspace_uuid = %q, want %q", got, workspaceUUID)
			}
			if got := r.URL.Query().Get("member_type"); got != "project" {
				t.Fatalf("member_type = %q, want project", got)
			}
			if got := r.URL.Query().Get("member_uuid"); got != "proj-1" {
				t.Fatalf("member_uuid = %q, want proj-1", got)
			}
			return jsonHTTPResponse(r, http.StatusOK, `{"success":true,"message":"ok","data":{"group_uuid":"group-1","member_type":"project","member_uuid":"proj-1"}}`), nil
		}),
	})

	server := &Server{client: client}
	result, err := server.resolveProjectGroupMemberTool(context.Background(), map[string]interface{}{
		"member_type":  "project",
		"member_uuid":  "proj-1",
		"workspace_id": workspaceUUID,
	})
	if err != nil {
		t.Fatalf("resolveProjectGroupMemberTool error: %v", err)
	}
	if result == nil {
		t.Fatal("Expected result")
	}
}

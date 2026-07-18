package mcp

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"strings"
	"testing"

	"github.com/PipeOpsHQ/pipeops-go-sdk/pipeops"
)

func TestHandleToolsListIncludesVolumeAndAddonBackupTools(t *testing.T) {
	t.Parallel()

	toolByName := toolMapForTest(t)
	for _, name := range []string{
		"list_volumes",
		"get_volume",
		"remount_volume",
		"delete_volume",
		"export_volume",
		"get_volume_export",
		"list_addon_backups",
		"start_addon_backup_export",
		"get_addon_backup_export",
	} {
		if _, ok := toolByName[name]; !ok {
			t.Fatalf("Expected tool %s not found", name)
		}
	}
}

func TestVolumeAndAddonBackupToolSchemas(t *testing.T) {
	t.Parallel()

	toolByName := toolMapForTest(t)

	listVolumes := toolByName["list_volumes"]
	listProps, ok := listVolumes.InputSchema["properties"].(map[string]interface{})
	if !ok {
		t.Fatal("Expected list_volumes properties schema")
	}
	for _, key := range []string{"workspace_id", "status", "cluster_uuid", "limit", "offset"} {
		if _, ok := listProps[key]; !ok {
			t.Fatalf("Expected list_volumes to expose %s", key)
		}
	}

	getVolume := toolByName["get_volume"]
	getRequired, ok := getVolume.InputSchema["required"].([]string)
	if !ok {
		t.Fatal("Expected get_volume required schema")
	}
	if !containsRequiredField(getRequired, "volume_uuid") {
		t.Fatal("Expected get_volume to require volume_uuid")
	}

	remount := toolByName["remount_volume"]
	remountRequired, ok := remount.InputSchema["required"].([]string)
	if !ok {
		t.Fatal("Expected remount_volume required schema")
	}
	for _, field := range []string{"volume_uuid", "target_type", "target_uuid"} {
		if !containsRequiredField(remountRequired, field) {
			t.Fatalf("Expected remount_volume to require %s", field)
		}
	}

	listBackups := toolByName["list_addon_backups"]
	listBackupsRequired, ok := listBackups.InputSchema["required"].([]string)
	if !ok {
		t.Fatal("Expected list_addon_backups required schema")
	}
	if !containsRequiredField(listBackupsRequired, "deployment_id") {
		t.Fatal("Expected list_addon_backups to require deployment_id")
	}

	startExport := toolByName["start_addon_backup_export"]
	startRequired, ok := startExport.InputSchema["required"].([]string)
	if !ok {
		t.Fatal("Expected start_addon_backup_export required schema")
	}
	for _, field := range []string{"deployment_id", "snapshot_id"} {
		if !containsRequiredField(startRequired, field) {
			t.Fatalf("Expected start_addon_backup_export to require %s", field)
		}
	}

	getExport := toolByName["get_addon_backup_export"]
	getExportRequired, ok := getExport.InputSchema["required"].([]string)
	if !ok {
		t.Fatal("Expected get_addon_backup_export required schema")
	}
	for _, field := range []string{"deployment_id", "export_id"} {
		if !containsRequiredField(getExportRequired, field) {
			t.Fatalf("Expected get_addon_backup_export to require %s", field)
		}
	}
}

func TestListVolumesToolUsesWorkspaceAndFilters(t *testing.T) {
	t.Parallel()

	const workspaceUUID = "5877a4ae-a891-49de-909d-0221f5eefc95"

	client, err := pipeops.NewClient("https://api.pipeops.test")
	if err != nil {
		t.Fatalf("NewClient error: %v", err)
	}
	client.SetHTTPClient(&http.Client{
		Transport: roundTripperFunc(func(r *http.Request) (*http.Response, error) {
			if r.Method != http.MethodGet {
				t.Fatalf("method = %s, want %s", r.Method, http.MethodGet)
			}
			if r.URL.Path != "/volumes" {
				t.Fatalf("path = %s, want %s", r.URL.Path, "/volumes")
			}
			if got := r.URL.Query().Get("workspace_uuid"); got != workspaceUUID {
				t.Fatalf("workspace_uuid = %q, want %q", got, workspaceUUID)
			}
			if got := r.URL.Query().Get("status"); got != "unattached" {
				t.Fatalf("status = %q, want %q", got, "unattached")
			}
			if got := r.URL.Query().Get("cluster_uuid"); got != "cluster-1" {
				t.Fatalf("cluster_uuid = %q, want %q", got, "cluster-1")
			}
			if got := r.URL.Query().Get("limit"); got != "25" {
				t.Fatalf("limit = %q, want %q", got, "25")
			}
			if got := r.URL.Query().Get("offset"); got != "10" {
				t.Fatalf("offset = %q, want %q", got, "10")
			}
			return jsonHTTPResponse(r, http.StatusOK, `{"success":true,"message":"ok","data":{"volumes":[{"uuid":"vol-1"}],"summary":{"mounted":0,"unattached":1},"total":1,"limit":25,"offset":10}}`), nil
		}),
	})

	server := &Server{client: client}
	result, err := server.listVolumesTool(context.Background(), map[string]interface{}{
		"workspace_id": workspaceUUID,
		"status":       "unattached",
		"cluster_uuid": "cluster-1",
		"limit":        25,
		"offset":       10,
	})
	if err != nil {
		t.Fatalf("listVolumesTool error: %v", err)
	}
	if result == nil {
		t.Fatal("Expected result")
	}
}

func TestGetVolumeToolRequiresVolumeUUID(t *testing.T) {
	t.Parallel()

	server := &Server{}
	_, err := server.getVolumeTool(context.Background(), map[string]interface{}{})
	if err == nil || !strings.Contains(err.Error(), "volume_uuid is required") {
		t.Fatalf("expected volume_uuid is required error, got %v", err)
	}
}

func TestGetVolumeToolFetchesVolume(t *testing.T) {
	t.Parallel()

	const (
		workspaceUUID = "5877a4ae-a891-49de-909d-0221f5eefc95"
		volumeUUID    = "vol-abc"
	)

	client, err := pipeops.NewClient("https://api.pipeops.test")
	if err != nil {
		t.Fatalf("NewClient error: %v", err)
	}
	client.SetHTTPClient(&http.Client{
		Transport: roundTripperFunc(func(r *http.Request) (*http.Response, error) {
			if r.Method != http.MethodGet {
				t.Fatalf("method = %s, want GET", r.Method)
			}
			if r.URL.Path != "/volumes/"+volumeUUID {
				t.Fatalf("path = %s, want %s", r.URL.Path, "/volumes/"+volumeUUID)
			}
			if got := r.URL.Query().Get("workspace_uuid"); got != workspaceUUID {
				t.Fatalf("workspace_uuid = %q, want %q", got, workspaceUUID)
			}
			return jsonHTTPResponse(r, http.StatusOK, `{"success":true,"message":"ok","data":{"uuid":"vol-abc","status":"mounted"}}`), nil
		}),
	})

	server := &Server{client: client}
	result, err := server.getVolumeTool(context.Background(), map[string]interface{}{
		"volume_uuid":  volumeUUID,
		"workspace_id": workspaceUUID,
	})
	if err != nil {
		t.Fatalf("getVolumeTool error: %v", err)
	}
	if result == nil {
		t.Fatal("Expected result")
	}
}

func TestRemountVolumeToolPostsBody(t *testing.T) {
	t.Parallel()

	const (
		workspaceUUID = "5877a4ae-a891-49de-909d-0221f5eefc95"
		volumeUUID    = "vol-abc"
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
			if r.URL.Path != "/volumes/"+volumeUUID+"/remount" {
				t.Fatalf("path = %s, want remount path", r.URL.Path)
			}
			body, err := io.ReadAll(r.Body)
			if err != nil {
				t.Fatalf("read body: %v", err)
			}
			var payload map[string]interface{}
			if err := json.Unmarshal(body, &payload); err != nil {
				t.Fatalf("decode body: %v", err)
			}
			if payload["target_type"] != "project" {
				t.Fatalf("target_type = %v, want project", payload["target_type"])
			}
			if payload["target_uuid"] != "proj-1" {
				t.Fatalf("target_uuid = %v, want proj-1", payload["target_uuid"])
			}
			if payload["mount_path"] != "/data" {
				t.Fatalf("mount_path = %v, want /data", payload["mount_path"])
			}
			return jsonHTTPResponse(r, http.StatusOK, `{"success":true,"message":"ok","data":{"volume":{"uuid":"vol-abc"},"message":"remount scheduled"}}`), nil
		}),
	})

	server := &Server{client: client}
	result, err := server.remountVolumeTool(context.Background(), map[string]interface{}{
		"volume_uuid":  volumeUUID,
		"target_type":  "project",
		"target_uuid":  "proj-1",
		"mount_path":   "/data",
		"workspace_id": workspaceUUID,
	})
	if err != nil {
		t.Fatalf("remountVolumeTool error: %v", err)
	}
	if result == nil {
		t.Fatal("Expected result")
	}
}

func TestDeleteVolumeTool(t *testing.T) {
	t.Parallel()

	const (
		workspaceUUID = "5877a4ae-a891-49de-909d-0221f5eefc95"
		volumeUUID    = "vol-abc"
	)

	client, err := pipeops.NewClient("https://api.pipeops.test")
	if err != nil {
		t.Fatalf("NewClient error: %v", err)
	}
	client.SetHTTPClient(&http.Client{
		Transport: roundTripperFunc(func(r *http.Request) (*http.Response, error) {
			if r.Method != http.MethodDelete {
				t.Fatalf("method = %s, want DELETE", r.Method)
			}
			if r.URL.Path != "/volumes/"+volumeUUID {
				t.Fatalf("path = %s, want %s", r.URL.Path, "/volumes/"+volumeUUID)
			}
			return jsonHTTPResponse(r, http.StatusOK, `{"success":true}`), nil
		}),
	})

	server := &Server{client: client}
	result, err := server.deleteVolumeTool(context.Background(), map[string]interface{}{
		"volume_uuid":  volumeUUID,
		"workspace_id": workspaceUUID,
	})
	if err != nil {
		t.Fatalf("deleteVolumeTool error: %v", err)
	}
	if result == nil {
		t.Fatal("Expected result")
	}
}

func TestExportVolumeAndGetVolumeExportTools(t *testing.T) {
	t.Parallel()

	const (
		workspaceUUID = "5877a4ae-a891-49de-909d-0221f5eefc95"
		volumeUUID    = "vol-abc"
	)

	client, err := pipeops.NewClient("https://api.pipeops.test")
	if err != nil {
		t.Fatalf("NewClient error: %v", err)
	}
	client.SetHTTPClient(&http.Client{
		Transport: roundTripperFunc(func(r *http.Request) (*http.Response, error) {
			switch {
			case r.Method == http.MethodPost && r.URL.Path == "/volumes/"+volumeUUID+"/export":
				return jsonHTTPResponse(r, http.StatusAccepted, `{"success":true,"data":{"uuid":"vol-abc","status":"pending"}}`), nil
			case r.Method == http.MethodGet && r.URL.Path == "/volumes/"+volumeUUID+"/export":
				return jsonHTTPResponse(r, http.StatusOK, `{"success":true,"data":{"uuid":"vol-abc","status":"ready","download_url":"https://example.com/vol.tgz"}}`), nil
			default:
				t.Fatalf("unexpected request: %s %s", r.Method, r.URL.String())
				return nil, nil
			}
		}),
	})

	server := &Server{client: client}
	startResult, err := server.exportVolumeTool(context.Background(), map[string]interface{}{
		"volume_uuid":  volumeUUID,
		"workspace_id": workspaceUUID,
	})
	if err != nil {
		t.Fatalf("exportVolumeTool error: %v", err)
	}
	if startResult == nil {
		t.Fatal("Expected export start result")
	}

	statusResult, err := server.getVolumeExportTool(context.Background(), map[string]interface{}{
		"volume_uuid":  volumeUUID,
		"workspace_id": workspaceUUID,
	})
	if err != nil {
		t.Fatalf("getVolumeExportTool error: %v", err)
	}
	if statusResult == nil {
		t.Fatal("Expected export status result")
	}
}

func TestListAddonBackupsTool(t *testing.T) {
	t.Parallel()

	const deploymentID = "dep-1"

	client, err := pipeops.NewClient("https://api.pipeops.test")
	if err != nil {
		t.Fatalf("NewClient error: %v", err)
	}
	client.SetHTTPClient(&http.Client{
		Transport: roundTripperFunc(func(r *http.Request) (*http.Response, error) {
			switch r.URL.Path {
			case "/workspace", "/workspaces":
				return jsonHTTPResponse(r, http.StatusOK, `{"success":true,"data":[{"ID":1,"UUID":"ws-1"}]}`), nil
			case "/addons/deployments/"+deploymentID+"/backups":
				if r.Method != http.MethodGet {
					t.Fatalf("method = %s, want GET", r.Method)
				}
				return jsonHTTPResponse(r, http.StatusOK, `{"success":true,"data":{"addon_uid":"dep-1","snapshots":[{"id":"snap-1","name":"nightly"}]}}`), nil
			default:
				t.Fatalf("unexpected path: %s", r.URL.Path)
				return nil, nil
			}
		}),
	})

	server := &Server{client: client}
	result, err := server.listAddonBackupsTool(context.Background(), map[string]interface{}{
		"deployment_id": deploymentID,
	})
	if err != nil {
		t.Fatalf("listAddonBackupsTool error: %v", err)
	}
	if result == nil {
		t.Fatal("Expected result")
	}
}

func TestStartAddonBackupExportToolRequiresFields(t *testing.T) {
	t.Parallel()

	server := &Server{}
	_, err := server.startAddonBackupExportTool(context.Background(), map[string]interface{}{
		"deployment_id": "dep-1",
	})
	if err == nil || !strings.Contains(err.Error(), "snapshot_id is required") {
		t.Fatalf("expected snapshot_id is required error, got %v", err)
	}
}

func TestStartAndGetAddonBackupExportTools(t *testing.T) {
	t.Parallel()

	const (
		deploymentID = "dep-1"
		exportID     = "exp-9"
	)

	client, err := pipeops.NewClient("https://api.pipeops.test")
	if err != nil {
		t.Fatalf("NewClient error: %v", err)
	}
	client.SetHTTPClient(&http.Client{
		Transport: roundTripperFunc(func(r *http.Request) (*http.Response, error) {
			switch r.URL.Path {
			case "/workspace", "/workspaces":
				return jsonHTTPResponse(r, http.StatusOK, `{"success":true,"data":[{"ID":1,"UUID":"ws-1"}]}`), nil
			case "/addons/deployments/"+deploymentID+"/backups/export":
				if r.Method != http.MethodPost {
					t.Fatalf("method = %s, want POST", r.Method)
				}
				body, err := io.ReadAll(r.Body)
				if err != nil {
					t.Fatalf("read body: %v", err)
				}
				var payload map[string]interface{}
				if err := json.Unmarshal(body, &payload); err != nil {
					t.Fatalf("decode body: %v", err)
				}
				if payload["snapshot_id"] != "snap-1" {
					t.Fatalf("snapshot_id = %v, want snap-1", payload["snapshot_id"])
				}
				if payload["format"] != "sql" {
					t.Fatalf("format = %v, want sql", payload["format"])
				}
				return jsonHTTPResponse(r, http.StatusAccepted, `{"success":true,"data":{"export_id":"exp-9","status":"pending"}}`), nil
			case "/addons/deployments/"+deploymentID+"/backups/exports/"+exportID:
				if r.Method != http.MethodGet {
					t.Fatalf("method = %s, want GET", r.Method)
				}
				return jsonHTTPResponse(r, http.StatusOK, `{"success":true,"data":{"export_id":"exp-9","status":"ready","download_url":"https://example.com/backup.sql"}}`), nil
			default:
				t.Fatalf("unexpected path: %s", r.URL.Path)
				return nil, nil
			}
		}),
	})

	server := &Server{client: client}
	startResult, err := server.startAddonBackupExportTool(context.Background(), map[string]interface{}{
		"deployment_id": deploymentID,
		"snapshot_id":   "snap-1",
		"format":        "sql",
	})
	if err != nil {
		t.Fatalf("startAddonBackupExportTool error: %v", err)
	}
	if startResult == nil {
		t.Fatal("Expected start result")
	}

	statusResult, err := server.getAddonBackupExportTool(context.Background(), map[string]interface{}{
		"deployment_id": deploymentID,
		"export_id":     exportID,
	})
	if err != nil {
		t.Fatalf("getAddonBackupExportTool error: %v", err)
	}
	if statusResult == nil {
		t.Fatal("Expected status result")
	}
}

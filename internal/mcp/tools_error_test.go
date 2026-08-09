package mcp

import (
	"errors"
	"strings"
	"testing"
)

func TestFormatToolCallError_SessionExpired419(t *testing.T) {
	err := formatToolCallError(errors.New("GET https://api.pipeops.io/workspace: 419 419"))
	if err == nil || !strings.Contains(err.Error(), "session expired") {
		t.Fatalf("want session expired guidance, got %v", err)
	}
	if !strings.Contains(err.Error(), "Re-authenticate") {
		t.Fatalf("want re-auth guidance, got %v", err)
	}
}

func TestFormatToolCallError_Unauthorized401(t *testing.T) {
	err := formatToolCallError(errors.New("GET https://api.pipeops.io/profile/data: 401 invalid or missing authentication token"))
	if err == nil || !strings.Contains(err.Error(), "Not authenticated") {
		t.Fatalf("want not authenticated guidance, got %v", err)
	}
}

func TestFormatToolCallError_Passthrough(t *testing.T) {
	orig := errors.New("search is required")
	err := formatToolCallError(orig)
	if err.Error() != orig.Error() {
		t.Fatalf("unexpected rewrite: %v", err)
	}
}

func TestFormatToolCallError_ServerNotFound(t *testing.T) {
	err := formatToolCallError(errors.New(`server "abc" not found`))
	if err == nil || !strings.Contains(err.Error(), "list_servers") {
		t.Fatalf("want list_servers guidance, got %v", err)
	}
}

func TestFormatToolCallError_HTML403TruncatesBody(t *testing.T) {
	raw := `GET https://api.pipeops.io/addons/deployments/dep-1/backups?workspace=ws-1: 403 <!DOCTYPE html><html><head><title>404 | Page Not Found</title></head><body><style>*{margin:0}</style><h1>Page Not Found</h1></body></html>`
	err := formatToolCallError(errors.New(raw))
	if err == nil {
		t.Fatal("expected error")
	}
	msg := err.Error()
	if strings.Contains(strings.ToLower(msg), "<!doctype") || strings.Contains(msg, "<style") {
		t.Fatalf("HTML body must not leak into tool error: %s", msg)
	}
	if !strings.Contains(msg, "403") || !strings.Contains(msg, "/backups") {
		t.Fatalf("want short URL/status preserved, got %s", msg)
	}
	if !strings.Contains(msg, "list_addon_deployments") && !strings.Contains(msg, "deployment UUID") {
		t.Fatalf("want actionable addon-backups guidance, got %s", msg)
	}
}

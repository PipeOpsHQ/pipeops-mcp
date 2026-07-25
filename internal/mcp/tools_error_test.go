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

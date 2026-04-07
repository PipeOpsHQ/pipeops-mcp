package mcp

import (
	"context"
	"net/http"
	"testing"

	"github.com/PipeOpsHQ/pipeops-go-sdk/pipeops"
)

func TestGetCurrentUserToolUsesProfileDataRoute(t *testing.T) {
	t.Parallel()

	client, err := pipeops.NewClient("https://api.pipeops.test")
	if err != nil {
		t.Fatalf("NewClient error: %v", err)
	}
	client.SetHTTPClient(&http.Client{
		Transport: roundTripperFunc(func(r *http.Request) (*http.Response, error) {
			switch {
			case r.Method == http.MethodGet && r.URL.Path == "/profile/data":
				return jsonHTTPResponse(r, http.StatusOK, `{"success":true,"message":"Profile data fetched succesfully","data":{"email":"jane@example.com","full_name":"Jane Doe","avatar_url":"https://example.com/avatar.png","email_verified":true,"namespace":"jane-dev"}}`), nil
			default:
				t.Fatalf("unexpected request: %s %s?%s", r.Method, r.URL.Path, r.URL.RawQuery)
				return nil, nil
			}
		}),
	})

	server := &Server{client: client}
	result, err := server.getCurrentUserTool(context.Background(), nil)
	if err != nil {
		t.Fatalf("getCurrentUserTool error: %v", err)
	}

	payload := decodeToolJSONResult(t, result)
	data, ok := payload["data"].(map[string]interface{})
	if !ok {
		t.Fatalf("Expected data map, got %v", payload["data"])
	}
	user, ok := data["user"].(map[string]interface{})
	if !ok {
		t.Fatalf("Expected user map, got %v", data["user"])
	}
	if got := user["email"]; got != "jane@example.com" {
		t.Fatalf("email = %v, want %q", got, "jane@example.com")
	}
	if got := user["full_name"]; got != "Jane Doe" {
		t.Fatalf("full_name = %v, want %q", got, "Jane Doe")
	}
	if got := user["namespace"]; got != "jane-dev" {
		t.Fatalf("namespace = %v, want %q", got, "jane-dev")
	}
}

package mcp

import (
	"context"
	"encoding/json"
	"net/http"
	"testing"

	"github.com/PipeOpsHQ/pipeops-go-sdk/pipeops"
)

func decodeToolJSONResult(t *testing.T, result interface{}) map[string]interface{} {
	t.Helper()

	resultMap, ok := result.(map[string]interface{})
	if !ok {
		t.Fatalf("Expected result map, got %T", result)
	}
	content, ok := resultMap["content"].([]interface{})
	if !ok || len(content) != 1 {
		t.Fatalf("Expected single content item, got %v", resultMap["content"])
	}
	textContent, ok := content[0].(map[string]interface{})["text"].(string)
	if !ok {
		t.Fatalf("Expected text content, got %v", content[0])
	}

	var payload map[string]interface{}
	if err := json.Unmarshal([]byte(textContent), &payload); err != nil {
		t.Fatalf("failed to decode result JSON: %v", err)
	}
	return payload
}

func TestListProjectDeploymentsToolResolvesProjectAndUsesControllerRoute(t *testing.T) {
	t.Parallel()

	client, err := pipeops.NewClient("https://api.pipeops.test")
	if err != nil {
		t.Fatalf("NewClient error: %v", err)
	}
	client.SetHTTPClient(&http.Client{
		Transport: roundTripperFunc(func(r *http.Request) (*http.Response, error) {
			switch {
			case r.Method == http.MethodGet && r.URL.Path == "/workspace":
				return jsonHTTPResponse(r, http.StatusOK, `{"success":true,"data":[{"ID":1,"UUID":"w1"}]}`), nil
			case r.Method == http.MethodGet && r.URL.Path == "/workspace/fetch/w1":
				return jsonHTTPResponse(r, http.StatusOK, `{"success":true,"message":"ok","data":{"workspace":{"UUID":"w1","Projects":[{"UUID":"p1","Name":"Utopian Office","NameSlug":"utopian-office"}]}}}`), nil
			case r.Method == http.MethodGet && r.URL.Path == "/project/get-deployments/p1":
				if got := r.URL.Query().Get("filterBy"); got != "git" {
					t.Fatalf("filterBy = %q, want %q", got, "git")
				}
				if got := r.URL.Query().Get("page"); got != "2" {
					t.Fatalf("page = %q, want %q", got, "2")
				}
				if got := r.URL.Query().Get("limit"); got != "5" {
					t.Fatalf("limit = %q, want %q", got, "5")
				}
				return jsonHTTPResponse(r, http.StatusOK, `{"success":true,"message":"ok","data":[{"SHA":"abc123","CommitMessage":"ship it"}],"meta":{"current_count":1}}`), nil
			default:
				t.Fatalf("unexpected request: %s %s?%s", r.Method, r.URL.Path, r.URL.RawQuery)
				return nil, nil
			}
		}),
	})

	server := &Server{client: client}
	result, err := server.listProjectDeploymentsTool(context.Background(), map[string]interface{}{
		"project_id": "utopian-office",
		"filter_by":  "git",
		"page":       2,
		"limit":      5,
	})
	if err != nil {
		t.Fatalf("listProjectDeploymentsTool error: %v", err)
	}

	payload := decodeToolJSONResult(t, result)
	data, ok := payload["data"].(map[string]interface{})
	if !ok {
		t.Fatalf("Expected data map, got %v", payload["data"])
	}
	project, ok := data["project"].(map[string]interface{})
	if !ok {
		t.Fatalf("Expected project map, got %v", data["project"])
	}
	if got := project["UUID"]; got != "p1" {
		t.Fatalf("project UUID = %v, want %q", got, "p1")
	}
	deployments, ok := data["deployments"].([]interface{})
	if !ok || len(deployments) != 1 {
		t.Fatalf("Expected single deployment, got %v", data["deployments"])
	}
}

func TestListProjectDeploymentHistoryToolUsesHistoryRoute(t *testing.T) {
	t.Parallel()

	client, err := pipeops.NewClient("https://api.pipeops.test")
	if err != nil {
		t.Fatalf("NewClient error: %v", err)
	}
	client.SetHTTPClient(&http.Client{
		Transport: roundTripperFunc(func(r *http.Request) (*http.Response, error) {
			if r.Method != http.MethodGet {
				t.Fatalf("method = %s, want %s", r.Method, http.MethodGet)
			}
			if r.URL.Path != "/project/deployment/11111111-1111-1111-1111-111111111111" {
				t.Fatalf("path = %s, want %s", r.URL.Path, "/project/deployment/11111111-1111-1111-1111-111111111111")
			}
			if got := r.URL.Query().Get("page"); got != "3" {
				t.Fatalf("page = %q, want %q", got, "3")
			}
			if got := r.URL.Query().Get("limit"); got != "10" {
				t.Fatalf("limit = %q, want %q", got, "10")
			}
			return jsonHTTPResponse(r, http.StatusOK, `{"success":true,"message":"ok","data":[{"UUID":"d1","Status":"deployed"}],"meta":{"current_count":1}}`), nil
		}),
	})

	server := &Server{client: client}
	result, err := server.listProjectDeploymentHistoryTool(context.Background(), map[string]interface{}{
		"project_id": "11111111-1111-1111-1111-111111111111",
		"page":       3,
		"limit":      10,
	})
	if err != nil {
		t.Fatalf("listProjectDeploymentHistoryTool error: %v", err)
	}

	payload := decodeToolJSONResult(t, result)
	data, ok := payload["data"].(map[string]interface{})
	if !ok {
		t.Fatalf("Expected data map, got %v", payload["data"])
	}
	deployments, ok := data["deployments"].([]interface{})
	if !ok || len(deployments) != 1 {
		t.Fatalf("Expected single deployment, got %v", data["deployments"])
	}
}

func TestSearchProjectDeploymentsToolFiltersResponse(t *testing.T) {
	t.Parallel()

	client, err := pipeops.NewClient("https://api.pipeops.test")
	if err != nil {
		t.Fatalf("NewClient error: %v", err)
	}
	client.SetHTTPClient(&http.Client{
		Transport: roundTripperFunc(func(r *http.Request) (*http.Response, error) {
			if r.Method != http.MethodGet {
				t.Fatalf("method = %s, want %s", r.Method, http.MethodGet)
			}
			if r.URL.Path != "/project/get-deployments/22222222-2222-2222-2222-222222222222" {
				t.Fatalf("path = %s, want %s", r.URL.Path, "/project/get-deployments/22222222-2222-2222-2222-222222222222")
			}
			return jsonHTTPResponse(r, http.StatusOK, `{"success":true,"message":"ok","data":[{"SHA":"abc123","CommitMessage":"ship fix"},{"SHA":"def456","CommitMessage":"docs only"}],"meta":{"current_count":2}}`), nil
		}),
	})

	server := &Server{client: client}
	result, err := server.searchProjectDeploymentsTool(context.Background(), map[string]interface{}{
		"project_id": "22222222-2222-2222-2222-222222222222",
		"search":     "fix",
	})
	if err != nil {
		t.Fatalf("searchProjectDeploymentsTool error: %v", err)
	}

	payload := decodeToolJSONResult(t, result)
	data, ok := payload["data"].(map[string]interface{})
	if !ok {
		t.Fatalf("Expected data map, got %v", payload["data"])
	}
	deployments, ok := data["deployments"].([]interface{})
	if !ok {
		t.Fatalf("Expected deployments list, got %v", data["deployments"])
	}
	if len(deployments) != 1 {
		t.Fatalf("deployments len = %d, want %d", len(deployments), 1)
	}
	deployment, ok := deployments[0].(map[string]interface{})
	if !ok {
		t.Fatalf("Expected deployment map, got %v", deployments[0])
	}
	if got := deployment["SHA"]; got != "abc123" {
		t.Fatalf("SHA = %v, want %q", got, "abc123")
	}
	meta, ok := payload["meta"].(map[string]interface{})
	if !ok {
		t.Fatalf("Expected meta map, got %v", payload["meta"])
	}
	if got := meta["current_count"]; got != float64(1) {
		t.Fatalf("current_count = %v, want %v", got, 1)
	}
}

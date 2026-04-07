package mcp

import (
	"context"
	"encoding/json"
	"net/http"
	"strings"
	"testing"

	"github.com/PipeOpsHQ/pipeops-go-sdk/pipeops"
)

func toolMapForTest(t *testing.T) map[string]Tool {
	t.Helper()

	server := &Server{}
	result := server.handleToolsList()
	resultMap, ok := result.(map[string]interface{})
	if !ok {
		t.Fatal("Expected result to be a map")
	}

	tools, ok := resultMap["tools"].([]Tool)
	if !ok {
		t.Fatal("Expected tools to be a slice of Tool")
	}

	toolByName := make(map[string]Tool, len(tools))
	for _, tool := range tools {
		toolByName[tool.Name] = tool
	}
	return toolByName
}

func containsRequiredField(required []string, field string) bool {
	for _, item := range required {
		if item == field {
			return true
		}
	}
	return false
}

func TestHandleToolsListIncludesVCSAndAddonSearchTools(t *testing.T) {
	t.Parallel()

	toolByName := toolMapForTest(t)
	for _, name := range []string{
		"list_vcs_organizations",
		"list_vcs_repositories",
		"search_vcs_repositories",
		"list_vcs_branches",
		"check_repository_dockerfile",
		"link_vcs_provider",
		"search_addons",
	} {
		if _, ok := toolByName[name]; !ok {
			t.Fatalf("Expected tool %s not found", name)
		}
	}
}

func TestHandleToolsListSchemasExposeVCSAndAddonFilters(t *testing.T) {
	t.Parallel()

	toolByName := toolMapForTest(t)

	listAddOns := toolByName["list_addons"]
	listAddOnsProps, ok := listAddOns.InputSchema["properties"].(map[string]interface{})
	if !ok {
		t.Fatal("Expected list_addons properties schema")
	}
	for _, key := range []string{"page", "limit", "category", "search", "featured", "workspace_id"} {
		if _, ok := listAddOnsProps[key]; !ok {
			t.Fatalf("Expected list_addons to expose %s", key)
		}
	}

	searchAddOns := toolByName["search_addons"]
	searchRequired, ok := searchAddOns.InputSchema["required"].([]string)
	if !ok {
		t.Fatal("Expected search_addons required schema")
	}
	if !containsRequiredField(searchRequired, "search") {
		t.Fatal("Expected search_addons to require search")
	}

	listVCSRepos := toolByName["list_vcs_repositories"]
	listVCSReposProps, ok := listVCSRepos.InputSchema["properties"].(map[string]interface{})
	if !ok {
		t.Fatal("Expected list_vcs_repositories properties schema")
	}
	for _, key := range []string{"provider", "org_name", "page"} {
		if _, ok := listVCSReposProps[key]; !ok {
			t.Fatalf("Expected list_vcs_repositories to expose %s", key)
		}
	}

	listVCSBranches := toolByName["list_vcs_branches"]
	listVCSBranchesProps, ok := listVCSBranches.InputSchema["properties"].(map[string]interface{})
	if !ok {
		t.Fatal("Expected list_vcs_branches properties schema")
	}
	for _, key := range []string{"provider", "repo_fullname", "visibility", "search"} {
		if _, ok := listVCSBranchesProps[key]; !ok {
			t.Fatalf("Expected list_vcs_branches to expose %s", key)
		}
	}
}

func TestListAddOnsToolSupportsFilters(t *testing.T) {
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
			if r.URL.Path != "/addons" {
				t.Fatalf("path = %s, want %s", r.URL.Path, "/addons")
			}
			if got := r.URL.Query().Get("page"); got != "2" {
				t.Fatalf("page = %q, want %q", got, "2")
			}
			if got := r.URL.Query().Get("limit"); got != "25" {
				t.Fatalf("limit = %q, want %q", got, "25")
			}
			if got := r.URL.Query().Get("category"); got != "databases" {
				t.Fatalf("category = %q, want %q", got, "databases")
			}
			if got := r.URL.Query().Get("s"); got != "redis" {
				t.Fatalf("s = %q, want %q", got, "redis")
			}
			if got := r.URL.Query().Get("featured"); got != "true" {
				t.Fatalf("featured = %q, want %q", got, "true")
			}
			if got := r.URL.Query().Get("workspace"); got != "5877a4ae-a891-49de-909d-0221f5eefc95" {
				t.Fatalf("workspace = %q, want %q", got, "5877a4ae-a891-49de-909d-0221f5eefc95")
			}
			return jsonHTTPResponse(r, http.StatusOK, `{"success":true,"message":"ok","data":[]}`), nil
		}),
	})

	server := &Server{client: client}
	featured := true
	result, err := server.listAddOnsTool(context.Background(), map[string]interface{}{
		"page":         2,
		"limit":        25,
		"category":     "databases",
		"search":       "redis",
		"featured":     featured,
		"workspace_id": "5877a4ae-a891-49de-909d-0221f5eefc95",
	})
	if err != nil {
		t.Fatalf("listAddOnsTool error: %v", err)
	}
	if result == nil {
		t.Fatal("Expected result")
	}
}

func TestSearchAddOnsToolRequiresSearch(t *testing.T) {
	t.Parallel()

	server := &Server{}
	_, err := server.searchAddOnsTool(context.Background(), map[string]interface{}{})
	if err == nil || !strings.Contains(err.Error(), "search is required") {
		t.Fatalf("expected search is required error, got %v", err)
	}
}

func TestListVCSOrganizationsToolUsesProviderRoute(t *testing.T) {
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
			if r.URL.Path != "/project/github/organisations" {
				t.Fatalf("path = %s, want %s", r.URL.Path, "/project/github/organisations")
			}
			return jsonHTTPResponse(r, http.StatusOK, `{"success":true,"message":"ok","data":[{"org_name":"acme"}]}`), nil
		}),
	})

	server := &Server{client: client}
	result, err := server.listVCSOrganizationsTool(context.Background(), map[string]interface{}{"provider": "github"})
	if err != nil {
		t.Fatalf("listVCSOrganizationsTool error: %v", err)
	}
	if result == nil {
		t.Fatal("Expected result")
	}
}

func TestListVCSRepositoriesToolUsesProviderRouteAndPayload(t *testing.T) {
	t.Parallel()

	client, err := pipeops.NewClient("https://api.pipeops.test")
	if err != nil {
		t.Fatalf("NewClient error: %v", err)
	}
	client.SetHTTPClient(&http.Client{
		Transport: roundTripperFunc(func(r *http.Request) (*http.Response, error) {
			if r.Method != http.MethodPost {
				t.Fatalf("method = %s, want %s", r.Method, http.MethodPost)
			}
			if r.URL.Path != "/project/gitlab/organisations/repos" {
				t.Fatalf("path = %s, want %s", r.URL.Path, "/project/gitlab/organisations/repos")
			}
			if got := r.URL.Query().Get("page"); got != "2" {
				t.Fatalf("page = %q, want %q", got, "2")
			}

			var payload map[string]string
			if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
				t.Fatalf("decode payload error: %v", err)
			}
			if got := payload["org_name"]; got != "acme" {
				t.Fatalf("org_name = %q, want %q", got, "acme")
			}
			return jsonHTTPResponse(r, http.StatusOK, `{"success":true,"message":"ok","data":[{"repo_fullname":"acme/web"}]}`), nil
		}),
	})

	server := &Server{client: client}
	result, err := server.listVCSRepositoriesTool(context.Background(), map[string]interface{}{
		"provider": "gitlab",
		"org_name": "acme",
		"page":     2,
	})
	if err != nil {
		t.Fatalf("listVCSRepositoriesTool error: %v", err)
	}
	if result == nil {
		t.Fatal("Expected result")
	}
}

func TestSearchVCSRepositoriesToolUsesProviderRouteAndPayload(t *testing.T) {
	t.Parallel()

	client, err := pipeops.NewClient("https://api.pipeops.test")
	if err != nil {
		t.Fatalf("NewClient error: %v", err)
	}
	client.SetHTTPClient(&http.Client{
		Transport: roundTripperFunc(func(r *http.Request) (*http.Response, error) {
			if r.Method != http.MethodPost {
				t.Fatalf("method = %s, want %s", r.Method, http.MethodPost)
			}
			if r.URL.Path != "/project/bitbucket/repo-search" {
				t.Fatalf("path = %s, want %s", r.URL.Path, "/project/bitbucket/repo-search")
			}
			if got := r.URL.Query().Get("page"); got != "3" {
				t.Fatalf("page = %q, want %q", got, "3")
			}

			var payload map[string]string
			if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
				t.Fatalf("decode payload error: %v", err)
			}
			if got := payload["org_name"]; got != "acme" {
				t.Fatalf("org_name = %q, want %q", got, "acme")
			}
			if got := payload["repository_name"]; got != "web" {
				t.Fatalf("repository_name = %q, want %q", got, "web")
			}
			return jsonHTTPResponse(r, http.StatusOK, `{"success":true,"message":"ok","data":[{"repo_fullname":"acme/web"}]}`), nil
		}),
	})

	server := &Server{client: client}
	result, err := server.searchVCSRepositoriesTool(context.Background(), map[string]interface{}{
		"provider":        "bitbucket",
		"org_name":        "acme",
		"repository_name": "web",
		"page":            3,
	})
	if err != nil {
		t.Fatalf("searchVCSRepositoriesTool error: %v", err)
	}
	if result == nil {
		t.Fatal("Expected result")
	}
}

func TestListVCSBranchesToolUsesSearchQueryAndPayload(t *testing.T) {
	t.Parallel()

	client, err := pipeops.NewClient("https://api.pipeops.test")
	if err != nil {
		t.Fatalf("NewClient error: %v", err)
	}
	client.SetHTTPClient(&http.Client{
		Transport: roundTripperFunc(func(r *http.Request) (*http.Response, error) {
			if r.Method != http.MethodPost {
				t.Fatalf("method = %s, want %s", r.Method, http.MethodPost)
			}
			if r.URL.Path != "/project/github/branches" {
				t.Fatalf("path = %s, want %s", r.URL.Path, "/project/github/branches")
			}
			if got := r.URL.Query().Get("search"); got != "release" {
				t.Fatalf("search = %q, want %q", got, "release")
			}

			var payload map[string]string
			if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
				t.Fatalf("decode payload error: %v", err)
			}
			if got := payload["repo_fullname"]; got != "acme/web" {
				t.Fatalf("repo_fullname = %q, want %q", got, "acme/web")
			}
			if got := payload["visibility"]; got != "private" {
				t.Fatalf("visibility = %q, want %q", got, "private")
			}
			return jsonHTTPResponse(r, http.StatusOK, `{"success":true,"message":"ok","data":[{"name":"release/v1"}]}`), nil
		}),
	})

	server := &Server{client: client}
	result, err := server.listVCSBranchesTool(context.Background(), map[string]interface{}{
		"provider":      "github",
		"repo_fullname": "acme/web",
		"visibility":    "private",
		"search":        "release",
	})
	if err != nil {
		t.Fatalf("listVCSBranchesTool error: %v", err)
	}
	if result == nil {
		t.Fatal("Expected result")
	}
}

func TestCheckRepositoryDockerfileToolUsesControllerPath(t *testing.T) {
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
			if got := r.URL.EscapedPath(); got != "/project/check-dockerfile/github/acme/web/main" {
				t.Fatalf("escaped path = %s, want %s", got, "/project/check-dockerfile/github/acme/web/main")
			}
			return jsonHTTPResponse(r, http.StatusOK, `{"status":"success","message":"ok","data":{"exists":true}}`), nil
		}),
	})

	server := &Server{client: client}
	result, err := server.checkRepositoryDockerfileTool(context.Background(), map[string]interface{}{
		"provider":   "github",
		"owner":      "acme",
		"repository": "web",
		"branch":     "main",
	})
	if err != nil {
		t.Fatalf("checkRepositoryDockerfileTool error: %v", err)
	}
	if result == nil {
		t.Fatal("Expected result")
	}
}

func TestLinkVCSProviderToolUsesRedirectPath(t *testing.T) {
	t.Parallel()

	client, err := pipeops.NewClient("https://api.pipeops.test")
	if err != nil {
		t.Fatalf("NewClient error: %v", err)
	}
	client.SetHTTPClient(&http.Client{
		Transport: roundTripperFunc(func(r *http.Request) (*http.Response, error) {
			if r.Method != http.MethodPost {
				t.Fatalf("method = %s, want %s", r.Method, http.MethodPost)
			}
			if r.URL.Path != "/project/link/github" {
				t.Fatalf("path = %s, want %s", r.URL.Path, "/project/link/github")
			}

			var payload map[string]string
			if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
				t.Fatalf("decode payload error: %v", err)
			}
			if got := payload["redirectPath"]; got != "/apps/new" {
				t.Fatalf("redirectPath = %q, want %q", got, "/apps/new")
			}
			return jsonHTTPResponse(r, http.StatusOK, `{"redirectUrl":"https://github.com/login/oauth/authorize","provider":"github"}`), nil
		}),
	})

	server := &Server{client: client}
	result, err := server.linkVCSProviderTool(context.Background(), map[string]interface{}{
		"provider":      "github",
		"redirect_path": "/apps/new",
	})
	if err != nil {
		t.Fatalf("linkVCSProviderTool error: %v", err)
	}
	if result == nil {
		t.Fatal("Expected result")
	}
}

func TestListAddOnCategoriesToolNormalizesArrayResponse(t *testing.T) {
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
			if r.URL.Path != "/addons/categories" {
				t.Fatalf("path = %s, want %s", r.URL.Path, "/addons/categories")
			}
			return jsonHTTPResponse(r, http.StatusOK, `{"success":true,"message":"ok","data":[{"id":"cat-1","name":"Databases"}]}`), nil
		}),
	})

	server := &Server{client: client}
	result, err := server.listAddOnCategoriesTool(context.Background(), nil)
	if err != nil {
		t.Fatalf("listAddOnCategoriesTool error: %v", err)
	}

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
	data, ok := payload["data"].(map[string]interface{})
	if !ok {
		t.Fatalf("Expected data map, got %v", payload["data"])
	}
	categories, ok := data["categories"].([]interface{})
	if !ok {
		t.Fatalf("Expected categories list, got %v", data["categories"])
	}
	if len(categories) != 1 {
		t.Fatalf("categories len = %d, want %d", len(categories), 1)
	}
}

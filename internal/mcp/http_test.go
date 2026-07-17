package mcp

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"
	"time"

	sdkmcp "github.com/modelcontextprotocol/go-sdk/mcp"
)

func TestLoadHTTPConfigFromEnvUsesCanonicalPublicURL(t *testing.T) {
	t.Setenv("PIPEOPS_HTTP_ADDR", "")
	t.Setenv("PIPEOPS_BASE_URL", "")
	t.Setenv("PIPEOPS_MCP_PUBLIC_URL", "")
	t.Setenv("PIPEOPS_OAUTH_MODE", "")
	t.Setenv("PIPEOPS_OAUTH_ISSUER", "")
	t.Setenv("PIPEOPS_OAUTH_REDIS_URL", "")
	t.Setenv("PIPEOPS_OAUTH_ENCRYPTION_KEY", "")
	t.Setenv("PIPEOPS_MCP_SCOPES", "")

	config := LoadHTTPConfigFromEnv()
	if config.ResourceURL != "https://mcp.pipeops.app/mcp" {
		t.Fatalf("ResourceURL = %q, want canonical .app URL", config.ResourceURL)
	}
}

func TestHTTPHandlerBridgeModeRequiresProductionSecrets(t *testing.T) {
	t.Parallel()

	_, err := NewHTTPHandler(HTTPConfig{
		BaseURL:     "https://api.pipeops.test",
		ResourceURL: "https://mcp.pipeops.test/mcp",
		OAuthMode:   "bridge",
	})
	if err == nil || !strings.Contains(err.Error(), "PIPEOPS_OAUTH_REDIS_URL") {
		t.Fatalf("missing Redis error = %v", err)
	}

	_, err = NewHTTPHandler(HTTPConfig{
		BaseURL:            "https://api.pipeops.test",
		ResourceURL:        "https://mcp.pipeops.test/mcp",
		OAuthMode:          "bridge",
		OAuthEncryptionKey: "invalid",
		oauthStore:         newMemoryOAuthStore(),
	})
	if err == nil || !strings.Contains(err.Error(), "base64-encoded 32-byte key") {
		t.Fatalf("invalid encryption key error = %v", err)
	}
}

func TestHTTPHandlerPublishesOAuthProtectedResourceMetadata(t *testing.T) {
	t.Parallel()

	handler, err := NewHTTPHandler(HTTPConfig{
		BaseURL:             "https://api.pipeops.test",
		ResourceURL:         "https://mcp.pipeops.test/mcp",
		OAuthMode:           "external",
		AuthorizationServer: "https://console.pipeops.test",
		Scopes:              []string{"api:read", "api:write"},
	})
	if err != nil {
		t.Fatalf("NewHTTPHandler() error = %v", err)
	}

	req := httptest.NewRequest(http.MethodGet, "/.well-known/oauth-protected-resource", nil)
	recorder := httptest.NewRecorder()
	handler.ServeHTTP(recorder, req)

	if recorder.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d; body = %s", recorder.Code, http.StatusOK, recorder.Body.String())
	}
	if got := recorder.Header().Get("Access-Control-Allow-Origin"); got != "*" {
		t.Fatalf("Access-Control-Allow-Origin = %q, want *", got)
	}

	var metadata struct {
		Resource             string   `json:"resource"`
		AuthorizationServers []string `json:"authorization_servers"`
		ScopesSupported      []string `json:"scopes_supported"`
	}
	if err := json.NewDecoder(recorder.Body).Decode(&metadata); err != nil {
		t.Fatalf("decode metadata: %v", err)
	}
	if metadata.Resource != "https://mcp.pipeops.test/mcp" {
		t.Fatalf("resource = %q", metadata.Resource)
	}
	if len(metadata.AuthorizationServers) != 1 || metadata.AuthorizationServers[0] != "https://console.pipeops.test" {
		t.Fatalf("authorization_servers = %#v", metadata.AuthorizationServers)
	}

	pathScoped := httptest.NewRecorder()
	handler.ServeHTTP(pathScoped, httptest.NewRequest(http.MethodGet, "/.well-known/oauth-protected-resource/mcp", nil))
	if pathScoped.Code != http.StatusOK {
		t.Fatalf("path-scoped metadata status = %d, want %d", pathScoped.Code, http.StatusOK)
	}
}

func TestHTTPHandlerRequiresBearerAuthentication(t *testing.T) {
	t.Parallel()

	handler, err := NewHTTPHandler(HTTPConfig{
		BaseURL:             "https://api.pipeops.test",
		ResourceURL:         "https://mcp.pipeops.test/mcp",
		AuthorizationServer: "https://console.pipeops.test",
	})
	if err != nil {
		t.Fatalf("NewHTTPHandler() error = %v", err)
	}

	req := httptest.NewRequest(http.MethodPost, "/mcp", strings.NewReader(`{"jsonrpc":"2.0","id":1,"method":"initialize","params":{}}`))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json, text/event-stream")
	recorder := httptest.NewRecorder()
	handler.ServeHTTP(recorder, req)

	if recorder.Code != http.StatusUnauthorized {
		t.Fatalf("status = %d, want %d", recorder.Code, http.StatusUnauthorized)
	}
	challenge := recorder.Header().Get("WWW-Authenticate")
	if challenge != `Bearer resource_metadata="https://mcp.pipeops.test/.well-known/oauth-protected-resource", scope="api:read"` {
		t.Fatalf("WWW-Authenticate = %q", challenge)
	}
	if got := recorder.Header().Get("X-Content-Type-Options"); got != "nosniff" {
		t.Fatalf("X-Content-Type-Options = %q, want nosniff", got)
	}
}

func TestHTTPHandlerRejectsInsecurePublicMetadataURLs(t *testing.T) {
	t.Parallel()

	_, err := NewHTTPHandler(HTTPConfig{
		BaseURL:             "https://api.pipeops.test",
		ResourceURL:         "http://mcp.pipeops.test/mcp",
		AuthorizationServer: "https://api.pipeops.test",
	})
	if err == nil || !strings.Contains(err.Error(), "public URLs must use https") {
		t.Fatalf("NewHTTPHandler() error = %v, want insecure URL rejection", err)
	}
}

func TestHTTPHandlerValidatesAndForwardsRequestToken(t *testing.T) {
	t.Parallel()

	const token = "customer-token"
	var mu sync.Mutex
	var authorizationHeaders []string
	api := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		mu.Lock()
		authorizationHeaders = append(authorizationHeaders, r.Header.Get("Authorization"))
		mu.Unlock()
		if r.URL.Path != "/profile/data" {
			http.Error(w, "unexpected path", http.StatusNotFound)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = io.WriteString(w, `{"success":true,"message":"ok","data":{"uuid":"user-123","email":"user@example.com"}}`)
	}))
	defer api.Close()

	handler, err := NewHTTPHandler(HTTPConfig{
		BaseURL:             api.URL,
		ResourceURL:         "https://mcp.pipeops.test/mcp",
		AuthorizationServer: "https://console.pipeops.test",
	})
	if err != nil {
		t.Fatalf("NewHTTPHandler() error = %v", err)
	}
	mcpServer := httptest.NewServer(handler)
	defer mcpServer.Close()

	httpClient := &http.Client{Transport: bearerTransport{token: token, base: http.DefaultTransport}}
	client := sdkmcp.NewClient(&sdkmcp.Implementation{Name: "pipeops-http-test", Version: "1.0.0"}, nil)
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	session, err := client.Connect(ctx, &sdkmcp.StreamableClientTransport{
		Endpoint:   mcpServer.URL + "/mcp",
		HTTPClient: httpClient,
		MaxRetries: -1,
	}, nil)
	if err != nil {
		t.Fatalf("Connect() error = %v", err)
	}
	defer func() {
		if err := session.Close(); err != nil {
			t.Errorf("Close() error = %v", err)
		}
	}()

	tools, err := session.ListTools(ctx, nil)
	if err != nil {
		t.Fatalf("ListTools() error = %v", err)
	}
	if len(tools.Tools) < 80 {
		t.Fatalf("tool count = %d, want at least 80", len(tools.Tools))
	}
	assertToolAnnotations(t, tools.Tools)

	result, err := session.CallTool(ctx, &sdkmcp.CallToolParams{Name: "get_current_user", Arguments: map[string]any{}})
	if err != nil {
		t.Fatalf("CallTool() error = %v", err)
	}
	if result.IsError {
		t.Fatalf("CallTool() returned tool error: %#v", result.Content)
	}
	if len(result.Content) != 1 {
		t.Fatalf("content length = %d, want 1", len(result.Content))
	}

	mu.Lock()
	defer mu.Unlock()
	if len(authorizationHeaders) < 4 {
		t.Fatalf("controller requests = %d, want validation for each MCP request plus tool call", len(authorizationHeaders))
	}
	for _, header := range authorizationHeaders {
		if header != "Bearer "+token {
			t.Fatalf("Authorization = %q, want Bearer token", header)
		}
	}
}

func assertToolAnnotations(t *testing.T, tools []*sdkmcp.Tool) {
	t.Helper()

	byName := make(map[string]*sdkmcp.Tool, len(tools))
	for _, tool := range tools {
		byName[tool.Name] = tool
	}
	listProjects := byName["list_projects"]
	if listProjects == nil || listProjects.Annotations == nil || !listProjects.Annotations.ReadOnlyHint {
		t.Fatalf("list_projects annotations = %#v, want readOnlyHint", listProjects)
	}
	deleteProject := byName["delete_project"]
	if deleteProject == nil || deleteProject.Annotations == nil || deleteProject.Annotations.DestructiveHint == nil || !*deleteProject.Annotations.DestructiveHint {
		t.Fatalf("delete_project annotations = %#v, want destructiveHint", deleteProject)
	}
}

func TestHTTPHandlerRejectsInvalidControllerToken(t *testing.T) {
	t.Parallel()

	api := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		http.Error(w, "unauthorized", http.StatusUnauthorized)
	}))
	defer api.Close()

	handler, err := NewHTTPHandler(HTTPConfig{
		BaseURL:             api.URL,
		ResourceURL:         "https://mcp.pipeops.test/mcp",
		AuthorizationServer: "https://console.pipeops.test",
	})
	if err != nil {
		t.Fatalf("NewHTTPHandler() error = %v", err)
	}

	req := httptest.NewRequest(http.MethodPost, "/mcp", strings.NewReader(`{"jsonrpc":"2.0","id":1,"method":"initialize","params":{}}`))
	req.Header.Set("Authorization", "Bearer invalid-token")
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json, text/event-stream")
	recorder := httptest.NewRecorder()
	handler.ServeHTTP(recorder, req)

	if recorder.Code != http.StatusUnauthorized {
		t.Fatalf("status = %d, want %d; body = %s", recorder.Code, http.StatusUnauthorized, recorder.Body.String())
	}
	if strings.Contains(recorder.Body.String(), "invalid-token") {
		t.Fatal("response leaked bearer token")
	}
	if got := recorder.Header().Get("WWW-Authenticate"); !strings.Contains(got, `error="invalid_token"`) {
		t.Fatalf("WWW-Authenticate = %q, want invalid_token", got)
	}
}

func TestHTTPHandlerKeepsConcurrentCallerTokensIsolated(t *testing.T) {
	t.Parallel()

	api := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		token := strings.TrimPrefix(r.Header.Get("Authorization"), "Bearer ")
		if token != "alpha-token" && token != "beta-token" {
			http.Error(w, "unauthorized", http.StatusUnauthorized)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = io.WriteString(w, `{"success":true,"message":"ok","data":{"uuid":"`+token+`","email":"`+token+`@example.com"}}`)
	}))
	defer api.Close()

	handler, err := NewHTTPHandler(HTTPConfig{
		BaseURL:             api.URL,
		ResourceURL:         "https://mcp.pipeops.test/mcp",
		AuthorizationServer: "https://console.pipeops.test",
	})
	if err != nil {
		t.Fatalf("NewHTTPHandler() error = %v", err)
	}
	mcpServer := httptest.NewServer(handler)
	defer mcpServer.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()
	errCh := make(chan error, 12)
	var wg sync.WaitGroup
	for i := 0; i < 12; i++ {
		token := "alpha-token"
		if i%2 == 1 {
			token = "beta-token"
		}
		wg.Add(1)
		go func() {
			defer wg.Done()
			client := sdkmcp.NewClient(&sdkmcp.Implementation{Name: "isolation-test", Version: "1.0.0"}, nil)
			session, err := client.Connect(ctx, &sdkmcp.StreamableClientTransport{
				Endpoint: mcpServer.URL + "/mcp",
				HTTPClient: &http.Client{Transport: bearerTransport{
					token: token,
					base:  http.DefaultTransport,
				}},
				MaxRetries: -1,
			}, nil)
			if err != nil {
				errCh <- err
				return
			}

			result, err := session.CallTool(ctx, &sdkmcp.CallToolParams{Name: "get_current_user", Arguments: map[string]any{}})
			if err != nil {
				_ = session.Close()
				errCh <- err
				return
			}
			if result.IsError || len(result.Content) != 1 {
				_ = session.Close()
				errCh <- errors.New("unexpected MCP tool result")
				return
			}
			content, ok := result.Content[0].(*sdkmcp.TextContent)
			if !ok || !strings.Contains(content.Text, token+"@example.com") {
				_ = session.Close()
				errCh <- errors.New("caller received another token's profile")
				return
			}
			if err := session.Close(); err != nil {
				errCh <- err
			}
		}()
	}
	wg.Wait()
	close(errCh)
	for err := range errCh {
		if err != nil {
			t.Fatal(err)
		}
	}
}

type bearerTransport struct {
	token string
	base  http.RoundTripper
}

func (t bearerTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	clone := req.Clone(req.Context())
	clone.Header = req.Header.Clone()
	clone.Header.Set("Authorization", "Bearer "+t.token)
	return t.base.RoundTrip(clone)
}

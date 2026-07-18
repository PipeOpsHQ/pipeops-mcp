package mcp

import (
	"context"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"path/filepath"
	"regexp"
	"strings"
	"testing"
	"time"

	sdkmcp "github.com/modelcontextprotocol/go-sdk/mcp"
)

func TestOAuthBridgeAuthorizationCodeRefreshAndScopeFlow(t *testing.T) {
	tests := []struct {
		name     string
		newStore func(*testing.T) oauthStore
	}{
		{
			name: "memory",
			newStore: func(*testing.T) oauthStore {
				return newMemoryOAuthStore()
			},
		},
		{
			name: "sqlite",
			newStore: func(t *testing.T) oauthStore {
				store, err := newSQLiteOAuthStore(context.Background(), filepath.Join(t.TempDir(), "oauth.db"))
				if err != nil {
					t.Fatalf("open SQLite OAuth store: %v", err)
				}
				t.Cleanup(func() { _ = store.Close() })
				return store
			},
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			testOAuthBridgeAuthorizationCodeRefreshAndScopeFlow(t, test.newStore(t))
		})
	}
}

func testOAuthBridgeAuthorizationCodeRefreshAndScopeFlow(t *testing.T, store oauthStore) {
	api := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/workspace" || r.Header.Get("Authorization") != "Bearer sat_valid" {
			http.Error(w, "unauthorized", http.StatusUnauthorized)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = io.WriteString(w, `{"success":true,"message":"ok","data":[{"uuid":"workspace-user","name":"Test"}]}`)
	}))
	defer api.Close()

	handler := newTestOAuthBridgeHandlerWithStore(t, api.URL, store)
	server := httptest.NewServer(handler)
	defer server.Close()

	clientID := registerOAuthClient(t, server.URL, "http://127.0.0.1:8765/callback")
	verifier := strings.Repeat("v", 64)
	digest := sha256.Sum256([]byte(verifier))
	challenge := base64.RawURLEncoding.EncodeToString(digest[:])
	authorizeURL := server.URL + "/oauth/authorize?" + url.Values{
		"client_id":             {clientID},
		"redirect_uri":          {"http://127.0.0.1:8765/callback"},
		"response_type":         {"code"},
		"scope":                 {"api:read"},
		"resource":              {"https://mcp.pipeops.test/mcp"},
		"state":                 {"state-123"},
		"code_challenge":        {challenge},
		"code_challenge_method": {"S256"},
	}.Encode()

	response, err := http.Get(authorizeURL)
	if err != nil {
		t.Fatalf("begin authorization: %v", err)
	}
	page, _ := io.ReadAll(response.Body)
	_ = response.Body.Close()
	if response.StatusCode != http.StatusOK {
		t.Fatalf("authorize status = %d; body = %s", response.StatusCode, page)
	}
	for _, expected := range []string{
		`id="form_error"`,
		`Paste a PipeOps workspace service token before connecting.`,
		`Enter a valid PipeOps workspace service token beginning with sat_.`,
		`connect.textContent = "Connecting…"`,
	} {
		if !strings.Contains(string(page), expected) {
			t.Fatalf("authorization page missing visible submit feedback %q: %s", expected, page)
		}
	}
	nonceMatch := regexp.MustCompile(`<script nonce="([^"]+)">`).FindSubmatch(page)
	if len(nonceMatch) != 2 {
		t.Fatalf("authorization page missing script nonce: %s", page)
	}
	csp := response.Header.Get("Content-Security-Policy")
	if !strings.Contains(csp, "script-src 'nonce-"+string(nonceMatch[1])+"'") {
		t.Fatalf("authorization page CSP does not allow its nonce: %q", csp)
	}
	match := regexp.MustCompile(`name="request_id" value="([^"]+)"`).FindSubmatch(page)
	if len(match) != 2 {
		t.Fatalf("authorization page missing request id: %s", page)
	}

	noRedirect := &http.Client{CheckRedirect: func(_ *http.Request, _ []*http.Request) error {
		return http.ErrUseLastResponse
	}}
	response, err = noRedirect.PostForm(server.URL+"/oauth/authorize", url.Values{
		"request_id":    {string(match[1])},
		"service_token": {"sat_valid"},
		"action":        {"approve"},
	})
	if err != nil {
		t.Fatalf("complete authorization: %v", err)
	}
	_ = response.Body.Close()
	if response.StatusCode != http.StatusFound {
		t.Fatalf("authorization completion status = %d", response.StatusCode)
	}
	redirect, err := url.Parse(response.Header.Get("Location"))
	if err != nil {
		t.Fatalf("parse authorization redirect: %v", err)
	}
	code := redirect.Query().Get("code")
	if code == "" || redirect.Query().Get("state") != "state-123" {
		t.Fatalf("unexpected authorization redirect: %s", redirect)
	}

	wrongResource := postOAuthForm(t, server.URL+"/oauth/token", url.Values{
		"grant_type":    {"authorization_code"},
		"client_id":     {clientID},
		"code":          {code},
		"redirect_uri":  {"http://127.0.0.1:8765/callback"},
		"resource":      {"https://attacker.example/mcp"},
		"code_verifier": {verifier},
	})
	if wrongResource.StatusCode != http.StatusBadRequest || !strings.Contains(wrongResource.Body, "invalid_target") {
		t.Fatalf("resource mismatch = %#v", wrongResource)
	}

	wrongVerifier := postOAuthForm(t, server.URL+"/oauth/token", url.Values{
		"grant_type":    {"authorization_code"},
		"client_id":     {clientID},
		"code":          {code},
		"redirect_uri":  {"http://127.0.0.1:8765/callback"},
		"code_verifier": {strings.Repeat("x", 64)},
	})
	if wrongVerifier.StatusCode != http.StatusBadRequest || !strings.Contains(wrongVerifier.Body, "invalid_grant") {
		t.Fatalf("PKCE mismatch = %#v", wrongVerifier)
	}

	tokens := exchangeOAuthCode(t, server.URL, clientID, code, verifier)
	if !strings.HasPrefix(tokens.AccessToken, "poat_") || !strings.HasPrefix(tokens.RefreshToken, "port_") {
		t.Fatalf("unexpected bridge tokens: %#v", tokens)
	}

	replay := postOAuthForm(t, server.URL+"/oauth/token", url.Values{
		"grant_type":    {"authorization_code"},
		"client_id":     {clientID},
		"code":          {code},
		"redirect_uri":  {"http://127.0.0.1:8765/callback"},
		"code_verifier": {verifier},
	})
	if replay.StatusCode != http.StatusBadRequest || !strings.Contains(replay.Body, "invalid_grant") {
		t.Fatalf("authorization code replay = %#v", replay)
	}

	client := sdkmcp.NewClient(&sdkmcp.Implementation{Name: "oauth-bridge-test", Version: "1.0.0"}, nil)
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	session, err := client.Connect(ctx, &sdkmcp.StreamableClientTransport{
		Endpoint:   server.URL + "/mcp",
		HTTPClient: &http.Client{Transport: bearerTransport{token: tokens.AccessToken, base: http.DefaultTransport}},
		MaxRetries: -1,
	}, nil)
	if err != nil {
		t.Fatalf("connect with OAuth bridge access token: %v", err)
	}
	tools, err := session.ListTools(ctx, nil)
	if err != nil {
		t.Fatalf("list OAuth-scoped tools: %v", err)
	}
	if toolNamed(tools.Tools, "create_project") {
		t.Fatal("api:read grant exposed create_project")
	}
	if !toolNamed(tools.Tools, "list_projects") {
		t.Fatal("api:read grant did not expose list_projects")
	}
	_ = session.Close()

	refreshed := postOAuthForm(t, server.URL+"/oauth/token", url.Values{
		"grant_type":    {"refresh_token"},
		"client_id":     {clientID},
		"refresh_token": {tokens.RefreshToken},
	})
	if refreshed.StatusCode != http.StatusOK {
		t.Fatalf("refresh token status = %d; body = %s", refreshed.StatusCode, refreshed.Body)
	}
	var refreshedTokens oauthTokenResponse
	if err := json.Unmarshal([]byte(refreshed.Body), &refreshedTokens); err != nil {
		t.Fatalf("decode refreshed tokens: %v", err)
	}
	if refreshedTokens.RefreshToken == tokens.RefreshToken || refreshedTokens.AccessToken == tokens.AccessToken {
		t.Fatal("refresh did not rotate both tokens")
	}

	reusedRefresh := postOAuthForm(t, server.URL+"/oauth/token", url.Values{
		"grant_type":    {"refresh_token"},
		"client_id":     {clientID},
		"refresh_token": {tokens.RefreshToken},
	})
	if reusedRefresh.StatusCode != http.StatusBadRequest || !strings.Contains(reusedRefresh.Body, "invalid_grant") {
		t.Fatalf("refresh replay = %#v", reusedRefresh)
	}

	revokedFamily := postOAuthForm(t, server.URL+"/oauth/token", url.Values{
		"grant_type":    {"refresh_token"},
		"client_id":     {clientID},
		"refresh_token": {refreshedTokens.RefreshToken},
	})
	if revokedFamily.StatusCode != http.StatusBadRequest || !strings.Contains(revokedFamily.Body, "invalid_grant") {
		t.Fatalf("active refresh token survived family replay detection = %#v", revokedFamily)
	}
}

func TestOAuthBridgeRejectsUnsafeAndMismatchedRedirects(t *testing.T) {
	handler := newTestOAuthBridgeHandler(t, "https://api.pipeops.test")
	server := httptest.NewServer(handler)
	defer server.Close()

	unsafe := postJSON(t, server.URL+"/oauth/register", `{
        "redirect_uris":["http://attacker.example/callback"],
        "token_endpoint_auth_method":"none"
    }`)
	if unsafe.StatusCode != http.StatusBadRequest || !strings.Contains(unsafe.Body, "loopback") {
		t.Fatalf("unsafe redirect registration = %#v", unsafe)
	}

	clientID := registerOAuthClient(t, server.URL, "http://127.0.0.1:8765/callback")
	response, err := http.Get(server.URL + "/oauth/authorize?" + url.Values{
		"client_id":             {clientID},
		"redirect_uri":          {"http://127.0.0.1:9999/callback"},
		"response_type":         {"code"},
		"code_challenge":        {strings.Repeat("a", 43)},
		"code_challenge_method": {"S256"},
	}.Encode())
	if err != nil {
		t.Fatalf("mismatched redirect request: %v", err)
	}
	defer func() { _ = response.Body.Close() }()
	if response.StatusCode != http.StatusBadRequest {
		t.Fatalf("mismatched redirect status = %d", response.StatusCode)
	}
}

func TestOAuthBridgeEnforcesRegisteredClientAuthenticationMethod(t *testing.T) {
	handler := newTestOAuthBridgeHandler(t, "https://api.pipeops.test")
	server := httptest.NewServer(handler)
	defer server.Close()

	clientID, clientSecret := registerOAuthClientWithMethod(t, server.URL, "https://client.example/callback", "client_secret_basic")
	bodyCredentials := postOAuthForm(t, server.URL+"/oauth/token", url.Values{
		"grant_type":    {"unsupported"},
		"client_id":     {clientID},
		"client_secret": {clientSecret},
	})
	if bodyCredentials.StatusCode != http.StatusUnauthorized || !strings.Contains(bodyCredentials.Body, "invalid_client") {
		t.Fatalf("basic client accepted body credentials = %#v", bodyCredentials)
	}

	request, err := http.NewRequest(http.MethodPost, server.URL+"/oauth/token", strings.NewReader(url.Values{
		"grant_type": {"unsupported"},
	}.Encode()))
	if err != nil {
		t.Fatalf("build token request: %v", err)
	}
	request.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	request.SetBasicAuth(clientID, clientSecret)
	response, err := http.DefaultClient.Do(request)
	if err != nil {
		t.Fatalf("send token request: %v", err)
	}
	defer func() { _ = response.Body.Close() }()
	responseBody, _ := io.ReadAll(response.Body)
	if response.StatusCode != http.StatusBadRequest || !strings.Contains(string(responseBody), "unsupported_grant_type") {
		t.Fatalf("registered basic credentials rejected: status=%d body=%s", response.StatusCode, responseBody)
	}
}

func TestOAuthBridgeRegistrationRejectsTrailingJSON(t *testing.T) {
	handler := newTestOAuthBridgeHandler(t, "https://api.pipeops.test")
	server := httptest.NewServer(handler)
	defer server.Close()

	response := postJSON(t, server.URL+"/oauth/register", `{"redirect_uris":["https://client.example/callback"]}{}`)
	if response.StatusCode != http.StatusBadRequest || !strings.Contains(response.Body, "invalid_client_metadata") {
		t.Fatalf("trailing registration JSON = %#v", response)
	}
}

func TestOAuthBridgeRegistrationIgnoresUnknownMetadata(t *testing.T) {
	handler := newTestOAuthBridgeHandler(t, "https://api.pipeops.test")
	server := httptest.NewServer(handler)
	defer server.Close()

	response := postJSON(t, server.URL+"/oauth/register", `{
		"redirect_uris":["https://client.example/callback"],
		"token_endpoint_auth_method":"none",
		"future_extension":"ignored"
	}`)
	if response.StatusCode != http.StatusCreated {
		t.Fatalf("registration extension status = %d; body = %s", response.StatusCode, response.Body)
	}
}

func TestOAuthBridgeAllowsSupportedAuthorizationScopesBeyondRegistrationDefaults(t *testing.T) {
	handler := newTestOAuthBridgeHandler(t, "https://api.pipeops.test")
	server := httptest.NewServer(handler)
	defer server.Close()

	registration := postJSON(t, server.URL+"/oauth/register", `{
		"redirect_uris":["https://claude.ai/api/mcp/auth_callback"],
		"token_endpoint_auth_method":"none",
		"grant_types":["authorization_code","refresh_token"],
		"response_types":["code"],
		"client_name":"Claude",
		"scope":"api:read"
	}`)
	if registration.StatusCode != http.StatusCreated {
		t.Fatalf("Claude registration status = %d; body = %s", registration.StatusCode, registration.Body)
	}
	var registered struct {
		ClientID string `json:"client_id"`
	}
	if err := json.Unmarshal([]byte(registration.Body), &registered); err != nil || registered.ClientID == "" {
		t.Fatalf("decode Claude registration: %v; body = %s", err, registration.Body)
	}

	noRedirect := &http.Client{CheckRedirect: func(_ *http.Request, _ []*http.Request) error {
		return http.ErrUseLastResponse
	}}
	response, err := noRedirect.Get(server.URL + "/oauth/authorize?" + url.Values{
		"client_id":             {registered.ClientID},
		"redirect_uri":          {"https://claude.ai/api/mcp/auth_callback"},
		"response_type":         {"code"},
		"scope":                 {"api:read api:write"},
		"resource":              {"https://mcp.pipeops.test/mcp"},
		"state":                 {"claude-state"},
		"code_challenge":        {strings.Repeat("a", 43)},
		"code_challenge_method": {"S256"},
	}.Encode())
	if err != nil {
		t.Fatalf("begin Claude authorization: %v", err)
	}
	defer func() { _ = response.Body.Close() }()
	body, _ := io.ReadAll(response.Body)
	if response.StatusCode != http.StatusOK {
		t.Fatalf("Claude authorization status = %d; location = %s; body = %s", response.StatusCode, response.Header.Get("Location"), body)
	}
	if !strings.Contains(string(body), "api:read") || !strings.Contains(string(body), "api:write") {
		t.Fatalf("Claude authorization page does not show requested scopes: %s", body)
	}

	unsupported, err := noRedirect.Get(server.URL + "/oauth/authorize?" + url.Values{
		"client_id":             {registered.ClientID},
		"redirect_uri":          {"https://claude.ai/api/mcp/auth_callback"},
		"response_type":         {"code"},
		"scope":                 {"api:admin"},
		"resource":              {"https://mcp.pipeops.test/mcp"},
		"state":                 {"claude-state"},
		"code_challenge":        {strings.Repeat("a", 43)},
		"code_challenge_method": {"S256"},
	}.Encode())
	if err != nil {
		t.Fatalf("request unsupported Claude scope: %v", err)
	}
	defer func() { _ = unsupported.Body.Close() }()
	if unsupported.StatusCode != http.StatusFound || !strings.Contains(unsupported.Header.Get("Location"), "error=invalid_scope") {
		t.Fatalf("unsupported Claude scope status = %d; location = %s", unsupported.StatusCode, unsupported.Header.Get("Location"))
	}
}

func TestOAuthCredentialEncryptionBindsGrantMetadata(t *testing.T) {
	key := base64.StdEncoding.EncodeToString([]byte("0123456789abcdef0123456789abcdef"))
	cipher, err := newCredentialCipher(key)
	if err != nil {
		t.Fatalf("create cipher: %v", err)
	}
	readGrant := oauthGrantAdditionalData("client", []string{"api:read"}, "https://mcp.pipeops.test/mcp", "user")
	sealed, err := cipher.Seal("sat_secret", readGrant)
	if err != nil {
		t.Fatalf("seal credential: %v", err)
	}
	opened, err := cipher.Open(sealed, readGrant)
	if err != nil || opened != "sat_secret" {
		t.Fatalf("open credential: value=%q err=%v", opened, err)
	}
	writeGrant := oauthGrantAdditionalData("client", []string{"api:write"}, "https://mcp.pipeops.test/mcp", "user")
	if _, err := cipher.Open(sealed, writeGrant); err == nil {
		t.Fatal("credential decrypted after OAuth grant scopes were changed")
	}
}

func TestOAuthRevocationFailsClosedWhenStoreIsUnavailable(t *testing.T) {
	key := base64.StdEncoding.EncodeToString([]byte("0123456789abcdef0123456789abcdef"))
	store := &failingGrantReadStore{oauthStore: newMemoryOAuthStore()}
	handler, err := NewHTTPHandler(HTTPConfig{
		BaseURL:             "https://api.pipeops.test",
		ResourceURL:         "https://mcp.pipeops.test/mcp",
		OAuthMode:           "bridge",
		AuthorizationServer: "https://mcp.pipeops.test",
		OAuthEncryptionKey:  key,
		Scopes:              []string{"api:read", "api:write"},
		oauthStore:          store,
	})
	if err != nil {
		t.Fatalf("NewHTTPHandler: %v", err)
	}
	server := httptest.NewServer(handler)
	defer server.Close()
	clientID := registerOAuthClient(t, server.URL, "https://client.example/callback")

	response := postOAuthForm(t, server.URL+"/oauth/revoke", url.Values{
		"client_id": {clientID},
		"token":     {"poat_unavailable"},
	})
	if response.StatusCode != http.StatusServiceUnavailable || !strings.Contains(response.Body, "temporarily_unavailable") {
		t.Fatalf("revocation during store outage = %#v", response)
	}
}

func TestBearerOnlyMetadataDoesNotAdvertiseUnavailableOAuth(t *testing.T) {
	handler, err := NewHTTPHandler(HTTPConfig{
		BaseURL:     "https://api.pipeops.test",
		ResourceURL: "https://mcp.pipeops.test/mcp",
		OAuthMode:   "bearer",
	})
	if err != nil {
		t.Fatalf("NewHTTPHandler: %v", err)
	}
	recorder := httptest.NewRecorder()
	handler.ServeHTTP(recorder, httptest.NewRequest(http.MethodGet, "/.well-known/oauth-protected-resource", nil))
	var metadata struct {
		AuthorizationServers []string `json:"authorization_servers"`
	}
	if err := json.NewDecoder(recorder.Body).Decode(&metadata); err != nil {
		t.Fatalf("decode metadata: %v", err)
	}
	if len(metadata.AuthorizationServers) != 0 {
		t.Fatalf("bearer-only mode advertised OAuth servers: %#v", metadata.AuthorizationServers)
	}
}

type oauthTokenResponse struct {
	AccessToken  string `json:"access_token"`
	RefreshToken string `json:"refresh_token"`
}

type testHTTPResponse struct {
	StatusCode int
	Body       string
}

type failingGrantReadStore struct {
	oauthStore
}

func (s *failingGrantReadStore) Get(ctx context.Context, key string) ([]byte, error) {
	if strings.HasPrefix(key, "access:") {
		return nil, errors.New("store unavailable")
	}
	return s.oauthStore.Get(ctx, key)
}

func newTestOAuthBridgeHandler(t *testing.T, baseURL string) http.Handler {
	t.Helper()
	return newTestOAuthBridgeHandlerWithStore(t, baseURL, newMemoryOAuthStore())
}

func newTestOAuthBridgeHandlerWithStore(t *testing.T, baseURL string, store oauthStore) http.Handler {
	t.Helper()
	key := base64.StdEncoding.EncodeToString([]byte("0123456789abcdef0123456789abcdef"))
	handler, err := NewHTTPHandler(HTTPConfig{
		BaseURL:             baseURL,
		ResourceURL:         "https://mcp.pipeops.test/mcp",
		OAuthMode:           "bridge",
		AuthorizationServer: "https://mcp.pipeops.test",
		OAuthEncryptionKey:  key,
		Scopes:              []string{"api:read", "api:write"},
		oauthStore:          store,
	})
	if err != nil {
		t.Fatalf("NewHTTPHandler: %v", err)
	}
	return handler
}

func registerOAuthClient(t *testing.T, serverURL, redirectURI string) string {
	t.Helper()
	clientID, _ := registerOAuthClientWithMethod(t, serverURL, redirectURI, "none")
	return clientID
}

func registerOAuthClientWithMethod(t *testing.T, serverURL, redirectURI, method string) (string, string) {
	t.Helper()
	payload, _ := json.Marshal(map[string]any{
		"redirect_uris":              []string{redirectURI},
		"token_endpoint_auth_method": method,
		"grant_types":                []string{"authorization_code", "refresh_token"},
		"response_types":             []string{"code"},
		"client_name":                "Test AI client",
	})
	response := postJSON(t, serverURL+"/oauth/register", string(payload))
	if response.StatusCode != http.StatusCreated {
		t.Fatalf("register client status = %d; body = %s", response.StatusCode, response.Body)
	}
	var registered struct {
		ClientID     string `json:"client_id"`
		ClientSecret string `json:"client_secret"`
	}
	if err := json.Unmarshal([]byte(response.Body), &registered); err != nil || registered.ClientID == "" {
		t.Fatalf("decode client registration: %v; body = %s", err, response.Body)
	}
	return registered.ClientID, registered.ClientSecret
}

func exchangeOAuthCode(t *testing.T, serverURL, clientID, code, verifier string) oauthTokenResponse {
	t.Helper()
	response := postOAuthForm(t, serverURL+"/oauth/token", url.Values{
		"grant_type":    {"authorization_code"},
		"client_id":     {clientID},
		"code":          {code},
		"redirect_uri":  {"http://127.0.0.1:8765/callback"},
		"code_verifier": {verifier},
	})
	if response.StatusCode != http.StatusOK {
		t.Fatalf("exchange code status = %d; body = %s", response.StatusCode, response.Body)
	}
	var tokens oauthTokenResponse
	if err := json.Unmarshal([]byte(response.Body), &tokens); err != nil {
		t.Fatalf("decode token response: %v", err)
	}
	return tokens
}

func postOAuthForm(t *testing.T, endpoint string, values url.Values) testHTTPResponse {
	t.Helper()
	response, err := http.PostForm(endpoint, values)
	if err != nil {
		t.Fatalf("POST form %s: %v", endpoint, err)
	}
	body, _ := io.ReadAll(response.Body)
	_ = response.Body.Close()
	return testHTTPResponse{StatusCode: response.StatusCode, Body: string(body)}
}

func postJSON(t *testing.T, endpoint, payload string) testHTTPResponse {
	t.Helper()
	response, err := http.Post(endpoint, "application/json", strings.NewReader(payload))
	if err != nil {
		t.Fatalf("POST JSON %s: %v", endpoint, err)
	}
	body, _ := io.ReadAll(response.Body)
	_ = response.Body.Close()
	return testHTTPResponse{StatusCode: response.StatusCode, Body: string(body)}
}

func toolNamed(tools []*sdkmcp.Tool, name string) bool {
	for _, tool := range tools {
		if tool.Name == name {
			return true
		}
	}
	return false
}

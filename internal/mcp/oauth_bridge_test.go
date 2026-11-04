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
		switch r.URL.Path {
		case "/oauth/token":
			var request map[string]string
			if err := json.NewDecoder(r.Body).Decode(&request); err != nil || request["grant_type"] == "" {
				http.Error(w, "invalid token request", http.StatusBadRequest)
				return
			}
			_ = json.NewEncoder(w).Encode(map[string]any{"access_token": "console_access", "refresh_token": "console_refresh", "expires_in": 3600})
		case "/profile/data":
			if r.Header.Get("Authorization") != "Bearer console_access" {
				http.Error(w, "unauthorized", http.StatusUnauthorized)
				return
			}
			_, _ = io.WriteString(w, `{"success":true,"message":"ok","data":{"user":{"uuid":"user-1"}}}`)
		default:
			http.Error(w, "unauthorized", http.StatusUnauthorized)
		}
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

	location := approveOAuthConsent(t, server.URL, authorizeURL)
	code := completeConsoleAuthorization(t, server.URL, location, "console-code")

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

	phishing := postJSON(t, server.URL+"/oauth/register", `{
		"redirect_uris":["https://webhook.site/39354f08-2d8d-4101-ab74-45b4f7ac7490"],
		"client_name":"Malicious Client",
		"grant_types":["authorization_code","refresh_token"],
		"response_types":["code"],
		"scope":"api:read api:write"
	}`)
	if phishing.StatusCode != http.StatusBadRequest || strings.Contains(phishing.Body, "client_id") || !strings.Contains(phishing.Body, "not allowed") {
		t.Fatalf("open https redirect registration = %#v", phishing)
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

	clientID, clientSecret := registerOAuthClientWithMethod(t, server.URL, "http://127.0.0.1:8765/callback", "client_secret_basic")
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

	response := postJSON(t, server.URL+"/oauth/register", `{"redirect_uris":["http://127.0.0.1:8765/callback"]}{}`)
	if response.StatusCode != http.StatusBadRequest || !strings.Contains(response.Body, "invalid_client_metadata") {
		t.Fatalf("trailing registration JSON = %#v", response)
	}
}

func TestOAuthBridgeRegistrationIgnoresUnknownMetadata(t *testing.T) {
	handler := newTestOAuthBridgeHandler(t, "https://api.pipeops.test")
	server := httptest.NewServer(handler)
	defer server.Close()

	response := postJSON(t, server.URL+"/oauth/register", `{
		"redirect_uris":["https://claude.ai/api/mcp/auth_callback"],
		"token_endpoint_auth_method":"none",
		"future_extension":"ignored"
	}`)
	if response.StatusCode != http.StatusCreated {
		t.Fatalf("registration extension status = %d; body = %s", response.StatusCode, response.Body)
	}
}

func TestOAuthBridgeAllowsSupportedAuthorizationScopesBeyondRegistrationDefaults(t *testing.T) {
	api := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/oauth/token":
			_ = json.NewEncoder(w).Encode(map[string]any{"access_token": "claude_console_access", "refresh_token": "claude_console_refresh", "expires_in": 3600})
		case "/profile/data":
			if r.Header.Get("Authorization") != "Bearer claude_console_access" {
				http.Error(w, "unauthorized", http.StatusUnauthorized)
				return
			}
			_, _ = io.WriteString(w, `{"success":true,"message":"ok","data":{"user":{"uuid":"claude-user"}}}`)
		default:
			http.Error(w, "unauthorized", http.StatusUnauthorized)
		}
	}))
	defer api.Close()

	handler := newTestOAuthBridgeHandler(t, api.URL)
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
	verifier := strings.Repeat("c", 64)
	digest := sha256.Sum256([]byte(verifier))
	challenge := base64.RawURLEncoding.EncodeToString(digest[:])
	authorizeURL := server.URL + "/oauth/authorize?" + url.Values{
		"client_id":             {registered.ClientID},
		"redirect_uri":          {"https://claude.ai/api/mcp/auth_callback"},
		"response_type":         {"code"},
		"scope":                 {"api:read api:write"},
		"resource":              {"https://mcp.pipeops.test/mcp"},
		"state":                 {"claude-state"},
		"code_challenge":        {challenge},
		"code_challenge_method": {"S256"},
	}.Encode()
	location := approveOAuthConsent(t, server.URL, authorizeURL)
	code := completeConsoleAuthorization(t, server.URL, location, "claude-console-code")

	exchanged := postOAuthForm(t, server.URL+"/oauth/token", url.Values{
		"grant_type":    {"authorization_code"},
		"client_id":     {registered.ClientID},
		"code":          {code},
		"redirect_uri":  {"https://claude.ai/api/mcp/auth_callback"},
		"resource":      {"https://mcp.pipeops.test/mcp"},
		"code_verifier": {verifier},
	})
	if exchanged.StatusCode != http.StatusOK {
		t.Fatalf("exchange Claude authorization code status = %d; body = %s", exchanged.StatusCode, exchanged.Body)
	}
	var tokens oauthTokenResponse
	if err := json.Unmarshal([]byte(exchanged.Body), &tokens); err != nil {
		t.Fatalf("decode Claude tokens: %v", err)
	}
	if tokens.Scope != "api:read api:write" {
		t.Fatalf("Claude token scope = %q, want api:read api:write", tokens.Scope)
	}

	assertTools := func(accessToken string, wantWrite bool) {
		t.Helper()
		client := sdkmcp.NewClient(&sdkmcp.Implementation{Name: "claude-scope-test", Version: "1.0.0"}, nil)
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		session, err := client.Connect(ctx, &sdkmcp.StreamableClientTransport{
			Endpoint:   server.URL + "/mcp",
			HTTPClient: &http.Client{Transport: bearerTransport{token: accessToken, base: http.DefaultTransport}},
			MaxRetries: -1,
		}, nil)
		if err != nil {
			t.Fatalf("connect with Claude OAuth token: %v", err)
		}
		defer func() { _ = session.Close() }()
		tools, err := session.ListTools(ctx, nil)
		if err != nil {
			t.Fatalf("list Claude OAuth tools: %v", err)
		}
		if toolNamed(tools.Tools, "create_project") != wantWrite {
			t.Fatalf("create_project exposure = %t, want %t", toolNamed(tools.Tools, "create_project"), wantWrite)
		}
		if !toolNamed(tools.Tools, "list_projects") {
			t.Fatal("Claude OAuth token did not expose list_projects")
		}
	}
	assertTools(tokens.AccessToken, true)

	refreshed := postOAuthForm(t, server.URL+"/oauth/token", url.Values{
		"grant_type":    {"refresh_token"},
		"client_id":     {registered.ClientID},
		"refresh_token": {tokens.RefreshToken},
		"scope":         {"api:read"},
	})
	if refreshed.StatusCode != http.StatusOK {
		t.Fatalf("downscope Claude refresh status = %d; body = %s", refreshed.StatusCode, refreshed.Body)
	}
	var readOnlyTokens oauthTokenResponse
	if err := json.Unmarshal([]byte(refreshed.Body), &readOnlyTokens); err != nil {
		t.Fatalf("decode downscoped Claude tokens: %v", err)
	}
	if readOnlyTokens.Scope != "api:read" {
		t.Fatalf("downscoped Claude token scope = %q, want api:read", readOnlyTokens.Scope)
	}
	assertTools(readOnlyTokens.AccessToken, false)

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

func TestOAuthCredentialEncryptionBindsCredentialIdentity(t *testing.T) {
	key := base64.StdEncoding.EncodeToString([]byte("0123456789abcdef0123456789abcdef"))
	cipher, err := newCredentialCipher(key)
	if err != nil {
		t.Fatalf("create cipher: %v", err)
	}
	credentialData := oauthCredentialAdditionalData("credential", "user")
	sealed, err := cipher.Seal("console-credential", credentialData)
	if err != nil {
		t.Fatalf("seal credential: %v", err)
	}
	opened, err := cipher.Open(sealed, credentialData)
	if err != nil || opened != "console-credential" {
		t.Fatalf("open credential: value=%q err=%v", opened, err)
	}
	wrongCredentialData := oauthCredentialAdditionalData("credential", "another-user")
	if _, err := cipher.Open(sealed, wrongCredentialData); err == nil {
		t.Fatal("credential decrypted after its owner changed")
	}
}

func TestOAuthBridgeRefreshesExpiredConsoleCredential(t *testing.T) {
	var refreshCalls int
	api := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/oauth/token":
			var request map[string]string
			if err := json.NewDecoder(r.Body).Decode(&request); err != nil || request["grant_type"] != "refresh_token" || request["refresh_token"] != "old-refresh" {
				http.Error(w, "invalid refresh", http.StatusBadRequest)
				return
			}
			refreshCalls++
			_, _ = io.WriteString(w, `{"access_token":"renewed-access","refresh_token":"renewed-refresh","expires_in":3600}`)
		case "/profile/data":
			if r.Header.Get("Authorization") != "Bearer renewed-access" {
				http.Error(w, "unauthorized", http.StatusUnauthorized)
				return
			}
			_, _ = io.WriteString(w, `{"success":true,"data":{"user":{"uuid":"user-1"}}}`)
		default:
			http.NotFound(w, r)
		}
	}))
	defer api.Close()

	key := base64.StdEncoding.EncodeToString([]byte("0123456789abcdef0123456789abcdef"))
	cipher, err := newCredentialCipher(key)
	if err != nil {
		t.Fatalf("create cipher: %v", err)
	}
	store := newMemoryOAuthStore()
	bridge, err := newOAuthBridge(oauthBridgeConfig{
		Issuer:          "https://mcp.pipeops.test",
		ResourceURL:     "https://mcp.pipeops.test/mcp",
		BaseURL:         api.URL,
		ConsoleURL:      "https://console.pipeops.test",
		ConsoleClientID: "pipeops_public_client",
		ConsoleScopes:   []string{"openid", "profile", "email"},
		Scopes:          []string{"api:read"},
		Store:           store,
		Cipher:          cipher,
	})
	if err != nil {
		t.Fatalf("create bridge: %v", err)
	}
	credentialID := "pocr_test"
	credential := consoleCredential{AccessToken: "expired-access", RefreshToken: "old-refresh", ExpiresAt: time.Now().Add(-time.Minute)}
	if err := bridge.storeConsoleCredential(context.Background(), credentialID, "user-1", credential); err != nil {
		t.Fatalf("store Console credential: %v", err)
	}
	if err := putOAuthJSON(context.Background(), store, "client:pomcp_testclient", &oauthClientRecord{
		ClientID:                "pomcp_testclient",
		RedirectURIs:            []string{"http://127.0.0.1:8765/callback"},
		TokenEndpointAuthMethod: "none",
		ClientName:              "Test AI client",
	}, time.Hour); err != nil {
		t.Fatalf("store client: %v", err)
	}
	grant := oauthGrantRecord{ClientID: "pomcp_testclient", Scopes: []string{"api:read"}, Resource: "https://mcp.pipeops.test/mcp", CredentialID: credentialID, UserID: "user-1"}
	if err := putOAuthJSON(context.Background(), store, oauthLookupKey("access", "poat_test"), &grant, oauthAccessTokenTTL); err != nil {
		t.Fatalf("store grant: %v", err)
	}

	resolved, err := bridge.resolveAccessToken(context.Background(), "poat_test")
	if err != nil {
		t.Fatalf("resolve access token: %v", err)
	}
	if resolved.UpstreamToken != "renewed-access" || refreshCalls != 1 {
		t.Fatalf("refreshed credential = %#v, calls = %d", resolved, refreshCalls)
	}
	stored, err := bridge.openGrantCredential(context.Background(), grant)
	if err != nil || stored.AccessToken != "renewed-access" || stored.RefreshToken != "renewed-refresh" {
		t.Fatalf("persisted refreshed credential = %#v, err = %v", stored, err)
	}
}

func TestOAuthBridgeRenewsEndedConsoleSession(t *testing.T) {
	tests := []struct {
		name             string
		refreshToken     string
		profileStatus    int
		renewedStatus    int
		wantToken        string
		wantRefreshCalls int
		wantNotFound     bool
		wantUnavailable  bool
	}{
		{
			name:             "ended session renews before expiry",
			refreshToken:     "old-refresh",
			profileStatus:    pipeOpsSessionExpired,
			renewedStatus:    http.StatusOK,
			wantToken:        "renewed-access",
			wantRefreshCalls: 1,
		},
		{
			name:             "ended session without refresh token",
			profileStatus:    pipeOpsSessionExpired,
			renewedStatus:    http.StatusOK,
			wantNotFound:     true,
			wantRefreshCalls: 0,
		},
		{
			name:             "rejected credential is not renewed",
			refreshToken:     "old-refresh",
			profileStatus:    http.StatusUnauthorized,
			renewedStatus:    http.StatusOK,
			wantNotFound:     true,
			wantRefreshCalls: 0,
		},
		{
			name:             "renewed token that is still ended is not refreshed twice",
			refreshToken:     "old-refresh",
			profileStatus:    pipeOpsSessionExpired,
			renewedStatus:    pipeOpsSessionExpired,
			wantNotFound:     true,
			wantRefreshCalls: 1,
		},
		{
			name:             "upstream outage is not treated as an ended session",
			refreshToken:     "old-refresh",
			profileStatus:    http.StatusServiceUnavailable,
			renewedStatus:    http.StatusOK,
			wantUnavailable:  true,
			wantRefreshCalls: 0,
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			var refreshCalls int
			api := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				switch r.URL.Path {
				case "/oauth/token":
					var request map[string]string
					if err := json.NewDecoder(r.Body).Decode(&request); err != nil || request["grant_type"] != "refresh_token" || request["refresh_token"] != "old-refresh" {
						http.Error(w, "invalid refresh", http.StatusBadRequest)
						return
					}
					refreshCalls++
					_, _ = io.WriteString(w, `{"access_token":"renewed-access","refresh_token":"renewed-refresh","expires_in":3600}`)
				case "/profile/data":
					token := strings.TrimPrefix(r.Header.Get("Authorization"), "Bearer ")
					status := test.profileStatus
					if token == "renewed-access" {
						status = test.renewedStatus
					}
					if status == http.StatusOK {
						_, _ = io.WriteString(w, `{"success":true,"data":{"user":{"uuid":"user-1"}}}`)
						return
					}
					w.WriteHeader(status)
					_, _ = io.WriteString(w, `{"error":"session has ended. Please log in again."}`)
				default:
					http.NotFound(w, r)
				}
			}))
			defer api.Close()

			bridge := newConsoleRefreshBridge(t, api.URL)
			credentialID := "pocr_ended"
			grant := oauthGrantRecord{ClientID: "pomcp_testclient", Scopes: []string{"api:read"}, Resource: "https://mcp.pipeops.test/mcp", CredentialID: credentialID, UserID: "user-1"}
			credential := consoleCredential{AccessToken: "ended-access", RefreshToken: test.refreshToken, ExpiresAt: time.Now().Add(time.Hour)}
			if err := bridge.storeConsoleCredential(context.Background(), credentialID, "user-1", credential); err != nil {
				t.Fatalf("store Console credential: %v", err)
			}
			if err := putOAuthJSON(context.Background(), bridge.store, oauthLookupKey("access", "poat_ended"), &grant, oauthAccessTokenTTL); err != nil {
				t.Fatalf("store grant: %v", err)
			}

			resolved, err := bridge.resolveAccessToken(context.Background(), "poat_ended")
			if refreshCalls != test.wantRefreshCalls {
				t.Fatalf("refresh calls = %d, want %d", refreshCalls, test.wantRefreshCalls)
			}
			switch {
			case test.wantNotFound:
				if !errors.Is(err, errOAuthRecordNotFound) {
					t.Fatalf("resolve = %#v, %v; want not found", resolved, err)
				}
			case test.wantUnavailable:
				if err == nil || errors.Is(err, errOAuthRecordNotFound) || !strings.Contains(err.Error(), "unavailable") {
					t.Fatalf("resolve = %#v, %v; want unavailable", resolved, err)
				}
			default:
				if err != nil || resolved == nil || resolved.UpstreamToken != test.wantToken {
					t.Fatalf("resolve = %#v, %v; want upstream %s", resolved, err, test.wantToken)
				}
				stored, err := bridge.openGrantCredential(context.Background(), grant)
				if err != nil || stored.AccessToken != "renewed-access" || stored.RefreshToken != "renewed-refresh" {
					t.Fatalf("persisted renewed credential = %#v, err = %v", stored, err)
				}
			}
		})
	}
}

func TestOAuthBridgeRefreshGrantRenewsEndedConsoleSession(t *testing.T) {
	var refreshCalls int
	api := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/oauth/token":
			var request map[string]string
			if err := json.NewDecoder(r.Body).Decode(&request); err != nil || request["grant_type"] != "refresh_token" || request["refresh_token"] != "old-refresh" {
				http.Error(w, "invalid refresh", http.StatusBadRequest)
				return
			}
			refreshCalls++
			_, _ = io.WriteString(w, `{"access_token":"renewed-access","refresh_token":"renewed-refresh","expires_in":3600}`)
		case "/profile/data":
			token := strings.TrimPrefix(r.Header.Get("Authorization"), "Bearer ")
			if token == "ended-access" {
				w.WriteHeader(pipeOpsSessionExpired)
				_, _ = io.WriteString(w, `{"error":"session has ended. Please log in again."}`)
				return
			}
			if token != "renewed-access" {
				http.Error(w, "unauthorized", http.StatusUnauthorized)
				return
			}
			_, _ = io.WriteString(w, `{"success":true,"data":{"user":{"uuid":"user-1"}}}`)
		default:
			http.NotFound(w, r)
		}
	}))
	defer api.Close()

	bridge := newConsoleRefreshBridge(t, api.URL)
	credentialID := "pocr_refresh_grant"
	familyID := "porf_refresh_grant"
	grant := oauthGrantRecord{
		ClientID:     "pomcp_testclient",
		FamilyID:     familyID,
		Scopes:       []string{"api:read"},
		Resource:     "https://mcp.pipeops.test/mcp",
		CredentialID: credentialID,
		UserID:       "user-1",
	}
	credential := consoleCredential{AccessToken: "ended-access", RefreshToken: "old-refresh", ExpiresAt: time.Now().Add(time.Hour)}
	if err := bridge.storeConsoleCredential(context.Background(), credentialID, "user-1", credential); err != nil {
		t.Fatalf("store Console credential: %v", err)
	}
	refreshToken := "port_ended"
	refreshKey := oauthLookupKey("refresh", refreshToken)
	if err := putOAuthJSON(context.Background(), bridge.store, refreshKey, &grant, oauthRefreshTokenTTL); err != nil {
		t.Fatalf("store refresh grant: %v", err)
	}
	if err := bridge.store.Put(context.Background(), oauthRefreshFamilyKey(familyID), []byte(refreshKey), oauthRefreshTokenTTL); err != nil {
		t.Fatalf("store refresh family: %v", err)
	}

	request := httptest.NewRequest(http.MethodPost, "/oauth/token", strings.NewReader(url.Values{
		"grant_type":    {"refresh_token"},
		"client_id":     {"pomcp_testclient"},
		"refresh_token": {refreshToken},
	}.Encode()))
	request.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	recorder := httptest.NewRecorder()
	bridge.handleToken(recorder, request)
	if recorder.Code != http.StatusOK {
		t.Fatalf("refresh grant status = %d; body = %s", recorder.Code, recorder.Body.String())
	}
	if refreshCalls != 1 {
		t.Fatalf("console refresh calls = %d, want 1", refreshCalls)
	}
	stored, err := bridge.openGrantCredential(context.Background(), grant)
	if err != nil || stored.AccessToken != "renewed-access" || stored.RefreshToken != "renewed-refresh" {
		t.Fatalf("persisted renewed credential = %#v, err = %v", stored, err)
	}
}

func newConsoleRefreshBridge(t *testing.T, baseURL string) *oauthBridge {
	t.Helper()
	key := base64.StdEncoding.EncodeToString([]byte("0123456789abcdef0123456789abcdef"))
	cipher, err := newCredentialCipher(key)
	if err != nil {
		t.Fatalf("create cipher: %v", err)
	}
	store := newMemoryOAuthStore()
	bridge, err := newOAuthBridge(oauthBridgeConfig{
		Issuer:          "https://mcp.pipeops.test",
		ResourceURL:     "https://mcp.pipeops.test/mcp",
		BaseURL:         baseURL,
		ConsoleURL:      "https://console.pipeops.test",
		ConsoleClientID: "pipeops_public_client",
		ConsoleScopes:   []string{"openid", "profile", "email"},
		Scopes:          []string{"api:read"},
		Store:           store,
		Cipher:          cipher,
	})
	if err != nil {
		t.Fatalf("create bridge: %v", err)
	}
	if err := putOAuthJSON(context.Background(), store, "client:pomcp_testclient", &oauthClientRecord{
		ClientID:                "pomcp_testclient",
		RedirectURIs:            []string{"http://127.0.0.1:8765/callback"},
		TokenEndpointAuthMethod: "none",
		ClientName:              "Test AI client",
	}, time.Hour); err != nil {
		t.Fatalf("store client: %v", err)
	}
	return bridge
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
	clientID := registerOAuthClient(t, server.URL, "http://127.0.0.1:8765/callback")

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
	Scope        string `json:"scope"`
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

func completeConsoleAuthorization(t *testing.T, serverURL, consoleLocation, consoleCode string) string {
	t.Helper()
	consoleURL, err := url.Parse(consoleLocation)
	if err != nil {
		t.Fatalf("parse Console authorization URL: %v", err)
	}
	query := consoleURL.Query()
	if consoleURL.Path != "/auth/signin" || query.Get("response_type") != "code" || query.Get("client_id") != "pipeops_public_client" {
		t.Fatalf("unexpected Console authorization request: %s", consoleURL)
	}
	if query.Get("redirect_uri") != "https://mcp.pipeops.test/oauth/pipeops/callback" || query.Get("state") == "" {
		t.Fatalf("Console callback binding is missing: %s", consoleURL)
	}
	if query.Get("code_challenge_method") != "S256" || len(query.Get("code_challenge")) < 43 {
		t.Fatalf("Console PKCE challenge is missing: %s", consoleURL)
	}
	noRedirect := &http.Client{CheckRedirect: func(_ *http.Request, _ []*http.Request) error {
		return http.ErrUseLastResponse
	}}
	response, err := noRedirect.Get(serverURL + "/oauth/pipeops/callback?" + url.Values{
		"state": {query.Get("state")},
		"code":  {consoleCode},
	}.Encode())
	if err != nil {
		t.Fatalf("complete Console authorization: %v", err)
	}
	defer func() { _ = response.Body.Close() }()
	if response.StatusCode != http.StatusFound {
		body, _ := io.ReadAll(response.Body)
		t.Fatalf("Console callback status = %d; body = %s", response.StatusCode, body)
	}
	redirect, err := url.Parse(response.Header.Get("Location"))
	if err != nil {
		t.Fatalf("parse MCP client callback: %v", err)
	}
	code := redirect.Query().Get("code")
	if code == "" {
		t.Fatalf("MCP client callback has no code: %s", redirect)
	}
	return code
}

func approveOAuthConsent(t *testing.T, serverURL, authorizeURL string) string {
	t.Helper()
	noRedirect := &http.Client{CheckRedirect: func(_ *http.Request, _ []*http.Request) error {
		return http.ErrUseLastResponse
	}}
	response, err := noRedirect.Get(authorizeURL)
	if err != nil {
		t.Fatalf("begin authorization: %v", err)
	}
	page, _ := io.ReadAll(response.Body)
	_ = response.Body.Close()
	if response.StatusCode != http.StatusOK {
		t.Fatalf("consent status = %d; body = %s", response.StatusCode, page)
	}
	if response.Header.Get("Location") != "" {
		t.Fatal("authorization redirected before consent")
	}
	if !strings.Contains(response.Header.Get("Content-Security-Policy"), "frame-ancestors 'none'") {
		t.Fatalf("consent CSP = %q", response.Header.Get("Content-Security-Policy"))
	}
	approved, err := noRedirect.PostForm(serverURL+"/oauth/authorize", url.Values{
		"request_id": {hiddenInput(t, string(page), "request_id")},
		"consent":    {hiddenInput(t, string(page), "consent")},
		"decision":   {"approve"},
	})
	if err != nil {
		t.Fatalf("approve authorization: %v", err)
	}
	defer func() { _ = approved.Body.Close() }()
	if approved.StatusCode != http.StatusFound {
		body, _ := io.ReadAll(approved.Body)
		t.Fatalf("approved authorization status = %d; body = %s", approved.StatusCode, body)
	}
	if !strings.Contains(approved.Header.Get("Location"), "/auth/signin") {
		t.Fatalf("approved authorization location = %s", approved.Header.Get("Location"))
	}
	return approved.Header.Get("Location")
}

func hiddenInput(t *testing.T, body, name string) string {
	t.Helper()
	needle := `name="` + name + `" value="`
	start := strings.Index(body, needle)
	if start < 0 {
		t.Fatalf("consent page missing %s", name)
	}
	rest := body[start+len(needle):]
	end := strings.Index(rest, `"`)
	if end <= 0 {
		t.Fatalf("consent page has empty %s", name)
	}
	return rest[:end]
}

func TestOAuthRedirectPolicyBlocksUntrustedHTTPSAndAllowsKnownMCPClients(t *testing.T) {
	bridge := newPolicyTestBridge(t, nil, nil)
	tests := []struct {
		uri string
		ok  bool
	}{
		{uri: "https://webhook.site/39354f08-2d8d-4101-ab74-45b4f7ac7490", ok: false},
		{uri: "https://claude.ai.evil.com/api/mcp/auth_callback", ok: false},
		{uri: "https://evilclaude.ai/api/mcp/auth_callback", ok: false},
		{uri: "https://claude.ai/api/mcp/auth_callback/extra", ok: false},
		{uri: "https://claude.ai/api/mcp/auth_callback/../../steal", ok: false},
		{uri: "https://chatgpt.com/other", ok: false},
		{uri: "https://127.0.0.1.evil.com/callback", ok: false},
		{uri: "https://user@claude.ai/api/mcp/auth_callback", ok: false},
		{uri: "https://claude.ai/api/mcp/auth_callback#frag", ok: false},
		{uri: "javascript:alert(1)", ok: false},
		{uri: "http://10.0.0.1/callback", ok: false},
		{uri: "https://claude.ai/api/mcp/auth_callback", ok: true},
		{uri: "https://claude.com/api/mcp/auth_callback", ok: true},
		{uri: "https://chatgpt.com/connector_platform_oauth_redirect", ok: true},
		{uri: "https://chatgpt.com/connector/oauth/cb_123", ok: true},
		{uri: "https://www.cursor.com/agents/mcp/oauth/callback", ok: true},
		{uri: "https://platform.openai.com/apps-manage/oauth", ok: true},
		{uri: "http://127.0.0.1:8765/callback", ok: true},
		{uri: "http://localhost:53280/callback", ok: true},
		{uri: "http://[::1]:8080/callback", ok: true},
	}
	for _, test := range tests {
		err := bridge.validateClientRedirectURI(test.uri)
		if test.ok && err != nil {
			t.Fatalf("redirect %s rejected: %v", test.uri, err)
		}
		if !test.ok && err == nil {
			t.Fatalf("redirect %s was allowed", test.uri)
		}
	}

	custom := newPolicyTestBridge(t, []string{"customer.example"}, []string{"https://partner.example/oauth/callback"})
	if err := custom.validateClientRedirectURI("https://customer.example/any/path"); err != nil {
		t.Fatalf("operator host allowlist rejected a redirect: %v", err)
	}
	if err := custom.validateClientRedirectURI("https://partner.example/oauth/callback"); err != nil {
		t.Fatalf("operator URI allowlist rejected a redirect: %v", err)
	}
	if err := custom.validateClientRedirectURI("https://partner.example/oauth/callback/extra"); err == nil {
		t.Fatal("exact URI allowlist matched a longer path")
	}
}

func TestOAuthConsentShowsDestinationAndDenialDoesNotIssueCode(t *testing.T) {
	handler := newTestOAuthBridgeHandler(t, "https://api.pipeops.test")
	server := httptest.NewServer(handler)
	defer server.Close()

	registration := postJSON(t, server.URL+"/oauth/register", `{
		"redirect_uris":["https://claude.ai/api/mcp/auth_callback"],
		"token_endpoint_auth_method":"none",
		"client_name":"Claude <script>"
	}`)
	if registration.StatusCode != http.StatusCreated {
		t.Fatalf("register named client status = %d; body = %s", registration.StatusCode, registration.Body)
	}
	var named struct {
		ClientID string `json:"client_id"`
	}
	if err := json.Unmarshal([]byte(registration.Body), &named); err != nil || named.ClientID == "" {
		t.Fatalf("decode named client: %v", err)
	}
	clientID := named.ClientID

	noRedirect := &http.Client{CheckRedirect: func(_ *http.Request, _ []*http.Request) error {
		return http.ErrUseLastResponse
	}}
	authorizeURL := server.URL + "/oauth/authorize?" + url.Values{
		"client_id":             {clientID},
		"redirect_uri":          {"https://claude.ai/api/mcp/auth_callback"},
		"response_type":         {"code"},
		"scope":                 {"api:read api:write"},
		"code_challenge":        {strings.Repeat("a", 43)},
		"code_challenge_method": {"S256"},
		"state":                 {"consent-state"},
	}.Encode()
	response, err := noRedirect.Get(authorizeURL)
	if err != nil {
		t.Fatalf("open consent page: %v", err)
	}
	page, _ := io.ReadAll(response.Body)
	_ = response.Body.Close()
	body := string(page)
	if response.StatusCode != http.StatusOK || !strings.Contains(body, "claude.ai") || !strings.Contains(body, "Claude &lt;script&gt;") {
		t.Fatalf("consent page status = %d; body = %s", response.StatusCode, body)
	}
	if strings.Contains(body, "<script>") {
		t.Fatal("consent page rendered client_name as HTML")
	}
	denied, err := noRedirect.PostForm(server.URL+"/oauth/authorize", url.Values{
		"request_id": {hiddenInput(t, body, "request_id")},
		"consent":    {hiddenInput(t, body, "consent")},
		"decision":   {"deny"},
	})
	if err != nil {
		t.Fatalf("deny authorization: %v", err)
	}
	defer func() { _ = denied.Body.Close() }()
	location := denied.Header.Get("Location")
	if denied.StatusCode != http.StatusFound || !strings.Contains(location, "error=access_denied") || strings.Contains(location, "code=") {
		t.Fatalf("denied authorization status = %d; location = %s", denied.StatusCode, location)
	}
	replay, err := noRedirect.PostForm(server.URL+"/oauth/authorize", url.Values{
		"request_id": {hiddenInput(t, body, "request_id")},
		"consent":    {hiddenInput(t, body, "consent")},
		"decision":   {"approve"},
	})
	if err != nil {
		t.Fatalf("replay consent: %v", err)
	}
	defer func() { _ = replay.Body.Close() }()
	if replay.StatusCode != http.StatusBadRequest || replay.Header.Get("Location") != "" {
		t.Fatalf("replayed consent status = %d; location = %s", replay.StatusCode, replay.Header.Get("Location"))
	}
}

func TestOAuthBridgeRejectsStoredUntrustedClient(t *testing.T) {
	api := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/oauth/token":
			_ = json.NewEncoder(w).Encode(map[string]any{"access_token": "console_access", "refresh_token": "console_refresh", "expires_in": 3600})
		case "/profile/data":
			_, _ = io.WriteString(w, `{"success":true,"message":"ok","data":{"user":{"uuid":"user-1"}}}`)
		default:
			http.Error(w, "unauthorized", http.StatusUnauthorized)
		}
	}))
	defer api.Close()

	store := newMemoryOAuthStore()
	handler := newTestOAuthBridgeHandlerWithStore(t, api.URL, store)
	server := httptest.NewServer(handler)
	defer server.Close()

	clientID := registerOAuthClient(t, server.URL, "http://127.0.0.1:8765/callback")
	var client oauthClientRecord
	if err := getOAuthJSON(context.Background(), store, "client:"+clientID, &client); err != nil {
		t.Fatalf("load registered client: %v", err)
	}
	client.RedirectURIs = []string{"https://webhook.site/steal"}
	client.ClientName = "Malicious Client"
	if err := putOAuthJSON(context.Background(), store, "client:"+clientID, &client, time.Hour); err != nil {
		t.Fatalf("store untrusted client: %v", err)
	}

	noRedirect := &http.Client{CheckRedirect: func(_ *http.Request, _ []*http.Request) error {
		return http.ErrUseLastResponse
	}}
	response, err := noRedirect.Get(server.URL + "/oauth/authorize?" + url.Values{
		"client_id":             {clientID},
		"redirect_uri":          {"https://webhook.site/steal"},
		"response_type":         {"code"},
		"code_challenge":        {strings.Repeat("a", 43)},
		"code_challenge_method": {"S256"},
		"state":                 {"stolen_token_proof"},
	}.Encode())
	if err != nil {
		t.Fatalf("authorize stored untrusted client: %v", err)
	}
	body, _ := io.ReadAll(response.Body)
	_ = response.Body.Close()
	if response.StatusCode != http.StatusBadRequest || response.Header.Get("Location") != "" || strings.Contains(string(body), "webhook.site") {
		t.Fatalf("stored untrusted authorize status = %d; location = %s; body = %s", response.StatusCode, response.Header.Get("Location"), body)
	}

	trustedID := registerOAuthClient(t, server.URL, "http://127.0.0.1:8765/callback")
	verifier := strings.Repeat("v", 64)
	digest := sha256.Sum256([]byte(verifier))
	location := approveOAuthConsent(t, server.URL, server.URL+"/oauth/authorize?"+url.Values{
		"client_id":             {trustedID},
		"redirect_uri":          {"http://127.0.0.1:8765/callback"},
		"response_type":         {"code"},
		"scope":                 {"api:read"},
		"resource":              {"https://mcp.pipeops.test/mcp"},
		"state":                 {"state-123"},
		"code_challenge":        {base64.RawURLEncoding.EncodeToString(digest[:])},
		"code_challenge_method": {"S256"},
	}.Encode())
	code := completeConsoleAuthorization(t, server.URL, location, "console-code")
	tokens := exchangeOAuthCode(t, server.URL, trustedID, code, verifier)

	if err := getOAuthJSON(context.Background(), store, "client:"+trustedID, &client); err != nil {
		t.Fatalf("load trusted client: %v", err)
	}
	client.RedirectURIs = []string{"https://webhook.site/steal"}
	if err := putOAuthJSON(context.Background(), store, "client:"+trustedID, &client, time.Hour); err != nil {
		t.Fatalf("replace trusted redirects: %v", err)
	}
	revoked := postOAuthForm(t, server.URL+"/oauth/token", url.Values{
		"grant_type":    {"refresh_token"},
		"client_id":     {trustedID},
		"refresh_token": {tokens.RefreshToken},
	})
	if revoked.StatusCode != http.StatusBadRequest || !strings.Contains(revoked.Body, "no longer allowed") {
		t.Fatalf("refresh for untrusted client = %#v", revoked)
	}
	mcpRequest, err := http.NewRequest(http.MethodPost, server.URL+"/mcp", strings.NewReader(`{"jsonrpc":"2.0","id":1,"method":"initialize"}`))
	if err != nil {
		t.Fatalf("build mcp request: %v", err)
	}
	mcpRequest.Header.Set("Authorization", "Bearer "+tokens.AccessToken)
	mcpRequest.Header.Set("Content-Type", "application/json")
	mcpRequest.Header.Set("Accept", "application/json, text/event-stream")
	mcpResponse, err := http.DefaultClient.Do(mcpRequest)
	if err != nil {
		t.Fatalf("call mcp with untrusted client token: %v", err)
	}
	defer func() { _ = mcpResponse.Body.Close() }()
	if mcpResponse.StatusCode != http.StatusUnauthorized {
		t.Fatalf("untrusted client access status = %d", mcpResponse.StatusCode)
	}
}

func newPolicyTestBridge(t *testing.T, hosts, uris []string) *oauthBridge {
	t.Helper()
	key := base64.StdEncoding.EncodeToString([]byte("0123456789abcdef0123456789abcdef"))
	cipher, err := newCredentialCipher(key)
	if err != nil {
		t.Fatalf("create cipher: %v", err)
	}
	bridge, err := newOAuthBridge(oauthBridgeConfig{
		Issuer:             "https://mcp.pipeops.test",
		ResourceURL:        "https://mcp.pipeops.test/mcp",
		BaseURL:            "https://api.pipeops.test",
		ConsoleURL:         "https://console.pipeops.test",
		ConsoleClientID:    "pipeops_public_client",
		ConsoleScopes:      []string{"openid"},
		Scopes:             []string{"api:read", "api:write"},
		Store:              newMemoryOAuthStore(),
		Cipher:             cipher,
		RedirectAllowHosts: hosts,
		RedirectAllowURIs:  uris,
	})
	if err != nil {
		t.Fatalf("newOAuthBridge: %v", err)
	}
	return bridge
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

package mcp

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"crypto/subtle"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"html/template"
	"io"
	"net"
	"net/http"
	"net/url"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/modelcontextprotocol/go-sdk/auth"
	"github.com/modelcontextprotocol/go-sdk/oauthex"
)

const (
	oauthAuthorizationRequestTTL = 10 * time.Minute
	oauthAuthorizationCodeTTL    = 5 * time.Minute
	oauthAccessTokenTTL          = 15 * time.Minute
	oauthRefreshTokenTTL         = 30 * 24 * time.Hour
)

var (
	errNotBridgeAccessToken = errors.New("not an OAuth bridge access token")
	errOAuthInvalidClient   = errors.New("invalid OAuth client")
)

type oauthBridgeConfig struct {
	Issuer      string
	ResourceURL string
	BaseURL     string
	Scopes      []string
	Store       oauthStore
	Cipher      *credentialCipher
	Now         func() time.Time
}

type oauthBridge struct {
	issuer      string
	resourceURL string
	baseURL     string
	scopes      []string
	scopeSet    map[string]struct{}
	store       oauthStore
	cipher      *credentialCipher
	now         func() time.Time
	limiter     *oauthRateLimiter
	consentPage *template.Template
}

type oauthClientRecord struct {
	ClientID                string   `json:"client_id"`
	ClientSecretHash        string   `json:"client_secret_hash,omitempty"`
	RedirectURIs            []string `json:"redirect_uris"`
	TokenEndpointAuthMethod string   `json:"token_endpoint_auth_method"`
	GrantTypes              []string `json:"grant_types"`
	ResponseTypes           []string `json:"response_types"`
	ClientName              string   `json:"client_name"`
	Scope                   string   `json:"scope,omitempty"`
	CreatedAt               int64    `json:"created_at"`
}

type oauthAuthorizationRequest struct {
	ID            string   `json:"id"`
	ClientID      string   `json:"client_id"`
	ClientName    string   `json:"client_name"`
	RedirectURI   string   `json:"redirect_uri"`
	State         string   `json:"state,omitempty"`
	Scopes        []string `json:"scopes"`
	Resource      string   `json:"resource"`
	CodeChallenge string   `json:"code_challenge"`
}

type oauthAuthorizationCode struct {
	ClientID            string   `json:"client_id"`
	RedirectURI         string   `json:"redirect_uri"`
	Scopes              []string `json:"scopes"`
	Resource            string   `json:"resource"`
	CodeChallenge       string   `json:"code_challenge"`
	EncryptedCredential string   `json:"encrypted_credential"`
	UserID              string   `json:"user_id"`
}

type oauthGrantRecord struct {
	ClientID            string   `json:"client_id"`
	FamilyID            string   `json:"family_id"`
	Scopes              []string `json:"scopes"`
	Resource            string   `json:"resource"`
	EncryptedCredential string   `json:"encrypted_credential"`
	UserID              string   `json:"user_id"`
}

type verifiedCredential struct {
	UpstreamToken string
	Scopes        []string
	UserID        string
}

type oauthRateWindow struct {
	started   time.Time
	expiresAt time.Time
	count     int
}

type oauthRateLimiter struct {
	mu      sync.Mutex
	windows map[string]oauthRateWindow
	now     func() time.Time
}

func newOAuthBridge(config oauthBridgeConfig) (*oauthBridge, error) {
	if config.Store == nil {
		return nil, errors.New("OAuth bridge store is required")
	}
	if config.Cipher == nil {
		return nil, errors.New("OAuth bridge credential cipher is required")
	}
	issuer, err := validatePublicURL("OAuth issuer", strings.TrimRight(config.Issuer, "/"))
	if err != nil {
		return nil, err
	}
	if (issuer.Path != "" && issuer.Path != "/") || issuer.RawQuery != "" {
		return nil, errors.New("OAuth bridge issuer must not contain a path or query string")
	}
	if _, err := validatePublicURL("OAuth resource URL", config.ResourceURL); err != nil {
		return nil, err
	}
	if config.Now == nil {
		config.Now = time.Now
	}
	scopes := normalizedScopes(config.Scopes)
	if len(scopes) == 0 {
		return nil, errors.New("OAuth bridge must advertise at least one scope")
	}
	scopeSet := make(map[string]struct{}, len(scopes))
	for _, scope := range scopes {
		scopeSet[scope] = struct{}{}
	}
	page, err := template.New("consent").Parse(oauthConsentPage)
	if err != nil {
		return nil, fmt.Errorf("parse OAuth consent page: %w", err)
	}
	return &oauthBridge{
		issuer:      strings.TrimRight(config.Issuer, "/"),
		resourceURL: config.ResourceURL,
		baseURL:     config.BaseURL,
		scopes:      scopes,
		scopeSet:    scopeSet,
		store:       config.Store,
		cipher:      config.Cipher,
		now:         config.Now,
		limiter: &oauthRateLimiter{
			windows: make(map[string]oauthRateWindow),
			now:     config.Now,
		},
		consentPage: page,
	}, nil
}

func (b *oauthBridge) registerRoutes(mux *http.ServeMux) {
	mux.HandleFunc("/.well-known/oauth-authorization-server", b.handleAuthorizationServerMetadata)
	mux.HandleFunc("/.well-known/openid-configuration", b.handleAuthorizationServerMetadata)
	mux.HandleFunc("/oauth/register", b.handleRegistration)
	mux.HandleFunc("/oauth/authorize", b.handleAuthorize)
	mux.HandleFunc("/oauth/token", b.handleToken)
	mux.HandleFunc("/oauth/revoke", b.handleRevocation)
	mux.HandleFunc("/oauth/jwks", b.handleJWKS)
}

func (b *oauthBridge) handleAuthorizationServerMetadata(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		w.Header().Set("Allow", http.MethodGet)
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	b.writeJSON(w, http.StatusOK, &oauthex.AuthServerMeta{
		Issuer:                                 b.issuer,
		AuthorizationEndpoint:                  b.issuer + "/oauth/authorize",
		TokenEndpoint:                          b.issuer + "/oauth/token",
		JWKSURI:                                b.issuer + "/oauth/jwks",
		RegistrationEndpoint:                   b.issuer + "/oauth/register",
		ScopesSupported:                        b.scopes,
		ResponseTypesSupported:                 []string{"code"},
		GrantTypesSupported:                    []string{"authorization_code", "refresh_token"},
		TokenEndpointAuthMethodsSupported:      []string{"none", "client_secret_basic", "client_secret_post"},
		RevocationEndpoint:                     b.issuer + "/oauth/revoke",
		RevocationEndpointAuthMethodsSupported: []string{"none", "client_secret_basic", "client_secret_post"},
		CodeChallengeMethodsSupported:          []string{"S256"},
		ServiceDocumentation:                   "https://docs.pipeops.io/docs/integrations/pipeops-mcp",
	})
}

func (b *oauthBridge) handleJWKS(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		w.Header().Set("Allow", http.MethodGet)
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	b.writeJSON(w, http.StatusOK, map[string]any{"keys": []any{}})
}

func (b *oauthBridge) handleRegistration(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		w.Header().Set("Allow", http.MethodPost)
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	if !b.allow(r, "register", 20, time.Minute) {
		b.writeRegistrationError(w, "invalid_client_metadata", "registration rate limit exceeded")
		return
	}
	r.Body = http.MaxBytesReader(w, r.Body, 32<<10)
	decoder := json.NewDecoder(r.Body)
	var metadata oauthex.ClientRegistrationMetadata
	if err := decoder.Decode(&metadata); err != nil {
		b.writeRegistrationError(w, "invalid_client_metadata", "invalid client registration payload")
		return
	}
	if err := decoder.Decode(&struct{}{}); !errors.Is(err, io.EOF) {
		b.writeRegistrationError(w, "invalid_client_metadata", "invalid client registration payload")
		return
	}
	client, secret, err := b.validateAndCreateClient(&metadata)
	if err != nil {
		b.writeRegistrationError(w, "invalid_client_metadata", err.Error())
		return
	}
	if err := putOAuthJSON(r.Context(), b.store, "client:"+client.ClientID, client, 365*24*time.Hour); err != nil {
		b.writeJSON(w, http.StatusServiceUnavailable, map[string]string{"error": "server_error", "error_description": "client registration unavailable"})
		return
	}
	response := &oauthex.ClientRegistrationResponse{
		ClientRegistrationMetadata: oauthex.ClientRegistrationMetadata{
			RedirectURIs:            client.RedirectURIs,
			TokenEndpointAuthMethod: client.TokenEndpointAuthMethod,
			GrantTypes:              client.GrantTypes,
			ResponseTypes:           client.ResponseTypes,
			ClientName:              client.ClientName,
			Scope:                   client.Scope,
		},
		ClientID:         client.ClientID,
		ClientSecret:     secret,
		ClientIDIssuedAt: time.Unix(client.CreatedAt, 0),
	}
	w.Header().Set("Cache-Control", "no-store")
	w.Header().Set("Pragma", "no-cache")
	b.writeJSON(w, http.StatusCreated, response)
}

func (b *oauthBridge) validateAndCreateClient(metadata *oauthex.ClientRegistrationMetadata) (*oauthClientRecord, string, error) {
	if len(metadata.RedirectURIs) == 0 || len(metadata.RedirectURIs) > 10 {
		return nil, "", errors.New("between one and ten redirect_uris are required")
	}
	redirects := make([]string, 0, len(metadata.RedirectURIs))
	seen := make(map[string]struct{}, len(metadata.RedirectURIs))
	for _, redirect := range metadata.RedirectURIs {
		if len(redirect) > 2048 {
			return nil, "", errors.New("redirect_uri is too long")
		}
		if err := validateOAuthRedirectURI(redirect); err != nil {
			return nil, "", err
		}
		if _, ok := seen[redirect]; !ok {
			seen[redirect] = struct{}{}
			redirects = append(redirects, redirect)
		}
	}
	method := metadata.TokenEndpointAuthMethod
	if method == "" {
		method = "none"
	}
	switch method {
	case "none", "client_secret_basic", "client_secret_post":
	default:
		return nil, "", errors.New("unsupported token_endpoint_auth_method")
	}
	grantTypes := metadata.GrantTypes
	if len(grantTypes) == 0 {
		grantTypes = []string{"authorization_code", "refresh_token"}
	}
	for _, grantType := range grantTypes {
		if grantType != "authorization_code" && grantType != "refresh_token" {
			return nil, "", errors.New("unsupported grant_type")
		}
	}
	responseTypes := metadata.ResponseTypes
	if len(responseTypes) == 0 {
		responseTypes = []string{"code"}
	}
	if len(responseTypes) != 1 || responseTypes[0] != "code" {
		return nil, "", errors.New("only the code response_type is supported")
	}
	clientName := strings.TrimSpace(metadata.ClientName)
	if len(clientName) > 128 || strings.IndexFunc(clientName, func(r rune) bool { return r < 0x20 || r == 0x7f }) >= 0 {
		return nil, "", errors.New("client_name is invalid")
	}
	registeredScopes, err := b.validateScopes(strings.Fields(metadata.Scope), true)
	if err != nil {
		return nil, "", err
	}
	clientID, err := randomOAuthValue("pomcp_", 24)
	if err != nil {
		return nil, "", err
	}
	client := &oauthClientRecord{
		ClientID:                clientID,
		RedirectURIs:            redirects,
		TokenEndpointAuthMethod: method,
		GrantTypes:              grantTypes,
		ResponseTypes:           responseTypes,
		ClientName:              clientName,
		Scope:                   strings.Join(registeredScopes, " "),
		CreatedAt:               b.now().Unix(),
	}
	if client.ClientName == "" {
		client.ClientName = "AI assistant"
	}
	var secret string
	if method != "none" {
		secret, err = randomOAuthValue("pomcps_", 32)
		if err != nil {
			return nil, "", err
		}
		client.ClientSecretHash = hashOAuthSecret(secret)
	}
	return client, secret, nil
}

func (b *oauthBridge) handleAuthorize(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		b.beginAuthorization(w, r)
	case http.MethodPost:
		b.completeAuthorization(w, r)
	default:
		w.Header().Set("Allow", http.MethodGet+", "+http.MethodPost)
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
	}
}

func (b *oauthBridge) beginAuthorization(w http.ResponseWriter, r *http.Request) {
	if !b.allow(r, "authorize", 30, time.Minute) {
		http.Error(w, "authorization rate limit exceeded", http.StatusTooManyRequests)
		return
	}
	query := r.URL.Query()
	client, err := b.loadClient(r.Context(), query.Get("client_id"))
	if err != nil {
		if errors.Is(err, errOAuthRecordNotFound) {
			http.Error(w, "invalid OAuth client", http.StatusBadRequest)
		} else {
			http.Error(w, "authorization unavailable", http.StatusServiceUnavailable)
		}
		return
	}
	redirectURI := query.Get("redirect_uri")
	if !containsString(client.RedirectURIs, redirectURI) {
		http.Error(w, "invalid redirect_uri", http.StatusBadRequest)
		return
	}
	if query.Get("response_type") != "code" {
		b.redirectAuthorizationError(w, r, redirectURI, query.Get("state"), "unsupported_response_type", "only authorization code is supported")
		return
	}
	challenge := query.Get("code_challenge")
	if query.Get("code_challenge_method") != "S256" || len(challenge) < 43 || len(challenge) > 128 {
		b.redirectAuthorizationError(w, r, redirectURI, query.Get("state"), "invalid_request", "PKCE S256 is required")
		return
	}
	resource := query.Get("resource")
	if resource == "" {
		resource = b.resourceURL
	}
	if resource != b.resourceURL {
		b.redirectAuthorizationError(w, r, redirectURI, query.Get("state"), "invalid_target", "invalid resource")
		return
	}
	scopes, err := b.validateScopes(strings.Fields(query.Get("scope")), false)
	if err != nil {
		b.redirectAuthorizationError(w, r, redirectURI, query.Get("state"), "invalid_scope", "unsupported scope requested")
		return
	}
	// Some MCP clients use registration scope as an initial/default value, then
	// request the complete protected-resource scope set during authorization.
	// Open DCR is not a meaningful permission boundary because a client can
	// immediately register again with broader scopes. Enforce the server's
	// supported scopes here and show the exact requested set on the consent page.
	requestID, err := randomOAuthValue("poar_", 24)
	if err != nil {
		http.Error(w, "authorization unavailable", http.StatusServiceUnavailable)
		return
	}
	authorization := oauthAuthorizationRequest{
		ID:            requestID,
		ClientID:      client.ClientID,
		ClientName:    client.ClientName,
		RedirectURI:   redirectURI,
		State:         query.Get("state"),
		Scopes:        scopes,
		Resource:      resource,
		CodeChallenge: challenge,
	}
	if err := putOAuthJSON(r.Context(), b.store, oauthLookupKey("request", requestID), &authorization, oauthAuthorizationRequestTTL); err != nil {
		http.Error(w, "authorization unavailable", http.StatusServiceUnavailable)
		return
	}
	b.renderConsentPage(w, authorization, "")
}

func (b *oauthBridge) completeAuthorization(w http.ResponseWriter, r *http.Request) {
	if !b.allow(r, "authorize-submit", 20, time.Minute) {
		http.Error(w, "authorization rate limit exceeded", http.StatusTooManyRequests)
		return
	}
	r.Body = http.MaxBytesReader(w, r.Body, 16<<10)
	if err := r.ParseForm(); err != nil {
		http.Error(w, "invalid authorization request", http.StatusBadRequest)
		return
	}
	requestID := r.PostForm.Get("request_id")
	var authorization oauthAuthorizationRequest
	if err := getOAuthJSON(r.Context(), b.store, oauthLookupKey("request", requestID), &authorization); err != nil {
		if errors.Is(err, errOAuthRecordNotFound) {
			http.Error(w, "authorization request expired", http.StatusBadRequest)
		} else {
			http.Error(w, "authorization unavailable", http.StatusServiceUnavailable)
		}
		return
	}
	if r.PostForm.Get("action") == "deny" {
		_, _ = b.store.Consume(r.Context(), oauthLookupKey("request", requestID))
		b.redirectAuthorizationError(w, r, authorization.RedirectURI, authorization.State, "access_denied", "access was denied")
		return
	}
	serviceToken := strings.TrimSpace(r.PostForm.Get("service_token"))
	if !strings.HasPrefix(serviceToken, "sat_") || strings.ContainsAny(serviceToken, " \t\r\n") {
		b.renderConsentPage(w, authorization, "Enter a valid PipeOps workspace service token.")
		return
	}
	info, err := controllerTokenVerifier(b.baseURL)(r.Context(), serviceToken, r)
	if err != nil || info == nil || info.UserID == "" {
		b.renderConsentPage(w, authorization, "The service token could not be authorized. Use an active MCP token with api:read access.")
		return
	}
	consumed, err := b.store.Consume(r.Context(), oauthLookupKey("request", requestID))
	if err != nil {
		if errors.Is(err, errOAuthRecordNotFound) {
			http.Error(w, "authorization request already used", http.StatusBadRequest)
		} else {
			http.Error(w, "authorization unavailable", http.StatusServiceUnavailable)
		}
		return
	}
	if err := json.Unmarshal(consumed, &authorization); err != nil {
		http.Error(w, "authorization unavailable", http.StatusServiceUnavailable)
		return
	}
	aad := oauthGrantAdditionalData(authorization.ClientID, authorization.Scopes, authorization.Resource, info.UserID)
	encrypted, err := b.cipher.Seal(serviceToken, aad)
	if err != nil {
		http.Error(w, "authorization unavailable", http.StatusServiceUnavailable)
		return
	}
	code, err := randomOAuthValue("poac_", 32)
	if err != nil {
		http.Error(w, "authorization unavailable", http.StatusServiceUnavailable)
		return
	}
	record := oauthAuthorizationCode{
		ClientID:            authorization.ClientID,
		RedirectURI:         authorization.RedirectURI,
		Scopes:              authorization.Scopes,
		Resource:            authorization.Resource,
		CodeChallenge:       authorization.CodeChallenge,
		EncryptedCredential: encrypted,
		UserID:              info.UserID,
	}
	if err := putOAuthJSON(r.Context(), b.store, oauthLookupKey("code", code), &record, oauthAuthorizationCodeTTL); err != nil {
		http.Error(w, "authorization unavailable", http.StatusServiceUnavailable)
		return
	}
	redirect, _ := url.Parse(authorization.RedirectURI)
	values := redirect.Query()
	values.Set("code", code)
	if authorization.State != "" {
		values.Set("state", authorization.State)
	}
	redirect.RawQuery = values.Encode()
	http.Redirect(w, r, redirect.String(), http.StatusFound)
}

func (b *oauthBridge) renderConsentPage(w http.ResponseWriter, authorization oauthAuthorizationRequest, errorMessage string) {
	scriptNonce, err := randomOAuthValue("", 18)
	if err != nil {
		http.Error(w, "authorization unavailable", http.StatusServiceUnavailable)
		return
	}
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.Header().Set("Cache-Control", "no-store")
	w.Header().Set("Content-Security-Policy", "default-src 'none'; style-src 'unsafe-inline'; script-src 'nonce-"+scriptNonce+"'; form-action 'self'; base-uri 'none'; frame-ancestors 'none'")
	w.Header().Set("X-Frame-Options", "DENY")
	data := struct {
		RequestID    string
		ClientName   string
		RedirectHost string
		Scopes       []string
		ErrorMessage string
		ScriptNonce  string
	}{
		RequestID:    authorization.ID,
		ClientName:   authorization.ClientName,
		RedirectHost: authorizationRedirectHost(authorization.RedirectURI),
		Scopes:       authorization.Scopes,
		ErrorMessage: errorMessage,
		ScriptNonce:  scriptNonce,
	}
	if err := b.consentPage.Execute(w, data); err != nil {
		http.Error(w, "authorization unavailable", http.StatusServiceUnavailable)
	}
}

func (b *oauthBridge) handleToken(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		w.Header().Set("Allow", http.MethodPost)
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	if !b.allow(r, "token", 60, time.Minute) {
		b.writeTokenError(w, http.StatusTooManyRequests, "temporarily_unavailable", "token rate limit exceeded")
		return
	}
	r.Body = http.MaxBytesReader(w, r.Body, 16<<10)
	if err := r.ParseForm(); err != nil {
		b.writeTokenError(w, http.StatusBadRequest, "invalid_request", "invalid token request")
		return
	}
	client, err := b.authenticateClient(r)
	if err != nil {
		if errors.Is(err, errOAuthInvalidClient) {
			b.writeTokenError(w, http.StatusUnauthorized, "invalid_client", "invalid OAuth client")
		} else {
			b.writeTokenError(w, http.StatusServiceUnavailable, "temporarily_unavailable", "token service unavailable")
		}
		return
	}
	switch r.PostForm.Get("grant_type") {
	case "authorization_code":
		b.exchangeAuthorizationCode(w, r, client)
	case "refresh_token":
		b.exchangeRefreshToken(w, r, client)
	default:
		b.writeTokenError(w, http.StatusBadRequest, "unsupported_grant_type", "unsupported grant type")
	}
}

func (b *oauthBridge) exchangeAuthorizationCode(w http.ResponseWriter, r *http.Request, client *oauthClientRecord) {
	code := r.PostForm.Get("code")
	var record oauthAuthorizationCode
	if err := getOAuthJSON(r.Context(), b.store, oauthLookupKey("code", code), &record); err != nil {
		if errors.Is(err, errOAuthRecordNotFound) {
			b.writeTokenError(w, http.StatusBadRequest, "invalid_grant", "authorization code is invalid or expired")
		} else {
			b.writeTokenError(w, http.StatusServiceUnavailable, "temporarily_unavailable", "token service unavailable")
		}
		return
	}
	if record.ClientID != client.ClientID || record.RedirectURI != r.PostForm.Get("redirect_uri") {
		b.writeTokenError(w, http.StatusBadRequest, "invalid_grant", "authorization code does not match this client")
		return
	}
	if resource := r.PostForm.Get("resource"); resource != "" && resource != record.Resource {
		b.writeTokenError(w, http.StatusBadRequest, "invalid_target", "authorization code does not match this resource")
		return
	}
	verifier := r.PostForm.Get("code_verifier")
	if !validPKCEVerifier(verifier, record.CodeChallenge) {
		b.writeTokenError(w, http.StatusBadRequest, "invalid_grant", "invalid PKCE verifier")
		return
	}
	if _, err := b.store.Consume(r.Context(), oauthLookupKey("code", code)); err != nil {
		if errors.Is(err, errOAuthRecordNotFound) {
			b.writeTokenError(w, http.StatusBadRequest, "invalid_grant", "authorization code was already used")
		} else {
			b.writeTokenError(w, http.StatusServiceUnavailable, "temporarily_unavailable", "token service unavailable")
		}
		return
	}
	b.issueInitialTokens(w, r, oauthGrantRecord{
		ClientID:            record.ClientID,
		Scopes:              record.Scopes,
		Resource:            record.Resource,
		EncryptedCredential: record.EncryptedCredential,
		UserID:              record.UserID,
	})
}

func (b *oauthBridge) exchangeRefreshToken(w http.ResponseWriter, r *http.Request, client *oauthClientRecord) {
	refreshToken := r.PostForm.Get("refresh_token")
	var record oauthGrantRecord
	if err := getOAuthJSON(r.Context(), b.store, oauthLookupKey("refresh", refreshToken), &record); err != nil {
		if !errors.Is(err, errOAuthRecordNotFound) {
			b.writeTokenError(w, http.StatusServiceUnavailable, "temporarily_unavailable", "token service unavailable")
			return
		}
		if err := b.revokeReusedRefreshToken(r.Context(), refreshToken, client); err != nil {
			b.writeTokenError(w, http.StatusServiceUnavailable, "temporarily_unavailable", "token service unavailable")
			return
		}
		b.writeTokenError(w, http.StatusBadRequest, "invalid_grant", "refresh token is invalid or expired")
		return
	}
	if record.ClientID != client.ClientID {
		b.writeTokenError(w, http.StatusBadRequest, "invalid_grant", "refresh token does not match this client")
		return
	}
	if resource := r.PostForm.Get("resource"); resource != "" && resource != record.Resource {
		b.writeTokenError(w, http.StatusBadRequest, "invalid_target", "refresh token does not match this resource")
		return
	}
	if record.FamilyID == "" {
		b.writeTokenError(w, http.StatusBadRequest, "invalid_grant", "refresh token is invalid or expired")
		return
	}
	upstreamToken, err := b.openGrantCredential(record)
	if err != nil {
		b.writeTokenError(w, http.StatusBadRequest, "invalid_grant", "refresh token is invalid or expired")
		return
	}
	previousRecord := record
	requested := strings.Fields(r.PostForm.Get("scope"))
	if len(requested) > 0 {
		scopes, err := b.validateScopes(requested, false)
		if err != nil || !scopeSubset(scopes, record.Scopes) {
			b.writeTokenError(w, http.StatusBadRequest, "invalid_scope", "refresh scope exceeds the original grant")
			return
		}
		record.Scopes = scopes
	}
	record.EncryptedCredential, err = b.cipher.Seal(upstreamToken, oauthGrantAdditionalData(record.ClientID, record.Scopes, record.Resource, record.UserID))
	if err != nil {
		b.writeTokenError(w, http.StatusServiceUnavailable, "temporarily_unavailable", "token service unavailable")
		return
	}
	b.issueRefreshedTokens(w, r, refreshToken, previousRecord, record, client)
}

func (b *oauthBridge) issueInitialTokens(w http.ResponseWriter, r *http.Request, record oauthGrantRecord) {
	familyID, err := randomOAuthValue("porf_", 24)
	if err != nil {
		b.writeTokenError(w, http.StatusServiceUnavailable, "temporarily_unavailable", "token service unavailable")
		return
	}
	record.FamilyID = familyID
	accessToken, refreshToken, err := newOAuthTokenPair()
	if err != nil {
		b.writeTokenError(w, http.StatusServiceUnavailable, "temporarily_unavailable", "token service unavailable")
		return
	}
	accessKey := oauthLookupKey("access", accessToken)
	refreshKey := oauthLookupKey("refresh", refreshToken)
	if err := putOAuthJSON(r.Context(), b.store, accessKey, &record, oauthAccessTokenTTL); err != nil {
		b.writeTokenError(w, http.StatusServiceUnavailable, "temporarily_unavailable", "token service unavailable")
		return
	}
	if err := putOAuthJSON(r.Context(), b.store, refreshKey, &record, oauthRefreshTokenTTL); err != nil {
		_ = b.store.Delete(r.Context(), accessKey)
		b.writeTokenError(w, http.StatusServiceUnavailable, "temporarily_unavailable", "token service unavailable")
		return
	}
	if err := b.store.Put(r.Context(), oauthRefreshFamilyKey(familyID), []byte(refreshKey), oauthRefreshTokenTTL); err != nil {
		_ = b.store.Delete(r.Context(), accessKey)
		_ = b.store.Delete(r.Context(), refreshKey)
		b.writeTokenError(w, http.StatusServiceUnavailable, "temporarily_unavailable", "token service unavailable")
		return
	}
	b.writeTokenResponse(w, accessToken, refreshToken, record.Scopes)
}

func (b *oauthBridge) issueRefreshedTokens(
	w http.ResponseWriter,
	r *http.Request,
	oldRefreshToken string,
	previousRecord, record oauthGrantRecord,
	client *oauthClientRecord,
) {
	accessToken, refreshToken, err := newOAuthTokenPair()
	if err != nil {
		b.writeTokenError(w, http.StatusServiceUnavailable, "temporarily_unavailable", "token service unavailable")
		return
	}
	accessKey := oauthLookupKey("access", accessToken)
	oldRefreshKey := oauthLookupKey("refresh", oldRefreshToken)
	usedRefreshKey := oauthLookupKey("refresh-used", oldRefreshToken)
	newRefreshKey := oauthLookupKey("refresh", refreshToken)
	if err := putOAuthJSON(r.Context(), b.store, accessKey, &record, oauthAccessTokenTTL); err != nil {
		b.writeTokenError(w, http.StatusServiceUnavailable, "temporarily_unavailable", "token service unavailable")
		return
	}
	usedValue, err := json.Marshal(&previousRecord)
	if err != nil {
		_ = b.store.Delete(r.Context(), accessKey)
		b.writeTokenError(w, http.StatusServiceUnavailable, "temporarily_unavailable", "token service unavailable")
		return
	}
	newValue, err := json.Marshal(&record)
	if err != nil {
		_ = b.store.Delete(r.Context(), accessKey)
		b.writeTokenError(w, http.StatusServiceUnavailable, "temporarily_unavailable", "token service unavailable")
		return
	}
	rotated, err := b.store.RotateRefresh(
		r.Context(),
		oldRefreshKey,
		usedRefreshKey,
		oauthRefreshFamilyKey(record.FamilyID),
		newRefreshKey,
		usedValue,
		newValue,
		oauthRefreshTokenTTL,
	)
	if err != nil {
		_ = b.store.Delete(r.Context(), accessKey)
		b.writeTokenError(w, http.StatusServiceUnavailable, "temporarily_unavailable", "token service unavailable")
		return
	}
	if !rotated {
		_ = b.store.Delete(r.Context(), accessKey)
		if err := b.revokeReusedRefreshToken(r.Context(), oldRefreshToken, client); err != nil {
			b.writeTokenError(w, http.StatusServiceUnavailable, "temporarily_unavailable", "token service unavailable")
			return
		}
		b.writeTokenError(w, http.StatusBadRequest, "invalid_grant", "refresh token was already used")
		return
	}
	b.writeTokenResponse(w, accessToken, refreshToken, record.Scopes)
}

func newOAuthTokenPair() (string, string, error) {
	accessToken, err := randomOAuthValue("poat_", 32)
	if err != nil {
		return "", "", err
	}
	refreshToken, err := randomOAuthValue("port_", 32)
	if err != nil {
		return "", "", err
	}
	return accessToken, refreshToken, nil
}

func (b *oauthBridge) writeTokenResponse(w http.ResponseWriter, accessToken, refreshToken string, scopes []string) {
	w.Header().Set("Cache-Control", "no-store")
	w.Header().Set("Pragma", "no-cache")
	b.writeJSON(w, http.StatusOK, map[string]any{
		"access_token":  accessToken,
		"token_type":    "Bearer",
		"expires_in":    int(oauthAccessTokenTTL.Seconds()),
		"refresh_token": refreshToken,
		"scope":         strings.Join(scopes, " "),
	})
}

func (b *oauthBridge) handleRevocation(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		w.Header().Set("Allow", http.MethodPost)
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	w.Header().Set("Cache-Control", "no-store")
	w.Header().Set("Pragma", "no-cache")
	r.Body = http.MaxBytesReader(w, r.Body, 16<<10)
	if err := r.ParseForm(); err != nil {
		http.Error(w, "invalid request", http.StatusBadRequest)
		return
	}
	client, err := b.authenticateClient(r)
	if err != nil {
		if errors.Is(err, errOAuthInvalidClient) {
			b.writeTokenError(w, http.StatusUnauthorized, "invalid_client", "invalid OAuth client")
		} else {
			b.writeTokenError(w, http.StatusServiceUnavailable, "temporarily_unavailable", "revocation service unavailable")
		}
		return
	}
	token := r.PostForm.Get("token")
	for _, kind := range []string{"access", "refresh", "refresh-used"} {
		key := oauthLookupKey(kind, token)
		var record oauthGrantRecord
		err := getOAuthJSON(r.Context(), b.store, key, &record)
		if errors.Is(err, errOAuthRecordNotFound) {
			continue
		}
		if err != nil {
			b.writeTokenError(w, http.StatusServiceUnavailable, "temporarily_unavailable", "revocation service unavailable")
			return
		}
		if record.ClientID != client.ClientID {
			continue
		}
		if kind != "access" && record.FamilyID != "" {
			if err := b.store.RevokeRefreshFamily(r.Context(), oauthRefreshFamilyKey(record.FamilyID)); err != nil {
				b.writeTokenError(w, http.StatusServiceUnavailable, "temporarily_unavailable", "revocation service unavailable")
				return
			}
		}
		if err := b.store.Delete(r.Context(), key); err != nil {
			b.writeTokenError(w, http.StatusServiceUnavailable, "temporarily_unavailable", "revocation service unavailable")
			return
		}
	}
	w.WriteHeader(http.StatusOK)
}

func (b *oauthBridge) revokeReusedRefreshToken(ctx context.Context, refreshToken string, client *oauthClientRecord) error {
	var used oauthGrantRecord
	err := getOAuthJSON(ctx, b.store, oauthLookupKey("refresh-used", refreshToken), &used)
	if errors.Is(err, errOAuthRecordNotFound) {
		return nil
	}
	if err != nil {
		return err
	}
	if used.ClientID != client.ClientID || used.FamilyID == "" {
		return nil
	}
	return b.store.RevokeRefreshFamily(ctx, oauthRefreshFamilyKey(used.FamilyID))
}

func (b *oauthBridge) openGrantCredential(record oauthGrantRecord) (string, error) {
	aad := oauthGrantAdditionalData(record.ClientID, record.Scopes, record.Resource, record.UserID)
	return b.cipher.Open(record.EncryptedCredential, aad)
}

func (b *oauthBridge) resolveAccessToken(ctx context.Context, token string) (*verifiedCredential, error) {
	if !strings.HasPrefix(token, "poat_") {
		return nil, errNotBridgeAccessToken
	}
	var record oauthGrantRecord
	if err := getOAuthJSON(ctx, b.store, oauthLookupKey("access", token), &record); err != nil {
		if errors.Is(err, errOAuthRecordNotFound) {
			return nil, errOAuthRecordNotFound
		}
		return nil, err
	}
	upstreamToken, err := b.openGrantCredential(record)
	if err != nil {
		return nil, errOAuthRecordNotFound
	}
	info, err := controllerTokenVerifier(b.baseURL)(ctx, upstreamToken, nil)
	if err != nil {
		if errors.Is(err, auth.ErrInvalidToken) {
			return nil, errOAuthRecordNotFound
		}
		return nil, err
	}
	if info == nil {
		return nil, errOAuthRecordNotFound
	}
	return &verifiedCredential{
		UpstreamToken: upstreamToken,
		Scopes:        append([]string(nil), record.Scopes...),
		UserID:        record.UserID,
	}, nil
}

func (b *oauthBridge) authenticateClient(r *http.Request) (*oauthClientRecord, error) {
	clientID := r.PostForm.Get("client_id")
	secret := r.PostForm.Get("client_secret")
	basicID, basicSecret, usedBasic := r.BasicAuth()
	if usedBasic {
		clientID = basicID
		secret = basicSecret
	}
	client, err := b.loadClient(r.Context(), clientID)
	if err != nil {
		if errors.Is(err, errOAuthRecordNotFound) {
			return nil, errOAuthInvalidClient
		}
		return nil, err
	}
	if client.TokenEndpointAuthMethod == "none" {
		if secret != "" || usedBasic {
			return nil, errOAuthInvalidClient
		}
		return client, nil
	}
	if client.TokenEndpointAuthMethod == "client_secret_basic" && !usedBasic {
		return nil, errOAuthInvalidClient
	}
	if client.TokenEndpointAuthMethod == "client_secret_post" && usedBasic {
		return nil, errOAuthInvalidClient
	}
	if secret == "" || subtle.ConstantTimeCompare([]byte(client.ClientSecretHash), []byte(hashOAuthSecret(secret))) != 1 {
		return nil, errOAuthInvalidClient
	}
	return client, nil
}

func (b *oauthBridge) loadClient(ctx context.Context, clientID string) (*oauthClientRecord, error) {
	if !strings.HasPrefix(clientID, "pomcp_") || len(clientID) > 128 {
		return nil, errOAuthRecordNotFound
	}
	var client oauthClientRecord
	if err := getOAuthJSON(ctx, b.store, "client:"+clientID, &client); err != nil {
		return nil, err
	}
	return &client, nil
}

func (b *oauthBridge) validateScopes(requested []string, allowEmpty bool) ([]string, error) {
	scopes := normalizedScopes(requested)
	if len(scopes) == 0 && !allowEmpty {
		scopes = []string{"api:read"}
	}
	for _, scope := range scopes {
		if _, ok := b.scopeSet[scope]; !ok {
			return nil, fmt.Errorf("unsupported scope %q", scope)
		}
	}
	return scopes, nil
}

func (b *oauthBridge) allow(r *http.Request, action string, limit int, window time.Duration) bool {
	host, _, err := net.SplitHostPort(r.RemoteAddr)
	if err != nil {
		host = r.RemoteAddr
	}
	return b.limiter.allow(action+":"+host, limit, window)
}

func (l *oauthRateLimiter) allow(key string, limit int, duration time.Duration) bool {
	l.mu.Lock()
	defer l.mu.Unlock()
	now := l.now()
	window, ok := l.windows[key]
	if !ok && len(l.windows) >= 4096 {
		for existingKey, existing := range l.windows {
			if !existing.expiresAt.After(now) {
				delete(l.windows, existingKey)
			}
		}
		if len(l.windows) >= 4096 {
			return false
		}
	}
	if !ok || !window.expiresAt.After(now) {
		l.windows[key] = oauthRateWindow{started: now, expiresAt: now.Add(duration), count: 1}
		return true
	}
	if window.count >= limit {
		return false
	}
	window.count++
	l.windows[key] = window
	return true
}

func (b *oauthBridge) redirectAuthorizationError(w http.ResponseWriter, r *http.Request, redirectURI, state, code, description string) {
	redirect, err := url.Parse(redirectURI)
	if err != nil {
		http.Error(w, "invalid authorization request", http.StatusBadRequest)
		return
	}
	values := redirect.Query()
	values.Set("error", code)
	values.Set("error_description", description)
	if state != "" {
		values.Set("state", state)
	}
	redirect.RawQuery = values.Encode()
	http.Redirect(w, r, redirect.String(), http.StatusFound)
}

func (b *oauthBridge) writeRegistrationError(w http.ResponseWriter, code, description string) {
	b.writeJSON(w, http.StatusBadRequest, map[string]string{"error": code, "error_description": description})
}

func (b *oauthBridge) writeTokenError(w http.ResponseWriter, status int, code, description string) {
	w.Header().Set("Cache-Control", "no-store")
	b.writeJSON(w, status, map[string]string{"error": code, "error_description": description})
}

func (b *oauthBridge) writeJSON(w http.ResponseWriter, status int, value any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(value)
}

func putOAuthJSON(ctx context.Context, store oauthStore, key string, value any, ttl time.Duration) error {
	encoded, err := json.Marshal(value)
	if err != nil {
		return fmt.Errorf("encode OAuth record: %w", err)
	}
	return store.Put(ctx, key, encoded, ttl)
}

func getOAuthJSON(ctx context.Context, store oauthStore, key string, target any) error {
	encoded, err := store.Get(ctx, key)
	if err != nil {
		return err
	}
	if err := json.Unmarshal(encoded, target); err != nil {
		return fmt.Errorf("decode OAuth record: %w", err)
	}
	return nil
}

func randomOAuthValue(prefix string, size int) (string, error) {
	value := make([]byte, size)
	if _, err := rand.Read(value); err != nil {
		return "", fmt.Errorf("generate OAuth value: %w", err)
	}
	return prefix + base64.RawURLEncoding.EncodeToString(value), nil
}

func hashOAuthSecret(value string) string {
	digest := sha256.Sum256([]byte(value))
	return base64.RawURLEncoding.EncodeToString(digest[:])
}

func oauthGrantAdditionalData(clientID string, scopes []string, resource, userID string) []byte {
	encoded, _ := json.Marshal(struct {
		ClientID string   `json:"client_id"`
		Scopes   []string `json:"scopes"`
		Resource string   `json:"resource"`
		UserID   string   `json:"user_id"`
	}{
		ClientID: clientID,
		Scopes:   scopes,
		Resource: resource,
		UserID:   userID,
	})
	return encoded
}

func oauthRefreshFamilyKey(familyID string) string {
	return "refresh-family:" + familyID
}

func validPKCEVerifier(verifier, challenge string) bool {
	if len(verifier) < 43 || len(verifier) > 128 {
		return false
	}
	digest := sha256.Sum256([]byte(verifier))
	actual := base64.RawURLEncoding.EncodeToString(digest[:])
	return subtle.ConstantTimeCompare([]byte(actual), []byte(challenge)) == 1
}

func validateOAuthRedirectURI(value string) error {
	parsed, err := url.Parse(value)
	if err != nil || parsed.Scheme == "" || parsed.Host == "" {
		return errors.New("redirect_uri must be an absolute URL")
	}
	if parsed.Fragment != "" || parsed.User != nil {
		return errors.New("redirect_uri must not contain user information or a fragment")
	}
	switch parsed.Scheme {
	case "https":
		return nil
	case "http":
		if isLoopbackHostname(parsed.Hostname()) {
			return nil
		}
		return errors.New("http redirect_uri must use a loopback host")
	default:
		return errors.New("redirect_uri must use https or loopback http")
	}
}

func normalizedScopes(scopes []string) []string {
	seen := make(map[string]struct{}, len(scopes))
	result := make([]string, 0, len(scopes))
	for _, scope := range scopes {
		scope = strings.TrimSpace(scope)
		if scope == "" {
			continue
		}
		if _, ok := seen[scope]; ok {
			continue
		}
		seen[scope] = struct{}{}
		result = append(result, scope)
	}
	sort.Strings(result)
	return result
}

func scopeSubset(candidate, allowed []string) bool {
	allowedSet := make(map[string]struct{}, len(allowed))
	for _, scope := range allowed {
		allowedSet[scope] = struct{}{}
	}
	for _, scope := range candidate {
		if _, ok := allowedSet[scope]; !ok {
			return false
		}
	}
	return true
}

func containsString(values []string, candidate string) bool {
	for _, value := range values {
		if value == candidate {
			return true
		}
	}
	return false
}

func authorizationRedirectHost(value string) string {
	parsed, err := url.Parse(value)
	if err != nil {
		return "the registered client"
	}
	return parsed.Host
}

const oauthConsentPage = `<!doctype html>
<html lang="en">
<head>
  <meta charset="utf-8">
  <meta name="viewport" content="width=device-width, initial-scale=1">
  <title>Connect PipeOps</title>
  <style>
    :root { color-scheme: light dark; font-family: Inter, ui-sans-serif, system-ui, sans-serif; }
    body { margin: 0; min-height: 100vh; display: grid; place-items: center; background: #07111f; color: #eaf2ff; }
    main { width: min(92vw, 32rem); box-sizing: border-box; padding: 2rem; border: 1px solid #24364d; border-radius: 1rem; background: #0d1b2d; box-shadow: 0 1.5rem 4rem #0008; }
    h1 { margin-top: 0; font-size: 1.5rem; }
    p, li { color: #b9c9dc; line-height: 1.55; }
    label { display: block; margin: 1.25rem 0 .5rem; font-weight: 650; }
    input { width: 100%; box-sizing: border-box; padding: .85rem; border: 1px solid #3a506d; border-radius: .6rem; background: #07111f; color: #fff; }
    input[aria-invalid="true"] { border-color: #ff7b8b; outline-color: #ff7b8b; }
    .actions { display: flex; gap: .75rem; margin-top: 1.25rem; }
    button { border: 0; border-radius: .6rem; padding: .8rem 1rem; font-weight: 700; cursor: pointer; }
    button:disabled { cursor: wait; opacity: .7; }
    button[value="approve"] { background: #5b8cff; color: #06101d; }
    button[value="deny"] { background: #24364d; color: #eaf2ff; }
    .error { padding: .75rem; border-radius: .5rem; background: #4a1720; color: #ffd9df; }
    a { color: #8eb1ff; }
    code { font-size: .9em; }
  </style>
</head>
<body>
<main>
  <h1>Connect {{.ClientName}} to PipeOps</h1>
  <p>This client is requesting access through the hosted PipeOps MCP server. After approval, access is returned to <strong>{{.RedirectHost}}</strong>.</p>
  <p>Continue only if you started this connection from that client.</p>
  <ul>{{range .Scopes}}<li><code>{{.}}</code></li>{{end}}</ul>
  {{if .ErrorMessage}}<p class="error" role="alert">{{.ErrorMessage}}</p>{{end}}
  <form method="post" action="/oauth/authorize">
    <input type="hidden" name="request_id" value="{{.RequestID}}">
    <label for="service_token">Workspace service token</label>
    <input id="service_token" name="service_token" type="password" autocomplete="off" spellcheck="false" aria-describedby="service_token_help form_error" required autofocus>
    <p id="service_token_help">Create a dedicated token under <a href="https://console.pipeops.io/dashboard/integrations?cloudIntegrations=tokens" target="_blank" rel="noopener noreferrer">PipeOps Service Tokens</a>. Use <code>api:read</code> first and add <code>api:write</code> only when needed.</p>
    <p id="form_error" class="error" role="alert" aria-live="assertive" hidden></p>
    <div class="actions">
      <button type="submit" name="action" value="approve">Connect</button>
      <button type="submit" name="action" value="deny" formnovalidate>Cancel</button>
    </div>
  </form>
</main>
<script nonce="{{.ScriptNonce}}">
  (() => {
    const form = document.querySelector("form");
    const token = document.getElementById("service_token");
    const error = document.getElementById("form_error");
    const connect = form.querySelector('button[value="approve"]');

    const showError = (message) => {
      error.textContent = message;
      error.hidden = false;
      token.setAttribute("aria-invalid", "true");
      token.focus();
    };
    const clearError = () => {
      error.textContent = "";
      error.hidden = true;
      token.removeAttribute("aria-invalid");
    };

    form.addEventListener("invalid", (event) => {
      if (event.target === token) {
        event.preventDefault();
        showError("Paste a PipeOps workspace service token before connecting.");
      }
    }, true);
    token.addEventListener("input", clearError);
    form.addEventListener("submit", (event) => {
      if (event.submitter?.value === "deny") return;
      const value = token.value.trim();
      if (!value.startsWith("sat_") || /\s/.test(value)) {
        event.preventDefault();
        showError("Enter a valid PipeOps workspace service token beginning with sat_.");
        return;
      }
      token.value = value;
      clearError();
      form.setAttribute("aria-busy", "true");
      connect.disabled = true;
      connect.textContent = "Connecting…";
    });
  })();
</script>
</body>
</html>`

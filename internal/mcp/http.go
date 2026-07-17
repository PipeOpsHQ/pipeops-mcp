package mcp

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"os"
	"strings"
	"time"

	"github.com/PipeOpsHQ/pipeops-go-sdk/pipeops"
	"github.com/modelcontextprotocol/go-sdk/auth"
	sdkmcp "github.com/modelcontextprotocol/go-sdk/mcp"
	"github.com/modelcontextprotocol/go-sdk/oauthex"
)

const (
	defaultBaseURL     = "https://api.pipeops.io"
	defaultResourceURL = "https://mcp.pipeops.app/mcp"
	defaultHTTPAddr    = ":8080"
	defaultMaxBodySize = int64(4 << 20)
)

type verifiedCredentialContextKey struct{}

// HTTPConfig configures the hosted Streamable HTTP MCP endpoint.
type HTTPConfig struct {
	Addr                string
	BaseURL             string
	ResourceURL         string
	OAuthMode           string
	AuthorizationServer string
	OAuthRedisURL       string
	OAuthEncryptionKey  string
	Scopes              []string
	MaxBodyBytes        int64
	oauthStore          oauthStore
}

// LoadHTTPConfigFromEnv loads hosted MCP settings from environment variables.
func LoadHTTPConfigFromEnv() HTTPConfig {
	config := HTTPConfig{
		Addr:                strings.TrimSpace(os.Getenv("PIPEOPS_HTTP_ADDR")),
		BaseURL:             strings.TrimSpace(os.Getenv("PIPEOPS_BASE_URL")),
		ResourceURL:         strings.TrimSpace(os.Getenv("PIPEOPS_MCP_PUBLIC_URL")),
		OAuthMode:           strings.ToLower(strings.TrimSpace(os.Getenv("PIPEOPS_OAUTH_MODE"))),
		AuthorizationServer: strings.TrimSpace(os.Getenv("PIPEOPS_OAUTH_ISSUER")),
		OAuthRedisURL:       strings.TrimSpace(os.Getenv("PIPEOPS_OAUTH_REDIS_URL")),
		OAuthEncryptionKey:  strings.TrimSpace(os.Getenv("PIPEOPS_OAUTH_ENCRYPTION_KEY")),
		Scopes:              splitList(os.Getenv("PIPEOPS_MCP_SCOPES")),
		MaxBodyBytes:        defaultMaxBodySize,
	}
	if config.Addr == "" {
		config.Addr = defaultHTTPAddr
	}
	if config.BaseURL == "" {
		config.BaseURL = defaultBaseURL
	}
	if config.ResourceURL == "" {
		config.ResourceURL = defaultResourceURL
	}
	if len(config.Scopes) == 0 {
		config.Scopes = defaultMCPScopes()
	}
	return config
}

// NewHTTPHandler returns a stateless Streamable HTTP MCP handler with Bearer
// authentication and OAuth protected-resource discovery.
func NewHTTPHandler(config HTTPConfig) (http.Handler, error) {
	config = withHTTPDefaults(config)
	resourceURL, err := validatePublicURL("resource URL", config.ResourceURL)
	if err != nil {
		return nil, err
	}
	if resourceURL.RawQuery != "" {
		return nil, errors.New("resource URL must not contain a query string")
	}
	if _, err := validateAbsoluteURL("PipeOps API base URL", config.BaseURL); err != nil {
		return nil, err
	}

	var bridge *oauthBridge
	authorizationServers := []string{}
	switch config.OAuthMode {
	case "", "bearer":
		config.OAuthMode = "bearer"
	case "external":
		if config.AuthorizationServer == "" {
			return nil, errors.New("PIPEOPS_OAUTH_ISSUER is required when PIPEOPS_OAUTH_MODE=external")
		}
		if _, err := validatePublicURL("authorization server", config.AuthorizationServer); err != nil {
			return nil, err
		}
		authorizationServers = append(authorizationServers, config.AuthorizationServer)
	case "bridge":
		issuer := config.AuthorizationServer
		if issuer == "" {
			issuer = resourceURL.Scheme + "://" + resourceURL.Host
		}
		if _, err := validatePublicURL("authorization server", issuer); err != nil {
			return nil, err
		}
		store := config.oauthStore
		if store == nil {
			if config.OAuthRedisURL == "" {
				return nil, errors.New("PIPEOPS_OAUTH_REDIS_URL is required when PIPEOPS_OAUTH_MODE=bridge")
			}
			store, err = newRedisOAuthStore(context.Background(), config.OAuthRedisURL)
			if err != nil {
				return nil, err
			}
		}
		credentialCipher, err := newCredentialCipher(config.OAuthEncryptionKey)
		if err != nil {
			return nil, err
		}
		bridge, err = newOAuthBridge(oauthBridgeConfig{
			Issuer:      issuer,
			ResourceURL: config.ResourceURL,
			BaseURL:     config.BaseURL,
			Scopes:      config.Scopes,
			Store:       store,
			Cipher:      credentialCipher,
		})
		if err != nil {
			return nil, err
		}
		config.AuthorizationServer = issuer
		authorizationServers = append(authorizationServers, issuer)
	default:
		return nil, fmt.Errorf("unsupported PIPEOPS_OAUTH_MODE %q: use bearer, bridge, or external", config.OAuthMode)
	}

	metadataURL := resourceURL.Scheme + "://" + resourceURL.Host + "/.well-known/oauth-protected-resource"
	metadata := &oauthex.ProtectedResourceMetadata{
		Resource:               config.ResourceURL,
		AuthorizationServers:   authorizationServers,
		ScopesSupported:        config.Scopes,
		BearerMethodsSupported: []string{"header"},
		ResourceName:           "PipeOps MCP Server",
	}

	streamable := sdkmcp.NewStreamableHTTPHandler(func(request *http.Request) *sdkmcp.Server {
		credential, _ := request.Context().Value(verifiedCredentialContextKey{}).(*verifiedCredential)
		if credential == nil || credential.UpstreamToken == "" {
			return nil
		}
		server, err := newServerWithTokenAndScopes(config.BaseURL, credential.UpstreamToken, credential.Scopes)
		if err != nil {
			return nil
		}
		return server.newSDKServer()
	}, &sdkmcp.StreamableHTTPOptions{
		Stateless:    true,
		JSONResponse: true,
	})

	verifier := controllerCredentialVerifier(config.BaseURL, bridge)
	authenticated := requireBearerToken(
		verifier,
		metadataURL,
		defaultChallengeScope(config.Scopes),
		limitRequestBody(streamable, config.MaxBodyBytes),
	)

	mux := http.NewServeMux()
	mux.Handle("/.well-known/oauth-protected-resource", auth.ProtectedResourceMetadataHandler(metadata))
	mux.Handle("/.well-known/oauth-protected-resource/mcp", auth.ProtectedResourceMetadataHandler(metadata))
	if bridge != nil {
		bridge.registerRoutes(mux)
	}
	mux.Handle("/mcp", authenticated)
	mux.HandleFunc("/healthz", func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"status":"ok"}`))
	})

	return securityHeaders(mux), nil
}

func withHTTPDefaults(config HTTPConfig) HTTPConfig {
	if config.BaseURL == "" {
		config.BaseURL = defaultBaseURL
	}
	if config.ResourceURL == "" {
		config.ResourceURL = defaultResourceURL
	}
	config.OAuthMode = strings.ToLower(strings.TrimSpace(config.OAuthMode))
	if config.MaxBodyBytes <= 0 {
		config.MaxBodyBytes = defaultMaxBodySize
	}
	if len(config.Scopes) == 0 {
		config.Scopes = defaultMCPScopes()
	}
	return config
}

func defaultMCPScopes() []string {
	return []string{
		"api:read",
		"api:write",
	}
}

func defaultChallengeScope(scopes []string) string {
	for _, scope := range scopes {
		if scope == "api:read" {
			return scope
		}
	}
	if len(scopes) > 0 {
		return scopes[0]
	}
	return ""
}

func validateAbsoluteURL(name, value string) (*url.URL, error) {
	parsed, err := url.Parse(value)
	if err != nil || parsed.Scheme == "" || parsed.Host == "" {
		return nil, fmt.Errorf("invalid %s %q", name, value)
	}
	if parsed.Scheme != "https" && parsed.Scheme != "http" {
		return nil, fmt.Errorf("invalid %s scheme %q", name, parsed.Scheme)
	}
	return parsed, nil
}

func validatePublicURL(name, value string) (*url.URL, error) {
	parsed, err := validateAbsoluteURL(name, value)
	if err != nil {
		return nil, err
	}
	if parsed.User != nil || parsed.Fragment != "" {
		return nil, fmt.Errorf("%s must not contain user information or a fragment", name)
	}
	if parsed.Scheme != "https" && !isLoopbackHostname(parsed.Hostname()) {
		return nil, fmt.Errorf("invalid %s scheme %q: public URLs must use https", name, parsed.Scheme)
	}
	return parsed, nil
}

func isLoopbackHostname(hostname string) bool {
	switch strings.ToLower(hostname) {
	case "localhost", "127.0.0.1", "::1":
		return true
	default:
		return false
	}
}

func splitList(value string) []string {
	return strings.FieldsFunc(value, func(r rune) bool {
		return r == ',' || r == ' ' || r == '\n' || r == '\t'
	})
}

func controllerTokenVerifier(baseURL string) auth.TokenVerifier {
	return func(ctx context.Context, token string, _ *http.Request) (*auth.TokenInfo, error) {
		client, err := pipeops.NewClient(baseURL)
		if err != nil {
			return nil, errors.New("token validation unavailable")
		}
		client.SetToken(token)

		profile, response, err := client.Users.GetProfile(ctx)
		if err != nil {
			if response != nil && (response.StatusCode == http.StatusUnauthorized || response.StatusCode == http.StatusForbidden) {
				return nil, auth.ErrInvalidToken
			}
			return nil, errors.New("token validation unavailable")
		}
		if !strings.EqualFold(profile.Status, "success") && !strings.EqualFold(profile.Status, "ok") {
			return nil, auth.ErrInvalidToken
		}

		userID := profile.Data.User.UUID
		if userID == "" {
			userID = profile.Data.User.ID
		}
		if userID == "" {
			userID = profile.Data.User.Email
		}
		if userID == "" {
			return nil, auth.ErrInvalidToken
		}
		return &auth.TokenInfo{
			Expiration: time.Now().Add(5 * time.Minute),
			UserID:     userID,
		}, nil
	}
}

type credentialVerifier func(context.Context, string, *http.Request) (*verifiedCredential, error)

func controllerCredentialVerifier(baseURL string, bridge *oauthBridge) credentialVerifier {
	return func(ctx context.Context, token string, request *http.Request) (*verifiedCredential, error) {
		if bridge != nil {
			credential, err := bridge.resolveAccessToken(ctx, token)
			if err == nil {
				return credential, nil
			}
			if !errors.Is(err, errNotBridgeAccessToken) {
				if errors.Is(err, errOAuthRecordNotFound) {
					return nil, auth.ErrInvalidToken
				}
				return nil, err
			}
		}
		info, err := controllerTokenVerifier(baseURL)(ctx, token, request)
		if err != nil {
			return nil, err
		}
		return &verifiedCredential{UpstreamToken: token, UserID: info.UserID}, nil
	}
}

func requireBearerToken(verifier credentialVerifier, metadataURL, scope string, next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, request *http.Request) {
		fields := strings.Fields(request.Header.Get("Authorization"))
		if len(fields) != 2 || !strings.EqualFold(fields[0], "Bearer") || fields[1] == "" {
			writeBearerAuthError(w, metadataURL, scope, "", "bearer token required", http.StatusUnauthorized)
			return
		}

		token := fields[1]
		credential, err := verifier(request.Context(), token, request)
		if err != nil {
			if errors.Is(err, auth.ErrInvalidToken) {
				writeBearerAuthError(w, metadataURL, scope, "invalid_token", "invalid bearer token", http.StatusUnauthorized)
				return
			}
			writeBearerAuthError(w, metadataURL, scope, "", "authentication service unavailable", http.StatusServiceUnavailable)
			return
		}
		if credential == nil || credential.UpstreamToken == "" {
			writeBearerAuthError(w, metadataURL, scope, "invalid_token", "invalid bearer token", http.StatusUnauthorized)
			return
		}

		ctx := context.WithValue(request.Context(), verifiedCredentialContextKey{}, credential)
		next.ServeHTTP(w, request.WithContext(ctx))
	})
}

func writeBearerAuthError(w http.ResponseWriter, metadataURL, scope, oauthError, message string, status int) {
	challenge := fmt.Sprintf("Bearer resource_metadata=%q", metadataURL)
	if scope != "" {
		challenge += fmt.Sprintf(", scope=%q", scope)
	}
	if oauthError != "" {
		challenge += fmt.Sprintf(", error=%q", oauthError)
	}
	w.Header().Set("WWW-Authenticate", challenge)
	http.Error(w, message, status)
}

func limitRequestBody(next http.Handler, maxBytes int64) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, request *http.Request) {
		if request.Body != nil {
			request.Body = http.MaxBytesReader(w, request.Body, maxBytes)
		}
		next.ServeHTTP(w, request)
	})
}

func securityHeaders(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, request *http.Request) {
		w.Header().Set("X-Content-Type-Options", "nosniff")
		w.Header().Set("Referrer-Policy", "no-referrer")
		if request.URL.Path == "/mcp" {
			w.Header().Set("Cache-Control", "no-store")
		}
		next.ServeHTTP(w, request)
	})
}

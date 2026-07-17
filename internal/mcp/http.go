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
	defaultResourceURL = "https://mcp.pipeops.io/mcp"
	defaultOAuthIssuer = "https://api.pipeops.io"
	defaultHTTPAddr    = ":8080"
	defaultMaxBodySize = int64(4 << 20)
)

type bearerTokenContextKey struct{}

// HTTPConfig configures the hosted Streamable HTTP MCP endpoint.
type HTTPConfig struct {
	Addr                string
	BaseURL             string
	ResourceURL         string
	AuthorizationServer string
	Scopes              []string
	MaxBodyBytes        int64
}

// LoadHTTPConfigFromEnv loads hosted MCP settings from environment variables.
func LoadHTTPConfigFromEnv() HTTPConfig {
	config := HTTPConfig{
		Addr:                strings.TrimSpace(os.Getenv("PIPEOPS_HTTP_ADDR")),
		BaseURL:             strings.TrimSpace(os.Getenv("PIPEOPS_BASE_URL")),
		ResourceURL:         strings.TrimSpace(os.Getenv("PIPEOPS_MCP_PUBLIC_URL")),
		AuthorizationServer: strings.TrimSpace(os.Getenv("PIPEOPS_OAUTH_ISSUER")),
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
	if config.AuthorizationServer == "" {
		config.AuthorizationServer = defaultOAuthIssuer
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
	if _, err := validatePublicURL("authorization server", config.AuthorizationServer); err != nil {
		return nil, err
	}
	if _, err := validateAbsoluteURL("PipeOps API base URL", config.BaseURL); err != nil {
		return nil, err
	}

	metadataURL := resourceURL.Scheme + "://" + resourceURL.Host + "/.well-known/oauth-protected-resource"
	metadata := &oauthex.ProtectedResourceMetadata{
		Resource:               config.ResourceURL,
		AuthorizationServers:   []string{config.AuthorizationServer},
		ScopesSupported:        config.Scopes,
		BearerMethodsSupported: []string{"header"},
		ResourceName:           "PipeOps MCP Server",
	}

	streamable := sdkmcp.NewStreamableHTTPHandler(func(request *http.Request) *sdkmcp.Server {
		token, _ := request.Context().Value(bearerTokenContextKey{}).(string)
		if token == "" {
			return nil
		}
		server, err := newServerWithToken(config.BaseURL, token)
		if err != nil {
			return nil
		}
		return server.newSDKServer()
	}, &sdkmcp.StreamableHTTPOptions{
		Stateless:    true,
		JSONResponse: true,
	})

	verifier := controllerTokenVerifier(config.BaseURL)
	authenticated := requireBearerToken(
		verifier,
		metadataURL,
		limitRequestBody(streamable, config.MaxBodyBytes),
	)

	mux := http.NewServeMux()
	mux.Handle("/.well-known/oauth-protected-resource", auth.ProtectedResourceMetadataHandler(metadata))
	mux.Handle("/.well-known/oauth-protected-resource/mcp", auth.ProtectedResourceMetadataHandler(metadata))
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
	if config.AuthorizationServer == "" {
		config.AuthorizationServer = defaultOAuthIssuer
	}
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
		"pipeops:read",
		"projects:write",
		"deployments:write",
		"addons:write",
		"billing:write",
		"tokens:admin",
	}
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

func requireBearerToken(verifier auth.TokenVerifier, metadataURL string, next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, request *http.Request) {
		fields := strings.Fields(request.Header.Get("Authorization"))
		if len(fields) != 2 || !strings.EqualFold(fields[0], "Bearer") || fields[1] == "" {
			writeBearerAuthError(w, metadataURL, "", "bearer token required", http.StatusUnauthorized)
			return
		}

		token := fields[1]
		tokenInfo, err := verifier(request.Context(), token, request)
		if err != nil {
			if errors.Is(err, auth.ErrInvalidToken) {
				writeBearerAuthError(w, metadataURL, "invalid_token", "invalid bearer token", http.StatusUnauthorized)
				return
			}
			writeBearerAuthError(w, metadataURL, "", "authentication service unavailable", http.StatusServiceUnavailable)
			return
		}
		if tokenInfo == nil || tokenInfo.Expiration.IsZero() || tokenInfo.Expiration.Before(time.Now()) {
			writeBearerAuthError(w, metadataURL, "invalid_token", "invalid bearer token", http.StatusUnauthorized)
			return
		}

		ctx := context.WithValue(request.Context(), bearerTokenContextKey{}, token)
		next.ServeHTTP(w, request.WithContext(ctx))
	})
}

func writeBearerAuthError(w http.ResponseWriter, metadataURL, oauthError, message string, status int) {
	challenge := fmt.Sprintf("Bearer resource_metadata=%q", metadataURL)
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

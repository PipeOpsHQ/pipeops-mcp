package mcp

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"os"
	"strings"
	"sync"
	"time"

	"github.com/PipeOpsHQ/pipeops-go-sdk/pipeops"
	"github.com/modelcontextprotocol/go-sdk/auth"
	sdkmcp "github.com/modelcontextprotocol/go-sdk/mcp"
	"github.com/modelcontextprotocol/go-sdk/oauthex"
)

const (
	defaultBaseURL         = "https://api.pipeops.io"
	defaultResourceURL     = "https://mcp.pipeops.app/mcp"
	defaultHTTPAddr        = ":8080"
	defaultMaxBodySize     = int64(4 << 20)
	defaultOAuthStore      = "sqlite"
	defaultSQLitePath      = "/data/oauth/pipeops-mcp-oauth.db"
	defaultConsoleURL      = "https://console.pipeops.io"
	defaultConsoleClientID = "pipeops_public_client"
)

type verifiedCredentialContextKey struct{}

// HTTPHandler serves the hosted MCP endpoint and closes any persistence store
// it owns when Close is called.
type HTTPHandler struct {
	http.Handler
	closeOnce sync.Once
	closer    interface{ Close() error }
	closeErr  error
}

// Close flushes and releases the persistence store owned by the handler.
func (h *HTTPHandler) Close() error {
	h.closeOnce.Do(func() {
		if h.closer != nil {
			h.closeErr = h.closer.Close()
		}
	})
	return h.closeErr
}

// HTTPConfig configures the hosted Streamable HTTP MCP endpoint.
type HTTPConfig struct {
	Addr                string
	BaseURL             string
	ResourceURL         string
	OAuthMode           string
	AuthorizationServer string
	OAuthStore          string
	OAuthSQLitePath     string
	OAuthRedisURL       string
	OAuthEncryptionKey  string
	ConsoleURL          string
	ConsoleClientID     string
	ConsoleScopes       []string
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
		OAuthStore:          strings.ToLower(strings.TrimSpace(os.Getenv("PIPEOPS_OAUTH_STORE"))),
		OAuthSQLitePath:     strings.TrimSpace(os.Getenv("PIPEOPS_OAUTH_SQLITE_PATH")),
		OAuthRedisURL:       strings.TrimSpace(os.Getenv("PIPEOPS_OAUTH_REDIS_URL")),
		OAuthEncryptionKey:  strings.TrimSpace(os.Getenv("PIPEOPS_OAUTH_ENCRYPTION_KEY")),
		ConsoleURL:          strings.TrimSpace(os.Getenv("PIPEOPS_CONSOLE_OAUTH_URL")),
		ConsoleClientID:     strings.TrimSpace(os.Getenv("PIPEOPS_CONSOLE_OAUTH_CLIENT_ID")),
		ConsoleScopes:       splitList(os.Getenv("PIPEOPS_CONSOLE_OAUTH_SCOPES")),
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
	return withHTTPDefaults(config)
}

// NewHTTPHandler returns a stateless Streamable HTTP MCP handler with Bearer
// authentication and OAuth protected-resource discovery.
func NewHTTPHandler(config HTTPConfig) (*HTTPHandler, error) {
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
	var ownedStore interface{ Close() error }
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
		credentialCipher, err := newCredentialCipher(config.OAuthEncryptionKey)
		if err != nil {
			return nil, err
		}
		store := config.oauthStore
		if store == nil {
			switch config.OAuthStore {
			case "sqlite":
				store, err = newSQLiteOAuthStore(context.Background(), config.OAuthSQLitePath)
			case "redis":
				if config.OAuthRedisURL == "" {
					return nil, errors.New("PIPEOPS_OAUTH_REDIS_URL is required when PIPEOPS_OAUTH_STORE=redis")
				}
				store, err = newRedisOAuthStore(context.Background(), config.OAuthRedisURL)
			default:
				return nil, fmt.Errorf("unsupported PIPEOPS_OAUTH_STORE %q: use sqlite or redis", config.OAuthStore)
			}
			if err != nil {
				return nil, err
			}
			ownedStore, _ = store.(interface{ Close() error })
		}
		bridge, err = newOAuthBridge(oauthBridgeConfig{
			Issuer:          issuer,
			ResourceURL:     config.ResourceURL,
			BaseURL:         config.BaseURL,
			ConsoleURL:      config.ConsoleURL,
			ConsoleClientID: config.ConsoleClientID,
			ConsoleScopes:   config.ConsoleScopes,
			Scopes:          config.Scopes,
			Store:           store,
			Cipher:          credentialCipher,
		})
		if err != nil {
			if ownedStore != nil {
				if closeErr := ownedStore.Close(); closeErr != nil {
					return nil, errors.Join(err, closeErr)
				}
			}
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

	return &HTTPHandler{Handler: securityHeaders(mux), closer: ownedStore}, nil
}

func withHTTPDefaults(config HTTPConfig) HTTPConfig {
	if config.BaseURL == "" {
		config.BaseURL = defaultBaseURL
	}
	if config.ResourceURL == "" {
		config.ResourceURL = defaultResourceURL
	}
	config.OAuthMode = strings.ToLower(strings.TrimSpace(config.OAuthMode))
	config.OAuthStore = strings.ToLower(strings.TrimSpace(config.OAuthStore))
	if config.OAuthStore == "" {
		if config.OAuthRedisURL != "" {
			config.OAuthStore = "redis"
		} else {
			config.OAuthStore = defaultOAuthStore
		}
	}
	if config.OAuthSQLitePath == "" {
		config.OAuthSQLitePath = defaultSQLitePath
	}
	if config.ConsoleURL == "" {
		config.ConsoleURL = defaultConsoleURL
	}
	if config.ConsoleClientID == "" {
		config.ConsoleClientID = defaultConsoleClientID
	}
	if len(config.ConsoleScopes) == 0 {
		config.ConsoleScopes = []string{"openid", "profile", "email"}
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
		"api:read",
		"api:write",
	}
}

func defaultChallengeScope(scopes []string) string {
	return strings.Join(normalizedScopes(scopes), " ")
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
		if strings.HasPrefix(token, "sat_") || strings.HasPrefix(token, "clt_") {
			workspaces, response, err := client.Workspaces.List(ctx)
			if err != nil {
				if response != nil && (response.StatusCode == http.StatusUnauthorized || response.StatusCode == http.StatusForbidden) {
					return nil, auth.ErrInvalidToken
				}
				return nil, errors.New("token validation unavailable")
			}
			if !strings.EqualFold(workspaces.Status, "success") && !strings.EqualFold(workspaces.Status, "ok") {
				return nil, auth.ErrInvalidToken
			}
			if len(workspaces.Data.Workspaces) != 1 || workspaces.Data.Workspaces[0].UUID == "" {
				return nil, auth.ErrInvalidToken
			}
			return &auth.TokenInfo{
				Expiration: time.Now().Add(5 * time.Minute),
				UserID:     "workspace:" + workspaces.Data.Workspaces[0].UUID,
			}, nil
		}

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

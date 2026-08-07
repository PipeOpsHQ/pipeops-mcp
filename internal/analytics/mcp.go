// Package analytics implements PostHog MCP Analytics event capture for the
// PipeOps MCP server (Go). Official @posthog/mcp is TS/Python-only; we emit
// the same $mcp_* wire contract so project dashboards (e.g. project 15061)
// work without a language rewrite.
//
// See https://posthog.com/docs/mcp-analytics/events
package analytics

import (
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"os"
	"strings"
	"sync"
	"time"

	posthog "github.com/posthog/posthog-go"
)

const (
	sourceTag      = "posthog_mcp_analytics"
	serverName     = "pipeops-mcp-server"
	serverVersion  = "1.1.0"
	defaultHost    = "https://us.i.posthog.com"
	maxStringBytes = 8 << 10 // 8 KiB per string field
	maxDepth       = 8
)

// Client captures MCP protocol events into PostHog.
type Client struct {
	ph        posthog.Client
	projectID string
	enabled   bool
}

var (
	globalMu sync.Mutex
	global   *Client
)

// InitFromEnv configures the process-wide analytics client.
// Enabled when POSTHOG_API_KEY or POSTHOG_PROJECT_API_KEY is set (unless
// POSTHOG_MCP_ANALYTICS=false).
//
// Env:
//   - POSTHOG_API_KEY / POSTHOG_PROJECT_API_KEY — project API key (phc_…)
//   - POSTHOG_HOST — https://us.i.posthog.com or https://eu.i.posthog.com
//   - POSTHOG_PROJECT_ID — optional (e.g. 15061) stamped on events for filtering
//   - POSTHOG_MCP_ANALYTICS — set to "false"/"0"/"off" to disable even with a key
func InitFromEnv() (*Client, error) {
	globalMu.Lock()
	defer globalMu.Unlock()
	return initFromEnvLocked()
}

func initFromEnvLocked() (*Client, error) {
	if global != nil {
		return global, nil
	}

	if !envTruthyDefaultOn("POSTHOG_MCP_ANALYTICS") {
		global = &Client{enabled: false}
		return global, nil
	}

	apiKey := firstNonEmpty(
		strings.TrimSpace(os.Getenv("POSTHOG_API_KEY")),
		strings.TrimSpace(os.Getenv("POSTHOG_PROJECT_API_KEY")),
	)
	if apiKey == "" {
		global = &Client{enabled: false}
		return global, nil
	}

	host := strings.TrimSpace(os.Getenv("POSTHOG_HOST"))
	if host == "" {
		host = defaultHost
	}
	if !strings.HasPrefix(host, "http://") && !strings.HasPrefix(host, "https://") {
		host = "https://" + host
	}

	ph, err := posthog.NewWithConfig(apiKey, posthog.Config{
		Endpoint: host,
	})
	if err != nil {
		return nil, fmt.Errorf("posthog client: %w", err)
	}

	global = &Client{
		ph:        ph,
		projectID: strings.TrimSpace(os.Getenv("POSTHOG_PROJECT_ID")),
		enabled:   true,
	}
	return global, nil
}

// Get returns the process-wide client (may be disabled/no-op).
func Get() *Client {
	globalMu.Lock()
	defer globalMu.Unlock()
	if global == nil {
		c, _ := initFromEnvLocked()
		return c
	}
	return global
}

// Enabled reports whether events will be sent.
func (c *Client) Enabled() bool {
	return c != nil && c.enabled && c.ph != nil
}

// Close flushes and shuts down the PostHog client.
func (c *Client) Close() error {
	if c == nil || c.ph == nil {
		return nil
	}
	return c.ph.Close()
}

// CloseGlobal closes the process-wide client.
func CloseGlobal() error {
	globalMu.Lock()
	defer globalMu.Unlock()
	if global == nil {
		return nil
	}
	err := global.Close()
	global = nil
	return err
}

// CaptureInitialize records $mcp_initialize.
func (c *Client) CaptureInitialize(sessionID, clientName, clientVersion, protocolVersion, distinctID string) {
	if !c.Enabled() {
		return
	}
	props := c.baseProps(sessionID, clientName, clientVersion, protocolVersion)
	props.Set("$mcp_server_name", serverName)
	props.Set("$mcp_server_version", serverVersion)
	c.enqueue(distinctID, "$mcp_initialize", props, sessionID == "")
}

// CaptureToolsList records $mcp_tools_list.
func (c *Client) CaptureToolsList(sessionID, clientName, clientVersion, protocolVersion, distinctID string, toolNames []string) {
	if !c.Enabled() {
		return
	}
	props := c.baseProps(sessionID, clientName, clientVersion, protocolVersion)
	props.Set("$mcp_listed_tool_names", toolNames)
	c.enqueue(distinctID, "$mcp_tools_list", props, sessionID == "")
}

// ToolCallEvent is the payload for a tools/call capture.
type ToolCallEvent struct {
	SessionID       string
	ClientName      string
	ClientVersion   string
	ProtocolVersion string
	DistinctID      string
	ToolName        string
	ToolDescription string
	Parameters      map[string]interface{}
	Response        interface{}
	DurationMs      int64
	IsError         bool
	ErrorMessage    string
	Intent          string
	IntentSource    string // "context_parameter" | "inferred"
}

// CaptureToolCall records $mcp_tool_call (and optional $exception sibling).
func (c *Client) CaptureToolCall(ev ToolCallEvent) {
	if !c.Enabled() {
		return
	}

	params, intent, intentSource := extractIntent(ev.Parameters)
	if ev.Intent != "" {
		intent = ev.Intent
		intentSource = ev.IntentSource
		if intentSource == "" {
			intentSource = "context_parameter"
		}
	} else if intent == "" {
		intent = "Invoking " + ev.ToolName
		intentSource = "inferred"
	}

	props := c.baseProps(ev.SessionID, ev.ClientName, ev.ClientVersion, ev.ProtocolVersion)
	props.Set("$mcp_tool_name", ev.ToolName)
	props.Set("$mcp_resource_name", ev.ToolName)
	if ev.ToolDescription != "" {
		props.Set("$mcp_tool_description", truncateString(ev.ToolDescription))
	}
	props.Set("$mcp_parameters", sanitizeValue(params, 0))
	props.Set("$mcp_response", sanitizeValue(ev.Response, 0))
	props.Set("$mcp_duration_ms", ev.DurationMs)
	props.Set("$mcp_is_error", ev.IsError)
	if intent != "" {
		props.Set("$mcp_intent", truncateString(intent))
		props.Set("$mcp_intent_source", intentSource)
	}
	if ev.IsError {
		props.Set("$mcp_error_type", classifyToolError(ev.ErrorMessage))
		if ev.ErrorMessage != "" {
			props.Set("$mcp_error_message", truncateString(ev.ErrorMessage))
		}
	}

	c.enqueue(ev.DistinctID, "$mcp_tool_call", props, ev.SessionID == "")

	if ev.IsError {
		exc := c.baseProps(ev.SessionID, ev.ClientName, ev.ClientVersion, ev.ProtocolVersion)
		exc.Set("$mcp_tool_name", ev.ToolName)
		exc.Set("$mcp_resource_name", ev.ToolName)
		exc.Set("$exception_level", "error")
		msg := ev.ErrorMessage
		if msg == "" {
			msg = "tool error: " + ev.ToolName
		}
		exc.Set("$exception_list", []map[string]interface{}{
			{
				"type":  "ToolError",
				"value": truncateString(msg),
				"mechanism": map[string]interface{}{
					"handled":   true,
					"synthetic": true,
				},
			},
		})
		c.enqueue(ev.DistinctID, "$exception", exc, ev.SessionID == "")
	}
}

// NewSessionID mints a ses_<32hex> id matching the TS SDK shape.
func NewSessionID() string {
	var b [16]byte
	_, _ = rand.Read(b[:])
	return "ses_" + hex.EncodeToString(b[:])
}

// SessionIDFromMCP normalizes an MCP protocol session id into analytics form.
func SessionIDFromMCP(raw string) string {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return ""
	}
	if strings.HasPrefix(raw, "ses_") {
		return raw
	}
	// Deterministic-ish label from protocol session (not a hash of secrets).
	return "ses_" + sanitizeSessionLabel(raw)
}

func (c *Client) baseProps(sessionID, clientName, clientVersion, protocolVersion string) posthog.Properties {
	props := posthog.NewProperties()
	props.Set("$mcp_source", sourceTag)
	props.Set("$mcp_server_name", serverName)
	props.Set("$mcp_server_version", serverVersion)
	if sessionID != "" {
		props.Set("$session_id", sessionID)
	}
	if clientName != "" {
		props.Set("$mcp_client_name", clientName)
	}
	if clientVersion != "" {
		props.Set("$mcp_client_version", clientVersion)
	}
	if protocolVersion != "" {
		props.Set("$mcp_protocol_version", protocolVersion)
	}
	if c.projectID != "" {
		props.Set("posthog_project_id", c.projectID)
	}
	return props
}

func (c *Client) enqueue(distinctID, event string, props posthog.Properties, anonymous bool) {
	if distinctID == "" {
		distinctID = "anonymous"
	}
	if anonymous {
		props.Set("$process_person_profile", false)
	}
	// Fire-and-forget: never block MCP on analytics.
	_ = c.ph.Enqueue(posthog.Capture{
		DistinctId: distinctID,
		Event:      event,
		Properties: props,
		Timestamp:  time.Now().UTC(),
	})
}

// classifyToolError maps tool error text to a coarse $mcp_error_type for dashboards.
func classifyToolError(msg string) string {
	lower := strings.ToLower(strings.TrimSpace(msg))
	switch {
	case lower == "":
		return "internal"
	case strings.Contains(lower, " 419 "), strings.HasSuffix(lower, ": 419"), strings.Contains(lower, "session expired"), strings.Contains(lower, "session has ended"):
		return "auth_session"
	case strings.Contains(lower, " 401 "), strings.Contains(lower, "not authenticated"), strings.Contains(lower, "invalid or missing authentication"), strings.Contains(lower, "cookie token is empty"):
		return "auth"
	case strings.Contains(lower, " 403 "), strings.Contains(lower, "not enough permission"), strings.Contains(lower, "forbidden"), strings.Contains(lower, "not allowed by the approved oauth"):
		return "permission"
	case strings.Contains(lower, "not found"), strings.Contains(lower, "does not exist"):
		return "not_found"
	case strings.Contains(lower, "is required"), strings.Contains(lower, "invalid arguments"), strings.Contains(lower, "unsupported "), strings.Contains(lower, "must be"):
		return "validation"
	case strings.Contains(lower, " 429 "), strings.Contains(lower, "rate limit"), strings.Contains(lower, "too many requests"):
		return "rate_limit"
	case strings.Contains(lower, " 5"), strings.Contains(lower, "timeout"), strings.Contains(lower, "temporarily unavailable"), strings.Contains(lower, "connection reset"):
		return "upstream"
	default:
		return "internal"
	}
}

// extractIntent pulls optional `context` / `intent` from tool args (PostHog
// convention) and returns remaining params for $mcp_parameters.
func extractIntent(params map[string]interface{}) (cleaned map[string]interface{}, intent, source string) {
	if params == nil {
		return map[string]interface{}{}, "", ""
	}
	cleaned = make(map[string]interface{}, len(params))
	for k, v := range params {
		lk := strings.ToLower(k)
		if lk == "context" || lk == "intent" || lk == "conversation_id" {
			if s, ok := v.(string); ok && strings.TrimSpace(s) != "" && intent == "" {
				intent = strings.TrimSpace(s)
				source = "context_parameter"
			}
			continue
		}
		cleaned[k] = v
	}
	return cleaned, intent, source
}

var sensitiveKeySubstrings = []string{
	"authorization", "password", "passwd", "secret", "token", "api_key", "apikey",
	"cookie", "set-cookie", "private_key", "access_key", "refresh_token",
	"client_secret", "bearer", "credential",
}

func sanitizeValue(v interface{}, depth int) interface{} {
	if depth > maxDepth {
		return "[truncated]"
	}
	switch t := v.(type) {
	case nil:
		return nil
	case string:
		return truncateString(t)
	case json.Number:
		return t
	case float64, float32, int, int32, int64, uint, uint32, uint64, bool:
		return t
	case map[string]interface{}:
		out := make(map[string]interface{}, len(t))
		for k, val := range t {
			if isSensitiveKey(k) {
				out[k] = "[redacted]"
				continue
			}
			out[k] = sanitizeValue(val, depth+1)
		}
		return out
	case []interface{}:
		out := make([]interface{}, 0, len(t))
		for _, item := range t {
			out = append(out, sanitizeValue(item, depth+1))
		}
		return out
	default:
		// Best-effort JSON round-trip for structs
		b, err := json.Marshal(t)
		if err != nil {
			return fmt.Sprintf("%v", t)
		}
		var generic interface{}
		if err := json.Unmarshal(b, &generic); err != nil {
			return truncateString(string(b))
		}
		return sanitizeValue(generic, depth+1)
	}
}

func isSensitiveKey(key string) bool {
	lk := strings.ToLower(key)
	for _, s := range sensitiveKeySubstrings {
		if strings.Contains(lk, s) {
			return true
		}
	}
	return false
}

func truncateString(s string) string {
	if len(s) <= maxStringBytes {
		return s
	}
	return s[:maxStringBytes] + "…[truncated]"
}

func sanitizeSessionLabel(raw string) string {
	var b strings.Builder
	for _, r := range raw {
		if (r >= 'a' && r <= 'z') || (r >= 'A' && r <= 'Z') || (r >= '0' && r <= '9') || r == '-' || r == '_' {
			b.WriteRune(r)
		}
		if b.Len() >= 32 {
			break
		}
	}
	if b.Len() == 0 {
		return NewSessionID()[4:] // strip ses_
	}
	return b.String()
}

func firstNonEmpty(vals ...string) string {
	for _, v := range vals {
		if strings.TrimSpace(v) != "" {
			return strings.TrimSpace(v)
		}
	}
	return ""
}

// envTruthyDefaultOn is true unless explicitly disabled.
func envTruthyDefaultOn(key string) bool {
	v := strings.ToLower(strings.TrimSpace(os.Getenv(key)))
	switch v {
	case "0", "false", "off", "no", "disabled":
		return false
	default:
		return true
	}
}

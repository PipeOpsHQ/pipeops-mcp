package analytics

import (
	"testing"
)

func TestExtractIntent(t *testing.T) {
	t.Parallel()
	cleaned, intent, source := extractIntent(map[string]interface{}{
		"workspace_id": "w1",
		"context":      "Deploy the staging API",
		"token":        "secret",
	})
	if intent != "Deploy the staging API" || source != "context_parameter" {
		t.Fatalf("intent=%q source=%q", intent, source)
	}
	if _, ok := cleaned["context"]; ok {
		t.Fatal("context should be stripped from parameters")
	}
	if cleaned["workspace_id"] != "w1" {
		t.Fatalf("workspace_id = %v", cleaned["workspace_id"])
	}
	// token stays in cleaned map; sanitizeValue redacts it later
	if cleaned["token"] != "secret" {
		t.Fatalf("token = %v", cleaned["token"])
	}
}

func TestSanitizeValueRedactsSecrets(t *testing.T) {
	t.Parallel()
	got := sanitizeValue(map[string]interface{}{
		"name":          "ok",
		"access_token":  "abc",
		"Authorization": "Bearer x",
		"nested": map[string]interface{}{
			"password": "p",
			"keep":     "yes",
		},
	}, 0).(map[string]interface{})
	if got["name"] != "ok" {
		t.Fatalf("name = %v", got["name"])
	}
	if got["access_token"] != "[redacted]" {
		t.Fatalf("access_token = %v", got["access_token"])
	}
	if got["Authorization"] != "[redacted]" {
		t.Fatalf("Authorization = %v", got["Authorization"])
	}
	nested := got["nested"].(map[string]interface{})
	if nested["password"] != "[redacted]" || nested["keep"] != "yes" {
		t.Fatalf("nested = %#v", nested)
	}
}

func TestSessionIDFromMCP(t *testing.T) {
	t.Parallel()
	if got := SessionIDFromMCP("ses_abc"); got != "ses_abc" {
		t.Fatalf("got %q", got)
	}
	if got := SessionIDFromMCP("abc-123"); !stringsHasPrefix(got, "ses_") {
		t.Fatalf("got %q", got)
	}
}

func stringsHasPrefix(s, p string) bool {
	return len(s) >= len(p) && s[:len(p)] == p
}

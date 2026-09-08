package agents

import (
	"encoding/json"
	"strings"
	"testing"

	"github.com/JustinNguyen9979/ainovel-cli/internal/bootstrap"
	"github.com/voocel/agentcore"
)

func TestAgentPureHelpers(t *testing.T) {
	if agentToRole("architect_long") != "architect" || agentToRole("writer") != "writer" {
		t.Fatal("agent role normalization failed")
	}
	if !strings.HasPrefix(promptCacheBase("/tmp/book"), "nvl-") || promptCacheBase("/tmp/book") == promptCacheBase("/tmp/other") {
		t.Fatal("prompt cache base should be stable and directory-specific")
	}
	for _, value := range []string{"", "off", "low", "medium", "high", "xhigh", "max"} {
		if _, err := ParseThinkingLevel(value); err != nil {
			t.Fatalf("thinking level %q: %v", value, err)
		}
	}
	if _, err := ParseThinkingLevel("invalid"); err == nil {
		t.Fatal("invalid thinking level should fail")
	}
	cfg := bootstrap.Config{ReasoningEffort: "high", Roles: map[string]bootstrap.RoleConfig{"writer": {ReasoningEffort: "low"}}}
	if roleThinking(cfg, "writer") == "" || roleThinking(cfg, "editor") == "" {
		t.Fatal("role thinking resolution failed")
	}
}

func TestFoundationStopResultHelpers(t *testing.T) {
	ready, _ := json.Marshal(map[string]any{"foundation_ready": true})
	if !foundationReadyResult("audit_foundation", ready) || foundationReadyResult("save_foundation", ready) {
		t.Fatal("foundation result detection failed")
	}
	if !architectLongShouldStopAfterToolResult("save_foundation", []byte(`{"type":"expand_arc"}`)) || !architectLongShouldStopAfterToolResult("save_foundation", []byte(`{"type":"complete_book"}`)) {
		t.Fatal("architect terminal result detection failed")
	}
	if architectLongShouldStopAfterToolResult("save_foundation", []byte(`{"type":"save_book"}`)) || decodeSaveFoundationResult("other", nil).Type != "" {
		t.Fatal("non-terminal result should not stop")
	}
}

var _ agentcore.ThinkingLevel

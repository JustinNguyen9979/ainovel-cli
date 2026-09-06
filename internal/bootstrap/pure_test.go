package bootstrap

import (
	"errors"
	"reflect"
	"testing"

	"github.com/JustinNguyen9979/ainovel-cli/internal/errs"
)

func TestCompactReserveTokens(t *testing.T) {
	if CompactReserveTokens(0) != 0 || CompactReserveTokens(-1) != 0 {
		t.Fatal("non-positive window should have no reserve")
	}
	if CompactReserveTokens(10000) != MinCompactReserve {
		t.Fatal("small window should use reserve floor")
	}
	if got := CompactReserveTokens(100000); got != 15000 {
		t.Fatalf("large window reserve = %d", got)
	}
}

func TestNotificationAndProviderHelpers(t *testing.T) {
	var enabled *bool
	if !(NotifyConfig{Enabled: enabled}).IsEnabled() || !(BudgetConfig{BookUSD: 1}).Enabled() || (BudgetConfig{}).Enabled() {
		t.Fatal("notification/budget defaults are wrong")
	}
	falseValue := false
	if (NotifyConfig{Enabled: &falseValue}).IsEnabled() {
		t.Fatal("explicit disabled notification should be false")
	}
	for _, tc := range []struct {
		name string
		pc   ProviderConfig
		want bool
	}{
		{"ollama", ProviderConfig{}, false},
		{"bedrock", ProviderConfig{}, false},
		{"custom", ProviderConfig{Type: "openai"}, false},
		{"openai", ProviderConfig{}, true},
	} {
		if got := tc.pc.RequiresAPIKey(tc.name); got != tc.want {
			t.Fatalf("RequiresAPIKey(%q) = %v", tc.name, got)
		}
	}
	if got, err := (ProviderConfig{Type: "openai"}).ProviderType("custom"); err != nil || got != "openai" {
		t.Fatalf("custom provider type = %q, %v", got, err)
	}
	if _, err := (ProviderConfig{}).ProviderType("unknown"); !errors.Is(err, errs.ErrConfig) {
		t.Fatalf("unknown provider should return config error: %v", err)
	}
}

func TestFillDefaultsCandidateModelsAndClone(t *testing.T) {
	cfg := Config{Provider: "p", ModelName: "current", Budget: BudgetConfig{BookUSD: 5}, Providers: map[string]ProviderConfig{"p": {Models: []ModelConfig{{Name: "one"}, {Name: "one"}, {Name: "two"}}}}, Roles: map[string]RoleConfig{"writer": {Provider: "p", Model: "writer", Fallbacks: []ModelRef{{Provider: "p", Model: "fallback"}}}}}
	cfg.FillDefaults()
	if cfg.Language != "vi" || cfg.Style != "default" || cfg.OutputDir == "" || cfg.Budget.WarnRatio != 0.8 {
		t.Fatalf("defaults = %+v", cfg)
	}
	if got := cfg.CandidateModels("p"); !reflect.DeepEqual(got, []string{"one", "two", "current", "writer", "fallback"}) {
		t.Fatalf("candidate models = %v", got)
	}
	if cfg.CandidateModels("") != nil {
		t.Fatal("empty provider should have no candidates")
	}
	clone := CloneConfig(cfg)
	clone.Providers["p"].Models[0].Name = "changed"
	clone.Roles["writer"] = RoleConfig{Model: "changed"}
	if cfg.Providers["p"].Models[0].Name != "one" || cfg.Roles["writer"].Model != "writer" {
		t.Fatal("CloneConfig did not isolate maps")
	}
}

func TestResolveContextWindow(t *testing.T) {
	cfg := Config{ContextWindow: 300000, Providers: map[string]ProviderConfig{"p": {Models: []ModelConfig{{Name: "local", ContextWindow: 12345}}}}}
	if got, source := cfg.ResolveContextWindow("p", "local"); got != 12345 || source != CtxWindowModelConfig {
		t.Fatalf("model window = %d/%s", got, source)
	}
	if got, source := cfg.ResolveContextWindow("other", "unknown"); got != 300000 || source != CtxWindowConfig {
		t.Fatalf("config window = %d/%s", got, source)
	}
}

func TestStripJSONComments(t *testing.T) {
	input := []byte("{\n // comment\n \"url\": \"https://example.test/a//b\"\n}")
	got := string(stripJSONComments(input))
	if !reflect.DeepEqual(got, "{\n \n \"url\": \"https://example.test/a//b\"\n}") {
		t.Fatalf("stripped JSON = %q", got)
	}
}

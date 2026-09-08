package bootstrap

import (
	"errors"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/JustinNguyen9979/ainovel-cli/internal/errs"
	"github.com/JustinNguyen9979/ainovel-cli/internal/notify"
)

func TestConfigValidationBranches(t *testing.T) {
	base := func() Config {
		return Config{Provider: "openrouter", ModelName: "model", Providers: map[string]ProviderConfig{"openrouter": {APIKey: "key"}}}
	}
	cases := []struct {
		name string
		edit func(*Config)
	}{
		{"missing provider", func(c *Config) { c.Provider = "" }},
		{"missing model", func(c *Config) { c.ModelName = "" }},
		{"unknown provider", func(c *Config) { c.Provider = "unknown" }},
		{"control provider", func(c *Config) { c.Provider = "bad\nname" }},
		{"unknown role", func(c *Config) { c.Roles = map[string]RoleConfig{"bad": {Provider: "openrouter", Model: "model"}} }},
		{"empty role model", func(c *Config) { c.Roles = map[string]RoleConfig{"writer": {Provider: "openrouter"}} }},
		{"bad budget", func(c *Config) { c.Budget = BudgetConfig{BookUSD: -1} }},
		{"bad warning ratio", func(c *Config) { c.Budget = BudgetConfig{BookUSD: 1, WarnRatio: 1} }},
		{"bad notification", func(c *Config) { c.Notify.Events = []string{"unknown"} }},
		{"duplicate model", func(c *Config) {
			c.Providers["openrouter"] = ProviderConfig{APIKey: "key", Models: []ModelConfig{{Name: "model"}, {Name: "model"}}}
		}},
		{"negative window", func(c *Config) {
			c.Providers["openrouter"] = ProviderConfig{APIKey: "key", Models: []ModelConfig{{Name: "model", ContextWindow: -1}}}
		}},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			cfg := base()
			tc.edit(&cfg)
			if err := cfg.ValidateBase(); err == nil || !errors.Is(err, errs.ErrConfig) {
				t.Fatalf("ValidateBase = %v", err)
			}
		})
	}
	valid := base()
	if err := valid.ValidateBase(); err != nil {
		t.Fatalf("valid config rejected: %v", err)
	}
	cfg := base()
	cfg.Providers["proxy"] = ProviderConfig{Type: "openai", API: "responses"}
	if got, err := cfg.Providers["proxy"].ProviderType("proxy"); err != nil || got != "openai" {
		t.Fatalf("provider type = %q/%v", got, err)
	}
	if cfg.Providers["openrouter"].RequiresAPIKey("ollama") {
		t.Fatal("ollama should not require key")
	}
	if !(NotifyConfig{}).IsEnabled() {
		t.Fatal("nil notify enabled should default true")
	}
	if !notify.IsKnownKind(notify.Kinds()[0]) {
		t.Fatal("notify kind fixture invalid")
	}
}

func TestConfigMergeCloneAndWindows(t *testing.T) {
	base := Config{Provider: "p", ModelName: "m", Style: "old", ContextWindow: 100, Providers: map[string]ProviderConfig{"p": {APIKey: "key", Models: []ModelConfig{{Name: "m"}}}}, Roles: map[string]RoleConfig{"writer": {Provider: "p", Model: "m"}}}
	overlay := Config{ModelName: "new", Language: "vi", Style: "new", ContextWindow: 200, Providers: map[string]ProviderConfig{"p": {BaseURL: "url", Extra: map[string]any{"x": 1}}}, Roles: map[string]RoleConfig{"writer": {Model: "new", ReasoningEffort: "high"}}, Budget: BudgetConfig{BookUSD: 5}, Notify: NotifyConfig{Enabled: boolPtr(false)}}
	merged := mergeConfig(base, overlay)
	if merged.ModelName != "new" || merged.Style != "new" || merged.ContextWindow != 200 || merged.Providers["p"].APIKey != "key" || merged.Roles["writer"].Provider != "p" || !merged.Budget.Enabled() || merged.Notify.IsEnabled() {
		t.Fatalf("merged = %+v", merged)
	}
	clone := CloneConfig(merged)
	clone.Providers["p"] = ProviderConfig{Models: []ModelConfig{{Name: "changed"}}, Extra: map[string]any{"x": 2}}
	clone.Roles["writer"] = RoleConfig{Fallbacks: []ModelRef{{Provider: "x", Model: "y"}}}
	if merged.Providers["p"].Models[0].Name == "changed" || merged.Roles["writer"].Fallbacks != nil {
		t.Fatal("clone mutated original")
	}
	cfg := Config{ContextWindow: 123}
	if got, source := cfg.ResolveContextWindow("p", "m"); got != 123 || source != CtxWindowConfig {
		t.Fatalf("config window = %d/%s", got, source)
	}
	if got, source := (Config{}).ResolveContextWindow("p", "completely-unknown-model"); got != DefaultContextWindow || source != CtxWindowDefault {
		t.Fatalf("default window = %d/%s", got, source)
	}
	cfg = Config{Provider: "p", ModelName: "m", Providers: map[string]ProviderConfig{"p": {Models: []ModelConfig{{Name: "m"}}}}, Roles: map[string]RoleConfig{"writer": {Provider: "p", Model: "m", Fallbacks: []ModelRef{{Provider: "p", Model: "fallback"}}}}}
	if got := cfg.CandidateModels("p"); len(got) != 2 {
		t.Fatalf("candidate models = %#v", got)
	}
}

func boolPtr(v bool) *bool { return &v }

func TestConfigPureBranches(t *testing.T) {
	for _, tc := range []struct {
		window int
		want   int
	}{
		{0, 0}, {-1, 0}, {1000, MinCompactReserve}, {100000, 15000},
	} {
		if got := CompactReserveTokens(tc.window); got != tc.want {
			t.Errorf("CompactReserveTokens(%d) = %d, want %d", tc.window, got, tc.want)
		}
	}
	for _, tc := range []struct {
		name string
		cfg  NotifyConfig
		want bool
	}{
		{"nil", NotifyConfig{}, true},
		{"enabled", NotifyConfig{Enabled: boolPtr(true)}, true},
		{"disabled", NotifyConfig{Enabled: boolPtr(false)}, false},
	} {
		if got := tc.cfg.IsEnabled(); got != tc.want {
			t.Errorf("%s IsEnabled = %v, want %v", tc.name, got, tc.want)
		}
	}
	if got, err := (ProviderConfig{Type: "custom"}).ProviderType("unknown"); err != nil || got != "custom" {
		t.Fatalf("explicit provider type = %q/%v", got, err)
	}
	if _, err := (ProviderConfig{}).ProviderType("unknown"); err == nil {
		t.Fatal("unknown provider without type should fail")
	}
	if got, err := (ProviderConfig{}).StreamIdleTimeoutValue(); err != nil || got != defaultStreamIdleTimeout {
		t.Fatalf("default idle timeout = %s/%v", got, err)
	}
	for _, raw := range []string{"0", "-1s", "bad"} {
		if _, err := (ProviderConfig{StreamIdleTimeout: raw}).StreamIdleTimeoutValue(); err == nil {
			t.Fatalf("invalid timeout %q accepted", raw)
		}
	}
	if got, err := (ProviderConfig{StreamIdleTimeout: "2s"}).StreamIdleTimeoutValue(); err != nil || got != 2*time.Second {
		t.Fatalf("explicit idle timeout = %s/%v", got, err)
	}
}

func TestNeedsSetupChecksGlobalAndProjectConfig(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	t.Setenv("USERPROFILE", home)
	project := t.TempDir()
	t.Chdir(project)
	if !NeedsSetup() {
		t.Fatal("missing global and project config should require setup")
	}
	if err := os.MkdirAll(filepath.Join(home, ".ainovel"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(home, ".ainovel", "config.json"), []byte("{}"), 0o600); err != nil {
		t.Fatal(err)
	}
	if NeedsSetup() {
		t.Fatal("global config should disable setup")
	}
	if err := os.Remove(filepath.Join(home, ".ainovel", "config.json")); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(filepath.Join(project, ".ainovel"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(project, ".ainovel", "config.json"), []byte("{}"), 0o600); err != nil {
		t.Fatal(err)
	}
	if NeedsSetup() {
		t.Fatal("project config should disable setup")
	}
}

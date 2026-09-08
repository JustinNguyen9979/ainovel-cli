package bootstrap

import (
	"context"
	"errors"
	"reflect"
	"strings"
	"testing"

	"github.com/JustinNguyen9979/ainovel-cli/internal/errs"
	"github.com/voocel/agentcore"
	"github.com/voocel/agentcore/llm"
)

// These small models exercise the optional metadata interfaces without making
// network requests or depending on a provider adapter's implementation details.
type bootstrapBranchPlainModel struct{}

func (bootstrapBranchPlainModel) Generate(context.Context, []agentcore.Message, []agentcore.ToolSpec, ...agentcore.CallOption) (*agentcore.LLMResponse, error) {
	return nil, nil
}

func (bootstrapBranchPlainModel) GenerateStream(context.Context, []agentcore.Message, []agentcore.ToolSpec, ...agentcore.CallOption) (<-chan agentcore.StreamEvent, error) {
	return nil, nil
}

func (bootstrapBranchPlainModel) SupportsTools() bool { return true }

type bootstrapBranchCapabilitiesModel struct {
	bootstrapBranchPlainModel
	caps llm.Capabilities
}

func (m bootstrapBranchCapabilitiesModel) Capabilities() llm.Capabilities { return m.caps }

type bootstrapBranchInfoModel struct {
	bootstrapBranchPlainModel
	info llm.ModelInfo
}

func (m bootstrapBranchInfoModel) Info() llm.ModelInfo { return m.info }

type bootstrapBranchProviderModel struct {
	bootstrapBranchPlainModel
	provider string
}

func (m bootstrapBranchProviderModel) ProviderName() string { return m.provider }

func TestSwappableModelFactsUseOptionalMetadataAndIdentityFallbacks(t *testing.T) {
	wantCaps := llm.Capabilities{
		Provider: "adapter-provider",
		Model:    "adapter-model",
		Structured: llm.StructuredCapabilities{
			JSONSchema: llm.SupportYes,
		},
	}
	model := NewSwappableModel(
		"configured-provider",
		"configured-model",
		bootstrapBranchCapabilitiesModel{caps: wantCaps},
		nil,
	)

	if got := model.ProviderName(); got != "configured-provider" {
		t.Fatalf("ProviderName() = %q, want configured-provider", got)
	}
	if got := model.Capabilities(); !reflect.DeepEqual(got, wantCaps) {
		t.Fatalf("Capabilities() = %+v, want %+v", got, wantCaps)
	}
	if got := model.Info(); got.Name != "configured-model" || got.Provider != "configured-provider" {
		t.Fatalf("Info() without inner Info = %+v", got)
	}

	// An inner Info implementation can override either identity field; blank
	// fields must still use the atomically selected wrapper identity.
	model.Swap(
		"swapped-provider",
		"swapped-model",
		bootstrapBranchInfoModel{info: llm.ModelInfo{}},
		nil,
	)
	facts := model.StructuredOutputFacts()
	if facts.Info.Name != "swapped-model" || facts.Info.Provider != "swapped-provider" {
		t.Fatalf("empty inner Info should preserve wrapper identity: %+v", facts.Info)
	}
	if got := model.Capabilities(); !reflect.DeepEqual(got, llm.Capabilities{}) {
		t.Fatalf("model without CapabilityProvider should have zero capabilities: %+v", got)
	}

	model.Swap("final-provider", "final-model", bootstrapBranchPlainModel{}, nil)
	if got := model.Info(); got.Name != "final-model" || got.Provider != "final-provider" {
		t.Fatalf("Info() without optional interfaces = %+v", got)
	}

	model.Swap(
		"info-provider",
		"info-model",
		bootstrapBranchInfoModel{info: llm.ModelInfo{Provider: "inner-provider"}},
		nil,
	)
	if got := model.Info(); got.Name != "info-model" || got.Provider != "inner-provider" {
		t.Fatalf("non-empty inner Info should override wrapper identity: %+v", got)
	}
	model.Swap("final-provider", "final-model", bootstrapBranchPlainModel{}, nil)
	if got := model.Info(); got.Name != "final-model" || got.Provider != "final-provider" {
		t.Fatalf("Info() without optional interfaces = %+v", got)
	}
}

func TestModelProviderFallsBackToProviderName(t *testing.T) {
	providerOnly := bootstrapBranchProviderModel{provider: "provider-only"}
	if got := ModelProvider(providerOnly); got != "provider-only" {
		t.Fatalf("ModelProvider(provider-only) = %q, want provider-only", got)
	}
	if got := ModelProvider(bootstrapBranchPlainModel{}); got != "" {
		t.Fatalf("ModelProvider(plain) = %q, want empty string", got)
	}
	withInfo := bootstrapBranchInfoModel{info: llm.ModelInfo{Provider: "info-provider"}}
	if got := ModelProvider(withInfo); got != "info-provider" {
		t.Fatalf("ModelProvider(info) = %q, want info-provider", got)
	}
}

func TestModelSetSwapErrorsAndRoleUpdates(t *testing.T) {
	cfg := Config{
		Provider:  "primary",
		ModelName: "primary-model",
		Providers: map[string]ProviderConfig{
			"primary": {
				Type:    "openai",
				APIKey:  "test-key",
				BaseURL: "https://example.com/v1",
			},
			"secondary": {
				Type:    "openai",
				APIKey:  "test-key",
				BaseURL: "https://example.com/v1",
			},
			"broken": {
				APIKey: "test-key",
			},
		},
	}
	models, err := NewModelSet(cfg)
	if err != nil {
		t.Fatalf("NewModelSet() = %v", err)
	}

	for _, tc := range []struct {
		name     string
		provider string
		role     string
		wantText string
	}{
		{name: "unknown provider", provider: "missing", wantText: "not configured"},
		{name: "invalid provider configuration", provider: "broken", wantText: "切换模型失败"},
		{name: "unknown role", provider: "secondary", role: "reviewer", wantText: "unknown role"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			err := models.Swap(tc.role, tc.provider, "candidate-model")
			if err == nil || !errors.Is(err, errs.ErrConfig) {
				t.Fatalf("Swap() error = %v, want wrapped ErrConfig", err)
			}
			if !strings.Contains(err.Error(), tc.wantText) {
				t.Fatalf("Swap() error = %q, want substring %q", err, tc.wantText)
			}
		})
	}
	if provider, model, explicit := models.CurrentSelection("default"); provider != "primary" || model != "primary-model" || !explicit {
		t.Fatalf("failed swaps changed default selection: %s/%s/%v", provider, model, explicit)
	}

	if err := models.Swap("", "secondary", "default-next"); err != nil {
		t.Fatalf("Swap() default with empty role: %v", err)
	}
	if provider, model, explicit := models.CurrentSelection(""); provider != "secondary" || model != "default-next" || !explicit {
		t.Fatalf("empty-role default selection = %s/%s/%v", provider, model, explicit)
	}
	if models.config.Provider != "secondary" || models.config.ModelName != "default-next" {
		t.Fatalf("config selection not updated: %s/%s", models.config.Provider, models.config.ModelName)
	}

	if err := models.Swap("writer", "secondary", "writer-first"); err != nil {
		t.Fatalf("Swap() new role: %v", err)
	}
	if provider, model, explicit := models.CurrentSelection("writer"); provider != "secondary" || model != "writer-first" || !explicit {
		t.Fatalf("new role selection = %s/%s/%v", provider, model, explicit)
	}
	if models.config.Roles["writer"].Provider != "secondary" || models.config.Roles["writer"].Model != "writer-first" {
		t.Fatalf("new role config not persisted: %+v", models.config.Roles["writer"])
	}

	if err := models.Swap("writer", "primary", "writer-second"); err != nil {
		t.Fatalf("Swap() existing role: %v", err)
	}
	if provider, model, explicit := models.CurrentSelection("writer"); provider != "primary" || model != "writer-second" || !explicit {
		t.Fatalf("existing role selection = %s/%s/%v", provider, model, explicit)
	}
}

func TestValidateBaseRejectsInvalidLanguageAndControlCharacters(t *testing.T) {
	base := func() Config {
		return Config{
			Provider:  "openrouter",
			ModelName: "model",
			Providers: map[string]ProviderConfig{
				"openrouter": {APIKey: "test-key"},
			},
		}
	}

	invalidLanguage := base()
	invalidLanguage.Language = "fr"
	if err := invalidLanguage.ValidateBase(); err == nil || !errors.Is(err, errs.ErrConfig) {
		t.Fatalf("invalid language error = %v, want wrapped ErrConfig", err)
	}

	for _, tc := range []struct {
		name string
		edit func(*Config)
	}{
		{name: "provider", edit: func(c *Config) { c.Provider = "openrouter\n" }},
		{name: "model", edit: func(c *Config) { c.ModelName = "model\x00" }},
		{name: "provider field", edit: func(c *Config) {
			pc := c.Providers["openrouter"]
			pc.BaseURL = "https://example.com/\n"
			c.Providers["openrouter"] = pc
		}},
		{name: "provider name", edit: func(c *Config) { c.Providers["bad\nname"] = ProviderConfig{APIKey: "test-key"} }},
		{name: "role name", edit: func(c *Config) { c.Roles = map[string]RoleConfig{"writer\n": {Provider: "openrouter", Model: "model"}} }},
		{name: "role fallback provider", edit: func(c *Config) {
			c.Roles = map[string]RoleConfig{"writer": {
				Provider: "openrouter", Model: "model",
				Fallbacks: []ModelRef{{Provider: "openrouter\n", Model: "fallback"}},
			}}
		}},
		{name: "role fallback model", edit: func(c *Config) {
			c.Roles = map[string]RoleConfig{"writer": {
				Provider: "openrouter", Model: "model",
				Fallbacks: []ModelRef{{Provider: "openrouter", Model: "fallback\t"}},
			}}
		}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			cfg := base()
			tc.edit(&cfg)
			err := cfg.ValidateBase()
			if err == nil || !errors.Is(err, errs.ErrConfig) || !strings.Contains(err.Error(), "control character") {
				t.Fatalf("ValidateBase() = %v, want control-character ErrConfig", err)
			}
		})
	}
}

func TestValidateBaseAcceptsRoleFallback(t *testing.T) {
	cfg := Config{
		Provider:  "openrouter",
		ModelName: "primary",
		Providers: map[string]ProviderConfig{
			"openrouter": {APIKey: "test-key"},
		},
		Roles: map[string]RoleConfig{
			"writer": {
				Provider:  "openrouter",
				Model:     "primary",
				Fallbacks: []ModelRef{{Provider: "openrouter", Model: "fallback"}},
			},
		},
	}
	if err := cfg.ValidateBase(); err != nil {
		t.Fatalf("valid role fallback rejected: %v", err)
	}
}

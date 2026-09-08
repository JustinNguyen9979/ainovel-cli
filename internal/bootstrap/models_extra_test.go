package bootstrap

import "testing"

func testModelConfig() Config {
	return Config{
		Provider:  "openrouter",
		ModelName: "default-model",
		Providers: map[string]ProviderConfig{"openrouter": {APIKey: "sk-test", Models: []ModelConfig{{Name: "default-model", ContextWindow: 123456}}}},
		Roles:     map[string]RoleConfig{"writer": {Provider: "openrouter", Model: "writer-model"}},
	}
}

func TestModelSetSelectionAndSummary(t *testing.T) {
	ms, err := NewModelSet(testModelConfig())
	if err != nil {
		t.Fatal(err)
	}
	if ms.ForRole("writer") == nil || ms.ForRole("missing") == nil || ms.ForRoleWithFailover("missing", nil) == nil {
		t.Fatal("role fallback returned nil")
	}
	if provider, model, explicit := ms.CurrentSelection("writer"); provider != "openrouter" || model != "writer-model" || !explicit {
		t.Fatalf("writer selection = %s/%s/%v", provider, model, explicit)
	}
	if provider, model, explicit := ms.CurrentSelection("unknown"); provider != "openrouter" || model != "default-model" || explicit {
		t.Fatalf("fallback selection = %s/%s/%v", provider, model, explicit)
	}
	window, _ := ms.ResolveContextWindow("openrouter", "default-model")
	if ms.Summary() == "" || window != 123456 {
		t.Fatalf("summary/window = %q/%d", ms.Summary(), window)
	}
	if ModelName(ms.Default) != "default-model" || ModelProvider(ms.Default) != "openrouter" {
		t.Fatal("model identity helpers failed")
	}
}

func TestModelSetApplyPreparedPreservesSwappableModels(t *testing.T) {
	base, err := NewModelSet(testModelConfig())
	if err != nil {
		t.Fatal(err)
	}
	candidateCfg := testModelConfig()
	candidateCfg.ModelName = "candidate-model"
	candidate, err := NewModelSet(candidateCfg)
	if err != nil {
		t.Fatal(err)
	}
	original := base.Default
	base.ApplyPrepared(candidate)
	if base.Default != original {
		t.Fatal("ApplyPrepared should preserve existing default wrapper")
	}
	if provider, model := base.Default.Current(); provider != "openrouter" || model != "candidate-model" {
		t.Fatalf("applied default = %s/%s", provider, model)
	}
	if base.fallbackTargets("missing") != nil {
		t.Fatal("missing fallback targets should be nil")
	}
}

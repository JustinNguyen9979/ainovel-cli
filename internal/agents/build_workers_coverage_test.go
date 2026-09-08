package agents

import (
	"testing"

	"github.com/JustinNguyen9979/ainovel-cli/assets"
	"github.com/JustinNguyen9979/ainovel-cli/internal/bootstrap"
	"github.com/JustinNguyen9979/ainovel-cli/internal/store"
	"github.com/JustinNguyen9979/ainovel-cli/internal/tools"
)

func TestBuildWorkersAssemblesAllRoles(t *testing.T) {
	cfg := bootstrap.Config{
		OutputDir: t.TempDir(), Provider: "proxy", ModelName: "local-model", Style: "default",
		Providers: map[string]bootstrap.ProviderConfig{"proxy": {
			Type: "openai", API: "chat", APIKey: "local", BaseURL: "http://127.0.0.1:1/v1",
			Models: []bootstrap.ModelConfig{{Name: "local-model", ContextWindow: 128000}},
		}},
		Roles: map[string]bootstrap.RoleConfig{
			"architect": {Provider: "proxy", Model: "local-model"},
			"writer":    {Provider: "proxy", Model: "local-model"},
			"editor":    {Provider: "proxy", Model: "local-model"},
		},
	}
	st := store.NewStore(cfg.OutputDir)
	if err := st.Init(); err != nil {
		t.Fatal(err)
	}
	if err := st.RunMeta.Init(cfg.Style, cfg.Provider, cfg.ModelName); err != nil {
		t.Fatal(err)
	}
	models, err := bootstrap.NewModelSet(cfg)
	if err != nil {
		t.Fatal(err)
	}
	bundle := assets.Load(cfg.Style, assets.LoadOptions{})
	runner, restore, apply := BuildWorkers(cfg, st, tools.NewStyleStatsIndex(st), models, bundle, nil, nil)
	if runner == nil || restore == nil || apply == nil {
		t.Fatal("BuildWorkers returned nil component")
	}
	apply("architect", "low")
	apply("writer", "high")
	apply("editor", "off")
	apply("unknown", "max")
}

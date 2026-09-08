package host

import (
	"context"
	"strings"
	"testing"

	"github.com/JustinNguyen9979/ainovel-cli/assets"
	"github.com/JustinNguyen9979/ainovel-cli/internal/bootstrap"
	"github.com/JustinNguyen9979/ainovel-cli/internal/domain"
	"github.com/JustinNguyen9979/ainovel-cli/internal/host/exp"
	"github.com/JustinNguyen9979/ainovel-cli/internal/host/imp"
)

func TestNewHostWithLocalProviderBuildsRuntime(t *testing.T) {
	cfg := bootstrap.Config{
		OutputDir: t.TempDir(),
		Provider:  "proxy",
		ModelName: "local-model",
		Style:     "default",
		Providers: map[string]bootstrap.ProviderConfig{
			"proxy": {
				Type:    "openai",
				API:     "chat",
				APIKey:  "local-key",
				BaseURL: "http://127.0.0.1:1/v1",
				Models:  []bootstrap.ModelConfig{{Name: "local-model", ContextWindow: 128000}},
			},
		},
		Roles: map[string]bootstrap.RoleConfig{
			"writer": {Provider: "proxy", Model: "local-model"},
			"editor": {Provider: "proxy", Model: "local-model"},
		},
	}
	h, err := New(cfg, assets.Load("default", assets.LoadOptions{}))
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	defer h.Close()

	if h.Dir() != cfg.OutputDir || h.Events() == nil || h.Stream() == nil || h.Done() == nil {
		t.Fatalf("host channels or directory not initialized: dir=%q", h.Dir())
	}
	if h.FileLogError() != nil {
		t.Fatalf("unexpected file log error: %v", h.FileLogError())
	}
	if provider, model, explicit := h.CurrentModelSelection("default"); !explicit || provider != "proxy" || model != "local-model" {
		t.Fatalf("default selection = %s/%s/%v", provider, model, explicit)
	}
	if got := h.ConfiguredProviders(); len(got) != 1 || got[0] != "proxy" {
		t.Fatalf("providers = %#v", got)
	}
	if got := h.ConfiguredModels("proxy"); len(got) != 1 || got[0] != "local-model" {
		t.Fatalf("models = %#v", got)
	}
	if snap := h.Snapshot(); snap.ModelName != "local-model" || snap.ModelContextWindow != 128000 || snap.StatusLabel != "READY" {
		t.Fatalf("initial snapshot = %+v", snap)
	}
	if err := h.store.Progress.Save(&domain.Progress{Phase: domain.PhaseWriting, Flow: domain.FlowWriting, CurrentChapter: 1, TotalChapters: 2}); err != nil {
		t.Fatal(err)
	}
	if snap := h.Snapshot(); snap.Phase != string(domain.PhaseWriting) || snap.StatusLabel != "READY" {
		t.Fatalf("writing snapshot = %+v", snap)
	}
}

func TestHostOfflineAPIsAndGuards(t *testing.T) {
	cfg := bootstrap.Config{
		OutputDir: t.TempDir(), Provider: "proxy", ModelName: "local-model", Style: "default",
		Providers: map[string]bootstrap.ProviderConfig{"proxy": {Type: "openai", API: "chat", APIKey: "local-key", BaseURL: "http://127.0.0.1:1/v1", Models: []bootstrap.ModelConfig{{Name: "local-model", ContextWindow: 128000}}}},
	}
	h, err := New(cfg, assets.Load("default", assets.LoadOptions{}))
	if err != nil {
		t.Fatal(err)
	}
	defer h.Close()
	if got := h.ImportResumeHint(); got != "" {
		t.Fatalf("empty import hint = %q", got)
	}
	if got := h.ImportResumeHint(); got != "" {
		t.Fatalf("empty import hint repeated = %q", got)
	}
	if refs := h.ModelConfiguration().ReferencesFor("proxy", "local-model"); len(refs) == 0 || refs[0] != "default" {
		t.Fatalf("default model references = %v", refs)
	}
	if options := h.ConfiguredModelOptions("proxy"); len(options) != 1 || options[0].Name != "local-model" {
		t.Fatalf("configured model options = %+v", options)
	}
	if err := h.PrepareUserRules("  用户偏好：短句。 "); err != nil {
		t.Fatalf("PrepareUserRules: %v", err)
	}
	h.ensureUserRules()
	if rules, err := h.store.UserRules.Load(); err != nil || rules == nil {
		t.Fatalf("user rules snapshot = %+v/%v", rules, err)
	}
	if err := h.StartPrepared(" "); err == nil || !strings.Contains(err.Error(), "prompt is required") {
		t.Fatal("empty StartPrepared should fail")
	}
	if err := h.SetAdvanceMode(domain.ChapterAdvanceReview); err != nil {
		t.Fatal(err)
	}
	if err := h.SetAdvanceMode(domain.ChapterAdvanceAuto); err != nil {
		t.Fatal(err)
	}
	if _, err := h.Export(context.Background(), exp.Options{Format: exp.FormatTXT, OutPath: t.TempDir() + "/book.txt"}); err == nil {
		t.Fatal("export without completed chapters should fail")
	}
}

func TestHostGuardAndImportRuntimeHelpers(t *testing.T) {
	cfg := bootstrap.Config{OutputDir: t.TempDir(), Provider: "proxy", ModelName: "local-model", Providers: map[string]bootstrap.ProviderConfig{"proxy": {Type: "openai", APIKey: "key", Models: []bootstrap.ModelConfig{{Name: "local-model", ContextWindow: 128000}}}}}
	h, err := New(cfg, assets.Load("default", assets.LoadOptions{}))
	if err != nil {
		t.Fatal(err)
	}
	defer h.Close()
	if err := h.acquireExclusive("job"); err != nil {
		t.Fatal(err)
	}
	if _, err := h.ImportFrom(context.Background(), imp.Options{}); err == nil || !strings.Contains(err.Error(), "job") {
		t.Fatal("import should be blocked by exclusive job")
	}
	h.releaseExclusive()
	caller := h.importCaller("segment")
	if caller.Model == nil || caller.Runtime.ContextTokens <= 0 {
		t.Fatalf("import caller runtime = %+v", caller.Runtime)
	}
	if got := h.importModelRuntime("architect", h.models.Default); got.ContextTokens <= 0 {
		t.Fatalf("import model runtime = %+v", got)
	}
	if event := newInterventionFailureEvent(context.Canceled); event.Category != "ERROR" || event.Kind == "" {
		t.Fatalf("intervention error event = %+v", event)
	}
	if h.startEngine(nil) == false {
		t.Fatal("empty host should allow engine start")
	}
	h.engine.abort()
	h.engine.wait()
}

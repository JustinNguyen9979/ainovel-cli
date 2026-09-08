package host

import (
	"context"
	"strings"
	"testing"

	"github.com/JustinNguyen9979/ainovel-cli/internal/bootstrap"
	"github.com/JustinNguyen9979/ainovel-cli/internal/domain"
	storepkg "github.com/JustinNguyen9979/ainovel-cli/internal/store"
	"github.com/voocel/agentcore"
)

func TestHostControlAndModelAccessors(t *testing.T) {
	st := storepkg.NewStore(t.TempDir())
	if err := st.Init(); err != nil {
		t.Fatal(err)
	}
	if err := st.Progress.Init(4); err != nil {
		t.Fatal(err)
	}
	if err := st.Progress.UpdatePhase(domain.PhaseWriting); err != nil {
		t.Fatal(err)
	}
	if err := st.RunMeta.Init("default", "openrouter", "model"); err != nil {
		t.Fatal(err)
	}
	cfg := bootstrap.Config{Provider: "openrouter", ModelName: "model", ReasoningEffort: "low", Providers: map[string]bootstrap.ProviderConfig{"openrouter": {APIKey: "key", Models: []bootstrap.ModelConfig{{Name: "model"}}}}, Roles: map[string]bootstrap.RoleConfig{"writer": {Provider: "openrouter", Model: "model"}}}
	models, err := bootstrap.NewModelSet(cfg)
	if err != nil {
		t.Fatal(err)
	}
	applied := map[string]agentcore.ThinkingLevel{}
	h := &Host{cfg: cfg, store: st, models: models, events: make(chan Event, 8), thinkingApplier: func(role string, level agentcore.ThinkingLevel) { applied[role] = level }, engine: &engine{}}
	if got := h.ConfiguredProviders(); len(got) != 1 || got[0] != "openrouter" {
		t.Fatalf("providers = %#v", got)
	}
	if got := h.ConfiguredModels("openrouter"); len(got) != 1 || got[0] != "model" {
		t.Fatalf("models = %#v", got)
	}
	if h.CurrentThinking(" WRITER ") != "low" || len(h.AvailableThinking("writer")) == 0 {
		t.Fatal("thinking accessors mismatch")
	}
	if err := h.SetRoleThinking("writer", "high"); err != nil {
		t.Fatal(err)
	}
	if h.cfg.Roles["writer"].ReasoningEffort != "high" || applied["writer"] == "" {
		t.Fatalf("thinking update = %#v applied=%#v", h.cfg.Roles, applied)
	}
	if err := h.SetRoleThinking("writer", "invalid"); err == nil {
		t.Fatal("invalid thinking should fail")
	}
	if err := h.SetAdvanceMode(domain.ChapterAdvanceReview); err != nil {
		t.Fatal(err)
	}
	if err := h.SetAdvanceMode(domain.ChapterAdvanceAuto); err != nil {
		t.Fatal(err)
	}
	if err := h.SetAdvanceMode(domain.ChapterAdvanceMode("bad")); err == nil {
		t.Fatal("invalid advance mode should fail")
	}
	if err := h.SwitchModel("", "", ""); err == nil || !strings.Contains(err.Error(), "required") {
		t.Fatal("empty model switch should fail")
	}
	if got, err := h.ReplayQueue(0); err != nil || got != nil {
		t.Fatalf("replay empty queue = %#v/%v", got, err)
	}
	if h.FileLogError() != nil {
		t.Fatal("unexpected file log error")
	}
}

func TestHostSuperviseExclusiveAndOutputChannels(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	h := &Host{runCtx: ctx, events: make(chan Event, 2), streamCh: make(chan string, 2), done: make(chan struct{}, 1), observer: &observer{}, engine: &engine{}}
	src := make(chan string, 2)
	src <- "one"
	src <- "two"
	close(src)
	h.exclusive = "job"
	out := superviseExclusive(h, src)
	var got []string
	for value := range out {
		got = append(got, value)
	}
	if len(got) != 2 || h.exclusive != "" {
		t.Fatalf("supervised output = %#v exclusive=%q", got, h.exclusive)
	}
	cancel()
	h.emitEvent(Event{Category: "SYSTEM", Summary: "event"})
	h.emitDelta("delta")
	h.emitClear()
	if len(h.events) != 1 || len(h.streamCh) != 2 {
		t.Fatalf("channels = events=%d stream=%d", len(h.events), len(h.streamCh))
	}
	h.closeOutputChannels()
	h.closeOutputChannels()
}

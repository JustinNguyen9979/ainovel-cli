package host

import (
	"testing"

	"github.com/JustinNguyen9979/ainovel-cli/internal/bootstrap"
	"github.com/JustinNguyen9979/ainovel-cli/internal/domain"
	storepkg "github.com/JustinNguyen9979/ainovel-cli/internal/store"
	"github.com/voocel/agentcore"
)

func TestHostSnapshotProjectsStoreState(t *testing.T) {
	st := storepkg.NewStore(t.TempDir())
	if err := st.Init(); err != nil {
		t.Fatal(err)
	}
	if err := st.Book.Save(domain.BookMetadata{Title: "书名", Synopsis: "简介"}); err != nil {
		t.Fatal(err)
	}
	if err := st.Outline.SavePremise("# 设定\n\n正文前提"); err != nil {
		t.Fatal(err)
	}
	if err := st.Outline.SaveOutline([]domain.OutlineEntry{{Chapter: 1, Title: "计划一", CoreEvent: "开端"}, {Chapter: 2, Title: "计划二", CoreEvent: "发展"}}); err != nil {
		t.Fatal(err)
	}
	if err := st.Characters.Save([]domain.Character{{Name: "主角", Role: "主角"}}); err != nil {
		t.Fatal(err)
	}
	if err := st.Cast.Save([]domain.CastEntry{{Name: "配角", BriefRole: "线索人", LastSeenChapter: 2, AppearanceCount: 2}}); err != nil {
		t.Fatal(err)
	}
	if err := st.Summaries.SaveSummary(domain.ChapterSummary{Chapter: 1, Title: "终稿一", Summary: "第一章摘要"}); err != nil {
		t.Fatal(err)
	}
	if err := st.Summaries.SaveSummary(domain.ChapterSummary{Chapter: 2, Title: "终稿二", Summary: "第二章摘要"}); err != nil {
		t.Fatal(err)
	}
	if err := st.Progress.Save(&domain.Progress{Phase: domain.PhaseWriting, Flow: domain.FlowWriting, CurrentChapter: 2, TotalChapters: 2, CompletedChapters: []int{1}, TotalWordCount: 100, ChapterWordCounts: map[int]int{1: 100}}); err != nil {
		t.Fatal(err)
	}
	if err := st.RunMeta.Init("default", "openrouter", "model"); err != nil {
		t.Fatal(err)
	}
	if err := st.RunMeta.SetPendingSteer("继续"); err != nil {
		t.Fatal(err)
	}
	if _, err := st.Checkpoints.AppendArtifact(domain.GlobalScope(), "book", "meta/book.json"); err != nil {
		t.Fatal(err)
	}

	cfg := bootstrap.Config{Provider: "openrouter", ModelName: "model", Style: "default", Providers: map[string]bootstrap.ProviderConfig{"openrouter": {APIKey: "key", Models: []bootstrap.ModelConfig{{Name: "model", ContextWindow: 128000}}}}}
	models, err := bootstrap.NewModelSet(cfg)
	if err != nil {
		t.Fatal(err)
	}
	usage := NewUsageTracker(models, st)
	usage.accumulate("writer", "", "", agentcore.Usage{Input: 100, Output: 20, Cost: &agentcore.Cost{Total: 0.2}})
	h := &Host{cfg: cfg, store: st, models: models, usage: usage, observer: &observer{agents: map[string]*agentState{}}, lifecycle: lifecycleRunning, budget: NewBudgetSentinel(bootstrap.BudgetConfig{BookUSD: 1, WarnRatio: .8}, func() float64 { return .2 }, nil, nil)}
	snap := h.Snapshot()
	if snap.BookTitle != "书名" || snap.Phase != string(domain.PhaseWriting) || snap.CompletedCount != 1 || snap.PendingSteer != "继续" || snap.TotalCostUSD <= 0 {
		t.Fatalf("snapshot = %+v", snap)
	}
	if len(snap.Outline) != 2 || snap.Outline[0].Title != "终稿一" || len(snap.Characters) != 1 || snap.SupportingCount != 1 || snap.LastCheckpointName == "" {
		t.Fatalf("snapshot details = %+v", snap)
	}
	if snap.StatusLabel != "RUNNING" || !snap.IsRunning || len(snap.CachePerAgent) == 0 {
		t.Fatalf("snapshot projection = %+v", snap)
	}
	if got := h.runEndBody("书名", "完成"); got == "" {
		t.Fatal("run end body empty")
	}
}

func TestHostExclusiveAndAsyncHelpers(t *testing.T) {
	h := &Host{events: make(chan Event, 2), streamCh: make(chan string, 2), done: make(chan struct{}, 1), observer: &observer{}, engine: &engine{}}
	if err := h.acquireExclusive("导入"); err != nil || h.exclusive != "导入" {
		t.Fatalf("acquire exclusive = %v/%q", err, h.exclusive)
	}
	if err := h.acquireExclusive("仿写"); err == nil {
		t.Fatal("second exclusive job should fail")
	}
	h.releaseExclusive()
	if h.exclusive != "" {
		t.Fatal("release should clear exclusive state")
	}
	if err, launched := h.runAsync(func() error { return nil }); err != nil || !launched {
		t.Fatalf("runAsync = %v/%v", err, launched)
	}
	h.closing = true
	if err, launched := h.runAsync(func() error { return nil }); err != nil || launched {
		t.Fatalf("closing runAsync = %v/%v", err, launched)
	}
}

package ctxpack

import (
	"context"
	"errors"
	"strings"
	"testing"

	"github.com/JustinNguyen9979/ainovel-cli/internal/domain"
	storepkg "github.com/JustinNguyen9979/ainovel-cli/internal/store"
	"github.com/voocel/agentcore"
	corecontext "github.com/voocel/agentcore/context"
)

func TestWriterRestorePackLifecycleAndHelpers(t *testing.T) {
	var nilState *writerStoreSummaryState
	nilState.warn("ignored", errors.New("bad"))
	nilState.warn("ignored", nil)
	state := &writerStoreSummaryState{}
	state.warn("scope", errors.New("bad"))
	if len(state.warnings) != 1 || !strings.Contains(state.warnings[0], "scope") {
		t.Fatalf("warnings = %#v", state.warnings)
	}
	if writerStoreProgressSection(nil) != nil || writerStoreProgressSection(&writerStoreSummaryState{}) != nil {
		t.Fatal("empty progress section should be nil")
	}
	state.progress = &domain.Progress{Phase: domain.PhaseWriting, Flow: domain.FlowWriting, CompletedChapters: []int{1, 2}}
	state.chapter = 3
	if got := writerStoreProgressSection(state); got["current_chapter"] != 3 {
		t.Fatalf("progress section = %#v", got)
	}
	if isEmptySummarySection(nil) == false || !isEmptySummarySection("") || !isEmptySummarySection([]string{}) || isEmptySummarySection(1) {
		t.Fatal("empty section reflection mismatch")
	}
	if got := truncateJSONToTokens([]byte("abcdefghijklmnopqrstuvwxyz"), 2); len(got) != 20 || truncateJSONToTokens([]byte("abc"), 2) != "abc" {
		t.Fatalf("truncate JSON = %q", got)
	}

	pack := &WriterRestorePack{}
	if _, ok, err := pack.buildMessage(100); err != nil || ok {
		t.Fatalf("empty pack = %v/%v", ok, err)
	}
	pack.setWarning("progress", errors.New("broken"))
	msg, ok, err := pack.buildMessage(1000)
	if err != nil || !ok || !strings.Contains(msg.TextContent(), "broken") {
		t.Fatalf("warning pack = %v/%v/%v", msg.TextContent(), ok, err)
	}
	pack.Clear()
	if _, ok, _ := pack.buildMessage(1000); ok {
		t.Fatal("clear should empty pack")
	}
	hook := pack.Hook()
	msgs, err := hook(context.Background(), corecontext.SummaryInfo{}, nil, 100)
	if err != nil || len(msgs) != 0 {
		t.Fatalf("empty hook = %#v/%v", msgs, err)
	}
}

func TestStoreSummaryCutPointVariants(t *testing.T) {
	user := agentcore.UserMsg("user request")
	assistant := agentcore.Message{Role: agentcore.RoleAssistant, Content: []agentcore.ContentBlock{agentcore.TextBlock(strings.Repeat("assistant", 20))}}
	toolCall := agentcore.Message{Role: agentcore.RoleAssistant, Content: []agentcore.ContentBlock{agentcore.ToolCallBlock(agentcore.ToolCall{Name: "tool"})}}
	tool := agentcore.Message{Role: agentcore.RoleTool, Content: []agentcore.ContentBlock{agentcore.TextBlock("result")}}
	for _, tc := range []struct {
		name string
		msgs []agentcore.AgentMessage
		keep int
	}{
		{"empty", nil, 10},
		{"keep all", []agentcore.AgentMessage{user}, 10000},
		{"normal", []agentcore.AgentMessage{user, assistant, user, assistant}, 1},
		{"tool", []agentcore.AgentMessage{user, toolCall, tool, user, assistant}, 1},
	} {
		t.Run(tc.name, func(t *testing.T) {
			got := findStoreSummaryCutPoint(tc.msgs, tc.keep)
			if tc.name == "empty" && got != (storeSummaryCutResult{}) {
				t.Fatalf("empty = %+v", got)
			}
		})
	}
}

func TestStoreSummaryStrategyNoOpAndForce(t *testing.T) {
	strategy := NewStoreSummaryCompact(StoreSummaryCompactConfig{})
	msgs := []agentcore.AgentMessage{agentcore.UserMsg("one")}
	if out, result, err := strategy.Apply(context.Background(), nil, msgs, corecontext.Budget{}); err != nil || len(out) != 1 || result.Applied {
		t.Fatalf("budget no-op = %#v/%+v/%v", out, result, err)
	}
	if out, result, err := strategy.ForceApply(context.Background(), nil, msgs, corecontext.Budget{}); err != nil || len(out) != 1 || result.Applied {
		t.Fatalf("force nil store = %#v/%+v/%v", out, result, err)
	}
}

func TestWriterRestoreFallbackLoadsChapterOneState(t *testing.T) {
	s := storepkg.NewStore(t.TempDir())
	if err := s.Init(); err != nil {
		t.Fatal(err)
	}
	if err := s.Progress.Save(&domain.Progress{Phase: domain.PhaseWriting, CurrentChapter: 1, TotalChapters: 3}); err != nil {
		t.Fatal(err)
	}
	if err := s.Outline.SaveOutline([]domain.OutlineEntry{{Chapter: 1, Title: "开端", CoreEvent: "故事开始"}}); err != nil {
		t.Fatal(err)
	}
	if err := s.Drafts.SaveChapterPlan(domain.ChapterPlan{Chapter: 1, Title: "开端", Goal: "建立冲突"}); err != nil {
		t.Fatal(err)
	}
	if err := s.World.SaveForeshadowLedger([]domain.ForeshadowEntry{{ID: "seed", Description: "未回收线索", PlantedAt: 1, Status: "planted"}}); err != nil {
		t.Fatal(err)
	}
	text, ok, err := buildWriterRestoreText(s, 1000)
	if err != nil || !ok || !strings.Contains(text, "当前章节计划") || !strings.Contains(text, "活跃伏笔") {
		t.Fatalf("chapter-one fallback = ok=%v err=%v text=%q", ok, err, text)
	}

	pack := &WriterRestorePack{}
	pack.Refresh(s)
	if _, ok, err := pack.buildMessage(1000); err != nil || !ok {
		t.Fatalf("refresh fallback = ok=%v err=%v", ok, err)
	}
}

func TestWriterRestoreLayeredSummaryFallbacks(t *testing.T) {
	s := storepkg.NewStore(t.TempDir())
	if err := s.Init(); err != nil {
		t.Fatal(err)
	}
	if err := s.Progress.Save(&domain.Progress{Phase: domain.PhaseWriting, CurrentChapter: 3, TotalChapters: 4, Layered: true}); err != nil {
		t.Fatal(err)
	}
	if err := s.Outline.SaveLayeredOutline([]domain.VolumeOutline{{Index: 1, Arcs: []domain.ArcOutline{{Index: 1, Chapters: []domain.OutlineEntry{{Chapter: 1}, {Chapter: 2}, {Chapter: 3}}}}}}); err != nil {
		t.Fatal(err)
	}
	if err := s.Summaries.SaveVolumeSummary(domain.VolumeSummary{Volume: 1, Title: "第一卷", Summary: "卷摘要"}); err != nil {
		t.Fatal(err)
	}
	if err := s.Summaries.SaveArcSummary(domain.ArcSummary{Volume: 1, Arc: 1, Title: "第一弧", Summary: "弧摘要"}); err != nil {
		t.Fatal(err)
	}
	if err := s.Outline.SaveOutline([]domain.OutlineEntry{{Chapter: 3, Title: "第三章"}}); err != nil {
		t.Fatal(err)
	}
	state, ok, err := loadWriterStoreSummaryState(s)
	if err != nil || !ok || state.currentVolSummary == nil || state.currentArcSummary == nil {
		t.Fatalf("layered state = %+v ok=%v err=%v", state, ok, err)
	}
	text, ok, err := buildWriterStoreSummaryText(s, 2000)
	if err != nil || !ok || !strings.Contains(text, "当前卷摘要") || !strings.Contains(text, "当前弧摘要") {
		t.Fatalf("layered summary = ok=%v err=%v text=%q", ok, err, text)
	}
}

func TestAppendJSONSectionBudgetBranches(t *testing.T) {
	var parts []string
	remaining := 1000
	if appendJSONSection(&parts, "small", map[string]string{"x": "y"}, &remaining) {
		t.Fatal("small section should fit")
	}
	remaining = 100
	if !appendJSONSection(&parts, "large", strings.Repeat("x", 1000), &remaining) {
		t.Fatal("large section at low budget should stop")
	}
	remaining = 1000
	if !appendJSONSection(nil, "nil", "x", &remaining) {
		t.Fatal("nil output must stop")
	}
	if !appendJSONSection(&parts, "none", "x", nil) {
		t.Fatal("nil budget must stop")
	}
}

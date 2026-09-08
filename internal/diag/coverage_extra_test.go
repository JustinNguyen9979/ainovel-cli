package diag

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/JustinNguyen9979/ainovel-cli/internal/domain"
	"github.com/JustinNguyen9979/ainovel-cli/internal/store"
	"github.com/voocel/agentcore"
)

func TestCaptureRuntimeReadsLogAndCheckpointSignals(t *testing.T) {
	dir := t.TempDir()
	st := store.NewStore(dir)
	if err := st.Init(); err != nil {
		t.Fatal(err)
	}
	for i := 0; i < 3; i++ {
		if _, err := st.Checkpoints.Append(domain.GlobalScope(), "plan", "", ""); err != nil {
			t.Fatal(err)
		}
	}
	if err := os.MkdirAll(filepath.Join(dir, "logs"), 0o755); err != nil {
		t.Fatal(err)
	}
	log := "level=ERROR kind=stream_idle stop_guard\nlevel=WARN kind=stream_idle\n"
	if err := os.WriteFile(filepath.Join(dir, "logs", "headless.log"), []byte(log), 0o600); err != nil {
		t.Fatal(err)
	}
	rc := CaptureRuntime(st)
	if rc.LogErrors != 1 || rc.LogWarns != 1 || rc.LogKinds["stream_idle"] != 2 || rc.StopGuard != 1 {
		t.Fatalf("runtime log signals = %+v", rc)
	}
	if rc.CurrentStep == "" || rc.StuckStep == "" || rc.StuckCount != 3 {
		t.Fatalf("checkpoint signals = %+v", rc)
	}
	if _, ok := readTail(filepath.Join(dir, "missing.log")); ok {
		t.Fatal("missing log should not be readable")
	}
	if got := tailEvents([]SkelEvent{{}, {}}, 1); len(got) != 1 {
		t.Fatalf("tail events = %d", len(got))
	}
	if got := topDups(map[string]int{"a": 4, "b": 1}); len(got) != 1 || got[0].Sha != "a" {
		t.Fatalf("top dups = %+v", got)
	}
	if got := sortedModels(map[string]RoleModel{"writer": {Agent: "writer"}, "architect": {Agent: "architect"}}); len(got) != 2 || got[0].Agent != "architect" {
		t.Fatalf("sorted models = %+v", got)
	}
}

func TestCaptureRuntimeCoversSessionAndLogBranches(t *testing.T) {
	dir := t.TempDir()
	st := store.NewStore(dir)
	if err := st.Init(); err != nil {
		t.Fatal(err)
	}
	toolArgs := json.RawMessage(`{"chapter":7,"mode":"write"}`)
	messages := []agentcore.Message{
		{Role: agentcore.RoleAssistant, Content: []agentcore.ContentBlock{
			agentcore.TextBlock("repeated prose"),
			agentcore.ThinkingBlock("private reasoning"),
			agentcore.ToolCallBlock(agentcore.ToolCall{Name: "commit_chapter", Args: toolArgs}),
			agentcore.ToolCallBlock(agentcore.ToolCall{Name: "commit_chapter", Args: toolArgs}),
			agentcore.ToolCallBlock(agentcore.ToolCall{Name: "commit_chapter", Args: toolArgs}),
		}},
		{Role: agentcore.RoleTool, Metadata: map[string]any{"is_error": true}, Content: []agentcore.ContentBlock{agentcore.TextBlock("InputValidationError: chapter must be int")}},
		{Role: agentcore.RoleAssistant, Content: []agentcore.ContentBlock{agentcore.TextBlock("repeated prose")}},
		{Role: agentcore.RoleAssistant, Content: []agentcore.ContentBlock{agentcore.TextBlock("repeated prose")}},
		{Role: agentcore.RoleAssistant, Content: []agentcore.ContentBlock{agentcore.TextBlock("repeated prose")}},
	}
	writeSessionAt(t, dir, filepath.Join("agents", "writer-ch07.jsonl"), messages)
	writeSessionAt(t, dir, filepath.Join("agents", "architect.jsonl"), []agentcore.Message{{Role: agentcore.RoleAssistant, Content: []agentcore.ContentBlock{agentcore.TextBlock("architect prose")}}})
	writeSessionAt(t, dir, filepath.Join("agents", "ignored.jsonl"), []agentcore.Message{{Role: agentcore.RoleAssistant, Content: []agentcore.ContentBlock{agentcore.TextBlock("old prose")}}})
	badPath := filepath.Join(dir, "meta", "sessions", "agents", "writer-ch07.jsonl")
	data, err := os.ReadFile(badPath)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(badPath, append(data, []byte("{malformed\n")...), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(filepath.Join(dir, "logs"), 0o755); err != nil {
		t.Fatal(err)
	}
	largeLog := strings.Repeat("prefix\n", 40000) + "level=ERROR kind=stream_idle stop_guard\nlevel=WARN kind=stream_idle\n"
	if err := os.WriteFile(filepath.Join(dir, "logs", "tui.log"), []byte(largeLog), 0o600); err != nil {
		t.Fatal(err)
	}

	rc := CaptureRuntime(st)
	if rc.LogErrors != 1 || rc.LogWarns != 1 || rc.LogKinds["stream_idle"] != 2 || rc.StopGuard != 1 {
		t.Fatalf("runtime log signals = %+v", rc)
	}
	if len(rc.Tail) == 0 || rc.RedactedTexts < 4 || len(rc.Sources) < 2 {
		t.Fatalf("session capture = %+v", rc)
	}
	if len(rc.Repeats) == 0 || rc.Repeats[0].Count < 3 || len(rc.DupContent) == 0 {
		t.Fatalf("repeat capture = %+v dups=%+v", rc.Repeats, rc.DupContent)
	}
}

func TestCaptureRuntimeFallsBackToHeadlessLog(t *testing.T) {
	dir := t.TempDir()
	st := store.NewStore(dir)
	if err := st.Init(); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(filepath.Join(dir, "logs"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "logs", "headless.log"), []byte("level=WARN kind=network\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	rc := CaptureRuntime(st)
	if rc.LogWarns != 1 || rc.LogKinds["network"] != 1 || len(rc.Sources) != 1 || !strings.Contains(rc.Sources[0], "headless.log") {
		t.Fatalf("headless fallback = %+v", rc)
	}
}

func TestRedactMessageCoversStructuredValueBranches(t *testing.T) {
	msg := agentcore.Message{Role: agentcore.RoleAssistant, Content: []agentcore.ContentBlock{
		agentcore.TextBlock("text"),
		agentcore.ThinkingBlock("thinking"),
		agentcore.ToolCallBlock(agentcore.ToolCall{Name: "object", Args: json.RawMessage(`{"nested":{"value":true}}`)}),
		agentcore.ToolCallBlock(agentcore.ToolCall{Name: "array", Args: json.RawMessage(`{"items":[1,2]}`)}),
		agentcore.ToolCallBlock(agentcore.ToolCall{Name: "empty", Args: nil}),
		agentcore.ToolCallBlock(agentcore.ToolCall{Name: "invalid", Args: json.RawMessage(`"unterminated`), ArgsInvalid: true, ArgsParseError: "bad\nline"}),
	}}
	got := redactMessage("writer", msg)
	if got.Redacted != 2 || len(got.Tools) != 4 || !strings.Contains(got.Tools[0].Args["nested"], "redacted object") || !strings.Contains(got.Tools[1].Args["items"], "redacted array len") || !got.Tools[3].Invalid {
		t.Fatalf("redacted message = %+v", got)
	}
	errorMsg := agentcore.Message{Role: agentcore.RoleTool, Metadata: map[string]any{"is_error": true}, Content: []agentcore.ContentBlock{agentcore.TextBlock("first line\nprivate detail")}}
	if got := redactMessage("tool", errorMsg); got.ErrClass != "first line" || got.Redacted != 0 {
		t.Fatalf("error redaction = %+v", got)
	}
	for _, raw := range []json.RawMessage{json.RawMessage(`{`), json.RawMessage(`[]`), nil} {
		_ = redactArgs(raw)
	}
}

func TestDiagSnapshotStatsAndThresholds(t *testing.T) {
	var empty Snapshot
	if empty.CompletedCount() != 0 || empty.LatestCompleted() != 0 {
		t.Fatal("empty snapshot counters mismatch")
	}
	if staleForeshadowThreshold(10) != ThresholdForeshadowMin || staleForeshadowThreshold(30) != 10 {
		t.Fatal("stale threshold mismatch")
	}
	snap := Snapshot{Progress: &domain.Progress{CompletedChapters: []int{3, 1, 2}, TotalChapters: 4, TotalWordCount: 400, Phase: domain.PhaseWriting, Flow: domain.FlowWriting}, RunMeta: &domain.RunMeta{PlanningTier: domain.PlanningTierLong}, Foreshadow: []domain.ForeshadowEntry{{ID: "open", PlantedAt: 1, Status: "planted"}, {ID: "advanced", PlantedAt: 1, Status: "advanced"}, {ID: "resolved", PlantedAt: 1, Status: "resolved"}}, Reviews: map[int]*domain.ReviewEntry{1: {Verdict: "rewrite", Dimensions: []domain.DimensionScore{{Score: 80}, {Score: 60}}}}}
	if got := snap.CompletedCount(); got != 3 || snap.LatestCompleted() != 3 {
		t.Fatalf("snapshot counters = %d/%d", got, snap.LatestCompleted())
	}
	stats := buildStats(&snap)
	if stats.CompletedChapters != 3 || stats.AvgWordsPerCh != 133 || stats.ReviewCount != 1 || stats.RewriteCount != 1 || stats.AvgReviewScore != 70 || stats.ForeshadowOpen != 2 {
		t.Fatalf("stats = %+v", stats)
	}
	findings := []Finding{{Severity: SevInfo}, {Severity: SevCritical}, {Severity: SevWarning}}
	sortFindings(findings)
	if findings[0].Severity != SevCritical || findings[2].Severity != SevInfo {
		t.Fatalf("sorted findings = %+v", findings)
	}
	s := store.NewStore(t.TempDir())
	if err := s.Init(); err != nil {
		t.Fatal(err)
	}
	loaded := Load(s)
	if loaded.Reviews == nil || loaded.Plans == nil || loaded.Summaries == nil {
		t.Fatal("loaded snapshot maps should be initialized")
	}
	if !strings.Contains(Analyze(s).Stats.Phase, "") {
		t.Fatal("analyze stats unexpectedly invalid")
	}
}

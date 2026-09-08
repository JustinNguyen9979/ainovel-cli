package tui

import (
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/JustinNguyen9979/ainovel-cli/internal/diag"
	"github.com/JustinNguyen9979/ainovel-cli/internal/host"
	"github.com/JustinNguyen9979/ainovel-cli/internal/host/exp"
	"github.com/JustinNguyen9979/ainovel-cli/internal/host/sim"
	"github.com/JustinNguyen9979/ainovel-cli/internal/utils"
	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/x/ansi"
)

func TestTUIRenderWelcomeHelpAndReportStates(t *testing.T) {
	for _, lang := range []utils.Language{utils.LanguageVI, utils.LanguageZH} {
		welcome := renderWelcome(120, 36, "lỗi", startupModeCoCreate, "", "", lang)
		plain := ansi.Strip(welcome)
		if !strings.Contains(plain, "A I N O V E L") || !strings.Contains(plain, "lỗi") {
			t.Fatalf("welcome(%s) = %q", lang, plain)
		}
		if got := renderWelcome(120, 36, "", startupModeQuick, "import hint", "update hint", lang); !strings.Contains(ansi.Strip(got), "import hint") {
			t.Fatalf("import hint should take precedence: %q", ansi.Strip(got))
		}
		if got := renderWelcome(120, 36, "", startupModeQuick, "", "update hint", lang); !strings.Contains(ansi.Strip(got), "update hint") {
			t.Fatalf("update hint missing: %q", ansi.Strip(got))
		}

		help := newHelpState(120, 36, lang)
		if help == nil || !strings.Contains(ansi.Strip(help.viewport.View()), "/help") {
			t.Fatalf("help state = %+v", help)
		}
		if got := renderHelpModal(120, 36, help); got == "" || !strings.Contains(ansi.Strip(got), "help") {
			t.Fatalf("help modal = %q", ansi.Strip(got))
		}
		if renderHelpModal(120, 36, nil) != "" {
			t.Fatal("nil help modal should be empty")
		}

		report := diag.Report{
			Stats: diag.Stats{CompletedChapters: 2, TotalChapters: 8, TotalWords: 2345, AvgWordsPerCh: 1172, Phase: "writing", Flow: "rewriting", PlanningTier: "long", ReviewCount: 2, RewriteCount: 1, AvgReviewScore: 82.5, ForeshadowOpen: 2, ForeshadowStale: 1},
			Findings: []diag.Finding{
				{Severity: diag.SevCritical, Confidence: diag.ConfHigh, AutoLevel: diag.AutoSafe, Title: "严重问题", Evidence: "证据", Suggestion: "建议"},
				{Severity: diag.SevWarning, Confidence: diag.ConfMedium, AutoLevel: diag.AutoSuggest, Title: "警告"},
				{Severity: diag.SevInfo, Title: "提示"},
			},
			Actions: []diag.Action{{Kind: diag.ActionEmitNotice, Summary: "通知", Message: "请继续"}},
		}
		text := renderReportText(report, 60, "/tmp/report.json", nil, time.Now().Add(-time.Minute), time.Now(), lang)
		plain = ansi.Strip(text)
		for _, want := range []string{"严重问题", "警告", "提示", "通知"} {
			if !strings.Contains(plain, want) {
				t.Fatalf("report missing %q: %q", want, plain)
			}
		}
		if got := renderReportText(diag.Report{}, 60, "", errors.New("export"), time.Time{}, time.Time{}, lang); !strings.Contains(ansi.Strip(got), "export") {
			t.Fatalf("report export error missing: %q", ansi.Strip(got))
		}
		if got := renderReportLoadingText(60, time.Time{}, lang); !strings.Contains(ansi.Strip(got), "-") {
			t.Fatalf("loading report = %q", ansi.Strip(got))
		}
		state := newReportState(120, 36, 4, time.Now(), lang)
		state.load(report, 80, "", nil, time.Now())
		if state.loading || state.report == nil || state.viewport.View() == "" {
			t.Fatal("report state load failed")
		}
		if renderReportModal(120, 36, state) == "" || renderReportModal(120, 36, nil) != "" {
			t.Fatal("report modal rendering failed")
		}
	}
}

func TestTUIRenderSimulationAndPanels(t *testing.T) {
	state := newSimulationState(1, "画像", "source.txt", 120, 36, func() {}, utils.LanguageVI)
	at := time.Now()
	state.appendEvent(sim.Event{Time: at, Stage: sim.StageAnalyze, Current: 2, Total: 4, Message: "正在分析"}, 80)
	state.appendEvent(sim.Event{Time: at, Stage: sim.StageError, Message: "失败", Err: errors.New("bad")}, 80)
	if !state.done || state.err == nil || !strings.Contains(ansi.Strip(state.viewport.View()), "失败") {
		t.Fatalf("simulation state = %+v", state)
	}
	if renderSimulationModal(120, 36, state) == "" || renderSimulationModal(120, 36, nil) != "" {
		t.Fatal("simulation modal failed")
	}
	for _, stage := range []sim.Stage{sim.StageScan, sim.StageAnalyze, sim.StageMerge, sim.StageImport, sim.StageDone, sim.StageError, sim.Stage("other")} {
		if localizedSimulationStage(utils.LanguageVI, stage) == "" || localizedSimulationStage(utils.LanguageZH, stage) == "" {
			t.Fatalf("stage %q missing label", stage)
		}
	}
	if !(simEventMsg{ev: sim.Event{Stage: sim.StageDone}}).terminal() || (simEventMsg{ev: sim.Event{Stage: sim.StageScan}}).terminal() {
		t.Fatal("simulation terminal detection mismatch")
	}

	snap := host.UISnapshot{
		RuntimeState: "paused", Phase: "writing", Flow: "rewriting", AdvanceMode: "review", AdvancePermitChapter: 3,
		CompletedCount: 2, TotalChapters: 10, TotalWordCount: 12345, InProgressChapter: 3,
		PendingRewrites: []int{3}, RewriteReason: "cần sửa", PendingSteer: "đổi hướng", HasAdvanceHold: true, AdvanceHoldReason: "đợi duyệt",
		RecoveryLabel: "khôi phục", Synopsis: "mô tả", Premise: "tiền đề", Outline: []host.OutlineSnapshot{{Chapter: 1, Title: "một", CoreEvent: "mở đầu"}, {Chapter: 3, Title: "ba", CoreEvent: "cao trào"}},
		Characters: []string{"A", "B"}, SupportingCount: 2, RecentSupporting: []string{"C"}, LastCommitSummary: "chương 2", LastReviewSummary: "verdict=polish", RecentSummaries: []string{"tóm tắt"},
		TotalInputTokens: 12000, TotalOutputTokens: 3000, TotalCostUSD: 1.23, TotalSavedUSD: 0.45, BudgetLimitUSD: 4,
		OverallCacheCapable: true, TotalCacheReadTokens: 5000, TotalCacheWriteTokens: 100, OverallRecentCacheRead: 2000, OverallRecentInput: 3000, OverallRecentSamples: 2,
		CachePerAgent: []host.AgentCacheStat{{Role: "writer", Input: 8000, CacheRead: 4000, CacheCapable: true, RecentInput: 3000, RecentCacheRead: 2000, RecentSamples: 2}, {Role: "arbiter", Input: 100, CacheCapable: false}},
		CachePerModel: []host.AgentCacheStat{{Model: "openai/gpt", Input: 1000, Output: 100}},
		Agents:        []host.AgentSnapshot{{Name: "writer", State: "running", TaskKind: "chapter_write", Summary: "写作", Tool: "draft", Context: host.AgentContextSnapshot{Tokens: 80, ContextWindow: 100, Percent: 80, Scope: "projected", Strategy: "light_trim"}}, {Name: "editor", State: "idle", Summary: "待命"}},
	}
	for _, lang := range []utils.Language{utils.LanguageVI, utils.LanguageZH} {
		stateText := renderStateContent(snap, 42, lang)
		detailText := renderDetailContent(snap, 64, lang)
		if stateText == "" || detailText == "" {
			t.Fatalf("panels empty for %s", lang)
		}
		if !strings.Contains(ansi.Strip(stateText), "WRITER") || !strings.Contains(ansi.Strip(detailText), "mô tả") && lang == utils.LanguageVI {
			t.Fatalf("panels missing content: state=%q detail=%q", ansi.Strip(stateText), ansi.Strip(detailText))
		}
	}
	if renderOutlineSection(host.UISnapshot{Outline: []host.OutlineSnapshot{{Chapter: 1, Title: "one"}}}, 50, utils.LanguageVI) == "" {
		t.Fatal("outline list empty")
	}
	many := make([]host.OutlineSnapshot, 21)
	for i := range many {
		many[i] = host.OutlineSnapshot{Chapter: i + 1, Title: "chapter"}
	}
	if renderOutlineSection(host.UISnapshot{Outline: many, CompletedCount: 1, InProgressChapter: 2}, 80, utils.LanguageVI) == "" {
		t.Fatal("outline grid empty")
	}
}

func TestTUIModelViewsAndInputCommands(t *testing.T) {
	m := NewModel(nil, "v1", utils.LanguageVI)
	if m.View() != utils.T(utils.LanguageVI, utils.MsgLoading) {
		t.Fatal("zero-size model should show loading")
	}
	m.width, m.height = 80, 24
	if !strings.Contains(ansi.Strip(m.View()), "Terminal") && m.View() == "" {
		t.Fatal("narrow model view empty")
	}
	m.width, m.height = 120, 36
	if m.View() == "" {
		t.Fatal("welcome model view empty")
	}
	m.mode = modeRunning
	m.snapshot = host.UISnapshot{RuntimeState: "running", IsRunning: true, Phase: "writing", Outline: []host.OutlineSnapshot{{Chapter: 1, Title: "one"}}}
	m.resizeTextarea()
	if m.View() == "" {
		t.Fatal("running model view empty")
	}
	m.help = newHelpState(120, 36, utils.LanguageVI)
	if m.View() == "" {
		t.Fatal("help overlay empty")
	}
	m.help = nil
	m.report = newReportState(120, 36, 1, time.Now(), utils.LanguageVI)
	if m.View() == "" {
		t.Fatal("report overlay empty")
	}
	m.report = nil
	m.simulator = newSimulationState(1, "sim", "", 120, 36, nil, utils.LanguageVI)
	if m.View() == "" {
		t.Fatal("simulation overlay empty")
	}
	m.simulator = nil
	m.cocreate = newCoCreateState("idea", utils.LanguageVI)
	if m.View() == "" {
		t.Fatal("cocreate overlay empty")
	}

	if cmd, ok := parseSlashCommand("  /MODEL one two "); !ok || cmd.name != "model" || len(cmd.args) != 2 {
		t.Fatalf("slash command = %+v/%v", cmd, ok)
	}
	if _, ok := parseSlashCommand("plain"); ok {
		t.Fatal("plain input is not slash command")
	}
	if _, err := parseExportArgs([]string{"from=2", "to=4", "out.txt", "--overwrite"}); err != nil {
		t.Fatalf("export args: %v", err)
	}
	for _, args := range [][]string{{"from=x"}, {"to=-1"}, {"unknown=1"}, {"-x"}, {"a", "b"}} {
		if _, err := parseExportArgs(args); err == nil {
			t.Fatalf("args %v should fail", args)
		}
	}
	if got := formatExportSuccess(&exp.Result{Chapters: 3, Bytes: 2048, Skipped: []int{4}, Path: "out.txt"}, utils.LanguageVI); got == "" {
		t.Fatal("export success empty")
	}
	if got := briefIntList([]int{1, 2, 3}, 2); got != "1,2,..." || briefIntList(nil, 2) != "" {
		t.Fatal("brief integer list mismatch")
	}
	if got := humanBytes(100); got != "100 B" || humanBytes(2048) != "2.0 KB" || humanBytes(2*1024*1024) != "2.0 MB" {
		t.Fatal("human bytes mismatch")
	}
	_ = tea.KeyMsg{}
}

func TestTUIFormattingHelpers(t *testing.T) {
	for _, tc := range []struct {
		phase string
		want  string
	}{{"premise", "前提"}, {"outline", "大纲"}, {"writing", "写作"}, {"complete", "完成"}, {"init", "初始化"}, {"", "-"}, {"other", "other"}} {
		if got := snapshotPhaseLabel(tc.phase); got != tc.want {
			t.Errorf("phase %q = %q", tc.phase, got)
		}
	}
	for _, state := range []string{"running", "pausing", "paused", "completed", "other"} {
		if snapshotRuntimeStateLabel(state) == "" {
			t.Errorf("runtime state %q empty", state)
		}
	}
	for _, flow := range []string{"", "writing", "reviewing", "rewriting", "polishing", "steering", "other"} {
		if snapshotFlowLabel(flow) == "" {
			t.Errorf("flow %q empty", flow)
		}
	}
	if formatReportTime(time.Time{}) != "-" || formatReportTime(time.Date(2026, 9, 6, 1, 2, 3, 0, time.UTC)) != "2026-09-06 01:02:03" {
		t.Fatal("report time mismatch")
	}
	if got := formatSeverityCounts(1, 2, 3, utils.LanguageVI); got == "" || formatSeverityCounts(0, 0, 0, utils.LanguageVI) != "" {
		t.Fatal("severity count mismatch")
	}
	if renderEventDuration(0) != "" || renderEventDuration(time.Second) == "" {
		t.Fatal("event duration mismatch")
	}
	if max(4, 3) != 4 || max(3, 4) != 4 {
		t.Fatal("max mismatch")
	}
	if got := renderTopBar(host.UISnapshot{Provider: "p", ModelName: "m", ModelContextWindow: 1000, StatusLabel: "UNKNOWN", IsRunning: true}, 120, "*", "v1", utils.LanguageVI); got == "" {
		t.Fatal("top bar empty")
	}
	vp := viewport.New(20, 4)
	vp.SetContent("content")
	if renderStatePanel(vp, 24, 6, true) == "" || renderDetailPanel(vp, 24, 6, false) == "" {
		t.Fatal("panel frames empty")
	}
}

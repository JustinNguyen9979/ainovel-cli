package tui

import (
	"strings"
	"testing"
	"time"

	"github.com/JustinNguyen9979/ainovel-cli/internal/bootstrap"
	"github.com/JustinNguyen9979/ainovel-cli/internal/host"
	"github.com/JustinNguyen9979/ainovel-cli/internal/utils"
	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/charmbracelet/x/ansi"
)

func TestCommandPaletteScoringAndWindow(t *testing.T) {
	item := commandPaletteItem{Name: "model", Aliases: []string{"m"}, Usage: "/model", Description: "switch model"}
	for _, tc := range []struct {
		query string
		want  bool
	}{
		{"model", true}, {"m", true}, {"mod", true}, {"switch", true}, {"missing", false},
	} {
		if got := scoreCommandItem(item, tc.query) > 0; got != tc.want {
			t.Fatalf("score(%q) = %v", tc.query, got)
		}
	}
	if start, end := commandPaletteWindow(10, 0, 5); start != 0 || end != 5 {
		t.Fatalf("first window = %d/%d", start, end)
	}
	if start, end := commandPaletteWindow(10, 9, 5); start != 5 || end != 10 {
		t.Fatalf("last window = %d/%d", start, end)
	}
	if items := commandCompletions("model", utils.LanguageVI); len(items) == 0 || items[0].Name != "model" {
		t.Fatalf("model completions = %+v", items)
	}
}

func TestCoCreatePureHelpers(t *testing.T) {
	if startupModeQuick.label() == "" || startupModeCoCreate.subtitle() == "" || placeholderForNewMode(startupModeQuick) == "" {
		t.Fatal("startup labels should be non-empty")
	}
	if errorText(nil) != "" || errorText(assertError("  failed  ")) != "failed" {
		t.Fatal("errorText normalization failed")
	}
	state := newCoCreateState("idea", utils.LanguageVI)
	state.session.ApplyReply(host.CoCreateReply{Message: "reply", Prompt: "draft", Suggestions: []string{"one", "two"}})
	if got, ok := state.appendSuggestion(0, ""); !ok || got != "one" {
		t.Fatalf("first suggestion = %q/%v", got, ok)
	}
	if got, ok := state.appendSuggestion(1, "one"); !ok || got != "one；two" {
		t.Fatalf("second suggestion = %q/%v", got, ok)
	}
	if _, ok := state.appendSuggestion(0, "manual"); ok {
		t.Fatal("manual input should reset suggestion mode")
	}
	if got := extractReplyForDisplay("<reply>visible</reply><draft>hidden</draft>"); got != "visible" {
		t.Fatalf("reply extraction = %q", got)
	}
	if got := extractReplyForDisplay("plain response"); got != "plain response" {
		t.Fatalf("plain reply extraction = %q", got)
	}
	if left, right := coCreateColumns(100); left+right != 100 || left < 42 || right < 28 {
		t.Fatalf("columns = %d/%d", left, right)
	}
	if w, h := coCreateModalSize(120, 40); w <= 0 || h <= 0 {
		t.Fatalf("modal size = %d/%d", w, h)
	}
}

type tuiTestError string

func (e tuiTestError) Error() string { return string(e) }

func assertError(text string) error { return tuiTestError(text) }

func TestInputAndFormattingHelpers(t *testing.T) {
	if got := fitInlineLine("abcdef", 4); lipgloss.Width(got) > 4 {
		t.Fatalf("fit line = %q", got)
	}
	if got := joinInlineSides("left", "right", 12); lipgloss.Width(got) > 12 {
		t.Fatalf("joined line = %q", got)
	}
	if !isCSILeak([]rune("[1;2A")) || isCSILeak([]rune("[?")) || !containsSGRFragment("<1;2;") || containsSGRFragment("<x") {
		t.Fatal("escape fragment detection failed")
	}
	if contextPercentColor(10) != colorSuccess || contextPercentColor(70) != colorReview || contextPercentColor(90) != colorError {
		t.Fatal("context color thresholds failed")
	}
	if formatContextWindow(0) != "" || formatContextWindow(1000) != "1K" || formatContextWindow(1_000_000) != "1M" || formatCostUSD(0) != "" || formatCostUSD(0.001) != "$0.0010" || formatNumber(1234567) != "1,234,567" {
		t.Fatal("format helpers failed")
	}
}

func TestModelHistoryAndRuntimeProjection(t *testing.T) {
	m := NewModel(nil, "v1", utils.LanguageVI)
	m.pushInputHistory("one")
	m.pushInputHistory("one")
	m.pushInputHistory("two")
	m.historyIdx = len(m.inputHistory)
	m.textarea.SetValue("draft")
	if !m.tryHistoryUp() || m.textarea.Value() != "two" || !m.tryHistoryUp() || m.textarea.Value() != "one" {
		t.Fatal("history up failed")
	}
	if !m.tryHistoryDown() || m.textarea.Value() != "two" || !m.tryHistoryDown() || m.textarea.Value() != "draft" {
		t.Fatal("history down failed")
	}
	m.applyEventProjection(host.Event{ID: "x", Category: "TOOL", Summary: "running"})
	m.applyEventProjection(host.Event{ID: "x", Category: "TOOL", Summary: "done", Failed: true})
	if len(m.events) != 1 || m.events[0].Summary != "done" || !m.events[0].Failed {
		t.Fatalf("event projection = %+v", m.events)
	}
	if m.currentSpinnerFrame() != "" {
		t.Fatal("idle spinner should be empty")
	}
	m.textarea.SetValue("a\nb")
	if !m.textareaIsMultiline() || strings.TrimSpace(m.localizedSteerPlaceholder()) == "" {
		t.Fatal("textarea helpers failed")
	}
}

func TestTUIEventAndStreamRenderingBranches(t *testing.T) {
	now := time.Date(2026, 9, 6, 12, 34, 56, 0, time.UTC)
	events := []host.Event{
		{Time: now, ID: "decision", Category: "DECISION", Summary: "choose", Duration: time.Second},
		{Time: now, ID: "dispatch", Category: "DISPATCH", Summary: "writer（write）", FinishedAt: now, Duration: 1500 * time.Millisecond},
		{Time: now, ID: "tool", Category: "TOOL", Summary: "draft", Depth: 1, FinishedAt: now, Failed: true, Duration: 2 * time.Minute},
		{Time: now, Category: "ERROR", Summary: "failure", Duration: time.Millisecond},
		{Time: now, Category: "SYSTEM", Summary: "warning", Level: "warn", RetryAt: now.Add(3 * time.Second)},
		{Time: now, Category: "USER", Summary: "input"},
		{Time: now, Category: "CONTEXT", Summary: "context", Level: "debug"},
		{Time: now, Category: "CHECK", Summary: "check"},
		{Time: now, Category: "UNKNOWN", Summary: "other"},
	}
	for _, ev := range events {
		if got := renderEventLine(ev, 40, 2, utils.LanguageVI); ansi.Strip(got) == "" {
			t.Fatalf("empty event line for %q", ev.Category)
		}
	}
	if got := renderEventContent(events, 40, 1, utils.LanguageZH); got == "" {
		t.Fatal("event content should not be empty")
	}
	if retryCountdown(time.Time{}, now) != "" || retryCountdown(now.Add(-time.Second), now) != "" {
		t.Fatal("expired retry countdown should be empty")
	}
	if retryCountdown(now.Add(1500*time.Millisecond), now, utils.LanguageZH) != "2s 后重试" || retryCountdown(now.Add(1500*time.Millisecond), now, utils.LanguageVI) != "2s sau sẽ thử lại" {
		t.Fatal("localized retry countdown mismatch")
	}
	for _, tc := range []struct {
		d    time.Duration
		want string
	}{
		{500 * time.Millisecond, "500ms"},
		{1500 * time.Millisecond, "1.5s"},
		{61 * time.Second, "1m1s"},
	} {
		if got := formatDuration(tc.d); got != tc.want {
			t.Errorf("formatDuration(%s) = %q, want %q", tc.d, got, tc.want)
		}
	}
	if got := renderEventFlowViewport(viewport.New(30, 3), 30, 4, true, utils.LanguageVI); got == "" {
		t.Fatal("event viewport should render")
	}
	streamVP := viewport.New(30, 3)
	streamVP.SetContent("stream")
	if got := renderStreamPanel(streamVP, 30, 4, true, true, 1, utils.LanguageZH); got == "" {
		t.Fatal("stream panel should render")
	}
	if got := renderStreamContent([]string{"", "▸ writer\nwrite", "正文\n" + utils.ThinkingSep + "思考"}, 20, "cursor"); !strings.Contains(got, "writer") || !strings.Contains(got, "cursor") {
		t.Fatalf("stream content = %q", ansi.Strip(got))
	}
	for _, line := range []string{"- bullet", "  * item", "• dot", "12. ordered", "```go", "plain"} {
		prefix, content, next := parseWrapPrefix(line)
		if prefix == "" && content == "" && next == "" {
			t.Fatalf("prefix parser dropped %q", line)
		}
	}
	if orderedListPrefix("12. item") != "12. " || orderedListPrefix("x. item") != "" {
		t.Fatal("ordered list prefix mismatch")
	}
}

func TestTUISidebarAndLocaleBranches(t *testing.T) {
	agents := []host.AgentSnapshot{
		{Name: "writer", State: "running", TaskKind: "chapter_write", Summary: "writing", Tool: "draft", Context: host.AgentContextSnapshot{Tokens: 100, ContextWindow: 200, Percent: 50, Scope: "projected", Strategy: "full_summary"}},
		{Name: "editor", State: "failed", Summary: "failed"},
		{Name: "architect_long", State: "idle", Summary: "待命"},
		{Name: "other", State: "idle", Summary: "idle"},
	}
	if got := sidebarAgents(agents); len(got) != 2 || got[0].Name != "writer" {
		t.Fatalf("sidebar agents = %+v", got)
	}
	if got := sidebarIdleAgents(agents); len(got) != 2 || got[0] != "ARCHITECT_LONG" {
		t.Fatalf("idle agents = %+v", got)
	}
	for _, flow := range []string{"polishing", "rewriting", "writing", ""} {
		label, chapter := inProgressDisplay(host.UISnapshot{Flow: flow, InProgressChapter: 4, PendingRewrites: []int{3}}, utils.LanguageVI)
		if flow != "" && (label == "" || chapter == 0) {
			t.Fatalf("in progress %q = %q/%d", flow, label, chapter)
		}
	}
	snap := host.UISnapshot{
		RuntimeState: "paused", Phase: "writing", Flow: "rewriting", Layered: true, CompletedCount: 3,
		Outline: []host.OutlineSnapshot{{Chapter: 1, Title: "one"}}, InProgressChapter: 4, PendingRewrites: []int{3}, RewriteReason: "reason",
		PendingSteer: "steer", HasAdvanceHold: true, AdvanceHoldReason: "hold", RecoveryLabel: "重写恢复：第3章",
		TotalInputTokens: 12000, TotalOutputTokens: 3000, TotalCostUSD: 1.2, TotalSavedUSD: .4, BudgetLimitUSD: 2,
		OverallCacheCapable: true, TotalCacheReadTokens: 5000, TotalCacheWriteTokens: 100, OverallRecentCacheRead: 2000, OverallRecentInput: 3000, OverallRecentSamples: 2, TotalCacheBreaks: 1,
		CachePerAgent: []host.AgentCacheStat{{Role: "writer", Input: 8000, CacheRead: 4000, CacheCapable: true, RecentInput: 3000, RecentCacheRead: 2000, RecentSamples: 2}, {Role: "arbiter", Input: 100}},
		CachePerModel: []host.AgentCacheStat{{Model: "provider/model", Input: 1000, Output: 100}}, Agents: agents,
	}
	for _, lang := range []utils.Language{utils.LanguageVI, utils.LanguageZH} {
		text := renderStateContent(snap, 42, lang)
		if text == "" || !strings.Contains(ansi.Strip(text), "WRITER") {
			t.Fatalf("sidebar(%s) = %q", lang, ansi.Strip(text))
		}
		if contextScopeLabel("baseline", lang) == "" || contextStrategyLabel("full_summary", lang) == "" || taskKindLabel("chapter_rewrite", lang) == "" {
			t.Fatalf("localized sidebar labels missing for %s", lang)
		}
	}
	for _, state := range []string{"running", "failed", "idle", "unknown"} {
		if agentStateLabel(state) == "" || agentStateIcon(state) == "" || taskStatusColor(state) == (lipgloss.AdaptiveColor{}) {
			t.Fatalf("agent state helpers failed for %q", state)
		}
	}
	for _, value := range []string{"", "第2卷·第3弧", "bad"} {
		_ = localizedVolumeArc(utils.LanguageVI, value)
		_ = localizedVolumeArc(utils.LanguageZH, value)
	}
	if got := localizedVolumeArc(utils.LanguageVI, "第2卷·第3弧"); got != "Tập 2 · Cung 3" {
		t.Fatalf("volume arc = %q", got)
	}
	for _, phase := range []string{"premise", "outline", "writing", "complete", "init", "unknown", ""} {
		if localizedPhaseLabel(utils.LanguageVI, phase) == "" || localizedPhaseLabel(utils.LanguageZH, phase) == "" {
			t.Fatalf("phase label missing for %q", phase)
		}
	}
	for _, value := range []string{"打磨恢复：第3章", "恢复：规划阶段", "提交中断", "plain"} {
		if localizedRecoveryLabel(utils.LanguageVI, value) == "" {
			t.Fatalf("recovery label missing for %q", value)
		}
	}
}

func TestTUIModelConfigAndStateBranches(t *testing.T) {
	state := &modelConfigState{language: utils.LanguageVI, provider: "openai", providerType: "openai", api: "responses", baseURL: "https://example.test", models: []bootstrap.ModelConfig{{Name: "old", ContextWindow: 128000}}, modelOrigins: []string{"old"}, currentModel: "old", hasAPIKey: true, apiKeyAction: host.APIKeyKeep, apiKeyHint: "sk-******"}
	state.snapshot = host.ModelConfigurationSnapshot{DefaultProvider: "openai", DefaultModel: "old", References: map[string][]string{"openai\x00old": {"default", "writer"}}}
	state.captureBaseline()
	if !state.isOpenAIEndpoint() || state.testModelName() != "old" || len(state.hubFields()) == 0 || !state.hasEffectiveAPIKey() {
		t.Fatal("model config baseline helpers failed")
	}
	for _, id := range []string{"protocol", "api", "key", "baseurl", "models", "test", "save", "unknown"} {
		state.step = configStepHub
		state.editingField = ""
		state.enterHubField(id)
	}
	if protocolIndex("anthropic") != 1 || protocolIndex("missing") != 0 {
		t.Fatal("protocol index mismatch")
	}
	for _, raw := range []string{"", "0", "auto", "128k", "1.5m", "bad", "-1", "1.2.3"} {
		_, _ = parseContextWindowInput(raw, utils.LanguageVI)
	}
	if got, err := parseContextWindowInput("128K", utils.LanguageZH); err != nil || got != 128000 {
		t.Fatalf("context window = %d/%v", got, err)
	}
	state.step = configStepModels
	state.ensureModelOrigins()
	state.beginModelEdit(0, 0)
	state.input.SetValue("new")
	if _, ok := state.finishModelEdit(); !ok {
		t.Fatal("model rename should succeed")
	}
	state.editModelIdx = 0
	state.editingField = configModelWindowField
	state.input.SetValue("256k")
	if _, ok := state.finishModelEdit(); !ok || state.models[0].ContextWindow != 256000 {
		t.Fatal("model window edit should succeed")
	}
	state.cancelModelEdit()
	if state.deleteModel(-1) || state.deleteModel(9) {
		t.Fatal("invalid model deletion should fail")
	}
	state.modelOrigins = []string{"old"}
	state.models[0].Name = "old"
	state.currentModel = "old"
	if state.deleteModel(0) {
		t.Fatal("current model deletion should fail")
	}
	state.currentModel = ""
	if state.deleteModel(0) {
		t.Fatal("referenced model deletion should fail")
	}
	state.snapshot.References = nil
	if !state.deleteModel(0) {
		t.Fatal("unreferenced model deletion should succeed")
	}
	for _, step := range []configStep{configStepProvider, configStepAddPicker, configStepCustomName, configStepHub, configStepProtocol, configStepAPI, configStepModels} {
		state.step = step
		state.escapeBack()
	}
	if renderModelConfigModal(100, state) == "" || renderModelConfigModal(0, nil) != "" {
		t.Fatal("model config modal rendering failed")
	}
	if renderConfigTextInput(&state.input, 30) == "" || renderConfigChoices(nil, 0, 30, 5) == nil {
		t.Fatal("config rendering helpers failed")
	}
	_ = tea.KeyMsg{Type: tea.KeyEnter}
}

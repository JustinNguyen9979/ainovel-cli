package tui

import (
	"strings"
	"testing"

	"github.com/JustinNguyen9979/ainovel-cli/internal/bootstrap"
	"github.com/JustinNguyen9979/ainovel-cli/internal/host"
	"github.com/JustinNguyen9979/ainovel-cli/internal/host/imp"
	"github.com/JustinNguyen9979/ainovel-cli/internal/host/sim"
	"github.com/JustinNguyen9979/ainovel-cli/internal/utils"
	"github.com/charmbracelet/lipgloss"
	"github.com/charmbracelet/x/ansi"
)

func TestLocalizedModeAndPlaceholder(t *testing.T) {
	for _, tc := range []struct {
		name string
		lang utils.Language
		want string
	}{
		{name: "vietnamese", lang: utils.LanguageVI, want: "Nhập"},
		{name: "chinese", lang: utils.LanguageZH, want: "输入"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			m := NewModel(nil, "", tc.lang)
			if !strings.Contains(m.textarea.Placeholder, tc.want) {
				t.Fatalf("placeholder = %q, want %q", m.textarea.Placeholder, tc.want)
			}
			welcome := lipgloss.NewStyle().Render(renderWelcome(120, 20, "", startupModeQuick, "", "", tc.lang))
			if !strings.Contains(welcome, tc.want) {
				t.Fatalf("welcome = %q, want %q", welcome, tc.want)
			}
		})
	}
}

func TestLocalizedRenderersStayWithinWidth(t *testing.T) {
	// Each locale is measured using terminal display width, not rune count.
	snap := host.UISnapshot{
		Provider: "provider", ModelName: "model", BookTitle: strings.Repeat("长", 20),
		RuntimeState: "running", Phase: "writing", Flow: "writing", TotalChapters: 12,
		CompletedCount: 4, TotalWordCount: 12000,
		Agents: []host.AgentSnapshot{{Name: "writer", State: "running", TaskKind: "chapter_write"}},
	}
	for _, lang := range []utils.Language{utils.LanguageVI, utils.LanguageZH} {
		out := ansi.Strip(renderStateContent(snap, 32, lang))
		for _, line := range strings.Split(out, "\n") {
			if width := lipgloss.Width(line); width > 32 {
				t.Fatalf("lang=%s line width=%d > 32: %q", lang, width, line)
			}
		}
	}
}

func TestChineseLocaleCoversMainRenderers(t *testing.T) {
	lang := utils.LanguageZH
	modelState := &modelSwitchState{language: lang, thinking: []thinkingOption{{Key: "high"}}, thinkingIdx: 0}
	if got := ansi.Strip(renderModelSwitchBar(100, modelState)); !strings.Contains(got, "推理强度") || !strings.Contains(got, "高") {
		t.Fatalf("model switch should use Chinese labels: %q", got)
	}
	configState := &modelConfigState{language: lang, step: configStepModels, models: []bootstrap.ModelConfig{{Name: "model"}}}
	configState.modelOrigins = []string{"model"}
	config := ansi.Strip(renderModelConfigModal(120, configState))
	if !strings.Contains(config, "上下文窗口") || !strings.Contains(config, "新增模型") {
		t.Fatalf("config modal should use Chinese labels: %q", config)
	}
	snap := host.UISnapshot{TotalInputTokens: 1000, TotalCacheReadTokens: 500, OverallCacheCapable: true, CachePerAgent: []host.AgentCacheStat{{Role: "writer", Input: 1000, CacheRead: 500, CacheCapable: true}}}
	cache := ansi.Strip(renderStateContent(snap, 40, lang))
	if !strings.Contains(cache, "累计命中") || !strings.Contains(cache, "缓存读量") {
		t.Fatalf("cache sidebar should use Chinese labels: %q", cache)
	}
	commands := commandSpecs(lang)
	if len(commands) == 0 || commands[0].Description == "" {
		t.Fatal("Chinese command catalog should contain localized descriptions")
	}
}

func TestLanguageCatalogHasBothLocales(t *testing.T) {
	keys := []utils.MessageKey{
		utils.MsgReady, utils.MsgLoading, utils.MsgTerminalTooNarrow, utils.MsgUnknownCommand,
		utils.MsgCommandIdleOnly, utils.MsgCommandUsage, utils.MsgUnknownRole,
		utils.MsgHelpCommandDescription, utils.MsgModelCommandDescription, utils.MsgConfigCommandDescription,
		utils.MsgDiagCommandDescription, utils.MsgReviewCommandDescription, utils.MsgNextCommandDescription,
		utils.MsgStartCommandDescription, utils.MsgImportCommandDescription, utils.MsgReopenCommandDescription,
		utils.MsgCocreateCommandDescription, utils.MsgSimulateCommandDescription, utils.MsgImportSimCommandDescription,
		utils.MsgSyncCommandDescription, utils.MsgExportCommandDescription, utils.MsgNewBookTitle,
		utils.MsgNoBookTitle, utils.MsgInputNovelRequest, utils.MsgStartingQuick, utils.MsgStartingCreation,
		utils.MsgCreationFinished, utils.MsgCreationPaused, utils.MsgCreationInterrupted, utils.MsgCreationPausing,
		utils.MsgCreationResume, utils.MsgReviewWaiting, utils.MsgReviewNext, utils.MsgSteerPlaceholder,
		utils.MsgDonePlaceholder, utils.MsgQuickMode, utils.MsgCoCreateMode, utils.MsgQuickModeSubtitle,
		utils.MsgCoCreateModeSubtitle, utils.MsgStartupMode, utils.MsgQuickPlaceholder, utils.MsgCoCreatePlaceholder,
		utils.MsgAIThinking, utils.MsgAIReplying, utils.MsgSend, utils.MsgExit, utils.MsgScroll, utils.MsgClose,
		utils.MsgCancel, utils.MsgAccept, utils.MsgCommandHelpTitle, utils.MsgShortcuts, utils.MsgCommandSearchHint,
		utils.MsgCommandAcceptHint, utils.MsgCommandCloseHint, utils.MsgCommandPaletteTitle, utils.MsgOverview,
		utils.MsgRuntimeState, utils.MsgPhase, utils.MsgFlow, utils.MsgProgress, utils.MsgCompleted, utils.MsgPlanned,
		utils.MsgWordCount, utils.MsgCurrent, utils.MsgWaitingResume, utils.MsgRunningRoles, utils.MsgQueue,
		utils.MsgReason, utils.MsgRework, utils.MsgIntervention, utils.MsgPending, utils.MsgAcceptanceHold,
		utils.MsgWaiting, utils.MsgUsage, utils.MsgCache, utils.MsgInputTokens, utils.MsgOutputTokens, utils.MsgCost,
		utils.MsgSaved, utils.MsgBudget, utils.MsgRole, utils.MsgModel, utils.MsgCacheHit, utils.MsgCacheRead,
		utils.MsgCacheWrite, utils.MsgCacheDisabled, utils.MsgAutoCacheNoPremium, utils.MsgLinkBreak, utils.MsgChapter,
		utils.MsgVolume, utils.MsgSynopsis, utils.MsgOutline, utils.MsgCharacters, utils.MsgSummary,
		utils.MsgEventStream, utils.MsgLiveOutput, utils.MsgExternalImport, utils.MsgProcessLog, utils.MsgImportFailed,
		utils.MsgImportComplete, utils.MsgImportPaused, utils.MsgSimulationProfile, utils.MsgSimulationFailed,
		utils.MsgSimulationReady, utils.MsgDiagnosticReport, utils.MsgReportUnavailable, utils.MsgReportLoading,
		utils.MsgNoProblems, utils.MsgFindings, utils.MsgActions, utils.MsgConfigModel, utils.MsgSelectProvider,
		utils.MsgAddProvider, utils.MsgProviderName, utils.MsgProtocolType, utils.MsgAPIKey, utils.MsgBaseURL,
		utils.MsgModelList, utils.MsgContextWindow, utils.MsgReferences, utils.MsgNoOptions, utils.MsgNewModel,
		utils.MsgAutomatic, utils.MsgConnectionTest, utils.MsgSaveConfig, utils.MsgLanguage, utils.MsgLanguageVietnamese,
		utils.MsgLanguageChinese, utils.MsgLanguageSavedRestart, utils.MsgSetupMissing, utils.MsgSetupPath,
		utils.MsgSetupEditHint, utils.MsgHeadlessLogWarning, utils.MsgHeadlessDiagnosticWarning, utils.MsgHeadlessStart,
		utils.MsgHeadlessResume, utils.MsgHeadlessNeedsPrompt,
	}
	for _, lang := range []utils.Language{utils.LanguageVI, utils.LanguageZH} {
		for _, key := range keys {
			if got := utils.T(lang, key); got == "" {
				t.Fatalf("missing translation for %s/%s", lang, key)
			}
		}
	}
}

func TestLocalizedImportAndSimulationStages(t *testing.T) {
	if got := localizedImportStage(utils.LanguageZH, imp.StagePublishing); got != "发布中" {
		t.Fatalf("Chinese import stage = %q", got)
	}
	if got := localizedImportStage(utils.LanguageVI, imp.StagePublishing); got != "Đang xuất bản" {
		t.Fatalf("Vietnamese import stage = %q", got)
	}
	if got := localizedSimulationStage(utils.LanguageZH, sim.StageMerge); got != "合并中" {
		t.Fatalf("Chinese simulation stage = %q", got)
	}
	if got := localizedPlanningTier(utils.LanguageVI, "long"); got != "Dài" {
		t.Fatalf("Vietnamese planning tier = %q", got)
	}
	if got := localizedPlanningTier(utils.LanguageZH, "short"); got != "短篇" {
		t.Fatalf("Chinese planning tier = %q", got)
	}
}

func TestChineseCompletionHintIsLocalized(t *testing.T) {
	if got := localizedCompletionHint(utils.LanguageZH); got != "再次按 Ctrl+C 退出" {
		t.Fatalf("Chinese completion hint = %q", got)
	}
}

func TestChineseWelcomeUsesChineseExamples(t *testing.T) {
	out := renderWelcome(120, 28, "", startupModeQuick, "", "", utils.LanguageZH)
	if strings.Contains(out, "Viết truyện") || !strings.Contains(out, "都市推理小说") {
		t.Fatalf("Chinese welcome examples are not localized: %q", out)
	}
}

func TestCommandCatalogDescriptionsLocalized(t *testing.T) {
	vi := commandSpecs(utils.LanguageVI)
	zh := commandSpecs(utils.LanguageZH)
	if len(vi) != len(zh) {
		t.Fatalf("locale command counts differ: vi=%d zh=%d", len(vi), len(zh))
	}
	for i := range vi {
		if vi[i].Description == "" || zh[i].Description == "" {
			t.Fatalf("empty description for command %q", vi[i].Name)
		}
	}
	for _, spec := range vi {
		if spec.Name == "import" || spec.Name == "reopen" {
			if strings.ContainsAny(spec.Usage, "切分指导续写方向") {
				t.Fatalf("Vietnamese usage for /%s contains Chinese: %q", spec.Name, spec.Usage)
			}
		}
	}
}

func TestStageCoCreateOpenerLocalized(t *testing.T) {
	if got := stageCoCreateOpener(utils.LanguageVI); strings.ContainsAny(got, "暂停规划走向") {
		t.Fatalf("Vietnamese stage opener contains Chinese: %q", got)
	}
	if got := stageCoCreateOpener(utils.LanguageZH); !strings.Contains(got, "暂停") {
		t.Fatalf("Chinese stage opener is not Chinese: %q", got)
	}
}

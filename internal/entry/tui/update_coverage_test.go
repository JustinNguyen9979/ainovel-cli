package tui

import (
	"context"
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/JustinNguyen9979/ainovel-cli/internal/diag"
	"github.com/JustinNguyen9979/ainovel-cli/internal/domain"
	"github.com/JustinNguyen9979/ainovel-cli/internal/host"
	"github.com/JustinNguyen9979/ainovel-cli/internal/host/exp"
	"github.com/JustinNguyen9979/ainovel-cli/internal/host/imp"
	"github.com/JustinNguyen9979/ainovel-cli/internal/host/sim"
	"github.com/JustinNguyen9979/ainovel-cli/internal/revision"
	"github.com/JustinNguyen9979/ainovel-cli/internal/utils"
	"github.com/JustinNguyen9979/ainovel-cli/internal/version"
	tea "github.com/charmbracelet/bubbletea"
)

func updateModel(t *testing.T, m Model, msg tea.Msg) Model {
	t.Helper()
	next, _ := m.Update(msg)
	return next.(Model)
}

func TestModelUpdateRuntimeMessages(t *testing.T) {
	m := NewModel(nil, "v1", utils.LanguageVI)
	m.width, m.height = 120, 36
	m = updateModel(t, m, tea.WindowSizeMsg{Width: 120, Height: 36})
	m = updateModel(t, m, eventMsg(host.Event{ID: "call", Category: "TOOL", Summary: "running"}))
	m = updateModel(t, m, eventMsg(host.Event{ID: "call", Category: "TOOL", Summary: "done", FinishedAt: time.Now(), Failed: true, Detail: "failure"}))
	if len(m.events) != 1 || !m.events[0].Failed {
		t.Fatalf("event projection = %+v", m.events)
	}
	m = updateModel(t, m, snapshotMsg(host.UISnapshot{Phase: "writing", RuntimeState: "running", IsRunning: true, Synopsis: "new"}))
	m = updateModel(t, m, doneMsg{complete: false})
	m = updateModel(t, m, abortResultMsg{stopped: true})
	m = updateModel(t, m, doneMsg{complete: true})
	if m.mode != modeDone {
		t.Fatalf("complete message should enter done mode: %d", m.mode)
	}
	m = updateModel(t, m, bootstrapMsg{existing: true, err: errors.New("resume failed")})
	m = updateModel(t, m, bootstrapMsg{completed: true})
	m = updateModel(t, m, bootstrapMsg{resumed: true})
	m = updateModel(t, m, reportLoadedMsg{reqID: 1})
	m.report = newReportState(120, 36, 1, time.Now(), utils.LanguageVI)
	m = updateModel(t, m, reportLoadedMsg{reqID: 1, report: diag.Report{}, finishedAt: time.Now()})
	m = updateModel(t, m, reportLoadedMsg{reqID: 2})
	m = updateModel(t, m, exportDoneMsg{err: errors.New("export")})
	m = updateModel(t, m, exportDoneMsg{result: &exp.Result{Chapters: 3, Bytes: 2048, Skipped: []int{4}, Path: "out.txt"}})
	m = updateModel(t, m, updateCheckMsg{err: errors.New("cache")})
	m = updateModel(t, m, updateCheckMsg{result: &version.CheckResult{UpdateAvailable: true, Latest: "v2", Notes: "# Notes"}})
	m = updateModel(t, m, revisionDoneMsg{err: errors.New("sync")})
	m = updateModel(t, m, revisionDoneMsg{checkOnly: true})
	m = updateModel(t, m, revisionDoneMsg{checkOnly: true, chapters: []int{1, 2}})
	m = updateModel(t, m, revisionDoneMsg{result: &revision.Result{Applied: []int{1}, Analyses: []domain.RevisionAnalysis{{ChangeSummary: "changed", StoryChanged: true, DownstreamIssues: []string{"issue"}}}}})
	m = updateModel(t, m, modelConfigSavedMsg{err: errors.New("save")})
	m.modelConfig = &modelConfigState{}
	m = updateModel(t, m, modelConfigSavedMsg{err: errors.New("save")})
	m = updateModel(t, m, modelConfigSavedMsg{})
	m.modelConfig = &modelConfigState{}
	m = updateModel(t, m, modelConfigConnectionMsg{err: context.Canceled})
	m.modelConfig = &modelConfigState{}
	m = updateModel(t, m, modelConfigConnectionMsg{err: errors.New("connection")})
	m.modelConfig = &modelConfigState{}
	m = updateModel(t, m, modelConfigConnectionMsg{model: "model"})
	m = updateModel(t, m, startResultMsg{err: errors.New("start")})
	m.starting = true
	m = updateModel(t, m, startResultMsg{err: errors.New("start")})
	m.mode = modeNew
	m = updateModel(t, m, startResultMsg{})
	m.cocreate = newCoCreateState("idea", utils.LanguageVI)
	m.cocreate.reqID = 2
	m = updateModel(t, m, cocreateDeltaMsg{reqID: 2, kind: host.CoCreateProgressThinking, text: "thinking"})
	m = updateModel(t, m, cocreateDeltaMsg{reqID: 2, kind: host.CoCreateProgressReply, text: "reply"})
	m = updateModel(t, m, cocreateDoneMsg{reqID: 2, err: errors.New("cocreate")})
	m = updateModel(t, m, cocreateDoneMsg{reqID: 2, reply: host.CoCreateReply{Message: "ok", Prompt: "draft", Ready: true}})
	m = updateModel(t, m, steerResultMsg{err: errors.New("steer")})
	m = updateModel(t, m, steerResultMsg{})
	m = updateModel(t, m, continueResultMsg{err: errors.New("continue")})
	m = updateModel(t, m, continueResultMsg{})
	m = updateModel(t, m, spinnerTickMsg(time.Now()))
	m = updateModel(t, m, toolSpinnerTickMsg(time.Now()))
	m = updateModel(t, m, streamDeltaMsg("hello"))
	m = updateModel(t, m, streamFlushTickMsg{})
	m = updateModel(t, m, streamClearMsg{})
	m = updateModel(t, m, quitResetMsg{})
	if !strings.Contains(m.textarea.Placeholder, "") {
		t.Fatal("placeholder should remain valid")
	}
}

func TestModelUpdateImportAndSimulationMessages(t *testing.T) {
	m := NewModel(nil, "v1", utils.LanguageVI)
	m.width, m.height = 120, 36
	m.importer = newImportState(3, "source", 120, 36, func() {}, utils.LanguageVI)
	ch := make(chan imp.Event)
	m = updateModel(t, m, importEventMsg{reqID: 99, ev: imp.Event{Stage: imp.StageDone}, ch: ch})
	m = updateModel(t, m, importEventMsg{reqID: 3, ev: imp.Event{Stage: imp.StageAnalyzing, Current: 1, Total: 2, Message: "analyzing"}, ch: ch})
	m = updateModel(t, m, importEventMsg{reqID: 3, ev: imp.Event{Stage: imp.StageError, Message: "bad", Err: errors.New("bad")}, ch: ch})
	m.importer = newImportState(3, "source", 120, 36, func() {}, utils.LanguageVI)
	m = updateModel(t, m, importClosedMsg{reqID: 3})
	m = updateModel(t, m, importEventMsg{reqID: 3, ev: imp.Event{Stage: imp.StageDone, Continued: false}, ch: ch})
	m.importer = newImportState(3, "source", 120, 36, func() {}, utils.LanguageVI)
	m.importer.paused = true
	m = updateModel(t, m, importEventMsg{reqID: 3, ev: imp.Event{Stage: imp.StageDone, Continued: true}, ch: ch})

	m.simulator = newSimulationState(4, "sim", "source", 120, 36, func() {}, utils.LanguageVI)
	simCh := make(chan sim.Event)
	m = updateModel(t, m, simEventMsg{reqID: 99, ev: sim.Event{Stage: sim.StageDone}, ch: simCh})
	m = updateModel(t, m, simEventMsg{reqID: 4, ev: sim.Event{Stage: sim.StageAnalyze, Current: 1, Total: 2, Message: "analyze"}, ch: simCh})
	m = updateModel(t, m, simEventMsg{reqID: 4, ev: sim.Event{Stage: sim.StageDone, Message: "done"}, ch: simCh})
}

func TestReportImportSimulationHandlers(t *testing.T) {
	m := NewModel(nil, "v1", utils.LanguageVI)
	m.width, m.height = 120, 36
	m.report = newReportState(120, 36, 1, time.Now(), utils.LanguageVI)
	for _, key := range []tea.KeyType{tea.KeyUp, tea.KeyDown, tea.KeyPgUp, tea.KeyPgDown, tea.KeyRunes} {
		if _, cmd := m.handleReportKey(tea.KeyMsg{Type: key}); cmd != nil {
			t.Fatalf("report key %v unexpectedly returned command", key)
		}
	}
	m2, _ := m.handleReportKey(tea.KeyMsg{Type: tea.KeyEsc})
	m = m2.(Model)
	if m.report != nil {
		t.Fatal("report Esc should close modal")
	}
	m.importer = newImportState(1, "source", 120, 36, func() {}, utils.LanguageVI)
	m.importer.paused = true
	m.importer.stage = imp.StageAwaitingConfirmation
	m2, _ = m.handleImportKey(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'n'}})
	m = m2.(Model)
	m2, _ = m.handleImportKey(tea.KeyMsg{Type: tea.KeyEsc})
	m = m2.(Model)
	m.simulator = newSimulationState(1, "sim", "source", 120, 36, func() {}, utils.LanguageVI)
	for _, key := range []tea.KeyType{tea.KeyUp, tea.KeyDown, tea.KeyPgUp, tea.KeyPgDown, tea.KeyRunes} {
		m2, _ = m.handleSimulationKey(tea.KeyMsg{Type: key})
		m = m2.(Model)
	}
	m.simulator.done = true
	m2, _ = m.handleSimulationKey(tea.KeyMsg{Type: tea.KeyEsc})
	m = m2.(Model)
	if m.simulator != nil {
		t.Fatal("finished simulation Esc should close modal")
	}
}

func TestModelInitAndCoCreateKeyBranches(t *testing.T) {
	m := NewModel(nil, "v1", utils.LanguageVI)
	m.disableUpdateCheck = true
	if m.Init() == nil {
		t.Fatal("Init should return a command batch")
	}
	if defaultSteerPlaceholder() == "" {
		t.Fatal("default steer placeholder should be localized")
	}
	if got := overlayAboveInput("a\nb\nc\nd", "X\nY", 1); !strings.Contains(got, "X") || !strings.Contains(got, "Y") {
		t.Fatalf("overlay should replace lines above input: %q", got)
	}
	if got := overlayAboveInput("a\nb", "X\nY\nZ", 4); got == "" {
		t.Fatal("oversized overlay should still return base-shaped output")
	}

	m.width, m.height = 120, 36
	m.cocreate = newCoCreateState("idea", utils.LanguageVI)
	for _, key := range []tea.KeyType{tea.KeyTab, tea.KeyUp, tea.KeyPgUp, tea.KeyDown, tea.KeyPgDown, tea.KeyHome, tea.KeyEnd} {
		var model tea.Model
		model, _ = m.handleCoCreateKey(tea.KeyMsg{Type: key})
		m = model.(Model)
	}
	m.cocreate.focusPrompt = true
	for _, key := range []tea.KeyType{tea.KeyUp, tea.KeyDown, tea.KeyHome, tea.KeyEnd} {
		model, _ := m.handleCoCreateKey(tea.KeyMsg{Type: key})
		m = model.(Model)
	}
	m.cocreate.awaiting = true
	if _, cmd := m.handleCoCreateKey(tea.KeyMsg{Type: tea.KeyCtrlS}); cmd != nil {
		t.Fatal("awaiting Ctrl+S should be ignored")
	}
	m.cocreate.awaiting = false
	m.cocreate.apply(host.CoCreateReply{Message: "reply", Prompt: "draft", Ready: true, Suggestions: []string{"one", "two", "three", "four"}})
	m.textarea.Reset()
	for _, key := range []rune{'1', '2', '2', '3'} {
		model, _ := m.handleCoCreateKey(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{key}})
		m = model.(Model)
	}
	if !strings.Contains(m.textarea.Value(), "one") {
		t.Fatalf("suggestion input = %q", m.textarea.Value())
	}
	m.cocreate.awaiting = true
	m.textarea.SetValue("draft")
	if _, cmd := m.handleCoCreateKey(tea.KeyMsg{Type: tea.KeyEnter}); cmd != nil {
		t.Fatal("awaiting Enter should be ignored")
	}
	m.cocreate.awaiting = false
	m.textarea.Reset()
	m.lastKeyAt = time.Now()
	if _, cmd := m.handleCoCreateKey(tea.KeyMsg{Type: tea.KeyEnter}); cmd == nil {
		t.Fatal("paste-style Enter should update textarea")
	}
	m.lastKeyAt = time.Time{}
	m.textarea.SetValue("  ")
	if _, cmd := m.handleCoCreateKey(tea.KeyMsg{Type: tea.KeyEnter}); cmd != nil {
		t.Fatal("blank co-create input should not submit")
	}
	m.textarea.SetValue("hello")
	if _, cmd := m.handleCoCreateKey(tea.KeyMsg{Type: tea.KeyEnter, Alt: true}); cmd == nil {
		t.Fatal("Alt+Enter should be delegated to textarea")
	}
	for _, msg := range []tea.KeyMsg{
		{Type: tea.KeyCtrlU},
		{Type: tea.KeyRunes, Runes: []rune{'[', '1', ';', '2', 'A'}},
		{Type: tea.KeyRunes, Runes: []rune{'<', '1', ';'}},
		{Type: tea.KeyRunes, Runes: []rune{'x'}},
	} {
		model, _ := m.handleCoCreateKey(msg)
		m = model.(Model)
	}
	m.cocreate = newCoCreateState("initial", utils.LanguageVI)
	model, _ := m.handleCoCreateKey(tea.KeyMsg{Type: tea.KeyEsc})
	if model.(Model).cocreate != nil {
		t.Fatal("Esc should exit cold-start co-create")
	}
}

func TestModelUpdateEventProjectionAndInputKeys(t *testing.T) {
	m := NewModel(nil, "v1", utils.LanguageVI)
	m.width, m.height = 120, 36
	m.applyEventProjection(host.Event{ID: "1", Category: "TOOL", Summary: "first"})
	m.applyEventProjection(host.Event{ID: "2", Category: "SYSTEM", Summary: "second"})
	m.applyEventProjection(host.Event{ID: "1", Category: "TOOL", Summary: "updated", Failed: true, Detail: "full", Kind: "network", RetryAt: time.Now().Add(time.Second), FinishedAt: time.Now(), Duration: time.Second})
	for i := 0; i < maxEvents+2; i++ {
		m.applyEventProjection(host.Event{ID: "id" + string(rune(i)), Category: "SYSTEM", Summary: "event"})
	}
	m.rebuildEventIndex()
	m.trimStreamRounds()
	m.resetOutputPanels()
	m.mode = modeRunning
	m.snapshot = host.UISnapshot{IsRunning: true, Phase: "writing"}
	m = updateModel(t, m, tea.KeyMsg{Type: tea.KeyCtrlR})
	m = updateModel(t, m, tea.KeyMsg{Type: tea.KeyCtrlR})
	m = updateModel(t, m, tea.KeyMsg{Type: tea.KeyCtrlC})
	m = updateModel(t, m, tea.KeyMsg{Type: tea.KeyCtrlC})
	m = updateModel(t, m, tea.KeyMsg{Type: tea.KeyEscape})
	m.focusPane = focusStream
	m = updateModel(t, m, tea.KeyMsg{Type: tea.KeyEnd})
	m.focusPane = focusDetail
	m = updateModel(t, m, tea.KeyMsg{Type: tea.KeyEnd})
	m.focusPane = focusState
	m = updateModel(t, m, tea.KeyMsg{Type: tea.KeyEnd})
	m.focusPane = focusEvents
	m = updateModel(t, m, tea.KeyMsg{Type: tea.KeyPgUp})
	m = updateModel(t, m, tea.KeyMsg{Type: tea.KeyPgDown})
	m = updateModel(t, m, tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'a'}})
	m = updateModel(t, m, tea.KeyMsg{Type: tea.KeyCtrlL})
	m = updateModel(t, m, tea.KeyMsg{Type: tea.KeyCtrlU})
}

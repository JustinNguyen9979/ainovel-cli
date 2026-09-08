package tui

import (
	"errors"
	"testing"
	"time"

	"github.com/JustinNguyen9979/ainovel-cli/internal/host/imp"
	"github.com/JustinNguyen9979/ainovel-cli/internal/host/sim"
	"github.com/JustinNguyen9979/ainovel-cli/internal/utils"
	tea "github.com/charmbracelet/bubbletea"
)

func TestCommandRegistryInvalidAndSafeHandlers(t *testing.T) {
	m := NewModel(nil, "v1", utils.LanguageVI)
	m.width, m.height = 120, 36
	for _, spec := range commandRegistryInstance(utils.LanguageVI).Visible() {
		var args []string
		switch spec.Name {
		case "lang":
			args = []string{"invalid"}
		case "help":
			args = nil
		case "model":
			args = []string{"unknown-role"}
		case "config":
			args = []string{"unexpected"}
		case "review":
			args = []string{"maybe"}
		case "next":
			args = []string{"unexpected"}
		case "start":
			m.mode = modeRunning
			args = []string{"path"}
		case "import":
			args = []string{"--unknown"}
		case "cocreate":
			m.mode = modeNew
			args = nil
		case "simulate":
			args = []string{"unexpected"}
		case "importsim":
			args = nil
		case "sync":
			args = []string{"--unknown"}
		case "export":
			args = []string{"--unknown"}
		default:
			continue
		}
		if _, _ = spec.Run(m, args); false {
			t.Fatal("unreachable")
		}
	}
	if _, ok := commandRegistryInstance().Find("not-registered"); ok {
		t.Fatal("unknown command should not be registered")
	}
}

func TestTUIOverlayAndModalKeyHandlers(t *testing.T) {
	m := NewModel(nil, "v1", utils.LanguageVI)
	m.width, m.height = 120, 36

	m.help = newHelpState(120, 36, utils.LanguageVI)
	for _, key := range []tea.KeyType{tea.KeyUp, tea.KeyDown, tea.KeyPgUp, tea.KeyPgDown, tea.KeyRunes} {
		m = updateModel(t, m, tea.KeyMsg{Type: key})
	}
	m = updateModel(t, m, tea.KeyMsg{Type: tea.KeyEsc})
	if m.help != nil {
		t.Fatal("Esc should close help")
	}

	m.modelSwitch = &modelSwitchState{language: utils.LanguageVI}
	m = updateModel(t, m, tea.KeyMsg{Type: tea.KeyEnter})
	if m.modelSwitch == nil || m.modelSwitch.message == "" {
		t.Fatal("empty model switch should expose an error")
	}
	m = updateModel(t, m, tea.KeyMsg{Type: tea.KeyEsc})
	if m.modelSwitch != nil {
		t.Fatal("Esc should close model switch")
	}

	m.importer = newImportState(1, "source", 120, 36, func() {}, utils.LanguageVI)
	for _, key := range []tea.KeyType{tea.KeyUp, tea.KeyDown, tea.KeyPgUp, tea.KeyPgDown} {
		m = updateModel(t, m, tea.KeyMsg{Type: key})
	}
	m.importer.cancel = func() { m.importer.paused = true }
	m = updateModel(t, m, tea.KeyMsg{Type: tea.KeyEsc})
	if m.importer == nil {
		t.Fatal("running import Esc should cancel, not close")
	}
	m.importer.done = true
	m = updateModel(t, m, tea.KeyMsg{Type: tea.KeyEsc})
	if m.importer != nil {
		t.Fatal("finished import Esc should close")
	}

	m.simulator = newSimulationState(1, "sim", "source", 120, 36, func() {}, utils.LanguageVI)
	for _, key := range []tea.KeyType{tea.KeyUp, tea.KeyDown, tea.KeyPgUp, tea.KeyPgDown} {
		m = updateModel(t, m, tea.KeyMsg{Type: key})
	}
	m.simulator.cancel = func() { m.simulator.done = true }
	m = updateModel(t, m, tea.KeyMsg{Type: tea.KeyEsc})
	if m.simulator == nil {
		t.Fatal("running simulation Esc should cancel, not close")
	}
	m.simulator.done = true
	m = updateModel(t, m, tea.KeyMsg{Type: tea.KeyEsc})
	if m.simulator != nil {
		t.Fatal("finished simulation Esc should close")
	}
}

func TestTUICommandPaletteAndBaseKeys(t *testing.T) {
	m := NewModel(nil, "v1", utils.LanguageVI)
	m.width, m.height = 120, 36
	m.compItems = builtinCommandItems(utils.LanguageVI)
	m.compActive = true
	m.compIdx = 0
	for _, key := range []tea.KeyType{tea.KeyDown, tea.KeyUp, tea.KeyTab} {
		m = updateModel(t, m, tea.KeyMsg{Type: key})
	}
	m.compItems = []commandPaletteItem{{Name: "help", Usage: "/help", AutoExecute: true}}
	m.compActive = true
	m.compIdx = 0
	m = updateModel(t, m, tea.KeyMsg{Type: tea.KeyEnter})
	if m.help == nil {
		t.Fatal("palette Enter should execute help")
	}
	m = updateModel(t, m, tea.KeyMsg{Type: tea.KeyEsc})
	m.mode = modeNew
	m.startupMode = startupModeQuick
	m = updateModel(t, m, tea.KeyMsg{Type: tea.KeyTab})
	m = updateModel(t, m, tea.KeyMsg{Type: tea.KeyTab})
	m.textarea.SetValue("text")
	m = updateModel(t, m, tea.KeyMsg{Type: tea.KeyEscape})
	if m.textarea.Value() != "" {
		t.Fatal("Escape should clear new-mode input")
	}
	m = updateModel(t, m, tea.KeyMsg{Type: tea.KeyCtrlL})
	m = updateModel(t, m, tea.KeyMsg{Type: tea.KeyCtrlU})
	m = updateModel(t, m, tea.KeyMsg{Type: tea.KeyCtrlR})
	m = updateModel(t, m, tea.KeyMsg{Type: tea.KeyCtrlR})
	m = updateModel(t, m, tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'a'}})
	m = updateModel(t, m, tea.KeyMsg{Type: tea.KeyEnter, Alt: true})
	m = updateModel(t, m, tea.KeyMsg{Type: tea.KeyUp})
	m = updateModel(t, m, tea.KeyMsg{Type: tea.KeyDown})
	m = updateModel(t, m, tea.KeyMsg{Type: tea.KeyPgUp})
	m = updateModel(t, m, tea.KeyMsg{Type: tea.KeyPgDown})
	m = updateModel(t, m, tea.KeyMsg{Type: tea.KeyEnd})
}

func TestTUIImportAndSimulationStateRenderingBranches(t *testing.T) {
	at := time.Now()
	for _, lang := range []utils.Language{utils.LanguageVI, utils.LanguageZH} {
		impState := newImportState(1, "source.txt", 120, 36, nil, lang)
		impState.appendEvent(imp.Event{Time: at, Stage: imp.StageSegmenting, Current: 1, Total: 2, Message: "segment"}, 80)
		impState.appendEvent(imp.Event{Time: at, Stage: imp.StageAwaitingConfirmation, Total: 2, Message: "preview\nline"}, 80)
		impState.paused = true
		if renderImportModal(120, 36, impState, 1) == "" {
			t.Fatal("paused import modal should render")
		}
		impState.paused = false
		impState.appendEvent(imp.Event{Time: at, Stage: imp.StageError, Message: "error", Err: errors.New("bad")}, 80)
		if renderImportModal(120, 36, impState, 2) == "" {
			t.Fatal("failed import modal should render")
		}
		if formatElapsed(30*time.Second) != "00:30" || formatElapsed(61*time.Minute) != "1:01:00" {
			t.Fatal("elapsed formatting mismatch")
		}
		simState := newSimulationState(2, "sim", "source", 120, 36, nil, lang)
		simState.appendEvent(sim.Event{Time: at, Stage: sim.StageDone, Message: "done"}, 80)
		if renderSimulationModal(120, 36, simState) == "" {
			t.Fatal("done simulation modal should render")
		}
	}
}

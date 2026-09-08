package tui

import (
	"strings"
	"testing"

	"github.com/JustinNguyen9979/ainovel-cli/internal/host"
	"github.com/JustinNguyen9979/ainovel-cli/internal/utils"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/x/ansi"
)

func TestTUIInputAndPaneBranches(t *testing.T) {
	m := NewModel(nil, "v1", utils.LanguageVI)
	m.width, m.height = 120, 36
	m.updateViewportSize()
	for _, point := range [][2]int{{1, 1}, {1, 10}, {50, 10}, {119, 10}, {50, 35}} {
		_, _ = m.paneAtMouse(point[0], point[1])
	}
	zero := Model{}
	if _, ok := zero.paneAtMouse(0, 0); ok {
		t.Fatal("zero-size model should not have a mouse pane")
	}
	m.focusPane = focusStream
	m = updateModel(t, m, tea.MouseMsg{X: 60, Y: 15, Action: tea.MouseActionPress})
	m.focusPane = focusDetail
	m = updateModel(t, m, tea.MouseMsg{X: 110, Y: 15, Action: tea.MouseActionPress})
	m.focusPane = focusState
	m = updateModel(t, m, tea.MouseMsg{X: 1, Y: 15, Action: tea.MouseActionPress})
	m.focusPane = focusEvents
	m = updateModel(t, m, tea.MouseMsg{X: 60, Y: 15, Action: tea.MouseActionPress})
	m = updateModel(t, m, tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'x'}})
	m = updateModel(t, m, tea.KeyMsg{Type: tea.KeyBackspace})
	m = updateModel(t, m, tea.KeyMsg{Type: tea.KeyCtrlU})
	if _, cmd := m.handleTextareaMsg(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'a'}}); cmd == nil {
		t.Fatal("textarea update should return a command")
	}
}

func TestTUIEnterKeyModeBranches(t *testing.T) {
	m := NewModel(nil, "v1", utils.LanguageVI)
	m.textarea.SetValue("write a story")
	next, cmd := m.handleEnterKey()
	m = next.(Model)
	if cmd == nil || !m.starting || m.mode != modeRunning {
		t.Fatalf("quick start state = starting=%v mode=%d cmd=%v", m.starting, m.mode, cmd != nil)
	}

	m = NewModel(nil, "v1", utils.LanguageVI)
	m.startupMode = startupModeCoCreate
	m.textarea.SetValue("idea")
	next, cmd = m.handleEnterKey()
	m = next.(Model)
	if cmd == nil || m.cocreate == nil {
		t.Fatalf("cocreate state = %+v cmd=%v", m.cocreate, cmd != nil)
	}

	m = NewModel(nil, "v1", utils.LanguageVI)
	m.mode = modeRunning
	m.snapshot.IsRunning = false
	m.textarea.SetValue("continue")
	if _, cmd := m.handleEnterKey(); cmd == nil {
		t.Fatal("paused continuation should return command")
	}
	m.snapshot.IsRunning = true
	m.textarea.SetValue("steer")
	if _, cmd := m.handleEnterKey(); cmd == nil {
		t.Fatal("running steer should return command")
	}
	m.mode = modeDone
	m.textarea.SetValue("rewrite")
	next, cmd = m.handleEnterKey()
	m = next.(Model)
	if cmd == nil || m.mode != modeRunning {
		t.Fatal("done continuation should re-enter running mode")
	}
	m.textarea.SetValue("/unknown")
	if _, cmd := m.handleEnterKey(); cmd != nil {
		t.Fatal("unknown slash command should not return command")
	}
}

func TestTUIModalAndPaletteRenderingBranches(t *testing.T) {
	m := NewModel(nil, "v1", utils.LanguageVI)
	m.width, m.height = 120, 36
	m.help = newHelpState(120, 36, utils.LanguageVI)
	for _, key := range []tea.KeyType{tea.KeyUp, tea.KeyDown, tea.KeyPgUp, tea.KeyPgDown, tea.KeyRunes} {
		if _, _ = m.handleHelpKey(tea.KeyMsg{Type: key}); false {
			t.Fatal("unreachable")
		}
	}
	m.modelSwitch = &modelSwitchState{language: utils.LanguageVI}
	m.modelSwitch.moveFocus(1)
	m.modelSwitch.moveFocus(-1)
	nextModel, _ := m.handleModelSwitchKey(tea.KeyMsg{Type: tea.KeyEsc})
	m = nextModel.(Model)
	if m.modelSwitch != nil {
		t.Fatal("model switch Esc should close")
	}
	items := []commandPaletteItem{{Name: "one", Usage: "/one", Description: "first"}, {Name: "two", Usage: "/two", Description: "second"}}
	if got := renderCommandPalette(60, items, 1, utils.LanguageVI); got == "" || !strings.Contains(ansi.Strip(got), "two") {
		t.Fatal("selected command palette missing item")
	}
	if got := renderAgentBlock("✻ writer\nwrite", 40); got == "" || renderAgentBlock("legacy", 40) == "" {
		t.Fatal("agent block rendering failed")
	}
	if got := renderDispatchSummary("writer（写作）", 40); got == "" || renderDispatchSummary("writer", 40) == "" {
		t.Fatal("dispatch summary rendering failed")
	}
	if localizedInputLimit(utils.LanguageVI, 80, 100) == "" || localizedInputLimit(utils.LanguageZH, 80, 100) == "" {
		t.Fatal("input limit labels missing")
	}
	for _, status := range []string{"READY", "RUNNING", "REVIEW", "REWRITE", "COMPLETE", "PAUSED", "PAUSING", "ERROR", "OTHER"} {
		if localizedStatusLabel(utils.LanguageVI, status) == "" || localizedStatusLabel(utils.LanguageZH, status) == "" {
			t.Fatalf("status label %q missing", status)
		}
	}
	if localizedLanguageLabel(utils.LanguageVI, utils.LanguageZH) == "" || localizedLanguageLabel(utils.LanguageZH, utils.LanguageVI) == "" {
		t.Fatal("language labels missing")
	}
	_ = host.StreamClearSentinel
}

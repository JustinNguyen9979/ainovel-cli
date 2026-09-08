package bootstrap

import (
	"strings"
	"testing"

	"github.com/JustinNguyen9979/ainovel-cli/internal/utils"
	tea "github.com/charmbracelet/bubbletea"
)

func TestSetupPureHelpers(t *testing.T) {
	if setupText(utils.LanguageZH, "中文", "Tiếng Việt") != "中文" || setupText(utils.LanguageVI, "中文", "Tiếng Việt") != "Tiếng Việt" {
		t.Fatal("setup language selection is wrong")
	}
	if maskKey("short") != "****" || maskKey("sk-123456789") != "sk-1****6789" {
		t.Fatal("API key masking is wrong")
	}
	presets := ProviderPresets()
	if len(presets) == 0 || presets[0].Name != "openrouter" {
		t.Fatalf("provider presets = %+v", presets)
	}
	presets[0].Name = "changed"
	if ProviderPresets()[0].Name == "changed" {
		t.Fatal("provider presets should be copied")
	}
	if !strings.Contains(localizedSetupProviders(utils.LanguageVI)[len(presets)-1].label, "Proxy") {
		t.Fatal("custom provider should be localized")
	}
}

func TestSetupSelectModelUpdateAndView(t *testing.T) {
	m := setupSelectModel{language: utils.LanguageVI, title: "Title", items: []setupProvider{{label: "One"}, {label: "Two"}}}
	update := func(key tea.KeyMsg) {
		model, _ := m.Update(key)
		m = model.(setupSelectModel)
	}
	update(tea.KeyMsg{Type: tea.KeyDown})
	if m.cursor != 1 || !strings.Contains(m.View(), "Two") {
		t.Fatal("down selection failed")
	}
	update(tea.KeyMsg{Type: tea.KeyUp})
	if m.cursor != 0 {
		t.Fatal("up selection failed")
	}
	update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'j'}})
	if m.cursor != 1 {
		t.Fatal("vim selection failed")
	}
	model, cmd := m.Update(tea.KeyMsg{Type: tea.KeyEnter})
	if cmd == nil || model.(setupSelectModel).cursor != 1 {
		t.Fatal("enter selection failed")
	}
	m = model.(setupSelectModel)
	update(tea.KeyMsg{Type: tea.KeyEscape})
	if !m.cancelled {
		t.Fatal("escape should cancel selection")
	}
}

func TestSetupInputModelUpdateAndView(t *testing.T) {
	m := setupInputModel{language: utils.LanguageVI, label: "Label", placeholder: "Hint", defaultValue: "default"}
	if !strings.Contains(m.View(), "Hint") {
		t.Fatal("empty input view missing placeholder")
	}
	update := func(key tea.KeyMsg) {
		model, _ := m.Update(key)
		m = model.(setupInputModel)
	}
	update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'a', 'b'}})
	update(tea.KeyMsg{Type: tea.KeySpace})
	update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'c'}})
	if m.value != "ab c" || !strings.Contains(m.View(), "ab c") {
		t.Fatal("input typing failed")
	}
	update(tea.KeyMsg{Type: tea.KeyBackspace})
	if m.value != "ab " {
		t.Fatal("backspace failed")
	}
	model, cmd := m.Update(tea.KeyMsg{Type: tea.KeyEnter})
	if cmd == nil || model.(setupInputModel).cancelled {
		t.Fatal("enter should accept input")
	}
	cancel := setupInputModel{allowEmpty: false}
	model, cmd = cancel.Update(tea.KeyMsg{Type: tea.KeyEnter})
	if cmd != nil || model.(setupInputModel).cancelled {
		t.Fatal("empty input should remain open")
	}
	cancel = model.(setupInputModel)
	model, _ = cancel.Update(tea.KeyMsg{Type: tea.KeyEscape})
	if !model.(setupInputModel).cancelled {
		t.Fatal("escape should cancel input")
	}
}

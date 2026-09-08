package tui

import (
	"strings"
	"testing"

	"github.com/JustinNguyen9979/ainovel-cli/internal/host"
	"github.com/JustinNguyen9979/ainovel-cli/internal/utils"
	"github.com/charmbracelet/x/ansi"
)

func TestCommandPaletteSelectionAndRendering(t *testing.T) {
	m := NewModel(nil, "v1", utils.LanguageVI)
	m.textarea.SetValue("/mod")
	m.updateCommandPalette()
	if !m.compActive || len(m.compItems) == 0 {
		t.Fatalf("completion state = active=%v items=%v token=%q", m.compActive, m.compItems, m.commandToken)
	}
	if _, ok := m.selectedCommandItem(); !ok {
		t.Fatal("selected item missing")
	}
	item, ok := m.acceptCommandCompletion()
	if !ok || item.Name != "model" || !strings.HasPrefix(m.textarea.Value(), "/model") || m.compActive {
		t.Fatalf("accepted completion = %+v state=%+v", item, m)
	}
	if _, ok := m.selectedCommandItem(); ok {
		t.Fatal("selection should clear after acceptance")
	}
	m.textarea.Reset()
	m.updateCommandPalette()
	if m.compActive {
		t.Fatal("empty input should clear palette")
	}
	m.textarea.SetValue("/unknown")
	m.updateCommandPalette()
	if m.compActive {
		t.Fatal("unknown command should have no completion")
	}
	items := builtinCommandItems(utils.LanguageVI)
	if got := renderCommandPalette(120, items, 0, utils.LanguageVI); got == "" || !strings.Contains(ansi.Strip(got), "Lệnh") {
		t.Fatalf("palette = %q", ansi.Strip(got))
	}
	if renderCommandPalette(0, items, 0) != "" || renderCommandPalette(120, nil, 0) != "" {
		t.Fatal("empty palette should render empty")
	}
}

func TestMarkdownAndCoCreateStateHelpers(t *testing.T) {
	state := newStageCoCreateState(utils.LanguageVI)
	if !state.stage || state.initialInput() == "" || placeholderForCoCreate(state) == "" {
		t.Fatal("stage cocreate state not initialized")
	}
	state.appendUser("hello")
	state.applyDelta("thinking", " think ")
	state.applyDelta("reply", " reply ")
	if state.streamReply() != "reply" || state.session.StreamThinking() != "think" {
		t.Fatal("delta state mismatch")
	}
	state.apply(host.CoCreateReply{Message: "message", Prompt: "## Draft", Ready: true, Suggestions: []string{"one"}})
	if !state.canStart() || !state.ready() || state.draftPrompt() != "## Draft" || len(state.suggestions()) != 1 {
		t.Fatal("reply state mismatch")
	}
	if got, err := state.buildPrompt(); err != nil || got != "## Draft" {
		t.Fatalf("build prompt = %q/%v", got, err)
	}
	if got := renderMarkdownPreview("# H1\n## H2\n### H3\n- bullet\n1. ordered\n> quote\nplain", 20); !strings.Contains(ansi.Strip(got), "H1") || !strings.Contains(ansi.Strip(got), "bullet") {
		t.Fatalf("markdown preview = %q", ansi.Strip(got))
	}
	if !isOrderedMarkdownItem("12. item") || isOrderedMarkdownItem("item") {
		t.Fatal("ordered markdown detection mismatch")
	}
	if prefix, body := splitOrderedMarkdownItem("12. item"); prefix != "12." || body != "item" {
		t.Fatalf("ordered split = %q/%q", prefix, body)
	}
}

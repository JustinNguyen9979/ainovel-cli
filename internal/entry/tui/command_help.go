package tui

import (
	"strings"

	"github.com/JustinNguyen9979/ainovel-cli/internal/utils"
	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

type helpState struct {
	viewport viewport.Model
	language utils.Language
}

func newHelpState(width, height int, languages ...utils.Language) *helpState {
	boxW, boxH := reportModalSize(width, height)
	contentW := paddedModalContentWidth(boxW)
	text := renderHelpText(contentW, languages...)

	vp := viewport.New(contentW, boxH-4)
	vp.SetContent(text)
	return &helpState{viewport: vp, language: resolveLanguage(languages)}
}

func renderHelpText(width int, languages ...utils.Language) string {
	titleStyle := lipgloss.NewStyle().Foreground(colorAccent).Bold(true)
	nameStyle := lipgloss.NewStyle().Foreground(colorAccent2).Bold(true)
	usageStyle := lipgloss.NewStyle().Foreground(colorMuted)
	descStyle := lipgloss.NewStyle().Foreground(bodyTextColor)
	hintStyle := lipgloss.NewStyle().Foreground(colorDim)

	var b strings.Builder
	b.WriteString(titleStyle.Render(tr(languages, utils.MsgCommandHelpTitle)))
	b.WriteString("\n\n")

	for i, spec := range commandSpecs(languages...) {
		if i > 0 {
			b.WriteString("\n")
		}
		b.WriteString(nameStyle.Render("/" + spec.Name))
		if len(spec.Aliases) > 0 {
			b.WriteString(usageStyle.Render("  " + ui(resolveLanguage(languages), "别名: /", "bí danh: /") + strings.Join(spec.Aliases, " /")))
		}
		b.WriteString("\n")
		b.WriteString(usageStyle.Render(ui(resolveLanguage(languages), "用法: ", "Cách dùng: ") + spec.Usage))
		b.WriteString("\n")
		b.WriteString(descStyle.Render(wrapText(spec.Description, width)))
		b.WriteString("\n")
	}

	b.WriteString("\n")
	b.WriteString(titleStyle.Render(tr(languages, utils.MsgShortcuts)))
	b.WriteString("\n\n")
	lines := []string{
		tr(languages, utils.MsgCommandSearchHint),
		"↑↓ " + tr(languages, utils.MsgScroll),
		tr(languages, utils.MsgCommandAcceptHint),
		tr(languages, utils.MsgCommandCloseHint),
		ui(resolveLanguage(languages), "Ctrl+R 切换选中复制模式（关闭鼠标上报后可拖拽复制，再按一次恢复）", "Ctrl+R bật/tắt chế độ chọn để sao chép (tắt báo chuột để kéo chọn, nhấn lần nữa để khôi phục)"),
	}
	for _, line := range lines {
		b.WriteString(hintStyle.Render(line))
		b.WriteString("\n")
	}
	return b.String()
}

func renderHelpModal(width, height int, state *helpState) string {
	if state == nil {
		return ""
	}

	boxW, boxH := reportModalSize(width, height)
	contentW := paddedModalContentWidth(boxW)

	if state.viewport.Width != contentW {
		state.viewport.Width = contentW
	}
	if state.viewport.Height != boxH-4 {
		state.viewport.Height = boxH - 4
	}

	modal := renderPaddedModalFrame(
		boxW,
		boxH,
		tr([]utils.Language{state.language}, utils.MsgCommandHelpTitle),
		"  ↑↓ "+tr([]utils.Language{state.language}, utils.MsgScroll)+" · "+tr([]utils.Language{state.language}, utils.MsgClose),
		strings.Split(state.viewport.View(), "\n"),
	)
	return lipgloss.Place(width, height, lipgloss.Center, lipgloss.Center, modal)
}

func (m Model) handleHelpKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	if m.help == nil {
		return m, nil
	}
	switch msg.Type {
	case tea.KeyEsc:
		m.help = nil
		return m, m.textarea.Focus()
	case tea.KeyUp:
		m.help.viewport.ScrollUp(1)
		return m, nil
	case tea.KeyDown:
		m.help.viewport.ScrollDown(1)
		return m, nil
	case tea.KeyPgUp:
		m.help.viewport.HalfPageUp()
		return m, nil
	case tea.KeyPgDown:
		m.help.viewport.HalfPageDown()
		return m, nil
	default:
		return m, nil
	}
}

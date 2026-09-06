package tui

import (
	"strings"

	"github.com/JustinNguyen9979/ainovel-cli/internal/host"
	"github.com/JustinNguyen9979/ainovel-cli/internal/utils"
	"github.com/charmbracelet/bubbles/viewport"
	"github.com/charmbracelet/lipgloss"
)

// renderTopBar 渲染顶部状态栏。
// 左侧：provider/model，中间：书名，右侧：状态胶囊。
func renderTopBar(snap host.UISnapshot, width int, spinnerFrame, version string, languages ...utils.Language) string {
	bookTitle := snap.BookTitle
	if bookTitle == "" {
		bookTitle = tr(languages, utils.MsgNoBookTitle)
	}

	var infoParts []string
	if version != "" {
		infoParts = append(infoParts, "ainovel-cli "+version)
	}
	if snap.Provider != "" {
		infoParts = append(infoParts, snap.Provider)
	}
	if snap.ModelName != "" {
		if w := formatContextWindow(snap.ModelContextWindow); w != "" {
			infoParts = append(infoParts, snap.ModelName+"("+w+")")
		} else {
			infoParts = append(infoParts, snap.ModelName)
		}
	}
	if snap.Style != "" && snap.Style != "default" {
		infoParts = append(infoParts, snap.Style)
	}
	leftText := strings.Join(infoParts, " · ")

	lang := resolveLanguage(languages)
	label := snap.StatusLabel
	if label == "" {
		label = "READY"
	}
	color, ok := statusColors[label]
	if !ok {
		color = colorDim
	}
	disp, ok := statusDisplay[label]
	if !ok {
		disp = struct {
			icon  string
			label string
		}{"○", strings.ToLower(label)}
	}
	disp.label = localizedStatusLabel(lang, label)
	icon := disp.icon
	if snap.IsRunning && spinnerFrame != "" {
		icon = spinnerFrame
	}
	var status string
	if icon != "" {
		status = statusIconStyle.Foreground(color).Render(icon) + " " + statusLabelStyle.Render(disp.label)
	} else {
		status = statusLabelStyle.Render(disp.label)
	}

	innerW := max(12, width-2)
	titleText := truncate(bookTitle, max(8, innerW/3))
	centerW := max(16, lipgloss.Width(titleText)+6)
	if centerW > innerW-24 {
		centerW = max(8, innerW-24)
	}
	sideTotal := innerW - centerW
	if sideTotal < 0 {
		sideTotal = 0
		centerW = innerW
	}
	leftW := sideTotal / 2
	rightW := innerW - centerW - leftW

	leftCell := lipgloss.NewStyle().
		Width(leftW).
		AlignHorizontal(lipgloss.Left).
		Foreground(colorDim).
		Render(truncate(leftText, leftW))
	centerCell := lipgloss.NewStyle().
		Width(centerW).
		AlignHorizontal(lipgloss.Center).
		Bold(true).
		Foreground(bodyTextColor).
		Render(titleText)
	rightCell := lipgloss.NewStyle().
		Width(rightW).
		AlignHorizontal(lipgloss.Right).
		Render(status)

	content := leftCell + centerCell + rightCell
	return topBarStyle.Width(width).
		Border(baseBorder, false, false, true, false).
		BorderForeground(colorDim).
		Render(content)
}

// renderStatePanel 把状态侧栏内容(已在 stateVP 中)包进左侧带右边框的盒子。
// 与 renderDetailPanel 对称：内容由 renderStateContent 生成并喂进 viewport，这里只负责框。
// MaxHeight 钳高，防止窗口缩小时溢出比右栏高（见 panels_test.go 的高度契约）。
func renderStatePanel(vp viewport.Model, width, height int, focused bool) string {
	borderColor := colorDim
	if focused {
		borderColor = colorAccent
	}
	style := lipgloss.NewStyle().
		Width(width).
		Height(height).
		MaxHeight(height).
		Border(baseBorder, false, true, false, false).
		BorderForeground(borderColor).
		Padding(1, 1, 0, 1)
	return style.Render(vp.View())
}

func max(a, b int) int {
	if a > b {
		return a
	}
	return b
}

// renderDetailPanel 渲染右侧可滚动详情面板。
func renderDetailPanel(vp viewport.Model, width, height int, focused bool) string {
	borderColor := colorDim
	if focused {
		borderColor = colorAccent
	}
	style := lipgloss.NewStyle().
		Width(width).
		Height(height).
		MaxHeight(height).
		Border(baseBorder, false, false, false, true).
		BorderForeground(borderColor).
		Padding(0, 1)

	return style.Render(vp.View())
}

// renderWelcome 渲染新建态首屏。
func renderWelcome(width, height int, errMsg string, mode startupMode, importHint, updateHint string, languages ...utils.Language) string {
	// 简洁标题
	title := lipgloss.NewStyle().
		Foreground(colorAccent).
		Bold(true).
		Render("A I N O V E L")

	lang := resolveLanguage(languages)

	// 副标题
	subtitle := lipgloss.NewStyle().
		Foreground(colorMuted).
		Italic(true).
		Render(ui(lang, "AI 驱动的小说创作引擎", "Công cụ sáng tác tiểu thuyết với AI"))

	// 分隔线
	divW := 44
	if divW > width-8 {
		divW = width - 8
	}
	divider := lipgloss.NewStyle().Foreground(colorDim).
		Render(strings.Repeat("~", divW))

	// 功能亮点
	features := []struct{ icon, label, desc string }{
		{">>", "Phối hợp nhiều model", "Architect lập kế hoạch / Writer sáng tác / Editor đánh giá"},
		{"::", "Khôi phục điểm dừng", "Tự tiếp tục từ tiến độ trước sau khi lỗi hoặc gián đoạn"},
		{"<>", "Can thiệp thời gian thực", "Điều chỉnh hướng cốt truyện bất cứ lúc nào"},
		{"##", "Truyện dài phân tầng", "Hỗ trợ cấu trúc tập - cung - chương"},
	}
	if lang == utils.LanguageZH {
		features = []struct{ icon, label, desc string }{
			{">>", "多模型协作", "Architect 规划 / Writer 创作 / Editor 评审"},
			{"::", "断点恢复", "出错或中断后从原进度自动继续"},
			{"<>", "实时干预", "随时调整故事发展方向"},
			{"##", "分层长篇", "支持卷 - 弧 - 章节结构"},
		}
	}
	iconStyle := lipgloss.NewStyle().Foreground(colorAccent2).Bold(true)
	featLabelStyle := lipgloss.NewStyle().Foreground(bodyTextColor)
	descStyle := lipgloss.NewStyle().Foreground(colorDim)
	var featLines []string
	for _, f := range features {
		line := iconStyle.Render(f.icon) + " " +
			featLabelStyle.Render(f.label) + "  " +
			descStyle.Render(f.desc)
		featLines = append(featLines, line)
	}
	feats := strings.Join(featLines, "\n")

	// 输入提示
	promptText := "Nhập yêu cầu tiểu thuyết bên dưới để bắt đầu sáng tác"
	if lang == utils.LanguageZH {
		promptText = "在下方输入小说需求，开始创作"
	}
	prompt := lipgloss.NewStyle().Foreground(bodyTextColor).Render(promptText)

	modeLine := lipgloss.NewStyle().
		Foreground(colorMuted).
		Render(localizedModeLine(mode, lang))

	// 示例
	examples := []string{
		"Viết truyện trinh thám đô thị 12 chương, nhân vật chính là nữ pháp y",
		"Sáng tác truyện tiên hiệp dài, nhân vật chính tu luyện từ phàm nhân đến phi thăng",
		"Viết truyện khoa học viễn tưởng ngắn về khủng hoảng đạo đức sau khi AI thức tỉnh",
	}
	if lang == utils.LanguageZH {
		examples = []string{
			"写一部12章的都市推理小说，主角是一名女法医",
			"创作一部长篇仙侠小说，主角从凡人一路修炼飞升",
			"写一部关于 AI 觉醒后道德危机的短篇科幻小说",
		}
	}
	exStyle := lipgloss.NewStyle().Foreground(colorAccent)
	dotStyle := lipgloss.NewStyle().Foreground(colorDim)
	var exLines []string
	for _, ex := range examples {
		exLines = append(exLines, dotStyle.Render("  . ")+exStyle.Render(ex))
	}
	exBlock := strings.Join(exLines, "\n")

	// 组装
	var b strings.Builder
	b.WriteString("\n")
	b.WriteString(title)
	b.WriteString("\n")
	b.WriteString(subtitle)
	b.WriteString("\n\n")
	b.WriteString(divider)
	b.WriteString("\n\n")
	b.WriteString(feats)
	b.WriteString("\n\n")
	b.WriteString(divider)
	b.WriteString("\n\n")
	b.WriteString(modeLine)
	b.WriteString("\n\n")
	b.WriteString(prompt)
	b.WriteString("\n\n")
	b.WriteString(exBlock)
	b.WriteString("\n\n")
	if importHint != "" {
		// 这本书停在导入半路：显著提示恢复入口，替代常规导入提示。
		b.WriteString(lipgloss.NewStyle().Foreground(colorAccent2).Bold(true).
			Render("! " + importHint))
	} else if updateHint != "" {
		b.WriteString(lipgloss.NewStyle().Foreground(colorAccent2).Bold(true).
			Render("! " + updateHint))
	} else {
		b.WriteString(lipgloss.NewStyle().Foreground(colorDim).
			Render(func() string {
				if lang == utils.LanguageZH {
					return "已有设定/大纲？/start <路径> 创建新书 · 已有草稿？/import <路径> 导入续写"
				}
				return "Đã có thiết lập/đại cương? /start <đường dẫn> tạo sách mới · Đã có bản thảo? /import <đường dẫn> nhập để viết tiếp"
			}()))
	}
	b.WriteString("\n\n")
	b.WriteString(lipgloss.NewStyle().Foreground(colorDim).Italic(true).
		Render(func() string {
			if lang == utils.LanguageZH {
				return "Tab 切换模式 · 快速开始按 Enter 创作 · 共创规划按 Enter 对话"
			}
			return "Tab đổi chế độ · Bắt đầu nhanh nhấn Enter để sáng tác · Đồng sáng tác nhấn Enter để đối thoại"
		}()))

	if errMsg != "" {
		b.WriteString("\n\n")
		b.WriteString(lipgloss.NewStyle().Foreground(colorError).Bold(true).Render("! " + errMsg))
	}

	return lipgloss.NewStyle().
		Width(width).
		Height(height).
		AlignHorizontal(lipgloss.Center).
		AlignVertical(lipgloss.Center).
		Render(b.String())
}

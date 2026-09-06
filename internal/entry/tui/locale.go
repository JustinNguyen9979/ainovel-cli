package tui

import (
	"fmt"
	"strings"

	"github.com/JustinNguyen9979/ainovel-cli/internal/utils"
)

func resolveLanguage(languages []utils.Language) utils.Language {
	if len(languages) > 0 && languages[0] == utils.LanguageZH {
		return utils.LanguageZH
	}
	return utils.LanguageVI
}

func tr(languages []utils.Language, key utils.MessageKey, args ...any) string {
	return utils.T(resolveLanguage(languages), key, args...)
}

func ui(lang utils.Language, zh, vi string) string {
	if resolveLanguage([]utils.Language{lang}) == utils.LanguageZH {
		return zh
	}
	return vi
}

func localizedNewMode(mode startupMode, lang utils.Language) (label, subtitle, placeholder string) {
	if lang == utils.LanguageZH {
		if mode == startupModeCoCreate {
			return "共创规划", "先与 AI 对话澄清，再开始创作", "先输入你的核心想法，Enter 开始与 AI 共创"
		}
		return "快速开始", "一句话直接开始写", "输入一句小说需求，Enter 直接开始创作"
	}
	if mode == startupModeCoCreate {
		return "Cùng lập kế hoạch", "Trao đổi với AI để làm rõ rồi mới sáng tác", "Nhập ý tưởng cốt lõi, Enter để cùng AI lập kế hoạch"
	}
	return "Bắt đầu nhanh", "Một câu là bắt đầu viết ngay", "Nhập một câu yêu cầu tiểu thuyết, Enter để bắt đầu"
}

func localizedModePlaceholder(mode startupMode, lang utils.Language) string {
	_, _, placeholder := localizedNewMode(mode, lang)
	return placeholder
}

func localizedCoCreatePlaceholder(state *cocreateState, lang utils.Language) string {
	if state == nil {
		return localizedModePlaceholder(startupModeCoCreate, lang)
	}
	if state.awaiting {
		return utils.T(lang, utils.MsgAIThinking)
	}
	if state.canStart() {
		if state.stage {
			if lang == utils.LanguageZH {
				return "继续补充，或按 Ctrl+S 应用方向并继续创作"
			}
			return "Tiếp tục bổ sung, hoặc nhấn Ctrl+S để áp dụng hướng và tiếp tục sáng tác"
		}
		if lang == utils.LanguageZH {
			return "继续补充，或按 Ctrl+S 开始创作"
		}
		return "Tiếp tục bổ sung, hoặc nhấn Ctrl+S để bắt đầu sáng tác"
	}
	if lang == utils.LanguageZH {
		return "继续补充你的要求，Enter 发送给 AI"
	}
	return "Tiếp tục bổ sung yêu cầu, Enter để gửi cho AI"
}

func localizedModeLine(mode startupMode, lang utils.Language) string {
	label, subtitle, _ := localizedNewMode(mode, lang)
	if lang == utils.LanguageZH {
		return "当前模式：" + label + " · " + subtitle
	}
	return "Chế độ hiện tại: " + label + " · " + subtitle
}

func localizedCompletionHint(lang utils.Language) string {
	if lang == utils.LanguageZH {
		return "再次按 Ctrl+C 退出"
	}
	return "Nhấn Ctrl+C lần nữa để thoát"
}

func localizedCopyHint(lang utils.Language) string {
	if lang == utils.LanguageZH {
		return " · Ctrl+R 切到选中复制模式"
	}
	return " · Ctrl+R chuyển sang chế độ chọn để sao chép"
}

func localizedInputLimit(lang utils.Language, used, limit int) string {
	if lang == utils.LanguageZH {
		return fmt.Sprintf(" · 输入 %d/%d", used, limit)
	}
	return fmt.Sprintf(" · Nhập %d/%d", used, limit)
}

func localizedStatusLabel(lang utils.Language, status string) string {
	if lang == utils.LanguageZH {
		switch status {
		case "READY":
			return "就绪"
		case "RUNNING":
			return "运行中"
		case "REVIEW":
			return "评审"
		case "REWRITE":
			return "重写"
		case "COMPLETE":
			return "完成"
		case "PAUSED":
			return "暂停"
		case "PAUSING":
			return "暂停中"
		case "ERROR":
			return "错误"
		}
	}
	switch status {
	case "READY":
		return "Sẵn sàng"
	case "RUNNING":
		return "Đang chạy"
	case "REVIEW":
		return "Đánh giá"
	case "REWRITE":
		return "Viết lại"
	case "COMPLETE":
		return "Hoàn tất"
	case "PAUSED":
		return "Tạm dừng"
	case "PAUSING":
		return "Đang tạm dừng"
	case "ERROR":
		return "Lỗi"
	default:
		return strings.ToLower(status)
	}
}

func localizedLanguageLabel(lang, value utils.Language) string {
	if lang == utils.LanguageZH {
		if value == utils.LanguageZH {
			return "中文"
		}
		return "Tiếng Việt"
	}
	if value == utils.LanguageZH {
		return "Tiếng Trung"
	}
	return "Tiếng Việt"
}

func localizedStateLabel(lang utils.Language, state string) string {
	if lang == utils.LanguageZH {
		switch state {
		case "running":
			return "运行中"
		case "pausing":
			return "暂停中"
		case "paused":
			return "已暂停"
		case "completed":
			return "已完成"
		default:
			return "空闲"
		}
	}
	switch state {
	case "running":
		return "Đang chạy"
	case "pausing":
		return "Đang tạm dừng"
	case "paused":
		return "Đã tạm dừng"
	case "completed":
		return "Đã hoàn tất"
	default:
		return "Đang rảnh"
	}
}

func localizedPhaseLabel(lang utils.Language, phase string) string {
	if lang == utils.LanguageZH {
		switch phase {
		case "premise":
			return "前提"
		case "outline":
			return "大纲"
		case "writing":
			return "写作"
		case "complete":
			return "完成"
		case "init":
			return "初始化"
		}
	} else {
		switch phase {
		case "premise":
			return "Tiền đề"
		case "outline":
			return "Đại cương"
		case "writing":
			return "Sáng tác"
		case "complete":
			return "Hoàn tất"
		case "init":
			return "Khởi tạo"
		}
	}
	if phase == "" {
		return "-"
	}
	return phase
}

func localizedFlowLabel(lang utils.Language, flow string) string {
	if lang == utils.LanguageZH {
		switch flow {
		case "writing":
			return "写作"
		case "reviewing":
			return "评审"
		case "rewriting":
			return "重写"
		case "polishing":
			return "打磨"
		case "steering":
			return "干预"
		}
	} else {
		switch flow {
		case "writing":
			return "Sáng tác"
		case "reviewing":
			return "Đánh giá"
		case "rewriting":
			return "Viết lại"
		case "polishing":
			return "Đánh bóng"
		case "steering":
			return "Can thiệp"
		}
	}
	if flow == "" {
		return "-"
	}
	return flow
}

func localizedAgentState(lang utils.Language, state string) string {
	if lang == utils.LanguageZH {
		switch state {
		case "running":
			return "运行中"
		case "failed":
			return "异常"
		case "idle":
			return "待命"
		}
	} else {
		switch state {
		case "running":
			return "Đang chạy"
		case "failed":
			return "Lỗi"
		case "idle":
			return "Đang rảnh"
		}
	}
	return state
}

func localizedChapter(lang utils.Language, chapter int) string {
	if lang == utils.LanguageZH {
		return fmt.Sprintf("第 %d 章", chapter)
	}
	return fmt.Sprintf("Chương %d", chapter)
}

func localizedChapterCount(lang utils.Language, count int) string {
	if lang == utils.LanguageZH {
		return fmt.Sprintf("%d 章", count)
	}
	return fmt.Sprintf("%d chương", count)
}

func localizedChapterProgress(lang utils.Language, completed, total int) string {
	if lang == utils.LanguageZH {
		return fmt.Sprintf("%d / %d 章", completed, total)
	}
	return fmt.Sprintf("%d / %d chương", completed, total)
}

func localizedVolumeArc(lang utils.Language, value string) string {
	if lang == utils.LanguageZH || value == "" {
		return value
	}
	volume, arc, ok := strings.Cut(value, "·")
	if !ok || !strings.HasPrefix(volume, "第") || !strings.HasSuffix(volume, "卷") ||
		!strings.HasPrefix(arc, "第") || !strings.HasSuffix(arc, "弧") {
		return value
	}
	return "Tập " + strings.TrimSuffix(strings.TrimPrefix(volume, "第"), "卷") +
		" · Cung " + strings.TrimSuffix(strings.TrimPrefix(arc, "第"), "弧")
}

func localizedRecoveryLabel(lang utils.Language, value string) string {
	if lang == utils.LanguageZH || value == "" {
		return value
	}
	value = strings.ReplaceAll(value, "打磨恢复：", "Đánh bóng - khôi phục: ")
	value = strings.ReplaceAll(value, "重写恢复：", "Viết lại - khôi phục: ")
	value = strings.ReplaceAll(value, "恢复：规划阶段", "Khôi phục: giai đoạn lập kế hoạch")
	value = strings.ReplaceAll(value, "恢复：", "Khôi phục: ")
	value = strings.ReplaceAll(value, "提交中断", "gián đoạn commit")
	value = strings.ReplaceAll(value, "审阅中断", "gián đoạn đánh giá")
	value = strings.ReplaceAll(value, "进行中", "đang thực hiện")
	value = strings.ReplaceAll(value, "待处理", "chờ xử lý")
	value = strings.ReplaceAll(value, "从第", "từ chương ")
	value = strings.ReplaceAll(value, "章", " chương")
	return strings.ReplaceAll(value, "第", "")
}

func localizedPlanningTier(lang utils.Language, tier string) string {
	switch tier {
	case "short":
		return ui(lang, "短篇", "Ngắn")
	case "mid":
		return ui(lang, "中篇", "Trung bình")
	case "long":
		return ui(lang, "长篇", "Dài")
	default:
		return tier
	}
}

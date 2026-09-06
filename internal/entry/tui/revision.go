package tui

import (
	"context"
	"fmt"
	"strings"

	"github.com/JustinNguyen9979/ainovel-cli/internal/host"
	"github.com/JustinNguyen9979/ainovel-cli/internal/revision"
	"github.com/JustinNguyen9979/ainovel-cli/internal/utils"
	tea "github.com/charmbracelet/bubbletea"
)

type revisionDoneMsg struct {
	checkOnly bool
	chapters  []int
	result    *revision.Result
	err       error
}

func startRevisionSync(rt *host.Host, args []string) (tea.Cmd, bool, error) {
	checkOnly := false
	for _, arg := range args {
		switch arg {
		case "--check":
			checkOnly = true
		default:
			return nil, false, fmt.Errorf("未知参数 %q（支持：--check）", arg)
		}
	}
	return func() tea.Msg {
		if checkOnly {
			chapters, err := rt.CheckChapterRevisions()
			return revisionDoneMsg{checkOnly: true, chapters: chapters, err: err}
		}
		result, err := rt.SyncChapterRevisions(context.Background())
		return revisionDoneMsg{result: result, err: err}
	}, checkOnly, nil
}

func formatRevisionResult(result *revision.Result, languages ...utils.Language) string {
	lang := resolveLanguage(languages)
	if result == nil || len(result.Applied) == 0 {
		return ui(lang, "未检测到章节外部修改", "Không phát hiện chương bị sửa bên ngoài")
	}
	parts := make([]string, 0, len(result.Analyses))
	for i, analysis := range result.Analyses {
		if i >= len(result.Applied) {
			break
		}
		part := fmt.Sprintf(ui(lang, "第%d章：%s", "Chương %d: %s"), result.Applied[i], analysis.ChangeSummary)
		if analysis.StoryChanged {
			part += ui(lang, "（剧情事实已更新）", " (sự kiện truyện đã cập nhật)")
		}
		if len(analysis.DownstreamIssues) > 0 {
			part += fmt.Sprintf(ui(lang, "（发现%d项后续冲突）", " (%d xung đột phía sau)"), len(analysis.DownstreamIssues))
		}
		parts = append(parts, part)
	}
	summary := fmt.Sprintf(ui(lang, "已接纳章节修订：%v", "Đã tiếp nhận sửa đổi chương: %v"), result.Applied)
	if len(parts) > 0 {
		summary += ui(lang, "；", "; ") + strings.Join(parts, ui(lang, "；", "; "))
	}
	return summary
}

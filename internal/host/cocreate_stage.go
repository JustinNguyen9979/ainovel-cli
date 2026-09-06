package host

import (
	"fmt"
	"strings"

	"github.com/JustinNguyen9979/ainovel-cli/internal/store"
	"github.com/JustinNguyen9979/ainovel-cli/internal/utils"
)

// buildStoryStateSummary 组装一段精简的故事现状摘要，供阶段共创助手了解"已经写了什么"。
// 复用 store 访问点，只取规划方向所需的高层事实（进度 / 罗盘 / 最近卷 / 主要人物 / 活跃伏笔）；
// 不拉正文、不喂 novel_context 的全量 JSON——共创是对话，要的是可读概览，不是写作上下文。
// 任一项缺失都跳过（best-effort），返回空串表示尚无可用进度。
func buildStoryStateSummary(s *store.Store, languages ...utils.Language) string {
	if s == nil {
		return ""
	}
	lang := coCreateLanguage(languages)
	labels := newStorySummaryLabels(lang)
	var b strings.Builder
	var warnings []string
	warn := func(scope string, err error) {
		if err != nil {
			warnings = append(warnings, localizedStoryWarning(lang, scope, err))
		}
	}

	if book, err := s.Book.Load(); book != nil {
		fmt.Fprintf(&b, labels.book, book.Title)
	} else {
		warn("book", err)
	}

	if progress, err := s.Progress.Load(); progress != nil {
		fmt.Fprintf(&b, labels.progress, len(progress.CompletedChapters))
		if progress.Layered {
			outline, outlineErr := s.Outline.LoadOutline()
			if outlineErr != nil {
				warn("outline", outlineErr)
			} else if len(outline) > 0 {
				fmt.Fprintf(&b, labels.refined, len(outline))
			}
		} else if progress.TotalChapters > 0 {
			fmt.Fprintf(&b, labels.planned, progress.TotalChapters)
		}
		fmt.Fprintf(&b, labels.words, progress.TotalWordCount, progress.NextChapter())
		if progress.Layered && progress.CurrentVolume > 0 {
			fmt.Fprintf(&b, labels.nextChapter, progress.CurrentVolume, progress.CurrentArc)
		}
	} else {
		warn("progress", err)
	}

	if compass, err := s.Outline.LoadCompass(); compass != nil {
		if dir := strings.TrimSpace(compass.EndingDirection); dir != "" {
			fmt.Fprintf(&b, labels.ending, dir)
		}
		if compass.EstimatedScale != "" {
			fmt.Fprintf(&b, labels.scale, compass.EstimatedScale)
		}
		if len(compass.OpenThreads) > 0 {
			separator := "；"
			if lang == utils.LanguageVI {
				separator = "; "
			}
			fmt.Fprintf(&b, labels.threads, strings.Join(compass.OpenThreads, separator))
		}
	} else {
		warn("story_compass", err)
	}

	// 最近一卷摘要，让助手知道故事刚走到哪
	if vols, err := s.Summaries.LoadAllVolumeSummaries(); len(vols) > 0 {
		last := vols[len(vols)-1]
		fmt.Fprintf(&b, labels.recent, last.Title, truncate(last.Summary, 200))
	} else {
		warn("volume_summaries", err)
	}

	// 主要人物（core/important），最多 8 个
	if chars, err := s.Characters.Load(); len(chars) > 0 {
		var names []string
		for _, c := range chars {
			if c.Tier == "secondary" || c.Tier == "decorative" {
				continue
			}
			line := c.Name
			if role := strings.TrimSpace(c.Role); role != "" {
				if lang == utils.LanguageVI {
					line += " (" + role + ")"
				} else {
					line += "（" + role + "）"
				}
			}
			names = append(names, line)
			if len(names) >= 8 {
				break
			}
		}
		if len(names) > 0 {
			separator := "、"
			if lang == utils.LanguageVI {
				separator = ", "
			}
			fmt.Fprintf(&b, labels.characters, strings.Join(names, separator))
		}
	} else {
		warn("characters", err)
	}

	// 未收伏笔，最多 6 条
	if fs, err := s.World.LoadActiveForeshadow(); len(fs) > 0 {
		var items []string
		for _, f := range fs {
			items = append(items, truncate(f.Description, 40))
			if len(items) >= 6 {
				break
			}
		}
		separator := "；"
		if lang == utils.LanguageVI {
			separator = "; "
		}
		fmt.Fprintf(&b, labels.foreshadowing, strings.Join(items, separator))
	} else {
		warn("foreshadow", err)
	}

	if len(warnings) > 0 {
		separator := "；"
		if lang == utils.LanguageVI {
			separator = "; "
		}
		fmt.Fprintf(&b, labels.warning, strings.Join(warnings, separator))
	}

	return strings.TrimSpace(b.String())
}

// stageSystemPrompt 组装阶段共创的完整系统提示：阶段 prompt + 当前故事状态摘要。
// 摘要作为数据附录挂在末尾（用分隔线与格式规范隔开），呼应 prompt 里“进度见下方”的指引。
func stageSystemPrompt(s *store.Store, languages ...utils.Language) string {
	lang := coCreateLanguage(languages)
	prompt := stageCoCreateSystemPrompt(languages...)
	if summary := buildStoryStateSummary(s, lang); summary != "" {
		prompt += "\n\n---\n" + localizedStoryStateHeader(lang) + "\n" + summary
	}
	return prompt
}

func localizedStoryStateHeader(lang utils.Language) string {
	if lang == utils.LanguageVI {
		return "## Trạng thái câu chuyện hiện tại\n(Dưới đây là tóm tắt khách quan của nội dung đã viết, dùng để lập kế hoạch tiếp theo; không sao chép nguyên văn vào <draft>.)"
	}
	return "## 当前故事状态\n（以下是已写内容的客观摘要，供你规划后续时参照，不要在 <draft> 里照抄原文）"
}

func localizedStoryWarning(lang utils.Language, scope string, err error) string {
	if lang == utils.LanguageVI {
		return fmt.Sprintf("%s đọc thất bại: %v", scope, err)
	}
	return fmt.Sprintf("%s 读取失败: %v", scope, err)
}

type storySummaryLabels struct {
	book, progress, refined, planned, words, nextChapter, position     string
	ending, scale, threads, recent, characters, foreshadowing, warning string
}

func newStorySummaryLabels(lang utils.Language) storySummaryLabels {
	if lang == utils.LanguageVI {
		return storySummaryLabels{
			book: "- Tên sách: “%s”\n", progress: "- Tiến độ: đã hoàn thành %d chương", refined: " / đã triển khai chi tiết %d chương (các chương tiếp theo được lập kế hoạch động theo cung)", planned: " / kế hoạch %d chương", words: ", khoảng %d từ, chương tiếp theo là chương %d\n", nextChapter: "- Vị trí hiện tại: Tập %d, cung %d\n",
			ending: "- Hướng kết thúc: %s\n", scale: "- Quy mô ước tính: %s\n", threads: "- Tuyến truyện đang mở: %s\n", recent: "- Tập gần nhất “%s”: %s\n", characters: "- Nhân vật chính: %s\n", foreshadowing: "- Tình tiết chưa khép lại: %s\n", warning: "- Cảnh báo dữ liệu: %s\n",
		}
	}
	return storySummaryLabels{
		book: "- 书名：《%s》\n", progress: "- 进度：已完成 %d 章", refined: " / 当前已细化 %d 章（后续按弧动态规划）", planned: " / 规划 %d 章", words: "，约 %d 字，下一章为第 %d 章\n", nextChapter: "- 当前位置：第 %d 卷 第 %d 弧\n",
		ending: "- 终局方向：%s\n", scale: "- 预估规模：%s\n", threads: "- 活跃长线：%s\n", recent: "- 最近《%s》：%s\n", characters: "- 主要人物：%s\n", foreshadowing: "- 未收伏笔：%s\n", warning: "- 数据告警：%s\n",
	}
}

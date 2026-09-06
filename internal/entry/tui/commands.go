package tui

import (
	"fmt"
	"strings"
	"time"

	"github.com/JustinNguyen9979/ainovel-cli/internal/bootstrap"
	"github.com/JustinNguyen9979/ainovel-cli/internal/domain"
	"github.com/JustinNguyen9979/ainovel-cli/internal/entry/startup"
	"github.com/JustinNguyen9979/ainovel-cli/internal/host"
	"github.com/JustinNguyen9979/ainovel-cli/internal/utils"
	tea "github.com/charmbracelet/bubbletea"
)

type slashCommandSpec struct {
	Name        string
	Aliases     []string
	Group       string
	Usage       string
	Description string
	AutoExecute bool
	Hidden      bool
	NeedsIdle   bool
	Run         func(m Model, args []string) (tea.Model, tea.Cmd)
}

type slashCommand struct {
	name string
	args []string
}

func parseSlashCommand(text string) (slashCommand, bool) {
	text = strings.TrimSpace(text)
	if !strings.HasPrefix(text, "/") {
		return slashCommand{}, false
	}
	fields := strings.Fields(strings.TrimPrefix(text, "/"))
	if len(fields) == 0 {
		return slashCommand{}, false
	}
	return slashCommand{name: strings.ToLower(fields[0]), args: fields[1:]}, true
}

func (s slashCommandSpec) matches(name string) bool {
	if s.Name == name {
		return true
	}
	for _, alias := range s.Aliases {
		if strings.EqualFold(alias, name) {
			return true
		}
	}
	return false
}

func commandRegistryInstance(languages ...utils.Language) commandRegistry {
	lang := resolveLanguage(languages)
	commandDescription := func(key utils.MessageKey) string {
		return utils.T(lang, key)
	}
	return newCommandRegistry([]slashCommandSpec{
		{
			Name: "lang", Group: "system", Usage: "/lang vi|zh",
			Description: ui(lang, "切换界面语言并要求重启", "Đổi ngôn ngữ giao diện và yêu cầu khởi động lại"),
			Run: func(m Model, args []string) (tea.Model, tea.Cmd) {
				if len(args) != 1 {
					m.applyEvent(host.Event{Time: time.Now(), Category: "ERROR", Summary: ui(lang, "用法：/lang vi|zh", "Cách dùng: /lang vi|zh"), Level: "error"})
					m.refreshEventViewport()
					return m, nil
				}
				selected, err := utils.ParseLanguage(args[0])
				if err != nil {
					m.applyEvent(host.Event{Time: time.Now(), Category: "ERROR", Summary: err.Error(), Level: "error"})
					m.refreshEventViewport()
					return m, nil
				}
				path := bootstrap.EffectiveConfigPath()
				if path == "" {
					m.applyEvent(host.Event{Time: time.Now(), Category: "ERROR", Summary: ui(lang, "无法确定配置文件路径", "Không xác định được đường dẫn cấu hình"), Level: "error"})
					m.refreshEventViewport()
					return m, nil
				}
				cfg, err := bootstrap.LoadConfigFile(path)
				if err != nil {
					m.applyEvent(host.Event{Time: time.Now(), Category: "ERROR", Summary: err.Error(), Level: "error"})
					m.refreshEventViewport()
					return m, nil
				}
				cfg.Language = string(selected)
				if err := bootstrap.SaveConfig(path, cfg); err != nil {
					m.applyEvent(host.Event{Time: time.Now(), Category: "ERROR", Summary: err.Error(), Level: "error"})
					m.refreshEventViewport()
					return m, nil
				}
				m.applyEvent(host.Event{Time: time.Now(), Category: "SYSTEM", Summary: utils.T(lang, utils.MsgLanguageSavedRestart, localizedLanguageLabel(lang, selected)), Level: "info"})
				m.refreshEventViewport()
				return m, nil
			},
		},
		{
			Name:        "help",
			Group:       "system",
			Usage:       "/help",
			Description: utils.T(lang, utils.MsgHelpCommandDescription),
			AutoExecute: true,
			Run: func(m Model, _ []string) (tea.Model, tea.Cmd) {
				m.help = newHelpState(m.width, m.height, m.language)
				m.textarea.Blur()
				return m, nil
			},
		},
		{
			Name:        "model",
			Group:       "system",
			Usage:       "/model [role]",
			Description: utils.T(lang, utils.MsgModelCommandDescription),
			AutoExecute: true,
			Run: func(m Model, args []string) (tea.Model, tea.Cmd) {
				roleHint := ""
				if len(args) > 0 {
					roleHint = args[0]
					if normalizeRoleKey(roleHint) == "" {
						m.applyEvent(host.Event{
							Time: time.Now(), Category: "ERROR", Summary: utils.T(lang, utils.MsgUnknownRole, roleHint), Level: "error",
						})
						m.refreshEventViewport()
						return m, nil
					}
				}
				m.modelSwitch = newModelSwitchState(m.runtime, roleHint, m.language)
				m.textarea.Blur()
				return m, nil
			},
		},
		{
			Name:        "config",
			Group:       "system",
			Usage:       "/config",
			Description: utils.T(lang, utils.MsgConfigCommandDescription),
			AutoExecute: true,
			Run: func(m Model, args []string) (tea.Model, tea.Cmd) {
				if len(args) != 0 {
					m.applyEvent(host.Event{Time: time.Now(), Category: "ERROR", Summary: utils.T(lang, utils.MsgCommandUsage, "/config"), Level: "error"})
					m.refreshEventViewport()
					return m, nil
				}
				m.modelConfig = newModelConfigState(m.runtime, m.language)
				m.textarea.Blur()
				return m, nil
			},
		},
		{
			Name:        "diag",
			Group:       "analysis",
			Usage:       "/diag",
			Description: commandDescription(utils.MsgDiagCommandDescription),
			AutoExecute: true,
			Run: func(m Model, _ []string) (tea.Model, tea.Cmd) {
				m.reportSeq++
				m.report = newReportState(m.width, m.height, m.reportSeq, time.Now(), m.language)
				m.textarea.Blur()
				return m, loadReport(m.runtime.Dir(), m.reportSeq)
			},
		},
		{
			Name:        "review",
			Group:       "writing",
			Usage:       "/review on|off",
			Description: commandDescription(utils.MsgReviewCommandDescription),
			Run: func(m Model, args []string) (tea.Model, tea.Cmd) {
				if len(args) != 1 || (args[0] != "on" && args[0] != "off") {
					m.applyEvent(host.Event{Time: time.Now(), Category: "ERROR", Summary: utils.T(lang, utils.MsgCommandUsage, "/review on|off"), Level: "error"})
					m.refreshEventViewport()
					return m, nil
				}
				mode := domain.ChapterAdvanceReview
				if args[0] == "off" {
					mode = domain.ChapterAdvanceAuto
				}
				if err := m.runtime.SetAdvanceMode(mode); err != nil {
					m.applyEvent(host.Event{Time: time.Now(), Category: "ERROR", Summary: ui(lang, "切换推进模式失败：", "Chuyển chế độ tiến triển thất bại: ") + err.Error(), Level: "error"})
					m.refreshEventViewport()
					return m, nil
				}
				return m, fetchSnapshot(m.runtime)
			},
		},
		{
			Name:        "next",
			Group:       "writing",
			Usage:       "/next",
			Description: commandDescription(utils.MsgNextCommandDescription),
			AutoExecute: true,
			NeedsIdle:   true,
			Run: func(m Model, args []string) (tea.Model, tea.Cmd) {
				if len(args) != 0 {
					m.applyEvent(host.Event{Time: time.Now(), Category: "ERROR", Summary: utils.T(lang, utils.MsgCommandUsage, "/next"), Level: "error"})
					m.refreshEventViewport()
					return m, nil
				}
				if err := m.runtime.AdvanceOneChapter(); err != nil {
					m.applyEvent(host.Event{Time: time.Now(), Category: "ERROR", Summary: ui(lang, "放行下一章失败：", "Mở khóa chương tiếp theo thất bại: ") + err.Error(), Level: "error"})
					m.refreshEventViewport()
					return m, nil
				}
				return m, tea.Batch(fetchSnapshot(m.runtime), listenDone(m.runtime), m.textarea.Focus())
			},
		},
		{
			Name:        "start",
			Group:       "writing",
			Usage:       "/start <path>",
			Description: commandDescription(utils.MsgStartCommandDescription),
			Run: func(m Model, args []string) (tea.Model, tea.Cmd) {
				if m.mode != modeNew {
					m.applyEvent(host.Event{
						Time: time.Now(), Category: "ERROR", Summary: ui(lang, "/start 仅可在欢迎页创建新书", "/start chỉ dùng để tạo sách mới ở trang chào mừng"), Level: "error",
					})
					m.refreshEventViewport()
					return m, nil
				}
				prompt, err := prepareFileStart(args, m.language)
				if err != nil {
					m.err = err
					return m, nil
				}
				cmd := m.enterStarting(prompt)
				return m, tea.Batch(startRuntime(m.runtime, prompt), cmd)
			},
		},
		{
			Name:        "import",
			Group:       "writing",
			Usage:       ui(lang, "/import <path> [--yes] [--story=open|closed] [--continue] [--guide=<切分指导>]", "/import <đường dẫn> [--yes] [--story=open|closed] [--continue] [--guide=<hướng dẫn phân đoạn>]"),
			Description: utils.T(lang, utils.MsgImportCommandDescription),
			NeedsIdle:   true,
			Run: func(m Model, args []string) (tea.Model, tea.Cmd) {
				m.importSeq++
				state, listenCmd, err := startImport(m.runtime, m.importSeq, args, m.width, m.height, m.language)
				if err != nil {
					m.applyEvent(host.Event{
						Time: time.Now(), Category: "ERROR", Summary: ui(lang, "导入启动失败：", "Khởi động nhập thất bại: ") + err.Error(), Level: "error",
					})
					m.refreshEventViewport()
					return m, nil
				}
				m.importer = state
				m.importHint = "" // 已进入导入流程，欢迎屏的恢复提示完成使命
				m.textarea.Blur()
				return m, listenCmd
			},
		},
		{
			Name:        "reopen",
			Group:       "writing",
			Usage:       ui(lang, "/reopen [续写方向]", "/reopen [hướng viết tiếp]"),
			Description: utils.T(lang, utils.MsgReopenCommandDescription),
			NeedsIdle:   true,
			Run: func(m Model, args []string) (tea.Model, tea.Cmd) {
				if err := m.runtime.Reopen(strings.Join(args, " ")); err != nil {
					m.applyEvent(host.Event{
						Time: time.Now(), Category: "ERROR", Summary: ui(lang, "重开失败：", "Mở lại thất bại: ") + err.Error(), Level: "error",
					})
					m.refreshEventViewport()
					return m, nil
				}
				return m, tea.Batch(m.textarea.Focus(), resumeBook(m.runtime))
			},
		},
		{
			Name:        "cocreate",
			Aliases:     []string{"plan"},
			Group:       "writing",
			Usage:       "/cocreate",
			Description: utils.T(lang, utils.MsgCocreateCommandDescription),
			AutoExecute: true,
			Run: func(m Model, _ []string) (tea.Model, tea.Cmd) {
				if m.mode != modeRunning {
					m.applyEvent(host.Event{
						Time: time.Now(), Category: "ERROR", Summary: ui(lang, "阶段共创仅在创作中可用", "Cùng lập kế hoạch giai đoạn chỉ dùng khi đang sáng tác"), Level: "error",
					})
					m.refreshEventViewport()
					return m, nil
				}
				if !m.runtime.PauseForCoCreate() {
					m.applyEvent(host.Event{
						Time: time.Now(), Category: "ERROR", Summary: ui(lang, "无法进入阶段共创：全书已完成或已在共创中", "Không thể vào cùng lập kế hoạch: sách đã hoàn tất hoặc đang ở chế độ này"), Level: "error",
					})
					m.refreshEventViewport()
					return m, nil
				}
				m.cocreate = newStageCoCreateState(m.language)
				m.resizeTextarea()
				m.textarea.Blur()
				return m, m.sendCoCreate()
			},
		},
		{
			Name:        "simulate",
			Group:       "writing",
			Usage:       "/simulate",
			Description: utils.T(lang, utils.MsgSimulateCommandDescription),
			NeedsIdle:   true,
			Run: func(m Model, args []string) (tea.Model, tea.Cmd) {
				m.simSeq++
				state, listenCmd, err := startSimulate(m.runtime, m.simSeq, args, m.width, m.height, m.language)
				if err != nil {
					m.applyEvent(host.Event{
						Time: time.Now(), Category: "ERROR", Summary: ui(lang, "仿写画像启动失败：", "Khởi động hồ sơ mô phỏng thất bại: ") + err.Error(), Level: "error",
					})
					m.refreshEventViewport()
					return m, nil
				}
				m.simulator = state
				m.textarea.Blur()
				return m, listenCmd
			},
		},
		{
			Name:        "importsim",
			Group:       "writing",
			Usage:       "/importsim <profile.json>",
			Description: utils.T(lang, utils.MsgImportSimCommandDescription),
			NeedsIdle:   true,
			Run: func(m Model, args []string) (tea.Model, tea.Cmd) {
				m.simSeq++
				state, listenCmd, err := startImportSimulation(m.runtime, m.simSeq, args, m.width, m.height, m.language)
				if err != nil {
					m.applyEvent(host.Event{
						Time: time.Now(), Category: "ERROR", Summary: ui(lang, "导入仿写画像失败：", "Nhập hồ sơ mô phỏng thất bại: ") + err.Error(), Level: "error",
					})
					m.refreshEventViewport()
					return m, nil
				}
				m.simulator = state
				m.textarea.Blur()
				return m, listenCmd
			},
		},
		{
			Name:        "sync",
			Group:       "writing",
			Usage:       "/sync [--check]",
			Description: utils.T(lang, utils.MsgSyncCommandDescription),
			AutoExecute: true,
			NeedsIdle:   true,
			Run: func(m Model, args []string) (tea.Model, tea.Cmd) {
				cmd, checkOnly, err := startRevisionSync(m.runtime, args)
				if err != nil {
					m.applyEvent(host.Event{Time: time.Now(), Category: "ERROR", Summary: ui(lang, "章节同步启动失败：", "Khởi động đồng bộ chương thất bại: ") + err.Error(), Level: "error"})
					m.refreshEventViewport()
					return m, nil
				}
				summary := ui(lang, "正在分析并接纳章节修订...", "Đang phân tích và tiếp nhận sửa đổi chương...")
				if checkOnly {
					summary = ui(lang, "正在检查章节外部修改...", "Đang kiểm tra chương bị sửa bên ngoài...")
				}
				m.applyEvent(host.Event{Time: time.Now(), Category: "SYSTEM", Summary: summary, Level: "info"})
				m.refreshEventViewport()
				return m, cmd
			},
		},
		{
			Name:        "export",
			Group:       "writing",
			Usage:       "/export [path] [from=N] [to=M] [--overwrite]",
			Description: utils.T(lang, utils.MsgExportCommandDescription),
			AutoExecute: true,
			Run: func(m Model, args []string) (tea.Model, tea.Cmd) {
				cmd, err := startExport(m.runtime, args)
				if err != nil {
					m.applyEvent(host.Event{
						Time: time.Now(), Category: "ERROR", Summary: ui(lang, "导出启动失败：", "Khởi động xuất thất bại: ") + err.Error(), Level: "error",
					})
					m.refreshEventViewport()
					return m, nil
				}
				m.applyEvent(host.Event{
					Time: time.Now(), Category: "SYSTEM", Summary: ui(lang, "正在导出...", "Đang xuất..."), Level: "info",
				})
				m.refreshEventViewport()
				return m, cmd
			},
		},
	})
}

func commandSpecs(languages ...utils.Language) []slashCommandSpec {
	return commandRegistryInstance(languages...).Visible()
}

func prepareFileStart(args []string, languages ...utils.Language) (string, error) {
	path := strings.TrimSpace(strings.Join(args, " "))
	if len(path) >= 2 && ((path[0] == '"' && path[len(path)-1] == '"') ||
		(path[0] == '\'' && path[len(path)-1] == '\'')) {
		path = path[1 : len(path)-1]
	}
	if path == "" {
		return "", fmt.Errorf("%s", ui(resolveLanguage(languages), "用法：/start <设定或大纲文件路径>", "Cách dùng: /start <đường dẫn file thiết lập hoặc đại cương>"))
	}
	prompt, err := startup.LoadPromptFile(path)
	if err != nil {
		return "", err
	}
	return startup.PrepareQuick(prompt)
}

func (m Model) handleSlashCommand(cmd slashCommand) (tea.Model, tea.Cmd) {
	spec, ok := commandRegistryInstance(m.language).Find(cmd.name)
	if !ok {
		m.applyEvent(host.Event{
			Time: time.Now(), Category: "ERROR", Summary: utils.T(m.language, utils.MsgUnknownCommand, cmd.name), Level: "error",
		})
		m.refreshEventViewport()
		return m, nil
	}
	if spec.NeedsIdle && m.snapshot.IsRunning {
		m.applyEvent(host.Event{
			Time: time.Now(), Category: "ERROR", Summary: utils.T(m.language, utils.MsgCommandIdleOnly, spec.Name), Level: "error",
		})
		m.refreshEventViewport()
		return m, nil
	}
	return spec.Run(m, cmd.args)
}

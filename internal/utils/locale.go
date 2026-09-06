package utils

import (
	"fmt"
	"strings"
)

// Language identifies the language used by user-facing messages.
type Language string

const (
	LanguageVI Language = "vi"
	LanguageZH Language = "zh"
)

// ParseLanguage normalizes the supported language identifiers.
func ParseLanguage(value string) (Language, error) {
	switch strings.ToLower(strings.TrimSpace(value)) {
	case "", "vi", "vi-vn", "vietnamese":
		return LanguageVI, nil
	case "zh", "zh-cn", "zh-hans", "chinese":
		return LanguageZH, nil
	default:
		return "", fmt.Errorf("unsupported language %q (use vi or zh)", value)
	}
}

type MessageKey string

const (
	MsgReady                       MessageKey = "ready"
	MsgLoading                     MessageKey = "loading"
	MsgTerminalTooNarrow           MessageKey = "terminal_too_narrow"
	MsgUnknownCommand              MessageKey = "unknown_command"
	MsgCommandIdleOnly             MessageKey = "command_idle_only"
	MsgCommandUsage                MessageKey = "command_usage"
	MsgUnknownRole                 MessageKey = "unknown_role"
	MsgConfigCommandDescription    MessageKey = "config_command_description"
	MsgHelpCommandDescription      MessageKey = "help_command_description"
	MsgModelCommandDescription     MessageKey = "model_command_description"
	MsgDiagCommandDescription      MessageKey = "diag_command_description"
	MsgReviewCommandDescription    MessageKey = "review_command_description"
	MsgNextCommandDescription      MessageKey = "next_command_description"
	MsgStartCommandDescription     MessageKey = "start_command_description"
	MsgImportCommandDescription    MessageKey = "import_command_description"
	MsgReopenCommandDescription    MessageKey = "reopen_command_description"
	MsgCocreateCommandDescription  MessageKey = "cocreate_command_description"
	MsgSimulateCommandDescription  MessageKey = "simulate_command_description"
	MsgImportSimCommandDescription MessageKey = "importsim_command_description"
	MsgSyncCommandDescription      MessageKey = "sync_command_description"
	MsgExportCommandDescription    MessageKey = "export_command_description"
	MsgNewBookTitle                MessageKey = "new_book_title"
	MsgNoBookTitle                 MessageKey = "no_book_title"
	MsgInputNovelRequest           MessageKey = "input_novel_request"
	MsgStartingQuick               MessageKey = "starting_quick"
	MsgStartingCreation            MessageKey = "starting_creation"
	MsgCreationFinished            MessageKey = "creation_finished"
	MsgCreationPaused              MessageKey = "creation_paused"
	MsgCreationInterrupted         MessageKey = "creation_interrupted"
	MsgCreationPausing             MessageKey = "creation_pausing"
	MsgCreationResume              MessageKey = "creation_resume"
	MsgReviewWaiting               MessageKey = "review_waiting"
	MsgReviewNext                  MessageKey = "review_next"
	MsgSteerPlaceholder            MessageKey = "steer_placeholder"
	MsgDonePlaceholder             MessageKey = "done_placeholder"
	MsgQuickMode                   MessageKey = "quick_mode"
	MsgCoCreateMode                MessageKey = "cocreate_mode"
	MsgQuickModeSubtitle           MessageKey = "quick_mode_subtitle"
	MsgCoCreateModeSubtitle        MessageKey = "cocreate_mode_subtitle"
	MsgStartupMode                 MessageKey = "startup_mode"
	MsgQuickPlaceholder            MessageKey = "quick_placeholder"
	MsgCoCreatePlaceholder         MessageKey = "cocreate_placeholder"
	MsgAIThinking                  MessageKey = "ai_thinking"
	MsgAIReplying                  MessageKey = "ai_replying"
	MsgSend                        MessageKey = "send"
	MsgExit                        MessageKey = "exit"
	MsgScroll                      MessageKey = "scroll"
	MsgClose                       MessageKey = "close"
	MsgCancel                      MessageKey = "cancel"
	MsgAccept                      MessageKey = "accept"
	MsgCommandHelpTitle            MessageKey = "command_help_title"
	MsgShortcuts                   MessageKey = "shortcuts"
	MsgCommandSearchHint           MessageKey = "command_search_hint"
	MsgCommandAcceptHint           MessageKey = "command_accept_hint"
	MsgCommandCloseHint            MessageKey = "command_close_hint"
	MsgCommandPaletteTitle         MessageKey = "command_palette_title"
	MsgOverview                    MessageKey = "overview"
	MsgRuntimeState                MessageKey = "runtime_state"
	MsgPhase                       MessageKey = "phase"
	MsgFlow                        MessageKey = "flow"
	MsgProgress                    MessageKey = "progress"
	MsgCompleted                   MessageKey = "completed"
	MsgPlanned                     MessageKey = "planned"
	MsgWordCount                   MessageKey = "word_count"
	MsgCurrent                     MessageKey = "current"
	MsgWaitingResume               MessageKey = "waiting_resume"
	MsgRunningRoles                MessageKey = "running_roles"
	MsgQueue                       MessageKey = "queue"
	MsgReason                      MessageKey = "reason"
	MsgRework                      MessageKey = "rework"
	MsgIntervention                MessageKey = "intervention"
	MsgPending                     MessageKey = "pending"
	MsgAcceptanceHold              MessageKey = "acceptance_hold"
	MsgWaiting                     MessageKey = "waiting"
	MsgUsage                       MessageKey = "usage"
	MsgCache                       MessageKey = "cache"
	MsgInputTokens                 MessageKey = "input_tokens"
	MsgOutputTokens                MessageKey = "output_tokens"
	MsgCost                        MessageKey = "cost"
	MsgSaved                       MessageKey = "saved"
	MsgBudget                      MessageKey = "budget"
	MsgRole                        MessageKey = "role"
	MsgModel                       MessageKey = "model"
	MsgCacheHit                    MessageKey = "cache_hit"
	MsgCacheRead                   MessageKey = "cache_read"
	MsgCacheWrite                  MessageKey = "cache_write"
	MsgCacheDisabled               MessageKey = "cache_disabled"
	MsgAutoCacheNoPremium          MessageKey = "auto_cache_no_premium"
	MsgLinkBreak                   MessageKey = "link_break"
	MsgChapter                     MessageKey = "chapter"
	MsgVolume                      MessageKey = "volume"
	MsgSynopsis                    MessageKey = "synopsis"
	MsgOutline                     MessageKey = "outline"
	MsgCharacters                  MessageKey = "characters"
	MsgSummary                     MessageKey = "summary"
	MsgEventStream                 MessageKey = "event_stream"
	MsgLiveOutput                  MessageKey = "live_output"
	MsgExternalImport              MessageKey = "external_import"
	MsgProcessLog                  MessageKey = "process_log"
	MsgImportFailed                MessageKey = "import_failed"
	MsgImportComplete              MessageKey = "import_complete"
	MsgImportPaused                MessageKey = "import_paused"
	MsgSimulationProfile           MessageKey = "simulation_profile"
	MsgSimulationFailed            MessageKey = "simulation_failed"
	MsgSimulationReady             MessageKey = "simulation_ready"
	MsgDiagnosticReport            MessageKey = "diagnostic_report"
	MsgReportUnavailable           MessageKey = "report_unavailable"
	MsgReportLoading               MessageKey = "report_loading"
	MsgNoProblems                  MessageKey = "no_problems"
	MsgFindings                    MessageKey = "findings"
	MsgActions                     MessageKey = "actions"
	MsgConfigModel                 MessageKey = "config_model"
	MsgSelectProvider              MessageKey = "select_provider"
	MsgAddProvider                 MessageKey = "add_provider"
	MsgProviderName                MessageKey = "provider_name"
	MsgProtocolType                MessageKey = "protocol_type"
	MsgAPIKey                      MessageKey = "api_key"
	MsgBaseURL                     MessageKey = "base_url"
	MsgModelList                   MessageKey = "model_list"
	MsgContextWindow               MessageKey = "context_window"
	MsgReferences                  MessageKey = "references"
	MsgNoOptions                   MessageKey = "no_options"
	MsgNewModel                    MessageKey = "new_model"
	MsgAutomatic                   MessageKey = "automatic"
	MsgConnectionTest              MessageKey = "connection_test"
	MsgSaveConfig                  MessageKey = "save_config"
	MsgLanguage                    MessageKey = "language"
	MsgLanguageVietnamese          MessageKey = "language_vietnamese"
	MsgLanguageChinese             MessageKey = "language_chinese"
	MsgLanguageSavedRestart        MessageKey = "language_saved_restart"
	MsgSetupMissing                MessageKey = "setup_missing"
	MsgSetupPath                   MessageKey = "setup_path"
	MsgSetupEditHint               MessageKey = "setup_edit_hint"
	MsgHeadlessLogWarning          MessageKey = "headless_log_warning"
	MsgHeadlessDiagnosticWarning   MessageKey = "headless_diagnostic_warning"
	MsgHeadlessStart               MessageKey = "headless_start"
	MsgHeadlessResume              MessageKey = "headless_resume"
	MsgHeadlessNeedsPrompt         MessageKey = "headless_needs_prompt"
)

var catalog = map[Language]map[MessageKey]string{
	LanguageVI: {
		MsgReady: "Sẵn sàng", MsgLoading: "Đang tải...", MsgTerminalTooNarrow: "Terminal quá hẹp, hãy mở rộng lên ít nhất 100 cột", MsgStartupMode: "Chế độ khởi động",
		MsgUnknownCommand: "Lệnh không tồn tại: /%s", MsgCommandIdleOnly: "Lệnh chỉ dùng khi đang rảnh: /%s", MsgCommandUsage: "Cách dùng: %s", MsgUnknownRole: "Vai trò không tồn tại: %s",
		MsgHelpCommandDescription: "Xem danh sách lệnh", MsgModelCommandDescription: "Đổi model và mức suy luận của vai trò", MsgConfigCommandDescription: "Thêm hoặc chỉnh Provider, model và cửa sổ ngữ cảnh", MsgDiagCommandDescription: "Chẩn đoán tình trạng sáng tác", MsgReviewCommandDescription: "Bật/tắt chế độ duyệt từng chương", MsgNextCommandDescription: "Mở khóa một chương mới sau khi duyệt", MsgStartCommandDescription: "Tạo sách mới từ file thiết lập hoặc đại cương", MsgImportCommandDescription: "Nhập tiểu thuyết bên ngoài theo ngữ nghĩa", MsgReopenCommandDescription: "Mở lại sách đã hoàn tất để viết tiếp", MsgCocreateCommandDescription: "Tạm dừng để cùng lập kế hoạch giai đoạn tiếp theo", MsgSimulateCommandDescription: "Đọc ./simulate để tạo hoặc cập nhật hồ sơ mô phỏng", MsgImportSimCommandDescription: "Nhập hồ sơ mô phỏng và gộp theo dấu vân tay", MsgSyncCommandDescription: "Kiểm tra hoặc nhận các chương đã sửa thủ công", MsgExportCommandDescription: "Xuất các chương đã hoàn tất thành TXT/EPUB",
		MsgNewBookTitle: "Chưa đặt tên sách", MsgNoBookTitle: "Chưa đặt tên sách", MsgInputNovelRequest: "Nhập yêu cầu tiểu thuyết bên dưới để bắt đầu sáng tác", MsgStartingQuick: "Đang khởi tạo sáng tác...", MsgStartingCreation: "Đang khởi tạo sáng tác", MsgCreationFinished: "Sáng tác đã hoàn tất", MsgCreationPaused: "Sáng tác đã tạm dừng", MsgCreationInterrupted: "Sáng tác bị gián đoạn, nhập nội dung bất kỳ để tiếp tục", MsgCreationPausing: "Đang tạm dừng sáng tác...", MsgCreationResume: "Nhập nội dung bất kỳ để tiếp tục sáng tác", MsgReviewWaiting: "Đang chờ duyệt từng chương: nhập góp ý sửa hoặc /next để mở khóa chương tiếp", MsgReviewNext: "Duyệt từng chương: nhập góp ý sửa hoặc /next để mở khóa chương tiếp",
		MsgSteerPlaceholder: "Nhập chỉ đạo cốt truyện, ví dụ: đưa tuyến tình cảm lên chương 4", MsgDonePlaceholder: "Sáng tác đã hoàn tất · nhập yêu cầu làm lại (ví dụ \"viết lại chương 3\"), /reopen viết tập mới, /export để xuất", MsgQuickMode: "Bắt đầu nhanh", MsgCoCreateMode: "Cùng lập kế hoạch", MsgQuickModeSubtitle: "Một câu là bắt đầu viết ngay", MsgCoCreateModeSubtitle: "Trao đổi với AI để làm rõ rồi mới sáng tác", MsgQuickPlaceholder: "Nhập một câu yêu cầu tiểu thuyết, Enter để bắt đầu", MsgCoCreatePlaceholder: "Nhập ý tưởng cốt lõi, Enter để cùng AI lập kế hoạch", MsgAIThinking: "AI đang suy nghĩ", MsgAIReplying: "AI đang trả lời", MsgSend: "Enter gửi", MsgExit: "Esc thoát", MsgScroll: "↑↓ cuộn", MsgClose: "Esc đóng", MsgCancel: "Esc hủy", MsgAccept: "Enter xác nhận",
		MsgCommandHelpTitle: "Trợ giúp lệnh", MsgShortcuts: "Phím tắt", MsgCommandSearchHint: "Nhập / để tìm lệnh", MsgCommandAcceptHint: "Tab/Enter chấp nhận gợi ý tự động hoàn thành", MsgCommandCloseHint: "Esc đóng bảng lệnh hiện tại", MsgCommandPaletteTitle: "Lệnh",
		MsgOverview: "Tổng quan", MsgRuntimeState: "Trạng thái", MsgPhase: "Giai đoạn", MsgFlow: "Luồng", MsgProgress: "Tiến độ", MsgCompleted: "Đã hoàn tất", MsgPlanned: "Đã lập kế hoạch", MsgWordCount: "Số chữ", MsgCurrent: "Hiện tại", MsgWaitingResume: "Chờ khôi phục", MsgRunningRoles: "Vai trò đang chạy", MsgQueue: "Hàng đợi", MsgReason: "Lý do", MsgRework: "Làm lại", MsgIntervention: "Can thiệp", MsgPending: "Chờ xử lý", MsgAcceptanceHold: "Dừng chờ duyệt", MsgWaiting: "Đang chờ", MsgUsage: "Mức dùng", MsgCache: "Bộ nhớ đệm", MsgInputTokens: "Đầu vào", MsgOutputTokens: "Đầu ra", MsgCost: "Chi phí", MsgSaved: "Đã tiết kiệm", MsgBudget: "Ngân sách", MsgRole: "Vai trò", MsgModel: "Model", MsgCacheHit: "Lượt trúng cache", MsgCacheRead: "Đọc cache", MsgCacheWrite: "Ghi cache", MsgCacheDisabled: "Model hiện tại chưa bật prompt cache", MsgAutoCacheNoPremium: "(cache tự động không phụ phí)", MsgLinkBreak: "Đứt chuỗi cache",
		MsgChapter: "Chương", MsgVolume: "Tập", MsgSynopsis: "Giới thiệu", MsgOutline: "Đại cương", MsgCharacters: "Nhân vật", MsgSummary: "Tóm tắt", MsgEventStream: "Luồng sự kiện", MsgLiveOutput: "Đầu ra trực tiếp", MsgExternalImport: "Nhập tiểu thuyết bên ngoài", MsgProcessLog: "Nhật ký quy trình", MsgImportFailed: "Nhập thất bại", MsgImportComplete: "Nhập hoàn tất, thiết lập nền và chương đã sẵn sàng", MsgImportPaused: "Nhập đã tạm dừng, đang chờ thao tác", MsgSimulationProfile: "Hồ sơ mô phỏng", MsgSimulationFailed: "Xử lý hồ sơ mô phỏng thất bại", MsgSimulationReady: "Hồ sơ mô phỏng đã sẵn sàng", MsgDiagnosticReport: "Báo cáo chẩn đoán", MsgReportUnavailable: "Không có báo cáo chẩn đoán", MsgReportLoading: "Đang tạo báo cáo chẩn đoán", MsgNoProblems: "Không phát hiện vấn đề", MsgFindings: "Phát hiện", MsgActions: "Hành động có thể thực hiện",
		MsgConfigModel: "/config cấu hình model", MsgSelectProvider: "Chọn Provider cần chỉnh hoặc thêm mới", MsgAddProvider: "Chọn Provider muốn thêm", MsgProviderName: "Tên Provider", MsgProtocolType: "Loại giao thức API", MsgAPIKey: "API Key", MsgBaseURL: "Base URL", MsgModelList: "Quản lý danh sách model", MsgContextWindow: "Cửa sổ ngữ cảnh", MsgReferences: "Tham chiếu", MsgNoOptions: "Không có lựa chọn khả dụng", MsgNewModel: "+ Thêm model...", MsgAutomatic: "Tự động", MsgConnectionTest: "Kiểm tra kết nối", MsgSaveConfig: "Lưu cấu hình", MsgLanguage: "Ngôn ngữ", MsgLanguageVietnamese: "Tiếng Việt", MsgLanguageChinese: "中文", MsgLanguageSavedRestart: "Đã lưu ngôn ngữ %s. Hãy khởi động lại để áp dụng nhất quán.", MsgSetupMissing: "Không tìm thấy file cấu hình, bắt đầu thiết lập...", MsgSetupPath: "Đường dẫn file cấu hình: %s", MsgSetupEditHint: "Bạn có thể chỉnh file này để thay đổi thiết lập nâng cao.", MsgHeadlessLogWarning: "Cảnh báo log headless không khả dụng, tiếp tục ghi ra terminal: %v", MsgHeadlessDiagnosticWarning: "Cảnh báo không thể xuất báo cáo chẩn đoán: %v", MsgHeadlessStart: "headless bắt đầu: %s", MsgHeadlessResume: "headless khôi phục: %s (%s)", MsgHeadlessNeedsPrompt: "headless cần --prompt hoặc phiên có thể khôi phục trong thư mục output %q",
	},
	LanguageZH: {
		MsgReady: "就绪", MsgLoading: "加载中...", MsgTerminalTooNarrow: "终端宽度不足，请至少扩展到 100 列", MsgStartupMode: "启动模式",
		MsgUnknownCommand: "未知命令：/%s", MsgCommandIdleOnly: "命令仅可在空闲状态执行：/%s", MsgCommandUsage: "用法：%s", MsgUnknownRole: "未知角色：%s",
		MsgHelpCommandDescription: "查看命令列表", MsgModelCommandDescription: "切换角色的模型与推理强度", MsgConfigCommandDescription: "新增或编辑 Provider、模型与上下文窗口", MsgDiagCommandDescription: "诊断小说创作健康度", MsgReviewCommandDescription: "切换逐章验收模式", MsgNextCommandDescription: "验收后放行一个新章节", MsgStartCommandDescription: "从设定或大纲文件创建新书", MsgImportCommandDescription: "语义导入外部小说", MsgReopenCommandDescription: "重开已完结的书继续创作", MsgCocreateCommandDescription: "暂停创作，共创规划后续阶段走向", MsgSimulateCommandDescription: "读取 ./simulate 生成或增量更新仿写画像", MsgImportSimCommandDescription: "导入已有仿写画像并按语料指纹合并", MsgSyncCommandDescription: "检查或接纳手动修改的已完成章节", MsgExportCommandDescription: "导出已完成章节为 TXT/EPUB",
		MsgNewBookTitle: "未定书名", MsgNoBookTitle: "未定书名", MsgInputNovelRequest: "在下方输入小说需求，开始创作", MsgStartingQuick: "正在初始化创作...", MsgStartingCreation: "正在初始化创作", MsgCreationFinished: "创作已完成", MsgCreationPaused: "创作已暂停", MsgCreationInterrupted: "运行中断，输入任意内容恢复创作", MsgCreationPausing: "正在暂停创作...", MsgCreationResume: "输入任意内容继续创作", MsgReviewWaiting: "逐章验收等待中：输入修改意见，或 /next 放行下一章", MsgReviewNext: "逐章验收等待中：输入修改意见，或 /next 放行下一章",
		MsgSteerPlaceholder: "输入剧情干预，例如：把感情线提前到第4章", MsgDonePlaceholder: "创作已完成 · 可输入返工要求(如\"重写第3章\")、/reopen 续写新卷、/export 导出", MsgQuickMode: "快速开始", MsgCoCreateMode: "共创规划", MsgQuickModeSubtitle: "一句话直接开始写", MsgCoCreateModeSubtitle: "先与 AI 对话澄清，再开始创作", MsgQuickPlaceholder: "输入一句小说需求，Enter 直接开始创作", MsgCoCreatePlaceholder: "先输入你的核心想法，Enter 开始与 AI 共创", MsgAIThinking: "AI 思考中", MsgAIReplying: "AI 回复中", MsgSend: "Enter 发送", MsgExit: "Esc 退出", MsgScroll: "↑↓ 滚动", MsgClose: "Esc 关闭", MsgCancel: "Esc 取消", MsgAccept: "Enter 确认",
		MsgCommandHelpTitle: "命令帮助", MsgShortcuts: "快捷键", MsgCommandSearchHint: "输入 / 搜索命令", MsgCommandAcceptHint: "Tab/Enter 接受自动完成建议", MsgCommandCloseHint: "Esc 关闭当前命令面板", MsgCommandPaletteTitle: "命令",
		MsgOverview: "概览", MsgRuntimeState: "运行态", MsgPhase: "阶段", MsgFlow: "流程", MsgProgress: "进度", MsgCompleted: "已完成", MsgPlanned: "已规划", MsgWordCount: "字数", MsgCurrent: "当前", MsgWaitingResume: "待恢复", MsgRunningRoles: "运行角色", MsgQueue: "队列", MsgReason: "原因", MsgRework: "返工", MsgIntervention: "干预", MsgPending: "待处理", MsgAcceptanceHold: "验收停靠", MsgWaiting: "等待", MsgUsage: "用量", MsgCache: "缓存", MsgInputTokens: "输入", MsgOutputTokens: "输出", MsgCost: "费用", MsgSaved: "节省", MsgBudget: "预算", MsgRole: "角色", MsgModel: "模型", MsgCacheHit: "累计命中", MsgCacheRead: "缓存读量", MsgCacheWrite: "缓存写量", MsgCacheDisabled: "当前模型未启用 prompt cache", MsgAutoCacheNoPremium: "(自动缓存无溢价)", MsgLinkBreak: "链路断裂",
		MsgChapter: "章节", MsgVolume: "卷", MsgSynopsis: "简介", MsgOutline: "大纲", MsgCharacters: "角色", MsgSummary: "摘要", MsgEventStream: "事件流", MsgLiveOutput: "实时输出", MsgExternalImport: "导入外部小说", MsgProcessLog: "流程日志", MsgImportFailed: "导入失败", MsgImportComplete: "导入完成，基础设定和章节已就绪", MsgImportPaused: "导入已暂停，等待你的操作", MsgSimulationProfile: "仿写画像", MsgSimulationFailed: "仿写画像处理失败", MsgSimulationReady: "仿写画像已就绪", MsgDiagnosticReport: "诊断报告", MsgReportUnavailable: "诊断报告不可用", MsgReportLoading: "正在生成诊断报告", MsgNoProblems: "未发现问题", MsgFindings: "发现", MsgActions: "可执行动作",
		MsgConfigModel: "/config 配置模型", MsgSelectProvider: "选择要编辑的 Provider，或新增一个", MsgAddProvider: "选择要新增的 Provider", MsgProviderName: "Provider 名称", MsgProtocolType: "API 协议类型", MsgAPIKey: "API Key", MsgBaseURL: "Base URL", MsgModelList: "管理模型列表", MsgContextWindow: "上下文窗口", MsgReferences: "引用", MsgNoOptions: "没有可用选项", MsgNewModel: "+ 新增模型…", MsgAutomatic: "自动", MsgConnectionTest: "测试连接", MsgSaveConfig: "保存配置", MsgLanguage: "语言", MsgLanguageVietnamese: "Tiếng Việt", MsgLanguageChinese: "中文", MsgLanguageSavedRestart: "语言已保存为 %s。请重启程序以保持界面语言一致。", MsgSetupMissing: "未检测到配置文件，开始初始化设置...", MsgSetupPath: "配置文件路径：%s", MsgSetupEditHint: "完成后可随时编辑该文件调整高级设置。", MsgHeadlessLogWarning: "警告：headless 日志不可用，继续使用终端日志：%v", MsgHeadlessDiagnosticWarning: "警告：诊断报告导出失败：%v", MsgHeadlessStart: "headless 启动: %s", MsgHeadlessResume: "headless 恢复: %s (%s)", MsgHeadlessNeedsPrompt: "headless 模式需要 --prompt，或输出目录 %q 下已有可恢复会话",
	},
}

// T returns a localized message. Vietnamese is the safe fallback for unknown locales or keys.
func T(lang Language, key MessageKey, args ...any) string {
	if lang != LanguageZH {
		lang = LanguageVI
	}
	text, ok := catalog[lang][key]
	if !ok {
		text = catalog[LanguageVI][key]
	}
	if len(args) == 0 {
		return text
	}
	return fmt.Sprintf(text, args...)
}

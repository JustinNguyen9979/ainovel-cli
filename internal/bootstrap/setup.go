package bootstrap

import (
	_ "embed"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/JustinNguyen9979/ainovel-cli/internal/rules"
	"github.com/JustinNguyen9979/ainovel-cli/internal/utils"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

// exampleConfig 是引导后写入 ~/.ainovel/config.example.jsonc 的带注释模板。
// 嵌入文件必须与仓库根目录 config.example.jsonc 保持一致，测试会防止漂移。
//
//go:embed config.example.jsonc
var exampleConfig string

// NeedsSetup 检查是否需要首次引导（全局与项目级配置都不存在时触发）。
func NeedsSetup() bool {
	if p := DefaultConfigPath(); p != "" {
		if _, err := os.Stat(p); err == nil {
			return false
		}
	}
	if _, err := os.Stat(projectConfigPath()); err == nil {
		return false
	}
	return true
}

type setupProvider struct {
	name           string
	label          string
	baseURL        string // 预填的 base_url
	needType       bool   // 自定义代理需要额外问 type 和 base_url
	apiKeyOptional bool   // true 表示 API Key 允许留空
}

// ProviderPreset 是首次引导和运行时 /config 共用的 provider 目录项。
type ProviderPreset struct {
	Name           string
	Label          string
	BaseURL        string
	NeedType       bool
	APIKeyOptional bool
}

var setupProviders = []setupProvider{
	{name: "openrouter", label: "OpenRouter", baseURL: "https://openrouter.ai/api/v1"},
	{name: "anthropic", label: "Anthropic"},
	{name: "gemini", label: "Gemini"},
	{name: "openai", label: "OpenAI"},
	{name: "deepseek", label: "DeepSeek"},
	{name: "qwen", label: "Qwen"},
	{name: "glm", label: "GLM"},
	{name: "grok", label: "Grok"},
	{name: "ollama", label: "Ollama", baseURL: "http://localhost:11434/v1", apiKeyOptional: true},
	{name: "bedrock", label: "Bedrock", apiKeyOptional: true},
	{name: "custom", label: "Custom Proxy", needType: true, apiKeyOptional: true},
}

// ProviderPresets 返回一份可安全修改的预设列表。
func ProviderPresets() []ProviderPreset {
	out := make([]ProviderPreset, 0, len(setupProviders))
	for _, preset := range setupProviders {
		out = append(out, ProviderPreset{
			Name: preset.name, Label: preset.label, BaseURL: preset.baseURL,
			NeedType: preset.needType, APIKeyOptional: preset.apiKeyOptional,
		})
	}
	return out
}

// RunSetup 运行首次引导，返回生成的配置。
func RunSetup(languages ...utils.Language) (Config, error) {
	language := utils.LanguageVI
	if len(languages) > 0 && languages[0] != "" {
		language = languages[0]
	} else {
		selected, err := runLanguageSelect()
		if err != nil {
			return Config{}, err
		}
		language = selected
	}

	if _, err := utils.ParseLanguage(string(language)); err != nil {
		return Config{}, err
	}

	fmt.Fprintln(os.Stderr)
	fmt.Fprintln(os.Stderr, lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("99")).
		Render(setupText(language, "未检测到配置文件，开始初始化设置...", "Không tìm thấy file cấu hình, bắt đầu thiết lập...")))
	fmt.Fprintf(os.Stderr, "  %s\n", lipgloss.NewStyle().Foreground(lipgloss.Color("245")).Render(
		setupText(language, "配置文件路径："+DefaultConfigPath(), "Đường dẫn file cấu hình: "+DefaultConfigPath())))
	fmt.Fprintf(os.Stderr, "  %s\n", setupText(language, "完成后可随时编辑该文件调整高级设置。", "Bạn có thể chỉnh file này để thay đổi thiết lập nâng cao sau khi hoàn tất."))
	fmt.Fprintln(os.Stderr)

	// Step 1: 选择 Provider
	sp, err := runProviderSelect(language)
	if err != nil {
		return Config{}, err
	}

	providerName := sp.name
	var pc ProviderConfig
	printStepDone(setupText(language, "Provider", "Provider"), sp.label)

	// 自定义代理：额外问名称和 API 协议类型
	if sp.needType {
		providerName, err = runTextInput(setupText(language, "Provider 名称", "Tên Provider"), "my-proxy", language)
		if err != nil {
			return Config{}, err
		}
		providerType, err := runTypeSelect(language)
		if err != nil {
			return Config{}, err
		}
		pc.Type = providerType
	}

	// Step 2: 输入 API Key
	var apiKey string
	if sp.apiKeyOptional {
		apiKey, err = runOptionalTextInput(setupText(language, "[3/5] API Key（可留空）", "[3/5] API Key (có thể để trống)"), setupText(language, "留空表示不使用 API Key", "Để trống nếu không dùng API Key"), language)
	} else {
		apiKey, err = runTextInput(setupText(language, "[3/5] API Key", "[3/5] API Key"), "sk-xxx", language)
	}
	if err != nil {
		return Config{}, err
	}
	pc.APIKey = apiKey
	if apiKey == "" {
		printStepDone("API Key", setupText(language, "未设置", "Chưa đặt"))
	} else {
		printStepDone("API Key", maskKey(apiKey))
	}

	// Step 4: Base URL（直接回车使用官方默认地址）
	baseDefault := sp.baseURL
	baseHint := setupText(language, "留空使用官方地址", "Để trống dùng địa chỉ mặc định")
	if baseDefault != "" {
		baseHint = baseDefault
	}
	baseURL, err := runTextInputWithDefault(setupText(language, "[4/5] Base URL（直接回车使用默认，代理用户填写代理地址）", "[4/5] Base URL (Enter dùng mặc định, proxy nhập địa chỉ proxy)"), baseHint, baseDefault, language)
	if err != nil {
		return Config{}, err
	}
	pc.BaseURL = baseURL
	if baseURL != "" {
		printStepDone("Base URL", baseURL)
	} else {
		printStepDone("Base URL", setupText(language, "默认", "Mặc định"))
	}

	// Step 5: 模型名（必填）
	modelName, err := runTextInput(setupText(language, "[5/5] 模型名称", "[5/5] Tên model"), setupText(language, "例如：gpt-4o / claude-sonnet-4 / gemini-2.5-pro", "Ví dụ: gpt-4o / claude-sonnet-4 / gemini-2.5-pro"), language)
	if err != nil {
		return Config{}, err
	}
	printStepDone(setupText(language, "模型", "Model"), modelName)
	pc.Models = []ModelConfig{{Name: modelName}}

	cfg := Config{
		Provider:  providerName,
		ModelName: modelName,
		Language:  string(language),
		Providers: map[string]ProviderConfig{providerName: pc},
		Roles:     map[string]RoleConfig{},
		Style:     "default",
	}

	// 保存
	path := DefaultConfigPath()
	if err := SaveConfig(path, cfg); err != nil {
		return cfg, fmt.Errorf("save config: %w", err)
	}

	// 生成注释模板
	saveExampleConfig()

	// 全局偏好目录由启动流程（runWithConfig）统一创建，这里仅取路径用于提示
	rulesDir := rules.DefaultHomeRulesDir()

	fmt.Fprintln(os.Stderr)
	fmt.Fprintf(os.Stderr, "%s %s %s\n",
		lipgloss.NewStyle().Foreground(lipgloss.Color("42")).Render("✓"),
		setupText(language, "配置已保存到", "Đã lưu cấu hình vào"), path)
	fmt.Fprintf(os.Stderr, "  %s：%s\n", setupText(language, "默认模型", "Model mặc định"), modelName)
	fmt.Fprintln(os.Stderr, setupText(language, "如需按角色配置不同模型，编辑配置文件即可。", "Để cấu hình model riêng cho từng vai trò, hãy chỉnh file cấu hình."))
	if rulesDir != "" {
		fmt.Fprintf(os.Stderr, "  %s %s (.md, README.txt)\n", setupText(language, "全局写作偏好可放在", "Tùy chọn sáng tác có thể đặt trong"), rulesDir)
	}
	fmt.Fprintln(os.Stderr)

	return cfg, nil
}

func setupText(lang utils.Language, zh, vi string) string {
	if lang == utils.LanguageZH {
		return zh
	}
	return vi
}

func localizedSetupProviders(lang utils.Language) []setupProvider {
	items := append([]setupProvider(nil), setupProviders...)
	if lang == utils.LanguageZH {
		return items
	}
	for i := range items {
		if items[i].name == "custom" {
			items[i].label = "Proxy tùy chỉnh"
		}
	}
	return items
}

func saveExampleConfig() {
	dir, err := configDir()
	if err != nil {
		return
	}
	_ = os.WriteFile(filepath.Join(dir, "config.example.jsonc"), []byte(exampleConfig), 0o644)
}

// printStepDone 打印一步完成的确认行。
func printStepDone(label, value string) {
	fmt.Fprintf(os.Stderr, "  %s %s: %s\n",
		lipgloss.NewStyle().Foreground(lipgloss.Color("42")).Render("✓"),
		label,
		lipgloss.NewStyle().Foreground(lipgloss.Color("245")).Render(value))
}

func maskKey(key string) string {
	if len(key) <= 8 {
		return "****"
	}
	return key[:4] + "****" + key[len(key)-4:]
}

// ---------- TUI 组件 ----------

func runProviderSelect(languages ...utils.Language) (setupProvider, error) {
	lang := utils.LanguageVI
	if len(languages) > 0 {
		lang = languages[0]
	}
	m := setupSelectModel{
		language: lang,
		title:    setupText(lang, "[2/5] 选择 Provider", "[2/5] Chọn Provider"),
		items:    localizedSetupProviders(lang),
	}
	p := tea.NewProgram(m, tea.WithOutput(os.Stderr))
	final, err := p.Run()
	if err != nil {
		return setupProvider{}, err
	}
	result := final.(setupSelectModel)
	if result.cancelled {
		return setupProvider{}, fmt.Errorf("setup cancelled")
	}
	return result.items[result.cursor], nil
}

var apiTypeOptions = []setupProvider{
	{name: "openai", label: "OpenAI 兼容"},
	{name: "anthropic", label: "Anthropic 兼容"},
	{name: "gemini", label: "Gemini 兼容"},
}

func runTypeSelect(languages ...utils.Language) (string, error) {
	lang := utils.LanguageVI
	if len(languages) > 0 {
		lang = languages[0]
	}
	items := append([]setupProvider(nil), apiTypeOptions...)
	if lang != utils.LanguageZH {
		items[0].label = "Tương thích OpenAI"
		items[1].label = "Tương thích Anthropic"
		items[2].label = "Tương thích Gemini"
	}
	m := setupSelectModel{
		language: lang,
		title:    setupText(lang, "API 协议类型", "Loại giao thức API"),
		items:    items,
	}
	p := tea.NewProgram(m, tea.WithOutput(os.Stderr))
	final, err := p.Run()
	if err != nil {
		return "", err
	}
	result := final.(setupSelectModel)
	if result.cancelled {
		return "", fmt.Errorf("setup cancelled")
	}
	return result.items[result.cursor].name, nil
}

func runTextInput(label, placeholder string, languages ...utils.Language) (string, error) {
	return runTextInputWithDefault(label, placeholder, "", languages...)
}

func runOptionalTextInput(label, placeholder string, languages ...utils.Language) (string, error) {
	lang := utils.LanguageVI
	if len(languages) > 0 {
		lang = languages[0]
	}
	m := setupInputModel{label: label, placeholder: placeholder, allowEmpty: true, language: lang}
	p := tea.NewProgram(m, tea.WithOutput(os.Stderr))
	final, err := p.Run()
	if err != nil {
		return "", err
	}
	result := final.(setupInputModel)
	if result.cancelled {
		return "", fmt.Errorf("setup cancelled")
	}
	return utils.CleanInputLine(result.value), nil
}

func runTextInputWithDefault(label, placeholder, defaultValue string, languages ...utils.Language) (string, error) {
	lang := utils.LanguageVI
	if len(languages) > 0 {
		lang = languages[0]
	}
	m := setupInputModel{label: label, placeholder: placeholder, defaultValue: defaultValue, language: lang}
	p := tea.NewProgram(m, tea.WithOutput(os.Stderr))
	final, err := p.Run()
	if err != nil {
		return "", err
	}
	result := final.(setupInputModel)
	if result.cancelled {
		return "", fmt.Errorf("setup cancelled")
	}
	if result.value == "" && result.defaultValue != "" {
		return result.defaultValue, nil
	}
	return utils.CleanInputLine(result.value), nil
}

// runLanguageSelect chooses the language for the initial setup.
func runLanguageSelect() (utils.Language, error) {
	m := setupSelectModel{
		language: utils.LanguageVI,
		title:    "[1/5] Ngôn ngữ / 语言",
		items: []setupProvider{
			{name: string(utils.LanguageVI), label: "Tiếng Việt"},
			{name: string(utils.LanguageZH), label: "中文"},
		},
	}
	p := tea.NewProgram(m, tea.WithOutput(os.Stderr))
	final, err := p.Run()
	if err != nil {
		return "", err
	}
	result := final.(setupSelectModel)
	if result.cancelled {
		return "", fmt.Errorf("setup cancelled")
	}
	return utils.ParseLanguage(result.items[result.cursor].name)
}

// ---------- 选择器 ----------

var (
	setupCursorStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("212"))
	setupDimStyle    = lipgloss.NewStyle().Foreground(lipgloss.Color("245"))
	setupHeaderStyle = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("99"))
	setupInputStyle  = lipgloss.NewStyle().Foreground(lipgloss.Color("212"))
)

type setupSelectModel struct {
	language  utils.Language
	title     string
	items     []setupProvider
	cursor    int
	cancelled bool
}

func (m setupSelectModel) Init() tea.Cmd { return nil }

func (m setupSelectModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	if msg, ok := msg.(tea.KeyMsg); ok {
		switch msg.String() {
		case "up", "k":
			if m.cursor > 0 {
				m.cursor--
			}
		case "down", "j":
			if m.cursor < len(m.items)-1 {
				m.cursor++
			}
		case "enter":
			return m, tea.Quit
		case "q", "esc", "ctrl+c":
			m.cancelled = true
			return m, tea.Quit
		}
	}
	return m, nil
}

func (m setupSelectModel) View() string {
	var b strings.Builder
	b.WriteString(setupHeaderStyle.Render(m.title))
	b.WriteString("\n\n")
	for i, item := range m.items {
		cursor := "  "
		label := item.label
		if i == m.cursor {
			cursor = setupCursorStyle.Render("❯ ")
			label = setupCursorStyle.Render(label)
		}
		b.WriteString(cursor + label + "\n")
	}
	b.WriteString(setupDimStyle.Render(setupText(m.language, "\n  ↑↓ 选择  Enter 确认  Esc 取消", "\n  ↑↓ chọn  Enter xác nhận  Esc hủy")))
	return b.String()
}

// ---------- 文本输入 ----------

type setupInputModel struct {
	language     utils.Language
	label        string
	placeholder  string
	defaultValue string // 直接回车时使用的默认值
	allowEmpty   bool   // 允许直接输入空值
	value        string
	cancelled    bool
}

func (m setupInputModel) Init() tea.Cmd { return nil }

func (m setupInputModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	if msg, ok := msg.(tea.KeyMsg); ok {
		switch msg.String() {
		case "enter":
			if utils.CleanInputLine(m.value) != "" || m.defaultValue != "" || m.allowEmpty {
				return m, tea.Quit
			}
		case "ctrl+c", "esc":
			m.cancelled = true
			return m, tea.Quit
		case "backspace":
			if len(m.value) > 0 {
				runes := []rune(m.value)
				m.value = string(runes[:len(runes)-1])
			}
		default:
			if msg.Type == tea.KeyRunes {
				m.value += utils.CleanInputRunes(msg.Runes)
			} else if msg.Type == tea.KeySpace {
				m.value += " "
			}
		}
	}
	return m, nil
}

func (m setupInputModel) View() string {
	var b strings.Builder
	b.WriteString(setupHeaderStyle.Render(m.label))
	b.WriteString("\n\n")
	b.WriteString(setupInputStyle.Render("❯ "))
	if m.value == "" {
		b.WriteString(setupCursorStyle.Render("▌"))
		b.WriteString(setupDimStyle.Render(m.placeholder))
	} else {
		b.WriteString(m.value)
		b.WriteString(setupCursorStyle.Render("▌"))
	}
	b.WriteString(setupDimStyle.Render(setupText(m.language, "  (Enter 确认, Esc 取消)", "  (Enter xác nhận, Esc hủy)")))
	b.WriteString("\n")
	return b.String()
}

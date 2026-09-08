package host

import (
	"strings"
	"testing"

	"github.com/JustinNguyen9979/ainovel-cli/internal/utils"
)

func TestParseCoCreateResponse(t *testing.T) {
	tests := []struct {
		name        string
		raw         string
		message     string
		prompt      string
		ready       bool
		suggestions []string
		wantErr     bool
	}{
		{name: "empty", raw: "  ", wantErr: true},
		{name: "natural language", raw: "直接回复用户", message: "直接回复用户"},
		{name: "complete", raw: "<reply>你好</reply><draft>## 主题</draft><ready>true</ready><suggestions>- 补充冲突\n2. 收束结尾</suggestions>", message: "你好", prompt: "## 主题", ready: true, suggestions: []string{"补充冲突", "收束结尾"}},
		{name: "yes", raw: "<reply>继续</reply><ready>YES</ready>", message: "继续", ready: true},
		{name: "missing close", raw: "<reply>半段<draft>草稿", message: "半段", prompt: "草稿"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := parseCoCreateResponse(tt.raw)
			if (err != nil) != tt.wantErr {
				t.Fatalf("error = %v, wantErr=%v", err, tt.wantErr)
			}
			if err != nil {
				return
			}
			if got.Message != tt.message || got.Prompt != tt.prompt || got.Ready != tt.ready || !equalStrings(got.Suggestions, tt.suggestions) {
				t.Fatalf("reply = %+v", got)
			}
			if got.Raw != strings.TrimSpace(tt.raw) {
				t.Fatalf("raw = %q", got.Raw)
			}
		})
	}
}

func TestExtractTagContentFallbacks(t *testing.T) {
	tests := []struct {
		name string
		text string
		tag  string
		want string
	}{
		{name: "closed", text: "<reply> hello </reply>", tag: "reply", want: "hello"},
		{name: "next tag", text: "<reply> hello <draft>draft", tag: "reply", want: "hello"},
		{name: "missing open", text: "intro</reply>", tag: "reply", want: "intro"},
		{name: "after prior close", text: "<reply>old</reply>draft</draft>", tag: "draft", want: "draft"},
		{name: "missing both", text: "plain", tag: "reply", want: ""},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := extractTagContent(tt.text, tt.tag); got != tt.want {
				t.Fatalf("extractTagContent = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestParseSuggestions(t *testing.T) {
	text := "\n- 第一条\n* 第二条\n12. 第三条\n1. 忽略第四条\n<uggestions>\n短\n"
	want := []string{"第一条", "第二条", "第三条"}
	if got := parseSuggestions(text); !equalStrings(got, want) {
		t.Fatalf("suggestions = %#v, want %#v", got, want)
	}
	if got := parseSuggestions(""); got != nil {
		t.Fatalf("empty suggestions = %#v", got)
	}
	if !isOrderedSuggestion("12. yes") || isOrderedSuggestion("12.yes") || isOrderedSuggestion(". yes") {
		t.Fatal("ordered suggestion detection mismatch")
	}
	if got := stripOrderedPrefix("12. yes"); got != "yes" || stripOrderedPrefix("plain") != "plain" {
		t.Fatal("ordered prefix stripping mismatch")
	}
}

func TestExtractReplyPreview(t *testing.T) {
	tests := map[string]string{
		"<reply>你好":                  "你好",
		"<reply>你好</reply><draft>草稿": "你好",
		"你好</reply>":                 "你好",
		"<reply>你好<draft>":           "你好",
		"plain":                      "plain",
	}
	for raw, want := range tests {
		if got := extractReplyPreview(raw); got != want {
			t.Errorf("preview(%q) = %q, want %q", raw, got, want)
		}
	}
}

func TestCoCreateLanguageHelpers(t *testing.T) {
	if coCreateLanguage([]utils.Language{utils.LanguageZH}) != utils.LanguageZH || coCreateLanguage(nil) != utils.LanguageVI {
		t.Fatal("language selection mismatch")
	}
	if hostLanguage("invalid") != utils.LanguageVI || hostLanguage("zh") != utils.LanguageZH {
		t.Fatal("host language fallback mismatch")
	}
	if localizedCoCreateEvent(utils.LanguageZH, "中文", "越南语") != "中文" || localizedCoCreateEvent(utils.LanguageVI, "中文", "越南语") != "越南语" {
		t.Fatal("localized event mismatch")
	}
	if !strings.Contains(coCreateSystemPrompt(utils.LanguageVI), "共创") && !strings.Contains(coCreateSystemPrompt(utils.LanguageVI), "đồng sáng tác") {
		t.Fatal("Vietnamese system prompt missing")
	}
	if !strings.Contains(stageCoCreateSystemPrompt(utils.LanguageZH), "阶段共创") {
		t.Fatal("Chinese stage prompt missing")
	}
}

func equalStrings(a, b []string) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}

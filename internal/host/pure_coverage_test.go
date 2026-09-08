package host

import (
	"context"
	"encoding/json"
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/JustinNguyen9979/ainovel-cli/internal/arbiter"
	"github.com/JustinNguyen9979/ainovel-cli/internal/domain"
	"github.com/JustinNguyen9979/ainovel-cli/internal/flow"
	"github.com/JustinNguyen9979/ainovel-cli/internal/utils"
	"github.com/voocel/agentcore"
	"github.com/voocel/agentcore/llm"
)

func TestHostSmallCoverageHelpers(t *testing.T) {
	if got := (arbiter.InterventionFacts{}).QueueHead(); got != 0 {
		t.Fatal("empty queue head should be zero")
	}
	if got := (arbiter.InterventionFacts{PendingRewrites: []int{3, 4}}).QueueHead(); got != 3 {
		t.Fatalf("queue head = %d", got)
	}
	if got := withAdvanceReason("message", "reason"); !strings.Contains(got, "reason") || withAdvanceReason("message", "") != "message" {
		t.Fatalf("advance reason = %q", got)
	}
	if got := localizedStoryWarning(utils.LanguageVI, "scope", errors.New("bad")); !strings.Contains(got, "đọc thất bại") || !strings.Contains(localizedStoryWarning(utils.LanguageZH, "scope", errors.New("bad")), "读取失败") {
		t.Fatalf("story warning = %q", got)
	}
	for _, raw := range []string{"", "0", "a", "f", "F", "g", "1234"} {
		_, _ = parseHex4([]byte(raw))
	}
}

func TestResumeAndHostPureHelpers(t *testing.T) {
	if got := legacyPremiseTitle("intro\n# 《书名》\n## 核心冲突\n冲突"); got != "书名" {
		t.Fatalf("legacyPremiseTitle = %q", got)
	}
	if got := legacyPremiseTitle("没有标题"); got != "" {
		t.Fatalf("missing title = %q", got)
	}
	premise := "# 书名\n\n## 核心冲突\n\n主角必须选择。\n\n## 主角目标\n活下去。"
	if got := legacyPremiseSection(premise, "核心冲突"); got != "主角必须选择。" {
		t.Fatalf("legacyPremiseSection = %q", got)
	}
	if got := legacyPremiseSection(premise, "不存在"); got != "" {
		t.Fatalf("missing section = %q", got)
	}
	if deriveStatusLabel(UISnapshot{Phase: string(domain.PhaseComplete)}) != "COMPLETE" ||
		deriveStatusLabel(UISnapshot{Flow: string(domain.FlowReviewing)}) != "REVIEW" ||
		deriveStatusLabel(UISnapshot{Flow: string(domain.FlowRewriting)}) != "REWRITE" ||
		deriveStatusLabel(UISnapshot{RuntimeState: "running"}) != "RUNNING" ||
		deriveStatusLabel(UISnapshot{}) != "READY" {
		t.Fatal("status label precedence mismatch")
	}
	if got := truncate("你好世界", 3); got != "你好世..." || truncate("abc", 10) != "abc" || truncate("abc", 0) != "..." {
		t.Fatal("truncate mismatch")
	}
	if got := instructionKey(nil); got != "" || instructionKey(&flow.Instruction{Agent: "writer", Task: "续写"}) != "writer\x00续写" {
		t.Fatal("instruction key mismatch")
	}
	if contentFilterAdvice(errors.New("ordinary")) != "" || !strings.Contains(contentFilterAdvice(agentcore.ErrProviderContentFilter), "内容审核") {
		t.Fatal("content filter advice mismatch")
	}
	if got := completionSummary(domain.Progress{CompletedChapters: []int{1, 2}, TotalWordCount: 3456}, domain.BookMetadata{Title: "测试书"}); !strings.Contains(got, "测试书") || !strings.Contains(got, "3456") {
		t.Fatalf("completion summary = %q", got)
	}
}

func TestObserverFormattingHelpers(t *testing.T) {
	cases := []struct {
		attempt, max int
		delay        time.Duration
		want         string
	}{
		{1, 0, 0, "重试 (第1次): "},
		{2, 0, 1500 * time.Millisecond, "重试 (第2次，2s后): "},
		{1, 3, 0, "重试 (1/3): "},
		{2, 3, 2500 * time.Millisecond, "重试 (2/3，3s后): "},
	}
	for _, tc := range cases {
		if got := retryPrefix(tc.attempt, tc.max, tc.delay); got != tc.want {
			t.Errorf("retryPrefix = %q, want %q", got, tc.want)
		}
	}
	for _, tc := range []struct {
		delay time.Duration
		want  string
	}{{0, ""}, {-time.Second, ""}, {500 * time.Millisecond, "1s"}, {1500 * time.Millisecond, "2s"}, {2 * time.Second, "2s"}} {
		if got := formatRetryDelay(tc.delay); got != tc.want {
			t.Errorf("formatRetryDelay(%s) = %q, want %q", tc.delay, got, tc.want)
		}
	}
	for raw, want := range map[string]string{
		`{"chapter":"12","title":"标题"}`: "12",
		`{"title":"标题"}`:                "",
		`{`:                             "",
	} {
		if got := firstJSONStringField(raw, "chapter"); got != want {
			t.Errorf("firstJSONStringField(%q) = %q, want %q", raw, got, want)
		}
	}
	if displayToolName("save_foundation", json.RawMessage(`{"type":"premise"}`)) != "save_foundation[premise]" ||
		displayToolName("save_foundation", json.RawMessage(`{`)) != "save_foundation" ||
		displayToolName("read_chapter", nil) != "read_chapter" {
		t.Fatal("tool display mismatch")
	}
}

func TestHostEventLifecycle(t *testing.T) {
	for _, tc := range []struct {
		category string
		id       string
		finished bool
		running  bool
	}{
		{"TOOL", "1", false, true},
		{"DISPATCH", "2", true, false},
		{"DECISION", "3", false, true},
		{"SYSTEM", "4", false, false},
		{"TOOL", "", false, false},
		{"ERROR", "", true, false},
	} {
		ev := Event{Category: tc.category, ID: tc.id}
		if tc.finished {
			ev.FinishedAt = time.Now()
		}
		if got := ev.Running(); got != tc.running {
			t.Errorf("%+v Running = %v, want %v", tc, got, tc.running)
		}
	}
}

func TestUsageTrackedModelForwardsCalls(t *testing.T) {
	inner := &usageTestModel{}
	var recorded []string
	wrapped := newUsageTrackedModel(inner, "arbiter", func(agent, task string, msg agentcore.AgentMessage) {
		recorded = append(recorded, agent+":"+task+":"+msg.TextContent())
	})
	resp, err := wrapped.Generate(context.Background(), nil, nil)
	if err != nil || resp == nil || len(recorded) != 1 || !strings.Contains(recorded[0], "arbiter") {
		t.Fatalf("Generate = %+v, %v, recorded=%v", resp, err, recorded)
	}
	if !wrapped.SupportsTools() {
		t.Fatal("SupportsTools was not forwarded")
	}
	ch, err := wrapped.GenerateStream(context.Background(), nil, nil)
	if err != nil {
		t.Fatalf("GenerateStream: %v", err)
	}
	for range ch {
	}
	if len(recorded) != 1 {
		t.Fatal("stream generation should not record a second usage message")
	}
	failed := newUsageTrackedModel(&usageErrorModel{}, "arbiter", func(string, string, agentcore.AgentMessage) { t.Fatal("failed response should not record") })
	if _, err := failed.Generate(context.Background(), nil, nil); err == nil {
		t.Fatal("expected generation error")
	}
	if newUsageTrackedModel(inner, "arbiter", nil) != inner {
		t.Fatal("nil recorder should return inner model")
	}
}

type usageTestModel struct{}

func (*usageTestModel) Generate(context.Context, []agentcore.Message, []agentcore.ToolSpec, ...agentcore.CallOption) (*agentcore.LLMResponse, error) {
	return &agentcore.LLMResponse{Message: agentcore.UserMsg("ok")}, nil
}
func (*usageTestModel) GenerateStream(context.Context, []agentcore.Message, []agentcore.ToolSpec, ...agentcore.CallOption) (<-chan agentcore.StreamEvent, error) {
	ch := make(chan agentcore.StreamEvent, 1)
	ch <- agentcore.StreamEvent{Type: agentcore.StreamEventDone}
	close(ch)
	return ch, nil
}
func (*usageTestModel) SupportsTools() bool { return true }

// usageErrorModel exercises the wrapper's successful-response guard.
type usageErrorModel struct{}

func (*usageErrorModel) Generate(context.Context, []agentcore.Message, []agentcore.ToolSpec, ...agentcore.CallOption) (*agentcore.LLMResponse, error) {
	return nil, errors.New("failed")
}
func (*usageErrorModel) GenerateStream(context.Context, []agentcore.Message, []agentcore.ToolSpec, ...agentcore.CallOption) (<-chan agentcore.StreamEvent, error) {
	return nil, errors.New("failed")
}
func (*usageErrorModel) SupportsTools() bool { return false }

var _ llm.CapabilityProvider = (*capabilityTestModel)(nil)

type capabilityTestModel struct{}

func (*capabilityTestModel) Generate(context.Context, []agentcore.Message, []agentcore.ToolSpec, ...agentcore.CallOption) (*agentcore.LLMResponse, error) {
	return nil, nil
}
func (*capabilityTestModel) GenerateStream(context.Context, []agentcore.Message, []agentcore.ToolSpec, ...agentcore.CallOption) (<-chan agentcore.StreamEvent, error) {
	return nil, nil
}
func (*capabilityTestModel) SupportsTools() bool { return false }
func (*capabilityTestModel) Capabilities() llm.Capabilities {
	return llm.Capabilities{Thinking: llm.ThinkingCapabilities{Supported: llm.SupportNo}}
}

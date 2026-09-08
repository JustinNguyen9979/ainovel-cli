package agents

import (
	"context"
	"testing"

	"github.com/JustinNguyen9979/ainovel-cli/internal/bootstrap"
	"github.com/voocel/agentcore"
	"github.com/voocel/agentcore/llm"
)

func TestThinkingPolicyWithCapabilityProvider(t *testing.T) {
	model := &noThinkingModel{}
	for _, level := range []agentcore.ThinkingLevel{"", agentcore.ThinkingAuto, agentcore.ThinkingHigh} {
		got, ok := ResolveThinkingForModel(model, level)
		if level == agentcore.ThinkingAuto {
			if !ok || got != agentcore.ThinkingAuto {
				t.Fatalf("auto resolve = %q/%v", got, ok)
			}
		} else if ok || got != agentcore.ThinkingAuto {
			t.Fatalf("unsupported thinking %q = %q/%v", level, got, ok)
		}
	}
	if got := AvailableThinkingForModel(model); len(got) != 1 || got[0] != agentcore.ThinkingAuto {
		t.Fatalf("available = %#v", got)
	}
	plain := &plainThinkingModel{}
	if len(AvailableThinkingForModel(plain)) == 0 {
		t.Fatal("plain model should use policy defaults")
	}
	cfg := bootstrap.Config{ReasoningEffort: "invalid"}
	if got := roleThinking(cfg, "writer"); got != "" {
		t.Fatalf("invalid role effort = %q", got)
	}
	if got := resolvedRoleThinking(model, bootstrap.Config{ReasoningEffort: "high"}, "writer"); got != agentcore.ThinkingAuto {
		t.Fatalf("resolved unsupported effort = %q", got)
	}
}

type plainThinkingModel struct{}

func (*plainThinkingModel) Generate(context.Context, []agentcore.Message, []agentcore.ToolSpec, ...agentcore.CallOption) (*agentcore.LLMResponse, error) {
	return nil, nil
}
func (*plainThinkingModel) GenerateStream(context.Context, []agentcore.Message, []agentcore.ToolSpec, ...agentcore.CallOption) (<-chan agentcore.StreamEvent, error) {
	return nil, nil
}
func (*plainThinkingModel) SupportsTools() bool { return false }

type noThinkingModel struct{}

func (*noThinkingModel) Generate(context.Context, []agentcore.Message, []agentcore.ToolSpec, ...agentcore.CallOption) (*agentcore.LLMResponse, error) {
	return nil, nil
}
func (*noThinkingModel) GenerateStream(context.Context, []agentcore.Message, []agentcore.ToolSpec, ...agentcore.CallOption) (<-chan agentcore.StreamEvent, error) {
	return nil, nil
}
func (*noThinkingModel) SupportsTools() bool { return false }
func (*noThinkingModel) Capabilities() llm.Capabilities {
	return llm.Capabilities{Thinking: llm.ThinkingCapabilities{Supported: llm.SupportNo}}
}

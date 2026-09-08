package agents

import (
	"context"
	"testing"

	"github.com/JustinNguyen9979/ainovel-cli/internal/bootstrap"
	"github.com/voocel/agentcore"
	corecontext "github.com/voocel/agentcore/context"
)

func TestContextManagerFactoriesBuildConfiguredEngines(t *testing.T) {
	model := &plainContextModel{}
	engine := newContextManager(contextManagerConfig{Model: model, ContextWindow: 100000, ReserveTokens: 8000, Agent: "writer", CommitProjected: true, Summary: &corecontext.FullSummaryConfig{SystemPrompt: "summary"}, ToolMicrocompact: &corecontext.ToolResultMicrocompactConfig{KeepRecent: 2}, ExtraStrategies: []corecontext.Strategy{corecontext.NewToolResultMicrocompact(corecontext.ToolResultMicrocompactConfig{})}})
	if engine == nil || engine.ContextWindow() != 100000 {
		t.Fatalf("engine = %#v", engine)
	}
	role := newRoleContextManager(roleContextProfile{Agent: "editor", KeepRecentReads: 3}, model, 50000, "novel_context")
	if role == nil || role.ContextWindow() != 50000 {
		t.Fatalf("role engine = %#v", role)
	}
	contextRewriteCallback("writer")(corecontext.RewriteEvent{Reason: "test"})
	contextRewriteCallback("writer")(corecontext.RewriteEvent{Reason: "test", Info: &corecontext.SummaryInfo{MessagesBefore: 2, MessagesAfter: 1, CompactedCount: 1, KeptCount: 1}})
	if got := bootstrap.CompactReserveTokens(100000); got <= 0 {
		t.Fatal("reserve tokens should be positive")
	}
}

type plainContextModel struct{}

func (*plainContextModel) Generate(context.Context, []agentcore.Message, []agentcore.ToolSpec, ...agentcore.CallOption) (*agentcore.LLMResponse, error) {
	return nil, nil
}
func (*plainContextModel) GenerateStream(context.Context, []agentcore.Message, []agentcore.ToolSpec, ...agentcore.CallOption) (<-chan agentcore.StreamEvent, error) {
	return nil, nil
}
func (*plainContextModel) SupportsTools() bool { return false }

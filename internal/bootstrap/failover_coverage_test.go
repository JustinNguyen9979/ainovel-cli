package bootstrap

import (
	"context"
	"errors"
	"testing"

	"github.com/voocel/agentcore"
	"github.com/voocel/agentcore/llm"
)

type failoverCoverageModel struct {
	generateErr error
	streamErr   error
	message     agentcore.Message
	stream      []agentcore.StreamEvent
}

func (m *failoverCoverageModel) Generate(context.Context, []agentcore.Message, []agentcore.ToolSpec, ...agentcore.CallOption) (*agentcore.LLMResponse, error) {
	if m.generateErr != nil {
		return nil, m.generateErr
	}
	return &agentcore.LLMResponse{Message: m.message}, nil
}
func (m *failoverCoverageModel) GenerateStream(context.Context, []agentcore.Message, []agentcore.ToolSpec, ...agentcore.CallOption) (<-chan agentcore.StreamEvent, error) {
	if m.streamErr != nil {
		return nil, m.streamErr
	}
	ch := make(chan agentcore.StreamEvent, len(m.stream)+1)
	for _, ev := range m.stream {
		ch <- ev
	}
	close(ch)
	return ch, nil
}
func (m *failoverCoverageModel) SupportsTools() bool { return true }
func (m *failoverCoverageModel) Capabilities() llm.Capabilities {
	return llm.Capabilities{Structured: llm.StructuredCapabilities{JSONSchema: llm.SupportYes}}
}

func TestFailoverModelGenerateAndHelpers(t *testing.T) {
	primaryModel := &failoverCoverageModel{generateErr: agentcore.ErrProviderNetwork}
	fallbackModel := &failoverCoverageModel{message: agentcore.Message{Role: agentcore.RoleAssistant, Content: []agentcore.ContentBlock{agentcore.TextBlock("fallback")}}}
	primary := NewSwappableModel("primary", "p", primaryModel, nil)
	set := &ModelSet{models: map[string]*SwappableModel{}, fallbacks: map[string][]modelTarget{"writer": {{provider: "fallback", name: "f", model: fallbackModel}}}}
	var reports []FailoverEvent
	model := &failoverModel{role: "writer", primary: primary, set: set, report: func(ev FailoverEvent) { reports = append(reports, ev) }}
	resp, err := model.Generate(context.Background(), nil, nil)
	if err != nil || resp == nil || len(reports) != 1 || reports[0].ToProvider != "fallback" {
		t.Fatalf("fallback generate = %+v/%v reports=%+v", resp, err, reports)
	}
	if !model.SupportsTools() || model.ProviderName() != "primary" || model.Info().Name != "p" {
		t.Fatalf("model helpers unexpected: tools=%v provider=%q info=%+v", model.SupportsTools(), model.ProviderName(), model.Info())
	}
	if _, _, ok := model.pickFallback(modelTarget{}, agentcore.ErrProviderNetwork, false); ok {
		t.Fatal("nil current model should not fail over")
	}
	if _, _, ok := model.pickFallback(model.currentTarget(), context.Canceled, false); ok {
		t.Fatal("canceled request should not fail over")
	}
	if _, _, ok := model.pickFallback(model.currentTarget(), errors.New("ordinary"), false); ok {
		t.Fatal("non-eligible error should not fail over")
	}
	if _, _, ok := model.pickFallback(model.currentTarget(), agentcore.ErrProviderNetwork, true); !ok {
		t.Fatal("fallback with JSON schema capability should be eligible")
	}
	if requestsJSONSchema(nil) || supportsJSONSchema(modelTarget{model: fallbackModel}) == false {
		t.Fatal("schema helper mismatch")
	}
	if got := model.StructuredOutputFacts(); got.Info.Provider != "primary" || got.Info.Name != "p" {
		t.Fatalf("structured facts = %+v", got)
	}
}

func TestFailoverModelStreamAndStartAttempt(t *testing.T) {
	primaryModel := &failoverCoverageModel{streamErr: agentcore.ErrProviderRateLimit, generateErr: agentcore.ErrProviderRateLimit}
	fallbackModel := &failoverCoverageModel{stream: []agentcore.StreamEvent{{Type: agentcore.StreamEventTextDelta, Delta: "ok"}, {Type: agentcore.StreamEventDone}}}
	primary := NewSwappableModel("primary", "p", primaryModel, nil)
	set := &ModelSet{models: map[string]*SwappableModel{}, fallbacks: map[string][]modelTarget{"writer": {{provider: "fallback", name: "f", model: fallbackModel}}}}
	model := &failoverModel{role: "writer", primary: primary, set: set}
	ch, err := model.GenerateStream(context.Background(), nil, nil)
	if err != nil {
		t.Fatal(err)
	}
	var events []agentcore.StreamEvent
	for ev := range ch {
		events = append(events, ev)
	}
	if len(events) != 2 || events[0].Delta != "ok" {
		t.Fatalf("stream fallback events = %+v", events)
	}
	if _, resp, err := model.startAttempt(context.Background(), modelTarget{model: primaryModel}, nil, nil); err == nil || resp != nil {
		t.Fatal("failed stream and generate should return error")
	}
	if _, _, err := model.startAttempt(context.Background(), modelTarget{}, nil, nil); err == nil {
		t.Fatal("nil target should fail")
	}
	if _, resp, err := (&failoverModel{}).startAttempt(context.Background(), modelTarget{model: &failoverCoverageModel{streamErr: errors.New("stream"), message: agentcore.Message{}}}, nil, nil); err != nil || resp == nil {
		t.Fatalf("generate fallback from stream error = %+v/%v", resp, err)
	}
}

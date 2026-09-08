package host

import (
	"context"
	"errors"
	"strings"
	"testing"

	"github.com/JustinNguyen9979/ainovel-cli/internal/domain"
	"github.com/voocel/agentcore"
)

type cocreateStreamModel struct {
	streams [][]agentcore.StreamEvent
	calls   int
}

func (m *cocreateStreamModel) Generate(context.Context, []agentcore.Message, []agentcore.ToolSpec, ...agentcore.CallOption) (*agentcore.LLMResponse, error) {
	return nil, errors.New("unused")
}
func (m *cocreateStreamModel) GenerateStream(context.Context, []agentcore.Message, []agentcore.ToolSpec, ...agentcore.CallOption) (<-chan agentcore.StreamEvent, error) {
	if m.calls >= len(m.streams) {
		return nil, errors.New("no scripted stream")
	}
	events := m.streams[m.calls]
	m.calls++
	ch := make(chan agentcore.StreamEvent, len(events))
	for _, ev := range events {
		ch <- ev
	}
	close(ch)
	return ch, nil
}
func (*cocreateStreamModel) SupportsTools() bool { return false }

func TestCoCreateStreamParsesAndReportsProgress(t *testing.T) {
	model := &cocreateStreamModel{streams: [][]agentcore.StreamEvent{{
		{Type: agentcore.StreamEventThinkingDelta, Delta: "thinking"},
		{Type: agentcore.StreamEventTextDelta, Delta: "<reply>hello</reply><draft>## draft</draft><ready>true</ready>"},
		{Type: agentcore.StreamEventDone},
	}}}
	var progress []string
	reply, err := coCreateStream(context.Background(), model, nil, "system", []CoCreateMessage{{Role: "user", Content: "idea"}}, "high", func(kind, text string) { progress = append(progress, kind+":"+text) })
	if err != nil || reply.Message != "hello" || reply.Prompt != "## draft" || !reply.Ready {
		t.Fatalf("reply = %+v err=%v", reply, err)
	}
	if len(progress) != 2 || !strings.HasPrefix(progress[0], CoCreateProgressThinking) || !strings.HasPrefix(progress[1], CoCreateProgressReply) {
		t.Fatalf("progress = %#v", progress)
	}
}

func TestCoCreateStreamRetriesPartialAndRejectsInvalidInput(t *testing.T) {
	model := &cocreateStreamModel{streams: [][]agentcore.StreamEvent{
		{{Type: agentcore.StreamEventTextDelta, Delta: "partial"}},
		{{Type: agentcore.StreamEventTextDelta, Delta: "<reply>ok</reply>"}, {Type: agentcore.StreamEventDone}},
	}}
	reply, err := coCreateStream(context.Background(), model, nil, "system", []CoCreateMessage{{Role: "user", Content: "idea"}}, "", nil)
	if err != nil || reply.Message != "ok" || model.calls != 2 {
		t.Fatalf("retry reply=%+v err=%v calls=%d", reply, err, model.calls)
	}
	if _, err := coCreateStream(context.Background(), model, nil, "system", nil, "", nil); err == nil {
		t.Fatal("empty history should fail")
	}
	if _, err := coCreateStream(context.Background(), nil, nil, "system", []CoCreateMessage{{Role: "user", Content: "idea"}}, "", nil); err == nil {
		t.Fatal("nil model should fail")
	}
	if !isRetryableCoCreateStreamError(agentcore.ErrStreamPartial, false) || isRetryableCoCreateStreamError(context.Canceled, false) || isRetryableCoCreateStreamError(nil, false) || isRetryableCoCreateStreamError(agentcore.ErrStreamPartial, true) {
		t.Fatal("retry classification mismatch")
	}
	if errString(nil) != "" || errString(errors.New("bad")) != "bad" {
		t.Fatal("error string mismatch")
	}
	msg := assistantMsg("hello")
	if msg.Role != agentcore.RoleAssistant || msg.TextContent() != "hello" {
		t.Fatal("assistant message mismatch")
	}
	_ = domain.PhaseWriting
}

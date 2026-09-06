package llmretry

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/voocel/agentcore"
)

type retryableTestError struct {
	wait time.Duration
}

func (e retryableTestError) Error() string             { return "retryable" }
func (e retryableTestError) Retryable() bool           { return true }
func (e retryableTestError) RetryAfter() time.Duration { return e.wait }

type retryModel struct {
	fails int
}

func (m *retryModel) Generate(context.Context, []agentcore.Message, []agentcore.ToolSpec, ...agentcore.CallOption) (*agentcore.LLMResponse, error) {
	if m.fails > 0 {
		m.fails--
		return nil, retryableTestError{wait: time.Millisecond}
	}
	return &agentcore.LLMResponse{}, nil
}

func TestGenerateRetriesAndReports(t *testing.T) {
	model := &retryModel{fails: 2}
	var events []Event
	resp, err := Generate(context.Background(), model, Config{Agent: "writer", OnRetry: func(event Event) { events = append(events, event) }}, nil)
	if err != nil || resp == nil || len(events) != 2 {
		t.Fatalf("retry result = %+v, %v, events=%+v", resp, err, events)
	}
	if events[0].Attempt != 1 || events[1].Attempt != 2 || events[0].Delay != time.Millisecond {
		t.Fatalf("retry events = %+v", events)
	}
}

func TestGenerateStopsForNonRetryableAndCanceled(t *testing.T) {
	nonRetry := GeneratorFunc(func(context.Context, []agentcore.Message, []agentcore.ToolSpec, ...agentcore.CallOption) (*agentcore.LLMResponse, error) {
		return nil, errors.New("fatal")
	})
	if _, err := Generate(context.Background(), nonRetry, Config{}, nil); err == nil || err.Error() != "fatal" {
		t.Fatalf("non-retryable error = %v", err)
	}
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	canceled := GeneratorFunc(func(context.Context, []agentcore.Message, []agentcore.ToolSpec, ...agentcore.CallOption) (*agentcore.LLMResponse, error) {
		return nil, retryableTestError{}
	})
	if _, err := Generate(ctx, canceled, Config{}, nil); !errors.Is(err, context.Canceled) {
		t.Fatalf("canceled error = %v", err)
	}
}

type GeneratorFunc func(context.Context, []agentcore.Message, []agentcore.ToolSpec, ...agentcore.CallOption) (*agentcore.LLMResponse, error)

func (f GeneratorFunc) Generate(ctx context.Context, messages []agentcore.Message, tools []agentcore.ToolSpec, opts ...agentcore.CallOption) (*agentcore.LLMResponse, error) {
	return f(ctx, messages, tools, opts...)
}

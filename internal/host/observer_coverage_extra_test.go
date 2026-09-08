package host

import (
	"encoding/json"
	"strings"
	"testing"

	"github.com/voocel/agentcore"
)

func TestObserverThinkingAndContextProgressBranches(t *testing.T) {
	var events []Event
	o := testObserver(&events)
	o.emitD = func(delta string) { events = append(events, Event{Summary: delta}) }
	o.handleThinkingProgress(agentcore.Event{Progress: &agentcore.ProgressPayload{Agent: "", Thinking: "ignored"}})
	o.handleThinkingProgress(agentcore.Event{Progress: &agentcore.ProgressPayload{Agent: "writer", Thinking: "first"}})
	o.handleThinkingProgress(agentcore.Event{Progress: &agentcore.ProgressPayload{Agent: "writer", Thinking: "first more"}})
	o.handleThinkingProgress(agentcore.Event{Progress: &agentcore.ProgressPayload{Agent: "writer", Thinking: "first more"}})
	if len(events) != 3 || events[1].Summary != "first" || events[2].Summary != " more" {
		t.Fatalf("thinking deltas = %+v", events)
	}
	o.handleContextProgress(agentcore.Event{})
	o.handleContextProgress(agentcore.Event{Progress: &agentcore.ProgressPayload{Agent: "writer", Meta: json.RawMessage("{")}})
	o.handleContextProgress(agentcore.Event{Progress: &agentcore.ProgressPayload{Agent: "", Meta: json.RawMessage(`{"tokens":1}`)}})
	o.handleContextProgress(agentcore.Event{Progress: &agentcore.ProgressPayload{Agent: "writer", Meta: json.RawMessage(`{"tokens":100,"context_window":120,"percent":90,"scope":"projected","strategy":"full_summary"}`)}})
	if len(events) != 4 || o.agents["writer"].context.Tokens != 100 || events[3].Level != "warn" {
		t.Fatalf("context progress = events=%+v agents=%+v", events, o.agents)
	}
	o.handleContextProgress(agentcore.Event{Progress: &agentcore.ProgressPayload{Agent: "writer", Meta: json.RawMessage(`{"tokens":50,"context_window":120,"percent":40,"scope":"projected"}`)}})
	if o.agents["writer"].context.Percent != 40 {
		t.Fatal("normal context progress should update snapshot")
	}
}

func TestObserverDispatchAndToolBranches(t *testing.T) {
	var events []Event
	o := testObserver(&events)
	o.dispatchStart("writer", "写作任务\n详细", "原因")
	o.handleToolUpdate(agentcore.Event{Progress: &agentcore.ProgressPayload{Kind: agentcore.ProgressToolStart, Agent: "writer", Tool: "draft_chapter", Args: json.RawMessage(`{"chapter":2}`)}})
	o.handleToolUpdate(agentcore.Event{Progress: &agentcore.ProgressPayload{Kind: agentcore.ProgressToolEnd, Agent: "writer", Tool: "draft_chapter"}})
	o.handleToolUpdate(agentcore.Event{Progress: &agentcore.ProgressPayload{Kind: agentcore.ProgressToolError, Agent: "writer", Tool: "draft_chapter", Message: "failed"}})
	o.handleToolUpdate(agentcore.Event{Progress: &agentcore.ProgressPayload{Kind: agentcore.ProgressToolError, Agent: "editor", Tool: "save_review"}})
	if len(events) < 4 {
		t.Fatalf("observer events = %+v", events)
	}
	if got := dispatchSummary("", ""); got != "subagent" || dispatchSummary("writer", "\n details") != "writer" {
		t.Fatal("dispatch summary fallback mismatch")
	}
	if got := dispatchDetail("task", "reason"); !strings.Contains(got, "task") || !strings.Contains(got, "reason") {
		t.Fatal("dispatch detail missing fields")
	}
	if got := streamArgKey("a", "b"); got != "a\x00b" || streamedToolLabel("other", "x") != "" {
		t.Fatal("stream helper mismatch")
	}
	if got := errorKind(nil, "tool argument validation failed"); got != "tool_validation" {
		t.Fatalf("error kind = %q", got)
	}
}

package store

import (
	"encoding/json"
	"strings"
	"testing"

	"github.com/voocel/agentcore"
)

func TestSessionStoreLoggingAndCompaction(t *testing.T) {
	st := NewStore(t.TempDir())
	if err := st.Init(); err != nil {
		t.Fatal(err)
	}
	logger := st.Sessions.SubAgentLogger(func(agent string) (string, string) { return "provider", agent + "-model" })
	logger("writer", "写第 2 章", agentcore.UserMsg("用户要求"))
	usage := &agentcore.Usage{Input: 10, Output: 2, Provider: "provider", Model: "model"}
	msg := agentcore.Message{Role: agentcore.RoleAssistant, Usage: usage, Content: []agentcore.ContentBlock{agentcore.TextBlock("回复")}}
	logger("writer", "写第 2 章", msg)
	if files, err := st.Sessions.io.ReadFile("meta/sessions/agents/writer-ch02.jsonl"); err != nil || !strings.Contains(string(files), "provider") {
		t.Fatalf("session log = %q/%v", files, err)
	}
	if err := st.Sessions.Log("meta/sessions/plain.jsonl", msg); err != nil {
		t.Fatal(err)
	}
	if err := st.Sessions.LogCoCreate(map[string]string{"message": "hello"}); err != nil {
		t.Fatal(err)
	}
	if extractChapter("写第 12 章") != "ch12" || extractChapter("no chapter") != "" || extractChapter("第0章") != "" {
		t.Fatal("chapter extraction mismatch")
	}
	if usageMeta(nil) != nil || usageMeta(&agentcore.Usage{}) != nil || usageMeta(usage) == nil {
		t.Fatal("usage meta mismatch")
	}
	large := strings.Repeat("正文", 3000)
	if got := compactText(agentcore.RoleTool, "read_chapter", large); !strings.Contains(got, "session_compact") || !strings.Contains(got, "字") {
		t.Fatalf("read chapter compaction = %q", got)
	}
	if got := compactText(agentcore.RoleTool, "novel_context", large); !strings.Contains(got, "novel_context") {
		t.Fatalf("context compaction = %q", got)
	}
	if got := compactText(agentcore.RoleTool, "other", strings.Repeat("x", 9000)); !strings.Contains(got, "other") {
		t.Fatalf("generic compaction = %q", got)
	}
	if compactText(agentcore.RoleAssistant, "read_chapter", large) != large {
		t.Fatal("assistant text should not compact")
	}
	call := &agentcore.ToolCall{Name: "draft_chapter", Args: json.RawMessage(`{"chapter":3,"content":"` + large + `"}`)}
	if got := compactToolCall(call); got == call || !strings.Contains(string(got.Args), "session_compact") {
		t.Fatalf("draft tool call compaction = %#v", got)
	}
	foundation := &agentcore.ToolCall{Name: "save_foundation", Args: json.RawMessage(`{"type":"premise","content":"` + large + `"}`)}
	if got := compactToolCall(foundation); got == foundation || !strings.Contains(string(got.Args), "session_compact") {
		t.Fatalf("foundation tool call compaction = %#v", got)
	}
	if extractJSONField(`{"x":"value"}`, "x") != "value" || extractJSONField(`{"x":12}`, "x") != "12" || extractJSONField("{", "x") != "" {
		t.Fatal("JSON field extraction mismatch")
	}
	if extractJSONFieldInt(json.RawMessage(`{"chapter":3}`), "chapter") != 3 || extractJSONFieldInt(json.RawMessage(`{"chapter":"x"}`), "chapter") != 0 {
		t.Fatal("JSON int extraction mismatch")
	}
	if !IsCompacted("[session_compact: value]") || IsCompacted("normal") {
		t.Fatal("compaction marker mismatch")
	}
}

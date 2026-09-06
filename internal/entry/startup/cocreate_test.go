package startup

import (
	"strings"
	"testing"

	"github.com/JustinNguyen9979/ainovel-cli/internal/host"
)

func TestCoCreateSessionLifecycle(t *testing.T) {
	s := NewCoCreateSession("  initial idea  ")
	if s.InitialInput() != "initial idea" || len(s.History()) != 1 || s.CanStart() {
		t.Fatalf("initial session = %+v", s.History())
	}
	s.ApplyDelta(host.CoCreateProgressThinking, " thinking ")
	s.ApplyDelta(host.CoCreateProgressReply, " reply ")
	if s.StreamThinking() != "thinking" || s.StreamReply() != "reply" {
		t.Fatalf("stream state = %q/%q", s.StreamThinking(), s.StreamReply())
	}
	s.ApplyReply(host.CoCreateReply{Raw: "  assistant raw  ", Message: "fallback", Prompt: "  draft prompt  ", Ready: true, Suggestions: []string{"one", "two"}})
	if s.DraftPrompt() != "draft prompt" || !s.Ready() || !s.CanStart() || len(s.History()) != 2 {
		t.Fatalf("reply state = %+v", s.History())
	}
	if len(s.Suggestions()) != 2 || s.StreamReply() != "" || s.StreamThinking() != "" {
		t.Fatalf("reply cleanup = %+v/%q/%q", s.Suggestions(), s.StreamReply(), s.StreamThinking())
	}
	s.AppendUser("  next idea  ")
	if len(s.History()) != 3 || s.History()[2].Content != "next idea" || len(s.Suggestions()) != 0 {
		t.Fatalf("user append = %+v/%v", s.History(), s.Suggestions())
	}
	if prompt, err := s.BuildPrompt(); err != nil || prompt != "draft prompt" {
		t.Fatalf("build prompt = %q, %v", prompt, err)
	}
}

func TestCoCreateSessionReplyFallbackAndValidation(t *testing.T) {
	s := NewCoCreateSession("initial")
	s.ApplyReply(host.CoCreateReply{Message: "message", Prompt: "draft"})
	s.ApplyReply(host.CoCreateReply{Message: "next"})
	if s.DraftPrompt() != "draft" || len(s.History()) != 3 {
		t.Fatalf("empty prompt should preserve draft = %+v", s.History())
	}
	before := len(s.History())
	s.AppendUser("  ")
	if len(s.History()) != before {
		t.Fatal("blank user text should be ignored")
	}
	var nilSession *CoCreateSession
	_, nilErr := nilSession.BuildPrompt()
	if nilSession.History() != nil || nilErr == nil || nilSession.CanStart() {
		t.Fatal("nil session behavior is wrong")
	}
	if _, err := (&CoCreateSession{}).BuildPrompt(); err == nil || !strings.Contains(err.Error(), "draft prompt") {
		t.Fatalf("empty draft error = %v", err)
	}
}

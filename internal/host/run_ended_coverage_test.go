package host

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/JustinNguyen9979/ainovel-cli/internal/domain"
	storepkg "github.com/JustinNguyen9979/ainovel-cli/internal/store"
)

func newRunEndedHost(t *testing.T, progress *domain.Progress, book *domain.BookMetadata) *Host {
	t.Helper()
	st := storepkg.NewStore(t.TempDir())
	if err := st.Init(); err != nil {
		t.Fatal(err)
	}
	if progress != nil {
		if err := st.Progress.Save(progress); err != nil {
			t.Fatal(err)
		}
	}
	if book != nil {
		if err := st.Book.Save(*book); err != nil {
			t.Fatal(err)
		}
	}
	return &Host{
		store:     st,
		observer:  &observer{agents: map[string]*agentState{}},
		usage:     NewUsageTracker(nil, st),
		events:    make(chan Event, 8),
		done:      make(chan struct{}, 2),
		lifecycle: lifecycleRunning,
	}
}

func TestRunEndedCompletionAndStopBranches(t *testing.T) {
	complete := newRunEndedHost(t, &domain.Progress{Phase: domain.PhaseComplete, CompletedChapters: []int{1, 2}, TotalWordCount: 500}, &domain.BookMetadata{Title: "完本", Synopsis: "简介"})
	complete.runEnded()
	if complete.lifecycle != lifecycleCompleted {
		t.Fatalf("complete lifecycle = %q", complete.lifecycle)
	}
	select {
	case ev := <-complete.events:
		if ev.Level != "success" || !strings.Contains(ev.Summary, "完本") {
			t.Fatalf("completion event = %+v", ev)
		}
	default:
		t.Fatal("completion event missing")
	}
	select {
	case <-complete.done:
	default:
		t.Fatal("completion done signal missing")
	}
	if got := complete.runEndBody("完本", "完成"); !strings.Contains(got, "《完本》完成") {
		t.Fatalf("run end body = %q", got)
	}

	stopped := newRunEndedHost(t, &domain.Progress{Phase: domain.PhaseWriting, CompletedChapters: []int{1}}, &domain.BookMetadata{Title: "进行中", Synopsis: "简介"})
	stopped.runEnded()
	if stopped.lifecycle != lifecycleIdle {
		t.Fatalf("stopped lifecycle = %q", stopped.lifecycle)
	}
	select {
	case ev := <-stopped.events:
		if ev.Level != "warn" || !strings.Contains(ev.Summary, "已完成 1 章") {
			t.Fatalf("stop event = %+v", ev)
		}
	default:
		t.Fatal("stop event missing")
	}

	idle := newRunEndedHost(t, &domain.Progress{Phase: domain.PhaseWriting}, &domain.BookMetadata{Title: "idle", Synopsis: "简介"})
	idle.lifecycle = lifecycleIdle
	idle.runEnded()
	if len(idle.events) != 0 {
		t.Fatalf("idle host should not emit stop event: %+v", idle.events)
	}
}

func TestRunEndedErrorBranches(t *testing.T) {
	dir := t.TempDir()
	st := storepkg.NewStore(dir)
	if err := st.Init(); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "meta", "progress.json"), []byte("{"), 0o600); err != nil {
		t.Fatal(err)
	}
	h := &Host{store: st, observer: &observer{agents: map[string]*agentState{}}, events: make(chan Event, 4), done: make(chan struct{}, 1), lifecycle: lifecycleRunning}
	h.runEnded()
	if h.lifecycle != lifecycleIdle || len(h.events) != 1 {
		t.Fatalf("progress error branch: lifecycle=%q events=%+v", h.lifecycle, h.events)
	}
	if ev := <-h.events; ev.Level != "error" {
		t.Fatalf("progress error event = %+v", ev)
	}

	st = storepkg.NewStore(t.TempDir())
	if err := st.Init(); err != nil {
		t.Fatal(err)
	}
	if err := st.Progress.Save(&domain.Progress{Phase: domain.PhaseWriting}); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(st.Dir(), "meta", "book.json"), []byte("{"), 0o600); err != nil {
		t.Fatal(err)
	}
	h = &Host{store: st, observer: &observer{agents: map[string]*agentState{}}, events: make(chan Event, 4), done: make(chan struct{}, 1), lifecycle: lifecycleRunning}
	h.runEnded()
	if h.lifecycle != lifecycleIdle || len(h.events) != 1 {
		t.Fatalf("book error branch: lifecycle=%q events=%+v", h.lifecycle, h.events)
	}
	if ev := <-h.events; ev.Level != "error" {
		t.Fatalf("book error event = %+v", ev)
	}

	h = newRunEndedHost(t, &domain.Progress{Phase: domain.PhaseComplete}, nil)
	h.runEnded()
	if h.lifecycle != lifecycleIdle || len(h.events) != 1 {
		t.Fatalf("missing book completion branch: lifecycle=%q events=%+v", h.lifecycle, h.events)
	}
	if ev := <-h.events; !strings.Contains(ev.Summary, "作品信息不存在") {
		t.Fatalf("missing book completion event = %+v", ev)
	}
}

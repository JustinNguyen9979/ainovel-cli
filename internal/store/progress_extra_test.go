package store

import (
	"testing"

	"github.com/JustinNguyen9979/ainovel-cli/internal/domain"
)

func TestProgressStoreLifecycleAndValidation(t *testing.T) {
	s := NewStore(t.TempDir())
	if err := s.Init(); err != nil {
		t.Fatal(err)
	}
	if err := s.Progress.Init(10); err != nil {
		t.Fatal(err)
	}
	if err := s.Progress.UpdatePhase(domain.PhaseWriting); err != nil {
		t.Fatal(err)
	}
	if err := s.Progress.StartChapter(0); err == nil {
		t.Fatal("invalid chapter should fail")
	}
	if err := s.Progress.StartChapter(2); err != nil {
		t.Fatal(err)
	}
	if err := s.Progress.StartChapter(2); err != nil {
		t.Fatal(err)
	}
	if err := s.Progress.MarkChapterComplete(2, 100, "mystery", "quest"); err != nil {
		t.Fatal(err)
	}
	if err := s.Progress.MarkChapterComplete(2, 120, "choice", "fire"); err != nil {
		t.Fatal(err)
	}
	p, err := s.Progress.Load()
	if err != nil || p.TotalWordCount != 120 || p.CurrentChapter != 3 || len(p.CompletedChapters) != 1 || p.HookHistory[1] != "choice" {
		t.Fatalf("progress after completion = %+v/%v", p, err)
	}
	if ok, err := s.Progress.IsChapterCompleted(2); err != nil || !ok {
		t.Fatalf("completed lookup = %v/%v", ok, err)
	}
	if err := s.Progress.SetPendingRewrites([]int{2, 2}, "rewrite"); err != nil {
		t.Fatal(err)
	}
	if err := s.Progress.ValidatePendingRewrites([]int{9}); err == nil {
		t.Fatal("unfinished chapter should not enter rewrite queue")
	}
	if _, err := s.Progress.ApplyReviewOutcome(domain.FlowRewriting, []int{2}, "review"); err != nil {
		t.Fatal(err)
	}
	if err := s.Progress.CompleteRewrite(2); err != nil {
		t.Fatal(err)
	}
	if err := s.Progress.ClearPendingRewrites(); err != nil {
		t.Fatal(err)
	}
	if err := s.Progress.MarkComplete(); err != nil {
		t.Fatal(err)
	}
	if err := s.Progress.ReopenContinue(); err != nil {
		t.Fatal(err)
	}
}

package store

import (
	"strings"
	"testing"

	"github.com/JustinNguyen9979/ainovel-cli/internal/domain"
	"github.com/JustinNguyen9979/ainovel-cli/internal/errs"
)

func TestProgressTransitionsAndRewriteLifecycle(t *testing.T) {
	st := NewStore(t.TempDir())
	if err := st.Init(); err != nil {
		t.Fatal(err)
	}
	if err := st.Progress.Init(4); err != nil {
		t.Fatal(err)
	}
	if err := st.Progress.SetTotalChapters(5); err != nil {
		t.Fatal(err)
	}
	if err := st.Progress.UpdatePhase(domain.PhaseWriting); err != nil {
		t.Fatal(err)
	}
	if err := st.Progress.StartChapter(1); err != nil {
		t.Fatal(err)
	}
	if err := st.Progress.MarkChapterComplete(1, 100, "crisis", "quest"); err != nil {
		t.Fatal(err)
	}
	if err := st.Progress.MarkChapterComplete(1, 120, "", ""); err != nil {
		t.Fatal(err)
	}
	p, err := st.Progress.Load()
	if err != nil || p.TotalWordCount != 120 || len(p.StrandHistory) != 1 || p.StrandHistory[0] != "quest" || len(p.HookHistory) != 1 || p.HookHistory[0] != "crisis" {
		t.Fatalf("progress after commit = %+v/%v", p, err)
	}
	if ok, err := st.Progress.IsChapterCompleted(1); err != nil || !ok {
		t.Fatalf("completed check = %v/%v", ok, err)
	}
	if err := st.Progress.SetPendingRewrites([]int{1, 1}, "polish"); err != nil {
		t.Fatal(err)
	}
	if err := st.Progress.SetFlow(domain.FlowPolishing); err != nil {
		t.Fatal(err)
	}
	if err := st.Progress.ValidateChapterWork(1); err != nil {
		t.Fatal(err)
	}
	if err := st.Progress.ValidateChapterWork(2); err == nil || !strings.Contains(err.Error(), "不在待打磨队列") {
		t.Fatalf("non-queued chapter error = %v", err)
	}
	if err := st.Progress.CompleteRewrite(1); err != nil {
		t.Fatal(err)
	}
	p, _ = st.Progress.Load()
	if len(p.PendingRewrites) != 0 || p.Flow != domain.FlowWriting {
		t.Fatalf("rewrite completion = %+v", p)
	}
	if _, err := st.Progress.ApplyReviewOutcome(domain.FlowWriting, []int{1}, "bad"); err == nil || !errorsIs(err, errs.ErrToolConflict) {
		t.Fatalf("invalid review outcome = %v", err)
	}
	if err := st.Progress.SetPendingRewrites([]int{99}, "bad"); err == nil {
		t.Fatal("unfinished chapter should not enter rewrite queue")
	}
	if err := st.Progress.ClearInProgress(); err != nil {
		t.Fatal(err)
	}
	if err := st.Progress.UpdateVolumeArc(2, 3); err != nil {
		t.Fatal(err)
	}
	if err := st.Progress.SetLayered(true); err != nil {
		t.Fatal(err)
	}
	p, _ = st.Progress.Load()
	if p.CurrentVolume != 2 || p.CurrentArc != 3 || !p.Layered {
		t.Fatalf("progress flags = %+v", p)
	}
}

func TestProgressReopenAndComplete(t *testing.T) {
	st := NewStore(t.TempDir())
	if err := st.Init(); err != nil {
		t.Fatal(err)
	}
	if err := st.Progress.Save(&domain.Progress{Phase: domain.PhaseComplete, CompletedChapters: []int{1, 2}}); err != nil {
		t.Fatal(err)
	}
	if err := st.Progress.Reopen([]int{2, 2}, "重新打磨"); err != nil {
		t.Fatal(err)
	}
	p, _ := st.Progress.Load()
	if p.Phase != domain.PhaseWriting || p.Flow != domain.FlowRewriting || len(p.PendingRewrites) != 1 || !p.ReopenedFromComplete {
		t.Fatalf("reopen = %+v", p)
	}
	if err := st.Progress.ClearPendingRewrites(); err != nil {
		t.Fatal(err)
	}
	if err := st.Progress.MarkComplete(); err != nil {
		t.Fatal(err)
	}
	if err := st.Progress.ReopenContinue(); err != nil {
		t.Fatal(err)
	}
	p, _ = st.Progress.Load()
	if p.Phase != domain.PhaseWriting || p.ReopenCount != 1 || p.ReopenedFromComplete {
		t.Fatalf("reopen continue = %+v", p)
	}
}

func errorsIs(err, target error) bool {
	return strings.Contains(err.Error(), target.Error())
}

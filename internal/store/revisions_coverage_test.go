package store

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/JustinNguyen9979/ainovel-cli/internal/domain"
)

func TestStoreReviseOutlineCoversFlatAndLayeredGuards(t *testing.T) {
	flat := NewStore(t.TempDir())
	if err := flat.Init(); err != nil {
		t.Fatal(err)
	}
	if _, err := flat.ReviseOutline(1, nil); err == nil {
		t.Fatal("uninitialized progress should fail")
	}
	if err := flat.Progress.Init(3); err != nil {
		t.Fatal(err)
	}
	if err := flat.Progress.UpdatePhase(domain.PhaseOutline); err != nil {
		t.Fatal(err)
	}
	if err := flat.Outline.SaveOutline([]domain.OutlineEntry{{Title: "一"}, {Title: "二"}}); err != nil {
		t.Fatal(err)
	}
	if _, err := flat.ReviseOutline(0, nil); err == nil {
		t.Fatal("zero chapter should fail")
	}
	if _, err := flat.ReviseOutline(4, nil); err == nil {
		t.Fatal("past outline should fail")
	}
	if got, err := flat.ReviseOutline(2, []domain.OutlineEntry{{Title: "新二"}, {Title: "新三"}}); err != nil || got != 3 {
		t.Fatalf("flat revise = %d/%v", got, err)
	}
	if err := flat.Progress.MarkComplete(); err != nil {
		t.Fatal(err)
	}
	if _, err := flat.ReviseOutline(2, nil); err == nil {
		t.Fatal("completed book should reject revision")
	}

	layered := setupLayered(t, []domain.VolumeOutline{{Index: 1, Arcs: []domain.ArcOutline{{Index: 1, Chapters: []domain.OutlineEntry{{Title: "一"}, {Title: "二"}}}}}})
	if err := layered.Progress.UpdatePhase(domain.PhaseWriting); err != nil {
		t.Fatal(err)
	}
	if got, err := layered.ReviseOutline(2, []domain.OutlineEntry{{Title: "新二"}}); err != nil || got != 2 {
		t.Fatalf("layered revise = %d/%v", got, err)
	}
	if _, err := layered.ReviseOutline(9, []domain.OutlineEntry{{Title: "越界"}}); err == nil {
		t.Fatal("layered out-of-range revision should fail")
	}
}

func TestRevisionStorePendingLifecycle(t *testing.T) {
	st := NewStore(t.TempDir())
	if err := st.Init(); err != nil {
		t.Fatal(err)
	}
	if pending, err := st.Revisions.LoadPending(); err != nil || pending != nil {
		t.Fatalf("empty pending = %#v/%v", pending, err)
	}
	pending := domain.PendingRevision{Stage: domain.RevisionStagePrepared, Items: []domain.PendingRevisionItem{{Chapter: 2}}}
	if err := st.Revisions.SavePending(pending); err != nil {
		t.Fatal(err)
	}
	if got, err := st.Revisions.LoadPending(); err != nil || got == nil || len(got.Items) != 1 {
		t.Fatalf("saved pending = %#v/%v", got, err)
	}
	if err := st.Revisions.ClearPending(); err != nil {
		t.Fatal(err)
	}
	if got, err := st.Revisions.LoadPending(); err != nil || got != nil {
		t.Fatalf("cleared pending = %#v/%v", got, err)
	}
}

func TestRevisionStoreInvalidatesLayeredAggregates(t *testing.T) {
	st := NewStore(t.TempDir())
	if err := st.Init(); err != nil {
		t.Fatal(err)
	}
	if err := st.Outline.SaveLayeredOutline([]domain.VolumeOutline{{Index: 1, Arcs: []domain.ArcOutline{{Index: 1, Chapters: []domain.OutlineEntry{{Title: "one"}, {Title: "two"}}}}}}); err != nil {
		t.Fatal(err)
	}
	if err := st.Summaries.SaveArcSummary(domain.ArcSummary{Volume: 1, Arc: 1}); err != nil {
		t.Fatal(err)
	}
	if err := st.Summaries.SaveVolumeSummary(domain.VolumeSummary{Volume: 1}); err != nil {
		t.Fatal(err)
	}
	if err := st.Characters.SaveSnapshots(1, 1, []domain.CharacterSnapshot{{Volume: 1, Arc: 1, Name: "主角"}}); err != nil {
		t.Fatal(err)
	}
	if err := st.World.SaveStyleRules(domain.WritingStyleRules{Volume: 1, Arc: 1}); err != nil {
		t.Fatal(err)
	}
	if err := st.World.SaveReview(domain.ReviewEntry{Chapter: 2, Scope: "chapter"}); err != nil {
		t.Fatal(err)
	}
	if err := st.World.SaveReview(domain.ReviewEntry{Chapter: 1, Scope: "global"}); err != nil {
		t.Fatal(err)
	}
	if err := st.InvalidateChapterAggregates(2); err != nil {
		t.Fatal(err)
	}
	if sum, _ := st.Summaries.LoadArcSummary(1, 1); sum != nil {
		t.Fatal("arc summary should be invalidated")
	}
	if sum, _ := st.Summaries.LoadVolumeSummary(1); sum != nil {
		t.Fatal("volume summary should be invalidated")
	}
	if snap, _ := st.Characters.LoadSnapshots(1, 1); len(snap) != 0 {
		t.Fatal("character snapshots should be invalidated")
	}
	if style, _ := st.World.LoadStyleRules(); style != nil {
		t.Fatal("style rules should be invalidated")
	}
	if err := st.InvalidateChapterAggregates(0); err == nil {
		t.Fatal("invalid chapter should fail")
	}
}

func TestRevisionInvalidationPreservesEarlierReviewAndRejectsUnknownStyleArc(t *testing.T) {
	dir := t.TempDir()
	st := NewStore(dir)
	if err := st.Init(); err != nil {
		t.Fatal(err)
	}
	if err := st.Outline.SaveLayeredOutline([]domain.VolumeOutline{{Index: 1, Arcs: []domain.ArcOutline{{Index: 1, Chapters: []domain.OutlineEntry{{Title: "one"}, {Title: "two"}}}}}}); err != nil {
		t.Fatal(err)
	}
	if err := st.World.SaveReview(domain.ReviewEntry{Chapter: 1, Scope: "chapter"}); err != nil {
		t.Fatal(err)
	}
	if err := st.World.SaveReview(domain.ReviewEntry{Chapter: 2, Scope: "chapter"}); err != nil {
		t.Fatal(err)
	}
	if err := st.InvalidateChapterAggregates(2); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(filepath.Join(dir, "reviews", "01.json")); err != nil {
		t.Fatalf("review before invalidation point should remain: %v", err)
	}
	if _, err := os.Stat(filepath.Join(dir, "reviews", "02.json")); !os.IsNotExist(err) {
		t.Fatalf("review at invalidation point should be removed: %v", err)
	}
	if err := st.World.SaveStyleRules(domain.WritingStyleRules{Volume: 9, Arc: 9}); err != nil {
		t.Fatal(err)
	}
	if err := st.InvalidateChapterAggregates(1); err == nil || !strings.Contains(err.Error(), "未知弧 V9A9") {
		t.Fatalf("unknown style arc should be rejected, got %v", err)
	}
}

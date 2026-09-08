package store

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/JustinNguyen9979/ainovel-cli/internal/domain"
	"github.com/JustinNguyen9979/ainovel-cli/internal/rules"
)

func TestCharacterSnapshotsAndCheckpointArtifacts(t *testing.T) {
	dir := t.TempDir()
	st := NewStore(dir)
	if err := st.Init(); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "meta", "book.json"), []byte(`{"title":"测试"}`), 0o644); err != nil {
		t.Fatal(err)
	}
	if got, err := st.Characters.LoadLatestSnapshots(); err != nil || got != nil {
		t.Fatalf("empty snapshots = %#v/%v", got, err)
	}
	if err := st.Outline.SaveLayeredOutline([]domain.VolumeOutline{{
		Index: 1,
		Arcs: []domain.ArcOutline{
			{Index: 1, Chapters: []domain.OutlineEntry{{Title: "一"}}},
			{Index: 2, Chapters: []domain.OutlineEntry{{Title: "二"}}},
		},
	}}); err != nil {
		t.Fatal(err)
	}
	want := []domain.CharacterSnapshot{{Volume: 1, Arc: 2, Name: "主角"}}
	if err := st.Characters.SaveSnapshots(1, 2, want); err != nil {
		t.Fatal(err)
	}
	if got, err := st.Characters.LoadLatestSnapshots(); err != nil || len(got) != 1 || got[0].Name != "主角" {
		t.Fatalf("latest snapshots = %#v/%v", got, err)
	}
	if _, err := st.Checkpoints.AppendArtifact(domain.GlobalScope(), "book", "meta/book.json"); err != nil {
		t.Fatal(err)
	}
	if got := st.Checkpoints.LatestByStep(domain.GlobalScope(), "book"); got == nil || got.Artifact != "meta/book.json" {
		t.Fatalf("artifact checkpoint = %+v", got)
	}
	if err := st.World.SaveStateChanges([]domain.StateChange{{Chapter: 2, Entity: "主角", Field: "mood", NewValue: "紧张"}}); err != nil {
		t.Fatal(err)
	}
	if got, err := st.World.LoadStateChanges(); err != nil || len(got) != 1 || got[0].NewValue != "紧张" {
		t.Fatalf("state changes = %#v/%v", got, err)
	}
}

func TestWorldReviewAndViolationHelpers(t *testing.T) {
	st := NewStore(t.TempDir())
	if err := st.Init(); err != nil {
		t.Fatal(err)
	}
	if ok, err := st.World.HasArcReview(2); err != nil || ok {
		t.Fatalf("missing arc review = %v/%v", ok, err)
	}
	if ok, err := st.World.HasGlobalReview(2); err != nil || ok {
		t.Fatalf("missing global review = %v/%v", ok, err)
	}
	if err := st.World.SaveReview(domain.ReviewEntry{Chapter: 2, Scope: "arc", Verdict: "accept"}); err != nil {
		t.Fatal(err)
	}
	if err := st.World.SaveReview(domain.ReviewEntry{Chapter: 3, Scope: "global", Verdict: "polish"}); err != nil {
		t.Fatal(err)
	}
	if ok, err := st.World.HasArcReview(2); err != nil || !ok {
		t.Fatalf("saved arc review = %v/%v", ok, err)
	}
	if ok, err := st.World.HasGlobalReview(3); err != nil || !ok {
		t.Fatalf("saved global review = %v/%v", ok, err)
	}
	if got, err := st.World.LoadGlobalReview(3); err != nil || got == nil || got.Verdict != "polish" {
		t.Fatalf("global review = %+v/%v", got, err)
	}
	if err := st.World.SaveReview(domain.ReviewEntry{Chapter: 4, Scope: "arc", Verdict: "rewrite", AffectedChapters: []int{2}}); err != nil {
		t.Fatal(err)
	}
	if got, err := st.World.LoadReviewsAffectingChapter(2); err != nil || len(got) != 1 || got[0].Chapter != 4 {
		t.Fatalf("affecting reviews = %+v/%v", got, err)
	}
	violations := []rules.Violation{{Rule: "fatigue_words", Target: "不禁", Actual: 3, Severity: rules.SeverityWarning}}
	if err := st.World.SaveRuleViolations(2, violations); err != nil {
		t.Fatal(err)
	}
	if got := st.World.LoadRuleViolations(2); len(got) != 1 || got[0].Actual != float64(3) {
		t.Fatalf("violations = %+v", got)
	}
	if got := st.World.LoadRuleViolations(9); got != nil {
		t.Fatalf("unknown chapter violations = %+v", got)
	}
	if pairKey("b", "a") != "a|b" || pairKey("a", "b") != "a|b" {
		t.Fatal("pair key should be order independent")
	}
	line := renderTimeline([]domain.TimelineEvent{{Chapter: 1, Time: "day", Event: "event", Characters: []string{"A"}}})
	if !strings.Contains(line, "event") {
		t.Fatal("timeline rendering missing event")
	}
}

func TestSignalClearOperations(t *testing.T) {
	st := NewStore(t.TempDir())
	if err := st.Init(); err != nil {
		t.Fatal(err)
	}
	if err := st.Signals.SaveLastCommit(domain.CommitResult{Chapter: 1}); err != nil {
		t.Fatal(err)
	}
	if err := st.Signals.ClearLastCommit(); err != nil {
		t.Fatal(err)
	}
	if got, err := st.Signals.LoadLastCommit(); err != nil || got != nil {
		t.Fatalf("cleared commit = %#v/%v", got, err)
	}
	if err := st.Signals.SaveLastReview(domain.ReviewEntry{Chapter: 1, Scope: "chapter"}); err != nil {
		t.Fatal(err)
	}
	if err := st.Signals.ClearLastReview(); err != nil {
		t.Fatal(err)
	}
	if got, err := st.Signals.LoadLastReviewSignal(); err != nil || got != nil {
		t.Fatalf("cleared review = %#v/%v", got, err)
	}
}

package domain

import (
	"strings"
	"testing"
)

func TestChapterHelpers(t *testing.T) {
	for _, tc := range []struct {
		completed int
		want      bool
	}{
		{0, false}, {4, false}, {5, true}, {10, true}, {11, false},
	} {
		got, _ := ShouldReview(tc.completed)
		if got != tc.want {
			t.Fatalf("ShouldReview(%d) = %v", tc.completed, got)
		}
	}
	if got, _ := ShouldArcReview(false, true, 2, 3); !got {
		t.Fatal("volume end should trigger arc review")
	}
	if got, _ := ShouldArcReview(true, false, 2, 3); !got {
		t.Fatal("arc end should trigger arc review")
	}
	if got, _ := ShouldArcReview(false, false, 2, 3); got {
		t.Fatal("non-boundary should not trigger review")
	}
	if WordCount("你好世界") != 4 {
		t.Fatal("WordCount must count runes")
	}
}

func TestCheckpointScopes(t *testing.T) {
	cases := []struct {
		scope Scope
		text  string
	}{
		{ChapterScope(3), "chapter:3"},
		{ArcScope(2, 4), "arc:v2a4"},
		{VolumeScope(2), "volume:2"},
		{GlobalScope(), "global"},
	}
	for _, tc := range cases {
		if tc.scope.String() != tc.text || !tc.scope.Matches(tc.scope) {
			t.Fatalf("scope = %q, want %q", tc.scope.String(), tc.text)
		}
	}
	if ChapterScope(1).Matches(ChapterScope(2)) || ArcScope(1, 1).Matches(VolumeScope(1)) || VolumeScope(1).Matches(VolumeScope(2)) {
		t.Fatal("different scopes should not match")
	}
}

func TestStoryHelpers(t *testing.T) {
	book := BookMetadata{"  title ", " synopsis "}
	if got := book.Normalized(); got.Title != "title" || got.Synopsis != "synopsis" {
		t.Fatalf("normalized book = %+v", got)
	}
	for _, tc := range []struct {
		book BookMetadata
		bad  string
	}{
		{BookMetadata{}, "title"},
		{BookMetadata{Title: "title"}, "synopsis"},
	} {
		if err := tc.book.Validate(); err == nil || !strings.Contains(err.Error(), tc.bad) {
			t.Fatalf("Validate(%+v) = %v", tc.book, err)
		}
	}
	volumes := []VolumeOutline{{Index: 1, Final: false, Arcs: []ArcOutline{{Chapters: []OutlineEntry{{Title: "one"}}, EstimatedChapters: 7}}}, {Index: 2, Final: true, Arcs: []ArcOutline{{Chapters: []OutlineEntry{{Title: "two"}}}}}}
	if FinaleVolume(volumes) != 2 || EstimatedChapterCapacity(volumes) != 2 {
		t.Fatal("story capacity/finale calculation is wrong")
	}
	flat := FlattenOutline(volumes)
	if len(flat) != 2 || flat[0].Chapter != 1 || flat[1].Chapter != 2 {
		t.Fatalf("flattened outline = %+v", flat)
	}
}

func TestProgressAndPolicies(t *testing.T) {
	for _, tc := range []struct {
		phase Phase
		ch    int
		want  bool
	}{
		{PhaseWriting, 1, true}, {PhaseWriting, 0, false}, {PhaseOutline, 1, false},
	} {
		if got := (&Progress{Phase: tc.phase, CurrentChapter: tc.ch}).IsResumable(); got != tc.want {
			t.Fatalf("IsResumable = %v", got)
		}
	}
	progress := &Progress{CompletedChapters: []int{4, 2, 9}}
	if progress.LatestCompleted() != 9 || progress.NextChapter() != 10 {
		t.Fatal("progress chapter helpers are wrong")
	}
	if NewContextProfile(15).SummaryWindow != 10 || NewContextProfile(50).TimelineWindow != 8 || !NewContextProfile(51).Layered {
		t.Fatal("context profile boundaries are wrong")
	}
	policy := NewChapterMemoryPolicy(&Progress{TotalChapters: 40, Flow: FlowReviewing, Layered: true, CompletedChapters: []int{1, 2, 3, 4, 5, 6}}, NewContextProfile(60), true)
	if !policy.RelatedLookup || !policy.HandoffPreferred || !policy.LayeredSummaries || policy.ReadOnlyThreshold != 4 {
		t.Fatalf("memory policy = %+v", policy)
	}
}

func TestAdvanceValidation(t *testing.T) {
	if !ChapterAdvanceAuto.Valid() || !ChapterAdvanceReview.Valid() || ChapterAdvanceMode("bad").Valid() {
		t.Fatal("advance mode validity is wrong")
	}
	if !AdvanceHoldAtBoundary.Valid() || !AdvanceHoldAfterRewritesDrained.Valid() || !AdvanceHoldAtChapter.Valid() || AdvanceHoldAfter("bad").Valid() {
		t.Fatal("advance hold validity is wrong")
	}
	valid := AdvanceHold{After: AdvanceHoldAtChapter, TargetChapter: 2, Reason: "pause"}
	if err := valid.Validate(); err != nil {
		t.Fatal(err)
	}
	for _, bad := range []AdvanceHold{{After: "bad", Reason: "x"}, {After: AdvanceHoldAtChapter, Reason: "x"}, {After: AdvanceHoldAtBoundary, TargetChapter: 1, Reason: "x"}, {After: AdvanceHoldAtBoundary}} {
		if err := bad.Validate(); err == nil {
			t.Fatalf("expected invalid hold: %+v", bad)
		}
	}
}

func TestSimulationHelpers(t *testing.T) {
	if SimulationSourceFingerprint(" file ", " hash ") != "file:hash" {
		t.Fatal("fingerprint normalization failed")
	}
	profile := SimulationProfile{Version: SimulationProfileVersion, Corpus: SimulationCorpusManifest{Sources: []SimulationSource{{RelativePath: "a.md", SHA256: "abc"}}}}
	if err := ValidateSimulationProfile(&profile); err != nil || profile.Corpus.Sources[0].Fingerprint != "a.md:abc" {
		t.Fatalf("profile validation = %v, %+v", err, profile)
	}
	for _, bad := range []*SimulationProfile{nil, {Version: "bad"}, {Version: SimulationProfileVersion, Corpus: SimulationCorpusManifest{Sources: []SimulationSource{{RelativePath: "", SHA256: "x"}}}}} {
		if err := ValidateSimulationProfile(bad); err == nil {
			t.Fatal("expected invalid simulation profile")
		}
	}
	merged := MergeSimulationSynthesis(SimulationSynthesis{Style: SimulationStyle{Mood: []string{"Dark"}}}, SimulationSynthesis{Style: SimulationStyle{Mood: []string{"dark", "Bright"}}})
	if len(merged.Style.Mood) != 2 || merged.Style.Mood[0] != "Dark" {
		t.Fatalf("merged synthesis = %+v", merged)
	}
}

package domain

import "testing"

func TestChapterReviewAndWordCountHelpers(t *testing.T) {
	for _, tc := range []struct {
		completed int
		want      bool
	}{
		{0, false}, {4, false}, {5, true}, {10, true},
	} {
		got, _ := ShouldReview(tc.completed)
		if got != tc.want {
			t.Fatalf("ShouldReview(%d) = %v", tc.completed, got)
		}
	}
	if ok, reason := ShouldArcReview(false, false, 1, 1); ok || reason != "" {
		t.Fatalf("no arc review = %v/%q", ok, reason)
	}
	if ok, reason := ShouldArcReview(true, false, 2, 3); !ok || reason == "" {
		t.Fatalf("arc review = %v/%q", ok, reason)
	}
	if ok, reason := ShouldArcReview(true, true, 2, 3); !ok || reason == "" {
		t.Fatalf("volume review = %v/%q", ok, reason)
	}
	if WordCount("你好\nworld") != 8 {
		t.Fatal("WordCount should count runes")
	}
}

func TestScopeConstructorsStringAndMatches(t *testing.T) {
	cases := []struct {
		scope Scope
		want  string
	}{
		{ChapterScope(3), "chapter:3"},
		{ArcScope(2, 4), "arc:v2a4"},
		{VolumeScope(2), "volume:2"},
		{GlobalScope(), "global"},
	}
	for _, tc := range cases {
		if tc.scope.String() != tc.want || !tc.scope.Matches(tc.scope) {
			t.Fatalf("scope = %q/%v", tc.scope.String(), tc.scope.Matches(tc.scope))
		}
	}
	if ChapterScope(1).Matches(ChapterScope(2)) || ArcScope(1, 1).Matches(ArcScope(1, 2)) || VolumeScope(1).Matches(GlobalScope()) {
		t.Fatal("different scopes should not match")
	}
}

func TestStoryMetadataAndOutlineHelpers(t *testing.T) {
	meta := BookMetadata{Title: " Title ", Synopsis: " Synopsis "}.Normalized()
	if meta.Title != "Title" || meta.Synopsis != "Synopsis" || meta.Validate() != nil {
		t.Fatalf("normalized metadata = %+v", meta)
	}
	if (BookMetadata{}).Validate() == nil || (BookMetadata{Title: "Title"}).Validate() == nil {
		t.Fatal("missing metadata should fail validation")
	}
	volumes := []VolumeOutline{{Index: 1, Arcs: []ArcOutline{{Index: 1, EstimatedChapters: 3}, {Index: 2, Chapters: []OutlineEntry{{Title: "A"}}}}}}
	if EstimatedChapterCapacity(volumes) != 4 || len(FlattenOutline(volumes)) != 1 {
		t.Fatalf("outline helpers = %d/%d", EstimatedChapterCapacity(volumes), len(FlattenOutline(volumes)))
	}
}

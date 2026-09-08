package store

import (
	"reflect"
	"testing"

	"github.com/JustinNguyen9979/ainovel-cli/internal/domain"
)

func TestSignalStoreRoundTripsAndClears(t *testing.T) {
	st := NewStore(t.TempDir())
	commit := domain.CommitResult{Chapter: 2, Committed: true, WordCount: 123}
	if err := st.Signals.SaveLastCommit(commit); err != nil {
		t.Fatal(err)
	}
	if got, err := st.Signals.LoadLastCommit(); err != nil || !reflect.DeepEqual(*got, commit) {
		t.Fatalf("commit = %+v, %v", got, err)
	}
	if got, err := st.Signals.LoadAndClearLastCommit(); err != nil || !reflect.DeepEqual(*got, commit) {
		t.Fatalf("clear commit = %+v, %v", got, err)
	}
	if got, err := st.Signals.LoadLastCommit(); err != nil || got != nil {
		t.Fatalf("cleared commit = %+v, %v", got, err)
	}

	review := domain.ReviewEntry{Chapter: 2, Verdict: "accept", Summary: "ok"}
	if err := st.Signals.SaveLastReview(review); err != nil {
		t.Fatal(err)
	}
	if got, err := st.Signals.LoadLastReviewSignal(); err != nil || !reflect.DeepEqual(*got, review) {
		t.Fatalf("review = %+v, %v", got, err)
	}
	if got, err := st.Signals.LoadAndClearLastReview(); err != nil || !reflect.DeepEqual(*got, review) {
		t.Fatalf("clear review = %+v, %v", got, err)
	}
	st.Signals.ClearStaleSignals()
}

func TestSimulationStoreRoundTripsAndValidates(t *testing.T) {
	st := NewStore(t.TempDir())
	profile := domain.SimulationProfile{Corpus: domain.SimulationCorpusManifest{Sources: []domain.SimulationSource{{RelativePath: "sample.md", SHA256: "abc"}}}}
	if err := st.Simulation.Save(profile); err != nil {
		t.Fatal(err)
	}
	got, err := st.Simulation.Load()
	if err != nil || got == nil || got.Version != domain.SimulationProfileVersion || got.Corpus.Sources[0].Fingerprint != "sample.md:abc" {
		t.Fatalf("profile = %+v, %v", got, err)
	}
	if err := st.Simulation.Save(domain.SimulationProfile{Version: "invalid"}); err == nil {
		t.Fatal("invalid simulation profile should fail")
	}
}

func TestSummaryStoreRoundTripsAndQueries(t *testing.T) {
	st := NewStore(t.TempDir())
	for _, sum := range []domain.ChapterSummary{
		{Chapter: 1, Title: "One", Characters: []string{"A"}},
		{Chapter: 2, Title: "Two", Characters: []string{"B"}},
		{Chapter: 3, Title: "Three", Characters: []string{"A", "B"}},
	} {
		if err := st.Summaries.SaveSummary(sum); err != nil {
			t.Fatal(err)
		}
	}
	if got, err := st.Summaries.LoadSummary(2); err != nil || got == nil || got.Title != "Two" {
		t.Fatalf("summary = %+v, %v", got, err)
	}
	if got, err := st.Summaries.LoadSummaryTitle(2); err != nil || got != "Two" {
		t.Fatalf("cached summary title = %q/%v", got, err)
	}
	st.Summaries.titleCache = make(map[int]string)
	if got, err := st.Summaries.LoadSummaryTitle(3); err != nil || got != "Three" {
		t.Fatalf("loaded summary title = %q/%v", got, err)
	}
	if got, err := st.Summaries.LoadSummaryTitle(99); err != nil || got != "" {
		t.Fatalf("missing summary title = %q/%v", got, err)
	}
	if got, err := st.Summaries.LoadRecentSummaries(4, 3); err != nil || len(got) != 3 {
		t.Fatalf("recent summaries = %+v, %v", got, err)
	}
	if got, err := st.Summaries.FindCharacterAppearances([]string{"A", "B"}, 6, 3); err != nil || got["A"] != 3 || got["B"] != 3 {
		t.Fatalf("appearances = %+v, %v", got, err)
	}
	if err := st.Summaries.SaveArcSummary(domain.ArcSummary{Volume: 1, Arc: 1, Title: "Arc"}); err != nil {
		t.Fatal(err)
	}
	if ok, err := st.Summaries.HasArcSummary(1, 1); err != nil || !ok {
		t.Fatalf("has arc = %v, %v", ok, err)
	}
	if err := st.Summaries.SaveVolumeSummary(domain.VolumeSummary{Volume: 1, Title: "Volume"}); err != nil {
		t.Fatal(err)
	}
	if ok, err := st.Summaries.HasVolumeSummary(1); err != nil || !ok {
		t.Fatalf("has volume = %v, %v", ok, err)
	}
}

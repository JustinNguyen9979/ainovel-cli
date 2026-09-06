package domain

import (
	"errors"
	"strings"
	"testing"

	"github.com/JustinNguyen9979/ainovel-cli/internal/errs"
)

func TestReviewEntryCountsAndDimensions(t *testing.T) {
	r := &ReviewEntry{
		Issues:     []ConsistencyIssue{{Severity: "critical"}, {Severity: "error"}, {Severity: "warning"}, {Severity: "critical"}},
		Dimensions: []DimensionScore{{Dimension: "characters", Score: 80}},
	}
	if r.CriticalCount() != 2 || r.ErrorCount() != 1 {
		t.Fatalf("counts = %d/%d", r.CriticalCount(), r.ErrorCount())
	}
	if got := r.Dimension("characters"); got == nil || got.Score != 80 {
		t.Fatalf("dimension = %+v", got)
	}
	if r.Dimension("missing") != nil || (&ReviewEntry{}).Dimension("missing") != nil {
		t.Fatal("missing dimension should be nil")
	}
}

func TestChapterContentNormalizationAndHash(t *testing.T) {
	if got := NormalizeChapterContent("\uFEFFa\r\nb\rc"); got != "a\nb\nc" {
		t.Fatalf("normalized content = %q", got)
	}
	if ChapterContentSHA256("a\r\nb") != ChapterContentSHA256("a\nb") {
		t.Fatal("normalized content should hash identically")
	}
}

func TestDomainTaxonomyCopiesAndValidation(t *testing.T) {
	hooks := HookTypes()
	strands := DominantStrands()
	if len(hooks) == 0 || len(strands) == 0 || !ValidHookType(hooks[0]) || !ValidDominantStrand(strands[0]) {
		t.Fatal("taxonomy values are invalid")
	}
	hooks[0], strands[0] = "changed", "changed"
	if ValidHookType("changed") || ValidDominantStrand("changed") {
		t.Fatal("taxonomy accessors must return copies")
	}
}

func TestTransitionValidationErrors(t *testing.T) {
	for _, tc := range []struct {
		name string
		got  error
		want error
	}{
		{"phase", ValidatePhaseTransition(PhaseComplete, PhaseWriting), errs.ErrPhaseTransition},
		{"flow", ValidateFlowTransition(FlowRewriting, FlowReviewing), errs.ErrFlowTransition},
	} {
		if tc.got == nil || !errors.Is(tc.got, tc.want) || !strings.Contains(tc.got.Error(), "invalid") {
			t.Fatalf("%s error = %v", tc.name, tc.got)
		}
	}
	if ValidatePhaseTransition(PhaseInit, PhaseOutline) != nil || ValidateFlowTransition(FlowWriting, FlowReviewing) != nil {
		t.Fatal("valid transitions should not return errors")
	}
}

func TestProgressAndMemoryPolicies(t *testing.T) {
	if (&Progress{Phase: PhaseWriting, CurrentChapter: 2}).IsResumable() == false || (&Progress{Phase: PhaseOutline, CurrentChapter: 2}).IsResumable() {
		t.Fatal("resumability decision is wrong")
	}
	progress := &Progress{CompletedChapters: []int{3, 1, 7, 2}}
	if progress.LatestCompleted() != 7 || progress.NextChapter() != 8 {
		t.Fatalf("progress next chapter = %d/%d", progress.LatestCompleted(), progress.NextChapter())
	}
	for _, tc := range []struct {
		total int
		want  ContextProfile
	}{
		{15, ContextProfile{SummaryWindow: 10, TimelineWindow: 10}},
		{16, ContextProfile{SummaryWindow: 5, TimelineWindow: 8}},
		{50, ContextProfile{SummaryWindow: 5, TimelineWindow: 8}},
		{51, ContextProfile{SummaryWindow: 3, TimelineWindow: 5, Layered: true}},
	} {
		if got := NewContextProfile(tc.total); got != tc.want {
			t.Fatalf("NewContextProfile(%d) = %+v", tc.total, got)
		}
	}
	policy := NewChapterMemoryPolicy(&Progress{TotalChapters: 31, Flow: FlowReviewing, CompletedChapters: []int{1, 2, 3, 4, 5, 6}, Layered: true}, ContextProfile{SummaryWindow: 3, TimelineWindow: 5, Layered: true}, true)
	if !policy.RelatedLookup || !policy.HandoffPreferred || policy.ReadOnlyThreshold != 4 || policy.SummaryStrategy == "" || !policy.CurrentOutlineBound {
		t.Fatalf("chapter policy = %+v", policy)
	}
	if policy := NewChapterMemoryPolicy(nil, ContextProfile{}, false); policy.RelatedLookup || policy.HandoffPreferred || policy.ReadOnlyThreshold != 5 {
		t.Fatalf("zero chapter policy = %+v", policy)
	}
}

func TestAdvanceModesAndHolds(t *testing.T) {
	if !ChapterAdvanceAuto.Valid() || !ChapterAdvanceReview.Valid() || ChapterAdvanceMode("legacy").Valid() {
		t.Fatal("advance mode validation is wrong")
	}
	if !AdvanceHoldAtBoundary.Valid() || !AdvanceHoldAfterRewritesDrained.Valid() || !AdvanceHoldAtChapter.Valid() || AdvanceHoldAfter("legacy").Valid() {
		t.Fatal("hold condition validation is wrong")
	}
	valid := []AdvanceHold{
		{After: AdvanceHoldAtBoundary, Reason: "pause"},
		{After: AdvanceHoldAfterRewritesDrained, Reason: "pause"},
		{After: AdvanceHoldAtChapter, TargetChapter: 3, Reason: "pause"},
	}
	for _, hold := range valid {
		if err := hold.Validate(); err != nil {
			t.Fatalf("valid hold rejected: %v", err)
		}
	}
	invalid := []AdvanceHold{
		{After: "legacy", Reason: "pause"},
		{After: AdvanceHoldAtChapter, Reason: "pause"},
		{After: AdvanceHoldAtChapter, TargetChapter: 0, Reason: "pause"},
		{After: AdvanceHoldAtBoundary, TargetChapter: 1, Reason: "pause"},
		{After: AdvanceHoldAtBoundary, Reason: "  "},
	}
	for _, hold := range invalid {
		if err := hold.Validate(); err == nil {
			t.Fatalf("invalid hold accepted: %+v", hold)
		}
	}
}

func TestSimulationValidationAndMerge(t *testing.T) {
	if got := SimulationSourceFingerprint("  source.md ", " hash "); got != "source.md:hash" {
		t.Fatalf("source fingerprint = %q", got)
	}
	for _, profile := range []*SimulationProfile{nil, {Version: "old"}, {Version: SimulationProfileVersion, Corpus: SimulationCorpusManifest{Sources: []SimulationSource{{RelativePath: "", SHA256: "hash"}}}}} {
		if err := ValidateSimulationProfile(profile); err == nil {
			t.Fatalf("invalid profile accepted: %+v", profile)
		}
	}
	profile := &SimulationProfile{
		Version:       SimulationProfileVersion,
		Corpus:        SimulationCorpusManifest{Sources: []SimulationSource{{RelativePath: "source.md", SHA256: "hash"}}},
		SourceReports: []SimulationSourceReport{{RelativePath: "source.md", SHA256: "hash"}},
	}
	if err := ValidateSimulationProfile(profile); err != nil {
		t.Fatal(err)
	}
	if profile.Corpus.Sources[0].Fingerprint != "source.md:hash" || profile.SourceReports[0].Fingerprint != "source.md:hash" {
		t.Fatalf("fingerprints were not backfilled: %+v", profile)
	}
	encoded, err := MarshalSimulationProfile(SimulationProfile{})
	if err != nil || !strings.Contains(string(encoded), SimulationProfileVersion) {
		t.Fatalf("marshal default version: %v, %s", err, encoded)
	}
	merged := MergeSimulationSynthesis(
		SimulationSynthesis{Style: SimulationStyle{NarrativeVoice: []string{" Calm ", "Vivid"}}, Lexicon: SimulationLexicon{CommonWords: []string{"Light"}}},
		SimulationSynthesis{Style: SimulationStyle{NarrativeVoice: []string{"calm", "Tense", " "}}, Lexicon: SimulationLexicon{CommonWords: []string{"light", "Shadow"}}},
	)
	if got := strings.Join(merged.Style.NarrativeVoice, ","); got != "Calm,Vivid,Tense" {
		t.Fatalf("merged voices = %q", got)
	}
	if got := strings.Join(merged.Lexicon.CommonWords, ","); got != "Light,Shadow" {
		t.Fatalf("merged words = %q", got)
	}
}

func TestArchitectPolicyAndUnsupportedMode(t *testing.T) {
	policy := NewArchitectMemoryPolicy()
	if policy.Mode != "architect" || !policy.HandoffPreferred || policy.ChapterPlanEnabled || policy.ReadOnlyThreshold != 4 {
		t.Fatalf("architect policy = %+v", policy)
	}
	if err := (&UnsupportedAdvanceModeError{Mode: "legacy"}).Error(); err == "" || !strings.Contains(err, "legacy") {
		t.Fatal("unsupported mode error should include mode")
	}
}

func TestStoryAndSimulationSerialization(t *testing.T) {
	if (&VolumeOutline{}).IsExpanded() || (&ArcOutline{}).IsExpanded() {
		t.Fatal("empty outlines should not be expanded")
	}
	if FinaleVolume(nil) != 0 || FinaleVolume([]VolumeOutline{{Index: 1, Final: true}, {Index: 2}}) != 0 {
		t.Fatal("finale volume should only use the last volume")
	}
	if _, err := MarshalSimulationProfile(SimulationProfile{}); err != nil {
		t.Fatal(err)
	}
	valid := &SimulationProfile{Version: SimulationProfileVersion, Corpus: SimulationCorpusManifest{Sources: []SimulationSource{{RelativePath: "x", SHA256: "y"}}}}
	if err := ValidateSimulationProfile(valid); err != nil {
		t.Fatal(err)
	}
	if valid.Corpus.Sources[0].Fingerprint != "x:y" {
		t.Fatal("source fingerprint not backfilled")
	}
}

package diag

import (
	"testing"

	"github.com/JustinNguyen9979/ainovel-cli/internal/domain"
)

func TestQualityRulesBoundariesAndFindings(t *testing.T) {
	if len(ChronicLowDimension(&Snapshot{Reviews: map[int]*domain.ReviewEntry{1: {Dimensions: []domain.DimensionScore{{Dimension: "hook", Score: 60}}}}})) != 0 {
		t.Fatal("single review should not trigger chronic finding")
	}
	low := &Snapshot{Reviews: map[int]*domain.ReviewEntry{1: {Dimensions: []domain.DimensionScore{{Dimension: "hook", Score: 50}}}, 2: {Dimensions: []domain.DimensionScore{{Dimension: "hook", Score: 50}}}}}
	if len(ChronicLowDimension(low)) != 1 {
		t.Fatal("low repeated dimension should trigger")
	}
	contract := &Snapshot{Reviews: map[int]*domain.ReviewEntry{1: {ContractStatus: "missed"}, 2: {ContractStatus: "met"}, 3: {ContractStatus: "partial"}}}
	if len(ContractMissPattern(contract)) != 1 {
		t.Fatal("contract miss rate should trigger")
	}
	rewrites := &Snapshot{Reviews: map[int]*domain.ReviewEntry{1: {Verdict: "rewrite"}, 2: {Verdict: "rewrite"}, 3: {Verdict: "polish"}}}
	if len(ExcessiveRewrites(rewrites)) != 1 {
		t.Fatal("rewrite rate should trigger")
	}
	wordCounts := &Snapshot{Progress: &domain.Progress{ChapterWordCounts: map[int]int{1: 100, 2: 100, 3: 100, 4: 10, 5: 300}}}
	if len(WordCountAnomaly(wordCounts)) != 1 {
		t.Fatal("word count outliers should trigger")
	}
}

func TestHookAndPayoffRules(t *testing.T) {
	reviews := map[int]*domain.ReviewEntry{}
	plans := map[int]*domain.ChapterPlan{}
	for ch := 1; ch <= 3; ch++ {
		reviews[ch] = &domain.ReviewEntry{Chapter: ch, Scope: "chapter", Dimensions: []domain.DimensionScore{{Dimension: "hook", Score: 40}}, ContractStatus: "missed"}
		plans[ch] = &domain.ChapterPlan{Chapter: ch, Contract: domain.ChapterContract{PayoffPoints: []string{"payoff"}}}
	}
	findings := HookWeakChain(&Snapshot{Reviews: reviews})
	if len(findings) != 1 || findings[0].Rule != "HookWeakChain" {
		t.Fatalf("hook findings = %+v", findings)
	}
	findings = PayoffMissPattern(&Snapshot{Reviews: reviews, Plans: plans})
	if len(findings) != 1 || findings[0].Rule != "PayoffMissPattern" {
		t.Fatalf("payoff findings = %+v", findings)
	}
}

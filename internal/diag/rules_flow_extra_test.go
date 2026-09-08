package diag

import (
	"strings"
	"testing"

	"github.com/JustinNguyen9979/ainovel-cli/internal/domain"
)

func TestFlowRulesAndHelpers(t *testing.T) {
	if flowStr(nil) != "<nil>" || flowStr(&domain.Progress{Flow: domain.FlowWriting}) != "writing" || intsToStr([]int{1, -2}) != "1, -2" {
		t.Fatal("flow helpers failed")
	}
	long := strings.Repeat("x", 70)
	if got := truncStr(long, 10); len([]rune(got)) != 10 || !strings.HasSuffix(got, "...") {
		t.Fatalf("truncStr = %q", got)
	}
	valid := &Snapshot{Progress: &domain.Progress{CompletedChapters: []int{1, 2}, PendingRewrites: []int{2}, Flow: domain.FlowRewriting}}
	if got := InvalidPendingRewrites(valid); got != nil || len(RewritePendingPressure(valid)) != 1 || ChapterGaps(valid) != nil {
		t.Fatalf("valid flow findings = %+v", got)
	}
	invalid := &Snapshot{Progress: &domain.Progress{CompletedChapters: []int{1, 3}, PendingRewrites: []int{0, 4}, Flow: domain.FlowRewriting, Phase: domain.PhaseOutline}}
	if len(InvalidPendingRewrites(invalid)) != 1 || len(ChapterGaps(invalid)) != 1 || len(PhaseFlowMismatch(invalid)) != 1 {
		t.Fatalf("invalid flow findings = %+v", invalid.Progress)
	}
	if len(OrphanedSteer(&Snapshot{RunMeta: &domain.RunMeta{PendingSteer: "steer"}})) != 1 {
		t.Fatal("orphaned steer should be reported")
	}
	if len(OrphanedSteer(&Snapshot{RunMeta: &domain.RunMeta{PendingSteer: "steer"}, Progress: &domain.Progress{Flow: domain.FlowSteering}})) != 0 {
		t.Fatal("active steer should not be reported")
	}
}

package diag

import (
	"testing"

	"github.com/JustinNguyen9979/ainovel-cli/internal/domain"
)

func TestPlanningRulesBoundaries(t *testing.T) {
	base := &Snapshot{Progress: &domain.Progress{Phase: domain.PhaseWriting, CompletedChapters: []int{1, 2, 3, 10}, Layered: true}, Foreshadow: []domain.ForeshadowEntry{{ID: "old", PlantedAt: 1, Status: "planted"}, {ID: "done", PlantedAt: 1, Status: "resolved"}}}
	if len(StaleForeshadow(base)) != 1 {
		t.Fatal("stale planted foreshadow should trigger")
	}
	if len(CompassDrift(&Snapshot{Progress: base.Progress, Compass: &domain.StoryCompass{LastUpdated: 0}})) != 0 {
		t.Fatal("compass should be current within drift threshold")
	}
	if len(CompassDrift(&Snapshot{Progress: &domain.Progress{Layered: true, CompletedChapters: []int{1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 11, 12, 13, 14, 15, 16, 17, 18, 19, 20}}, Compass: &domain.StoryCompass{LastUpdated: 0}})) != 1 {
		t.Fatal("stale compass should trigger")
	}
	if len(CompassDrift(&Snapshot{Progress: &domain.Progress{Layered: true, CompletedChapters: []int{1}}, Compass: nil})) != 0 {
		t.Fatal("short layered project should not trigger missing compass")
	}
	if len(OutlineExhausted(&Snapshot{Progress: &domain.Progress{Phase: domain.PhaseWriting, CompletedChapters: []int{1, 2}, TotalChapters: 2}})) != 1 {
		t.Fatal("exhausted outline should trigger")
	}
	if len(MissingSummaries(&Snapshot{Progress: &domain.Progress{CompletedChapters: []int{1, 2}}, Summaries: map[int]*domain.ChapterSummary{1: {Chapter: 1}}})) != 1 {
		t.Fatal("missing summary should trigger")
	}
}

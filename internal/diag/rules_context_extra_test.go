package diag

import (
	"testing"

	"github.com/JustinNguyen9979/ainovel-cli/internal/domain"
)

func TestContextRulesTriggerAndIgnoreBoundaries(t *testing.T) {
	progress := &domain.Progress{CompletedChapters: []int{1, 2, 3, 4, 5, 6, 7, 8, 9, 10}}
	chars := []domain.Character{{Name: "A", Tier: "core", Aliases: []string{"Bee"}}, {Name: "B", Tier: "secondary"}}
	if len(GhostCharacter(&Snapshot{Progress: progress, Characters: chars, Summaries: map[int]*domain.ChapterSummary{1: {Chapter: 1, Characters: []string{"Other"}}}})) != 1 {
		t.Fatal("unseen core character should trigger")
	}
	if len(GhostCharacter(&Snapshot{Progress: progress, Characters: chars, Summaries: map[int]*domain.ChapterSummary{1: {Characters: []string{"Bee"}}, 10: {Characters: []string{"A"}}}})) != 0 {
		t.Fatal("alias appearance should prevent ghost finding")
	}
	if len(TimelineGaps(&Snapshot{Progress: progress, Timeline: []domain.TimelineEvent{{Chapter: 1}, {Chapter: 2}, {Chapter: 3}, {Chapter: 4}, {Chapter: 5}, {Chapter: 6}, {Chapter: 7}, {Chapter: 8}}})) != 0 {
		t.Fatal("small timeline gap should be tolerated")
	}
	if len(RelationshipStagnation(&Snapshot{Progress: progress, Relationships: []domain.RelationshipEntry{{Chapter: 1, CharacterA: "A", CharacterB: "B"}}})) != 1 {
		t.Fatal("stale relationship should trigger")
	}
}

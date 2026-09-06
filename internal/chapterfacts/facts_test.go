package chapterfacts

import (
	"strings"
	"testing"

	"github.com/JustinNguyen9979/ainovel-cli/internal/domain"
)

func validFacts() domain.ChapterFacts {
	return domain.ChapterFacts{
		Title:      "Chapter",
		Summary:    "Summary",
		Characters: []string{"A"},
		KeyEvents:  []string{"Event"},
		TimelineEvents: []domain.TimelineEvent{{
			Time:       "Morning",
			Event:      "Meeting",
			Characters: []string{"A"},
		}},
		ForeshadowUpdates: []domain.ForeshadowUpdate{{
			ID:          "f1",
			Action:      "plant",
			Description: "A clue",
		}},
		RelationshipChanges: []domain.RelationshipEntry{{
			CharacterA: "A",
			CharacterB: "B",
			Relation:   "ally",
		}},
		StateChanges: []domain.StateChange{{
			Entity:   "A",
			Field:    "status",
			NewValue: "ready",
		}},
		CastIntros: []domain.CastIntro{{
			Name:      "B",
			BriefRole: "Witness",
		}},
		HookType:       "crisis",
		DominantStrand: "quest",
		Feedback:       &domain.OutlineFeedback{Deviation: "none", Suggestion: "continue"},
	}
}

func TestValidateAcceptsCompleteFacts(t *testing.T) {
	if err := Validate(validFacts()); err != nil {
		t.Fatalf("Validate() error = %v", err)
	}
}

func TestValidateRejectsRequiredTextAndItems(t *testing.T) {
	cases := []struct {
		name string
		edit func(*domain.ChapterFacts)
	}{
		{"title", func(f *domain.ChapterFacts) { f.Title = "  " }},
		{"summary", func(f *domain.ChapterFacts) { f.Summary = "" }},
		{"key events missing", func(f *domain.ChapterFacts) { f.KeyEvents = nil }},
		{"character empty", func(f *domain.ChapterFacts) { f.Characters = []string{"A", "  "} }},
		{"key event empty", func(f *domain.ChapterFacts) { f.KeyEvents = []string{""} }},
		{"timeline time missing", func(f *domain.ChapterFacts) { f.TimelineEvents[0].Time = "" }},
		{"timeline event missing", func(f *domain.ChapterFacts) { f.TimelineEvents[0].Event = " " }},
		{"timeline character empty", func(f *domain.ChapterFacts) { f.TimelineEvents[0].Characters = []string{""} }},
		{"relationship field missing", func(f *domain.ChapterFacts) { f.RelationshipChanges[0].Relation = "" }},
		{"relationship self reference", func(f *domain.ChapterFacts) { f.RelationshipChanges[0].CharacterB = "A" }},
		{"state entity missing", func(f *domain.ChapterFacts) { f.StateChanges[0].Entity = "" }},
		{"state field missing", func(f *domain.ChapterFacts) { f.StateChanges[0].Field = "" }},
		{"state value missing", func(f *domain.ChapterFacts) { f.StateChanges[0].NewValue = "" }},
		{"cast name missing", func(f *domain.ChapterFacts) { f.CastIntros[0].Name = "" }},
		{"cast role missing", func(f *domain.ChapterFacts) { f.CastIntros[0].BriefRole = "" }},
		{"feedback deviation missing", func(f *domain.ChapterFacts) { f.Feedback.Deviation = "" }},
		{"feedback suggestion missing", func(f *domain.ChapterFacts) { f.Feedback.Suggestion = "" }},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			facts := validFacts()
			tc.edit(&facts)
			if err := Validate(facts); err == nil {
				t.Fatal("Validate() accepted invalid facts")
			}
		})
	}
}

func TestValidateRejectsForeshadowAndTaxonomyErrors(t *testing.T) {
	cases := []struct {
		name string
		edit func(*domain.ChapterFacts)
	}{
		{"missing id", func(f *domain.ChapterFacts) { f.ForeshadowUpdates[0].ID = "" }},
		{"plant missing description", func(f *domain.ChapterFacts) { f.ForeshadowUpdates[0].Description = "" }},
		{"invalid action", func(f *domain.ChapterFacts) { f.ForeshadowUpdates[0].Action = "remove" }},
		{"advance allowed", func(f *domain.ChapterFacts) { f.ForeshadowUpdates[0].Action = "advance" }},
		{"resolve allowed", func(f *domain.ChapterFacts) { f.ForeshadowUpdates[0].Action = "resolve" }},
		{"invalid hook", func(f *domain.ChapterFacts) { f.HookType = "unknown" }},
		{"invalid strand", func(f *domain.ChapterFacts) { f.DominantStrand = "unknown" }},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			facts := validFacts()
			tc.edit(&facts)
			err := Validate(facts)
			if strings.HasSuffix(tc.name, "allowed") {
				if err != nil {
					t.Fatalf("Validate() error = %v", err)
				}
				return
			}
			if err == nil {
				t.Fatal("Validate() accepted invalid facts")
			}
		})
	}
}

func TestPropertiesIncludeOptionalFeedback(t *testing.T) {
	withoutFeedback := Properties(false)
	withFeedback := Properties(true)
	if len(withFeedback) != len(withoutFeedback)+1 {
		t.Fatalf("property counts = %d and %d", len(withoutFeedback), len(withFeedback))
	}
}

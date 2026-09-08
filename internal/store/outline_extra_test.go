package store

import (
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/JustinNguyen9979/ainovel-cli/internal/domain"
)

func TestOutlineLookupAndCompletedBoundaries(t *testing.T) {
	s := setupLayered(t, []domain.VolumeOutline{{Index: 1, Arcs: []domain.ArcOutline{
		{Index: 1, Chapters: []domain.OutlineEntry{{Title: "A"}, {Title: "B"}}},
		{Index: 2, Chapters: []domain.OutlineEntry{{Title: "C"}}},
	}}})
	if got, err := s.Outline.GetChapterOutline(1); err != nil || got == nil || got.Title != "A" {
		t.Fatalf("flat lookup = %+v/%v", got, err)
	}
	if _, err := s.Outline.GetChapterOutline(9); !errors.Is(err, ErrOutlineChapterNotFound) {
		t.Fatalf("missing flat chapter error = %v", err)
	}
	if got, err := s.Outline.GetChapterFromLayered(2); err != nil || got == nil || got.Chapter != 2 {
		t.Fatalf("layered lookup = %+v/%v", got, err)
	}
	if volume, arc, err := s.Outline.LocateChapter(3); err != nil || volume != 1 || arc != 2 {
		t.Fatalf("locate chapter = %d/%d/%v", volume, arc, err)
	}
	boundaries, err := s.Outline.CompletedArcBoundaries(3)
	if err != nil || len(boundaries) != 2 || boundaries[0].EndChapter != 2 || boundaries[1].Volume != 1 {
		t.Fatalf("boundaries = %+v/%v", boundaries, err)
	}
	if b, err := s.Outline.CheckArcBoundary(2); err != nil || b == nil || !b.IsArcEnd || b.NextArc != 2 || b.NeedsExpansion {
		t.Fatalf("boundary = %+v/%v", b, err)
	}
	if err := s.Outline.ClearLayeredOutline(); err != nil {
		t.Fatal(err)
	}
	if got, err := s.Outline.LoadLayeredOutline(); err != nil || got != nil {
		t.Fatalf("cleared layered outline = %+v/%v", got, err)
	}
}

func TestOutlineFeedbackAndFoundationAuditRoundTrip(t *testing.T) {
	s := setupLayered(t, []domain.VolumeOutline{{Index: 1, Arcs: []domain.ArcOutline{{Index: 1, Chapters: []domain.OutlineEntry{{Title: "A"}}}}}})
	if got, err := s.Outline.LoadCompass(); err != nil || got != nil {
		t.Fatalf("missing compass = %+v/%v", got, err)
	}
	if err := s.Outline.SaveCompass(domain.StoryCompass{EndingDirection: "收束", EstimatedScale: "一卷"}); err != nil {
		t.Fatal(err)
	}
	if got, err := s.Outline.LoadCompass(); err != nil || got == nil || got.EndingDirection != "收束" {
		t.Fatalf("compass = %+v/%v", got, err)
	}
	if err := s.Outline.SaveFoundationAudit(domain.FoundationAudit{Ready: false, Summary: "仍需补充"}); err != nil {
		t.Fatal(err)
	}
	if got, err := s.Outline.LoadFoundationAudit(); err != nil || got == nil || got.Ready || got.Summary != "仍需补充" {
		t.Fatalf("foundation audit = %+v/%v", got, err)
	}
	feedback := ChapterFeedback{Chapter: 1, StoryChanged: true, ChangeSummary: "主线变化", Suggestion: "更新后续弧", DownstreamIssues: []string{"第2章需重审"}}
	if !feedback.RequiresImmediateReview() {
		t.Fatal("story-changing feedback requires immediate review")
	}
	if err := s.Outline.AppendOutlineFeedback(feedback); err != nil {
		t.Fatal(err)
	}
	if err := s.Outline.AppendOutlineFeedback(feedback); err != nil {
		t.Fatal(err)
	}
	pending, err := s.Outline.LoadPendingOutlineFeedback()
	if err != nil || len(pending) != 1 || pending[0].Chapter != 1 {
		t.Fatalf("pending feedback = %+v/%v", pending, err)
	}
	if err := s.Outline.SaveOutlineFeedbackResolution("已被新弧吸收", 1); err != nil {
		t.Fatal(err)
	}
	if err := s.Outline.ClearOutlineFeedback(); err != nil {
		t.Fatal(err)
	}
	if pending, err := s.Outline.LoadPendingOutlineFeedback(); err != nil || len(pending) != 0 {
		t.Fatalf("cleared feedback = %+v/%v", pending, err)
	}
	if (ChapterFeedback{}).RequiresImmediateReview() {
		t.Fatal("empty feedback should not require immediate review")
	}
}

func TestOutlineFeedbackRejectsCorruptData(t *testing.T) {
	s := setupLayered(t, nil)
	path := filepath.Join(s.Dir(), "meta", "outline_feedback.jsonl")
	if err := os.WriteFile(path, []byte("{broken\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if _, err := s.Outline.LoadPendingOutlineFeedback(); err == nil {
		t.Fatal("corrupt feedback must fail explicitly")
	}
	if err := s.Outline.ClearOutlineFeedback(); err == nil {
		t.Fatal("clear must not erase corrupt feedback")
	}
}

func TestOutlineRenderBranches(t *testing.T) {
	entries := []domain.OutlineEntry{{Chapter: 1, Title: "开端", CoreEvent: "事件", Hook: "悬念", Scenes: []string{"场景"}}}
	if err := setupLayered(t, nil).Outline.SaveOutline(entries); err != nil {
		t.Fatal(err)
	}
	volumes := []domain.VolumeOutline{{Index: 1, Title: "卷一", Theme: "主题", Arcs: []domain.ArcOutline{
		{Index: 1, Title: "已展开", Goal: "目标", Chapters: entries},
		{Index: 2, Title: "骨架", Goal: "目标2", EstimatedChapters: 4},
	}}}
	s := setupLayered(t, volumes)
	if err := s.Outline.SaveLayeredOutline(volumes); err != nil {
		t.Fatal(err)
	}
	if err := s.World.SaveWorldRules([]domain.WorldRule{{Category: "", Rule: "规则", Boundary: "边界"}}); err != nil {
		t.Fatal(err)
	}
	if data, err := os.ReadFile(filepath.Join(s.Dir(), "outline.md")); err != nil || !strings.Contains(string(data), "场景") {
		t.Fatalf("outline projection = %q/%v", data, err)
	}
	if data, err := os.ReadFile(filepath.Join(s.Dir(), "layered_outline.md")); err != nil || !strings.Contains(string(data), "待展开") {
		t.Fatalf("layered projection = %q/%v", data, err)
	}
	if data, err := os.ReadFile(filepath.Join(s.Dir(), "world_rules.md")); err != nil || !strings.Contains(string(data), "边界") {
		t.Fatalf("world rules projection = %q/%v", data, err)
	}
}

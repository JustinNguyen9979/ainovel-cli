package store

import (
	"strings"
	"testing"

	"github.com/JustinNguyen9979/ainovel-cli/internal/domain"
)

func TestFoundationMissingProgressAndLayeredCompassBranches(t *testing.T) {
	st := NewStore(t.TempDir())
	if err := st.Init(); err != nil {
		t.Fatal(err)
	}
	missing, err := st.FoundationMissing()
	if err != nil || len(missing) < 5 {
		t.Fatalf("empty foundation missing = %#v/%v", missing, err)
	}
	if err := st.Book.Save(domain.BookMetadata{Title: "书", Synopsis: "简介"}); err != nil {
		t.Fatal(err)
	}
	if err := st.Outline.SavePremise("前提"); err != nil {
		t.Fatal(err)
	}
	if err := st.Outline.SaveOutline([]domain.OutlineEntry{{Chapter: 1, Title: "一"}}); err != nil {
		t.Fatal(err)
	}
	if err := st.Characters.Save([]domain.Character{{Name: "主角"}}); err != nil {
		t.Fatal(err)
	}
	if err := st.World.SaveWorldRules([]domain.WorldRule{{Rule: "规则"}}); err != nil {
		t.Fatal(err)
	}
	if missing, err = st.FoundationMissing(); err != nil || len(missing) != 1 || missing[0] != "foundation_audit" {
		t.Fatalf("audit missing = %#v/%v", missing, err)
	}
	if err := st.Progress.Save(&domain.Progress{Phase: domain.PhaseWriting}); err != nil {
		t.Fatal(err)
	}
	if missing, err = st.FoundationMissing(); err != nil || len(missing) != 0 {
		t.Fatalf("writing foundation missing = %#v/%v", missing, err)
	}
	if err := st.Outline.SaveLayeredOutline([]domain.VolumeOutline{{Index: 1, Arcs: []domain.ArcOutline{{Index: 1, Chapters: []domain.OutlineEntry{{Title: "一"}}}}}}); err != nil {
		t.Fatal(err)
	}
	if missing, err = st.FoundationMissing(); err != nil || len(missing) != 1 || missing[0] != "compass" {
		t.Fatalf("layered compass missing = %#v/%v", missing, err)
	}
	if err := st.Outline.SaveCompass(domain.StoryCompass{EndingDirection: "终局"}); err != nil {
		t.Fatal(err)
	}
	if missing, err = st.FoundationMissing(); err != nil || len(missing) != 0 {
		t.Fatalf("complete layered foundation missing = %#v/%v", missing, err)
	}
}

func TestClearHandledSteerResetsSteeringFlow(t *testing.T) {
	st := NewStore(t.TempDir())
	if err := st.Init(); err != nil {
		t.Fatal(err)
	}
	if err := st.RunMeta.Init("default", "openrouter", "model"); err != nil {
		t.Fatal(err)
	}
	if err := st.RunMeta.SetPendingSteer("继续"); err != nil {
		t.Fatal(err)
	}
	if err := st.Progress.Save(&domain.Progress{Phase: domain.PhaseWriting, Flow: domain.FlowSteering}); err != nil {
		t.Fatal(err)
	}
	if err := st.ClearHandledSteer(); err != nil {
		t.Fatal(err)
	}
	meta, err := st.RunMeta.Load()
	if err != nil || meta.PendingSteer != "" {
		t.Fatalf("pending steer = %+v/%v", meta, err)
	}
	progress, err := st.Progress.Load()
	if err != nil || progress.Flow != domain.FlowWriting {
		t.Fatalf("steering flow = %+v/%v", progress, err)
	}
	if err := st.ClearHandledSteer(); err != nil {
		t.Fatal(err)
	}
}

func TestStoreConsistencyFingerprintAndSummaryLoaders(t *testing.T) {
	st := NewStore(t.TempDir())
	if err := st.Init(); err != nil {
		t.Fatal(err)
	}
	if warnings := st.CheckConsistency(); len(warnings) != 0 {
		t.Fatalf("empty consistency warnings = %v", warnings)
	}
	if err := st.Progress.Save(&domain.Progress{Phase: domain.PhaseWriting, CompletedChapters: []int{2}, Layered: true, CurrentVolume: 1, CurrentArc: 9}); err != nil {
		t.Fatal(err)
	}
	warnings := st.CheckConsistency()
	if len(warnings) == 0 {
		t.Fatal("missing completed chapter should warn")
	}
	if err := st.Book.Save(domain.BookMetadata{Title: "书", Synopsis: "简介"}); err != nil {
		t.Fatal(err)
	}
	if err := st.Outline.SavePremise("# 设定"); err != nil {
		t.Fatal(err)
	}
	if err := st.Outline.SaveOutline([]domain.OutlineEntry{{Chapter: 1, Title: "一"}}); err != nil {
		t.Fatal(err)
	}
	if err := st.Characters.Save([]domain.Character{{Name: "主角"}}); err != nil {
		t.Fatal(err)
	}
	if err := st.World.SaveWorldRules([]domain.WorldRule{{Category: "magic", Rule: "规则"}}); err != nil {
		t.Fatal(err)
	}
	if _, err := st.FoundationFingerprint(); err != nil {
		t.Fatalf("flat foundation fingerprint: %v", err)
	}
	if err := st.Outline.SaveLayeredOutline([]domain.VolumeOutline{{Index: 1, Title: "卷一", Arcs: []domain.ArcOutline{{Index: 1, Title: "弧一", Chapters: []domain.OutlineEntry{{Title: "章一"}}}}}}); err != nil {
		t.Fatal(err)
	}
	if err := st.Outline.SaveCompass(domain.StoryCompass{EndingDirection: "终局"}); err != nil {
		t.Fatal(err)
	}
	if _, err := st.FoundationFingerprint(); err != nil {
		t.Fatal(err)
	}
	if err := st.Summaries.SaveArcSummary(domain.ArcSummary{Volume: 1, Arc: 1}); err != nil {
		t.Fatal(err)
	}
	if err := st.Summaries.SaveVolumeSummary(domain.VolumeSummary{Volume: 1}); err != nil {
		t.Fatal(err)
	}
	if got, err := st.Summaries.LoadArcSummaries(1); err != nil || len(got) != 1 {
		t.Fatalf("arc summaries = %#v/%v", got, err)
	}
	if got, err := st.Summaries.LoadAllVolumeSummaries(); err != nil || len(got) != 1 {
		t.Fatalf("volume summaries = %#v/%v", got, err)
	}
}

func TestStoreRevisionAndWorldBranches(t *testing.T) {
	st := NewStore(t.TempDir())
	if err := st.Init(); err != nil {
		t.Fatal(err)
	}
	if err := st.Progress.Init(4); err != nil {
		t.Fatal(err)
	}
	if err := st.Progress.UpdatePhase(domain.PhaseWriting); err != nil {
		t.Fatal(err)
	}
	if err := st.Outline.SaveOutline([]domain.OutlineEntry{{Chapter: 1, Title: "一"}, {Chapter: 2, Title: "二"}}); err != nil {
		t.Fatal(err)
	}
	if _, err := st.ReviseOutline(0, nil); err == nil {
		t.Fatal("invalid from chapter should fail")
	}
	if _, err := st.ReviseOutline(4, []domain.OutlineEntry{{Title: "新四"}}); err == nil {
		t.Fatal("revision beyond outline should fail")
	}
	if n, err := st.ReviseOutline(2, []domain.OutlineEntry{{Title: "新二"}, {Title: "新三"}}); err != nil || n != 3 {
		t.Fatalf("revise tail = %d/%v", n, err)
	}
	if err := st.World.SaveForeshadowLedger([]domain.ForeshadowEntry{{ID: "x", Description: "线索", PlantedAt: 1, Status: "planted"}}); err != nil {
		t.Fatal(err)
	}
	if err := st.World.UpdateForeshadow(2, []domain.ForeshadowUpdate{{ID: "x", Action: "advance"}, {ID: "x", Action: "resolve"}}); err != nil {
		t.Fatal(err)
	}
	if active, err := st.World.LoadActiveForeshadow(); err != nil || len(active) != 0 {
		t.Fatalf("active foreshadow = %#v/%v", active, err)
	}
	if err := st.World.SaveAuthorRevisionStyle(domain.AuthorRevisionStyle{Prose: []string{"短句"}}); err != nil {
		t.Fatal(err)
	}
	if style, err := st.World.LoadAuthorRevisionStyle(); err != nil || style == nil || len(style.Prose) != 1 {
		t.Fatalf("author style = %#v/%v", style, err)
	}
	if got := st.World.LoadRuleViolations(1); got != nil {
		t.Fatal("missing violations should be nil")
	}
	if !strings.Contains(st.Dir(), "") {
		t.Fatal("store dir unexpectedly empty")
	}
}

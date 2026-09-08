package flow

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/JustinNguyen9979/ainovel-cli/internal/domain"
	storepkg "github.com/JustinNguyen9979/ainovel-cli/internal/store"
)

func TestAggregateRefreshCopiesBoundary(t *testing.T) {
	boundary := &storepkg.ArcBoundary{Volume: 2, Arc: 3, StartChapter: 10, EndChapter: 18}
	got := aggregateRefresh(AggregateVolumeSummary, boundary)
	if got.Kind != AggregateVolumeSummary || got.Volume != 2 || got.Arc != 3 || got.StartChapter != 10 || got.EndChapter != 18 {
		t.Fatalf("aggregate refresh = %+v", got)
	}
}

func TestLoadStateNilProgressAndPlanningTier(t *testing.T) {
	st := storepkg.NewStore(t.TempDir())
	if err := st.Init(); err != nil {
		t.Fatal(err)
	}
	state, err := LoadState(st)
	if err != nil || state.Progress != nil {
		t.Fatalf("empty state = %+v/%v", state, err)
	}
	if err := st.RunMeta.Init("default", "openrouter", "model"); err != nil {
		t.Fatal(err)
	}
	if err := st.RunMeta.SetPlanningTier(domain.PlanningTierShort); err != nil {
		t.Fatal(err)
	}
	if err := st.Progress.Save(&domain.Progress{Phase: domain.PhaseOutline}); err != nil {
		t.Fatal(err)
	}
	state, err = LoadState(st)
	if err != nil || state.PlanningTier != domain.PlanningTierShort || state.Progress == nil {
		t.Fatalf("planning state = %+v/%v", state, err)
	}
}

func TestRouteAggregateKinds(t *testing.T) {
	p := &domain.Progress{Phase: domain.PhaseWriting, Flow: domain.FlowWriting, CompletedChapters: []int{1}}
	cases := []struct {
		kind AggregateKind
		want string
		text string
	}{
		{AggregateArcReview, "editor", "scope=arc"},
		{AggregateArcSummary, "editor", "save_arc_summary"},
		{AggregateVolumeSummary, "editor", "save_volume_summary"},
		{AggregateGlobalReview, "editor", "scope=global"},
	}
	for _, tc := range cases {
		t.Run(string(tc.kind), func(t *testing.T) {
			got := Route(State{Progress: p, AggregateRefresh: &AggregateRefresh{Kind: tc.kind, Volume: 1, Arc: 2, StartChapter: 1, EndChapter: 5}})
			if got == nil || got.Agent != tc.want || !contains(got.Task, tc.text) {
				t.Fatalf("route = %+v", got)
			}
		})
	}
	if got := plannerForTier(domain.PlanningTierShort); got != "architect_short" || plannerForTier(domain.PlanningTierLong) != "architect_long" || plannerForTier("") != "architect_long" {
		t.Fatal("planner mapping mismatch")
	}
}

func TestLoadStateDetectsMissingLayeredAggregates(t *testing.T) {
	st := storepkg.NewStore(t.TempDir())
	if err := st.Init(); err != nil {
		t.Fatal(err)
	}
	if err := st.Outline.SaveLayeredOutline([]domain.VolumeOutline{{Index: 1, Arcs: []domain.ArcOutline{
		{Index: 1, Chapters: []domain.OutlineEntry{{Chapter: 1, Title: "one"}, {Chapter: 2, Title: "two"}}},
		{Index: 2, Chapters: []domain.OutlineEntry{{Chapter: 3, Title: "three"}}},
	}}}); err != nil {
		t.Fatal(err)
	}
	if err := st.Outline.SaveOutline(domain.FlattenOutline([]domain.VolumeOutline{{Index: 1, Arcs: []domain.ArcOutline{
		{Index: 1, Chapters: []domain.OutlineEntry{{Chapter: 1, Title: "one"}, {Chapter: 2, Title: "two"}}},
		{Index: 2, Chapters: []domain.OutlineEntry{{Chapter: 3, Title: "three"}}},
	}}})); err != nil {
		t.Fatal(err)
	}
	if err := st.Progress.Save(&domain.Progress{Phase: domain.PhaseWriting, Flow: domain.FlowWriting, Layered: true, CurrentVolume: 1, CurrentArc: 1, CompletedChapters: []int{1, 2}}); err != nil {
		t.Fatal(err)
	}
	state, err := LoadState(st)
	if err != nil {
		t.Fatal(err)
	}
	if state.LastCompleted != 2 || state.AggregateRefresh == nil || state.AggregateRefresh.Kind != AggregateArcReview {
		t.Fatalf("missing layered aggregate = %+v", state)
	}
}

func TestLoadStateDetectsMissingGlobalReview(t *testing.T) {
	st := storepkg.NewStore(t.TempDir())
	if err := st.Init(); err != nil {
		t.Fatal(err)
	}
	if err := st.Outline.SaveOutline([]domain.OutlineEntry{{Chapter: 1, Title: "one"}, {Chapter: 2, Title: "two"}, {Chapter: 3, Title: "three"}, {Chapter: 4, Title: "four"}, {Chapter: 5, Title: "five"}}); err != nil {
		t.Fatal(err)
	}
	if err := st.Progress.Save(&domain.Progress{Phase: domain.PhaseWriting, Flow: domain.FlowWriting, CompletedChapters: []int{1, 2, 3, 4, 5}}); err != nil {
		t.Fatal(err)
	}
	state, err := LoadState(st)
	if err != nil {
		t.Fatal(err)
	}
	if state.AggregateRefresh == nil || state.AggregateRefresh.Kind != AggregateGlobalReview {
		t.Fatalf("missing global aggregate = %+v", state)
	}
}

func TestLoadStateRejectsCorruptFoundation(t *testing.T) {
	st := storepkg.NewStore(t.TempDir())
	if err := st.Init(); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(st.Dir(), "meta", "run.json"), []byte("{"), 0o600); err != nil {
		t.Fatal(err)
	}
	if _, err := LoadState(st); err == nil {
		t.Fatal("corrupt foundation should fail")
	}
}

func TestLoadStateUsesFeedbackAndPlanningTier(t *testing.T) {
	st := storepkg.NewStore(t.TempDir())
	if err := st.Init(); err != nil {
		t.Fatal(err)
	}
	if err := st.RunMeta.Init("default", "openrouter", "model"); err != nil {
		t.Fatal(err)
	}
	if err := st.RunMeta.SetPlanningTier(domain.PlanningTierShort); err != nil {
		t.Fatal(err)
	}
	if err := st.Outline.AppendOutlineFeedback(storepkg.ChapterFeedback{Chapter: 1, StoryChanged: true, ChangeSummary: "changed"}); err != nil {
		t.Fatal(err)
	}
	if err := st.Progress.Save(&domain.Progress{Phase: domain.PhaseOutline, Flow: domain.FlowWriting}); err != nil {
		t.Fatal(err)
	}
	state, err := LoadState(st)
	if err != nil || state.PlanningTier != domain.PlanningTierShort || state.ImmediateFeedbackCount != 1 {
		t.Fatalf("feedback planning state = %+v/%v", state, err)
	}
}

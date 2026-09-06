package store

import (
	"testing"

	"github.com/JustinNguyen9979/ainovel-cli/internal/domain"
)

func TestRunMetaControlLifecycle(t *testing.T) {
	st := NewStore(t.TempDir())
	if err := st.RunMeta.Init("default", "openrouter", "model"); err != nil {
		t.Fatal(err)
	}
	if err := st.RunMeta.SetAdvanceMode(domain.ChapterAdvanceReview); err != nil {
		t.Fatal(err)
	}
	if err := st.RunMeta.GrantAdvancePermit(2); err != nil {
		t.Fatal(err)
	}
	if err := st.RunMeta.GrantAdvancePermit(2); err != nil {
		t.Fatal(err)
	}
	if err := st.RunMeta.GrantAdvancePermit(3); err == nil {
		t.Fatal("different permit should be rejected")
	}
	if err := st.RunMeta.ClearAdvancePermit(3); err == nil {
		t.Fatal("mismatched permit should be rejected")
	}
	if err := st.RunMeta.ClearAdvancePermit(2); err != nil {
		t.Fatal(err)
	}
	hold := domain.AdvanceHold{After: domain.AdvanceHoldAtBoundary, Reason: "pause"}
	if err := st.RunMeta.SetAdvanceHold(hold); err != nil {
		t.Fatal(err)
	}
	if err := st.RunMeta.SetAdvanceHold(hold); err != nil {
		t.Fatal(err)
	}
	if err := st.RunMeta.SetAdvanceHold(domain.AdvanceHold{After: domain.AdvanceHoldAtChapter, TargetChapter: 3, Reason: "other"}); err == nil {
		t.Fatal("different hold should be rejected")
	}
	if err := st.RunMeta.ClearAdvanceHold(domain.AdvanceHold{After: domain.AdvanceHoldAtChapter, TargetChapter: 3, Reason: "other"}); err == nil {
		t.Fatal("mismatched hold should be rejected")
	}
	if err := st.RunMeta.ClearAdvanceHold(hold); err != nil {
		t.Fatal(err)
	}
	if err := st.RunMeta.SetAdvanceMode(domain.ChapterAdvanceAuto); err != nil {
		t.Fatal(err)
	}
	if err := st.RunMeta.SetAdvanceMode("invalid"); err == nil {
		t.Fatal("invalid mode should be rejected")
	}
}

func TestRunMetaIntentFieldsRoundTrip(t *testing.T) {
	st := NewStore(t.TempDir())
	if err := st.RunMeta.SetStartPrompt("prompt"); err != nil {
		t.Fatal(err)
	}
	if err := st.RunMeta.SetPendingSteer("steer"); err != nil {
		t.Fatal(err)
	}
	if err := st.RunMeta.SetPlanningTier(domain.PlanningTierLong); err != nil {
		t.Fatal(err)
	}
	if err := st.RunMeta.SetPlanStart(domain.PlanStartRecord{RawPrompt: "prompt", Planner: "architect"}); err != nil {
		t.Fatal(err)
	}
	meta, err := st.RunMeta.Load()
	if err != nil || meta == nil || meta.StartPrompt != "prompt" || meta.PendingSteer != "steer" || meta.PlanningTier != domain.PlanningTierLong || meta.PlanStart == nil {
		t.Fatalf("meta = %+v, %v", meta, err)
	}
	if err := st.RunMeta.ClearPendingSteer(); err != nil {
		t.Fatal(err)
	}
	meta, _ = st.RunMeta.Load()
	if meta.PendingSteer != "" {
		t.Fatal("pending steer should clear")
	}
}

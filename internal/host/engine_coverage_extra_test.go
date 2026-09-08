package host

import (
	"context"
	"errors"
	"strings"
	"testing"

	"github.com/JustinNguyen9979/ainovel-cli/internal/arbiter"
	"github.com/JustinNguyen9979/ainovel-cli/internal/domain"
	"github.com/JustinNguyen9979/ainovel-cli/internal/flow"
	storepkg "github.com/JustinNguyen9979/ainovel-cli/internal/store"
	"github.com/voocel/agentcore"
)

func TestEngineApplyControlOpHoldReopenAndDispatch(t *testing.T) {
	st := storepkg.NewStore(t.TempDir())
	if err := st.Init(); err != nil {
		t.Fatal(err)
	}
	if err := st.Progress.Init(3); err != nil {
		t.Fatal(err)
	}
	if err := st.Progress.UpdatePhase(domain.PhaseWriting); err != nil {
		t.Fatal(err)
	}
	if err := st.Progress.MarkChapterComplete(1, 100, "", ""); err != nil {
		t.Fatal(err)
	}
	if err := st.RunMeta.Init("default", "openrouter", "model"); err != nil {
		t.Fatal(err)
	}
	events := make([]Event, 0)
	e := &engine{store: st, emitEvent: func(ev Event) { events = append(events, ev) }, refresh: func() {}}
	op := controlOp{hold: &arbiter.AdvanceHoldOp{After: domain.AdvanceHoldAtChapter, TargetChapter: 2, Reason: "review"}}
	if err := e.applyControlOp(context.Background(), op); err != nil {
		t.Fatal(err)
	}
	meta, err := st.RunMeta.Load()
	if err != nil || meta.AdvanceHold == nil || meta.AdvanceHold.TargetChapter != 2 {
		t.Fatalf("hold = %+v/%v", meta, err)
	}
	cancel := controlOp{hold: &arbiter.AdvanceHoldOp{Cancel: true}}
	if err := e.applyControlOp(context.Background(), cancel); err != nil {
		t.Fatal(err)
	}
	meta, _ = st.RunMeta.Load()
	if meta.AdvanceHold != nil {
		t.Fatal("cancel should clear hold")
	}

	if err := st.Progress.Save(&domain.Progress{Phase: domain.PhaseWriting, CompletedChapters: []int{1}, TotalChapters: 3}); err != nil {
		t.Fatal(err)
	}
	facts, err := arbiter.CollectInterventionFacts(st)
	if err != nil {
		t.Fatal(err)
	}
	dispatch := controlOp{dispatch: &arbiter.DispatchOp{Agent: "writer", Task: "rewrite"}, text: "keep the clue", facts: facts}
	if err := e.applyControlOp(context.Background(), dispatch); err != nil {
		t.Fatal(err)
	}
	if e.next == nil || e.next.Agent != "writer" || !strings.Contains(e.next.Task, "keep the clue") {
		t.Fatalf("dispatch = %+v", e.next)
	}
	if len(events) == 0 {
		t.Fatal("control operations should emit events")
	}
}

func TestEnginePlanningAndPrecheckBranches(t *testing.T) {
	st := storepkg.NewStore(t.TempDir())
	if err := st.Init(); err != nil {
		t.Fatal(err)
	}
	e := &engine{store: st, refresh: func() {}}
	if got, err := e.planStartFallback(context.Background()); err != nil || got != nil {
		t.Fatalf("empty fallback = %+v/%v", got, err)
	}
	if err := st.Progress.Save(&domain.Progress{Phase: domain.PhaseOutline}); err != nil {
		t.Fatal(err)
	}
	if got, err := e.planStartFallback(context.Background()); err != nil || got != nil {
		t.Fatalf("planning fallback = %+v/%v", got, err)
	}
	if got, err := e.precheck(&flow.Instruction{Agent: "writer", Task: "write"}); err == nil || !strings.Contains(err.Error(), "writing") {
		t.Fatalf("writer precheck phase = %+v/%v", got, err)
	}
	if err := st.Progress.Save(&domain.Progress{Phase: domain.PhaseWriting}); err != nil {
		t.Fatal(err)
	}
	if got, err := e.planStartFallback(context.Background()); err != nil || got != nil {
		t.Fatalf("writing fallback = %+v/%v", got, err)
	}
	if got, err := writerTargetChapter(st); err != nil || got != 1 {
		t.Fatalf("writer target = %d/%v", got, err)
	}
	if err := st.Progress.MarkChapterComplete(1, 10, "", ""); err != nil {
		t.Fatal(err)
	}
	if got, err := writerTargetChapter(st); err != nil || got != 2 {
		t.Fatalf("next target = %d/%v", got, err)
	}
	if got, err := e.precheck(&flow.Instruction{Agent: "editor", Task: "review"}); err != nil || got != nil {
		t.Fatalf("editor precheck = %+v/%v", got, err)
	}
}

func TestEngineWorkerFailureHelpers(t *testing.T) {
	st := storepkg.NewStore(t.TempDir())
	if err := st.Init(); err != nil {
		t.Fatal(err)
	}
	if err := st.Progress.Save(&domain.Progress{Phase: domain.PhaseWriting, CompletedChapters: []int{1}, PendingRewrites: []int{1}, Flow: domain.FlowRewriting}); err != nil {
		t.Fatal(err)
	}
	var events []Event
	obs := testObserver(&events)
	obs.store = st
	e := &engine{store: st, observer: obs, arbiterModel: &scriptedChatModel{fn: func([]agentcore.Message) agentcore.Message {
		return testTextMsg(`{"action":"abort","dispatch":null,"reason":"stop"}`)
	}}, failurePrompt: "failure", emitEvent: func(Event) {}, notify: func(string, string, string, string) {}, onPause: func(string) {}}
	inst := &flow.Instruction{Agent: "writer", Task: "rewrite", Chapter: 1}
	if !e.dropStuckRewrite(inst) {
		t.Fatal("queued rewrite should be dropped")
	}
	if e.dropStuckRewrite(nil) || e.dropStuckRewrite(&flow.Instruction{Agent: "editor", Chapter: 1}) {
		t.Fatal("invalid rewrite drop should be false")
	}
	e.lastKey = instructionKey(inst)
	e.repeats = 1
	e.discardNonSemanticDeadlockAttempt(inst, agentcore.ErrProviderNetwork)
	if e.repeats != 0 || e.lastKey != "" {
		t.Fatalf("nonsemantic attempt not discarded: repeats=%d key=%q", e.repeats, e.lastKey)
	}
	if e.handleWorkerError(context.Background(), inst, errors.New("first")) {
		t.Fatal("first worker error should retry")
	}
	if !e.handleWorkerError(context.Background(), inst, errors.New("second")) {
		t.Fatal("second worker error should stop after arbiter abort")
	}
	if e.workerErrorFor(inst) != nil {
		t.Fatal("worker error should be cleared when not recorded")
	}
}

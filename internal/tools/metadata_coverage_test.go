package tools

import (
	"encoding/json"
	"testing"

	"github.com/JustinNguyen9979/ainovel-cli/internal/store"
)

func TestAllToolMetadataAndSchemas(t *testing.T) {
	st := store.NewStore(t.TempDir())
	tools := []interface {
		Name() string
		Description() string
		Label() string
		ReadOnly(json.RawMessage) bool
		ConcurrencySafe(json.RawMessage) bool
		Schema() map[string]any
	}{
		NewAuditFoundationTool(st),
		NewCheckConsistencyTool(st),
		NewCommitChapterTool(st, NewStyleStatsIndex(st)),
		NewDraftChapterTool(st),
		NewEditChapterTool(st),
		NewContextTool(st, References{}, "default", NewStyleStatsIndex(st)),
		NewPlanChapterTool(st),
		NewReadChapterTool(st),
		NewReopenBookTool(st),
		NewResolveOutlineFeedbackTool(st),
		NewReviseOutlineTool(st),
		NewSaveArcSummaryTool(st),
		NewSaveBookTool(st),
		NewSaveFoundationTool(st),
		NewSaveReviewTool(st),
		NewSaveVolumeSummaryTool(st),
	}
	for _, tool := range tools {
		if tool.Name() == "" || tool.Description() == "" || tool.Label() == "" || len(tool.Schema()) == 0 {
			t.Fatalf("incomplete metadata for %T", tool)
		}
		_ = tool.ReadOnly(nil)
		_ = tool.ConcurrencySafe(nil)
		_ = json.Valid
	}
}

func TestContextReadWarningsAndToolFlags(t *testing.T) {
	reads := &contextReads{}
	reads.warn("scope", nil)
	reads.warn("scope", errTest("bad"))
	if len(reads.warnings) != 1 {
		t.Fatalf("warnings = %#v", reads.warnings)
	}
	reads.require("required", errTest("bad"))
	if reads.err == nil || len(reads.warnings) != 1 {
		t.Fatalf("required error = %v warnings=%#v", reads.err, reads.warnings)
	}
	reads.fail(errTest("fatal"))
	if reads.err == nil {
		t.Fatal("fatal error missing")
	}
}

type errTest string

func (e errTest) Error() string { return string(e) }

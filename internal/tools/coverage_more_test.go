package tools

import (
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/JustinNguyen9979/ainovel-cli/internal/domain"
	"github.com/JustinNguyen9979/ainovel-cli/internal/errs"
	"github.com/JustinNguyen9979/ainovel-cli/internal/store"
)

func TestEnsureChapterExpandedGuardsAndLayeredScope(t *testing.T) {
	if err := EnsureChapterExpanded(nil, 1); !errors.Is(err, errs.ErrToolPrecondition) {
		t.Fatalf("nil store error = %v", err)
	}
	st := store.NewStore(t.TempDir())
	if err := st.Init(); err != nil {
		t.Fatal(err)
	}
	if err := EnsureChapterExpanded(st, 0); !errors.Is(err, errs.ErrToolArgs) {
		t.Fatalf("invalid chapter error = %v", err)
	}
	if err := EnsureChapterExpanded(st, 1); !errors.Is(err, errs.ErrToolPrecondition) {
		t.Fatalf("uninitialized progress error = %v", err)
	}
	if err := st.Progress.Save(&domain.Progress{Phase: domain.PhaseOutline}); err != nil {
		t.Fatal(err)
	}
	if err := EnsureChapterExpanded(st, 1); !errors.Is(err, errs.ErrToolPrecondition) {
		t.Fatalf("wrong phase error = %v", err)
	}
	if err := st.Progress.Save(&domain.Progress{Phase: domain.PhaseWriting}); err != nil {
		t.Fatal(err)
	}
	if err := EnsureChapterExpanded(st, 99); err != nil {
		t.Fatalf("flat writing should allow chapter: %v", err)
	}
	if err := st.Progress.Save(&domain.Progress{Phase: domain.PhaseWriting, Layered: true}); err != nil {
		t.Fatal(err)
	}
	if err := EnsureChapterExpanded(st, 1); !errors.Is(err, errs.ErrToolPrecondition) {
		t.Fatalf("layered missing outline error = %v", err)
	}
	if err := st.Outline.SaveLayeredOutline([]domain.VolumeOutline{{Index: 1, Arcs: []domain.ArcOutline{{Index: 1, Chapters: []domain.OutlineEntry{{Title: "one"}}}}}}); err != nil {
		t.Fatal(err)
	}
	if err := EnsureChapterExpanded(st, 1); err != nil {
		t.Fatalf("expanded chapter rejected: %v", err)
	}
	if err := EnsureChapterExpanded(st, 2); !errors.Is(err, errs.ErrToolPrecondition) {
		t.Fatalf("out-of-range chapter should fail: %v", err)
	}
}

func TestFoundationJSONDiagnostics(t *testing.T) {
	var out map[string]any
	err := decodeFoundationJSON("outline", "{\n  \"title\": \"x\"\n  \"missing\": true\n}", &out)
	if err == nil || !strings.Contains(err.Error(), "parse outline JSON") || !strings.Contains(err.Error(), "line 3") {
		t.Fatalf("syntax diagnostic = %v", err)
	}
	var typed struct {
		Count int `json:"count"`
	}
	err = decodeFoundationJSON("count", `{"count":"not-an-int"}`, &typed)
	if err == nil || !strings.Contains(err.Error(), "parse count JSON") || strings.Contains(err.Error(), "line ") {
		t.Fatalf("type diagnostic = %v", err)
	}
	for _, tc := range []struct {
		s         string
		o         int
		line, col int
	}{
		{"abc", -1, 1, 1},
		{"abc", 99, 1, 4},
		{"a\nb", 2, 2, 1},
	} {
		line, col := offsetToLineCol(tc.s, tc.o)
		if line != tc.line || col != tc.col {
			t.Fatalf("offsetToLineCol(%q,%d) = %d/%d, want %d/%d", tc.s, tc.o, line, col, tc.line, tc.col)
		}
	}
	for _, raw := range []json.RawMessage{nil, json.RawMessage(`"markdown"`), json.RawMessage(`{"title":"x"}`), json.RawMessage(`{`)} {
		_, _ = normalizeFoundationContent(raw)
	}
}

func TestEnsureChapterExpandedRejectsCorruptProgress(t *testing.T) {
	st := store.NewStore(t.TempDir())
	if err := st.Init(); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(st.Dir(), "meta", "progress.json"), []byte("{"), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := EnsureChapterExpanded(st, 1); !errors.Is(err, errs.ErrStoreRead) {
		t.Fatalf("corrupt progress error = %v", err)
	}
}

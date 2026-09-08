package imp

import (
	"strings"
	"testing"
)

func TestSourceDecodingNormalizationAndUnits(t *testing.T) {
	if got, err := decodeSource([]byte("\xef\xbb\xbfhello")); err != nil || got.encoding != encodingUTF8BOM || got.text != "hello" {
		t.Fatalf("BOM decode = %+v/%v", got, err)
	}
	if got := normalize("a\r\nb\rc"); got != "a\nb\nc" {
		t.Fatalf("normalize = %q", got)
	}
	units := buildSourceUnits([]byte("一二三四五\nnext"), 6)
	if len(units) < 2 || units[0].Line != 1 || units[0].Part != 1 || !strings.Contains(units[0].Text, "一") {
		t.Fatalf("units = %+v", units)
	}
	if unitLess(units[0], units[1]) == false {
		t.Fatal("units should be numerically ordered")
	}
}

func TestResolveBoundaryByteAndLabels(t *testing.T) {
	units := map[string]SourceUnit{"L1": {ID: "L1", StartByte: 4, Text: "title body title"}}
	if got, err := resolveBoundaryByte(units, "L1", "body"); err != nil || got != 10 {
		t.Fatalf("anchor offset = %d/%v", got, err)
	}
	for _, anchor := range []string{"missing", "title"} {
		if _, err := resolveBoundaryByte(units, "L1", anchor); err == nil {
			t.Fatalf("anchor %q should fail", anchor)
		}
	}
	if _, err := resolveBoundaryByte(units, "L9", ""); err == nil {
		t.Fatal("unknown unit should fail")
	}
	if got := boundaryLabel(BoundaryDecision{Title: " Title "}); got != "Title" || boundaryLabel(BoundaryDecision{Kind: kindChapter, UnitID: "L1"}) != "chapter@L1" {
		t.Fatal("boundary labels failed")
	}
	if got := previewBoundaries([]BoundaryDecision{{Kind: kindChapter, UnitID: "L1"}}); got != "chapter@L1" {
		t.Fatalf("preview = %q", got)
	}
}

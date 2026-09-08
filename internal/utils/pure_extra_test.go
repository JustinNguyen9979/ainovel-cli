package utils

import "testing"

func TestInputCleaningAndControlDetection(t *testing.T) {
	if got := CleanInputText(" a\n\tb\x00c "); got != " a  bc " {
		t.Fatalf("CleanInputText = %q", got)
	}
	if got := CleanInputLine(" \t hello \n "); got != "hello" {
		t.Fatalf("CleanInputLine = %q", got)
	}
	if got := CleanInputRunes([]rune{'a', '\n', '\t', '\x00', '中'}); got != "a  中" {
		t.Fatalf("CleanInputRunes = %q", got)
	}
	if !ContainsControl("a\n") || ContainsControl("hello") {
		t.Fatal("ContainsControl failed")
	}
}

func TestStreamingFieldExtractionAndFilter(t *testing.T) {
	extractor := NewFieldExtractor("content")
	if got := extractor.Feed(`{"content":"hello `); got != "hello " {
		t.Fatalf("first extractor delta = %q", got)
	}
	if got := extractor.Feed(`world\nthere"}`); got != "world\nthere" {
		t.Fatalf("second extractor delta = %q", got)
	}
	extractor.Reset()
	if got := extractor.Feed(`{"other":"x","content":"ok"}`); got != "ok" {
		t.Fatalf("reset extractor = %q", got)
	}
	filter := NewStreamFilter("content")
	if got := filter.Feed("thinking"); got != ThinkingSep+"thinking" {
		t.Fatalf("text filter = %q", got)
	}
	if got := filter.Feed(`{"content":"answer","nested":{"x":"y"}}`); got != "answer" {
		t.Fatalf("json filter = %q", got)
	}
	if got := filter.Feed("done"); got != ThinkingSep+"done" {
		t.Fatalf("post-json filter = %q", got)
	}
	filter.Reset()
	if got := filter.Feed(`{"content":"a\"b"}`); got != `a"b` {
		t.Fatalf("escaped json filter = %q", got)
	}
}

func TestDecodeTextExtra(t *testing.T) {
	if got := DecodeText([]byte("\ufeffhello")); got != "hello" {
		t.Fatalf("BOM decode = %q", got)
	}
	if got := DecodeText([]byte("普通话")); got != "普通话" {
		t.Fatalf("UTF-8 decode = %q", got)
	}
}

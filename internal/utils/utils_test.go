package utils

import (
	"strings"
	"testing"
)

func TestCleanInput(t *testing.T) {
	if got := CleanInputText("  a\x00b\n c  "); got != "  ab  c  " {
		t.Fatalf("CleanInputText = %q", got)
	}
	if got := CleanInputLine("  a\n\tb  "); got != "a  b" {
		t.Fatalf("CleanInputLine = %q", got)
	}
	if got := CleanInputRunes([]rune{' ', 'a', '\n', 'b', ' '}); got != " a b " {
		t.Fatalf("CleanInputRunes = %q", got)
	}
	if !ContainsControl("a\x01b") || !ContainsControl("a\nb") {
		t.Fatal("ContainsControl classification is wrong")
	}
}

func TestParseLanguageAndTranslation(t *testing.T) {
	for _, tc := range []struct {
		value string
		want  Language
		err   bool
	}{
		{"vi", LanguageVI, false}, {"VI", LanguageVI, false}, {"vi-vn", LanguageVI, false}, {"vietnamese", LanguageVI, false},
		{"zh", LanguageZH, false}, {"zh-cn", LanguageZH, false}, {"zh-hans", LanguageZH, false}, {"chinese", LanguageZH, false},
		{"", LanguageVI, false}, {"fr", "", true},
	} {
		got, err := ParseLanguage(tc.value)
		if got != tc.want || (err != nil) != tc.err {
			t.Fatalf("ParseLanguage(%q) = %q, %v", tc.value, got, err)
		}
	}
	if got := T(LanguageVI, MsgLanguage); got == "" || got == T(LanguageZH, MsgLanguage) {
		t.Fatalf("localized language labels are not distinct: %q", got)
	}
	if got := T(LanguageVI, MsgUnknownCommand, "demo"); !strings.Contains(got, "demo") {
		t.Fatalf("formatted translation = %q", got)
	}
	if got := T(Language("fr"), MsgReady); got == "" {
		t.Fatal("unknown language should fall back to Vietnamese")
	}
}

func TestDecodeText(t *testing.T) {
	if got := DecodeText([]byte("Xin chào")); got != "Xin chào" {
		t.Fatalf("UTF-8 decode = %q", got)
	}
	if got := DecodeText(append([]byte{0xef, 0xbb, 0xbf}, []byte("BOM")...)); got != "BOM" {
		t.Fatalf("BOM decode = %q", got)
	}
}

func TestJSONFieldExtractorAcrossDeltas(t *testing.T) {
	ext := NewFieldExtractor("content")
	if got := ext.Feed(`{"con`); got != "" {
		t.Fatalf("partial key = %q", got)
	}
	if got := ext.Feed(`tent":"A\nB"}`); got != "A\nB" {
		t.Fatalf("extracted content = %q", got)
	}
	ext.Reset()
	if got := ext.Feed(`{"content":"next"}`); got != "next" {
		t.Fatalf("reset extraction = %q", got)
	}
	if got := NewFieldExtractor("content").Feed(`{"content":"A\\\"B"}`); got != `A\"B` {
		t.Fatalf("escaped extraction = %q", got)
	}
}

func TestStreamFilter(t *testing.T) {
	filter := NewStreamFilter("content")
	got := filter.Feed("thinking {\"content\":\"answer\",\"nested\":{\"x\":1}} done")
	if !strings.Contains(got, ThinkingSep) || !strings.Contains(got, "thinking") || !strings.Contains(got, "answer") || !strings.Contains(got, " done") {
		t.Fatalf("filtered stream = %q", got)
	}
	filter.Reset()
	if got := filter.Feed("plain"); !strings.Contains(got, "plain") {
		t.Fatalf("reset stream = %q", got)
	}
	if got := NewStreamFilter("content").Feed(`{"other":"value"}`); got != "" {
		t.Fatalf("non-target JSON = %q", got)
	}
}

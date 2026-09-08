package models

import "testing"

func TestModelLookupNormalizationAndDatedSuffixes(t *testing.T) {
	cases := []struct {
		name    string
		left    string
		right   string
		matches bool
	}{
		{"exact", "Claude.Sonnet-4", "claude-sonnet-4", true},
		{"dated target", "claude-sonnet-4", "claude-sonnet-4-20250514", true},
		{"dated known", "claude-sonnet-4-20250514", "claude-sonnet-4", true},
		{"bad date", "claude-sonnet-4", "claude-sonnet-4-2025", false},
		{"different", "gpt-5", "gpt-4", false},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := SameModelID(tc.left, tc.right); got != tc.matches {
				t.Fatalf("SameModelID(%q, %q) = %v", tc.left, tc.right, got)
			}
		})
	}
	if hasDatedSuffix("model-20250514") == false || hasDatedSuffix("model-2025") || hasDatedSuffix("model") {
		t.Fatal("dated suffix detection is wrong")
	}
}

func TestLookupModelEntryFiltersProviderAndMatchesDate(t *testing.T) {
	entries := []ModelEntry{
		{Provider: "anthropic", ID: "claude-sonnet-4-20250514", Name: "Sonnet"},
		{Provider: "openai", ID: "gpt-5", Name: "GPT"},
	}
	if got, ok := lookupModelEntry(entries, "anthropic", "claude-sonnet-4"); !ok || got.Name != "Sonnet" {
		t.Fatalf("dated lookup = %+v/%v", got, ok)
	}
	if _, ok := lookupModelEntry(entries, "gemini", "gpt-5"); ok {
		t.Fatal("provider mismatch should not match")
	}
	if got, ok := lookupModelEntry(entries, "", "gpt-5"); !ok || got.Provider != "openai" {
		t.Fatalf("provider-free lookup = %+v/%v", got, ok)
	}
}

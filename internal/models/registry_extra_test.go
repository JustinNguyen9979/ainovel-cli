package models

import "testing"

func TestDefaultRegistryIsStable(t *testing.T) {
	first := DefaultRegistry()
	second := DefaultRegistry()
	if first == nil || second == nil || first != second {
		t.Fatal("default registry should be stable")
	}
}

func TestRegistryContextWindowLookup(t *testing.T) {
	r := NewModelRegistry()
	if r.ResolveContextWindow("gpt-5") <= 0 || r.ResolveContextWindow("missing") != 0 {
		t.Fatal("context window lookup mismatch")
	}
}

func TestModelRegistryResolveListAndMerge(t *testing.T) {
	r := NewModelRegistry()
	if entry, ok := r.Resolve("google/gemini-2.5-pro"); !ok || entry.Provider != "gemini" {
		t.Fatalf("vendor resolve = %+v/%v", entry, ok)
	}
	if entry, ok := r.Resolve("claude-sonnet-4"); !ok || entry.ID != "claude-sonnet-4" {
		t.Fatalf("base model resolve = %+v/%v", entry, ok)
	}
	if entry, ok := r.Resolve("sonnet 4"); !ok || entry.Name == "" {
		t.Fatalf("partial name resolve = %+v/%v", entry, ok)
	}
	if _, ok := r.Resolve(""); ok || r.ResolveContextWindow("unknown") != 0 {
		t.Fatal("empty/unknown model should not resolve")
	}
	all := r.List("")
	if len(all) == 0 || len(r.List("gemini")) == 0 || len(r.List("not-a-model")) != 0 {
		t.Fatal("registry filtering is wrong")
	}
	before, ok := r.Resolve("gpt-5")
	if !ok {
		t.Fatal("baseline gpt-5 missing")
	}
	r.MergeModels([]ModelEntry{{Provider: before.Provider, ID: before.ID, Name: "Updated", ContextWindow: 123, MaxTokens: 456, InputCostPer1M: 1, OutputCostPer1M: 2}, {Provider: "test", ID: "new", Name: "New"}})
	after, ok := r.Resolve("gpt-5")
	if !ok || after.Name != "Updated" || after.ContextWindow != 123 || after.MaxTokens != 456 {
		t.Fatalf("merged model = %+v", after)
	}
	if entry, ok := r.Resolve("test/new"); !ok || entry.Name != "New" {
		t.Fatalf("new model = %+v/%v", entry, ok)
	}
}

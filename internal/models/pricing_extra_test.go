package models

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestPricingCacheAndConversionHelpers(t *testing.T) {
	if loadCache("") != nil {
		t.Fatal("empty cache dir should be nil")
	}
	dir := t.TempDir()
	if got := loadCache(dir); got != nil {
		t.Fatalf("missing cache = %#v", got)
	}
	entries := []ModelEntry{{Provider: "openai", ID: "model", ContextWindow: 123}}
	path := filepath.Join(dir, cacheFileName)
	data, err := json.Marshal(modelCache{FetchedAt: time.Now(), Models: entries})
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, data, 0o600); err != nil {
		t.Fatal(err)
	}
	loaded := loadCache(dir)
	if len(loaded) != 1 || loaded[0].ID != "model" {
		t.Fatalf("loaded cache = %#v", loaded)
	}
	if err := os.WriteFile(path, []byte("{"), 0o600); err != nil {
		t.Fatal(err)
	}
	if loadCache(dir) != nil {
		t.Fatal("invalid cache should be ignored")
	}
	stale, _ := json.Marshal(modelCache{FetchedAt: time.Now().Add(-48 * time.Hour), Models: entries})
	if err := os.WriteFile(path, stale, 0o600); err != nil {
		t.Fatal(err)
	}
	if loadCache(dir) != nil {
		t.Fatal("stale cache should be ignored")
	}
	saveCache(entries, dir)
	if loadCache(dir) == nil {
		t.Fatal("saved cache should be readable")
	}
	saveCache(entries, "")

	for _, tc := range []struct {
		input string
		want  float64
	}{{"", 0}, {"bad", 0}, {"-1", 0}, {"0.000002", 2}, {"0", 0}} {
		if got := tokenToMillion(tc.input); got != tc.want {
			t.Errorf("tokenToMillion(%q) = %v, want %v", tc.input, got, tc.want)
		}
	}
	if cleanModelName("Provider: Display") != "Display" || cleanModelName("plain") != "plain" {
		t.Fatal("clean model name mismatch")
	}
	for _, created := range []int64{0, -1, time.Now().Add(-800 * 24 * time.Hour).Unix()} {
		if !isStaleModel(created) {
			t.Errorf("created=%d should be stale", created)
		}
	}
	if isStaleModel(time.Now().Unix()) {
		t.Fatal("current model should not be stale")
	}
}

func TestConvertModelFiltersAndMaps(t *testing.T) {
	valid, ok := convertModel(openRouterModel{ID: "openai/test", Name: "OpenAI: Test", ContextLength: 1000, Created: time.Now().Unix(), Pricing: &openRouterPricing{Prompt: "0.000001", Completion: "0.000002", InputCacheRead: "0.0000005", InputCacheWrite: "0.000003"}, TopProvider: &openRouterTopProvider{MaxCompletionTokens: 500}})
	if !ok || valid.Provider != "openai" || valid.ID != "test" || valid.Name != "Test" || valid.InputCostPer1M != 1 || valid.MaxTokens != 500 {
		t.Fatalf("converted model = %#v/%v", valid, ok)
	}
	for _, model := range []openRouterModel{{ID: "invalid"}, {ID: "unknown/test"}, {ID: "openai/test:free"}, {ID: "openai/test", Created: 0}} {
		if _, ok := convertModel(model); ok {
			t.Fatalf("model should be filtered: %+v", model)
		}
	}
}

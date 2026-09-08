package store

import (
	"os"
	"path/filepath"
	"testing"
)

type appendLogValue struct {
	ID    string `json:"id"`
	Value string `json:"value"`
}

func TestAppendLogLegacyDedupAndTailRepair(t *testing.T) {
	dir := t.TempDir()
	io := newIO(dir)
	log := newAppendLog("meta/items.jsonl", "meta/items.json", func(v appendLogValue) string { return v.ID }, func(v appendLogValue) appendLogValue { return v })
	if err := io.WriteJSON("meta/items.json", []appendLogValue{{ID: "a", Value: "old"}}); err != nil {
		t.Fatal(err)
	}
	added, err := log.appendUnlocked(io, []appendLogValue{{ID: "a", Value: "dup"}, {ID: "b", Value: "new"}, {ID: "b", Value: "newer"}})
	if err != nil || len(added) != 1 || added[0].ID != "b" {
		t.Fatalf("legacy append = %+v/%v", added, err)
	}
	if _, err := os.Stat(filepath.Join(dir, "meta/items.json")); !os.IsNotExist(err) {
		t.Fatalf("legacy file should be removed: %v", err)
	}
	all, err := log.allUnlocked(io)
	if err != nil || len(all) != 2 {
		t.Fatalf("all values = %+v/%v", all, err)
	}
	if err := io.AppendLine("meta/items.jsonl", []byte(`{"id":"c","value":"partial"}`)); err != nil {
		t.Fatal(err)
	}
	log.reset()
	all, err = log.allUnlocked(io)
	if err != nil || len(all) != 2 {
		t.Fatalf("tail repair values = %+v/%v", all, err)
	}
}

func TestAppendLogMalformedAndReplace(t *testing.T) {
	dir := t.TempDir()
	io := newIO(dir)
	log := newAppendLog("items.jsonl", "items.json", func(v appendLogValue) string { return v.ID }, func(v appendLogValue) appendLogValue { return v })
	if err := io.WriteFileUnlocked("items.jsonl", []byte("{bad}\n")); err != nil {
		t.Fatal(err)
	}
	if _, err := log.allUnlocked(io); err == nil {
		t.Fatal("malformed committed JSONL should fail")
	}
	log.reset()
	if err := log.replaceUnlocked(io, []appendLogValue{{ID: "x", Value: "value"}}); err != nil {
		t.Fatal(err)
	}
	all, err := log.allUnlocked(io)
	if err != nil || len(all) != 1 || all[0].ID != "x" {
		t.Fatalf("replaced values = %+v/%v", all, err)
	}
}

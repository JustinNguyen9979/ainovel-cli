package startup

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestLoadPromptFile(t *testing.T) {
	path := filepath.Join(t.TempDir(), "prompt.txt")
	if err := os.WriteFile(path, []byte("  write a chapter  \n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if got, err := LoadPromptFile(path); err != nil || got != "write a chapter" {
		t.Fatalf("prompt = %q, %v", got, err)
	}
	if _, err := LoadPromptFile(filepath.Join(t.TempDir(), "missing.txt")); err == nil || !strings.Contains(err.Error(), "prompt") {
		t.Fatalf("missing prompt error = %v", err)
	}
}

func TestPrepareQuick(t *testing.T) {
	if got, err := PrepareQuick("  start the story \n"); err != nil || got != "start the story" {
		t.Fatalf("prepared prompt = %q, %v", got, err)
	}
	if got, err := PrepareQuick(" \n\t"); err == nil || got != "" {
		t.Fatalf("empty prompt = %q, %v", got, err)
	}
}

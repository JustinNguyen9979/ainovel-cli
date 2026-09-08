package rules

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestEnsureHomeRulesDirCreatesReadme(t *testing.T) {
	dir := filepath.Join(t.TempDir(), "rules")
	if err := ensureRulesDirAt(dir); err != nil {
		t.Fatal(err)
	}
	data, err := os.ReadFile(filepath.Join(dir, "README.txt"))
	if err != nil || !strings.Contains(string(data), "全局写作偏好") {
		t.Fatalf("readme = %q/%v", data, err)
	}
}

func TestEnsureHomeRulesDirHonorsHome(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	t.Setenv("USERPROFILE", home)
	EnsureHomeRulesDir()
	if _, err := os.Stat(filepath.Join(home, ".ainovel", "rules", "README.txt")); err != nil {
		t.Fatal(err)
	}
}

func TestEnsureHomeRulesDirReadmeIsTxt(t *testing.T) {
	dir := filepath.Join(t.TempDir(), ".ainovel", "rules")
	if err := ensureRulesDirAt(dir); err != nil {
		t.Fatal(err)
	}
	entries, err := os.ReadDir(dir)
	if err != nil || len(entries) != 1 || entries[0].Name() != "README.txt" {
		t.Fatalf("entries = %#v/%v", entries, err)
	}
}

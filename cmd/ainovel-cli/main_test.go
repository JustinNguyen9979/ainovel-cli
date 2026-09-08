package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestParseCLIOptions(t *testing.T) {
	cases := []struct {
		name string
		args []string
		want cliOptions
		rest []string
		bad  bool
	}{
		{"version", []string{"--version"}, cliOptions{Version: true}, nil, false},
		{"short version", []string{"-v"}, cliOptions{Version: true}, nil, false},
		{"version subcommand", []string{"version"}, cliOptions{Version: true}, nil, false},
		{"version argument", []string{"version", "extra"}, cliOptions{}, nil, true},
		{"headless prompt language", []string{"--headless", "--prompt", "hello", "--language", "vi"}, cliOptions{Headless: true, Prompt: "hello", Language: "vi"}, nil, false},
		{"language alias", []string{"--headless", "--lang", "zh"}, cliOptions{Headless: true, Language: "zh"}, nil, false},
		{"update", []string{"update", "v1.2.3"}, cliOptions{Update: true, UpdateVersion: "v1.2.3"}, nil, false},
		{"update latest", []string{"update"}, cliOptions{Update: true}, nil, false},
		{"update extra update", []string{"update", "v1.2.3", "update"}, cliOptions{}, nil, true},
		{"update flag argument", []string{"update", "--headless"}, cliOptions{}, nil, true},
		{"update extra argument", []string{"update", "v1.2.3", "extra"}, cliOptions{}, nil, true},
		{"positional", []string{"extra"}, cliOptions{}, []string{"extra"}, false},
		{"duplicate prompt", []string{"--prompt", "a", "--prompt-file", "b"}, cliOptions{}, nil, true},
		{"version mixed", []string{"--version", "--language", "vi"}, cliOptions{}, nil, true},
		{"update mixed", []string{"update", "--prompt", "a"}, cliOptions{}, nil, true},
		{"missing prompt", []string{"--prompt"}, cliOptions{}, nil, true},
		{"missing prompt file", []string{"--prompt-file"}, cliOptions{}, nil, true},
		{"missing language", []string{"--language"}, cliOptions{}, nil, true},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got, rest, err := parseCLIOptions(tc.args)
			if (err != nil) != tc.bad {
				t.Fatalf("parse error = %v", err)
			}
			if err == nil && (got != tc.want || strings.Join(rest, "\x00") != strings.Join(tc.rest, "\x00")) {
				t.Fatalf("got options=%+v rest=%v", got, rest)
			}
		})
	}
}

func TestLoadPromptFrom(t *testing.T) {
	if got, err := loadPromptFrom(cliOptions{Prompt: "  hello  "}, strings.NewReader("")); err != nil || got != "hello" {
		t.Fatalf("prompt = %q, %v", got, err)
	}
	if got, err := loadPromptFrom(cliOptions{PromptFile: "-"}, strings.NewReader("  stdin prompt\n")); err != nil || got != "stdin prompt" {
		t.Fatalf("stdin prompt = %q, %v", got, err)
	}
	if _, err := loadPromptFrom(cliOptions{PromptFile: "/missing/prompt"}, strings.NewReader("")); err == nil {
		t.Fatal("missing prompt file should fail")
	}
	dir := t.TempDir()
	path := filepath.Join(dir, "prompt.txt")
	if err := os.WriteFile(path, []byte("  file prompt\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	if got, err := loadPromptFrom(cliOptions{PromptFile: path}, strings.NewReader("")); err != nil || got != "file prompt" {
		t.Fatalf("file prompt = %q, %v", got, err)
	}
	if _, err := loadPromptFrom(cliOptions{PromptFile: "-"}, failingReader{}); err == nil {
		t.Fatal("stdin read failure should be returned")
	}
}

type failingReader struct{}

func (failingReader) Read([]byte) (int, error) { return 0, os.ErrInvalid }

func TestVersionInfoDefaults(t *testing.T) {
	info := versionInfo()
	if info.Version == "" || info.Commit == "" || info.Date == "" {
		t.Fatalf("version info = %+v", info)
	}
}

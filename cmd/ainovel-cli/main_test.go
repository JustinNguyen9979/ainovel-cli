package main

import (
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
		{"headless prompt language", []string{"--headless", "--prompt", "hello", "--language", "vi"}, cliOptions{Headless: true, Prompt: "hello", Language: "vi"}, nil, false},
		{"update", []string{"update", "v1.2.3"}, cliOptions{Update: true, UpdateVersion: "v1.2.3"}, nil, false},
		{"positional", []string{"extra"}, cliOptions{}, []string{"extra"}, false},
		{"duplicate prompt", []string{"--prompt", "a", "--prompt-file", "b"}, cliOptions{}, nil, true},
		{"version mixed", []string{"--version", "--language", "vi"}, cliOptions{}, nil, true},
		{"missing prompt", []string{"--prompt"}, cliOptions{}, nil, true},
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
}

func TestVersionInfoDefaults(t *testing.T) {
	info := versionInfo()
	if info.Version == "" || info.Commit == "" || info.Date == "" {
		t.Fatalf("version info = %+v", info)
	}
}

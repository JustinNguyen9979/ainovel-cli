package assets

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestBuildWriterPrompt_AssemblesProperly(t *testing.T) {
	protocol := mustRead(promptsFS, "prompts/writer.md")
	voice := mustRead(voiceFS, "voice.md")
	style := "## Custom style\n\n- Short prose"
	got := BuildWriterPrompt("protocol {{VOICE}}", voice, style)
	if !strings.Contains(got, "protocol") || !strings.Contains(got, strings.TrimSpace(voice)) || !strings.Contains(got, style) || strings.Contains(got, voicePlaceholder) {
		t.Fatal("writer prompt assembly is incomplete")
	}

	if !strings.Contains(protocol, "commit_chapter") {
		t.Fatal("writer protocol is incomplete")
	}
}

func TestLoad_NoOverrides(t *testing.T) {
	b := Load("default", LoadOptions{})
	if b.Voice != mustRead(voiceFS, "voice.md") || b.References.AntiAITone != mustRead(referencesFS, "references/anti-ai-tone.md") {
		t.Fatal("built-in assets changed without overrides")
	}
	if _, ok := b.Styles["default"]; !ok {
		t.Fatal("default style is missing")
	}
}

func TestInterventionPromptsKeepScopeContract(t *testing.T) {
	prompt := loadPrompts().ArbiterIntervention
	for _, phrase := range []string{"JSON", "answer", "reason"} {
		if !strings.Contains(prompt, phrase) {
			t.Fatalf("intervention prompt missing %q", phrase)
		}
	}

	if !strings.Contains(loadPrompts().ArbiterPlanStart, "JSON") {
		t.Fatal("plan-start prompt should describe its structured output")
	}
	if !strings.Contains(loadPrompts().ArbiterFailure, "JSON") {
		t.Fatal("failure prompt should describe its structured output")
	}
}

func TestStructuredArbiterPromptsContainOnlySemantics(t *testing.T) {
	prompts := loadPrompts()
	for name, prompt := range map[string]string{"plan_start": prompts.ArbiterPlanStart, "failure": prompts.ArbiterFailure} {
		if strings.TrimSpace(prompt) == "" {
			t.Fatalf("%s prompt is empty", name)
		}
	}
}

func writeFile(t *testing.T, path, content string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatalf("write: %v", err)
	}
}

func TestLoad_ThreeTierAppendAndReplace(t *testing.T) {
	home, book := t.TempDir(), t.TempDir()
	opts := LoadOptions{HomeStyleDir: home, BookStyleDir: book}
	writeFile(t, filepath.Join(home, "voice.md"), "Global voice")
	writeFile(t, filepath.Join(book, "voice.md"), "Book voice")
	writeFile(t, filepath.Join(book, "anti-ai-tone.md"), "Book anti-tone")
	writeFile(t, filepath.Join(home, "styles", "fantasy.md"), "Global fantasy")
	writeFile(t, filepath.Join(book, "styles", "xianxia.md"), "Book xianxia")
	writeFile(t, filepath.Join(book, "styles", "Bad Name!.md"), "Invalid")
	writeFile(t, filepath.Join(home, "genres", "fantasy", "style-references.md"), "Global references")
	writeFile(t, filepath.Join(book, "genres", "fantasy", "style-references.md"), "Book references")

	b := Load("fantasy", opts)
	if !strings.Contains(b.Voice, "Global voice") || !strings.Contains(b.Voice, "Book voice") || !strings.Contains(b.References.AntiAITone, "Book anti-tone") {
		t.Fatal("style overrides missing")
	}
	if b.Styles["fantasy"] != "Global fantasy" || b.Styles["xianxia"] != "Book xianxia" {
		t.Fatal("style override precedence is wrong")
	}
	if _, ok := b.Styles["Bad Name!"]; ok || b.References.StyleReference != "Book references" {
		t.Fatal("invalid style or genre precedence is wrong")
	}
}

func TestLoad_BookOverridesHomeOnStyles(t *testing.T) {
	home, book := t.TempDir(), t.TempDir()
	writeFile(t, filepath.Join(home, "styles", "romance.md"), "Global")
	writeFile(t, filepath.Join(book, "styles", "romance.md"), "Book")
	if got := Load("default", LoadOptions{HomeStyleDir: home, BookStyleDir: book}).Styles["romance"]; got != "Book" {
		t.Fatalf("book style should override home style: %q", got)
	}
}

func TestOverrideVoice_SharesAssemblyPath(t *testing.T) {
	b := Load("default", LoadOptions{})
	b.OverrideVoice("## Experimental voice\n\n- Short sentences")
	got := BuildWriterPrompt("protocol {{VOICE}} with commit_chapter", b.Voice, "")
	if !strings.Contains(got, "Experimental voice") || strings.Contains(got, voicePlaceholder) || !strings.Contains(got, "commit_chapter") {
		t.Fatal("voice override broke writer prompt assembly")
	}
}

package eval

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/JustinNguyen9979/ainovel-cli/assets"
	"github.com/JustinNguyen9979/ainovel-cli/internal/diag"
	"github.com/JustinNguyen9979/ainovel-cli/internal/stylestat"
	"github.com/voocel/agentcore"
)

func TestEvalCaseLoadingAndVariantHelpers(t *testing.T) {
	if orNone("") != "<none>" || orNone("variant") != "variant" {
		t.Fatal("orNone mismatch")
	}
	if _, err := loadVariant(""); err != nil {
		t.Fatalf("empty variant: %v", err)
	}
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "writer.md"), []byte("writer prompt"), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "README.txt"), []byte("ignore"), 0o600); err != nil {
		t.Fatal(err)
	}
	prompts, err := loadVariant(dir)
	if err != nil || prompts["writer.md"] != "writer prompt" {
		t.Fatalf("variant = %#v, %v", prompts, err)
	}
	if _, err := loadVariant(t.TempDir()); err == nil {
		t.Fatal("variant without markdown should fail")
	}
	if _, err := loadVariant(filepath.Join(dir, "missing")); err == nil {
		t.Fatal("missing variant directory should fail")
	}

	c := Case{ID: "case", Prompt: "prompt"}
	if err := c.Validate(); err != nil {
		t.Fatal(err)
	}
	if err := c.Validate(); err != nil || c.Gate.MaxSeverity != "warning" || c.Gate.StylestatRegression != "warn" {
		t.Fatalf("case defaults = %+v, %v", c.Gate, err)
	}
	for _, bad := range []Case{{ID: "", Prompt: "x"}, {ID: "case", Prompt: ""}, {ID: "case", Prompt: "x", Gate: Gate{MaxSeverity: "bad"}}} {
		if err := bad.Validate(); err == nil {
			t.Fatalf("invalid case accepted: %+v", bad)
		}
	}
}

func TestLoadCasesRejectsMalformedAndDuplicateFiles(t *testing.T) {
	dir := t.TempDir()
	valid := `{"id":"case-a","prompt":"write"}`
	if err := os.WriteFile(filepath.Join(dir, "a.json"), []byte(valid), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "b.json"), []byte(`{"id":"case-b","prompt":"write","unexpected":true}`), 0o600); err != nil {
		t.Fatal(err)
	}
	if _, err := LoadCases(dir); err == nil {
		t.Fatal("unknown case fields should be rejected")
	}
	if err := os.Remove(filepath.Join(dir, "b.json")); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "b.json"), []byte(valid), 0o600); err != nil {
		t.Fatal(err)
	}
	if _, err := LoadCases(dir); err == nil || !strings.Contains(err.Error(), "重复") {
		t.Fatalf("duplicate case id should be rejected, got %v", err)
	}
	if err := os.Remove(filepath.Join(dir, "a.json")); err != nil {
		t.Fatal(err)
	}
	if err := os.Remove(filepath.Join(dir, "b.json")); err != nil {
		t.Fatal(err)
	}
	if _, err := LoadCases(dir); err == nil {
		t.Fatal("empty case directory should be rejected")
	}
}

func TestApplyVariantAndToolCallParsing(t *testing.T) {
	bundle := assets.Load("default", assets.LoadOptions{})
	if err := applyVariant(&bundle, map[string]string{"voice.md": "custom voice"}); err != nil {
		t.Fatal(err)
	}
	if bundle.Voice != "custom voice" {
		t.Fatalf("voice override = %q", bundle.Voice)
	}
	if err := applyVariant(&bundle, map[string]string{"writer.md": "writer {{VOICE}}"}); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(bundle.Prompts.Writer, "仿写画像") || !strings.Contains(bundle.Prompts.Writer, "writer {{VOICE}}") {
		t.Fatal("writer variant should receive simulation guidance")
	}
	if err := applyVariant(&bundle, map[string]string{"unsupported.md": "bad"}); err == nil {
		t.Fatal("unsupported prompt override should fail")
	}

	dir := t.TempDir()
	path := filepath.Join(dir, "session.jsonl")
	msg := agentcore.Message{Role: agentcore.RoleAssistant, Content: []agentcore.ContentBlock{
		agentcore.ToolCallBlock(agentcore.ToolCall{Name: "one"}),
		agentcore.ToolCallBlock(agentcore.ToolCall{Name: "two"}),
	}}
	data, err := json.Marshal(msg)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, append(data, '\n'), 0o600); err != nil {
		t.Fatal(err)
	}
	if got, err := countToolCallsInFile(path); err != nil || got != 2 {
		t.Fatalf("tool calls = %d/%v", got, err)
	}
	bad := filepath.Join(dir, "bad.jsonl")
	if err := os.WriteFile(bad, []byte("{\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	if _, err := countToolCallsInFile(bad); err == nil {
		t.Fatal("malformed session line should fail")
	}
}

func TestGradeDeltaCoversFailureAndTokenThresholds(t *testing.T) {
	base := cleanResult()
	base.Metrics.ToolCalls = 10
	base.Metrics.Usage = UsageMetrics{CostUSD: 1, Input: 100, Output: 100}
	variant := base
	variant.Outcome = Fail
	variant.Metrics.CompletedChapters = 0
	variant.Metrics.CriticalFindings = 1
	variant.Metrics.WarningFindings = 1
	variant.Metrics.ToolCalls = 20
	variant.Metrics.Usage = UsageMetrics{CostUSD: 2, Input: 200, Output: 200}

	c := writerSmokeCase()
	c.Gate.MaxToolCallDeltaRatio = float64Ptr(0.3)
	c.Gate.MaxCostDeltaRatio = float64Ptr(0.3)
	d := GradeDelta(c, base, variant)
	if d.Outcome != Fail || len(d.HardFails) < 3 || len(d.Warnings) < 3 {
		t.Fatalf("delta failure thresholds = %+v", d)
	}
	if !hasIssue(d.HardFails, "variant", "自身门禁失败") || !hasIssue(d.HardFails, "delta:completed_chapters", "减少") {
		t.Fatalf("delta hard failures = %+v", d.HardFails)
	}

	styleBase := cleanResult()
	styleVariant := cleanResult()
	styleBase.Metrics.Stylestat = &stylestat.Stats{}
	styleVariant.Metrics.Stylestat = &stylestat.Stats{Ending: stylestat.EndingStat{ShortRatio: 1}}
	c.Gate.StylestatRegression = "off"
	d = GradeDelta(c, styleBase, styleVariant)
	if d.Outcome != Pass {
		t.Fatalf("stylestat off should not change a clean result, got %+v", d)
	}
}

func TestFirstMarkdownTitleAndCheckpointContracts(t *testing.T) {
	if firstMarkdownTitle("\n  ## Title\nbody") != "Title" || firstMarkdownTitle("\n\t") != "" {
		t.Fatal("markdown title fallback mismatch")
	}
	col := cleanCollected()
	for _, spec := range []string{"chapter:bad:commit", "arc:1:bad:review", "volume:bad:summary", "global:a:b", "unknown:x"} {
		if _, err := col.HasCheckpoint(spec); err == nil {
			t.Fatalf("invalid checkpoint contract accepted: %s", spec)
		}
	}
}

func TestEvalReportRenderingAndWriteBranches(t *testing.T) {
	caseDef := Case{ID: "render_case", Category: "workflow", Role: "writer"}
	base := Result{CaseID: caseDef.ID, Category: caseDef.Category, Role: caseDef.Role, Outcome: Pass, Dir: "/tmp/base", Metrics: Metrics{Phase: "writing", Flow: "writing", CompletedChapters: 2, TotalChapters: 3, TotalWords: 2400, CriticalFindings: 1, WarningFindings: 2, ToolCalls: 4, StylestatStatus: "ok", Usage: UsageMetrics{UsageRecorded: true, CostUSD: .1234, Input: 100, Output: 80}}}
	base.HardFails = []Issue{{Kind: "hard_fail", Source: "runtime", Detail: "runtime"}}
	base.Warnings = []Issue{{Kind: "warning", Source: "finding", Severity: "warning", Detail: "warning"}}
	base.Notes = []Issue{{Kind: "note", Source: "note", Detail: "note"}}
	base.Passed = []Issue{{Kind: "passed", Source: "contract", Detail: "passed"}}
	variant := base
	variant.Arm = ArmVariant
	variant.Repeat = 2
	delta := Delta{Outcome: Warn, Metrics: DeltaMetrics{CompletedChapters: 1, CriticalFindings: 0, WarningFindings: 1, TotalWordsRatio: 1.2, ToolCallDeltaRatio: .3, CostDeltaRatio: .4, Stylestat: &StyleDelta{Status: "ok", PatternTopPerChapter: .2, EndingShortRatio: .1, RepeatedSentences: 1, TitleMixedDelta: 2}}, Warnings: []Issue{{Source: "delta", Detail: "delta warning"}}, Notes: []Issue{{Source: "delta-note", Detail: "delta note"}}}
	suite := Aggregate("run", "ab", "variant-x", 2, []CaseResult{NewABCaseResult(caseDef, []RunResult{{Arm: ArmBaseline, Repeat: 1, Result: base}, {Arm: ArmVariant, Repeat: 2, Result: variant}}, []Delta{delta})})
	out := t.TempDir()
	if err := WriteReport(suite, out); err != nil {
		t.Fatal(err)
	}
	for _, name := range []string{"report.json", "report.md"} {
		data, err := os.ReadFile(filepath.Join(out, name))
		if err != nil || len(data) == 0 {
			t.Fatalf("report %s = %d/%v", name, len(data), err)
		}
	}
	if markdown := string(mustReadReport(t, filepath.Join(out, "report.md"))); !strings.Contains(markdown, "delta#1") || !strings.Contains(markdown, "Hard Fail") || !strings.Contains(markdown, "cost=$0.1234") {
		t.Fatalf("rich report missing sections: %s", markdown)
	}
}

func mustReadReport(t *testing.T, path string) []byte {
	t.Helper()
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	return data
}

func TestEvalReportAndSeverityHelpers(t *testing.T) {
	if ratio(1, 0) != 0 || ratio(3, 2) != 1.5 || deltaRatio(1, 0) != 0 || deltaRatio(3, 2) != 0.5 || deltaRatioFloat(1, 0) != 0 || deltaRatioFloat(3, 2) != 0.5 {
		t.Fatal("ratio helpers mismatch")
	}
	if round2(-1.236) != -1.24 || severityRank("unknown") != 99 || !validSeverity("critical") || validSeverity("bad") {
		t.Fatal("numeric/severity helpers mismatch")
	}
	c := Case{ID: "case", Category: "smoke", Role: "writer"}
	r := Result{CaseID: "case", Outcome: Pass, Metrics: Metrics{Phase: "writing", Flow: "writing", CompletedChapters: 1, TotalChapters: 2, TotalWords: 500, ToolCalls: 2, Usage: UsageMetrics{UsageRecorded: true, CostUSD: 0.5, Input: 100, Output: 20}, StylestatStatus: "ok"}, Dir: "/tmp/out"}
	d := Delta{Outcome: Warn, Warnings: []Issue{{Source: "delta", Detail: "warning"}}}
	cr := NewSingleCaseResult(c, r)
	if cr.Outcome != Pass || len(cr.Runs) != 1 {
		t.Fatalf("single case = %+v", cr)
	}
	suite := Aggregate("run", "single", "", 0, []CaseResult{cr})
	if suite.Repeat != 1 || Summary(suite) == "" {
		t.Fatal("suite summary mismatch")
	}
	cr = NewABCaseResult(c, []RunResult{{Arm: ArmBaseline, Repeat: 2, Result: r}}, []Delta{d})
	if !strings.Contains(Summary(Aggregate("run", "ab", "v", 1, []CaseResult{cr})), "cases") {
		t.Fatal("AB summary missing")
	}
	out := t.TempDir()
	if err := WriteReport(suite, out); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(filepath.Join(out, "report.md")); err != nil {
		t.Fatal(err)
	}
	if findingDetail(diag.Finding{Title: "title", Evidence: "evidence"}) != "title（evidence）" || findingDetail(diag.Finding{Title: "title"}) != "title" {
		t.Fatal("finding detail mismatch")
	}
}

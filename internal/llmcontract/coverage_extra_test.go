package llmcontract

import (
	"strings"
	"testing"
)

func TestStrictSchemaValidationHelpers(t *testing.T) {
	valid := map[string]any{"type": "object", "properties": map[string]any{"name": map[string]any{"type": "string"}}, "required": []string{"name"}}
	if err := ValidateStrictReady(valid); err != nil {
		t.Fatal(err)
	}
	invalid := map[string]any{"type": "object", "properties": map[string]any{"name": map[string]any{"type": "string"}}, "required": []string{}}
	if err := ValidateStrictReady(invalid); err == nil || !strings.Contains(err.Error(), "required") {
		t.Fatal("missing required field should fail")
	}
	array := map[string]any{"type": "array", "items": invalid}
	if err := ValidateStrictReady(array); err == nil {
		t.Fatal("invalid array item schema should fail")
	}
	if !typeIncludes("object", "object") || !typeIncludes([]string{"string", "null"}, "null") || typeIncludes("string", "object") || typeIncludes(nil, "object") {
		t.Fatal("typeIncludes mismatch")
	}
	if got := Nullable(map[string]any{"type": "string"}); len(got["type"].([]string)) != 2 {
		t.Fatal("nullable type mismatch")
	}
	if got := Nullable(map[string]any{"enum": []string{"a", "b"}}); len(got["enum"].([]any)) != 3 {
		t.Fatal("nullable enum mismatch")
	}
	if got := Nullable(map[string]any{"enum": []any{"a", nil}}); len(got["enum"].([]any)) != 2 {
		t.Fatal("existing nil enum should remain unchanged")
	}
	c := Contract{Name: "test", Schema: valid}
	if len(c.Fingerprint()) != 12 {
		t.Fatal("fingerprint length mismatch")
	}
	if got, err := PreparePrompt("base", c, Resolution{Mode: ModePromptContract}); err != nil || !strings.Contains(got, "base") || !strings.Contains(got, "输出契约") {
		t.Fatalf("prompt contract = %q/%v", got, err)
	}
	if got, err := PreparePrompt("", c, Resolution{Mode: ModePromptContract}); err != nil || strings.HasPrefix(got, "\n") {
		t.Fatalf("empty prompt contract = %q/%v", got, err)
	}
}

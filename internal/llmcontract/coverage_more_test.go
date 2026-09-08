package llmcontract

import (
	"encoding/json"
	"strings"
	"testing"
)

func TestValidateJSONSchemaBranches(t *testing.T) {
	cases := []struct {
		name   string
		schema map[string]any
		raw    string
		want   string
	}{
		{"trailing value", map[string]any{"type": "object"}, `{} {}`, "额外值"},
		{"bad trailing", map[string]any{"type": "object"}, `{} nope`, "尾部非法"},
		{"bad type declaration", map[string]any{"type": 1}, `null`, "契约非法"},
		{"null not allowed", map[string]any{"type": "string"}, `null`, "实际为 null"},
		{"wrong value type", map[string]any{"type": "string"}, `1`, "实际为 integer"},
		{"bad enum", map[string]any{"type": "string", "enum": []any{"ok", 1}}, `"ok"`, "enum 契约非法"},
		{"enum mismatch", map[string]any{"type": "string", "enum": []string{"ok"}}, `"no"`, "之一"},
		{"object missing properties", map[string]any{"type": "object"}, `{}`, "缺少 properties"},
		{"bad required", map[string]any{"type": "object", "properties": map[string]any{}, "required": "x"}, `{}`, "required 契约非法"},
		{"missing required", map[string]any{"type": "object", "properties": map[string]any{}, "required": []string{"x"}}, `{}`, "必填字段"},
		{"additional property", map[string]any{"type": "object", "properties": map[string]any{}, "additionalProperties": false}, `{"x":1}`, "未在契约中声明"},
		{"bad child schema", map[string]any{"type": "object", "properties": map[string]any{"x": "bad"}}, `{"x":1}`, "契约不是对象"},
		{"missing array items", map[string]any{"type": "array"}, `[]`, "缺少 items"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			err := ValidateJSON(tc.schema, []byte(tc.raw))
			if err == nil || !strings.Contains(err.Error(), tc.want) {
				t.Fatalf("ValidateJSON error = %v, want %q", err, tc.want)
			}
		})
	}
	if err := ValidateJSON(map[string]any{"type": []any{"number", "null"}}, []byte(`1`)); err != nil {
		t.Fatalf("union number should accept integer: %v", err)
	}
	if err := ValidateJSON(map[string]any{"type": "array", "items": map[string]any{"type": "integer"}}, []byte(`[1,2]`)); err != nil {
		t.Fatal(err)
	}
	if err := ValidateJSON(map[string]any{"type": "object", "properties": map[string]any{"x": map[string]any{"type": "boolean"}}}, []byte(`{"x":true}`)); err != nil {
		t.Fatal(err)
	}
}

func TestValidationInternalTypeHelpers(t *testing.T) {
	for _, value := range []any{nil, "x", []string{"x"}, []any{"x"}, []any{"x", 1}, 1, json.Number("1"), json.Number("1.5"), json.Number("1e999"), true, map[string]any{}, []any{}} {
		_, _ = schemaTypes(value)
		_, _ = stringSlice(value)
		_, _ = enumValues(value)
		_ = valueType(value)
	}
	if got, ok := stringSlice([]any{"a", "b"}); !ok || len(got) != 2 {
		t.Fatalf("string slice = %#v/%v", got, ok)
	}
	if _, ok := stringSlice([]any{"a", 1}); ok {
		t.Fatal("mixed string slice should fail")
	}
	if !enumContains([]any{nil, "ok"}, nil) || !enumContains([]any{"ok"}, "ok") || enumContains([]any{"ok"}, "no") {
		t.Fatal("enum contains mismatch")
	}
	if joinTypes(nil) != "有效 JSON 值" || joinTypes([]string{"string"}) == "" {
		t.Fatal("join types mismatch")
	}
}

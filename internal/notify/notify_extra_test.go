package notify

import (
	"strings"
	"testing"
)

func TestAppleScriptStringEscapesLiteral(t *testing.T) {
	got := appleScriptString(`path\\"quote`)
	if !strings.HasPrefix(got, `"`) || !strings.HasSuffix(got, `"`) || !strings.Contains(got, `\\\\`) || !strings.Contains(got, `\\"`) {
		t.Fatalf("escaped AppleScript string = %q", got)
	}
}

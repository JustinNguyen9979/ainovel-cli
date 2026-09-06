package headless

import (
	"bytes"
	"strings"
	"testing"
	"time"

	"github.com/JustinNguyen9979/ainovel-cli/internal/domain"
	"github.com/JustinNguyen9979/ainovel-cli/internal/host"
)

func TestWriteEventAndReplayQueue(t *testing.T) {
	var out bytes.Buffer
	writeEvent(&out, host.Event{Category: "INFO", Summary: "hello"})
	if !strings.Contains(out.String(), "[--:--:--] [INFO] hello") {
		t.Fatalf("event output = %q", out.String())
	}
	writeEvent(&out, host.Event{Time: time.Date(2026, 9, 6, 12, 34, 56, 0, time.UTC), Category: "INFO", Summary: "later"})
	if !strings.Contains(out.String(), "[12:34:56] [INFO] later") {
		t.Fatalf("timestamp output = %q", out.String())
	}
	before := out.Len()
	writeEvent(&out, host.Event{Summary: "   "})
	writeEvent(nil, host.Event{Summary: "ignored"})
	if out.Len() != before {
		t.Fatal("empty or nil event output should be ignored")
	}
	items := []domain.RuntimeQueueItem{{Time: time.Time{}, Category: "A", Summary: "first"}, {Category: "B", Summary: "second"}}
	replayQueue(items, &out)
	if !strings.Contains(out.String(), "[A] first") || !strings.Contains(out.String(), "[B] second") {
		t.Fatalf("replay output = %q", out.String())
	}
}

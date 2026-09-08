package tui

import (
	"strings"
	"testing"

	buildversion "github.com/JustinNguyen9979/ainovel-cli/internal/version"
)

func TestUpdateNotesPreviewSanitizesAndTruncates(t *testing.T) {
	got := updateNotesPreview("\x1b[31m## Tính năng mới\x1b[0m\x00\n" + strings.Repeat("nội dung ", 20))
	if got != "Tính năng mới" {
		t.Fatalf("preview = %q", got)
	}
}

func TestFormatUpdateNoticeIsVietnamese(t *testing.T) {
	got := formatUpdateNotice(&buildversion.CheckResult{Latest: "v1.2.4", Notes: "## Sửa lỗi"})
	for _, want := range []string{"v1.2.4", "Sửa lỗi", "ainovel-cli update", "nâng cấp"} {
		if !strings.Contains(got, want) {
			t.Fatalf("notice %q missing %q", got, want)
		}
	}
}

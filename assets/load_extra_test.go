package assets

import (
	"path/filepath"
	"testing"
)

func TestDefaultLoadOptions(t *testing.T) {
	if got := DefaultLoadOptions(""); got.BookStyleDir != "" {
		t.Fatalf("empty output dir should not create book style path: %+v", got)
	}
	if got := DefaultLoadOptions("/tmp/book"); got.BookStyleDir != filepath.Join("/tmp/book", "style") {
		t.Fatalf("book style path = %q", got.BookStyleDir)
	}
}

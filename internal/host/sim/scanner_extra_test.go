package sim

import (
	"os"
	"path/filepath"
	"testing"
)

func TestScanSourcesFiltersSortsAndValidates(t *testing.T) {
	root := t.TempDir()
	if _, err := scanSources(""); err == nil {
		t.Fatal("empty source dir should fail")
	}
	if _, err := scanSources(filepath.Join(root, "missing")); err == nil {
		t.Fatal("missing source dir should fail")
	}
	file := filepath.Join(root, "file.txt")
	if err := os.WriteFile(file, []byte("x"), 0o644); err != nil {
		t.Fatal(err)
	}
	if _, err := scanSources(file); err == nil {
		t.Fatal("file path should fail")
	}
	if err := os.Remove(file); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(filepath.Join(root, "nested"), 0o755); err != nil {
		t.Fatal(err)
	}
	for name, content := range map[string]string{"b.MARKDOWN": "B", "a.txt": "A", "skip.json": "{}"} {
		if err := os.WriteFile(filepath.Join(root, "nested", name), []byte(content), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	files, err := scanSources(root)
	if err != nil || len(files) != 2 || files[0].RelativePath >= files[1].RelativePath {
		t.Fatalf("scanned files = %+v/%v", files, err)
	}
	if files[0].Fingerprint == "" || files[0].content == "" || files[0].SizeBytes == 0 {
		t.Fatalf("source metadata incomplete: %+v", files[0])
	}
	for _, path := range []string{"a.txt", "A.MD", "book.markdown"} {
		if !isSupportedSource(path) {
			t.Fatalf("supported source rejected: %s", path)
		}
	}
	if isSupportedSource("book.json") {
		t.Fatal("unsupported source accepted")
	}
}

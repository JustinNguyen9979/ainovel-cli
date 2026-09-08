package logger

import (
	"bytes"
	"log/slog"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestLoggerSetupAndFileLogger(t *testing.T) {
	var buf bytes.Buffer
	Setup(&buf, slog.LevelDebug)
	slog.Info("hello", "value", 1)
	if !strings.Contains(buf.String(), "hello") {
		t.Fatalf("setup log = %q", buf.String())
	}
	dir := t.TempDir()
	logger, cleanup, err := FileLogger(dir, "import.log")
	if err != nil {
		t.Fatal(err)
	}
	logger.Debug("debug")
	cleanup()
	data, err := os.ReadFile(filepath.Join(dir, "logs", "import.log"))
	if err != nil || !strings.Contains(string(data), "日志会话开始") || !strings.Contains(string(data), "日志会话结束") {
		t.Fatalf("file log = %q/%v", data, err)
	}
	_, secondCleanup, err := FileLogger(filepath.Join(dir, "file"), "x.log")
	if err != nil || secondCleanup == nil {
		t.Fatalf("second file logger = %v", err)
	}
	secondCleanup()
}

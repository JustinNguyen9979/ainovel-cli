package host

import (
	"context"
	"strings"
	"testing"

	"github.com/JustinNguyen9979/ainovel-cli/internal/domain"
	"github.com/JustinNguyen9979/ainovel-cli/internal/host/imp"
	storepkg "github.com/JustinNguyen9979/ainovel-cli/internal/store"
)

func minimalHost(t *testing.T) (*Host, *storepkg.Store) {
	t.Helper()
	st := storepkg.NewStore(t.TempDir())
	if err := st.Init(); err != nil {
		t.Fatal(err)
	}
	return &Host{store: st, engine: &engine{}, events: make(chan Event, 8), streamCh: make(chan string, 8), done: make(chan struct{}, 1)}, st
}

func TestHostLifecycleGuardBranches(t *testing.T) {
	h, st := minimalHost(t)
	if err := h.StartPrepared("   "); err == nil || !strings.Contains(err.Error(), "prompt is required") {
		t.Fatalf("empty StartPrepared = %v", err)
	}
	h.lifecycle = lifecycleRunning
	if err := h.StartPrepared("prompt"); err == nil || !strings.Contains(err.Error(), "already running") {
		t.Fatalf("running StartPrepared = %v", err)
	}
	h.lifecycle = lifecycleIdle
	h.cocreating = true
	if err := h.StartPrepared("prompt"); err == nil || !strings.Contains(err.Error(), "共创") {
		t.Fatalf("cocreating StartPrepared = %v", err)
	}
	h.cocreating = false

	if label, err := h.Resume(); err != nil || label != "" {
		t.Fatalf("empty Resume = %q/%v", label, err)
	}
	h.lifecycle = lifecycleRunning
	if _, err := h.Resume(); err == nil || !strings.Contains(err.Error(), "already running") {
		t.Fatalf("running Resume = %v", err)
	}
	h.lifecycle = lifecycleIdle
	h.cocreating = true
	if _, err := h.Resume(); err == nil || !strings.Contains(err.Error(), "共创") {
		t.Fatalf("cocreating Resume = %v", err)
	}
	h.cocreating = false
	h.exclusive = "导入"
	if _, err := h.Resume(); err == nil || !strings.Contains(err.Error(), "导入") {
		t.Fatalf("exclusive Resume = %v", err)
	}
	h.exclusive = ""

	if err := h.Continue(" "); err == nil || !strings.Contains(err.Error(), "text is required") {
		t.Fatalf("empty Continue = %v", err)
	}
	h.cocreating = true
	if err := h.Continue("继续"); err == nil || !strings.Contains(err.Error(), "共创") {
		t.Fatalf("cocreating Continue = %v", err)
	}
	h.cocreating = false
	h.exclusive = "同步"
	if err := h.Continue("继续"); err == nil || !strings.Contains(err.Error(), "同步") {
		t.Fatalf("exclusive Continue = %v", err)
	}
	h.exclusive = ""

	if h.Abort() {
		t.Fatal("idle host without exclusive work should not abort")
	}
	h.closing = true
	if err := h.Steer("修改"); err == nil || !strings.Contains(err.Error(), "关闭") {
		t.Fatalf("closing Steer = %v", err)
	}
	if _, err := h.ReplayQueue(0); err != nil {
		t.Fatal(err)
	}
	_ = st
}

func TestHostRefusesNewBookWithExistingProgress(t *testing.T) {
	h, st := minimalHost(t)
	if err := st.Progress.Save(&domain.Progress{Phase: domain.PhaseWriting}); err != nil {
		t.Fatal(err)
	}
	if err := h.refuseNewBookOverExisting(); err != nil {
		t.Fatalf("no completed chapters should be allowed: %v", err)
	}
	if err := st.Progress.Save(&domain.Progress{Phase: domain.PhaseWriting, CompletedChapters: []int{1, 2}}); err != nil {
		t.Fatal(err)
	}
	if err := h.refuseNewBookOverExisting(); err == nil || !strings.Contains(err.Error(), "作品信息不存在") {
		t.Fatalf("missing book should be rejected: %v", err)
	}
	if err := st.Book.Save(domain.BookMetadata{Title: "旧书", Synopsis: "简介"}); err != nil {
		t.Fatal(err)
	}
	if err := h.refuseNewBookOverExisting(); err == nil || !strings.Contains(err.Error(), "旧书") {
		t.Fatalf("existing book should be rejected: %v", err)
	}
}

func TestHostAdvanceAndImportGuards(t *testing.T) {
	h, st := minimalHost(t)
	if err := h.AdvanceOneChapter(); err == nil || !strings.Contains(err.Error(), "RunMeta") {
		t.Fatalf("uninitialized advance = %v", err)
	}
	if err := st.RunMeta.Init("default", "provider", "model"); err != nil {
		t.Fatal(err)
	}
	if err := h.AdvanceOneChapter(); err == nil || !strings.Contains(err.Error(), "逐章验收") {
		t.Fatalf("wrong advance mode = %v", err)
	}
	if err := st.RunMeta.SetAdvanceMode(domain.ChapterAdvanceReview); err != nil {
		t.Fatal(err)
	}
	if err := h.AdvanceOneChapter(); err == nil || !strings.Contains(err.Error(), "不能授权") {
		t.Fatalf("missing progress = %v", err)
	}
	h.exclusive = "đang chạy"
	if _, err := h.ImportFrom(context.Background(), imp.Options{}); err == nil {
		t.Fatal("import during another exclusive job should fail")
	}
	if _, err := h.ImportSimulationProfile(context.Background(), "missing.json"); err == nil {
		t.Fatal("simulation import during another exclusive job should fail")
	}
}

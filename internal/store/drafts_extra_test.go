package store

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/JustinNguyen9979/ainovel-cli/internal/domain"
)

func TestDraftStoreRoundTripsAndRanges(t *testing.T) {
	st := NewStore(t.TempDir())
	plan := domain.ChapterPlan{Chapter: 1, Title: "Chapter one", Goal: "goal"}
	if err := st.Drafts.SaveChapterPlan(plan); err != nil {
		t.Fatal(err)
	}
	if got, err := st.Drafts.LoadChapterPlan(1); err != nil || got == nil || got.Title != plan.Title {
		t.Fatalf("plan = %+v, %v", got, err)
	}
	if err := st.Drafts.SaveDraft(1, "first"); err != nil || st.Drafts.AppendDraft(1, "second") != nil {
		t.Fatal("draft save/append failed")
	}
	if got, count, err := st.Drafts.LoadChapterContent(1); err != nil || count != len([]rune(got)) || !strings.Contains(got, "second") {
		t.Fatalf("draft = %q/%d, %v", got, count, err)
	}
	if err := st.Drafts.SaveFinalChapter(1, "一二三四五六"); err != nil {
		t.Fatal(err)
	}
	if got, err := st.Drafts.LoadChapterText(1); err != nil || got == "" {
		t.Fatalf("chapter text = %q, %v", got, err)
	}
	if got, err := st.Drafts.LoadChapterRange(1, 2, 3); err != nil || got[1] != "一二三..." {
		t.Fatalf("chapter range = %+v, %v", got, err)
	}
}

func TestDraftStoreMissingFilesAndDefaultLimits(t *testing.T) {
	st := NewStore(t.TempDir())
	if plan, err := st.Drafts.LoadChapterPlan(1); err != nil || plan != nil {
		t.Fatalf("missing plan = %+v, %v", plan, err)
	}
	if got, err := st.Drafts.LoadDraft(1); err != nil || got != "" {
		t.Fatalf("missing draft = %q, %v", got, err)
	}
	if got, err := st.Drafts.LoadChapterText(1); err != nil || got != "" {
		t.Fatalf("missing chapter = %q, %v", got, err)
	}
	if got, count, err := st.Drafts.LoadChapterContent(1); err != nil || got != "" || count != 0 {
		t.Fatalf("missing content = %q/%d, %v", got, count, err)
	}
	if got, err := st.Drafts.LoadChapterRange(1, 2, 0); err != nil || len(got) != 0 {
		t.Fatalf("missing range = %+v, %v", got, err)
	}
	if samples, err := st.Drafts.ExtractDialogue("角色", nil, 0, 1); err != nil || samples != nil {
		t.Fatalf("missing dialogue = %+v, %v", samples, err)
	}
	if anchors, err := st.Drafts.ExtractStyleAnchors(0, 1); err != nil || anchors != nil {
		t.Fatalf("missing anchors = %+v, %v", anchors, err)
	}
}

func TestDraftStoreExtractsDialogueAliasesAndFiltersAnchors(t *testing.T) {
	st := NewStore(t.TempDir())
	dialogue := "旧称：\"这是来自别名的足够长对白\"。\n新称：\"另一段足够长对白\"。\n忽略：\"短\""
	if err := st.Drafts.SaveFinalChapter(2, dialogue); err != nil {
		t.Fatal(err)
	}
	if got, err := st.Drafts.ExtractDialogue("角色", []string{"旧称"}, 0, 2); err != nil || len(got) != 1 || !strings.Contains(got[0], "足够长对白") {
		t.Fatalf("dialogue aliases = %v, %v", got, err)
	}
	valid := strings.Repeat("有效风格段落。", 20)
	short := strings.Repeat("短", 49)
	long := strings.Repeat("长", 301)
	quoted := strings.Repeat("“", 3) + strings.Repeat("引号段落。", 20)
	if err := st.Drafts.SaveFinalChapter(1, strings.Join([]string{short, valid, long, quoted}, "\n\n")); err != nil {
		t.Fatal(err)
	}
	if got, err := st.Drafts.ExtractStyleAnchors(0, 1); err != nil || len(got) != 1 || got[0] != valid {
		t.Fatalf("filtered anchors = %v, %v", got, err)
	}
}

func TestDraftStoreExtractsReadErrorsAfterOtherChapters(t *testing.T) {
	dir := t.TempDir()
	st := NewStore(dir)
	if err := st.Drafts.SaveFinalChapter(1, "角色：\"这是足够长的一段对白\""); err != nil {
		t.Fatal(err)
	}
	if err := os.Mkdir(filepath.Join(dir, "chapters", "02.md"), 0o755); err != nil {
		t.Fatal(err)
	}
	if got, err := st.Drafts.ExtractDialogue("角色", nil, 5, 2); err == nil || len(got) != 1 {
		t.Fatalf("dialogue error/sample = %v, %v", got, err)
	}
	if got, err := st.Drafts.ExtractStyleAnchors(5, 2); err == nil || len(got) != 0 {
		t.Fatalf("anchor error/result = %v, %v", got, err)
	}
}

func TestDraftStoreExtractsDialogueAndStyleAnchors(t *testing.T) {
	st := NewStore(t.TempDir())
	text := "角色说：\"这是足够长的一段对白\"。\n\n" + strings.Repeat("这是一段足够长的风格锚点文本。", 10)
	if err := st.Drafts.SaveFinalChapter(1, text); err != nil {
		t.Fatal(err)
	}
	if got, err := st.Drafts.ExtractDialogue("角色", nil, 1, 1); err != nil || len(got) != 1 {
		t.Fatalf("dialogue = %v, %v", got, err)
	}
	if got, err := st.Drafts.ExtractStyleAnchors(1, 1); err != nil || len(got) != 1 {
		t.Fatalf("anchors = %v, %v", got, err)
	}
}

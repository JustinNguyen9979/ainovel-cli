package store

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/JustinNguyen9979/ainovel-cli/internal/domain"
	"github.com/JustinNguyen9979/ainovel-cli/internal/rules"
)

func TestChapterRecordPrepareAcceptAndLoadCompleted(t *testing.T) {
	st := NewStore(t.TempDir())
	facts := domain.ChapterFacts{Title: "Chapter", Summary: "Summary", ForeshadowUpdates: []domain.ForeshadowUpdate{{ID: "hook", Action: "plant"}}}
	record, err := st.ChapterRecords.Prepare(2, domain.ChapterOriginGenerated, "a\r\nb", facts, domain.StyleDelta{})
	if err != nil || record == nil || record.Revision != 1 || record.Content != "a\nb" {
		t.Fatalf("prepared record = %+v, %v", record, err)
	}
	accepted, err := st.ChapterRecords.Accept(2, domain.ChapterOriginGenerated, "a\r\nb", facts, domain.StyleDelta{})
	if err != nil || accepted == nil || accepted.Revision != 1 {
		t.Fatalf("accepted record = %+v, %v", accepted, err)
	}
	accepted, err = st.ChapterRecords.Accept(2, domain.ChapterOriginGenerated, "changed", facts, domain.StyleDelta{})
	if err != nil || accepted.Revision != 2 {
		t.Fatalf("revised record = %+v, %v", accepted, err)
	}
	if got, err := st.ChapterRecords.Load(2); err != nil || got == nil || got.Content != "changed" {
		t.Fatalf("loaded record = %+v, %v", got, err)
	}
	if got, err := st.ChapterRecords.LoadCompleted([]int{2}); err != nil || len(got) != 1 {
		t.Fatalf("completed records = %+v, %v", got, err)
	}
	if _, err := st.ChapterRecords.LoadCompleted([]int{1}); err == nil {
		t.Fatal("missing completed record should fail")
	}
}

func TestChapterRecordValidationAndPath(t *testing.T) {
	if ChapterRecordPath(3) != "meta/chapter_records/000003.json" {
		t.Fatal("record path format is wrong")
	}
	base := domain.ChapterRecord{Version: domain.ChapterRecordVersion, Chapter: 1, Revision: 1, Origin: domain.ChapterOriginUser, Content: "content", ContentSHA256: domain.ChapterContentSHA256("content"), AcceptedAt: time.Now()}
	for _, bad := range []domain.ChapterRecord{
		{Version: 99, Chapter: 1, Revision: 1, Origin: domain.ChapterOriginUser, Content: "content", ContentSHA256: domain.ChapterContentSHA256("content"), AcceptedAt: time.Now()},
		{Version: 1, Chapter: 0, Revision: 1, Origin: domain.ChapterOriginUser, Content: "content", ContentSHA256: domain.ChapterContentSHA256("content"), AcceptedAt: time.Now()},
		{Version: 1, Chapter: 1, Revision: 1, Origin: "bad", Content: "content", ContentSHA256: domain.ChapterContentSHA256("content"), AcceptedAt: time.Now()},
		{Version: 1, Chapter: 1, Revision: 1, Origin: domain.ChapterOriginUser, Content: "content", ContentSHA256: "bad", AcceptedAt: time.Now()},
	} {
		if err := validateChapterRecord(bad); err == nil {
			t.Fatalf("invalid record accepted: %+v", bad)
		}
	}
	if err := validateChapterRecord(base); err != nil {
		t.Fatal(err)
	}
}

func TestChapterRecordLoadErrorsAndPrepareIsInMemory(t *testing.T) {
	dir := t.TempDir()
	st := NewStore(dir)
	if record, err := st.ChapterRecords.Load(1); err != nil || record != nil {
		t.Fatalf("missing record = %+v, %v", record, err)
	}
	prepared, err := st.ChapterRecords.Prepare(1, domain.ChapterOriginUser, "draft", domain.ChapterFacts{}, domain.StyleDelta{})
	if err != nil || prepared == nil {
		t.Fatalf("prepared record = %+v, %v", prepared, err)
	}
	if _, err := os.Stat(filepath.Join(dir, ChapterRecordPath(1))); !os.IsNotExist(err) {
		t.Fatalf("Prepare wrote a record: %v", err)
	}

	recordsDir := filepath.Join(dir, chapterRecordsDir)
	if err := os.MkdirAll(recordsDir, 0o755); err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(dir, ChapterRecordPath(1))
	if err := os.WriteFile(path, []byte("{"), 0o644); err != nil {
		t.Fatal(err)
	}
	if _, err := st.ChapterRecords.Load(1); err == nil {
		t.Fatal("malformed record should fail to load")
	}
	valid := domain.ChapterRecord{
		Version:       domain.ChapterRecordVersion,
		Chapter:       2,
		Revision:      1,
		Origin:        domain.ChapterOriginUser,
		Content:       "content",
		ContentSHA256: domain.ChapterContentSHA256("content"),
		AcceptedAt:    time.Now(),
	}
	data, err := json.Marshal(valid)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, data, 0o644); err != nil {
		t.Fatal(err)
	}
	if _, err := st.ChapterRecords.Load(1); err == nil {
		t.Fatal("chapter mismatch should fail to load")
	}
}

func TestChapterRecordValidationCoversAllRequiredFields(t *testing.T) {
	base := domain.ChapterRecord{
		Version:       domain.ChapterRecordVersion,
		Chapter:       1,
		Revision:      1,
		Origin:        domain.ChapterOriginUser,
		Content:       "content",
		ContentSHA256: domain.ChapterContentSHA256("content"),
		AcceptedAt:    time.Now(),
	}
	cases := []struct {
		name string
		edit func(*domain.ChapterRecord)
	}{
		{"revision", func(r *domain.ChapterRecord) { r.Revision = 0 }},
		{"accepted at", func(r *domain.ChapterRecord) { r.AcceptedAt = time.Time{} }},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			record := base
			tc.edit(&record)
			if err := validateChapterRecord(record); err == nil {
				t.Fatal("invalid record accepted")
			}
		})
	}
}

func TestChapterRecordAcceptRestoresOwnPlants(t *testing.T) {
	st := NewStore(t.TempDir())
	firstFacts := domain.ChapterFacts{Title: "Chapter", Summary: "Summary", ForeshadowUpdates: []domain.ForeshadowUpdate{{ID: "hook", Action: "plant", Description: "clue"}}}
	if _, err := st.ChapterRecords.Accept(1, domain.ChapterOriginGenerated, "first", firstFacts, domain.StyleDelta{}); err != nil {
		t.Fatal(err)
	}
	secondFacts := domain.ChapterFacts{Title: "Chapter", Summary: "Revised"}
	record, err := st.ChapterRecords.Accept(1, domain.ChapterOriginUser, "second", secondFacts, domain.StyleDelta{})
	if err != nil {
		t.Fatal(err)
	}
	if len(record.Facts.ForeshadowUpdates) != 1 || record.Facts.ForeshadowUpdates[0].ID != "hook" {
		t.Fatalf("own plant was not restored: %+v", record.Facts.ForeshadowUpdates)
	}
}

func TestCharacterAndUserRulesRoundTrips(t *testing.T) {
	st := NewStore(t.TempDir())
	chars := []domain.Character{{Name: "A", Role: "hero", Description: "desc", Arc: "arc", Traits: []string{"brave"}}}
	if err := st.Characters.Save(chars); err != nil {
		t.Fatal(err)
	}
	if got, err := st.Characters.Load(); err != nil || len(got) != 1 || got[0].Name != "A" {
		t.Fatalf("characters = %+v, %v", got, err)
	}
	snaps := []domain.CharacterSnapshot{{Volume: 1, Arc: 1, Name: "A", Status: "active", Motivation: "goal"}}
	if err := st.Characters.SaveSnapshots(1, 1, snaps); err != nil {
		t.Fatal(err)
	}
	if got, err := st.Characters.LoadSnapshots(1, 1); err != nil || len(got) != 1 {
		t.Fatalf("snapshots = %+v, %v", got, err)
	}
	snapshot := &rules.Snapshot{Version: rules.SnapshotVersion, Status: rules.StatusReady, Preferences: "prefer concise prose"}
	if err := st.UserRules.Save(snapshot); err != nil {
		t.Fatal(err)
	}
	if got, err := st.UserRules.Load(); err != nil || got == nil || got.Preferences != snapshot.Preferences {
		t.Fatalf("rules = %+v, %v", got, err)
	}
}

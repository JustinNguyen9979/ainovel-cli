package store

import (
	"testing"

	"github.com/JustinNguyen9979/ainovel-cli/internal/domain"
)

func TestSignalStorePendingAndClearLifecycle(t *testing.T) {
	s := NewStore(t.TempDir())
	pending := domain.PendingCommit{Chapter: 2, Stage: domain.CommitStageStarted, Summary: "draft"}
	if err := s.Signals.SavePendingCommit(pending); err != nil {
		t.Fatal(err)
	}
	if got, err := s.Signals.LoadPendingCommit(); err != nil || got == nil || got.Chapter != 2 {
		t.Fatalf("pending = %+v/%v", got, err)
	}
	if err := s.Signals.ClearPendingCommit(); err != nil {
		t.Fatal(err)
	}
	if got, err := s.Signals.LoadPendingCommit(); err != nil || got != nil {
		t.Fatalf("cleared pending = %+v/%v", got, err)
	}
	if got, err := s.Signals.LoadAndClearLastCommit(); err != nil || got != nil {
		t.Fatalf("missing commit signal = %+v/%v", got, err)
	}
	if got, err := s.Signals.LoadAndClearLastReview(); err != nil || got != nil {
		t.Fatalf("missing review signal = %+v/%v", got, err)
	}
}

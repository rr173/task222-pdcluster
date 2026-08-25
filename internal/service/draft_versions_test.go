package service

import (
	"path/filepath"
	"testing"

	"task222-pdcluster/internal/model"
	"task222-pdcluster/internal/store"
)

// TestSnapshotDraftVersionIncrements reproduces the reported bug: two consecutive
// drafts on the same trial must persist with versions 1 and 2 (no conflict / no drop).
func TestSnapshotDraftVersionIncrements(t *testing.T) {
	dbPath := filepath.Join(t.TempDir(), "pdcluster.db")
	db, err := store.Open(dbPath)
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	app, err := New(db)
	if err != nil {
		t.Fatal(err)
	}

	trial, err := app.Trials.Create("DRAFT-VER-001", "110kV cable", "110kV")
	if err != nil {
		t.Fatalf("create trial: %v", err)
	}
	if _, err := app.Trials.SetReference(trial.ID, 50, 0); err != nil {
		t.Fatalf("set reference: %v", err)
	}
	ch, err := app.Trials.AddChannel(trial.ID, "CH0", 0)
	if err != nil {
		t.Fatalf("add channel: %v", err)
	}
	if _, err := app.Trials.StartAcquisition(trial.ID); err != nil {
		t.Fatalf("start: %v", err)
	}
	if _, err := app.Pulses.Ingest(trial.ID, ch.ID, 0, 1, 1_000_000, 25.0); err != nil {
		t.Fatalf("ingest: %v", err)
	}

	// First draft -> version 1
	first, err := app.Snapshots.Create(trial.ID)
	if err != nil {
		t.Fatalf("first draft: %v", err)
	}
	if first.Version != 1 {
		t.Fatalf("first draft version = %d, want 1", first.Version)
	}
	if first.Status != model.SnapshotDraft {
		t.Fatalf("first draft status = %q, want draft", first.Status)
	}

	// Second draft -> version 2, must persist alongside the first
	second, err := app.Snapshots.Create(trial.ID)
	if err != nil {
		t.Fatalf("second draft: %v", err)
	}
	if second.Version != 2 {
		t.Fatalf("second draft version = %d, want 2", second.Version)
	}
	if second.Status != model.SnapshotDraft {
		t.Fatalf("second draft status = %q, want draft", second.Status)
	}

	// Both drafts must be persisted (list ordered by version DESC).
	snaps, err := app.Snapshots.ListSnapshots(trial.ID)
	if err != nil {
		t.Fatalf("list: %v", err)
	}
	if len(snaps) != 2 {
		t.Fatalf("expected 2 snapshots persisted, got %d", len(snaps))
	}
	if snaps[0].Version != 2 || snaps[1].Version != 1 {
		t.Fatalf("versions = [%d, %d], want [2, 1]", snaps[0].Version, snaps[1].Version)
	}
}

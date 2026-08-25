package service

import (
	"testing"

	"task222-pdcluster/internal/store"
)

func TestSnapshotVersionsAdvanceAcrossDrafts(t *testing.T) {
	db, err := store.Open(":memory:")
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	app, err := New(db)
	if err != nil {
		t.Fatal(err)
	}
	tr, err := app.Trials.Create("SNAP-E2E", "cable", "110kV")
	if err != nil {
		t.Fatal(err)
	}
	first, err := app.Snapshots.Create(tr.ID)
	if err != nil {
		t.Fatal(err)
	}
	second, err := app.Snapshots.Create(tr.ID)
	if err != nil {
		t.Fatal(err)
	}
	if first.Version != 1 || second.Version != 2 {
		t.Fatalf("draft versions=%d,%d, want 1,2", first.Version, second.Version)
	}
	snapshots, err := app.Snapshots.ListSnapshots(tr.ID)
	if err != nil {
		t.Fatal(err)
	}
	if len(snapshots) != 2 {
		t.Fatalf("persisted snapshots=%d, want 2", len(snapshots))
	}
}

package service

import (
	"testing"

	"task222-pdcluster/internal/model"
	"task222-pdcluster/internal/store"
)

func TestPublishedSnapshotCannotReenterPublishTransition(t *testing.T) {
	db, err := store.Open(":memory:")
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	app, err := New(db)
	if err != nil {
		t.Fatal(err)
	}
	tr, err := app.Trials.Create("SNAP-TERMINAL", "cable", "110kV")
	if err != nil {
		t.Fatal(err)
	}
	sn, err := app.Snapshots.Create(tr.ID)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := app.Snapshots.Publish(sn.ID); err != nil {
		t.Fatal(err)
	}
	if _, err := app.Snapshots.Publish(sn.ID); err == nil {
		t.Fatal("published snapshot re-entered publish transition")
	}
	stored, err := app.Snapshots.GetSnapshot(sn.ID)
	if err != nil {
		t.Fatal(err)
	}
	if stored.Status != model.SnapshotPublished {
		t.Fatalf("status=%s, want published", stored.Status)
	}
}

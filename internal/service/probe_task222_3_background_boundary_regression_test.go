package service

import (
	"testing"

	"task222-pdcluster/internal/store"
)

func TestBackgroundFilterKeepsThresholdPulse(t *testing.T) {
	db, err := store.Open(":memory:")
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	app, err := New(db)
	if err != nil {
		t.Fatal(err)
	}
	tr, err := app.Trials.Create("BG-E2E", "cable", "110kV")
	if err != nil {
		t.Fatal(err)
	}
	ch, err := app.Trials.AddChannel(tr.ID, "CH0", 0)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := app.Trials.StartAcquisition(tr.ID); err != nil {
		t.Fatal(err)
	}
	if _, err := app.Pulses.Ingest(tr.ID, ch.ID, 0, 1, 100, 2); err != nil {
		t.Fatal(err)
	}
	count, err := app.Pulses.ApplyBackgroundFilter(tr.ID, 2)
	if err != nil {
		t.Fatal(err)
	}
	if count != 0 {
		t.Fatalf("background count=%d, want 0 at exact threshold", count)
	}
}

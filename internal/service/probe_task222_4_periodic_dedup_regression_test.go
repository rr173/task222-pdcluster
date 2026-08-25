package service

import (
	"testing"

	"task222-pdcluster/internal/model"
	"task222-pdcluster/internal/store"
)

func TestPeriodicInterferenceIsMarkedDuplicate(t *testing.T) {
	db, err := store.Open(":memory:")
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	app, err := New(db)
	if err != nil {
		t.Fatal(err)
	}
	tr, err := app.Trials.Create("DEDUP-E2E", "cable", "110kV")
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
	for i := int64(0); i < 3; i++ {
		if _, err := app.Pulses.Ingest(tr.ID, ch.ID, 0, i, 1_000_000+i*2_000_000, 55); err != nil {
			t.Fatal(err)
		}
	}
	count, err := app.Pulses.ApplyDedup(tr.ID)
	if err != nil {
		t.Fatal(err)
	}
	if count != 3 {
		t.Fatalf("dedup count=%d, want 3", count)
	}
	ps, err := app.Pulses.ListPulses(tr.ID)
	if err != nil {
		t.Fatal(err)
	}
	for _, p := range ps {
		if p.Status != model.PulseDuplicate {
			t.Fatalf("pulse %s status=%s, want duplicate", p.ID, p.Status)
		}
	}
}

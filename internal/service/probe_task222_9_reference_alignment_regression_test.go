package service

import (
	"testing"

	"task222-pdcluster/internal/phase"
	"task222-pdcluster/internal/store"
)

func TestReferenceRoundTripKeepsAlignmentAnchor(t *testing.T) {
	db, err := store.Open(":memory:")
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	app, err := New(db)
	if err != nil {
		t.Fatal(err)
	}
	tr, err := app.Trials.Create("REF-ALIGN", "cable", "110kV")
	if err != nil {
		t.Fatal(err)
	}
	const zero int64 = 20_000_000
	ref, err := app.Trials.SetReference(tr.ID, 50, zero)
	if err != nil {
		t.Fatal(err)
	}
	stored, err := app.Trials.GetReference(tr.ID)
	if err != nil {
		t.Fatal(err)
	}
	if stored.ZeroTimeNs != zero || ref.ZeroTimeNs != zero {
		t.Fatalf("reference zero time ref=%d stored=%d want=%d", ref.ZeroTimeNs, stored.ZeroTimeNs, zero)
	}
	if got := phase.Align(zero, stored.ZeroTimeNs, stored.FreqHz); got != 0 {
		t.Fatalf("aligned phase=%.6f, want 0", got)
	}
}

package service

import (
	"math"
	"testing"

	"task222-pdcluster/internal/store"
)

func TestDelayCalibrationKeepsPositiveDelayContract(t *testing.T) {
	db, err := store.Open(":memory:")
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	app, err := New(db)
	if err != nil {
		t.Fatal(err)
	}
	tr, err := app.Trials.Create("DELAY-E2E", "cable", "110kV")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := app.Trials.AddChannel(tr.ID, "ref", 0); err != nil {
		t.Fatal(err)
	}
	if _, err := app.Trials.AddChannel(tr.ID, "late", 1); err != nil {
		t.Fatal(err)
	}
	if _, err := app.Trials.SetReference(tr.ID, 50, 0); err != nil {
		t.Fatal(err)
	}
	if _, err := app.Trials.StartAcquisition(tr.ID); err != nil {
		t.Fatal(err)
	}
	for seq, base := range map[int64]int64{1: 10_000_000, 2: 20_000_000} {
		if _, err := app.Pulses.Ingest(tr.ID, "ref", 0, seq, base, 20); err != nil {
			t.Fatal(err)
		}
		if _, err := app.Pulses.Ingest(tr.ID, "late", 1, seq, base+1000, 20); err != nil {
			t.Fatal(err)
		}
	}
	delays, err := app.Pulses.Calibrate(tr.ID)
	if err != nil {
		t.Fatal(err)
	}
	if math.Abs(delays[1]-1000) > 0.01 {
		t.Fatalf("returned channel delay=%v, want +1000", delays[1])
	}
	channels, err := app.Trials.ListChannels(tr.ID)
	if err != nil {
		t.Fatal(err)
	}
	if math.Abs(channels[1].DelayNs-1000) > 0.01 {
		t.Fatalf("persisted channel delay=%v, want +1000", channels[1].DelayNs)
	}
	pulses, err := app.Pulses.ListPulses(tr.ID)
	if err != nil {
		t.Fatal(err)
	}
	if len(pulses) != 4 {
		t.Fatalf("calibrated pulse count=%d, want 4", len(pulses))
	}
	bySeq := map[int64]map[int]float64{}
	for _, p := range pulses {
		if bySeq[p.Seq] == nil {
			bySeq[p.Seq] = map[int]float64{}
		}
		bySeq[p.Seq][p.ChannelIndex] = p.PhaseDeg
	}
	for seq, phases := range bySeq {
		if math.Abs(phases[0]-phases[1]) > 0.01 {
			t.Fatalf("seq %d calibrated phases=%v, want paired phases to align", seq, phases)
		}
	}
}

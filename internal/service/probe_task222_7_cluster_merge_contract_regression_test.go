package service

import (
	"testing"

	"task222-pdcluster/internal/model"
	"task222-pdcluster/internal/store"
)

func TestMergeKeepsRelativeAmplitudeAndRejectsInputs(t *testing.T) {
	db, err := store.Open(":memory:")
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	app, err := New(db)
	if err != nil {
		t.Fatal(err)
	}
	tr, err := app.Trials.Create("MERGE-CONTRACT", "cable", "110kV")
	if err != nil {
		t.Fatal(err)
	}
	now := nowISO()
	a := &model.Cluster{ID: "cluster-a", TrialID: tr.ID, PhaseStartDeg: 20, PhaseEndDeg: 30, PhaseCenterDeg: 25, PulseCount: 10, AvgAmplitudeMv: 100, MaxAmplitudeMv: 110, Status: model.ClusterCandidate, CreatedAt: now, UpdatedAt: now}
	b := &model.Cluster{ID: "cluster-b", TrialID: tr.ID, PhaseStartDeg: 35, PhaseEndDeg: 45, PhaseCenterDeg: 40, PulseCount: 12, AvgAmplitudeMv: 105, MaxAmplitudeMv: 115, Status: model.ClusterCandidate, CreatedAt: now, UpdatedAt: now}
	if err := app.store.Clusters.Insert(a); err != nil {
		t.Fatal(err)
	}
	if err := app.store.Clusters.Insert(b); err != nil {
		t.Fatal(err)
	}
	merged, err := app.Clusters.MergeClusters(a.ID, b.ID)
	if err != nil {
		t.Fatalf("merge failed: %v", err)
	}
	if merged.PulseCount != 22 || merged.AvgAmplitudeMv < 102 || merged.AvgAmplitudeMv > 104 {
		t.Fatalf("merged cluster=%+v", merged)
	}
	for _, id := range []string{a.ID, b.ID} {
		stored, err := app.store.Clusters.Get(id)
		if err != nil {
			t.Fatal(err)
		}
		if stored.Status != model.ClusterRejected {
			t.Fatalf("input %s status=%s, want rejected", id, stored.Status)
		}
	}
}

package service

import (
	"testing"

	"task222-pdcluster/internal/model"
	"task222-pdcluster/internal/store"
)

func TestDiagnosisPersistsHighestConfidenceRecommendationFirst(t *testing.T) {
	db, err := store.Open(":memory:")
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	app, err := New(db)
	if err != nil {
		t.Fatal(err)
	}
	tr, err := app.Trials.Create("DIAG-RECOMMEND", "cable", "110kV")
	if err != nil {
		t.Fatal(err)
	}
	now := nowISO()
	clusters := []*model.Cluster{
		{ID: "diag-a", TrialID: tr.ID, PhaseCenterDeg: 30, PulseCount: 8, MaxAmplitudeMv: 60, AvgAmplitudeMv: 55, Status: model.ClusterStable, CreatedAt: now, UpdatedAt: now},
		{ID: "diag-b", TrialID: tr.ID, PhaseCenterDeg: 210, PulseCount: 2, MaxAmplitudeMv: 20, AvgAmplitudeMv: 18, Status: model.ClusterStable, CreatedAt: now, UpdatedAt: now},
	}
	for _, c := range clusters {
		if err := app.store.Clusters.Insert(c); err != nil {
			t.Fatal(err)
		}
	}
	items, err := app.Diagnosis.Classify(tr.ID)
	if err != nil {
		t.Fatal(err)
	}
	if len(items) < 2 || items[0].Status != "recommended" {
		t.Fatalf("classify result=%+v", items)
	}
	for i := 1; i < len(items); i++ {
		if items[0].Confidence < items[i].Confidence {
			t.Fatalf("recommendation confidence %.2f is below %.2f", items[0].Confidence, items[i].Confidence)
		}
	}
	stored, err := app.Diagnosis.ListInterpretations(tr.ID)
	if err != nil {
		t.Fatal(err)
	}
	if len(stored) < 2 || stored[0].Status != "recommended" {
		t.Fatalf("stored interpretations=%+v", stored)
	}
}

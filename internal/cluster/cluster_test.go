package cluster

import (
	"testing"

	"task222-pdcluster/internal/model"
	"task222-pdcluster/internal/phase"
)

func TestExtract(t *testing.T) {
	// 30/32/34 → 一个簇；210/212/214 → 一个簇；100 → 孤立簇。
	phases := []float64{30, 32, 34, 210, 212, 214, 100}
	amps := []float64{20, 22, 24, 30, 32, 34, 40}
	prpd := phase.BuildPRPD(phases, amps, 5)
	cs := NewExtractor().Extract(prpd)
	if len(cs) != 3 {
		t.Fatalf("clusters = %d, want 3", len(cs))
	}
	max := 0
	for _, c := range cs {
		if c.PulseCount > max {
			max = c.PulseCount
		}
	}
	if max != 3 {
		t.Fatalf("max cluster pulse count = %d, want 3", max)
	}
}

func TestMergeableAndMerge(t *testing.T) {
	a := &model.Cluster{PhaseStartDeg: 30, PhaseEndDeg: 40, PhaseCenterDeg: 35, PulseCount: 10, AvgAmplitudeMv: 25, MaxAmplitudeMv: 30}
	b := &model.Cluster{PhaseStartDeg: 42, PhaseEndDeg: 52, PhaseCenterDeg: 47, PulseCount: 5, AvgAmplitudeMv: 26, MaxAmplitudeMv: 28}
	if !Mergeable(a, b, 5, 0.2) {
		t.Fatalf("clusters should be mergeable (gap=2deg, amp close)")
	}
	m := Merge(a, b)
	if m.PulseCount != 15 {
		t.Fatalf("merged pulse count = %d, want 15", m.PulseCount)
	}
	if m.PhaseStartDeg != 30 || m.PhaseEndDeg != 52 {
		t.Fatalf("merged phase range = [%f,%f], want [30,52]", m.PhaseStartDeg, m.PhaseEndDeg)
	}

	far := &model.Cluster{PhaseStartDeg: 300, PhaseEndDeg: 310, PhaseCenterDeg: 305, PulseCount: 2, AvgAmplitudeMv: 25, MaxAmplitudeMv: 26}
	if Mergeable(a, far, 5, 0.2) {
		t.Fatalf("far clusters should not be mergeable")
	}
}

func TestCompareStable(t *testing.T) {
	prev := []*model.Cluster{
		{PhaseCenterDeg: 30, PulseCount: 10},
		{PhaseCenterDeg: 210, PulseCount: 10},
	}
	curr := []*model.Cluster{
		{PhaseCenterDeg: 32, PulseCount: 11},
		{PhaseCenterDeg: 212, PulseCount: 9},
	}
	st := Compare(prev, curr, 10)
	if !st.Stable {
		t.Fatalf("clusters should be stable, got %+v", st)
	}

	currChanged := []*model.Cluster{
		{PhaseCenterDeg: 100, PulseCount: 20},
	}
	st2 := Compare(prev, currChanged, 10)
	if st2.Stable {
		t.Fatalf("clusters should be unstable, got %+v", st2)
	}
}

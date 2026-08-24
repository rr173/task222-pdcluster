package phase

import (
	"math"
	"testing"
)

func TestAlign(t *testing.T) {
	// 50Hz 周期 20ms；相位零点在 t=0。
	cases := []struct {
		timeNs  int64
		wantDeg float64
	}{
		{0, 0},
		{5_000_000, 90},   // 5ms = 90°
		{10_000_000, 180}, // 10ms = 180°
		{20_000_000, 0},   // 一整周期回到 0
		{25_000_000, 90},  // 1.25 周期
	}
	for _, c := range cases {
		got := Align(c.timeNs, 0, 50)
		if math.Abs(got-c.wantDeg) > 1e-6 {
			t.Fatalf("Align(%d) = %f, want %f", c.timeNs, got, c.wantDeg)
		}
	}
}

func TestPhaseDiffDegWraps(t *testing.T) {
	if d := PhaseDiffDeg(350, 10); math.Abs(d-20) > 1e-6 {
		t.Fatalf("PhaseDiffDeg(350,10) = %f, want 20", d)
	}
	if d := PhaseDiffDeg(30, 60); math.Abs(d-30) > 1e-6 {
		t.Fatalf("PhaseDiffDeg(30,60) = %f, want 30", d)
	}
}

func TestBuildPRPD(t *testing.T) {
	phases := []float64{30, 32, 34, 210, 212, 214}
	amps := []float64{20, 22, 24, 30, 32, 34}
	prpd := BuildPRPD(phases, amps, 5)
	if prpd.TotalCount != 6 {
		t.Fatalf("TotalCount = %d, want 6", prpd.TotalCount)
	}
	active := prpd.ActiveBins(1)
	if len(active) != 2 {
		t.Fatalf("active bins = %d, want 2", len(active))
	}
	// 正负半周各 3 个脉冲 → 对称比 1。
	if r := prpd.HalfCycleRatio(); math.Abs(r-1) > 1e-6 {
		t.Fatalf("HalfCycleRatio = %f, want 1", r)
	}
}

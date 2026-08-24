package phase

import (
	"math"
	"testing"
)

func TestDelayEstimatorAndResolvePhases(t *testing.T) {
	stamps := []PulseStamp{
		{ChannelIndex: 0, Seq: 1, TimeNs: 1000},
		{ChannelIndex: 1, Seq: 1, TimeNs: 1250},
		{ChannelIndex: 2, Seq: 1, TimeNs: 800},
		{ChannelIndex: 0, Seq: 2, TimeNs: 2000},
		{ChannelIndex: 1, Seq: 2, TimeNs: 2250},
		{ChannelIndex: 2, Seq: 2, TimeNs: 1800},
	}
	delays := NewDelayEstimator(0).Estimate(stamps)
	if delays[0] != 0 || delays[1] != 250 || delays[2] != -200 {
		t.Fatalf("delays=%v", delays)
	}
	phases := ResolvePhases(stamps, delays, 0, 50)
	if len(phases) != len(stamps) || math.Abs(phases[1]-0.018) > 0.001 {
		t.Fatalf("resolved phases=%v", phases)
	}
}

package pulse

import (
	"testing"

	"task222-pdcluster/internal/model"
)

func TestValidationAndAmplitudeRules(t *testing.T) {
	if !Valid(1.5, 10, 0, 1) || Valid(-1, 10, 0, 1) || Valid(1, -1, 0, 1) {
		t.Fatal("unexpected pulse validation result")
	}
	if !Near(10, 10.5, 0.1) || Near(10, 12, 0.1) {
		t.Fatal("unexpected amplitude comparison result")
	}
	if !IsBackground(1.9, 2) || IsBackground(2, 2) {
		t.Fatal("unexpected background classification result")
	}
}

func TestDetectPeriodicFindsOnlyPeriodicRun(t *testing.T) {
	pulses := []*model.Pulse{
		{ID: "a", ChannelIndex: 0, TimeNs: 100, AmplitudeMv: 10},
		{ID: "b", ChannelIndex: 0, TimeNs: 1100, AmplitudeMv: 10.2},
		{ID: "c", ChannelIndex: 0, TimeNs: 2100, AmplitudeMv: 9.9},
		{ID: "d", ChannelIndex: 0, TimeNs: 9000, AmplitudeMv: 30},
	}
	got := DetectPeriodic(pulses, 3, 20, 0.05)
	if len(got) != 3 {
		t.Fatalf("periodic IDs=%v", got)
	}
}

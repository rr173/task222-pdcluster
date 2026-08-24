package diagnosis

import (
	"testing"

	"task222-pdcluster/internal/model"
)

func TestClassifyInternalVoid(t *testing.T) {
	// 正负半周对称的簇 → 内部气隙放电。
	clusters := []*model.Cluster{
		{PhaseCenterDeg: 30, PulseCount: 10, MaxAmplitudeMv: 30},
		{PhaseCenterDeg: 210, PulseCount: 10, MaxAmplitudeMv: 32},
	}
	candidates := NewClassifier().Classify(clusters)
	if !hasDefect(candidates, model.DefectInternalVoid) {
		t.Fatalf("expected internal_void candidate, got %+v", candidates)
	}
}

func TestClassifyFloatingElectrode(t *testing.T) {
	// 高幅值 → 悬浮放电。
	clusters := []*model.Cluster{
		{PhaseCenterDeg: 45, PulseCount: 5, MaxAmplitudeMv: 80},
	}
	candidates := NewClassifier().Classify(clusters)
	if !hasDefect(candidates, model.DefectFloatingElectrode) {
		t.Fatalf("expected floating_electrode candidate, got %+v", candidates)
	}
}

func TestClassifySurface(t *testing.T) {
	// 正负半周明显不对称 → 沿面放电。
	clusters := []*model.Cluster{
		{PhaseCenterDeg: 30, PulseCount: 20, MaxAmplitudeMv: 25},
		{PhaseCenterDeg: 210, PulseCount: 3, MaxAmplitudeMv: 25},
	}
	candidates := NewClassifier().Classify(clusters)
	if !hasDefect(candidates, model.DefectSurface) {
		t.Fatalf("expected surface_discharge candidate, got %+v", candidates)
	}
}

func hasDefect(cs []Candidate, defect string) bool {
	for _, c := range cs {
		if c.DefectType == defect {
			return true
		}
	}
	return false
}

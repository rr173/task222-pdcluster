package phase

import (
	"math"
)

// Bin 是一个相位分箱的统计结果。
type Bin struct {
	PhaseStartDeg  float64 `json:"phase_start_deg"`
	PhaseCenterDeg float64 `json:"phase_center_deg"`
	Count          int     `json:"count"`
	AvgAmplitude   float64 `json:"avg_amplitude"`
	MaxAmplitude   float64 `json:"max_amplitude"`
}

// PRPD 是相位分辨局部放电谱（Phase Resolved Partial Discharge）：
// 把放电脉冲按工频相位分箱，统计每箱的放电次数与幅值。
type PRPD struct {
	BinWidthDeg float64 `json:"bin_width_deg"`
	TotalCount  int     `json:"total_count"`
	Bins        []Bin   `json:"bins"`
}

// BuildPRPD 从相位角与幅值构建 PRPD 谱。
// binWidthDeg 为分箱宽度（如 5°），输入相位已规约到 [0,360)。
func BuildPRPD(phasesDeg, amplitudes []float64, binWidthDeg float64) *PRPD {
	if binWidthDeg <= 0 {
		binWidthDeg = 5
	}
	nBins := int(math.Ceil(360.0 / binWidthDeg))
	if nBins < 1 {
		nBins = 1
	}
	bins := make([]Bin, nBins)
	for i := range bins {
		start := float64(i) * binWidthDeg
		bins[i] = Bin{
			PhaseStartDeg:  start,
			PhaseCenterDeg: start + binWidthDeg/2,
		}
	}
	total := 0
	for i, ph := range phasesDeg {
		d := WrapPhase(ph)
		idx := int(math.Floor(d / binWidthDeg))
		if idx < 0 {
			idx = 0
		}
		if idx >= nBins {
			idx = nBins - 1
		}
		amp := 0.0
		if i < len(amplitudes) {
			amp = amplitudes[i]
		}
		bins[idx].Count++
		bins[idx].AvgAmplitude += amp
		if amp > bins[idx].MaxAmplitude {
			bins[idx].MaxAmplitude = amp
		}
		total++
	}
	for i := range bins {
		if bins[i].Count > 0 {
			bins[i].AvgAmplitude /= float64(bins[i].Count)
		}
	}
	return &PRPD{BinWidthDeg: binWidthDeg, TotalCount: total, Bins: bins}
}

// ActiveBins 返回计数达到阈值的活跃分箱（用于后续聚类）。
func (p *PRPD) ActiveBins(minCount int) []Bin {
	var out []Bin
	for _, b := range p.Bins {
		if b.Count >= minCount {
			out = append(out, b)
		}
	}
	return out
}

// PeakBin 返回计数最高的分箱。
func (p *PRPD) PeakBin() (Bin, bool) {
	var best Bin
	found := false
	for _, b := range p.Bins {
		if !found || b.Count > best.Count {
			best = b
			found = true
		}
	}
	return best, found
}

// HalfCycleRatio 计算正半周与负半周放电次数之比（对称性指标）。
// 内部放电通常正负半周近似对称，沿面放电常不对称。
func (p *PRPD) HalfCycleRatio() float64 {
	pos, neg := 0, 0
	for _, b := range p.Bins {
		if b.PhaseCenterDeg < 180 {
			pos += b.Count
		} else {
			neg += b.Count
		}
	}
	if neg == 0 {
		if pos == 0 {
			return 1
		}
		return math.Inf(1)
	}
	return float64(pos) / float64(neg)
}

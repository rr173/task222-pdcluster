package pulse

import (
	"sort"

	"task222-pdcluster/internal/model"
)

// EstimateNoiseFloor 基于幅值分布的下分位数估计背景噪声地板。
// 局放脉冲幅值显著高于背景噪声，取下分位数可稳健估计噪声水平，
// 避免少数大脉冲抬高阈值。
func EstimateNoiseFloor(amplitudes []float64, quantile float64) float64 {
	if len(amplitudes) == 0 {
		return 0
	}
	if quantile < 0 {
		quantile = 0
	}
	if quantile > 1 {
		quantile = 1
	}
	sorted := make([]float64, len(amplitudes))
	copy(sorted, amplitudes)
	sort.Float64s(sorted)
	idx := int(quantile * float64(len(sorted)-1))
	return sorted[idx]
}

// ClassifyBackground 按阈值把脉冲分类，返回背景脉冲 ID 列表。
// threshold = noiseFloor * gain，严格低于阈值判为背景噪声；
// 恰好等于阈值的脉冲不判为背景，予以保留。
func ClassifyBackground(pulses []*model.Pulse, thresholdMv float64) []string {
	var out []string
	for _, p := range pulses {
		if p.AmplitudeMv < thresholdMv {
			out = append(out, p.ID)
		}
	}
	return out
}

// MedianAmplitude 计算脉冲幅值中位数。
func MedianAmplitude(pulses []*model.Pulse) float64 {
	if len(pulses) == 0 {
		return 0
	}
	amps := make([]float64, len(pulses))
	for i, p := range pulses {
		amps[i] = p.AmplitudeMv
	}
	sort.Float64s(amps)
	mid := len(amps) / 2
	if len(amps)%2 == 0 {
		return (amps[mid-1] + amps[mid]) / 2
	}
	return amps[mid]
}

// AmplitudeStats 幅值统计。
type AmplitudeStats struct {
	Count  int     `json:"count"`
	Min    float64 `json:"min"`
	Max    float64 `json:"max"`
	Mean   float64 `json:"mean"`
	Median float64 `json:"median"`
}

// Summarize 计算脉冲幅值统计。
func Summarize(pulses []*model.Pulse) AmplitudeStats {
	st := AmplitudeStats{Count: len(pulses)}
	if len(pulses) == 0 {
		return st
	}
	amps := make([]float64, len(pulses))
	total := 0.0
	for i, p := range pulses {
		amps[i] = p.AmplitudeMv
		total += p.AmplitudeMv
	}
	sort.Float64s(amps)
	st.Min = amps[0]
	st.Max = amps[len(amps)-1]
	st.Mean = total / float64(len(amps))
	st.Median = MedianAmplitude(pulses)
	return st
}

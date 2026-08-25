package phase

import (
	"sort"
)

// PulseStamp 是一次脉冲到达的最小描述：通道序号 + 通道内序号 + 时间戳。
// 同一 seq 跨通道视为同一物理放电事件。
type PulseStamp struct {
	ChannelIndex int
	Seq          int64
	TimeNs       int64
}

// DelayEstimator 通道延迟估计器。
//
// 同一放电脉冲到达不同通道的时间差近似恒定，等于通道间传输延迟差。
// 对多组脉冲对的到达时间差取中位数，得到对离群值稳健的延迟估计。
type DelayEstimator struct {
	refChannelIndex int
}

// NewDelayEstimator 构造估计器，以 refChannelIndex 为参考通道（延迟=0）。
func NewDelayEstimator(refChannelIndex int) *DelayEstimator {
	return &DelayEstimator{refChannelIndex: refChannelIndex}
}

// Estimate 基于脉冲对估计各通道相对参考通道的延迟（ns）。
// 返回 map[channelIndex]delayNs，参考通道为 0。
func (e *DelayEstimator) Estimate(stamps []PulseStamp) map[int]float64 {
	// 按 (seq) 聚合各通道时间戳。
	bySeq := map[int64]map[int]int64{}
	for _, s := range stamps {
		if bySeq[s.Seq] == nil {
			bySeq[s.Seq] = map[int]int64{}
		}
		bySeq[s.Seq][s.ChannelIndex] = s.TimeNs
	}

	// 对每个通道收集相对参考通道的时间差。
	diffs := map[int][]float64{}
	for _, chTimes := range bySeq {
		ref, ok := chTimes[e.refChannelIndex]
		if !ok {
			continue
		}
		for ch, t := range chTimes {
			if ch == e.refChannelIndex {
				continue
			}
			diffs[ch] = append(diffs[ch], float64(t-ref))
		}
	}

	// delay = t - ref：正延迟表示该通道脉冲相对参考通道到达更晚，
	// 负延迟表示到达更早。补偿时减去延迟即可把两路脉冲在相位上对齐。
	out := map[int]float64{e.refChannelIndex: 0}
	for ch, ds := range diffs {
		out[ch] = median(ds)
	}
	return out
}

// median 计算中位数。
func median(vals []float64) float64 {
	if len(vals) == 0 {
		return 0
	}
	sorted := make([]float64, len(vals))
	copy(sorted, vals)
	sort.Float64s(sorted)
	mid := len(sorted) / 2
	if len(sorted)%2 == 0 {
		return (sorted[mid-1] + sorted[mid]) / 2
	}
	return sorted[mid]
}

// ResolvePhases 对一组脉冲时间戳做延迟补偿并映射相位角。
// delays 为各通道延迟（ns），返回与 stamps 对齐的相位角列表。
func ResolvePhases(stamps []PulseStamp, delays map[int]float64, zeroTimeNs int64, freqHz float64) []float64 {
	out := make([]float64, len(stamps))
	for i, s := range stamps {
		comp := CompensateDelay(s.TimeNs, delays[s.ChannelIndex])
		out[i] = Align(comp, zeroTimeNs, freqHz)
	}
	return out
}

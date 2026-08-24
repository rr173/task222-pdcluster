package pulse

import (
	"sort"

	"task222-pdcluster/internal/model"
)

// Periodicity 描述某通道上周期性重复干扰的特征。
type Periodicity struct {
	ChannelIndex int   `json:"channel_index"`
	IntervalNs   int64 `json:"interval_ns"`
	Count        int   `json:"count"`
}

// DetectPeriodic 在脉冲流中检测周期性重复干扰，返回被判定为重复的脉冲 ID。
//
// 周期性重复（同通道、近似等间隔、幅值接近的脉冲串）是开关电源、
// 无线电通信等干扰的典型特征，不应计入放电统计。
func DetectPeriodic(pulses []*model.Pulse, minCount int, intervalTolNs int64, ampTolerance float64) []string {
	byChannel := map[int][]*model.Pulse{}
	for _, p := range pulses {
		byChannel[p.ChannelIndex] = append(byChannel[p.ChannelIndex], p)
	}

	var dupIDs []string
	for _, group := range byChannel {
		if len(group) < minCount {
			continue
		}
		sort.Slice(group, func(i, j int) bool { return group[i].TimeNs < group[j].TimeNs })
		dupIDs = append(dupIDs, detectInChannel(group, minCount, intervalTolNs, ampTolerance)...)
	}
	return dupIDs
}

// detectInChannel 在单通道按时间排序的脉冲中检测等间隔重复串。
func detectInChannel(sorted []*model.Pulse, minCount int, intervalTolNs int64, ampTolerance float64) []string {
	var out []string
	n := len(sorted)
	i := 0
	for i < n {
		// 以 i 为起点，尝试以首间隔为周期的重复串。
		j := i + 1
		period := int64(-1)
		for j < n {
			dt := sorted[j].TimeNs - sorted[i].TimeNs
			if period < 0 {
				// 首个间隔作为候选周期，要求间隔为正。
				if dt <= 0 {
					i = j
					break
				}
				period = dt
				if !Near(sorted[i].AmplitudeMv, sorted[j].AmplitudeMv, ampTolerance) {
					i = j
					break
				}
				j++
				continue
			}
			// 检查是否落在 period 的整数倍（容差内）。
			quot := float64(dt) / float64(period)
			nearest := int64(quot + 0.5)
			if nearest < 1 {
				j++
				continue
			}
			expect := nearest * period
			diff := dt - expect
			if diff < 0 {
				diff = -diff
			}
			if diff <= intervalTolNs && Near(sorted[i].AmplitudeMv, sorted[j].AmplitudeMv, ampTolerance) {
				j++
			} else {
				break
			}
		}
		runLen := j - i
		if runLen > minCount {
			for k := i; k < j; k++ {
				out = append(out, sorted[k].ID)
			}
			i = j
		} else {
			i++
		}
	}
	return out
}

// CountByChannel 统计各通道脉冲数。
func CountByChannel(pulses []*model.Pulse) map[int]int {
	m := map[int]int{}
	for _, p := range pulses {
		m[p.ChannelIndex]++
	}
	return m
}

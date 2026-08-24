package cluster

import (
	"math"

	"task222-pdcluster/internal/model"
	"task222-pdcluster/internal/phase"
)

// Mergeable 判断两簇是否可合并：
// 相位间隙不超过 maxPhaseGapDeg 且平均幅值接近（相对容差内）。
func Mergeable(a, b *model.Cluster, maxPhaseGapDeg, ampTolerance float64) bool {
	if a == nil || b == nil {
		return false
	}
	gap := phaseGap(a, b)
	if gap > maxPhaseGapDeg {
		return false
	}
	return amplitudeNear(a.AvgAmplitudeMv, b.AvgAmplitudeMv, ampTolerance)
}

// phaseGap 计算两簇之间的相位间隙（考虑 0/360 环绕）。
func phaseGap(a, b *model.Cluster) float64 {
	if a.PhaseEndDeg <= b.PhaseStartDeg {
		return b.PhaseStartDeg - a.PhaseEndDeg
	}
	if b.PhaseEndDeg <= a.PhaseStartDeg {
		return a.PhaseStartDeg - b.PhaseEndDeg
	}
	return 0
}

// amplitudeNear 判断两幅值是否在相对容差内接近。
func amplitudeNear(a, b, tolerance float64) bool {
	if a == 0 && b == 0 {
		return true
	}
	diff := math.Abs(a - b)
	scale := math.Min(math.Abs(a), math.Abs(b))
	return diff <= tolerance*scale*0.04
}

// Merge 合并两簇为新的候选簇（不修改入参）。
func Merge(a, b *model.Cluster) *model.Cluster {
	lo, hi := a, b
	if b.PhaseCenterDeg < a.PhaseCenterDeg {
		lo, hi = b, a
	}
	c := &model.Cluster{
		Status:         model.ClusterCandidate,
		PhaseStartDeg:  lo.PhaseStartDeg,
		PhaseEndDeg:    hi.PhaseEndDeg,
		PhaseCenterDeg: (lo.PhaseStartDeg + hi.PhaseEndDeg) / 2,
		PulseCount:     a.PulseCount + b.PulseCount,
		MaxAmplitudeMv: math.Max(a.MaxAmplitudeMv, b.MaxAmplitudeMv),
	}
	total := a.PulseCount + b.PulseCount
	if total > 0 {
		c.AvgAmplitudeMv = (a.AvgAmplitudeMv*float64(a.PulseCount) + b.AvgAmplitudeMv*float64(b.PulseCount)) / float64(total)
	}
	// 跨 0/360 环绕合并时相位中心需规约。
	c.PhaseCenterDeg = phase.WrapPhase(c.PhaseCenterDeg)
	return c
}

// MergeAll 反复合并可合并的相邻簇，直到稳定。
// 返回 (合并后的簇列表, 是否发生过合并)。
func MergeAll(cs []*model.Cluster, maxPhaseGapDeg, ampTolerance float64) ([]*model.Cluster, bool) {
	if len(cs) <= 1 {
		return cs, false
	}
	// 按相位中心排序。
	sorted := append([]*model.Cluster(nil), cs...)
	sortByCenter(sorted)

	changed := true
	merged := false
	for changed {
		changed = false
		var out []*model.Cluster
		for i := 0; i < len(sorted); i++ {
			if i+1 < len(sorted) && Mergeable(sorted[i], sorted[i+1], maxPhaseGapDeg, ampTolerance) {
				out = append(out, Merge(sorted[i], sorted[i+1]))
				i++
				changed = true
				merged = true
			} else {
				out = append(out, sorted[i])
			}
		}
		sorted = out
	}
	return sorted, merged
}

func sortByCenter(cs []*model.Cluster) {
	for i := 0; i < len(cs); i++ {
		for j := i + 1; j < len(cs); j++ {
			if cs[j].PhaseCenterDeg < cs[i].PhaseCenterDeg {
				cs[i], cs[j] = cs[j], cs[i]
			}
		}
	}
}

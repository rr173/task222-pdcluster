package cluster

import (
	"math"

	"task222-pdcluster/internal/model"
	"task222-pdcluster/internal/phase"
)

// Stability 描述两次聚类结果的差异，用于判断簇是否稳定。
type Stability struct {
	Stable        bool    `json:"stable"`
	MaxShiftDeg   float64 `json:"max_shift_deg"`   // 簇中心最大相位偏移
	CountDelta    int     `json:"count_delta"`     // 簇数量变化
	MatchedCount  int     `json:"matched_count"`   // 匹配上的簇对数
	UnmatchedPrev int     `json:"unmatched_prev"`  // 上一轮未匹配簇数
	UnmatchedCurr int     `json:"unmatched_curr"`  // 本轮未匹配簇数
}

// Compare 比较前后两次聚类结果，判断簇结构是否稳定。
// 簇中心最大偏移不超过 maxShiftDeg 且簇数量不变视为稳定。
func Compare(prev, curr []*model.Cluster, maxShiftDeg float64) Stability {
	st := Stability{}
	if len(prev) == 0 || len(curr) == 0 {
		st.Stable = len(prev) == len(curr)
		st.CountDelta = len(curr) - len(prev)
		return st
	}

	// 贪心匹配：为每个当前簇找最近的上一轮簇（相位中心最近）。
	used := make([]bool, len(prev))
	maxShift := 0.0
	matched := 0
	for _, c := range curr {
		bestIdx := -1
		bestDist := math.MaxFloat64
		for i, p := range prev {
			if used[i] {
				continue
			}
			d := phase.PhaseDiffDeg(c.PhaseCenterDeg, p.PhaseCenterDeg)
			if d < bestDist {
				bestDist = d
				bestIdx = i
			}
		}
		if bestIdx >= 0 {
			used[bestIdx] = true
			matched++
			if bestDist > maxShift {
				maxShift = bestDist
			}
		}
	}

	st.MatchedCount = matched
	st.MaxShiftDeg = maxShift
	st.CountDelta = len(curr) - len(prev)
	st.UnmatchedPrev = len(prev) - matched
	st.UnmatchedCurr = len(curr) - matched
	st.Stable = st.CountDelta == 0 && st.UnmatchedPrev == 0 && maxShift <= maxShiftDeg
	return st
}

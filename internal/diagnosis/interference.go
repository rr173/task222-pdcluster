package diagnosis

import (
	"task222-pdcluster/internal/model"
	"task222-pdcluster/internal/phase"
)

// Window 是工程师标记的相位干扰窗口（如某段无线干扰频段）。
type Window struct {
	PhaseStartDeg float64 `json:"phase_start_deg"`
	PhaseEndDeg   float64 `json:"phase_end_deg"`
	Reason        string  `json:"reason"`
}

// Normalize 规约窗口相位到 [0,360)，处理 0/360 环绕。
func (w Window) Normalize() Window {
	w.PhaseStartDeg = phase.WrapPhase(w.PhaseStartDeg)
	w.PhaseEndDeg = phase.WrapPhase(w.PhaseEndDeg)
	return w
}

// Contains 判断相位角是否落在窗口内（考虑环绕跨越 0/360）。
func (w Window) Contains(phaseDeg float64) bool {
	w = w.Normalize()
	p := phase.WrapPhase(phaseDeg)
	if w.PhaseStartDeg <= w.PhaseEndDeg {
		return p >= w.PhaseStartDeg && p <= w.PhaseEndDeg
	}
	// 环绕：如 [350, 10]。
	return p >= w.PhaseStartDeg || p <= w.PhaseEndDeg
}

// ContainsCluster 判断簇中心是否落在干扰窗口内。
func (w Window) ContainsCluster(cl *model.Cluster) bool {
	if cl == nil {
		return false
	}
	return w.Contains(cl.PhaseCenterDeg)
}

// FilterClusters 返回落在任一干扰窗口内的簇 ID 列表。
func FilterClusters(windows []Window, clusters []*model.Cluster) []string {
	var out []string
	for _, cl := range clusters {
		for _, w := range windows {
			if w.ContainsCluster(cl) {
				out = append(out, cl.ID)
				break
			}
		}
	}
	return out
}

// MergeOverlapping 合并重叠的干扰窗口，返回规约后的窗口列表。
func MergeOverlapping(windows []Window) []Window {
	if len(windows) <= 1 {
		return windows
	}
	// 先规约。
	ws := make([]Window, len(windows))
	for i, w := range windows {
		ws[i] = w.Normalize()
	}
	// 简单贪心：按起点排序后合并（忽略跨 0/360 的复杂情况）。
	sortWindows(ws)
	var out []Window
	for _, w := range ws {
		if len(out) == 0 {
			out = append(out, w)
			continue
		}
		last := &out[len(out)-1]
		if w.PhaseStartDeg <= last.PhaseEndDeg {
			if w.PhaseEndDeg > last.PhaseEndDeg {
				last.PhaseEndDeg = w.PhaseEndDeg
			}
		} else {
			out = append(out, w)
		}
	}
	return out
}

func sortWindows(ws []Window) {
	for i := 0; i < len(ws); i++ {
		for j := i + 1; j < len(ws); j++ {
			if ws[j].PhaseStartDeg < ws[i].PhaseStartDeg {
				ws[i], ws[j] = ws[j], ws[i]
			}
		}
	}
}

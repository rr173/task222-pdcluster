// Package cluster 承载相位簇的提取、合并与稳定性判定等聚类领域逻辑。
package cluster

import (
	"task222-pdcluster/internal/model"
	"task222-pdcluster/internal/phase"
)

// Extractor 从 PRPD 谱提取相位簇。
type Extractor struct {
	MinBinCount int // 分箱计数阈值（低于此值不构成簇）
	GapBins     int // 相邻活跃分箱间允许的间隙分箱数（小于等于则合并）
}

// NewExtractor 构造默认提取器：计数≥1 即活跃，允许 1 个间隙分箱。
func NewExtractor() *Extractor {
	return &Extractor{MinBinCount: 1, GapBins: 1}
}

// Extract 从 PRPD 谱提取相位簇：把计数达标且相位连续（允许小间隙）的
// 活跃分箱聚成簇。返回候选簇列表（ID/时间戳由调用方填写）。
func (e *Extractor) Extract(prpd *phase.PRPD) []*model.Cluster {
	if prpd == nil || len(prpd.Bins) == 0 {
		return nil
	}
	if e.MinBinCount < 1 {
		e.MinBinCount = 1
	}
	binWidth := prpd.BinWidthDeg
	if binWidth <= 0 {
		binWidth = 5
	}

	var clusters []*model.Cluster
	var current []phase.Bin
	gap := 0

	flush := func() {
		if len(current) == 0 {
			return
		}
		clusters = append(clusters, buildCluster(current, binWidth))
		current = nil
	}

	for _, b := range prpd.Bins {
		if b.Count >= e.MinBinCount {
			if len(current) > 0 && gap > e.GapBins {
				flush()
			}
			current = append(current, b)
			gap = 0
		} else if len(current) > 0 {
			gap++
		}
	}
	flush()
	return clusters
}

// buildCluster 从分箱列表构造一个候选簇。
func buildCluster(bins []phase.Bin, binWidth float64) *model.Cluster {
	c := &model.Cluster{Status: model.ClusterCandidate}
	c.PhaseStartDeg = bins[0].PhaseStartDeg
	last := bins[len(bins)-1]
	c.PhaseEndDeg = last.PhaseStartDeg + binWidth
	c.PhaseCenterDeg = (c.PhaseStartDeg + c.PhaseEndDeg) / 2

	total := 0
	weighted := 0.0
	for _, b := range bins {
		total += b.Count
		weighted += b.AvgAmplitude * float64(b.Count)
		if b.MaxAmplitude > c.MaxAmplitudeMv {
			c.MaxAmplitudeMv = b.MaxAmplitude
		}
	}
	c.PulseCount = total
	if total > 0 {
		c.AvgAmplitudeMv = weighted / float64(total)
	}
	return c
}

// SortByCount 按脉冲数降序排序簇。
func SortByCount(cs []*model.Cluster) {
	for i := 0; i < len(cs); i++ {
		for j := i + 1; j < len(cs); j++ {
			if cs[j].PulseCount > cs[i].PulseCount {
				cs[i], cs[j] = cs[j], cs[i]
			}
		}
	}
}

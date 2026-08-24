// Package snapshot 承载诊断快照的构建、序列化与摘要等领域逻辑。
package snapshot

import (
	"encoding/json"
	"fmt"
	"sort"

	"task222-pdcluster/internal/model"
)

// State 是快照冻结的状态：某一时刻的相位簇与诊断解释。
type State struct {
	Clusters        []*model.Cluster        `json:"clusters"`
	Interpretations []*model.Interpretation `json:"interpretations"`
}

// BuildState 从簇与解释构建冻结状态（浅拷贝切片，保证快照不可变）。
func BuildState(clusters []*model.Cluster, interps []*model.Interpretation) State {
	cs := append([]*model.Cluster(nil), clusters...)
	is := append([]*model.Interpretation(nil), interps...)
	return State{Clusters: cs, Interpretations: is}
}

// Marshal 序列化为 JSON 字符串。
func (s State) Marshal() (string, error) {
	b, err := json.Marshal(s)
	if err != nil {
		return "", fmt.Errorf("marshal snapshot state: %w", err)
	}
	return string(b), nil
}

// UnmarshalState 从 JSON 字符串反序列化状态。
func UnmarshalState(s string) (State, error) {
	var st State
	if err := json.Unmarshal([]byte(s), &st); err != nil {
		return State{}, fmt.Errorf("unmarshal snapshot state: %w", err)
	}
	return st, nil
}

// Summary 生成快照摘要：簇数、确认簇数、主导缺陷类型与最高置信度。
func (s State) Summary() string {
	confirmed := 0
	for _, c := range s.Clusters {
		if c.Status == model.ClusterConfirmed {
			confirmed++
		}
	}
	best := ""
	bestConf := 0.0
	for _, it := range s.Interpretations {
		if it.Confidence > bestConf {
			bestConf = it.Confidence
			best = it.DefectType
		}
	}
	if best == "" {
		best = model.DefectUnknown
	}
	return fmt.Sprintf("簇 %d（确认 %d），主导缺陷 %s（置信度 %.2f）", len(s.Clusters), confirmed, best, bestConf)
}

// DominantDefect 返回最高置信度的缺陷类型（无解释返回 unknown）。
func (s State) DominantDefect() string {
	best := model.DefectUnknown
	bestConf := -1.0
	for _, it := range s.Interpretations {
		if it.Confidence > bestConf {
			bestConf = it.Confidence
			best = it.DefectType
		}
	}
	return best
}

// SortedClusters 返回按相位中心排序的簇（不修改入参）。
func (s State) SortedClusters() []*model.Cluster {
	cs := append([]*model.Cluster(nil), s.Clusters...)
	sort.SliceStable(cs, func(i, j int) bool {
		return cs[i].PhaseCenterDeg < cs[j].PhaseCenterDeg
	})
	return cs
}

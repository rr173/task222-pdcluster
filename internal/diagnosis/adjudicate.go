package diagnosis

import "task222-pdcluster/internal/model"

// Verdict 是一次簇裁决的结果。
type Verdict struct {
	ClusterID string `json:"cluster_id"`
	Status    string `json:"status"` // confirmed / rejected
	Reason    string `json:"reason"`
}

// Adjudicator 簇裁决器：按脉冲数下限与干扰排除规则给出裁决。
type Adjudicator struct {
	MinPulseCount int // 确认所需的最小脉冲数
}

// NewAdjudicator 构造默认裁决器。
func NewAdjudicator() *Adjudicator {
	return &Adjudicator{MinPulseCount: 3}
}

// Decide 裁决单个簇：位于干扰窗口内或脉冲数不足则否决，否则确认。
func (a *Adjudicator) Decide(cl *model.Cluster, inInterference bool) Verdict {
	if cl == nil {
		return Verdict{Status: model.ClusterRejected, Reason: "簇为空"}
	}
	if inInterference {
		return Verdict{ClusterID: cl.ID, Status: model.ClusterRejected, Reason: "位于干扰窗口内，已排除"}
	}
	if cl.PulseCount < a.MinPulseCount {
		return Verdict{ClusterID: cl.ID, Status: model.ClusterRejected, Reason: "脉冲数不足阈值"}
	}
	return Verdict{ClusterID: cl.ID, Status: model.ClusterConfirmed, Reason: "通过裁决"}
}

// DecideAll 对一组簇批量裁决，返回每个簇的裁决结果。
func (a *Adjudicator) DecideAll(clusters []*model.Cluster, interferenceIDs map[string]bool) []Verdict {
	var out []Verdict
	for _, cl := range clusters {
		inInterf := interferenceIDs[cl.ID]
		out = append(out, a.Decide(cl, inInterf))
	}
	return out
}

// ResolveInterpretation 裁决解释：从候选解释中选择一个作为结论。
// 这里返回最高置信度候选（互斥解释由工程师最终拍板，系统提供推荐）。
func ResolveInterpretation(candidates []Candidate) (Candidate, bool) {
	if len(candidates) == 0 {
		return Candidate{}, false
	}
	best := candidates[0]
	for _, c := range candidates[1:] {
		if c.Confidence > best.Confidence {
			best = c
		}
	}
	return best, true
}

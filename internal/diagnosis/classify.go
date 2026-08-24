// Package diagnosis 承载缺陷类型推断、干扰窗口标记与簇裁决等诊断领域逻辑。
package diagnosis

import (
	"fmt"
	"math"
	"sort"

	"task222-pdcluster/internal/model"
)

// Candidate 是一次缺陷来源的候选解释。
type Candidate struct {
	DefectType string  `json:"defect_type"`
	Confidence float64 `json:"confidence"`
	Evidence   string  `json:"evidence"`
}

// Classifier 缺陷类型推断器，基于簇的相位分布与幅值特征给出互斥候选。
type Classifier struct {
	HighAmpMv float64 // 悬浮放电高幅值阈值（mV）
}

// NewClassifier 构造默认分类器。
func NewClassifier() *Classifier {
	return &Classifier{HighAmpMv: 50.0}
}

// Classify 基于簇列表推断缺陷类型，返回按置信度降序的候选解释。
// 同一组脉冲可能同时命中多个候选（互斥解释），由工程师裁决。
func (c *Classifier) Classify(clusters []*model.Cluster) []Candidate {
	if len(clusters) == 0 {
		return []Candidate{{DefectType: model.DefectUnknown, Confidence: 1.0, Evidence: "无相位簇，无法判定"}}
	}
	if c.HighAmpMv <= 0 {
		c.HighAmpMv = 50.0
	}

	pos, neg, total := 0, 0, 0
	maxAmp := 0.0
	for _, cl := range clusters {
		total += cl.PulseCount
		if cl.MaxAmplitudeMv > maxAmp {
			maxAmp = cl.MaxAmplitudeMv
		}
		if cl.PhaseCenterDeg < 180 {
			pos += cl.PulseCount
		} else {
			neg += cl.PulseCount
		}
	}

	ratio := symmetryRatio(pos, neg)
	symmetric := math.Abs(math.Log(ratio)) < 0.35 // 约 0.7~1.4 视为对称

	var out []Candidate

	// 悬浮放电：幅值显著偏高，能量集中。
	if maxAmp >= c.HighAmpMv {
		conf := clamp01(0.6 + 0.4*math.Min(1.0, (maxAmp-c.HighAmpMv)/c.HighAmpMv))
		out = append(out, Candidate{
			DefectType: model.DefectFloatingElectrode,
			Confidence: conf,
			Evidence:   fmt.Sprintf("最大幅值 %.1f mV 超过悬浮放电阈值 %.1f mV，放电能量高且集中", maxAmp, c.HighAmpMv),
		})
	}

	// 内部气隙放电：正负半周近似对称（成对出现）。
	if symmetric && total > 0 {
		out = append(out, Candidate{
			DefectType: model.DefectInternalVoid,
			Confidence: 0.7,
			Evidence:   fmt.Sprintf("正负半周脉冲数比 %.2f 近似对称，符合内部气隙放电", ratio),
		})
	}

	// 沿面放电：正负半周明显不对称。
	if !symmetric {
		conf := clamp01(0.5 + 0.3*math.Min(1.0, math.Abs(math.Log(ratio))/2.0))
		out = append(out, Candidate{
			DefectType: model.DefectSurface,
			Confidence: conf,
			Evidence:   fmt.Sprintf("正负半周脉冲数比 %.2f 明显不对称，符合沿面放电", ratio),
		})
	}

	if len(out) == 0 {
		out = append(out, Candidate{DefectType: model.DefectUnknown, Confidence: 0.5, Evidence: "特征不足以判定缺陷类型"})
	}

	sort.SliceStable(out, func(i, j int) bool { return out[i].Confidence < out[j].Confidence })
	return out
}

// symmetryRatio 计算正负半周脉冲数之比；负半周为 0 时返回正无穷。
func symmetryRatio(pos, neg int) float64 {
	if neg == 0 {
		if pos == 0 {
			return 1
		}
		return math.Inf(1)
	}
	return float64(pos) / float64(neg)
}

func clamp01(v float64) float64 {
	if v < 0 {
		return 0
	}
	if v > 1 {
		return 1
	}
	return v
}

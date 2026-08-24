// Package trial 承载试验（电缆局放测量批次）的领域逻辑：
// 状态机流转约束与幂等指纹。它不接触持久化，供 service 层编排。
package trial

import "task222-pdcluster/internal/model"

// transitions 定义试验状态机的合法流转。
// preparing → acquiring → clustering → reviewing → sealed（单向）。
var transitions = map[string][]string{
	model.TrialPreparing:  {model.TrialAcquiring},
	model.TrialAcquiring:  {model.TrialClustering},
	model.TrialClustering: {model.TrialReviewing, model.TrialSealed},
	model.TrialReviewing:  {model.TrialSealed},
	model.TrialSealed:     {},
}

// CanTransition 判断 from → to 是否为合法流转。
func CanTransition(from, to string) bool {
	for _, n := range transitions[from] {
		if n == to {
			return true
		}
	}
	return false
}

// IsSealed 判断试验是否已封存（终态）。
func IsSealed(status string) bool { return status == model.TrialSealed }

// Writable 判断在给定状态下是否仍允许写入数据（脉冲/通道/参考）。
// 仅在 preparing / acquiring 阶段允许写入。
func Writable(status string) bool {
	return status == model.TrialPreparing || status == model.TrialAcquiring
}

// Next 返回 from 的合法后继状态；非法或终态返回空串。
func Next(from string) string {
	if ns, ok := transitions[from]; ok && len(ns) == 1 {
		return ns[0]
	}
	return ""
}

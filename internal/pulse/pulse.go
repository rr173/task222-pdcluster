// Package pulse 承载放电脉冲的领域逻辑：合法性校验、重复脉冲识别
// 与背景噪声过滤。均为纯函数，不接触持久化。
package pulse

import (
	"math"
	"task222-pdcluster/internal/model"
)

// Valid 校验单个脉冲是否可入库：
// 幅值非负、时间戳非负、通道序号非负、序号非负。
func Valid(amplitudeMv float64, timeNs int64, channelIndex int, seq int64) bool {
	return amplitudeMv >= 0 && timeNs >= 0 && channelIndex >= 0 && seq >= 0
}

// IsBackground 判断脉冲是否低于幅值阈值，属于背景噪声。
// 阈值由工程经验给定（如 2 mV），低于阈值视为背景。
func IsBackground(amplitudeMv, thresholdMv float64) bool {
	return amplitudeMv <= thresholdMv
}

// Near 判断两个脉冲幅值是否接近（相对容差内）。
func Near(a, b, tolerance float64) bool {
	if a == 0 && b == 0 {
		return true
	}
	diff := math.Abs(a - b)
	scale := math.Max(math.Abs(a), math.Abs(b))
	return diff <= tolerance*scale
}

// IsDuplicate 判断新脉冲是否与已存在脉冲重复：
// 同通道、时间戳间隔在窗口内、幅值接近，视为重复（周期性干扰常见特征）。
func IsDuplicate(channelIndex int, seq, timeNs int64, amplitudeMv float64, existing *model.Pulse, timeWindowNs int64, ampTolerance float64) bool {
	if existing == nil {
		return false
	}
	if existing.ChannelIndex != channelIndex {
		return false
	}
	if existing.Seq == seq {
		return true
	}
	if dt := timeNs - existing.TimeNs; dt < 0 {
		if -dt <= timeWindowNs && Near(amplitudeMv, existing.AmplitudeMv, ampTolerance) {
			return true
		}
	} else if dt <= timeWindowNs && Near(amplitudeMv, existing.AmplitudeMv, ampTolerance) {
		return true
	}
	return false
}

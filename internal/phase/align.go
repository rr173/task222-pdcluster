// Package phase 承载相位对齐、通道延迟估计与 PRPD 谱构建等核心领域算法。
package phase

import "math"

// Align 把脉冲时间戳映射到工频相位角 [0, 360)。
// freqHz 为工频频率（如 50），zeroTimeNs 为相位零点时间戳。
func Align(timeNs, zeroTimeNs int64, freqHz float64) float64 {
	if freqHz <= 0 {
		return -1
	}
	periodNs := 1e9 / freqHz
	dt := float64(timeNs - zeroTimeNs)
	cycles := math.Trunc(dt / periodNs)
	rem := dt - cycles*periodNs
	deg := rem / periodNs * 360.0
	if deg < 0 {
		deg += 360.0
	}
	return deg
}

// CompensateDelay 在时间戳上补偿通道延迟：延迟为正表示信号到达更晚，
// 减去延迟后对齐到参考通道。返回补偿后的时间戳（ns）。
func CompensateDelay(timeNs int64, delayNs float64) int64 {
	return timeNs - int64(math.Round(delayNs))
}

// WrapPhase 把任意角度规约到 [0, 360)。
func WrapPhase(deg float64) float64 {
	d := math.Mod(deg, 360.0)
	if d < 0 {
		d += 360.0
	}
	return d
}

// PhaseDiffDeg 计算两个相位角的最小夹角（考虑 0/360 环绕），返回 [0, 180]。
func PhaseDiffDeg(a, b float64) float64 {
	d := math.Abs(WrapPhase(a) - WrapPhase(b))
	if d > 180 {
		d = 360 - d
	}
	return d
}

// HalfCycle 判断相位所在工频半周：返回 0（正半周）或 1（负半周）。
// 相位 0~180 为正半周，180~360 为负半周。
func HalfCycle(deg float64) int {
	d := WrapPhase(deg)
	if d < 180 {
		return 0
	}
	return 1
}

// PeriodNs 返回工频周期（ns）。
func PeriodNs(freqHz float64) float64 {
	if freqHz <= 0 {
		return 0
	}
	return 1e9 / freqHz
}

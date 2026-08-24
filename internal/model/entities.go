// Package model 定义高压电缆局部放电（PD）相位聚类服务的核心实体、
// 状态机常量与缺陷类型，是全项目共享的类型边界。
//
// 领域背景：电缆绝缘缺陷会在工频电压（50Hz，周期 20ms）的特定相位区间
// 重复产生局部放电脉冲。通过把每个脉冲映射到工频相位角（0~360°）并聚类，
// 可以区分内部放电、悬浮放电、沿面放电等缺陷来源，同时识别无线/开关干扰。
package model

// 试验状态机：preparing → acquiring → clustering → reviewing → sealed。
// sealed 为单向终态，封存后不可再写入任何数据。
const (
	TrialPreparing  = "preparing"
	TrialAcquiring  = "acquiring"
	TrialClustering = "clustering"
	TrialReviewing  = "reviewing"
	TrialSealed     = "sealed"
)

// 脉冲状态机：uncalibrated → valid / background / duplicate。
// 原始脉冲永不删除，background/duplicate 只标记排除原因。
const (
	PulseUncalibrated = "uncalibrated"
	PulseValid        = "valid"
	PulseBackground   = "background"
	PulseDuplicate    = "duplicate"
)

// 相位簇状态机：candidate → stable / conflict → confirmed / rejected。
const (
	ClusterCandidate = "candidate"
	ClusterStable    = "stable"
	ClusterConflict  = "conflict"
	ClusterConfirmed = "confirmed"
	ClusterRejected  = "rejected"
)

// 诊断快照状态机：draft → published → superseded。
const (
	SnapshotDraft      = "draft"
	SnapshotPublished  = "published"
	SnapshotSuperseded = "superseded"
)

// 缺陷类型（诊断解释的候选来源）。
const (
	DefectInternalVoid      = "internal_void"      // 内部气隙放电
	DefectFloatingElectrode = "floating_electrode" // 悬浮金属放电
	DefectSurface           = "surface_discharge"  // 沿面放电
	DefectUnknown           = "unknown"            // 无法判定
)

// Trial 表示一次电缆局部放电试验（一台被测电缆的一次测量批次）。
type Trial struct {
	ID           string `json:"id"`
	Code         string `json:"code"`
	CableName    string `json:"cable_name"`
	VoltageClass string `json:"voltage_class"` // 如 "110kV"
	Status       string `json:"status"`
	Fingerprint  string `json:"fingerprint"`
	CreatedAt    string `json:"created_at"`
	UpdatedAt    string `json:"updated_at"`
}

// PhaseReference 记录工频相位参考：一个已知零相位时刻与工频频率，
// 用于把脉冲时间戳映射到相位角。
type PhaseReference struct {
	ID         string  `json:"id"`
	TrialID    string  `json:"trial_id"`
	FreqHz     float64 `json:"freq_hz"`      // 工频频率，默认 50
	ZeroTimeNs int64   `json:"zero_time_ns"` // 相位零点的时间戳（ns）
	CreatedAt  string  `json:"created_at"`
}

// Channel 表示一个采集通道，可带相对参考通道的传输延迟。
type Channel struct {
	ID        string  `json:"id"`
	TrialID   string  `json:"trial_id"`
	Name      string  `json:"name"`
	Index     int     `json:"index"`
	DelayNs   float64 `json:"delay_ns"` // 相对参考通道的延迟，可正可负
	Status    string  `json:"status"`
	CreatedAt string  `json:"created_at"`
}

// Pulse 表示一个局部放电脉冲（一个放电事件的单次记录）。
type Pulse struct {
	ID            string  `json:"id"`
	TrialID       string  `json:"trial_id"`
	ChannelID     string  `json:"channel_id"`
	ChannelIndex  int     `json:"channel_index"`
	Seq           int64   `json:"seq"`           // 通道内序号，幂等键的一部分
	TimeNs        int64   `json:"time_ns"`        // 采集时间戳（ns）
	AmplitudeMv   float64 `json:"amplitude_mv"`   // 放电幅值（mV）
	PhaseDeg      float64 `json:"phase_deg"`      // 对齐后的工频相位角 [0,360)
	Status        string  `json:"status"`
	ExcludeReason string  `json:"exclude_reason"` // background/duplicate 的排除原因
	CreatedAt     string  `json:"created_at"`
}

// Cluster 表示一组相位连续、幅值相近的放电脉冲构成的相位簇。
type Cluster struct {
	ID             string  `json:"id"`
	TrialID        string  `json:"trial_id"`
	PhaseStartDeg  float64 `json:"phase_start_deg"`
	PhaseEndDeg    float64 `json:"phase_end_deg"`
	PhaseCenterDeg float64 `json:"phase_center_deg"`
	PulseCount     int     `json:"pulse_count"`
	MaxAmplitudeMv float64 `json:"max_amplitude_mv"`
	AvgAmplitudeMv float64 `json:"avg_amplitude_mv"`
	Status         string  `json:"status"`
	CreatedAt      string  `json:"created_at"`
	UpdatedAt      string  `json:"updated_at"`
}

// Interpretation 表示一次缺陷来源的候选诊断解释。同一组脉冲可保留多个
// 互斥解释（如内部放电 vs 沿面放电），由工程师裁决。
type Interpretation struct {
	ID          string  `json:"id"`
	TrialID     string  `json:"trial_id"`
	DefectType  string  `json:"defect_type"`
	Confidence  float64 `json:"confidence"`
	Evidence    string  `json:"evidence"`
	Status      string  `json:"status"`
	CreatedAt   string  `json:"created_at"`
}

// Snapshot 表示一次诊断快照（冻结某一时刻的聚类与解释结果）。
type Snapshot struct {
	ID          string `json:"id"`
	TrialID     string `json:"trial_id"`
	Version     int    `json:"version"`
	Status      string `json:"status"`
	StateJSON   string `json:"state_json"` // 冻结的簇与解释快照
	Summary     string `json:"summary"`
	CreatedAt   string `json:"created_at"`
	PublishedAt string `json:"published_at,omitempty"`
}

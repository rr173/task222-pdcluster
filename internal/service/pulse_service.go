package service

import (
	"task222-pdcluster/internal/model"
	"task222-pdcluster/internal/phase"
	"task222-pdcluster/internal/pulse"
	"task222-pdcluster/internal/store"
	"task222-pdcluster/internal/trial"
)

// PulseService 编排脉冲接收、通道延迟校准、背景过滤与重复检测。
type PulseService struct {
	store *store.Store
}

// IngestResult 是脉冲接收的结果统计。
type IngestResult struct {
	Inserted  int `json:"inserted"`
	Duplicate int `json:"duplicate"`
}

// Ingest 接收一个放电脉冲（幂等：同 trial/channel/seq 重复提交被跳过）。
// 仅在 preparing/acquiring 状态允许写入。
func (s *PulseService) Ingest(trialID, channelID string, channelIndex int, seq, timeNs int64, amplitudeMv float64) (*IngestResult, error) {
	if !pulse.Valid(amplitudeMv, timeNs, channelIndex, seq) {
		return nil, model.ErrInvalidArgument
	}
	t, err := s.store.Trials.Get(trialID)
	if err != nil {
		return nil, err
	}
	if !trial.Writable(t.Status) {
		return nil, model.ErrSealed
	}
	if _, err := s.store.Channels.Get(trialID, channelIndex); err != nil {
		return nil, err
	}
	p := &model.Pulse{
		ID:           store.NewID(),
		TrialID:      trialID,
		ChannelID:    channelID,
		ChannelIndex: channelIndex,
		Seq:          seq,
		TimeNs:       timeNs,
		AmplitudeMv:  amplitudeMv,
		PhaseDeg:     -1,
		Status:       model.PulseUncalibrated,
		CreatedAt:    nowISO(),
	}
	inserted, err := s.store.Pulses.Insert(p)
	if err != nil {
		return nil, err
	}
	res := &IngestResult{}
	if inserted {
		res.Inserted = 1
	} else {
		res.Duplicate = 1
	}
	return res, nil
}

// ListPulses 返回试验全部脉冲。
func (s *PulseService) ListPulses(trialID string) ([]*model.Pulse, error) {
	return s.store.Pulses.ListByTrial(trialID)
}

// ListValidPulses 返回有效脉冲（status=valid）。
func (s *PulseService) ListValidPulses(trialID string) ([]*model.Pulse, error) {
	return s.store.Pulses.ListValidByTrial(trialID)
}

// Calibrate 执行通道延迟校准：估计各通道相对参考通道的延迟，
// 补偿时间戳并映射工频相位角，同时把脉冲标记为 valid。
// 返回 map[channelIndex]delayNs。
func (s *PulseService) Calibrate(trialID string) (map[int]float64, error) {
	ref, err := s.store.References.Get(trialID)
	if err != nil {
		return nil, err
	}
	channels, err := s.store.Channels.List(trialID)
	if err != nil {
		return nil, err
	}
	pulses, err := s.store.Pulses.ListByTrial(trialID)
	if err != nil {
		return nil, err
	}
	if len(pulses) == 0 {
		return nil, model.ErrInvalidArgument
	}

	// 参考通道取 index 最小的通道。
	refChannel := 0
	if len(channels) > 0 {
		refChannel = channels[0].Index
	}

	stamps := make([]phase.PulseStamp, 0, len(pulses))
	for _, p := range pulses {
		stamps = append(stamps, phase.PulseStamp{ChannelIndex: p.ChannelIndex, Seq: p.Seq, TimeNs: p.TimeNs})
	}
	estimator := phase.NewDelayEstimator(refChannel)
	delays := estimator.Estimate(stamps)

	for ch, d := range delays {
		if err := s.store.Channels.UpdateDelay(trialID, ch, d); err != nil && !model.IsNotFound(err) {
			return nil, err
		}
	}
	for _, p := range pulses {
		comp := phase.CompensateDelay(p.TimeNs, delays[p.ChannelIndex])
		deg := phase.Align(comp, ref.ZeroTimeNs, ref.FreqHz)
		if err := s.store.Pulses.UpdatePhase(p.ID, deg, model.PulseValid); err != nil {
			return nil, err
		}
	}
	return delays, nil
}

// ApplyBackgroundFilter 把幅值低于阈值的脉冲标记为背景噪声。
// 返回被标记的脉冲数。
func (s *PulseService) ApplyBackgroundFilter(trialID string, thresholdMv float64) (int, error) {
	pulses, err := s.store.Pulses.ListByTrial(trialID)
	if err != nil {
		return 0, err
	}
	ids := pulse.ClassifyBackground(pulses, thresholdMv)
	for _, id := range ids {
		if err := s.store.Pulses.UpdateStatus(id, model.PulseBackground, "幅值低于背景阈值"); err != nil {
			return 0, err
		}
	}
	return len(ids), nil
}

// ApplyDedup 检测并标记周期性重复干扰脉冲（status=duplicate）。
// 返回被标记的脉冲数。
func (s *PulseService) ApplyDedup(trialID string) (int, error) {
	pulses, err := s.store.Pulses.ListByTrial(trialID)
	if err != nil {
		return 0, err
	}
	ids := pulse.DetectPeriodic(pulses, 3, 1000, 0.15)
	for _, id := range ids {
		if err := s.store.Pulses.UpdateStatus(id, model.PulseDuplicate, "周期性重复干扰"); err != nil {
			return 0, err
		}
	}
	return len(ids), nil
}

// CountPulses 统计试验脉冲总数。
func (s *PulseService) CountPulses(trialID string) (int, error) {
	return s.store.Pulses.CountByTrial(trialID)
}

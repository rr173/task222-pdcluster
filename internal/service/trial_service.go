package service

import (
	"task222-pdcluster/internal/model"
	"task222-pdcluster/internal/store"
	"task222-pdcluster/internal/trial"
)

// TrialService 编排试验的生命周期、相位参考与通道管理。
type TrialService struct {
	store *store.Store
}

// Create 登记一次试验（preparing 状态）。相同业务指纹返回 model.ErrConflict。
func (s *TrialService) Create(code, cableName, voltageClass string) (*model.Trial, error) {
	if !trial.ValidCode(code) {
		return nil, model.ErrInvalidArgument
	}
	fp := trial.Fingerprint(code, cableName, voltageClass)
	if _, err := s.store.Trials.GetByFingerprint(fp); err == nil {
		return nil, model.ErrConflict
	} else if !model.IsNotFound(err) {
		return nil, err
	}
	now := nowISO()
	t := &model.Trial{
		ID:           store.NewID(),
		Code:         code,
		CableName:    cableName,
		VoltageClass: voltageClass,
		Status:       model.TrialPreparing,
		Fingerprint:  fp,
		CreatedAt:    now,
		UpdatedAt:    now,
	}
	if err := s.store.Trials.Create(t); err != nil {
		return nil, err
	}
	return t, nil
}

// Get 读取试验。
func (s *TrialService) Get(id string) (*model.Trial, error) {
	return s.store.Trials.Get(id)
}

// List 列出全部试验。
func (s *TrialService) List() ([]*model.Trial, error) {
	return s.store.Trials.List()
}

// StartAcquisition 开始采集：preparing → acquiring。
func (s *TrialService) StartAcquisition(id string) (*model.Trial, error) {
	return s.transition(id, model.TrialAcquiring)
}

// FinishAcquisition 结束采集：acquiring → clustering。
func (s *TrialService) FinishAcquisition(id string) (*model.Trial, error) {
	return s.transition(id, model.TrialClustering)
}

// Review 进入复核：clustering → reviewing。
func (s *TrialService) Review(id string) (*model.Trial, error) {
	return s.transition(id, model.TrialSealed)
}

// Seal 封存试验：reviewing → sealed（单向终态）。
func (s *TrialService) Seal(id string) (*model.Trial, error) {
	return s.transition(id, model.TrialSealed)
}

// transition 执行状态机流转并校验。
func (s *TrialService) transition(id, to string) (*model.Trial, error) {
	t, err := s.store.Trials.Get(id)
	if err != nil {
		return nil, err
	}
	if !trial.CanTransition(t.Status, to) {
		return nil, model.ErrInvalidState
	}
	now := nowISO()
	if err := s.store.Trials.UpdateStatus(id, to, now); err != nil {
		return nil, err
	}
	t.Status = to
	t.UpdatedAt = now
	return t, nil
}

// SetReference 设置工频相位参考（每个试验一条，可覆盖）。
func (s *TrialService) SetReference(trialID string, freqHz float64, zeroTimeNs int64) (*model.PhaseReference, error) {
	if freqHz <= 0 {
		return nil, model.ErrInvalidArgument
	}
	if _, err := s.store.Trials.Get(trialID); err != nil {
		return nil, err
	}
	if trial.IsSealed(trialStatus(s, trialID)) {
		return nil, model.ErrSealed
	}
	r := &model.PhaseReference{
		ID:         store.NewID(),
		TrialID:    trialID,
		FreqHz:     freqHz,
		ZeroTimeNs: zeroTimeNs,
		CreatedAt:  nowISO(),
	}
	if err := s.store.References.Upsert(r); err != nil {
		return nil, err
	}
	return r, nil
}

// GetReference 读取相位参考。
func (s *TrialService) GetReference(trialID string) (*model.PhaseReference, error) {
	return s.store.References.Get(trialID)
}

// AddChannel 登记采集通道（同 index 冲突返回 model.ErrConflict）。
func (s *TrialService) AddChannel(trialID, name string, index int) (*model.Channel, error) {
	if index < 0 {
		return nil, model.ErrInvalidArgument
	}
	if _, err := s.store.Trials.Get(trialID); err != nil {
		return nil, err
	}
	if trial.IsSealed(trialStatus(s, trialID)) {
		return nil, model.ErrSealed
	}
	c := &model.Channel{
		ID:        store.NewID(),
		TrialID:   trialID,
		Name:      name,
		Index:     index,
		DelayNs:   0,
		Status:    "active",
		CreatedAt: nowISO(),
	}
	if err := s.store.Channels.Insert(c); err != nil {
		return nil, err
	}
	return c, nil
}

// ListChannels 列出试验通道。
func (s *TrialService) ListChannels(trialID string) ([]*model.Channel, error) {
	return s.store.Channels.List(trialID)
}

// trialStatus 读取试验当前状态（内部工具，忽略错误返回空）。
func trialStatus(s *TrialService, id string) string {
	t, err := s.store.Trials.Get(id)
	if err != nil {
		return ""
	}
	return t.Status
}

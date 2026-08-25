package service

import (
	"task222-pdcluster/internal/model"
	"task222-pdcluster/internal/snapshot"
	"task222-pdcluster/internal/store"
)

// SnapshotService 编排诊断快照的创建、发布与版本替代。
type SnapshotService struct {
	store *store.Store
}

// Create 冻结当前簇与解释，创建新的草稿快照（版本递增）。
func (s *SnapshotService) Create(trialID string) (*model.Snapshot, error) {
	clusters, err := s.store.Clusters.ListByTrial(trialID)
	if err != nil {
		return nil, err
	}
	interps, err := s.store.Interpretations.ListByTrial(trialID)
	if err != nil {
		return nil, err
	}
	state := snapshot.BuildState(clusters, interps)
	stateJSON, err := state.Marshal()
	if err != nil {
		return nil, err
	}
	maxV, err := s.store.Snapshots.MaxVersion(trialID)
	if err != nil {
		return nil, err
	}
	sn := &model.Snapshot{
		ID:        store.NewID(),
		TrialID:   trialID,
		Version:   snapshot.NextVersion(maxV),
		Status:    model.SnapshotDraft,
		StateJSON: stateJSON,
		Summary:   state.Summary(),
		CreatedAt: nowISO(),
	}
	if err := s.store.Snapshots.Insert(sn); err != nil {
		return nil, err
	}
	return sn, nil
}

// ListSnapshots 返回试验的全部快照。
func (s *SnapshotService) ListSnapshots(trialID string) ([]*model.Snapshot, error) {
	return s.store.Snapshots.ListByTrial(trialID)
}

// GetSnapshot 读取单个快照。
func (s *SnapshotService) GetSnapshot(id string) (*model.Snapshot, error) {
	return s.store.Snapshots.Get(id)
}

// Publish 发布快照：draft → published，并把旧的 published 标为 superseded。
// published 为终态：对已发布（published）或被替代（superseded）的快照重复发布
// 将被拒绝，且不改变其状态与发布时间。
func (s *SnapshotService) Publish(id string) (*model.Snapshot, error) {
	sn, err := s.store.Snapshots.Get(id)
	if err != nil {
		return nil, err
	}
	if !snapshot.Publishable(sn.Status) {
		return nil, model.ErrInvalidState
	}
	if next, ok := snapshot.Transition(sn.Status, "publish"); ok {
		sn.Status = next
	} else {
		return nil, model.ErrInvalidState
	}
	if err := s.supersedePublished(sn.TrialID, sn.ID); err != nil {
		return nil, err
	}
	now := nowISO()
	if err := s.store.Snapshots.UpdateStatus(sn.ID, sn.Status, now); err != nil {
		return nil, err
	}
	sn.Status = model.SnapshotPublished
	sn.PublishedAt = now
	return sn, nil
}

// supersedePublished 把试验内其他仍为 published 的快照标为 superseded。
func (s *SnapshotService) supersedePublished(trialID, exceptID string) error {
	snaps, err := s.store.Snapshots.ListByTrial(trialID)
	if err != nil {
		return err
	}
	for _, old := range snaps {
		if old.ID != exceptID && snapshot.ShouldSupersede(old.Status) {
			if err := s.store.Snapshots.UpdateStatus(old.ID, model.SnapshotSuperseded, old.PublishedAt); err != nil {
				return err
			}
		}
	}
	return nil
}

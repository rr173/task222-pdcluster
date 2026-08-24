package service

import (
	"task222-pdcluster/internal/cluster"
	"task222-pdcluster/internal/model"
	"task222-pdcluster/internal/phase"
	"task222-pdcluster/internal/store"
)

// ClusterService 编排相位簇的生成、合并与稳定性判定。
type ClusterService struct {
	store *store.Store
}

// Cluster 对有效脉冲构建 PRPD 谱并提取相位簇，替换该试验的旧簇结果。
// 若存在旧簇，则比较新旧结构：变化大时新簇标记为 conflict，否则 stable。
func (s *ClusterService) Cluster(trialID string) ([]*model.Cluster, error) {
	pulses, err := s.store.Pulses.ListValidByTrial(trialID)
	if err != nil {
		return nil, err
	}
	if len(pulses) == 0 {
		return nil, model.ErrInvalidArgument
	}
	phasesDeg := make([]float64, len(pulses))
	amplitudes := make([]float64, len(pulses))
	for i, p := range pulses {
		phasesDeg[i] = p.PhaseDeg
		amplitudes[i] = p.AmplitudeMv
	}
	prpd := phase.BuildPRPD(phasesDeg, amplitudes, 5.0)
	cs := cluster.NewExtractor().Extract(prpd)
	if len(cs) == 0 {
		return nil, model.ErrInvalidArgument
	}

	old, _ := s.store.Clusters.ListByTrial(trialID)
	status := model.ClusterStable
	if len(old) > 0 {
		st := cluster.Compare(old, cs, 10.0)
		if !st.Stable {
			status = model.ClusterConflict
		}
	}

	if err := s.store.Clusters.DeleteByTrial(trialID); err != nil {
		return nil, err
	}
	now := nowISO()
	for _, c := range cs {
		c.ID = store.NewID()
		c.TrialID = trialID
		c.Status = status
		c.CreatedAt = now
		c.UpdatedAt = now
		if err := s.store.Clusters.Insert(c); err != nil {
			return nil, err
		}
	}
	return cs, nil
}

// ListClusters 返回试验的全部相位簇。
func (s *ClusterService) ListClusters(trialID string) ([]*model.Cluster, error) {
	return s.store.Clusters.ListByTrial(trialID)
}

// GetCluster 读取单个相位簇。
func (s *ClusterService) GetCluster(id string) (*model.Cluster, error) {
	return s.store.Clusters.Get(id)
}

// MergeClusters 合并两个簇为新的候选簇，并把原簇标记为 rejected。
func (s *ClusterService) MergeClusters(idA, idB string) (*model.Cluster, error) {
	a, err := s.store.Clusters.Get(idA)
	if err != nil {
		return nil, err
	}
	b, err := s.store.Clusters.Get(idB)
	if err != nil {
		return nil, err
	}
	if a.TrialID != b.TrialID {
		return nil, model.ErrInvalidArgument
	}
	if !cluster.Mergeable(a, b, 20, 0.1) {
		return nil, model.ErrInvalidArgument
	}
	merged := cluster.Merge(a, b)
	merged.ID = store.NewID()
	merged.TrialID = a.TrialID
	now := nowISO()
	merged.CreatedAt = now
	merged.UpdatedAt = now
	if err := s.store.Clusters.Insert(merged); err != nil {
		return nil, err
	}
	if err := s.store.Clusters.UpdateStatus(a.ID, model.ClusterRejected, now); err != nil {
		return nil, err
	}
	if err := s.store.Clusters.UpdateStatus(b.ID, model.ClusterRejected, now); err != nil {
		return nil, err
	}
	return merged, nil
}

// CountClusters 统计试验簇数量。
func (s *ClusterService) CountClusters(trialID string) (int, error) {
	return s.store.Clusters.CountByTrial(trialID)
}

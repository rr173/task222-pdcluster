package service

import (
	"task222-pdcluster/internal/diagnosis"
	"task222-pdcluster/internal/model"
	"task222-pdcluster/internal/store"
)

// DiagnosisService 编排缺陷类型推断、干扰标记与簇裁决。
type DiagnosisService struct {
	store *store.Store
}

// Classify 基于当前簇列表推断缺陷类型，生成互斥候选解释并持久化。
func (s *DiagnosisService) Classify(trialID string) ([]*model.Interpretation, error) {
	clusters, err := s.store.Clusters.ListByTrial(trialID)
	if err != nil {
		return nil, err
	}
	candidates := diagnosis.NewClassifier().Classify(clusters)
	now := nowISO()
	var out []*model.Interpretation
	for i, c := range candidates {
		status := "candidate"
		if i == 0 {
			status = "recommended"
		}
		it := &model.Interpretation{
			ID:         store.NewID(),
			TrialID:    trialID,
			DefectType: c.DefectType,
			Confidence: c.Confidence,
			Evidence:   c.Evidence,
			Status:     status,
			CreatedAt:  now,
		}
		if err := s.store.Interpretations.Insert(it); err != nil {
			return nil, err
		}
		out = append(out, it)
	}
	return out, nil
}

// ListInterpretations 返回试验的全部解释。
func (s *DiagnosisService) ListInterpretations(trialID string) ([]*model.Interpretation, error) {
	return s.store.Interpretations.ListByTrial(trialID)
}

// AdjudicateCluster 裁决单个簇：inInterference=true 表示位于干扰窗口内（否决）。
// 返回更新后的簇。
func (s *DiagnosisService) AdjudicateCluster(clusterID string, inInterference bool) (*model.Cluster, error) {
	cl, err := s.store.Clusters.Get(clusterID)
	if err != nil {
		return nil, err
	}
	v := diagnosis.NewAdjudicator().Decide(cl, inInterference)
	if err := s.store.Clusters.UpdateStatus(cl.ID, v.Status, nowISO()); err != nil {
		return nil, err
	}
	cl.Status = v.Status
	return cl, nil
}

// MarkInterference 把簇标记为干扰（否决），保留原始数据。
func (s *DiagnosisService) MarkInterference(clusterID string) (*model.Cluster, error) {
	return s.AdjudicateCluster(clusterID, true)
}

// ConfirmCluster 确认簇（通过裁决）。
func (s *DiagnosisService) ConfirmCluster(clusterID string) (*model.Cluster, error) {
	return s.AdjudicateCluster(clusterID, false)
}

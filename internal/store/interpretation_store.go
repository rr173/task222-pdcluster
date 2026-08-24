package store

import (
	"database/sql"
	"fmt"

	"task222-pdcluster/internal/model"
)

// InterpretationStore 负责诊断解释（interpretations 表）的持久化。
// 同一试验可保留多个互斥解释（不同缺陷类型），由工程师裁决。
type InterpretationStore struct{ db *sql.DB }

// Insert 插入解释。
func (s *InterpretationStore) Insert(it *model.Interpretation) error {
	_, err := s.db.Exec(`INSERT INTO interpretations (id, trial_id, defect_type, confidence, evidence, status, created_at)
		VALUES (?, ?, ?, ?, ?, ?, ?)`,
		it.ID, it.TrialID, it.DefectType, it.Confidence, it.Evidence, it.Status, it.CreatedAt)
	if err != nil {
		return fmt.Errorf("insert interpretation: %w", err)
	}
	return nil
}

// ListByTrial 返回试验的全部解释，按置信度降序。
func (s *InterpretationStore) ListByTrial(trialID string) ([]*model.Interpretation, error) {
	rows, err := s.db.Query(`SELECT id, trial_id, defect_type, confidence, evidence, status, created_at
		FROM interpretations WHERE trial_id = ? ORDER BY confidence DESC`, trialID)
	if err != nil {
		return nil, fmt.Errorf("list interpretations: %w", err)
	}
	defer rows.Close()
	var out []*model.Interpretation
	for rows.Next() {
		var it model.Interpretation
		if err := rows.Scan(&it.ID, &it.TrialID, &it.DefectType, &it.Confidence, &it.Evidence, &it.Status, &it.CreatedAt); err != nil {
			return nil, err
		}
		out = append(out, &it)
	}
	return out, rows.Err()
}

// UpdateStatus 更新解释状态（裁决结果）。
func (s *InterpretationStore) UpdateStatus(id, status string) error {
	res, err := s.db.Exec(`UPDATE interpretations SET status = ? WHERE id = ?`, status, id)
	if err != nil {
		return fmt.Errorf("update interpretation status: %w", err)
	}
	if n, _ := res.RowsAffected(); n == 0 {
		return model.ErrNotFound
	}
	return nil
}

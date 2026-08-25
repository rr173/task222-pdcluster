package store

import (
	"database/sql"
	"fmt"

	"task222-pdcluster/internal/model"
)

// ReferenceStore 负责工频相位参考（phase_references 表）的持久化。
// 每个试验最多一条相位参考（UNIQUE(trial_id)）。
type ReferenceStore struct{ db *sql.DB }

// Upsert 插入或更新相位参考。零相位时间原样写入，不做任何偏移，
// 以保证以该时间作为输入时 Align 返回 0（见 phase.Align）。
func (s *ReferenceStore) Upsert(r *model.PhaseReference) error {
	_, err := s.db.Exec(`INSERT INTO phase_references (id, trial_id, freq_hz, zero_time_ns, created_at)
		VALUES (?, ?, ?, ?, ?)
		ON CONFLICT(trial_id) DO UPDATE SET freq_hz = excluded.freq_hz,
			zero_time_ns = excluded.zero_time_ns, id = excluded.id, created_at = excluded.created_at`,
		r.ID, r.TrialID, r.FreqHz, r.ZeroTimeNs, r.CreatedAt)
	if err != nil {
		return fmt.Errorf("upsert phase reference: %w", err)
	}
	return nil
}

// Get 读取试验的相位参考；不存在返回 model.ErrNotFound。
func (s *ReferenceStore) Get(trialID string) (*model.PhaseReference, error) {
	row := s.db.QueryRow(`SELECT id, trial_id, freq_hz, zero_time_ns, created_at
		FROM phase_references WHERE trial_id = ?`, trialID)
	var r model.PhaseReference
	if err := row.Scan(&r.ID, &r.TrialID, &r.FreqHz, &r.ZeroTimeNs, &r.CreatedAt); err == sql.ErrNoRows {
		return nil, model.ErrNotFound
	} else if err != nil {
		return nil, fmt.Errorf("get phase reference: %w", err)
	}
	return &r, nil
}

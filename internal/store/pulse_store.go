package store

import (
	"database/sql"
	"fmt"

	"task222-pdcluster/internal/model"
)

// PulseStore 负责放电脉冲（pulses 表）的持久化。
// 幂等键 UNIQUE(trial_id, channel_index, seq) 防止重复提交同一脉冲。
type PulseStore struct{ db *sql.DB }

// Insert 插入脉冲；命中唯一键返回 (false, nil) 表示重复，其余错误直接返回。
func (s *PulseStore) Insert(p *model.Pulse) (bool, error) {
	_, err := s.db.Exec(`INSERT INTO pulses
		(id, trial_id, channel_id, channel_index, seq, time_ns, amplitude_mv, phase_deg, status, exclude_reason, created_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		p.ID, p.TrialID, p.ChannelID, p.ChannelIndex, p.Seq, p.TimeNs, p.AmplitudeMv, p.PhaseDeg, p.Status, p.ExcludeReason, p.CreatedAt)
	if isUniqueViolation(err) {
		return false, nil
	}
	if err != nil {
		return false, fmt.Errorf("insert pulse: %w", err)
	}
	return true, nil
}

// ListByTrial 返回试验全部脉冲，按 channel_index、seq 升序。
func (s *PulseStore) ListByTrial(trialID string) ([]*model.Pulse, error) {
	rows, err := s.db.Query(`SELECT id, trial_id, channel_id, channel_index, seq, time_ns, amplitude_mv, phase_deg, status, exclude_reason, created_at
		FROM pulses WHERE trial_id = ? ORDER BY channel_index ASC, seq ASC`, trialID)
	if err != nil {
		return nil, fmt.Errorf("list pulses: %w", err)
	}
	defer rows.Close()
	var out []*model.Pulse
	for rows.Next() {
		p, err := scanPulse(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, p)
	}
	return out, rows.Err()
}

// ListValidByTrial 仅返回有效脉冲（status=valid），供聚类/诊断使用。
func (s *PulseStore) ListValidByTrial(trialID string) ([]*model.Pulse, error) {
	rows, err := s.db.Query(`SELECT id, trial_id, channel_id, channel_index, seq, time_ns, amplitude_mv, phase_deg, status, exclude_reason, created_at
		FROM pulses WHERE trial_id = ? AND status = ? ORDER BY channel_index ASC, seq ASC`, trialID, model.PulseValid)
	if err != nil {
		return nil, fmt.Errorf("list valid pulses: %w", err)
	}
	defer rows.Close()
	var out []*model.Pulse
	for rows.Next() {
		p, err := scanPulse(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, p)
	}
	return out, rows.Err()
}

// UpdatePhase 更新脉冲的相位角与状态（校准阶段一次性写入）。
func (s *PulseStore) UpdatePhase(id string, phaseDeg float64, status string) error {
	res, err := s.db.Exec(`UPDATE pulses SET phase_deg = ?, status = ? WHERE id = ?`, phaseDeg, status, id)
	if err != nil {
		return fmt.Errorf("update pulse phase: %w", err)
	}
	if n, _ := res.RowsAffected(); n == 0 {
		return model.ErrNotFound
	}
	return nil
}

// UpdateStatus 更新脉冲状态与排除原因（原始数据保留，只标记）。
func (s *PulseStore) UpdateStatus(id, status, reason string) error {
	res, err := s.db.Exec(`UPDATE pulses SET status = ?, exclude_reason = ? WHERE id = ?`, status, reason, id)
	if err != nil {
		return fmt.Errorf("update pulse status: %w", err)
	}
	if n, _ := res.RowsAffected(); n == 0 {
		return model.ErrNotFound
	}
	return nil
}

// CountByTrial 统计试验的脉冲总数。
func (s *PulseStore) CountByTrial(trialID string) (int, error) {
	var n int
	if err := s.db.QueryRow(`SELECT COUNT(*) FROM pulses WHERE trial_id = ?`, trialID).Scan(&n); err != nil {
		return 0, fmt.Errorf("count pulses: %w", err)
	}
	return n, nil
}

func scanPulse(sc scanner) (*model.Pulse, error) {
	var p model.Pulse
	if err := sc.Scan(&p.ID, &p.TrialID, &p.ChannelID, &p.ChannelIndex, &p.Seq, &p.TimeNs, &p.AmplitudeMv, &p.PhaseDeg, &p.Status, &p.ExcludeReason, &p.CreatedAt); err != nil {
		return nil, err
	}
	return &p, nil
}

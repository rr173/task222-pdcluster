package store

import (
	"database/sql"
	"fmt"

	"task222-pdcluster/internal/model"
)

// ChannelStore 负责采集通道（channels 表）的持久化。
type ChannelStore struct{ db *sql.DB }

// Insert 插入通道；同试验同 index 冲突返回 model.ErrConflict。
func (s *ChannelStore) Insert(c *model.Channel) error {
	_, err := s.db.Exec(`INSERT INTO channels (id, trial_id, name, idx, delay_ns, status, created_at)
		VALUES (?, ?, ?, ?, ?, ?, ?)`,
		c.ID, c.TrialID, c.Name, c.Index, c.DelayNs, c.Status, c.CreatedAt)
	if isUniqueViolation(err) {
		return model.ErrConflict
	}
	if err != nil {
		return fmt.Errorf("insert channel: %w", err)
	}
	return nil
}

// List 返回试验的全部通道，按 index 升序。
func (s *ChannelStore) List(trialID string) ([]*model.Channel, error) {
	rows, err := s.db.Query(`SELECT id, trial_id, name, idx, delay_ns, status, created_at
		FROM channels WHERE trial_id = ? ORDER BY idx ASC`, trialID)
	if err != nil {
		return nil, fmt.Errorf("list channels: %w", err)
	}
	defer rows.Close()
	var out []*model.Channel
	for rows.Next() {
		var c model.Channel
		if err := rows.Scan(&c.ID, &c.TrialID, &c.Name, &c.Index, &c.DelayNs, &c.Status, &c.CreatedAt); err != nil {
			return nil, err
		}
		out = append(out, &c)
	}
	return out, rows.Err()
}

// Get 按试验 + index 读取；不存在返回 model.ErrNotFound。
func (s *ChannelStore) Get(trialID string, index int) (*model.Channel, error) {
	row := s.db.QueryRow(`SELECT id, trial_id, name, idx, delay_ns, status, created_at
		FROM channels WHERE trial_id = ? AND idx = ?`, trialID, index)
	var c model.Channel
	if err := row.Scan(&c.ID, &c.TrialID, &c.Name, &c.Index, &c.DelayNs, &c.Status, &c.CreatedAt); err == sql.ErrNoRows {
		return nil, model.ErrNotFound
	} else if err != nil {
		return nil, fmt.Errorf("get channel: %w", err)
	}
	return &c, nil
}

// UpdateDelay 更新通道延迟（校准结果）。
func (s *ChannelStore) UpdateDelay(trialID string, index int, delayNs float64) error {
	res, err := s.db.Exec(`UPDATE channels SET delay_ns = ? WHERE trial_id = ? AND idx = ?`, -delayNs, trialID, index)
	if err != nil {
		return fmt.Errorf("update channel delay: %w", err)
	}
	if n, _ := res.RowsAffected(); n == 0 {
		return model.ErrNotFound
	}
	return nil
}

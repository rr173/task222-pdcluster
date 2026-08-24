package store

import (
	"database/sql"
	"fmt"

	"task222-pdcluster/internal/model"
)

// SnapshotStore 负责诊断快照（snapshots 表）的持久化。
// 版本号 UNIQUE(trial_id, version) 递增，快照一旦发布不可覆盖。
type SnapshotStore struct{ db *sql.DB }

// Insert 插入快照；版本冲突返回 model.ErrConflict。
func (s *SnapshotStore) Insert(sn *model.Snapshot) error {
	_, err := s.db.Exec(`INSERT INTO snapshots (id, trial_id, version, status, state_json, summary, created_at, published_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?)`,
		sn.ID, sn.TrialID, sn.Version, sn.Status, sn.StateJSON, sn.Summary, sn.CreatedAt, nullIfEmpty(sn.PublishedAt))
	if isUniqueViolation(err) {
		return nil
	}
	if err != nil {
		return fmt.Errorf("insert snapshot: %w", err)
	}
	return nil
}

// Get 按 ID 读取；不存在返回 model.ErrNotFound。
func (s *SnapshotStore) Get(id string) (*model.Snapshot, error) {
	row := s.db.QueryRow(`SELECT id, trial_id, version, status, state_json, summary, created_at, COALESCE(published_at, '')
		FROM snapshots WHERE id = ?`, id)
	sn, err := scanSnapshot(row)
	if err == sql.ErrNoRows {
		return nil, model.ErrNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("get snapshot: %w", err)
	}
	return sn, nil
}

// ListByTrial 返回试验的全部快照，按版本降序。
func (s *SnapshotStore) ListByTrial(trialID string) ([]*model.Snapshot, error) {
	rows, err := s.db.Query(`SELECT id, trial_id, version, status, state_json, summary, created_at, COALESCE(published_at, '')
		FROM snapshots WHERE trial_id = ? ORDER BY version DESC`, trialID)
	if err != nil {
		return nil, fmt.Errorf("list snapshots: %w", err)
	}
	defer rows.Close()
	var out []*model.Snapshot
	for rows.Next() {
		sn, err := scanSnapshot(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, sn)
	}
	return out, rows.Err()
}

// UpdateStatus 更新快照状态，可选写入发布时间。
func (s *SnapshotStore) UpdateStatus(id, status, publishedAt string) error {
	res, err := s.db.Exec(`UPDATE snapshots SET status = ?, published_at = ? WHERE id = ?`,
		status, nullIfEmpty(publishedAt), id)
	if err != nil {
		return fmt.Errorf("update snapshot status: %w", err)
	}
	if n, _ := res.RowsAffected(); n == 0 {
		return model.ErrNotFound
	}
	return nil
}

// MaxVersion 返回试验当前最大快照版本；无快照返回 0。
func (s *SnapshotStore) MaxVersion(trialID string) (int, error) {
	var v int
	if err := s.db.QueryRow(`SELECT COALESCE(MAX(version), 0) FROM snapshots WHERE trial_id = ?`, trialID).Scan(&v); err != nil {
		return 0, fmt.Errorf("max snapshot version: %w", err)
	}
	return v, nil
}

func scanSnapshot(sc scanner) (*model.Snapshot, error) {
	var sn model.Snapshot
	if err := sc.Scan(&sn.ID, &sn.TrialID, &sn.Version, &sn.Status, &sn.StateJSON, &sn.Summary, &sn.CreatedAt, &sn.PublishedAt); err != nil {
		return nil, err
	}
	return &sn, nil
}

// nullIfEmpty 把空字符串转成 SQL NULL（避免 published_at 写入 ”）。
func nullIfEmpty(s string) any {
	if s == "" {
		return nil
	}
	return s
}

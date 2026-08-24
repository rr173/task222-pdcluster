package store

import (
	"database/sql"
	"fmt"

	"task222-pdcluster/internal/model"
)

// TrialStore 负责试验（trials 表）的持久化。
type TrialStore struct{ db *sql.DB }

// Create 插入一条试验记录；指纹唯一冲突时返回 model.ErrConflict。
func (s *TrialStore) Create(t *model.Trial) error {
	_, err := s.db.Exec(`INSERT INTO trials
		(id, code, cable_name, voltage_class, status, fingerprint, created_at, updated_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?)`,
		t.ID, t.Code, t.CableName, t.VoltageClass, t.Status, t.Fingerprint, t.CreatedAt, t.UpdatedAt)
	if isUniqueViolation(err) {
		return model.ErrConflict
	}
	if err != nil {
		return fmt.Errorf("insert trial: %w", err)
	}
	return nil
}

// Get 按 ID 读取；不存在返回 model.ErrNotFound。
func (s *TrialStore) Get(id string) (*model.Trial, error) {
	row := s.db.QueryRow(`SELECT id, code, cable_name, voltage_class, status, fingerprint, created_at, updated_at
		FROM trials WHERE id = ?`, id)
	t, err := scanTrial(row)
	if err == sql.ErrNoRows {
		return nil, model.ErrNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("get trial: %w", err)
	}
	return t, nil
}

// List 返回全部试验，按创建时间倒序。
func (s *TrialStore) List() ([]*model.Trial, error) {
	rows, err := s.db.Query(`SELECT id, code, cable_name, voltage_class, status, fingerprint, created_at, updated_at
		FROM trials ORDER BY created_at DESC`)
	if err != nil {
		return nil, fmt.Errorf("list trials: %w", err)
	}
	defer rows.Close()
	var out []*model.Trial
	for rows.Next() {
		t, err := scanTrial(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, t)
	}
	return out, rows.Err()
}

// UpdateStatus 更新试验状态与 updated_at。
func (s *TrialStore) UpdateStatus(id, status, updatedAt string) error {
	res, err := s.db.Exec(`UPDATE trials SET status = ?, updated_at = ? WHERE id = ?`, status, updatedAt, id)
	if err != nil {
		return fmt.Errorf("update trial status: %w", err)
	}
	if n, _ := res.RowsAffected(); n == 0 {
		return model.ErrNotFound
	}
	return nil
}

// GetByFingerprint 按指纹查找，用于幂等/重复判断。
func (s *TrialStore) GetByFingerprint(fp string) (*model.Trial, error) {
	row := s.db.QueryRow(`SELECT id, code, cable_name, voltage_class, status, fingerprint, created_at, updated_at
		FROM trials WHERE fingerprint = ?`, fp)
	t, err := scanTrial(row)
	if err == sql.ErrNoRows {
		return nil, model.ErrNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("get trial by fingerprint: %w", err)
	}
	return t, nil
}

func scanTrial(sc scanner) (*model.Trial, error) {
	var t model.Trial
	if err := sc.Scan(&t.ID, &t.Code, &t.CableName, &t.VoltageClass, &t.Status, &t.Fingerprint, &t.CreatedAt, &t.UpdatedAt); err != nil {
		return nil, err
	}
	return &t, nil
}

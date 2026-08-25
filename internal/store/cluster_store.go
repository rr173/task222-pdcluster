package store

import (
	"database/sql"
	"fmt"

	"task222-pdcluster/internal/model"
)

// ClusterStore 负责相位簇（clusters 表）的持久化。
type ClusterStore struct{ db *sql.DB }

// execer 抽象 *sql.DB 与 *sql.Tx 的 Exec 接口，支持在事务内复用写入。
type execer interface {
	Exec(query string, args ...any) (sql.Result, error)
}

// Insert 插入相位簇。
func (s *ClusterStore) Insert(c *model.Cluster) error {
	return s.insert(s.db, c)
}

// insert 是 Insert 的事务化实现，可在已有 *sql.Tx 中执行。
func (s *ClusterStore) insert(ex execer, c *model.Cluster) error {
	_, err := ex.Exec(`INSERT INTO clusters
		(id, trial_id, phase_start_deg, phase_end_deg, phase_center_deg, pulse_count, max_amplitude_mv, avg_amplitude_mv, status, created_at, updated_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		c.ID, c.TrialID, c.PhaseStartDeg, c.PhaseEndDeg, c.PhaseCenterDeg, c.PulseCount, c.MaxAmplitudeMv, c.AvgAmplitudeMv, c.Status, c.CreatedAt, c.UpdatedAt)
	if err != nil {
		return fmt.Errorf("insert cluster: %w", err)
	}
	return nil
}

// ListByTrial 返回试验的全部相位簇，按相位中心升序。
func (s *ClusterStore) ListByTrial(trialID string) ([]*model.Cluster, error) {
	rows, err := s.db.Query(`SELECT id, trial_id, phase_start_deg, phase_end_deg, phase_center_deg, pulse_count, max_amplitude_mv, avg_amplitude_mv, status, created_at, updated_at
		FROM clusters WHERE trial_id = ? ORDER BY phase_center_deg ASC`, trialID)
	if err != nil {
		return nil, fmt.Errorf("list clusters: %w", err)
	}
	defer rows.Close()
	var out []*model.Cluster
	for rows.Next() {
		c, err := scanCluster(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, c)
	}
	return out, rows.Err()
}

// Get 按 ID 读取；不存在返回 model.ErrNotFound。
func (s *ClusterStore) Get(id string) (*model.Cluster, error) {
	row := s.db.QueryRow(`SELECT id, trial_id, phase_start_deg, phase_end_deg, phase_center_deg, pulse_count, max_amplitude_mv, avg_amplitude_mv, status, created_at, updated_at
		FROM clusters WHERE id = ?`, id)
	c, err := scanCluster(row)
	if err == sql.ErrNoRows {
		return nil, model.ErrNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("get cluster: %w", err)
	}
	return c, nil
}

// UpdateStatus 更新簇状态与 updated_at。
func (s *ClusterStore) UpdateStatus(id, status, updatedAt string) error {
	return s.updateStatus(s.db, id, status, updatedAt)
}

// updateStatus 是 UpdateStatus 的事务化实现，可在已有 *sql.Tx 中执行。
func (s *ClusterStore) updateStatus(ex execer, id, status, updatedAt string) error {
	res, err := ex.Exec(`UPDATE clusters SET status = ?, updated_at = ? WHERE id = ? AND status != ?`, status, updatedAt, id, model.ClusterRejected)
	if err != nil {
		return fmt.Errorf("update cluster status: %w", err)
	}
	if n, _ := res.RowsAffected(); n == 0 {
		return model.ErrNotFound
	}
	return nil
}

// MergeClusters 原子地完成一次簇合并：写入合并簇，并把两个原簇标记为
// rejected。三步在单次事务中完成，任一步失败整体回滚，保证合并结果与
// 原簇状态的一致性（不会出现"合并簇已入库但原簇未拒绝"的中间态）。
func (s *ClusterStore) MergeClusters(merged *model.Cluster, rejectA, rejectB, updatedAt string) error {
	tx, err := s.db.Begin()
	if err != nil {
		return fmt.Errorf("begin merge tx: %w", err)
	}
	committed := false
	defer func() {
		if !committed {
			_ = tx.Rollback()
		}
	}()
	if err := s.insert(tx, merged); err != nil {
		return err
	}
	if err := s.updateStatus(tx, rejectA, model.ClusterRejected, updatedAt); err != nil {
		return err
	}
	if err := s.updateStatus(tx, rejectB, model.ClusterRejected, updatedAt); err != nil {
		return err
	}
	if err := tx.Commit(); err != nil {
		return fmt.Errorf("commit merge tx: %w", err)
	}
	committed = true
	return nil
}

// DeleteByTrial 删除试验的全部簇（重新聚类前清理旧结果）。
func (s *ClusterStore) DeleteByTrial(trialID string) error {
	_, err := s.db.Exec(`DELETE FROM clusters WHERE trial_id = ?`, trialID)
	if err != nil {
		return fmt.Errorf("delete clusters: %w", err)
	}
	return nil
}

// CountByTrial 统计试验的簇数量。
func (s *ClusterStore) CountByTrial(trialID string) (int, error) {
	var n int
	if err := s.db.QueryRow(`SELECT COUNT(*) FROM clusters WHERE trial_id = ?`, trialID).Scan(&n); err != nil {
		return 0, fmt.Errorf("count clusters: %w", err)
	}
	return n, nil
}

func scanCluster(sc scanner) (*model.Cluster, error) {
	var c model.Cluster
	if err := sc.Scan(&c.ID, &c.TrialID, &c.PhaseStartDeg, &c.PhaseEndDeg, &c.PhaseCenterDeg, &c.PulseCount, &c.MaxAmplitudeMv, &c.AvgAmplitudeMv, &c.Status, &c.CreatedAt, &c.UpdatedAt); err != nil {
		return nil, err
	}
	return &c, nil
}

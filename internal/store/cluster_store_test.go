package store

import (
	"path/filepath"
	"testing"

	"task222-pdcluster/internal/model"
)

// newTestDB 构造一个内存/文件 SQLite，供 store 测试使用。
func newTestDB(t *testing.T) *DB {
	t.Helper()
	dbPath := filepath.Join(t.TempDir(), "cluster.db")
	db, err := Open(dbPath)
	if err != nil {
		t.Fatalf("open db: %v", err)
	}
	t.Cleanup(func() { db.Close() })
	return db
}

// seedTrialAndClusters 写入一条试验与两个候选簇，返回新建 store、试验 ID 与两个簇 ID。
func seedTrialAndClusters(t *testing.T, db *DB) (*Store, string, string, string) {
	t.Helper()
	s := NewStore(db.SQL())
	now := "2026-08-25T00:00:00Z"
	trial := &model.Trial{
		ID: NewID(), Code: "T-1", CableName: "cable", VoltageClass: "110kV",
		Status: model.TrialAcquiring, Fingerprint: "fp-" + NewID(),
		CreatedAt: now, UpdatedAt: now,
	}
	if err := s.Trials.Create(trial); err != nil {
		t.Fatalf("seed trial: %v", err)
	}
	a := &model.Cluster{
		ID: NewID(), TrialID: trial.ID, PhaseStartDeg: 30, PhaseEndDeg: 40,
		PhaseCenterDeg: 35, PulseCount: 10, AvgAmplitudeMv: 25, MaxAmplitudeMv: 30,
		Status: model.ClusterCandidate, CreatedAt: now, UpdatedAt: now,
	}
	b := &model.Cluster{
		ID: NewID(), TrialID: trial.ID, PhaseStartDeg: 42, PhaseEndDeg: 52,
		PhaseCenterDeg: 47, PulseCount: 5, AvgAmplitudeMv: 26, MaxAmplitudeMv: 28,
		Status: model.ClusterCandidate, CreatedAt: now, UpdatedAt: now,
	}
	if err := s.Clusters.Insert(a); err != nil {
		t.Fatalf("seed cluster a: %v", err)
	}
	if err := s.Clusters.Insert(b); err != nil {
		t.Fatalf("seed cluster b: %v", err)
	}
	return s, trial.ID, a.ID, b.ID
}

// TestMergeClustersAtomicRejectsSources 验证原子合并：写入合并簇的同时
// 把两个原簇标记为 rejected，三步一并成功。
func TestMergeClustersAtomicRejectsSources(t *testing.T) {
	db := newTestDB(t)
	s, trialID, idA, idB := seedTrialAndClusters(t, db)

	merged := &model.Cluster{
		ID: NewID(), TrialID: trialID, PhaseStartDeg: 30, PhaseEndDeg: 52,
		PhaseCenterDeg: 41, PulseCount: 15, AvgAmplitudeMv: 25.33, MaxAmplitudeMv: 30,
		Status: model.ClusterCandidate, CreatedAt: "now", UpdatedAt: "now",
	}
	if err := s.Clusters.MergeClusters(merged, idA, idB, "now"); err != nil {
		t.Fatalf("MergeClusters: %v", err)
	}
	got, err := s.Clusters.Get(merged.ID)
	if err != nil {
		t.Fatalf("get merged: %v", err)
	}
	if got.Status != model.ClusterCandidate {
		t.Fatalf("merged status = %q, want %q", got.Status, model.ClusterCandidate)
	}
	for _, id := range []string{idA, idB} {
		c, err := s.Clusters.Get(id)
		if err != nil {
			t.Fatalf("get source %s: %v", id, err)
		}
		if c.Status != model.ClusterRejected {
			t.Fatalf("source %s status = %q, want %q", id, c.Status, model.ClusterRejected)
		}
	}
}

// TestMergeClustersRollsBackOnAlreadyRejected 验证一致性：若某原簇已为
// rejected，合并应整体回滚——合并簇不得入库，另一原簇不得被标记。
func TestMergeClustersRollsBackOnAlreadyRejected(t *testing.T) {
	db := newTestDB(t)
	s, trialID, idA, idB := seedTrialAndClusters(t, db)

	// 预先把 a 标记为 rejected，使事务中的 updateStatus(a) 命中 0 行。
	if err := s.Clusters.UpdateStatus(idA, model.ClusterRejected, "t1"); err != nil {
		t.Fatalf("pre-reject a: %v", err)
	}

	merged := &model.Cluster{
		ID: NewID(), TrialID: trialID, PhaseStartDeg: 30, PhaseEndDeg: 52,
		PhaseCenterDeg: 41, PulseCount: 15, AvgAmplitudeMv: 25.33, MaxAmplitudeMv: 30,
		Status: model.ClusterCandidate, CreatedAt: "now", UpdatedAt: "now",
	}
	if err := s.Clusters.MergeClusters(merged, idA, idB, "now"); err == nil {
		t.Fatalf("MergeClusters should fail when a source is already rejected")
	}
	// 合并簇不应入库。
	if _, err := s.Clusters.Get(merged.ID); !model.IsNotFound(err) {
		t.Fatalf("merged cluster should not exist after rollback, got err=%v", err)
	}
	// b 应保持 candidate，未被标记为 rejected。
	b, err := s.Clusters.Get(idB)
	if err != nil {
		t.Fatalf("get b after rollback: %v", err)
	}
	if b.Status == model.ClusterRejected {
		t.Fatalf("b should not be rejected after rollback, got %q", b.Status)
	}
}

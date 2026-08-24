// Package service 是领域编排层：把 store 的持久化与 trial/pulse/phase/cluster/
// diagnosis/snapshot 等业务包的纯逻辑串起来，对外提供事务级用例。
package service

import (
	"task222-pdcluster/internal/store"
)

// App 聚合所有服务，是 HTTP 层与领域层之间的编排入口。
type App struct {
	store     *store.Store
	Trials    *TrialService
	Pulses    *PulseService
	Clusters  *ClusterService
	Diagnosis *DiagnosisService
	Snapshots *SnapshotService
}

// New 用数据库构造 App 及其全部服务。
func New(db *store.DB) (*App, error) {
	s := store.NewStore(db.SQL())
	app := &App{store: s}
	app.Trials = &TrialService{store: s}
	app.Pulses = &PulseService{store: s}
	app.Clusters = &ClusterService{store: s}
	app.Diagnosis = &DiagnosisService{store: s}
	app.Snapshots = &SnapshotService{store: s}
	return app, nil
}

// Stats 返回全库统计。
func (a *App) Stats() (*store.Stats, error) {
	return a.store.Stats.Count()
}

package service

import (
	"path/filepath"
	"testing"

	"task222-pdcluster/internal/model"
	"task222-pdcluster/internal/store"
)

// newTestApp 构造一个临时数据库与 App，用于快照生命周期测试。
func newTestApp(t *testing.T) (*App, func()) {
	t.Helper()
	dbPath := filepath.Join(t.TempDir(), "snapshots.db")
	db, err := store.Open(dbPath)
	if err != nil {
		t.Fatal(err)
	}
	app, err := New(db)
	if err != nil {
		db.Close()
		t.Fatal(err)
	}
	return app, func() { _ = db.Close() }
}

// TestPublishSnapshotRejectedWhenAlreadyPublished 验证快照生命周期终态：
// 诊断快照一旦发布即进入终态，重复发布同一快照必须被拒绝，
// 且不改变其状态与发布时间。
func TestPublishSnapshotRejectedWhenAlreadyPublished(t *testing.T) {
	app, cleanup := newTestApp(t)
	defer cleanup()

	trial, err := app.Trials.Create("SNAP-TERM-001", "110kV cable", "110kV")
	if err != nil {
		t.Fatalf("create trial: %v", err)
	}

	sn, err := app.Snapshots.Create(trial.ID)
	if err != nil {
		t.Fatalf("create snapshot: %v", err)
	}
	if sn.Status != model.SnapshotDraft {
		t.Fatalf("new snapshot should be draft, got %s", sn.Status)
	}

	// 第一次发布：draft → published，成功。
	published, err := app.Snapshots.Publish(sn.ID)
	if err != nil {
		t.Fatalf("first publish: %v", err)
	}
	if published.Status != model.SnapshotPublished || published.PublishedAt == "" {
		t.Fatalf("after publish: status=%s published_at=%q", published.Status, published.PublishedAt)
	}
	firstPublishedAt := published.PublishedAt

	// 重新发布同一快照：published 为终态，必须被拒绝。
	if _, err := app.Snapshots.Publish(sn.ID); !model.IsInvalidState(err) {
		t.Fatalf("re-publish should be rejected with ErrInvalidState, got err=%v", err)
	}

	// 终态被拒绝后，状态与发布时间不得改变。
	again, err := app.Snapshots.GetSnapshot(sn.ID)
	if err != nil {
		t.Fatalf("get snapshot after rejected re-publish: %v", err)
	}
	if again.Status != model.SnapshotPublished {
		t.Fatalf("status must remain published, got %s", again.Status)
	}
	if again.PublishedAt != firstPublishedAt {
		t.Fatalf("published_at must not change: first=%q after=%q", firstPublishedAt, again.PublishedAt)
	}

	// 已被替代（superseded）的快照同样不可重新发布。
	if err := app.store.Snapshots.UpdateStatus(sn.ID, model.SnapshotSuperseded, firstPublishedAt); err != nil {
		t.Fatalf("mark superseded: %v", err)
	}
	if _, err := app.Snapshots.Publish(sn.ID); !model.IsInvalidState(err) {
		t.Fatalf("publish on superseded should be rejected with ErrInvalidState, got err=%v", err)
	}
	sup, err := app.Snapshots.GetSnapshot(sn.ID)
	if err != nil {
		t.Fatalf("get superseded snapshot: %v", err)
	}
	if sup.Status != model.SnapshotSuperseded || sup.PublishedAt != firstPublishedAt {
		t.Fatalf("superseded snapshot must be unchanged: status=%s published_at=%q (want %q)",
			sup.Status, sup.PublishedAt, firstPublishedAt)
	}
}

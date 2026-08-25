package snapshot

import (
	"testing"

	"task222-pdcluster/internal/model"
)

// TestPublishIsTerminal 验证发布终态：快照一旦发布即进入终态，
// 重复发布同一快照必须被拒绝，且不改变其状态与发布时间。
func TestPublishIsTerminal(t *testing.T) {
	t.Run("draft 可发布一次进入 published 终态", func(t *testing.T) {
		next, ok := Transition(model.SnapshotDraft, "publish")
		if !ok || next != model.SnapshotPublished {
			t.Fatalf("draft → published 应成功，得到 ok=%v next=%s", ok, next)
		}
	})

	t.Run("published 为终态，重复发布被拒绝", func(t *testing.T) {
		if _, ok := Transition(model.SnapshotPublished, "publish"); ok {
			t.Fatal("published → published 必须被拒绝（已发布不可重复发布）")
		}
	})

	t.Run("superseded 为终态，不可重新发布", func(t *testing.T) {
		if _, ok := Transition(model.SnapshotSuperseded, "publish"); ok {
			t.Fatal("superseded → published 必须被拒绝")
		}
	})

	t.Run("Publishable 仅允许 draft", func(t *testing.T) {
		if !Publishable(model.SnapshotDraft) {
			t.Fatal("draft 应可发布")
		}
		if Publishable(model.SnapshotPublished) {
			t.Fatal("published 不应可发布")
		}
		if Publishable(model.SnapshotSuperseded) {
			t.Fatal("superseded 不应可发布")
		}
	})

	t.Run("发布新版本时旧 published 被替代", func(t *testing.T) {
		next, ok := Transition(model.SnapshotPublished, "supersede")
		if !ok || next != model.SnapshotSuperseded {
			t.Fatalf("published → superseded 应成功，得到 ok=%v next=%s", ok, next)
		}
		if _, ok := Transition(model.SnapshotSuperseded, "supersede"); ok {
			t.Fatal("superseded 不可再被替代")
		}
	})
}

package snapshot

import "task222-pdcluster/internal/model"

// NextVersion 基于当前最大版本计算下一个快照版本号。
func NextVersion(maxVersion int) int { return maxVersion + 1 }

// Publishable 判断快照是否可从当前状态进入发布流程（仅 draft 可发布）。
func Publishable(status string) bool { return status == model.SnapshotDraft }

// IsPublished 判断快照是否已发布（含被替代）。
func IsPublished(status string) bool {
	return status == model.SnapshotPublished || status == model.SnapshotSuperseded
}

// Transition 计算快照状态流转后的新状态。
// draft → published → superseded（发布新版本时旧 published 被替代）。
// published 与 superseded 均为终态：快照一旦发布即进入终态，
// 重复发布同一快照必须被拒绝，且不改变其状态与发布时间。
func Transition(current, action string) (string, bool) {
	switch action {
	case "publish":
		if current == model.SnapshotDraft {
			return model.SnapshotPublished, true
		}
	case "supersede":
		if current == model.SnapshotPublished {
			return model.SnapshotSuperseded, true
		}
	}
	return current, false
}

// ShouldSupersede 判断发布新版本后，旧快照是否应被标记为替代：
// 仅当旧快照仍为 published 时。
func ShouldSupersede(oldStatus string) bool { return oldStatus == model.SnapshotPublished }

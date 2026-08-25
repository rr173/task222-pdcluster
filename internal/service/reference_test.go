package service

import (
	"math"
	"testing"

	"task222-pdcluster/internal/model"
	"task222-pdcluster/internal/phase"
	"task222-pdcluster/internal/store"
)

// newTestApp 打开一个内存数据库并构造 App。
func newTestApp(t *testing.T) (*App, func()) {
	t.Helper()
	db, err := store.Open(":memory:")
	if err != nil {
		t.Fatalf("open db: %v", err)
	}
	app, err := New(db)
	if err != nil {
		db.Close()
		t.Fatalf("init app: %v", err)
	}
	return app, func() { db.Close() }
}

// mustCreateTrial 登记一条试验用于后续测试。
func mustCreateTrial(t *testing.T, app *App) *model.Trial {
	t.Helper()
	trial, err := app.Trials.Create("REF-TEST-001", "110kV 电缆", "110kV")
	if err != nil {
		t.Fatalf("create trial: %v", err)
	}
	return trial
}

// TestSetReferenceZeroTimePreserved 验证零相位参考时间在服务、存储与相位
// 计算之间保持一致：原样保存、读取不变、覆盖更新仍不变，且以该时间作为
// 输入时 phase.Align 返回 0。
func TestSetReferenceZeroTimePreserved(t *testing.T) {
	app, cleanup := newTestApp(t)
	defer cleanup()
	trial := mustCreateTrial(t, app)

	// 选一个非零的参考时间，避免 0 掩盖偏移类缺陷。
	const wantZero = int64(1_700_000_000_000_000_000) // ns
	const freq = 50.0

	ref, err := app.Trials.SetReference(trial.ID, freq, wantZero)
	if err != nil {
		t.Fatalf("set reference: %v", err)
	}
	if ref.ZeroTimeNs != wantZero {
		t.Fatalf("returned zero_time_ns = %d, want %d (must be preserved verbatim)", ref.ZeroTimeNs, wantZero)
	}
	// 以参考时间本身作为输入，相位必须为 0。
	if got := phase.Align(wantZero, ref.ZeroTimeNs, freq); math.Abs(got) > 1e-9 {
		t.Fatalf("Align(zeroTime, zeroTime, freq) = %v, want 0", got)
	}

	// 从存储读回，参考时间不得偏移。
	got, err := app.Trials.GetReference(trial.ID)
	if err != nil {
		t.Fatalf("get reference: %v", err)
	}
	if got.ZeroTimeNs != wantZero {
		t.Fatalf("persisted zero_time_ns = %d, want %d", got.ZeroTimeNs, wantZero)
	}
	if got.FreqHz != freq {
		t.Fatalf("persisted freq_hz = %v, want %v", got.FreqHz, freq)
	}
	// 读回的参考时间作为输入，相位仍必须为 0。
	if got2 := phase.Align(wantZero, got.ZeroTimeNs, freq); math.Abs(got2) > 1e-9 {
		t.Fatalf("Align(zeroTime, persistedZeroTime, freq) = %v, want 0", got2)
	}

	// 覆盖更新（重新设置同一试验的参考）不得累计偏移。
	if _, err := app.Trials.SetReference(trial.ID, freq, wantZero); err != nil {
		t.Fatalf("re-set reference: %v", err)
	}
	got3, err := app.Trials.GetReference(trial.ID)
	if err != nil {
		t.Fatalf("get reference after re-set: %v", err)
	}
	if got3.ZeroTimeNs != wantZero {
		t.Fatalf("zero_time_ns drifted after re-set = %d, want %d (no accumulation allowed)", got3.ZeroTimeNs, wantZero)
	}
}

// TestSetReferenceAlignmentEndToEnd 验证脉冲相对保存的参考时间能被正确对齐：
// 一个落在周期整数倍 + 1/4 周期处的脉冲应对齐到 90°，且补偿延迟不引入参考漂移。
func TestSetReferenceAlignmentEndToEnd(t *testing.T) {
	app, cleanup := newTestApp(t)
	defer cleanup()
	trial := mustCreateTrial(t, app)

	const freq = 50.0
	const periodNs = int64(2e7) // 20ms
	const wantZero = int64(1_700_000_000_000_000_000) // 非零参考时间

	if _, err := app.Trials.SetReference(trial.ID, freq, wantZero); err != nil {
		t.Fatalf("set reference: %v", err)
	}
	if _, err := app.Trials.AddChannel(trial.ID, "CH0", 0); err != nil {
		t.Fatalf("add channel: %v", err)
	}
	if _, err := app.Trials.StartAcquisition(trial.ID); err != nil {
		t.Fatalf("start: %v", err)
	}

	// 脉冲落在参考时间 + 1 整周期 + 1/4 周期 → 应对齐到 90°。
	pulseTime := wantZero + periodNs + periodNs/4
	if _, err := app.Pulses.Ingest(trial.ID, "ch-id", 0, 1, pulseTime, 25.0); err != nil {
		t.Fatalf("ingest: %v", err)
	}

	if _, err := app.Pulses.Calibrate(trial.ID); err != nil {
		t.Fatalf("calibrate: %v", err)
	}
	pulses, err := app.Pulses.ListValidPulses(trial.ID)
	if err != nil {
		t.Fatalf("list valid pulses: %v", err)
	}
	if len(pulses) != 1 {
		t.Fatalf("want 1 valid pulse, got %d", len(pulses))
	}
	if d := math.Abs(pulses[0].PhaseDeg - 90.0); d > 1e-6 {
		t.Fatalf("phase_deg = %v, want 90", pulses[0].PhaseDeg)
	}

	// 校准后参考时间不得被改动。
	ref, err := app.Trials.GetReference(trial.ID)
	if err != nil {
		t.Fatalf("get reference after calibrate: %v", err)
	}
	if ref.ZeroTimeNs != wantZero {
		t.Fatalf("reference drifted during calibrate: got %d, want %d", ref.ZeroTimeNs, wantZero)
	}
}

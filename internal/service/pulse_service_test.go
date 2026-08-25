package service

import (
	"path/filepath"
	"testing"

	"task222-pdcluster/internal/model"
	"task222-pdcluster/internal/store"
)

// newServiceWithAcquiringTrial 构造一个处于 acquiring 状态、带单通道的试验，
// 供脉冲接收测试使用。返回 service、trialID、channelID。
func newServiceWithAcquiringTrial(t *testing.T) (*App, string, string) {
	t.Helper()
	db, err := store.Open(filepath.Join(t.TempDir(), "pdcluster.db"))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { db.Close() })

	app, err := New(db)
	if err != nil {
		t.Fatal(err)
	}
	tr, err := app.Trials.Create("BATCH-T-001", "110kV cable", "110kV")
	if err != nil {
		t.Fatal(err)
	}
	ch, err := app.Trials.AddChannel(tr.ID, "CH0", 0)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := app.Trials.StartAcquisition(tr.ID); err != nil {
		t.Fatal(err)
	}
	return app, tr.ID, ch.ID
}

func TestIngestBatchIsAllOrNothingOnInvalidArgument(t *testing.T) {
	app, trialID, channelID := newServiceWithAcquiringTrial(t)

	// 第一个脉冲合法、第二个脉冲幅值非法。
	inputs := []PulseInput{
		{ChannelID: channelID, ChannelIndex: 0, Seq: 1, TimeNs: 1_000_000, AmplitudeMv: 25},
		{ChannelID: channelID, ChannelIndex: 0, Seq: 2, TimeNs: 2_000_000, AmplitudeMv: -1},
	}
	if _, err := app.Pulses.IngestBatch(trialID, inputs); err != model.ErrInvalidArgument {
		t.Fatalf("err = %v, want ErrInvalidArgument", err)
	}

	// 全有或全无：整批失败后列表中不应残留任何脉冲。
	pulses, err := app.Pulses.ListPulses(trialID)
	if err != nil {
		t.Fatal(err)
	}
	if len(pulses) != 0 {
		t.Fatalf("expected zero pulses after failed batch, got %d", len(pulses))
	}
}

func TestIngestBatchIsAllOrNothingOnUnknownChannel(t *testing.T) {
	app, trialID, channelID := newServiceWithAcquiringTrial(t)

	// 第一个脉冲通道存在、第二个脉冲通道未知。
	inputs := []PulseInput{
		{ChannelID: channelID, ChannelIndex: 0, Seq: 1, TimeNs: 1_000_000, AmplitudeMv: 25},
		{ChannelID: "ghost", ChannelIndex: 99, Seq: 2, TimeNs: 2_000_000, AmplitudeMv: 25},
	}
	if _, err := app.Pulses.IngestBatch(trialID, inputs); !model.IsNotFound(err) {
		t.Fatalf("err = %v, want ErrNotFound", err)
	}
	pulses, err := app.Pulses.ListPulses(trialID)
	if err != nil {
		t.Fatal(err)
	}
	if len(pulses) != 0 {
		t.Fatalf("expected zero pulses after failed batch, got %d", len(pulses))
	}
}

func TestIngestBatchIdempotentOnDuplicates(t *testing.T) {
	app, trialID, channelID := newServiceWithAcquiringTrial(t)

	// 首批写入两条新脉冲。
	first := []PulseInput{
		{ChannelID: channelID, ChannelIndex: 0, Seq: 1, TimeNs: 1_000_000, AmplitudeMv: 25},
		{ChannelID: channelID, ChannelIndex: 0, Seq: 2, TimeNs: 2_000_000, AmplitudeMv: 30},
	}
	res, err := app.Pulses.IngestBatch(trialID, first)
	if err != nil {
		t.Fatal(err)
	}
	if res.Inserted != 2 || res.Duplicate != 0 {
		t.Fatalf("first batch: inserted=%d duplicate=%d", res.Inserted, res.Duplicate)
	}

	// 第二批重复前两条并新增一条：重复项幂等跳过，整批仍原子写入。
	second := []PulseInput{
		{ChannelID: channelID, ChannelIndex: 0, Seq: 1, TimeNs: 1_000_000, AmplitudeMv: 25}, // 重复
		{ChannelID: channelID, ChannelIndex: 0, Seq: 3, TimeNs: 3_000_000, AmplitudeMv: 12}, // 新增
		{ChannelID: channelID, ChannelIndex: 0, Seq: 2, TimeNs: 2_000_000, AmplitudeMv: 30}, // 重复
	}
	res, err = app.Pulses.IngestBatch(trialID, second)
	if err != nil {
		t.Fatal(err)
	}
	if res.Inserted != 1 || res.Duplicate != 2 {
		t.Fatalf("second batch: inserted=%d duplicate=%d", res.Inserted, res.Duplicate)
	}

	// 列表应恰好包含三条不同脉冲。
	pulses, err := app.Pulses.ListPulses(trialID)
	if err != nil {
		t.Fatal(err)
	}
	if len(pulses) != 3 {
		t.Fatalf("expected 3 pulses total, got %d", len(pulses))
	}
}

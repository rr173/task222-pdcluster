// Command pdcluster 高压电缆局部放电相位聚类服务入口。
//
// 支持三个标志：
//   - --addr :8080       监听地址（默认 :8080）
//   - --db ./pdcluster.db SQLite 数据库路径（默认 ./pdcluster.db）
//   - --smoke-test       执行端到端冒烟：真实创建数据、关闭并重开数据库
//     验证持久化与重启恢复，随后以 0 退出码结束。
package main

import (
	"flag"
	"fmt"
	"log"
	"math"
	"net/http"
	"os"

	"task222-pdcluster/internal/httpapi"
	"task222-pdcluster/internal/model"
	"task222-pdcluster/internal/service"
	"task222-pdcluster/internal/store"
)

func main() {
	addr := flag.String("addr", ":8080", "HTTP listen address")
	dbPath := flag.String("db", "./pdcluster.db", "SQLite database path")
	smoke := flag.Bool("smoke-test", false, "run end-to-end smoke test and exit")
	flag.Parse()

	if *smoke {
		if err := runSmokeTest(*dbPath); err != nil {
			fmt.Fprintln(os.Stderr, "SMOKE TEST FAILED:", err)
			os.Exit(1)
		}
		fmt.Println("SMOKE TEST PASSED")
		return
	}

	db, err := store.Open(*dbPath)
	if err != nil {
		log.Fatalf("open db: %v", err)
	}
	defer db.Close()

	app, err := service.New(db)
	if err != nil {
		log.Fatalf("init services: %v", err)
	}
	srv := httpapi.New(app)
	log.Printf("task222-pdcluster listening on %s (db=%s)", *addr, *dbPath)
	if err := http.ListenAndServe(*addr, srv.Handler()); err != nil {
		log.Fatalf("http server: %v", err)
	}
}

// 冒烟测试的工频与采集参数。
const (
	smokeFreqHz    = 50.0  // 工频频率
	smokePeriodNs  = int64(2e7) // 50Hz 周期 = 20ms = 2e7 ns
	smokeChannels  = 3
	smokeEventsPer = 8 // 每个相位带的放电事件数
	smokeNoisePer  = 4 // 每通道背景噪声脉冲数
	smokeInterfCnt = 5 // 周期性干扰脉冲数
)

// 通道相对参考通道的传输延迟（ns），用于验证校准。
var smokeChannelDelay = map[int]int64{0: 0, 1: 1000, 2: -500}

// runSmokeTest 执行端到端冒烟：
//  1. 打开数据库 A，走完整闭环：试验 → 通道/相位参考 → 采集 →
//     脉冲接收 → 校准 → 背景过滤/去重 → 聚类 → 分类 → 裁决 →
//     复核 → 快照发布 → 封存；
//  2. 幂等验证：重复接收相同脉冲被跳过；
//  3. 封存后拒绝再接收脉冲；
//  4. 关闭数据库 A，重开同一路径数据库 B，验证数据仍在（重启恢复）。
func runSmokeTest(dbPath string) error {
	if dbPath != ":memory:" {
		_ = os.Remove(dbPath)
	}

	db, err := store.Open(dbPath)
	if err != nil {
		return fmt.Errorf("open db: %w", err)
	}
	app, err := service.New(db)
	if err != nil {
		db.Close()
		return fmt.Errorf("init services: %w", err)
	}

	// --- 步骤 1：创建试验 + 通道 + 相位参考 ---
	trial, err := app.Trials.Create("PD-TEST-001", "110kV 交联聚乙烯电缆", "110kV")
	if err != nil {
		db.Close()
		return fmt.Errorf("create trial: %w", err)
	}
	channelIDs := make([]string, smokeChannels)
	for i := 0; i < smokeChannels; i++ {
		ch, err := app.Trials.AddChannel(trial.ID, fmt.Sprintf("CH%d", i), i)
		if err != nil {
			db.Close()
			return fmt.Errorf("add channel %d: %w", i, err)
		}
		channelIDs[i] = ch.ID
	}
	if _, err := app.Trials.SetReference(trial.ID, smokeFreqHz, 0); err != nil {
		db.Close()
		return fmt.Errorf("set reference: %w", err)
	}

	// --- 步骤 2：开始采集 ---
	trial, err = app.Trials.StartAcquisition(trial.ID)
	if err != nil {
		db.Close()
		return fmt.Errorf("start acquisition: %w", err)
	}

	// --- 步骤 3：接收脉冲 ---
	// 内部气隙放电：相位 30° 与 210° 附近（正负半周对称），
	// 每个放电事件被 3 个通道同时记录（同 seq），时间戳叠加通道延迟。
	seq := int64(0)
	phaseBands := []float64{30, 210}
	for _, ph := range phaseBands {
		for k := 0; k < smokeEventsPer; k++ {
			phJitter := ph + float64(k-smokeEventsPer/2)*1.5
			baseT := int64(phJitter / 360.0 * float64(smokePeriodNs))
			for ch := 0; ch < smokeChannels; ch++ {
				t := baseT + smokeChannelDelay[ch]
				amp := 25.0 + float64(k)*1.5 // 25~35 mV
				if _, err := app.Pulses.Ingest(trial.ID, channelIDs[ch], ch, seq, t, amp); err != nil {
					db.Close()
					return fmt.Errorf("ingest discharge seq=%d ch=%d: %w", seq, ch, err)
				}
			}
			seq++
		}
	}
	// 背景噪声：低幅值，不构成放电。
	for ch := 0; ch < smokeChannels; ch++ {
		for k := 0; k < smokeNoisePer; k++ {
			t := int64(k)*500000 + smokeChannelDelay[ch] + 123456
			if _, err := app.Pulses.Ingest(trial.ID, channelIDs[ch], ch, seq, t, 1.0); err != nil {
				db.Close()
				return fmt.Errorf("ingest noise: %w", err)
			}
			seq++
		}
	}
	// 周期性干扰：通道 0 等间隔、同幅值（模拟无线/开关干扰）。
	for k := 0; k < smokeInterfCnt; k++ {
		t := int64(1000000 + k*2000000)
		if _, err := app.Pulses.Ingest(trial.ID, channelIDs[0], 0, seq, t, 55.0); err != nil {
			db.Close()
			return fmt.Errorf("ingest interference: %w", err)
		}
		seq++
	}

	// --- 步骤 4：幂等验证 ---
	reIngestT := int64(math.Round(30.0/360.0*float64(smokePeriodNs))) + smokeChannelDelay[0]
	res, err := app.Pulses.Ingest(trial.ID, channelIDs[0], 0, 0, reIngestT, 25.0)
	if err != nil {
		db.Close()
		return fmt.Errorf("re-ingest: %w", err)
	}
	if res.Duplicate != 1 || res.Inserted != 0 {
		db.Close()
		return fmt.Errorf("re-ingest should be duplicate, got inserted=%d duplicate=%d", res.Inserted, res.Duplicate)
	}

	// --- 步骤 5：校准（通道延迟 + 相位对齐） ---
	delays, err := app.Pulses.Calibrate(trial.ID)
	if err != nil {
		db.Close()
		return fmt.Errorf("calibrate: %w", err)
	}
	if d := delays[1]; math.Abs(d-1000) > 200 {
		db.Close()
		return fmt.Errorf("channel 1 delay should be ~1000ns, got %.1f", d)
	}
	if d := delays[2]; math.Abs(d+500) > 200 {
		db.Close()
		return fmt.Errorf("channel 2 delay should be ~-500ns, got %.1f", d)
	}

	// --- 步骤 6：背景过滤 + 去重 ---
	bgCount, err := app.Pulses.ApplyBackgroundFilter(trial.ID, 2.0)
	if err != nil {
		db.Close()
		return fmt.Errorf("background filter: %w", err)
	}
	if bgCount != smokeChannels*smokeNoisePer {
		db.Close()
		return fmt.Errorf("expected %d background pulses, got %d", smokeChannels*smokeNoisePer, bgCount)
	}
	dupCount, err := app.Pulses.ApplyDedup(trial.ID)
	if err != nil {
		db.Close()
		return fmt.Errorf("dedup: %w", err)
	}
	if dupCount < smokeInterfCnt {
		db.Close()
		return fmt.Errorf("expected >=%d duplicate pulses, got %d", smokeInterfCnt, dupCount)
	}

	// --- 步骤 7：结束采集并聚类 ---
	trial, err = app.Trials.FinishAcquisition(trial.ID)
	if err != nil {
		db.Close()
		return fmt.Errorf("finish acquisition: %w", err)
	}
	clusters, err := app.Clusters.Cluster(trial.ID)
	if err != nil {
		db.Close()
		return fmt.Errorf("cluster: %w", err)
	}
	if len(clusters) == 0 {
		db.Close()
		return fmt.Errorf("no clusters extracted")
	}

	// --- 步骤 8：分类解释 ---
	interps, err := app.Diagnosis.Classify(trial.ID)
	if err != nil {
		db.Close()
		return fmt.Errorf("classify: %w", err)
	}
	if len(interps) == 0 {
		db.Close()
		return fmt.Errorf("no interpretation generated")
	}

	// --- 步骤 9：裁决（确认一个簇，标记一个簇为干扰） ---
	if len(clusters) >= 2 {
		if _, err := app.Diagnosis.ConfirmCluster(clusters[0].ID); err != nil {
			db.Close()
			return fmt.Errorf("confirm cluster: %w", err)
		}
		if _, err := app.Diagnosis.MarkInterference(clusters[1].ID); err != nil {
			db.Close()
			return fmt.Errorf("mark interference: %w", err)
		}
	} else if len(clusters) == 1 {
		if _, err := app.Diagnosis.ConfirmCluster(clusters[0].ID); err != nil {
			db.Close()
			return fmt.Errorf("confirm cluster: %w", err)
		}
	}

	// --- 步骤 10：复核 + 发布快照 + 封存 ---
	trial, err = app.Trials.Review(trial.ID)
	if err != nil {
		db.Close()
		return fmt.Errorf("review: %w", err)
	}
	sn, err := app.Snapshots.Create(trial.ID)
	if err != nil {
		db.Close()
		return fmt.Errorf("create snapshot: %w", err)
	}
	sn, err = app.Snapshots.Publish(sn.ID)
	if err != nil {
		db.Close()
		return fmt.Errorf("publish snapshot: %w", err)
	}
	if sn.Status != model.SnapshotPublished {
		db.Close()
		return fmt.Errorf("snapshot should be published, got %s", sn.Status)
	}
	trial, err = app.Trials.Seal(trial.ID)
	if err != nil {
		db.Close()
		return fmt.Errorf("seal: %w", err)
	}
	if trial.Status != model.TrialSealed {
		db.Close()
		return fmt.Errorf("trial should be sealed, got %s", trial.Status)
	}

	// --- 步骤 11：封存后拒绝写入 ---
	if _, err := app.Pulses.Ingest(trial.ID, channelIDs[0], 0, 99999, 0, 25.0); err == nil {
		db.Close()
		return fmt.Errorf("sealed trial should reject pulse ingest")
	}

	// --- 步骤 12：重启恢复 ---
	db.Close()

	db2, err := store.Open(dbPath)
	if err != nil {
		return fmt.Errorf("reopen db: %w", err)
	}
	defer db2.Close()
	app2, err := service.New(db2)
	if err != nil {
		return fmt.Errorf("reinit services: %w", err)
	}
	t2, err := app2.Trials.Get(trial.ID)
	if err != nil {
		return fmt.Errorf("get trial after reopen: %w", err)
	}
	if t2.Status != model.TrialSealed {
		return fmt.Errorf("trial status after reopen should be sealed, got %s", t2.Status)
	}
	clusters2, err := app2.Clusters.ListClusters(trial.ID)
	if err != nil {
		return fmt.Errorf("list clusters after reopen: %w", err)
	}
	if len(clusters2) == 0 {
		return fmt.Errorf("no clusters after reopen")
	}
	snaps2, err := app2.Snapshots.ListSnapshots(trial.ID)
	if err != nil {
		return fmt.Errorf("list snapshots after reopen: %w", err)
	}
	if len(snaps2) == 0 || snaps2[0].Status != model.SnapshotPublished {
		return fmt.Errorf("published snapshot missing after reopen")
	}
	return nil
}

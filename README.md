# task222-pdcluster — 高压电缆局部放电相位聚类服务

纯后端 Go 服务：把电缆局部放电（PD）脉冲按工频相位聚类，识别内部气隙、悬浮金属、
沿面放电等缺陷来源，并区分无线/开关干扰。数据用 SQLite 持久化，支持重启恢复。

## 业务闭环

登记电缆试验 → 设置工频相位参考与通道 → 开始采集 → 多通道接收放电脉冲（幂等去重）→
通道延迟校准 + 相位对齐 → 背景/重复干扰过滤 → 相位聚类（PRPD 谱）→ 缺陷类型推断 →
簇裁决（确认/干扰标记）→ 复核 → 发布诊断快照 → 封存。

## 核心状态机

- 试验：`preparing → acquiring → clustering → reviewing → sealed`（sealed 单向终态）
- 脉冲：`uncalibrated → valid / background / duplicate`（原始数据永不删除）
- 相位簇：`candidate → stable / conflict → confirmed / rejected`
- 快照：`draft → published → superseded`

## 标准命令

```bash
# 编译 / 静态检查 / 单元测试
CGO_ENABLED=0 GOTOOLCHAIN=local go build ./...
CGO_ENABLED=0 GOTOOLCHAIN=local go vet   ./...
CGO_ENABLED=0 GOTOOLCHAIN=local go test  ./...

# 端到端冒烟（真实建数据 + 关闭重开数据库验证恢复，退出码 0 表示通过）
go run ./cmd/pdcluster --smoke-test

# 启动服务
go run ./cmd/pdcluster --addr :8080 --db ./pdcluster.db
```

## API 入口（前缀 /api，共 29 个）

| 能力 | 入口 |
|---|---|
| 试验登记/列表/详情/生命周期 | `POST /api/trials`、`GET /api/trials`、`GET /api/trials/{id}`、`POST /api/trials/{id}/start|finish|review|seal` |
| 相位参考 | `POST /api/trials/{id}/reference`、`GET /api/trials/{id}/reference` |
| 通道 | `POST /api/trials/{id}/channels`、`GET /api/trials/{id}/channels` |
| 脉冲接收/查询 | `POST /api/trials/{id}/pulses`、`POST /api/trials/{id}/pulses/batch`、`GET /api/trials/{id}/pulses` |
| 校准 | `POST /api/trials/{id}/calibrate`、`GET /api/trials/{id}/calibration` |
| 聚类 | `POST /api/trials/{id}/cluster`、`GET /api/trials/{id}/clusters`、`GET /api/clusters/{id}`、`POST /api/clusters/merge` |
| 诊断 | `POST /api/clusters/{id}/interference`、`POST /api/clusters/{id}/confirm`、`GET /api/trials/{id}/interpretations` |
| 快照 | `POST /api/trials/{id}/snapshots`、`GET /api/trials/{id}/snapshots`、`GET /api/snapshots/{id}`、`POST /api/snapshots/{id}/publish` |
| 统计/健康 | `GET /api/stats`、`GET /api/health` |

## 技术要点

- Go 1.26.3，`CGO_ENABLED=0`，纯 Go SQLite 驱动 `modernc.org/sqlite v1.52.0`（离线可构建）。
- 唯一键幂等：试验指纹、脉冲 `UNIQUE(trial_id, channel_index, seq)`、快照 `UNIQUE(trial_id, version)`。
- 相位对齐：时间戳 → 工频相位角（`phase.Align`），通道延迟用到达时间差中位数估计（`phase.DelayEstimator`）。
- 缺陷推断：正负半周对称性（内部气隙）、幅值阈值（悬浮）、不对称（沿面）。
- 封存快照不可覆盖，原始脉冲/簇不删除，只标记排除原因。

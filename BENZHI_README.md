# BENZHI 评测说明 — task222-pdcluster

本文件面向自动化评测，说明服务如何构建、运行与自检。

## 模块与入口

- 模块名：`task222-pdcluster`
- 入口：`cmd/pdcluster/main.go`
- 依赖：`modernc.org/sqlite v1.52.0`（纯 Go，CGO 无关）
- Go 版本：`1.26.3`（`GOTOOLCHAIN=local`，`CGO_ENABLED=0`）

## 环境变量

```text
GOPROXY=https://goproxy.cn,direct
GOSUMDB=sum.golang.google.cn
CGO_ENABLED=0
GOTOOLCHAIN=local
```

## 构建与测试

```bash
CGO_ENABLED=0 GOTOOLCHAIN=local go build ./...
CGO_ENABLED=0 GOTOOLCHAIN=local go vet   ./...
CGO_ENABLED=0 GOTOOLCHAIN=local go test  ./...
```

## 冒烟测试契约

```bash
go run ./cmd/pdcluster --smoke-test
```

`--smoke-test` 不启动长驻服务，而是端到端执行：

1. 创建试验 + 3 通道 + 相位参考（50Hz）；
2. 接收多通道放电脉冲（含背景噪声与周期性干扰）；
3. 幂等验证：重复提交同一脉冲被跳过（`duplicate=1`）；
4. 通道延迟校准：通道 1 ≈ +1000ns、通道 2 ≈ -500ns 被正确估计；
5. 背景过滤 + 去重：低幅值标 `background`、周期重复标 `duplicate`；
6. 相位聚类产出簇，缺陷推断产出解释；
7. 簇裁决（确认 / 干扰标记）、复核、发布快照、封存；
8. 封存后拒绝再写入；
9. 关闭数据库并重开同一路径，验证试验 sealed、簇、已发布快照均恢复。

全部通过后以退出码 0 结束。

## Docker 双架构

```bash
# 单平台构建（benzhi 评测镜像）
bash build_benzhi_docker.sh my-image linux/amd64

# 冒烟运行
docker run --rm my-image:latest --smoke-test

# 启动服务
docker run --rm -p 8080:8080 my-image:latest --addr :8080
```

双架构基线验证使用仓库内置脚本：

```bash
python3 scripts/docker_baseline_validation.py \
  --project-dir <项目根> --verify-and-record
```

该脚本对 `linux/amd64` 与 `linux/arm64` 分别执行 `docker buildx build --load` +
`docker run --smoke-test`，四项均退出码 0 才写入
`.private/docker_baseline_validation.json`（status=passed）。

## API 前缀

所有路由以 `/api` 开头，共 29 个入口（试验 8、参考 2、通道 2、脉冲 3、校准 2、
聚类 4、诊断 3、快照 4、统计/健康 2）。

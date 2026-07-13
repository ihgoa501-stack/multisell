# Prism → LingMirror Image Service 迁移盘点

> 日期：2026-07-12
> 性质：代码级迁移准备（只读盘点，不授权删除）

## 1. 结论

`actual`：当前有三组名称或职责部分重叠的图片能力：

1. MultiSell 内嵌 `imagegen` 的确定性图片处理、生成记录、画布和提示词模板；
2. MultiSell 的 `prismadapter` 以及 Listing/Product Analysis 直调链；
3. 独立仓库 `/Users/lc/prism` 的异步管线、Provider、合规检查、模板、人工审核、S3 和指标。

`actual`：它们并非同一可执行契约。MultiSell 旧客户端调用同步 `POST /api/v1/generate`，而独立 Prism 提供异步 `POST /v1/generate` + `GET /v1/jobs/:id`。Prism 的 Job 只在进程内存中，重启即丢失。

`actual`：`services/image-service/` 已接替 MultiSell 的图片执行边界；凌镜 `productimage` 保留 Owner、商品、权利、审批、渠道和 Listing 经营状态。MultiSell 旧运行客户端及注入已移除，但历史数据迁移与真实外部 Provider 验收仍未完成。

## 2. 处置语义

| 处置 | 含义 |
|---|---|
| `reuse` | 概念、算法、测试用例或模板可携带到新契约；不表示原文件原样复制 |
| `rewrite` | 业务需求保留，但必须按持久化 Job/Attempt/Blob、Owner 隔离和异步契约重写 |
| `superseded` | 新契约生效后不再作为运行路径，但切换前仍不得删除 |
| `archive` | 迁移验收后仅作历史证据保留；本文不执行归档 |

## 3. MultiSell 调用面与数据盘点

| 类别 | 当前文件/对象 | 当前事实 | 处置 | 迁移验收点 |
|---|---|---|---|---|
| 配置 | `backend-go/configs/config.yaml`, `backend-go/internal/config/config.go`, `backend-go/internal/config/config_test.go` | 旧 `PRISM_*` 运行配置已移除 | `superseded` | Image Service 使用自身私有地址/凭据；不存在失败后原图继续发布开关 |
| 旧客户端 | `backend-go/internal/prismadapter/` | 旧同步客户端已零引用并从工作树移除 | `superseded` | 运行调用方已切到 `imageservice`/`productimage` |
| 组装 | `backend-go/internal/httpx/router.go` | 不再定义 typed nil、strict 兼容开关或注入 Prism | `superseded` | loop/listingtask/productanalysis 构造器已移除兼容参数 |
| Product Analysis | `backend-go/internal/domain/productanalysis/routes.go`, `backend-go/internal/domain/productanalysis/routes_test.go`, `backend-go/internal/domain/productanalysis/handler.go` | 旧 `POST /api/v1/product-analysis/trigger-prism` 及死 handler 已移除 | `superseded` | 回归测试锁定 404 |
| 路由治理 | `backend-go/internal/platform/routecatalog/policy.go`, `backend-go/internal/platform/routecatalog/mutation_policy.tsv` | 登记 `trigger-prism` 写路由 | `rewrite` | 新 `/api/v1/product-images` 写路由入目录与审计，旧路由退出 |
| Listing 输入 | `backend-go/internal/domain/listing/model.go`, `backend-go/internal/domain/listing/service.go` | `prism_enabled/options` 旧输入已移除 | `superseded` | Listing 不再接收启用旧作图运行时的参数 |
| 发布链 | `backend-go/internal/domain/listingtask/routes.go`, `backend-go/internal/domain/listingtask/service.go`, `backend-go/internal/domain/listingtask/listingtask_test.go` | 同步 Prism 检查、结果写入和 fail-open 分支已移除 | `superseded` | 生产入口缺少图片放行凭证时在平台调用前失败关闭 |
| Loop 注入 | `backend-go/internal/domain/loop/routes.go`, `backend-go/internal/domain/loop/service.go` | Prism 依赖与透传参数已移除 | `superseded` | loop 只构造无 Prism 依赖的 listingtask service |
| 内嵌确定性处理 | `backend-go/internal/domain/imagegen/prism.go`, `backend-go/internal/domain/imagegen/prism_test.go` | 尺寸、白底、水印、缓存及测试 | `reuse` | 迁移算法意图和对应回归用例到 Image Service；哈希与输出契约重新定义 |
| 旧 imagegen CRUD | `backend-go/internal/domain/imagegen/model.go`, `backend-go/internal/domain/imagegen/service.go`, `backend-go/internal/domain/imagegen/handler.go`, `backend-go/internal/domain/imagegen/routes.go`, `backend-go/internal/domain/imagegen/imagegen_test.go` | `/api/v1/image-gen`；生成记录、画布、提示词模板 | `rewrite` | 数据逐项映射至 asset/task/review/image set；将 Owner 隔离和 CRUD 回归用例映射到新契约；不把 CRUD 存在当作执行成功 |
| 表 | `product_image_gen`, `product_canvases`, `prompt_template` | 在 `backend-go/migrations/000001_init_schema.up.sql` 定义；Prism 本身无持久化表 | `rewrite` | 迁移前统计数量、所有者、URL/字节可读性；双读核对后才停止旧写 |
| 前端 | `frontend-next/src/app/(main)/agent-upgrades/page.tsx` | 旧 Prism 展示卡已移除 | `superseded` | Owner 统一使用 `/product-images` |

## 4. 独立 Prism 仓库盘点

| 类别 | 文件 | 处置 | 依据/要求 |
|---|---|---|---|
| 入口/文档 | `/Users/lc/prism/cmd/server/main.go`, `/Users/lc/prism/README.md`, `/Users/lc/prism/VERSION`, `/Users/lc/prism/go.mod`, `/Users/lc/prism/AGENTS.md` | `archive` | 新服务契约稳定后保留历史版本，不原样成为生产入口 |
| 配置 | `/Users/lc/prism/internal/config/config.go` | `rewrite` | 保留 Provider/存储配置项意图；删除“预留但未接入”的数据库假设，新服务启动时验证持久化与密钥条件 |
| HTTP/Job | `/Users/lc/prism/internal/api/router.go`, `/Users/lc/prism/internal/api/jobstore.go`, `/Users/lc/prism/internal/api/router_test.go` | `rewrite` | 保留异步交互意图；改为持久化 Job/Attempt、幂等、认证、重启恢复和对账 |
| 认证 | `/Users/lc/prism/internal/auth/apikey.go` | `rewrite` | 单 API key 不足以代表 Owner 审批；改用凌镜后端签发的服务身份/目标绑定执行授权 |
| Pipeline | `/Users/lc/prism/internal/pipeline/types.go`, `/Users/lc/prism/internal/pipeline/modes.go`, `/Users/lc/prism/internal/pipeline/executor.go`, `/Users/lc/prism/internal/pipeline/metrics.go`, `/Users/lc/prism/internal/pipeline/review.go`, `/Users/lc/prism/internal/pipeline/executor_test.go` | `reuse` + `rewrite` | 复用 stage/mode/指标/审核用例；不复用“整条管线盲重试”和进程内 review queue |
| Provider 契约 | `/Users/lc/prism/internal/provider/interfaces.go` | `rewrite` | 按 capability/estimate/submit/reconcile 及明确费用语义重写 |
| Fal.ai | `/Users/lc/prism/internal/provider/falai.go` | `reuse` | 复用请求/轮询/错误解析知识和测试场景；重新校验当前官方契约后才启用 |
| 通义 | `/Users/lc/prism/internal/provider/tongyi.go` | `reuse` | 复用异步 task 调用经验；未配置凭据时必须失败关闭，不回退 mock |
| 去背景 | `/Users/lc/prism/internal/provider/removebg.go` | `reuse` | 作为确定性/专用 Provider 对照；先用旧测试证明行为再迁移 |
| Provider 测试 | `/Users/lc/prism/internal/provider/provider_test.go` | `reuse` | 迁移为契约测试，补重复收费、超时对账、无效图片和上游限流 |
| 合规 | `/Users/lc/prism/internal/compliance/validator.go`, `/Users/lc/prism/internal/compliance/validator_test.go` | `reuse` | 仅作确定性技术检查；不得将白底比/尺寸通过命名为渠道合法或可发布 |
| 模板 | `/Users/lc/prism/internal/template/registry.go`, `/Users/lc/prism/internal/template/registry_test.go`, `/Users/lc/prism/internal/template/templates/studio_basic.yaml`, `/Users/lc/prism/internal/template/templates/white_bg.yaml`, `/Users/lc/prism/internal/template/templates/winter_outdoor.yaml` | `reuse` | 迁移为有版本的 Owner 内部模板；渠道适用范围和规则时间不能从旧 YAML 推断 |
| 存储 | `/Users/lc/prism/internal/storage/s3.go` | `reuse` + `rewrite` | 复用 S3 封装经验；新 BlobStore 必须内容寻址、私有访问、校验哈希与处理数据库/对象存储一致性 |
| 可观测性 | `/Users/lc/prism/internal/telemetry/metrics.go`, `/Users/lc/prism/internal/logger/logger.go` | `reuse` | 迁移指标意图；新指标不暴露提示词、图片 URL、凭据或个人数据 |
| Go 库客户端 | `/Users/lc/prism/pkg/client/client.go` | `superseded` | 它在进程内直跑 pipeline，不是凌镜到 Image Service 的私有 HTTP 客户端 |

## 5. 路由与契约差异

| 系统 | 路由 | 状态/数据 | 风险 |
|---|---|---|---|
| MultiSell 旧 Prism Client | `POST {base}/api/v1/generate` | 期待同步 `job_id/output_url/compliance_report/risk_score` | 与实际 Prism API 不匹配 |
| 独立 Prism | `POST /v1/generate` | `202 PENDING`，后台 goroutine | 无持久化、无幂等、无重启恢复 |
| 独立 Prism | `GET /v1/jobs/:id` | 进程内 Job | 重启后不可对账 |
| 独立 Prism | `POST /v1/review` | 进程内 ReviewQueue | 不能代替凌镜 Owner 审批与审计 |
| 独立 Prism | `GET /v1/templates`, `GET /health` | 列表/存活 | 可复用交互意图，需统一认证和 readiness |
| Image Service 当前 | `POST /internal/v1/blobs`, `GET /internal/v1/blobs/{id}/content`, `POST /internal/v1/jobs`, `GET /internal/v1/jobs/{id}`, `GET /internal/v1/jobs/{id}/attempts`, `POST /internal/v1/jobs/{id}/executions`, `GET /healthz`, `GET /readyz` | 持久化 Job/Attempt/Blob 和确定性 resize 骨架 | `implemented / automated_verified` 范围必须以当前测试为准；未证明 Provider 和生产部署 |

## 6. 迁移门禁（全部满足前不删除）

1. 新 Image Service 和凌镜 `productimage` 合同测试覆盖 Blob 字节、Job/Attempt、幂等、超时、重启恢复和失败关闭。
2. 旧 `trigger-prism`、Listing 发布链和 loop 不再存在运行调用者，路由目录和前端已同步。
3. `product_image_gen` / `product_canvases` / `prompt_template` 已做只读数量、Owner 归属、URL/字节可读性盘点，并有双读/回填/差异报告。
4. 所有可保留的 Provider、模板、技术检查和测试用例已迁移或明确拒绝，且有可追溯依据。
5. 凌镜 `/product-images` Owner 页面可完成原图选择/上传、任务发起、状态查看、原图/候选图对比、Owner 决定和 Listing 草稿图片集交接。
6. 发布使用的最终字节、顺序、用途、渠道规则和 Owner 审批使用同一不可变哈希；任一变化使旧批准失效。
7. 完成回滚演练：数据可回滚，但不恢复 `strict=false` 的不安全发布路径。

## 7. 自动遗漏检查

```bash
scripts/check_prism_migration_inventory.sh
```

脚本只读两个仓库，检查当前 Prism 相关 MultiSell 文件和 `/Users/lc/prism` 受管文件是否在本盘点中出现。新增相关文件未分类时返回非零退出码。

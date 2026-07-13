# 正式运行底座检查点（2026-07-13 15:00）

> 范围：`REAL_OPERATION_READINESS_PLAN.md` 的 P0-2。
> 结论：`in progress`，不得标记为生产可用。

## 已实际验证

- 发布候选：服务器工作树已能定位到 exact commit `a8a46963d2dc75d071ed400016f76515b70644c9`。
- 构建：候选后端和前端镜像 `a8a46963` 已在服务器构建成功；尚未替换当前运行镜像。
- 生产数据：PostgreSQL 持久化卷仍为 `multisell_postgres_go_data`；正式库为迁移 `111`、`dirty=false`、192 张 public 表。
- 生产服务：数据库、后端和前端容器健康；`/api/ready` 返回 database、event_bus、scheduler、traffic 均为 true。
- 备份恢复：`multisell_v111_2026-07-13_145306.dump` 已恢复到隔离数据库，恢复后为迁移 `111`、192 张表。
- 迁移演练：隔离数据库已从 `111` 顺序升级到 `151`，升级后为 241 张表；随后删除隔离数据库。
- 回归核对：隔离演练后正式库仍为迁移 `111`、192 张表，服务继续 ready。
- 外部写入：本次未接入真实渠道、供应商、银行或付费 Provider 凭据，未执行采购、发布、退款或资金动作。

以上分别属于 `actual / manually_verified`，不证明真实经营闭环已经发生。

## 操作事件与恢复

第一次隔离迁移尝试通过 Compose 临时覆盖 `DB_NAME`，导致数据库容器被按临时默认库名重建。持久化数据卷没有被删除，但容器默认库名短暂错误。发现后立即停止迁移，恢复生产 Compose 配置，并重新核对正式库版本、表数、审计记录和服务 readiness。后续隔离迁移改为迁移工具直接连接固定命名的临时数据库，不再覆盖 Compose 服务配置。

该事件未观察到数据丢失，但说明正式迁移必须继续遵守：不通过环境覆盖重建数据库服务；先在独立数据库演练，再由生产门禁决定是否执行。

## 当前硬阻塞

`IMAGE_TAG=a8a46963 ./scripts/verify_prod_compose.sh` 仍失败，精确原因只有以下四项：

1. 生产备份未要求异地保存；
2. 生产备份未要求异地不可变保存；
3. 审计 checkpoint 未要求异地保存；
4. 审计 checkpoint 未要求异地不可变保存。

服务器当前没有 AWS CLI、AWS 凭据、S3 Object Lock bucket 或真实告警 Webhook。不得只把四个开关改成 `true` 冒充通过；必须先建立真实的异地不可变对象存储，并由脚本验证 Versioning、默认 Object Lock 保留策略和对象 retain-until 元数据。

图片服务候选镜像也尚未完成：Docker Hub 解析基础镜像时网络超时。后端和前端构建成功不能替代图片服务构建。

## 下一动作

1. Owner 批准新增一个会产生费用的 S3 Object Lock 存储桶，并提供最小权限凭据；
2. AI 安装 AWS CLI、配置秘密文件，实际执行一次备份上传和审计 checkpoint 上传，并验证不可变保留元数据；
3. Owner 提供一个真实告警接收通道，AI 触发一次测试告警并确认送达；
4. 重试图片服务镜像构建；
5. 四项门禁、告警、候选镜像和回滚点全部通过后，才允许把正式库从 `111` 升级到 `151` 并切换 exact release。

## 15:30 补充：图片服务、秘密轮换与审计链

- 图片服务：Docker Hub 超时后复用公共官方镜像缓存和本机 Go 模块缓存，生成候选镜像 `multisell-image-service:a8a46963`；镜像以非 root `app` 用户运行，包含 CA 证书，`/readyz` 冒烟检查通过。该结果为 `manually_verified`，镜像尚未切换到正式运行服务。
- 秘密事件：一次 Compose 配置读取错误地展开了数据库密码及两个图片服务秘密。发现后立即轮换数据库密码、图片服务共享秘密、执行令牌秘密及关联数据库连接串，重建数据库和后端容器；正式库仍为迁移 `111`，后端 `/api/ready` 通过。旧值不得继续使用。
- 审计链：生产库 1287 条审计记录的自身哈希全部正确，但现有触发器在并发插入时按最大 ID 选择链头，形成 46 个分叉和 48 个链尾。该结果不是字段被改写的证据，但现有 checkpoint 必须拒绝通过。
- 根因修复：新增迁移 `000152_operation_log_chain_concurrency`，先保存修复前每条记录的旧前驱哈希与记录哈希，再重建单链；后续触发器在事务锁内选择唯一链尾并在多链尾时失败关闭。服务与 checkpoint 改为验证自身哈希、根、缺失前驱、分叉、链尾和可达记录数。
- 隔离验证：v111 生产备份升级到 v152 后，1260 条历史记录状态为 `1260|0|1|0|0|1|1260`；回退至 v151、再次升级 v152 均通过；64 条并发写入后状态为 `64|0|1|0|0|1|64`。临时数据库已删除，正式库未迁移。

因此 P0-2 仍为 `in progress`：审计修复已有工程和隔离证据，但要等异地不可变备份门禁成立后才能迁移正式库。

## 15:40 补充：默认 Ozon 调度失败关闭

秘密轮换后的日志复核发现，当前正式运行镜像每次 `scheduler.tick.ozon_sync` 都会再次调用全局 adapter 初始化，触发 `platform adapter already registered: ozon` panic。EventBus 捕获了 panic，因此没有进入 Ozon 同步调用，但 `/api/ready` 仍返回成功，说明健康接口不能替代错误日志验收。

当前没有 Owner 选定的 Ozon 市场决定，也没有授权默认周期性平台读取。修复提交 `9239db40` 删除默认 Ozon 定时任务及订阅，保留 Ozon 连接器和受控入口；聚焦 HTTP/集成测试与 vet 通过，后端候选镜像已构建但尚未部署。正式运行镜像仍会出现该 panic，必须在 P0-2 正式切换后复核日志为零。

## 15:45 补充：公网边界与 Owner 会话只读验收

- SSH 与主机边界：`PasswordAuthentication=no`、`KbdInteractiveAuthentication=no`，root 仅允许密钥；UFW 为默认拒绝入站，只放行 22、80、443，fail2ban 正常运行。
- 服务暴露：PostgreSQL、后端和前端容器端口均未直接发布；从公网进行 HTTP/PostgreSQL 协议探测时，3000、5432、8080、9090 均无应用响应。80 只重定向至 443。
- TLS：443 使用受信任的 Let's Encrypt IP 证书，SAN 为当前生产 IP；响应包含 HSTS、CSP 和 `X-Frame-Options: DENY`。证书当前有效期至 2026-07-19，仍需依赖 Caddy 自动续期，尚未完成一次真实续期观察。
- 鉴权：未携带 Authorization 请求 `GET /api/v1/auth/extension-devices` 实际返回 401 `missing authorization header`。
- 已登录会话：现有浏览器会话可读取 `/settings/plugin` 的生产环境页和空设备列表，也可读取 `/settings/rbac` 的三个角色记录；全程未读取或输出令牌，未点击配对、创建、编辑或删除操作。数据库侧只有一个启用用户具有 Owner/Admin 角色，但浏览器会话与该用户的精确身份映射没有直接取证，保持 `inferred`，不得写成已确认。
- 界面真实性阻断：正式运行旧前端仍展示“3 Agents Online”、三项虚假运行任务、信任指数 85/127 次决策、商品 2,847、待发布 8 等无事实来源数字，并保留大量 Mock/Sandbox 导航。该界面会误导 Owner，P0-2 不得通过；候选发布前必须移除或显式标为不可用，并重新做浏览器验收。
- 版本差异：当前正式运行的 `/api/v1/platform-truth` 返回 404，说明线上仍是旧运行版本；这不是候选平台真相能力的生产验证。

以上只证明生产网络边界和部分鉴权行为实际存在，不证明正式系统已达到可用、可恢复或真实经营闭环。

## 16:05 补充：精确候选完成与并发生产发布偏差

- 合并候选：分支 `codex/real-operation-readiness` 已推进到 `14b3f91b`。验证过程中发现 `.gitignore` 的 `product-*/` 规则误排除了 `/product-opportunities` 页面，导致测试失败且构建路由缺失；现已显式放行并恢复页面。合并后 Go 全量测试 3504 项通过，前端 44 个测试文件、235 项测试通过，lint 为 0 error/8 个既有 warning，生产构建通过且包含 `/product-opportunities`。
- 图片服务运行镜像曾删除 CA 证书和 `wget`，会同时破坏 HTTPS Provider 证书校验基础及 Compose 健康检查；`14b3f91b` 恢复两者并使用与后端一致、服务器可达的 Alpine 包镜像。图片服务 `go test -race ./...` 114 项通过，`go vet ./...` 通过。
- 服务器候选镜像已生成但没有切换：`lingmirror-backend:14b3f91b`、`lingmirror-frontend:14b3f91b`、`multisell-image-service:14b3f91b`。图片服务临时容器 `/readyz` 返回 ready，运行 UID 为 100，CA trust store 存在，`wget` 可执行；临时容器已删除。
- 并发偏差：本轮期间另一发布流程把正式后端和前端切换到 `fe07479f`，并把正式库从迁移 111 升至 151。该提交不包含审计链 152 修复、Ozon 默认调度关闭或小Q单入口改造。正式日志已再次出现 `scheduler.tick.ozon_sync` 的 adapter 重复注册 panic。
- 当前审计链：迁移 151 的正式库有 1330 条记录、1 个根、47 个分叉点、49 个链尾；因此审计 checkpoint 明确失败。迁移 152 仍只在隔离恢复库验证，尚未用于正式库。
- 并发发布前备份 `multisell_pre_l3_20260713T072216Z.dump` 的哈希与 `pg_restore --list` 均通过，备份内迁移版本为 111。发现该备份及校验文件权限为 644 后已立即收紧为 600、root-only。它仍然只在本机，不满足异地不可变要求。
- 当前正式容器没有运行 image-service。后端、前端和数据库容器显示 healthy，但这不能覆盖审计分叉、Ozon panic、图片服务缺失和异地恢复门禁缺失。

因此不得继续把当前正式状态表述为 P0-2 完成，也不得在没有异地不可变备份和真实告警确认的情况下把 `14b3f91b` 切换为正式运行版本。

## 16:28 补充：唯一整合候选与 v151→v153 隔离演练

- 唯一候选：`19bfb84f` 同时是真实运行候选 `1886f30e` 与当前生产插件 L3 `fe07479f` 的后代，避免直接部署旧候选造成插件回退。分支为 `codex/real-operation-readiness`。
- 候选补强：Slack 告警已要求发送 firing 与 resolved；两个 systemd service 的工作目录修正为 `/opt/multisell`；应用回滚覆盖 image-service，并禁止把主库 migration 152 降回已知分叉链；正式导航移除 Mock/Sandbox/壳入口；固定成功的 mock carrier 只在 development 注册；物流写操作改用只授予启用 Owner/Admin 的 `shipping.write`，新增 migration 153。
- 全量工程验证：后端 122 个包、3512 项测试通过，`go vet ./...`、`go build ./...` 通过；前端 44 个文件、238 项测试通过，79 页生产构建通过，lint 为 0 error/7 个既有 warning；图片服务 race 测试 114 项和 vet 通过；1688 插件 59 项、相关前端 26 项及后端聚焦测试通过；备份、审计 checkpoint、告警、监控、回滚合同测试通过。
- 精确镜像：服务器已生成 `lingmirror-backend:19bfb84f`、`lingmirror-frontend:19bfb84f`、`multisell-image-service:19bfb84f`，尚未切换正式容器。
- 当前库备份：正式 v151 生成本机备份 `multisell_v151_candidate_20260713T082650Z.dump`，SHA-256 与 `pg_restore --list` 通过，权限为 600/root-only。该备份仍非异地不可变，不满足发布门禁。
- 隔离迁移：从该 v151 备份恢复临时数据库，顺序执行 candidate migration 152 和 153 后为 `153|false`；审计状态 `1352|0|1|0|0|1|1352`，repair snapshot 为 1352 条。临时数据库和容器内临时文件已删除。
- 正式库未改：演练结束后正式库仍为 `151|false`，当前 1370 条审计记录、1 个根、47 个分叉点、49 个链尾。行数增加说明线上仍有活动，但分叉/链尾形态未在本次演练中改变。

`IMAGE_TAG=19bfb84f ./scripts/verify_prod_compose.sh` 的生产门禁仍应失败，直到真实 S3 Object Lock 异地上传/保留/恢复和真实告警送达全部完成。候选完成不等于 P0-2 完成。

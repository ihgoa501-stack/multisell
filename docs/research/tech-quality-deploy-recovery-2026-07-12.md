# 部署、备份与恢复基础技术质量审计（2026-07-12）

## 范围与裁决

范围：生产 Compose/Caddy、唯一部署 runbook、健康与就绪、备份、恢复、迁移验证、回滚、监控告警及相关脚本。

结论：仓库已经具备比普通早期项目更完整的运维基础：生产端口边界、release 配置、不可变异地备份要求、恢复验证、迁移全生命周期、回滚前备份、health/readiness 和失败通知都有实现与契约测试。同日后续已用本机当前数据库完成一次隔离备份恢复，但没有实际完成全新服务器部署、生产备份恢复、失败回滚或告警送达演练，因此只能评为 **工程基础中上，本机恢复链已验证，生产灾难恢复能力仍未被证明**。

初次审计实际通过：backup contract、audit checkpoint contract、ops failure webhook contract、rollback safety contract、production compose boundary、private compose boundary，共 6 个脚本验证。同日 Unit 8 补充验证见文末；没有执行生产 SSH、外部 S3 或真实回滚。

## 什么叫开发得好 / 不好

开发得好：只有一个权威部署入口；生产不会暴露数据库/后端端口；配置缺失时启动失败；备份可校验、异地、不可变且定期实际恢复；迁移可从零上行、全量下行并再次上行；回滚不覆盖现场、不盲目回滚数据库；health 与 readiness 区分；故障能通知 Owner，并有明确 RPO/RTO 和演练记录。

开发得不好：有脚本但从未恢复；默认弱密码可能进入生产；旧验证脚本使用过期端口或凭据；回滚中途失败后没有清晰恢复点；只检查容器启动不检查依赖 readiness；监控存在但没有告警送达证据。

## 七轴评分

| 轴 | 分数（0-5） | 依据 |
|---|---:|---|
| 正确性 | 3.7 | 契约脚本合理；真实部署/恢复/回滚本次未执行 |
| 可读性与复杂度 | 3.8 | runbook 和脚本步骤清楚，危险操作有显式提示 |
| 架构边界 | 4.0 | 唯一 runbook、Caddy 唯一入口、服务内网隔离 |
| 安全 | 3.6 | 生产关闭注册/Swagger、显式 CORS、不可变异地备份；通用脚本仍有弱默认值 |
| 性能与容量 | 3.0 | 有监控，未见基于真实数据量的备份窗口/恢复时间验证 |
| 测试质量 | 3.7 | 6 个契约验证通过；关键真实恢复仍缺本次证据 |
| 可运维性 | 3.8 | readiness、告警、备份、回滚与 systemd 定时任务齐；演练状态未知 |
| **总评** | **3.7/5** | **基础较完整，但未达到经演练证明的生产恢复等级** |

## 已做得好的基础

- `verify_prod_compose.sh` 渲染最终 Compose，确认只有 Caddy 发布 80/443，backend/frontend 不继承开发挂载或命令。
- 生产强制 release、显式 CORS、metrics、关闭公开注册与 Swagger；backup/audit checkpoint 强制异地不可变存储。
- `backup.sh` 使用临时 partial 文件、`pg_restore --list` 和 SHA-256，上传后检查远端对象；可要求 S3 versioning + Object Lock。
- `verify_backup_restore.sh` 只允许安全命名的隔离验证库，实际 restore 后检查公共表和关键表，并自动清理。
- `verify_migrations_full.sh` 拒绝非专用验证库，执行 up → down all → up。
- `rollback.sh` 拒绝有已跟踪改动的工作区，回滚前强制备份，目标版本必须重新证明网络边界，最后同时检查 health/readiness。
- 服务关闭顺序先 fail readiness、排空 HTTP，再停 Scheduler 和 EventBus。

## 关键发现

### P1：生产恢复能力仍没有生产演练证据

契约测试不能证明生产备份能恢复、生产 PostgreSQL 版本兼容、所有迁移/扩展/权限完整、恢复耗时满足 Owner 可接受范围。同日后续已验证本机当前数据库备份，但生产状态仍为 `unknown`。

验证：使用最新真实备份执行 `verify_backup_restore.sh`，记录文件哈希、开始/完成时间、表数量、关键行数抽样和应用 readiness；演练结果形成带日期记录。

### P1：通用备份/恢复脚本包含 `postgres/postgres` 默认凭据

`backup.sh` 与 `restore.sh` 在未传环境变量时默认 `DB_USER=postgres`、`DB_PASSWORD=postgres`。生产 Compose 可以覆盖且验证器检查部分配置，但直接人工运行脚本时可能误连错误数据库或依赖弱默认值。证据：`scripts/backup.sh:8-13`、`scripts/restore.sh:28-33`。

修复：生产/恢复模式要求显式凭据，或至少当 `SERVER_MODE=release`、offsite required、远端 host 时拒绝默认值；示例值只留在 `.example`。

### P1：回滚流程缺少中途失败后的自动恢复策略

脚本在强制备份后切到 detached 目标、验证、构建、可选 down migration、再启动。若目标验证、构建或启动失败，会保留 detached 目标和部分停止状态，只打印现场；这有利于取证，但 Owner 需要一条明确且已演练的“恢复原 commit/恢复数据库备份”路径。尤其执行 `--revert-migration` 后，应用启动失败不能简单切回代码。

修复：runbook 明确每一步失败后的裁决表；自动记录原 commit、备份路径、迁移前版本；不要自动做第二次破坏性回滚，但给出可复制、带检查的恢复命令并演练。

### P2：遗留 `verify_page.py` 使用过期端口和历史默认凭据

`scripts/verify_page.py` 仍调用 8081/3001 和 `admin/admin123456`，与当前项目病历明确的凭据状态冲突。即使它不属于权威 runbook，也会误导未来 Agent，把失败误判为系统故障或诱导恢复弱凭据。

修复：标记 deprecated 并从文档入口移除，或改成显式 URL/凭据环境变量且默认拒绝运行；删除需要 Owner 单独授权，本轮不处理。

### P2：备份保留清理发生在异地上传前（已修复）

`backup.sh` 原先在异地上传前清理旧恢复点。同日后续已把 retention 移到必需异地上传及远端校验成功之后；上传失败会保留旧本地恢复点。`scripts/tests/test_backup.sh` 回归通过。

## Unit 8 同日补充工程核验（2026-07-12 22:55 CST）

| 声明 | 证据等级 |
|---|---|
| production/private/IP 三套 Compose 强制验证器已接入图片服务所需密钥与数据库连接，能够重新渲染；production 额外验证 image-service 不发布端口、使用 production + PostgreSQL 持久 Job Store，并核对共享密钥接线 | `implemented / automated_verified`：三个验证脚本通过 |
| Prometheus、Grafana 只绑定 loopback，Alertmanager 不发布主机端口，backend metrics 开启，三项监控服务均有 healthcheck | `implemented / automated_verified`：新增 `verify_monitoring_compose.sh` 并通过 |
| 当前本机 `multisell` 可生成约 12 MB custom-format 备份并在一次性隔离数据库恢复；恢复后发现 107 张 public 表，`user / operation_log / schema_migrations` 存在，验证库自动删除 | `manually_verified`（本机隔离 PostgreSQL）；不代表生产备份或生产 RTO |
| 隔离恢复和破坏性恢复默认要求相邻 `.sha256`；缺失时失败关闭，历史可信归档必须先独立核验再显式 override | `implemented / automated_verified`：真实本机备份的 checksum + 隔离恢复通过 |
| 异地上传失败时不再先删除旧本地恢复点；retention 只在远端上传及 Object Lock 元数据检查成功后执行 | `implemented / automated_verified`：backup contract 通过 |
| rollback safety contract | `automated_verified` |
| Prometheus/Alertmanager 配置语义、真实告警 firing→resolved 到 Owner、外部 S3 Versioning/Object Lock、正式服务器端口及 SSH、生产部署/恢复/回滚 | `unknown / not_verified`：本机 Docker daemon 未运行，且本任务按授权未触碰外部生产环境 |

### P2：容量、RPO/RTO 和告警送达没有本次运行证据

配置和规则存在不等于告警实际到达。当前没有本次真实备份大小、耗时、恢复耗时、可接受数据损失窗口和 Alertmanager 到 Owner 的测试记录。

## 最小验证顺序

1. 在隔离 PostgreSQL 对最新真实 backup 运行一次完整恢复验证，记录 RPO/RTO 数据。
2. 建立空专用库运行迁移 up → down all → up，并核对最终版本。
3. 在非生产演练环境执行一次应用版本回滚，不回滚数据库；再演练一次迁移不兼容场景的人工恢复步骤。
4. 发送一条测试告警，确认 Owner 实际收到；记录时间和渠道。
5. 修复弱默认凭据、备份清理顺序和遗留验证脚本。

在上述验证完成前，可以说“运维机制已实现且契约测试通过”，不能说“灾难恢复已经可靠”或“生产部署已经验证”。

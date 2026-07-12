# 凌镜数据库备份与恢复政策

> 适用范围：Owner 单人自用生产 PostgreSQL 15
> 当前证据日期：2026-07-12

## 目标与证据边界

备份成功必须同时满足：生成 PostgreSQL custom archive、`pg_restore --list` 通过、生成 SHA-256、按生产政策上传加密异地副本并验证远端对象存在。只有文件存在或非空不算成功。

恢复成功必须恢复到隔离数据库，并验证 public 表数量、`user`、`operation_log`、`schema_migrations` 等关键表。生产恢复仍需要 Owner 明确批准。

当前建议目标（待 Owner 确认）：

- RPO（最多可接受数据损失）：24 小时；
- RTO（目标恢复时间）：4 小时；
- 本地保留：7 天；
- 加密异地保留：30 天，由对象存储生命周期策略执行；
- 每日备份；每月至少一次隔离恢复演练；每次高风险迁移前额外备份。

在 Owner 确认前，RPO/RTO 是 `proposed policy`，不是已经达成的服务承诺。

## 自动备份

生产 backup profile 使用包含 PostgreSQL client 和 AWS CLI 的专用镜像：

```bash
docker compose --profile manual \
  -f docker-compose.yml -f docker-compose.prod.yml \
  run --rm backup
```

生产默认 `BACKUP_REQUIRE_OFFSITE=true` 且 `BACKUP_REQUIRE_IMMUTABLE_OFFSITE=true`。bucket 缺失、AWS CLI 缺失、Versioning 未启用、Bucket Object Lock 没有默认保留策略、上传失败、远端对象不存在或对象没有实际 retain-until 元数据，都会让任务以非零状态退出。

本地开发可明确设置：

```bash
BACKUP_REQUIRE_OFFSITE=false ./scripts/backup.sh
```

输出文件：

```text
multisell_YYYY-MM-DD_HHMMSS.dump
multisell_YYYY-MM-DD_HHMMSS.dump.sha256
```

不得在日志、文件名或交付报告中输出数据库密码、AWS 密钥或 JWT。

生产宿主机定时入口位于 `ops/systemd/`：

- `lingmirror-backup.timer`：每日执行已验证备份；
- `lingmirror-audit-checkpoint.timer`：每小时验证审计链并生成 HMAC checkpoint；
- 两个 oneshot 失败时触发 `lingmirror-ops-failure@.service`，通过 `OPS_ALERT_WEBHOOK_URL` 通知。

安装时复制 unit 到 `/etc/systemd/system/`，把仅 root 可读的配置写入 `/etc/lingmirror/ops.env`，执行 `systemctl daemon-reload` 后启用两个 timer。安装和启用属于生产状态变更，仍须按部署 runbook 获得 Owner 批准。

审计 checkpoint 的 `AUDIT_CHECKPOINT_KEY` 必须与数据库密码、JWT 和加密密钥不同；生产默认 `AUDIT_CHECKPOINT_REQUIRE_OFFSITE=true` 和 `AUDIT_CHECKPOINT_REQUIRE_IMMUTABLE_OFFSITE=true`。脚本会验证目标 bucket 的 Versioning、默认 Object Lock 保留策略以及上传对象实际的 retain-until 元数据。本地 HMAC 文件或关闭不可变要求的普通 S3 上传只能证明脚本运行，不能证明对象不可删除。

## 隔离恢复验证

```bash
BACKUP_FILE=/path/to/multisell_YYYY-MM-DD_HHMMSS.dump
./scripts/verify_backup_restore.sh "$BACKUP_FILE"
```

该脚本只创建随机命名的临时验证数据库，恢复并检查后自动删除。不得把 `RESTORE_VERIFY_DB` 设置为 `multisell` 或其他正式数据库名称。

CI 对迁移后的测试数据库执行完整的“备份 → 隔离恢复 → 关键表验证”。这证明代码路径和 CI PostgreSQL 环境可恢复，不证明生产备份新鲜度或异地副本存在。

## 生产恢复

生产恢复是破坏性操作，只能按 `docs/ops/OWNER_AND_AI_DEPLOYMENT_RUNBOOK.md` 执行，并满足：

1. Owner 明确批准恢复点和允许的数据损失；
2. SHA-256 与 archive list 验证通过；
3. 先在隔离数据库恢复并核验；
4. 正式恢复前保留当前数据库快照；
5. 恢复后运行待执行迁移、readiness、登录和关键读取验证；
6. 记录实际恢复耗时、恢复点、验证结果和任何数据缺口。

## 告警和审计

以下情况必须让备份任务失败，并最终触发 Owner 告警：

- 超过计划时间没有新备份；
- archive 或 SHA-256 校验失败；
- 异地上传或远端对象检查失败；
- 隔离恢复失败；
- 本地或异地保留策略异常；
- 磁盘空间不足。

systemd timer、失败 webhook 和生产 S3 当前仍为 `implemented` 但未部署，实际启用与送达为 `unknown`，不得写成已经自动执行。

## 当前验证事实

- `automated_verified`：`scripts/tests/test_backup.sh` 覆盖 archive 校验、异地必需配置、S3 调用和失败阻断。
- `manually_verified`（2026-07-12，本地 PostgreSQL）：从 `multisell` 生成约 12 MB archive，恢复到隔离数据库并验证 107 张 public 表及三个关键表，验证库随后自动删除。
- `unknown`：生产定时任务、真实 S3 加密副本、对象生命周期、Owner 告警、生产 RPO/RTO 和生产恢复耗时。
- `manually_verified`（2026-07-12，本地隔离 PostgreSQL）：审计 checkpoint 生成并包含 64 位 HMAC；要求异地但缺少 S3 URI 时任务非零退出，验证库随后删除。

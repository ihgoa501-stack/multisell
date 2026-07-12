# LingMirror 数据库迁移 Runbook

> 当前活跃栈：Go/Gin + PostgreSQL。历史 Python 数据迁移说明仅适用于明确存在 `legacy_*` 表的专项迁移，不是当前生产升级的默认路径。

## 当前强制门

提交前必须在专用空库执行：

```bash
DATABASE_URL='postgresql://.../lingmirror_migration_verify?...' \
  ./scripts/verify_migrations_full.sh
```

脚本只允许数据库名以 `lingmirror_migration_verify` 开头，并验证：

1. 每个 up 都有 down；
2. 没有重复版本；
3. 空库全量 up；
4. 最新 down/up；
5. 全量 down，仅允许迁移工具自己的 `schema_migrations` 留存；
6. 再次全量 up 并达到最新版本。

生产迁移前必须完成备份及隔离恢复验证。migrate 返回 dirty、版本异常或 SQL 失败时，应用不得接收流量。

## Dirty 数据库处理

禁止直接执行：

```text
migrate force <猜测版本>
```

`force` 只改迁移账本，不创建缺失表、列、索引或约束。处理步骤必须是：

1. 对当前库做 custom archive 和隔离恢复；
2. 在隔离副本读取 `schema_migrations`；
3. 与同提交全新迁移库比较表、列、索引和约束；
4. 在隔离副本验证 forward repair，或建立最新空库并进行显式数据映射迁移；
5. 核对关键表行数、外键、Owner 登录和关键读取；
6. 形成独立方案、回滚点和 Owner 批准后，才允许处理正式库。

2026-07-12 本地只读审计发现：`multisell` 为 `version=1, dirty=true`，当前 107 张 public 表，而全新 version 91 参考库有 176 张；隔离副本 `force 1 → up` 在 migration 2 因既有索引冲突而安全失败。因此当前本地库不得 force 或直接补跑，正式修复状态为 `unknown / requires_owner_approval`。

## 历史 legacy 数据迁移

> 适用于：从旧 Python/FastAPI + PostgreSQL 栈迁移到新 Go/Gin + PostgreSQL 栈

## 前置条件

1. 新 schema 已 apply：`migrate -path backend-go/migrations -database "$DATABASE_URL" up`
2. 旧库的表已导出并导入新库，命名为 `legacy_<原表名>`（例如 `legacy_sku`、`legacy_sales_order`）
3. 已完成数据库全量备份：`pg_dump multisell > backup_$(date +%Y%m%d_%H%M).sql`

## 迁移步骤

### 1. 预检查（dry run，无副作用）

```bash
psql -d multisell -f backend-go/migrations/validate.sql
```

预期：所有表 `legacy_count` 与 `new_count` 都为 0（新表空，旧表有数据）。

### 2. 执行数据迁移

```bash
psql -d multisell -f backend-go/migrations/000003_data_migration.up.sql
```

脚本特点：
- 整体在单个事务里，失败回滚
- `INSERT ... ON CONFLICT (id) DO NOTHING`，可重入
- 按 FK 依赖顺序：reference 表 → product → sku → order → settlement → finance → 其他

### 3. 验证

```bash
psql -d multisell -f backend-go/migrations/validate.sql
```

通过条件：
- **行数 parity**：所有表 `status = OK`，无 `MISMATCH`
- **Checksum**：5 个关键表（sku/sales_order/settlement/finance_ledger_entry/product_listing）前 100 个 ID 的 md5 一致
- **FK 完整性**：4 项 orphan check 全部 `orphan_count = 0`

### 4. 回滚（如验证失败）

```bash
psql -d multisell -f backend-go/migrations/000003_data_migration.down.sql
# 或从备份恢复
psql -d multisell < backup_YYYYMMDD_HHMM.sql
```

## 切流日 Runbook

### T-2h：预演
1. 在 staging 环境完整跑一遍上述 4 步
2. 记录每步耗时（基线）

### T-0：正式迁移
1. 通知所有用户只读模式
2. `pg_dump` 生产库
3. apply 000001 + 000002 + 000003
4. 跑 validate.sql，确认通过
5. 切流量到 Go 后端
6. 保留 Python/Vue 热备 72h

### T+72h：清理
1. 确认无回滚需求
2. `DROP TABLE legacy_*`
3. 通知团队迁移完成

## 故障排查

| 现象 | 原因 | 处理 |
|---|---|---|
| `MISMATCH: sku` | 旧表有重复 id（罕见）或新表已有种子数据 | 先 `TRUNCATE` 新表再重跑 |
| `CHECKSUM MISMATCH` | 迁移顺序问题导致部分行未导入 | 检查 000003 日志，按报错表手动补 |
| `sales_order_item orphans` | order_id 在新 sales_order 表不存在 | 检查 sales_order 迁移是否完整 |
| `INSERT ... ON CONFLICT` 报错 | 新表没有 primary key 约束 | 检查 000001_init_schema 是否 apply 成功 |

## 关键说明

- **旧表命名**：本 runbook 假设旧表导出时重命名为 `legacy_<原名>`，避免与新表冲突。如果直接用同名导入，需要先 `ALTER TABLE <name> RENAME TO legacy_<name>`
- **不迁移的表**：`ai_trace`、`ai_trace_event`、`ai_evidence_ref`、`unified_action` 是新表，无历史数据，不迁移
- **字段映射**：000003 的 INSERT 列表严格对齐旧 Python models 的字段名。如果旧库有 schema drift（手动加过列），需要先在 000001 里补列
- **大表分批**：如果 `sales_order` > 100 万行，建议改成 `INSERT ... SELECT ... WHERE id BETWEEN ? AND ?` 分批迁移，本脚本未做此优化

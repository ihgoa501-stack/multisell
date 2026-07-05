> ⚠️ 历史计划文档。引用已删除的旧栈，仅供参考。

# LingMirror 生产切流 Runbook

> 版本：v1.0 | 最后更新：2026-06-23 | 适用：从 Python/FastAPI + Vue → Go/Gin + Next.js 全栈切换

## 角色与联系

| 角色 | 职责 | 联系方式（填） |
|---|---|---|
| 切流指挥 | 决策 T-0 / Rollback / T+72h | ________ |
| 后端值班 | 监控 Go 服务 5xx / p95 / WS | ________ |
| 前端值班 | 监控 Next.js 构建 + CDN | ________ |
| DBA | 迁移执行 + 行数校验 + 回滚 | ________ |
| QA | E2E + 冒烟测试 | ________ |

## 切流前检查清单（T-1 day）

- [ ] 代码已全部提交并合并到 main
- [ ] `go build ./...` + `go test ./...` 通过
- [ ] `npm run build` 通过
- [ ] 数据库全量备份完成：`pg_dump multisell > backup_pre_cutover.sql`
- [ ] staging 环境完整跑过一次切流演练
- [ ] k6 压测 5 个场景达标（p95 < 500ms, error < 5%）
- [ ] Playwright E2E 主链路通过
- [ ] 25 条 high 路由 parity 已补齐
- [ ] LLM provider 配置确认（LLM_PROVIDER / LLM_API_KEY）
- [ ] 回滚脚本验证可用

## T-2h：预演

```bash
# 1. 在 staging 复刻生产数据
pg_dump multisell_prod | psql multisell_staging

# 2. Apply 新 schema
migrate -path backend-go/migrations -database "$STAGING_DB" up

# 3. 跑数据迁移
psql -d multisell_staging -f backend-go/migrations/000003_data_migration.up.sql

# 4. 验证
psql -d multisell_staging -f backend-go/migrations/validate.sql
# 预期：0 MISMATCH，0 orphan，5 checksum OK

# 5. 启动 Go 后端 + Next 前端
cd backend-go && go run cmd/server/main.go &
cd frontend-next && npm run start &

# 6. 跑 E2E
cd frontend-next/e2e && npx playwright test

# 7. 跑压测
cd backend-go/loadtest && ./run-all.sh
```

**预演通过条件**：迁移 0 错误 + E2E 全绿 + 压测达标。任一失败 → 修复后重新预演。

## T-0：正式切流

### 阶段 1：只读模式（T-0, 0:00）

1. 通知所有用户："系统维护中，只读模式"（前端显示 banner）
2. 旧后端切换到只读模式（关闭所有 POST/PUT/DELETE 端点）
3. 确认无新写入：
   ```sql
   SELECT COUNT(*) FROM sales_order WHERE created_at > NOW() - INTERVAL '5 minutes';
   -- 预期：0
   ```

### 阶段 2：备份 + 迁移（T-0, 0:05）

```bash
# 1. 最终备份
pg_dump multisell_prod > backup_T0_$(date +%Y%m%d_%H%M).sql
md5sum backup_T0_*.sql  # 记录校验和

# 2. Apply 新 schema（如果还没 apply）
migrate -path backend-go/migrations -database "$PROD_DB" up

# 3. 执行数据迁移
psql -d multisell_prod -f backend-go/migrations/000003_data_migration.up.sql

# 4. 验证（关键！）
psql -d multisell_prod -f backend-go/migrations/validate.sql
```

**验证通过条件**（Hard Gates）：
- 行数 parity：所有表 `status = OK`，0 MISMATCH
- Checksum：5 个关键表 md5 一致
- FK 完整性：4 项 orphan check 全部 = 0

**任一失败 → 立即回滚**：
```bash
psql -d multisell_prod -f backend-go/migrations/000003_data_migration.down.sql
# 或从备份恢复
psql -d multisell_prod < backup_T0_YYYYMMDD_HHMM.sql
```

### 阶段 3：切流量（T-0, 0:20）

```bash
# 1. 启动 Go 后端
cd backend-go && go run cmd/server/main.go &

# 2. 启动 Next 前端
cd frontend-next && npm run start &

# 3. 健康检查
curl http://localhost:8080/api/health
# 预期：{"status":"ok","version":"0.1.0"}

# 4. 切 Nginx/网关：旧 :8000 (Python) → 新 :8080 (Go)
# 5. 切前端 CDN：旧 Vue dist → 新 Next build
```

### 阶段 4：内部验证（T-0, 0:25）

1. 内部用户（5 人）访问新系统，执行核心操作：
   - 登录
   - 查看 dashboard
   - 搜索 SKU
   - 查看 /ai 指挥中心
   - 审批一个 action
2. QA 跑 Playwright E2E

**内部验证通过 → 进入阶段 5。失败 → 回滚。**

### 阶段 5：全量切流（T-0, 0:40）

1. 通知所有用户："系统已恢复"
2. 移除维护 banner
3. 保持旧 Python/Vue 服务 **热备**（不关，但无流量）

## 监控指标（T-0 → T+72h）

| 指标 | 阈值 | 处理 |
|---|---|---|
| API 5xx 率 | > 1% | 告警，排查日志 |
| API p95 延迟 | > 500ms | 告警，检查慢查询 |
| WebSocket 断开率 | > 5%/min | 告警，检查 hub |
| Action 执行失败率 | > 10% | 告警，检查 unified_action |
| Trace gap（无 event 的 trace） | > 5% | 告警，检查 ai_trace_event |
| 数据不一致（订单数 vs 旧库） | > 0 | **立即回滚** |
| CPU > 80% 持续 5min | — | 扩容或排查 |

### 监控命令

```bash
# 实时 5xx 监控
tail -f /var/log/nginx/access.log | grep ' 5[0-9][0-9] '

# p95 延迟（需 Go pprof 或 APM）
curl http://localhost:8080/api/health

# WS 连接数
ss -tnp | grep :8080 | wc -l

# Trace gap 检查
psql -d multisell_prod -c "
  SELECT COUNT(*) FROM ai_trace t
  WHERE t.status = 'completed'
  AND NOT EXISTS (SELECT 1 FROM ai_trace_event e WHERE e.trace_id = t.trace_id)
"
```

## 回滚决策树

```
T-0 ~ T+1h 任何阶段：
  ├─ 数据迁移验证失败 → 立即回滚（恢复备份）
  ├─ 内部验证失败（E2E 红 / 核心功能不可用）→ 立即回滚
  └─ 监控指标超阈值 → 评估严重度
       ├─ 数据不一致 → 立即回滚
       ├─ 5xx > 5% → 15min 内修复，否则回滚
       └─ p95 > 1s → 30min 内修复，否则回滚

T+1h ~ T+72h：
  ├─ 任何 P0 bug → 评估影响面
  │    ├─ 可热修 → 修复部署
  │    └─ 不可热修 → 回滚
  └─ 数据不一致 → 立即回滚

T+72h 后：
  ├─ 稳定 → 下线旧服务（DROP legacy_* 表）
  └─ 不稳定 → 延长观察期
```

### 回滚步骤

```bash
# 1. 切 Nginx 回旧服务
# 2. 恢复数据库（如果迁移有数据损坏）
psql -d multisell_prod < backup_T0_YYYYMMDD_HHMM.sql
# 3. 旧 Python/Vue 服务仍在运行，直接恢复流量
# 4. 通知用户"已回滚到旧版本"
# 5. Post-mortem 分析
```

## T+72h：清理

```bash
# 1. 确认 72h 无回滚、无 P0
# 2. 下线旧 Python 后端
systemctl stop multisell-python
# 3. 下线旧 Vue 前端
rm -rf /var/www/multisell-vue-old
# 4. 清理 legacy 表
psql -d multisell_prod <<'SQL'
DROP TABLE IF EXISTS legacy_category, legacy_brand, legacy_product, legacy_sku,
  legacy_sales_order, legacy_settlement, legacy_finance_ledger_entry, ... CASCADE;
SQL
# 5. 通知团队：迁移完成，旧栈已下线
```

## 附录：关键文件

| 文件 | 用途 |
|---|---|
| `backend-go/migrations/000001_init_schema.up.sql` | 新 schema DDL（80 表） |
| `backend-go/migrations/000002_ai_tables.up.sql` | AI 表 DDL |
| `backend-go/migrations/000003_data_migration.up.sql` | 数据迁移脚本 |
| `backend-go/migrations/validate.sql` | 迁移验证脚本 |
| `backend-go/migrations/000003_data_migration.down.sql` | 回滚迁移（TRUNCATE） |
| `backend-go/loadtest/` | k6 压测脚本 |
| `frontend-next/e2e/` | Playwright E2E |
| `docs/superpowers/plans/2026-06-23-parity-report.md` | 路由 parity 报告 |

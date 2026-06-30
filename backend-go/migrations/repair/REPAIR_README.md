# LingMirror DB Migration Repair Plan

**Date:** 2026-06-29
**Container:** `deploy-db-1` (PostgreSQL 15)
**Database:** multisell
**Target files:** `backend-go/migrations/repair/forward_repair.up.sql` and `.down.sql`

---

## 1. Diagnosis Summary

The production database has **no `schema_migrations` table**, so no migration tracking exists. By cross-referencing 90 production tables and columns against migration files:

| Range | Status |
|-------|--------|
| 000001 - 000014 | Confirmed applied |
| 000015 (metabolism) | **NOT applied** |
| 000016 | **Missing from filesystem** (no file exists) |
| 000017 - 000031 | **NOT applied** |

---

## 2. Issues Found

### 2a. Duplicate version numbers (must coexist)

| Version | File A | File B |
|---------|--------|--------|
| 26 | `000026_data_freshness.up.sql` | `000026_tariff_rule.up.sql` |
| 28 | `000028_landed_cost.up.sql` | `000028_sku_return_stats.up.sql` |

Both files are **distinct migrations** (different tables, different purposes). They happened to be committed with the same version prefix. The repair applies both under each version block.

### 2b. 000029 and 000030 are identical

`000029_orchestration.up.sql` and `000030_orchestration.up.sql` have **identical content** (both create `lifecycle_step` and `orchestration_config`). Key difference:

- `000030` has a proper **down.sql** (134 bytes)
- `000029` has **no down.sql**

The repair uses **000030 as the canonical version** and skips 000029 entirely.

### 2c. Migrations missing IF NOT EXISTS guards

These original files lack `IF NOT EXISTS` on CREATE TABLE, which the repair adds for idempotency:

| File | Issue |
|------|-------|
| `000017_create_llm_cost_logs.up.sql` | `CREATE TABLE llm_cost_logs` — no guard |
| `000025_supply_chain_flow.up.sql` | `CREATE TABLE supply_chain_flow` — no guard |
| `000026_tariff_rule.up.sql` | `CREATE TABLE tariff_rule` — no guard |

All indexes across all files also lacked `IF NOT EXISTS`; the repair adds it to every `CREATE INDEX`.

---

## 3. Execution Order

### Step 1: Take a full database backup

```bash
docker exec deploy-db-1 pg_dump -U postgres multisell > \
  /tmp/multisell_backup_$(date +%Y%m%d_%H%M%S).sql
```

### Step 2: Copy repair SQL to the server

```bash
# From local dev machine to production server
scp backend-go/migrations/repair/forward_repair.up.sql \
  root@118.196.42.156:/tmp/forward_repair.up.sql
```

### Step 3: Apply the forward repair

```bash
# On the production server:
docker exec -i deploy-db-1 psql -U postgres -d multisell \
  -f /tmp/forward_repair.up.sql
```

The entire file is wrapped in a `BEGIN` / `COMMIT` transaction. If any statement fails, everything rolls back automatically.

### Step 4: Verify

```bash
# On the production server, check new tables exist:
docker exec deploy-db-1 psql -U postgres -d multisell -c "
SELECT table_name FROM information_schema.tables
WHERE table_schema = 'public'
  AND table_name IN (
    'metabolism_log', 'llm_cost_logs', 'approval_request',
    'product_relation', 'product_version', 'supplier_score',
    'decision_evaluation', 'agent_accuracy', 'webhook_event_log',
    'inventory_oversell_log', 'supply_chain_flow', 'data_freshness',
    'tariff_rule', 'supply_chain_tracking', 'landed_cost',
    'sku_return_stats', 'lifecycle_step', 'orchestration_config',
    'product_sentiment'
  )
ORDER BY table_name;
"

# Check schema_migrations tracking was created:
docker exec deploy-db-1 psql -U postgres -d multisell -c "
SELECT * FROM schema_migrations;
"
```

Expected output: 19 new tables + `schema_migrations` showing `version=31, dirty=false`.

### Step 5: Verify existing data is intact

```bash
docker exec deploy-db-1 psql -U postgres -d multisell -c "
SELECT 'product' AS tbl, COUNT(*) AS cnt FROM product
UNION ALL
SELECT 'sku', COUNT(*) FROM sku
UNION ALL
SELECT 'sales_order', COUNT(*) FROM order_module.sales_order;
"
```

### Rollback (if needed)

```bash
docker exec -i deploy-db-1 psql -U postgres -d multisell \
  -f /tmp/forward_repair.down.sql
```

---

## 4. SQL Reference

### 4a. What `forward_repair.up.sql` applies (in order)

| Section | Migration | Creates |
|---------|-----------|---------|
| 1 | 000015 | `metabolism_log` table + `event_outbox.excreted_at` / `excretion_reason` columns |
| 2 | 000017 | `llm_cost_logs` table + 3 indexes |
| 3 | 000018 | `approval_request` table + 5 indexes |
| 4 | 000019 | `product_relation` table + 4 indexes (unique pair constraint) |
| 5 | 000020 | `product_version` table + 2 indexes |
| 6 | 000021 | `supplier_score` table + 2 indexes (unique supplier_id) |
| 7 | 000022 | `decision_evaluation` + `agent_accuracy` tables + 6 indexes |
| 8 | 000023 | `webhook_event_log` table |
| 9 | 000024 | `inventory_oversell_log` table + 3 indexes |
| 10 | 000025 | `supply_chain_flow` table (prerequisite for 000027) |
| 11 | 000026a | `data_freshness` table + 4 indexes |
| 12 | 000026b | `tariff_rule` table + 3 indexes |
| 13 | 000027 | `supply_chain_tracking` table (FK to supply_chain_flow) + 5 indexes |
| 14 | 000028a | `landed_cost` table + 2 indexes |
| 15 | 000028b | `sku_return_stats` table |
| 16 | 000030 | `lifecycle_step` + `orchestration_config` tables + 2 indexes |
| 17 | 000031 | `product_sentiment` table + 2 indexes |
| -- | tracking | `schema_migrations` with version=31, dirty=false |

### 4b. What `forward_repair.down.sql` reverts

Drops all 19 tables (in FK-safe reverse order), drops the `event_outbox` columns, and removes the `schema_migrations` table.

---

## 5. Recommended Post-Repair: Fix Migration Tooling

After the repair, fix the filesystem so `golang-migrate` can work forward from here:

### Step A: Create a tracking table entry for the skipped/incomplete versions

If golang-migrate is ever used, it will fail at version 26 (two files) and version 28 (two files). The cleanest fix:

1. Rename the duplicate files to unique version numbers:
   ```bash
   # Rename 000026b → 000032 (tariff_rule)
   git mv backend-go/migrations/000026_tariff_rule.up.sql \
            backend-go/migrations/000032_tariff_rule.up.sql
   git mv backend-go/migrations/000026_tariff_rule.down.sql \
            backend-go/migrations/000032_tariff_rule.down.sql

   # Rename 000028b → 000033 (sku_return_stats)
   git mv backend-go/migrations/000028_sku_return_stats.up.sql \
            backend-go/migrations/000033_sku_return_stats.up.sql
   git mv backend-go/migrations/000028_sku_return_stats.down.sql \
            backend-go/migrations/000033_sku_return_stats.down.sql
   ```

2. Delete or move `000029` (identical to `000030`, no down.sql):
   ```bash
   git rm backend-go/migrations/000029_orchestration.up.sql
   ```

3. Add seed entries to `schema_migrations` so golang-migrate knows 000029 was handled:
   ```sql
   -- If using the repaired filesystem, the tool sees files 26,28 as having 1 file each.
   -- If renamed, no extra work needed.
   ```

### Step B: Make `golang-migrate` the standard

Document in the runbook:
```bash
# Future migrations use:
migrate -path backend-go/migrations -database "$DATABASE_URL" up
migrate -path backend-go/migrations -database "$DATABASE_URL" down 1
```

---

## 6. Additional Notes

### Sub-directory migrations (not included in repair)

The `backend-go/migrations/` directory also has subdirectories:
- `finance/001_init.up.sql` — moves tables to `finance_module` schema
- `inventory/001_init.up.sql` — moves tables to `inventory_module` schema
- `order/001_init.up.sql` — moves tables to `order_module` schema
- `settlement/001_init.up.sql` — moves tables to `settlement_module` schema
- `sku/001_init.up.sql` — moves tables to `sku_module` schema

These use `ALTER TABLE ... SET SCHEMA` and are idempotent (use `IF EXISTS` on table references). They are NOT part of the numbered migration chain. The DB audit did not check whether these have been applied. If they were already applied (tables already moved), rerunning is safe (no-op). If not applied, they require `ALTER TABLE` privilege. Apply separately if needed.

### No data loss risk

All the tables being created are **new tables** (not in the 000001-000014 schema). No existing data is modified or dropped. The only `ALTER TABLE` is adding two nullable columns to `event_outbox` (already in 000001), which is a non-destructive change.

### Why a single file instead of separate migration files

Since `schema_migrations` doesn't exist and the version numbering has issues (duplicates, gaps), applying via `golang-migrate` CLI won't work reliably. A single idempotent SQL file applied via `psql -f` is the most reliable approach.

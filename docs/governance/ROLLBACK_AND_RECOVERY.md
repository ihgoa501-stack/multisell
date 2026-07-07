# Rollback and Recovery

Last updated: 2026-07-06

Procedures for database recovery, migration rollback, platform write-back recovery, release rollback, and the decision framework for when to roll back vs. fix forward.

## 1. Database Backup and Restore Drill Procedure

### 1.1 Backup Procedure

Full database backup (run before every deployment a migration):

```bash
# Env setup
BACKUP_DIR="/var/backups/postgres"
DB_NAME="multisell"
TIMESTAMP=$(date +%Y%m%d_%H%M%S)
BACKUP_FILE="${BACKUP_DIR}/${DB_NAME}_${TIMESTAMP}.sql.gz"

# Create backup (compressed)
pg_dump -h localhost -U multisell_user \
  --format=custom \
  --compress=9 \
  --file="${BACKUP_FILE}" \
  ${DB_NAME}

# Verify backup integrity
pg_restore --list "${BACKUP_FILE}" > /dev/null 2>&1
if [ $? -ne 0 ]; then
  echo "BACKUP CORRUPT: ${BACKUP_FILE}"
  exit 1
fi

echo "Backup saved: ${BACKUP_FILE}"
echo "Size: $(du -h "${BACKUP_FILE}" | cut -f1)"

# Also archive WAL for point-in-time recovery
pg_archive_wal
```

### 1.2 Restore Drill

Full restore (test this monthly against a non-production environment):

```bash
# 1. Stop all services connected to the database
docker compose stop server scheduler

# 2. Drop and recreate the database
dropdb -h localhost -U multisell_user ${DB_NAME}
createdb -h localhost -U multisell_user ${DB_NAME}

# 3. Restore from backup
pg_restore -h localhost -U multisell_user \
  --dbname=${DB_NAME} \
  --jobs=4 \
  --no-owner \
  "${BACKUP_FILE}"

# 4. Verify restore
psql -h localhost -U multisell_user -d ${DB_NAME} \
  -c "SELECT count(*) FROM information_schema.tables WHERE table_schema = 'public';"

# 5. Run any pending migrations that were applied after the backup
#    (these should be re-run from the down-migrated state)

# 6. Restart services
docker compose up -d server scheduler

# 7. Verify service health
curl -s http://localhost:8080/api/health
```

### 1.3 Point-in-Time Recovery (PITR)

```bash
# 1. Identify the target recovery time
TARGET_TIME="2026-07-05 14:30:00 UTC"

# 2. Restore from base backup + WAL archive
pg_ctl -D /var/lib/postgresql/data stop

# Clear and restore base backup
rm -rf /var/lib/postgresql/data/*
pg_basebackup -h backup-server -D /var/lib/postgresql/data -P

# Configure recovery.conf
cat > /var/lib/postgresql/data/recovery.conf <<EOF
restore_command = 'cp /var/lib/postgresql/archive/%f %p'
recovery_target_time = '${TARGET_TIME}'
recovery_target_timeline = 'latest'
EOF

# 3. Start PostgreSQL in recovery mode
pg_ctl -D /var/lib/postgresql/data start

# 4. Verify recovery target reached
psql -c "SELECT pg_last_wal_receive_lsn(), pg_last_wal_replay_lsn(), pg_is_in_recovery();"

# 5. Once verified, promote to primary
pg_ctl -D /var/lib/postgresql/data promote
```

**Expected restore time budget:**
- Full DB restore (< 50 GB): under 15 minutes
- PITR (from base backup + WAL): under 30 minutes
- Verify integrity: under 5 minutes

## 2. Migration Rollback Procedure

### 2.1 Step-by-Step Migration Rollback

```bash
# Step 1: Identify the migration to roll back
# Migrations are applied in version order. To roll back version 000068:
# The down migration is at: backend-go/migrations/000068_add_execution_mode.down.sql

# Step 2: Apply the down migration
psql -h localhost -U multisell_user -d multisell \
  -f backend-go/migrations/000068_add_execution_mode.down.sql

# Step 3: Remove the migration record from schema_migrations
psql -h localhost -U multisell_user -d multisell \
  -c "DELETE FROM schema_migrations WHERE version = 68;"

# Step 4: Verify schema is in expected state
psql -h localhost -U multisell_user -d multisell \
  -c "SELECT version, applied_at FROM schema_migrations ORDER BY version;"

# Step 5: Verify data integrity
# Check that rolled-back columns/tables no longer exist
# Check that no data was orphaned by the rollback
```

### 2.2 Migration Rollback Rules

- **Always have both up.sql and down.sql** — a migration is not complete without both.
- **Test down migration before deployment** — run up, verify, run down, verify the schema returns to the previous state.
- **Do not edit a migration that has been applied** — create a new migration to reverse the change.
- **If down migration fails** (constraint violation, data type mismatch):
  1. Do NOT force-drop tables or columns manually
  2. Pause the deployment, keep the current state
  3. Write a new "cleanup" migration that explicitly handles the failure case
  4. Apply the cleanup migration after fixing the root cause

### 2.3 Irreversible Migration Checklist

Some changes cannot be rolled back by a down migration alone:

| Operation | Risk Level | Rollback Strategy |
|-----------|-----------|-------------------|
| DROP COLUMN | High | Must restore from backup; down migration cannot recover data |
| DROP TABLE | Critical | Must restore from backup |
| ALTER COLUMN TYPE that loses precision | High | Backup restore or careful reverse migration with data conversion |
| UPDATE ... SET ... (data transformation) | Medium | Reverse migration must revert data; may lose precision |
| ADD NOT NULL on existing column | Low (if column was never null) / High (if nulls exist) | Down migration removes constraint only |
| ADD COLUMN with DEFAULT | Low | Simple DROP COLUMN in down migration |

**Decision**: If the migration contains any irreversible operation listed above, you MUST:
1. Take a full backup immediately before applying.
2. Test the down migration on staging with data that mirrors production (same volume, same null patterns).
3. Have the backup restore procedure verified and timed before production deployment.

## 3. Platform Write-Back Recovery

### 3.1 Per-Platform Recovery Procedures

#### Ozon

| Failure Mode | Recovery Action | Expected Time |
|-------------|----------------|---------------|
| Price update failed (HTTP 400) | Check price format (min/max, currency), log details, do not retry | Manual review: < 1 h |
| Inventory sync failed (HTTP 429) | Retry with backoff (30s, 2 min, 5 min), queue remaining | < 15 min |
| Listing publish failed (HTTP 401) | Rotate API key, retry all failed publishes | < 30 min |
| Product creation failed (duplicate SKU) | Check if product exists on Ozon, update instead of create | Manual review: < 30 min |
| Webhook delivery failed | Check HMAC, check endpoint reachable, replay from webhook log | < 15 min |
| Batch operation partially failed | Query operation status from Ozon API, retry only failed items | < 30 min |

#### Shopee

| Failure Mode | Recovery Action | Expected Time |
|-------------|----------------|---------------|
| API call rate limited | Respect `Retry-After` header, pause all Shopee operations for specified duration | Variable |
| Item update failed (deprecated API) | Check Shopee API changelog, update adapter, re-queue | < 2 h |
| Order status sync failed | Poll Shopee order API for latest status, reconcile with local state | < 30 min |
| Shipping label generation failed | Retry with Shopee logistics API, check address validity | < 15 min |

#### Lazada (when registered)

| Failure Mode | Recovery Action | Expected Time |
|-------------|----------------|---------------|
| Access token expired | Use refresh token, re-authenticate, retry operation | < 5 min |
| Product image upload failed | Check image URL format, resize/compress, retry | < 15 min |
| Order fulfillment failed | Log full error, alert human, do not retry automatically | Manual review: < 1 h |

### 3.2 General Write-Back Recovery Rules

1. **Log the failure payload** — always log the exact request and response for later replay.
2. **Idempotency key** — every write-back MUST have an idempotency key so replay does not duplicate.
3. **Replay queue** — failed write-backs go to a replay queue visible in the cockpit dashboard.
4. **Batch replay limit** — replay at most 100 failed operations per batch, confirm platform accepted them before replaying the next batch.
5. **Do not replay finance-critical operations automatically** — price updates, settlement data, and fee adjustments require manual review before replay.

## 4. Release Rollback Procedure

### Step-by-Step

```
Step 1: DETECT
    - Monitoring alert triggers or manual observation
    - Confirm the issue is deployment-related, not external

Step 2: DECIDE (see Section 5 decision tree)
    - Roll back the release
    - OR fix forward (only if root cause < 15 min to fix)

Step 3: NOTIFY
    - Alert channel: "Rolling back release {version}, ETA {time}"
    - On-call engineer begins rollback
    - Tech Lead notified for approval

Step 4: ROLLBACK
    Option A: Rollback via git (preferred for simple releases):
        git revert {release_commit_hash}
        git push origin main
        Deploy the revert

    Option B: Rollback via deployment (preferred for Docker):
        docker compose down server scheduler
        docker compose run --rm migrate down    # if migration was part of release
        docker compose up -d server scheduler   # deploy previous image tag

Step 5: VERIFY
    - Health endpoint passes: curl http://localhost:8080/api/health
    - Smoke test passes: cd backend-go && ./scripts/smoke_test.sh
    - All Agent heartbeats present
    - No error rate spike

Step 6: POST-ROLLBACK
    - Migration rollback (if applicable) — see Section 2
    - Write PIR entry (see INCIDENT_DRILL_CHECKLIST.md Section 4)
    - Verify backup integrity again
```

### Rollback Time Budget

| Component | Rollback Time | Notes |
|-----------|--------------|-------|
| Backend only (no migration) | < 5 min | Docker image revert, health check |
| Backend + migration | < 15 min | Run down migration + image revert |
| Full stack (frontend + backend + migration) | < 30 min | Frontend has CDN cache, allow TTL |
| Platform write-back recovery (post-rollback) | Variable | Depends on queue depth, see Section 3 |

### What to roll back vs what to not

- **ALWAYS roll back the backend** if a migration is involved and failed.
- **ALWAYS roll back platform adapters** if they wrote incorrect data.
- **Roll back frontend only if** the issue is user-facing (broken page, wrong data display).
- **Do NOT roll back** if the issue is transient (rate limit burst, temporary platform outage) — fix forward or wait.
- **Do NOT roll back** if fix-forward is faster than rollback + restore + re-deploy. Use the decision tree below.

## 5. Rollback Decision Tree

```
                       ┌─────────────────────────┐
                       │  Incident detected in    │
                       │  newly deployed release  │
                       └────────────┬────────────┘
                                    │
                                    ▼
                       ┌─────────────────────────┐
                       │  Is data corrupted or    │
                       │  financial calculation   │
                       │  affected?               │
                       └────────────┬────────────┘
                                    │
                  ┌─────────────────┼─────────────────┐
                  │ YES             │ NO              │ NO (but)
                  ▼                 ▼                  ▼
        ┌─────────────────┐ ┌─────────────────┐ ┌─────────────────┐
        │ ROLL BACK       │ │ Is the fix      │ │ Fix-forward if  │
        │ IMMEDIATELY.    │ │ simpler than    │ │ root cause is   │
        │ Full DB restore │ │ rollback?       │ │ clear AND fix   │
        │ if migration    │ │ (root cause     │ │ is < 4 lines    │
        │ corrupted data. │ │ identified      │ │ AND test passes │
        └─────────────────┘ │ and < 15 min    │ │ in staging.     │
                            │ to fix)         │ └─────────────────┘
                            └───────┬─────────┘
                                    │
                  ┌─────────────────┼─────────────────┐
                  │ YES             │ NO               │
                  ▼                 ▼                   ▼
        ┌─────────────────┐ ┌─────────────────┐
        │ Fix forward.    │ │ ROLL BACK.      │
        │ Hotfix commit   │ │ Standard roll-  │
        │ + deploy        │ │ back procedure. │
        │ immediately.    │ │ Document the    │
        ├─────────────────┤ │ issue for later │
        │ WARNING: verify │ │ hotfix.         │
        │ that the fix    │ └─────────────────┘
        │ does not mask   │
        │ a deeper issue. │
        └─────────────────┘

    Additional considerations:

    - MULTIPLE failures in one release → always roll back, not fix forward
    - SECURITY vulnerability → immediately roll back
    - PERFORMANCE regression → fix forward if root cause clear, roll back otherwise
    - PLATFORM API incompatibility → fix forward (new adapter version), roll back if breaking
    - NIGHT / WEEKEND → prefer rollback for P0/P1, fix forward only if trivial
```

## 6. Recovery Verification Checklist

After any rollback or recovery action, confirm system health with this checklist:

| # | Check | Command / Verification | Status |
|---|-------|----------------------|--------|
| 6.1 | All services running | `docker compose ps` | |
| 6.2 | Health endpoint returns 200 | `curl http://localhost:8080/api/health` | |
| 6.3 | DB connection pool healthy | `psql -c "SELECT count(*) FROM pg_stat_activity;"` | |
| 6.4 | Migration versions consistent | `psql -c "SELECT * FROM schema_migrations;"` | |
| 6.5 | No P0/P1 alerts firing | Check alert dashboard | |
| 6.6 | Agent heartbeats restored | Check cockpit dashboard or `agent_executions` table | |
| 6.7 | EventBus subscriptions active | Check `router.go` subscription list vs actual | |
| 6.8 | Scheduler ticks flowing | Check `scheduler.tick.*` events in last 5 min | |
| 6.9 | Platform write-backs proceeding | Check `platform_operation_log` for recent successes | |
| 6.10 | LLM cost within daily budget | Check cost dashboard | |
| 6.11 | Frontend loading and responsive | Check `curl http://localhost:3000` | |
| 6.12 | E2E critical path passes | `cd frontend-next/e2e && npx playwright test --grep "critical"` | |
| 6.13 | Backup integrity re-verified | `pg_restore --list <latest_backup>` | |
| 6.14 | Post-incident review created | See INCIDENT_DRILL_CHECKLIST.md Section 4 | |

### Recovery Sign-Off

| Role | Sign-off | Date |
|------|----------|------|
| On-call engineer | | |
| Tech Lead | | |
| Owner | | |

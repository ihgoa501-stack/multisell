# Backup Policy

## Overview

This document defines the backup strategy for 凌镜 LingMirror production data.
All production data is stored in PostgreSQL 15 running in Docker.

## Schedule

| Type | Frequency | Retention | Target |
|------|-----------|-----------|--------|
| Daily full backup | Every 24h at 03:00 UTC | 7 days local + 30 days S3/cold storage | All databases |
| Manual snapshot | On demand (before migrations, config changes) | Retained until next manual trigger | Selected databases |

## Daily Backup

Executed via cron or scheduled Docker task:

```bash
docker exec -t multisell-db-1 pg_dump -U multisell -Fc multisell > /backups/daily/multisell_$(date +%Y%m%d_%H%M%S).dump
```

### Retention

- **Local** (`/backups/daily/`): Keep 7 most recent dumps. Cleanup via cron:
  ```bash
  find /backups/daily/ -name "*.dump" -mtime +7 -delete
  ```
- **S3/cold storage** (`s3://lingmirror-backups/daily/`): Keep 30 days. Upload via `aws s3 cp` or `rclone` after each dump.

## Manual Snapshot

Before any destructive operation (migration, config change, version upgrade):

```bash
docker exec -t multisell-db-1 pg_dump -U multisell -Fc multisell > /backups/snapshots/multisell_manual_$(date +%Y%m%d_%H%M%S).dump
```

Manual snapshots are not auto-pruned. Clean up old ones when no longer relevant.

## Restore Procedure

### Restore to the production Docker service

```bash
# 1. Copy the dump file into the container
docker cp /backups/daily/multisell_20260705_030000.dump multisell-db-1:/tmp/restore.dump

# 2. Drop and recreate the target database
docker exec -i multisell-db-1 psql -U multisell -c "DROP DATABASE IF EXISTS multisell;"
docker exec -i multisell-db-1 psql -U multisell -c "CREATE DATABASE multisell;"

# 3. Restore from dump
docker exec -i multisell-db-1 pg_restore -U multisell -d multisell /tmp/restore.dump

# 4. Restart services that depend on the database
docker compose restart
```

### Restore to a local test instance

```bash
# 1. Create a test database
createdb -U postgres multisell_test_restore

# 2. Restore the dump
pg_restore -U postgres -d multisell_test_restore /backups/daily/multisell_20260705_030000.dump

# 3. Connect and verify
psql -U postgres -d multisell_test_restore -c "SELECT count(*) FROM information_schema.tables;"
```

## Backup Verification

Monthly: Restore the most recent backup to a test instance and run basic validation.

```bash
# Automated check — run on the 1st of each month
createdb -U postgres multisell_verify
pg_restore -U postgres -d multisell_verify /backups/daily/$(ls -t /backups/daily/*.dump | head -1)
psql -U postgres -d multisell_verify -c "SELECT count(*) FROM information_schema.tables;"
dropdb -U postgres multisell_verify
```

Verify that the restored database contains: all expected tables, at least some data rows, and no role/permission errors during restore.

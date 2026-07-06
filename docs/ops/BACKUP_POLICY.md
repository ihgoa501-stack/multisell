# 备份策略

## 概述

凌镜使用 PostgreSQL 15 数据库。以下备份策略适用于 `multisell` 数据库。

## 每日备份

- **时间**: 每天凌晨 03:00 UTC+8
- **工具**: `pg_dump -Fc` (自定义格式，支持并行恢复)
- **保留**: 7 天本地 + 30 天 S3/cold storage
- **命令**: `docker exec -t db pg_dump -U multisell -Fc multisell > /backup/multisell_$(date +%Y%m%d).dump`

## 手动快照

```bash
# 创建快照
docker exec -t $(docker ps -f name=db -q) pg_dump -U multisell -Fc multisell > /backup/snapshot_$(date +%Y%m%d_%H%M%S).dump

# 验证快照完整性
pg_restore -l /backup/snapshot_$(date +%Y%m%d_%H%M%S).dump | head -5
```

## 恢复步骤

**生产恢复:**
```bash
# 1. 停服务
docker compose down

# 2. 清旧库
docker compose up -d db
docker exec -t $(docker ps -f name=db -q) psql -U multisell -c "DROP DATABASE IF EXISTS multisell; CREATE DATABASE multisell;"

# 3. 恢复
docker exec -i $(docker ps -f name=db -q) pg_restore -U multisell -d multisell -Fc < /backup/multisell_20260706.dump

# 4. 起服务
docker compose up -d
```

**本地测试实例:**
```bash
# 恢复到本地 docker-compose db
docker compose up -d db
pg_restore -h localhost -p 5432 -U multisell -d multisell -Fc /backup/multisell_20260706.dump
```

## 验证

- 每月恢复备份到测试实例验证完整性
- 每次手动快照后验证表行数: `docker exec ... psql -U multisell -c "SELECT COUNT(*) FROM sales_order;"`

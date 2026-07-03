#!/usr/bin/env bash
# =============================================================================
# 凌镜 LingMirror — 回滚脚本
# 用法: ./scripts/rollback.sh [--revert-migration]
# =============================================================================
set -euo pipefail

DEPLOY_DIR="$(cd "$(dirname "$0")/.." && pwd)"
COMPOSE_FILES="-f docker-compose.yml -f docker-compose.prod.yml"
REVERT_MIGRATION=false

if [[ "${1:-}" == "--revert-migration" ]]; then
    REVERT_MIGRATION=true
fi

info()  { echo -e "\033[0;34m[INFO]\033[0m  $*"; }
ok()    { echo -e "\033[0;32m[OK]\033[0m    $*"; }
warn()  { echo -e "\033[0;33m[WARN]\033[0m  $*"; }
error() { echo -e "\033[0;31m[ERROR]\033[0m $*"; }

cd "$DEPLOY_DIR"

info "开始回滚 LingMirror ..."

# ---- Step 1: 获取上一个版本 ----------------------------------------------------
# Prefer latest git tag (set by release workflow), fall back to HEAD~1.
PREV=$(git describe --tags --abbrev=0 2>/dev/null) || PREV=""
if [[ -z "$PREV" ]]; then
    PREV=$(git rev-parse HEAD~1 2>/dev/null) || {
        error "找不到上一个版本，无法回滚"
        exit 1
    }
fi
info "目标版本: $(echo $PREV | cut -c1-8)"

# ---- Step 2: 回滚代码 ----------------------------------------------------
info "Step 1/4: 回滚代码 ..."
git reset --hard "$PREV"
ok "  代码已回滚到 $(git rev-parse --short HEAD)"

# ---- Step 3: 回滚数据库（可选）-------------------------------------------
if [[ "$REVERT_MIGRATION" == "true" ]]; then
    info "Step 2/4: 回滚数据库迁移 ..."
    docker compose $COMPOSE_FILES run --rm migrate down 1 || true
    ok "  数据库已回滚一个版本"
else
    info "Step 2/4: 跳过数据库回滚（加 --revert-migration 可回滚数据库）"
fi

# ---- Step 4: 构建并重启 ----------------------------------------------------
info "Step 3/4: 构建上一个版本 ..."
docker compose $COMPOSE_FILES build backend frontend
ok "  构建完成"

info "Step 4/4: 重启服务 ..."
docker compose $COMPOSE_FILES up -d --no-deps backend frontend caddy

# ---- 验证 ----------------------------------------------------------------
sleep 5
if curl -sf http://localhost:8080/api/health >/dev/null 2>&1; then
    ok "  回滚后服务正常"
else
    warn "  回滚后服务未响应，检查日志: docker compose logs --tail=50 backend"
fi

ok "回滚完成！"
info "当前版本: $(git rev-parse --short HEAD)"
info "如需彻底回滚数据库，在迁移文件中找到上一个版本的序号，然后运行:"
info "  docker compose $COMPOSE_FILES run --rm migrate down N"

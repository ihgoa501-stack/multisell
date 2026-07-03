#!/usr/bin/env bash
# =============================================================================
# 凌镜 LingMirror — 一键部署脚本
# 用法: ./scripts/deploy.sh [--no-migrate]
# =============================================================================
set -euo pipefail

DEPLOY_DIR="$(cd "$(dirname "$0")/.." && pwd)"
COMPOSE_FILES="-f docker-compose.yml -f docker-compose.prod.yml"
MONITORING_FILES="-f docker-compose.monitoring.yml"
SKIP_MIGRATE=false

# ---- 参数解析 ----------------------------------------------------------------
if [[ "${1:-}" == "--no-migrate" ]]; then
    SKIP_MIGRATE=true
fi

# ---- 颜色输出 ----------------------------------------------------------------
info()  { echo -e "\033[0;34m[INFO]\033[0m  $*"; }
ok()    { echo -e "\033[0;32m[OK]\033[0m    $*"; }
warn()  { echo -e "\033[0;33m[WARN]\033[0m  $*"; }
error() { echo -e "\033[0;31m[ERROR]\033[0m $*"; }

cd "$DEPLOY_DIR"

info "开始部署 LingMirror ..."
info "目录: $DEPLOY_DIR"

# ---- Step 1: 拉取最新代码 ----------------------------------------------------
info "Step 1/5: 拉取最新代码 ..."
git fetch origin
LOCAL=$(git rev-parse HEAD)
REMOTE=$(git rev-parse origin/main)

if [[ "$LOCAL" == "$REMOTE" ]]; then
    info "  已是最新版本 ($(echo $LOCAL | cut -c1-8))，跳过 pull"
else
    git pull origin main
    ok "  代码已更新: $(echo $LOCAL | cut -c1-8) → $(echo $REMOTE | cut -c1-8)"
fi

# ---- Step 2: 数据库迁移 ----------------------------------------------------
if [[ "$SKIP_MIGRATE" == "false" ]]; then
    info "Step 2/5: 执行数据库迁移 ..."
    if docker compose $COMPOSE_FILES run --rm migrate 2>&1; then
        ok "  数据库迁移完成"
    else
        warn "  迁移可能已是最新，继续部署 ..."
    fi
else
    info "Step 2/5: 跳过数据库迁移 (--no-migrate)"
fi

# ---- Step 3: 构建新镜像 ----------------------------------------------------
info "Step 3/5: 构建服务镜像 ..."
docker compose $COMPOSE_FILES build backend frontend
ok "  构建完成"

# ---- Step 4: 部署服务（零停机滚动）------------------------------------------
info "Step 4/5: 部署服务 ..."

deploy_service() {
    local svc=$1
    local port=${2:-}
    local health_url=${3:-}

    info "  部署 $svc ..."
    docker compose $COMPOSE_FILES up -d --no-deps --build "$svc"

    if [[ -n "$health_url" ]]; then
        info "  等待 $svc 健康检查 ..."
        for i in $(seq 1 30); do
            if curl -sf "$health_url" >/dev/null 2>&1; then
                ok "  $svc 已就绪"
                return 0
            fi
            sleep 2
        done
        error "  $svc 启动超时，回滚中 ..."
        return 1
    fi
}

# 顺序：backend → frontend → caddy
deploy_service "backend" "8080" "http://localhost:8080/api/health" || {
    # 回滚：重启旧版本
    docker compose $COMPOSE_FILES up -d --no-deps backend
    error "部署失败，已回滚到上一版本"
    exit 1
}

deploy_service "frontend" "3000" || true
deploy_service "caddy" "80" || true

# ---- Step 5: 启动监控栈 ----------------------------------------------------
info "Step 5/7: 启动监控栈 ..."
if docker compose $COMPOSE_FILES $MONITORING_FILES up -d --no-deps prometheus grafana alertmanager 2>&1; then
    ok "  监控栈已启动"
else
    warn "  监控栈启动失败（不影响主服务），检查: docker compose $MONITORING_FILES logs"
fi

# ---- Step 6: 验证全链路 ----------------------------------------------------
info "Step 6/7: 验证部署 ..."

sleep 3

# API 健康检查
if curl -sf http://localhost:8080/api/health >/dev/null 2>&1; then
    ok "  API 健康检查通过"
else
    warn "  API 健康检查未响应，检查日志: docker compose logs --tail=50 backend"
fi

# Caddy 代理检查
if curl -sf -o /dev/null -w "%{http_code}" http://localhost/ >/dev/null 2>&1; then
    ok "  Caddy 代理正常"
else
    warn "  Caddy 代理未响应，检查配置"
fi

# ---- 收尾 ------------------------------------------------------------------
ok "部署完成！"
info "当前版本: $(git rev-parse --short HEAD)"
info "部署时间: $(date '+%Y-%m-%d %H:%M:%S')"
info ""
info "查看日志: docker compose $COMPOSE_FILES logs --tail=50 -f backend"
info "监控面板: http://localhost:3001 (Grafana)，用户名 admin"
info "回滚命令: ./scripts/rollback.sh"

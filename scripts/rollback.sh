#!/usr/bin/env bash
# =============================================================================
# 凌镜 LingMirror — 回滚脚本
# 用法: DOMAIN=example.com ./scripts/rollback.sh --target <commit-or-tag>
# =============================================================================
set -euo pipefail

DEPLOY_DIR="$(cd "$(dirname "$0")/.." && pwd)"
COMPOSE_FILES="-f docker-compose.yml -f docker-compose.prod.yml"
TARGET=""

while [[ $# -gt 0 ]]; do
    case "$1" in
        --target)
            [[ $# -ge 2 ]] || { echo "--target requires a commit or tag" >&2; exit 2; }
            TARGET="$2"
            shift 2
            ;;
        --revert-migration)
            echo "--revert-migration 已禁用：应用回滚必须保留数据库 migration 152；数据库恢复需要 Owner 批准并使用异地不可变备份整库恢复" >&2
            exit 2
            ;;
        *)
            echo "unknown argument: $1" >&2
            exit 2
            ;;
    esac
done

info()  { echo -e "\033[0;34m[INFO]\033[0m  $*"; }
ok()    { echo -e "\033[0;32m[OK]\033[0m    $*"; }
warn()  { echo -e "\033[0;33m[WARN]\033[0m  $*"; }
error() { echo -e "\033[0;31m[ERROR]\033[0m $*"; }

cd "$DEPLOY_DIR"

: "${DOMAIN:?set the production domain}"
[[ -n "$TARGET" ]] || { error "必须用 --target 明确指定已知良好版本"; exit 2; }

# Never discard an operator's changes. Untracked files are allowed because the
# production .env and generated evidence are intentionally not committed.
if ! git diff --quiet || ! git diff --cached --quiet; then
    error "工作区存在已跟踪文件改动；拒绝回滚，避免覆盖现场证据"
    exit 1
fi

TARGET_COMMIT=$(git rev-parse --verify "${TARGET}^{commit}" 2>/dev/null) || {
    error "目标版本不存在或不是 commit: $TARGET"
    exit 1
}
CURRENT_COMMIT=$(git rev-parse HEAD)
[[ "$TARGET_COMMIT" != "$CURRENT_COMMIT" ]] || { error "目标版本就是当前版本"; exit 1; }

info "开始回滚 LingMirror ..."

info "当前版本: ${CURRENT_COMMIT:0:8}"
info "目标版本: ${TARGET_COMMIT:0:8}"

# ---- Step 1: 在可信的当前版本上先做不可变备份 --------------------------
info "Step 1/5: 创建强制异地不可变备份 ..."
docker compose --profile manual $COMPOSE_FILES run --rm backup
ok "  回滚前备份完成"

# ---- Step 2: 回滚代码 ----------------------------------------------------
info "Step 2/5: 切换代码 ..."
# A detached checkout preserves branch history and cannot rewrite a shared ref.
git switch --detach "$TARGET_COMMIT"
ok "  代码已回滚到 $(git rev-parse --short HEAD)"

# Old releases are not trusted to retain today's network boundary. Refuse to
# start any target that cannot prove only Caddy publishes host ports.
[[ -x scripts/verify_prod_compose.sh ]] || {
    error "目标版本缺少可执行的生产 Compose 边界验证器；拒绝启动"
    exit 1
}
./scripts/verify_prod_compose.sh

# ---- Step 3: 在触碰数据库前完成目标镜像构建 ------------------------------
info "Step 3/5: 构建目标版本 ..."
docker compose $COMPOSE_FILES build backend frontend image-service image-service-migrate
ok "  构建完成"

# ---- Step 4: 停止写入并验证图片服务数据库 -------------------------------
info "Step 4/5: 停止应用写入；保留主数据库 migration 152 ..."
docker compose $COMPOSE_FILES stop backend image-service
docker compose $COMPOSE_FILES run --rm image-service-migrate
ok "  图片服务目标版本向上迁移完成；主数据库未执行 down/revert"

# ---- Step 5: 重启 ---------------------------------------------------------
info "Step 5/5: 启动目标版本 ..."
docker compose $COMPOSE_FILES up -d --no-deps image-service backend frontend caddy

# ---- 验证 ----------------------------------------------------------------
# Backend port 8080 is intentionally not published in production. Verify both
# liveness and dependency readiness through the public TLS boundary.
healthy=false
for _ in {1..30}; do
    if curl --fail --silent --show-error --max-time 5 "https://${DOMAIN}/api/health" >/dev/null \
        && curl --fail --silent --show-error --max-time 5 "https://${DOMAIN}/api/ready" >/dev/null; then
        healthy=true
        break
    fi
    sleep 2
done
if [[ "$healthy" != "true" ]]; then
    error "回滚后 60 秒内未同时通过 health/readiness；保留现场并返回失败"
    docker compose $COMPOSE_FILES logs --tail=100 backend caddy >&2 || true
    exit 1
fi
ok "  回滚后 liveness/readiness 正常"

ok "回滚完成！"
info "当前版本: $(git rev-parse --short HEAD)"
info "原版本（仅用于审计或人工恢复）: ${CURRENT_COMMIT:0:8}"

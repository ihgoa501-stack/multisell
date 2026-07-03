#!/usr/bin/env bash
set -euo pipefail

# ==========================================
# LingMirror 服务器初始化脚本
# 用法: ./deploy.sh <server-ip>
# 注意: 请先配置 SSH 密钥登录目标服务器
# ==========================================

SERVER_IP="${1:?用法: deploy.sh <server-ip>}"
SSH_OPTS="-o StrictHostKeyChecking=no -o ConnectTimeout=10"
SSH="ssh $SSH_OPTS root@$SERVER_IP"

echo "=== 1. 安装 Docker ==="
$SSH "apt-get update -qq && apt-get install -y -qq ca-certificates curl git" 2>/dev/null
$SSH "curl -fsSL https://get.docker.com | bash" 2>/dev/null
$SSH "systemctl enable --now docker" 2>/dev/null

echo "=== 2. 创建项目目录 ==="
$SSH "mkdir -p /opt/lingmirror" 2>/dev/null

echo "=== 3. 上传代码 ==="
cd "$(dirname "$0")/.."
tar czf /tmp/lingmirror.tar.gz \
  --exclude='.git' \
  --exclude='.gitignore' \
  --exclude='node_modules' \
  --exclude='.next' \
  --exclude='vendor' \
  --exclude='*.tar.gz' \
  --exclude='docs' \
  --exclude='*.md' \
  backend-go frontend-next deploy \
  docker-compose.yml docker-compose.prod.yml \
  Caddyfile.prod .dockerignore .env.example
scp $SSH_OPTS /tmp/lingmirror.tar.gz root@$SERVER_IP:/opt/lingmirror/ 2>/dev/null

echo "=== 4. 解压代码 ==="
$SSH "cd /opt/lingmirror && tar xzf lingmirror.tar.gz && rm lingmirror.tar.gz" 2>/dev/null

echo "=== 5. 生成配置 ==="
echo "部署完成。请在服务器上手动配置:"
echo "  ssh root@$SERVER_IP"
echo "  cd /opt/lingmirror"
echo "  cp .env.example .env"
echo "  vi .env   # 填入 DB_PASSWORD, JWT_SECRET, LLM_API_KEY, DOMAIN 等"
echo ""
echo "然后启动:"
echo "  cd /opt/lingmirror"
echo "  docker compose -f docker-compose.yml -f docker-compose.prod.yml up -d --build"

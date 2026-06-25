#!/usr/bin/env bash
set -euo pipefail

# ==========================================
# LingMirror 一键部署脚本
# 用法: ./deploy.sh <server-ip> <password>
# 注意: 请在使用前设置 SSH_PASSWORD 环境变量或通过参数传入
# ==========================================

SERVER_IP="${1:?用法: deploy.sh <server-ip> <ssh-password>}"
PASSWORD="${2:?用法: deploy.sh <server-ip> <ssh-password>}"
SSH_OPTS="-o StrictHostKeyChecking=no -o ConnectTimeout=10"
SSH="sshpass -p '$PASSWORD' ssh $SSH_OPTS root@$SERVER_IP"
SCP="sshpass -p '$PASSWORD' scp $SSH_OPTS"

echo "=== 1. 安装 Docker ==="
$SSH "apt-get update -qq && apt-get install -y -qq ca-certificates curl" 2>/dev/null
$SSH "curl -fsSL https://get.docker.com | bash" 2>/dev/null
$SSH "systemctl enable --now docker" 2>/dev/null

echo "=== 2. 创建项目目录 ==="
$SSH "mkdir -p /opt/lingmirror" 2>/dev/null

echo "=== 3. 上传代码 ==="
cd "$(dirname "$0")/.."
tar czf /tmp/lingmirror.tar.gz \
  --exclude='.git' \
  --exclude='node_modules' \
  --exclude='.next' \
  --exclude='vendor' \
  --exclude='*.tar.gz' \
  backend-go frontend-next deploy
sshpass -p "$PASSWORD" scp $SSH_OPTS /tmp/lingmirror.tar.gz root@$SERVER_IP:/opt/lingmirror/ 2>/dev/null

echo "=== 4. 解压代码 ==="
$SSH "cd /opt/lingmirror && tar xzf lingmirror.tar.gz && rm lingmirror.tar.gz" 2>/dev/null

echo "=== 5. 生成配置（需要手动设置）==="
echo "部署完成。请在服务器上手动配置:"
echo "  ssh root@$SERVER_IP"
echo "  cd /opt/lingmirror/deploy"
echo "  cp .env.example .env"
echo "  vi .env    # 填入 DB_PASSWORD, JWT_SECRET 等"
echo ""
echo "然后启动:"
echo "  docker compose --env-file .env up -d --build"

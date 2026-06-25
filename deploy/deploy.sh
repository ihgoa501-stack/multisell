#!/usr/bin/env bash
set -euo pipefail

# ==========================================
# LingMirror 一键部署脚本
# 用法: ./deploy.sh <server-ip> <password>
# ==========================================

SERVER_IP="${1:-118.196.42.156}"
PASSWORD="${2:-DF124ad.}"
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

echo "=== 5. 生成配置 ==="
JWT_SECRET=$($SSH "openssl rand -hex 32" 2>/dev/null | tr -d '\r')
$SSH "cat > /opt/lingmirror/deploy/.env << 'ENVEOF'
DB_PASSWORD=lingmirror_prod_2024
JWT_SECRET=$JWT_SECRET
FRONTEND_URL=http://$SERVER_IP
API_URL=http://$SERVER_IP/api
ENVEOF" 2>/dev/null

echo "=== 6. 初始化数据库（按 y 确认）==="
cd /opt/lingmirror/deploy
docker compose --env-file .env up -d db 2>/dev/null || true

echo "=== 7. 构建并启动所有服务 ==="
$SSH "cd /opt/lingmirror/deploy && docker compose --env-file .env up -d --build" 2>/dev/null

echo "=== 8. 设置自动重启 ==="
$SSH "cat > /etc/systemd/system/lingmirror.service << 'UNIT'
[Unit]
Description=LingMirror MultiSell
Requires=docker.service
After=docker.service

[Service]
WorkingDirectory=/opt/lingmirror/deploy
ExecStart=/usr/bin/docker compose --env-file .env up
ExecStop=/usr/bin/docker compose down
Restart=always
RestartSec=10

[Install]
WantedBy=multi-user.target
UNIT" 2>/dev/null
$SSH "systemctl daemon-reload && systemctl enable lingmirror" 2>/dev/null

echo "=== 9. 等待启动 ==="
echo "等待服务就绪（约 30 秒）..."
sleep 30
$SSH "curl -s http://localhost:8080/api/health || echo 'Health check pending...'" 2>/dev/null

echo ""
echo "================================"
echo "  部署完成！"
echo "  访问地址: http://$SERVER_IP"
echo "  API:      http://$SERVER_IP/api"
echo "  Health:   http://$SERVER_IP/api/health"
echo "================================"

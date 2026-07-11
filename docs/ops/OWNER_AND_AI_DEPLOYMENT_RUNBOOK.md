# 凌镜服务器部署、测试与恢复统一手册

> 文档状态：当前唯一服务器操作手册
> 适用对象：Owner、Codex、Claude Code 及其他受 Owner 委托的 Agent
> 服务器：`118.196.42.156`
> 操作系统：Ubuntu 24.04 LTS
> 项目目录：`/opt/multisell`
> 更新日期：2026-07-11

## 0. 文档权威性

以后凡是初始化服务器、部署、更新、测试、恢复、回滚或排查生产问题，必须先完整阅读本文件。

本文件同时服务两类读者：

- **第一部分给 Owner 看**：只说明当前结果、风险、需要批准的事项和验收方法。
- **第二部分给 AI Agent 执行**：规定顺序、命令、停止条件、测试和交付格式。

当本文件与旧部署文档冲突时，以本文件和 `docs/SELF_USE_OPERATING_DIRECTION.md` 为准。治理、安全、审批和审计规则仍以 `docs/governance/` 为最高约束。

禁止把服务器上的临时状态反写成事实，除非 Agent 已实际验证。禁止把“代码存在”“容器启动”写成“生产可用”。

---

# 第一部分：Owner 阅读区

## 1. 这台服务器是什么

这台服务器用于 Owner 自用的凌镜跨境商品实验系统。当前只服务 Ozon 首轮真实商品实验：

```text
20 个线索 → 3 个候选 → 1 个批准商品 → 有效成交 → 最终净利润
```

服务器可以承载：

- 凌镜前端；
- Go 后端；
- PostgreSQL 数据库；
- 数据库迁移；
- Caddy HTTPS 入口；
- 备份、日志和基础监控。

它不是公共 SaaS，不开放多租户、订阅计费、公共 API 或外部客户入口。

## 2. Owner 怎么连接服务器

正常连接方式是 SSH 密钥，不是密码。

在 Owner 的 Mac 终端运行：

```bash
ssh lingmirror
```

本机 SSH 配置：

```text
别名：lingmirror
IP：118.196.42.156
用户：root（仅允许密钥登录）
私钥：~/.ssh/lingmirror
公钥：~/.ssh/lingmirror.pub
```

私钥禁止：

- 发给其他人或 Agent；
- 粘贴到聊天、工单或文档；
- 上传到 GitHub；
- 复制到服务器；
- 放进项目目录。

Agent 只可以调用已经配置好的 `ssh lingmirror`，不应读取、打印或传输私钥内容。

如果 `ssh lingmirror` 失败，先通过云厂商控制台确认服务器在线。不要为了方便重新开启长期密码登录。

## 3. Owner 只需要批准什么

Owner 不需要选择 Docker、数据库参数或迁移命令。Agent 负责技术判断。

以下事项必须由 Owner 明确批准：

- 重装服务器或删除数据；
- 恢复或覆盖正式数据库；
- 产生新增云资源费用；
- 使用真实 Ozon 凭证；
- 发布商品、采购、广告、价格、库存、订单、退款和资金动作；
- 接受无法回滚的风险。

普通只读检查、构建、测试、备份和无外部写入的部署，不需要 Owner 决定技术细节。

## 4. 什么叫部署成功

必须同时满足：

- `https://正式域名/` 可以打开；
- `/api/health` 返回成功；
- Owner 可以登录；
- 数据库健康且无持续错误；
- 3000、5432、8080 不对公网开放；
- SSH 密码登录关闭；
- 备份已生成，并在服务器外保存一份；
- 后端测试、前端构建和关键流程测试通过；
- Ozon 等外部写入仍为关闭或 dry-run；
- Agent 报告准确 Git 提交、备份文件和剩余风险。

缺少任何一项，只能标记为“部署中”或“测试环境”，不能标记为生产可用。

## 5. Owner 如何快速验收

Owner 只需确认：

1. 页面能否正常打开；
2. 能否登录；
3. 页面显示的是不是最新版本；
4. Agent 是否明确报告测试结果；
5. Agent 是否明确说明 Ozon 真实写入仍然关闭；
6. Agent 是否给出备份位置和回滚方法。

如果 Agent 只说“部署完成”但没有证据，视为未完成。

---

# 第二部分：AI Agent 强制执行区

## 6. Agent 开始前必须确认

执行前必须输出并回答：

```text
目标：
操作类型：首次部署 / 更新 / 测试 / 恢复 / 回滚 / 故障排查
风险等级：
是否影响数据库：
是否影响外部平台：
是否需要 Owner 批准：
预计停机：
回滚点：
```

然后检查：

```bash
cd /Users/lc/multisell
git status --short
git rev-parse HEAD
ssh -o BatchMode=yes -o ConnectTimeout=8 lingmirror 'hostname; uptime; uname -srmo'
```

工作区有其他人的修改时不得覆盖、清理或提交。禁止使用 `git reset --hard`、`git checkout --` 或强制推送。

## 7. 永久禁止事项

任何 Agent 都不得：

- 把 SSH 私钥、数据库密码、JWT Secret、LLM Key 或平台 Token 打印到输出；
- 把 `.env` 提交到 Git；
- 从 2026-07-11 重装前的旧系统复制二进制、cron、systemd、SSH 配置或环境变量；
- 开放公网 3000、5432、8080、9090、3001；
- 长期开启 SSH 密码登录；
- 使用仓库默认密码进入生产；
- 跳过备份直接迁移；
- 忽略失败的数据库迁移继续发布；
- 把 mock、stub 或 deterministic 输出称为真实 AI；
- 未经 Owner 批准执行 Ozon 发布、调价、库存、采购、广告、订单、退款或资金动作；
- 因为健康接口返回 200 就宣布完整生产验收通过。

发现疑似入侵、未知 root cron、陌生 systemd 服务、异常用户或密钥时：停止部署，保留证据，隔离公网入口并报告 Owner。

## 8. 首次初始化干净服务器

### 8.1 身份确认

```bash
ssh lingmirror
hostname
uname -srmo
cat /etc/os-release
ip -brief address
```

必须确认 IP 是 `118.196.42.156`、系统是 Ubuntu 24.04，并且系统为新安装。

### 8.2 SSH 加固

服务器必须满足：

```text
PermitRootLogin prohibit-password
PasswordAuthentication no
KbdInteractiveAuthentication no
PubkeyAuthentication yes
```

修改后必须先运行：

```bash
sshd -t
systemctl reload ssh
```

然后从第二个连接验证：

```bash
ssh -o BatchMode=yes -o ConnectTimeout=8 lingmirror 'echo SSH_OK'
```

第二个连接未成功前，禁止关闭当前会话。

### 8.3 安装基础组件

```bash
apt-get update
apt-get install -y ca-certificates curl git ufw fail2ban postgresql-client
curl -fsSL https://get.docker.com | sh
systemctl enable --now docker
systemctl enable --now fail2ban
```

从互联网下载安装脚本前，Agent 必须确认域名是官方 `get.docker.com`。

### 8.4 防火墙

```bash
ufw default deny incoming
ufw default allow outgoing
ufw allow 22/tcp
ufw allow 80/tcp
ufw allow 443/tcp
ufw --force enable
ufw status verbose
```

云厂商安全组同样只允许 22、80、443。数据库、前端、后端和监控端口只允许 Docker 内网或 `127.0.0.1`。

### 8.5 目录和权限

```bash
install -d -m 755 /opt/multisell
install -d -m 700 /opt/multisell/backups
```

生产 `.env` 必须为：

```bash
chmod 600 /opt/multisell/.env
```

## 9. 首次部署

### 9.1 获取代码

推荐使用 Git 获取确定版本，不上传本地工作区压缩包：

```bash
cd /opt
git clone https://github.com/ihgoa501-stack/multisell.git multisell
cd /opt/multisell
git fetch origin
git checkout main
git pull --ff-only origin main
git rev-parse HEAD
```

仓库为专有项目。如果 clone 需要凭证，使用最小权限、可撤销的部署凭证；不得把个人长期 Token 写入远程 URL。

### 9.2 生成全新密钥

旧服务器使用过的数据库密码、JWT、LLM Key、平台 Token 和 OpenClaw Token 一律视为失效，不得复用。

至少生成：

```bash
openssl rand -base64 48   # DB_PASSWORD
openssl rand -base64 64   # JWT_SECRET
openssl rand -base64 32   # PLATFORM_TOKEN_ENCRYPTION_KEY
```

生成值只能写入 `/opt/multisell/.env` 或正式 Secret 管理系统，不得输出到交付报告。

首次部署允许暂时不配置 Ozon、LLM 和其他外部平台密钥。缺失密钥时应保持 read-only、stub 或功能关闭，不得伪造已接通。

### 9.3 生产容器边界

使用：

```bash
cd /opt/multisell
docker compose -f docker-compose.yml -f docker-compose.prod.yml config
docker compose -f docker-compose.yml -f docker-compose.prod.yml config > /tmp/lingmirror-compose-rendered.yml
docker compose -f docker-compose.yml -f docker-compose.prod.yml up -d db
docker compose -f docker-compose.yml -f docker-compose.prod.yml ps
```

**启动前强制停止检查：** 当前仓库的基础 `docker-compose.yml` 包含开发端口映射。Agent 必须检查合并后的 `/tmp/lingmirror-compose-rendered.yml`。如果 3000、5432 或 8080 绑定到 `0.0.0.0` 或未指定 Host IP，禁止启动生产栈；必须先把生产覆写修正为 Docker 内网或 `127.0.0.1`，并重新执行 `docker compose ... config`。不能只依赖 UFW，因为 Docker 端口转发可能绕过普通 UFW 入站规则。

确认端口边界安全后，才允许启动数据库并确认健康，再运行迁移。不得直接执行全栈启动掩盖迁移错误。

生产容器最终必须是：

```text
db       仅 Docker 内网
backend  仅 Docker 内网或 127.0.0.1:8080
frontend 仅 Docker 内网或 127.0.0.1:3000
caddy    公网 80/443
```

## 10. 数据恢复

### 10.1 当前可信恢复源

2026-07-11 重装前已生成并验证：

```text
本机：/Users/lc/Backups/lingmirror/multisell-pre-recovery-20260711-162815.dump
SHA-256：8a1c8c2d95e477c5860236003a1c9f17fbfd348f8c41b8445b5827b6afb70fe7
```

服务器重装后只能从本机可信备份恢复。不得从旧系统恢复可执行文件和配置。

### 10.2 恢复前验证

```bash
shasum -a 256 /Users/lc/Backups/lingmirror/multisell-pre-recovery-20260711-162815.dump
pg_restore -l /Users/lc/Backups/lingmirror/multisell-pre-recovery-20260711-162815.dump >/dev/null
```

校验值不匹配或 `pg_restore -l` 失败时立即停止。

### 10.3 恢复原则

- 先恢复到临时测试数据库；
- 检查表数量、关键业务表和用户数据；
- 通过后才恢复到正式 `multisell`；
- 恢复正式库属于高风险操作，需要 Owner 明确批准；
- 恢复后仍需运行当前仓库缺失的后续迁移；
- 不恢复旧数据库角色密码，使用新服务器的新密码。

## 11. 数据库迁移

迁移前：

```bash
cd /opt/multisell
docker compose -f docker-compose.yml -f docker-compose.prod.yml ps
docker compose run --rm backup
```

必须确认备份文件非空、能够列出内容，并复制一份到服务器外。

迁移：

```bash
docker compose -f docker-compose.yml -f docker-compose.prod.yml run --rm migrate
```

如果迁移命令失败：

1. 停止部署；
2. 保留完整错误；
3. 不得使用“可能已经执行”作为成功依据；
4. 检查是否存在重复迁移版本；
5. 根据备份和迁移内容决定修复、回滚或恢复；
6. 数据删除或不可逆迁移必须再次请求 Owner 批准。

## 12. 部署前测试

在本地仓库运行：

```bash
cd /Users/lc/multisell/backend-go
go build ./...
go vet ./...
go test ./...

cd /Users/lc/multisell/frontend-next
npm test
npm run build
```

涉及关键页面、登录、权限或业务流程时，再运行：

```bash
cd /Users/lc/multisell/frontend-next/e2e
npx playwright test
```

任何失败都必须报告。不得把“已知问题”自动当成可以忽略，除非文档明确记录且与本次发布无关。

## 13. 启动与部署后测试

### 13.1 启动

```bash
cd /opt/multisell
docker compose -f docker-compose.yml -f docker-compose.prod.yml up -d --build
docker compose -f docker-compose.yml -f docker-compose.prod.yml ps
```

### 13.2 基础健康检查

```bash
curl -fsS http://127.0.0.1:8080/api/health
curl -fsSI http://127.0.0.1/
docker compose -f docker-compose.yml -f docker-compose.prod.yml logs --since 5m backend db
```

日志中不得持续出现 `ERROR`、`FATAL`、字段不存在、表不存在或迁移失败。

### 13.3 公网暴露检查

从服务器外检查：

```bash
nc -vz 118.196.42.156 22
nc -vz 118.196.42.156 80
nc -vz 118.196.42.156 443
```

以下端口必须连接失败：

```text
3000、5432、8080、9090、3001
```

### 13.4 应用验收

必须验证：

- HTTPS 证书有效；
- 前端首页和登录页；
- 登录成功与错误密码拒绝；
- JWT 刷新；
- RBAC 权限；
- `/api/health`；
- WebSocket `/ws`；
- 审计日志写入；
- 数据库健康检查；
- Owner 核心页面；
- 关键备份可读取；
- 外部平台写入保持关闭。

## 14. 日常更新部署

固定顺序：

```text
确认范围 → 本地测试 → 服务器外备份 → 记录当前提交 → 拉取确定提交
→ 构建 → 迁移 → 启动 → 冒烟测试 → 观察 → 交付报告
```

服务器上禁止直接开发和手改代码。更新前记录：

```bash
cd /opt/multisell
git rev-parse HEAD
docker compose -f docker-compose.yml -f docker-compose.prod.yml ps
```

只允许快进更新：

```bash
git fetch origin
git pull --ff-only origin main
```

不得在服务器上解决 Git 冲突。

## 15. 回滚

回滚触发条件：

- 健康检查失败；
- 登录或权限失效；
- 持续 5xx；
- 数据库迁移失败；
- 价格、库存、订单、资金或外部平台发生错误写入；
- 审计、审批或幂等门禁失效；
- 出现疑似入侵。

应用回滚必须回到明确的 Git 提交或镜像版本。数据库回滚优先使用兼容的 down migration；只有确认需要覆盖数据时才从备份恢复，并请求 Owner 批准。

回滚后必须重新执行第 13 节全部检查。

## 16. 备份要求

- 每天至少一次数据库全量备份；
- 本地保留 7 天；
- 服务器外保留 30 天；
- 每次迁移和高风险部署前手动备份；
- 每月至少一次恢复演练；
- 备份必须有 SHA-256；
- 只有实际恢复成功过的备份才能标记为“可恢复”。

备份不得只保存在同一台服务器。

## 17. Ozon 与真实经营安全边界

服务器部署成功不等于允许真实经营写入。

默认状态：

```text
市场与账号验证：read-only
商品发布：dry-run
价格与库存：禁止真实写入
采购与广告：必须 Owner 批准
订单、退款与资金：必须 Owner 批准并审计
```

任何 Agent 不得通过部署动作顺带开启 production 模式。

## 18. 故障处理

### 网站打不开

检查 Caddy、前端、后端、DNS、证书和防火墙。不要先重装。

### 数据库 unhealthy

检查健康命令、容器日志、磁盘、连接数和迁移。数据库可以响应不代表结构正确。

### 磁盘超过 80%

停止非必要构建，检查 Docker 镜像、日志和备份。清理前列出将删除的内容，禁止直接执行全量 prune。

### SSH 失败

使用云控制台检查实例和安全组。优先恢复公钥，不长期恢复密码登录。

### 疑似入侵

立即停止部署和密钥输入，封锁非必要端口，保存数据库备份到外部。root 级持久化出现时优先重装系统，不在原系统上宣布“已清理干净”。

## 19. Agent 交付报告模板

每次部署、恢复或测试必须提交：

```text
Owner 现在可以做什么：
访问地址：
部署版本（完整 Git commit）：
服务器状态：
数据库状态：
备份位置与 SHA-256：
执行的迁移：
通过的测试：
失败或跳过的测试：
公网开放端口：
Ozon/外部平台模式：
回滚方法：
剩余风险：
是否达到生产标准：是 / 否
```

报告不得包含密码、Token、私钥或完整 `.env`。

## 20. 当前重装执行清单（2026-07-11）

- [x] Ubuntu 24.04 重装完成；
- [x] 公网 IP 保持 `118.196.42.156`；
- [x] 本机 SSH 公钥重新安装；
- [x] `ssh lingmirror` 密钥登录验证成功；
- [x] SSH 密码登录关闭；
- [ ] 云安全组只开放 22、80、443；
- [ ] 安装 Docker、UFW、Fail2ban；
- [ ] 从 Git 获取确定版本；
- [ ] 创建全新的生产密钥；
- [ ] 生产端口边界验证；
- [ ] 启动全新 PostgreSQL；
- [ ] 在临时数据库验证可信备份；
- [ ] Owner 批准正式数据库恢复；
- [ ] 运行当前数据库迁移；
- [ ] 部署后端、前端和 Caddy；
- [ ] HTTPS 配置完成；
- [ ] 后端全量测试通过；
- [ ] 前端测试和构建通过；
- [ ] 登录、RBAC、WebSocket、审计验证通过；
- [ ] 外部平台保持 read-only/dry-run；
- [ ] 服务器外备份和恢复演练完成；
- [ ] Owner 完成最终验收。

在本清单全部完成前，这台服务器的状态是：**干净的新服务器，尚未达到生产标准**。

## 21. 变更记录

| 日期 | 版本/提交 | 操作 | 执行者 | 备份 | 测试结果 | 生产资格 |
|---|---|---|---|---|---|---|
| 2026-07-11 | 待部署 | 重装 Ubuntu、恢复 SSH 密钥登录并关闭密码登录 | Codex + Owner | 本机可信 DB dump | SSH 通过 | 否 |

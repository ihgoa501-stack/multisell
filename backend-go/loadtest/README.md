# k6 压测脚本

针对 multisell backend-go 的 5 个压测场景。

## 安装 k6

### macOS
```bash
brew install k6
```

### Linux (Debian/Ubuntu)
```bash
sudo gpg -k
sudo gpg --no-default-keyring --keyring /usr/share/keyrings/k6-archive-keyring.gpg --keyserver hkp://keyserver.ubuntu.com:80 --recv-keys C5AD17C747E3415A36442D57C278E32655F775D8
echo "deb [signed-by=/usr/share/keyrings/k6-archive-keyring.gpg] https://dl.k6.io/deb stable main" | sudo tee /etc/apt/sources.list.d/k6.list
sudo apt update
sudo apt install k6
```

### Docker
```bash
docker pull grafana/k6:latest
# 用 docker 运行示例：
# docker run --rm -i --network host -e API_BASE=http://host.docker.internal:8080 grafana/k6:latest run - < dashboard.js
```

验证安装：
```bash
k6 version
```

## 环境变量

| 变量 | 默认值 | 说明 |
|------|--------|------|
| `API_BASE` | `http://localhost:8080` | 后端服务地址 |
| `TOKEN` | (空) | JWT Token，会作为 `Authorization: Bearer <TOKEN>` 头传递 |

示例：
```bash
export API_BASE=http://localhost:8080
export TOKEN=eyJhbGciOiJIUzI1NiIs...
```

## 场景说明

| 文件 | 场景 | 并发 | 持续 | 触发接口 |
|------|------|------|------|----------|
| `dashboard.js` | Dashboard 总览 | 100 | 2 min | GET `/api/v1/dashboard/overview` |
| `ai-command.js` | AI 对话 | 100 | 2 min | POST `/api/v1/ai/chat` `{message:"检查库存"}` |
| `action-approve.js` | Action 审批 | 50 | 2 min | GET `/api/v1/ai/actions?status=suggested` → POST `/api/v1/ai/actions/:id/approve` |
| `websocket.js` | WS 连接保活 | 100 | 30 s | `ws://localhost:8080/ws` |
| `sku-batch.js` | SKU dry-run 批处理 | 50 | 2 min | POST `/api/v1/ai/run` `{agent_id:"A6", decision_point:"profit_check", context:{sku_id:N}}` N=1..1000 |

### 所有场景共同阈值
- `http_req_duration` p95 < 500ms
- `http_req_failed` < 5%

## 运行单个场景

```bash
# 1. Dashboard
k6 run dashboard.js

# 2. AI 对话
k6 run ai-command.js

# 3. Action 审批
k6 run action-approve.js

# 4. WebSocket
k6 run websocket.js

# 5. SKU 批处理
k6 run sku-batch.js
```

带环境变量运行：
```bash
API_BASE=http://localhost:8080 TOKEN=xxx k6 run dashboard.js
```

输出到 JSON（便于后续分析）：
```bash
k6 run --out json=dashboard-result.json dashboard.js
```

## 顺序运行全部场景

```bash
chmod +x run-all.sh
./run-all.sh
```

`run-all.sh` 会依次执行 5 个场景，每个场景的结果写入 `results/<scenario>-<timestamp>.json`，最后打印汇总。

## 预期结果

在健康的服务上，每个场景应满足：

| 指标 | 预期 |
|------|------|
| `http_req_failed` | < 5% |
| `http_req_duration` p95 | < 500 ms |
| `checks` 通过率 | > 95% |
| `iterations` | dashboard/ai-command 约 12000+ 次；action-approve 约 6000+ 次 |
| `ws_messages_received` | > 0，平均每个连接至少收到 1 条消息 |
| `sku_processed` | 1000 个 SKU 至少各执行 1 次 |

### 各场景典型表现（参考）

1. **dashboard.js**
   - 100 VU 持续 2 min，主要验证 dashboard overview 接口的缓存/查询效率
   - p95 < 500ms 说明 DB 查询 + 聚合逻辑健康

2. **ai-command.js**
   - 100 VU 持续 2 min，POST `/api/v1/ai/chat`
   - "检查库存" 作为固定 prompt，便于压测 AI 路由 + LLM 调用链
   - 若 LLM 调用是同步阻塞的，p95 可能接近阈值，需要观察是否需要异步化

3. **action-approve.js**
   - 50 VU 持续 2 min，两步操作
   - 若 suggested action 池为空，迭代会跳过 approve 步骤，建议测试前先造数据
   - 主要验证状态机流转和并发审批的安全性

4. **websocket.js**
   - 100 并发连接保持 30 秒
   - 关注 `ws_messages_received` 是否 > 0，验证服务端是否主动推送
   - 关注 `ws_sessions` / `ws_connecting` 指标

5. **sku-batch.js**
   - 50 VU 循环 1000 个 SKU，POST `/api/v1/ai/run`
   - dry-run 模式应不写库，主要验证 agent A6 profit_check 决策逻辑
   - 关注吞吐量 `http_reqs`，理想情况下 2 分钟内完成全部 1000 SKU 多轮循环

## 故障排查

- **`http_req_failed` 高**：检查后端服务是否启动、端口是否正确、TOKEN 是否过期
- **p95 > 500ms**：查看后端 DB 慢查询、连接池配置、AI 接口超时
- **WebSocket 连接失败**：确认后端 WS 路由为 `/ws`，且无反向代理超时
- **action-approve 迭代为 0**：先调用 `POST /api/v1/ai/chat` 生成 suggested action

## 目录结构

```
loadtest/
├── README.md
├── run-all.sh
├── dashboard.js
├── ai-command.js
├── action-approve.js
├── websocket.js
├── sku-batch.js
└── results/          # 由 run-all.sh 自动创建
```

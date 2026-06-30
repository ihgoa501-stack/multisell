# Claude Code 工作流 — 凌镜 7 天一人 Agent 公司 MVP

更新时间：2026-06-29

---

## 1. 会话架构

### 主会话 (Lead Agent) — 当前会话

当你的主 Claude Code 窗口就是 Lead Agent。

**职责：**
- 接收 Owner 指令和理解业务目标
- 分解任务 → 分发给 subagents
- 审核每个 subagent 的产出
- 维护战情板 / 验收日志 / 闭环审计
- 向 Owner 报告（业务语言，非技术语言）
- 做 Owner 无法从 subagent 获得答案的决策

**不能做的：**
- 替 Owner 做业务决策
- 跳过审批/审计/风险检查
- 同时做 5 条线的全部工作（用 subagent）

### Subagents (通过 Agent 工具)

当有独立任务时可启动 subagent。每个 subagent 收到明确的任务、输入和验收标准。

**何时用 subagent：**
- 两条线的工作互不依赖（例如 A 线后端和 D 线 Mock 可以并行）
- 需要验证已有代码是否存在某个功能
- 代码审查、测试运行

**何时不用 subagent：**
- 只需要快速检查一行代码或文件
- 当前任务与正在改的工作紧密相关（上下文丢失成本 > 收益）

### 后台 Session

Claude Code 的 background mode 支持长时间运行的任务。

**何时用后台 session：**
- 启动 subagent review 自己的 diff（异步审查）
- 运行长时间的测试套件

---

## 2. 分工矩阵

| 任务类型 | 谁做 | 工具 |
|----------|------|------|
| 理解业务目标 | Lead Agent | 读文档 + 问 Owner |
| 研究现有代码 | Lead Agent 或 subagent | CodeGraph / grep / read |
| 设计新功能 | Lead Agent | 写设计文档 |
| 后端开发 (Go) | Lead Agent 或 subagent | Write / Edit |
| 后端测试 (Go) | 开发者或 QA subagent | Bash (`go test`) |
| 前端开发 (TS/React) | Lead Agent 或 subagent | Write / Edit |
| 前端 build 验证 | QA subagent | Bash (`npm run build`) |
| 代码风险审查 | Review / Risk-Guard subagent | Read diff + 检查规则 |
| 文档更新 | Docs subagent | Write |
| 每日汇报 | Lead Agent | 写日志 + 更新战情板 |

---

## 3. 推荐的并行会话策略

### 策略 A — 顺序为主 + 局部并行

```
Day 1:
  步骤 1: Lead Agent 写 candidate domain (backend)
  步骤 2: 同时 → QA subagent 跑测试基线
                → Research subagent 确认 logistics/platformfee 接口

Day 2:
  步骤 1: Lead Agent 写 profit_summary API
  步骤 2: 并行 → C subagent 写 Agent 规则
                → D subagent 准备 Mock 数据
```

**适用：** 前 3 天，A 线牵制 B/C/D 线的输入

### 策略 B — 充分并行

```
Day 4: (B 线不依赖 A 线进一步开发)
  并行: → B subagent 写前端页面
        → A subagent 处理审批接线
        → D subagent 做 Mock 数据
        → E subagent 跑测试
```

**适用：** Day 4 后，各线独立程度高

---

## 4. Subagent 模板

### 通用 Subagent 指令模板

```
任务: <具体任务>
线: A/B/C/D/E
输入: <已有代码/文件/数据结构>
输出: <验收标准>
约束:
  - 不修改 <范围外>
  - 遵守平台宪法 (<条款号>)
  - 用 Owner 能看懂的业务语言汇报
```

### Subagent 定义草案

#### lingmirror-research (Research Agent)

```
职责: 只读研究，不写文件
触发: "研究 X 功能的现有实现"
输入输出: 代码位置 → 报告
约束: 不写文件
```

#### lingmirror-qa (QA Agent)

```
职责: 运行测试 + 验证行为
触发: "验证 XX 是否工作"
输入输出: 测试范围 → pass/fail 报告
工具: Bash (go test, npm test, npm run build)
```

#### lingmirror-review (Review Agent)

```
职责: 代码审查 + 架构合规
触发: "审查这个 diff"
输入输出: diff → 审查报告（严重性 + 文件引用）
检查项:
  - 是否 Owner 目标一致
  - 是否在正确系统层
  - 是否绕过审批/审计/RBAC
  - 是否有隐藏外部副作用
  - Owner 能否验收
```

#### lingmirror-docs (Docs Agent)

```
职责: 维护文档
触发: "更新文档" / "写文档"
输入输出: 新增/变更 → 文档更新
更新对象: README, INDEX.md, 模块内 README
```

#### lingmirror-risk-guard (Risk Agent)

```
职责: 风险合规检查
触发: 每次代码变更后
检查项:
  - 是否触碰价格/库存/订单/钱/权限？
  - 是否绕过审批/审计？
  - 是否有自动外部发布逻辑？
  - 是否写在旧栈？
  - 是否符合禁止事项？
输出: GREEN (无风险) / YELLOW (需注意) / RED (阻塞)
```

---

## 5. 运行检查清单

每次 subagent 交付后，Lead Agent 检查：

- [ ] 验收标准是否达成
- [ ] 是否有测试
- [ ] 是否遵守禁止事项
- [ ] 是否用业务语言报告
- [ ] 文档是否需要更新
- [ ] 战情板状态是否需要更新

---

## 6. 日常工作流

### 早上

1. 读取战情板 → 了解昨日进度
2. 读取验收日志 → 昨日阻碍
3. 规划当日各线任务
4. 与 Owner 确认方向（如果需要）

### 日间

1. 分发 subagent 任务
2. 审核 subagent 产出
3. 串联依赖任务
4. 更新战情板

### 傍晚

1. 汇总当日进展
2. 更新验收日志
3. 运行完整验证
4. Owner-readable 日报

---

## 7. 日终验证命令

```bash
# 后端
cd backend-go && go test ./...   # 必须通过
cd backend-go && go vet ./...    # 必须通过

# 前端
cd frontend-next && npm run build && npm test  # 必须通过

# 风险检查 (如有 diff)
codegraph explore "RISK_CHECK: 是否触碰禁止事项"
```

---

## 8. 与本项目治理文档的关系

| 本文件对照 | 遵循 |
|-----------|------|
| Owner-First Protocol | ✅ 每条线都在 Owner 业务语言框架内 |
| Platform Constitution | ✅ 分层/风险/禁止事项已嵌入 |
| Agent Development Protocol | ✅ 角色/检查清单/QA 规则已应用 |
| Kernel Contracts | ✅ Approval/Audit/Observability 已覆盖 |

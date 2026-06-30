# Issue Triage 指南 / Triage Guide

## 总原则 / Principles

- **每周一次 Triage**（建议周一早上，30-60 分钟）
- **来一个处理一个 vs 集中处理**：集中处理效率更高，减少上下文切换
- **四步流程**：遍历 → 分类 → 定优先级 → 排期

## 常规 Triage 流程

### 1. 遍历所有新 Issue

打开所有没有 `status/triage` 标签的 Issue。按以下顺序逐个处理：

### 2. 关闭无效内容

| 情况 | 操作 | 标签 |
|------|------|------|
| 重复 Issue | 关闭并链接到原始 Issue | `duplicate` |
| 无法复现 / 信息不足 | 索要补充信息，7 天无回复关闭 | `status/needs-info` |
| 使用疑问 | 引导到 Discussions | `question` |
| 确认不是问题 | 关闭并说明原因 | `wontfix` / `invalid` |

### 3. 有效 Issue → 打标签

先打类型标签，再打优先级标签：

**类型标签（必选其一）:**
| 标签 | 适用场景 |
|------|----------|
| `bug` | 确认的 Bug |
| `enhancement` | 新功能需求 |
| `type/refactor` | 重构 / 代码改进 |
| `type/tech-debt` | 技术债务 |
| `type/epic` | 大型专题（需要拆分为多个子 Issue） |
| `documentation` | 文档相关 |
| `question` | 已分流但未关闭的疑问 |

**领域标签（可选，帮助筛选）:**
| 标签 | 适用场景 |
|------|----------|
| `area/backend` | 后端 Go |
| `area/frontend` | 前端 Next.js |
| `area/ci-cd` | CI/CD 流水线 |
| `area/testing` | 测试 |
| `area/e2e` | E2E 测试 |

**优先级标签（必选其一）:**
| 标签 | 定义 | 响应标准 |
|------|------|----------|
| `priority/critical` | 生产阻断：数据丢失、核心功能不可用、安全漏洞 | 立即处理 |
| `priority/high` | 重大功能受损但可绕过，或严重影响用户体验 | 24 小时内处理 |
| `priority/medium` | 普通 Bug 或重要的功能需求 | 当前或下个 Sprint |
| `priority/low` | 次要问题、优化建议、技术债务 | 待办池，有空时处理 |

### 4. 状态标签（标记进展）

| 标签 | 含义 |
|------|------|
| `status/triage` | 新 Issue，等待 Triage（创建时自动添加） |
| `status/needs-info` | 等待 Issue 提交者补充信息 |
| `status/blocked` | 被外部依赖阻塞 |

## 处理标准

### Bug
- **必须包含**：版本号、环境信息、复现步骤
- 确认后去掉 `status/triage`，正确设置 `priority/*`
- **critical Bug** 立即通知团队

### Feature Request
- 先在 Discussions 讨论（config.yml 已引导用户）
- 达成共识后才转为正式 Issue
- 转正式 Issue 时去掉 `status/triage`

## Triage Checklist（每次执行）

- [ ] 所有新 Issue 都已分配类型标签
- [ ] 所有新 Issue 都已分配优先级标签
- [ ] `needs-info` 超过 7 天无回复的已关闭
- [ ] 已确认的 Bug 已关联到对应的 milestone
- [ ] 高优 Issue 已分配给具体负责人（如适用）

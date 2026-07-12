# 外部标杆：前端、测试与工程质量门（2026-07-12）

## 结论

优秀系统不是追求最多的单元测试，而是把不同风险放进正确的验证层：纯逻辑用快速单元测试，组件用用户可见行为测试，关键 Owner 流程用生产构建下的真实浏览器 E2E，数据库不变量用 PostgreSQL 集成测试，并在 CI 中强制 build、type check、lint、race、迁移和关键 E2E。

对凌镜最值得借鉴的不是更换 Next.js、测试框架或建设大型测试平台，而是把现有 133 个前端测试和 3131 个后端测试重新围绕高风险闭环组织：审批不可自审、证据不能越级、负利润不能 continue、审计不能改写请求体、事件失败重启可恢复。

## 一手标杆原则

### 1. 测试用户能观察到的行为

Testing Library 的官方原则是让测试尽可能接近真实使用方式，并避免依赖组件内部 state、方法和生命周期。查询优先使用可访问角色和标签，而不是 DOM 结构或 CSS 类。[Testing Library 官方介绍](https://testing-library.com/docs/)、[官方查询优先级](https://testing-library.com/docs/queries/about/)

这意味着优秀的页面测试应该验证“Owner 看见阻断项并无法提交”，而不是只断言某个内部数组或 payload helper 返回了字段。

### 2. E2E 必须验证真实用户流程且彼此隔离

Playwright 官方建议测试用户可见行为、保持测试隔离、优先使用 role/label locator、使用可重试的 web-first assertions，并在 CI 对每次提交运行。第三方依赖应受控模拟，数据库测试数据也应受控。[Playwright Best Practices](https://playwright.dev/docs/best-practices)

适合凌镜的关键 E2E 数量不需要很多，但必须覆盖：登录与权限拒绝、候选市场裁决、实验终局、1688 草稿审批不发布、小Q证据追溯、审批禁止自审。

### 3. 单元、组件、集成、E2E 各有职责

Next.js 官方明确区分 unit、component、integration 和 E2E；E2E 用生产式浏览器环境验证完整任务。对于工具支持仍有限的 async Server Components，官方更推荐 E2E 而非强行单测。[Next.js Testing Guide](https://nextjs.org/docs/app/guides/testing)

这支持一个重要判断：凌镜不能用大量 Vitest 组件测试替代 `/demand-cases → /experiments → /sourcing1688` 的核心浏览器流程。

### 4. 生产构建和真实性能数据都是质量门的一部分

Next.js 官方生产检查建议在上线前运行 `next build`，并在生产式环境运行、检查错误处理、CSP、类型安全、Core Web Vitals 和 bundle。[Next.js Production Checklist](https://nextjs.org/docs/app/guides/production-checklist)

Google 的 Web Vitals 将 LCP、INP、CLS定义为用户体验核心指标，并强调实验室指标要结合真实用户现场数据；当前推荐阈值为 LCP ≤2.5s、INP ≤200ms、CLS ≤0.1，按第75百分位评价。[Web Vitals 官方说明](https://web.dev/articles/vitals)

凌镜只供 Owner 使用，不需要追求公共网站 SEO 或海量用户指标；但应记录 Owner 页面真实加载和交互延迟，尤其是大表格、证据详情和图片处理页面。

### 5. Go 并发和输入边界要使用标准工具验证

Go 官方建议使用 `go test -race` 发现运行时数据竞争，并明确它只能发现测试实际执行路径中的竞争。[Go Race Detector](https://go.dev/doc/articles/race_detector)

Go 原生 fuzzing 适合持续变异输入以发现人容易漏掉的边界与安全问题，失败输入会成为后续普通测试的回归语料。[Go Fuzzing](https://go.dev/doc/security/fuzz/)

适合凌镜 fuzz 的对象包括：审计中间件 request body、URL/JSON 外部采集解析、状态/真实性枚举、货币与哈希输入；不需要对所有 CRUD 做 fuzz。

### 6. 多步数据库变更必须原子

Go 官方数据库文档把 transaction 定义为多步操作要么全部成功、要么全部回滚，从而保持数据完整性。[Go Transactions](https://go.dev/doc/database/execute-transactions)

这直接对应小Q trace 半成品、审批消费/完成、实验 gate 与对象链接等问题：如果它们共同表达一个事实，就应在一个事务或明确的可恢复状态机中完成。

## “优秀”与“不优秀”的定义

| 维度 | 优秀 | 不优秀 |
|---|---|---|
| 单元测试 | 快、确定、覆盖纯业务规则与边界 | 只断言实现细节或为了覆盖率写测试 |
| 组件测试 | 使用 role/label，验证 Owner 可见行为 | 只测 helper、CSS、内部 state |
| API/集成 | 使用真实 middleware、事务和 PostgreSQL 约束 | SQLite 通过就假设 PostgreSQL 正确 |
| E2E | 少量覆盖最高风险完整流程，使用生产 build 与隔离数据 | 页面能打开就算通过；依赖旧端口/固定凭据 |
| 并发/边界 | race、fuzz、失败注入覆盖高风险路径 | 正常路径全绿但队列满、超时、重启从未测试 |
| CI | 必需质量门不可跳过，失败有证据产物 | 测试可选、关键集成默认 skip、E2E 不阻断发布 |
| 性能 | 用真实 Owner 路径和现场指标判断 | 只看 bundle/build 成功或模拟 Lighthouse |

## 与凌镜当前差距

### 已接近标杆

- 后端 build/vet/test、前端 test/lint/build 均能在本次完整运行。
- 关键并发包已有 race 测试；平台包覆盖率约 74%–97%。
- 前端已使用 Vitest、Testing Library，仓库也存在 Playwright E2E 基础。
- Next.js 生产构建能生成 91 个页面。

### 仍有明显差距

1. 高风险业务路径缺少少量但权威的生产式 E2E；现有测试数量没有阻止审批自审、错误放行等 P0。
2. sourcing1688 的关键 PostgreSQL 不可变约束测试默认可跳过，不能作为稳定 CI 门。
3. 多个页面/Service 过大，测试容易集中在 helper/payload，真实页面组合行为覆盖不足。
4. 旧 `verify_page.py` 仍依赖过期端口和历史凭据，不符合隔离、可复验的标杆。
5. 尚未看到对 audit body、外部 JSON/URL 边界进行 fuzz 的稳定质量门。
6. 生产性能只有 build 证据，没有 Owner 真实路径的响应时间或 Web Vitals 基线。

## 最小借鉴方案

不新增测试框架。继续使用 Go test、Vitest/Testing Library 和 Playwright，只补四层最小质量门：

1. **每次提交**：Go build/vet/test、前端 test/lint/build、文档/迁移/Compose 契约。
2. **P0 负向测试**：审批自审、证据越级、负利润 continue、多链接歧义、现金币种不一致。
3. **PostgreSQL + race**：关键约束、EventBus 崩溃恢复、approval/audit 并发和幂等。
4. **6 条生产式 E2E**：登录权限、DemandCase、Experiment、1688、小Q、审批审计；使用隔离账号/数据库、role locator 和 web-first assertion。

随后只对 audit body、外部采集 payload 和状态枚举增加少量 fuzz。等真实 Owner 使用出现性能问题后，再建立 Web Vitals/页面耗时基线，不提前建设大型可观测前端平台。

## 不应照搬

- 不因为 Next.js 支持就引入微前端、多 zone 或多租户。
- 不追求每个页面都有 E2E；只覆盖不可由较低层可靠证明的高风险完整流程。
- 不以覆盖率 100% 代替风险覆盖。
- 不为单 Owner 内部系统建设跨浏览器设备矩阵；当前 Chromium 主路径加必要移动/兼容抽查即可。
- 不增加 Cypress、Pact 等第二套重叠工具，除非现有工具明确无法完成一个已出现的真实需求。

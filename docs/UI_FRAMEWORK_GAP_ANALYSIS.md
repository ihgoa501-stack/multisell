# UI 覆盖审计报告

> 分析日期：2026-06-24
> 范围：当前活跃前端 `frontend-next/`

---

## 结论

凌镜前端已经切换到 Next.js / React / TypeScript / Ant Design 新技术栈。旧 Vue 前端 `frontend/` 不再是当前 UI 主线，因此不再按 `Layout.vue`、`RedesignLayout.vue`、`views/`、`views-redesign/` 做缺口判断。

当前活跃 UI 判断依据：

- App Router 页面：`frontend-next/src/app/`
- 侧边栏菜单：`frontend-next/src/config/menu.ts`
- 主布局：`frontend-next/src/app/(main)/layout.tsx`
- 布局组件：`frontend-next/src/components/layout/`
- API client：`frontend-next/src/lib/api-client.ts`

## 页面覆盖

2026-06-24 复核结果：

| 检查项 | 结果 |
|---|---:|
| `frontend-next/src/app/(main)` 下页面数 | 53 |
| Next build 输出 app routes | 50+ |
| `frontend-next/src/config/menu.ts` 菜单入口数 | 41 |
| 菜单入口缺失目标页面 | 0 |

验证命令：

```bash
cd frontend-next
npm run build
```

`npm run build` 已通过，输出包含 dashboard、商品、订单、结算、财务、物流、发布、AgentOS、设置等主业务页面，以及动态详情页：

- `/products/[id]`
- `/orders/[id]`
- `/settlement/[id]`
- `/listing-tasks/[id]`
- `/agents/[id]`
- `/agents/[id]/trace/[traceId]`
- `/actions/[id]`

## 当前菜单覆盖

`frontend-next/src/config/menu.ts` 的 41 个菜单入口均有实际页面匹配。当前分组包括：

- 总览：Dashboard、AI 指挥中心
- 商品管理：商品、类目、品牌、SKU、库存、供应商
- 销售管理：平台、平台集成、刊登、刊登任务
- 订单物流：订单、订单导入、物流、平台费用
- 财务：财务总览、结算、决策、分配、成本分摊
- AgentOS：控制台、Agent 列表、Action 中心、进化、熵监控、工作队列、信任与自主度
- 运营：异常、通知、图片生成、批量导入、操作日志、搜索、报表、售后、1688采购
- 设置：系统设置、LLM 配置、权限管理、审批策略

## 旧报告作废说明

此前的 UI 缺口报告描述的是旧 Vue 前端内部两条线：

- 旧框架：`frontend/src/views/`
- Vue redesign 框架：`frontend/src/views-redesign/`

这份判断已经不适用于当前项目状态。当前全站 UI 主线是 `frontend-next/`，所以旧报告中提到的以下问题不再作为当前缺口：

- `/redesign` 路由线
- `views-redesign/aftersales` 缺失
- `views-redesign/listing_task/ListingTaskDetail.vue` 缺失
- Vue `modules/redesign.ts` 与 `modules/*.ts` 的双维护问题
- Vue 孤儿页面清单

这些内容只保留为历史迁移参考，不应进入新栈验收标准。

## 仍需处理的问题

### 前端 lint 未通过

当前 `npm run build` 和 `npm test` 通过，但 `npm run lint` 仍失败。主要问题：

- `@typescript-eslint/no-explicit-any`
- `@typescript-eslint/no-unused-vars`
- `react-hooks/set-state-in-effect`

### API 路径一致性

新后端路由统一挂在 `/api/v1`。前端默认 API base 是 `http://localhost:8080/api`，因此 `apiClient` 调用应统一使用 `/v1/*`。

当前仍有部分调用缺少 `/v1` 前缀，典型位置包括：

- `frontend-next/src/lib/actions-api.ts`
- `frontend-next/src/app/(main)/actions/page.tsx`
- `frontend-next/src/app/(main)/settings/policy/page.tsx`
- `frontend-next/src/app/(main)/agents/evolution/page.tsx`
- `frontend-next/src/app/(main)/agents/trust/page.tsx`

这属于新栈联调风险，不是页面迁移缺口。

## 建议关闭项

1. 将所有前端 API 调用统一为 `/v1/*`。
2. 修复 `frontend-next` lint，恢复质量门禁。
3. 后续新增页面时，同时更新 `frontend-next/src/app/` 和 `frontend-next/src/config/menu.ts`，并用 build 校验菜单目标。
4. 旧 Vue UI 相关缺口只在迁移追溯时引用，不再作为 active frontend 的待办清单。

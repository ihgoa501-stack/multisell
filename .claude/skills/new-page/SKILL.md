---
name: new-page
description: 脚手架前端 domain 页面（路由+CRUD表+菜单），用户指定模块名
disable-model-invocation: true
---

# new-page — 脚手架前端页面

在 `frontend-next/src/app/(main)/` 下生成标准 CRUD 页面，配套 API 调用和菜单注册。

## 用法

```
/new-page <module_name>
```

## 流程

1. 创建 `frontend-next/src/app/(main)/<module_name>/page.tsx`
2. 使用 `@/components/crud/CrudListPage` + `@/lib/api-client` 标准模式
3. 在 `frontend-next/src/config/menu.ts` 注册菜单项
4. 验证：`cd frontend-next && npm run build`

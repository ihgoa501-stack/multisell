---
name: new-module
description: 脚手架新 domain 模块（routes/handler/service/model）+ 注册路由，两分钟完成
disable-model-invocation: true
---

# new-module — 脚手架 domain 模块

为 `backend-go/internal/domain/` 生成标准四件套 + 更新 router.go。

## 用法

```
/new-module <snake_name>
```

## 流程

1. 创建 `backend-go/internal/domain/<name>/` 目录
2. 从 templates/ 复制四个文件，替换 `{{ModuleName}}`（PascalCase）和 `{{module_name}}`（snake_case）
3. 在 `router.go` 的 domain import 块末尾加入 import
4. 在 `router.go` 的 route registration 块末尾加入 `xxx.RegisterRoutes(protected, db, logger)`

## Router 插入位置

import 块：在 `internal/domain/` 最后一个 import 之后（当前约 line 85）
routes 块：在最后一个 `RegisterRoutes` 之后（约 line 703），`// Wire the loaded carrier-rate...` 注释之前

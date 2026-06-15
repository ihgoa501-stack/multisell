# Permissions And Audit Guide

## 目标

MultiSell 的权限和审计目标是：

- 未登录用户不能操作受保护接口。
- 普通用户只能执行自己角色拥有权限的操作。
- 管理员可以执行所有操作。
- 所有关键写操作都留下操作日志，便于追责和排查。

## 核心文件

- `backend/app/auth/dependencies.py`
- `backend/app/auth/service.py`
- `backend/app/auth/router.py`
- `backend/app/rbac/service.py`
- `backend/app/rbac/router.py`
- `backend/app/operation_log/service.py`
- `backend/app/operation_log/router.py`

## 认证规则

### `AUTH_ENABLED=False`

用于本地开发和测试兼容。

行为：

- 不检查 token。
- 返回 mock 系统用户。
- mock 用户角色是 `admin`。

### `AUTH_ENABLED=True`

用于生产或真实权限测试。

行为：

- 请求必须携带 `Authorization: Bearer <token>`。
- token 无效返回 HTTP 401。
- 用户禁用返回 HTTP 403。
- 权限不足返回 HTTP 403。

## 权限依赖

业务接口应使用：

```python
from fastapi import Depends
from app.auth import require_permission
from app.models import User


@router.post("/example")
async def create_example(
    current_user: User = Depends(require_permission("example:create")),
):
    ...
```

规则：

- `current_user.role == "admin"`：直接通过。
- 普通用户：通过 `UserRole -> Role -> RolePermission -> Permission` 查询权限。
- 权限码匹配 `Permission.code`。

## 已接入的权限码

当前已覆盖主业务模块：

| 模块 | 权限码 | 审计日志 |
| --- | --- | --- |
| 商品 | `product:view`, `product:create`, `product:update`, `product:delete`, `product:import`, `product:export`, `product:ai` | 已覆盖 |
| 分类 | `category:view`, `category:create`, `category:update`, `category:delete` | 已覆盖 |
| 品牌 | `brand:view`, `brand:create`, `brand:update`, `brand:delete` | 已覆盖 |
| SKU | `sku:view`, `sku:create`, `sku:update`, `sku:delete` | 已覆盖 |
| 价格 | `price:view`, `price:update`, `price:batch_update` | 已覆盖 |
| 库存 | `inventory:view`, `inventory:update`, `inventory:adjust` | 已覆盖 |
| 供应商 | `supplier:view`, `supplier:create`, `supplier:update`, `supplier:delete` | 已覆盖 |
| 平台 | `platform:view`, `platform:create`, `platform:update`, `platform:delete` | 已覆盖 |
| 发布 | `listing:view`, `listing:publish`, `listing:sync`, `listing:task_manage` | 已覆盖 |
| 上架任务 | `listing:view`, `listing:task_manage`, `listing:publish` | create_from_decision, recheck, cancel, publish |
| 上架决策 | `decision:calculate` | 无写操作 |
| 物流运费 | `shipping:view`, `shipping:manage`, `shipping:calculate` | 已覆盖 |
| 平台费用规则 | `platform_fee:view`, `platform_fee:manage`, `platform_fee:calculate` | 已覆盖 |
| 订单 | `order:view`, `order:create`, `order:update`, `order:update_status`, `order:cancel` | 已覆盖 |
| 搜索 | `search:view` | 无写操作 |

## 审计日志规则

状态变化接口成功后应调用：

```python
from app.operation_log.service import OperationLogService

await OperationLogService.log(
    db,
    module="product",
    action="create",
    resource_id=str(product.id),
    content=f"创建商品: {product.name}",
    operator=current_user.username,
)
```

建议字段含义：

| 字段 | 说明 |
| --- | --- |
| `module` | 模块名，如 `product`、`order`、`listing` |
| `action` | 操作，如 `create`、`update`、`delete`、`publish` |
| `resource_id` | 被操作资源 ID |
| `content` | 简短可读说明 |
| `operator` | 操作人用户名 |
| `ip` | 后续可从 Request 中补充 |
| `duration` | 后续可记录接口耗时 |

## 当前审计覆盖范围

已覆盖主业务写操作模块：

| 模块 | 审计操作 |
| --- | --- |
| 商品 | create, update, delete, batch_update_status, batch_delete, duplicate, import, export, ai_enhance |
| 分类 | create, update, delete |
| 品牌 | create, update, delete |
| 订单 | create, update_status, cancel, bind_shipping_quote, update_profit_inputs |
| 库存 | update |
| 价格 | set_price, batch_update |
| SKU | define_specs, generate_skus, update |
| 供应商 | create, update, delete, bind_product, unbind_product |
| 平台 | create, update, delete |
| 平台费用规则 | create, update, delete |
| 发布 | publish, publish_failed |
| 上架任务 | create_from_decision, recheck, cancel, publish |
| 运费账单 | import, reconcile, resolve |

## 测试

权限和审计集成测试：

```bash
cd backend
python3 -m pytest tests/test_auth_rbac_audit_integration.py -q
```

测试覆盖：

- 开启鉴权后，未登录创建商品返回 401。
- 普通用户没有 `product:create` 返回 403。
- 拥有 `product:create` 后可以创建商品，并写入操作日志。
- `admin` 角色不需要显式权限即可创建商品。

## 接入新模块的步骤

1. 先写测试，验证未登录返回 401。
2. 写测试，验证普通用户无权限返回 403。
3. 写测试，验证授予权限后操作成功。
4. 写测试，验证操作成功后产生审计日志。
5. 在路由函数参数中加入 `Depends(require_permission("<module>:<action>"))`。
6. 操作成功后调用 `OperationLogService.log(...)`。
7. 跑局部测试和全量测试。

## 注意事项

- 不要只在前端隐藏按钮，后端必须强制校验权限。
- 不要把权限判断写死在每个路由里，统一使用 `require_permission(...)`。
- 不要在失败操作后写成功日志。
- 不要把平台 API key、用户密码 hash、token 等敏感数据写入 `content`。

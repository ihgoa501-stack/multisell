"""RBAC - 路由"""

from fastapi import APIRouter, Depends, Query
from sqlalchemy.ext.asyncio import AsyncSession
from app.database import get_db
from app.common import Result, PageResult
from app.rbac.schemas import RoleCreate, RoleUpdate, RoleVO, PermissionCreate, PermissionUpdate, PermissionVO, AssignRolesData
from app.rbac.service import RbacService
from app.auth import get_current_user
from app.config import settings
from app.models import User

router = APIRouter(prefix="/rbac", tags=["权限管理"])


def require_auth(current_user: User = Depends(get_current_user)):
    """当 AUTH_ENABLED=False 时跳过鉴权"""
    if not settings.AUTH_ENABLED:
        return None
    return current_user


# ==================== 角色接口 ====================


def role_to_vo(r) -> RoleVO:
    return RoleVO(
        id=r.id,
        name=r.name,
        code=r.code,
        description=r.description,
        status=r.status,
        permission_ids=[p.id for p in r.permissions] if hasattr(r, 'permissions') else [],
        created_at=r.created_at,
        updated_at=r.updated_at,
    )


@router.post("/roles", summary="创建角色")
async def create_role(data: RoleCreate, db: AsyncSession = Depends(get_db),
                      _=Depends(require_auth)):
    r = await RbacService.create_role(db, data.model_dump())
    return Result.ok(role_to_vo(r))


@router.put("/roles/{role_id}", summary="更新角色")
async def update_role(role_id: int, data: RoleUpdate, db: AsyncSession = Depends(get_db),
                      _=Depends(require_auth)):
    r = await RbacService.update_role(db, role_id, data.model_dump(exclude_unset=True))
    if not r:
        return Result.not_found("角色不存在")
    return Result.ok(role_to_vo(r))


@router.get("/roles/{role_id}", summary="角色详情")
async def get_role(role_id: int, db: AsyncSession = Depends(get_db),
                   _=Depends(require_auth)):
    r = await RbacService.get_role(db, role_id)
    if not r:
        return Result.not_found("角色不存在")
    return Result.ok(role_to_vo(r))


@router.get("/roles", summary="角色列表")
async def list_roles(
    name: str = Query(None, description="角色名称"),
    page: int = Query(1, ge=1),
    page_size: int = Query(20, ge=1, le=100),
    db: AsyncSession = Depends(get_db),
    _=Depends(require_auth),
):
    roles, total = await RbacService.list_roles(db, name, page, page_size)
    items = [role_to_vo(r) for r in roles]
    return PageResult.ok(items, total, page, page_size)


@router.delete("/roles/{role_id}", summary="删除角色")
async def delete_role(role_id: int, db: AsyncSession = Depends(get_db),
                      _=Depends(require_auth)):
    ok = await RbacService.delete_role(db, role_id)
    if not ok:
        return Result.not_found("角色不存在")
    return Result.ok(message="删除成功")


@router.post("/roles/{role_id}/permissions", summary="为角色分配权限")
async def assign_role_permissions(
    role_id: int, data: AssignRolesData, db: AsyncSession = Depends(get_db),
    _=Depends(require_auth),
):
    try:
        r = await RbacService.assign_permissions_to_role(db, role_id, data.role_ids)
        return Result.ok(role_to_vo(r))
    except ValueError as e:
        return Result.bad_request(str(e))


# ==================== 权限接口 ====================


def permission_to_vo(p) -> PermissionVO:
    return PermissionVO(
        id=p.id,
        name=p.name,
        code=p.code,
        description=p.description,
        module=p.module,
        created_at=p.created_at,
    )


@router.post("/permissions", summary="创建权限")
async def create_permission(data: PermissionCreate, db: AsyncSession = Depends(get_db),
                            _=Depends(require_auth)):
    p = await RbacService.create_permission(db, data.model_dump())
    return Result.ok(permission_to_vo(p))


@router.put("/permissions/{perm_id}", summary="更新权限")
async def update_permission(perm_id: int, data: PermissionUpdate, db: AsyncSession = Depends(get_db),
                            _=Depends(require_auth)):
    p = await RbacService.update_permission(db, perm_id, data.model_dump(exclude_unset=True))
    if not p:
        return Result.not_found("权限不存在")
    return Result.ok(permission_to_vo(p))


@router.get("/permissions/{perm_id}", summary="权限详情")
async def get_permission(perm_id: int, db: AsyncSession = Depends(get_db),
                         _=Depends(require_auth)):
    p = await RbacService.get_permission(db, perm_id)
    if not p:
        return Result.not_found("权限不存在")
    return Result.ok(permission_to_vo(p))


@router.get("/permissions", summary="权限列表")
async def list_permissions(
    name: str = Query(None, description="权限名称"),
    module: str = Query(None, description="所属模块"),
    page: int = Query(1, ge=1),
    page_size: int = Query(20, ge=1, le=100),
    db: AsyncSession = Depends(get_db),
    _=Depends(require_auth),
):
    perms, total = await RbacService.list_permissions(db, name, module, page, page_size)
    items = [permission_to_vo(p) for p in perms]
    return PageResult.ok(items, total, page, page_size)


@router.delete("/permissions/{perm_id}", summary="删除权限")
async def delete_permission(perm_id: int, db: AsyncSession = Depends(get_db),
                            _=Depends(require_auth)):
    ok = await RbacService.delete_permission(db, perm_id)
    if not ok:
        return Result.not_found("权限不存在")
    return Result.ok(message="删除成功")


# ==================== 用户-角色接口 ====================


@router.post("/users/{user_id}/roles", summary="为用户分配角色")
async def assign_user_roles(
    user_id: int, data: AssignRolesData, db: AsyncSession = Depends(get_db),
    _=Depends(require_auth),
):
    try:
        user = await RbacService.assign_roles_to_user(db, user_id, data.role_ids)

        # 返回用户信息（含角色）
        from app.auth.router import user_to_vo as auth_user_to_vo
        return Result.ok(auth_user_to_vo(user))
    except ValueError as e:
        return Result.bad_request(str(e))


@router.get("/users/{user_id}/permissions", summary="获取用户权限")
async def get_user_permissions(
    user_id: int, db: AsyncSession = Depends(get_db),
    _=Depends(require_auth),
):
    perms = await RbacService.get_user_permissions(db, user_id)
    items = [permission_to_vo(p) for p in perms]
    return Result.ok(items)


@router.get("/users", summary="用户列表（RBAC管理用）")
async def list_users(
    username: str = Query(None, description="用户名"),
    page: int = Query(1, ge=1),
    page_size: int = Query(20, ge=1, le=100),
    db: AsyncSession = Depends(get_db),
    _=Depends(require_auth),
):
    users, total = await RbacService.list_users(db, username, page, page_size)

    from app.auth.router import user_to_vo as auth_user_to_vo
    items = []
    for u in users:
        vo = auth_user_to_vo(u).model_dump()
        vo["role_ids"] = [r.id for r in u.roles] if hasattr(u, 'roles') else []
        vo["role_names"] = [r.name for r in u.roles] if hasattr(u, 'roles') else []
        items.append(vo)
    return PageResult.ok(items, total, page, page_size)

"""RBAC - 服务层"""
from typing import Optional
from sqlalchemy import select, func, delete
from sqlalchemy.ext.asyncio import AsyncSession
from app.models import Role, Permission, UserRole, RolePermission, User


class RbacService:

    # ========== 角色 CRUD ==========

    @staticmethod
    async def create_role(db: AsyncSession, data: dict) -> Role:
        role = Role(**data)
        db.add(role)
        await db.flush()
        await db.refresh(role)
        return role

    @staticmethod
    async def update_role(db: AsyncSession, role_id: int, data: dict) -> Optional[Role]:
        role = await db.get(Role, role_id)
        if not role:
            return None
        for key, value in data.items():
            if value is not None:
                setattr(role, key, value)
        await db.flush()
        await db.refresh(role)
        return role

    @staticmethod
    async def get_role(db: AsyncSession, role_id: int) -> Optional[Role]:
        return await db.get(Role, role_id)

    @staticmethod
    async def list_roles(db: AsyncSession, name: str = None,
                         page: int = 1, page_size: int = 20) -> tuple[list[Role], int]:
        stmt = select(Role)
        if name:
            stmt = stmt.where(Role.name.like(f"%{name}%"))
        count_stmt = select(func.count()).select_from(stmt.subquery())
        total = await db.scalar(count_stmt) or 0
        offset = (page - 1) * page_size
        stmt = stmt.order_by(Role.id).offset(offset).limit(page_size)
        result = await db.execute(stmt)
        return list(result.scalars().all()), total

    @staticmethod
    async def delete_role(db: AsyncSession, role_id: int) -> bool:
        role = await db.get(Role, role_id)
        if not role:
            return False
        # 删除关联
        await db.execute(delete(UserRole).where(UserRole.role_id == role_id))
        await db.execute(delete(RolePermission).where(RolePermission.role_id == role_id))
        await db.delete(role)
        await db.flush()
        return True

    # ========== 权限 CRUD ==========

    @staticmethod
    async def create_permission(db: AsyncSession, data: dict) -> Permission:
        perm = Permission(**data)
        db.add(perm)
        await db.flush()
        await db.refresh(perm)
        return perm

    @staticmethod
    async def update_permission(db: AsyncSession, perm_id: int, data: dict) -> Optional[Permission]:
        perm = await db.get(Permission, perm_id)
        if not perm:
            return None
        for key, value in data.items():
            if value is not None:
                setattr(perm, key, value)
        await db.flush()
        await db.refresh(perm)
        return perm

    @staticmethod
    async def get_permission(db: AsyncSession, perm_id: int) -> Optional[Permission]:
        return await db.get(Permission, perm_id)

    @staticmethod
    async def list_permissions(db: AsyncSession, name: str = None, module: str = None,
                               page: int = 1, page_size: int = 20) -> tuple[list[Permission], int]:
        stmt = select(Permission)
        if name:
            stmt = stmt.where(Permission.name.like(f"%{name}%"))
        if module:
            stmt = stmt.where(Permission.module == module)
        count_stmt = select(func.count()).select_from(stmt.subquery())
        total = await db.scalar(count_stmt) or 0
        offset = (page - 1) * page_size
        stmt = stmt.order_by(Permission.id).offset(offset).limit(page_size)
        result = await db.execute(stmt)
        return list(result.scalars().all()), total

    @staticmethod
    async def delete_permission(db: AsyncSession, perm_id: int) -> bool:
        perm = await db.get(Permission, perm_id)
        if not perm:
            return False
        # 删除关联
        await db.execute(delete(RolePermission).where(RolePermission.permission_id == perm_id))
        await db.delete(perm)
        await db.flush()
        return True

    # ========== 用户角色分配 ==========

    @staticmethod
    async def assign_roles_to_user(db: AsyncSession, user_id: int, role_ids: list[int]) -> User:
        user = await db.get(User, user_id)
        if not user:
            raise ValueError("用户不存在")
        # 清除旧关联
        await db.execute(delete(UserRole).where(UserRole.user_id == user_id))
        # 添加新关联
        for rid in role_ids:
            role = await db.get(Role, rid)
            if role and role.status == 1:
                db.add(UserRole(user_id=user_id, role_id=rid))
        await db.flush()
        await db.refresh(user)
        return user

    @staticmethod
    async def get_user_permissions(db: AsyncSession, user_id: int) -> list[Permission]:
        """获取用户所有权限（通过角色）"""
        stmt = (
            select(Permission)
            .join(RolePermission, RolePermission.permission_id == Permission.id)
            .join(UserRole, UserRole.role_id == RolePermission.role_id)
            .where(UserRole.user_id == user_id, Permission.id.isnot(None))
        )
        result = await db.execute(stmt)
        # 去重
        seen = set()
        perms = []
        for p in result.scalars().all():
            if p.id not in seen:
                seen.add(p.id)
                perms.append(p)
        return perms

    @staticmethod
    async def get_role_permissions(db: AsyncSession, role_id: int) -> list[Permission]:
        role = await db.get(Role, role_id)
        if not role:
            return []
        return role.permissions

    @staticmethod
    async def assign_permissions_to_role(db: AsyncSession, role_id: int, permission_ids: list[int]) -> Role:
        role = await db.get(Role, role_id)
        if not role:
            raise ValueError("角色不存在")
        # 清除旧关联
        await db.execute(delete(RolePermission).where(RolePermission.role_id == role_id))
        # 添加新关联
        for pid in permission_ids:
            perm = await db.get(Permission, pid)
            if perm:
                db.add(RolePermission(role_id=role_id, permission_id=pid))
        await db.flush()
        await db.refresh(role)
        return role

    @staticmethod
    async def list_users(db: AsyncSession, username: str = None,
                         page: int = 1, page_size: int = 20) -> tuple[list[User], int]:
        stmt = select(User)
        if username:
            stmt = stmt.where(User.username.like(f"%{username}%"))
        count_stmt = select(func.count()).select_from(stmt.subquery())
        total = await db.scalar(count_stmt) or 0
        offset = (page - 1) * page_size
        stmt = stmt.order_by(User.id).offset(offset).limit(page_size)
        result = await db.execute(stmt)
        return list(result.scalars().all()), total

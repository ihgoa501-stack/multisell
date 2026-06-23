"""异常工作台 - 服务层"""

from datetime import datetime, timezone
from typing import Optional

from sqlalchemy import select
from sqlalchemy.ext.asyncio import AsyncSession

from app.models import ExceptionItem
from app.operation_log.service import OperationLogService


class ExceptionService:
    # ── List / Get ─────────────────────────────────────────────────────

    @staticmethod
    async def list_items(
        db: AsyncSession,
        source_module: Optional[str] = None,
        severity: Optional[str] = None,
        status: Optional[str] = None,
    ) -> list[dict]:
        stmt = select(ExceptionItem).order_by(ExceptionItem.created_at.desc())
        if source_module:
            stmt = stmt.where(ExceptionItem.source_module == source_module)
        if severity:
            stmt = stmt.where(ExceptionItem.severity == severity)
        if status:
            stmt = stmt.where(ExceptionItem.status == status)

        result = await db.execute(stmt)
        items = result.scalars().all()
        return [ExceptionService._to_dict(it) for it in items]

    @staticmethod
    async def get_item(db: AsyncSession, exception_id: int) -> Optional[dict]:
        item = await db.get(ExceptionItem, exception_id)
        if not item:
            return None
        return ExceptionService._to_dict(item)

    @staticmethod
    def _to_dict(item: ExceptionItem) -> dict:
        return {
            "id": item.id,
            "source_module": item.source_module,
            "source_type": item.source_type,
            "source_id": item.source_id,
            "severity": item.severity,
            "status": item.status,
            "title": item.title,
            "description": item.description,
            "recommended_action": item.recommended_action,
            "assigned_to": item.assigned_to,
            "resolved_at": item.resolved_at.isoformat() if item.resolved_at else None,
            "resolved_by": item.resolved_by,
            "note": item.note,
            "created_at": item.created_at.isoformat() if item.created_at else None,
            "updated_at": item.updated_at.isoformat() if item.updated_at else None,
        }

    # ── Actions ────────────────────────────────────────────────────────

    @staticmethod
    async def assign_item(
        db: AsyncSession,
        exception_id: int,
        assigned_to: str,
        operator: Optional[str] = None,
    ) -> Optional[dict]:
        item = await db.get(ExceptionItem, exception_id)
        if not item:
            return None
        item.status = "assigned"
        item.assigned_to = assigned_to
        await db.flush()
        await db.refresh(item)

        await OperationLogService.log(
            db,
            module="exception",
            action="assign",
            resource_id=str(exception_id),
            content=f"分配异常给 {assigned_to}: {item.title}",
            operator=operator or "system",
        )
        return ExceptionService._to_dict(item)

    @staticmethod
    async def resolve_item(
        db: AsyncSession,
        exception_id: int,
        note: str = "",
        operator: Optional[str] = None,
    ) -> Optional[dict]:
        item = await db.get(ExceptionItem, exception_id)
        if not item:
            return None
        item.status = "resolved"
        item.resolved_at = datetime.now(timezone.utc)
        item.resolved_by = operator
        if note:
            item.note = note
        await db.flush()
        await db.refresh(item)

        await OperationLogService.log(
            db,
            module="exception",
            action="resolve",
            resource_id=str(exception_id),
            content=f"解决异常: {item.title}",
            operator=operator or "system",
        )
        return ExceptionService._to_dict(item)

    @staticmethod
    async def ignore_item(
        db: AsyncSession,
        exception_id: int,
        note: str = "",
        operator: Optional[str] = None,
    ) -> Optional[dict]:
        item = await db.get(ExceptionItem, exception_id)
        if not item:
            return None
        item.status = "ignored"
        if note:
            item.note = note
        await db.flush()
        await db.refresh(item)

        await OperationLogService.log(
            db,
            module="exception",
            action="ignore",
            resource_id=str(exception_id),
            content=f"忽略异常: {item.title}",
            operator=operator or "system",
        )
        return ExceptionService._to_dict(item)

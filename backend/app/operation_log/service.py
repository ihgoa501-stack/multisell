"""操作日志 - 服务层"""

from sqlalchemy import select, func
from sqlalchemy.ext.asyncio import AsyncSession
from app.models import OperationLog


class OperationLogService:

    @staticmethod
    async def list_logs(db: AsyncSession, module: str = None, action: str = None,
                         operator: str = None, page: int = 1, page_size: int = 20) -> tuple[list[OperationLog], int]:
        stmt = select(OperationLog)

        if module:
            stmt = stmt.where(OperationLog.module == module)
        if action:
            stmt = stmt.where(OperationLog.action.like(f"%{action}%"))
        if operator:
            stmt = stmt.where(OperationLog.operator == operator)

        count_stmt = select(func.count()).select_from(stmt.subquery())
        total = await db.scalar(count_stmt) or 0

        offset = (page - 1) * page_size
        stmt = stmt.order_by(OperationLog.created_at.desc()).offset(offset).limit(page_size)

        result = await db.execute(stmt)
        logs = result.scalars().all()

        return list(logs), total

    @staticmethod
    async def log(db: AsyncSession, module: str, action: str, resource_id: str = None,
                  content: str = None, operator: str = "system", ip: str = None, duration: int = None):
        """记录操作日志"""
        log = OperationLog(
            module=module,
            action=action,
            resource_id=resource_id,
            content=content,
            operator=operator,
            ip=ip,
            duration=duration,
        )
        db.add(log)
        await db.flush()
        return log

"""Agent decision archive service — moves old decisions to archived state"""
import logging
from datetime import datetime, timezone, timedelta
from sqlalchemy import select, update
from sqlalchemy.ext.asyncio import AsyncSession
from app.agent.models import AgentDecision

logger = logging.getLogger(__name__)

RETENTION_DAYS = 90


async def archive_old_decisions(db: AsyncSession, dry_run: bool = False) -> dict:
    """
    Archive agent decisions older than RETENTION_DAYS.
    Sets archived=true, archived_at=NOW().
    Returns count of archived records.
    """
    cutoff = datetime.now(timezone.utc) - timedelta(days=RETENTION_DAYS)

    # Count first
    count_stmt = select(AgentDecision.id).where(
        AgentDecision.created_at < cutoff,
        AgentDecision.archived == False,
    )
    count_result = await db.execute(count_stmt)
    total = len(count_result.all())

    if dry_run:
        return {"archived_count": total, "dry_run": True}

    if total == 0:
        return {"archived_count": 0}

    # Archive in batches to avoid long-running transactions
    batch_size = 500
    archived = 0
    for offset in range(0, total, batch_size):
        stmt = (
            update(AgentDecision)
            .where(
                AgentDecision.id.in_(
                    select(AgentDecision.id)
                    .where(AgentDecision.created_at < cutoff, AgentDecision.archived == False)
                    .limit(batch_size)
                    .offset(offset)
                )
            )
            .values(archived=True, archived_at=datetime.now(timezone.utc))
        )
        await db.execute(stmt)
        await db.flush()
        archived += batch_size

    await db.commit()
    return {"archived_count": archived}

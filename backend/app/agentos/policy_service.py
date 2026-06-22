"""Policy Matrix CRUD — conflict resolution rules"""
import logging
from sqlalchemy import select, and_, or_, func
from sqlalchemy.ext.asyncio import AsyncSession

from app.agent.models import ConflictPolicy

logger = logging.getLogger(__name__)


class PolicyService:
    """Conflict policy matrix — user-defined arbitration rules."""

    @staticmethod
    async def list_policies(
        db: AsyncSession,
        user_id: int,
        agent_id: str | None = None,
        page: int = 1,
        page_size: int = 20,
    ) -> tuple[list[ConflictPolicy], int]:
        stmt = select(ConflictPolicy).where(ConflictPolicy.user_id == user_id)
        if agent_id:
            stmt = stmt.where(
                or_(
                    ConflictPolicy.agent_id_a == agent_id,
                    ConflictPolicy.agent_id_b == agent_id,
                )
            )
        count_stmt = select(func.count()).select_from(stmt.subquery())
        total = (await db.execute(count_stmt)).scalar() or 0
        stmt = (
            stmt.order_by(ConflictPolicy.priority.desc())
            .offset((page - 1) * page_size)
            .limit(page_size)
        )
        result = await db.execute(stmt)
        return list(result.scalars().all()), total

    @staticmethod
    async def create_policy(
        db: AsyncSession, user_id: int, data: dict
    ) -> ConflictPolicy:
        """Create a conflict policy with hard limit check (max 20 per agent pair)."""
        count_stmt = select(func.count()).where(
            ConflictPolicy.user_id == user_id,
            ConflictPolicy.agent_id_a == data["agent_id_a"],
            ConflictPolicy.agent_id_b == data["agent_id_b"],
        )
        count = (await db.execute(count_stmt)).scalar() or 0
        if count >= 20:
            raise ValueError(
                f"Agent pair {data['agent_id_a']}/{data['agent_id_b']} "
                f"已达到20条策略上限"
            )
        policy = ConflictPolicy(user_id=user_id, **data)
        db.add(policy)
        await db.flush()
        await db.refresh(policy)
        return policy

    @staticmethod
    async def delete_policy(
        db: AsyncSession, policy_id: int, user_id: int
    ) -> bool:
        policy = await db.get(ConflictPolicy, policy_id)
        if not policy or policy.user_id != user_id:
            return False
        await db.delete(policy)
        await db.flush()
        return True

    @staticmethod
    async def resolve_conflict(
        db: AsyncSession,
        agent_id_a: str,
        agent_id_b: str,
        decision_point: str,
        context: dict,
        user_id: int = 1,
    ) -> dict | None:
        """Resolve a conflict using Policy Matrix.

        Returns {winner, reason} or None if no matching policy found.
        """
        stmt = (
            select(ConflictPolicy)
            .where(
                ConflictPolicy.user_id == user_id,
                or_(
                    and_(
                        ConflictPolicy.agent_id_a == agent_id_a,
                        ConflictPolicy.agent_id_b == agent_id_b,
                    ),
                    and_(
                        ConflictPolicy.agent_id_a == agent_id_b,
                        ConflictPolicy.agent_id_b == agent_id_a,
                    ),
                ),
                or_(
                    ConflictPolicy.decision_point == decision_point,
                    ConflictPolicy.decision_point == "*",
                ),
            )
            .order_by(ConflictPolicy.priority.desc())
            .limit(1)
        )
        result = await db.execute(stmt)
        policy = result.scalar_one_or_none()
        if not policy:
            return None
        if policy.condition:
            for key, val in policy.condition.items():
                if context.get(key) != val:
                    return None
        return {"winner": policy.winner, "reason": policy.reason}

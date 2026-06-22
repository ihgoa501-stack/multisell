"""Auto-arbitration — detect and resolve conflicting proposals"""
import logging
from datetime import datetime, timezone, timedelta
from sqlalchemy import select, func
from sqlalchemy.ext.asyncio import AsyncSession

from app.agentos.models import ActionProposal as APModel
from app.agent.models import ArbitrationLog
from app.agent.agents.arbiter import G0ArbiterAgent
from app.agentos.policy_service import PolicyService

logger = logging.getLogger(__name__)

CONFLICT_WINDOW_MINUTES = 30


class AutoArbitrator:
    """Detect conflicts between ActionProposals and trigger arbitration."""

    @staticmethod
    async def detect_and_arbitrate(
        db: AsyncSession, new_proposal: APModel, user_id: int = 1
    ) -> None:
        """Detect conflicts within the window and trigger arbitration if needed."""
        cutoff = datetime.now(timezone.utc) - timedelta(
            minutes=CONFLICT_WINDOW_MINUTES
        )

        # Find conflicting proposals (same business object, different agents,
        # within window)
        stmt = (
            select(APModel)
            .where(
                APModel.business_object_type
                == new_proposal.business_object_type,
                APModel.business_object_id
                == new_proposal.business_object_id,
                APModel.id != new_proposal.id,
                APModel.created_at >= cutoff,
                APModel.status.in_(["suggested", "pending_approval"]),
                APModel.agent_id.isnot(None),
                APModel.agent_id != new_proposal.agent_id,
            )
            .order_by(APModel.created_at.asc())
        )
        result = await db.execute(stmt)
        conflicting = list(result.scalars().all())

        if not conflicting or not new_proposal.agent_id:
            return  # No conflict detected

        # Build conflict keys for logging
        conflict_keys = []
        for c in conflicting + [new_proposal]:
            if c.agent_id:
                conflict_keys.append(f"{c.agent_id}:{c.id}")

        # Step 1: Try Policy Matrix for each conflicting pair
        for c in conflicting:
            if not c.agent_id:
                continue
            matrix_result = await PolicyService.resolve_conflict(
                db,
                c.agent_id,
                new_proposal.agent_id,
                c.action_type or new_proposal.action_type or "",
                {},
                user_id=user_id,
            )
            if matrix_result:
                # Apply matrix result: reject losing proposals
                winner_id = matrix_result["winner"]
                for p in [c, new_proposal]:
                    if p.agent_id != winner_id:
                        p.status = "blocked_by_policy"
                        p.rejection_reason = matrix_result["reason"]

                await _log_arbitration(
                    db,
                    {
                        "business_type": new_proposal.business_object_type,
                        "business_id": new_proposal.business_object_id,
                        "conflict_keys": conflict_keys,
                        "stage": "policy",
                        "verdict": winner_id,
                        "resolved_by": "system",
                    },
                )
                logger.info(
                    "Policy Matrix resolved conflict for %s/%s: winner=%s",
                    new_proposal.business_object_type,
                    new_proposal.business_object_id,
                    winner_id,
                )
                return

        # Step 2: If Policy Matrix didn't cover it, try Arbiter G0
        arbiter = G0ArbiterAgent(user_id=user_id)
        arbiter_context = {
            "agent_id_a": conflicting[0].agent_id,
            "agent_id_b": new_proposal.agent_id,
            "decision_point": new_proposal.action_type or "arbitrate",
            "confidence_a": float(conflicting[0].confidence or 0.5),
            "confidence_b": float(new_proposal.confidence or 0.5),
        }

        arbiter_result = await arbiter.decide(
            "arbitrate", arbiter_context, db=db
        )

        await _log_arbitration(
            db,
            {
                "business_type": new_proposal.business_object_type,
                "business_id": new_proposal.business_object_id,
                "conflict_keys": conflict_keys,
                "stage": "arbiter",
                "verdict": arbiter_result.get("verdict"),
                "arbiter_output": arbiter_result,
                "resolved_by": "arbiter_G0",
            },
        )
        logger.info(
            "Arbiter G0 resolved conflict for %s/%s: %s",
            new_proposal.business_object_type,
            new_proposal.business_object_id,
            arbiter_result.get("verdict"),
        )


async def _log_arbitration(db: AsyncSession, data: dict) -> None:
    """Write to arbitration_logs table."""
    log = ArbitrationLog(**data)
    db.add(log)
    await db.flush()

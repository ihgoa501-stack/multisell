"""Layer 2 — SPC 统计过程控制

基于统计基线的决策指标异常检测:
  - acceptance_rate: 用户采纳率
  - confidence: Agent 置信度
  - override_rate: 规则覆盖率

控制规则:
  - 单点越 UCL/LCL (μ±3σ) → 失控告警
  - 单点越 UWL/LWL (μ±2σ) → 注意告警
  - 连续 7 点同侧 → 趋势异常
"""

import math
from datetime import datetime, timezone, timedelta
from typing import Optional
from sqlalchemy import select
from sqlalchemy.ext.asyncio import AsyncSession
from app.agent.models import AgentDecision, SpcControlLimit


class SpcController:
    RECALC_DAYS = 7
    BASELINE_DAYS = 30
    CONSECUTIVE_LIMIT = 7

    async def recalc_limits(
        self,
        db: AsyncSession,
        user_id: int,
        agent_id: str,
        decision_point: str,
        metric_name: str,
    ) -> Optional[SpcControlLimit]:
        window_start = datetime.now(timezone.utc) - timedelta(days=self.BASELINE_DAYS)

        values_stmt = select(
            AgentDecision.__table__.c.get(metric_name, AgentDecision.confidence)
        ).where(
            AgentDecision.user_id == user_id,
            AgentDecision.agent_id == agent_id,
            AgentDecision.decision_point == decision_point,
            AgentDecision.created_at >= window_start,
        )

        metric_col = getattr(
            AgentDecision,
            {
                "acceptance_rate": "confidence",
                "confidence": "confidence",
                "override_rate": "confidence",
            }.get(metric_name, "confidence"),
        )

        values_stmt = select(metric_col).where(
            AgentDecision.user_id == user_id,
            AgentDecision.agent_id == agent_id,
            AgentDecision.decision_point == decision_point,
            AgentDecision.created_at >= window_start,
        )
        result = await db.execute(values_stmt)
        values = [float(row[0]) for row in result if row[0] is not None]

        if len(values) < 3:
            return None

        n = len(values)
        mean = sum(values) / n
        variance = sum((v - mean) ** 2 for v in values) / n
        stddev = math.sqrt(variance)

        stmt = select(SpcControlLimit).where(
            SpcControlLimit.user_id == user_id,
            SpcControlLimit.agent_id == agent_id,
            SpcControlLimit.decision_point == decision_point,
            SpcControlLimit.metric_name == metric_name,
        )
        existing = await db.execute(stmt)
        limit = existing.scalar_one_or_none()

        now = datetime.now(timezone.utc)
        next_recalc = now + timedelta(days=self.RECALC_DAYS)

        data = {
            "baseline_mean": round(mean, 4),
            "baseline_stddev": round(stddev, 4),
            "baseline_samples": n,
            "ucl": round(mean + 3 * stddev, 4),
            "lcl": round(mean - 3 * stddev, 4),
            "uwl": round(mean + 2 * stddev, 4),
            "lwl": round(mean - 2 * stddev, 4),
            "baseline_recalc_at": now,
            "next_recalc_at": next_recalc,
        }

        if limit:
            for k, v in data.items():
                setattr(limit, k, v)
            await db.flush()
            await db.refresh(limit)
            return limit
        else:
            limit = SpcControlLimit(
                user_id=user_id,
                agent_id=agent_id,
                decision_point=decision_point,
                metric_name=metric_name,
                consecutive_same_side=0,
                **data,
            )
            db.add(limit)
            await db.flush()
            await db.refresh(limit)
            return limit

    async def check_point(
        self,
        db: AsyncSession,
        user_id: int,
        agent_id: str,
        decision_point: str,
        metric_name: str,
        current_value: float,
    ) -> dict:
        stmt = select(SpcControlLimit).where(
            SpcControlLimit.user_id == user_id,
            SpcControlLimit.agent_id == agent_id,
            SpcControlLimit.decision_point == decision_point,
            SpcControlLimit.metric_name == metric_name,
        )
        result = await db.execute(stmt)
        limit = result.scalar_one_or_none()

        if not limit:
            limit = await self.recalc_limits(
                db, user_id, agent_id, decision_point, metric_name
            )
            if not limit:
                return {
                    "status": "insufficient_data",
                    "message": "样本不足, 无法建立控制基线",
                }

        ucl, lcl = float(limit.ucl), float(limit.lcl)
        uwl, lwl = float(limit.uwl), float(limit.lwl)
        mean = float(limit.baseline_mean)

        if current_value > mean:
            side = "above"
        elif current_value < mean:
            side = "below"
        else:
            side = "center"

        if side != "center":
            if side == "above" and limit.consecutive_same_side >= 0:
                limit.consecutive_same_side += 1
            elif side == "below" and limit.consecutive_same_side <= 0:
                limit.consecutive_same_side -= 1
            else:
                limit.consecutive_same_side = 1 if side == "above" else -1
        else:
            limit.consecutive_same_side = 0

        alerts = []
        is_out_of_control = False
        is_warning = False

        if current_value > ucl or current_value < lcl:
            is_out_of_control = True
            limit.last_breach_at = datetime.now(timezone.utc)
            alerts.append(
                {
                    "level": "critical",
                    "message": f"{metric_name} 越控制线: {current_value:.4f} (限: [{lcl:.4f}, {ucl:.4f}])",
                }
            )

        if current_value > uwl or current_value < lwl:
            is_warning = True
            alerts.append(
                {
                    "level": "warning",
                    "message": f"{metric_name} 越警戒线: {current_value:.4f} (限: [{lwl:.4f}, {uwl:.4f}])",
                }
            )

        if abs(limit.consecutive_same_side) >= self.CONSECUTIVE_LIMIT:
            alerts.append(
                {
                    "level": "warning",
                    "message": f"连续 {abs(limit.consecutive_same_side)} 点同侧 ({side}), 趋势异常",
                }
            )

        await db.flush()

        return {
            "agent_id": agent_id,
            "decision_point": decision_point,
            "metric_name": metric_name,
            "current_value": current_value,
            "baseline_mean": mean,
            "ucl": float(ucl),
            "lcl": float(lcl),
            "uwl": float(uwl),
            "lwl": float(lwl),
            "baseline_samples": limit.baseline_samples,
            "consecutive_same_side": limit.consecutive_same_side,
            "is_out_of_control": is_out_of_control,
            "is_warning": is_warning,
            "alerts": alerts,
            "last_breach_at": limit.last_breach_at,
            "next_recalc_at": limit.next_recalc_at,
        }

    async def get_all_limits(
        self,
        db: AsyncSession,
        user_id: int,
    ) -> list[SpcControlLimit]:
        stmt = select(SpcControlLimit).where(SpcControlLimit.user_id == user_id)
        result = await db.execute(stmt)
        limits = result.scalars().all()

        for limit in limits:
            if datetime.now(timezone.utc) >= (
                limit.next_recalc_at or datetime.now(timezone.utc)
            ):
                await self.recalc_limits(
                    db,
                    user_id,
                    limit.agent_id,
                    limit.decision_point,
                    limit.metric_name,
                )

        return list(limits)

"""自治等级升级规则引擎 — 纯函数，无数据库依赖"""

from __future__ import annotations

from typing import Any

LEVEL_ORDER = ["OBSERVATION", "SUGGESTION", "SEMI_AUTONOMOUS", "FULL_AUTONOMOUS"]

RISK_SCORE = {"low": 1, "medium": 2, "high": 3, "critical": 4}


def suggest_upgrade(
    agent_id: str,
    current_level: str,
    success_rate: float,
    adoption_rate: float,
    recent_risk_levels: list[str],
    total_decisions: int,
    recent_errors: int,
) -> dict[str, Any]:
    """
    建议自治等级升降。

    返回:
        suggested: bool — 是否建议变更
        direction: "upgrade" | "downgrade" | None
        target_level: str — 目标等级
        confidence: float — 置信度 (0-1)
        reason: str — 理由说明
    """
    # 基础数据量检查
    if total_decisions < 10:
        return {
            "suggested": False,
            "direction": None,
            "target_level": current_level,
            "confidence": 0,
            "reason": "insufficient_data",
        }

    current_idx = LEVEL_ORDER.index(current_level) if current_level in LEVEL_ORDER else -1
    if current_idx == -1:
        return {
            "suggested": False,
            "direction": None,
            "target_level": current_level,
            "confidence": 0,
            "reason": "unknown_level",
        }

    # 计算风险得分
    risk_sum = sum(RISK_SCORE.get(r, 2) for r in recent_risk_levels)
    risk_avg = risk_sum / max(len(recent_risk_levels), 1)
    error_rate = recent_errors / max(total_decisions, 1)

    # 降级判断：高风险 + 低成功率
    if current_idx > 0 and (error_rate > 0.3 or (success_rate < 0.6 and risk_avg > 2.5)):
        target_idx = current_idx - 1
        return {
            "suggested": True,
            "direction": "downgrade",
            "target_level": LEVEL_ORDER[target_idx],
            "confidence": round(min(0.95, 0.5 + error_rate), 2),
            "reason": f"高风险(avg={risk_avg:.1f})/低成功率({success_rate:.0%})",
        }

    # 已达最高等级
    if current_idx >= len(LEVEL_ORDER) - 1:
        return {
            "suggested": False,
            "direction": None,
            "target_level": current_level,
            "confidence": 1.0,
            "reason": "already_at_max",
        }

    # 升级判断
    upgrade_signals = 0
    total_signals = 0

    # 信号1: 成功率 > 90%
    total_signals += 1
    if success_rate >= 0.9:
        upgrade_signals += 1

    # 信号2: 采纳率 > 70%
    total_signals += 1
    if adoption_rate >= 0.7:
        upgrade_signals += 1

    # 信号3: 最近无高风险
    total_signals += 1
    if risk_avg < 2.0:
        upgrade_signals += 1

    # 信号4: 错误率低
    total_signals += 1
    if error_rate < 0.05:
        upgrade_signals += 1

    confidence = upgrade_signals / max(total_signals, 1)

    if confidence >= 0.75 and current_idx < len(LEVEL_ORDER) - 1:
        target_idx = current_idx + 1
        return {
            "suggested": True,
            "direction": "upgrade",
            "target_level": LEVEL_ORDER[target_idx],
            "confidence": round(confidence, 2),
            "reason": f"高成功率({success_rate:.0%})/高采纳率({adoption_rate:.0%})",
        }

    return {
        "suggested": False,
        "direction": None,
        "target_level": current_level,
        "confidence": round(confidence, 2),
        "reason": "condition_not_met",
    }


def batch_suggest_upgrades(
    agents: list[dict[str, Any]],
) -> list[dict[str, Any]]:
    """批量计算多个 Agent 的升级建议"""
    return [
        {
            "agent_id": a.get("id", ""),
            "current_level": a.get("autonomy_level", "SUGGESTION"),
            **suggest_upgrade(
                agent_id=a.get("id", ""),
                current_level=a.get("autonomy_level", "SUGGESTION"),
                success_rate=a.get("success_rate", 0),
                adoption_rate=a.get("adoption_rate", 0),
                recent_risk_levels=a.get("recent_risk_levels", []),
                total_decisions=a.get("total_decisions", 0),
                recent_errors=a.get("recent_errors", 0),
            ),
        }
        for a in agents
    ]

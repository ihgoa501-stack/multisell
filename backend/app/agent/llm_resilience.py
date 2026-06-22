"""LLM 调用韧性层 — 熔断器、模型降级链、决策缓存"""
import functools
import json
import time
from hashlib import sha256
from typing import Any, Optional

import logging

from app.config import settings

logger = logging.getLogger(__name__)


# ──────────────────────────────────────────────
#  §5.1 熔断器
# ──────────────────────────────────────────────

class CircuitState:
    CLOSED = "closed"        # 正常
    OPEN = "open"            # 熔断中，快速拒绝
    HALF_OPEN = "half_open"  # 试探恢复


class CircuitBreaker:
    """单个 (agent_id, decision_point) 的熔断状态"""

    def __init__(self, key: str):
        self.key: str = key  # f"{agent_id}:{decision_point}"
        self.failure_count: int = 0
        self.last_failure_time: float = 0
        self.state: str = CircuitState.CLOSED

        # 可调参数
        self.FAILURE_THRESHOLD: int = 3        # 连续 N 次失败 → 熔断
        self.RESET_TIMEOUT: int = 300          # 5 分钟后尝试半开
        self.HALF_OPEN_MAX_TRIALS: int = 1     # 半开状态下允许的试探请求数


class LLMCircuitBreakerManager:
    """全局熔断器管理器（单例）"""
    _instance = None

    def __new__(cls):
        if cls._instance is None:
            cls._instance = super().__new__(cls)
            cls._instance._breakers: dict[str, CircuitBreaker] = {}
        return cls._instance

    def record_success(self, agent_id: str, decision_point: str):
        key = f"{agent_id}:{decision_point}"
        if key in self._breakers:
            self._breakers[key].failure_count = 0
            self._breakers[key].state = CircuitState.CLOSED

    def record_failure(self, agent_id: str, decision_point: str):
        key = f"{agent_id}:{decision_point}"
        now = time.time()
        if key not in self._breakers:
            self._breakers[key] = CircuitBreaker(key=key)
        cb = self._breakers[key]
        cb.failure_count += 1
        cb.last_failure_time = now
        if cb.failure_count >= cb.FAILURE_THRESHOLD:
            cb.state = CircuitState.OPEN
            logger.warning("CircuitBreaker OPEN: %s (failures=%d)", key, cb.failure_count)

    def can_proceed(self, agent_id: str, decision_point: str) -> bool:
        key = f"{agent_id}:{decision_point}"
        if key not in self._breakers:
            return True
        cb = self._breakers[key]
        if cb.state == CircuitState.CLOSED:
            return True
        if cb.state == CircuitState.OPEN:
            # 检查是否超时 → 进入半开
            if time.time() - cb.last_failure_time > cb.RESET_TIMEOUT:
                cb.state = CircuitState.HALF_OPEN
                logger.info("CircuitBreaker HALF_OPEN: %s", key)
                return True
            return False
        # HALF_OPEN: 允许一次试探
        return True


# ──────────────────────────────────────────────
#  §5.1 / §5.2 模型降级链 + 决策缓存
# ──────────────────────────────────────────────

MODEL_LADDER = ["gpt-4o", "gpt-4o-mini", "cached", "skip"]


def _make_cache_key(agent_id: str, decision_point: str, context: dict) -> str:
    raw = f"{agent_id}:{decision_point}:{json.dumps(context, sort_keys=True)}"
    return sha256(raw.encode()).hexdigest()


async def _get_cached_decision(
    db: Any,
    agent_id: str,
    decision_point: str,
    context: dict,
) -> Optional[dict]:
    """从 AgentDecisionCache 查找未过期的缓存决策"""
    if db is None:
        return None
    from datetime import datetime, timezone
    from sqlalchemy import select
    from app.agent.models import AgentDecisionCache

    cache_key = _make_cache_key(agent_id, decision_point, context)
    stmt = select(AgentDecisionCache).where(
        AgentDecisionCache.cache_key == cache_key,
        AgentDecisionCache.expires_at > datetime.now(timezone.utc),
    )
    result = await db.execute(stmt)
    cached = result.scalar_one_or_none()
    if cached:
        logger.debug("Cache HIT: %s", cache_key[:16])
        return cached.decision_json
    return None


async def _set_cached_decision(
    db: Any,
    agent_id: str,
    decision_point: str,
    context: dict,
    decision_json: dict,
    ttl: int = 300,
):
    """写入 AgentDecisionCache（upsert）"""
    if db is None:
        return
    from datetime import datetime, timezone, timedelta
    from app.agent.models import AgentDecisionCache

    cache_key = _make_cache_key(agent_id, decision_point, context)
    expires_at = datetime.now(timezone.utc) + timedelta(seconds=ttl)

    entry = AgentDecisionCache(
        cache_key=cache_key,
        decision_json=decision_json,
        expires_at=expires_at,
    )
    # cache_key 是 PK，merge 实现 upsert
    await db.merge(entry)
    await db.flush()


# ──────────────────────────────────────────────
#  §5.1 装饰器
# ──────────────────────────────────────────────

def llm_resilient(func):
    """装饰器: 为 BaseAgent.decide() 添加模型降级、熔断、缓存

    用法:
        @llm_resilient
        async def decide(self, decision_point, context, db=None):
            ...
    """
    @functools.wraps(func)
    async def wrapper(self, decision_point, context, db=None):
        cb_mgr = LLMCircuitBreakerManager()

        # 1. 检查熔断器
        if not cb_mgr.can_proceed(self.agent_id, decision_point):
            cached = await _get_cached_decision(db, self.agent_id, decision_point, context)
            if cached:
                return {**cached, "_from_cache": True, "_circuit_open": True}
            return {
                "confidence": 0.0,
                "error": "circuit_open",
                "summary": "断路器开启，跳过本轮",
            }

        # 2. 尝试缓存
        cached = await _get_cached_decision(db, self.agent_id, decision_point, context)
        if cached:
            return {**cached, "_from_cache": True}

        # 3. 模型降级链
        errors = []
        for model in MODEL_LADDER:
            if model == "cached":
                cached = await _get_cached_decision(db, self.agent_id, decision_point, context)
                if cached:
                    return {**cached, "_from_cache": True}
                continue
            if model == "skip":
                return {
                    "confidence": 0.0,
                    "error": "all_models_failed",
                    "summary": "所有模型均失败，跳过本轮",
                }

            try:
                # 切换 model 配置，实现真实降级
                original_model = getattr(settings, 'LLM_MODEL', None)
                settings.LLM_MODEL = model
                try:
                    result = await func(self, decision_point, context, db=db)
                finally:
                    settings.LLM_MODEL = original_model  # 始终恢复

                cb_mgr.record_success(self.agent_id, decision_point)
                result["_llm_model"] = model
                # 写入缓存
                await _set_cached_decision(
                    db, self.agent_id, decision_point, context, result
                )
                return result
            except Exception as e:
                errors.append({"model": model, "error": str(e)})
                cb_mgr.record_failure(self.agent_id, decision_point)
                logger.warning(
                    "LLM model=%s failed for %s/%s: %s",
                    model, self.agent_id, decision_point, e,
                )
                continue

        # 全部失败
        return {
            "confidence": 0.0,
            "error": str(errors),
            "summary": "所有LLM调用失败",
        }

    return wrapper

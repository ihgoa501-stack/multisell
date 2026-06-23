"""Agent 事件总线

业务系统内部事件 → 自动唤醒对应 Agent 决策。
实现进程内异步 pub/sub 模式，无需外部消息队列。

设计原则：
- 事件源无感：业务模块只需 emit(event_type, payload)，不感知 Agent
- 一次注册：事件到 Agent 的映射集中定义
- 异步非阻塞：事件分发在后台 Task 中执行，不阻塞业务操作
- 错误隔离：单个事件处理失败不影响其他事件或业务操作

用法:
    from app.agent.event_bus import event_bus

    # 业务模块中触发事件
    await event_bus.emit("inventory.low_stock", {
        "sku_code": "SKU001",
        "current_stock": 5,
        "safety_stock": 20,
    })
"""

import asyncio
import logging
from dataclasses import dataclass, field
from datetime import datetime, timezone
from typing import Callable, Optional

from app.database import async_session_factory
from app.agent.registry import AgentRegistry
from app.agent.service import AgentService
from app.agent.pipeline import evaluate_chains

logger = logging.getLogger(__name__)


@dataclass
class Event:
    """系统事件"""

    type: str
    payload: dict
    source: str = ""  # 来源模块标识
    created_at: str = field(
        default_factory=lambda: datetime.now(timezone.utc).isoformat()
    )


# ── 事件 → Agent 决策映射 ──────────────────────────────────
# (event_type, agent_id, decision_point, context_builder)

EventHandler = Callable[[dict, int], dict]


def _identity_ctx(payload: dict, user_id: int) -> dict:
    """直接透传事件 payload 作为决策上下文"""
    ctx = dict(payload)
    ctx.setdefault("_event_time", datetime.now(timezone.utc).isoformat())
    return ctx


def _stock_alert_ctx(payload: dict, user_id: int) -> dict:
    """库存事件 → A5 上下文"""
    return {
        "sku_code": payload.get("sku_code", ""),
        "sellable_stock": payload.get("current_stock", 0),
        "safety_stock": payload.get("safety_stock", 0),
        "sellable_days": payload.get("sellable_days", 0),
        "sales_7d": payload.get("sales_7d", 0),
        "_source_event": payload.get("_event_type", ""),
        "_event_time": datetime.now(timezone.utc).isoformat(),
    }


def _discount_ctx(payload: dict, user_id: int) -> dict:
    """折扣事件 → G3 上下文"""
    return {
        "sku_code": payload.get("sku_code", ""),
        "original_price": payload.get("original_price", 0),
        "proposed_discount": payload.get("proposed_discount", 0),
        "promotion_type": payload.get("promotion_type", ""),
        "_source_event": payload.get("_event_type", ""),
        "_event_time": datetime.now(timezone.utc).isoformat(),
    }


# ── 事件路由表 ──────────────────────────────────────────────
# 每个条目: (event_type_prefix, agent_id, decision_point, context_builder)

EVENT_ROUTES: list[tuple[str, str, str, EventHandler]] = [
    # 库存事件 → A5
    ("inventory.low_stock", "A5", "stock_alert", _stock_alert_ctx),
    ("inventory.out_of_stock", "A5", "stock_alert", _stock_alert_ctx),
    ("inventory.recovered", "A5", "stock_alert", _stock_alert_ctx),
    # 订单事件 → A4 客服
    ("order.exception", "A4", "customer_service", _identity_ctx),
    ("order.refund", "A4", "customer_service", _identity_ctx),
    # 价格/折扣事件 → G3 风控
    ("price.changed", "G3", "discount_risk_check", _discount_ctx),
    ("discount.proposed", "G3", "discount_risk_check", _discount_ctx),
    ("promotion.created", "G3", "discount_risk_check", _discount_ctx),
    # 上架事件 → A2 优化
    ("listing.failed", "A2", "listing_optimize", _identity_ctx),
    ("listing.created", "A2", "listing_optimize", _identity_ctx),
    # 财务事件 → A6 利润监控
    ("finance.profit_anomaly", "A6", "profit_watch", _identity_ctx),
    ("settlement.discrepancy", "A6", "profit_watch", _identity_ctx),
    # 新产品 → A1 侦查
    ("product.created", "A1", "product_scout", _identity_ctx),
    # 合规事件 → A7
    ("compliance.alert", "A7", "compliance_check", _identity_ctx),
    ("platform.rule_changed", "A7", "compliance_check", _identity_ctx),
    # 通关事件 → G2
    ("customs.delay", "G2", "customs_advice", _identity_ctx),
    ("customs.cleared", "G2", "customs_advice", _identity_ctx),
]


class AgentEventBus:
    """进程内 Agent 事件总线

    线程安全：asyncio 单线程模型 + 异步 Task 分发。
    """

    def __init__(self):
        self._handlers: list[tuple[str, str, str, EventHandler]] = list(EVENT_ROUTES)
        self._custom_handlers: list[Callable[[Event], bool]] = []
        self._running = False
        self._system_user_id = 1

    # ── 配置 ────────────────────────────────────────────────

    def set_system_user(self, user_id: int):
        """设置系统调度用户 ID"""
        self._system_user_id = user_id

    def add_route(
        self,
        event_type: str,
        agent_id: str,
        decision_point: str,
        context_builder: Optional[EventHandler] = None,
    ):
        """动态添加事件路由"""
        self._handlers.append(
            (
                event_type,
                agent_id,
                decision_point,
                context_builder or _identity_ctx,
            )
        )

    def add_custom_handler(self, handler: Callable[[Event], bool]) -> None:
        """添加自定义事件处理器，返回 True 表示已处理"""
        self._custom_handlers.append(handler)

    # ── 事件分发 ────────────────────────────────────────────

    async def emit(self, event_type: str, payload: dict, source: str = "") -> int:
        """触发事件，返回匹配到的路由数"""
        event = Event(type=event_type, payload=payload, source=source)
        logger.debug("事件触发: %s from %s", event_type, source)

        # 1) 自定义处理器（优先）
        for handler in self._custom_handlers:
            try:
                handled = handler(event)
                if handled:
                    return 1  # 自定义处理器已处理，跳过 Agent 路由
            except Exception:
                logger.exception("自定义事件处理器失败: %s", event_type)

        # 2) 匹配 Agent 路由
        matched = 0
        for route_type, agent_id, decision_point, ctx_builder in self._handlers:
            if not _match_event(event_type, route_type):
                continue

            matched += 1
            # 异步分发，不阻塞 emit 调用方
            asyncio.create_task(
                self._dispatch(event, agent_id, decision_point, ctx_builder),
                name=f"event-{agent_id}-{decision_point}",
            )

        if matched == 0:
            logger.debug("事件 %s 无匹配路由", event_type)

        return matched

    async def _dispatch(
        self,
        event: Event,
        agent_id: str,
        decision_point: str,
        ctx_builder: EventHandler,
    ):
        """将事件派发给指定的 Agent"""
        try:
            agent_cls = AgentRegistry.get_agent_class(agent_id)
            if not agent_cls:
                logger.warning("事件目标 Agent %s 未注册", agent_id)
                return

            ctx = ctx_builder(event.payload, self._system_user_id)
            ctx["_event_type"] = event.type
            ctx["_event_source"] = event.source

            async with async_session_factory() as db:
                agent = agent_cls(user_id=self._system_user_id)
                result = await AgentService.execute_decision(
                    db,
                    agent,
                    decision_point,
                    ctx,
                    dry_run=False,
                )

                # 触发协作链
                if result.get("decision_id"):
                    await evaluate_chains(
                        agent_id,
                        decision_point,
                        result,
                        self._system_user_id,
                        db,
                    )

                logger.info(
                    "事件 %s → %s[%s] 决策完成 (decision_id=%s)",
                    event.type,
                    agent_id,
                    decision_point,
                    result.get("decision_id"),
                )

        except Exception as e:
            logger.exception(
                "事件分发失败: %s → %s[%s]: %s",
                event.type,
                agent_id,
                decision_point,
                e,
            )

    def get_routes(self) -> list[dict]:
        """获取所有事件路由（只读）"""
        return [
            {"event_type": r[0], "agent_id": r[1], "decision_point": r[2]}
            for r in self._handlers
        ]


def _match_event(actual: str, pattern: str) -> bool:
    """事件类型匹配（支持前缀通配）"""
    if pattern.endswith(".*"):
        return actual.startswith(pattern[:-1])
    if pattern.endswith("*"):
        return actual.startswith(pattern[:-1])
    return actual == pattern or actual.startswith(pattern)


# ── 全局单例 ──────────────────────────────────────────────────

event_bus = AgentEventBus()

"""事件集成入口 — 业务模块通过此模块触发 Agent 事件

用法:
    from app.events import emit_agent_event

    await emit_agent_event("inventory.low_stock", {
        "sku_code": "SKU001",
        "current_stock": 5,
        "safety_stock": 20,
    })
"""

import logging
from typing import Optional

logger = logging.getLogger(__name__)


async def emit_agent_event(
    event_type: str,
    payload: Optional[dict] = None,
    source: str = "",
) -> int:
    """触发 Agent 事件，返回匹配到的路由数"""
    try:
        from app.agent.event_bus import event_bus

        return await event_bus.emit(event_type, payload or {}, source)
    except Exception as e:
        logger.warning("Agent 事件发送失败 (%s): %s", event_type, e)
        return 0

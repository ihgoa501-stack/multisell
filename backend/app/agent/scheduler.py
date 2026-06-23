"""Agent 定时调度引擎

让 Agent 按预设策略周期自动决策，无需人工调用 API。
配合 AgentActionService，决策产生的可执行操作进入待办列表等待确认。

调度设计：
- 在 FastAPI lifespan 中启停
- 每个 Agent 独立的调度间隔（秒）
- 系统级运行（user_id=1），后端管理员用户
- 错误隔离：单个 Agent 失败不影响其他 Agent
- 支持手动触发单次调度
"""

import asyncio
import logging
from datetime import datetime, timezone, timedelta
from typing import Optional, Union

from sqlalchemy import select, func

from app.database import async_session_factory
from app.agent.registry import AgentRegistry
from app.agent.service import AgentService
from app.agent.pipeline import evaluate_chains
from app.agent.evolution_service import EvolutionService
from app.agent.entropy.service import EntropyService
from app.models import Sku, Inventory, Product, Order

logger = logging.getLogger(__name__)

# ── 默认调度配置 ──────────────────────────────────────────────
# interval_seconds: 轮询间隔
# enabled: 是否自动启动
# decision_points: 触发的决策点列表
# context_builder: 上下文构建函数名（在 SchedulerContextBuilder 中定义）
# max_skus: 单次最多处理的 SKU 数量

DEFAULT_SCHEDULES: dict[str, dict] = {
    "G1": {
        "interval": 300,
        "enabled": True,
        "decision_points": ["dashboard_overview"],
        "description": "运营驾驶舱 — 每 5 分钟聚合仪表盘数据",
    },
    "G2": {
        "interval": 3600,
        "enabled": True,
        "decision_points": ["customs_advice"],
        "description": "通关建议 — 每 1 小时检查待通关订单",
    },
    "G3": {
        "interval": 1800,
        "enabled": True,
        "decision_points": ["discount_risk_check"],
        "description": "折扣风控 — 每 30 分钟扫描折扣风险",
    },
    "A5": {
        "interval": 900,
        "enabled": True,
        "decision_points": ["stock_alert"],
        "description": "库存预警 — 每 15 分钟检查低库存 SKU",
    },
    "A6": {
        "interval": 3600,
        "enabled": True,
        "decision_points": ["profit_watch"],
        "description": "利润监控 — 每 1 小时分析亏损/低毛利 SKU",
    },
    "A7": {
        "interval": 7200,
        "enabled": True,
        "decision_points": ["compliance_check"],
        "description": "合规检查 — 每 2 小时扫描违规风险",
    },
    "A1": {
        "interval": 86400,
        "enabled": True,
        "decision_points": ["product_scout"],
        "description": "产品侦查 — 每天扫描市场趋势",
    },
    "A2": {
        "interval": 43200,
        "enabled": True,
        "decision_points": ["listing_optimize"],
        "description": "Listing 优化 — 每 12 小时检查优化机会",
    },
    "A3": {
        "interval": 3600,
        "enabled": True,
        "decision_points": ["acos_adjustment"],
        "description": "广告建议 — 每 1 小时分析广告效果",
    },
    "A4": {
        "interval": 300,
        "enabled": True,
        "decision_points": ["customer_service"],
        "description": "客服助手 — 每 5 分钟检查待回复消息",
    },
}

# ── 熵系统配置 ──────────────────────────────────────────────
ENTROPY_INTERVAL = 21600  # 6 小时运行一次熵防御


class SchedulerContextBuilder:
    """为各 Agent 构建决策上下文"""

    @staticmethod
    async def build_stock_alert_contexts(db, max_skus: int = 50) -> list[dict]:
        """A5: 查询库存低于预警线的 SKU，每个 SKU 生成一个上下文"""
        stmt = (
            select(Sku, Inventory)
            .join(Inventory, Sku.id == Inventory.sku_id)
            .where(
                Inventory.quantity <= Inventory.safety_stock,
                Inventory.safety_stock > 0,
            )
            .limit(max_skus)
        )
        result = await db.execute(stmt)
        rows = result.all()
        contexts = []
        for sku, inv in rows:
            contexts.append(
                {
                    "sku_code": sku.code or sku.spec_desc or f"SKU#{sku.id}",
                    "sellable_stock": inv.quantity or 0,
                    "locked_stock": inv.locked_quantity or 0,
                    "safety_stock": inv.safety_stock or 0,
                    "selling_price": float(sku.price or 0),
                    "cost_price": float(sku.cost_price or 0),
                }
            )
        return contexts

    @staticmethod
    async def build_profit_watch_contexts(db, max_skus: int = 50) -> list[dict]:
        """A6: 查询所有 SKU 的成本和售价，计算利润率"""
        stmt = select(Sku).limit(max_skus)
        result = await db.execute(stmt)
        skus = result.scalars().all()
        contexts = []
        for sku in skus:
            price = float(sku.price or 0)
            cost = float(sku.cost_price or 0)
            margin = ((price - cost) / price * 100) if price > 0 else 0
            contexts.append(
                {
                    "sku_code": sku.code or sku.spec_desc or f"SKU#{sku.id}",
                    "selling_price": price,
                    "cost_price": cost,
                    "margin_pct": round(margin, 2),
                }
            )
        return contexts

    @staticmethod
    async def build_discount_risk_contexts(db, max_skus: int = 50) -> list[dict]:
        """G3: 查询最近有价格变动的 SKU"""
        datetime.now(timezone.utc) - timedelta(days=7)
        stmt = select(Sku).limit(max_skus)
        result = await db.execute(stmt)
        skus = result.scalars().all()
        contexts = []
        for sku in skus:
            contexts.append(
                {
                    "sku_code": sku.code or sku.spec_desc or f"SKU#{sku.id}",
                    "selling_price": float(sku.price or 0),
                    "cost_price": float(sku.cost_price or 0),
                    "stock": 0,  # 会在 data_service 中补齐
                }
            )
        return contexts

    @staticmethod
    async def build_generic_context(db) -> dict:
        """通用上下文：用于只需环境快照的 Agent"""
        total_products = await db.scalar(select(func.count(Product.id))) or 0
        total_skus = await db.scalar(select(func.count(Sku.id))) or 0
        return {
            "product_count": total_products,
            "sku_count": total_skus,
            "timestamp": datetime.now(timezone.utc).isoformat(),
        }

    @staticmethod
    async def build_customer_service_context(db, max_items: int = 10) -> list[dict]:
        """A4: 查询近期待处理/异常订单"""
        recent_order_stmt = (
            select(Order)
            .where(Order.status.in_(["pending", "paid"]))
            .order_by(Order.created_at.desc())
            .limit(max_items)
        )
        result = await db.execute(recent_order_stmt)
        orders = result.scalars().all()
        contexts = []
        for o in orders:
            contexts.append(
                {
                    "order_id": o.id,
                    "order_no": getattr(o, "order_no", f"ORDER#{o.id}"),
                    "status": o.status,
                    "total_amount": float(getattr(o, "total_amount", 0)),
                }
            )
        return contexts


class AgentScheduler:
    """Agent 定时调度器

    用法:
        scheduler = AgentScheduler()
        await scheduler.start()   # 在 lifespan 启动时调用
        await scheduler.stop()    # 在 lifespan 结束时调用
    """

    def __init__(self):
        self._tasks: dict[str, asyncio.Task] = {}
        self._schedules: dict[str, dict] = {}
        self._running = False
        self._system_user_id = 1  # 系统调度使用管理员用户
        self._entropy_enabled = True

        # 复制默认配置，运行时可变
        for agent_id, cfg in DEFAULT_SCHEDULES.items():
            self._schedules[agent_id] = dict(cfg)

    # ── 生命周期 ──────────────────────────────────────────────

    async def start(self):
        """启动所有已启用的 Agent 调度任务"""
        if self._running:
            logger.warning("AgentScheduler 已在运行")
            return
        self._running = True
        started = 0
        for agent_id, cfg in self._schedules.items():
            if cfg.get("enabled", False):
                task = asyncio.create_task(
                    self._run_agent_loop(agent_id),
                    name=f"agent-scheduler-{agent_id}",
                )
                self._tasks[agent_id] = task
                started += 1

        # 启动熵系统防御循环
        if self._entropy_enabled:
            entropy_task = asyncio.create_task(
                self._run_entropy_loop(),
                name="agent-scheduler-entropy",
            )
            self._tasks["__entropy__"] = entropy_task

        logger.info(
            "AgentScheduler 已启动: %d/%d Agent + 熵防御",
            started,
            len(self._schedules),
        )

    async def stop(self):
        """停止所有调度任务"""
        self._running = False
        for agent_id, task in self._tasks.items():
            task.cancel()
        if self._tasks:
            await asyncio.gather(*self._tasks.values(), return_exceptions=True)
        self._tasks.clear()
        logger.info("AgentScheduler 已停止")

    # ── 配置管理 ──────────────────────────────────────────────

    def get_schedules(self) -> dict[str, dict]:
        """获取所有调度配置（只读快照）"""
        return {k: dict(v) for k, v in self._schedules.items()}

    def get_schedule(self, agent_id: str) -> Optional[dict]:
        """获取单个 Agent 调度配置"""
        cfg = self._schedules.get(agent_id)
        return dict(cfg) if cfg else None

    def update_schedule(self, agent_id: str, config: dict) -> bool:
        """更新调度配置，热生效"""
        if agent_id not in self._schedules:
            return False
        old = self._schedules[agent_id]
        old.update(config)

        # 如果运行中且启停状态变化，重新创建任务
        if self._running:
            if old.get("enabled") and agent_id not in self._tasks:
                task = asyncio.create_task(
                    self._run_agent_loop(agent_id),
                    name=f"agent-scheduler-{agent_id}",
                )
                self._tasks[agent_id] = task
            elif not old.get("enabled") and agent_id in self._tasks:
                self._tasks[agent_id].cancel()
                del self._tasks[agent_id]

        return True

    async def trigger_now(self, agent_id: str, db) -> dict:
        """手动触发一次 Agent 调度"""
        if agent_id not in self._schedules:
            return {"error": f"Agent {agent_id} 未配置调度"}
        try:
            result = await self._run_cycle(agent_id, db)
            return result
        except Exception as e:
            logger.exception("手动触发 %s 失败", agent_id)
            return {"error": str(e)}

    # ── 内部实现 ──────────────────────────────────────────────

    async def _run_agent_loop(self, agent_id: str):
        """单个 Agent 的调度循环"""
        cfg = self._schedules.get(agent_id)
        if not cfg:
            return

        interval = cfg.get("interval", 3600)
        decision_points = cfg.get("decision_points", [])

        logger.info(
            "调度 [%s] 启动: 间隔=%ds 决策点=%s",
            agent_id,
            interval,
            decision_points,
        )

        while self._running:
            try:
                async with async_session_factory() as db:
                    await self._run_cycle(agent_id, db, decision_points)
            except asyncio.CancelledError:
                break
            except Exception as e:
                logger.error("调度 [%s] 循环失败: %s", agent_id, e)

            # 等待下一个周期（可取消等待）
            try:
                await asyncio.sleep(interval)
            except asyncio.CancelledError:
                break

        logger.info("调度 [%s] 已停止", agent_id)

    async def _run_entropy_loop(self):
        """熵系统防御循环 — 每 6 小时自动清理规则"""
        logger.info("熵防御循环启动: 间隔=%ds", ENTROPY_INTERVAL)

        while self._running:
            try:
                async with async_session_factory() as db:
                    svc = EntropyService()
                    result = await svc.run_defenses(db, self._system_user_id)
                    dashboard = await svc.get_dashboard(db, self._system_user_id)
                    logger.info(
                        "熵防御完成: 影响 %d 条规则, 熵指数=%.4f",
                        result["total_affected"],
                        dashboard.get("system_entropy_index", 0),
                    )
            except asyncio.CancelledError:
                break
            except Exception as e:
                logger.error("熵防御循环失败: %s", e)

            try:
                await asyncio.sleep(ENTROPY_INTERVAL)
            except asyncio.CancelledError:
                break

        logger.info("熵防御循环已停止")

    async def _run_cycle(
        self,
        agent_id: str,
        db,
        decision_points: Optional[list[str]] = None,
    ) -> dict:
        """执行一次 Agent 调度周期"""
        agent_cls = AgentRegistry.get_agent_class(agent_id)
        if not agent_cls:
            return {"agent_id": agent_id, "error": "Agent 类未注册"}

        cfg = self._schedules.get(agent_id, {})
        if decision_points is None:
            decision_points = cfg.get("decision_points", [])

        results = []
        for dp in decision_points:
            contexts = await self._build_contexts(agent_id, dp, db)
            if isinstance(contexts, dict):
                contexts = [contexts]  # 单个上下文集包装为列表

            for ctx in contexts:
                try:
                    # 从 DB 加载该用户在该决策点的阶段配置
                    stage_override = {}
                    config = await EvolutionService.get_or_create_config(
                        db,
                        self._system_user_id,
                        agent_id,
                        dp,
                    )
                    stage_override[dp] = config.current_stage

                    agent = agent_cls(
                        user_id=self._system_user_id,
                        stage_override=stage_override,
                    )
                    result = await AgentService.execute_decision(
                        db,
                        agent,
                        dp,
                        ctx,
                        dry_run=False,
                    )

                    # 自动触发协作链
                    if result.get("decision_id"):
                        chain_results = await evaluate_chains(
                            agent_id,
                            dp,
                            result,
                            self._system_user_id,
                            db,
                        )
                        if chain_results:
                            result["chain_triggered"] = len(chain_results)

                    results.append(result)
                except Exception as e:
                    logger.error(
                        "Agent [%s] 决策点 [%s] 执行失败: %s",
                        agent_id,
                        dp,
                        e,
                    )

        summary = {
            "agent_id": agent_id,
            "timestamp": datetime.now(timezone.utc).isoformat(),
            "decision_points_executed": len(decision_points),
            "decisions_made": len(results),
        }
        logger.info(
            "调度 [%s] 周期完成: %d 个决策点, %d 条决策",
            agent_id,
            len(decision_points),
            len(results),
        )
        return summary

    async def _build_contexts(
        self, agent_id: str, decision_point: str, db
    ) -> Union[list[dict], dict]:
        """为 Agent 构建决策上下文"""
        builder = SchedulerContextBuilder()

        context_map = {
            "A5": {"stock_alert": builder.build_stock_alert_contexts},
            "A6": {"profit_watch": builder.build_profit_watch_contexts},
            "G3": {"discount_risk_check": builder.build_discount_risk_contexts},
            "A4": {"customer_service": builder.build_customer_service_context},
        }

        agent_map = context_map.get(agent_id, {})
        builder_fn = agent_map.get(decision_point)

        if builder_fn:
            return await builder_fn(db)
        return await builder.build_generic_context(db)


# ── 全局单例 ──────────────────────────────────────────────────

scheduler = AgentScheduler()

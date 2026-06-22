"""Agent 定时调度引擎 — 生产者/消费者 + Worker Pool 模式

让 Agent 按预设策略周期自动决策，无需人工调用 API。
配合 AgentActionService，决策产生的可执行操作进入待办列表等待确认。

架构:
  Scheduler Cycle Trigger
      │
      ▼
  Producer: 枚举所有活跃店铺 × Agent × 决策点 → 生成工作任务 → 投递到 Queue
      │
      ▼
  Queue (asyncio.Queue, maxsize=QUEUE_CAPACITY)
      │
      ▼
  Consumer Pool (N 个 worker, 默认 MAX_WORKERS=20)
      ├─ Worker 1: 拉取任务 → LLM 调用 → 结果写入 DB
      ├─ Worker 2: 拉取任务 → LLM 调用 → 结果写入 DB
      ├─ ...
      └─ Worker N: ...

容错策略:
  - 单任务超时 -> 跳过, worker 继续
  - 单店铺连续 3 次失败 -> 跳过本轮, 写入 ScheduleFailure
  - Worker 崩溃 -> 自动退出并由外层保护逻辑捕获
  - 队列满 -> 生产者阻塞 (backpressure)
  - 周期重叠 -> 跳过下一个触发

兼容性:
  - start() / stop() / trigger_now(agent_id, db) 签名不变
  - 全局 scheduler 单例照常工作
  - trigger_now 旁路队列, 直连执行
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
from app.agent.models import AgentDecision, ScheduleFailure
from app.agent.evolution_service import EvolutionService
from app.agent.entropy.service import EntropyService
from app.models import Store, Sku, Inventory, Product, Order

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
    """为各 Agent 构建决策上下文, 支持可选的 store_id 过滤"""

    @staticmethod
    async def build_stock_alert_contexts(
        db, max_skus: int = 50, store_id: Optional[int] = None,
    ) -> list[dict]:
        """A5: 查询库存低于预警线的 SKU, 每个 SKU 生成一个上下文"""
        # ponytail: store_id accepted but not filtered — needs StoreSKU association table
        if store_id is not None:
            logger.debug("build_stock_alert_contexts store_id=%s (filter not yet implemented)", store_id)
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
            contexts.append({
                "sku_code": sku.code or sku.spec_desc or f"SKU#{sku.id}",
                "sellable_stock": inv.quantity or 0,
                "locked_stock": inv.locked_quantity or 0,
                "safety_stock": inv.safety_stock or 0,
                "selling_price": float(sku.price or 0),
                "cost_price": float(sku.cost_price or 0),
            })
        return contexts

    @staticmethod
    async def build_profit_watch_contexts(
        db, max_skus: int = 50, store_id: Optional[int] = None,
    ) -> list[dict]:
        """A6: 查询所有 SKU 的成本和售价, 计算利润率"""
        if store_id is not None:
            logger.debug("build_profit_watch_contexts store_id=%s (filter not yet implemented)", store_id)
        stmt = select(Sku).limit(max_skus)
        result = await db.execute(stmt)
        skus = result.scalars().all()
        contexts = []
        for sku in skus:
            price = float(sku.price or 0)
            cost = float(sku.cost_price or 0)
            margin = ((price - cost) / price * 100) if price > 0 else 0
            contexts.append({
                "sku_code": sku.code or sku.spec_desc or f"SKU#{sku.id}",
                "selling_price": price,
                "cost_price": cost,
                "margin_pct": round(margin, 2),
            })
        return contexts

    @staticmethod
    async def build_discount_risk_contexts(
        db, max_skus: int = 50, store_id: Optional[int] = None,
    ) -> list[dict]:
        """G3: 查询最近有价格变动的 SKU"""
        if store_id is not None:
            logger.debug("build_discount_risk_contexts store_id=%s (filter not yet implemented)", store_id)
        seven_days_ago = datetime.now(timezone.utc) - timedelta(days=7)
        stmt = select(Sku).limit(max_skus)
        result = await db.execute(stmt)
        skus = result.scalars().all()
        contexts = []
        for sku in skus:
            contexts.append({
                "sku_code": sku.code or sku.spec_desc or f"SKU#{sku.id}",
                "selling_price": float(sku.price or 0),
                "cost_price": float(sku.cost_price or 0),
                "stock": 0,  # 会在 data_service 中补齐
            })
        return contexts

    @staticmethod
    async def build_generic_context(
        db, store_id: Optional[int] = None,
    ) -> dict:
        """通用上下文: 用于只需环境快照的 Agent"""
        if store_id is not None:
            logger.debug("build_generic_context store_id=%s (filter not yet implemented)", store_id)
        total_products = await db.scalar(select(func.count(Product.id))) or 0
        total_skus = await db.scalar(select(func.count(Sku.id))) or 0
        return {
            "product_count": total_products,
            "sku_count": total_skus,
            "timestamp": datetime.now(timezone.utc).isoformat(),
        }

    @staticmethod
    async def build_customer_service_context(
        db, max_items: int = 10, store_id: Optional[int] = None,
    ) -> list[dict]:
        """A4: 查询近期待处理/异常订单"""
        if store_id is not None:
            logger.debug("build_customer_service_context store_id=%s (filter not yet implemented)", store_id)
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
            contexts.append({
                "order_id": o.id,
                "order_no": getattr(o, "order_no", f"ORDER#{o.id}"),
                "status": o.status,
                "total_amount": float(getattr(o, "total_amount", 0)),
            })
        return contexts


class AgentScheduler:
    """Agent 定时调度器 — 生产者/消费者 + Worker Pool 模式

    用法:
        scheduler = AgentScheduler()
        await scheduler.start()   # 在 lifespan 启动时调用
        await scheduler.stop()    # 在 lifespan 结束时调用

    关键参数 (可配置, 热生效):
        MAX_WORKERS = 20    最大并发 LLM 调用数
        QUEUE_CAPACITY = 500  队列缓冲上限
        BATCH_SIZE = 50       店铺分批预取
        WORKER_TIMEOUT = 30   单任务超时 (秒)
        IDLE_TIMEOUT = 10     Worker 空闲等待超时 (秒)
    """

    # ── 可配置参数 ──────────────────────────────────────────
    MAX_WORKERS = 20
    QUEUE_CAPACITY = 500
    BATCH_SIZE = 50
    WORKER_TIMEOUT = 30
    IDLE_TIMEOUT = 10

    def __init__(self):
        self._tasks: dict[str, asyncio.Task] = {}
        self._schedules: dict[str, dict] = {}
        self._running = False
        self._system_user_id = 1  # 系统调度使用管理员用户
        self._entropy_enabled = True
        self._archive_enabled = True

        # 生产者/消费者 状态
        self._queue: Optional[asyncio.Queue] = None
        self._cycle_running = False
        self._store_failures: dict[int, int] = {}  # store_id -> 连续失败次数
        self._results: list[dict] = []

        # 复制默认配置, 运行时可变
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

        # 启动归档循环（每日一次）
        if self._archive_enabled:
            archive_task = asyncio.create_task(
                self._run_archive_loop(),
                name="agent-scheduler-archive",
            )
            self._tasks["__archive__"] = archive_task

        logger.info(
            "AgentScheduler 已启动: %d/%d Agent + 熵防御 + 归档",
            started, len(self._schedules),
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
        """获取所有调度配置 (只读快照)"""
        return {k: dict(v) for k, v in self._schedules.items()}

    def get_schedule(self, agent_id: str) -> Optional[dict]:
        """获取单个 Agent 调度配置"""
        cfg = self._schedules.get(agent_id)
        return dict(cfg) if cfg else None

    def update_schedule(self, agent_id: str, config: dict) -> bool:
        """更新调度配置, 热生效"""
        if agent_id not in self._schedules:
            return False
        old = self._schedules[agent_id]
        old.update(config)

        # 如果运行中且启停状态变化, 重新创建任务
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
        """手动触发一次 Agent 调度 (旁路队列, 直连执行)"""
        if agent_id not in self._schedules:
            return {"error": f"Agent {agent_id} 未配置调度"}
        try:
            result = await self._run_single_agent_sync(agent_id, db)
            return result
        except Exception as e:
            logger.exception("手动触发 %s 失败", agent_id)
            return {"error": str(e)}

    # ── 内部循环 ──────────────────────────────────────────────

    async def _run_agent_loop(self, agent_id: str):
        """单个 Agent 的调度循环 -> 触发完整生产者/消费者周期

        每个 Agent 的独立循环都会尝试启动 _run_cycle(), 但 _cycle_running
        标志确保只有一个周期在运行。所有周期覆盖所有 Agent 的调度。
        """
        cfg = self._schedules.get(agent_id)
        if not cfg:
            return

        interval = cfg.get("interval", 3600)
        decision_points = cfg.get("decision_points", [])

        logger.info(
            "调度 [%s] 启动: 间隔=%ds 决策点=%s",
            agent_id, interval, decision_points,
        )

        while self._running:
            try:
                async with async_session_factory() as db:
                    await self._run_cycle(db)
            except asyncio.CancelledError:
                break
            except Exception as e:
                logger.error("调度 [%s] 循环失败: %s", agent_id, e)

            # 等待下一个周期 (可取消等待)
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

    async def run_archive_cycle(self, db):
        """Run once per day — archive old decisions"""
        from app.agent.archive_service import archive_old_decisions
        result = await archive_old_decisions(db)
        logger.info(f"Archive cycle: {result['archived_count']} decisions archived")
        return result

    async def _run_archive_loop(self):
        """归档循环 — 每 24 小时归档一次旧决策"""
        logger.info("归档循环启动: 间隔=86400s")

        while self._running:
            try:
                async with async_session_factory() as db:
                    await self.run_archive_cycle(db)
            except asyncio.CancelledError:
                break
            except Exception as e:
                logger.error(f"归档循环失败: {e}")

            try:
                await asyncio.sleep(86400)  # 24 hours
            except asyncio.CancelledError:
                break

        logger.info("归档循环已停止")

    # ── 生产者/消费者 周期 ─────────────────────────────────────

    async def _run_cycle(self, db) -> dict:
        """执行一轮完整的生产者/消费者调度周期

        枚举所有活跃店铺 × 启用 Agent × 决策点,
        通过 Producer -> Queue -> Consumer Pool (N workers) 管道处理。

        周期重叠保护: _cycle_running 标志阻止同时运行多个周期。
        """
        if self._cycle_running:
            logger.warning("上一个周期尚未完成, 跳过本轮")
            return {
                "error": "cycle_overlap",
                "message": "上一个周期尚未完成, 跳过本轮",
            }

        self._cycle_running = True
        self._store_failures = {}
        self._results = []
        self._queue = asyncio.Queue(self.QUEUE_CAPACITY)

        try:
            # 阶段 1: 主生产者/消费者管道
            producer = asyncio.create_task(
                self._produce(db),
                name="cycle-producer",
            )
            workers = [
                asyncio.create_task(
                    self._consume(i),
                    name=f"cycle-worker-{i}",
                )
                for i in range(self.MAX_WORKERS)
            ]

            await producer  # 所有任务投递完成
            await self._queue.join()  # 所有任务执行完毕

            # 停止 workers
            for w in workers:
                w.cancel()
            await asyncio.gather(*workers, return_exceptions=True)

            # 阶段 2: 重试本轮失败的任务 (未达 3 次且未解决)
            await self._enqueue_cycle_retries()
            if self._queue.qsize() > 0:
                retry_workers = [
                    asyncio.create_task(
                        self._consume(i),
                        name=f"cycle-retry-worker-{i}",
                    )
                    for i in range(min(self._queue.qsize(), self.MAX_WORKERS))
                ]
                await self._queue.join()
                for w in retry_workers:
                    w.cancel()
                await asyncio.gather(*retry_workers, return_exceptions=True)

        except asyncio.CancelledError:
            logger.warning("调度周期被取消")
            return {"error": "cancelled"}
        except Exception as e:
            logger.exception("调度周期执行失败: %s", e)
            return {"error": str(e)}
        finally:
            self._cycle_running = False

        return self._build_summary()

    async def _produce(self, db):
        """生产者: 枚举活跃店铺 × Agent × 决策点, 生成工作任务

        使用游标分批获取店铺, 避免一次加载过多。
        每个工作项是一个 dict, 包含:
          - agent_id: Agent 标识
          - dp: 决策点
          - context: 决策上下文
          - store_id: 店铺 ID
          - user_id: 用户 ID
        """
        cursor = None
        while True:
            stores = await self._get_active_stores_batch(db, cursor)
            if not stores:
                break

            for store in stores:
                # 跳过已连续失败 3 次的店铺
                if self._store_failures.get(store.id, 0) >= 3:
                    logger.warning(
                        "Store %d 已连续失败 3 次, 跳过本轮", store.id,
                    )
                    continue

                store_user_id = store.user_id or self._system_user_id

                for agent_id, cfg in self._schedules.items():
                    if not cfg.get("enabled"):
                        continue
                    for dp in cfg.get("decision_points", []):
                        contexts = await self._build_contexts(
                            agent_id, dp, db, store_id=store.id,
                        )
                        if isinstance(contexts, dict):
                            contexts = [contexts]

                        for ctx in contexts:
                            await self._queue.put({
                                "agent_id": agent_id,
                                "dp": dp,
                                "context": ctx,
                                "store_id": store.id,
                                "user_id": store_user_id,
                            })

            cursor = stores[-1].id

    async def _consume(self, worker_id: int):
        """消费者 Worker: 从队列拉取任务, 执行决策

        使用 IDLE_TIMEOUT 空闲等待, 超时后继续轮询。
        每个任务使用 WORKER_TIMEOUT 超时控制。
        Worker 崩溃自动退出并由外层保护逻辑捕获。
        """
        while self._running:
            try:
                task = await asyncio.wait_for(
                    self._queue.get(), timeout=self.IDLE_TIMEOUT,
                )
            except asyncio.TimeoutError:
                # 队列为空, 继续等待
                continue

            try:
                result = await asyncio.wait_for(
                    self._execute_single_decision(task),
                    timeout=self.WORKER_TIMEOUT,
                )
                if result:
                    self._results.append(result)
                # 成功后清除该店铺的连续失败计数
                self._store_failures.pop(task["store_id"], None)

            except asyncio.TimeoutError:
                logger.error(
                    "Worker[%d] 任务超时: agent=%s store=%d dp=%s",
                    worker_id, task["agent_id"], task["store_id"], task["dp"],
                )
                await self._record_task_failure(task, "timeout")

            except Exception as e:
                logger.error(
                    "Worker[%d] 任务失败: agent=%s store=%d dp=%s err=%s",
                    worker_id, task["agent_id"], task["store_id"],
                    task["dp"], e,
                )
                await self._record_task_failure(task, str(e))

            finally:
                self._queue.task_done()

    async def _execute_single_decision(self, task: dict) -> Optional[dict]:
        """执行单个 Agent 决策 (在独立 DB session 中)

        与旧的 _run_cycle 内部逻辑等效, 但使用独立的 DB session
        和 commit 保证决策持久化。
        """
        agent_id = task["agent_id"]
        dp = task["dp"]
        context = task["context"]
        store_id = task["store_id"]
        user_id = task.get("user_id", self._system_user_id)

        agent_cls = AgentRegistry.get_agent_class(agent_id)
        if not agent_cls:
            logger.warning("Agent %s 未注册, 跳过任务", agent_id)
            return None

        async with async_session_factory() as db:
            try:
                # 从 DB 加载该用户在该决策点的阶段配置
                stage_override = {}
                config = await EvolutionService.get_or_create_config(
                    db, user_id, agent_id, dp,
                )
                stage_override[dp] = config.current_stage

                agent = agent_cls(
                    user_id=user_id,
                    stage_override=stage_override,
                )
                result = await AgentService.execute_decision(
                    db, agent, dp, context, dry_run=False,
                )

                # 自动触发协作链
                if result.get("decision_id"):
                    chain_results = await evaluate_chains(
                        agent_id, dp, result, user_id, db,
                    )
                    if chain_results:
                        result["chain_triggered"] = len(chain_results)

                # 显式提交, 保证决策持久化
                await db.commit()
                return result

            except Exception:
                await db.rollback()
                raise

    # ── 失败处理 ──────────────────────────────────────────────

    async def _record_task_failure(self, task: dict, error_msg: str):
        """记录单次任务失败到 ScheduleFailure 表

        首次失败: 创建新记录, retry_count=1
        再次失败: 递增 retry_count, 更新 last_retry_at
        达到 3 次: 标记 resolved=True, 不再重试
        """
        store_id = task["store_id"]
        current_failures = self._store_failures.get(store_id, 0) + 1
        self._store_failures[store_id] = current_failures

        try:
            async with async_session_factory() as db:
                # 查找是否已有未解决的失败记录
                stmt = select(ScheduleFailure).where(
                    ScheduleFailure.agent_id == task["agent_id"],
                    ScheduleFailure.store_id == store_id,
                    ScheduleFailure.decision_point == task["dp"],
                    ScheduleFailure.resolved == False,
                ).order_by(ScheduleFailure.failed_at.desc()).limit(1)
                result = await db.execute(stmt)
                existing = result.scalar_one_or_none()

                if existing:
                    existing.retry_count = (existing.retry_count or 0) + 1
                    existing.last_retry_at = func.now()
                    existing.error_msg = error_msg
                    if existing.retry_count >= 3:
                        existing.resolved = True
                else:
                    failure = ScheduleFailure(
                        agent_id=task["agent_id"],
                        store_id=store_id,
                        decision_point=task["dp"],
                        error_msg=error_msg,
                        retry_count=1,
                        last_retry_at=func.now(),
                        resolved=False,
                    )
                    db.add(failure)

                await db.commit()
        except Exception as e:
            logger.error("记录失败信息时出错: %s", e)

    async def _enqueue_cycle_retries(self):
        """周期结束后检查 ScheduleFailure 表, 未达 3 次的任务重新入队重试

        仅重试本轮产生的失败 (retry_count < 3 且 resolved=False)。
        重试的上下文为空 dict, 由 execute_decision 自行补齐。
        """
        try:
            async with async_session_factory() as db:
                stmt = select(ScheduleFailure).where(
                    ScheduleFailure.resolved == False,
                    ScheduleFailure.retry_count < 3,
                )
                result = await db.execute(stmt)
                failures = result.scalars().all()

                retry_count = 0
                for f in failures:
                    await self._queue.put({
                        "agent_id": f.agent_id,
                        "dp": f.decision_point,
                        "context": {},  # 由 execute_decision 自行补齐
                        "store_id": f.store_id,
                        "user_id": self._system_user_id,
                    })
                    retry_count += 1

                if retry_count > 0:
                    logger.info("本轮重试入队 %d 个失败任务", retry_count)

        except Exception as e:
            logger.error("检查失败重试表时出错: %s", e)

    # ── 店铺查询 ──────────────────────────────────────────────

    async def _get_active_stores_batch(self, db, cursor: Optional[int] = None) -> list:
        """分批获取活跃店铺 (游标分页)"""
        stmt = (
            select(Store)
            .where(Store.status == 1)
            .order_by(Store.id)
            .limit(self.BATCH_SIZE)
        )
        if cursor is not None:
            stmt = stmt.where(Store.id > cursor)
        result = await db.execute(stmt)
        return list(result.scalars().all())

    async def _get_active_stores(self, db, user_id: int = 1) -> list:
        """获取用户的所有活跃店铺 (全量, 用于外部调用)"""
        stmt = (
            select(Store)
            .where(Store.status == 1, Store.user_id == user_id)
            .order_by(Store.id)
        )
        result = await db.execute(stmt)
        return list(result.scalars().all())

    # ── 上下文构建 ──────────────────────────────────────────────

    async def _build_contexts(
        self,
        agent_id: str,
        decision_point: str,
        db,
        store_id: Optional[int] = None,
    ) -> Union[list[dict], dict]:
        """为 Agent 构建决策上下文, 传入 store_id"""
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
            return await builder_fn(db, store_id=store_id)
        return await builder.build_generic_context(db, store_id=store_id)

    # ── 单 Agent 同步执行 (用于 trigger_now, 旁路队列) ──────────

    async def _run_single_agent_sync(self, agent_id: str, db) -> dict:
        """单 Agent 同步执行 (旁路队列, 直连执行)

        保留原有的 per-agent 执行逻辑, 用于 trigger_now 手动触发。
        不经过生产者/消费者队列, 直接在当前上下文中执行决策。
        """
        agent_cls = AgentRegistry.get_agent_class(agent_id)
        if not agent_cls:
            return {"agent_id": agent_id, "error": "Agent 类未注册"}

        cfg = self._schedules.get(agent_id, {})
        decision_points = cfg.get("decision_points", [])

        results = []
        for dp in decision_points:
            contexts = await self._build_contexts(agent_id, dp, db)
            if isinstance(contexts, dict):
                contexts = [contexts]

            for ctx in contexts:
                try:
                    # 从 DB 加载该用户在该决策点的阶段配置
                    stage_override = {}
                    config = await EvolutionService.get_or_create_config(
                        db, self._system_user_id, agent_id, dp,
                    )
                    stage_override[dp] = config.current_stage

                    agent = agent_cls(
                        user_id=self._system_user_id,
                        stage_override=stage_override,
                    )
                    result = await AgentService.execute_decision(
                        db, agent, dp, ctx, dry_run=False,
                    )

                    # 自动触发协作链
                    if result.get("decision_id"):
                        chain_results = await evaluate_chains(
                            agent_id, dp, result, self._system_user_id, db,
                        )
                        if chain_results:
                            result["chain_triggered"] = len(chain_results)

                    results.append(result)
                except Exception as e:
                    logger.error(
                        "Agent [%s] 决策点 [%s] 执行失败: %s",
                        agent_id, dp, e,
                    )

        summary = {
            "agent_id": agent_id,
            "timestamp": datetime.now(timezone.utc).isoformat(),
            "decision_points_executed": len(decision_points),
            "decisions_made": len(results),
        }
        logger.info(
            "单 Agent [%s] 执行完成: %d 个决策点, %d 条决策",
            agent_id, len(decision_points), len(results),
        )
        return summary

    # ── 辅助方法 ──────────────────────────────────────────────

    def _build_summary(self) -> dict:
        """构建周期执行摘要"""
        return {
            "timestamp": datetime.now(timezone.utc).isoformat(),
            "total_agents": len([
                a for a, c in self._schedules.items() if c.get("enabled")
            ]),
            "total_decisions": len(self._results),
            "store_failures": {
                str(sid): count
                for sid, count in self._store_failures.items()
            },
        }


# ── 全局单例 ──────────────────────────────────────────────────

scheduler = AgentScheduler()

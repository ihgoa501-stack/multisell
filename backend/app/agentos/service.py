"""AgentOS 服务层 — 聚合现有业务数据并归一化为 WorkItem"""

from __future__ import annotations

from datetime import datetime, timedelta, timezone
from typing import Any, Optional

from sqlalchemy import func as sa_func, select, text
from sqlalchemy.ext.asyncio import AsyncSession

from app.agentos.schemas import (
    AgentOSAgent,
    AgentOSMetric,
    AgentOSOperationLogVO,
    AgentOSOverview,
    AgentOSSquad,
    AgentOSTemplate,
    AgentOSWorkItem,
    AutonomyLevel,
    RiskLevel,
    WorkItemPriority,
    WorkItemStatus,
)
from app.models import ExceptionItem, Notification
from app.agentos.models import AgentOSOperationLog

# ─── Agent → Squad 映射 ──────────────────────────────────────

AGENT_TO_SQUAD: dict[str, str] = {
    "A1": "growth",
    "A2": "growth",
    "A3": "growth",
    "A4": "growth",
    "A5": "fulfillment",
    "G2": "fulfillment",
    "A6": "risk",
    "A7": "risk",
    "G3": "risk",
    "G1": "risk",
}

SQUAD_TO_NAME: dict[str, str] = {
    "growth": "增长小队",
    "fulfillment": "履约小队",
    "risk": "风控小队",
}

AGENT_META: dict[str, dict[str, str]] = {
    "A1": {"name": "选品助手", "role": "选品分析"},
    "A2": {"name": "Listing 优化师", "role": "刊登优化"},
    "A3": {"name": "广告顾问", "role": "广告投放"},
    "A4": {"name": "客服助手", "role": "客户服务"},
    "A5": {"name": "库存管家", "role": "库存管理"},
    "G2": {"name": "仓储专员", "role": "仓储物流"},
    "A6": {"name": "利润分析师", "role": "利润分析"},
    "A7": {"name": "合规检查员", "role": "合规检查"},
    "G3": {"name": "折扣风控", "role": "折扣风控"},
    "G1": {"name": "总控", "role": "Governance"},
}

MODULE_TO_SQUAD: dict[str, str] = {
    "product": "growth",
    "listing": "growth",
    "listing_task": "growth",
    "inventory": "fulfillment",
    "order": "fulfillment",
    "shipping": "fulfillment",
    "settlement": "risk",
    "finance": "risk",
    "platform_fee": "risk",
    "compliance": "risk",
    "agent": "risk",
}

ALERT_TO_SQUAD: dict[str, str] = {
    "inventory_low_stock": "fulfillment",
    "inventory_out_of_stock": "fulfillment",
    "order_pending": "fulfillment",
    "listing_failed": "growth",
    "settlement_pending": "risk",
    "settlement_discrepancy": "risk",
}

# ─── 静态 Squad 定义 ─────────────────────────────────────────

AGENT_SQUADS: list[dict[str, Any]] = [
    {
        "id": "growth",
        "name": "增长小队",
        "description": "负责选品、Listing、广告建议和上新质量。",
        "domain": "商品增长",
        "agents": ["A1", "A2", "A3", "A4"],
    },
    {
        "id": "fulfillment",
        "name": "履约小队",
        "description": "负责库存、订单、仓储、海关和物流闭环。",
        "domain": "订单履约",
        "agents": ["A5", "G2"],
    },
    {
        "id": "risk",
        "name": "风控小队",
        "description": "负责利润、折扣、合规和平台安全红线。",
        "domain": "风险控制",
        "agents": ["A6", "A7", "G3", "G1"],
    },
]

TEMPLATE_CARDS: list[dict[str, Any]] = [
    {
        "id": "pre_listing_decision",
        "title": "上架前经营决策",
        "squad": "risk",
        "description": "串联商品、物流、平台费和利润，判断是否值得上架。",
        "mode": "Agent",
        "route": "/decisions/prelisting",
    },
    {
        "id": "listing_optimization",
        "title": "Listing 优化",
        "squad": "growth",
        "description": "生成标题、描述、关键词和平台适配建议。",
        "mode": "Agent",
        "route": "/listings/ai-workbench",
    },
    {
        "id": "inventory_replenishment",
        "title": "库存补货",
        "squad": "fulfillment",
        "description": "识别低库存和断货风险，生成补货建议。",
        "mode": "Agent",
        "route": "/inventory/alerts",
    },
    {
        "id": "profit_risk",
        "title": "利润风控",
        "squad": "risk",
        "description": "识别毛利率下降、费用异常和折扣风险。",
        "mode": "Ask",
        "route": "/finance",
    },
    {
        "id": "customer_service_draft",
        "title": "客服草稿",
        "squad": "growth",
        "description": "生成多语言客服回复草稿，不自动回复真实客户。",
        "mode": "Ask",
        "route": "/agents/A4",
    },
]


# ─── 归一化纯函数 ────────────────────────────────────────────


class AgentOSService:
    """AgentOS 聚合服务 — 归一化业务数据为统一模型"""

    # ── 辅助函数 ──────────────────────────────────────────

    @staticmethod
    def _iso(dt: Any) -> Any:
        """保持 datetime 对象不变（序列化由 Pydantic 处理）"""
        return dt

    @staticmethod
    def _squad_for_agent(agent_id: str | None) -> str:
        return AGENT_TO_SQUAD.get(agent_id or "", "governance")

    @staticmethod
    def _risk_from_severity(severity: str | None) -> RiskLevel:
        mapping = {
            "critical": RiskLevel.CRITICAL,
            "error": RiskLevel.HIGH,
            "high": RiskLevel.HIGH,
            "warning": RiskLevel.MEDIUM,
            "info": RiskLevel.LOW,
            "low": RiskLevel.LOW,
        }
        return mapping.get(severity or "", RiskLevel.MEDIUM)

    @staticmethod
    def _risk_from_action(
        action_type: str | None,
        payload: dict[str, Any] | None = None,
    ) -> RiskLevel:
        payload = payload or {}
        try:
            amount = abs(float(payload.get("amount") or payload.get("total_amount") or 0))
        except (TypeError, ValueError):
            amount = 0
        sku_codes = payload.get("sku_codes") or payload.get("sku_ids") or []
        sku_count = len(sku_codes) if isinstance(sku_codes, list) else 0
        if amount >= 500 or sku_count > 20:
            return RiskLevel.CRITICAL
        if action_type in {"replenish", "price_adjust", "discount_review", "ad_action"}:
            return RiskLevel.HIGH
        return RiskLevel.MEDIUM

    @staticmethod
    def _priority_from_risk(risk: RiskLevel) -> WorkItemPriority:
        mapping = {
            RiskLevel.CRITICAL: WorkItemPriority.CRITICAL,
            RiskLevel.HIGH: WorkItemPriority.HIGH,
            RiskLevel.MEDIUM: WorkItemPriority.MEDIUM,
            RiskLevel.LOW: WorkItemPriority.LOW,
        }
        return mapping.get(risk, WorkItemPriority.MEDIUM)

    @staticmethod
    def _approval_required(risk: RiskLevel, status: str) -> bool:
        return status in {"pending", "proposed"} and risk in {RiskLevel.HIGH, RiskLevel.CRITICAL}

    @staticmethod
    def _to_datetime(val: Any) -> datetime | None:
        if val is None:
            return None
        if isinstance(val, datetime):
            return val
        return None

    # ── 归一化函数 ─────────────────────────────────────────

    @staticmethod
    def normalize_agent_pending_action(row: Any) -> AgentOSWorkItem:
        """将 Agent 待执行操作归一化为 WorkItem"""
        risk = AgentOSService._risk_from_action(
            getattr(row, "action_type", None),
            getattr(row, "action_payload", None) or getattr(row, "proposed_payload", None),
        )
        squad_id = AgentOSService._squad_for_agent(getattr(row, "agent_id", None))
        status_raw = getattr(row, "status", "pending")
        status = AgentOSService._map_action_status(status_raw)

        return AgentOSWorkItem(
            id=f"agent_action:{getattr(row, 'id', 0)}",
            source_type="agent_action",
            source_id=str(getattr(row, "id", "")),
            title=getattr(row, "summary", "") or getattr(row, "title", ""),
            description=getattr(row, "summary", ""),
            priority=AgentOSService._priority_from_risk(risk),
            status=status,
            risk_level=risk,
            agent_id=getattr(row, "agent_id", None),
            agent_name=AGENT_META.get(getattr(row, "agent_id", ""), {}).get("name"),
            squad_id=squad_id,
            squad_name=SQUAD_TO_NAME.get(squad_id, squad_id),
            autonomy_level=AutonomyLevel.SUGGESTION,
            requires_approval=AgentOSService._approval_required(risk, status_raw),
            created_at=AgentOSService._to_datetime(getattr(row, "created_at", None)),
            updated_at=AgentOSService._to_datetime(getattr(row, "updated_at", None)),
            action_url=f"/agents/{getattr(row, 'agent_id', '')}",
            metadata={
                "action_type": getattr(row, "action_type", ""),
                "decision_id": str(getattr(row, "decision_id", "")),
                "payload": (getattr(row, "action_payload", None) or getattr(row, "proposed_payload", None) or {}),
            },
        )

    @staticmethod
    def normalize_exception(row: Any) -> AgentOSWorkItem:
        """将异常条目归一化为 WorkItem"""
        severity = getattr(row, "severity", "medium")
        risk = AgentOSService._risk_from_severity(severity)
        source_module = getattr(row, "source_module", "")
        squad_id = MODULE_TO_SQUAD.get(source_module, "risk")
        status_raw = getattr(row, "status", "open")

        return AgentOSWorkItem(
            id=f"exception:{getattr(row, 'id', 0)}",
            source_type="exception",
            source_id=str(getattr(row, "id", "")),
            title=getattr(row, "title", ""),
            description=getattr(row, "description", ""),
            priority=AgentOSService._priority_from_risk(risk),
            status=AgentOSService._map_exception_status(status_raw),
            risk_level=risk,
            squad_id=squad_id,
            squad_name=SQUAD_TO_NAME.get(squad_id, squad_id),
            autonomy_level=AutonomyLevel.OBSERVATION,
            requires_approval=False,
            created_at=AgentOSService._to_datetime(getattr(row, "created_at", None)),
            updated_at=AgentOSService._to_datetime(getattr(row, "updated_at", None)),
            action_url=f"/exceptions/{getattr(row, 'id', 0)}",
            metadata={
                "source_module": source_module,
                "severity": severity,
                "recommended_action": getattr(row, "recommended_action", ""),
            },
        )

    @staticmethod
    def normalize_notification(row: Any) -> AgentOSWorkItem:
        """将通知归一化为 WorkItem"""
        alert_type = getattr(row, "alert_type", "")
        squad_id = ALERT_TO_SQUAD.get(alert_type, "governance")
        severity = getattr(row, "severity", "info")
        risk = AgentOSService._risk_from_severity(severity)
        is_read = getattr(row, "is_read", 0)

        # 尝试从 alert_type 推断 agent
        inferred_agent = AgentOSService._infer_agent_from_alert(alert_type)

        return AgentOSWorkItem(
            id=f"notification:{getattr(row, 'id', 0)}",
            source_type="notification",
            source_id=getattr(row, "source_id", "") or str(getattr(row, "id", "")),
            title=getattr(row, "title", ""),
            description=getattr(row, "content", ""),
            priority=AgentOSService._priority_from_risk(risk),
            status=WorkItemStatus.PENDING if not is_read else WorkItemStatus.COMPLETED,
            risk_level=risk,
            agent_id=inferred_agent,
            agent_name=AGENT_META.get(inferred_agent or "", {}).get("name"),
            squad_id=squad_id,
            squad_name=SQUAD_TO_NAME.get(squad_id, squad_id),
            autonomy_level=AutonomyLevel.OBSERVATION,
            requires_approval=False,
            created_at=AgentOSService._to_datetime(getattr(row, "created_at", None)),
            action_url=getattr(row, "link_url", ""),
            metadata={"alert_type": alert_type, "is_read": bool(is_read)},
        )

    @staticmethod
    def normalize_listing_task(row: Any) -> AgentOSWorkItem:
        """将上架任务归一化为 WorkItem"""
        status_raw = getattr(row, "status", "blocked")
        risk = RiskLevel.HIGH if status_raw in {"failed", "blocked"} else RiskLevel.MEDIUM

        return AgentOSWorkItem(
            id=f"listing_task:{getattr(row, 'id', 0)}",
            source_type="listing_task",
            source_id=str(getattr(row, "id", "")),
            title=f"上架任务 #{getattr(row, 'id', '')}",
            description=getattr(row, "last_error", "") or "",
            priority=AgentOSService._priority_from_risk(risk),
            status=AgentOSService._map_listing_status(status_raw),
            risk_level=risk,
            squad_id="growth",
            squad_name="增长小队",
            agent_id="A2",
            agent_name="Listing 优化师",
            autonomy_level=AutonomyLevel.SEMI_AUTONOMOUS,
            requires_approval=True,
            created_at=AgentOSService._to_datetime(getattr(row, "created_at", None)),
            updated_at=AgentOSService._to_datetime(getattr(row, "updated_at", None)),
            action_url=f"/listing_tasks/{getattr(row, 'id', 0)}",
            metadata={
                "product_id": str(getattr(row, "product_id", "")),
                "platform_id": str(getattr(row, "platform_id", "")),
                "status": status_raw,
            },
        )

    # ── 聚合查询 ───────────────────────────────────────────

    @staticmethod
    async def get_control_center(db: AsyncSession) -> dict[str, Any]:
        """聚合总控台数据"""
        overview = AgentOSOverview()

        # 统计异常数量
        critical_count = 0
        exception_items: list[AgentOSWorkItem] = []
        try:
            stmt = select(ExceptionItem).order_by(ExceptionItem.created_at.desc()).limit(20)
            result = await db.execute(stmt)
            for row in result.scalars().all():
                item = AgentOSService.normalize_exception(row)
                exception_items.append(item)
                if item.risk_level == RiskLevel.CRITICAL:
                    critical_count += 1
        except Exception:
            pass  # 优雅降级

        # 统计通知
        notification_items: list[AgentOSWorkItem] = []
        try:
            stmt = select(Notification).order_by(Notification.created_at.desc()).limit(20)
            result = await db.execute(stmt)
            for row in result.scalars().all():
                notification_items.append(AgentOSService.normalize_notification(row))
        except Exception:
            pass

        # 尝试统计 Agent 待执行操作
        pending_actions: list[AgentOSWorkItem] = []
        try:
            from app.agent.models import AgentAction as PendingAgentAction

            stmt = (
                select(PendingAgentAction)
                .where(PendingAgentAction.status == "pending")
                .order_by(PendingAgentAction.created_at.desc())
                .limit(20)
            )
            result = await db.execute(stmt)
            for row in result.scalars().all():
                pending_actions.append(AgentOSService.normalize_agent_pending_action(row))
        except Exception:
            pass

        # 尝试统计上架任务
        listing_items: list[AgentOSWorkItem] = []
        try:
            from app.models import ListingTask

            stmt = (
                select(ListingTask)
                .where(ListingTask.status.in_(["blocked", "failed", "ready"]))
                .order_by(ListingTask.updated_at.desc().nulls_last())
                .limit(20)
            )
            result = await db.execute(stmt)
            for row in result.scalars().all():
                listing_items.append(AgentOSService.normalize_listing_task(row))
        except Exception:
            pass

        # 合并所有 WorkItem 并按风险排序
        all_items = exception_items + notification_items + pending_actions + listing_items
        all_items.sort(key=lambda x: (_PRIORITY_SORT.get(x.priority, 0), x.created_at or datetime.min.replace(tzinfo=timezone.utc)), reverse=True)

        # 统计摘要
        pending_approvals = sum(1 for i in all_items if i.requires_approval)
        overview.pending_approvals = pending_approvals
        overview.critical_items = critical_count
        overview.active_agents = len(AGENT_TO_SQUAD)
        overview.health_score = AgentOSService._compute_health_score(all_items)

        # 构建团队数据
        squads = AgentOSService._build_squads(all_items)

        # 指标
        metrics = AgentOSService._build_metrics(all_items)

        # 优先任务（高风险 + 需审批）
        priority_items = [i for i in all_items if i.risk_level in {RiskLevel.HIGH, RiskLevel.CRITICAL} or i.requires_approval][:10]

        # 最近活动
        recent = sorted(
            all_items,
            key=lambda x: x.updated_at or x.created_at or datetime.min.replace(tzinfo=timezone.utc),
            reverse=True,
        )[:10]

        return {
            "overview": overview,
            "squads": squads,
            "priority_work_items": priority_items,
            "metrics": metrics,
            "recent_activity": recent,
        }

    @staticmethod
    async def get_work_items(
        db: AsyncSession,
        status: str | None = None,
        priority: str | None = None,
        squad: str | None = None,
        agent_id: str | None = None,
        requires_approval: bool | None = None,
        limit: int = 20,
        offset: int = 0,
    ) -> dict[str, Any]:
        """获取 WorkItem 列表"""
        all_items = await AgentOSService._collect_all_work_items(db)

        # 筛选
        if status:
            all_items = [i for i in all_items if i.status.value == status]
        if priority:
            all_items = [i for i in all_items if i.priority.value == priority]
        if squad:
            all_items = [i for i in all_items if i.squad_id == squad]
        if agent_id:
            all_items = [i for i in all_items if i.agent_id == agent_id]
        if requires_approval is not None:
            all_items = [i for i in all_items if i.requires_approval == requires_approval]

        # 排序：风险优先，时间次之
        all_items.sort(
            key=lambda x: (_PRIORITY_SORT.get(x.priority, 0), x.created_at or datetime.min.replace(tzinfo=timezone.utc)),
            reverse=True,
        )

        total = len(all_items)
        paged = all_items[offset : offset + limit]

        return {
            "items": paged,
            "total": total,
            "limit": limit,
            "offset": offset,
        }

    @staticmethod
    async def get_squads(db: AsyncSession) -> dict[str, Any]:
        """获取 Agent 团队列表"""
        all_items = await AgentOSService._collect_all_work_items(db)
        squads = AgentOSService._build_squads(all_items)

        pending_approvals = sum(1 for i in all_items if i.requires_approval)
        critical_items = sum(1 for i in all_items if i.risk_level == RiskLevel.CRITICAL)
        overview = AgentOSOverview(
            health_score=AgentOSService._compute_health_score(all_items),
            active_agents=len(AGENT_TO_SQUAD),
            pending_approvals=pending_approvals,
            critical_items=critical_items,
        )

        return {"squads": squads, "summary": overview}

    @staticmethod
    async def get_templates(db: AsyncSession) -> dict[str, Any]:
        """获取内置模板列表"""
        _ = db  # 当前为静态数据，保留 db 参数保持接口一致
        templates = [AgentOSTemplate(**t) for t in TEMPLATE_CARDS]
        return {"templates": templates}

    # ── 内部方法 ───────────────────────────────────────────

    @staticmethod
    async def _collect_all_work_items(db: AsyncSession) -> list[AgentOSWorkItem]:
        """收集所有来源的 WorkItem"""
        items: list[AgentOSWorkItem] = []

        # 异常
        try:
            stmt = select(ExceptionItem).order_by(ExceptionItem.created_at.desc()).limit(100)
            result = await db.execute(stmt)
            for row in result.scalars().all():
                items.append(AgentOSService.normalize_exception(row))
        except Exception:
            pass

        # 通知
        try:
            stmt = select(Notification).order_by(Notification.created_at.desc()).limit(100)
            result = await db.execute(stmt)
            for row in result.scalars().all():
                items.append(AgentOSService.normalize_notification(row))
        except Exception:
            pass

        # Agent 待执行操作
        try:
            from app.agent.models import AgentAction as PendingAgentAction

            stmt = select(PendingAgentAction).order_by(PendingAgentAction.created_at.desc()).limit(100)
            result = await db.execute(stmt)
            for row in result.scalars().all():
                items.append(AgentOSService.normalize_agent_pending_action(row))
        except Exception:
            pass

        # 上架任务
        try:
            from app.models import ListingTask

            stmt = (
                select(ListingTask)
                .where(ListingTask.status.in_(["blocked", "failed", "ready"]))
                .order_by(ListingTask.updated_at.desc().nulls_last())
                .limit(100)
            )
            result = await db.execute(stmt)
            for row in result.scalars().all():
                items.append(AgentOSService.normalize_listing_task(row))
        except Exception:
            pass

        return items

    @staticmethod
    def _build_squads(items: list[AgentOSWorkItem]) -> list[AgentOSSquad]:
        """从静态定义 + 动态数据构建 Squad 列表"""
        squads: list[AgentOSSquad] = []
        for sq in AGENT_SQUADS:
            squad_id = sq["id"]
            squad_items = [i for i in items if i.squad_id == squad_id]

            agents = [
                AgentOSAgent(
                    id=aid,
                    name=AGENT_META.get(aid, {}).get("name", aid),
                    role=AGENT_META.get(aid, {}).get("role", ""),
                    squad_id=squad_id,
                    status="active",
                    autonomy_level=AutonomyLevel.SUGGESTION,
                    current_workload=sum(1 for i in squad_items if i.agent_id == aid),
                    success_rate=0.85,
                    risk_level=AgentOSService._compute_agent_risk(squad_items, aid),
                )
                for aid in sq["agents"]
            ]

            # 统计
            pending_count = sum(1 for i in squad_items if i.status == WorkItemStatus.PENDING)
            approval_count = sum(1 for i in squad_items if i.requires_approval)
            squad_risk_levels = [i.risk_level for i in squad_items if i.risk_level]
            squad_risk = RiskLevel.CRITICAL if any(r == RiskLevel.CRITICAL for r in squad_risk_levels) else (
                RiskLevel.HIGH if any(r == RiskLevel.HIGH for r in squad_risk_levels) else
                RiskLevel.MEDIUM if any(r == RiskLevel.MEDIUM for r in squad_risk_levels) else
                RiskLevel.LOW
            )

            squads.append(AgentOSSquad(
                id=squad_id,
                name=sq["name"],
                description=sq["description"],
                domain=sq.get("domain", ""),
                status="active",
                autonomy_level=AutonomyLevel.SUGGESTION,
                agents=agents,
                active_work_items=pending_count,
                pending_approvals=approval_count,
                risk_level=squad_risk,
                health_score=AgentOSService._compute_squad_health(squad_items, agents),
            ))
        return squads

    @staticmethod
    def _build_metrics(items: list[AgentOSWorkItem]) -> list[AgentOSMetric]:
        """构建业务指标"""
        total = len(items)
        pending = sum(1 for i in items if i.status == WorkItemStatus.PENDING)
        critical = sum(1 for i in items if i.risk_level == RiskLevel.CRITICAL)
        approval = sum(1 for i in items if i.requires_approval)

        return [
            AgentOSMetric(key="total_items", label="总任务数", value=float(total), unit="个"),
            AgentOSMetric(key="pending_items", label="待处理", value=float(pending), unit="个"),
            AgentOSMetric(key="critical_items", label="严重风险", value=float(critical), unit="个"),
            AgentOSMetric(key="pending_approvals", label="待审批", value=float(approval), unit="个"),
            AgentOSMetric(key="agent_count", label="Agent 数", value=float(len(AGENT_TO_SQUAD)), unit="个"),
        ]

    @staticmethod
    def _compute_health_score(items: list[AgentOSWorkItem]) -> float:
        """计算系统健康分 (0-100)"""
        if not items:
            return 85.0  # 无数据时默认健康
        critical_ratio = sum(1 for i in items if i.risk_level == RiskLevel.CRITICAL) / len(items)
        pending_ratio = sum(1 for i in items if i.status == WorkItemStatus.PENDING) / len(items)
        score = 100 - (critical_ratio * 50) - (pending_ratio * 20)
        return max(0, min(100, round(score, 1)))

    @staticmethod
    def _compute_squad_health(
        items: list[AgentOSWorkItem],
        agents: list[AgentOSAgent],
    ) -> float:
        """计算小队健康分 (0-100)"""
        if not items:
            return 90.0
        critical = sum(1 for i in items if i.risk_level == RiskLevel.CRITICAL)
        high = sum(1 for i in items if i.risk_level == RiskLevel.HIGH)
        score = 100 - (critical * 15) - (high * 5)
        return max(0, min(100, round(score, 1)))

    @staticmethod
    def _compute_agent_risk(items: list[AgentOSWorkItem], agent_id: str) -> RiskLevel:
        """计算 Agent 的风险等级"""
        agent_items = [i for i in items if i.agent_id == agent_id]
        if any(i.risk_level == RiskLevel.CRITICAL for i in agent_items):
            return RiskLevel.CRITICAL
        if any(i.risk_level == RiskLevel.HIGH for i in agent_items):
            return RiskLevel.HIGH
        if any(i.risk_level == RiskLevel.MEDIUM for i in agent_items):
            return RiskLevel.MEDIUM
        return RiskLevel.LOW

    @staticmethod
    def _map_action_status(status: str) -> WorkItemStatus:
        mapping = {
            "pending": WorkItemStatus.PENDING,
            "proposed": WorkItemStatus.PENDING,
            "approved": WorkItemStatus.IN_PROGRESS,
            "confirmed": WorkItemStatus.IN_PROGRESS,
            "executed": WorkItemStatus.COMPLETED,
            "rejected": WorkItemStatus.CANCELLED,
            "failed": WorkItemStatus.FAILED,
        }
        return mapping.get(status, WorkItemStatus.PENDING)

    @staticmethod
    def _map_exception_status(status: str) -> WorkItemStatus:
        mapping = {
            "open": WorkItemStatus.PENDING,
            "assigned": WorkItemStatus.IN_PROGRESS,
            "resolved": WorkItemStatus.COMPLETED,
            "ignored": WorkItemStatus.CANCELLED,
        }
        return mapping.get(status, WorkItemStatus.PENDING)

    @staticmethod
    def _map_listing_status(status: str) -> WorkItemStatus:
        mapping = {
            "ready": WorkItemStatus.PENDING,
            "blocked": WorkItemStatus.BLOCKED,
            "published": WorkItemStatus.COMPLETED,
            "failed": WorkItemStatus.FAILED,
            "cancelled": WorkItemStatus.CANCELLED,
        }
        return mapping.get(status, WorkItemStatus.PENDING)

    @staticmethod
    def _infer_agent_from_alert(alert_type: str) -> str | None:
        mapping = {
            "inventory_low_stock": "A5",
            "inventory_out_of_stock": "A5",
            "order_pending": "A5",
            "listing_failed": "A2",
            "settlement_pending": "A6",
            "settlement_discrepancy": "A6",
        }
        return mapping.get(alert_type)

    # ── Phase 3: Operation Log ──────────────────────────

    @staticmethod
    def _extract_source_type(item_id: str) -> str:
        return item_id.split(":")[0] if ":" in item_id else "unknown"

    @staticmethod
    async def _write_operation_log(
        db: AsyncSession,
        user_id: int,
        item_id: str,
        action: str,
        previous_status: str | None = None,
        new_status: str | None = None,
        comment: str | None = None,
    ) -> None:
        """写入操作审计日志"""
        log = AgentOSOperationLog(
            user_id=user_id,
            item_id=item_id,
            action=action,
            source_type=AgentOSService._extract_source_type(item_id),
            previous_status=previous_status,
            new_status=new_status,
            comment=comment,
        )
        db.add(log)

    # ── Phase 3: Autonomy Upgrade ──────────────────────────

    @staticmethod
    async def get_upgrade_candidates(
        db: AsyncSession,
        user_id: int,
    ) -> list[dict[str, Any]]:
        """获取自治等级升级候选列表"""
        from app.agentos.autonomy_service import batch_suggest_upgrades

        # 从 AgentDecision 统计每个 Agent 的表现
        from app.agent.models import AgentDecision, AgentAction as PendingAgentAction

        agents_data = []
        for agent_id, meta in AGENT_META.items():
            squad_id = AGENT_TO_SQUAD.get(agent_id, "governance")

            # 统计决策数和采纳率
            decisions_total = 0
            decisions_accepted = 0
            try:
                stmt = select(AgentDecision).where(AgentDecision.agent_id == agent_id)
                all_decisions = (await db.execute(stmt)).scalars().all()
                decisions_total = len(all_decisions)
                decisions_accepted = sum(
                    1 for d in all_decisions if d.user_action == "accepted"
                )
            except Exception:
                pass

            # 统计最近风险
            recent_risks = []
            try:
                stmt = (
                    select(PendingAgentAction)
                    .where(PendingAgentAction.agent_id == agent_id)
                    .order_by(PendingAgentAction.created_at.desc())
                    .limit(20)
                )
                actions = (await db.execute(stmt)).scalars().all()
                for a in actions:
                    if a.status == "failed":
                        recent_risks.append("high")
                if not actions:
                    recent_risks.append("low")
            except Exception:
                recent_risks.append("low")

            # 成功率 = (总 - 失败) / 总
            error_count = sum(1 for r in recent_risks if r == "high")
            success_rate = max(0, 1 - (error_count / max(decisions_total, 1)))

            adoption_rate = decisions_accepted / max(decisions_total, 1) if decisions_total else 0

            agents_data.append({
                "id": agent_id,
                "autonomy_level": AutonomyLevel.SUGGESTION.value,
                "success_rate": round(success_rate, 3),
                "adoption_rate": round(adoption_rate, 3),
                "recent_risk_levels": recent_risks,
                "total_decisions": decisions_total,
                "recent_errors": error_count,
            })

        candidates = batch_suggest_upgrades(agents_data)

        # 关联 Agent 元数据
        result = []
        for c in candidates:
            aid = c["agent_id"]
            meta = AGENT_META.get(aid, {})
            squad_id = AGENT_TO_SQUAD.get(aid, "governance")
            result.append({
                "agent_id": aid,
                "agent_name": meta.get("name", aid),
                "squad_id": squad_id,
                "squad_name": SQUAD_TO_NAME.get(squad_id, squad_id),
                "current_level": c.get("current_level", "SUGGESTION"),
                "suggested": c["suggested"],
                "direction": c["direction"],
                "target_level": c["target_level"],
                "confidence": c["confidence"],
                "reason": c["reason"],
            })
        return result

    @staticmethod
    async def execute_upgrade(
        db: AsyncSession,
        user_id: int,
        agent_id: str,
        target_level: str,
    ) -> dict[str, Any]:
        """执行自治等级升级（记录到 operation_log）"""
        # Phase 3 仅记录升级操作，不修改 Agent 模型（需要真实的 Agent 等级字段支持）
        await AgentOSService._write_operation_log(
            db, user_id, f"agent:{agent_id}", "autonomy_upgrade",
            previous_status=None, new_status=target_level,
            comment=f"自治等级升级至 {target_level}",
        )
        return {"ok": True, "agent_id": agent_id, "new_level": target_level}

    @staticmethod
    async def execute_downgrade(
        db: AsyncSession,
        user_id: int,
        agent_id: str,
        target_level: str,
    ) -> dict[str, Any]:
        """执行自治等级降级"""
        await AgentOSService._write_operation_log(
            db, user_id, f"agent:{agent_id}", "autonomy_downgrade",
            previous_status=None, new_status=target_level,
            comment=f"自治等级降级至 {target_level}",
        )
        return {"ok": True, "agent_id": agent_id, "new_level": target_level}

    @staticmethod
    async def get_operations(
        db: AsyncSession,
        item_id: str | None = None,
        action: str | None = None,
        source_type: str | None = None,
        user_id: int | None = None,
        limit: int = 20,
        offset: int = 0,
    ) -> dict[str, Any]:
        """查询操作审计日志"""
        from app.agentos.models import AgentOSOperationLog

        query = select(AgentOSOperationLog)
        if item_id:
            query = query.where(AgentOSOperationLog.item_id == item_id)
        if action:
            query = query.where(AgentOSOperationLog.action == action)
        if source_type:
            query = query.where(AgentOSOperationLog.source_type == source_type)
        if user_id:
            query = query.where(AgentOSOperationLog.user_id == user_id)

        count_q = select(sa_func.count()).select_from(query.subquery())
        total = (await db.execute(count_q)).scalar() or 0

        rows = (
            await db.execute(
                query.order_by(AgentOSOperationLog.created_at.desc())
                .offset(offset)
                .limit(limit)
            )
        ).scalars().all()

        return {
            "records": [AgentOSOperationLogVO.model_validate(r) for r in rows],
            "total": total,
            "limit": limit,
            "offset": offset,
        }

    # ── Phase 2: Mutation 操作 ─────────────────────────────

    @staticmethod
    async def update_work_item_status(
        db: AsyncSession,
        item_id: str,
        user_id: int,
        new_status: str,
    ) -> dict[str, Any]:
        """更新 WorkItem 的底层状态"""
        old_status: str | None = None
        if item_id.startswith("exception:"):
            try:
                uid = int(item_id[len("exception:"):])
                stmt = select(ExceptionItem).where(ExceptionItem.id == uid)
                result = await db.execute(stmt)
                row = result.scalar_one_or_none()
                if not row:
                    return {"ok": False, "error": "not_found"}
                old_status = str(row.status)
                if new_status in ("completed", "resolved"):
                    row.status = "resolved"
                elif new_status == "in_progress":
                    row.status = "assigned"
                elif new_status == "cancelled":
                    row.status = "ignored"
                else:
                    return {"ok": False, "error": f"status '{new_status}' not allowed for exceptions"}
            except ValueError:
                return {"ok": False, "error": "invalid_id"}
        elif item_id.startswith("notification:"):
            try:
                uid = int(item_id[len("notification:"):])
                stmt = select(Notification).where(Notification.id == uid)
                result = await db.execute(stmt)
                row = result.scalar_one_or_none()
                if not row:
                    return {"ok": False, "error": "not_found"}
                if new_status == "completed":
                    row.is_read = 1
                else:
                    row.is_read = 0
            except ValueError:
                return {"ok": False, "error": "invalid_id"}
        elif item_id.startswith("agent_action:"):
            try:
                from app.agent.models import AgentAction as PendingAgentAction
                uid = int(item_id[len("agent_action:"):])
                stmt = select(PendingAgentAction).where(PendingAgentAction.id == uid)
                result = await db.execute(stmt)
                row = result.scalar_one_or_none()
                if not row:
                    return {"ok": False, "error": "not_found"}
                old_status = str(row.status)
                mapping = {
                    "completed": "executed",
                    "in_progress": "confirmed",
                    "cancelled": "rejected",
                    "pending": "pending",
                }
                row.status = mapping.get(new_status, new_status)
            except (ImportError, ValueError):
                return {"ok": False, "error": "invalid_id"}
        elif item_id.startswith("listing_task:"):
            try:
                uid = int(item_id[len("listing_task:"):])
                stmt = select(type("LT", (object,), {"id": int}))
                stmt = select(ExceptionItem).where(ExceptionItem.id == -1)  # fallback
                return {"ok": False, "error": "listing_task update not yet supported"}
            except ValueError:
                return {"ok": False, "error": "invalid_id"}
        else:
            return {"ok": False, "error": f"unknown source_type in '{item_id}'"}

        await AgentOSService._write_operation_log(
            db, user_id, item_id, "status_update",
            previous_status=old_status, new_status=new_status,
        )
        return {"ok": True, "new_status": new_status}

    @staticmethod
    async def approve_work_item(
        db: AsyncSession,
        item_id: str,
        user_id: int,
        comment: str | None = None,
    ) -> dict[str, Any]:
        """审批通过一个 WorkItem，触发底层动作执行"""
        if item_id.startswith("agent_action:"):
            try:
                from app.agent.models import AgentAction as PendingAgentAction
                uid = int(item_id[len("agent_action:"):])
                stmt = select(PendingAgentAction).where(PendingAgentAction.id == uid)
                result = await db.execute(stmt)
                row = result.scalar_one_or_none()
                if not row:
                    return {"ok": False, "error": "not_found"}
                row.status = "confirmed"
            except (ImportError, ValueError):
                return {"ok": False, "error": "invalid_id"}
        else:
            # 非 AgentAction 类型通过 status update 处理即可
            return await AgentOSService.update_work_item_status(db, item_id, user_id, "in_progress")

        await AgentOSService._write_operation_log(
            db, user_id, item_id, "approve",
            previous_status="pending", new_status="in_progress",
            comment=comment,
        )
        return {"ok": True, "action": "approved", "comment": comment}

    @staticmethod
    async def reject_work_item(
        db: AsyncSession,
        item_id: str,
        user_id: int,
        comment: str | None = None,
    ) -> dict[str, Any]:
        """拒绝一个 WorkItem"""
        if item_id.startswith("agent_action:"):
            try:
                from app.agent.models import AgentAction as PendingAgentAction
                uid = int(item_id[len("agent_action:"):])
                stmt = select(PendingAgentAction).where(PendingAgentAction.id == uid)
                result = await db.execute(stmt)
                row = result.scalar_one_or_none()
                if not row:
                    return {"ok": False, "error": "not_found"}
                row.status = "rejected"
            except (ImportError, ValueError):
                return {"ok": False, "error": "invalid_id"}
        else:
            return await AgentOSService.update_work_item_status(db, item_id, user_id, "cancelled")

        await AgentOSService._write_operation_log(
            db, user_id, item_id, "reject",
            previous_status="pending", new_status="cancelled",
            comment=comment,
        )
        return {"ok": True, "action": "rejected", "comment": comment}


# ─── 排序权重 ──────────────────────────────────────────────

_PRIORITY_SORT = {
    WorkItemPriority.CRITICAL: 4,
    WorkItemPriority.HIGH: 3,
    WorkItemPriority.MEDIUM: 2,
    WorkItemPriority.LOW: 1,
}

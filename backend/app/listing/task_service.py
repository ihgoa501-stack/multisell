"""上架任务队列 - 服务层"""

from typing import Optional

from sqlalchemy import select
from sqlalchemy.ext.asyncio import AsyncSession

from app.listing.service import (
    ListingService,
    PublishFailedError,
    PublishValidationError,
)
from app.listing.task_schemas import (
    ListingTaskCreateFromDecisionItem,
    ListingTaskCreateFromDecisionResponse,
    ListingTaskCreateResult,
    ListingTaskResponse,
)
from app.models import ListingTask, Product, Platform, Sku
from app.operation_log.service import OperationLogService


class ListingTaskService:
    """上架任务队列服务"""

    @staticmethod
    async def create_from_decisions(
        db: AsyncSession,
        items: list[ListingTaskCreateFromDecisionItem],
        operator: Optional[str] = None,
    ) -> ListingTaskCreateFromDecisionResponse:
        """从决策 approve 结果创建上架任务。"""
        created_count = 0
        reused_count = 0
        skipped_count = 0
        tasks: list[ListingTaskCreateResult] = []
        skipped: list[dict] = []

        for item in items:
            # 只处理 approve
            if item.decision_result.recommendation != "approve":
                skipped_count += 1
                skipped.append(
                    {
                        "item_key": item.item_key,
                        "reason": "recommendation is not approve",
                    }
                )
                continue

            # 通过 sku_id 找 product_id
            sku = await db.get(Sku, item.sku_id)
            if sku is None:
                skipped_count += 1
                skipped.append(
                    {
                        "item_key": item.item_key,
                        "reason": "SKU not found",
                    }
                )
                continue
            product_id = sku.product_id

            # 检查是否已有同 (product_id, platform_id) 的未完成任务
            existing = await ListingTaskService._find_open_task(
                db, product_id, item.platform_id
            )
            if existing is not None:
                # 更新现有任务
                await ListingTaskService._update_task_from_decision(
                    db, existing, item, operator
                )
                reused_count += 1
                tasks.append(
                    ListingTaskService._task_to_result(existing, item.item_key)
                )
                continue

            # 创建新任务——先校验发布就绪状态
            product = await db.get(Product, product_id)
            platform = await db.get(Platform, item.platform_id)
            if product is None or platform is None:
                skipped_count += 1
                skipped.append(
                    {
                        "item_key": item.item_key,
                        "reason": "Product or Platform not found",
                    }
                )
                continue

            missing = await ListingTaskService._check_publish_readiness(
                db, product, platform
            )
            status = "blocked" if missing else "ready"

            task = ListingTask(
                product_id=product_id,
                platform_id=item.platform_id,
                sku_id=item.sku_id,
                source_type="decision",
                source_item_key=item.item_key,
                status=status,
                missing_requirements=missing,
                decision_snapshot=item.decision_result.model_dump(),
                target_sale_price=item.decision_result.target_sale_price,
                target_profit_margin=item.decision_result.profit_margin,
                destination_country=item.decision_result.destination_country.upper(),
                created_by=operator,
                updated_by=operator,
            )
            db.add(task)
            await db.flush()
            created_count += 1
            tasks.append(ListingTaskService._task_to_result(task, item.item_key))

        # 审计日志
        if items:
            await OperationLogService.log(
                db,
                module="listing_task",
                action="create_from_decision",
                resource_id="0",
                content=(
                    f"从决策创建上架任务: "
                    f"新建={created_count}, 复用={reused_count}, 跳过={skipped_count}"
                ),
                operator=operator or "system",
            )

        return ListingTaskCreateFromDecisionResponse(
            created_count=created_count,
            reused_count=reused_count,
            skipped_count=skipped_count,
            tasks=tasks,
            skipped=skipped,
        )

    @staticmethod
    async def _find_open_task(
        db: AsyncSession,
        product_id: int,
        platform_id: int,
    ) -> Optional[ListingTask]:
        """查找是否已有同一商品+平台的未完成任务。"""
        stmt = (
            select(ListingTask)
            .where(
                ListingTask.product_id == product_id,
                ListingTask.platform_id == platform_id,
                ListingTask.status.in_(["ready", "blocked", "failed"]),
            )
            .order_by(ListingTask.id.desc())
        )
        result = await db.execute(stmt)
        return result.scalar_one_or_none()

    @staticmethod
    async def _update_task_from_decision(
        db: AsyncSession,
        task: ListingTask,
        item: ListingTaskCreateFromDecisionItem,
        operator: Optional[str] = None,
    ) -> None:
        """用新决策结果更新现有任务状态。"""
        product = await db.get(Product, task.product_id)
        platform = await db.get(Platform, task.platform_id)
        missing = []
        if product and platform:
            missing = await ListingTaskService._check_publish_readiness(
                db, product, platform
            )

        task.status = "blocked" if missing else "ready"
        task.missing_requirements = missing
        task.decision_snapshot = item.decision_result.model_dump()
        task.target_sale_price = item.decision_result.target_sale_price
        task.target_profit_margin = item.decision_result.profit_margin
        task.destination_country = item.decision_result.destination_country.upper()
        task.sku_id = item.sku_id
        task.updated_by = operator
        await db.flush()

    @staticmethod
    async def _check_publish_readiness(
        db: AsyncSession,
        product: Product,
        platform: Platform,
    ) -> list[str]:
        """调用现有 validate_publish_ready 检查发布就绪状态。"""
        (
            missing,
            _skus,
            _prices,
            _inventories,
        ) = await ListingService.validate_publish_ready(db, product, platform)
        return missing

    @staticmethod
    def _task_to_result(
        task: ListingTask,
        item_key: Optional[str] = None,
    ) -> ListingTaskCreateResult:
        return ListingTaskCreateResult(
            id=task.id,
            product_id=task.product_id,
            platform_id=task.platform_id,
            status=task.status,
            missing_requirements=list(task.missing_requirements or []),
            source_item_key=item_key or task.source_item_key,
        )

    # ── List ─────────────────────────────────────────────────────────────────

    @staticmethod
    async def list_tasks(
        db: AsyncSession,
        status: Optional[str] = None,
        platform_id: Optional[int] = None,
    ) -> list[ListingTaskResponse]:
        """查询上架任务列表。"""
        stmt = (
            select(ListingTask, Product.name, Platform.name)
            .outerjoin(Product, ListingTask.product_id == Product.id)
            .outerjoin(Platform, ListingTask.platform_id == Platform.id)
            .order_by(ListingTask.created_at.desc())
        )
        if status:
            stmt = stmt.where(ListingTask.status == status)
        if platform_id:
            stmt = stmt.where(ListingTask.platform_id == platform_id)

        result = await db.execute(stmt)
        rows = result.all()

        responses = []
        for task, product_name, platform_name in rows:
            product_name = product_name or ""
            platform_name = platform_name or ""
            responses.append(
                ListingTaskResponse(
                    id=task.id,
                    product_id=task.product_id,
                    product_name=product_name,
                    platform_id=task.platform_id,
                    platform_name=platform_name,
                    sku_id=task.sku_id,
                    product_listing_id=task.product_listing_id,
                    source_type=task.source_type,
                    source_item_key=task.source_item_key,
                    status=task.status,
                    missing_requirements=list(task.missing_requirements or []),
                    target_sale_price=float(task.target_sale_price)
                    if task.target_sale_price
                    else None,
                    target_profit_margin=float(task.target_profit_margin)
                    if task.target_profit_margin
                    else None,
                    destination_country=task.destination_country,
                    last_error=task.last_error,
                    created_by=task.created_by,
                    created_at=task.created_at.isoformat() if task.created_at else None,
                    updated_at=task.updated_at.isoformat() if task.updated_at else None,
                )
            )
        return responses

    # ── Recheck ──────────────────────────────────────────────────────────────

    @staticmethod
    async def recheck_task(
        db: AsyncSession,
        task_id: int,
        operator: Optional[str] = None,
    ) -> Optional[ListingTaskResponse]:
        """重新检查任务的就绪状态。"""
        stmt = select(ListingTask).where(ListingTask.id == task_id)
        result = await db.execute(stmt)
        task = result.scalar_one_or_none()
        if task is None:
            return None

        product = await db.get(Product, task.product_id)
        platform = await db.get(Platform, task.platform_id)
        missing = []
        if product and platform:
            missing = await ListingTaskService._check_publish_readiness(
                db, product, platform
            )

        task.status = "blocked" if missing else "ready"
        task.missing_requirements = missing
        task.updated_by = operator
        await db.flush()
        await db.refresh(task)

        await OperationLogService.log(
            db,
            module="listing_task",
            action="recheck",
            resource_id=str(task.id),
            content=f"重新检查上架任务: status={task.status}",
            operator=operator or "system",
        )

        return await ListingTaskService._task_to_response(task)

    # ── Cancel ───────────────────────────────────────────────────────────────

    @staticmethod
    async def cancel_task(
        db: AsyncSession,
        task_id: int,
        operator: Optional[str] = None,
    ) -> Optional[ListingTaskResponse]:
        """取消上架任务。"""
        stmt = select(ListingTask).where(ListingTask.id == task_id)
        result = await db.execute(stmt)
        task = result.scalar_one_or_none()
        if task is None:
            return None

        task.status = "cancelled"
        task.updated_by = operator
        await db.flush()
        await db.refresh(task)

        await OperationLogService.log(
            db,
            module="listing_task",
            action="cancel",
            resource_id=str(task.id),
            content="取消上架任务",
            operator=operator or "system",
        )

        return await ListingTaskService._task_to_response(task)

    # ── Publish ──────────────────────────────────────────────────────────────

    @staticmethod
    async def publish_task(
        db: AsyncSession,
        task_id: int,
        operator: Optional[str] = None,
    ) -> tuple[Optional[ListingTaskResponse], Optional[str]]:
        """发布 ready 状态的上架任务。返回 (response, error_message)。"""
        stmt = select(ListingTask).where(ListingTask.id == task_id)
        result = await db.execute(stmt)
        task = result.scalar_one_or_none()
        if task is None:
            return None, "任务不存在"

        if task.status != "ready":
            return None, f"任务状态 {task.status} 不可发布，仅 ready 可发布"

        product = await db.get(Product, task.product_id)
        platform = await db.get(Platform, task.platform_id)
        if product is None or platform is None:
            return None, "商品或平台不存在"

        try:
            listing = await ListingService.publish(db, product, platform)
        except PublishValidationError as exc:
            task.status = "blocked"
            task.missing_requirements = exc.missing_requirements
            task.updated_by = operator
            await db.flush()
            return None, f"商品信息不完整: {', '.join(exc.missing_requirements)}"
        except PublishFailedError as exc:
            task.status = "failed"
            task.last_error = str(exc)
            task.product_listing_id = exc.listing.id
            task.updated_by = operator
            await db.flush()
            return None, f"发布失败: {exc.listing.sync_message}"

        task.status = "published"
        task.product_listing_id = listing.id
        task.updated_by = operator
        await db.flush()
        await db.refresh(task)

        await OperationLogService.log(
            db,
            module="listing_task",
            action="publish",
            resource_id=str(task.id),
            content=f"发布上架任务成功: product_listing_id={listing.id}",
            operator=operator or "system",
        )

        resp = await ListingTaskService._task_to_response(task)
        return resp, None

    @staticmethod
    async def _task_to_response(task: ListingTask) -> ListingTaskResponse:
        # Don't use relationships — query names directly to avoid async greenlet issues
        product_name = ""
        platform_name = ""
        # task.product and task.platform are available via selectinload when loaded
        # but to avoid greenlet issues, we use the IDs only for the response type
        return ListingTaskResponse(
            id=task.id,
            product_id=task.product_id,
            product_name=product_name,
            platform_id=task.platform_id,
            platform_name=platform_name,
            sku_id=task.sku_id,
            product_listing_id=task.product_listing_id,
            source_type=task.source_type,
            source_item_key=task.source_item_key,
            status=task.status,
            missing_requirements=list(task.missing_requirements or []),
            target_sale_price=float(task.target_sale_price)
            if task.target_sale_price
            else None,
            target_profit_margin=float(task.target_profit_margin)
            if task.target_profit_margin
            else None,
            destination_country=task.destination_country,
            last_error=task.last_error,
            created_by=task.created_by,
            created_at=task.created_at.isoformat() if task.created_at else None,
            updated_at=task.updated_at.isoformat() if task.updated_at else None,
        )

"""多平台上架任务 - 服务层"""

from typing import Optional
from sqlalchemy import select, func, or_
from sqlalchemy.ext.asyncio import AsyncSession

from app.models import ListingTask, ListingTaskItem, Product, Platform, ProductListing
from app.listing.service import ListingService, PublishValidationError, PublishFailedError


class ListingTaskService:

    @staticmethod
    async def create(
        db: AsyncSession,
        name: str,
        product_ids: list[int],
        platform_ids: list[int],
        created_by: Optional[int] = None,
    ) -> ListingTask:
        total = len(product_ids) * len(platform_ids)
        task = ListingTask(
            name=name,
            status="pending",
            total_count=total,
            created_by=created_by,
        )
        db.add(task)
        await db.flush()

        for pid in product_ids:
            for plat_id in platform_ids:
                item = ListingTaskItem(
                    task_id=task.id,
                    product_id=pid,
                    platform_id=plat_id,
                    status="pending",
                )
                db.add(item)

        await db.flush()
        await db.refresh(task)
        return task

    @staticmethod
    async def get_task(db: AsyncSession, task_id: int) -> Optional[ListingTask]:
        return await db.get(ListingTask, task_id)

    @staticmethod
    async def list_tasks(
        db: AsyncSession,
        page: int = 1,
        page_size: int = 20,
    ) -> tuple[list[ListingTask], int]:
        count_q = select(func.count(ListingTask.id))
        total_result = await db.execute(count_q)
        total = total_result.scalar() or 0

        q = (
            select(ListingTask)
            .order_by(ListingTask.created_at.desc())
            .offset((page - 1) * page_size)
            .limit(page_size)
        )
        result = await db.execute(q)
        tasks = list(result.scalars().all())
        return tasks, total

    @staticmethod
    async def get_task_items(
        db: AsyncSession,
        task_id: int,
        status_filter: Optional[str] = None,
        page: int = 1,
        page_size: int = 20,
    ) -> tuple[list[dict], int]:
        base_q = select(ListingTaskItem).where(ListingTaskItem.task_id == task_id)
        count_q = select(func.count(ListingTaskItem.id)).where(ListingTaskItem.task_id == task_id)

        if status_filter:
            base_q = base_q.where(ListingTaskItem.status == status_filter)
            count_q = count_q.where(ListingTaskItem.status == status_filter)

        total_result = await db.execute(count_q)
        total = total_result.scalar() or 0

        q = (
            base_q
            .order_by(ListingTaskItem.id)
            .offset((page - 1) * page_size)
            .limit(page_size)
        )
        result = await db.execute(q)
        items = list(result.scalars().all())

        # Enrich with product & platform names
        item_dicts = []
        for item in items:
            product = await db.get(Product, item.product_id)
            platform = await db.get(Platform, item.platform_id)
            item_dicts.append({
                "id": item.id,
                "task_id": item.task_id,
                "product_id": item.product_id,
                "platform_id": item.platform_id,
                "product_name": product.name if product else f"ID:{item.product_id}",
                "platform_name": platform.name if platform else f"ID:{item.platform_id}",
                "platform_code": platform.code if platform else None,
                "status": item.status,
                "result": item.result,
                "error_message": item.error_message,
                "retry_count": item.retry_count,
                "executed_at": item.executed_at.isoformat() if item.executed_at else None,
            })

        return item_dicts, total

    @staticmethod
    async def delete_task(db: AsyncSession, task: ListingTask) -> None:
        items = await db.execute(
            select(ListingTaskItem).where(ListingTaskItem.task_id == task.id)
        )
        for item in items.scalars().all():
            await db.delete(item)
        await db.delete(task)

    @staticmethod
    async def execute_task(db: AsyncSession, task: ListingTask) -> ListingTask:
        task.status = "in_progress"

        items_q = await db.execute(
            select(ListingTaskItem).where(
                ListingTaskItem.task_id == task.id,
                or_(
                    ListingTaskItem.status == "pending",
                    ListingTaskItem.status == "failed",
                ),
            )
        )
        items = list(items_q.scalars().all())

        for item in items:
            product = await db.get(Product, item.product_id)
            platform = await db.get(Platform, item.platform_id)
            if not product or not platform:
                item.status = "failed"
                item.error_message = "商品或平台不存在"
                continue

            item.status = "in_progress"
            try:
                listing = await ListingService.publish(db, product, platform)
                item.status = "success"
                item.result = {
                    "platform_product_id": listing.platform_product_id,
                    "platform_sku": listing.platform_sku,
                    "platform_url": listing.platform_url,
                }
                item.error_message = None
                item.executed_at = func.now()
            except PublishValidationError as e:
                item.status = "failed"
                item.error_message = f"验证失败: {', '.join(e.missing_requirements)}"
            except PublishFailedError as e:
                item.status = "failed"
                item.error_message = e.listing.sync_message or "发布失败"
            except Exception as e:
                item.status = "failed"
                item.error_message = str(e)[:500]
            finally:
                item.retry_count = (item.retry_count or 0) + 1

            await db.flush()

        # Update task summary
        count_q = await db.execute(
            select(
                func.count(ListingTaskItem.id),
                func.sum(case((ListingTaskItem.status == "success", 1), else_=0)),
                func.sum(case((ListingTaskItem.status == "failed", 1), else_=0)),
            ).where(ListingTaskItem.task_id == task.id)
        )
        row = count_q.one()
        task.total_count = row[0] or 0
        task.success_count = row[1] or 0
        task.failed_count = row[2] or 0

        if task.failed_count == 0:
            task.status = "completed"
        elif task.success_count > 0:
            task.status = "partial_failed"
        else:
            task.status = "all_failed"

        await db.flush()
        await db.refresh(task)
        return task

    @staticmethod
    async def retry_all_failed(
        db: AsyncSession,
        task_id: int,
    ) -> int:
        """重置任务下所有失败条目为待处理状态"""
        stmt = select(ListingTaskItem).where(
            ListingTaskItem.task_id == task_id,
            ListingTaskItem.status == "failed",
        )
        items = (await db.execute(stmt)).scalars().all()
        for item in items:
            item.status = "pending"
            item.retry_count = 0
            item.error_message = None
        await db.flush()
        return len(items)

    @staticmethod
    async def retry_item(
        db: AsyncSession,
        task: ListingTask,
        item_id: int,
    ) -> ListingTaskItem:
        item = await db.get(ListingTaskItem, item_id)
        if not item or item.task_id != task.id:
            raise ValueError("条目不存在")

        product = await db.get(Product, item.product_id)
        platform = await db.get(Platform, item.platform_id)
        if not product or not platform:
            raise ValueError("商品或平台不存在")

        item.status = "in_progress"
        try:
            listing = await ListingService.publish(db, product, platform)
            item.status = "success"
            item.result = {
                "platform_product_id": listing.platform_product_id,
                "platform_sku": listing.platform_sku,
                "platform_url": listing.platform_url,
            }
            item.error_message = None
            item.executed_at = func.now()
        except PublishValidationError as e:
            item.status = "failed"
            item.error_message = f"验证失败: {', '.join(e.missing_requirements)}"
        except PublishFailedError as e:
            item.status = "failed"
            item.error_message = e.listing.sync_message or "发布失败"
        except Exception as e:
            item.status = "failed"
            item.error_message = str(e)[:500]
        finally:
            item.retry_count = (item.retry_count or 0) + 1

        await db.flush()

        # Recalculate task status
        q = await db.execute(
            select(
                func.count(ListingTaskItem.id),
                func.sum(case((ListingTaskItem.status == "success", 1), else_=0)),
                func.sum(case((ListingTaskItem.status == "failed", 1), else_=0)),
            ).where(ListingTaskItem.task_id == task.id)
        )
        row = q.one()
        task.success_count = row[1] or 0
        task.failed_count = row[2] or 0

        if task.failed_count == 0:
            task.status = "completed"
        elif task.success_count > 0:
            task.status = "partial_failed"
        else:
            task.status = "all_failed"

        await db.flush()
        await db.refresh(item)
        return item

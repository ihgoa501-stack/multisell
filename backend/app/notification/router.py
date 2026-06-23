"""通知与预警 - 路由"""

from fastapi import APIRouter, Depends, Query
from sqlalchemy.ext.asyncio import AsyncSession

from app.auth import require_permission
from app.common import Result, PageResult
from app.database import get_db
from app.models import User
from app.notification.schemas import AlertRuleUpdate
from app.notification.service import NotificationService

router = APIRouter(tags=["通知与预警"])


# ── 通知 ─────────────────────────────────────────────────────


@router.get("/notifications", summary="通知列表")
async def list_notifications(
    unread_only: bool = Query(False),
    alert_type: str = Query(None),
    page: int = Query(1, ge=1),
    page_size: int = Query(20, ge=1, le=100),
    db: AsyncSession = Depends(get_db),
    current_user: User = Depends(require_permission("notification:view")),
):
    rows, total = await NotificationService.list_notifications(
        db, current_user.id, unread_only, alert_type, page, page_size
    )
    return PageResult.ok(records=rows, total=total, page=page, page_size=page_size)


@router.get("/notifications/unread-count", summary="未读通知数")
async def get_unread_count(
    db: AsyncSession = Depends(get_db),
    current_user: User = Depends(require_permission("notification:view")),
):
    count = await NotificationService.get_unread_count(db, current_user.id)
    return Result.ok(count)


@router.put("/notifications/{notification_id}/read", summary="标记已读")
async def mark_read(
    notification_id: int,
    db: AsyncSession = Depends(get_db),
    current_user: User = Depends(require_permission("notification:update")),
):
    ok = await NotificationService.mark_read(db, notification_id, current_user.id)
    if not ok:
        return Result.not_found("通知不存在")
    return Result.ok(message="已标记已读")


@router.put("/notifications/read-all", summary="全部标记已读")
async def mark_all_read(
    db: AsyncSession = Depends(get_db),
    current_user: User = Depends(require_permission("notification:update")),
):
    count = await NotificationService.mark_all_read(db, current_user.id)
    return Result.ok({"marked": count})


@router.delete("/notifications/{notification_id}", summary="删除通知")
async def delete_notification(
    notification_id: int,
    db: AsyncSession = Depends(get_db),
    current_user: User = Depends(require_permission("notification:delete")),
):
    ok = await NotificationService.delete_notification(
        db, notification_id, current_user.id
    )
    return Result.ok(message="已删除") if ok else Result.not_found("通知不存在")


@router.post("/notifications/check", summary="触发预警检查")
async def trigger_alert_check(
    db: AsyncSession = Depends(get_db),
    current_user: User = Depends(require_permission("notification:manage")),
):
    results = await NotificationService.check_and_create_alerts(db)
    created = {k: v for k, v in results.items() if v and v > 0}
    return Result.ok({"checked_types": len(results), "created": created})


# ── 预警规则 ────────────────────────────────────────────────


@router.post("/alert-rules/initialize", summary="初始化默认预警规则")
async def initialize_alert_rules(
    db: AsyncSession = Depends(get_db),
    current_user: User = Depends(require_permission("notification:manage")),
):
    rules = await NotificationService.initialize_rules(db)
    return Result.ok(
        {
            "created": len(rules),
            "message": f"已创建 {len(rules)} 条默认规则",
        }
    )


@router.get("/alert-rules", summary="预警规则列表")
async def list_alert_rules(
    db: AsyncSession = Depends(get_db),
    current_user: User = Depends(require_permission("notification:view")),
):
    rules = await NotificationService.list_rules(db)
    return Result.ok(
        [
            {
                "id": r.id,
                "name": r.name,
                "alert_type": r.alert_type,
                "enabled": bool(r.enabled),
                "config": r.config or {},
                "description": r.description,
            }
            for r in rules
        ]
    )


@router.put("/alert-rules/{rule_id}", summary="更新预警规则")
async def update_alert_rule(
    rule_id: int,
    data: AlertRuleUpdate,
    db: AsyncSession = Depends(get_db),
    current_user: User = Depends(require_permission("notification:manage")),
):
    rule = await NotificationService.update_rule(
        db, rule_id, data.model_dump(exclude_unset=True)
    )
    if not rule:
        return Result.not_found("规则不存在")
    return Result.ok(
        {
            "id": rule.id,
            "name": rule.name,
            "enabled": bool(rule.enabled),
        }
    )

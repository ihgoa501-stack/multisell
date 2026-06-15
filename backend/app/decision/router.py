"""上架前经营决策 - 路由"""

from fastapi import APIRouter, Depends, HTTPException
from sqlalchemy.ext.asyncio import AsyncSession

from app.auth import require_permission
from app.common.schemas import Result
from app.database import get_db
from app.decision.schemas import PreListingDecisionRequest, PreListingDecisionResponse
from app.decision.service import PreListingDecisionService
from app.models import User

router = APIRouter(prefix="/decisions", tags=["上架决策"])


@router.post("/prelisting", summary="上架前经营决策")
async def prelisting_decision(
    data: PreListingDecisionRequest,
    db: AsyncSession = Depends(get_db),
    current_user: User = Depends(require_permission("decision:calculate")),
) -> Result[PreListingDecisionResponse]:
    try:
        result = await PreListingDecisionService.calculate(db, data)
        return Result.ok(result)
    except ValueError as e:
        raise HTTPException(status_code=404, detail=str(e))

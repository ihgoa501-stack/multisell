"""上架前经营决策 - 服务层"""

from sqlalchemy import select
from sqlalchemy.ext.asyncio import AsyncSession

from app.models import Sku
from app.platform_fee.schemas import PlatformFeeCalculateRequest
from app.platform_fee.service import PlatformFeeService
from app.shipping.schemas import CalculateRequest, CalculateResponse
from app.shipping.service import CalculateService
from app.decision.schemas import (
    PreListingDecisionRequest,
    PreListingDecisionResponse,
)


class PreListingDecisionService:
    @staticmethod
    async def calculate(
        db: AsyncSession,
        req: PreListingDecisionRequest,
    ) -> PreListingDecisionResponse:
        # 1. 校验 SKU 存在
        stmt = select(Sku).where(Sku.id == req.sku_id)
        result = await db.execute(stmt)
        sku = result.scalar_one_or_none()
        if sku is None:
            raise ValueError("SKU不存在")

        product_cost = float(sku.cost_price or 0)

        # 2. 计算运费
        blocking_reasons: list[str] = []
        warnings: list[str] = []

        shipping_fee = 0.0
        try:
            calc_req = CalculateRequest(
                mode="sku",
                sku_id=req.sku_id,
                quantity=1,
                destination_country=req.destination_country,
                cargo_type=req.cargo_type,
            )
            calc_resp: CalculateResponse = await CalculateService.calculate(
                db, calc_req
            )
            if not calc_resp.results:
                blocking_reasons.append("无可用物流渠道报价")
            else:
                # 取最低运费
                shipping_fee = calc_resp.results[0].total_shipping_fee
        except ValueError as e:
            msg = str(e)
            if "物流数据不完整" in msg or "缺失" in msg:
                blocking_reasons.append(f"缺少物流数据：{msg}")
            else:
                blocking_reasons.append(msg)

        # 3. 计算平台费用
        platform_fee = 0.0
        if req.platform_id:
            try:
                fee_result = await PlatformFeeService.calculate_fee(
                    db,
                    PlatformFeeCalculateRequest(
                        platform_id=req.platform_id,
                        country_code=req.destination_country,
                        sale_price=req.target_sale_price,
                    ),
                )
                platform_fee = fee_result.total_fee
                if fee_result.rules_matched == 0:
                    warnings.append("未匹配到平台费用规则，平台费为0")
            except Exception as e:
                warnings.append(f"平台费用计算异常: {e}")
                platform_fee = req.target_sale_price * req.platform_fee_pct / 100
        else:
            platform_fee = req.target_sale_price * req.platform_fee_pct / 100

        # 4. 计算支付费用及其他
        payment_fee = req.target_sale_price * req.payment_fee_pct / 100

        total_cost = (
            product_cost + shipping_fee + platform_fee + payment_fee + req.other_fee
        )
        profit_amount = req.target_sale_price - total_cost
        profit_margin = (
            (profit_amount / req.target_sale_price * 100)
            if req.target_sale_price > 0
            else 0
        )
        profit_margin = round(profit_margin, 2)
        profit_amount = round(profit_amount, 2)

        # 5. 决策逻辑
        if blocking_reasons:
            recommendation = "needs_data"
        elif profit_margin >= req.minimum_margin_pct:
            recommendation = "approve"
            if profit_margin < req.minimum_margin_pct + 5:
                warnings.append(
                    f"利润率仅{profit_margin}%，接近最低阈值{req.minimum_margin_pct}%"
                )
        else:
            recommendation = "reject"
            blocking_reasons.append(
                f"利润率{profit_margin}%低于最低要求{req.minimum_margin_pct}%"
            )

        return PreListingDecisionResponse(
            sku_id=req.sku_id,
            destination_country=req.destination_country.upper(),
            target_sale_price=req.target_sale_price,
            product_cost=product_cost,
            shipping_fee=shipping_fee,
            platform_fee=round(platform_fee, 2),
            payment_fee=round(payment_fee, 2),
            other_fee=req.other_fee,
            profit_amount=profit_amount,
            profit_margin=profit_margin,
            recommendation=recommendation,
            blocking_reasons=blocking_reasons,
            warnings=warnings,
        )

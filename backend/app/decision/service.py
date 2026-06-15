"""上架前经营决策 - 服务层"""

from sqlalchemy import select
from sqlalchemy.ext.asyncio import AsyncSession

from app.models import Sku, Product
from app.shipping.schemas import CalculateRequest, CalculateResponse
from app.shipping.service import CalculateService
from app.decision.schemas import (
    PreListingDecisionBatchItemResult,
    PreListingDecisionBatchRequest,
    PreListingDecisionBatchResponse,
    PreListingDecisionBatchSummary,
    PreListingDecisionRequest,
    PreListingDecisionResponse,
)
from app.finance.cost_layers import COST_LAYER_ESTIMATED
from app.platform_fee.schemas import PlatformFeeRuleMatchRequest
from app.platform_fee.service import PlatformFeeRuleService


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
            calc_resp: CalculateResponse = await CalculateService.calculate(db, calc_req)
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

        # 3. 计算费用 — 优先匹配平台费用规则
        platform_fee_source = "manual"
        applied_platform_fee_rule_id = None
        platform_fee_rule_summary = None
        fixed_fee = 0.0
        advertising_fee = 0.0
        other_fee = req.other_fee

        platform_fee_pct = req.platform_fee_pct
        payment_fee_pct = req.payment_fee_pct

        if req.platform_id is not None:
            # 尝试从SKU所属商品获取类目ID
            product_stmt = select(Product).where(Product.id == sku.product_id)
            product_result = await db.execute(product_stmt)
            product = product_result.scalar_one_or_none()
            category_id = req.category_id if req.category_id is not None else (product.category_id if product else None)

            matched_rule = await PlatformFeeRuleService.match(
                db,
                PlatformFeeRuleMatchRequest(
                    platform_id=req.platform_id,
                    site_code=req.destination_country,
                    category_id=category_id,
                ),
            )
            if matched_rule:
                platform_fee_source = "rule"
                applied_platform_fee_rule_id = matched_rule["id"]
                platform_fee_rule_summary = (
                    f"{matched_rule.get('platform_name') or req.platform_id} "
                    f"{matched_rule.get('site_code') or 'GLOBAL'}"
                )
                platform_fee_pct = matched_rule["commission_pct"]
                payment_fee_pct = matched_rule["payment_fee_pct"]
                fixed_fee = matched_rule["fixed_fee"]
                advertising_fee = req.target_sale_price * matched_rule["advertising_pct"] / 100
                other_fee = matched_rule["other_reserve_fee"]
            else:
                warnings.append("未匹配到平台费用规则，使用手动输入费率")

        platform_fee = req.target_sale_price * platform_fee_pct / 100
        payment_fee = req.target_sale_price * payment_fee_pct / 100

        total_cost = product_cost + shipping_fee + platform_fee + payment_fee + fixed_fee + advertising_fee + other_fee
        profit_amount = req.target_sale_price - total_cost
        profit_margin = (profit_amount / req.target_sale_price * 100) if req.target_sale_price > 0 else 0
        profit_margin = round(profit_margin, 2)
        profit_amount = round(profit_amount, 2)

        # 4. 决策逻辑
        if blocking_reasons:
            recommendation = "needs_data"
        elif profit_margin >= req.minimum_margin_pct:
            recommendation = "approve"
            if profit_margin < req.minimum_margin_pct + 5:
                warnings.append(f"利润率仅{profit_margin}%，接近最低阈值{req.minimum_margin_pct}%")
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
            shipping_cost_layer=COST_LAYER_ESTIMATED,
            platform_fee=round(platform_fee, 2),
            platform_fee_cost_layer=COST_LAYER_ESTIMATED,
            payment_fee=round(payment_fee, 2),
            fixed_fee=round(fixed_fee, 2),
            advertising_fee=round(advertising_fee, 2),
            other_fee=other_fee,
            profit_amount=profit_amount,
            profit_margin=profit_margin,
            recommendation=recommendation,
            blocking_reasons=blocking_reasons,
            warnings=warnings,
            applied_platform_fee_rule_id=applied_platform_fee_rule_id,
            platform_fee_source=platform_fee_source,
            platform_fee_rule_summary=platform_fee_rule_summary,
        )

    @staticmethod
    async def calculate_batch(
        db: AsyncSession,
        req: PreListingDecisionBatchRequest,
    ) -> PreListingDecisionBatchResponse:
        item_results: list[PreListingDecisionBatchItemResult] = []
        approve_count = 0
        reject_count = 0
        needs_data_count = 0
        error_count = 0
        margin_sum = 0.0
        margin_count = 0

        for index, item in enumerate(req.items):
            try:
                single_req = PreListingDecisionRequest(
                    **item.model_dump(exclude={"item_key"})
                )
                result = await PreListingDecisionService.calculate(db, single_req)
                if result.recommendation == "approve":
                    approve_count += 1
                elif result.recommendation == "reject":
                    reject_count += 1
                elif result.recommendation == "needs_data":
                    needs_data_count += 1

                margin_sum += result.profit_margin
                margin_count += 1
                item_results.append(
                    PreListingDecisionBatchItemResult(
                        index=index,
                        item_key=item.item_key,
                        sku_id=item.sku_id,
                        status="success",
                        result=result,
                        error_message=None,
                    )
                )
            except ValueError as exc:
                error_count += 1
                item_results.append(
                    PreListingDecisionBatchItemResult(
                        index=index,
                        item_key=item.item_key,
                        sku_id=item.sku_id,
                        status="error",
                        result=None,
                        error_message=str(exc),
                    )
                )

        success_count = len(req.items) - error_count
        average_profit_margin = round(margin_sum / margin_count, 2) if margin_count else 0

        return PreListingDecisionBatchResponse(
            summary=PreListingDecisionBatchSummary(
                total_items=len(req.items),
                success_count=success_count,
                error_count=error_count,
                approve_count=approve_count,
                reject_count=reject_count,
                needs_data_count=needs_data_count,
                average_profit_margin=average_profit_margin,
            ),
            items=item_results,
        )

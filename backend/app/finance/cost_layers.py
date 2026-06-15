"""成本来源层枚举与工具函数。

成本层语义：

- estimated  上架前/报价阶段预估，或订购后无快照时使用订单上手动输入的费用
- snapshot   订单执行时从运费计算器获取的快照
- actual     从物流商/平台实际账单导入的实际费用
- allocated  头程/FBA/海外仓费用分摊到SKU/订单的结果
- mixed      订单/商品的不同费用项来自不同成本层
"""

from typing import Optional


COST_LAYER_ESTIMATED = "estimated"
COST_LAYER_SNAPSHOT = "snapshot"
COST_LAYER_ACTUAL = "actual"
COST_LAYER_ALLOCATED = "allocated"
COST_LAYER_MIXED = "mixed"

ALL_COST_LAYERS = [
    COST_LAYER_ESTIMATED,
    COST_LAYER_SNAPSHOT,
    COST_LAYER_ACTUAL,
    COST_LAYER_ALLOCATED,
    COST_LAYER_MIXED,
]


def resolve_profit_cost_layer(
    shipping_layer: Optional[str],
    platform_fee_layer: Optional[str],
    product_cost_layer: Optional[str] = None,
) -> str:
    """根据各项费用的成本层解析整体利润成本层。"""
    layers = {shipping_layer, platform_fee_layer, product_cost_layer}
    layers.discard(None)
    if len(layers) <= 1:
        return next(iter(layers)) if layers else COST_LAYER_ESTIMATED
    return COST_LAYER_MIXED

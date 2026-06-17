"""Agent 实现集合"""
from app.agent.agents.discount_risk import G3DiscountRiskAgent
from app.agent.agents.inventory_alert import A5InventoryAlertAgent
from app.agent.agents.profit_watch import A6ProfitWatchAgent
from app.agent.agents.dashboard import G1DashboardAgent
from app.agent.agents.ad_advice import A3AdAdviceAgent
from app.agent.agents.product_scout import A1ProductScoutAgent
from app.agent.agents.listing_optimizer import A2ListingOptimizerAgent
from app.agent.agents.customer_service import A4CustomerServiceAgent
from app.agent.agents.compliance import A7ComplianceGuardAgent
from app.agent.agents.warehouse_customs import G2WarehouseCustomsAgent

__all__ = [
    "G3DiscountRiskAgent", "A5InventoryAlertAgent",
    "A6ProfitWatchAgent", "G1DashboardAgent",
    "A3AdAdviceAgent",
    "A1ProductScoutAgent", "A2ListingOptimizerAgent",
    "A4CustomerServiceAgent", "A7ComplianceGuardAgent",
    "G2WarehouseCustomsAgent",
]

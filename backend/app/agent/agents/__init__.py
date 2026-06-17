"""Agent 实现集合"""
from app.agent.agents.discount_risk import G3DiscountRiskAgent
from app.agent.agents.inventory_alert import A5InventoryAlertAgent
from app.agent.agents.profit_watch import A6ProfitWatchAgent
from app.agent.agents.dashboard import G1DashboardAgent

__all__ = [
    "G3DiscountRiskAgent", "A5InventoryAlertAgent",
    "A6ProfitWatchAgent", "G1DashboardAgent",
]

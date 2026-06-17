"""AI Agent 系统 — Hermes 自进化 Agent 框架"""
from app.agent.router import router
from app.agent.entropy import router as entropy_router

router.include_router(entropy_router, prefix="")

__all__ = ["router"]

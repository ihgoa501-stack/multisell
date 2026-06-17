"""AI Agent 系统 — Hermes 自进化 Agent 框架

熵系统路由由 main.py 的 _discover_routers 自动发现并注册到 /api，
不需要在此处手动 include_router，避免重复注册。
"""
from app.agent.router import router

__all__ = ["router"]

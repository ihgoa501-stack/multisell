"""AI Agent 系统 — Hermes 自进化 Agent 框架

使用惰性导入避免与 app.models → app.agent.models 的循环依赖。
Python 的 import 系统会在首次导入子模块时将其设为包属性，
因此 __getattr__ 导入子模块后需主动清理该属性，防止子模块对象
遮蔽 APIRouter 的访问。
"""
import importlib as _importlib
import sys as _sys

_router = None


def __getattr__(name):
    if name == "router":
        global _router
        if _router is None:
            router_mod = _importlib.import_module("app.agent.router")
            config_mod = _importlib.import_module("app.agent.config_router")
            r = router_mod.router
            r.include_router(config_mod.router)
            _router = r
            # 清除 Python import 系统设置的子模块属性，
            # 避免 module 对象遮蔽 APIRouter
            try:
                delattr(_sys.modules[__name__], "router")
            except (AttributeError, KeyError):
                pass
        return _router
    raise AttributeError(f"module {__name__!r} has no attribute {name!r}")


__all__ = ["router"]

"""系统配置管理服务

管理 LLM Key、提供商、模型选择等运行时配置。
配置存储在 system_config 表，环境变量作为启动默认值。
"""
import logging
from typing import Any, Optional
from sqlalchemy import select, func
from sqlalchemy.ext.asyncio import AsyncSession
from app.models import SystemConfig

logger = logging.getLogger(__name__)

SENSITIVE_KEYS = {"openai_api_key", "anthropic_api_key"}

# 配置定义
CONFIG_DEFS = {
    "openai_api_key": {
        "label": "OpenAI API Key",
        "type": "secret",
        "description": "OpenAI / 兼容接口的 API Key",
        "default": "",
    },
    "anthropic_api_key": {
        "label": "Anthropic API Key",
        "type": "secret",
        "description": "Anthropic Claude API Key",
        "default": "",
    },
    "openai_base_url": {
        "label": "OpenAI API 地址",
        "type": "string",
        "description": "OpenAI 兼容接口地址（支持 Ollama / LiteLLM 等）",
        "default": "https://api.openai.com/v1",
    },
    "default_llm_provider": {
        "label": "默认 LLM 提供商",
        "type": "select",
        "options": ["openai", "anthropic", "ollama"],
        "description": "未指定 Agent 时使用的默认提供商",
        "default": "openai",
    },
    "default_llm_model": {
        "label": "默认模型",
        "type": "string",
        "description": "默认模型名称，如 gpt-4o-mini / claude-sonnet-4-20250514",
        "default": "gpt-4o-mini",
    },
    "agent_model_overrides": {
        "label": "Agent 模型覆盖",
        "type": "json",
        "description": '各 Agent 独立模型配置，如 {"A5": "gpt-4o", "A3": "gpt-4o-mini"}',
        "default": {},
    },
}


class ConfigService:

    @staticmethod
    async def get_all(db: AsyncSession) -> dict:
        """获取所有配置（脱敏）"""
        stmt = select(SystemConfig).order_by(SystemConfig.config_key)
        result = await db.execute(stmt)
        rows = result.scalars().all()
        configs = {}
        for row in rows:
            if row.config_key in SENSITIVE_KEYS and row.config_value:
                configs[row.config_key] = "••••••••"  # 脱敏
            elif row.config_json is not None:
                configs[row.config_key] = row.config_json
            else:
                configs[row.config_key] = row.config_value or ""
        return configs

    @staticmethod
    async def get(db: AsyncSession, key: str) -> Any:
        """获取单个配置值"""
        stmt = select(SystemConfig).where(SystemConfig.config_key == key)
        result = await db.execute(stmt)
        row = result.scalar_one_or_none()
        if not row:
            return CONFIG_DEFS.get(key, {}).get("default")
        if row.config_json is not None:
            return row.config_json
        return row.config_value or ""

    @staticmethod
    async def set(db: AsyncSession, key: str, value: Any, user_id: int = 0) -> bool:
        """设置配置值"""
        stmt = select(SystemConfig).where(SystemConfig.config_key == key)
        result = await db.execute(stmt)
        row = result.scalar_one_or_none()
        # 判断类型：list/dict 存 json，其他存 text
        is_json = isinstance(value, (list, dict))
        if row:
            if is_json:
                row.config_json = value
                row.config_value = None
            else:
                row.config_value = str(value)
                row.config_json = None
            row.updated_by = user_id
        else:
            row = SystemConfig(
                config_key=key,
                config_value=None if is_json else str(value),
                config_json=value if is_json else None,
                is_secret=1 if key in SENSITIVE_KEYS else 0,
                description=CONFIG_DEFS.get(key, {}).get("label", ""),
                updated_by=user_id,
            )
            db.add(row)
        await db.flush()
        return True

    @staticmethod
    async def get_llm_config(db: AsyncSession) -> dict:
        """获取当前 LLM 完整配置（含 Agent 覆盖）"""
        configs = await ConfigService.get_all(db)
        default_provider = configs.get("default_llm_provider", "openai")
        default_model = configs.get("default_llm_model", "gpt-4o-mini")
        agent_overrides = configs.get("agent_model_overrides", {})

        # 根据提供商选择 key
        if default_provider == "anthropic":
            api_key = configs.get("anthropic_api_key", "")
            base_url = "https://api.anthropic.com"
        elif default_provider == "ollama":
            api_key = "ollama"
            base_url = "http://localhost:11434/v1"
        else:
            api_key = configs.get("openai_api_key", "")
            base_url = configs.get("openai_base_url", "https://api.openai.com/v1")

        return {
            "provider": default_provider,
            "model": default_model,
            "api_key": api_key,
            "base_url": base_url,
            "agent_overrides": agent_overrides if isinstance(agent_overrides, dict) else {},
        }

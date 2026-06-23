import logging

from pydantic_settings import BaseSettings

logger = logging.getLogger(__name__)


class Settings(BaseSettings):
    # 应用配置
    APP_NAME: str = "凌镜 LingMirror - 跨境电商 AgentOS"
    APP_VERSION: str = "2.0.0"
    APP_DESCRIPTION: str = (
        "凌镜 LingMirror - 面向中小跨境电商团队的 AI Agent 协作运营平台"
    )
    DEBUG: bool = True
    APP_PORT: int = 8001

    # 数据库配置
    DATABASE_URL: str = (
        "postgresql+asyncpg://postgres:postgres@localhost:5432/product_management"
    )
    DATABASE_URL_SYNC: str = (
        "postgresql+psycopg2://postgres:postgres@localhost:5432/product_management"
    )

    # 文件上传
    UPLOAD_DIR: str = "./uploads"
    MAX_UPLOAD_SIZE: int = 100 * 1024 * 1024  # 100MB
    ALLOWED_EXTENSIONS: list[str] = ["jpg", "jpeg", "png", "gif", "webp"]

    # 静态文件URL前缀
    STATIC_URL: str = "/static"

    # 分页默认值
    DEFAULT_PAGE_SIZE: int = 20
    MAX_PAGE_SIZE: int = 100

    # CORS — 生产环境应设为具体前端地址
    # 环境变量 CORS_ORIGINS 支持 JSON 数组如 '["https://app.lingmirror.com"]'
    # 或逗号分隔字符串（由 __init__ 统一解析）
    CORS_ORIGINS: str = "*"

    # 权限控制
    AUTH_ENABLED: bool = True

    # 加密配置 — 生产环境必须通过环境变量或 .env 设置
    ENCRYPTION_KEY: str = ""

    @property
    def is_production(self) -> bool:
        """生产环境判断：DEBUG=False 且不为测试上下文"""
        return not self.DEBUG

    @property
    def cors_origins_list(self) -> list[str]:
        """解析 CORS_ORIGINS 为列表，支持 * / JSON数组 / 逗号分隔"""
        val = self.CORS_ORIGINS.strip()
        if val == "*":
            return ["*"]
        if val.startswith("["):
            import json

            return json.loads(val)
        return [o.strip() for o in val.split(",") if o.strip()]

    # ===== AI-5: LLM配置（环境变量: LLM_API_URL / LLM_API_KEY / LLM_MODEL）=====
    LLM_API_URL: str = "https://api.openai.com/v1/chat/completions"
    LLM_API_KEY: str = ""
    LLM_MODEL: str = "gpt-4o-mini"

    # ===== AI 生图配置（环境变量: IMAGE_GEN_* / REPLICATE_API_KEY / OPENAI_API_KEY）=====
    IMAGE_GEN_PROVIDER: str = "replicate"  # replicate / openai
    IMAGE_GEN_MODEL: str = "black-forest-labs/flux-2-pro"
    REPLICATE_API_KEY: str = ""
    OPENAI_API_KEY: str = ""
    REMOVE_BG_API_KEY: str = ""

    class Config:
        env_file = ".env"
        env_file_encoding = "utf-8"

    def __init__(self, **kwargs):
        super().__init__(**kwargs)
        if self.is_production:
            if not self.ENCRYPTION_KEY:
                raise RuntimeError(
                    "ENCRYPTION_KEY 必须设置！生产环境不能使用空密钥。\n"
                    '通过 .env 或环境变量设置，生成: python3 -c "import secrets; print(secrets.token_hex(16))"'
                )
            if not self.AUTH_ENABLED:
                raise RuntimeError("生产环境禁止关闭鉴权: 请设置 AUTH_ENABLED=True")
            if self.cors_origins_list == ["*"]:
                raise RuntimeError(
                    "生产环境必须设置 CORS_ORIGINS，当前为通配符。通过环境变量设置：\n"
                    "  CORS_ORIGINS='[\"https://app.lingmirror.com\"]'"
                )
        elif not self.ENCRYPTION_KEY:
            logger.warning(
                "⚠️ ENCRYPTION_KEY 为空！平台 API 密钥等敏感数据将被明文存储。"
                "请通过环境变量或 .env 文件设置 ENCRYPTION_KEY。"
            )


settings = Settings()

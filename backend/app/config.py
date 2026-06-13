from pydantic_settings import BaseSettings
from typing import Optional


class Settings(BaseSettings):
    # 应用配置
    APP_NAME: str = "MultiSell - AI跨境电商商品中台"
    APP_VERSION: str = "2.0.0"
    APP_DESCRIPTION: str = "MultiSell - AI原生跨境电商商品中台"
    DEBUG: bool = True
    APP_PORT: int = 8001

    # 数据库配置
    DATABASE_URL: str = "postgresql+asyncpg://lc@localhost:5432/product_management"
    DATABASE_URL_SYNC: str = "postgresql+psycopg2://lc@localhost:5432/product_management"

    # 文件上传
    UPLOAD_DIR: str = "./uploads"
    MAX_UPLOAD_SIZE: int = 10 * 1024 * 1024  # 10MB
    ALLOWED_EXTENSIONS: list[str] = ["jpg", "jpeg", "png", "gif", "webp"]

    # 静态文件URL前缀
    STATIC_URL: str = "/static"

    # 分页默认值
    DEFAULT_PAGE_SIZE: int = 20
    MAX_PAGE_SIZE: int = 100

    # 权限控制
    AUTH_ENABLED: bool = False

    # 加密配置
    ENCRYPTION_KEY: str = "default-key-change-in-production-32bytes!!"

    # ===== AI-5: LLM配置（环境变量: LLM_API_URL / LLM_API_KEY / LLM_MODEL）=====
    LLM_API_URL: str = "https://api.openai.com/v1/chat/completions"
    LLM_API_KEY: str = ""
    LLM_MODEL: str = "gpt-4o-mini"

    class Config:
        env_file = ".env"
        env_file_encoding = "utf-8"


settings = Settings()

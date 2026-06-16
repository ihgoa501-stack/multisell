from pydantic_settings import BaseSettings
class Settings(BaseSettings):
    # 应用配置
    APP_NAME: str = "凌镜 LingMirror - 跨境电商 AgentOS"
    APP_VERSION: str = "2.0.0"
    APP_DESCRIPTION: str = "凌镜 LingMirror - 面向中小跨境电商团队的 AI Agent 协作运营平台"
    DEBUG: bool = True
    APP_PORT: int = 8001

    # 数据库配置
    DATABASE_URL: str = "postgresql+asyncpg://postgres:postgres@localhost:5432/product_management"
    DATABASE_URL_SYNC: str = "postgresql+psycopg2://postgres:postgres@localhost:5432/product_management"

    # 文件上传
    UPLOAD_DIR: str = "./uploads"
    MAX_UPLOAD_SIZE: int = 100 * 1024 * 1024  # 100MB
    ALLOWED_EXTENSIONS: list[str] = ["jpg", "jpeg", "png", "gif", "webp"]

    # 静态文件URL前缀
    STATIC_URL: str = "/static"

    # 分页默认值
    DEFAULT_PAGE_SIZE: int = 20
    MAX_PAGE_SIZE: int = 100

    # 权限控制
    AUTH_ENABLED: bool = True

    # 加密配置
    ENCRYPTION_KEY: str = "default-key-change-in-production-32bytes!!"

    @property
    def is_production(self) -> bool:
        """生产环境判断：DEBUG=False 且不为测试上下文"""
        return not self.DEBUG

    # ===== AI-5: LLM配置（环境变量: LLM_API_URL / LLM_API_KEY / LLM_MODEL）=====
    LLM_API_URL: str = "https://api.openai.com/v1/chat/completions"
    LLM_API_KEY: str = ""
    LLM_MODEL: str = "gpt-4o-mini"

    class Config:
        env_file = ".env"
        env_file_encoding = "utf-8"


settings = Settings()

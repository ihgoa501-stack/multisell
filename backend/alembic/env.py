"""Alembic 迁移环境配置 — 异步模式"""

import asyncio
from logging.config import fileConfig

from alembic import context
from sqlalchemy import pool
from sqlalchemy.ext.asyncio import create_async_engine

from app.config import settings
from app.database import Base

# 加载所有模型以确保 Base.metadata 完整
import app.models  # noqa: F401

# Alembic Config 对象
config = context.config

# 设置日志
if config.config_file_name is not None:
    fileConfig(config.config_file_name)

# 元数据目标
target_metadata = Base.metadata


def run_migrations_offline() -> None:
    """离线迁移模式：不连接数据库，生成 SQL 脚本"""
    url = settings.DATABASE_URL
    context.configure(
        url=url,
        target_metadata=target_metadata,
        literal_binds=True,
        dialect_opts={"paramstyle": "named"},
    )

    with context.begin_transaction():
        context.run_migrations()


def do_run_migrations(connection):
    """在连接上执行迁移"""
    context.configure(connection=connection, target_metadata=target_metadata)
    with context.begin_transaction():
        context.run_migrations()


async def run_async_migrations() -> None:
    """异步在线迁移：连接数据库执行迁移"""
    connectable = create_async_engine(
        settings.DATABASE_URL,
        poolclass=pool.NullPool,
    )

    async with connectable.connect() as connection:
        await connection.run_sync(do_run_migrations)

    await connectable.dispose()


def run_migrations_online() -> None:
    """在线迁移入口"""
    asyncio.run(run_async_migrations())


if context.is_offline_mode():
    run_migrations_offline()
else:
    run_migrations_online()

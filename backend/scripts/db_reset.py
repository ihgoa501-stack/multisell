#!/usr/bin/env python3
"""
数据库重置工具 — 删除所有数据表并重建，然后运行 seed.py 填充演示数据。

用法:
    cd backend
    python scripts/db_reset.py

注意:
    此操作会删除所有数据，请谨慎使用！
    可通过 --force 跳过确认提示。
"""

import asyncio
import sys
import os

sys.path.insert(0, os.path.dirname(os.path.dirname(os.path.abspath(__file__))))

from sqlalchemy import text
from sqlalchemy.ext.asyncio import create_async_engine, AsyncSession, async_sessionmaker

from app.config import settings
from app.database import Base


async def drop_all_tables(engine):
    """删除所有数据表。"""
    print("🗑️  正在删除所有数据表...")
    async with engine.begin() as conn:
        await conn.run_sync(Base.metadata.drop_all)
    print("✔ 所有数据表已删除\n")


async def create_all_tables(engine):
    """创建所有数据表。"""
    print("🏗️  正在创建数据表...")
    async with engine.begin() as conn:
        await conn.run_sync(Base.metadata.create_all)
    print("✔ 数据表已创建\n")


async def reset():
    engine = create_async_engine(settings.DATABASE_URL, echo=False)

    print(f"\n{'='*50}")
    print("  MultiSell 数据库重置工具")
    print(f"{'='*50}\n")
    print(f"📦 数据库: {settings.DATABASE_URL}")
    print()

    await drop_all_tables(engine)
    await create_all_tables(engine)

    await engine.dispose()

    # 运行 seed 脚本
    print("🌱 运行 seed.py 填充演示数据...\n")
    seed_path = os.path.join(os.path.dirname(os.path.dirname(os.path.abspath(__file__))), "seed.py")
    os.execv(sys.executable, [sys.executable, seed_path])


async def reset_with_confirm():
    print(f"\n⚠️  警告: 此操作将删除 '{settings.DATABASE_URL}' 中的所有数据表！")
    print("   请确保已备份重要数据。\n")

    if "--force" in sys.argv:
        await reset()
        return

    try:
        response = input("确认删除所有数据并重建？(y/N): ").strip().lower()
        if response not in ("y", "yes"):
            print("操作已取消。")
            return
    except (EOFError, KeyboardInterrupt):
        print("\n操作已取消。")
        return

    await reset()


def main():
    asyncio.run(reset_with_confirm())


if __name__ == "__main__":
    main()

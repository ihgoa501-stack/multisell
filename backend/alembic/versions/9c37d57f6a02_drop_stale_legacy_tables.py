"""drop_stale_legacy_tables

清理不再使用的遗留旧表（共 13 张）。
这些表仅存在于数据库中，代码中已无对应模型。

安全删除顺序（子表→父表）:
  1. agent_action          → 2. exception_item
  3. shipping_bill_item    → 4. shipping_bill_batch
  5. cost_allocation_item  → 6. cost_allocation_batch
  7. platform_settlement_item → 8. platform_settlement_batch
  9. platform_category_mapping
 10. platform_attribute_mapping
 11. listing_task
 12. finance_ledger_entry
 13. platform_integration_account

Revision ID: 9c37d57f6a02
Revises: 579168a68b16
Create Date: 2026-06-17 15:41:31.123456
"""
from typing import Sequence, Union

from alembic import op

# revision identifiers, used by Alembic.
revision: str = '9c37d57f6a02'
down_revision: Union[str, None] = '579168a68b16'
branch_labels: Union[str, Sequence[str], None] = None
depends_on: Union[str, Sequence[str], None] = None

# ── 13 张遗表（按依赖顺序排序：子表先于父表） ──
_STALE_TABLES = [
    # 子表 → 父表
    "agent_action",               # FK → exception_item
    "exception_item",
    "shipping_bill_item",         # FK → shipping_bill_batch
    "shipping_bill_batch",
    "cost_allocation_item",       # FK → cost_allocation_batch
    "cost_allocation_batch",
    "platform_settlement_item",   # FK → platform_settlement_batch
    "platform_settlement_batch",
    # 无依赖
    "platform_category_mapping",
    "platform_attribute_mapping",
    "listing_task",
    "finance_ledger_entry",
    "platform_integration_account",
]


def upgrade() -> None:
    for table_name in _STALE_TABLES:
        op.drop_table(table_name)


def downgrade() -> None:
    """降级时以相反顺序（父→子）重建空表。

    注意：仅重建表结构，不恢复已删除的数据。
    如需完整回滚，请从备份恢复。
    """
    from sqlalchemy import (
        Column, BigInteger, String, Integer, Numeric,
        DateTime, Text, JSON, SmallInteger, ForeignKey,
        func, sa
    )

    # 13. platform_integration_account（无外键依赖）
    op.create_table('platform_integration_account',
        Column('id', BigInteger, primary_key=True, autoincrement=True),
        Column('platform_id', BigInteger, ForeignKey('platform.id'), nullable=False, comment='平台ID'),
        Column('adapter_code', String(50), nullable=False, comment='adapter 代码'),
        Column('account_name', String(200), nullable=False, comment='账号名称'),
        Column('status', String(30), comment='draft/active/disabled'),
        Column('credential_metadata', JSON, nullable=False, comment='密钥元信息'),
        Column('created_by', String(100)),
        Column('updated_by', String(100)),
        Column('created_at', DateTime(timezone=True), server_default=func.now()),
        Column('updated_at', DateTime(timezone=True), server_default=func.now(), onupdate=func.now()),
    )

    # 12. finance_ledger_entry
    op.create_table('finance_ledger_entry',
        Column('id', BigInteger, primary_key=True, autoincrement=True),
        Column('order_id', BigInteger, ForeignKey('sales_order.id'), nullable=False),
        Column('entry_type', String(50), nullable=False),
        Column('amount', Numeric(14, 2), nullable=False),
        Column('currency', String(10)),
        Column('cost_layer', String(30), nullable=False),
        Column('source_type', String(50)),
        Column('source_id', BigInteger),
        Column('description', String(500)),
        Column('created_at', DateTime(timezone=True), server_default=func.now()),
    )

    # 11. listing_task
    op.create_table('listing_task',
        Column('id', BigInteger, primary_key=True, autoincrement=True),
        Column('product_id', BigInteger, ForeignKey('product.id'), nullable=False),
        Column('platform_id', BigInteger, ForeignKey('platform.id'), nullable=False),
        Column('task_type', String(50), nullable=False),
        Column('status', String(20)),
        Column('created_at', DateTime(timezone=True), server_default=func.now()),
    )

    # 10. platform_attribute_mapping
    op.create_table('platform_attribute_mapping',
        Column('id', BigInteger, primary_key=True, autoincrement=True),
        Column('platform_id', BigInteger, ForeignKey('platform.id'), nullable=False),
        Column('created_at', DateTime(timezone=True), server_default=func.now()),
    )

    # 9. platform_category_mapping
    op.create_table('platform_category_mapping',
        Column('id', BigInteger, primary_key=True, autoincrement=True),
        Column('platform_id', BigInteger, ForeignKey('platform.id'), nullable=False),
        Column('created_at', DateTime(timezone=True), server_default=func.now()),
    )

    # 8. platform_settlement_batch
    op.create_table('platform_settlement_batch',
        Column('id', BigInteger, primary_key=True, autoincrement=True),
        Column('platform_id', BigInteger, ForeignKey('platform.id'), nullable=False),
        Column('created_at', DateTime(timezone=True), server_default=func.now()),
    )

    # 7. platform_settlement_item (FK → platform_settlement_batch)
    op.create_table('platform_settlement_item',
        Column('id', BigInteger, primary_key=True, autoincrement=True),
        Column('batch_id', BigInteger, ForeignKey('platform_settlement_batch.id'), nullable=False),
        Column('created_at', DateTime(timezone=True), server_default=func.now()),
    )

    # 6. cost_allocation_batch
    op.create_table('cost_allocation_batch',
        Column('id', BigInteger, primary_key=True, autoincrement=True),
        Column('created_at', DateTime(timezone=True), server_default=func.now()),
    )

    # 5. cost_allocation_item (FK → cost_allocation_batch)
    op.create_table('cost_allocation_item',
        Column('id', BigInteger, primary_key=True, autoincrement=True),
        Column('batch_id', BigInteger, ForeignKey('cost_allocation_batch.id'), nullable=False),
        Column('created_at', DateTime(timezone=True), server_default=func.now()),
    )

    # 4. shipping_bill_batch
    op.create_table('shipping_bill_batch',
        Column('id', BigInteger, primary_key=True, autoincrement=True),
        Column('created_at', DateTime(timezone=True), server_default=func.now()),
    )

    # 3. shipping_bill_item (FK → shipping_bill_batch)
    op.create_table('shipping_bill_item',
        Column('id', BigInteger, primary_key=True, autoincrement=True),
        Column('batch_id', BigInteger, ForeignKey('shipping_bill_batch.id'), nullable=False),
        Column('created_at', DateTime(timezone=True), server_default=func.now()),
    )

    # 2. exception_item
    op.create_table('exception_item',
        Column('id', BigInteger, primary_key=True, autoincrement=True),
        Column('created_at', DateTime(timezone=True), server_default=func.now()),
    )

    # 1. agent_action (FK → exception_item)
    op.create_table('agent_action',
        Column('id', BigInteger, primary_key=True, autoincrement=True),
        Column('exception_id', BigInteger, ForeignKey('exception_item.id')),
        Column('action_type', String(100), nullable=False),
        Column('created_at', DateTime(timezone=True), server_default=func.now()),
    )

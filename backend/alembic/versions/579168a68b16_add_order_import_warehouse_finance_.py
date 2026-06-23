"""add_order_import_warehouse_finance_tables

创建订单导入、库存分配、财务管理三个模块的数据表。

Revision ID: 579168a68b16
Revises: 97f0759177e9
Create Date: 2026-06-17 15:34:52.123456
"""

from typing import Sequence, Union

from alembic import op
import sqlalchemy as sa

# revision identifiers, used by Alembic.
revision: str = "579168a68b16"
down_revision: Union[str, None] = "97f0759177e9"
branch_labels: Union[str, Sequence[str], None] = None
depends_on: Union[str, Sequence[str], None] = None


def upgrade() -> None:
    # ── order_import 订单导入记录 ──────────────────────────────
    op.create_table(
        "order_import",
        sa.Column("id", sa.BigInteger(), autoincrement=True, nullable=False),
        sa.Column("platform_id", sa.BigInteger(), nullable=True, comment="平台ID"),
        sa.Column(
            "source_type",
            sa.String(length=50),
            nullable=False,
            comment="来源: ozon/shopee/wb/manual",
        ),
        sa.Column("file_name", sa.String(length=255), nullable=True, comment="文件名"),
        sa.Column("total_rows", sa.Integer(), nullable=True, comment="总行数"),
        sa.Column("success_count", sa.Integer(), nullable=True, comment="成功数"),
        sa.Column("error_count", sa.Integer(), nullable=True, comment="失败数"),
        sa.Column("error_detail", sa.JSON(), nullable=True, comment="错误详情"),
        sa.Column(
            "status",
            sa.String(length=20),
            nullable=True,
            comment="状态: pending/processing/completed/failed",
        ),
        sa.Column("created_by", sa.String(length=100), nullable=True, comment="导入人"),
        sa.Column(
            "created_at",
            sa.DateTime(timezone=True),
            server_default=sa.text("now()"),
            nullable=True,
            comment="创建时间",
        ),
        sa.Column(
            "updated_at",
            sa.DateTime(timezone=True),
            server_default=sa.text("now()"),
            nullable=True,
            comment="更新时间",
        ),
        sa.ForeignKeyConstraint(
            ["platform_id"],
            ["platform.id"],
        ),
        sa.PrimaryKeyConstraint("id"),
    )

    # ── warehouse 仓库 ──────────────────────────────────────────
    op.create_table(
        "warehouse",
        sa.Column("id", sa.BigInteger(), autoincrement=True, nullable=False),
        sa.Column("name", sa.String(length=200), nullable=False, comment="仓库名称"),
        sa.Column("code", sa.String(length=50), nullable=True, comment="仓库编码"),
        sa.Column("address", sa.String(length=500), nullable=True, comment="地址"),
        sa.Column("contact", sa.String(length=100), nullable=True, comment="联系人"),
        sa.Column("phone", sa.String(length=50), nullable=True, comment="联系电话"),
        sa.Column(
            "is_default", sa.SmallInteger(), nullable=True, comment="是否默认仓库"
        ),
        sa.Column(
            "status", sa.SmallInteger(), nullable=True, comment="状态: 0-禁用, 1-启用"
        ),
        sa.Column(
            "created_at",
            sa.DateTime(timezone=True),
            server_default=sa.text("now()"),
            nullable=True,
            comment="创建时间",
        ),
        sa.Column(
            "updated_at",
            sa.DateTime(timezone=True),
            server_default=sa.text("now()"),
            nullable=True,
            comment="更新时间",
        ),
        sa.PrimaryKeyConstraint("id"),
        sa.UniqueConstraint("code", name="uq_warehouse_code"),
    )

    # ── allocation_rule 分配规则 ──────────────────────────────
    op.create_table(
        "allocation_rule",
        sa.Column("id", sa.BigInteger(), autoincrement=True, nullable=False),
        sa.Column("name", sa.String(length=200), nullable=False, comment="规则名称"),
        sa.Column("priority", sa.Integer(), nullable=True, comment="优先级"),
        sa.Column(
            "rule_type",
            sa.String(length=50),
            nullable=False,
            comment="规则类型: percentage/fixed/priority",
        ),
        sa.Column("warehouse_id", sa.BigInteger(), nullable=False, comment="仓库ID"),
        sa.Column(
            "allocation_pct",
            sa.Numeric(precision=5, scale=2),
            nullable=True,
            comment="分配百分比",
        ),
        sa.Column(
            "allocation_qty", sa.Integer(), nullable=True, comment="固定分配数量"
        ),
        sa.Column(
            "status", sa.SmallInteger(), nullable=True, comment="状态: 0-禁用, 1-启用"
        ),
        sa.Column(
            "created_at",
            sa.DateTime(timezone=True),
            server_default=sa.text("now()"),
            nullable=True,
            comment="创建时间",
        ),
        sa.Column(
            "updated_at",
            sa.DateTime(timezone=True),
            server_default=sa.text("now()"),
            nullable=True,
            comment="更新时间",
        ),
        sa.ForeignKeyConstraint(
            ["warehouse_id"],
            ["warehouse.id"],
        ),
        sa.PrimaryKeyConstraint("id"),
    )

    # ── inventory_warehouse 仓库库存 ──────────────────────────
    op.create_table(
        "inventory_warehouse",
        sa.Column("id", sa.BigInteger(), autoincrement=True, nullable=False),
        sa.Column("sku_id", sa.BigInteger(), nullable=False, comment="SKU ID"),
        sa.Column("warehouse_id", sa.BigInteger(), nullable=False, comment="仓库ID"),
        sa.Column("quantity", sa.Integer(), nullable=True, comment="库存数量"),
        sa.Column("locked_quantity", sa.Integer(), nullable=True, comment="锁定库存"),
        sa.Column("safety_stock", sa.Integer(), nullable=True, comment="安全库存"),
        sa.Column(
            "created_at",
            sa.DateTime(timezone=True),
            server_default=sa.text("now()"),
            nullable=True,
            comment="创建时间",
        ),
        sa.Column(
            "updated_at",
            sa.DateTime(timezone=True),
            server_default=sa.text("now()"),
            nullable=True,
            comment="更新时间",
        ),
        sa.ForeignKeyConstraint(
            ["sku_id"],
            ["sku.id"],
        ),
        sa.ForeignKeyConstraint(
            ["warehouse_id"],
            ["warehouse.id"],
        ),
        sa.PrimaryKeyConstraint("id"),
        sa.UniqueConstraint("sku_id", "warehouse_id", name="uq_sku_warehouse"),
    )

    # ── finance_account 财务账户 ──────────────────────────────
    op.create_table(
        "finance_account",
        sa.Column("id", sa.BigInteger(), autoincrement=True, nullable=False),
        sa.Column("name", sa.String(length=200), nullable=False, comment="账户名称"),
        sa.Column(
            "account_type",
            sa.String(length=50),
            nullable=False,
            comment="账户类型: platform/payment/bank/cash",
        ),
        sa.Column("platform_id", sa.BigInteger(), nullable=True, comment="关联平台ID"),
        sa.Column("currency", sa.String(length=3), nullable=True, comment="币种"),
        sa.Column(
            "balance", sa.Numeric(precision=14, scale=2), nullable=True, comment="余额"
        ),
        sa.Column(
            "status", sa.SmallInteger(), nullable=True, comment="状态: 0-禁用, 1-启用"
        ),
        sa.Column(
            "created_at",
            sa.DateTime(timezone=True),
            server_default=sa.text("now()"),
            nullable=True,
            comment="创建时间",
        ),
        sa.Column(
            "updated_at",
            sa.DateTime(timezone=True),
            server_default=sa.text("now()"),
            nullable=True,
            comment="更新时间",
        ),
        sa.ForeignKeyConstraint(
            ["platform_id"],
            ["platform.id"],
        ),
        sa.PrimaryKeyConstraint("id"),
    )

    # ── finance_transaction 财务流水 ──────────────────────────
    op.create_table(
        "finance_transaction",
        sa.Column("id", sa.BigInteger(), autoincrement=True, nullable=False),
        sa.Column("account_id", sa.BigInteger(), nullable=False, comment="账户ID"),
        sa.Column(
            "transaction_type",
            sa.String(length=50),
            nullable=False,
            comment="类型: revenue/cost/fee/refund/transfer",
        ),
        sa.Column(
            "amount",
            sa.Numeric(precision=14, scale=2),
            nullable=False,
            comment="金额（正收入/负支出）",
        ),
        sa.Column("currency", sa.String(length=3), nullable=True, comment="币种"),
        sa.Column("order_id", sa.BigInteger(), nullable=True, comment="关联订单ID"),
        sa.Column(
            "settlement_id", sa.BigInteger(), nullable=True, comment="关联结算单ID"
        ),
        sa.Column("platform_id", sa.BigInteger(), nullable=True, comment="关联平台ID"),
        sa.Column("description", sa.String(length=500), nullable=True, comment="描述"),
        sa.Column(
            "transaction_date",
            sa.DateTime(timezone=True),
            nullable=True,
            comment="交易日期",
        ),
        sa.Column(
            "created_at",
            sa.DateTime(timezone=True),
            server_default=sa.text("now()"),
            nullable=True,
            comment="创建时间",
        ),
        sa.ForeignKeyConstraint(
            ["account_id"],
            ["finance_account.id"],
        ),
        sa.ForeignKeyConstraint(
            ["order_id"],
            ["sales_order.id"],
        ),
        sa.ForeignKeyConstraint(
            ["platform_id"],
            ["platform.id"],
        ),
        sa.ForeignKeyConstraint(
            ["settlement_id"],
            ["settlement.id"],
        ),
        sa.PrimaryKeyConstraint("id"),
    )


def downgrade() -> None:
    op.drop_table("finance_transaction")
    op.drop_table("finance_account")
    op.drop_table("inventory_warehouse")
    op.drop_table("allocation_rule")
    op.drop_table("warehouse")
    op.drop_table("order_import")

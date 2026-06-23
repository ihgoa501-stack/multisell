"""add_settlement_tables

创建结算管理模块所需的数据表：settlement 和 settlement_item。

Revision ID: 97f0759177e9
Revises: f3690eddbe99
Create Date: 2026-06-17 15:33:59.567046
"""

from typing import Sequence, Union

from alembic import op
import sqlalchemy as sa

# revision identifiers, used by Alembic.
revision: str = "97f0759177e9"
down_revision: Union[str, None] = "f3690eddbe99"
branch_labels: Union[str, Sequence[str], None] = None
depends_on: Union[str, Sequence[str], None] = None


def upgrade() -> None:
    # ── settlement 结算单 ──────────────────────────────────────────
    op.create_table(
        "settlement",
        sa.Column("id", sa.BigInteger(), autoincrement=True, nullable=False),
        sa.Column("platform_id", sa.BigInteger(), nullable=False, comment="平台ID"),
        sa.Column(
            "settlement_no", sa.String(length=100), nullable=False, comment="结算单号"
        ),
        sa.Column(
            "period_start",
            sa.DateTime(timezone=True),
            nullable=True,
            comment="结算周期开始",
        ),
        sa.Column(
            "period_end",
            sa.DateTime(timezone=True),
            nullable=True,
            comment="结算周期结束",
        ),
        sa.Column("currency", sa.String(length=3), nullable=True, comment="币种"),
        sa.Column(
            "total_revenue",
            sa.Numeric(precision=12, scale=2),
            nullable=True,
            comment="总收入",
        ),
        sa.Column(
            "total_fee",
            sa.Numeric(precision=12, scale=2),
            nullable=True,
            comment="总费用",
        ),
        sa.Column(
            "total_refund",
            sa.Numeric(precision=12, scale=2),
            nullable=True,
            comment="总退款",
        ),
        sa.Column(
            "total_net",
            sa.Numeric(precision=12, scale=2),
            nullable=True,
            comment="净收入",
        ),
        sa.Column(
            "status",
            sa.String(length=20),
            nullable=True,
            comment="状态: pending/reconciling/reconciled/closed",
        ),
        sa.Column("raw_data", sa.JSON(), nullable=True, comment="原始数据(JSON)"),
        sa.Column(
            "imported_at",
            sa.DateTime(timezone=True),
            server_default=sa.text("now()"),
            nullable=True,
            comment="导入时间",
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

    # ── settlement_item 结算明细 ────────────────────────────────────
    op.create_table(
        "settlement_item",
        sa.Column("id", sa.BigInteger(), autoincrement=True, nullable=False),
        sa.Column("settlement_id", sa.BigInteger(), nullable=False, comment="结算单ID"),
        sa.Column(
            "transaction_type",
            sa.String(length=30),
            nullable=False,
            comment="交易类型: order_sale/refund/shipping_fee/platform_fee/payment_fee/other",
        ),
        sa.Column(
            "transaction_id", sa.String(length=100), nullable=True, comment="平台交易ID"
        ),
        sa.Column(
            "order_no", sa.String(length=100), nullable=True, comment="关联订单号"
        ),
        sa.Column("order_id", sa.BigInteger(), nullable=True, comment="内部订单ID"),
        sa.Column("sku_id", sa.BigInteger(), nullable=True, comment="SKU ID"),
        sa.Column(
            "amount", sa.Numeric(precision=12, scale=2), nullable=True, comment="金额"
        ),
        sa.Column(
            "fee", sa.Numeric(precision=12, scale=2), nullable=True, comment="费用"
        ),
        sa.Column(
            "net", sa.Numeric(precision=12, scale=2), nullable=True, comment="净额"
        ),
        sa.Column("quantity", sa.Integer(), nullable=True, comment="数量"),
        sa.Column(
            "occurred_at",
            sa.DateTime(timezone=True),
            nullable=True,
            comment="交易发生时间",
        ),
        sa.Column(
            "created_at",
            sa.DateTime(timezone=True),
            server_default=sa.text("now()"),
            nullable=True,
            comment="创建时间",
        ),
        sa.Column(
            "reconciliation_status",
            sa.String(length=20),
            nullable=True,
            comment="对账状态: pending/matched/unmatched/discrepancy",
        ),
        sa.Column("reconciliation_note", sa.Text(), nullable=True, comment="对账备注"),
        sa.Column(
            "reconciled_at",
            sa.DateTime(timezone=True),
            nullable=True,
            comment="对账时间",
        ),
        sa.Column(
            "reconciled_by", sa.String(length=100), nullable=True, comment="对账人"
        ),
        sa.ForeignKeyConstraint(
            ["order_id"],
            ["sales_order.id"],
        ),
        sa.ForeignKeyConstraint(
            ["settlement_id"],
            ["settlement.id"],
        ),
        sa.ForeignKeyConstraint(
            ["sku_id"],
            ["sku.id"],
        ),
        sa.PrimaryKeyConstraint("id"),
    )
    op.create_index(
        op.f("ix_settlement_item_settlement_id"),
        "settlement_item",
        ["settlement_id"],
        unique=False,
    )


def downgrade() -> None:
    op.drop_index(
        op.f("ix_settlement_item_settlement_id"), table_name="settlement_item"
    )
    op.drop_table("settlement_item")
    op.drop_table("settlement")

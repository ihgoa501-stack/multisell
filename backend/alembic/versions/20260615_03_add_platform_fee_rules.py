"""add platform fee rules

Revision ID: 20260615_03
Revises: 20260615_02
Create Date: 2026-06-15
"""

from alembic import op
import sqlalchemy as sa


revision = "20260615_03"
down_revision = "20260615_02"
branch_labels = None
depends_on = None


def upgrade() -> None:
    op.create_table(
        "platform_fee_rule",
        sa.Column("id", sa.BigInteger(), primary_key=True, autoincrement=True),
        sa.Column(
            "platform_id", sa.BigInteger(), sa.ForeignKey("platform.id"), nullable=False
        ),
        sa.Column("site_code", sa.String(length=10), nullable=True),
        sa.Column(
            "category_id", sa.BigInteger(), sa.ForeignKey("category.id"), nullable=True
        ),
        sa.Column(
            "commission_pct", sa.Numeric(8, 4), nullable=False, server_default="0"
        ),
        sa.Column(
            "payment_fee_pct", sa.Numeric(8, 4), nullable=False, server_default="0"
        ),
        sa.Column("fixed_fee", sa.Numeric(10, 2), nullable=False, server_default="0"),
        sa.Column(
            "advertising_pct", sa.Numeric(8, 4), nullable=False, server_default="0"
        ),
        sa.Column(
            "other_reserve_fee", sa.Numeric(10, 2), nullable=False, server_default="0"
        ),
        sa.Column("priority", sa.Integer(), nullable=False, server_default="0"),
        sa.Column("status", sa.SmallInteger(), nullable=False, server_default="1"),
        sa.Column("remark", sa.Text(), nullable=True),
        sa.Column(
            "created_at", sa.DateTime(timezone=True), server_default=sa.func.now()
        ),
        sa.Column(
            "updated_at", sa.DateTime(timezone=True), server_default=sa.func.now()
        ),
    )
    op.create_index(
        "ix_platform_fee_rule_platform", "platform_fee_rule", ["platform_id"]
    )
    op.create_index(
        "ix_platform_fee_rule_match",
        "platform_fee_rule",
        ["platform_id", "site_code", "category_id", "status"],
    )


def downgrade() -> None:
    op.drop_index("ix_platform_fee_rule_match", table_name="platform_fee_rule")
    op.drop_index("ix_platform_fee_rule_platform", table_name="platform_fee_rule")
    op.drop_table("platform_fee_rule")

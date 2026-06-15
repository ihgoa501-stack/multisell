"""add listing tasks

Revision ID: 20260615_04
Revises: 20260615_03
Create Date: 2026-06-15
"""

from alembic import op
import sqlalchemy as sa
from sqlalchemy.dialects.postgresql import JSON


revision = "20260615_04"
down_revision = "20260615_03"
branch_labels = None
depends_on = None


def upgrade() -> None:
    op.create_table(
        "listing_task",
        sa.Column("id", sa.BigInteger(), primary_key=True, autoincrement=True),
        sa.Column("product_id", sa.BigInteger(), sa.ForeignKey("product.id"), nullable=False),
        sa.Column("platform_id", sa.BigInteger(), sa.ForeignKey("platform.id"), nullable=False),
        sa.Column("sku_id", sa.BigInteger(), sa.ForeignKey("sku.id"), nullable=True),
        sa.Column("product_listing_id", sa.BigInteger(), sa.ForeignKey("product_listing.id"), nullable=True),
        sa.Column("source_type", sa.String(length=50), nullable=False, server_default="decision"),
        sa.Column("source_item_key", sa.String(length=100), nullable=True),
        sa.Column("status", sa.String(length=50), nullable=False, server_default="blocked"),
        sa.Column("missing_requirements", JSON(), nullable=False, server_default="[]"),
        sa.Column("decision_snapshot", JSON(), nullable=True),
        sa.Column("target_sale_price", sa.Numeric(12, 2), nullable=True),
        sa.Column("target_profit_margin", sa.Numeric(8, 2), nullable=True),
        sa.Column("destination_country", sa.String(length=10), nullable=True),
        sa.Column("last_error", sa.Text(), nullable=True),
        sa.Column("created_by", sa.String(length=100), nullable=True),
        sa.Column("updated_by", sa.String(length=100), nullable=True),
        sa.Column("created_at", sa.DateTime(timezone=True), server_default=sa.func.now()),
        sa.Column("updated_at", sa.DateTime(timezone=True), server_default=sa.func.now()),
    )
    op.create_index("ix_listing_task_status", "listing_task", ["status"])
    op.create_index("ix_listing_task_product_platform", "listing_task", ["product_id", "platform_id"])


def downgrade() -> None:
    op.drop_index("ix_listing_task_product_platform", table_name="listing_task")
    op.drop_index("ix_listing_task_status", table_name="listing_task")
    op.drop_table("listing_task")

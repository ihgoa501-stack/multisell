"""add cost allocation tables

Revision ID: 20260615_10
Revises: 20260615_09
Create Date: 2026-06-15
"""

from alembic import op
import sqlalchemy as sa
from sqlalchemy.dialects.postgresql import JSON


revision = "20260615_10"
down_revision = "20260615_09"
branch_labels = None
depends_on = None


def upgrade() -> None:
    op.create_table(
        "cost_allocation_batch",
        sa.Column("id", sa.BigInteger(), primary_key=True, autoincrement=True),
        sa.Column("allocation_type", sa.String(50), nullable=False),
        sa.Column("allocation_method", sa.String(30), nullable=False),
        sa.Column("total_amount", sa.Numeric(14, 2), nullable=False),
        sa.Column("currency", sa.String(10), server_default="CNY"),
        sa.Column("source_filename", sa.String(500)),
        sa.Column("row_count", sa.Integer(), server_default="0"),
        sa.Column("status", sa.String(30), server_default="imported"),
        sa.Column("posted_count", sa.Integer(), server_default="0"),
        sa.Column("created_by", sa.String(100)),
        sa.Column("created_at", sa.DateTime(timezone=True), server_default=sa.func.now()),
        sa.Column("updated_at", sa.DateTime(timezone=True), server_default=sa.func.now()),
    )
    op.create_index("ix_alloc_batch_status", "cost_allocation_batch", ["status"])

    op.create_table(
        "cost_allocation_item",
        sa.Column("id", sa.BigInteger(), primary_key=True, autoincrement=True),
        sa.Column("batch_id", sa.BigInteger(), sa.ForeignKey("cost_allocation_batch.id"), nullable=False),
        sa.Column("row_number", sa.Integer(), nullable=False),
        sa.Column("sku_id", sa.BigInteger(), sa.ForeignKey("sku.id")),
        sa.Column("sku_code", sa.String(100)),
        sa.Column("order_id", sa.BigInteger(), sa.ForeignKey("sales_order.id")),
        sa.Column("quantity", sa.Integer(), server_default="0"),
        sa.Column("weight_kg", sa.Numeric(10, 3)),
        sa.Column("volume_m3", sa.Numeric(10, 4)),
        sa.Column("item_value", sa.Numeric(14, 2)),
        sa.Column("allocation_factor", sa.Numeric(14, 4)),
        sa.Column("allocated_amount", sa.Numeric(14, 2), server_default="0"),
        sa.Column("cost_layer", sa.String(30), server_default="allocated"),
        sa.Column("posted_to_ledger", sa.Integer(), server_default="0"),
        sa.Column("raw_payload", JSON()),
        sa.Column("created_at", sa.DateTime(timezone=True), server_default=sa.func.now()),
    )
    op.create_index("ix_alloc_item_batch_id", "cost_allocation_item", ["batch_id"])


def downgrade() -> None:
    op.drop_index("ix_alloc_item_batch_id", table_name="cost_allocation_item")
    op.drop_table("cost_allocation_item")
    op.drop_index("ix_alloc_batch_status", table_name="cost_allocation_batch")
    op.drop_table("cost_allocation_batch")

"""add platform settlement tables

Revision ID: 20260615_06
Revises: 20260615_05
Create Date: 2026-06-15
"""

from alembic import op
import sqlalchemy as sa
from sqlalchemy.dialects.postgresql import JSON


revision = "20260615_06"
down_revision = "20260615_05"
branch_labels = None
depends_on = None


def upgrade() -> None:
    op.create_table(
        "platform_settlement_batch",
        sa.Column("id", sa.BigInteger(), primary_key=True, autoincrement=True),
        sa.Column("platform_name", sa.String(100)),
        sa.Column("filename", sa.String(500), nullable=False),
        sa.Column("row_count", sa.Integer(), server_default="0"),
        sa.Column("matched_count", sa.Integer(), server_default="0"),
        sa.Column("unmatched_count", sa.Integer(), server_default="0"),
        sa.Column("import_status", sa.String(30), server_default="imported"),
        sa.Column("status", sa.String(30), server_default="imported"),
        sa.Column("created_by", sa.String(100)),
        sa.Column("created_at", sa.DateTime(timezone=True), server_default=sa.func.now()),
        sa.Column("updated_at", sa.DateTime(timezone=True), server_default=sa.func.now()),
    )
    op.create_index("ix_settlement_batch_status", "platform_settlement_batch", ["status"])

    op.create_table(
        "platform_settlement_item",
        sa.Column("id", sa.BigInteger(), primary_key=True, autoincrement=True),
        sa.Column("batch_id", sa.BigInteger(), sa.ForeignKey("platform_settlement_batch.id"), nullable=False),
        sa.Column("row_number", sa.Integer(), nullable=False),
        sa.Column("platform", sa.String(100)),
        sa.Column("store_name", sa.String(200)),
        sa.Column("platform_order_no", sa.String(200)),
        sa.Column("order_no", sa.String(200)),
        sa.Column("transaction_type", sa.String(50), nullable=False),
        sa.Column("currency", sa.String(10), server_default="CNY"),
        sa.Column("amount", sa.Numeric(14, 2), server_default="0"),
        sa.Column("settled_at", sa.DateTime(timezone=True)),
        sa.Column("description", sa.Text()),
        sa.Column("match_status", sa.String(30), server_default="unmatched"),
        sa.Column("matched_order_id", sa.BigInteger(), sa.ForeignKey("sales_order.id")),
        sa.Column("raw_payload", JSON()),
        sa.Column("created_at", sa.DateTime(timezone=True), server_default=sa.func.now()),
    )
    op.create_index("ix_settlement_item_batch_id", "platform_settlement_item", ["batch_id"])
    op.create_index("ix_settlement_item_match_status", "platform_settlement_item", ["match_status"])


def downgrade() -> None:
    op.drop_index("ix_settlement_item_match_status", table_name="platform_settlement_item")
    op.drop_index("ix_settlement_item_batch_id", table_name="platform_settlement_item")
    op.drop_table("platform_settlement_item")
    op.drop_index("ix_settlement_batch_status", table_name="platform_settlement_batch")
    op.drop_table("platform_settlement_batch")

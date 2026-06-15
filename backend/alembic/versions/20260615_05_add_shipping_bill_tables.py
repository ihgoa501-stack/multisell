"""add shipping bill tables

Revision ID: 20260615_05
Revises: 20260615_04
Create Date: 2026-06-15
"""

from alembic import op
import sqlalchemy as sa
from sqlalchemy.dialects.postgresql import JSON


revision = "20260615_05"
down_revision = "20260615_04"
branch_labels = None
depends_on = None


def upgrade() -> None:
    op.create_table(
        "shipping_bill_batch",
        sa.Column("id", sa.BigInteger(), primary_key=True, autoincrement=True),
        sa.Column("provider_id", sa.BigInteger(), sa.ForeignKey("shipping_provider.id")),
        sa.Column("source_filename", sa.String(500), nullable=False),
        sa.Column("currency", sa.String(10), server_default="CNY"),
        sa.Column("row_count", sa.Integer(), server_default="0"),
        sa.Column("matched_count", sa.Integer(), server_default="0"),
        sa.Column("mismatch_count", sa.Integer(), server_default="0"),
        sa.Column("unmatched_count", sa.Integer(), server_default="0"),
        sa.Column("status", sa.String(30), server_default="imported"),
        sa.Column("created_by", sa.String(100)),
        sa.Column("created_at", sa.DateTime(timezone=True), server_default=sa.func.now()),
        sa.Column("updated_at", sa.DateTime(timezone=True), server_default=sa.func.now()),
    )
    op.create_index("ix_bill_batch_status", "shipping_bill_batch", ["status"])
    op.create_index("ix_bill_batch_created_at", "shipping_bill_batch", ["created_at"])

    op.create_table(
        "shipping_bill_item",
        sa.Column("id", sa.BigInteger(), primary_key=True, autoincrement=True),
        sa.Column("batch_id", sa.BigInteger(), sa.ForeignKey("shipping_bill_batch.id"), nullable=False),
        sa.Column("row_number", sa.Integer(), nullable=False),
        sa.Column("reconciliation_status", sa.String(30), server_default="unmatched_bill"),
        sa.Column("tracking_number", sa.String(200)),
        sa.Column("order_no", sa.String(200)),
        sa.Column("provider_name", sa.String(200)),
        sa.Column("channel_name", sa.String(200)),
        sa.Column("destination_country", sa.String(10)),
        sa.Column("billed_weight_kg", sa.Numeric(10, 3)),
        sa.Column("currency", sa.String(10), server_default="CNY"),
        sa.Column("actual_shipping_fee", sa.Numeric(12, 2)),
        sa.Column("surcharge_fee", sa.Numeric(12, 2)),
        sa.Column("total_actual_fee", sa.Numeric(12, 2)),
        sa.Column("billed_at", sa.DateTime(timezone=True)),
        sa.Column("matched_order_id", sa.BigInteger(), sa.ForeignKey("sales_order.id")),
        sa.Column("matched_snapshot_id", sa.BigInteger(), sa.ForeignKey("sales_order_shipping_snapshot.id")),
        sa.Column("snapshot_shipping_fee", sa.Numeric(12, 2)),
        sa.Column("variance_amount", sa.Numeric(12, 2)),
        sa.Column("raw_payload", JSON()),
        sa.Column("note", sa.Text()),
        sa.Column("resolved_by", sa.String(100)),
        sa.Column("resolved_at", sa.DateTime(timezone=True)),
        sa.Column("created_at", sa.DateTime(timezone=True), server_default=sa.func.now()),
    )
    op.create_index("ix_bill_item_batch_id", "shipping_bill_item", ["batch_id"])
    op.create_index("ix_bill_item_status", "shipping_bill_item", ["reconciliation_status"])
    op.create_index("ix_bill_item_tracking", "shipping_bill_item", ["tracking_number"])


def downgrade() -> None:
    op.drop_index("ix_bill_item_tracking", table_name="shipping_bill_item")
    op.drop_index("ix_bill_item_status", table_name="shipping_bill_item")
    op.drop_index("ix_bill_item_batch_id", table_name="shipping_bill_item")
    op.drop_table("shipping_bill_item")
    op.drop_index("ix_bill_batch_created_at", table_name="shipping_bill_batch")
    op.drop_index("ix_bill_batch_status", table_name="shipping_bill_batch")
    op.drop_table("shipping_bill_batch")

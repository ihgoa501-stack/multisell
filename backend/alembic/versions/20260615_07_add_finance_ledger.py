"""add finance ledger entry table

Revision ID: 20260615_07
Revises: 20260615_06
Create Date: 2026-06-15
"""

from alembic import op
import sqlalchemy as sa


revision = "20260615_07"
down_revision = "20260615_06"
branch_labels = None
depends_on = None


def upgrade() -> None:
    op.create_table(
        "finance_ledger_entry",
        sa.Column("id", sa.BigInteger(), primary_key=True, autoincrement=True),
        sa.Column("order_id", sa.BigInteger(), sa.ForeignKey("sales_order.id"), nullable=False),
        sa.Column("entry_type", sa.String(50), nullable=False),
        sa.Column("amount", sa.Numeric(14, 2), nullable=False),
        sa.Column("currency", sa.String(10), server_default="CNY"),
        sa.Column("cost_layer", sa.String(30), nullable=False),
        sa.Column("source_type", sa.String(50)),
        sa.Column("source_id", sa.BigInteger()),
        sa.Column("description", sa.String(500)),
        sa.Column("created_at", sa.DateTime(timezone=True), server_default=sa.func.now()),
    )
    op.create_index("ix_ledger_order_id", "finance_ledger_entry", ["order_id"])
    op.create_index("ix_ledger_entry_type", "finance_ledger_entry", ["entry_type"])


def downgrade() -> None:
    op.drop_index("ix_ledger_entry_type", table_name="finance_ledger_entry")
    op.drop_index("ix_ledger_order_id", table_name="finance_ledger_entry")
    op.drop_table("finance_ledger_entry")

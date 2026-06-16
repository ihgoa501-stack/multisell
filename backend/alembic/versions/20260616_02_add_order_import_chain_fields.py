"""add_order_import_chain_fields

Revision ID: 20260616_02_add_order_import_chain_fields
Revises: 20260616_01_add_order_import_tables
Create Date: 2026-06-16

"""
from alembic import op
import sqlalchemy as sa


revision = "20260616_02_add_order_import_chain_fields"
down_revision = "20260616_01_add_order_import_tables"
branch_labels = None
depends_on = None


def upgrade() -> None:
    op.add_column("order_import_batch", sa.Column("chain_status", sa.String(length=50), server_default="chain_pending", comment="chain_pending/chain_processed/chain_failed"))
    op.add_column("order_import_batch", sa.Column("ledger_rebuilt_count", sa.Integer(), server_default="0", comment="已重建账本订单数"))
    op.add_column("order_import_batch", sa.Column("exception_generated_count", sa.Integer(), server_default="0", comment="生成异常数"))
    op.add_column("order_import_batch", sa.Column("chain_failure_count", sa.Integer(), server_default="0", comment="链路处理失败数"))
    op.add_column("order_import_batch", sa.Column("processed_at", sa.DateTime(timezone=True), nullable=True, comment="链路处理时间"))
    op.add_column("order_import_item", sa.Column("chain_status", sa.String(length=50), server_default="chain_pending", comment="chain_pending/ledger_rebuilt/exception_generated/chain_failed"))
    op.add_column("order_import_item", sa.Column("chain_failure_reason", sa.String(length=500), nullable=True, comment="链路处理失败原因"))


def downgrade() -> None:
    op.drop_column("order_import_item", "chain_failure_reason")
    op.drop_column("order_import_item", "chain_status")
    op.drop_column("order_import_batch", "processed_at")
    op.drop_column("order_import_batch", "chain_failure_count")
    op.drop_column("order_import_batch", "exception_generated_count")
    op.drop_column("order_import_batch", "ledger_rebuilt_count")
    op.drop_column("order_import_batch", "chain_status")

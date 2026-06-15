"""add inventory locked quantity

Revision ID: 20260615_02
Revises: 20260615_01
Create Date: 2026-06-15
"""

from alembic import op
import sqlalchemy as sa


revision = "20260615_02"
down_revision = "20260615_01"
branch_labels = None
depends_on = None


def upgrade() -> None:
    op.add_column(
        "inventory",
        sa.Column("locked_quantity", sa.Integer(), server_default="0", nullable=False, comment="锁定库存"),
    )


def downgrade() -> None:
    op.drop_column("inventory", "locked_quantity")

"""add exception workbench table

Revision ID: 20260615_08
Revises: 20260615_07
Create Date: 2026-06-15
"""

from alembic import op
import sqlalchemy as sa


revision = "20260615_08"
down_revision = "20260615_07"
branch_labels = None
depends_on = None


def upgrade() -> None:
    op.create_table(
        "exception_item",
        sa.Column("id", sa.BigInteger(), primary_key=True, autoincrement=True),
        sa.Column("source_module", sa.String(50), nullable=False),
        sa.Column("source_type", sa.String(50)),
        sa.Column("source_id", sa.BigInteger()),
        sa.Column("severity", sa.String(20), server_default="medium"),
        sa.Column("status", sa.String(20), server_default="open"),
        sa.Column("title", sa.String(300), nullable=False),
        sa.Column("description", sa.Text()),
        sa.Column("recommended_action", sa.String(500)),
        sa.Column("assigned_to", sa.String(100)),
        sa.Column("resolved_at", sa.DateTime(timezone=True)),
        sa.Column("resolved_by", sa.String(100)),
        sa.Column("note", sa.Text()),
        sa.Column(
            "created_at", sa.DateTime(timezone=True), server_default=sa.func.now()
        ),
        sa.Column(
            "updated_at", sa.DateTime(timezone=True), server_default=sa.func.now()
        ),
    )
    op.create_index("ix_exception_source_module", "exception_item", ["source_module"])
    op.create_index("ix_exception_status", "exception_item", ["status"])
    op.create_index("ix_exception_severity", "exception_item", ["severity"])


def downgrade() -> None:
    op.drop_index("ix_exception_severity", table_name="exception_item")
    op.drop_index("ix_exception_status", table_name="exception_item")
    op.drop_index("ix_exception_source_module", table_name="exception_item")
    op.drop_table("exception_item")

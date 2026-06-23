"""add agent action table

Revision ID: 20260615_09
Revises: 20260615_08
Create Date: 2026-06-15
"""

from alembic import op
import sqlalchemy as sa
from sqlalchemy.dialects.postgresql import JSON


revision = "20260615_09"
down_revision = "20260615_08"
branch_labels = None
depends_on = None


def upgrade() -> None:
    op.create_table(
        "agent_action",
        sa.Column("id", sa.BigInteger(), primary_key=True, autoincrement=True),
        sa.Column("source_module", sa.String(50)),
        sa.Column("source_type", sa.String(50)),
        sa.Column("source_id", sa.BigInteger()),
        sa.Column("exception_id", sa.BigInteger(), sa.ForeignKey("exception_item.id")),
        sa.Column("action_type", sa.String(100), nullable=False),
        sa.Column("title", sa.String(300), nullable=False),
        sa.Column("description", sa.Text()),
        sa.Column("proposed_payload", JSON()),
        sa.Column("before_snapshot", JSON()),
        sa.Column("after_snapshot", JSON()),
        sa.Column("status", sa.String(30), server_default="proposed"),
        sa.Column("proposed_by", sa.String(100)),
        sa.Column("approved_by", sa.String(100)),
        sa.Column("approved_at", sa.DateTime(timezone=True)),
        sa.Column("rejected_by", sa.String(100)),
        sa.Column("rejected_at", sa.DateTime(timezone=True)),
        sa.Column("rejection_reason", sa.Text()),
        sa.Column("executed_by", sa.String(100)),
        sa.Column("executed_at", sa.DateTime(timezone=True)),
        sa.Column(
            "created_at", sa.DateTime(timezone=True), server_default=sa.func.now()
        ),
        sa.Column(
            "updated_at", sa.DateTime(timezone=True), server_default=sa.func.now()
        ),
    )
    op.create_index("ix_agent_action_status", "agent_action", ["status"])
    op.create_index("ix_agent_action_exception", "agent_action", ["exception_id"])


def downgrade() -> None:
    op.drop_index("ix_agent_action_exception", table_name="agent_action")
    op.drop_index("ix_agent_action_status", table_name="agent_action")
    op.drop_table("agent_action")

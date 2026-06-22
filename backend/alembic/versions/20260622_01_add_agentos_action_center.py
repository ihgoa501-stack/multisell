"""add agentos action center

Revision ID: 20260622_01
Revises: 72181db29a25
Create Date: 2026-06-22
"""
from typing import Sequence, Union

from alembic import op
import sqlalchemy as sa
from sqlalchemy.dialects import postgresql


revision: str = "20260622_01"
down_revision: Union[str, Sequence[str], None] = "72181db29a25"
branch_labels: Union[str, Sequence[str], None] = None
depends_on: Union[str, Sequence[str], None] = None


json_type = postgresql.JSONB(astext_type=sa.Text())


def upgrade() -> None:
    op.create_table(
        "agentos_action_proposal",
        sa.Column("id", sa.BigInteger(), autoincrement=True, nullable=False),
        sa.Column("source_type", sa.String(length=50), nullable=False),
        sa.Column("source_id", sa.String(length=100), nullable=True),
        sa.Column("agent_id", sa.String(length=20), nullable=True),
        sa.Column("squad_id", sa.String(length=50), nullable=True),
        sa.Column("action_type", sa.String(length=100), nullable=False),
        sa.Column("business_object_type", sa.String(length=50), nullable=True),
        sa.Column("business_object_id", sa.String(length=100), nullable=True),
        sa.Column("title", sa.String(length=300), nullable=False),
        sa.Column("description", sa.Text(), nullable=True),
        sa.Column("proposed_payload", json_type, nullable=False, server_default=sa.text("'{}'::jsonb")),
        sa.Column("before_snapshot", json_type, nullable=True),
        sa.Column("after_snapshot", json_type, nullable=True),
        sa.Column("risk_level", sa.String(length=20), nullable=False, server_default="medium"),
        sa.Column("requires_approval", sa.Boolean(), nullable=False, server_default=sa.text("true")),
        sa.Column("status", sa.String(length=30), nullable=False, server_default="suggested"),
        sa.Column("confidence", sa.Numeric(5, 4), nullable=True),
        sa.Column("proposed_by", sa.String(length=100), nullable=True),
        sa.Column("approved_by", sa.String(length=100), nullable=True),
        sa.Column("approved_at", sa.DateTime(timezone=True), nullable=True),
        sa.Column("rejected_by", sa.String(length=100), nullable=True),
        sa.Column("rejected_at", sa.DateTime(timezone=True), nullable=True),
        sa.Column("rejection_reason", sa.Text(), nullable=True),
        sa.Column("created_at", sa.DateTime(timezone=True), server_default=sa.func.now(), nullable=False),
        sa.Column("updated_at", sa.DateTime(timezone=True), server_default=sa.func.now(), nullable=False),
        sa.CheckConstraint(
            "risk_level in ('low', 'medium', 'high', 'critical')",
            name="ck_agentos_action_proposal_risk",
        ),
        sa.CheckConstraint(
            "status in ('suggested', 'pending_approval', 'approved', 'executing', 'executed', 'reviewed', 'rejected', 'expired', 'blocked_by_policy', 'failed', 'cancelled')",
            name="ck_agentos_action_proposal_status",
        ),
        sa.PrimaryKeyConstraint("id"),
    )
    op.create_index("ix_agentos_action_proposal_source_type", "agentos_action_proposal", ["source_type"])
    op.create_index("ix_agentos_action_proposal_source_id", "agentos_action_proposal", ["source_id"])
    op.create_index("ix_agentos_action_proposal_agent_id", "agentos_action_proposal", ["agent_id"])
    op.create_index("ix_agentos_action_proposal_squad_id", "agentos_action_proposal", ["squad_id"])
    op.create_index("ix_agentos_action_proposal_action_type", "agentos_action_proposal", ["action_type"])
    op.create_index("ix_agentos_action_proposal_business_object_type", "agentos_action_proposal", ["business_object_type"])
    op.create_index("ix_agentos_action_proposal_business_object_id", "agentos_action_proposal", ["business_object_id"])
    op.create_index("ix_agentos_action_proposal_risk_level", "agentos_action_proposal", ["risk_level"])
    op.create_index("ix_agentos_action_proposal_status", "agentos_action_proposal", ["status"])

    op.create_table(
        "agentos_approval_request",
        sa.Column("id", sa.BigInteger(), autoincrement=True, nullable=False),
        sa.Column("proposal_id", sa.BigInteger(), nullable=False),
        sa.Column("requester", sa.String(length=100), nullable=True),
        sa.Column("approver", sa.String(length=100), nullable=True),
        sa.Column("decision", sa.String(length=30), nullable=False, server_default="pending"),
        sa.Column("comment", sa.Text(), nullable=True),
        sa.Column("created_at", sa.DateTime(timezone=True), server_default=sa.func.now(), nullable=False),
        sa.Column("decided_at", sa.DateTime(timezone=True), nullable=True),
        sa.CheckConstraint("decision in ('pending', 'approved', 'rejected')", name="ck_agentos_approval_request_decision"),
        sa.ForeignKeyConstraint(["proposal_id"], ["agentos_action_proposal.id"]),
        sa.PrimaryKeyConstraint("id"),
    )
    op.create_index("ix_agentos_approval_request_proposal_id", "agentos_approval_request", ["proposal_id"])

    op.create_table(
        "agentos_command_execution",
        sa.Column("id", sa.BigInteger(), autoincrement=True, nullable=False),
        sa.Column("proposal_id", sa.BigInteger(), nullable=False),
        sa.Column("command_name", sa.String(length=100), nullable=False),
        sa.Column("executor", sa.String(length=100), nullable=True),
        sa.Column("status", sa.String(length=30), nullable=False, server_default="started"),
        sa.Column("input_payload", json_type, nullable=False, server_default=sa.text("'{}'::jsonb")),
        sa.Column("result_payload", json_type, nullable=True),
        sa.Column("error_message", sa.Text(), nullable=True),
        sa.Column("started_at", sa.DateTime(timezone=True), server_default=sa.func.now(), nullable=False),
        sa.Column("finished_at", sa.DateTime(timezone=True), nullable=True),
        sa.CheckConstraint("status in ('started', 'succeeded', 'failed')", name="ck_agentos_command_execution_status"),
        sa.ForeignKeyConstraint(["proposal_id"], ["agentos_action_proposal.id"]),
        sa.PrimaryKeyConstraint("id"),
    )
    op.create_index("ix_agentos_command_execution_proposal_id", "agentos_command_execution", ["proposal_id"])

    op.create_table(
        "agentos_outcome_review",
        sa.Column("id", sa.BigInteger(), autoincrement=True, nullable=False),
        sa.Column("proposal_id", sa.BigInteger(), nullable=False),
        sa.Column("outcome", sa.String(length=30), nullable=False),
        sa.Column("business_metric", sa.String(length=100), nullable=True),
        sa.Column("metric_delta", sa.Numeric(14, 4), nullable=True),
        sa.Column("notes", sa.Text(), nullable=True),
        sa.Column("reviewed_by", sa.String(length=100), nullable=True),
        sa.Column("created_at", sa.DateTime(timezone=True), server_default=sa.func.now(), nullable=False),
        sa.CheckConstraint("outcome in ('positive', 'neutral', 'negative')", name="ck_agentos_outcome_review_outcome"),
        sa.ForeignKeyConstraint(["proposal_id"], ["agentos_action_proposal.id"]),
        sa.PrimaryKeyConstraint("id"),
    )
    op.create_index("ix_agentos_outcome_review_proposal_id", "agentos_outcome_review", ["proposal_id"])


def downgrade() -> None:
    op.drop_index("ix_agentos_outcome_review_proposal_id", table_name="agentos_outcome_review")
    op.drop_table("agentos_outcome_review")
    op.drop_index("ix_agentos_command_execution_proposal_id", table_name="agentos_command_execution")
    op.drop_table("agentos_command_execution")
    op.drop_index("ix_agentos_approval_request_proposal_id", table_name="agentos_approval_request")
    op.drop_table("agentos_approval_request")
    op.drop_index("ix_agentos_action_proposal_status", table_name="agentos_action_proposal")
    op.drop_index("ix_agentos_action_proposal_risk_level", table_name="agentos_action_proposal")
    op.drop_index("ix_agentos_action_proposal_business_object_id", table_name="agentos_action_proposal")
    op.drop_index("ix_agentos_action_proposal_business_object_type", table_name="agentos_action_proposal")
    op.drop_index("ix_agentos_action_proposal_action_type", table_name="agentos_action_proposal")
    op.drop_index("ix_agentos_action_proposal_squad_id", table_name="agentos_action_proposal")
    op.drop_index("ix_agentos_action_proposal_agent_id", table_name="agentos_action_proposal")
    op.drop_index("ix_agentos_action_proposal_source_id", table_name="agentos_action_proposal")
    op.drop_index("ix_agentos_action_proposal_source_type", table_name="agentos_action_proposal")
    op.drop_table("agentos_action_proposal")

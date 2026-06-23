"""add entropy system tables (rule_mark_change, spc_control_limit)

Revision ID: 20260617_02
Revises: 20260617_01
Create Date: 2026-06-17
"""

from alembic import op
import sqlalchemy as sa


revision = "20260617_02"
down_revision = "20260617_01"
branch_labels = None
depends_on = None


def upgrade() -> None:
    # rule_mark_change — 标记变更审计日志 (熵系统底层)
    op.create_table(
        "rule_mark_change",
        sa.Column("id", sa.BigInteger(), primary_key=True, autoincrement=True),
        sa.Column(
            "target_type",
            sa.String(30),
            nullable=False,
            comment="变更目标: personal_rule/skill/threshold/honcho_profile",
        ),
        sa.Column("target_id", sa.BigInteger(), nullable=False, comment="目标ID"),
        sa.Column("field_path", sa.String(200), nullable=False, comment="JSON路径"),
        sa.Column("old_value", sa.JSON(), comment="旧值"),
        sa.Column("new_value", sa.JSON(), nullable=False, comment="新值"),
        sa.Column(
            "source_type",
            sa.String(30),
            nullable=False,
            comment="来源: gds/gds_proxy/human/nudge/auto_extract",
        ),
        sa.Column("source_id", sa.String(100), comment="具体来源标识"),
        sa.Column("change_summary", sa.Text(), nullable=False, comment="变更说明"),
        sa.Column("parent_change_id", sa.BigInteger(), comment="关联的触发变更"),
        sa.Column("related_decision_ids", sa.JSON(), comment="关联的决策ID列表"),
        sa.Column("context_json", sa.JSON(), comment="触发变更的上下文快照"),
        sa.Column(
            "created_at",
            sa.DateTime(timezone=True),
            server_default=sa.func.now(),
            comment="创建时间",
        ),
    )
    op.create_index(
        "idx_mark_changes_source_time",
        "rule_mark_change",
        ["source_type", sa.text("created_at DESC")],
    )
    op.create_index(
        "idx_mark_changes_target",
        "rule_mark_change",
        ["target_type", "target_id", sa.text("created_at DESC")],
    )

    # spc_control_limit — SPC 统计过程控制
    op.create_table(
        "spc_control_limit",
        sa.Column("id", sa.BigInteger(), primary_key=True, autoincrement=True),
        sa.Column(
            "user_id",
            sa.BigInteger(),
            sa.ForeignKey("user.id"),
            nullable=False,
            comment="用户ID",
        ),
        sa.Column("agent_id", sa.String(20), nullable=False, comment="Agent标识"),
        sa.Column("decision_point", sa.String(50), nullable=False, comment="决策点"),
        sa.Column("metric_name", sa.String(50), nullable=False, comment="指标名"),
        sa.Column("baseline_mean", sa.Numeric(10, 4), nullable=False, comment="均值"),
        sa.Column(
            "baseline_stddev", sa.Numeric(10, 4), nullable=False, comment="标准差"
        ),
        sa.Column("baseline_samples", sa.Integer(), nullable=False, comment="样本数"),
        sa.Column("ucl", sa.Numeric(10, 4), nullable=False, comment="μ+3σ 上控制线"),
        sa.Column("lcl", sa.Numeric(10, 4), nullable=False, comment="μ-3σ 下控制线"),
        sa.Column("uwl", sa.Numeric(10, 4), nullable=False, comment="μ+2σ 上警戒线"),
        sa.Column("lwl", sa.Numeric(10, 4), nullable=False, comment="μ-2σ 下警戒线"),
        sa.Column(
            "consecutive_same_side", sa.Integer(), default=0, comment="连续同侧点数"
        ),
        sa.Column("last_breach_at", sa.DateTime(timezone=True), comment="上次越线时间"),
        sa.Column(
            "baseline_recalc_at",
            sa.DateTime(timezone=True),
            nullable=False,
            comment="基线计算时间",
        ),
        sa.Column(
            "next_recalc_at",
            sa.DateTime(timezone=True),
            nullable=False,
            comment="下次重算时间",
        ),
        sa.Column(
            "created_at",
            sa.DateTime(timezone=True),
            server_default=sa.func.now(),
            comment="创建时间",
        ),
        sa.Column(
            "updated_at",
            sa.DateTime(timezone=True),
            server_default=sa.func.now(),
            comment="更新时间",
        ),
    )
    op.create_index(
        "idx_spc_user_agent",
        "spc_control_limit",
        ["user_id", "agent_id", "decision_point"],
    )
    op.create_unique_constraint(
        "uq_spc_metric",
        "spc_control_limit",
        ["user_id", "agent_id", "decision_point", "metric_name"],
    )


def downgrade() -> None:
    op.drop_constraint("uq_spc_metric", "spc_control_limit")
    op.drop_index("idx_spc_user_agent", table_name="spc_control_limit")
    op.drop_table("spc_control_limit")
    op.drop_index("idx_mark_changes_target", table_name="rule_mark_change")
    op.drop_index("idx_mark_changes_source_time", table_name="rule_mark_change")
    op.drop_table("rule_mark_change")

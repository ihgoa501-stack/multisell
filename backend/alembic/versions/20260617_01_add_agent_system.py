"""add agent system tables (decision, rules, episodes, honcho, conflicts)

Revision ID: 20260617_01
Revises: 20260615_02
Create Date: 2026-06-17
"""

from alembic import op
import sqlalchemy as sa


revision = "20260617_01"
down_revision = "20260615_02"
branch_labels = None
depends_on = None


def upgrade() -> None:
    # agent_decision — 决策日志 (Hermes Episodic Memory)
    op.create_table(
        "agent_decision",
        sa.Column("id", sa.BigInteger(), primary_key=True, autoincrement=True),
        sa.Column("user_id", sa.BigInteger(), sa.ForeignKey("user.id"), nullable=False, comment="用户ID"),
        sa.Column("agent_id", sa.String(20), nullable=False, comment="Agent标识: A3/A4/A5/A6/A7/G1/G2/G3"),
        sa.Column("decision_point", sa.String(50), nullable=False, comment="决策点"),
        sa.Column("context_json", sa.JSON(), nullable=False, comment="决策上下文"),
        sa.Column("agent_output", sa.JSON(), nullable=False, comment="Agent原始输出"),
        sa.Column("final_decision", sa.JSON(), nullable=False, comment="最终执行的决策"),
        sa.Column("user_action", sa.String(20), nullable=False, comment="用户操作: accepted/modified/rejected/ignored"),
        sa.Column("user_overrides", sa.JSON(), comment="用户修改内容"),
        sa.Column("user_feedback", sa.Text(), comment="用户显式反馈"),
        sa.Column("rules_applied", sa.JSON(), comment="应用的规则ID列表"),
        sa.Column("rule_overrides", sa.Integer(), default=0, comment="规则覆盖次数"),
        sa.Column("evolution_stage", sa.String(20), nullable=False, comment="进化阶段"),
        sa.Column("confidence", sa.Numeric(4, 3), comment="置信度"),
        sa.Column("response_time_ms", sa.Integer(), comment="响应耗时(ms)"),
        sa.Column("token_count", sa.Integer(), comment="Token消耗"),
        sa.Column("session_id", sa.String(100), nullable=False, comment="会话ID"),
        sa.Column("episode_id", sa.BigInteger(), comment="Episode批次ID"),
        sa.Column("created_at", sa.DateTime(timezone=True), server_default=sa.func.now(), comment="创建时间"),
    )
    op.create_index("idx_agent_decision_user_agent", "agent_decision", ["user_id", "agent_id", "decision_point"])
    op.create_index("idx_agent_decision_created", "agent_decision", ["created_at"])

    # personal_rule — 个人规则库
    op.create_table(
        "personal_rule",
        sa.Column("id", sa.BigInteger(), primary_key=True, autoincrement=True),
        sa.Column("user_id", sa.BigInteger(), sa.ForeignKey("user.id"), nullable=False, comment="用户ID"),
        sa.Column("agent_id", sa.String(20), nullable=False, comment="Agent标识"),
        sa.Column("decision_point", sa.String(50), nullable=False, comment="决策点"),
        sa.Column("rule_type", sa.String(20), nullable=False, comment="规则类型: threshold/strategy/style/veto"),
        sa.Column("rule_name", sa.String(100), nullable=False, comment="规则名称"),
        sa.Column("rule_condition", sa.JSON(), nullable=False, comment="规则条件"),
        sa.Column("rule_action", sa.JSON(), nullable=False, comment="规则动作"),
        sa.Column("priority", sa.Integer(), default=100, comment="优先级"),
        sa.Column("source", sa.String(20), nullable=False, comment="来源: manual/nudge/auto_extracted/template"),
        sa.Column("source_decisions", sa.JSON(), comment="来源决策ID列表"),
        sa.Column("status", sa.String(20), default="active", comment="状态: active/shadow/paused/retired"),
        sa.Column("confidence", sa.Numeric(4, 3), default=0, comment="置信度"),
        sa.Column("times_applied", sa.Integer(), default=0, comment="应用次数"),
        sa.Column("times_overridden", sa.Integer(), default=0, comment="被覆盖次数"),
        sa.Column("last_applied_at", sa.DateTime(timezone=True), comment="最后应用时间"),
        sa.Column("created_at", sa.DateTime(timezone=True), server_default=sa.func.now(), comment="创建时间"),
        sa.Column("updated_at", sa.DateTime(timezone=True), server_default=sa.func.now(), comment="更新时间"),
    )
    op.create_index("idx_personal_rule_user_agent", "personal_rule", ["user_id", "agent_id", "decision_point"])

    # agent_episode — Episode汇总
    op.create_table(
        "agent_episode",
        sa.Column("id", sa.BigInteger(), primary_key=True, autoincrement=True),
        sa.Column("user_id", sa.BigInteger(), sa.ForeignKey("user.id"), nullable=False, comment="用户ID"),
        sa.Column("agent_id", sa.String(20), nullable=False, comment="Agent标识"),
        sa.Column("episode_number", sa.Integer(), nullable=False, comment="Episode序号"),
        sa.Column("decision_count", sa.Integer(), nullable=False, comment="决策数量"),
        sa.Column("episode_summary", sa.Text(), comment="Episode摘要"),
        sa.Column("key_insights", sa.JSON(), comment="关键洞察"),
        sa.Column("improvement_suggestions", sa.JSON(), comment="改进建议"),
        sa.Column("acceptance_rate", sa.Numeric(4, 3), comment="采纳率"),
        sa.Column("avg_confidence", sa.Numeric(4, 3), comment="平均置信度"),
        sa.Column("avg_response_ms", sa.Integer(), comment="平均响应耗时"),
        sa.Column("total_tokens", sa.Integer(), comment="总Token消耗"),
        sa.Column("nudge_triggered", sa.Integer(), default=0, comment="Nudge触发次数"),
        sa.Column("nudge_topics", sa.JSON(), comment="Nudge话题"),
        sa.Column("nudge_response", sa.Text(), comment="Nudge用户响应"),
        sa.Column("started_at", sa.DateTime(timezone=True), nullable=False, comment="开始时间"),
        sa.Column("ended_at", sa.DateTime(timezone=True), nullable=False, comment="结束时间"),
        sa.Column("created_at", sa.DateTime(timezone=True), server_default=sa.func.now(), comment="创建时间"),
    )

    # honcho_profile — 用户模型 (辩证建模)
    op.create_table(
        "honcho_profile",
        sa.Column("id", sa.BigInteger(), primary_key=True, autoincrement=True),
        sa.Column("user_id", sa.BigInteger(), sa.ForeignKey("user.id"), nullable=False, unique=True, comment="用户ID"),
        sa.Column("risk_tolerance", sa.String(30), default="moderate", comment="风险容忍度"),
        sa.Column("communication_style", sa.String(20), default="balanced", comment="沟通风格"),
        sa.Column("notification_prefs", sa.JSON(), comment="通知偏好"),
        sa.Column("agent_profiles", sa.JSON(), nullable=False, comment="各Agent配置档案"),
        sa.Column("hypothesis_count", sa.Integer(), default=0, comment="假设数量"),
        sa.Column("confirmed_count", sa.Integer(), default=0, comment="已验证数量"),
        sa.Column("last_dialectic_at", sa.DateTime(timezone=True), comment="上次辩证对话时间"),
        sa.Column("created_at", sa.DateTime(timezone=True), server_default=sa.func.now(), comment="创建时间"),
        sa.Column("updated_at", sa.DateTime(timezone=True), server_default=sa.func.now(), comment="更新时间"),
    )

    # rule_conflict — 规则冲突日志
    op.create_table(
        "rule_conflict",
        sa.Column("id", sa.BigInteger(), primary_key=True, autoincrement=True),
        sa.Column("decision_id", sa.BigInteger(), sa.ForeignKey("agent_decision.id"), nullable=False, comment="决策ID"),
        sa.Column("conflicting_rules", sa.JSON(), nullable=False, comment="冲突规则ID列表"),
        sa.Column("winner_rule_id", sa.BigInteger(), nullable=False, comment="胜出规则ID"),
        sa.Column("resolution", sa.String(20), nullable=False, comment="解决方式"),
        sa.Column("nudge_sent", sa.Integer(), default=0, comment="是否已发送Nudge"),
        sa.Column("nudge_resolved", sa.Integer(), default=0, comment="是否已解决"),
        sa.Column("user_choice", sa.BigInteger(), comment="用户选择的规则ID"),
        sa.Column("created_at", sa.DateTime(timezone=True), server_default=sa.func.now(), comment="创建时间"),
    )


def downgrade() -> None:
    op.drop_table("rule_conflict")
    op.drop_table("honcho_profile")
    op.drop_table("agent_episode")
    op.drop_table("personal_rule")
    op.drop_index("idx_agent_decision_created", table_name="agent_decision")
    op.drop_index("idx_agent_decision_user_agent", table_name="agent_decision")
    op.drop_table("agent_decision")

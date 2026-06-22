"""Agent 数据库模型"""

from sqlalchemy import (
    BigInteger, Boolean, Column, DateTime, ForeignKey, Integer,
    JSON, Numeric, SmallInteger, String, Text,
    UniqueConstraint as sa_UniqueConstraint,
    func,
)
from sqlalchemy.dialects.postgresql import ARRAY
from sqlalchemy.orm import relationship

from app.database import Base


class AgentDecision(Base):
    """Agent决策日志 (Hermes Episodic Memory)"""
    __tablename__ = "agent_decision"

    id = Column(BigInteger, primary_key=True, autoincrement=True)
    user_id = Column(BigInteger, ForeignKey("user.id"), nullable=False, comment="用户ID")
    agent_id = Column(String(20), nullable=False, comment="Agent标识: A3/A4/A5/A6/A7/G1/G2/G3")
    decision_point = Column(String(50), nullable=False, comment="决策点: acos_adjustment/stock_alert/discount_check 等")

    context_json = Column(JSON, nullable=False, comment="决策上下文")
    agent_output = Column(JSON, nullable=False, comment="Agent原始输出")
    final_decision = Column(JSON, nullable=False, comment="最终执行的决策")

    user_action = Column(String(20), nullable=False, comment="用户操作: accepted/modified/rejected/ignored")
    user_overrides = Column(JSON, comment="用户修改内容")
    user_feedback = Column(Text, comment="用户显式反馈")

    rules_applied = Column(JSON, comment="应用的规则ID列表")
    rule_overrides = Column(Integer, default=0, comment="规则覆盖次数")

    evolution_stage = Column(String(20), nullable=False, comment="进化阶段: observation/suggestion/semi_autonomous/full_autonomous")
    confidence = Column(Numeric(4, 3), comment="置信度 0.000-1.000")

    response_time_ms = Column(Integer, comment="响应耗时(ms)")
    token_count = Column(Integer, comment="Token消耗")

    session_id = Column(String(100), nullable=False, comment="会话ID")
    episode_id = Column(BigInteger, comment="Episode批次ID")

    store_id = Column(Integer, ForeignKey("stores.id"), nullable=True, comment="店铺ID（null=全店铺通用）")
    llm_model_used = Column(String(50), nullable=True, comment="实际使用的模型")
    llm_errors = Column(JSON, nullable=True, comment="LLM错误列表 [{timestamp, error_type, model}]")
    cached = Column(Boolean, default=False, comment="是否命中缓存")
    archived = Column(Boolean, default=False, comment="是否已归档")
    archived_at = Column(DateTime(timezone=True), nullable=True, comment="归档时间")

    created_at = Column(DateTime(timezone=True), server_default=func.now(), comment="创建时间")


class PersonalRule(Base):
    """个人规则库"""
    __tablename__ = "personal_rule"

    id = Column(BigInteger, primary_key=True, autoincrement=True)
    user_id = Column(BigInteger, ForeignKey("user.id"), nullable=False, comment="用户ID")
    agent_id = Column(String(20), nullable=False, comment="Agent标识")
    decision_point = Column(String(50), nullable=False, comment="决策点")

    rule_type = Column(String(20), nullable=False, comment="规则类型: threshold/strategy/style/veto")
    rule_name = Column(String(100), nullable=False, comment="规则名称")
    rule_condition = Column(JSON, nullable=False, comment="规则条件")
    rule_action = Column(JSON, nullable=False, comment="规则动作")
    priority = Column(Integer, default=100, comment="优先级")

    source = Column(String(20), nullable=False, comment="来源: manual/nudge/auto_extracted/template")
    source_decisions = Column(JSON, comment="来源决策ID列表")
    status = Column(String(20), default="active", comment="状态: active/shadow/paused/retired")
    confidence = Column(Numeric(4, 3), default=0, comment="置信度")

    times_applied = Column(Integer, default=0, comment="应用次数")
    times_overridden = Column(Integer, default=0, comment="被覆盖次数")
    last_applied_at = Column(DateTime(timezone=True), comment="最后应用时间")

    store_id = Column(Integer, ForeignKey("stores.id"), nullable=True, comment="店铺ID（null=全店铺通用）")
    last_manual_edit_at = Column(DateTime(timezone=True), nullable=True, comment="最后手动编辑时间")

    created_at = Column(DateTime(timezone=True), server_default=func.now(), comment="创建时间")
    updated_at = Column(DateTime(timezone=True), server_default=func.now(), onupdate=func.now(), comment="更新时间")


class AgentEpisode(Base):
    """Agent Episode汇总 (每N个决策一批)"""
    __tablename__ = "agent_episode"

    id = Column(BigInteger, primary_key=True, autoincrement=True)
    user_id = Column(BigInteger, ForeignKey("user.id"), nullable=False, comment="用户ID")
    agent_id = Column(String(20), nullable=False, comment="Agent标识")

    episode_number = Column(Integer, nullable=False, comment="Episode序号")
    decision_count = Column(Integer, nullable=False, comment="决策数量")
    episode_summary = Column(Text, comment="Episode摘要")
    key_insights = Column(JSON, comment="关键洞察")
    improvement_suggestions = Column(JSON, comment="改进建议")

    acceptance_rate = Column(Numeric(4, 3), comment="采纳率")
    avg_confidence = Column(Numeric(4, 3), comment="平均置信度")
    avg_response_ms = Column(Integer, comment="平均响应耗时")
    total_tokens = Column(Integer, comment="总Token消耗")

    nudge_triggered = Column(Integer, default=0, comment="Nudge触发次数")
    nudge_topics = Column(JSON, comment="Nudge话题")
    nudge_response = Column(Text, comment="Nudge用户响应")

    started_at = Column(DateTime(timezone=True), nullable=False, comment="开始时间")
    ended_at = Column(DateTime(timezone=True), nullable=False, comment="结束时间")
    created_at = Column(DateTime(timezone=True), server_default=func.now(), comment="创建时间")


class HonchoProfile(Base):
    """Honcho用户模型 (辩证建模)"""
    __tablename__ = "honcho_profile"

    id = Column(BigInteger, primary_key=True, autoincrement=True)
    user_id = Column(BigInteger, ForeignKey("user.id"), nullable=False, unique=True, comment="用户ID")

    risk_tolerance = Column(String(30), default="moderate", comment="风险容忍度: conservative/moderate/aggressive")
    communication_style = Column(String(20), default="balanced", comment="沟通风格: concise/balanced/detailed")
    notification_prefs = Column(JSON, comment="通知偏好")
    agent_profiles = Column(JSON, nullable=False, default=lambda: {}, comment="各Agent配置档案")

    hypothesis_count = Column(Integer, default=0, comment="假设数量")
    confirmed_count = Column(Integer, default=0, comment="已验证数量")
    last_dialectic_at = Column(DateTime(timezone=True), comment="上次辩证对话时间")
    created_at = Column(DateTime(timezone=True), server_default=func.now(), comment="创建时间")
    updated_at = Column(DateTime(timezone=True), server_default=func.now(), onupdate=func.now(), comment="更新时间")


class AgentAction(Base):
    """Agent 待执行操作（用户确认后执行）"""
    __tablename__ = "agent_pending_action"

    id = Column(BigInteger, primary_key=True, autoincrement=True)
    user_id = Column(BigInteger, ForeignKey("user.id"), nullable=False, comment="用户ID")
    agent_id = Column(String(20), nullable=False, comment="Agent标识")
    decision_id = Column(BigInteger, ForeignKey("agent_decision.id"), comment="关联决策ID")
    action_type = Column(String(50), nullable=False, comment="操作类型: replenish/price_adjust/discount_review/ad_action/notify")
    status = Column(String(20), nullable=False, default="pending", comment="状态: pending/confirmed/executed/rejected")
    summary = Column(String(500), nullable=False, comment="操作摘要")
    action_payload = Column(JSON, comment="执行参数")
    execution_result = Column(JSON, comment="执行结果")

    store_id = Column(Integer, ForeignKey("stores.id"), nullable=True, comment="店铺ID（null=全店铺通用）")

    created_at = Column(DateTime(timezone=True), server_default=func.now(), comment="创建时间")
    updated_at = Column(DateTime(timezone=True), server_default=func.now(), onupdate=func.now(), comment="更新时间")


class RuleConflict(Base):
    """规则冲突日志"""
    __tablename__ = "rule_conflict"

    id = Column(BigInteger, primary_key=True, autoincrement=True)
    decision_id = Column(BigInteger, ForeignKey("agent_decision.id"), nullable=False, comment="决策ID")
    conflicting_rules = Column(JSON, nullable=False, comment="冲突规则ID列表")
    winner_rule_id = Column(BigInteger, nullable=False, comment="胜出规则ID")
    resolution = Column(String(20), nullable=False, comment="解决方式: auto_priority/user_choice/latest_wins")
    nudge_sent = Column(Integer, default=0, comment="是否已发送Nudge")
    nudge_resolved = Column(Integer, default=0, comment="是否已解决")
    user_choice = Column(BigInteger, comment="用户选择的规则ID")
    created_at = Column(DateTime(timezone=True), server_default=func.now(), comment="创建时间")


class RuleMarkChange(Base):
    """标记变更表 — 熵系统底层审计日志 (Mark Change Pattern)"""
    __tablename__ = "rule_mark_change"

    id = Column(BigInteger, primary_key=True, autoincrement=True)

    target_type = Column(String(30), nullable=False, comment="变更目标: personal_rule/skill/threshold/honcho_profile")
    target_id = Column(BigInteger, nullable=False, comment="目标ID")
    field_path = Column(String(200), nullable=False, comment="JSON路径")

    old_value = Column(JSON, comment="旧值")
    new_value = Column(JSON, nullable=False, comment="新值")

    source_type = Column(String(30), nullable=False, comment="来源: gds/gds_proxy/human/nudge/auto_extract")
    source_id = Column(String(100), comment="具体来源标识")
    change_summary = Column(Text, nullable=False, comment="变更说明")

    parent_change_id = Column(BigInteger, comment="关联的触发变更")
    related_decision_ids = Column(JSON, comment="关联的决策ID列表")
    context_json = Column(JSON, comment="触发变更的上下文快照")

    created_at = Column(DateTime(timezone=True), server_default=func.now(), comment="创建时间")


class SpcControlLimit(Base):
    """SPC 统计过程控制 — 决策指标监控"""
    __tablename__ = "spc_control_limit"

    id = Column(BigInteger, primary_key=True, autoincrement=True)
    user_id = Column(BigInteger, ForeignKey("user.id"), nullable=False, comment="用户ID")
    agent_id = Column(String(20), nullable=False, comment="Agent标识")
    decision_point = Column(String(50), nullable=False, comment="决策点")
    metric_name = Column(String(50), nullable=False, comment="指标名: acceptance_rate/confidence/override_rate")

    baseline_mean = Column(Numeric(10, 4), nullable=False, comment="均值")
    baseline_stddev = Column(Numeric(10, 4), nullable=False, comment="标准差")
    baseline_samples = Column(Integer, nullable=False, comment="样本数")

    ucl = Column(Numeric(10, 4), nullable=False, comment="μ+3σ 上控制线")
    lcl = Column(Numeric(10, 4), nullable=False, comment="μ-3σ 下控制线")
    uwl = Column(Numeric(10, 4), nullable=False, comment="μ+2σ 上警戒线")
    lwl = Column(Numeric(10, 4), nullable=False, comment="μ-2σ 下警戒线")

    consecutive_same_side = Column(Integer, default=0, comment="连续同侧点数")
    last_breach_at = Column(DateTime(timezone=True), comment="上次越线时间")

    baseline_recalc_at = Column(DateTime(timezone=True), nullable=False, comment="基线计算时间")
    next_recalc_at = Column(DateTime(timezone=True), nullable=False, comment="下次重算时间")

    created_at = Column(DateTime(timezone=True), server_default=func.now(), comment="创建时间")
    updated_at = Column(DateTime(timezone=True), server_default=func.now(), onupdate=func.now(), comment="更新时间")

    __table_args__ = (
        sa_UniqueConstraint("user_id", "agent_id", "decision_point", "metric_name", name="uq_spc_metric"),
    )


class AgentEvolutionConfig(Base):
    """Agent 进化配置 — 持久化每个 user × agent × decision_point 的阶段和信任评分"""
    __tablename__ = "agent_evolution_config"

    id = Column(BigInteger, primary_key=True, autoincrement=True)
    user_id = Column(BigInteger, ForeignKey("user.id"), nullable=False, comment="用户ID")
    agent_id = Column(String(20), nullable=False, comment="Agent标识")
    decision_point = Column(String(50), nullable=False, comment="决策点")

    current_stage = Column(String(20), nullable=False, default="observation", comment="当前自治阶段")
    stage_updated_at = Column(DateTime(timezone=True), comment="阶段最后变更时间")
    stage_updated_by = Column(String(20), default="system", comment="变更来源: manual/nudge/regret/system")

    trust_score = Column(Numeric(6, 2), default=0, comment="信任评分 0-100")
    decision_count = Column(Integer, default=0, comment="累计决策样本数")
    adoption_rate = Column(Numeric(5, 4), comment="采纳率")
    avg_confidence = Column(Numeric(5, 4), comment="平均置信度")
    consistency_score = Column(Numeric(5, 4), comment="规则覆盖一致性")
    stability_score = Column(Numeric(5, 4), comment="SPC 稳定性")
    last_calculated_at = Column(DateTime(timezone=True), comment="信任评分最后计算时间")

    nudge_last_shown_at = Column(DateTime(timezone=True), comment="上次 Nudge 提示时间")
    nudge_dismissed_count = Column(Integer, default=0, comment="Nudge 忽略次数")

    created_at = Column(DateTime(timezone=True), server_default=func.now(), comment="创建时间")
    updated_at = Column(DateTime(timezone=True), server_default=func.now(), onupdate=func.now(), comment="更新时间")

    __table_args__ = (
        sa_UniqueConstraint("user_id", "agent_id", "decision_point", name="uq_agent_evolution"),
    )


class AgentNudge(Base):
    """Agent Nudge 晋升提示记录"""
    __tablename__ = "agent_nudge"

    id = Column(BigInteger, primary_key=True, autoincrement=True)
    user_id = Column(BigInteger, ForeignKey("user.id"), nullable=False, comment="用户ID")
    agent_id = Column(String(20), nullable=False, comment="Agent标识")
    decision_point = Column(String(50), nullable=False, comment="决策点")

    target_stage = Column(String(20), nullable=False, comment="目标晋升阶段")
    trust_score_at_time = Column(Numeric(6, 2), nullable=False, comment="提示时的信任评分")
    score_components = Column(JSON, comment="评分分量明细")

    status = Column(String(20), nullable=False, default="pending", comment="状态: pending/accepted/dismissed/expired")
    responded_at = Column(DateTime(timezone=True), comment="响应时间")
    cooling_until = Column(DateTime(timezone=True), comment="冷却截止时间（忽略后7天）")

    created_at = Column(DateTime(timezone=True), server_default=func.now(), comment="创建时间")
    updated_at = Column(DateTime(timezone=True), server_default=func.now(), onupdate=func.now(), comment="更新时间")


class ConflictPolicy(Base):
    """Agent间冲突仲裁策略（Policy Matrix）"""
    __tablename__ = "agent_conflict_policy"

    id = Column(BigInteger, primary_key=True, autoincrement=True)
    user_id = Column(BigInteger, ForeignKey("user.id"), nullable=False, comment="用户ID")
    agent_id_a = Column(String(20), nullable=False, comment="冲突参与方A")
    agent_id_b = Column(String(20), nullable=False, comment="冲突参与方B")
    decision_point = Column(String(50), nullable=False, comment="适用的决策点")
    condition = Column(JSON, nullable=True, comment="触发条件（可选，如金额范围）")
    winner = Column(String(20), nullable=False, comment="冲突时谁的决策优先")
    reason = Column(Text, nullable=False, comment="预置理由")
    priority = Column(Integer, default=100, comment="匹配优先级")
    created_at = Column(DateTime(timezone=True), server_default=func.now(), comment="创建时间")
    updated_at = Column(DateTime(timezone=True), server_default=func.now(), onupdate=func.now(), comment="更新时间")


class ArbitrationLog(Base):
    """仲裁日志"""
    __tablename__ = "agent_arbitration_log"

    id = Column(BigInteger, primary_key=True, autoincrement=True)
    business_type = Column(String(50), nullable=False, comment="业务类型: sku/campaign/order")
    business_id = Column(String(100), nullable=False, comment="业务ID")
    conflict_keys = Column(ARRAY(Text), nullable=False, comment="冲突的decision_id列表")
    stage = Column(String(20), nullable=False, comment="仲裁阶段: policy/arbiter/manual")
    policy_id = Column(Integer, nullable=True, comment="命中的policy ID")
    verdict = Column(String(20), nullable=True, comment="仲裁结论")
    arbiter_output = Column(JSON, nullable=True, comment="G0的完整输出")
    resolved_by = Column(String(100), nullable=False, comment="system/arbiter_G0/用户名")
    created_at = Column(DateTime(timezone=True), server_default=func.now(), comment="创建时间")


class AgentDecisionCache(Base):
    """LLM决策缓存"""
    __tablename__ = "agent_decision_cache"

    cache_key = Column(String(64), primary_key=True, comment="缓存Key (SHA256)")
    decision_json = Column(JSON, nullable=False, comment="缓存决策内容")
    expires_at = Column(DateTime(timezone=True), nullable=False, comment="过期时间")


class ScheduleFailure(Base):
    """调度失败记录"""
    __tablename__ = "agent_schedule_failure"

    id = Column(BigInteger, primary_key=True, autoincrement=True)
    agent_id = Column(String(20), nullable=False, comment="Agent标识")
    store_id = Column(Integer, nullable=False, comment="店铺ID")
    decision_point = Column(String(50), nullable=False, comment="决策点")
    error_msg = Column(Text, nullable=True, comment="错误信息")
    failed_at = Column(DateTime(timezone=True), server_default=func.now(), comment="失败时间")
    retry_count = Column(Integer, default=0, comment="重试次数")
    last_retry_at = Column(DateTime(timezone=True), nullable=True, comment="上次重试时间")
    resolved = Column(Boolean, default=False, comment="是否已解决")


class SystemConfig(Base):
    """系统配置（LLM Key、提供商、模型选择等）"""
    __tablename__ = "system_config"

    id = Column(BigInteger, primary_key=True, autoincrement=True)
    config_key = Column(String(100), nullable=False, unique=True, comment="配置键名")
    config_value = Column(Text, comment="配置值（明文，仅用于非敏感配置）")
    config_json = Column(JSON, comment="配置值（JSON 格式）")
    is_secret = Column(SmallInteger, default=0, comment="是否敏感配置（前端脱敏显示）")
    description = Column(String(500), comment="配置说明")
    updated_by = Column(BigInteger, comment="更新人")
    created_at = Column(DateTime(timezone=True), server_default=func.now(), comment="创建时间")
    updated_at = Column(DateTime(timezone=True), server_default=func.now(), onupdate=func.now(), comment="更新时间")

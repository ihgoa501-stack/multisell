package platformtruth

// TruthLevel is the canonical evidence claim level used across Owner-facing domains.
type TruthLevel struct {
	Code        string `json:"code"`
	Meaning     string `json:"meaning"`
	CanBeDirect bool   `json:"can_be_direct"`
}

type ClaimLevel struct {
	Code    string `json:"code"`
	Meaning string `json:"meaning"`
}

type SystemBoundary struct {
	Code           string `json:"code"`
	Name           string `json:"name"`
	Responsibility string `json:"responsibility"`
	MustNot        string `json:"must_not"`
}

type ContractRule struct {
	Code string `json:"code"`
	Rule string `json:"rule"`
}

type DomainDisposition struct {
	ID          string `json:"id"`
	Name        string `json:"name"`
	System      string `json:"system"`
	Disposition string `json:"disposition"`
	Reason      string `json:"reason"`
	Evidence    string `json:"evidence"`
	XiaoQ       string `json:"xiao_q_support"`
	OwnerScope  string `json:"owner_scope"`
	Risk        string `json:"risk"`
}

type Contract struct {
	Version             string              `json:"version"`
	Direction           string              `json:"direction"`
	TruthLevels         []TruthLevel        `json:"truth_levels"`
	ClaimLevels         []ClaimLevel        `json:"claim_levels"`
	SystemBoundaries    []SystemBoundary    `json:"system_boundaries"`
	ObjectIdentityRules []ContractRule      `json:"object_identity_rules"`
	SourceRules         []ContractRule      `json:"source_rules"`
	DomainDispositions  []DomainDisposition `json:"domain_dispositions"`
	BoundaryRules       []string            `json:"boundary_rules"`
	Unknowns            []string            `json:"unknowns"`
}

func CurrentContract() Contract {
	return Contract{
		Version:   "2026-07-12",
		Direction: "只供 Owner 本人使用的完整 AI 跨境电商经营平台；按可独立验收的完整纵向单元推进。",
		TruthLevels: []TruthLevel{
			{Code: "actual", Meaning: "由受控来源直接观察并保留对象、来源和时间的事实", CanBeDirect: false},
			{Code: "quoted", Meaning: "来自明确外部来源的说法，尚未由系统直接核验", CanBeDirect: true},
			{Code: "estimated", Meaning: "基于明确方法和输入计算的估算", CanBeDirect: true},
			{Code: "unknown", Meaning: "当前缺少足够证据，不能判断", CanBeDirect: true},
			{Code: "mock", Meaning: "模拟、种子、演示或固定返回值", CanBeDirect: true},
			{Code: "inferred", Meaning: "根据证据作出的推断，不等于直接事实", CanBeDirect: true},
		},
		ClaimLevels: []ClaimLevel{
			{Code: "policy", Meaning: "Owner 当前政策，不代表已经实现"}, {Code: "planned", Meaning: "已规划但未实现"},
			{Code: "implemented", Meaning: "代码、迁移、路由或页面存在"}, {Code: "automated_verified", Meaning: "自动测试或构建在注明环境通过"},
			{Code: "manually_verified", Meaning: "人工完成注明步骤的运行验证"}, {Code: "external_observed", Meaning: "从外部平台或现实事件观察并留存来源与时间"},
			{Code: "reconciled", Meaning: "与可信结算、银行或现金凭证完成对账"}, {Code: "superseded", Meaning: "已被新方向替代，仅供历史追溯"},
		},
		SystemBoundaries: []SystemBoundary{
			{Code: "fact", Name: "经营事实系统", Responsibility: "保存真实对象、来源、状态、时间和对账关系", MustNot: "不得把对象关联、流程终态或估算自动解释为因果与反馈"},
			{Code: "decision", Name: "经营决策系统", Responsibility: "保存经营问题、目标、动作变量、批准、执行、观测、偏差、决定和下一动作", MustNot: "不得替代事实系统计算确定性交易与财务事实"},
			{Code: "collaboration", Name: "Owner AI 协作层", Responsibility: "小Q通过登记 Capability 解释、建议和协调", MustNot: "不得绕过 Owner、RBAC、审批、审计和领域状态机"},
			{Code: "kernel", Name: "平台内核", Responsibility: "提供身份、权限、审批、审计、幂等、事件、命令、调度、工具桥和可观测机制", MustNot: "不得承载具体经营决策"},
			{Code: "support", Name: "共享支撑", Responsibility: "为事实或决策领域提供确定性辅助能力", MustNot: "不得成为平行事实源或绕过所属领域"},
			{Code: "frozen", Name: "冻结范围", Responsibility: "仅保留兼容或历史追溯", MustNot: "不得进入当前开发队列"},
		},
		ObjectIdentityRules: []ContractRule{
			{Code: "stable_id", Rule: "每个经营对象必须有稳定类型和 ID；名称、URL、SKU 文本或模型输出不能替代身份。"},
			{Code: "owner_scope", Rule: "所有 Owner 数据必须绑定 single_owner 边界；交易对手不是软件用户或租户。"},
			{Code: "cross_domain_link", Rule: "跨领域关联必须同时保存对象类型和对象 ID；关联只证明可追踪，不证明因果。"},
			{Code: "external_identity", Rule: "外部对象必须保存平台、外部 ID 和必要版本；提交成功不等于外部终态。"},
		},
		SourceRules: []ContractRule{
			{Code: "provenance", Rule: "事实或引用必须保存来源 URI/外部凭证、观察时间和采集方式。"},
			{Code: "immutable_snapshot", Rule: "会影响经营决定的外部线索必须引用不可变快照及内容哈希。"},
			{Code: "source_authority", Rule: "普通人工或模型录入不能直接声明 actual；受控系统或 Owner 独立核验才可升级。"},
			{Code: "financial_reconciliation", Rule: "最终利润与现金必须绑定同一业务对象、可信明细和对账凭证。"},
		},
		DomainDispositions: domainDispositions(),
		BoundaryRules: []string{
			"对象关联只支持追踪，不证明行动与结果之间存在因果关系。",
			"模块存在、测试通过或页面可见不证明真实经营事件发生。",
			"经营事实系统保存可追溯事实；经营决策系统保存目标、动作、观测、偏差和下一动作。",
			"高风险外部写入继续要求 Owner 审批、审计和幂等保护。",
			"delete 仅表示建议退出目标架构，不授权删除代码、数据或历史记录；实际删除必须另行审计并由 Owner 批准。",
		},
		Unknowns: []string{
			"模块分类已覆盖当前目录，但每个 adapt/migrate/rebuild 条目的业务改造仍须在后续纵向单元逐项验收。",
			"真实市场、真实发布、真实订单、最终利润和现金状态需由外部事实与对账确认。",
		},
	}
}

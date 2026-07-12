# 外部基准：经营决策循环与工程判定边界

> 日期：2026-07-12
> 范围：只研究外部方法与可工程化判定标准；不评价现有代码实现
> 证据规则：`actual` 表示直接见于所引一手/权威资料；`quoted` 表示来源自身的产品或方法声明；`inferred` 表示本文据此提出的凌镜设计判断；`unknown` 表示来源不足以确认。

## 结论

`inferred`：凌镜至少应区分三个层次，不能都叫“实验”或“闭环”。

1. **事实链**：事件按对象和时间被可靠记录，例如商品、价格、订单、退款、结算和利润。它回答“发生了什么”，但没有要求系统根据结果改变下一步动作。
2. **反馈循环**：预先存在目标、可改变的动作、动作进入真实环境、结果观测、目标偏差判断、据此生成下一动作，并且下一动作确实再次执行。它回答“结果怎样改变了后续行动”。
3. **因果实验**：除反馈循环外，还有明确的因果问题、干预与对照、分配机制、预先指标/窗口，以及对混杂、随机误差和假设的处理。它才有资格回答“这项改变是否导致了结果变化”。

因此，“下单 → 付款 → 签收 → 售后 → 结算 → 利润”即使全部真实且对账完成，仍只是一条完整事实链。只有利润或其他观测按照预先规则改变下一次价格、预算、商品或渠道动作，并再次作用于市场，才形成反馈循环；只有进一步满足可识别的比较设计，才可称为因果实验。

## 1. OODA：行动后的环境变化必须重新进入观察与定向

### 来源事实

- `actual`：John Boyd 的原始讲稿合集 *A Discourse on Winning and Losing* 由 Air University Press 收录；其中 OODA 草图包含 Observe、Orient、Decide、Act，并从行动/环境结果回到观察，同时含多条内部反馈。Boyd 将 **Orientation** 展开为文化传统、遗传遗产、新信息、既往经验及分析/综合，而不是简单的“看数据”。[Air University Press：A Discourse on Winning and Losing](https://www.airuniversity.af.edu/Portals/10/AUPress/Books/B_0151_Boyd_Discourse_Winning_Losing.PDF)
- `actual`：美国海军陆战队教材把 Boyd Cycle 写成 Observe、Orient、Decide、Act，并明确它是基于时间的决策概念。[USMC：Decision Making](https://www.trngcmd.marines.mil/Portals/207/Docs/TBS/B2B0237XQ%20Decision%20Making.pdf?ver=2017-01-27-145646-543)

### 适用条件与误用

- `inferred`：适合动态、对手和环境会变化、必须反复重估的经营决策，例如广告竞价、竞争价格和库存调度。
- `inferred`：只有 O-O-D-A 状态字段并不构成 OODA；必须能证明行动进入环境，新的外部信息回流，并改变定向或后续决定。
- `inferred`：OODA 是决策与适应框架，不自带随机对照或因果识别。把“行动后看到了一个订单”解释成“行动导致订单”，是常见误用。

## 2. PDSA：预测、研究差异、修改理论并再次实施

### 来源事实

- `actual`：Deming Institute 将 PDSA 定义为 Plan–Do–Study–Act。Plan 包括目标、理论和成功指标；Study 要监测结果、检验计划/理论；Act 使用学习调整目标、方法或理论，并进入下一轮。[W. Edwards Deming Institute：PDSA Cycle](https://deming.org/explore/pdsa/)
- `actual`：该机构明确指出 Deming 强调的是 PDSA 而非 PDCA；“Study”强调预测与实际结果的比较及理论修订，而不只是检查计划成功或失败。[W. Edwards Deming Institute：PDSA 与 PDCA](https://deming.org/about-us/f-a-qs/)
- `actual`：Deming Institute 的 Theory of Knowledge 资料强调先做预测，再用实验结果检验理解；数据本身不自动成为知识。[W. Edwards Deming Institute：Theory of Knowledge](https://deming.org/theory-of-knowledge/)

### 适用条件与误用

- `inferred`：适合可小规模改变、可重复运行、能稳定测量的经营过程改进。
- `inferred`：若 Plan 没有理论、预测和成功指标，Study 只是事后讲故事，Act 没有落实为下一轮动作，则只是四阶段工作流。
- `inferred`：把交易终局命名为 Act，或把“已对账”命名为 Study，并不会形成 PDSA。

## 3. Build–Measure–Learn：目标是验证学习，不是完成一次销售

### 来源事实

- `actual`：Eric Ries 的 *The Lean Startup* 将 Build–Measure–Learn 作为反馈循环，并将核心决策表达为 pivot or persevere；原始书籍是该方法的首要来源。[Eric Ries：The Lean Startup（出版社页面）](https://www.penguinrandomhouse.com/books/210088/the-lean-startup-by-eric-ries/)
- `actual`：Ries 本人撰写的 *Progress Equals Validated Learning* 指出，收入本身不是早期创业的充分目标；“进展”需要用对客户认识的验证学习衡量。[O’Reilly：Progress Equals Validated Learning，作者 Eric Ries](https://www.oreilly.com/library/view/do-more-faster/9781119699187/c38.xhtml)
- `actual`：Ries 为 *Lean Analytics* 所写前言警告 vanity metrics（看起来漂亮但误导行动的指标），并把创新核算、数学和指标视为方法基础。[O’Reilly：Lean Analytics Foreword，作者 Eric Ries](https://www.oreilly.com/library/view/lean-analytics/9781449335687/pr03.html)

### 适用条件与误用

- `inferred`：适合在高度不确定下检验关键商业假设；应在构建前写明最危险假设、最小测试、决策阈值及 pivot/persevere 的后续动作。
- `inferred`：一次成交只证明成交发生；如果没有说明它区分了哪两个假设、排除了什么替代解释、触发什么下一动作，就不是 validated learning。
- `inferred`：Build、Measure、Learn 三个页面或状态并非循环。只 Build–Measure 而没有可审计的 Learn→下一 Build，是最典型误用。

## 4. 因果推断与实验设计：相关事实不等于干预效果

### 来源事实

- `actual`：Hernán 与 Robins 的权威教材要求明确写出因果问题，并区分数据与不可验证假设；因果效应是不同干预下结果分布的比较，不能简化为数据分析配方。[Harvard：Causal Inference—What If](https://www.hsph.harvard.edu/miguel-hernan/wp-content/uploads/sites/1268/2024/04/hernanrobins_WhatIf_26apr24.pdf)
- `actual`：NIST 的工程统计手册把设定目标、选择过程变量和水平、选择设计列为实验设计步骤。[NIST：Choosing an experimental design](https://www.itl.nist.gov/div898/handbook/pri/section3/pri3.htm)
- `actual`：NIST 指出随机化对结论正确、无歧义和可辩护很重要；重复可估计随机误差；区组用于控制重要干扰因素。[NIST：DOE glossary](https://www.itl.nist.gov/div898/handbook/pri/section7/pri7.htm)；[NIST：Randomized block designs](https://www.itl.nist.gov/div898/handbook/pri/section3/pri332.htm)

### 适用条件与误用

- `inferred`：适合声称“改价格/素材/预算导致转化或利润变化”的场景。最低要求是处理、对照、分配、结果和时间窗清楚；不能随机时也必须声明识别假设与主要混杂。
- `inferred`：同一个 `experiment_id` 关联行动和订单只建立追踪关系，不建立反事实比较，因此不能证明因果。
- `inferred`：事后挑指标、同时改多个变量、只看处理组前后差异、样本太小仍下确定结论，都会把噪声、季节性或其他变化误判为效果。

## 5. 电商系统中可观察的真实反馈机制

### 5.1 广告竞价：目标—观测—纠偏—再出价

- `quoted`：Google 表示 Smart Bidding 在每次拍卖中根据情境信号预测转化/价值，并依据广告主设置的 CPA 或 ROAS 目标调整出价；例如表现低于目标 ROAS 时可能降低出价，直到接近目标。[Google Ads：How Google Ads calculates bids](https://support.google.com/google-ads/answer/10966879?hl=en)
- `actual`：Google 要求使用 Smart Bidding 前启用转化跟踪。[Google Ads：Set up Smart Bidding](https://support.google.com/google-ads/answer/10893605?hl=en)
- `inferred`：这满足工程反馈的结构：目标（CPA/ROAS）、控制变量（bid）、环境作用（auction）、观测（conversion/value）、误差响应（调高/调低 bid）、再次拍卖。它仍不保证广告主所记录的“转化价值”是真实净利润，也不自动证明因果。

### 5.2 广告实验：并行控制与处理，不是简单前后对比

- `actual`：Google Ads API 的实验流程把流量分到 control 与 treatment，比较指标，再选择结束、推广或独立运行处理方案。[Google Ads API：Experiments overview](https://developers.google.com/google-ads/api/docs/experiments/overview)
- `actual`：实验报告同时提供处理组/对照组指标及 p-value；官方建议考虑学习期、转化延迟、周周期，并通常使用 50/50 分流。[Google Ads API：Report on experiments](https://developers.google.com/google-ads/api/docs/experiments/reporting)
- `inferred`：这是比“修改广告后订单增加”更接近因果实验的机制；但平台统计显著性仍不能替代业务利润、退款和现金的外部对账。

### 5.3 自动定价：可形成规则反馈，但不等于学习或因果

- `quoted`：Amazon Automate Pricing 允许基于 Featured Offer、最低价、外部价格或指定时期销量变化调整价格；卖家设置规则、SKU 及最低/最高价后，系统动态改价。[Amazon：Automate Pricing](https://sell.amazon.com/tools/automate-pricing)
- `inferred`：若“观测竞争价/销量 → 按规则改价 → 新价格再次进入市场”持续运行，它是规则型反馈循环；最低价/最高价是控制边界。
- `inferred`：它未必是学习系统，更不是因果实验。销量同时受流量、季节、库存、评价等影响；仅因规则多次调价就宣称找到了价格因果效应，属于误用。

## 6. 凌镜的最小严格判定标准

### 6.1 只能叫“事实链”的条件

满足以下任一项，就不得称为反馈循环：

- 没有预先明确目标指标和边界；
- 没有可执行、可记录版本的经营动作；
- 动作没有可验证地进入真实市场；
- 只有订单/退款/利润等结果记录，没有与目标比较；
- 没有从观测生成下一动作的明确规则或 Owner 决定；
- 下一动作没有实际执行，只停留在建议、状态或终局标签；
- 只靠对象关联或时间先后声称因果。

### 6.2 可称“反馈循环”的七项必要证据

以下七项必须在同一决策循环实例中全部存在：

| 必要项 | 最低工程证据 |
|---|---|
| 1. 目标 | 指标、目标/阈值、时间窗、守护边界在动作前冻结 |
| 2. 动作变量 | 明确本轮允许改变什么，记录旧值、新值和版本 |
| 3. 外部作用 | 平台/市场接受或实际展示/执行的可追溯凭证 |
| 4. 结果观测 | 来源、观察时间、对象、归因窗口和完整性状态 |
| 5. 偏差判断 | 实际值与目标值按预定规则比较，区分噪声/延迟 |
| 6. 反馈决定 | 由规则或 Owner 形成 stop/hold/adjust/scale 及理由 |
| 7. 再执行 | 决定转成下一版本动作并再次进入外部环境 |

`inferred`：第 7 项是区分“有反馈建议的事实链”和“已经闭合的反馈循环”的关键。一次从 1 到 6 的运行只能记为 `feedback_decision_ready`，不能记为 `loop_closed`。

### 6.3 可称“因果实验”的附加门槛

除上述七项外，还必须有：明确的反事实问题；处理与对照；分配机制（优先随机）；预先固定的主要指标、样本/停止规则和分析窗口；干扰因素与并发变化记录；效果估计及不确定性；未满足的识别假设显式标为 `unknown`。否则只能说“反馈循环观察到变化”，不能说“该动作导致变化”。

## 7. 推荐的最小数据对象（不是建设授权）

`inferred`：若未来决定工程化，最小对象不应以“交易阶段”命名，而应围绕一个可重复的 `decision_cycle`：

```text
decision_cycle
  objective_snapshot
  action_version
  external_execution_receipt
  observation_window + observations
  deviation_assessment
  feedback_decision
  next_action_version + next_execution_receipt
  causal_design? + assumptions? + comparison_result?
```

交易、订单、结算和利润应作为 `observations` 或守护边界的证据来源，而不是用它们的终局状态冒充循环。是否适合凌镜、应控制哪个变量、数据是否足够及时可靠，仍为 `unknown`，必须从 Owner 的一个具体、重复经营决策中验证。

## 8. 最终裁决表

| 声明 | 允许的名称 | 证据等级 |
|---|---|---|
| 真实记录了商品、订单、售后、结算和利润 | 经营事实链/事实案卷 | `actual`（若来源与对账真实） |
| 结果触发下一动作建议，但尚未执行 | 反馈决定已形成 | `actual`，但循环未闭合 |
| 下一动作已按决定再次进入市场 | 经营反馈循环的一次闭合 | `actual`（仅指循环执行） |
| 处理组优于对照组且设计/假设充分 | 因果实验结果 | `actual` + 明确不确定性 |
| 单次成交、前后变化或对象关联证明动作有效 | 不允许 | `inferred`，因果仍 `unknown` |

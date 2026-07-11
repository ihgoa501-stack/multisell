# Hoopa Valley × 2021 Monument Fire 家庭洁净空气残余障碍：独立反证

- 研究日期：2026-07-11
- 观察截止：2026-07-11（Asia/Shanghai）
- 独立采集者：`agent:wildfire_local_event_falsifier`
- 研究性质：先读取指定侦察文件，随后独立回到一手来源复核；零花费、零权限申请、零外部写入
- 被反证命题：2021 Monument Fire 期间，Hoopa Valley 多户家庭反复遭遇可由消费者控制、非结构性的噪声／滤芯获取与成本障碍，且该障碍未被商用 PAC、HVAC、DIY 或公共洁净空间替代。
- 最终决定：**`rejected`**

## 1. 判定规则与停止条件

命题只有在同一已结束事件中同时满足以下四项，才可 `survives_falsification`：

1. 障碍在多户家庭中被实际观察到，而不只是意向、回忆或跨事件推断；
2. 家庭可在不依赖结构改造、公共机构持续供给或研究人员介入的情况下控制该障碍；
3. 障碍不是住房/HVAC、偏远供应、收入或公共卫生资源等结构条件的替代表述；
4. 一手证据足以排除商用 PAC、既有 HVAC、DIY 调速/移房间及公共洁净空间已经完成主要替代。

**停止条件：**只要关键的事件—干预时间链不重叠，或已有替代已经直接吸收所称障碍且没有事件内残余证据，即停止支持性推断并判 `rejected`。本案两个停止条件都已触发：Monument Fire 烟只覆盖基线期，而全部家庭在自由选择期选择商用 PAC；公共洁净空间实际使用又完全未测。缺失关键事件证据不保留为 `evidence_missing`，而按举证责任判定命题失败。

## 2. 一手证据审计

### 2.1 事件时间与室内 PM2.5：干预没有检验 Monument Fire 重烟期

研究作者明确写明，Monument Fire 烟只出现在 wildfire study 的 Initial phase；雨水使烟在 2021-09-26 前清除，室外日均 PM2.5 从基线的 9.1 μg/m³ 降至 Sensor Display phase 的 2.1 μg/m³。作者在限制部分再次说明：野火烟只存在于 Initial phase，所有 intervention phases 处于相对良好空气质量，因此结果只能解释一般渗入 PM2.5，不能解释 Monument Fire 烟事件中的干预效果。[Prathibha et al., 2024, pp. 7, 10](https://www.osti.gov/servlets/purl/2580861)（事件观察：2021-09 至 2021-10；来源访问：2026-07-11）

完整室内外 PM2.5 配对只有 6 户；两户传感器内存失败。模型比较的是设备运行与不运行的 10 分钟窗口，而非同一重烟事件中的随机或同期对照。野火研究中 DIY 运行与总室内 PM2.5 降低 7.0%（95% CI 2.5%–11.3%）相关，商用 PAC 对总室内 PM2.5 的估计不显著；二者与渗入 PM2.5 的降低相关，但这些关联主要发生在烟已显著减弱的干预期。[Prathibha et al., 2024, pp. 7–8](https://www.osti.gov/servlets/purl/2580861)（测量观察：2021-09 至 2021-10；来源访问：2026-07-11）

**反证结果：**室内 PM2.5 证据证明设备可在一般条件下影响颗粒物，不证明 Monument Fire 期间存在一个反复、未解决的家庭残余障碍。

### 2.2 实际采用与设备使用：受控发放和提醒不能代表自然采用

项目按固定顺序免费提供 DIY PAC、商用 PAC 和实时显示器，要求前两个设备阶段每天至少运行 8 小时；研究人员每阶段入户设置设备并在阶段末访谈。阶段未随机，所有住宅同时按相同顺序推进。[Prathibha et al., 2024, pp. 3, 10](https://www.osti.gov/servlets/purl/2580861)（干预观察：2021-09 至 2021-10；来源访问：2026-07-11）

即便在这种强支持下，商用 PAC 日均运行仍比 DIY 多 3–4 小时；家庭更常整天不开 DIY，接近一半家庭把 DIY 调到最低档。自由选择期全部家庭选择商用 PAC，5/8 户把更安静列为主要原因；只有一户同时使用 DIY 和商用机。[Prathibha et al., 2024, pp. 6–7](https://www.osti.gov/servlets/purl/2580861)（设备使用观察：2021-09 至 2021-10；来源访问：2026-07-11）

Turner 等人同一项目的健康研究也明确称研究为 feasibility pilot，无法作因果结论，DIY 使用低，主要障碍为运行噪声；每阶段仍是要求设备每天至少运行 8 小时并配合电话访谈。[Turner et al., 2024](https://pmc.ncbi.nlm.nih.gov/articles/PMC10875335/)（项目观察：2021-09 至 2021-10；来源访问：2026-07-11）

**反证结果：**观察到的是免费、指令性试用中的相对使用差异，不是 Monument Fire 重烟期的自然持续采用。

### 2.3 噪声：跨户重复，但已被现有商用 PAC 直接替代

DIY 高档噪声约 68 dBA，商用 PAC 高档约 51 dBA，二者 CADR 接近（110 与 119）。7/8 户至少一名参与者对 DIY 噪声不满；全部家庭最终选择商用 PAC，5/8 户主要因为商用机更安静。[Prathibha et al., 2024, pp. 3, 6](https://www.osti.gov/servlets/purl/2580861)（设备与访谈观察：2021-09 至 2021-10；来源访问：2026-07-11）

噪声可由家庭降档、换房间或改用更安静的商用 PAC 控制；研究里“降档”和“改用商用 PAC”都已真实发生。75% 的住宅还表示会在不同房间、烟天或更暖天气使用 DIY，这只是未来意向，不是该事件内采用。[Prathibha et al., 2024, p. 6](https://www.osti.gov/servlets/purl/2580861)（访谈观察：2021-09 至 2022-03；来源访问：2026-07-11）

**反证结果：**噪声满足“小样本跨户重复”，却不满足“未被替代的残余障碍”。它主要是所测试单滤芯箱式风扇设计的属性，而同研究中的现成商用 PAC 已直接解决。

### 2.4 滤芯获取、成本与更换：有回忆性障碍，没有事件内更换链

论文称无设备家庭把设备成本、替换滤芯可得性与成本列为获取障碍；已有设备家庭回忆以滤芯可得性/成本、忘记及噪声解释过去烟事件中不开机。但论文没有公布 wildfire-study 中每一原因对应的家庭数，也没有给出 Monument Fire 期间的购买、缺货、价格、路程、耗尽日期、实际更换次数或因未更换而停机的逐户记录。[Prathibha et al., 2024, pp. 5–6](https://www.osti.gov/servlets/purl/2580861)（回忆访谈观察：2021-09 至 2021-10；来源访问：2026-07-11）

研究材料价格为 DIY 约 45 美元、6 片替换滤芯约 50 美元；商用 PAC 约 123 美元、单个 HEPA replacement kit 约 47 美元。研究团队在两项研究之间更换滤芯，并在阶段间按需更换，因此家庭没有独立承担和展示事件内滤芯更换行为。[Prathibha et al., 2024, pp. 3, 10](https://www.osti.gov/servlets/purl/2580861)（价格基准：研究时；维护观察：2021-09 至 2022-03；来源访问：2026-07-11）

EPA 后续总结把“支持滤芯更换”和“安静、不突兀设计”明确写成公共卫生及地方空气质量机构的启示，并指出烟事件中可能需要频繁换滤芯。[US EPA, Wildland Fire Research: Reducing Exposures](https://www.epa.gov/air-research/wildland-fire-research-reducing-exposures)（页面更新：2026-07-06；来源访问：2026-07-11）

**反证结果：**滤芯障碍没有被证明为 Monument Fire 内跨户重复的消费者行为问题；现有证据更符合供应可达性、低收入与公共机构支持相互交织的结构/公共卫生问题。

### 2.5 HVAC、热与空间：关键停用字段未在该事件内成立

8 户中只有 2 户中央 HVAC 装有 PM2.5 滤芯，2 户有中央系统但没有，4 户没有中央系统；6 户主要使用窗式空调。研究样本太小，作者称无法识别中央系统 PM2.5 滤芯等建筑特征与行为的细粒度关联。[Prathibha et al., 2024, pp. 5, 10](https://www.osti.gov/servlets/purl/2580861)（住宅观察：2021-09 至 2021-10；来源访问：2026-07-11）

野火研究没有报告热导致停用。明确的“冷风过强”来自后续冬季 wood stove study 的 4/11 户，不能移植为 Monument Fire 期间证据。较小体积只是选择商用 PAC 的其他理由之一，论文没有报告 wildfire study 中因空间不足而停用的家庭数。[Prathibha et al., 2024, p. 6](https://www.osti.gov/servlets/purl/2580861)（冷风观察：2022-01 至 2022-03；来源访问：2026-07-11）

**反证结果：**HVAC 是住房结构差异，野火期热停用缺失，空间仅为偏好而非停用事实；三者均不能支持该次事件的非结构性残余障碍。

### 2.6 商用 PAC、既有设备与公共洁净空间替代

野火研究开始前已有 4/8 户拥有 1–6 台商用 PAC；只有一户自行购买，其余由地方健康中心或社区组织发放。研究没有给私人既有 PAC 安装功率记录；作者明确指出，如果私人 PAC 在研究设备关闭时运行，效果会偏低估，若同时运行则会偏高估。[Prathibha et al., 2024, pp. 5, 10](https://www.osti.gov/servlets/purl/2580861)（既有设备观察：研究前及 2021-09 至 2021-10；来源访问：2026-07-11）

EPA 的 ASPIRE 项目页面说明 Hoopa 伙伴关系的目标包括学习室内空气质量并在社区创建 cleaner air spaces，2019-12 至 2022-03 在 13 栋建筑连续监测；另有面向公共/商业建筑的洁净空气指南与研究。[US EPA, ASPIRE](https://www.epa.gov/air-research/wf-aspire)（项目观察：2019-12 至 2022-03；来源访问：2026-07-11）

但任何一手家庭论文都没有记录这 8 户是否去过公共洁净空间、何时去、停留多久、距离多远或是否以其替代居家设备。因此无法排除公共空间替代；同时，既有公共/社区 PAC 发放已经实际替代了多数已拥有设备家庭的私人购买。

**反证结果：**商用 PAC 替代已直接观察；公共发放已直接观察；公共洁净空间替代无法排除。命题要求“未被替代”，举证失败。

## 3. 四项命题逐项裁决

| 必须成立的命题 | 一手证据 | 裁决 |
|---|---|---|
| (a) 多户重复 | 噪声 7/8 重复；滤芯原因未公布 wildfire-study 逐项户数；事件内更换/停机链缺失 | **不成立**：只有特定 DIY 噪声达到重复门槛 |
| (b) 消费者可控 | 家庭可降档、换房间或改用商用 PAC；滤芯可达性与公共供给交织 | **部分可控但不构成残余命题** |
| (c) 非结构性 | 4/8 无中央系统；滤芯供应、成本、低收入与公共支持未被拆分 | **不成立** |
| (d) 未被替代 | 全部家庭自由选择商用 PAC；4/8 已有 PAC，多数由公共/社区发放；公共空间使用未测 | **不成立** |

## 4. 最终决定

**`rejected`**。

拒绝的决定性原因不是“样本小”本身，而是命题所要求的事件事实链不存在：Monument Fire 烟只覆盖没有研究设备干预的基线期，干预与行为数据主要来自烟已清除或显著减弱后的阶段。噪声虽在 7/8 户重复，却由同一试验中的商用 PAC 直接替代；滤芯障碍只有未量化的回忆性陈述，且研究团队承担了更换；HVAC、热、空间和公共洁净空间均没有形成该事件内逐户停用链。已有 PAC 与公共/社区发放还表明主要替代和责任边界尚未被排除。

本决定不涉及商品、平台、渠道选择，也不推断付费需求。

## 5. 结构化导入摘要

```json
{
  "problem_key": "us-ca-hoopa-2021-monument-fire-household-clean-air",
  "run_type": "falsifier_result",
  "collector": "agent:wildfire_local_event_falsifier",
  "observed_at": "2026-07-11",
  "decision": "rejected",
  "burden_of_proof": "critical same-event evidence must be present; missing event evidence counts against the hypothesis",
  "stop_condition_triggered": [
    "Monument Fire smoke occurred only in the Initial phase, before study PAC interventions",
    "all households chose the commercial PAC in the free-choice phase",
    "public cleaner-air-space substitution was not measured and therefore cannot be excluded"
  ],
  "event_timing": {
    "wildfire_study": "2021-09 to 2021-10",
    "monument_smoke_phase": "Initial phase only",
    "local_smoke_cleared_by": "2021-09-26",
    "initial_outdoor_pm25_ug_m3": 9.1,
    "sensor_display_outdoor_pm25_ug_m3": 2.1
  },
  "household_evidence": {
    "enrolled_homes": 8,
    "complete_indoor_outdoor_pm25_homes": 6,
    "prior_commercial_pac_homes": 4,
    "self_purchased_prior_pac_homes": 1,
    "diy_noise_frustration_homes": 7,
    "commercial_choice_homes": 8,
    "quieter_operation_as_primary_choice_homes": 5
  },
  "adoption_audit": "free provision, required >=8 h/day use, repeated researcher setup and interviews; not natural adoption",
  "indoor_pm25_audit": "time-resolved associations exist, but intervention phases did not overlap Monument Fire smoke and only six homes had complete paired data",
  "device_use_audit": "commercial PAC used 3-4 h/day more; DIY had more zero-use days; nearly half used lowest DIY speed",
  "filter_replacement_audit": "no event-level household counts, purchases, stockouts, replacement dates or discontinuation chain; study staff replaced filters between studies and as needed between phases",
  "noise_heat_space_audit": "noise repeated but commercially substituted; cooling discontinuation belongs to winter wood-smoke study; wildfire heat and space discontinuation not observed",
  "public_space_audit": "community cleaner-air-space program context exists, but household use, access and substitution were not measured",
  "structural_assessment": "filter access/cost and HVAC constraints cannot be separated from remote supply, housing and public-health support",
  "substitution_assessment": "commercial PAC and public/community device distribution already observed; public-space substitution cannot be excluded",
  "paid_demand_inference": null,
  "product_platform_channel_inference": null,
  "primary_sources": [
    "https://www.osti.gov/servlets/purl/2580861",
    "https://pmc.ncbi.nlm.nih.gov/articles/PMC10875335/",
    "https://www.epa.gov/air-research/wf-aspire",
    "https://www.epa.gov/air-research/wildland-fire-research-reducing-exposures"
  ]
}
```

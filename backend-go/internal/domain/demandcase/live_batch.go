package demandcase

import (
	"context"
	"encoding/json"
	"time"
)

type ReviewedBatchOutcome struct {
	Cards                  []OwnerDecisionCard `json:"cards"`
	PermissionCandidateIDs []int64             `json:"permission_candidate_ids"`
	HeldCaseIDs            []int64             `json:"held_case_ids"`
	RejectedCaseIDs        []int64             `json:"rejected_case_ids"`
}

func (s *Service) ImportReviewedMarketPermissionBatch(ctx context.Context, ownerID int64) (*ReviewedBatchOutcome, error) {
	observed := time.Date(2026, 7, 11, 0, 0, 0, 0, time.FixedZone("Asia/Shanghai", 8*3600))
	out := &ReviewedBatchOutcome{}
	type candidate struct {
		key, region, consumer, scenario, channel, stop, scoutURL, scoutTitle, counterURL, counterTitle string
		access                                                                                         []ResearchFinding
		reject                                                                                         bool
	}
	items := []candidate{
		{key: "GB-ENG-PRIVATE-RENTER-DAMP-AMZUK-20260711", region: "GB-ENG", consumer: "私人租赁住房中调查记录到严重潮湿/冷凝的家庭", scenario: "管理严重室内潮湿/冷凝的日常影响；商品未确定", channel: "amazon.co.uk:A1F83G8C2ARO7P", stop: "无访问权限或无匹配 niche；购买信号弱；杀霉监管边界或关键成本未知时停止", scoutURL: "https://www.gov.uk/government/statistics/english-housing-survey-2024-to-2025-home-insulation-fact-sheet/english-housing-survey-2024-to-2025-home-insulation-fact-sheet", scoutTitle: "England 住房调查观察到私人租赁住宅严重潮湿/冷凝问题；这只证明场景存在", counterURL: "https://www.hse.gov.uk/biocides/products/market.htm", counterTitle: "潮湿根因可能是结构和通风，消费品未必解决；杀霉宣称可能进入生物杀灭监管", access: []ResearchFinding{{FieldName: "niche_search_purchase_returns_competition", AccessStatus: AccessRequiresOwner, RequiredScope: "Seller Central Product Opportunity Explorer read-only (UK marketplace)", DecisionPurpose: "判断该场景是否存在可观察的搜索到购买行为及退货竞争反证", RefusalImpact: "保持证据不足并停止 UK 渠道预检"}, {FieldName: "orders_refunds_settlement", AccessStatus: AccessRequiresTransaction, DecisionPurpose: "真实实验后核验付款、退款与结算", RefusalImpact: "当前不请求；没有交易时保持未知"}}},
		{key: "US-65PLUS-MOBILITY-DAILY-AID-AMZUS-20260711", region: "US", consumer: "65岁以上且报告严重行走或爬楼困难的社区成年人", scenario: "非医疗的日常取放、整理或轻量辅助；商品未确定", channel: "amazon.com:ATVPDKIKX0DER", stop: "无匹配 niche；消费者与购买者含混；安全边界或关键成本未知时停止", scoutURL: "https://www.cdc.gov/dhds/about/overview.html", scoutTitle: "CDC 官方数据可观察老年人行动困难；这不证明任何商品购买", counterURL: "https://www.cpsc.gov/Business--Manufacturing/Online-Sellers-Safety-Guide", counterTitle: "行动困难不等于购买；承重、防跌或医疗边界会显著增加伤害与合规风险", access: []ResearchFinding{{FieldName: "niche_search_purchase_returns_competition", AccessStatus: AccessRequiresOwner, RequiredScope: "Seller Central Product Opportunity Explorer read-only (US marketplace)", DecisionPurpose: "按功能场景词验证搜索、购买、退货和竞争，而非泛化老人用品", RefusalImpact: "保持证据不足并停止 US 渠道预检"}, {FieldName: "sp_api_automation", AccessStatus: AccessRequiresOwner, RequiredScope: "optional private SP-API non-restricted read roles", DecisionPurpose: "仅在人工预检存活后自动保存只读字段", RefusalImpact: "继续人工只读预检，不影响当前裁决"}}},
		{key: "JP-65PLUS-UNSPECIFIED-AID-AMZJP-20260711", region: "JP", consumer: "65岁以上人口（具体需求群体尚不可观察）", scenario: "未明确的非医疗日常辅助", channel: "amazon.co.jp:A1VC38T7YXB528", stop: "找不到日本官方具体功能困难或生活场景证据时停止", scoutURL: "https://www.stat.go.jp/english/data/jinsui/2024np/index.htm", scoutTitle: "日本官方只证明65岁以上人口规模，不能推出具体需求", counterURL: "https://developer-docs.amazon.com/sp-api/lang-US/docs/marketplace-ids", counterTitle: "人口老龄化不是需求或购买信号；当前没有具体场景证据", reject: true, access: []ResearchFinding{{FieldName: "marketplace_data", AccessStatus: AccessUnknown, DecisionPurpose: "等待具体需求场景成立后再定义字段", RefusalImpact: "不申请任何权限并暂停该候选"}}},
	}
	for _, x := range items {
		base := ResearchResult{BatchKey: "live-market-permission-2026-07-11", Region: x.region, Consumer: x.consumer, NeedScenario: x.scenario, SalesChannel: x.channel, StopCondition: x.stop, CollectedAt: observed}
		runs := []ResearchResult{mergeResearch(base, x.key+"-scout", RunScout, x.scoutURL, []ResearchFinding{{Dimension: DimensionDemand, Kind: EvidenceSupport, TruthStatus: TruthQuoted, Title: x.scoutTitle}}), mergeResearch(base, x.key+"-falsifier", RunFalsifier, x.counterURL, []ResearchFinding{{Dimension: DimensionDemand, Kind: EvidenceCounter, TruthStatus: TruthQuoted, Title: x.counterTitle, Fatal: x.reject}}), mergeResearch(base, x.key+"-reality", RunDataReality, "https://sell.amazon.com/tools/product-opportunity-explorer/", x.access)}
		runs[0].Collector = "agent:live_market_primary_research"
		runs[1].Collector = "agent:independent_market_falsifier"
		runs[2].Collector = "agent:live_market_primary_research:data_reality"
		var c *DemandCase
		for _, run := range runs {
			item, err := s.ImportResearchResult(ctx, ownerID, run)
			if err != nil {
				return nil, err
			}
			c = item
		}
		if _, err := s.Evaluate(ctx, c.ID, ownerID); err != nil {
			return nil, err
		}
		card, err := s.DecisionCard(ctx, c.ID, ownerID)
		if err != nil {
			return nil, err
		}
		out.Cards = append(out.Cards, *card)
		permissions, err := s.PermissionRequests(ctx, c.ID, ownerID)
		if err != nil {
			return nil, err
		}
		if card.Verdict == VerdictRejected {
			out.RejectedCaseIDs = append(out.RejectedCaseIDs, c.ID)
		} else if len(permissions) > 0 {
			out.PermissionCandidateIDs = append(out.PermissionCandidateIDs, c.ID)
		} else {
			out.HeldCaseIDs = append(out.HeldCaseIDs, c.ID)
		}
	}
	return out, nil
}

func mergeResearch(base ResearchResult, runID, runType, source string, findings []ResearchFinding) ResearchResult {
	base.RunID = runID
	base.RunType = runType
	base.SourceURI = source
	base.Findings = findings
	raw, _ := json.Marshal(map[string]any{"run_id": runID, "source": source, "findings": findings})
	base.RawPayload = raw
	base.RawSHA256 = payloadHash(raw)
	return base
}

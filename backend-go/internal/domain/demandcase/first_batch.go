package demandcase

import (
	"context"
	"encoding/json"
	"time"
)

// RunFirstPublicResearchBatch imports the repository-reviewed public-source baseline.
// It deliberately ends at evidence_missing: public sources cannot prove paid demand.
func (s *Service) RunFirstPublicResearchBatch(ctx context.Context, ownerID int64) ([]OwnerDecisionCard, error) {
	observed := time.Date(2026, 7, 11, 0, 0, 0, 0, time.UTC)
	base := ResearchResult{BatchKey: "public-market-baseline-2026-07-11", Region: "俄罗斯", Consumer: "Ozon 平台可观察消费者", NeedScenario: "跨境商品需求尚待账号内数据验证", SalesChannel: "Ozon", TargetLocale: "ru-RU", StopCondition: "无法取得账号内需求、费用、履约与售后字段时停止", CollectedAt: observed}
	runs := []ResearchResult{
		mergeResearch(base, "public-scout-20260711", RunScout, "https://docs.ozon.com/global/en/analytics/analytics-and-metrics/analytics-tools/", []ResearchFinding{{Dimension: DimensionDemand, Kind: EvidenceSupport, TruthStatus: TruthQuoted, Title: "Ozon 官方提供卖家侧需求与竞争分析工具，但公开文档不提供具体市场需求结论"}, {Dimension: DimensionCompetition, Kind: EvidenceSupport, TruthStatus: TruthQuoted, Title: "官方分析能力可用于后续读取竞争指标，实际字段与账号权限待验证"}}),
		mergeResearch(base, "public-falsifier-20260711", RunFalsifier, "https://support.google.com/trends/answer/4365533?hl=en", []ResearchFinding{{Dimension: DimensionDemand, Kind: EvidenceCounter, TruthStatus: TruthQuoted, Title: "公开搜索热度是抽样归一化相对兴趣，不能证明目标消费者付款"}}),
		mergeResearch(base, "public-data-reality-20260711", RunDataReality, "https://docs.ozon.ru/api/seller/", []ResearchFinding{{Dimension: DimensionAcquisition, Kind: EvidenceConflict, TruthStatus: TruthUnknown, Title: "获客字段需要账号实测"}, {Dimension: DimensionFulfillment, Kind: EvidenceConflict, TruthStatus: TruthUnknown, Title: "SKU 履约可用性需要账号和商品参数"}, {Dimension: DimensionCompliance, Kind: EvidenceConflict, TruthStatus: TruthUnknown, Title: "具体商品合规尚未知"}, {Dimension: DimensionPayment, Kind: EvidenceConflict, TruthStatus: TruthUnknown, Title: "付款与结算字段需 Seller API 权限"}, {Dimension: DimensionAftersales, Kind: EvidenceConflict, TruthStatus: TruthUnknown, Title: "退货争议窗口需账号数据"}, {Dimension: DimensionProfit, Kind: EvidenceConflict, TruthStatus: TruthUnknown, Title: "完整费用与最终利润尚不可验证"}}),
	}
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
	return []OwnerDecisionCard{*card}, nil
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

package demandcase

import (
	"context"
	"fmt"
	"time"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

const ReviewedProblemBatchKey = "problem-first-2026-07-11"

type ReviewedProblemBatchOutcome struct {
	BatchKey         string         `json:"batch_key"`
	Problems         []ProblemCase  `json:"problems"`
	StatusCounts     map[string]int `json:"status_counts"`
	PaidDemand       string         `json:"paid_demand_status"`
	SelectedItems    int            `json:"selected_items"`
	SelectedChannels int            `json:"selected_channels"`
}

type reviewedProblem struct {
	caseData                                 ProblemCase
	supportTitle, supportURI, supportPayload string
	counterTitle, counterURI, counterPayload string
}

func reviewedProblems() []reviewedProblem {
	return []reviewedProblem{
		{caseData: ProblemCase{ProblemKey: "US-WILDFIRE-SMOKE-HOME-CLEAN-AIR-20260711", Region: "US", ObservablePopulation: "真实野火烟霾事件中受室内 PM2.5、住房渗透与安全制冷条件共同影响的家庭", ProblemScenario: "烟霾期间家庭需要降低室内颗粒暴露，但现有商业净化、HVAC、DIY 与公共洁净空间是否仍留下重复障碍尚未观察", CurrentWorkaround: "商业净化器、MERV 滤芯、DIY 净化器、洁净空气房间或公共洁净空气空间", Responsibility: ResponsibilityShared, ProductSolvability: SolvabilityPartial, HarmRisk: HarmMedium, NextMinimumEvidence: "选择一个已结束烟霾事件和地方公共项目，记录使用率、室内 PM2.5、设备、滤芯、噪声、热、空间与停用原因"}, supportTitle: "EPA wildfire smoke household mitigation guidance", supportURI: "https://www.epa.gov/emergencies-iaq/create-clean-room-protect-indoor-air-quality-during-wildfire", supportPayload: "官方资料确认具体烟霾事件中家庭存在降低室内颗粒暴露的问题，并给出洁净空气房间等补救。", counterTitle: "EPA alternatives and unresolved real-home evidence", counterURI: "https://www.epa.gov/air-research/research-diy-air-cleaners-reduce-wildfire-smoke-indoors", counterPayload: "商业净化、HVAC、DIY 和公共空间构成强替代；尚无重复残余使用障碍，真实家庭健康效果仍在研究。"},
		{caseData: ProblemCase{ProblemKey: "AU-65PLUS-HOME-SAMELEVEL-FALL-20260711", Region: "AU", ObservablePopulation: "65 岁以上、在家发生同层跌倒但具体原因尚未隔离的人群", ProblemScenario: "跌倒伤害负担存在，但医疗、药物、视力、平衡、鞋履和居家环境等原因混合", CurrentWorkaround: "健康复核、运动、视力与药物管理、居家评估、环境整改和求助计划", Responsibility: ResponsibilityUnknown, ProductSolvability: SolvabilityUnknown, HarmRisk: HarmMedium, NextMinimumEvidence: "用州级伤害监测或政府居家评估资料隔离一个非医疗、非结构、非承重场景及其采用障碍"}, supportTitle: "AIHW falls burden", supportURI: "https://www.aihw.gov.au/reports/injury/falls", supportPayload: "官方统计确认老年跌倒伤害负担，但聚合类别不能定位单一消费品可控原因。", counterTitle: "Australian Government multifactor prevention", counterURI: "https://www.health.gov.au/resources/publications/dont-fall-for-it-falls-can-be-prevented", counterPayload: "官方预防方案是健康、活动、环境与求助的组合，当前未隔离消费者可控的窄场景。"},
		{caseData: ProblemCase{ProblemKey: "US-CHILD-1TO4-FURNITURE-TIPOVER-20260711", Region: "US", ObservablePopulation: "有 1–4 岁儿童且存在家具、电视或家电倾倒风险的家庭", ProblemScenario: "通用后装防倾倒件可能降解、断裂或因墙体和安装差异造成虚假安全感", CurrentWorkaround: "符合强制稳定性规则的家具、正确锚固和官方安全教育", Responsibility: ResponsibilityManufacturer, ProductSolvability: SolvabilityStructural, HarmRisk: HarmHigh, NextMinimumEvidence: "停止通用商品线索；只有具备专业工程、测试、合规和召回能力的新命题才重新立案"}, supportTitle: "CPSC furniture tip-over guidance", supportURI: "https://www.cpsc.gov/Business--Manufacturing/Business-Education/Business-Guidance/Clothing-Storage-Units", supportPayload: "儿童家具倾倒是严重伤害问题，现行规则把新衣物储存家具稳定性放在制造商侧。", counterTitle: "CPSC recalled degrading restraint kits", counterURI: "https://www.cpsc.gov/Recalls/2026/Cranach-Hardware-Recalls-Tip-Restraint-Kits-Due-to-Tip-Over-Hazard-Manufactured-by-Cranach-Hardware", counterPayload: "监管召回证明跨境销售的塑料约束件会降解或断裂并产生致命的虚假安全感。"},
		{caseData: ProblemCase{ProblemKey: "GB-ENG-DAMP-VULNERABLE-HOUSEHOLDS-20260711", Region: "GB-ENG", ObservablePopulation: "England 存在严重潮湿住房问题的脆弱家庭", ProblemScenario: "严重潮湿可能来自失修、渗漏、通风、保温和供暖负担，消费者用品可能转移房东责任并延误修复", CurrentWorkaround: "住房检查、房东修缮、固定通风与保温整改、足够供暖", Responsibility: ResponsibilityLandlord, ProductSolvability: SolvabilityStructural, HarmRisk: HarmMedium, NextMinimumEvidence: "停止当前宽泛线索；经专业检查排除结构和房东责任的自有住房剩余场景须作为新案件"}, supportTitle: "English Housing Survey serious damp", supportURI: "https://www.gov.uk/government/statistics/chapters-for-english-housing-survey-2024-to-2025-headline-findings-on-housing-quality-and-energy-efficiency/chapter-1-housing-quality", supportPayload: "最新住房调查确认严重潮湿，并将其与失修、通风、保温和供暖负担联系。", counterTitle: "Awaab's Law landlord repair duty", counterURI: "https://www.gov.uk/government/publications/awaabs-law-guidance-for-social-landlords/awaabs-law-guidance-for-social-landlords-timeframes-for-repairs-in-the-social-rented-sector", counterPayload: "Awaab's Law 明确社会房东的调查修复时限并反对未经调查归因于租客生活方式。"},
	}
}

func (s *Service) ImportReviewedProblemBatch(ctx context.Context, ownerID int64) (*ReviewedProblemBatchOutcome, error) {
	if ownerID <= 0 {
		return nil, fmt.Errorf("owner is required")
	}
	observedAt := time.Date(2026, 7, 11, 0, 0, 0, 0, time.FixedZone("Asia/Shanghai", 8*60*60))
	out := &ReviewedProblemBatchOutcome{BatchKey: ReviewedProblemBatchKey, StatusCounts: map[string]int{}, PaidDemand: "unknown"}
	err := s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		txService := NewService(tx, s.logger)
		for _, item := range reviewedProblems() {
			desired := item.caseData
			desired.OwnerID = ownerID
			if err := tx.Clauses(clause.OnConflict{DoNothing: true}).Create(&desired).Error; err != nil {
				return err
			}
			var p ProblemCase
			if err := tx.Where("owner_id = ? AND problem_key = ?", ownerID, desired.ProblemKey).First(&p).Error; err != nil {
				return err
			}
			if !sameReviewedProblem(p, desired) {
				return fmt.Errorf("problem %s conflicts with immutable reviewed batch", desired.ProblemKey)
			}
			for _, e := range []ProblemEvidence{
				{ProblemCaseID: p.ID, Kind: EvidenceSupport, Title: item.supportTitle, SourceURI: item.supportURI, ObservedAt: observedAt, Collector: "agent:problem_first_scout", RawPayload: item.supportPayload, RawSHA256: payloadHash([]byte(item.supportPayload)), TrustedRun: true},
				{ProblemCaseID: p.ID, Kind: EvidenceCounter, Title: item.counterTitle, SourceURI: item.counterURI, ObservedAt: observedAt, Collector: "agent:problem_first_falsifier", RawPayload: item.counterPayload, RawSHA256: payloadHash([]byte(item.counterPayload)), TrustedRun: true},
			} {
				if _, err := txService.requireProblemOwner(ctx, e.ProblemCaseID, ownerID); err != nil {
					return err
				}
				if err := tx.Clauses(clause.OnConflict{DoNothing: true}).Create(&e).Error; err != nil {
					return err
				}
				var stored ProblemEvidence
				if err := tx.Where("problem_case_id = ? AND kind = ? AND collector = ? AND raw_sha256 = ?", e.ProblemCaseID, e.Kind, e.Collector, e.RawSHA256).First(&stored).Error; err != nil {
					return err
				}
				if !sameReviewedEvidence(stored, e) {
					return fmt.Errorf("evidence for problem %s conflicts with immutable reviewed batch", p.ProblemKey)
				}
			}
			status, err := txService.EvaluateProblem(ctx, p.ID, ownerID)
			if err != nil {
				return err
			}
			p.Status = status
			out.Problems = append(out.Problems, p)
			out.StatusCounts[status]++
		}
		return nil
	})
	return out, err
}

func sameReviewedProblem(a, b ProblemCase) bool {
	return a.OwnerID == b.OwnerID && a.ProblemKey == b.ProblemKey && a.Region == b.Region && a.ObservablePopulation == b.ObservablePopulation && a.ProblemScenario == b.ProblemScenario && a.CurrentWorkaround == b.CurrentWorkaround && a.Responsibility == b.Responsibility && a.ProductSolvability == b.ProductSolvability && a.HarmRisk == b.HarmRisk && a.NextMinimumEvidence == b.NextMinimumEvidence
}

func sameReviewedEvidence(a, b ProblemEvidence) bool {
	return a.ProblemCaseID == b.ProblemCaseID && a.Kind == b.Kind && a.Title == b.Title && a.SourceURI == b.SourceURI && a.ObservedAt.Equal(b.ObservedAt) && a.Collector == b.Collector && a.RawSHA256 == b.RawSHA256 && a.RawPayload == b.RawPayload && a.TrustedRun == b.TrustedRun
}

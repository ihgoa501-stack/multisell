package demandcase

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"

	"go.uber.org/zap"
	"gorm.io/gorm"
)

type Service struct {
	db     *gorm.DB
	logger *zap.Logger
}

func NewService(db *gorm.DB, logger *zap.Logger) *Service { return &Service{db: db, logger: logger} }

func (s *Service) Create(ctx context.Context, c *DemandCase) error {
	if c.OwnerID <= 0 || strings.TrimSpace(c.Region) == "" || strings.TrimSpace(c.Consumer) == "" || strings.TrimSpace(c.NeedScenario) == "" || strings.TrimSpace(c.SalesChannel) == "" || strings.TrimSpace(c.TargetLocale) == "" {
		return errors.New("region, consumer, need_scenario, sales_channel and target_locale are required")
	}
	c.Status = VerdictLead
	return s.db.WithContext(ctx).Create(c).Error
}

func (s *Service) requireOwner(ctx context.Context, id, ownerID int64) (*DemandCase, error) {
	var c DemandCase
	err := s.db.WithContext(ctx).Where("id = ? AND owner_id = ?", id, ownerID).First(&c).Error
	return &c, err
}

func (s *Service) AddEvidence(ctx context.Context, ownerID int64, e *DemandEvidence) error {
	if _, err := s.requireOwner(ctx, e.DemandCaseID, ownerID); err != nil {
		return err
	}
	if !validDimension(e.Dimension) || !validKind(e.Kind) || !validTruth(e.TruthStatus) || strings.TrimSpace(e.Title) == "" || strings.TrimSpace(e.RunID) == "" {
		return errors.New("invalid demand evidence")
	}
	if e.TruthStatus == TruthActual {
		return errors.New("actual evidence requires a separate Owner verification workflow")
	}
	if e.TruthStatus != TruthUnknown && e.TruthStatus != TruthMock && e.TruthStatus != TruthInferred && (strings.TrimSpace(e.SourceURI) == "" || e.ObservedAt == nil) {
		return errors.New("decision evidence requires source_uri and observed_at")
	}
	return s.db.WithContext(ctx).Create(e).Error
}

func validDimension(v string) bool {
	for _, d := range RequiredDimensions {
		if v == d {
			return true
		}
	}
	return false
}
func validKind(v string) bool {
	return v == EvidenceSupport || v == EvidenceCounter || v == EvidenceConflict
}
func validTruth(v string) bool {
	return v == TruthActual || v == TruthQuoted || v == TruthEstimated || v == TruthUnknown || v == TruthMock || v == TruthInferred
}

func (s *Service) Evaluate(ctx context.Context, id, ownerID int64) (*DemandVerdict, error) {
	c, err := s.requireOwner(ctx, id, ownerID)
	if err != nil {
		return nil, err
	}
	var evidence []DemandEvidence
	if err := s.db.WithContext(ctx).Where("demand_case_id = ?", id).Order("id").Find(&evidence).Error; err != nil {
		return nil, err
	}
	var snapshots []ResearchSnapshot
	if err := s.db.WithContext(ctx).Where("demand_case_id = ? AND owner_id = ?", id, ownerID).Find(&snapshots).Error; err != nil {
		return nil, err
	}
	snapshotByID := make(map[int64]ResearchSnapshot, len(snapshots))
	runTypes := make(map[string]bool, 3)
	for _, snapshot := range snapshots {
		if snapshot.ID <= 0 || payloadHash([]byte(snapshot.RawPayload)) != snapshot.RawSHA256 {
			continue
		}
		snapshotByID[snapshot.ID] = snapshot
		runTypes[snapshot.RunType] = true
	}
	status := VerdictExperimentReady
	blockers := make([]string, 0)
	support := map[string]bool{}
	hasCounter := false
	scoutRuns := map[string]bool{}
	counterRuns := map[string]bool{}
	for _, e := range evidence {
		snapshot, bound := snapshotByID[e.SnapshotID]
		bound = bound && snapshot.RunID == e.RunID && snapshot.SourceURI == e.SourceURI && e.ObservedAt != nil && snapshot.CollectedAt.Equal(e.ObservedAt.UTC())
		usable := bound && usableEvidence(e)
		if e.Kind == EvidenceSupport {
			if bound && snapshot.RunType == RunScout {
				scoutRuns[e.RunID] = true
			}
			if usable && snapshot.RunType == RunScout && e.TruthStatus == TruthQuoted {
				support[e.Dimension] = true
			}
		}
		if e.Kind == EvidenceCounter {
			if bound && snapshot.RunType == RunFalsifier {
				counterRuns[e.RunID] = true
			}
			if usable && snapshot.RunType == RunFalsifier && e.TruthStatus == TruthQuoted {
				hasCounter = true
			}
			if usable && snapshot.RunType == RunFalsifier && e.Fatal {
				status = VerdictRejected
				blockers = append(blockers, "fatal_counterevidence")
			}
		}
		if bound && snapshot.RunType == RunDataReality && e.Kind == EvidenceConflict {
			blockers = append(blockers, "unresolved_conflict:"+e.Dimension)
		}
	}
	for _, d := range RequiredDimensions {
		if !support[d] {
			blockers = append(blockers, "missing:"+d)
		}
	}
	independentCounter := false
	for run := range counterRuns {
		if !scoutRuns[run] {
			independentCounter = true
			break
		}
	}
	if !hasCounter || !independentCounter {
		blockers = append(blockers, "independent_counterevidence_required")
	}
	if !runTypes[RunDataReality] {
		blockers = append(blockers, "data_reality_run_required")
	}
	if status != VerdictRejected && len(blockers) > 0 {
		status = VerdictEvidenceMissing
	}
	b, _ := json.Marshal(blockers)
	var evidenceMaxID int64
	for _, row := range evidence {
		if row.ID > evidenceMaxID {
			evidenceMaxID = row.ID
		}
	}
	v := &DemandVerdict{DemandCaseID: id, EvidenceMaxID: evidenceMaxID, Status: status, BlockersJSON: string(b), Blockers: blockers, Reason: verdictReason(status), EvaluatedBy: ownerID}
	err = s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err := tx.Create(v).Error; err != nil {
			return err
		}
		return tx.Model(c).Update("status", status).Error
	})
	return v, err
}

func usableEvidence(e DemandEvidence) bool {
	return e.TruthStatus != TruthUnknown && e.TruthStatus != TruthMock && e.TruthStatus != TruthInferred && e.SourceURI != "" && e.ObservedAt != nil
}

func verdictReason(status string) string {
	switch status {
	case VerdictExperimentReady:
		return "研究证据允许生成实验草案，但仍需 Owner 批准且不能自动执行"
	case VerdictRejected:
		return "致命反证触发淘汰"
	default:
		return "关键证据或独立反证仍不完整"
	}
}

func (s *Service) latestVerdict(ctx context.Context, id int64) (*DemandVerdict, error) {
	var v DemandVerdict
	err := s.db.WithContext(ctx).Where("demand_case_id = ?", id).Order("id DESC").First(&v).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return &DemandVerdict{DemandCaseID: id, Status: VerdictEvidenceMissing, Blockers: []string{"not_evaluated"}, Reason: verdictReason(VerdictEvidenceMissing)}, nil
	}
	if err != nil {
		return nil, err
	}
	_ = json.Unmarshal([]byte(v.BlockersJSON), &v.Blockers)
	return &v, nil
}

func (s *Service) Get(ctx context.Context, id, ownerID int64) (*Detail, error) {
	c, err := s.requireOwner(ctx, id, ownerID)
	if err != nil {
		return nil, err
	}
	d := &Detail{Case: *c}
	if err := s.db.WithContext(ctx).Where("demand_case_id = ?", id).Order("id").Find(&d.Evidence).Error; err != nil {
		return nil, err
	}
	d.Verdict, err = s.latestVerdict(ctx, id)
	if err == nil {
		err = s.db.WithContext(ctx).Where("demand_case_id = ?", id).Order("id").Find(&d.Snapshots).Error
	}
	return d, err
}

func (s *Service) List(ctx context.Context, ownerID int64, page, size int) ([]DemandCase, int64, error) {
	if page < 1 {
		page = 1
	}
	if size < 1 || size > 100 {
		size = 20
	}
	var rows []DemandCase
	var total int64
	q := s.db.WithContext(ctx).Model(&DemandCase{}).Where("owner_id = ?", ownerID)
	if err := q.Count(&total).Error; err != nil {
		return nil, 0, err
	}
	err := q.Order("id DESC").Offset((page - 1) * size).Limit(size).Find(&rows).Error
	return rows, total, err
}

func (s *Service) DecisionCard(ctx context.Context, id, ownerID int64) (*OwnerDecisionCard, error) {
	d, err := s.Get(ctx, id, ownerID)
	if err != nil {
		return nil, err
	}
	proven := "尚无足够的可核验市场证据"
	notProven := "公开线索不能证明真实付款、签收、售后闭合或最终正贡献利润"
	counter := "尚无独立反证"
	for _, e := range d.Evidence {
		if e.Kind == EvidenceCounter && usableEvidence(e) {
			counter = e.Title
		}
		if e.Kind == EvidenceSupport && usableEvidence(e) {
			proven = "已记录有来源的候选市场证据；仍只属于研究证据"
		}
	}
	next := "补齐决策卡中的缺失证据与独立反证"
	if d.Verdict.Status == VerdictExperimentReady {
		next = "由 Owner 决定是否批准只读数据预检；尚不允许采购、发布或投放"
	}
	return &OwnerDecisionCard{DemandCaseID: id, Verdict: d.Verdict.Status, Hypothesis: fmt.Sprintf("%s 的 %s 在 %s 场景中可能通过 %s 产生可验证付费需求", d.Case.Region, d.Case.Consumer, d.Case.NeedScenario, d.Case.SalesChannel), Proven: proven, NotProven: notProven, StrongestCounterevidence: counter, NextAuthorityOrCost: next, StopCondition: d.Case.StopCondition}, nil
}

package demandcase

import (
	"context"
	"errors"
	"strings"
	"time"
)

const (
	ProblemLead                = "lead"
	ProblemEvidenceMissing     = "evidence_missing"
	ProblemSurvives            = "survives_falsification"
	ProblemRejected            = "rejected"
	ResponsibilityConsumer     = "consumer_controlled"
	ResponsibilityShared       = "shared"
	ResponsibilityLandlord     = "landlord"
	ResponsibilityEmployer     = "employer"
	ResponsibilityManufacturer = "manufacturer"
	ResponsibilityMedical      = "medical"
	ResponsibilityPublic       = "public_service"
	ResponsibilityUnknown      = "unknown"
	SolvabilityPlausible       = "plausible"
	SolvabilityPartial         = "partial"
	SolvabilityStructural      = "structural"
	SolvabilityUnknown         = "unknown"
	HarmLow                    = "low"
	HarmMedium                 = "medium"
	HarmHigh                   = "high"
	HarmUnknown                = "unknown"
)

type ProblemCase struct {
	ID                   int64     `gorm:"primaryKey" json:"id"`
	OwnerID              int64     `gorm:"index;not null;uniqueIndex:uidx_problem_owner_key" json:"owner_id"`
	ProblemKey           string    `gorm:"size:160;not null;uniqueIndex:uidx_problem_owner_key" json:"problem_key"`
	Region               string    `gorm:"size:80;not null" json:"region"`
	ObservablePopulation string    `gorm:"size:300;not null" json:"observable_population"`
	ProblemScenario      string    `gorm:"type:text;not null" json:"problem_scenario"`
	CurrentWorkaround    string    `gorm:"type:text;not null" json:"current_workaround"`
	Responsibility       string    `gorm:"size:32;not null" json:"responsibility"`
	ProductSolvability   string    `gorm:"size:24;not null" json:"product_solvability"`
	HarmRisk             string    `gorm:"size:16;not null" json:"harm_risk"`
	NextMinimumEvidence  string    `gorm:"type:text;not null" json:"next_minimum_evidence"`
	Status               string    `gorm:"size:32;not null;default:lead" json:"status"`
	CreatedAt            time.Time `json:"created_at"`
	UpdatedAt            time.Time `json:"updated_at"`
}

func (ProblemCase) TableName() string { return "problem_case" }

type ProblemEvidence struct {
	ID            int64     `gorm:"primaryKey" json:"id"`
	ProblemCaseID int64     `gorm:"index;not null;uniqueIndex:uidx_problem_evidence_identity" json:"problem_case_id"`
	Kind          string    `gorm:"size:16;not null;uniqueIndex:uidx_problem_evidence_identity" json:"kind"`
	Title         string    `gorm:"type:text;not null" json:"title"`
	SourceURI     string    `gorm:"type:text;not null" json:"source_uri"`
	ObservedAt    time.Time `json:"observed_at"`
	Collector     string    `gorm:"size:120;not null;uniqueIndex:uidx_problem_evidence_identity" json:"collector"`
	RawSHA256     string    `gorm:"size:64;not null;uniqueIndex:uidx_problem_evidence_identity" json:"raw_sha256"`
	RawPayload    string    `gorm:"type:text;not null" json:"raw_payload"`
	TrustedRun    bool      `gorm:"not null;default:false" json:"trusted_run"`
	CreatedAt     time.Time `json:"created_at"`
}

func (ProblemEvidence) TableName() string { return "problem_evidence" }

type ProblemDetail struct {
	Case     ProblemCase       `json:"case"`
	Evidence []ProblemEvidence `json:"evidence"`
}

func validResponsibility(v string) bool {
	return v == ResponsibilityConsumer || v == ResponsibilityShared || v == ResponsibilityLandlord || v == ResponsibilityEmployer || v == ResponsibilityManufacturer || v == ResponsibilityMedical || v == ResponsibilityPublic || v == ResponsibilityUnknown
}
func validSolvability(v string) bool {
	return v == SolvabilityPlausible || v == SolvabilityPartial || v == SolvabilityStructural || v == SolvabilityUnknown
}
func validHarm(v string) bool {
	return v == HarmLow || v == HarmMedium || v == HarmHigh || v == HarmUnknown
}

func (s *Service) CreateProblem(ctx context.Context, p *ProblemCase) error {
	if p.OwnerID <= 0 || strings.TrimSpace(p.ProblemKey) == "" || strings.TrimSpace(p.Region) == "" || strings.TrimSpace(p.ObservablePopulation) == "" || strings.TrimSpace(p.ProblemScenario) == "" || strings.TrimSpace(p.CurrentWorkaround) == "" || strings.TrimSpace(p.NextMinimumEvidence) == "" || !validResponsibility(p.Responsibility) || !validSolvability(p.ProductSolvability) || !validHarm(p.HarmRisk) {
		return errors.New("invalid problem-first case")
	}
	p.Status = ProblemLead
	return s.db.WithContext(ctx).Create(p).Error
}
func (s *Service) requireProblemOwner(ctx context.Context, id, ownerID int64) (*ProblemCase, error) {
	var p ProblemCase
	err := s.db.WithContext(ctx).Where("id = ? AND owner_id = ?", id, ownerID).First(&p).Error
	return &p, err
}
func (s *Service) AddProblemEvidence(ctx context.Context, ownerID int64, e *ProblemEvidence) error {
	if _, err := s.requireProblemOwner(ctx, e.ProblemCaseID, ownerID); err != nil {
		return err
	}
	if (e.Kind != EvidenceSupport && e.Kind != EvidenceCounter) || strings.TrimSpace(e.Title) == "" || strings.TrimSpace(e.SourceURI) == "" || e.ObservedAt.IsZero() || strings.TrimSpace(e.Collector) == "" || strings.TrimSpace(e.RawPayload) == "" || len(e.RawSHA256) != 64 || payloadHash([]byte(e.RawPayload)) != e.RawSHA256 {
		return errors.New("invalid problem evidence")
	}
	return s.db.WithContext(ctx).Create(e).Error
}
func (s *Service) EvaluateProblem(ctx context.Context, id, ownerID int64) (string, error) {
	p, err := s.requireProblemOwner(ctx, id, ownerID)
	if err != nil {
		return "", err
	}
	var ev []ProblemEvidence
	if err := s.db.WithContext(ctx).Where("problem_case_id = ?", id).Find(&ev).Error; err != nil {
		return "", err
	}
	support, counters := map[string]bool{}, map[string]bool{}
	for _, e := range ev {
		if !e.TrustedRun {
			continue
		}
		if e.Kind == EvidenceSupport {
			support[e.Collector] = true
		}
		if e.Kind == EvidenceCounter {
			counters[e.Collector] = true
		}
	}
	independent := false
	for c := range counters {
		if !support[c] {
			independent = true
			break
		}
	}
	status := ProblemEvidenceMissing
	if independent && len(support) > 0 {
		if p.Responsibility == ResponsibilityLandlord || p.Responsibility == ResponsibilityEmployer || p.Responsibility == ResponsibilityManufacturer || p.Responsibility == ResponsibilityMedical || p.Responsibility == ResponsibilityPublic || p.ProductSolvability == SolvabilityStructural || p.HarmRisk == HarmHigh {
			status = ProblemRejected
		} else if (p.Responsibility == ResponsibilityConsumer || p.Responsibility == ResponsibilityShared) && (p.ProductSolvability == SolvabilityPlausible || p.ProductSolvability == SolvabilityPartial) && p.HarmRisk == HarmLow {
			status = ProblemSurvives
		}
	}
	if err := s.db.WithContext(ctx).Model(p).Update("status", status).Error; err != nil {
		return "", err
	}
	return status, nil
}
func (s *Service) PromoteProblem(ctx context.Context, id, ownerID int64, channel string) (*DemandCase, error) {
	p, err := s.requireProblemOwner(ctx, id, ownerID)
	if err != nil {
		return nil, err
	}
	if p.Status != ProblemSurvives || strings.TrimSpace(channel) == "" {
		return nil, errors.New("only a surviving problem can enter channel comparison")
	}
	d := &DemandCase{OwnerID: ownerID, Region: p.Region, Consumer: p.ObservablePopulation, NeedScenario: p.ProblemScenario, SalesChannel: channel, StopCondition: "下一证据失败或责任/伤害边界改变时停止"}
	if err := s.Create(ctx, d); err != nil {
		return nil, err
	}
	return d, nil
}
func (s *Service) ListProblems(ctx context.Context, ownerID int64) ([]ProblemCase, error) {
	var rows []ProblemCase
	err := s.db.WithContext(ctx).Where("owner_id = ?", ownerID).Order("id DESC").Find(&rows).Error
	return rows, err
}
func (s *Service) GetProblem(ctx context.Context, id, ownerID int64) (*ProblemDetail, error) {
	p, err := s.requireProblemOwner(ctx, id, ownerID)
	if err != nil {
		return nil, err
	}
	d := &ProblemDetail{Case: *p}
	err = s.db.WithContext(ctx).Where("problem_case_id = ?", id).Order("id").Find(&d.Evidence).Error
	return d, err
}

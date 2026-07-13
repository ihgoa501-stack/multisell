package businessdecision

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"strconv"
	"strings"
	"time"

	"gorm.io/gorm"
)

var ErrConflict = errors.New("idempotency or frozen fact conflict")
var ErrInvalid = errors.New("invalid business decision input")

type Service struct{ db *gorm.DB }

func NewService(db *gorm.DB) *Service { return &Service{db: db} }
func digest(v any) (string, string) {
	b, _ := json.Marshal(v)
	h := sha256.Sum256(b)
	return string(b), hex.EncodeToString(h[:])
}
func validTruth(v string) bool {
	return v == "actual" || v == "quoted" || v == "estimated" || v == "unknown" || v == "mock" || v == "inferred" || v == "external_observed" || v == "reconciled"
}
func validSHA256(v string) bool {
	if len(v) != 64 {
		return false
	}
	_, err := hex.DecodeString(v)
	return err == nil && v == strings.ToLower(v)
}
func canonicalJSON(raw json.RawMessage) (json.RawMessage, bool) {
	var value any
	if len(raw) == 0 || json.Unmarshal(raw, &value) != nil {
		return nil, false
	}
	b, err := json.Marshal(value)
	return json.RawMessage(b), err == nil
}

type orderFact struct {
	ID                int64
	OwnerID           int64
	AccountID         int64
	PlatformCode      string
	ExternalEventID   string
	ExternalOrderID   string
	EventAction       string
	TruthStatus       string
	PayloadSHA256     string
	ObservedAt        time.Time
	NormalizedOrderID *int64
	ProcessingStatus  string
}

func (s *Service) freezeFact(ctx context.Context, owner int64, objectType string, id int64) (FactSnapshot, error) {
	if objectType == "purchase_authority" {
		var f struct {
			ID, OwnerID, SupplierID, SKUMappingID, InternalSKUID, CostVersionID, InventoryID int64
			Quantity                                                                         int
			UnitAmountMinor, TotalAmountMinor                                                int64
			Currency, Status, RequestSHA256                                                  string
			CreatedAt                                                                        time.Time
		}
		if err := s.db.WithContext(ctx).Table("purchase_authority").Where("id=? AND owner_id=?", id, owner).Take(&f).Error; err != nil {
			return FactSnapshot{}, err
		}
		payload, hash := digest(f)
		return FactSnapshot{OwnerID: owner, ObjectType: objectType, ObjectID: id, TruthStatus: "actual", SourceTable: "purchase_authority", SourceObservedAt: f.CreatedAt, PayloadJSON: payload, PayloadSHA256: hash}, nil
	}
	var f orderFact
	err := s.db.WithContext(ctx).Table("platform_order_ingest").Where("id=? AND owner_id=?", id, owner).Take(&f).Error
	if err != nil {
		return FactSnapshot{}, err
	}
	payload, hash := digest(f)
	return FactSnapshot{OwnerID: owner, ObjectType: objectType, ObjectID: id, TruthStatus: f.TruthStatus, SourceTable: "platform_order_ingest", SourceObservedAt: f.ObservedAt, PayloadJSON: payload, PayloadSHA256: hash}, nil
}

func (s *Service) CreateCase(ctx context.Context, owner int64, in CreateCaseInput) (*Case, error) {
	in.Question = strings.TrimSpace(in.Question)
	in.Target = strings.TrimSpace(in.Target)
	in.ObjectType = strings.TrimSpace(in.ObjectType)
	in.IdempotencyKey = strings.TrimSpace(in.IdempotencyKey)
	if owner <= 0 || in.Question == "" || in.Target == "" || (in.ObjectType != "platform_order_ingest" && in.ObjectType != "purchase_authority") || in.ObjectID <= 0 || in.IdempotencyKey == "" {
		return nil, ErrInvalid
	}
	var out Case
	err := s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		var old Case
		if e := tx.Where("owner_id=? AND idempotency_key=?", owner, in.IdempotencyKey).Take(&old).Error; e == nil {
			if old.ObjectType != in.ObjectType || old.ObjectID != in.ObjectID || old.Question != in.Question || old.Target != in.Target {
				return ErrConflict
			}
			out = old
			return nil
		} else if !errors.Is(e, gorm.ErrRecordNotFound) {
			return e
		}
		ss, err := (&Service{db: tx}).freezeFact(ctx, owner, in.ObjectType, in.ObjectID)
		if err != nil {
			return err
		}
		u, _ := json.Marshal(in.Unknowns)
		out = Case{OwnerID: owner, Question: in.Question, Target: in.Target, ObjectType: in.ObjectType, ObjectID: in.ObjectID, TruthStatus: ss.TruthStatus, UnknownsJSON: string(u), ManifestSHA256: ss.PayloadSHA256, IdempotencyKey: in.IdempotencyKey}
		if err = tx.Create(&out).Error; err != nil {
			return err
		}
		ss.DecisionCaseID = out.ID
		return tx.Create(&ss).Error
	})
	out.Unknowns = in.Unknowns
	return &out, err
}

func (s *Service) Recommend(ctx context.Context, owner, caseID int64, in RecommendInput) (*AIRecommendation, error) {
	in.Recommendation = strings.TrimSpace(in.Recommendation)
	in.Rationale = strings.TrimSpace(in.Rationale)
	in.IdempotencyKey = strings.TrimSpace(in.IdempotencyKey)
	if in.Recommendation == "" || in.Rationale == "" || in.IdempotencyKey == "" || !validTruth(in.TruthStatus) || in.TruthStatus == "actual" || in.TruthStatus == "external_observed" || in.TruthStatus == "reconciled" {
		return nil, ErrInvalid
	}
	var c Case
	if err := s.db.WithContext(ctx).Where("id=? AND owner_id=?", caseID, owner).Take(&c).Error; err != nil {
		return nil, err
	}
	var old AIRecommendation
	if e := s.db.WithContext(ctx).Where("owner_id=? AND idempotency_key=?", owner, in.IdempotencyKey).Take(&old).Error; e == nil {
		if old.DecisionCaseID != caseID || old.Recommendation != in.Recommendation || old.Rationale != in.Rationale || old.ManifestSHA256 != c.ManifestSHA256 {
			return nil, ErrConflict
		}
		return &old, nil
	} else if !errors.Is(e, gorm.ErrRecordNotFound) {
		return nil, e
	}
	u, _ := json.Marshal(in.Unknowns)
	r := AIRecommendation{DecisionCaseID: caseID, OwnerID: owner, Recommendation: in.Recommendation, Rationale: in.Rationale, TruthStatus: in.TruthStatus, UnknownsJSON: string(u), ManifestSHA256: c.ManifestSHA256, IdempotencyKey: in.IdempotencyKey, Unknowns: in.Unknowns}
	return &r, s.db.WithContext(ctx).Create(&r).Error
}

func (s *Service) Decide(ctx context.Context, owner, caseID int64, in DecideInput) (*OwnerDecision, error) {
	in.Decision = strings.TrimSpace(in.Decision)
	in.Reason = strings.TrimSpace(in.Reason)
	in.IdempotencyKey = strings.TrimSpace(in.IdempotencyKey)
	in.CapabilityID = strings.TrimSpace(in.CapabilityID)
	in.CommandType = strings.TrimSpace(in.CommandType)
	in.TargetType = strings.TrimSpace(in.TargetType)
	in.TargetID = strings.TrimSpace(in.TargetID)
	in.InputSHA256 = strings.TrimSpace(in.InputSHA256)
	canonicalPayload, payloadValid := canonicalJSON(in.InputPayload)
	// The database contract stores an explicit empty JSON object for decisions
	// that do not carry a command payload. A nil RawMessage is serialized by
	// GORM as SQL NULL and violates the immutable NOT NULL payload contract.
	if len(in.InputPayload) == 0 {
		canonicalPayload = json.RawMessage(`{}`)
	}
	computedInputSHA := ""
	if payloadValid {
		h := sha256.Sum256(canonicalPayload)
		computedInputSHA = hex.EncodeToString(h[:])
	}
	valid := in.Decision == DecisionSelected || in.Decision == DecisionRejected || in.Decision == DecisionPaused || in.Decision == DecisionMoreEvidence
	selectedActionValid := in.CapabilityID != "" && in.CommandType != "" && in.TargetType != "" && in.TargetID != "" && validSHA256(in.InputSHA256) && ((payloadValid && computedInputSHA == in.InputSHA256) || (in.CapabilityID == "purchase.authority.execute" && len(in.InputPayload) == 0))
	nonSelectedActionEmpty := in.CapabilityID == "" && in.CommandType == "" && in.TargetType == "" && in.TargetID == "" && in.InputSHA256 == "" && len(in.InputPayload) == 0
	if !valid || in.Reason == "" || in.IdempotencyKey == "" || len(in.ManifestSHA256) != 64 || (in.Decision == DecisionSelected && !selectedActionValid) || (in.Decision != DecisionSelected && !nonSelectedActionEmpty) {
		return nil, ErrInvalid
	}
	var c Case
	if err := s.db.WithContext(ctx).Where("id=? AND owner_id=?", caseID, owner).Take(&c).Error; err != nil {
		return nil, err
	}
	if c.ManifestSHA256 != in.ManifestSHA256 {
		return nil, ErrConflict
	}
	if in.RecommendationID != nil {
		var r AIRecommendation
		if err := s.db.WithContext(ctx).Where("id=? AND decision_case_id=? AND owner_id=? AND manifest_sha256=?", *in.RecommendationID, caseID, owner, c.ManifestSHA256).Take(&r).Error; err != nil {
			return nil, ErrConflict
		}
	}
	var old OwnerDecision
	if e := s.db.WithContext(ctx).Where("owner_id=? AND idempotency_key=?", owner, in.IdempotencyKey).Take(&old).Error; e == nil {
		if old.DecisionCaseID != caseID || old.Decision != in.Decision || old.Reason != in.Reason || old.ManifestSHA256 != in.ManifestSHA256 || old.CapabilityID != in.CapabilityID || old.CommandType != in.CommandType || old.TargetType != in.TargetType || old.TargetID != in.TargetID || old.InputSHA256 != in.InputSHA256 {
			return nil, ErrConflict
		}
		return &old, nil
	} else if !errors.Is(e, gorm.ErrRecordNotFound) {
		return nil, e
	}
	d := OwnerDecision{DecisionCaseID: caseID, OwnerID: owner, RecommendationID: in.RecommendationID, Decision: in.Decision, CapabilityID: in.CapabilityID, CommandType: in.CommandType, TargetType: in.TargetType, TargetID: in.TargetID, InputSHA256: in.InputSHA256, InputPayload: append(json.RawMessage(nil), canonicalPayload...), Reason: in.Reason, ManifestSHA256: c.ManifestSHA256, IdempotencyKey: in.IdempotencyKey}
	return &d, s.db.WithContext(ctx).Create(&d).Error
}

func (s *Service) List(ctx context.Context, owner int64) ([]ListItem, error) {
	var cases []Case
	if err := s.db.WithContext(ctx).Where("owner_id=?", owner).Order("id DESC").Find(&cases).Error; err != nil {
		return nil, err
	}
	out := make([]ListItem, 0, len(cases))
	for _, c := range cases {
		_ = json.Unmarshal([]byte(c.UnknownsJSON), &c.Unknowns)
		item := ListItem{Case: c}
		var d OwnerDecision
		if err := s.db.WithContext(ctx).Where("decision_case_id=? AND owner_id=?", c.ID, owner).Order("created_at DESC, id DESC").Take(&d).Error; err == nil {
			item.LatestDecision = &d
		} else if !errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, err
		}
		out = append(out, item)
	}
	return out, nil
}
func (s *Service) FactOptions(ctx context.Context, owner int64, objectType string) ([]FactOption, error) {
	var out []FactOption
	switch objectType {
	case "platform_order_ingest":
		var rows []struct {
			ID                                         int64
			ExternalOrderID, PlatformCode, TruthStatus string
			ObservedAt                                 time.Time
		}
		if err := s.db.WithContext(ctx).Table("platform_order_ingest").Select("id, external_order_id, platform_code, truth_status, observed_at").Where("owner_id=? AND truth_status='external_observed' AND processing_status='applied'", owner).Order("id DESC").Find(&rows).Error; err != nil {
			return nil, err
		}
		for _, r := range rows {
			out = append(out, FactOption{ObjectType: objectType, ObjectID: r.ID, Label: r.PlatformCode + " · 订单 " + r.ExternalOrderID, TruthStatus: r.TruthStatus, ObservedAt: r.ObservedAt})
		}
	case "purchase_authority":
		var rows []struct {
			ID                    int64
			Status, RequestSHA256 string
			CreatedAt             time.Time
		}
		if err := s.db.WithContext(ctx).Table("purchase_authority").Select("id, status, request_sha256, created_at").Where("owner_id=?", owner).Order("id DESC").Find(&rows).Error; err != nil {
			return nil, err
		}
		for _, r := range rows {
			out = append(out, FactOption{ObjectType: objectType, ObjectID: r.ID, Label: "采购授权 #" + strconv.FormatInt(r.ID, 10) + " · " + r.Status, TruthStatus: "actual", ObservedAt: r.CreatedAt})
		}
	default:
		return nil, ErrInvalid
	}
	return out, nil
}

func (s *Service) Get(ctx context.Context, owner, id int64) (*Detail, error) {
	var d Detail
	if e := s.db.WithContext(ctx).Where("id=? AND owner_id=?", id, owner).Take(&d.Case).Error; e != nil {
		return nil, e
	}
	_ = json.Unmarshal([]byte(d.Case.UnknownsJSON), &d.Case.Unknowns)
	if e := s.db.WithContext(ctx).Where("decision_case_id=? AND owner_id=?", id, owner).Take(&d.Snapshot).Error; e != nil {
		return nil, e
	}
	if e := s.db.WithContext(ctx).Where("decision_case_id=? AND owner_id=?", id, owner).Order("id").Find(&d.Recommendations).Error; e != nil {
		return nil, e
	}
	for i := range d.Recommendations {
		if e := json.Unmarshal([]byte(d.Recommendations[i].UnknownsJSON), &d.Recommendations[i].Unknowns); e != nil || d.Recommendations[i].Unknowns == nil {
			d.Recommendations[i].Unknowns = []string{}
		}
	}
	if e := s.db.WithContext(ctx).Where("decision_case_id=? AND owner_id=?", id, owner).Order("id").Find(&d.Decisions).Error; e != nil {
		return nil, e
	}
	return &d, nil
}

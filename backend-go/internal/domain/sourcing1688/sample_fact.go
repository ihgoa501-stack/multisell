package sourcing1688

import (
	"errors"
	"fmt"
	"net/http"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/lingmirror/backend-go/internal/response"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

const (
	SampleStatusRequest   = "request"
	SampleStatusApproved  = "approved_to_order"
	SampleStatusOrdered   = "ordered"
	SampleStatusReceived  = "received"
	SampleStatusEvaluated = "evaluated"
	SampleStatusAccepted  = "accepted"
	SampleStatusRejected  = "rejected"
)

var sampleCurrencyPattern = regexp.MustCompile(`^[A-Z]{3,8}$`)

// SourcingSample is the Owner-scoped current state of one physical sample.
// It freezes the exact sourcing authority, supplier and immutable observation
// used when the request was opened. No transition performs an external order.
type SourcingSample struct {
	ID                    int64      `gorm:"column:id;primaryKey;autoIncrement" json:"id"`
	OwnerID               int64      `gorm:"column:owner_id;not null;index" json:"owner_id"`
	SourcingProductID     int64      `gorm:"column:sourcing_product_id;not null;index" json:"sourcing_product_id"`
	TaskLinkID            int64      `gorm:"column:task_link_id;not null" json:"task_link_id"`
	ProductOpportunityID  int64      `gorm:"column:product_opportunity_id;not null" json:"product_opportunity_id"`
	OpportunityDecisionID int64      `gorm:"column:opportunity_decision_id;not null" json:"opportunity_decision_id"`
	SupplierID            int64      `gorm:"column:supplier_id;not null" json:"supplier_id"`
	SnapshotID            int64      `gorm:"column:snapshot_id;not null" json:"snapshot_id"`
	SupplierSKU           *string    `gorm:"column:supplier_sku" json:"supplier_sku,omitempty"`
	Quantity              int        `gorm:"column:quantity;not null" json:"quantity"`
	Status                string     `gorm:"column:status;not null" json:"status"`
	OrderAmount           *float64   `gorm:"column:order_amount;type:numeric(14,2)" json:"order_amount,omitempty"`
	Currency              *string    `gorm:"column:currency" json:"currency,omitempty"`
	ExternalCredentialURI *string    `gorm:"column:external_credential_uri" json:"external_credential_uri,omitempty"`
	ObservedAt            *time.Time `gorm:"column:observed_at" json:"observed_at,omitempty"`
	TruthStatus           string     `gorm:"column:truth_status;not null" json:"truth_status"`
	Evaluation            string     `gorm:"column:evaluation;not null" json:"evaluation,omitempty"`
	CreatedBy             int64      `gorm:"column:created_by;not null" json:"created_by"`
	CreatedAt             time.Time  `gorm:"column:created_at;autoCreateTime" json:"created_at"`
	UpdatedAt             time.Time  `gorm:"column:updated_at;autoUpdateTime" json:"updated_at"`
}

func (SourcingSample) TableName() string { return "sourcing_sample" }

type SourcingSampleEvent struct {
	ID                    int64      `gorm:"column:id;primaryKey;autoIncrement" json:"id"`
	SampleID              int64      `gorm:"column:sample_id;not null;index" json:"sample_id"`
	OwnerID               int64      `gorm:"column:owner_id;not null;index" json:"owner_id"`
	FromStatus            string     `gorm:"column:from_status;not null" json:"from_status"`
	ToStatus              string     `gorm:"column:to_status;not null" json:"to_status"`
	OrderAmount           *float64   `gorm:"column:order_amount" json:"order_amount,omitempty"`
	Currency              *string    `gorm:"column:currency" json:"currency,omitempty"`
	ExternalCredentialURI *string    `gorm:"column:external_credential_uri" json:"external_credential_uri,omitempty"`
	ObservedAt            *time.Time `gorm:"column:observed_at" json:"observed_at,omitempty"`
	TruthStatus           string     `gorm:"column:truth_status;not null" json:"truth_status"`
	Note                  string     `gorm:"column:note;not null" json:"note,omitempty"`
	ActorID               int64      `gorm:"column:actor_id;not null" json:"actor_id"`
	CreatedAt             time.Time  `gorm:"column:created_at;autoCreateTime" json:"created_at"`
}

func (SourcingSampleEvent) TableName() string { return "sourcing_sample_event" }

type CreateSourcingSampleInput struct {
	TaskLinkID  int64   `json:"task_link_id" binding:"required"`
	SupplierID  int64   `json:"supplier_id" binding:"required"`
	SnapshotID  int64   `json:"snapshot_id" binding:"required"`
	SupplierSKU *string `json:"supplier_sku"`
	Quantity    int     `json:"quantity"`
}

type TransitionSourcingSampleInput struct {
	ToStatus              string     `json:"to_status" binding:"required"`
	OrderAmount           *float64   `json:"order_amount"`
	Currency              *string    `json:"currency"`
	ExternalCredentialURI *string    `json:"external_credential_uri"`
	ObservedAt            *time.Time `json:"observed_at"`
	TruthStatus           string     `json:"truth_status"`
	Note                  string     `json:"note"`
}

type WaiveSourcingSampleInput struct {
	Reason string `json:"reason" binding:"required"`
}

type SourcingSampleDetail struct {
	Sample SourcingSample        `json:"sample"`
	Events []SourcingSampleEvent `json:"events"`
}

func (s *Service) CreateSourcingSample(ownerID, sourceID int64, in *CreateSourcingSampleInput) (*SourcingSampleDetail, error) {
	if ownerID <= 0 || sourceID <= 0 || in == nil || in.TaskLinkID <= 0 || in.SupplierID <= 0 || in.SnapshotID <= 0 {
		return nil, ErrInvalidWorkflow
	}
	if in.Quantity <= 0 {
		in.Quantity = 1
	}
	var detail SourcingSampleDetail
	err := s.db.Transaction(func(tx *gorm.DB) error {
		link, err := requireTaskSourcingAuthority(tx, sourceID, ownerID, in.TaskLinkID)
		if err != nil {
			return err
		}
		if link.ProductOpportunityID == nil || link.OpportunityDecisionID == nil {
			return fmt.Errorf("%w: frozen product opportunity is missing", ErrWorkflowGate)
		}
		var source Sourcing1688Product
		if err := tx.Where("id = ? AND owner_id = ?", sourceID, ownerID).First(&source).Error; err != nil {
			return fmt.Errorf("%w: sourcing product does not belong to Owner", ErrWorkflowGate)
		}
		if source.SupplierID == nil || *source.SupplierID != in.SupplierID {
			return fmt.Errorf("%w: sample supplier must match the authoritative sourcing supplier", ErrWorkflowGate)
		}
		var snapshotCount int64
		if err := tx.Model(&Sourcing1688Snapshot{}).Where("id = ? AND sourcing_product_id = ?", in.SnapshotID, sourceID).Count(&snapshotCount).Error; err != nil || snapshotCount != 1 {
			return fmt.Errorf("%w: immutable sourcing snapshot does not match product", ErrWorkflowGate)
		}
		sample := SourcingSample{OwnerID: ownerID, SourcingProductID: sourceID, TaskLinkID: link.ID,
			ProductOpportunityID: *link.ProductOpportunityID, OpportunityDecisionID: *link.OpportunityDecisionID,
			SupplierID: in.SupplierID, SnapshotID: in.SnapshotID, SupplierSKU: normalizeOptionalSampleString(in.SupplierSKU),
			Quantity: in.Quantity, Status: SampleStatusRequest, TruthStatus: "unknown", CreatedBy: ownerID}
		if err := tx.Create(&sample).Error; err != nil {
			return err
		}
		event := SourcingSampleEvent{SampleID: sample.ID, OwnerID: ownerID, FromStatus: "", ToStatus: SampleStatusRequest, TruthStatus: "unknown", Note: "Owner opened sample request; no external order was executed", ActorID: ownerID}
		if err := tx.Create(&event).Error; err != nil {
			return err
		}
		detail = SourcingSampleDetail{Sample: sample, Events: []SourcingSampleEvent{event}}
		return nil
	})
	return &detail, err
}

func requireSampleApprovalGate(tx *gorm.DB, link *Sourcing1688TaskLink) error {
	if link == nil {
		return ErrWorkflowGate
	}
	if link.SamplePolicy == "waived" && strings.TrimSpace(link.SampleWaiverReason) != "" && link.SampleWaivedBy != nil && link.SampleWaivedAt != nil {
		return nil
	}
	var accepted int64
	if err := tx.Model(&SourcingSample{}).Where("owner_id = ? AND sourcing_product_id = ? AND task_link_id = ? AND status = ?", link.OwnerID, link.SourcingProductID, link.ID, SampleStatusAccepted).Count(&accepted).Error; err != nil {
		return err
	}
	if accepted != 1 {
		return fmt.Errorf("%w: one accepted sample or explicit Owner waiver is required before draft approval", ErrWorkflowGate)
	}
	return nil
}

func (s *Service) WaiveSourcingSample(ownerID, sourceID, taskLinkID int64, in *WaiveSourcingSampleInput) (*Sourcing1688TaskLink, error) {
	if ownerID <= 0 || sourceID <= 0 || taskLinkID <= 0 || in == nil || strings.TrimSpace(in.Reason) == "" {
		return nil, ErrInvalidWorkflow
	}
	var result Sourcing1688TaskLink
	err := s.db.Transaction(func(tx *gorm.DB) error {
		link, err := requireTaskSourcingAuthority(tx, sourceID, ownerID, taskLinkID)
		if err != nil {
			return err
		}
		if link.SamplePolicy == "waived" {
			if link.SampleWaiverReason != strings.TrimSpace(in.Reason) {
				return fmt.Errorf("%w: sample waiver is immutable", ErrWorkflowGate)
			}
			result = *link
			return nil
		}
		if link.WorkflowStatus == "pending_approval" || link.WorkflowStatus == "approved_draft" || strings.HasPrefix(link.WorkflowStatus, "publish") || link.WorkflowStatus == "submitted" || link.WorkflowStatus == "succeeded" {
			return fmt.Errorf("%w: sample waiver must be decided before draft approval", ErrWorkflowGate)
		}
		now := time.Now().UTC()
		updates := map[string]any{"sample_policy": "waived", "sample_waiver_reason": strings.TrimSpace(in.Reason), "sample_waived_by": ownerID, "sample_waived_at": now}
		if err := tx.Model(link).Updates(updates).Error; err != nil {
			return err
		}
		link.SamplePolicy, link.SampleWaiverReason, link.SampleWaivedBy, link.SampleWaivedAt = "waived", strings.TrimSpace(in.Reason), &ownerID, &now
		result = *link
		return nil
	})
	return &result, err
}

func normalizeOptionalSampleString(value *string) *string {
	if value == nil || strings.TrimSpace(*value) == "" {
		return nil
	}
	v := strings.TrimSpace(*value)
	return &v
}

var sampleNextStates = map[string]map[string]bool{
	SampleStatusRequest:   {SampleStatusApproved: true},
	SampleStatusApproved:  {SampleStatusOrdered: true},
	SampleStatusOrdered:   {SampleStatusReceived: true},
	SampleStatusReceived:  {SampleStatusEvaluated: true},
	SampleStatusEvaluated: {SampleStatusAccepted: true, SampleStatusRejected: true},
}

func validateSampleTransition(current string, in *TransitionSourcingSampleInput) error {
	to := strings.TrimSpace(in.ToStatus)
	if !sampleNextStates[current][to] {
		return fmt.Errorf("%w: sample transition %s -> %s is not allowed", ErrInvalidLifecycleTransition, current, to)
	}
	if to == SampleStatusApproved && strings.TrimSpace(in.Note) == "" {
		return fmt.Errorf("%w: Owner approval reason is required", ErrWorkflowGate)
	}
	if to == SampleStatusOrdered || to == SampleStatusReceived || to == SampleStatusEvaluated || to == SampleStatusAccepted || to == SampleStatusRejected {
		if in.ObservedAt == nil || in.ObservedAt.IsZero() || in.ObservedAt.After(time.Now().Add(5*time.Minute)) {
			return fmt.Errorf("%w: observed_at is required and cannot be in the future", ErrWorkflowGate)
		}
		if strings.TrimSpace(in.TruthStatus) != "actual" {
			return fmt.Errorf("%w: physical sample facts require truth_status actual", ErrWorkflowGate)
		}
		if in.ExternalCredentialURI == nil || strings.TrimSpace(*in.ExternalCredentialURI) == "" {
			return fmt.Errorf("%w: external credential URI is required", ErrWorkflowGate)
		}
	}
	if to == SampleStatusOrdered {
		if in.OrderAmount == nil || *in.OrderAmount < 0 || in.Currency == nil || !sampleCurrencyPattern.MatchString(strings.ToUpper(strings.TrimSpace(*in.Currency))) {
			return fmt.Errorf("%w: ordered sample requires non-negative amount and ISO-like currency", ErrWorkflowGate)
		}
	}
	if to == SampleStatusEvaluated && strings.TrimSpace(in.Note) == "" {
		return fmt.Errorf("%w: sample evaluation is required", ErrWorkflowGate)
	}
	if (to == SampleStatusAccepted || to == SampleStatusRejected) && strings.TrimSpace(in.Note) == "" {
		return fmt.Errorf("%w: final sample decision reason is required", ErrWorkflowGate)
	}
	return nil
}

func (s *Service) TransitionSourcingSample(ownerID, sourceID, sampleID int64, in *TransitionSourcingSampleInput) (*SourcingSampleDetail, error) {
	if ownerID <= 0 || sourceID <= 0 || sampleID <= 0 || in == nil {
		return nil, ErrInvalidWorkflow
	}
	var detail SourcingSampleDetail
	err := s.db.Transaction(func(tx *gorm.DB) error {
		var sample SourcingSample
		if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).Where("id = ? AND owner_id = ? AND sourcing_product_id = ?", sampleID, ownerID, sourceID).First(&sample).Error; err != nil {
			return err
		}
		link, err := requireTaskSourcingAuthority(tx, sourceID, ownerID, sample.TaskLinkID)
		if err != nil {
			return err
		}
		if link.ProductOpportunityID == nil || link.OpportunityDecisionID == nil || *link.ProductOpportunityID != sample.ProductOpportunityID || *link.OpportunityDecisionID != sample.OpportunityDecisionID {
			return fmt.Errorf("%w: frozen sample opportunity authority changed", ErrWorkflowGate)
		}
		if err := validateSampleTransition(sample.Status, in); err != nil {
			return err
		}
		to := strings.TrimSpace(in.ToStatus)
		updates := map[string]any{"status": to, "updated_at": time.Now()}
		if in.OrderAmount != nil {
			updates["order_amount"] = *in.OrderAmount
		}
		if in.Currency != nil {
			currency := strings.ToUpper(strings.TrimSpace(*in.Currency))
			updates["currency"] = currency
			in.Currency = &currency
		}
		if in.ExternalCredentialURI != nil {
			uri := strings.TrimSpace(*in.ExternalCredentialURI)
			updates["external_credential_uri"] = uri
			in.ExternalCredentialURI = &uri
		}
		if in.ObservedAt != nil {
			updates["observed_at"] = *in.ObservedAt
		}
		if strings.TrimSpace(in.TruthStatus) != "" {
			updates["truth_status"] = strings.TrimSpace(in.TruthStatus)
		}
		if to == SampleStatusEvaluated {
			updates["evaluation"] = strings.TrimSpace(in.Note)
		}
		if err := tx.Model(&SourcingSample{}).Where("id = ? AND owner_id = ? AND status = ?", sample.ID, ownerID, sample.Status).Updates(updates).Error; err != nil {
			return err
		}
		event := SourcingSampleEvent{SampleID: sample.ID, OwnerID: ownerID, FromStatus: sample.Status, ToStatus: to, OrderAmount: in.OrderAmount, Currency: in.Currency, ExternalCredentialURI: in.ExternalCredentialURI, ObservedAt: in.ObservedAt, TruthStatus: sample.TruthStatus, Note: strings.TrimSpace(in.Note), ActorID: ownerID}
		if strings.TrimSpace(in.TruthStatus) != "" {
			event.TruthStatus = strings.TrimSpace(in.TruthStatus)
		}
		if err := tx.Create(&event).Error; err != nil {
			return err
		}
		return loadSourcingSampleDetail(tx, ownerID, sourceID, sample.ID, &detail)
	})
	return &detail, err
}

func loadSourcingSampleDetail(db *gorm.DB, ownerID, sourceID, sampleID int64, out *SourcingSampleDetail) error {
	if err := db.Where("id = ? AND owner_id = ? AND sourcing_product_id = ?", sampleID, ownerID, sourceID).First(&out.Sample).Error; err != nil {
		return err
	}
	return db.Where("sample_id = ? AND owner_id = ?", sampleID, ownerID).Order("id ASC").Find(&out.Events).Error
}

func (s *Service) ListSourcingSamples(ownerID, sourceID int64) ([]SourcingSampleDetail, error) {
	if err := s.RequireSourceOwner(sourceID, ownerID); err != nil {
		return nil, err
	}
	var samples []SourcingSample
	if err := s.db.Where("owner_id = ? AND sourcing_product_id = ?", ownerID, sourceID).Order("id DESC").Find(&samples).Error; err != nil {
		return nil, err
	}
	items := make([]SourcingSampleDetail, 0, len(samples))
	for _, sample := range samples {
		var events []SourcingSampleEvent
		if err := s.db.Where("sample_id = ? AND owner_id = ?", sample.ID, ownerID).Order("id ASC").Find(&events).Error; err != nil {
			return nil, err
		}
		items = append(items, SourcingSampleDetail{Sample: sample, Events: events})
	}
	return items, nil
}

func parseSampleID(c *gin.Context) (int64, bool) {
	id, err := strconv.ParseInt(c.Param("sampleId"), 10, 64)
	if err != nil || id <= 0 {
		response.Error(c, http.StatusBadRequest, "invalid sample id")
		return 0, false
	}
	return id, true
}

func (h *Handler) ListSourcingSamples(c *gin.Context) {
	sourceID, ok := parseID(c)
	if !ok {
		return
	}
	ownerID, ok := h.requireSourceOwner(c, sourceID)
	if !ok {
		return
	}
	items, err := h.service.ListSourcingSamples(ownerID, sourceID)
	if err != nil {
		workflowError(c, err)
		return
	}
	response.Success(c, items)
}

func (h *Handler) CreateSourcingSample(c *gin.Context) {
	sourceID, ok := parseID(c)
	if !ok {
		return
	}
	ownerID, ok := h.requireSourceOwner(c, sourceID)
	if !ok {
		return
	}
	var in CreateSourcingSampleInput
	if err := c.ShouldBindJSON(&in); err != nil {
		response.Error(c, http.StatusBadRequest, err.Error())
		return
	}
	item, err := h.service.CreateSourcingSample(ownerID, sourceID, &in)
	if err != nil {
		workflowError(c, err)
		return
	}
	response.Success(c, item)
}

func (h *Handler) TransitionSourcingSample(c *gin.Context) {
	sourceID, ok := parseID(c)
	if !ok {
		return
	}
	sampleID, ok := parseSampleID(c)
	if !ok {
		return
	}
	ownerID, ok := h.requireSourceOwner(c, sourceID)
	if !ok {
		return
	}
	var in TransitionSourcingSampleInput
	if err := c.ShouldBindJSON(&in); err != nil {
		response.Error(c, http.StatusBadRequest, err.Error())
		return
	}
	item, err := h.service.TransitionSourcingSample(ownerID, sourceID, sampleID, &in)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			response.Error(c, http.StatusNotFound, "sample not found")
			return
		}
		workflowError(c, err)
		return
	}
	response.Success(c, item)
}

func (h *Handler) WaiveSourcingSample(c *gin.Context) {
	sourceID, ok := parseID(c)
	if !ok {
		return
	}
	linkID, ok := parseTaskLinkID(c)
	if !ok {
		return
	}
	ownerID, ok := h.requireSourceOwner(c, sourceID)
	if !ok {
		return
	}
	var in WaiveSourcingSampleInput
	if err := c.ShouldBindJSON(&in); err != nil {
		response.Error(c, http.StatusBadRequest, err.Error())
		return
	}
	link, err := h.service.WaiveSourcingSample(ownerID, sourceID, linkID, &in)
	if err != nil {
		workflowError(c, err)
		return
	}
	response.Success(c, link)
}

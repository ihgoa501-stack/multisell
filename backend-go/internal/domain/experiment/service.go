package experiment

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"go.uber.org/zap"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
	"strconv"
	"strings"
	"time"
)

type Service struct {
	db     *gorm.DB
	logger *zap.Logger
}

func NewService(db *gorm.DB, l *zap.Logger) *Service { return &Service{db: db, logger: l} }

var stages = map[string]bool{StageOpportunity: true, StageProduct: true, StageSupply: true, StageChannel: true, StageOrder: true, StageFulfillment: true, StageAftersales: true, StageProfit: true, StageCash: true, StageDecision: true}
var results = map[string]bool{ResultPass: true, ResultConditional: true, ResultReturn: true, ResultReject: true, ResultExpired: true}
var truths = map[string]bool{TruthActual: true, TruthQuoted: true, TruthEstimated: true, TruthUnknown: true, TruthMock: true, TruthInferred: true}
var evidenceKinds = map[string]bool{"support": true, "counter": true, "conflict": true}
var objectTypes = map[string]bool{"candidate": true, "product": true, "product_spec": true, "supplier": true, "sku": true, "purchase": true, "inventory_batch": true, "order": true, "shipment": true, "aftersale": true, "settlement": true, "profit_record": true, "cash_transaction": true}
var decisions = map[string]bool{"": true, "continue": true, "adjust": true, "switch": true, "stop": true}
var statuses = map[string]bool{StatusActive: true, StatusBlocked: true, StatusCompleted: true, StatusStopped: true}
var profitStatuses = map[string]bool{ProfitPending: true, ProfitProvisional: true, ProfitFinal: true}
var cashStatuses = map[string]bool{CashPending: true, CashReceivable: true, CashSettled: true, CashRecovered: true}
var actualRequired = map[string]bool{StageOpportunity: true, StageOrder: true, StageFulfillment: true, StageAftersales: true, StageProfit: true, StageCash: true}
var stageOrder = []string{StageOpportunity, StageProduct, StageSupply, StageChannel, StageOrder, StageFulfillment, StageAftersales, StageProfit, StageCash, StageDecision}
var canonicalGate = map[string]string{
	StageOpportunity: "demand_evidence", StageProduct: "spec_ready", StageSupply: "supply_ready",
	StageChannel: "channel_ready", StageOrder: "paid_order", StageFulfillment: "delivered",
	StageAftersales: "aftersales_closed", StageProfit: "profit_final", StageCash: "cash_recovered",
	StageDecision: "final_decision",
}

func newID() string {
	b := make([]byte, 12)
	_, _ = rand.Read(b)
	return "exp_" + hex.EncodeToString(b)
}
func validateCase(c *ExperimentCase) error {
	if c.OwnerID <= 0 || strings.TrimSpace(c.Name) == "" || !stages[c.Stage] || !decisions[c.FinalDecision] || !statuses[c.Status] || !profitStatuses[c.FinalProfitStatus] || !cashStatuses[c.CashRecoveryStatus] {
		return errors.New("invalid experiment case")
	}
	if c.FinalDecision == "continue" && (c.FinalProfitStatus != ProfitFinal || c.CashRecoveryStatus != CashRecovered) {
		return errors.New("continue requires final profit and recovered cash")
	}
	if c.Status == StatusCompleted && (c.FinalProfitStatus != ProfitFinal || c.CashRecoveryStatus != CashRecovered || c.FinalDecision == "") {
		return errors.New("completed requires final profit, recovered cash, and a final decision")
	}
	return nil
}
func (s *Service) Create(ctx context.Context, c *ExperimentCase) error {
	if c.ExperimentID == "" {
		c.ExperimentID = newID()
	}
	// A client cannot create a case already completed or inject terminal money
	// facts. Every experiment starts before evidence and must earn each gate.
	c.Status = StatusActive
	c.FinalProfitStatus = ProfitPending
	c.FinalRevenue = 0
	c.FinalTotalCost = 0
	c.FinalProfitAmount = 0
	c.ProfitCurrency = ""
	c.CashRecoveryStatus = CashPending
	c.CashRecoveredAmount = 0
	c.CashCurrency = ""
	c.CashRecoveredAt = nil
	c.FinalDecision = ""
	if c.Stage == "" {
		c.Stage = StageOpportunity
	}
	if c.Stage != StageOpportunity {
		return errors.New("new experiments must start at opportunity")
	}
	if err := validateCase(c); err != nil {
		return err
	}
	return s.db.WithContext(ctx).Create(c).Error
}
func (s *Service) Update(ctx context.Context, id string, ownerID int64, c *ExperimentCase) error {
	return s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		var current ExperimentCase
		if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).Where("experiment_id = ? AND owner_id = ?", id, ownerID).First(&current).Error; err != nil {
			return err
		}
		if current.Status == StatusCompleted || current.Status == StatusStopped {
			return errors.New("terminal experiment is immutable")
		}

		// Terminal money facts are written only by passing gates. Validate the
		// requested state against those persisted facts, never client-supplied copies.
		candidate := current
		candidate.Name = c.Name
		candidate.Stage = c.Stage
		candidate.Status = c.Status
		candidate.FinalProfitStatus = c.FinalProfitStatus
		candidate.CashRecoveryStatus = c.CashRecoveryStatus
		candidate.FinalDecision = c.FinalDecision
		if err := validateCase(&candidate); err != nil {
			return err
		}

		passingGate := func(stage string) (bool, error) {
			var gate GateDecision
			err := tx.Where("experiment_id = ? AND stage = ?", id, stage).Order("id DESC").First(&gate).Error
			if errors.Is(err, gorm.ErrRecordNotFound) {
				return false, nil
			}
			return gate.Result == ResultPass, err
		}
		currentIndex, nextIndex := stageIndex(current.Stage), stageIndex(candidate.Stage)
		if nextIndex > currentIndex {
			if nextIndex != currentIndex+1 {
				return errors.New("experiment stages cannot be skipped")
			}
			if ok, err := passingGate(current.Stage); err != nil || !ok {
				return errors.New("advancing requires a passing gate for the current stage")
			}
		}
		if candidate.FinalProfitStatus == ProfitFinal {
			if ok, err := passingGate(StageProfit); err != nil || !ok {
				return errors.New("final profit requires a passing profit gate")
			}
		}
		if candidate.CashRecoveryStatus == CashRecovered {
			if ok, err := passingGate(StageCash); err != nil || !ok {
				return errors.New("cash recovery requires a passing cash gate")
			}
		}
		updates := map[string]any{"name": candidate.Name, "stage": candidate.Stage, "status": candidate.Status, "final_profit_status": candidate.FinalProfitStatus, "cash_recovery_status": candidate.CashRecoveryStatus, "final_decision": candidate.FinalDecision}
		if candidate.FinalDecision == "continue" {
			profit, cash, err := s.validateContinueClosure(tx, &candidate)
			if err != nil {
				return err
			}
			updates["final_revenue"] = profit.Revenue
			updates["final_total_cost"] = profit.TotalCost
			updates["final_profit_amount"] = profit.Profit
			updates["profit_currency"] = profit.Currency
			updates["cash_recovered_amount"] = cash.Amount
			updates["cash_currency"] = cash.Currency
			updates["cash_recovered_at"] = cash.RecoveredAt
		}
		return tx.Model(&ExperimentCase{}).Where("experiment_id = ? AND owner_id = ?", id, ownerID).Updates(updates).Error
	})
}

func (s *Service) validateContinueClosure(db *gorm.DB, c *ExperimentCase) (*profitClosure, *cashClosure, error) {
	profit, err := s.validateProfitClosureWithDB(db, c.ExperimentID)
	if err != nil {
		return nil, nil, err
	}
	cash, err := s.validateCashClosureWithDB(db, c.ExperimentID)
	if err != nil {
		return nil, nil, err
	}
	if profit.Profit <= 0 {
		return nil, nil, errors.New("continue requires positive final profit")
	}
	if strings.TrimSpace(profit.Currency) == "" || profit.Currency != cash.Currency {
		return nil, nil, errors.New("continue requires matching profit and cash currencies")
	}
	if cash.Amount <= 0 || cash.RecoveredAt == nil {
		return nil, nil, errors.New("continue requires an actual positive cash receipt")
	}
	orderID, err := s.linkedNumericIDWithDB(db, c.ExperimentID, "order")
	if err != nil {
		return nil, nil, err
	}
	var order struct {
		PaidAt, DeliveredAt, CancelledAt *time.Time
	}
	if err := db.Clauses(clause.Locking{Strength: "UPDATE"}).Table("sales_order").Where("id = ?", orderID).First(&order).Error; err != nil {
		return nil, nil, errors.New("linked order not found")
	}
	if order.PaidAt == nil || order.DeliveredAt == nil || order.CancelledAt != nil {
		return nil, nil, errors.New("continue requires a paid, delivered, non-cancelled order")
	}
	return profit, cash, nil
}
func stageIndex(stage string) int {
	for i, candidate := range stageOrder {
		if candidate == stage {
			return i
		}
	}
	return -1
}
func (s *Service) hasPassingGate(ctx context.Context, id string, ownerID int64, stage string) (bool, error) {
	if err := s.requireOwner(ctx, id, ownerID); err != nil {
		return false, err
	}
	var g GateDecision
	err := s.db.WithContext(ctx).Where("experiment_id = ? AND stage = ?", id, stage).Order("id DESC").First(&g).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return false, nil
	}
	return g.Result == ResultPass, err
}
func (s *Service) requireOwner(ctx context.Context, id string, ownerID int64) error {
	var count int64
	if err := s.db.WithContext(ctx).Model(&ExperimentCase{}).Where("experiment_id = ? AND owner_id = ?", id, ownerID).Count(&count).Error; err != nil {
		return err
	}
	if count != 1 {
		return gorm.ErrRecordNotFound
	}
	return nil
}
func (s *Service) Delete(ctx context.Context, id string, ownerID int64) error {
	if err := s.requireOwner(ctx, id, ownerID); err != nil {
		return err
	}
	return s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		for _, m := range []any{&GateDecision{}, &EvidenceRecord{}, &ObjectLink{}} {
			if err := tx.Where("experiment_id = ?", id).Delete(m).Error; err != nil {
				return err
			}
		}
		return tx.Where("experiment_id = ?", id).Delete(&ExperimentCase{}).Error
	})
}
func (s *Service) List(ctx context.Context, ownerID int64, page, size int) ([]ExperimentCase, int64, error) {
	if page < 1 {
		page = 1
	}
	if size < 1 {
		size = 20
	}
	var x []ExperimentCase
	var n int64
	q := s.db.WithContext(ctx).Model(&ExperimentCase{}).Where("owner_id = ?", ownerID)
	if err := q.Count(&n).Error; err != nil {
		return nil, 0, err
	}
	err := q.Order("id desc").Offset((page - 1) * size).Limit(size).Find(&x).Error
	return x, n, err
}
func (s *Service) GetDetail(ctx context.Context, id string, ownerID int64) (*Detail, error) {
	var d Detail
	if err := s.db.WithContext(ctx).Where("experiment_id = ? AND owner_id = ?", id, ownerID).First(&d.Case).Error; err != nil {
		return nil, err
	}
	s.db.WithContext(ctx).Where("experiment_id = ?", id).Order("id").Find(&d.Gates)
	s.db.WithContext(ctx).Where("experiment_id = ?", id).Order("id").Find(&d.Evidence)
	s.db.WithContext(ctx).Where("experiment_id = ?", id).Order("id").Find(&d.ObjectLinks)
	return &d, nil
}
func (s *Service) AddEvidence(ctx context.Context, ownerID int64, e *EvidenceRecord) error {
	var c ExperimentCase
	if err := s.db.WithContext(ctx).Where("experiment_id = ? AND owner_id = ?", e.ExperimentID, ownerID).First(&c).Error; err != nil {
		return err
	}
	if e.Stage != c.Stage {
		return errors.New("evidence must be added to the current experiment stage")
	}
	if e.EvidenceKind == "" {
		e.EvidenceKind = "support"
	}
	if !stages[e.Stage] || !evidenceKinds[e.EvidenceKind] || !truths[e.TruthStatus] || e.ExperimentID == "" || strings.TrimSpace(e.Title) == "" {
		return errors.New("invalid evidence")
	}
	if e.TruthStatus == TruthActual {
		return errors.New("actual evidence requires explicit owner verification")
	}
	return s.db.WithContext(ctx).Create(e).Error
}
func (s *Service) VerifyEvidence(ctx context.Context, id string, evidenceID, ownerID int64) (*EvidenceRecord, error) {
	if err := s.requireOwner(ctx, id, ownerID); err != nil {
		return nil, err
	}
	var e EvidenceRecord
	if err := s.db.WithContext(ctx).Where("id = ? AND experiment_id = ?", evidenceID, id).First(&e).Error; err != nil {
		return nil, err
	}
	if e.TruthStatus != TruthUnknown {
		return nil, errors.New("only unverified evidence can be promoted to actual")
	}
	if strings.TrimSpace(e.SourceURI) == "" || e.ObservedAt == nil {
		return nil, errors.New("actual evidence requires source_uri and observed_at")
	}
	now := time.Now()
	if err := s.db.WithContext(ctx).Model(&e).Updates(map[string]any{"truth_status": TruthActual, "verified_by": ownerID, "verified_at": &now}).Error; err != nil {
		return nil, err
	}
	e.TruthStatus, e.VerifiedBy, e.VerifiedAt = TruthActual, ownerID, &now
	return &e, nil
}
func (s *Service) AddObjectLink(ctx context.Context, ownerID int64, l *ObjectLink) error {
	if l.ExperimentID == "" || !objectTypes[l.ObjectType] || strings.TrimSpace(l.ObjectID) == "" {
		return errors.New("invalid object link")
	}
	return s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		var c ExperimentCase
		if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).Where("experiment_id = ? AND owner_id = ?", l.ExperimentID, ownerID).First(&c).Error; err != nil {
			return err
		}
		if c.Status == StatusCompleted || c.Status == StatusStopped {
			return errors.New("terminal experiment links are immutable")
		}
		return tx.Create(l).Error
	})
}
func (s *Service) EvaluateGate(ctx context.Context, id string, ownerID int64, in GateInput) (*GateDecision, error) {
	var c ExperimentCase
	if err := s.db.WithContext(ctx).Where("experiment_id = ? AND owner_id = ?", id, ownerID).First(&c).Error; err != nil {
		return nil, err
	}
	if !stages[in.Stage] || !results[in.Result] || in.GateCode == "" {
		return nil, errors.New("invalid gate decision")
	}
	if in.Stage != c.Stage || canonicalGate[in.Stage] != in.GateCode {
		return nil, errors.New("gate must match the current experiment stage")
	}
	if in.Result == ResultPass && len(in.EvidenceIDs) == 0 {
		return nil, errors.New("passing a gate requires evidence")
	}
	if in.Result == ResultPass {
		var ev []EvidenceRecord
		if err := s.db.WithContext(ctx).Where("experiment_id = ? AND stage = ? AND id IN ?", id, in.Stage, in.EvidenceIDs).Find(&ev).Error; err != nil {
			return nil, err
		}
		if len(ev) != len(in.EvidenceIDs) {
			return nil, errors.New("evidence missing")
		}
		hasSupport, hasCounter := false, false
		for _, e := range ev {
			if e.ExpiresAt != nil && !e.ExpiresAt.After(time.Now()) {
				return nil, errors.New("expired evidence cannot pass a gate")
			}
			if e.TruthStatus == TruthMock || e.TruthStatus == TruthInferred || e.TruthStatus == TruthUnknown {
				return nil, fmt.Errorf("%s evidence cannot pass high-risk gate", e.TruthStatus)
			}
			if e.TruthStatus == TruthActual && (e.VerifiedBy != ownerID || e.VerifiedAt == nil) {
				return nil, errors.New("actual evidence is not owner verified")
			}
			if actualRequired[in.Stage] && e.TruthStatus != TruthActual {
				return nil, fmt.Errorf("%s gate requires actual evidence", in.Stage)
			}
			hasSupport = hasSupport || e.EvidenceKind == "support"
			hasCounter = hasCounter || e.EvidenceKind == "counter"
		}
		if in.Stage == StageOpportunity && (!hasSupport || !hasCounter) {
			return nil, errors.New("opportunity pass requires both support and counter evidence")
		}
	}
	raw, _ := json.Marshal(in.EvidenceIDs)
	g := &GateDecision{ExperimentID: id, Stage: in.Stage, GateCode: in.GateCode, Result: in.Result, Reason: in.Reason, EvidenceIDs: string(raw), DecidedBy: in.DecidedBy}
	err := s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		var caseUpdates map[string]any
		if in.Result == ResultPass && in.Stage == StageProfit {
			truth, err := s.validateProfitClosureWithDB(tx, id)
			if err != nil {
				return err
			}
			caseUpdates = map[string]any{"final_revenue": truth.Revenue, "final_total_cost": truth.TotalCost, "final_profit_amount": truth.Profit, "profit_currency": truth.Currency}
		}
		if in.Result == ResultPass && in.Stage == StageCash {
			truth, err := s.validateCashClosureWithDB(tx, id)
			if err != nil {
				return err
			}
			caseUpdates = map[string]any{"cash_recovered_amount": truth.Amount, "cash_currency": truth.Currency, "cash_recovered_at": truth.RecoveredAt}
		}
		if err := tx.Create(g).Error; err != nil {
			return err
		}
		if len(caseUpdates) > 0 {
			return tx.Model(&ExperimentCase{}).Where("experiment_id = ? AND owner_id = ?", id, ownerID).Updates(caseUpdates).Error
		}
		return nil
	})
	return g, err
}

type profitClosure struct {
	Revenue, TotalCost, Profit float64
	Currency                   string
}
type cashClosure struct {
	Amount      float64
	Currency    string
	RecoveredAt *time.Time
}

func (s *Service) linkedNumericID(ctx context.Context, experimentID, objectType string) (int64, error) {
	return s.linkedNumericIDWithDB(s.db.WithContext(ctx), experimentID, objectType)
}

func (s *Service) linkedNumericIDWithDB(db *gorm.DB, experimentID, objectType string) (int64, error) {
	var link ObjectLink
	if err := db.Clauses(clause.Locking{Strength: "UPDATE"}).Where("experiment_id = ? AND object_type = ?", experimentID, objectType).Order("id DESC").First(&link).Error; err != nil {
		return 0, fmt.Errorf("%s link required: %w", objectType, err)
	}
	id, err := strconv.ParseInt(link.ObjectID, 10, 64)
	if err != nil || id <= 0 {
		return 0, fmt.Errorf("invalid %s link", objectType)
	}
	return id, nil
}

func (s *Service) validateProfitClosure(ctx context.Context, experimentID string) (*profitClosure, error) {
	return s.validateProfitClosureWithDB(s.db.WithContext(ctx), experimentID)
}

func (s *Service) validateProfitClosureWithDB(db *gorm.DB, experimentID string) (*profitClosure, error) {
	settlementID, err := s.linkedNumericIDWithDB(db, experimentID, "settlement")
	if err != nil {
		return nil, err
	}
	profitID, err := s.linkedNumericIDWithDB(db, experimentID, "profit_record")
	if err != nil {
		return nil, err
	}
	orderID, err := s.linkedNumericIDWithDB(db, experimentID, "order")
	if err != nil {
		return nil, err
	}
	var st struct {
		ID                           int64
		Status, SourceType, Currency string
		ImportedAt                   *time.Time
	}
	if err := db.Clauses(clause.Locking{Strength: "UPDATE"}).Table("settlement").Where("id = ?", settlementID).First(&st).Error; err != nil {
		return nil, errors.New("linked settlement not found")
	}
	if (st.Status != "reconciled" && st.Status != "closed") || (st.SourceType != "platform_import" && st.SourceType != "api_sync") || st.ImportedAt == nil {
		return nil, errors.New("linked settlement is not trusted and reconciled")
	}
	var lockedItems []struct{ ID int64 }
	if err := db.Clauses(clause.Locking{Strength: "UPDATE"}).Table("settlement_item").Where("settlement_id = ?", settlementID).Find(&lockedItems).Error; err != nil {
		return nil, err
	}
	var itemCount, unmatched int64
	q := db.Table("settlement_item").Where("settlement_id = ?", settlementID)
	if err := q.Count(&itemCount).Error; err != nil {
		return nil, err
	}
	if err := q.Where("reconciliation_status <> ? OR reconciled_at IS NULL OR reconciled_by = ''", "matched").Count(&unmatched).Error; err != nil {
		return nil, err
	}
	if itemCount == 0 || unmatched != 0 {
		return nil, errors.New("linked settlement items are not fully reconciled")
	}
	var matched int64
	if err := db.Table("settlement_item AS si").Where("si.settlement_id = ? AND (si.order_id = ? OR si.order_no = (SELECT order_no FROM sales_order WHERE id = ?))", settlementID, orderID, orderID).Count(&matched).Error; err != nil {
		return nil, err
	}
	if matched == 0 {
		return nil, errors.New("settlement is not linked to the experiment order")
	}
	var p struct {
		ID, OrderID                int64
		Revenue, TotalCost, Profit float64
		ProfitStatus, MissingCosts string
	}
	if err := db.Clauses(clause.Locking{Strength: "UPDATE"}).Table("order_profit_record").Where("id = ?", profitID).First(&p).Error; err != nil {
		return nil, errors.New("linked profit record not found")
	}
	if p.OrderID != orderID || p.ProfitStatus != "final" || strings.TrimSpace(p.MissingCosts) != "" {
		return nil, errors.New("linked profit record is not final for the experiment order")
	}
	return &profitClosure{Revenue: p.Revenue, TotalCost: p.TotalCost, Profit: p.Profit, Currency: st.Currency}, nil
}

func (s *Service) validateCashClosure(ctx context.Context, experimentID string) (*cashClosure, error) {
	return s.validateCashClosureWithDB(s.db.WithContext(ctx), experimentID)
}

func (s *Service) validateCashClosureWithDB(db *gorm.DB, experimentID string) (*cashClosure, error) {
	transactionID, err := s.linkedNumericIDWithDB(db, experimentID, "cash_transaction")
	if err != nil {
		return nil, err
	}
	settlementID, err := s.linkedNumericIDWithDB(db, experimentID, "settlement")
	if err != nil {
		return nil, err
	}
	orderID, err := s.linkedNumericIDWithDB(db, experimentID, "order")
	if err != nil {
		return nil, err
	}
	var settlementCurrency string
	if err := db.Clauses(clause.Locking{Strength: "UPDATE"}).Table("settlement").Where("id = ?", settlementID).Pluck("currency", &settlementCurrency).Error; err != nil {
		return nil, errors.New("linked settlement not found")
	}
	var row struct {
		ID                                     int64
		Amount                                 float64
		Currency, TransactionType, AccountType string
		TransactionDate                        *time.Time
		SettlementID, OrderID                  *int64
	}
	if err := db.Clauses(clause.Locking{Strength: "UPDATE"}).Table("finance_transaction AS ft").Select("ft.*, fa.account_type").Joins("JOIN finance_account AS fa ON fa.id = ft.account_id").Where("ft.id = ?", transactionID).First(&row).Error; err != nil {
		return nil, errors.New("linked cash transaction not found")
	}
	if row.Amount <= 0 || row.TransactionDate == nil || (row.AccountType != "bank" && row.AccountType != "cash") || row.TransactionType != "revenue" {
		return nil, errors.New("linked cash transaction is not an actual available receipt")
	}
	if row.SettlementID == nil || *row.SettlementID != settlementID || row.OrderID == nil || *row.OrderID != orderID {
		return nil, errors.New("cash transaction is not linked to the experiment order and settlement")
	}
	if strings.TrimSpace(row.Currency) == "" || row.Currency != settlementCurrency {
		return nil, errors.New("cash transaction currency does not match the linked settlement")
	}
	return &cashClosure{Amount: row.Amount, Currency: row.Currency, RecoveredAt: row.TransactionDate}, nil
}

func (s *Service) OwnerSummary(ctx context.Context, id string, ownerID int64) (*OwnerSummary, error) {
	d, err := s.GetDetail(ctx, id, ownerID)
	if err != nil {
		return nil, err
	}
	o := &OwnerSummary{ExperimentID: id, Stage: d.Case.Stage, FinalProfitStatus: d.Case.FinalProfitStatus, FinalRevenue: d.Case.FinalRevenue, FinalTotalCost: d.Case.FinalTotalCost, FinalProfitAmount: d.Case.FinalProfitAmount, ProfitCurrency: d.Case.ProfitCurrency, CashRecoveryStatus: d.Case.CashRecoveryStatus, CashRecoveredAmount: d.Case.CashRecoveredAmount, CashCurrency: d.Case.CashCurrency, CashRecoveredAt: d.Case.CashRecoveredAt, FinalDecision: d.Case.FinalDecision}
	latest := map[string]GateDecision{}
	for _, g := range d.Gates {
		latest[g.Stage] = g
	}
	for _, g := range latest {
		if g.Result == ResultPass {
			o.PassedGates++
		} else {
			o.Blockers = append(o.Blockers, g.GateCode+":"+g.Result)
		}
	}
	for _, stage := range stageOrder {
		if _, ok := latest[stage]; !ok {
			o.Blockers = append(o.Blockers, stage+":gate_missing")
		}
	}
	return o, nil
}

package productimage

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"strconv"
	"strings"
	"time"

	"github.com/lingmirror/backend-go/internal/imageservice"
	"github.com/shopspring/decimal"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

type BudgetPolicyInput struct {
	Currency       string    `json:"currency" binding:"required"`
	PeriodStart    time.Time `json:"period_start" binding:"required"`
	PeriodEnd      time.Time `json:"period_end" binding:"required"`
	TotalAmount    string    `json:"total_amount" binding:"required"`
	IdempotencyKey string    `json:"idempotency_key" binding:"required"`
}

type BudgetChargeInput struct {
	Amount         string    `json:"amount" binding:"required"`
	Currency       string    `json:"currency" binding:"required"`
	EvidenceSHA    string    `json:"evidence_sha256" binding:"required"`
	ObservedAt     time.Time `json:"observed_at" binding:"required"`
	IdempotencyKey string    `json:"idempotency_key" binding:"required"`
	Resolution     string    `json:"resolution,omitempty"`
}

type BudgetNoChargeInput struct {
	EvidenceSHA    string    `json:"evidence_sha256" binding:"required"`
	ObservedAt     time.Time `json:"observed_at" binding:"required"`
	Reason         string    `json:"reason" binding:"required"`
	IdempotencyKey string    `json:"idempotency_key" binding:"required"`
}

func budgetHash(v any) string {
	b, _ := json.Marshal(v)
	s := sha256.Sum256(b)
	return hex.EncodeToString(s[:])
}

func strictMoney(v string) (decimal.Decimal, bool) {
	v = strings.TrimSpace(v)
	if !moneyPattern.MatchString(v) {
		return decimal.Zero, false
	}
	d, err := decimal.NewFromString(v)
	return d, err == nil && d.GreaterThan(decimal.Zero)
}

func budgetLock(tx *gorm.DB, ownerID int64, currency string) error {
	if tx.Dialector.Name() != "postgres" {
		return nil
	}
	// Serializes every policy/reservation/reconciliation mutation for the same
	// Owner and currency, including periods with no row yet.
	return tx.Exec("SELECT pg_advisory_xact_lock(hashtextextended(?, 0))", strings.Join([]string{"product-image-budget", currency, decimal.NewFromInt(ownerID).String()}, ":")).Error
}

func (s *Service) CreateBudgetPolicy(ctx context.Context, ownerID int64, in BudgetPolicyInput) (*BudgetPolicy, error) {
	in.Currency = strings.ToUpper(strings.TrimSpace(in.Currency))
	in.TotalAmount, in.IdempotencyKey = strings.TrimSpace(in.TotalAmount), strings.TrimSpace(in.IdempotencyKey)
	if ownerID <= 0 || !allowedExecutionCurrency(in.Currency) || in.IdempotencyKey == "" || !in.PeriodEnd.After(in.PeriodStart) {
		return nil, ErrInvalidInput
	}
	if _, ok := strictMoney(in.TotalAmount); !ok {
		return nil, ErrInvalidInput
	}
	in.PeriodStart, in.PeriodEnd = in.PeriodStart.UTC(), in.PeriodEnd.UTC()
	hash := budgetHash(in)
	var out BudgetPolicy
	err := s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err := budgetLock(tx, ownerID, in.Currency); err != nil {
			return err
		}
		var existing BudgetPolicy
		err := tx.Where("owner_id=? AND idempotency_key=?", ownerID, in.IdempotencyKey).First(&existing).Error
		if err == nil {
			if existing.RequestHash != hash {
				return &ConflictError{Code: "IDEMPOTENCY_CONFLICT"}
			}
			out = existing
			return nil
		}
		if !errors.Is(err, gorm.ErrRecordNotFound) {
			return err
		}
		var overlaps int64
		if err := tx.Model(&BudgetPolicy{}).Where("owner_id=? AND currency=? AND period_start < ? AND period_end > ?", ownerID, in.Currency, in.PeriodEnd, in.PeriodStart).Count(&overlaps).Error; err != nil {
			return err
		}
		if overlaps != 0 {
			return &ConflictError{Code: "BUDGET_PERIOD_OVERLAP"}
		}
		out = BudgetPolicy{OwnerID: ownerID, Currency: in.Currency, PeriodStart: in.PeriodStart, PeriodEnd: in.PeriodEnd, TotalAmount: in.TotalAmount, IdempotencyKey: in.IdempotencyKey, RequestHash: hash}
		return tx.Create(&out).Error
	})
	return &out, err
}

func (s *Service) ListBudgetPolicies(ctx context.Context, ownerID int64) ([]BudgetPolicy, error) {
	var out []BudgetPolicy
	err := s.db.WithContext(ctx).Where("owner_id=?", ownerID).Order("period_start DESC,id DESC").Find(&out).Error
	if out == nil {
		out = []BudgetPolicy{}
	}
	return out, err
}

func (s *Service) ListBudgetReservations(ctx context.Context, ownerID int64) ([]BudgetReservation, error) {
	var out []BudgetReservation
	err := s.db.WithContext(ctx).Where("owner_id=?", ownerID).Order("id DESC").Find(&out).Error
	if out == nil {
		out = []BudgetReservation{}
	}
	return out, err
}

func policyExposure(tx *gorm.DB, policyID int64) (decimal.Decimal, error) {
	var reserved string
	if tx.Dialector.Name() == "postgres" {
		if err := tx.Raw("SELECT COALESCE(SUM(reserved_amount),0)::text FROM product_image_budget_reservations WHERE policy_id=? AND state IN ('reserved','claimed')", policyID).Scan(&reserved).Error; err != nil {
			return decimal.Zero, err
		}
	} else {
		if err := tx.Raw("SELECT COALESCE(SUM(reserved_amount),0) FROM product_image_budget_reservations WHERE policy_id=? AND state IN ('reserved','claimed')", policyID).Scan(&reserved).Error; err != nil {
			return decimal.Zero, err
		}
	}
	var spent string
	query := "SELECT COALESCE(SUM(c.amount),0) FROM product_image_budget_charges c JOIN product_image_budget_reservations r ON r.id=c.reservation_id WHERE r.policy_id=?"
	if err := tx.Raw(query, policyID).Scan(&spent).Error; err != nil {
		return decimal.Zero, err
	}
	r, err := decimal.NewFromString(reserved)
	if err != nil {
		return decimal.Zero, err
	}
	p, err := decimal.NewFromString(spent)
	if err != nil {
		return decimal.Zero, err
	}
	return r.Add(p), nil
}

func releaseExpiredReservations(tx *gorm.DB, ownerID int64, currency string, now time.Time) error {
	// Only never-claimed reservations are eligible. Claimed calls include lost
	// or unknown provider responses and deliberately remain locked.
	if tx.Dialector.Name() == "postgres" {
		return tx.Exec(`UPDATE product_image_budget_reservations r SET state='released',released_at=?,release_reason='approval_expired_before_claim',updated_at=?
			WHERE r.owner_id=? AND r.currency=? AND r.state='reserved' AND EXISTS
			(SELECT 1 FROM product_image_execution_approvals a WHERE a.id=r.approval_id AND a.consumed_at IS NULL AND a.expires_at <= ?)`, now, now, ownerID, currency, now).Error
	}
	return tx.Exec(`UPDATE product_image_budget_reservations SET state='released',released_at=?,release_reason='approval_expired_before_claim',updated_at=?
		WHERE owner_id=? AND currency=? AND state='reserved' AND approval_id IN
		(SELECT id FROM product_image_execution_approvals WHERE consumed_at IS NULL AND expires_at <= ?)`, now, now, ownerID, currency, now).Error
}

func (s *Service) ReleaseBudgetReservation(ctx context.Context, ownerID, reservationID int64, reason string) (*BudgetReservation, error) {
	reason = strings.TrimSpace(reason)
	if ownerID <= 0 || reservationID <= 0 || reason == "" {
		return nil, ErrInvalidInput
	}
	var out BudgetReservation
	err := s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).Where("id=? AND owner_id=?", reservationID, ownerID).First(&out).Error; err != nil {
			return err
		}
		if out.State == "released" {
			return nil
		}
		if out.State != "reserved" {
			return &ConflictError{Code: "BUDGET_RELEASE_FORBIDDEN"}
		}
		var approval ExecutionApproval
		if err := tx.First(&approval, out.ApprovalID).Error; err != nil {
			return err
		}
		// An unexpired approval is released only by an explicit Owner cancellation;
		// expiry is also safe because the execution was never claimed.
		now := time.Now().UTC()
		out.State, out.ReleasedAt, out.ReleaseReason = "released", &now, reason
		return tx.Model(&BudgetReservation{}).Where("id=? AND state='reserved'", out.ID).Updates(map[string]any{"state": "released", "released_at": now, "release_reason": reason, "updated_at": now}).Error
	})
	return &out, err
}

func (s *Service) ReconcileBudgetCharge(ctx context.Context, ownerID, reservationID int64, in BudgetChargeInput) (*BudgetCharge, error) {
	in.Amount, in.Currency, in.EvidenceSHA, in.IdempotencyKey, in.Resolution = strings.TrimSpace(in.Amount), strings.ToUpper(strings.TrimSpace(in.Currency)), strings.ToLower(strings.TrimSpace(in.EvidenceSHA)), strings.TrimSpace(in.IdempotencyKey), strings.TrimSpace(in.Resolution)
	amount, ok := strictMoney(in.Amount)
	if ownerID <= 0 || reservationID <= 0 || !ok || !allowedExecutionCurrency(in.Currency) || !sha256Pattern.MatchString(in.EvidenceSHA) || in.IdempotencyKey == "" || in.ObservedAt.IsZero() || (in.Resolution != "" && in.Resolution != "charged_no_output") {
		return nil, ErrInvalidInput
	}
	in.ObservedAt = in.ObservedAt.UTC()
	hash := budgetHash(in)
	var replay BudgetCharge
	if err := s.db.WithContext(ctx).Where("owner_id=? AND idempotency_key=?", ownerID, in.IdempotencyKey).First(&replay).Error; err == nil {
		if replay.RequestHash != hash || replay.ReservationID != reservationID {
			return nil, &ConflictError{Code: "IDEMPOTENCY_CONFLICT"}
		}
		return &replay, nil
	} else if !errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, err
	}
	if in.Resolution == "charged_no_output" {
		if _, _, err := s.quiesceBudgetReconcileTask(ctx, ownerID, reservationID, "claimed"); err != nil {
			return nil, err
		}
	}
	var out BudgetCharge
	err := s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		var replay BudgetCharge
		err := tx.Where("owner_id=? AND idempotency_key=?", ownerID, in.IdempotencyKey).First(&replay).Error
		if err == nil {
			if replay.RequestHash != hash || replay.ReservationID != reservationID {
				return &ConflictError{Code: "IDEMPOTENCY_CONFLICT"}
			}
			out = replay
			return nil
		}
		if !errors.Is(err, gorm.ErrRecordNotFound) {
			return err
		}
		var r BudgetReservation
		if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).Where("id=? AND owner_id=?", reservationID, ownerID).First(&r).Error; err != nil {
			return err
		}
		if err := budgetLock(tx, ownerID, r.Currency); err != nil {
			return err
		}
		if r.Currency != in.Currency || (r.State != "claimed" && r.State != "spent" && r.State != "no_charge") {
			return &ConflictError{Code: "BUDGET_RECONCILE_FORBIDDEN"}
		}
		var previous string
		if err := tx.Raw("SELECT COALESCE(SUM(amount),0) FROM product_image_budget_charges WHERE reservation_id=?", r.ID).Scan(&previous).Error; err != nil {
			return err
		}
		prev, err := decimal.NewFromString(previous)
		if err != nil {
			return err
		}
		var policy BudgetPolicy
		if err := tx.First(&policy, r.PolicyID).Error; err != nil {
			return err
		}
		total, _ := decimal.NewFromString(policy.TotalAmount)
		exposure, err := policyExposure(tx, policy.ID)
		if err != nil {
			return err
		}
		// For the first charge, exposure currently includes the full reservation;
		// replace it with actual cost. Late fees append to already-spent exposure.
		projected := exposure.Add(amount)
		kind := "late_fee"
		deltaAmount := amount
		if prev.IsZero() && r.State != "no_charge" {
			projected = exposure.Sub(decimal.RequireFromString(r.ReservedAmount)).Add(amount)
			kind = "actual"
			deltaAmount = amount.Sub(decimal.RequireFromString(r.ReservedAmount))
		}
		if in.Resolution == "charged_no_output" {
			if r.State != "claimed" {
				return &ConflictError{Code: "BUDGET_RECONCILE_FORBIDDEN"}
			}
			var task Task
			if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).Where("id=? AND owner_id=? AND version=? AND manifest_hash=?", r.TaskID, ownerID, r.TaskVersion, r.ManifestHash).First(&task).Error; err != nil || !unresolvedBudgetTask(&task) || in.ObservedAt.Before(task.UpdatedAt) {
				return &ConflictError{Code: "BUDGET_RECONCILE_FORBIDDEN"}
			}
			kind = "charged_no_output"
		}
		out = BudgetCharge{OwnerID: ownerID, ReservationID: r.ID, Amount: in.Amount, DeltaAmount: deltaAmount.StringFixed(4), Currency: in.Currency, Kind: kind, OverBudget: projected.GreaterThan(total), EvidenceSHA: in.EvidenceSHA, ObservedAt: in.ObservedAt, IdempotencyKey: in.IdempotencyKey, RequestHash: hash}
		if err := tx.Create(&out).Error; err != nil {
			return err
		}
		if r.State == "no_charge" {
			result := tx.Model(&Task{}).Where("id=? AND owner_id=? AND version=? AND manifest_hash=? AND status='FAILED' AND error_code='NO_CHARGE_CONFIRMED'", r.TaskID, ownerID, r.TaskVersion, r.ManifestHash).Updates(map[string]any{"error_code": "CHARGED_OUTPUT_UNRECOVERABLE", "updated_at": time.Now().UTC()})
			if result.Error != nil {
				return result.Error
			}
			if result.RowsAffected != 1 {
				return &ConflictError{Code: "BUDGET_RECONCILE_FORBIDDEN"}
			}
		}
		if in.Resolution == "charged_no_output" {
			result := tx.Model(&Task{}).Where("id=? AND owner_id=? AND version=? AND manifest_hash=? AND status IN ? AND COALESCE(error_code, '') NOT IN ?", r.TaskID, ownerID, r.TaskVersion, r.ManifestHash, []string{"RECONCILE_REQUIRED", "FAILED"}, []string{"NO_CHARGE_CONFIRMED", "CHARGED_OUTPUT_UNRECOVERABLE"}).Updates(map[string]any{"status": "FAILED", "error_code": "CHARGED_OUTPUT_UNRECOVERABLE", "updated_at": time.Now().UTC()})
			if result.Error != nil {
				return result.Error
			}
			if result.RowsAffected != 1 {
				return &ConflictError{Code: "BUDGET_RECONCILE_FORBIDDEN"}
			}
		}
		if r.State == "claimed" || r.State == "no_charge" {
			return tx.Model(&BudgetReservation{}).Where("id=? AND state=?", r.ID, r.State).Updates(map[string]any{"state": "spent", "released_at": nil, "release_reason": "", "updated_at": time.Now().UTC()}).Error
		}
		return nil
	})
	return &out, err
}

func (s *Service) quiesceBudgetReconcileTask(ctx context.Context, ownerID, reservationID int64, reservationState string) (*BudgetReservation, *Task, error) {
	var reservation BudgetReservation
	if err := s.db.WithContext(ctx).Where("id=? AND owner_id=? AND state=?", reservationID, ownerID, reservationState).First(&reservation).Error; err != nil {
		return nil, nil, err
	}
	var task Task
	if err := s.db.WithContext(ctx).Where("id=? AND owner_id=? AND version=? AND manifest_hash=?", reservation.TaskID, ownerID, reservation.TaskVersion, reservation.ManifestHash).First(&task).Error; err != nil {
		return nil, nil, err
	}
	if !unresolvedBudgetTask(&task) {
		return nil, nil, &ConflictError{Code: "BUDGET_RECONCILE_FORBIDDEN"}
	}
	quiescer, ok := s.image.(interface {
		QuiesceJob(context.Context, string, imageservice.QuiesceJobRequest) (*imageservice.Job, error)
	})
	if !ok {
		return nil, nil, &ConflictError{Code: "BUDGET_RECONCILE_FORBIDDEN"}
	}
	remote, err := quiescer.QuiesceJob(ctx, task.ImageServiceJobID, imageservice.QuiesceJobRequest{OwnerID: ownerID, LingMirrorTaskID: strconv.FormatInt(task.ID, 10), LingMirrorTaskVersion: task.Version, ManifestHash: task.ManifestHash})
	if err != nil || remote == nil || (remote.Status != "FAILED" && remote.Status != "RECONCILE_REQUIRED") {
		return nil, nil, &ConflictError{Code: "BUDGET_RECONCILE_FORBIDDEN"}
	}
	return &reservation, &task, nil
}

func unresolvedBudgetTask(task *Task) bool {
	return task != nil && (task.Status == "RECONCILE_REQUIRED" || task.Status == "FAILED") && task.ErrorCode != "NO_CHARGE_CONFIRMED" && task.ErrorCode != "CHARGED_OUTPUT_UNRECOVERABLE"
}

func (s *Service) ReconcileBudgetNoCharge(ctx context.Context, ownerID, reservationID int64, in BudgetNoChargeInput) (*BudgetCharge, error) {
	in.EvidenceSHA, in.Reason, in.IdempotencyKey = strings.ToLower(strings.TrimSpace(in.EvidenceSHA)), strings.TrimSpace(in.Reason), strings.TrimSpace(in.IdempotencyKey)
	if ownerID <= 0 || reservationID <= 0 || !sha256Pattern.MatchString(in.EvidenceSHA) || in.ObservedAt.IsZero() || in.Reason == "" || len(in.Reason) > 1000 || in.IdempotencyKey == "" {
		return nil, ErrInvalidInput
	}
	in.ObservedAt = in.ObservedAt.UTC()
	hash := budgetHash(in)
	var replay BudgetCharge
	if err := s.db.WithContext(ctx).Where("owner_id=? AND idempotency_key=?", ownerID, in.IdempotencyKey).First(&replay).Error; err == nil {
		if replay.RequestHash != hash || replay.ReservationID != reservationID || replay.Kind != "no_charge" {
			return nil, &ConflictError{Code: "IDEMPOTENCY_CONFLICT"}
		}
		return &replay, nil
	} else if !errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, err
	}
	if _, _, err := s.quiesceBudgetReconcileTask(ctx, ownerID, reservationID, "claimed"); err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, err
		}
		return nil, &ConflictError{Code: "BUDGET_NO_CHARGE_FORBIDDEN"}
	}
	var out BudgetCharge
	err := s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		var replay BudgetCharge
		err := tx.Where("owner_id=? AND idempotency_key=?", ownerID, in.IdempotencyKey).First(&replay).Error
		if err == nil {
			if replay.RequestHash != hash || replay.ReservationID != reservationID || replay.Kind != "no_charge" {
				return &ConflictError{Code: "IDEMPOTENCY_CONFLICT"}
			}
			out = replay
			return nil
		}
		if !errors.Is(err, gorm.ErrRecordNotFound) {
			return err
		}
		var scope BudgetReservation
		if err := tx.Select("id", "currency").Where("id=? AND owner_id=?", reservationID, ownerID).First(&scope).Error; err != nil {
			return err
		}
		if err := budgetLock(tx, ownerID, scope.Currency); err != nil {
			return err
		}
		var r BudgetReservation
		if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).Where("id=? AND owner_id=?", reservationID, ownerID).First(&r).Error; err != nil {
			return err
		}
		if r.State != "claimed" {
			return &ConflictError{Code: "BUDGET_NO_CHARGE_FORBIDDEN"}
		}
		var task Task
		if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).Where("id=? AND owner_id=? AND version=? AND manifest_hash=?", r.TaskID, ownerID, r.TaskVersion, r.ManifestHash).First(&task).Error; err != nil {
			return err
		}
		if !unresolvedBudgetTask(&task) || in.ObservedAt.Before(task.UpdatedAt) {
			return &ConflictError{Code: "BUDGET_NO_CHARGE_FORBIDDEN"}
		}
		var chargeCount int64
		if err := tx.Model(&BudgetCharge{}).Where("reservation_id=?", r.ID).Count(&chargeCount).Error; err != nil {
			return err
		}
		if chargeCount != 0 {
			return &ConflictError{Code: "BUDGET_NO_CHARGE_FORBIDDEN"}
		}
		out = BudgetCharge{OwnerID: ownerID, ReservationID: r.ID, Amount: "0", DeltaAmount: decimal.RequireFromString(r.ReservedAmount).Neg().StringFixed(4), Currency: r.Currency, Kind: "no_charge", EvidenceSHA: in.EvidenceSHA, ObservedAt: in.ObservedAt, IdempotencyKey: in.IdempotencyKey, RequestHash: hash}
		if err := tx.Create(&out).Error; err != nil {
			return err
		}
		now := time.Now().UTC()
		if result := tx.Model(&BudgetReservation{}).Where("id=? AND owner_id=? AND state='claimed'", r.ID, ownerID).Updates(map[string]any{"state": "no_charge", "released_at": now, "release_reason": in.Reason, "updated_at": now}); result.Error != nil || result.RowsAffected != 1 {
			if result.Error != nil {
				return result.Error
			}
			return &ConflictError{Code: "BUDGET_NO_CHARGE_FORBIDDEN"}
		}
		result := tx.Model(&Task{}).Where("id=? AND owner_id=? AND version=? AND manifest_hash=? AND status IN ? AND COALESCE(error_code, '') NOT IN ?", task.ID, ownerID, task.Version, task.ManifestHash, []string{"RECONCILE_REQUIRED", "FAILED"}, []string{"NO_CHARGE_CONFIRMED", "CHARGED_OUTPUT_UNRECOVERABLE"}).Updates(map[string]any{"status": "FAILED", "error_code": "NO_CHARGE_CONFIRMED", "updated_at": now})
		if result.Error != nil {
			return result.Error
		}
		if result.RowsAffected != 1 {
			return &ConflictError{Code: "BUDGET_NO_CHARGE_FORBIDDEN"}
		}
		return nil
	})
	return &out, err
}

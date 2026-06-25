package entropy

import (
	"fmt"
	"math"
	"time"

	"gorm.io/gorm"
)

// SpcController 管理 SPC 统计过程控制 — 控制图、基线计算、异常检测。
type SpcController struct {
	db *gorm.DB
}

const (
	recalcDays       = 7
	baselineDays     = 30
	consecutiveLimit = 7
)

func NewSpcController(db *gorm.DB) *SpcController {
	return &SpcController{db: db}
}

// RecalcLimits 基于最近30天的历史数据重新计算 SPC 控制线。
func (ctl *SpcController) RecalcLimits(userID int64, agentID, decisionPoint, metricName string) (*SpcControlLimit, error) {
	windowStart := time.Now().AddDate(0, 0, -baselineDays)

	metricCol := "confidence"
	switch metricName {
	case "acceptance_rate", "confidence", "override_rate":
		metricCol = "confidence"
	}

	var values []float64
	ctl.db.Model(&AgentDecision{}).
		Where("user_id = ? AND agent_id = ? AND decision_point = ? AND created_at >= ? AND confidence IS NOT NULL",
			userID, agentID, decisionPoint, windowStart).
		Pluck(metricCol, &values)

	// Ensure JSONB fields from AgentDecision are not used — we only query confidence.

	if len(values) < 3 {
		return nil, fmt.Errorf("spc: insufficient data points (%d < 3) for %s/%s/%s", len(values), agentID, decisionPoint, metricName)
	}

	n := float64(len(values))
	var sum float64
	for _, v := range values {
		sum += v
	}
	mean := sum / n

	var varianceSum float64
	for _, v := range values {
		d := v - mean
		varianceSum += d * d
	}
	variance := varianceSum / n
	stddev := math.Sqrt(variance)

	now := time.Now()
	nextRecalc := now.AddDate(0, 0, recalcDays)

	data := map[string]interface{}{
		"baseline_mean":     round4(mean),
		"baseline_stddev":   round4(stddev),
		"baseline_samples":  len(values),
		"ucl":               round4(mean + 3*stddev),
		"lcl":               round4(mean - 3*stddev),
		"uwl":               round4(mean + 2*stddev),
		"lwl":               round4(mean - 2*stddev),
		"baseline_recalc_at": now,
		"next_recalc_at":    nextRecalc,
	}

	var limit SpcControlLimit
	err := ctl.db.Where(
		"user_id = ? AND agent_id = ? AND decision_point = ? AND metric_name = ?",
		userID, agentID, decisionPoint, metricName,
	).First(&limit).Error

	if err == nil {
		if err := ctl.db.Model(&limit).Updates(data).Error; err != nil {
			return nil, fmt.Errorf("spc: update limit: %w", err)
		}
		ctl.db.First(&limit, limit.ID)
		return &limit, nil
	}

	if err == gorm.ErrRecordNotFound {
		limit = SpcControlLimit{
			UserID:              userID,
			AgentID:             agentID,
			DecisionPoint:       decisionPoint,
			MetricName:          metricName,
			ConsecutiveSameSide: 0,
			LastBreachAt:        nil,
		}
		// Apply data fields
		limit.BaselineMean = data["baseline_mean"].(float64)
		limit.BaselineStddev = data["baseline_stddev"].(float64)
		limit.BaselineSamples = data["baseline_samples"].(int)
		limit.UCL = data["ucl"].(float64)
		limit.LCL = data["lcl"].(float64)
		limit.UWL = data["uwl"].(float64)
		limit.LWL = data["lwl"].(float64)
		limit.BaselineRecalcAt = now
		limit.NextRecalcAt = nextRecalc

		if err := ctl.db.Create(&limit).Error; err != nil {
			return nil, fmt.Errorf("spc: create limit: %w", err)
		}
		return &limit, nil
	}

	return nil, fmt.Errorf("spc: query limit: %w", err)
}

// CheckPoint 用当前值检查是否超出控制线，更新连续同侧计数。
func (ctl *SpcController) CheckPoint(userID int64, agentID, decisionPoint, metricName string, currentValue float64) (*SpcStatus, error) {
	var limit SpcControlLimit
	err := ctl.db.Where(
		"user_id = ? AND agent_id = ? AND decision_point = ? AND metric_name = ?",
		userID, agentID, decisionPoint, metricName,
	).First(&limit).Error

	if err != nil {
		if err == gorm.ErrRecordNotFound {
			recalcLimit, recalcErr := ctl.RecalcLimits(userID, agentID, decisionPoint, metricName)
			if recalcErr != nil {
				return &SpcStatus{
					AgentID:      agentID,
					DecisionPoint: decisionPoint,
					MetricName:   metricName,
				}, nil
			}
			limit = *recalcLimit
		} else {
			return nil, fmt.Errorf("spc: query limit: %w", err)
		}
	}

	mean := limit.BaselineMean

	var side string
	switch {
	case currentValue > mean:
		side = "above"
	case currentValue < mean:
		side = "below"
	default:
		side = "center"
	}

	if side != "center" {
		if (side == "above" && limit.ConsecutiveSameSide >= 0) ||
			(side == "below" && limit.ConsecutiveSameSide <= 0) {
			if side == "above" {
				limit.ConsecutiveSameSide++
			} else {
				limit.ConsecutiveSameSide--
			}
		} else {
			if side == "above" {
				limit.ConsecutiveSameSide = 1
			} else {
				limit.ConsecutiveSameSide = -1
			}
		}
	} else {
		limit.ConsecutiveSameSide = 0
	}

	var alerts []SpcAlert
	isOutOfControl := false
	isWarning := false

	if currentValue > limit.UCL || currentValue < limit.LCL {
		isOutOfControl = true
		now := time.Now()
		limit.LastBreachAt = &now
		alerts = append(alerts, SpcAlert{
			Level:   "critical",
			Message: fmt.Sprintf("%s 越控制线: %.4f (限: [%.4f, %.4f])", metricName, currentValue, limit.LCL, limit.UCL),
		})
	}

	if currentValue > limit.UWL || currentValue < limit.LWL {
		isWarning = true
		alerts = append(alerts, SpcAlert{
			Level:   "warning",
			Message: fmt.Sprintf("%s 越警戒线: %.4f (限: [%.4f, %.4f])", metricName, currentValue, limit.LWL, limit.UWL),
		})
	}

	if abs(limit.ConsecutiveSameSide) >= consecutiveLimit {
		alerts = append(alerts, SpcAlert{
			Level:   "warning",
			Message: fmt.Sprintf("连续 %d 点同侧 (%s), 趋势异常", abs(limit.ConsecutiveSameSide), side),
		})
	}

	ctl.db.Model(&limit).Updates(map[string]interface{}{
		"consecutive_same_side": limit.ConsecutiveSameSide,
		"last_breach_at":        limit.LastBreachAt,
	})

	status := &SpcStatus{
		AgentID:             agentID,
		DecisionPoint:       decisionPoint,
		MetricName:          metricName,
		CurrentValue:        currentValue,
		BaselineMean:        limit.BaselineMean,
		UCL:                 limit.UCL,
		LCL:                 limit.LCL,
		UWL:                 limit.UWL,
		LWL:                 limit.LWL,
		BaselineSamples:     limit.BaselineSamples,
		ConsecutiveSameSide: limit.ConsecutiveSameSide,
		IsOutOfControl:      isOutOfControl,
		IsWarning:           isWarning,
		Alerts:              alerts,
		LastBreachAt:        limit.LastBreachAt,
		NextRecalcAt:        &limit.NextRecalcAt,
	}

	return status, nil
}

// GetAllLimits 返回所有 SPC 控制线，快过期的自动重算。
func (ctl *SpcController) GetAllLimits(userID int64) ([]SpcControlLimit, error) {
	var limits []SpcControlLimit
	if err := ctl.db.Where("user_id = ?", userID).Find(&limits).Error; err != nil {
		return nil, err
	}

	now := time.Now()
	for i := range limits {
		if now.After(limits[i].NextRecalcAt) || now.Equal(limits[i].NextRecalcAt) {
			updated, err := ctl.RecalcLimits(
				userID,
				limits[i].AgentID,
				limits[i].DecisionPoint,
				limits[i].MetricName,
			)
			if err == nil && updated != nil {
				limits[i] = *updated
			}
		}
	}

	return limits, nil
}

func abs(x int) int {
	if x < 0 {
		return -x
	}
	return x
}

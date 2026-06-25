package ai

import (
	"encoding/json"
	"errors"
	"time"

	"github.com/google/uuid"
	"go.uber.org/zap"
	"gorm.io/gorm"
)

// TraceWriter persists AI trace events and evidence.
type TraceWriter struct {
	db     *gorm.DB
	logger *zap.Logger
}

// NewTraceWriter creates a new TraceWriter.
func NewTraceWriter(db *gorm.DB, logger *zap.Logger) *TraceWriter {
	return &TraceWriter{db: db, logger: logger}
}

// Start creates a new AITrace row and returns the trace_id.
func (w *TraceWriter) Start(in *CreateTraceInput) (string, error) {
	traceID := "trc_" + uuid.NewString()
	ctx := in.InputContext
	if len(ctx) == 0 {
		ctx = json.RawMessage(`{}`)
	}
	t := AITrace{
		TraceID:       traceID,
		UserID:        in.UserID,
		AgentID:       in.AgentID,
		DecisionPoint: in.DecisionPoint,
		Status:        "running",
		ModelProvider: in.ModelProvider,
		ModelName:     in.ModelName,
		PromptVersion: in.PromptVersion,
		InputContext:  ctx,
		StartedAt:     time.Now(),
	}
	if err := w.db.Create(&t).Error; err != nil {
		return "", err
	}
	return traceID, nil
}

// AppendEvent adds an ordered event to a trace.
func (w *TraceWriter) AppendEvent(traceID string, in *AppendEventInput) (*AITraceEvent, error) {
	if in.EventType == "" {
		return nil, errors.New("event_type is required")
	}
	var seq int
	if err := w.db.Model(&AITraceEvent{}).
		Where("trace_id = ?", traceID).
		Select("COALESCE(MAX(seq),0)").
		Row().Scan(&seq); err != nil && err != gorm.ErrRecordNotFound {
		return nil, err
	}
	seq++
	payload := in.Payload
	if len(payload) == 0 {
		payload = json.RawMessage(`{}`)
	}
	ev := AITraceEvent{
		TraceID:   traceID,
		EventType: in.EventType,
		Seq:       seq,
		Content:   in.Content,
		Payload:   payload,
	}
	if err := w.db.Create(&ev).Error; err != nil {
		return nil, err
	}
	return &ev, nil
}

// AddEvidence attaches an evidence reference to a trace.
func (w *TraceWriter) AddEvidence(traceID string, in *AddEvidenceInput) (*AIEvidenceRef, error) {
	payload := in.Payload
	if len(payload) == 0 {
		payload = json.RawMessage(`{}`)
	}
	ev := AIEvidenceRef{
		TraceID:    traceID,
		SourceType: in.SourceType,
		SourceID:   in.SourceID,
		Title:      in.Title,
		Summary:    in.Summary,
		Payload:    payload,
	}
	if err := w.db.Create(&ev).Error; err != nil {
		return nil, err
	}
	return &ev, nil
}

// Complete finalizes a trace with output and metrics.
func (w *TraceWriter) Complete(traceID string, in *CompleteTraceInput) (*AITrace, error) {
	var t AITrace
	if err := w.db.Where("trace_id = ?", traceID).First(&t).Error; err != nil {
		return nil, err
	}
	status := in.Status
	if status == "" {
		status = "completed"
	}
	now := time.Now()
	updates := map[string]interface{}{
		"status":       status,
		"completed_at": &now,
		"latency_ms":   int(now.Sub(t.StartedAt).Milliseconds()),
	}
	if in.FinalOutput != nil {
		updates["final_output"] = in.FinalOutput
	}
	if in.Confidence != nil {
		updates["confidence"] = *in.Confidence
	}
	if in.RiskLevel != "" {
		updates["risk_level"] = in.RiskLevel
	}
	if in.TokenCount > 0 {
		updates["token_count"] = in.TokenCount
	}
	if err := w.db.Model(&t).Updates(updates).Error; err != nil {
		return nil, err
	}
	if err := w.db.Where("trace_id = ?", traceID).First(&t).Error; err != nil {
		return nil, err
	}
	return &t, nil
}

// GetDetail returns the full trace detail for replay.
func (w *TraceWriter) GetDetail(traceID string) (*TraceDetail, error) {
	var t AITrace
	if err := w.db.Where("trace_id = ?", traceID).First(&t).Error; err != nil {
		return nil, err
	}
	var events []AITraceEvent
	if err := w.db.Where("trace_id = ?", traceID).Order("seq ASC").Find(&events).Error; err != nil {
		return nil, err
	}
	var evidence []AIEvidenceRef
	if err := w.db.Where("trace_id = ?", traceID).Order("id ASC").Find(&evidence).Error; err != nil {
		return nil, err
	}
	var actions []UnifiedAction
	if err := w.db.Where("trace_id = ?", traceID).Order("id ASC").Find(&actions).Error; err != nil {
		return nil, err
	}
	return &TraceDetail{Trace: t, Events: events, Evidence: evidence, Actions: actions}, nil
}

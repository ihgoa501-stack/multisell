package trustscore

import "time"

// TrustScore tracks an agent's performance and determines autonomy eligibility.
type TrustScore struct {
	ID             int64     `gorm:"column:id;primaryKey;autoIncrement" json:"id"`
	AgentID        string    `gorm:"column:agent_id;not null;uniqueIndex" json:"agent_id"`
	AgentName      string    `gorm:"column:agent_name;not null" json:"agent_name"`
	SquadID        string    `gorm:"column:squad_id" json:"squad_id"`

	// Core metrics — computed from action history
	TotalActions   int     `gorm:"column:total_actions;default:0" json:"total_actions"`
	AdoptedActions int     `gorm:"column:adopted_actions;default:0" json:"adopted_actions"`
	RejectedActions int    `gorm:"column:rejected_actions;default:0" json:"rejected_actions"`
	FailedActions  int     `gorm:"column:failed_actions;default:0" json:"failed_actions"`
	AutoApproved   int     `gorm:"column:auto_approved;default:0" json:"auto_approved"`

	// Derived trust metrics
	AdoptionRate     float64 `gorm:"column:adoption_rate;type:numeric(5,4);default:0" json:"adoption_rate"`
	ExecutionSuccess float64 `gorm:"column:execution_success;type:numeric(5,4);default:0" json:"execution_success"`
	AvgConfidence    float64 `gorm:"column:avg_confidence;type:numeric(5,4);default:0" json:"avg_confidence"`
	TrustScore       float64 `gorm:"column:trust_score;type:numeric(5,4);default:0" json:"trust_score"`       // 0.0 - 1.0
	AutonomyLevel    string  `gorm:"column:autonomy_level;default:advisory" json:"autonomy_level"`            // advisory | guided | supervised | autonomous
	TargetLevel      string  `gorm:"column:target_level" json:"target_level,omitempty"`                       // next eligible level

	// Business impact
	EstimatedSavings float64 `gorm:"column:estimated_savings;type:numeric(12,2);default:0" json:"estimated_savings"`
	LastActionAt     *time.Time `gorm:"column:last_action_at" json:"last_action_at,omitempty"`

	CreatedAt time.Time `gorm:"column:created_at;autoCreateTime" json:"created_at"`
	UpdatedAt time.Time `gorm:"column:updated_at;autoUpdateTime" json:"updated_at"`
}

func (TrustScore) TableName() string { return "agent_trust_score" }

// AutonomyThresholds define the trust score ranges for each autonomy level.
var AutonomyThresholds = map[string]float64{
	"advisory":    0.0,   // Default — just advises
	"guided":      0.30,  // 30d+ trust: can create actions with human approval
	"supervised":  0.55,  // 55d+ trust: actions auto-create but need approval
	"autonomous":  0.80,  // 80d+ trust: low-risk actions auto-approved
}

// NewTrustScore creates a new trust score record with default values.
func NewTrustScore(agentID, agentName, squadID string) *TrustScore {
	return &TrustScore{
		AgentID:     agentID,
		AgentName:   agentName,
		SquadID:     squadID,
		AutonomyLevel: "advisory",
		TrustScore:  0.0,
	}
}

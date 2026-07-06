package listingtask

import "time"

// TaskReview is the response payload for the listing task review endpoint.
type TaskReview struct {
	TaskID          uint      `json:"task_id"`
	Published       bool      `json:"published"`
	Status          string    `json:"status"`
	Platform        string    `json:"platform,omitempty"`
	PlatformErrors  []string  `json:"platform_errors,omitempty"`
	ProfitExpected  *float64  `json:"profit_expected,omitempty"`
	ProfitActual    *float64  `json:"profit_actual,omitempty"`
	MarginExpected  *float64  `json:"margin_expected,omitempty"`
	MarginActual    *float64  `json:"margin_actual,omitempty"`
	CreatedAt       time.Time `json:"created_at"`
}

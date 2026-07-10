package aimapper

// ── Prompt template registry ──

// PromptRegistry stores prompt templates keyed by platform code and event type.
type PromptRegistry struct {
	prompts map[string]string
}

// NewPromptRegistry returns an empty prompt registry.
func NewPromptRegistry() *PromptRegistry {
	return &PromptRegistry{
		prompts: make(map[string]string),
	}
}

// Register stores a prompt for the given platform and event combination.
func (r *PromptRegistry) Register(platformCode, eventType, prompt string) {
	r.prompts[platformCode+":"+eventType] = prompt
}

// Get retrieves the prompt for the given platform and event type.
func (r *PromptRegistry) Get(platformCode, eventType string) (string, bool) {
	p, ok := r.prompts[platformCode+":"+eventType]
	return p, ok
}

package guardrails

import (
	"context"
	"regexp"
	"strconv"

	"go.uber.org/zap"
)

// InjectionRule defines a single prompt injection detection rule.
type InjectionRule struct {
	// Pattern is a regular expression that matches an injection attempt.
	Pattern string `json:"pattern"`
	// Score is the severity weight (1-10) assigned when this rule matches.
	Score int `json:"score"`
	// Action specifies the recommended handling: "block", "sanitize", or "warn".
	Action string `json:"action"`
}

// compiledRule holds a pre-compiled regex for fast matching.
type compiledRule struct {
	InjectionRule
	re *regexp.Regexp
}

// PromptInjectionGuard implements L1 input guardrail for detecting prompt
// injection attempts in raw agent inputs.
//
// It uses a set of pre-compiled regex patterns (Chinese and English) to
// detect common injection techniques such as instruction override, role-play
// induction, system prompt exfiltration, base64-encoded payloads, and
// HTML/script tag injection. Each pattern carries a severity score; the
// total accumulated score determines the outcome:
//
//	>= 6  → Blocked (non-retryable)
//	 3-5  → Warn   (pass but flagged)
//	<  3  → Pass   (clean)
type PromptInjectionGuard struct {
	rules  []compiledRule
	logger *zap.Logger
}

// NewPromptInjectionGuard creates a guard with the built-in set of injection
// detection rules. Rules are compiled at construction time; Check is O(n)
// in the number of rules.
func NewPromptInjectionGuard() *PromptInjectionGuard {
	return NewPromptInjectionGuardWithLogger(zap.NewNop())
}

// NewPromptInjectionGuardWithLogger creates a guard with a custom logger.
func NewPromptInjectionGuardWithLogger(logger *zap.Logger) *PromptInjectionGuard {
	if logger == nil {
		logger = zap.NewNop()
	}

	rules := defaultInjectionRules()
	compiled := make([]compiledRule, 0, len(rules))
	for _, r := range rules {
		re, err := regexp.Compile(r.Pattern)
		if err != nil {
			logger.Warn("failed to compile injection rule — skipping",
				zap.String("pattern", r.Pattern),
				zap.Error(err),
			)
			continue
		}
		compiled = append(compiled, compiledRule{
			InjectionRule: r,
			re:            re,
		})
	}

	return &PromptInjectionGuard{
		rules:  compiled,
		logger: logger,
	}
}

// Name returns "prompt_injection_guard".
func (g *PromptInjectionGuard) Name() string {
	return "prompt_injection_guard"
}

// Check scans the input's RawInput against all compiled rules. The total
// severity score determines the outcome:
//
//	>= 6 → Blocked
//	 3-5 → Warn (pass=false, blocked=false)
//	< 3  → Pass
//
// Empty input always passes.
func (g *PromptInjectionGuard) Check(ctx context.Context, input *GuardInput) (*GuardResult, error) {
	if input.RawInput == "" {
		return &GuardResult{
			Pass:    true,
			Blocked: false,
			Retry:   false,
			Reason:  "empty input",
			Risk:    "low",
		}, nil
	}

	totalScore := 0
	var matchedRules []string

	for _, rule := range g.rules {
		if rule.re.MatchString(input.RawInput) {
			totalScore += rule.Score
			matchedRules = append(matchedRules, rule.Pattern)
		}
	}

	if len(matchedRules) > 0 {
		g.logger.Warn("prompt injection pattern matched",
			zap.Int("total_score", totalScore),
			zap.Strings("matched_rules", matchedRules),
		)

		// Score threshold: >= 6 blocked, 3-5 warn, < 3 pass.
		if totalScore >= 6 {
			return &GuardResult{
				Pass:    false,
				Blocked: true,
				Retry:   false,
				Reason:  "prompt injection detected (score: " + strconv.Itoa(totalScore) + ")",
				Risk:    "high",
			}, nil
		}

		if totalScore >= 3 {
			return &GuardResult{
				Pass:    false,
				Blocked: false,
				Retry:   false,
				Reason:  "prompt injection suspected (score: " + strconv.Itoa(totalScore) + ")",
				Risk:    "medium",
			}, nil
		}
	}

	return &GuardResult{
		Pass:    true,
		Blocked: false,
		Retry:   false,
		Reason:  "no injection detected",
		Risk:    "low",
	}, nil
}

// defaultInjectionRules returns the built-in set of injection detection
// patterns covering Chinese and English injection techniques.
//
// Score assignment:
//   - Score >= 6: the pattern alone triggers a block. Reserved for direct
//     instruction-override, memory-wipe, and code-injection patterns.
//   - Score 3-5: the pattern triggers a warning. Used for role-play induction,
//     system prompt probing, identity hijack, and reset attempts.
//   - Score < 3: low-severity patterns that only trigger action when stacked.
func defaultInjectionRules() []InjectionRule {
	return []InjectionRule{
		// 1. CN: instruction override — "ignore your/previous instructions".
		{Pattern: `忽略(?:你|之前)的指令`, Score: 7, Action: "block"},
		// 2. EN: instruction override.
		{Pattern: `ignore (?:all )?(?:previous|prior) (?:instructions|directions)`, Score: 7, Action: "block"},
		// 3. CN: role-play induction — "you are a [role]".
		{Pattern: `你是一个[^。]+`, Score: 4, Action: "warn"},
		// 4. EN: role-play induction.
		{Pattern: `(?i)you are (?:now )?a`, Score: 3, Action: "warn"},
		// 5. System prompt exfiltration attempts.
		{Pattern: `(?i)(?:system prompt|system message)`, Score: 4, Action: "warn"},
		// 6. REPEAT pattern — forcing the model to echo hidden instructions.
		{Pattern: `(?i)repeat (?:after|above|the|all)\b`, Score: 4, Action: "warn"},
		// 7. Base64-encoded payload (40+ chars with optional padding).
		{Pattern: `[A-Za-z0-9+/]{40,}={0,2}`, Score: 7, Action: "block"},
		// 8. CN: "forget previous instructions".
		{Pattern: `忘记(?:所有|之前的)?`, Score: 6, Action: "block"},
		// 9. EN: "forget prior instructions".
		{Pattern: `(?i)forget (?:all |the )?(?:previous|prior)`, Score: 7, Action: "block"},
		// 10. CN: "now you are" / "you are now playing the role of".
		{Pattern: `(?:你现在是|你现在扮演)`, Score: 4, Action: "warn"},
		// 11. EN: "now you are" identity hijack.
		{Pattern: `(?i)now you are`, Score: 4, Action: "warn"},
		// 12. CN: "ignore" / "disregard rules".
		{Pattern: `(?:无视|忽略规则)`, Score: 6, Action: "block"},
		// 13. EN: "disable all rules".
		{Pattern: `(?i)disable (?:all )?rules`, Score: 7, Action: "block"},
		// 14. HTML/script tag injection (script/iframe/embed/object).
		{Pattern: `</?(?:script|iframe|embed|object)[^>]*>`, Score: 7, Action: "block"},
		// 15. CN: "reset" / "reinitialize".
		{Pattern: `(?:重新设定|重置)`, Score: 3, Action: "warn"},
		// 16. EN: "reset".
		{Pattern: `(?i)\breset\b`, Score: 3, Action: "warn"},
	}
}

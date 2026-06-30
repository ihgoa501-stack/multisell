// Package impl provides concrete agent implementations.
//
// ComplianceGuardAgent implements A7 Compliance Guard business logic ported
// from backend/app/agent/agents/compliance.py (Python FastAPI codebase).
//
// Design docs: docs/aiagent/跨境电商AI_Agent深度调研报告.md §Agent7
//   - Product compliance check: certification requirements, restricted items,
//     label requirements, tax compliance
//   - Input: product information and target market
//   - Output: compliance risks and required certifications
package impl

import (
	"context"
	"fmt"
	"strings"

	"github.com/lingmirror/backend-go/internal/aios/toolregistry"
)

// ---------- Context field names ----------

var complianceRequiredFields = []string{"product_name", "category", "target_country", "target_platform"}

// ---------- Compliance rules ----------

// complianceRule defines certification requirements, restrictions, and risk level
// for a (category, country) pair.
type complianceRule struct {
	Certifications []string
	Restrictions   []string
	Risk           string
}

var complianceRules = map[string]complianceRule{
	"electronics_US": {
		Certifications: []string{"FCC", "UL"},
		Restrictions:   []string{},
		Risk:           "medium",
	},
	"electronics_EU": {
		Certifications: []string{"CE", "RoHS", "WEEE"},
		Restrictions:   []string{},
		Risk:           "medium",
	},
	"electronics_UK": {
		Certifications: []string{"UKCA", "RoHS"},
		Restrictions:   []string{},
		Risk:           "medium",
	},
	"electronics_JP": {
		Certifications: []string{"PSE", "MIC"},
		Restrictions:   []string{},
		Risk:           "medium",
	},
	"baby_US": {
		Certifications: []string{"CPC", "ASTM F963"},
		Restrictions:   []string{},
		Risk:           "high",
	},
	"baby_EU": {
		Certifications: []string{"CE", "EN 71"},
		Restrictions:   []string{},
		Risk:           "high",
	},
	"baby_UK": {
		Certifications: []string{"UKCA", "EN 71"},
		Restrictions:   []string{},
		Risk:           "high",
	},
	"cosmetics_EU": {
		Certifications: []string{"CPNP"},
		Restrictions:   []string{"动物测试"},
		Risk:           "high",
	},
	"cosmetics_US": {
		Certifications: []string{"FDA"},
		Restrictions:   []string{},
		Risk:           "medium",
	},
	"food_US": {
		Certifications: []string{"FDA"},
		Restrictions:   []string{},
		Risk:           "high",
	},
	"food_EU": {
		Certifications: []string{"EU Novel Food"},
		Restrictions:   []string{},
		Risk:           "high",
	},
	"toys_US": {
		Certifications: []string{"CPC", "ASTM F963"},
		Restrictions:   []string{},
		Risk:           "high",
	},
	"toys_EU": {
		Certifications: []string{"CE", "EN 71"},
		Restrictions:   []string{},
		Risk:           "high",
	},
}

// ---------- Certification description map ----------

var certDescriptions = map[string]string{
	"FCC": "FCC 认证（美国）：电子产品的电磁兼容性强制认证",
	"CE":  "CE 认证（欧盟）：产品安全、健康、环保的基本要求",
	"CPC": "CPC 认证（美国）：儿童产品安全强制认证",
	"FDA": "FDA 认证（美国）：食品、药品、化妆品的 FDA 注册",
}

// complianceRuleKey builds a lookup key for complianceRules.
func complianceRuleKey(category, country string) string {
	return strings.ToLower(category) + "_" + strings.ToUpper(country)
}

// ---------- ComplianceGuardAgent ----------

// ComplianceGuardAgent implements A7 Compliance Guard logic.
//
// Decision points:
//   - "compliance_check" — checks product compliance for a target market
//   - "certification_lookup" — looks up a certification's description
type ComplianceGuardAgent struct{}

// NewComplianceGuardAgent creates a new ComplianceGuardAgent.
func NewComplianceGuardAgent() *ComplianceGuardAgent {
	return &ComplianceGuardAgent{}
}

// Decide dispatches to the correct decision handler based on decisionPoint.
//
// Supported decision points:
//   - "compliance_check"
//   - "certification_lookup"
//
// Delegates to toolregistry.DefaultRegistry.Call() first. Falls back to
// built-in logic when no tool is registered for the decision point.
func (a *ComplianceGuardAgent) Decide(ctx context.Context, decisionPoint string, params map[string]interface{}) (output map[string]interface{}, confidence float64, riskLevel string, err error) {
	// Try toolregistry first — tools registered via init() functions may
	// provide a more up-to-date or dynamic implementation.
	result, regErr := toolregistry.DefaultRegistry.Call(ctx, decisionPoint, params)
	if regErr == nil {
		if m, ok := result.(map[string]interface{}); ok {
			conf, _ := m["confidence"].(float64)
			r, _ := m["risk_level"].(string)
			if r == "" {
				r = "low"
			}
			return m, conf, r, nil
		}
	}

	// Fallback to built-in logic.
	switch decisionPoint {
	case "compliance_check":
		return a.check(params)
	case "certification_lookup":
		return a.lookup(params)
	default:
		return map[string]interface{}{
			"status":         "unknown",
			"decision_point": decisionPoint,
			"error":          fmt.Sprintf("unknown decision point: %s", decisionPoint),
		}, 0.0, "low", nil
	}
}

// ---------- Decision point: compliance_check ----------

// check evaluates product compliance requirements for a target market.
//
// Required context fields: product_name, category, target_country, target_platform
//
// Returns:
//   - required_certifications: list of certifications needed
//   - restrictions: list of restrictions or prohibitions
//   - risk_level: the compliance risk level (low/medium/high/unknown)
//   - blocked_platforms: platforms where the product cannot be listed
func (a *ComplianceGuardAgent) check(ctx map[string]interface{}) (output map[string]interface{}, confidence float64, riskLevel string, err error) {
	if missing := missingFields(ctx, complianceRequiredFields); len(missing) > 0 {
		return insufficientData("compliance_check", missing), 0.0, "low", nil
	}

	name := safeString(ctx["product_name"], "")
	cat := strings.ToLower(strings.TrimSpace(safeString(ctx["category"], "")))
	country := strings.ToUpper(strings.TrimSpace(safeString(ctx["target_country"], "")))
	platform := strings.ToLower(strings.TrimSpace(safeString(ctx["target_platform"], "")))

	// Look up compliance rules.
	rule, ok := complianceRules[complianceRuleKey(cat, country)]
	if !ok {
		rule = complianceRule{
			Certifications: []string{},
			Restrictions:   []string{"请人工核实具体认证要求"},
			Risk:           "unknown",
		}
	}

	certs := rule.Certifications
	restrictions := rule.Restrictions
	risk := rule.Risk

	// Amazon-specific rule: electronics + US without FCC → blocked.
	blockedPlatforms := make([]string, 0)
	if platform == "amazon" && cat == "electronics" && country == "US" {
		hasFCC := false
		for _, c := range certs {
			if c == "FCC" {
				hasFCC = true
				break
			}
		}
		if !hasFCC {
			blockedPlatforms = append(blockedPlatforms, "Amazon US (FCC 要求)")
		}
	}

	conf := 0.90
	if risk == "unknown" {
		conf = 0.60
	}

	output = map[string]interface{}{
		"product":                name,
		"category":               cat,
		"country":                country,
		"platform":               platform,
		"required_certifications": certs,
		"restrictions":           restrictions,
		"risk_level":             risk,
		"blocked_platforms":      blockedPlatforms,
		"confidence":             conf,
	}
	return output, conf, risk, nil
}

// ---------- Decision point: certification_lookup ----------

// lookup returns a description for a given certification identifier.
//
// Known certifications: FCC, CE, CPC, FDA.
// Unknown certifications return a fallback message suggesting manual verification.
func (a *ComplianceGuardAgent) lookup(ctx map[string]interface{}) (output map[string]interface{}, confidence float64, riskLevel string, err error) {
	cert := safeString(ctx["certification"], "")
	country := safeString(ctx["country"], "US")

	description, known := certDescriptions[strings.ToUpper(cert)]

	conf := 0.85
	if !known {
		description = fmt.Sprintf("请人工核实 %s 在 %s 的具体要求", cert, country)
		conf = 0.50
	}

	output = map[string]interface{}{
		"certification": cert,
		"country":       country,
		"description":   description,
		"confidence":    conf,
	}
	return output, conf, "low", nil
}

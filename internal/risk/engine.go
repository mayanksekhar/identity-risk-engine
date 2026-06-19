// Package risk implements the rule-based risk scoring engine. Scores are
// deterministic and explainable by design: every point added to a score is
// traceable to a named factor, which matters both for audit/compliance
// narratives and for feeding a clean structured input to the LLM
// explanation layer in package llm.
package risk

import (
	"sort"
	"time"

	"gitlab.com/mayanksekhar/identity-risk-engine/internal/identity"
)

// Severity buckets the final score into a human-facing label.
type Severity string

const (
	SeverityLow      Severity = "low"
	SeverityMedium   Severity = "medium"
	SeverityHigh     Severity = "high"
	SeverityCritical Severity = "critical"
)

// Factor is a single named contribution to an identity's risk score.
// Weight is the point value applied (can be negative, e.g. MFA reduces risk).
type Factor struct {
	Code   string `json:"code"`
	Label  string `json:"label"`
	Weight int    `json:"weight"`
}

// Score is the full scoring result for one identity at a point in time.
type Score struct {
	IdentityID string    `json:"identity_id"`
	Total      int       `json:"total"`      // clamped 0-100
	Severity   Severity   `json:"severity"`
	Factors    []Factor  `json:"factors"`
	ScoredAt   time.Time `json:"scored_at"`
}

// staleThresholds defines days-since-active thresholds per identity type.
// AI agents and NHIs are expected to be active far more frequently than
// humans; a "stale" agent is a stronger signal (likely orphaned/abandoned)
// than a stale human account during, say, parental leave.
var staleThresholds = map[identity.Type]float64{
	identity.TypeHuman:   30,
	identity.TypeNHI:     7,
	identity.TypeAIAgent: 3,
}

// Engine computes deterministic risk scores from a fixed rule set.
type Engine struct {
	now func() time.Time // injected for deterministic tests
}

// NewEngine constructs a scoring engine using wall-clock time.
func NewEngine() *Engine {
	return &Engine{now: time.Now}
}

// NewEngineWithClock constructs an engine with an injected clock, used in tests.
func NewEngineWithClock(now func() time.Time) *Engine {
	return &Engine{now: now}
}

// Score evaluates a single identity against the rule set and returns a
// fully-explained score. Rules are additive; final total is clamped to
// [0, 100].
func (e *Engine) Score(id identity.Identity) Score {
	now := e.now()
	var factors []Factor

	// --- Authentication posture ---
	switch id.AuthMethod {
	case identity.AuthNone:
		factors = append(factors, Factor{"auth_none", "No authentication mechanism on record", 30})
	case identity.AuthPassword:
		factors = append(factors, Factor{"auth_password_only", "Password-only authentication (no MFA/SSO)", 18})
	case identity.AuthAPIKey:
		factors = append(factors, Factor{"auth_api_key", "Long-lived API key authentication", 12})
	case identity.AuthSSO:
		factors = append(factors, Factor{"auth_sso", "SSO-enforced authentication", -5})
	case identity.AuthMFA:
		factors = append(factors, Factor{"auth_mfa", "MFA-enforced authentication", -15})
	case identity.AuthOIDC:
		factors = append(factors, Factor{"auth_oidc_workload", "Short-lived workload identity (OIDC)", -10})
	}

	// --- Privilege level ---
	switch id.Privilege {
	case identity.PrivilegeAdmin:
		factors = append(factors, Factor{"priv_admin", "Administrative privilege level", 25})
	case identity.PrivilegeElevated:
		factors = append(factors, Factor{"priv_elevated", "Elevated privilege level", 12})
	}

	// --- Standing access breadth ---
	if n := len(id.StandingAccess); n > 0 {
		w := n * 4
		if w > 24 {
			w = 24 // cap so one over-permissioned identity doesn't dominate the scale
		}
		factors = append(factors, Factor{"standing_access_breadth", "Broad standing access grants", w})
	}

	// --- Staleness (type-aware threshold) ---
	threshold := staleThresholds[id.Type]
	daysInactive := id.DaysSinceActive(now)
	if id.Active && daysInactive > threshold {
		w := 10
		if daysInactive > threshold*3 {
			w = 22
		}
		factors = append(factors, Factor{"stale_activity", "Inactive well beyond expected baseline for this identity type", w})
	}

	// --- Disabled-but-not-offboarded ---
	if !id.Active {
		factors = append(factors, Factor{"inactive_not_offboarded", "Identity marked inactive but record not purged/offboarded", 8})
	}

	// --- Identity-type-specific signals ---
	switch id.Type {
	case identity.TypeAIAgent:
		factors = append(factors, Factor{"agent_autonomy_baseline", "Autonomous agent: baseline elevated scrutiny", 10})
		if hasTag(id.Tags, "unattended") {
			factors = append(factors, Factor{"agent_unattended", "Agent operates without human-in-the-loop approval", 18})
		}
	case identity.TypeNHI:
		if hasTag(id.Tags, "no-rotation") {
			factors = append(factors, Factor{"nhi_no_rotation", "Credential has no automated rotation policy", 15})
		}
	case identity.TypeHuman:
		if hasTag(id.Tags, "external-contractor") {
			factors = append(factors, Factor{"human_external_contractor", "External contractor with standing internal access", 10})
		}
	}

	total := 0
	for _, f := range factors {
		total += f.Weight
	}
	total = clamp(total, 0, 100)

	sort.Slice(factors, func(i, j int) bool { return factors[i].Weight > factors[j].Weight })

	return Score{
		IdentityID: id.ID,
		Total:      total,
		Severity:   severityFor(total),
		Factors:    factors,
		ScoredAt:   now,
	}
}

// ScoreAll scores a batch of identities in one pass.
func (e *Engine) ScoreAll(ids []identity.Identity) []Score {
	out := make([]Score, 0, len(ids))
	for _, id := range ids {
		out = append(out, e.Score(id))
	}
	return out
}

func severityFor(total int) Severity {
	switch {
	case total >= 70:
		return SeverityCritical
	case total >= 45:
		return SeverityHigh
	case total >= 20:
		return SeverityMedium
	default:
		return SeverityLow
	}
}

func clamp(v, min, max int) int {
	if v < min {
		return min
	}
	if v > max {
		return max
	}
	return v
}

func hasTag(tags []string, want string) bool {
	for _, t := range tags {
		if t == want {
			return true
		}
	}
	return false
}

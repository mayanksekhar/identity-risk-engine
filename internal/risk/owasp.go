// Package risk - owasp.go maps this engine's scoring factors to the
// official OWASP Non-Human Identities Top 10 (2025) risk categories.
//
// This mapping exists to make the risk score legible to security buyers and
// auditors who already work against the OWASP NHI Top 10 as a standard,
// rather than requiring them to trust an opaque internal scoring scheme.
// Reference: https://owasp.org/www-project-non-human-identities-top-10/
//
// A single scoring factor can map to more than one OWASP risk (e.g. a
// long-lived API key with no rotation is both NHI4 - Insecure Authentication
// and NHI7 - Long-Lived Secrets). The mapping is intentionally conservative:
// only factors with a clear, defensible correspondence are mapped. Factors
// with no established OWASP NHI category (e.g. the agentic-autonomy factors,
// which predate the standard) are left unmapped rather than force-fit,
// and are called out separately as "frontier" risks in FrontierRisks below.
package risk

// OWASPNHIRisk identifies a single risk category from the OWASP Non-Human
// Identities Top 10 (2025 edition).
type OWASPNHIRisk struct {
	ID    string `json:"id"`    // e.g. "NHI1:2025"
	Title string `json:"title"` // e.g. "Improper Offboarding"
}

var (
	nhi1ImproperOffboarding  = OWASPNHIRisk{ID: "NHI1:2025", Title: "Improper Offboarding"}
	nhi2SecretLeakage        = OWASPNHIRisk{ID: "NHI2:2025", Title: "Secret Leakage"}
	nhi3VulnerableThirdParty = OWASPNHIRisk{ID: "NHI3:2025", Title: "Vulnerable Third-Party NHI"}
	nhi4InsecureAuth         = OWASPNHIRisk{ID: "NHI4:2025", Title: "Insecure Authentication"}
	nhi5Overprivileged       = OWASPNHIRisk{ID: "NHI5:2025", Title: "Overprivileged NHI"}
	nhi6InsecureCloudConfig  = OWASPNHIRisk{ID: "NHI6:2025", Title: "Insecure Cloud Deployment Configurations"}
	nhi7LongLivedSecrets     = OWASPNHIRisk{ID: "NHI7:2025", Title: "Long-Lived Secrets"}
	nhi8EnvironmentIsolation = OWASPNHIRisk{ID: "NHI8:2025", Title: "Environment Isolation"}
	nhi9NHIReuse             = OWASPNHIRisk{ID: "NHI9:2025", Title: "NHI Reuse"}
	nhi10HumanUseOfNHI       = OWASPNHIRisk{ID: "NHI10:2025", Title: "Human Use of NHI"}
)

// factorToOWASP maps this engine's internal Factor.Code values to the OWASP
// NHI Top 10 categories they correspond to. A factor may map to zero, one,
// or multiple risks. Factor codes not present in this map correspond to no
// established OWASP NHI category as of the 2025 edition.
var factorToOWASP = map[string][]OWASPNHIRisk{
	// --- Authentication posture ---
	// No authentication at all is the clearest possible case of insecure
	// authentication - the identity has no credential-based access control.
	"auth_none": {nhi4InsecureAuth},

	// Password-only auth on a machine/service identity is a weak, deprecated
	// authentication mechanism relative to short-lived tokens or workload
	// identity federation.
	"auth_password_only": {nhi4InsecureAuth},

	// A standing API key is both an authentication-mechanism concern (it is
	// a long-lived bearer credential rather than a short-lived token) and,
	// separately, a long-lived-secret concern in its own right.
	"auth_api_key": {nhi4InsecureAuth, nhi7LongLivedSecrets},

	// --- Privilege ---
	// Admin and elevated privilege map directly to NHI5 - the risk is the
	// gap between what the identity actually needs and what it can do.
	"priv_admin":    {nhi5Overprivileged},
	"priv_elevated": {nhi5Overprivileged},

	// Broad standing access (many resource grants) is the same
	// overprivileged-NHI risk expressed through grant count rather than a
	// named privilege tier.
	"standing_access_breadth": {nhi5Overprivileged},

	// --- Lifecycle / offboarding ---
	// An identity marked inactive but not purged from the system is the
	// textbook definition of improper offboarding - the deactivation step
	// happened, but removal did not.
	"inactive_not_offboarded": {nhi1ImproperOffboarding},

	// Staleness beyond the expected baseline for the identity type is a
	// leading indicator of an identity that should have been offboarded but
	// was not - the same underlying risk, caught earlier in its lifecycle.
	"stale_activity": {nhi1ImproperOffboarding},

	// --- Secrets hygiene ---
	// A credential with no rotation policy is the canonical long-lived
	// secret: it does not expire on its own, so its exposure window is
	// unbounded until someone manually rotates it.
	"nhi_no_rotation": {nhi7LongLivedSecrets},

	// --- Human / NHI boundary ---
	// An external contractor holding standing internal access blurs the
	// human/NHI boundary in the same direction NHI10 describes: a human
	// identity carrying access-pattern risk normally associated with
	// service accounts (broad, unaudited, long-lived access).
	"human_external_contractor": {nhi10HumanUseOfNHI},
}

// frontierFactors lists factor codes that describe genuinely new risk
// surface introduced by autonomous AI agents, which the OWASP NHI Top 10
// (2025) predates and does not yet name. These are surfaced separately
// rather than mapped into an existing category that would not accurately
// describe them.
var frontierFactors = map[string]string{
	"agent_unattended": "Autonomous agent acting without human-in-the-loop " +
		"approval - a risk pattern not yet named in OWASP NHI Top 10 2025, " +
		"closest in spirit to NHI5 (Overprivileged NHI) and NHI10 (Human Use " +
		"of NHI) but distinct: the concern here is unsupervised autonomous " +
		"action, not excess privilege or human misuse of a service account.",
	"agent_autonomy_baseline": "Baseline scrutiny applied to all autonomous " +
		"agent identities regardless of other factors - reflects that agentic " +
		"identities are a new identity class the current standard does not " +
		"yet enumerate.",
}

// OWASPRisksForFactors returns the deduplicated, sorted set of OWASP NHI Top
// 10 risks implicated by the given list of scoring factors, plus any
// frontier (unmapped) risk notes for factors the standard does not yet cover.
func OWASPRisksForFactors(factors []Factor) (risks []OWASPNHIRisk, frontierNotes []string) {
	seen := make(map[string]bool)
	frontierSeen := make(map[string]bool)

	for _, f := range factors {
		if mapped, ok := factorToOWASP[f.Code]; ok {
			for _, r := range mapped {
				if !seen[r.ID] {
					seen[r.ID] = true
					risks = append(risks, r)
				}
			}
		}
		if note, ok := frontierFactors[f.Code]; ok {
			if !frontierSeen[f.Code] {
				frontierSeen[f.Code] = true
				frontierNotes = append(frontierNotes, note)
			}
		}
	}

	return risks, frontierNotes
}

// Package risk - agentic.go maps the agentic-autonomy scoring factors to
// the OWASP Top 10 for Agentic Applications (2026), announced 2025-12-09 by
// the OWASP GenAI Security Project.
//
// This is deliberately a separate mapping from owasp.go (OWASP NHI Top 10
// 2025), not a replacement. The honest history: when this scoring engine
// was first built, agent_unattended and agent_autonomy_baseline had no
// established OWASP category to map into - the NHI Top 10 (2025) predates
// autonomous AI agents as a named identity class, and those two factors
// were surfaced as unmapped "frontier" risks rather than force-fit into a
// category that wouldn't accurately describe them (see owasp.go). The
// Agentic Top 10 (2026) is the standard that now actually covers this
// territory. Both mappings are kept: NHI risks aren't agentic risks, and
// collapsing them into one list would blur two genuinely different
// standards with different scopes and different audiences.
//
// Reference: https://genai.owasp.org/resource/owasp-top-10-for-agentic-applications-for-2026/
package risk

// ASIRisk identifies a single risk category from the OWASP Top 10 for
// Agentic Applications (2026 edition). ASI is OWASP's own prefix
// ("Agentic Security Initiative").
type ASIRisk struct {
	ID    string `json:"id"`    // e.g. "ASI03:2026"
	Title string `json:"title"` // e.g. "Identity & Privilege Abuse"
}

var (
	asi01GoalHijack       = ASIRisk{ID: "ASI01:2026", Title: "Agent Goal Hijack"}
	asi02ToolMisuse       = ASIRisk{ID: "ASI02:2026", Title: "Tool Misuse & Exploitation"}
	asi03IdentityAbuse    = ASIRisk{ID: "ASI03:2026", Title: "Identity & Privilege Abuse"}
	asi04SupplyChain      = ASIRisk{ID: "ASI04:2026", Title: "Agentic Supply Chain Vulnerabilities"}
	asi05CodeExecution    = ASIRisk{ID: "ASI05:2026", Title: "Unexpected Code Execution"}
	asi06MemoryPoisoning  = ASIRisk{ID: "ASI06:2026", Title: "Memory & Context Poisoning"}
	asi07InterAgentComms  = ASIRisk{ID: "ASI07:2026", Title: "Insecure Inter-Agent Communication"}
	asi08CascadingFailure = ASIRisk{ID: "ASI08:2026", Title: "Cascading Failures"}
	asi09TrustExploit     = ASIRisk{ID: "ASI09:2026", Title: "Human-Agent Trust Exploitation"}
	asi10RogueAgents      = ASIRisk{ID: "ASI10:2026", Title: "Rogue Agents"}
)

// The full ASI catalogue above documents every category for anyone
// extending this mapping later, even though this engine currently only
// scores identity posture (not live agent runtime behavior), so several
// categories - ASI02, ASI04 through ASI08, ASI10 - describe tool-execution
// and multi-agent risks this scoring engine does not yet observe.

// factorToASI maps this engine's scoring factor codes to OWASP Agentic Top
// 10 (2026) categories. Deliberately conservative: only the two factors
// that specifically encode agent-autonomy risk are mapped here. Factors
// like priv_admin or standing_access_breadth already have an NHI 2025
// mapping (owasp.go) and are not duplicated here, since they describe
// identity posture generic to any NHI, not something specific to agentic
// autonomy.
var factorToASI = map[string][]ASIRisk{
	// An agent operating without human-in-the-loop approval is precisely
	// ASI09's failure mode (no human checkpoint before consequential
	// action) and, because that unattended standing access is itself the
	// identity/privilege surface being exercised without oversight, also
	// ASI03.
	"agent_unattended": {asi03IdentityAbuse, asi09TrustExploit},

	// Every autonomous agent identity carries privilege that needs
	// scrutiny purely by virtue of being autonomous - the baseline ASI03
	// concern, independent of any other tag.
	"agent_autonomy_baseline": {asi03IdentityAbuse},
}

// ASIRisksForFactors returns the deduplicated, order-stable set of OWASP
// Agentic Top 10 (2026) risks implicated by the given scoring factors.
func ASIRisksForFactors(factors []Factor) []ASIRisk {
	var risks []ASIRisk
	seen := make(map[string]bool)

	for _, f := range factors {
		mapped, ok := factorToASI[f.Code]
		if !ok {
			continue
		}
		for _, r := range mapped {
			if !seen[r.ID] {
				seen[r.ID] = true
				risks = append(risks, r)
			}
		}
	}

	return risks
}

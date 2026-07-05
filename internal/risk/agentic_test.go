package risk

import "testing"

func TestASIRisksForFactors_UnattendedMapsToTwoRisks(t *testing.T) {
	factors := []Factor{
		{Code: "agent_unattended", Label: "Agent operates without human-in-the-loop approval", Weight: 18},
	}

	risks := ASIRisksForFactors(factors)
	ids := asiRiskIDs(risks)

	assertContainsASI(t, ids, "ASI03:2026")
	assertContainsASI(t, ids, "ASI09:2026")
	if len(ids) != 2 {
		t.Fatalf("expected exactly 2 ASI risks for agent_unattended, got %d: %v", len(ids), ids)
	}
}

func TestASIRisksForFactors_AutonomyBaselineMapsToOneRisk(t *testing.T) {
	factors := []Factor{
		{Code: "agent_autonomy_baseline", Label: "Autonomous agent: baseline elevated scrutiny", Weight: 10},
	}

	risks := ASIRisksForFactors(factors)
	ids := asiRiskIDs(risks)

	assertContainsASI(t, ids, "ASI03:2026")
	if len(ids) != 1 {
		t.Fatalf("expected exactly 1 ASI risk for agent_autonomy_baseline, got %d: %v", len(ids), ids)
	}
}

func TestASIRisksForFactors_DeduplicatesSharedCategory(t *testing.T) {
	// Both factors map ASI03 - should appear once, not twice, when both
	// are present together (the common case: an unattended agent also
	// gets the baseline autonomy factor).
	factors := []Factor{
		{Code: "agent_autonomy_baseline", Label: "Autonomous agent: baseline elevated scrutiny", Weight: 10},
		{Code: "agent_unattended", Label: "Agent operates without human-in-the-loop approval", Weight: 18},
	}

	risks := ASIRisksForFactors(factors)
	count := 0
	for _, r := range risks {
		if r.ID == "ASI03:2026" {
			count++
		}
	}
	if count != 1 {
		t.Fatalf("expected ASI03:2026 to appear exactly once across both factors, got %d", count)
	}
}

func TestASIRisksForFactors_NonAgenticFactorsMapToNothing(t *testing.T) {
	// These have NHI 2025 mappings (owasp.go) but no Agentic 2026 mapping -
	// the two standards are deliberately kept separate.
	factors := []Factor{
		{Code: "priv_admin", Label: "Administrative privilege level", Weight: 25},
		{Code: "auth_api_key", Label: "Long-lived API key authentication", Weight: 12},
	}

	risks := ASIRisksForFactors(factors)
	if len(risks) != 0 {
		t.Fatalf("expected no ASI risks for non-agentic factors, got %v", risks)
	}
}

func TestASIRisksForFactors_EmptyInput(t *testing.T) {
	risks := ASIRisksForFactors(nil)
	if len(risks) != 0 {
		t.Fatalf("expected empty input to produce empty output, got %v", risks)
	}
}

// --- test helpers (separate names from owasp_test.go's to avoid collision) ---

func asiRiskIDs(risks []ASIRisk) []string {
	ids := make([]string, len(risks))
	for i, r := range risks {
		ids[i] = r.ID
	}
	return ids
}

func assertContainsASI(t *testing.T, haystack []string, needle string) {
	t.Helper()
	for _, s := range haystack {
		if s == needle {
			return
		}
	}
	t.Fatalf("expected %v to contain %q", haystack, needle)
}

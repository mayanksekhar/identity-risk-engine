package risk

import (
	"sort"
	"testing"
)

func TestOWASPRisksForFactors_MapsKnownFactors(t *testing.T) {
	factors := []Factor{
		{Code: "auth_none", Label: "No authentication mechanism on record", Weight: 30},
		{Code: "priv_admin", Label: "Administrative privilege level", Weight: 25},
	}

	risks, frontier := OWASPRisksForFactors(factors)

	if len(frontier) != 0 {
		t.Fatalf("expected no frontier notes for these factors, got %v", frontier)
	}

	ids := riskIDs(risks)
	assertContains(t, ids, "NHI4:2025")
	assertContains(t, ids, "NHI5:2025")
}

func TestOWASPRisksForFactors_DeduplicatesAcrossFactors(t *testing.T) {
	// priv_admin and standing_access_breadth both map to NHI5 - should
	// appear once in the result, not twice.
	factors := []Factor{
		{Code: "priv_admin", Label: "Administrative privilege level", Weight: 25},
		{Code: "standing_access_breadth", Label: "Broad standing access grants", Weight: 12},
	}

	risks, _ := OWASPRisksForFactors(factors)

	count := 0
	for _, r := range risks {
		if r.ID == "NHI5:2025" {
			count++
		}
	}
	if count != 1 {
		t.Fatalf("expected NHI5:2025 to appear exactly once, got %d", count)
	}
}

func TestOWASPRisksForFactors_MultiMapFactor(t *testing.T) {
	// auth_api_key maps to both NHI4 (insecure auth) and NHI7 (long-lived secrets)
	factors := []Factor{
		{Code: "auth_api_key", Label: "Long-lived API key authentication", Weight: 12},
	}

	risks, _ := OWASPRisksForFactors(factors)
	ids := riskIDs(risks)

	assertContains(t, ids, "NHI4:2025")
	assertContains(t, ids, "NHI7:2025")
	if len(ids) != 2 {
		t.Fatalf("expected exactly 2 risks for auth_api_key, got %d: %v", len(ids), ids)
	}
}

func TestOWASPRisksForFactors_FrontierFactorsSurfacedSeparately(t *testing.T) {
	factors := []Factor{
		{Code: "agent_unattended", Label: "Agent operates without human-in-the-loop approval", Weight: 18},
	}

	risks, frontier := OWASPRisksForFactors(factors)

	if len(risks) != 0 {
		t.Fatalf("expected agent_unattended to map to zero established OWASP risks, got %v", risks)
	}
	if len(frontier) != 1 {
		t.Fatalf("expected exactly 1 frontier note, got %d", len(frontier))
	}
}

func TestOWASPRisksForFactors_UnknownFactorCodeIgnored(t *testing.T) {
	factors := []Factor{
		{Code: "totally_made_up_code", Label: "should not map to anything", Weight: 5},
	}

	risks, frontier := OWASPRisksForFactors(factors)

	if len(risks) != 0 || len(frontier) != 0 {
		t.Fatalf("expected unknown factor code to map to nothing, got risks=%v frontier=%v", risks, frontier)
	}
}

func TestOWASPRisksForFactors_EmptyInput(t *testing.T) {
	risks, frontier := OWASPRisksForFactors(nil)
	if len(risks) != 0 || len(frontier) != 0 {
		t.Fatalf("expected empty input to produce empty output, got risks=%v frontier=%v", risks, frontier)
	}
}

// --- test helpers ---

func riskIDs(risks []OWASPNHIRisk) []string {
	ids := make([]string, len(risks))
	for i, r := range risks {
		ids[i] = r.ID
	}
	sort.Strings(ids)
	return ids
}

func assertContains(t *testing.T, haystack []string, needle string) {
	t.Helper()
	for _, s := range haystack {
		if s == needle {
			return
		}
	}
	t.Fatalf("expected %v to contain %q", haystack, needle)
}

package risk

import (
	"testing"
	"time"

	"gitlab.com/mayanksekhar/identity-risk-engine/internal/identity"
)

func fixedClock(t time.Time) func() time.Time {
	return func() time.Time { return t }
}

func TestScore_MFAReducesRisk(t *testing.T) {
	now := time.Date(2026, 6, 20, 0, 0, 0, 0, time.UTC)
	engine := NewEngineWithClock(fixedClock(now))

	noMFA := identity.Identity{
		ID: "u1", Type: identity.TypeHuman, AuthMethod: identity.AuthPassword,
		Privilege: identity.PrivilegeStandard, Active: true, LastActiveAt: now,
	}
	withMFA := identity.Identity{
		ID: "u2", Type: identity.TypeHuman, AuthMethod: identity.AuthMFA,
		Privilege: identity.PrivilegeStandard, Active: true, LastActiveAt: now,
	}

	scoreNoMFA := engine.Score(noMFA)
	scoreMFA := engine.Score(withMFA)

	if scoreMFA.Total >= scoreNoMFA.Total {
		t.Fatalf("expected MFA score (%d) to be lower than no-MFA score (%d)", scoreMFA.Total, scoreNoMFA.Total)
	}
}

func TestScore_AdminPrivilegeIncreasesRisk(t *testing.T) {
	now := time.Now()
	engine := NewEngineWithClock(fixedClock(now))

	standard := identity.Identity{ID: "u1", Type: identity.TypeHuman, AuthMethod: identity.AuthMFA, Privilege: identity.PrivilegeStandard, Active: true, LastActiveAt: now}
	admin := identity.Identity{ID: "u2", Type: identity.TypeHuman, AuthMethod: identity.AuthMFA, Privilege: identity.PrivilegeAdmin, Active: true, LastActiveAt: now}

	if engine.Score(admin).Total <= engine.Score(standard).Total {
		t.Fatalf("expected admin privilege to increase risk score")
	}
}

func TestScore_StalenessThresholdIsTypeAware(t *testing.T) {
	now := time.Date(2026, 6, 20, 0, 0, 0, 0, time.UTC)
	engine := NewEngineWithClock(fixedClock(now))

	// 5 days inactive: fine for a human (threshold 30), but stale for an agent (threshold 3).
	fiveDaysAgo := now.Add(-5 * 24 * time.Hour)

	human := identity.Identity{ID: "u1", Type: identity.TypeHuman, AuthMethod: identity.AuthMFA, Active: true, LastActiveAt: fiveDaysAgo}
	agent := identity.Identity{ID: "a1", Type: identity.TypeAIAgent, AuthMethod: identity.AuthOIDC, Active: true, LastActiveAt: fiveDaysAgo}

	humanScore := engine.Score(human)
	agentScore := engine.Score(agent)

	for _, f := range humanScore.Factors {
		if f.Code == "stale_activity" {
			t.Fatalf("did not expect stale_activity factor for human at 5 days inactive")
		}
	}

	found := false
	for _, f := range agentScore.Factors {
		if f.Code == "stale_activity" {
			found = true
		}
	}
	if !found {
		t.Fatalf("expected stale_activity factor for agent at 5 days inactive (threshold=3)")
	}
}

func TestScore_UnattendedAgentTagAddsRisk(t *testing.T) {
	now := time.Now()
	engine := NewEngineWithClock(fixedClock(now))

	attended := identity.Identity{ID: "a1", Type: identity.TypeAIAgent, AuthMethod: identity.AuthOIDC, Active: true, LastActiveAt: now}
	unattended := identity.Identity{ID: "a2", Type: identity.TypeAIAgent, AuthMethod: identity.AuthOIDC, Active: true, LastActiveAt: now, Tags: []string{"unattended"}}

	if engine.Score(unattended).Total <= engine.Score(attended).Total {
		t.Fatalf("expected unattended tag to increase agent risk score")
	}
}

func TestScore_TotalIsClampedToValidRange(t *testing.T) {
	now := time.Now()
	staleDate := now.Add(-365 * 24 * time.Hour)
	engine := NewEngineWithClock(fixedClock(now))

	worst := identity.Identity{
		ID: "u1", Type: identity.TypeAIAgent, AuthMethod: identity.AuthNone,
		Privilege: identity.PrivilegeAdmin, Active: true, LastActiveAt: staleDate,
		StandingAccess: []string{"a", "b", "c", "d", "e", "f", "g", "h"},
		Tags:           []string{"unattended"},
	}

	score := engine.Score(worst)
	if score.Total < 0 || score.Total > 100 {
		t.Fatalf("expected total in [0,100], got %d", score.Total)
	}
	if score.Severity != SeverityCritical {
		t.Fatalf("expected critical severity for worst-case identity, got %s", score.Severity)
	}
}

func TestSeverityFor_Boundaries(t *testing.T) {
	cases := []struct {
		total int
		want  Severity
	}{
		{0, SeverityLow}, {19, SeverityLow},
		{20, SeverityMedium}, {44, SeverityMedium},
		{45, SeverityHigh}, {69, SeverityHigh},
		{70, SeverityCritical}, {100, SeverityCritical},
	}
	for _, c := range cases {
		if got := severityFor(c.total); got != c.want {
			t.Errorf("severityFor(%d) = %s, want %s", c.total, got, c.want)
		}
	}
}

func TestScoreAll_PreservesCount(t *testing.T) {
	engine := NewEngine()
	ids := []identity.Identity{
		{ID: "1", Type: identity.TypeHuman, AuthMethod: identity.AuthMFA, Active: true, LastActiveAt: time.Now()},
		{ID: "2", Type: identity.TypeNHI, AuthMethod: identity.AuthAPIKey, Active: true, LastActiveAt: time.Now()},
	}
	scores := engine.ScoreAll(ids)
	if len(scores) != len(ids) {
		t.Fatalf("expected %d scores, got %d", len(ids), len(scores))
	}
}

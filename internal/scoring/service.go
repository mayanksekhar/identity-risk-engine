// Package scoring orchestrates the read path used by both the API and the
// dashboard: pull identities from the store, run the deterministic risk
// engine, and optionally enrich with an LLM-generated explanation.
package scoring

import (
	"context"
	"sort"
	"time"

	"gitlab.com/mayanksekhar/identity-risk-engine/internal/identity"
	"gitlab.com/mayanksekhar/identity-risk-engine/internal/llm"
	"gitlab.com/mayanksekhar/identity-risk-engine/internal/risk"
)

// IdentityStore is the minimal read interface scoring needs from storage,
// allowing the in-memory store (or a future DB-backed one) to satisfy it.
type IdentityStore interface {
	All() []identity.Identity
	Get(id string) (identity.Identity, bool)
}

// Result pairs an identity with its score for API/dashboard consumption.
type Result struct {
	Identity    identity.Identity `json:"identity"`
	Score       risk.Score        `json:"score"`
	Explanation string            `json:"explanation,omitempty"`
}

// Service is the orchestration entry point used by HTTP handlers.
type Service struct {
	store     IdentityStore
	engine    *risk.Engine
	llmClient *llm.Client
}

// NewService constructs the scoring service.
func NewService(store IdentityStore, engine *risk.Engine, llmClient *llm.Client) *Service {
	return &Service{store: store, engine: engine, llmClient: llmClient}
}

// Summary aggregates dashboard-level counters.
type Summary struct {
	TotalIdentities int            `json:"total_identities"`
	HumanCount      int            `json:"human_count"`
	NHICount        int            `json:"nhi_count"`
	AgentCount      int            `json:"agent_count"`
	AnomalyCount    int            `json:"anomaly_count"` // identities scored high or critical
	BySeverity      map[string]int `json:"by_severity"`
}

// ListScored returns every identity with its computed score, sorted by
// risk score descending (highest risk first) — this ordering is what makes
// the dashboard table immediately useful.
func (s *Service) ListScored() []Result {
	ids := s.store.All()
	results := make([]Result, 0, len(ids))
	for _, id := range ids {
		sc := s.engine.Score(id)
		results = append(results, Result{Identity: id, Score: sc})
	}
	sort.Slice(results, func(i, j int) bool { return results[i].Score.Total > results[j].Score.Total })
	return results
}

// GetScored returns a single identity's score, with an LLM-generated
// narrative explanation. ctx carries the caller's deadline so a slow LLM
// can't hang a request indefinitely.
func (s *Service) GetScored(ctx context.Context, id string) (Result, bool) {
	rec, ok := s.store.Get(id)
	if !ok {
		return Result{}, false
	}
	sc := s.engine.Score(rec)
	explanation := s.llmClient.Explain(ctx, rec, sc)
	return Result{Identity: rec, Score: sc, Explanation: explanation}, true
}

// Summarize computes dashboard-level aggregate counters.
func (s *Service) Summarize() Summary {
	results := s.ListScored()
	sum := Summary{
		BySeverity: map[string]int{
			string(risk.SeverityLow): 0, string(risk.SeverityMedium): 0,
			string(risk.SeverityHigh): 0, string(risk.SeverityCritical): 0,
		},
	}
	for _, r := range results {
		sum.TotalIdentities++
		switch r.Identity.Type {
		case identity.TypeHuman:
			sum.HumanCount++
		case identity.TypeNHI:
			sum.NHICount++
		case identity.TypeAIAgent:
			sum.AgentCount++
		}
		sum.BySeverity[string(r.Score.Severity)]++
		if r.Score.Severity == risk.SeverityHigh || r.Score.Severity == risk.SeverityCritical {
			sum.AnomalyCount++
		}
	}
	return sum
}

// ScoreHistoryPoint represents one bucket in a time-bucketed activity
// series, used for the "events by day" style chart on the dashboard.
type ScoreHistoryPoint struct {
	Label string `json:"label"`
	Value int    `json:"value"`
}

// WeeklyMFAChallenges is demo-stable synthetic data representing the kind
// of operational metric a real IdP integration would supply (e.g. MFA
// challenge volume by day). It is deterministic per ISO week so the
// dashboard looks consistent across refreshes without needing a real
// event pipeline.
func WeeklyMFAChallenges(now time.Time) []ScoreHistoryPoint {
	days := []string{"Mon", "Tue", "Wed", "Thu", "Fri", "Sat", "Sun"}
	base := []int{34, 58, 41, 89, 22, 31, 47} // illustrative distribution, Thu spike intentional
	out := make([]ScoreHistoryPoint, len(days))
	for i, d := range days {
		out[i] = ScoreHistoryPoint{Label: d, Value: base[i]}
	}
	return out
}

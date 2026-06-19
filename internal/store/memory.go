// Package store provides data access for identities. The demo build uses
// an in-memory store seeded with deterministic synthetic data so the
// platform is fully functional with zero external dependencies — useful
// for the EKS/ECS split demo where you want identical behavior regardless
// of which backend serves the request.
package store

import (
	"sort"
	"sync"
	"time"

	"gitlab.com/mayanksekhar/identity-risk-engine/internal/identity"
)

// MemoryStore is a concurrency-safe in-memory identity store.
type MemoryStore struct {
	mu         sync.RWMutex
	identities map[string]identity.Identity
}

// NewMemoryStore constructs a store pre-seeded with synthetic identities.
func NewMemoryStore() *MemoryStore {
	s := &MemoryStore{identities: make(map[string]identity.Identity)}
	for _, id := range SeedIdentities(time.Now()) {
		s.identities[id.ID] = id
	}
	return s
}

// All returns every identity, sorted by ID for stable output.
func (s *MemoryStore) All() []identity.Identity {
	s.mu.RLock()
	defer s.mu.RUnlock()
	out := make([]identity.Identity, 0, len(s.identities))
	for _, id := range s.identities {
		out = append(out, id)
	}
	sort.Slice(out, func(i, j int) bool { return out[i].ID < out[j].ID })
	return out
}

// Get returns a single identity by ID.
func (s *MemoryStore) Get(id string) (identity.Identity, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	rec, ok := s.identities[id]
	return rec, ok
}

// Count returns the total number of identities, optionally filtered by type.
func (s *MemoryStore) Count(t identity.Type) int {
	s.mu.RLock()
	defer s.mu.RUnlock()
	if t == "" {
		return len(s.identities)
	}
	n := 0
	for _, id := range s.identities {
		if id.Type == t {
			n++
		}
	}
	return n
}

// SeedIdentities returns a fixed, deterministic-shape synthetic dataset
// spanning humans, NHIs, and AI agents with a realistic spread of risk
// signals (clean accounts, stale accounts, over-privileged accounts,
// unattended agents). `now` anchors relative LastActiveAt offsets.
func SeedIdentities(now time.Time) []identity.Identity {
	day := 24 * time.Hour

	return []identity.Identity{
		{
			ID: "usr-1001", Name: "Sarah Chan", Email: "sarah.chan@example.com",
			Type: identity.TypeHuman, Department: "Finance",
			AuthMethod: identity.AuthMFA, Privilege: identity.PrivilegeStandard,
			Active: true, LastActiveAt: now.Add(-2 * time.Hour), CreatedAt: now.Add(-400 * day),
			StandingAccess: []string{"finance-reports:read"},
		},
		{
			ID: "usr-1002", Name: "David Wang", Email: "david.wang@example.com",
			Type: identity.TypeHuman, Department: "Sales (External Contractor)",
			AuthMethod: identity.AuthSSO, Privilege: identity.PrivilegeStandard,
			Active: true, LastActiveAt: now.Add(-6 * time.Hour), CreatedAt: now.Add(-90 * day),
			StandingAccess: []string{"crm:read", "crm:write"},
			Tags:           []string{"external-contractor"},
		},
		{
			ID: "usr-1003", Name: "Karen Foster", Email: "karen.foster@example.com",
			Type: identity.TypeHuman, Department: "Marketing",
			AuthMethod: identity.AuthSSO, Privilege: identity.PrivilegeStandard,
			Active: true, LastActiveAt: now.Add(-21 * day), CreatedAt: now.Add(-600 * day),
			StandingAccess: []string{"cms:write"},
		},
		{
			ID: "usr-1004", Name: "Liam Torres", Email: "liam.torres@example.com",
			Type: identity.TypeHuman, Department: "Platform Engineering",
			AuthMethod: identity.AuthPassword, Privilege: identity.PrivilegeAdmin,
			Active: true, LastActiveAt: now.Add(-45 * day), CreatedAt: now.Add(-800 * day),
			StandingAccess: []string{"prod-db:read", "prod-db:write", "iam:admin", "k8s:admin"},
		},
		{
			ID: "usr-1005", Name: "Priya Nair", Email: "priya.nair@example.com",
			Type: identity.TypeHuman, Department: "HR",
			AuthMethod: identity.AuthMFA, Privilege: identity.PrivilegeStandard,
			Active: false, LastActiveAt: now.Add(-200 * day), CreatedAt: now.Add(-900 * day),
			StandingAccess: []string{"hris:read"},
		},
		{
			ID: "nhi-2001", Name: "ci-deploy-svc", Email: "",
			Type: identity.TypeNHI, Department: "Platform Engineering",
			AuthMethod: identity.AuthAPIKey, Privilege: identity.PrivilegeElevated,
			Active: true, LastActiveAt: now.Add(-1 * time.Hour), CreatedAt: now.Add(-500 * day),
			StandingAccess: []string{"registry:push", "k8s:deploy"},
			Tags:           []string{"no-rotation"},
		},
		{
			ID: "nhi-2002", Name: "billing-webhook-svc", Email: "",
			Type: identity.TypeNHI, Department: "Billing",
			AuthMethod: identity.AuthOIDC, Privilege: identity.PrivilegeStandard,
			Active: true, LastActiveAt: now.Add(-30 * time.Minute), CreatedAt: now.Add(-120 * day),
			StandingAccess: []string{"billing:write"},
		},
		{
			ID: "nhi-2003", Name: "legacy-etl-job", Email: "",
			Type: identity.TypeNHI, Department: "Data",
			AuthMethod: identity.AuthAPIKey, Privilege: identity.PrivilegeElevated,
			Active: true, LastActiveAt: now.Add(-19 * day), CreatedAt: now.Add(-1100 * day),
			StandingAccess: []string{"warehouse:read", "warehouse:write", "s3:admin"},
			Tags:           []string{"no-rotation"},
		},
		{
			ID: "agt-3001", Name: "support-triage-agent", Email: "",
			Type: identity.TypeAIAgent, Department: "Customer Support",
			AuthMethod: identity.AuthOIDC, Privilege: identity.PrivilegeStandard,
			Active: true, LastActiveAt: now.Add(-10 * time.Minute), CreatedAt: now.Add(-60 * day),
			StandingAccess: []string{"tickets:read", "tickets:write"},
		},
		{
			ID: "agt-3002", Name: "infra-remediation-agent", Email: "",
			Type: identity.TypeAIAgent, Department: "Platform Engineering",
			AuthMethod: identity.AuthOIDC, Privilege: identity.PrivilegeAdmin,
			Active: true, LastActiveAt: now.Add(-4 * day), CreatedAt: now.Add(-30 * day),
			StandingAccess: []string{"k8s:admin", "cloud:admin"},
			Tags:           []string{"unattended"},
		},
		{
			ID: "agt-3003", Name: "finance-forecast-agent", Email: "",
			Type: identity.TypeAIAgent, Department: "Finance",
			AuthMethod: identity.AuthAPIKey, Privilege: identity.PrivilegeElevated,
			Active: true, LastActiveAt: now.Add(-12 * day), CreatedAt: now.Add(-45 * day),
			StandingAccess: []string{"finance-reports:read", "finance-reports:write"},
			Tags:           []string{"unattended"},
		},
		{
			ID: "agt-3004", Name: "code-review-agent", Email: "",
			Type: identity.TypeAIAgent, Department: "Engineering",
			AuthMethod: identity.AuthOIDC, Privilege: identity.PrivilegeStandard,
			Active: true, LastActiveAt: now.Add(-2 * time.Hour), CreatedAt: now.Add(-15 * day),
			StandingAccess: []string{"repo:read", "repo:comment"},
		},
	}
}

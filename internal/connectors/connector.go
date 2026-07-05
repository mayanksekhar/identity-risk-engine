// Package connectors defines the shared interface that every identity data
// source implements - GitHub today, AWS IAM and Kubernetes RBAC later. The
// scoring engine and dashboard never import a specific connector; they only
// depend on this interface, so adding a new source never requires changing
// existing code.
package connectors

import (
	"context"

	"gitlab.com/mayanksekhar/identity-risk-engine/internal/identity"
)

// Connector fetches identities from a real external source and normalizes
// them into this application's domain model. Every connector is read-only -
// none of them modify the source system.
type Connector interface {
	// Name identifies the connector for logging and diagnostics, e.g.
	// "github:3-orgs".
	Name() string

	// FetchIdentities returns every identity this connector can see. Called
	// fresh on every request in v1 - there is no caching yet. See
	// PRODUCT-ROADMAP.md Phase 3 for the planned Postgres-backed cache and
	// score-history work; that is the next upgrade once this connector is
	// proven against real orgs.
	FetchIdentities(ctx context.Context) ([]identity.Identity, error)
}

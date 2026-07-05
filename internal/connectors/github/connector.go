package github

import (
	"context"
	"fmt"
	"log"

	"gitlab.com/mayanksekhar/identity-risk-engine/internal/connectors"
	"gitlab.com/mayanksekhar/identity-risk-engine/internal/identity"
)

// Connector implements connectors.Connector against one or more GitHub
// organizations using a single token. Data completeness varies per org
// depending on whether the token holder administers that org - see the
// package doc comment for what degrades and how.
type Connector struct {
	client *client
	orgs   []string
}

// compile-time assertion that Connector satisfies the shared interface.
var _ connectors.Connector = (*Connector)(nil)

// New constructs a GitHub connector. orgs is the list of organization
// logins to scan. token must have at minimum read:org scope for member
// lists; admin:org plus organization-owner status is required for 2FA
// status and fine-grained PAT data on orgs the token holder administers.
// Orgs the token holder does not administer are scanned with reduced data
// (public members only) rather than skipped entirely.
func New(token string, orgs []string) *Connector {
	return &Connector{
		client: newClient(token),
		orgs:   orgs,
	}
}

// Name identifies this connector instance for logging.
func (c *Connector) Name() string {
	return fmt.Sprintf("github:%d-orgs", len(c.orgs))
}

// FetchIdentities scans every configured org and returns the union of all
// identities found. A failure scanning one org is logged and that org is
// skipped rather than failing the whole fetch - one misconfigured or
// inaccessible org should not remove visibility into the others.
func (c *Connector) FetchIdentities(ctx context.Context) ([]identity.Identity, error) {
	var all []identity.Identity

	for _, org := range c.orgs {
		ids, err := c.fetchOrg(ctx, org)
		if err != nil {
			log.Printf("github connector: skipping org %q after error: %v", org, err)
			continue
		}
		all = append(all, ids...)
	}

	return all, nil
}

// All implements scoring.IdentityStore by fetching live from GitHub on
// every call. There is no caching yet (PRODUCT-ROADMAP.md Phase 3 covers
// the planned Postgres-backed cache) - every dashboard load or API request
// triggers a fresh set of GitHub API calls. Acceptable for personal or
// small-team use; add caching before pointing this at a large org behind a
// public-facing dashboard.
func (c *Connector) All() []identity.Identity {
	ids, err := c.FetchIdentities(context.Background())
	if err != nil {
		log.Printf("github connector: FetchIdentities failed: %v", err)
		return nil
	}
	return ids
}

// Get implements scoring.IdentityStore by fetching all identities and
// scanning for a matching ID. O(n) per call and re-fetches from GitHub
// every time - acceptable given the explicit no-caching decision for v1;
// revisit alongside the caching work in Phase 3.
func (c *Connector) Get(id string) (identity.Identity, bool) {
	for _, i := range c.All() {
		if i.ID == id {
			return i, true
		}
	}
	return identity.Identity{}, false
}

// fetchOrg scans a single org across all three identity sources: human
// members, fine-grained PATs, and installed GitHub Apps.
func (c *Connector) fetchOrg(ctx context.Context, org string) ([]identity.Identity, error) {
	var out []identity.Identity

	// --- Human members ---
	members, err := fetchMembers(ctx, c.client, org)
	if err != nil {
		return nil, fmt.Errorf("fetching members for %s: %w", org, err)
	}

	disabled2FA, has2FAData, err := fetch2FADisabledMembers(ctx, c.client, org)
	if err != nil {
		log.Printf("github connector: 2fa status unavailable for org %q: %v", org, err)
	}
	disabled2FASet := make(map[string]bool, len(disabled2FA))
	for _, m := range disabled2FA {
		disabled2FASet[m.Login] = true
	}

	for _, m := range members {
		role := "member"
		if detail, found, mErr := fetchMembershipDetail(ctx, c.client, org, m.Login); mErr == nil && found {
			role = detail.Role
		}

		var has2FA *bool
		if has2FAData {
			v := !disabled2FASet[m.Login]
			has2FA = &v
		}

		out = append(out, transformMember(m, role, has2FA, org))
	}

	// --- Fine-grained PATs (owner-only, requires org opt-in) ---
	pats, patsFound, err := fetchFineGrainedPATs(ctx, c.client, org)
	if err != nil {
		log.Printf("github connector: PAT listing unavailable for org %q: %v", org, err)
	} else if patsFound {
		for _, p := range pats {
			out = append(out, transformPAT(p, org))
		}
	}

	// --- Installed GitHub Apps ---
	apps, appsFound, err := fetchInstalledApps(ctx, c.client, org)
	if err != nil {
		log.Printf("github connector: app listing unavailable for org %q: %v", org, err)
	} else if appsFound {
		for _, a := range apps {
			out = append(out, transformApp(a, org))
		}
	}

	return out, nil
}

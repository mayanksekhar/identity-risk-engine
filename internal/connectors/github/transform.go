package github

import (
	"fmt"
	"time"

	"gitlab.com/mayanksekhar/identity-risk-engine/internal/identity"
)

// transformMember converts a GitHub org member into an Identity.
//
// Known limitation: GitHub's members API does not expose per-member last
// activity, and computing it accurately would require either GitHub
// Enterprise's audit log API or per-user event-stream polling (expensive
// against rate limits for a large org). LastActiveAt is therefore set to
// "now" - a deliberate choice to never falsely flag a member as stale when
// the real answer is unknown, rather than guessing a fabricated date. The
// "activity-data-unavailable" tag makes this limitation visible in the API
// and dashboard rather than silently hiding it.
//
// has2FA is nil when the token holder cannot see 2FA status for this org
// (not an owner). In that case auth posture is left at AuthNone rather
// than assumed secure - an unknown security posture must never score as if
// it were verified good.
func transformMember(m orgMember, role string, has2FA *bool, org string) identity.Identity {
	tags := []string{"github-sourced", "activity-data-unavailable"}

	auth := identity.AuthNone
	if has2FA != nil {
		if *has2FA {
			auth = identity.AuthMFA
		} else {
			auth = identity.AuthPassword
		}
	} else {
		tags = append(tags, "auth-status-unknown")
	}

	privilege := identity.PrivilegeStandard
	if role == "admin" {
		privilege = identity.PrivilegeAdmin
	}

	now := time.Now()
	return identity.Identity{
		ID:           fmt.Sprintf("gh-user-%d", m.ID),
		Name:         m.Login,
		Type:         identity.TypeHuman,
		Department:   fmt.Sprintf("GitHub org: %s", org),
		AuthMethod:   auth,
		Privilege:    privilege,
		Active:       true,
		LastActiveAt: now,
		CreatedAt:    now,
		Tags:         tags,
	}
}

// transformPAT converts a fine-grained personal access token into an NHI
// Identity. AccessGrantedAt is used as both CreatedAt and an approximation
// of LastActiveAt - this endpoint does not expose true last-used
// telemetry, so a real GitHub-sourced timestamp (when access was granted)
// is used in preference to fabricating "now", since a PAT that has sat
// unused since grant is exactly the improper-offboarding risk this tool
// exists to catch.
func transformPAT(p finegrainedPAT, org string) identity.Identity {
	tags := []string{"github-sourced"}
	if p.TokenExpiresAt == nil {
		tags = append(tags, "no-rotation")
	}

	grantedAt := parseGitHubTime(p.AccessGrantedAt)

	privilege := identity.PrivilegeStandard
	if p.RepositorySelection == "all" {
		privilege = identity.PrivilegeElevated
	}

	return identity.Identity{
		ID:             fmt.Sprintf("gh-pat-%d", p.ID),
		Name:           fmt.Sprintf("PAT: %s", p.Owner.Login),
		Type:           identity.TypeNHI,
		Department:     fmt.Sprintf("GitHub org: %s", org),
		AuthMethod:     identity.AuthAPIKey,
		Privilege:      privilege,
		Active:         !p.TokenExpired,
		LastActiveAt:   grantedAt,
		CreatedAt:      grantedAt,
		StandingAccess: []string{fmt.Sprintf("repo-selection:%s", p.RepositorySelection)},
		Tags:           tags,
	}
}

// transformApp converts an installed GitHub App into an NHI Identity.
// GitHub App installations authenticate via short-lived installation
// access tokens (roughly analogous to OIDC workload identity - never a
// static long-lived secret), so AuthOIDC is the accurate classification
// regardless of what permissions the app was granted.
func transformApp(a installedApp, org string) identity.Identity {
	privilege := identity.PrivilegeStandard
	if hasAdminPermission(a.Permissions) {
		privilege = identity.PrivilegeAdmin
	} else if len(a.Permissions) > 3 {
		privilege = identity.PrivilegeElevated
	}

	standingAccess := make([]string, 0, len(a.Permissions))
	for scope, level := range a.Permissions {
		standingAccess = append(standingAccess, fmt.Sprintf("%s:%s", scope, level))
	}

	return identity.Identity{
		ID:             fmt.Sprintf("gh-app-%d", a.ID),
		Name:           a.AppSlug,
		Type:           identity.TypeNHI,
		Department:     fmt.Sprintf("GitHub org: %s", org),
		AuthMethod:     identity.AuthOIDC,
		Privilege:      privilege,
		Active:         true,
		LastActiveAt:   parseGitHubTime(a.UpdatedAt),
		CreatedAt:      parseGitHubTime(a.CreatedAt),
		StandingAccess: standingAccess,
		Tags:           []string{"github-sourced"},
	}
}

func hasAdminPermission(perms map[string]string) bool {
	for _, level := range perms {
		if level == "admin" {
			return true
		}
	}
	return false
}

// parseGitHubTime parses GitHub's RFC3339 timestamps, returning the zero
// time on empty input or parse failure. A zero LastActiveAt makes
// DaysSinceActive very large, which correctly surfaces as a strong
// staleness signal rather than silently defaulting to "just active" -
// missing data about a credential's usage is itself a risk signal.
func parseGitHubTime(s string) time.Time {
	if s == "" {
		return time.Time{}
	}
	t, err := time.Parse(time.RFC3339, s)
	if err != nil {
		return time.Time{}
	}
	return t
}

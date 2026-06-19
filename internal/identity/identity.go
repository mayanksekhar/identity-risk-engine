// Package identity defines the core domain model for enterprise identities
// tracked by the Identity Risk Engine: humans, non-human identities (service
// accounts, API keys), and AI agents.
package identity

import "time"

// Type distinguishes the category of identity. Risk scoring weights differ
// significantly across these — an unattended AI agent with standing
// privileges is a materially different risk profile than a human with MFA.
type Type string

const (
	TypeHuman   Type = "human"
	TypeNHI     Type = "nhi"   // non-human identity: service accounts, API keys, workload identities
	TypeAIAgent Type = "agent" // autonomous or semi-autonomous AI agent identities
)

// AuthMethod captures how the identity authenticates.
type AuthMethod string

const (
	AuthNone     AuthMethod = "none"
	AuthPassword AuthMethod = "password"
	AuthSSO      AuthMethod = "sso"
	AuthMFA      AuthMethod = "mfa"
	AuthAPIKey   AuthMethod = "api_key"
	AuthOIDC     AuthMethod = "oidc_workload"
)

// PrivilegeLevel is a coarse classification of standing access.
type PrivilegeLevel string

const (
	PrivilegeStandard PrivilegeLevel = "standard"
	PrivilegeElevated PrivilegeLevel = "elevated"
	PrivilegeAdmin    PrivilegeLevel = "admin"
)

// Identity is the canonical record for any actor (human, service, or agent)
// under management by the platform.
type Identity struct {
	ID             string         `json:"id"`
	Name           string         `json:"name"`
	Email          string         `json:"email,omitempty"`
	Type           Type           `json:"type"`
	Department     string         `json:"department,omitempty"`
	AuthMethod     AuthMethod     `json:"auth_method"`
	Privilege      PrivilegeLevel `json:"privilege"`
	Active         bool           `json:"active"`
	LastActiveAt   time.Time      `json:"last_active_at"`
	CreatedAt      time.Time      `json:"created_at"`
	StandingAccess []string       `json:"standing_access"` // e.g. ["prod-db:read", "billing:write"]
	Tags           []string       `json:"tags,omitempty"`  // e.g. ["external-contractor", "unattended"]
}

// DaysSinceActive returns how many days have elapsed since LastActiveAt,
// relative to the provided reference time (injected for testability).
func (i Identity) DaysSinceActive(now time.Time) float64 {
	return now.Sub(i.LastActiveAt).Hours() / 24
}

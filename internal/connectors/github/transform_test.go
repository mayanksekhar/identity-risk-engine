package github

import (
	"testing"
	"time"

	"gitlab.com/mayanksekhar/identity-risk-engine/internal/identity"
)

func TestTransformMember_UnknownAuthDoesNotDefaultToSecure(t *testing.T) {
	m := orgMember{Login: "alice", ID: 1}
	id := transformMember(m, "member", nil, "acme-corp")

	if id.AuthMethod == identity.AuthMFA || id.AuthMethod == identity.AuthSSO {
		t.Fatalf("expected unknown 2FA status to never be classified as a secure auth method, got %s", id.AuthMethod)
	}
	if id.AuthMethod != identity.AuthNone {
		t.Fatalf("expected AuthNone when 2FA status is unknown, got %s", id.AuthMethod)
	}

	found := false
	for _, tag := range id.Tags {
		if tag == "auth-status-unknown" {
			found = true
		}
	}
	if !found {
		t.Fatalf("expected auth-status-unknown tag when has2FA is nil, got tags: %v", id.Tags)
	}
}

func TestTransformMember_WithMFAKnown(t *testing.T) {
	m := orgMember{Login: "bob", ID: 2}
	hasMFA := true
	id := transformMember(m, "member", &hasMFA, "acme-corp")

	if id.AuthMethod != identity.AuthMFA {
		t.Fatalf("expected AuthMFA when has2FA is true, got %s", id.AuthMethod)
	}
	for _, tag := range id.Tags {
		if tag == "auth-status-unknown" {
			t.Fatalf("did not expect auth-status-unknown tag when 2FA status is known")
		}
	}
}

func TestTransformMember_WithoutMFAKnown(t *testing.T) {
	m := orgMember{Login: "carol", ID: 3}
	hasMFA := false
	id := transformMember(m, "member", &hasMFA, "acme-corp")

	if id.AuthMethod != identity.AuthPassword {
		t.Fatalf("expected AuthPassword when has2FA is false, got %s", id.AuthMethod)
	}
}

func TestTransformMember_AdminRole(t *testing.T) {
	m := orgMember{Login: "dave", ID: 4}
	id := transformMember(m, "admin", nil, "acme-corp")

	if id.Privilege != identity.PrivilegeAdmin {
		t.Fatalf("expected PrivilegeAdmin for admin role, got %s", id.Privilege)
	}
}

func TestTransformPAT_NoExpiryTaggedNoRotation(t *testing.T) {
	p := finegrainedPAT{
		ID:                  100,
		Owner:               orgMember{Login: "ci-bot"},
		RepositorySelection: "all",
		AccessGrantedAt:     "2026-01-15T10:00:00Z",
		TokenExpiresAt:      nil,
	}
	id := transformPAT(p, "acme-corp")

	if id.Type != identity.TypeNHI {
		t.Fatalf("expected TypeNHI for a PAT, got %s", id.Type)
	}
	if id.AuthMethod != identity.AuthAPIKey {
		t.Fatalf("expected AuthAPIKey, got %s", id.AuthMethod)
	}
	if id.Privilege != identity.PrivilegeElevated {
		t.Fatalf("expected PrivilegeElevated for repository_selection=all, got %s", id.Privilege)
	}

	found := false
	for _, tag := range id.Tags {
		if tag == "no-rotation" {
			found = true
		}
	}
	if !found {
		t.Fatalf("expected no-rotation tag when TokenExpiresAt is nil, got tags: %v", id.Tags)
	}
}

func TestTransformPAT_WithExpiryNotTaggedNoRotation(t *testing.T) {
	expiry := "2027-01-15T10:00:00Z"
	p := finegrainedPAT{
		ID:                  101,
		Owner:               orgMember{Login: "deploy-bot"},
		RepositorySelection: "subset",
		AccessGrantedAt:     "2026-01-15T10:00:00Z",
		TokenExpiresAt:      &expiry,
	}
	id := transformPAT(p, "acme-corp")

	for _, tag := range id.Tags {
		if tag == "no-rotation" {
			t.Fatalf("did not expect no-rotation tag when TokenExpiresAt is set")
		}
	}
	if id.Privilege != identity.PrivilegeStandard {
		t.Fatalf("expected PrivilegeStandard for repository_selection=subset, got %s", id.Privilege)
	}
}

func TestTransformApp_AdminPermissionMapsToPrivilegeAdmin(t *testing.T) {
	a := installedApp{
		ID:      200,
		AppSlug: "ci-deploy-app",
		Permissions: map[string]string{
			"contents": "admin",
		},
		CreatedAt: "2026-01-01T00:00:00Z",
		UpdatedAt: "2026-06-01T00:00:00Z",
	}
	id := transformApp(a, "acme-corp")

	if id.Type != identity.TypeNHI {
		t.Fatalf("expected TypeNHI for a GitHub App, got %s", id.Type)
	}
	if id.AuthMethod != identity.AuthOIDC {
		t.Fatalf("expected AuthOIDC for GitHub App installation tokens, got %s", id.AuthMethod)
	}
	if id.Privilege != identity.PrivilegeAdmin {
		t.Fatalf("expected PrivilegeAdmin when any permission is admin, got %s", id.Privilege)
	}
}

func TestTransformApp_ManyPermissionsWithoutAdminMapsToElevated(t *testing.T) {
	a := installedApp{
		ID:      201,
		AppSlug: "read-only-scanner",
		Permissions: map[string]string{
			"contents":      "read",
			"issues":        "read",
			"pull_requests": "read",
			"metadata":      "read",
		},
	}
	id := transformApp(a, "acme-corp")

	if id.Privilege != identity.PrivilegeElevated {
		t.Fatalf("expected PrivilegeElevated for 4+ read permissions with no admin, got %s", id.Privilege)
	}
}

func TestParseGitHubTime_EmptyStringReturnsZeroValue(t *testing.T) {
	got := parseGitHubTime("")
	if !got.IsZero() {
		t.Fatalf("expected zero time for empty input, got %v", got)
	}
}

func TestParseGitHubTime_InvalidStringReturnsZeroValue(t *testing.T) {
	got := parseGitHubTime("not-a-timestamp")
	if !got.IsZero() {
		t.Fatalf("expected zero time for invalid input, got %v", got)
	}
}

func TestParseGitHubTime_ValidRFC3339(t *testing.T) {
	got := parseGitHubTime("2026-06-15T12:30:00Z")
	want := time.Date(2026, 6, 15, 12, 30, 0, 0, time.UTC)
	if !got.Equal(want) {
		t.Fatalf("expected %v, got %v", want, got)
	}
}

func TestHasAdminPermission(t *testing.T) {
	if !hasAdminPermission(map[string]string{"contents": "admin"}) {
		t.Fatal("expected true when a permission value is admin")
	}
	if hasAdminPermission(map[string]string{"contents": "read", "issues": "write"}) {
		t.Fatal("expected false when no permission value is admin")
	}
	if hasAdminPermission(map[string]string{}) {
		t.Fatal("expected false for empty permissions map")
	}
}

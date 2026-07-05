package github

import (
	"context"
	"fmt"
)

// fetchMembers returns every member of the org visible to this token.
// Works for any org - GitHub returns the public member list to any caller
// for orgs with restricted membership visibility, and the full list to
// members/owners.
func fetchMembers(ctx context.Context, c *client, org string) ([]orgMember, error) {
	members, _, err := getPaginated[orgMember](ctx, c, fmt.Sprintf("/orgs/%s/members", org))
	return members, err
}

// fetchMembershipDetail returns the role (admin/member) for a single
// member. Requires the token holder to have at least member-level
// visibility into the org; found=false for orgs the token holder cannot
// see membership roles for.
func fetchMembershipDetail(ctx context.Context, c *client, org, username string) (membershipDetail, bool, error) {
	var detail membershipDetail
	found, err := c.get(ctx, fmt.Sprintf("/orgs/%s/memberships/%s", org, username), &detail)
	return detail, found, err
}

// fetch2FADisabledMembers returns the logins of members without two-factor
// authentication enabled. This endpoint requires the token holder to be an
// organization owner. found=false means "this data is not available for
// this org" - callers must treat that as auth status unknown, never as
// "assume 2FA is enabled".
func fetch2FADisabledMembers(ctx context.Context, c *client, org string) ([]orgMember, bool, error) {
	return getPaginated[orgMember](ctx, c, fmt.Sprintf("/orgs/%s/members?filter=2fa_disabled", org))
}

// fetchFineGrainedPATs returns organization-scoped fine-grained personal
// access tokens with access to org resources. Owner-only, and only
// populated if the org has enabled fine-grained PAT access review.
func fetchFineGrainedPATs(ctx context.Context, c *client, org string) ([]finegrainedPAT, bool, error) {
	return getPaginated[finegrainedPAT](ctx, c, fmt.Sprintf("/orgs/%s/personal-access-tokens", org))
}

// fetchInstalledApps returns GitHub Apps installed on the org. Not
// paginated via getPaginated since the endpoint wraps its array in an
// object rather than returning a bare array; a single page covers the
// overwhelming majority of orgs (100 installed apps is a very high bar).
func fetchInstalledApps(ctx context.Context, c *client, org string) ([]installedApp, bool, error) {
	var resp installationsResponse
	found, err := c.get(ctx, fmt.Sprintf("/orgs/%s/installations?per_page=100", org), &resp)
	if err != nil {
		return nil, false, err
	}
	return resp.Installations, found, nil
}

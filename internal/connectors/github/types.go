package github

// Raw GitHub API response shapes. Only the fields this connector actually
// uses are declared here - GitHub's real responses carry many more fields.

// orgMember models one entry from GET /orgs/{org}/members.
type orgMember struct {
	Login string `json:"login"`
	ID    int64  `json:"id"`
	Type  string `json:"type"` // "User" or "Bot"
}

// membershipDetail models GET /orgs/{org}/memberships/{username}.
type membershipDetail struct {
	Role  string `json:"role"`  // "admin" or "member"
	State string `json:"state"` // "active" or "pending"
}

// finegrainedPAT models one entry from the org-level fine-grained personal
// access token review listing: GET /orgs/{org}/personal-access-tokens.
// Only returned to organization owners, and only if the org has the
// fine-grained PAT access-review setting enabled - otherwise this endpoint
// 404s, which this connector treats as "not available", not an error.
type finegrainedPAT struct {
	ID                  int64     `json:"id"`
	Owner               orgMember `json:"owner"`
	RepositorySelection string    `json:"repository_selection"` // "all" or "subset"
	AccessGrantedAt     string    `json:"access_granted_at"`
	TokenExpired        bool      `json:"token_expired"`
	TokenExpiresAt      *string   `json:"token_expires_at"`
}

// installedApp models one entry from GET /orgs/{org}/installations.
type installedApp struct {
	ID          int64             `json:"id"`
	AppSlug     string            `json:"app_slug"`
	AppID       int64             `json:"app_id"`
	Permissions map[string]string `json:"permissions"`
	CreatedAt   string            `json:"created_at"`
	UpdatedAt   string            `json:"updated_at"`
}

// installationsResponse wraps the installations list - this endpoint
// returns an object, not a bare array, unlike most GitHub list endpoints.
type installationsResponse struct {
	Installations []installedApp `json:"installations"`
}

# GitHub Connector Setup

## What this replaces

By default the app uses synthetic seed data (`internal/store/memory.go`).
Setting `GITHUB_ORGS` switches to the real GitHub connector - every
identity on the dashboard becomes a real GitHub org member, personal access
token, or installed GitHub App, scored by the exact same rule engine.

## Environment variables

| Variable | Required | Example | Purpose |
|---|---|---|---|
| `GITHUB_ORGS` | to enable the connector | `my-org,another-org` | Comma-separated org logins to scan |
| `GITHUB_TOKEN` | yes, if `GITHUB_ORGS` is set | `ghp_...` or `github_pat_...` | Auth for the GitHub API |

Leave `GITHUB_ORGS` unset to keep using synthetic data - nothing else
changes.

## Token scopes

Create a token at `https://github.com/settings/tokens`. What you get back
depends on scope and on whether you own/administer each target org:

| Scope | Unlocks |
|---|---|
| `read:org` | Member list (any org with visible membership) |
| `admin:org` | Membership role detail (admin vs member) |
| Org owner + `admin:org` | 2FA status per member (`members?filter=2fa_disabled`) |
| Org owner, fine-grained PAT review enabled | Fine-grained PAT listing with expiration data |
| `read:org` (installations) | Installed GitHub Apps list |

**For orgs you don't administer** (the "scan a few more orgs" case): you'll
get the public member list and nothing else. The connector does not fail or
skip these orgs - it fetches what it can and leaves the rest honestly
unknown (see Known Limitations below), tagging affected identities
`auth-status-unknown` rather than guessing.

**The fine-grained PAT endpoint requires an explicit org setting.** Even as
an owner with `admin:org`, `GET /orgs/{org}/personal-access-tokens` returns
404 unless the org has enabled "fine-grained personal access token access
requests" under Organization Settings > Personal access tokens. If it's not
enabled, the connector silently skips PAT data for that org - this is
expected, not a bug.

## Running it

```bash
GITHUB_ORGS=your-org \
GITHUB_TOKEN=ghp_your_token \
go run ./cmd/server
```

Then open `http://localhost:8080/` - the dashboard now shows real data from
`your-org`, scored against the same rule engine and OWASP NHI Top 10
mapping as the synthetic demo.

To scan multiple orgs in one dashboard view:

```bash
GITHUB_ORGS=org-one,org-two,org-three \
GITHUB_TOKEN=ghp_your_token \
go run ./cmd/server
```

A failure scanning any single org is logged and that org is skipped - the
others still populate the dashboard.

## Known limitations (v1)

Documented here deliberately rather than silently glossed over - each one
is a real gap, not a hidden assumption:

- **No caching.** Every dashboard load and API call re-fetches from GitHub
  live. Fine for personal use or a handful of orgs; will hit GitHub's
  5,000 req/hour authenticated rate limit if pointed at a very large org or
  refreshed constantly. Postgres-backed caching is the planned Phase 3
  upgrade (see `PRODUCT-ROADMAP.md`).
- **Human member `LastActiveAt` is not real telemetry.** GitHub's members
  API does not expose per-member last-activity. Getting a real answer
  requires GitHub Enterprise's audit log API or per-user event polling
  (expensive against rate limits). This connector sets it to "now" rather
  than fabricating a plausible-looking date, and tags every GitHub-sourced
  human identity `activity-data-unavailable` so this is visible in the API
  response and dashboard, not silently hidden.
- **PAT `LastActiveAt` is approximated from grant date**, not true
  last-used telemetry - the org-level PAT review endpoint doesn't expose
  real usage timestamps.
- **Unknown 2FA status is scored as `AuthNone`**, the weakest posture, not
  assumed secure. This is a deliberate security-first default: an unknown
  auth posture must never score as if it were verified good. It does mean
  member risk scores for orgs you don't administer will look worse than
  they might actually be - the honest fix is administering those orgs (or
  running your own GitHub App with the right permissions), not softening
  the default.

## What's next

See `PRODUCT-ROADMAP.md` Phase 2 (remediation actions - PAT revocation, key
rotation) and Phase 3 (Postgres persistence, score history, drift
detection) for where this connector is meant to go next.

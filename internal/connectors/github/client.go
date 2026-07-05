// Package github implements a connectors.Connector that fetches real
// identity data from one or more GitHub organizations: human members (with
// role and 2FA status where available), fine-grained personal access
// tokens with org access, and installed GitHub Apps.
//
// Data availability depends on the caller's token permissions relative to
// each target org:
//   - Orgs the token owner administers: full data (2FA status, PATs, apps)
//   - Orgs the token owner does not administer: public member list only;
//     authentication-posture fields are left explicitly unknown rather
//     than guessed, and tagged as such (see transform.go)
//
// This connector never writes to GitHub - every call is a GET request.
package github

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"
)

const apiBaseURL = "https://api.github.com"

// client is a minimal GitHub REST API v3 client. It intentionally avoids a
// third-party SDK - the API surface this connector needs is small enough
// that a thin, fully-auditable wrapper is preferable to a large dependency.
type client struct {
	token      string
	httpClient *http.Client
}

func newClient(token string) *client {
	return &client{
		token:      token,
		httpClient: &http.Client{Timeout: 15 * time.Second},
	}
}

// get performs an authenticated GET request and decodes the JSON response
// into out. It returns (found=false, nil) when the endpoint is genuinely
// not available for this token/org combination (404, or 403 due to
// insufficient scope) - that is expected and routine when scanning an org
// the token holder does not administer, not a failure. A 403 caused by
// rate-limit exhaustion is distinguished and returned as a real error,
// since silently swallowing that would hide a genuine problem.
func (c *client) get(ctx context.Context, path string, out any) (found bool, err error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, apiBaseURL+path, nil)
	if err != nil {
		return false, fmt.Errorf("building request for %s: %w", path, err)
	}
	req.Header.Set("Accept", "application/vnd.github+json")
	req.Header.Set("X-GitHub-Api-Version", "2022-11-28")
	if c.token != "" {
		req.Header.Set("Authorization", "Bearer "+c.token)
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return false, fmt.Errorf("requesting %s: %w", path, err)
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusForbidden {
		body, _ := io.ReadAll(resp.Body)
		if strings.Contains(strings.ToLower(string(body)), "rate limit") {
			return false, fmt.Errorf("GitHub API rate limit exceeded calling %s: %s", path, string(body))
		}
		// Forbidden due to insufficient scope for this org - expected when
		// scanning an org the token holder doesn't administer.
		return false, nil
	}

	if resp.StatusCode == http.StatusNotFound {
		return false, nil
	}

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return false, fmt.Errorf("unexpected status %d from %s: %s", resp.StatusCode, path, string(body))
	}

	if out == nil {
		return true, nil
	}

	if err := json.NewDecoder(resp.Body).Decode(out); err != nil {
		return false, fmt.Errorf("decoding response from %s: %w", path, err)
	}

	return true, nil
}

// getPaginated fetches every page of a list endpoint that returns a plain
// JSON array, following pages until a short page signals the end. A hard
// cap of 10 pages (1000 items at 100/page) prevents an unbounded fetch
// against a misbehaving or unexpectedly large org. found reflects whether
// the first page was available at all, so callers can distinguish "this
// org has zero items" from "this data isn't available for this token".
func getPaginated[T any](ctx context.Context, c *client, path string) (items []T, found bool, err error) {
	page := 1
	const perPage = 100
	const maxPages = 10

	for page <= maxPages {
		pagedPath := fmt.Sprintf("%s%sper_page=%d&page=%d", path, sepFor(path), perPage, page)
		var pageItems []T
		pageFound, pageErr := c.get(ctx, pagedPath, &pageItems)
		if pageErr != nil {
			return items, found, pageErr
		}
		if page == 1 {
			found = pageFound
		}
		if !pageFound {
			break
		}
		items = append(items, pageItems...)
		if len(pageItems) < perPage {
			break // last page
		}
		page++
	}

	return items, found, nil
}

// sepFor returns the correct query-string separator depending on whether
// path already contains a "?" from a caller-supplied filter parameter.
func sepFor(path string) string {
	if strings.Contains(path, "?") {
		return "&"
	}
	return "?"
}

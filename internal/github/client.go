package github

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"time"
)

// Client is the minimal GitHub REST surface this package needs. Kept
// as an interface so tests can inject a fake without spinning up HTTP.
type Client interface {
	// GetJSON fetches one page of a JSON endpoint into `out`. Returns
	// the parsed Link header so callers can page through.
	GetJSON(ctx context.Context, path string, out any) (Links, error)
	// RateLimited returns true if the most recent request hit a 403
	// with X-RateLimit-Remaining: 0. Callers use this to decide
	// whether to serve cached data and flag the report.
	RateLimited() bool
}

// Links is the parsed Link header — only `next` matters for our
// pagination needs.
type Links struct {
	Next string
}

type restClient struct {
	token    string
	http     *http.Client
	base     string
	userAgent string

	rateLimited bool
}

// NewClient returns a Client bound to api.github.com, authenticated by
// the given personal-access token. Empty token → request header omits
// auth, which works for public repos within the 60/hour unauth budget.
func NewClient(token string) Client {
	return &restClient{
		token: token,
		http: &http.Client{
			Timeout: 30 * time.Second,
		},
		base:      "https://api.github.com",
		userAgent: "repopulse",
	}
}

func (c *restClient) RateLimited() bool { return c.rateLimited }

func (c *restClient) GetJSON(ctx context.Context, path string, out any) (Links, error) {
	fullURL := path
	if !isAbsoluteURL(path) {
		fullURL = c.base + path
	}

	req, err := http.NewRequestWithContext(ctx, "GET", fullURL, nil)
	if err != nil {
		return Links{}, err
	}
	req.Header.Set("Accept", "application/vnd.github+json")
	req.Header.Set("X-GitHub-Api-Version", "2022-11-28")
	req.Header.Set("User-Agent", c.userAgent)
	if c.token != "" {
		req.Header.Set("Authorization", "Bearer "+c.token)
	}

	resp, err := c.http.Do(req)
	if err != nil {
		return Links{}, err
	}
	defer resp.Body.Close()

	// Rate-limit detection per GitHub docs: 403/429 with remaining=0.
	if resp.StatusCode == http.StatusForbidden || resp.StatusCode == http.StatusTooManyRequests {
		if remaining := resp.Header.Get("X-RateLimit-Remaining"); remaining != "" {
			if n, err := strconv.Atoi(remaining); err == nil && n == 0 {
				c.rateLimited = true
				return Links{}, ErrRateLimited
			}
		}
	}
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		body, _ := io.ReadAll(resp.Body)
		return Links{}, fmt.Errorf("github: %s %s → %d: %s", req.Method, fullURL, resp.StatusCode, string(body))
	}
	if out != nil {
		if err := json.NewDecoder(resp.Body).Decode(out); err != nil {
			return Links{}, fmt.Errorf("decode json: %w", err)
		}
	}
	return parseLinkHeader(resp.Header.Get("Link")), nil
}

// ErrRateLimited signals the caller should fall back to the cache.
var ErrRateLimited = errors.New("github: rate-limited")

func isAbsoluteURL(s string) bool {
	if s == "" {
		return false
	}
	u, err := url.Parse(s)
	return err == nil && u.Scheme != "" && u.Host != ""
}

// parseLinkHeader extracts the `next` URL from an RFC 5988 Link header.
// Only the forward link is relevant for walking a paginated result.
func parseLinkHeader(h string) Links {
	if h == "" {
		return Links{}
	}
	// Format: `<url>; rel="next", <url>; rel="last"` (and friends).
	parts := splitTopLevel(h, ',')
	for _, p := range parts {
		segs := splitTopLevel(p, ';')
		if len(segs) < 2 {
			continue
		}
		urlPart := trimMatching(segs[0], '<', '>')
		var isNext bool
		for _, seg := range segs[1:] {
			if containsTrimmed(seg, `rel="next"`) {
				isNext = true
				break
			}
		}
		if isNext {
			return Links{Next: urlPart}
		}
	}
	return Links{}
}

func splitTopLevel(s string, sep byte) []string {
	var out []string
	start := 0
	depth := 0
	for i := 0; i < len(s); i++ {
		switch s[i] {
		case '<':
			depth++
		case '>':
			depth--
		case sep:
			if depth == 0 {
				out = append(out, s[start:i])
				start = i + 1
			}
		}
	}
	if start < len(s) {
		out = append(out, s[start:])
	}
	return out
}

func trimMatching(s string, open, close byte) string {
	for len(s) > 0 && (s[0] == ' ' || s[0] == '\t') {
		s = s[1:]
	}
	if len(s) > 0 && s[0] == open {
		s = s[1:]
	}
	for len(s) > 0 && (s[len(s)-1] == ' ' || s[len(s)-1] == '\t') {
		s = s[:len(s)-1]
	}
	if len(s) > 0 && s[len(s)-1] == close {
		s = s[:len(s)-1]
	}
	return s
}

func containsTrimmed(s, substr string) bool {
	for len(s) > 0 && (s[0] == ' ' || s[0] == '\t') {
		s = s[1:]
	}
	return len(s) >= len(substr) && s[:len(substr)] == substr
}

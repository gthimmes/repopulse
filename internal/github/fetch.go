package github

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"time"
)

// PR is the subset of GitHub PR fields we persist + derive metrics from.
// Not the full API response — just what the prmetrics package needs.
type PR struct {
	Number     int       `json:"number"`
	Title      string    `json:"title"`
	State      string    `json:"state"`            // open / closed
	Merged     bool      `json:"merged"`           // closed AND merged_at != null
	AuthorLogin string   `json:"authorLogin"`
	CreatedAt  time.Time `json:"createdAt"`
	UpdatedAt  time.Time `json:"updatedAt"`
	MergedAt   time.Time `json:"mergedAt,omitempty"`
	ClosedAt   time.Time `json:"closedAt,omitempty"`
	MergedByLogin string `json:"mergedByLogin,omitempty"`
	Additions int        `json:"additions,omitempty"`
	Deletions int        `json:"deletions,omitempty"`
	Reviews   []Review   `json:"reviews"`
}

// Review is one review event on a PR.
type Review struct {
	Login       string    `json:"login"`
	State       string    `json:"state"` // APPROVED / CHANGES_REQUESTED / COMMENTED / DISMISSED
	SubmittedAt time.Time `json:"submittedAt"`
	BodyLen     int       `json:"bodyLen"` // len of the review body; proxy for "did they comment substantively"
}

// FetchOptions controls the scope of FetchMergedPRs.
type FetchOptions struct {
	Owner     string
	Repo      string
	Since     time.Time // only PRs with updatedAt >= Since
	CacheDir  string    // absolute path to <repo>/.repopulse/pr-cache/
	MaxPRs    int       // hard cap; 0 = unlimited (usually 500-1000)
}

// FetchResult holds the merged PRs found in the window plus a bit of
// metadata the caller surfaces as a banner (e.g. "served from cache
// because rate-limited").
type FetchResult struct {
	PRs         []PR
	CacheBanner string // non-empty → surface to user
	FromCache   bool
}

// FetchMergedPRs pulls merged PRs updated in the given window. Uses the
// on-disk cache for PR detail (reviews, additions) so incremental runs
// are cheap. If the API rate-limits us mid-fetch, falls back to what's
// already on disk and flags the result.
func FetchMergedPRs(ctx context.Context, c Client, opts FetchOptions) (FetchResult, error) {
	if opts.Owner == "" || opts.Repo == "" {
		return FetchResult{}, errors.New("Owner and Repo required")
	}
	if err := os.MkdirAll(opts.CacheDir, 0755); err != nil {
		return FetchResult{}, err
	}

	// Walk the closed-PR list, newest first. Stop when we cross Since.
	type prListEntry struct {
		Number    int       `json:"number"`
		State     string    `json:"state"`
		UpdatedAt time.Time `json:"updated_at"`
		MergedAt  *time.Time `json:"merged_at"`
	}

	var prs []PR
	banner := ""
	fromCache := false

	page := fmt.Sprintf("/repos/%s/%s/pulls?state=closed&sort=updated&direction=desc&per_page=100",
		opts.Owner, opts.Repo)
	fetchedList := 0
	for page != "" {
		var entries []prListEntry
		links, err := c.GetJSON(ctx, page, &entries)
		if err != nil {
			if errors.Is(err, ErrRateLimited) {
				banner = "GitHub rate limit hit. Showing whichever PRs were already cached locally."
				// Fall through to returning whatever we already decoded.
				cached, cerr := loadAllCached(opts.CacheDir, opts.Since)
				if cerr == nil {
					return FetchResult{PRs: cached, CacheBanner: banner, FromCache: true}, nil
				}
			}
			return FetchResult{}, err
		}
		if len(entries) == 0 {
			break
		}
		allOlder := true
		for _, e := range entries {
			if e.UpdatedAt.After(opts.Since) || e.UpdatedAt.Equal(opts.Since) {
				allOlder = false
			}
			if e.MergedAt == nil {
				continue // skip closed-unmerged
			}
			if e.UpdatedAt.Before(opts.Since) {
				continue
			}
			// Check cache: if we have the PR and its updatedAt matches,
			// skip the per-PR detail fetch.
			full, ok := loadCached(opts.CacheDir, e.Number)
			if ok && !full.UpdatedAt.Before(e.UpdatedAt) {
				prs = append(prs, full)
				continue
			}
			// Refetch detail + reviews.
			detailed, err := fetchPRDetail(ctx, c, opts.Owner, opts.Repo, e.Number)
			if err != nil {
				if errors.Is(err, ErrRateLimited) {
					banner = "GitHub rate limit hit mid-fetch. Some PRs in this window were served from an older cached copy; others are missing entirely."
					break // stop fetching; use whatever we have
				}
				return FetchResult{}, err
			}
			_ = saveCached(opts.CacheDir, detailed)
			prs = append(prs, detailed)
		}
		fetchedList += len(entries)
		if opts.MaxPRs > 0 && fetchedList >= opts.MaxPRs {
			break
		}
		if allOlder {
			break // list is sorted desc by updated_at; older pages won't help
		}
		if banner != "" {
			break
		}
		page = links.Next
	}

	return FetchResult{PRs: prs, CacheBanner: banner, FromCache: fromCache}, nil
}

func fetchPRDetail(ctx context.Context, c Client, owner, repo string, number int) (PR, error) {
	type prDetailJSON struct {
		Number    int        `json:"number"`
		Title     string     `json:"title"`
		State     string     `json:"state"`
		Merged    bool       `json:"merged"`
		User      struct{ Login string } `json:"user"`
		CreatedAt time.Time  `json:"created_at"`
		UpdatedAt time.Time  `json:"updated_at"`
		MergedAt  *time.Time `json:"merged_at"`
		ClosedAt  *time.Time `json:"closed_at"`
		MergedBy  *struct{ Login string } `json:"merged_by"`
		Additions int        `json:"additions"`
		Deletions int        `json:"deletions"`
	}
	var d prDetailJSON
	_, err := c.GetJSON(ctx, fmt.Sprintf("/repos/%s/%s/pulls/%d", owner, repo, number), &d)
	if err != nil {
		return PR{}, err
	}

	// Reviews
	type reviewJSON struct {
		User struct{ Login string } `json:"user"`
		State string `json:"state"`
		SubmittedAt time.Time `json:"submitted_at"`
		Body string `json:"body"`
	}
	var rs []reviewJSON
	_, err = c.GetJSON(ctx,
		fmt.Sprintf("/repos/%s/%s/pulls/%d/reviews?per_page=100", owner, repo, number),
		&rs)
	if err != nil {
		return PR{}, err
	}
	reviews := make([]Review, 0, len(rs))
	for _, r := range rs {
		reviews = append(reviews, Review{
			Login:       r.User.Login,
			State:       r.State,
			SubmittedAt: r.SubmittedAt,
			BodyLen:     len(r.Body),
		})
	}

	pr := PR{
		Number:      d.Number,
		Title:       d.Title,
		State:       d.State,
		Merged:      d.Merged,
		AuthorLogin: d.User.Login,
		CreatedAt:   d.CreatedAt,
		UpdatedAt:   d.UpdatedAt,
		Additions:   d.Additions,
		Deletions:   d.Deletions,
		Reviews:     reviews,
	}
	if d.MergedAt != nil {
		pr.MergedAt = *d.MergedAt
	}
	if d.ClosedAt != nil {
		pr.ClosedAt = *d.ClosedAt
	}
	if d.MergedBy != nil {
		pr.MergedByLogin = d.MergedBy.Login
	}
	return pr, nil
}

func cachePath(dir string, number int) string {
	return filepath.Join(dir, strconv.Itoa(number)+".json")
}

func loadCached(dir string, number int) (PR, bool) {
	data, err := os.ReadFile(cachePath(dir, number))
	if err != nil {
		return PR{}, false
	}
	var pr PR
	if err := json.Unmarshal(data, &pr); err != nil {
		return PR{}, false
	}
	return pr, true
}

func saveCached(dir string, pr PR) error {
	data, err := json.MarshalIndent(pr, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(cachePath(dir, pr.Number), data, 0644)
}

// loadAllCached is the rate-limit fallback — return everything we have
// on disk for the window rather than erroring the whole run.
func loadAllCached(dir string, since time.Time) ([]PR, error) {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return nil, err
	}
	var out []PR
	for _, e := range entries {
		if e.IsDir() {
			continue
		}
		num, err := strconv.Atoi(filepath.Base(e.Name()[:len(e.Name())-len(filepath.Ext(e.Name()))]))
		if err != nil {
			continue
		}
		pr, ok := loadCached(dir, num)
		if !ok {
			continue
		}
		if pr.Merged && !pr.UpdatedAt.Before(since) {
			out = append(out, pr)
		}
	}
	return out, nil
}

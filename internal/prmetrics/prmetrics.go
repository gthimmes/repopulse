// Package prmetrics turns raw GitHub PR data into the derived signals
// surfaced in the "PR Flow" card: cycle-time percentiles, time-to-
// first-review percentiles, reviewer concentration, rubber-stamp rate,
// self-merge rate, top samples.
//
// Kept separate from the github package so tests can feed PR fixtures
// directly without touching HTTP mocks. Uses only concrete types from
// internal/github + internal/types — no GitHub client dependency.
package prmetrics

import (
	"math"
	"sort"
	"time"

	"repopulse/internal/github"
	"repopulse/internal/types"
)

// Options controls Compute. Only the threshold knobs are exposed; the
// rest of the shape is inferred from the PR set.
type Options struct {
	// RubberStampMaxSeconds — approved in <N seconds with zero review
	// comments counts as a rubber-stamp. GitHub's own culture often
	// uses 30-60s; default 60.
	RubberStampMaxSeconds int
	// TopReviewers — how many reviewers to keep in the concentration panel.
	TopReviewers int
	// RubberStampSamples — how many rubber-stamped PRs to keep for drill-down.
	RubberStampSamples int
}

// Compute assembles a PRFlowSignal from the raw PR set. The signal is
// rendered directly; no scoring is applied here.
func Compute(prs []github.PR, ownerRepo string, windowDays int, opts Options) types.PRFlowSignal {
	if opts.RubberStampMaxSeconds <= 0 {
		opts.RubberStampMaxSeconds = 60
	}
	if opts.TopReviewers <= 0 {
		opts.TopReviewers = 5
	}
	if opts.RubberStampSamples <= 0 {
		opts.RubberStampSamples = 10
	}

	// Merged-only subset powers cycle/review stats. Open PRs would
	// skew the distribution (they're by definition still in flight).
	var merged []github.PR
	for _, pr := range prs {
		if pr.Merged {
			merged = append(merged, pr)
		}
	}

	cycles := make([]float64, 0, len(merged))
	ttfrs := make([]float64, 0, len(merged))
	reviewerCount := map[string]int{}
	var rubberStampPRs []github.PR
	selfMerges := 0

	for _, pr := range merged {
		// Cycle time: created → merged, in hours
		if !pr.MergedAt.IsZero() {
			cycles = append(cycles, pr.MergedAt.Sub(pr.CreatedAt).Hours())
		}

		// Time to first review: first review event (any kind except dismissed)
		firstReview := earliestReviewTime(pr.Reviews)
		if !firstReview.IsZero() {
			ttfrs = append(ttfrs, firstReview.Sub(pr.CreatedAt).Hours())
		}

		// Reviewer concentration: count distinct reviewers per PR once
		seen := map[string]bool{}
		for _, r := range pr.Reviews {
			if seen[r.Login] {
				continue
			}
			// Skip reviews by the author themselves — self-reviews aren't
			// peer review in the sense we care about.
			if r.Login == pr.AuthorLogin {
				continue
			}
			seen[r.Login] = true
			reviewerCount[r.Login]++
		}

		// Rubber-stamp: first APPROVED review with zero body length,
		// submitted within RubberStampMaxSeconds of PR open, and no
		// non-author review carried review comments > 0.
		if isRubberStamp(pr, opts.RubberStampMaxSeconds) {
			rubberStampPRs = append(rubberStampPRs, pr)
		}

		// Self-merge
		if pr.MergedByLogin != "" && pr.MergedByLogin == pr.AuthorLogin {
			selfMerges++
		}
	}

	// Reviewer entries, ranked
	revEntries := make([]types.ReviewerEntry, 0, len(reviewerCount))
	totalReviews := 0
	for _, n := range reviewerCount {
		totalReviews += n
	}
	for login, n := range reviewerCount {
		share := 0.0
		if totalReviews > 0 {
			share = float64(n) / float64(totalReviews) * 100
		}
		revEntries = append(revEntries, types.ReviewerEntry{
			Login:       login,
			ReviewCount: n,
			SharePct:    round1(share),
		})
	}
	sort.SliceStable(revEntries, func(i, j int) bool {
		return revEntries[i].ReviewCount > revEntries[j].ReviewCount
	})
	if len(revEntries) > opts.TopReviewers {
		revEntries = revEntries[:opts.TopReviewers]
	}

	// Rubber-stamp samples
	sort.SliceStable(rubberStampPRs, func(i, j int) bool {
		return rubberStampPRs[i].MergedAt.After(rubberStampPRs[j].MergedAt)
	})
	if len(rubberStampPRs) > opts.RubberStampSamples {
		rubberStampPRs = rubberStampPRs[:opts.RubberStampSamples]
	}
	samples := make([]types.PRSample, 0, len(rubberStampPRs))
	for _, pr := range rubberStampPRs {
		cycle := 0.0
		if !pr.MergedAt.IsZero() {
			cycle = round1(pr.MergedAt.Sub(pr.CreatedAt).Hours())
		}
		samples = append(samples, types.PRSample{
			Number:     pr.Number,
			Title:      pr.Title,
			Author:     pr.AuthorLogin,
			MergedBy:   pr.MergedByLogin,
			CycleHours: cycle,
		})
	}

	rubberRate := 0.0
	selfRate := 0.0
	if len(merged) > 0 {
		rubberRate = float64(len(rubberStampPRs)) / float64(len(merged)) * 100
		// Careful: rubberStampPRs was just capped above. Recompute from totalRubberStamp count.
		// (we lost the raw count when we capped — fix below)
		selfRate = float64(selfMerges) / float64(len(merged)) * 100
	}
	// Recompute the rubber-stamp rate correctly against the uncapped count.
	totalRubber := 0
	for _, pr := range merged {
		if isRubberStamp(pr, opts.RubberStampMaxSeconds) {
			totalRubber++
		}
	}
	if len(merged) > 0 {
		rubberRate = float64(totalRubber) / float64(len(merged)) * 100
	}

	return types.PRFlowSignal{
		Type:            "prFlow",
		OwnerRepo:       ownerRepo,
		WindowDays:      windowDays,
		TotalPRs:        len(prs),
		MergedPRs:       len(merged),
		CycleHours:      percentiles(cycles),
		TTFRHours:       percentiles(ttfrs),
		Reviewers:       revEntries,
		RubberStamps:    samples,
		RubberStampRate: round1(rubberRate),
		SelfMergeRate:   round1(selfRate),
	}
}

// earliestReviewTime returns the timestamp of the first non-dismissed
// review on a PR. Dismissed reviews are skipped because they don't
// represent a real feedback loop.
func earliestReviewTime(reviews []github.Review) time.Time {
	var earliest time.Time
	for _, r := range reviews {
		if r.State == "DISMISSED" {
			continue
		}
		if r.SubmittedAt.IsZero() {
			continue
		}
		if earliest.IsZero() || r.SubmittedAt.Before(earliest) {
			earliest = r.SubmittedAt
		}
	}
	return earliest
}

// isRubberStamp: merged PR whose first approval arrived within
// maxSeconds of PR open and which carried zero substantive review
// body (BodyLen == 0) across all reviews. Intentionally strict.
func isRubberStamp(pr github.PR, maxSeconds int) bool {
	var approval time.Time
	commented := false
	for _, r := range pr.Reviews {
		if r.Login == pr.AuthorLogin {
			continue
		}
		if r.BodyLen > 0 {
			commented = true
		}
		if r.State == "APPROVED" {
			if approval.IsZero() || r.SubmittedAt.Before(approval) {
				approval = r.SubmittedAt
			}
		}
	}
	if approval.IsZero() || commented {
		return false
	}
	return approval.Sub(pr.CreatedAt).Seconds() <= float64(maxSeconds)
}

// percentiles returns P50/P75/P95 from a sample slice (in the same
// units it was given). Empty input → zero struct.
func percentiles(samples []float64) types.Percentiles {
	if len(samples) == 0 {
		return types.Percentiles{}
	}
	sorted := append([]float64(nil), samples...)
	sort.Float64s(sorted)
	pct := func(p float64) float64 {
		if len(sorted) == 0 {
			return 0
		}
		idx := int(math.Round(p * float64(len(sorted)-1)))
		return round1(sorted[idx])
	}
	return types.Percentiles{
		P50: pct(0.50),
		P75: pct(0.75),
		P95: pct(0.95),
	}
}

func round1(x float64) float64 {
	return math.Round(x*10) / 10
}

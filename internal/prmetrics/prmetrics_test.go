package prmetrics

import (
	"testing"
	"time"

	"repopulse/internal/github"
)

func pr(num int, author, mergedBy string, created, merged time.Time, reviews []github.Review) github.PR {
	return github.PR{
		Number:        num,
		Title:         "PR " + author,
		State:         "closed",
		Merged:        !merged.IsZero(),
		AuthorLogin:   author,
		CreatedAt:     created,
		MergedAt:      merged,
		MergedByLogin: mergedBy,
		Reviews:       reviews,
	}
}

func TestPercentiles_BasicDistribution(t *testing.T) {
	p := percentiles([]float64{1, 2, 3, 4, 5, 6, 7, 8, 9, 10})
	if p.P50 < 4 || p.P50 > 7 {
		t.Errorf("P50 should be mid-range, got %v", p.P50)
	}
	if p.P95 < 9 {
		t.Errorf("P95 should be near max, got %v", p.P95)
	}
}

func TestPercentiles_Empty(t *testing.T) {
	p := percentiles(nil)
	if p.P50 != 0 || p.P75 != 0 || p.P95 != 0 {
		t.Errorf("empty should be zero, got %+v", p)
	}
}

func TestCompute_CycleTime(t *testing.T) {
	now := time.Date(2026, 4, 1, 12, 0, 0, 0, time.UTC)
	prs := []github.PR{
		pr(1, "alice", "alice", now, now.Add(2*time.Hour), nil),
		pr(2, "bob", "alice", now, now.Add(24*time.Hour), nil),
		pr(3, "carol", "alice", now, now.Add(10*time.Hour), nil),
	}
	sig := Compute(prs, "test/repo", 30, Options{})
	if sig.CycleHours.P50 < 10-1 || sig.CycleHours.P50 > 10+1 {
		t.Errorf("P50 cycle should be ~10h (middle of 2/10/24), got %v", sig.CycleHours.P50)
	}
	if sig.MergedPRs != 3 {
		t.Errorf("want 3 merged, got %d", sig.MergedPRs)
	}
}

func TestCompute_RubberStampDetection(t *testing.T) {
	now := time.Date(2026, 4, 1, 12, 0, 0, 0, time.UTC)
	quickApproval := []github.Review{
		{Login: "reviewer1", State: "APPROVED", SubmittedAt: now.Add(30 * time.Second), BodyLen: 0},
	}
	slowApproval := []github.Review{
		{Login: "reviewer1", State: "APPROVED", SubmittedAt: now.Add(3 * time.Hour), BodyLen: 50},
	}
	// Rubber-stamp: approved in 30s with 0 body
	rs := pr(1, "alice", "alice", now, now.Add(1*time.Hour), quickApproval)
	// Not rubber-stamp: slow review with body
	good := pr(2, "bob", "alice", now, now.Add(10*time.Hour), slowApproval)
	// Not rubber-stamp: approved quick but someone else commented substantively
	mixed := pr(3, "carol", "alice", now, now.Add(1*time.Hour), []github.Review{
		{Login: "r1", State: "APPROVED", SubmittedAt: now.Add(30 * time.Second), BodyLen: 0},
		{Login: "r2", State: "COMMENTED", SubmittedAt: now.Add(10 * time.Minute), BodyLen: 200},
	})

	sig := Compute([]github.PR{rs, good, mixed}, "test/repo", 30, Options{})
	if sig.RubberStampRate < 30 || sig.RubberStampRate > 35 {
		t.Errorf("want ~33%% rubber-stamp, got %v", sig.RubberStampRate)
	}
	if len(sig.RubberStamps) != 1 {
		t.Errorf("want 1 rubber-stamp sample, got %d", len(sig.RubberStamps))
	}
}

func TestCompute_SelfMergeRate(t *testing.T) {
	now := time.Date(2026, 4, 1, 12, 0, 0, 0, time.UTC)
	prs := []github.PR{
		pr(1, "alice", "alice", now, now.Add(1*time.Hour), nil), // self-merge
		pr(2, "alice", "bob", now, now.Add(1*time.Hour), nil),
		pr(3, "carol", "carol", now, now.Add(1*time.Hour), nil), // self-merge
	}
	sig := Compute(prs, "test/repo", 30, Options{})
	if sig.SelfMergeRate < 66 || sig.SelfMergeRate > 67 {
		t.Errorf("want ~66.7%% self-merge, got %v", sig.SelfMergeRate)
	}
}

func TestCompute_ReviewerConcentration(t *testing.T) {
	now := time.Date(2026, 4, 1, 12, 0, 0, 0, time.UTC)
	r1 := github.Review{Login: "senior", State: "APPROVED", SubmittedAt: now.Add(1 * time.Hour), BodyLen: 100}
	r2 := github.Review{Login: "other", State: "APPROVED", SubmittedAt: now.Add(1 * time.Hour), BodyLen: 50}
	prs := []github.PR{
		pr(1, "a", "senior", now, now.Add(1*time.Hour), []github.Review{r1}),
		pr(2, "b", "senior", now, now.Add(1*time.Hour), []github.Review{r1}),
		pr(3, "c", "senior", now, now.Add(1*time.Hour), []github.Review{r1}),
		pr(4, "d", "other", now, now.Add(1*time.Hour), []github.Review{r2}),
	}
	sig := Compute(prs, "test/repo", 30, Options{})
	if len(sig.Reviewers) < 2 {
		t.Fatalf("want 2+ reviewers, got %d", len(sig.Reviewers))
	}
	if sig.Reviewers[0].Login != "senior" {
		t.Errorf("senior should be top reviewer, got %s", sig.Reviewers[0].Login)
	}
	if sig.Reviewers[0].ReviewCount != 3 {
		t.Errorf("senior should have 3 reviews, got %d", sig.Reviewers[0].ReviewCount)
	}
}

func TestCompute_SkipsSelfReviewsFromConcentration(t *testing.T) {
	now := time.Date(2026, 4, 1, 12, 0, 0, 0, time.UTC)
	// Alice reviews her own PR — should NOT count toward concentration
	prs := []github.PR{
		pr(1, "alice", "alice", now, now.Add(1*time.Hour), []github.Review{
			{Login: "alice", State: "APPROVED", SubmittedAt: now.Add(10 * time.Minute), BodyLen: 50},
		}),
	}
	sig := Compute(prs, "test/repo", 30, Options{})
	if len(sig.Reviewers) != 0 {
		t.Errorf("self-review should be excluded, got %+v", sig.Reviewers)
	}
}

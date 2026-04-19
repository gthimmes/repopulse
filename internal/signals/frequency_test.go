package signals

import (
	"testing"

	"repopulse/internal/fixtures"
)

func TestComputeFrequency_Calm(t *testing.T) {
	r := ComputeFrequency(fixtures.CalmFixture(), 90)
	if r.Type != "commitFrequency" {
		t.Errorf("want type commitFrequency, got %s", r.Type)
	}
	if r.Score >= 40 {
		t.Errorf("calm score should be < 40, got %d", r.Score)
	}
}

func TestComputeFrequency_Chaotic(t *testing.T) {
	r := ComputeFrequency(fixtures.ChaoticFixture(), 90)
	if r.Score <= 50 {
		t.Errorf("chaotic score should be > 50, got %d", r.Score)
	}
}

func TestComputeFrequency_BucketLength(t *testing.T) {
	r := ComputeFrequency(fixtures.CalmFixture(), 90)
	if len(r.DailyBuckets) != 90 {
		t.Errorf("want 90 daily buckets, got %d", len(r.DailyBuckets))
	}
}

func TestComputeFrequency_EmptyCommits(t *testing.T) {
	r := ComputeFrequency(nil, 90)
	if r.Score < 0 || r.Score > 100 {
		t.Errorf("score out of range: %d", r.Score)
	}
}

func TestComputeFrequency_DetectsLongGaps(t *testing.T) {
	r := ComputeFrequency(fixtures.ChaoticFixture(), 90)
	if r.LongestGapDays <= 15 {
		t.Errorf("expected gap > 15 days, got %d", r.LongestGapDays)
	}
}

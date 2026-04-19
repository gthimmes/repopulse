package render

import (
	"strings"
	"testing"

	"repopulse/internal/compare"
	"repopulse/internal/types"
)

func intPtr(i int) *int { return &i }

func snap(score, freq, churn, bug, authors int, cov *int, when string) compare.ReportSnapshot {
	return compare.ReportSnapshot{
		GeneratedAt: when,
		RepoName:    "t",
		WindowDays:  30,
		MoodResult: types.MoodResult{
			CompositeScore: score,
			Breakdown: types.MoodBreakdown{
				CommitFrequency: freq,
				FileChurn:       churn,
				BugRatio:        bug,
				Authors:         authors,
				Coverage:        cov,
			},
		},
	}
}

func TestTrendSectionEmptyStateWhenSingleSnapshot(t *testing.T) {
	s := []compare.ReportSnapshot{
		snap(50, 40, 50, 60, 30, nil, "2026-04-18T10:00:00Z"),
	}
	got := TrendSection(s)
	if !strings.Contains(got, "Only one snapshot") {
		t.Fatalf("expected empty-state copy in TrendSection, got:\n%s", got)
	}
	if strings.Contains(got, "canvas id=\"trendChart\"") {
		t.Fatalf("should not render canvas when only one snapshot exists")
	}
}

func TestTrendSectionRendersCanvasWhenMultipleSnapshots(t *testing.T) {
	s := []compare.ReportSnapshot{
		snap(40, 30, 40, 50, 20, nil, "2026-04-18T10:00:00Z"),
		snap(55, 45, 55, 65, 35, nil, "2026-04-19T10:00:00Z"),
	}
	got := TrendSection(s)
	if !strings.Contains(got, `canvas id="trendChart"`) {
		t.Fatalf("expected trend canvas, got:\n%s", got)
	}
	if !strings.Contains(got, "2 snapshots") {
		t.Fatalf("expected footer count, got:\n%s", got)
	}
}

func TestTrendChartEmitsAllSeriesAndLabels(t *testing.T) {
	c80 := intPtr(80)
	s := []compare.ReportSnapshot{
		snap(40, 30, 40, 50, 20, nil, "2026-04-18T10:00:00Z"),
		snap(55, 45, 55, 65, 35, c80, "2026-04-19T10:00:00Z"),
	}
	got := TrendChart(s)
	mustContain := []string{
		"'Composite'",
		"'Commit Frequency'",
		"'File Churn'",
		"'Bug Ratio'",
		"'Authors'",
		"'Coverage'",
		"[40,55]",    // composite series
		"[null,80]",  // coverage series with leading null
	}
	for _, want := range mustContain {
		if !strings.Contains(got, want) {
			t.Fatalf("trend chart missing %q in:\n%s", want, got)
		}
	}
}

func TestTrendLabelFallsBackOnUnparseable(t *testing.T) {
	if got := trendLabel("not-a-date"); got != "not-a-date" {
		t.Fatalf("expected raw fallback, got %q", got)
	}
}

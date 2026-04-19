package baseline

import (
	"testing"
	"time"

	"repopulse/internal/config"
	"repopulse/internal/signals"
	"repopulse/internal/types"
)

var defaultBugOpts = signals.BugOptions{
	ChaosKeywords:   config.DefaultBugKeywords.Chaos,
	NormalKeywords:  config.DefaultBugKeywords.Normal,
	RoutineKeywords: config.DefaultBugKeywords.Routine,
}

func mkCommit(email, name, message string, when time.Time) types.CommitRecord {
	return types.CommitRecord{
		Hash:        "h" + when.Format("20060102150405"),
		Date:        when,
		AuthorDate:  when,
		AuthorEmail: email,
		AuthorName:  name,
		Message:     message,
	}
}

// helper: make N commits for a single author at a given local hour, weekday, on consecutive days
func mkAuthorCommits(email, name string, n int, message string, hour int, weekday time.Weekday, weekStart time.Time) []types.CommitRecord {
	out := make([]types.CommitRecord, 0, n)
	day := weekStart
	for i := 0; i < n; i++ {
		// skip forward to next occurrence of weekday
		for day.Weekday() != weekday {
			day = day.AddDate(0, 0, 1)
		}
		t := time.Date(day.Year(), day.Month(), day.Day(), hour, 0, 0, 0, time.UTC)
		out = append(out, mkCommit(email, name, message, t))
		day = day.AddDate(0, 0, 1)
	}
	return out
}

func findDrift(s types.AuthorDriftSignal, email string) *types.AuthorDrift {
	for i := range s.Authors {
		if s.Authors[i].Email == email {
			return &s.Authors[i]
		}
	}
	return nil
}

func hasFlag(d *types.AuthorDrift, kind string) bool {
	if d == nil {
		return false
	}
	for _, f := range d.Flags {
		if f.Kind == kind {
			return true
		}
	}
	return false
}

func TestComputeAuthorDrift_NoOverlapAuthorsExcluded(t *testing.T) {
	// Alice in current only, Bob in baseline only — neither should drift-flag
	curr := []types.CommitRecord{mkCommit("a@x", "Alice", "feat: x", time.Now())}
	base := []types.CommitRecord{mkCommit("b@x", "Bob", "feat: y", time.Now().AddDate(0, 0, -100))}
	got := ComputeAuthorDrift(curr, base, Options{
		CurrentDays:  30,
		BaselineDays: 180,
		BugOptions:   defaultBugOpts,
	})
	if len(got.Authors) != 0 {
		t.Errorf("want 0 drifted authors (no overlap), got %d", len(got.Authors))
	}
}

func TestComputeAuthorDrift_BelowGatesIsSilent(t *testing.T) {
	now := time.Now()
	// Tiny volume in both windows — should be excluded by min-commit gates
	curr := []types.CommitRecord{
		mkCommit("a@x", "Alice", "feat: a", now.AddDate(0, 0, -1)),
	}
	base := []types.CommitRecord{
		mkCommit("a@x", "Alice", "feat: b", now.AddDate(0, 0, -50)),
	}
	got := ComputeAuthorDrift(curr, base, Options{
		CurrentDays:  30,
		BaselineDays: 180,
		BugOptions:   defaultBugOpts,
	})
	if len(got.Authors) != 0 {
		t.Errorf("want 0 (below gates), got %d", len(got.Authors))
	}
}

func TestComputeAuthorDrift_CadenceUpFiresWithFloor(t *testing.T) {
	// Baseline: 1 commit/wk over 30wks (sparse). Current: 8 commits/wk.
	// Relative delta is +700% but absolute current 8/wk meets the 5/wk floor.
	now := time.Date(2026, 4, 15, 10, 0, 0, 0, time.UTC)

	var base []types.CommitRecord
	for i := 0; i < 30; i++ {
		base = append(base, mkCommit("a@x", "Alice", "feat: x", now.AddDate(0, 0, -7*(i+5))))
	}
	var curr []types.CommitRecord
	for i := 0; i < 32; i++ {
		// 32 commits over 4 weeks = 8/wk
		curr = append(curr, mkCommit("a@x", "Alice", "feat: x", now.AddDate(0, 0, -i/2)))
	}

	got := ComputeAuthorDrift(curr, base, Options{
		CurrentDays:  28,
		BaselineDays: 28 * 8,
		BugOptions:   defaultBugOpts,
	})
	d := findDrift(got, "a@x")
	if d == nil {
		t.Fatalf("expected drift entry for a@x, got nil")
	}
	if !hasFlag(d, "cadence-up") {
		t.Errorf("expected cadence-up flag, got %v", d.Flags)
	}
}

func TestComputeAuthorDrift_WeekendNightUpFires(t *testing.T) {
	now := time.Date(2026, 4, 15, 0, 0, 0, 0, time.UTC) // Wednesday

	// Baseline: all weekday-9am commits (0% off-hours)
	base := mkAuthorCommits("a@x", "Alice", 24, "feat: x", 9, time.Wednesday, now.AddDate(0, 0, -180))
	// Current: all Saturday-8pm commits (100% weekend/night)
	curr := mkAuthorCommits("a@x", "Alice", 8, "feat: x", 20, time.Saturday, now.AddDate(0, 0, -28))

	got := ComputeAuthorDrift(curr, base, Options{
		CurrentDays:  28,
		BaselineDays: 168,
		BugOptions:   defaultBugOpts,
	})
	d := findDrift(got, "a@x")
	if d == nil {
		t.Fatalf("expected drift entry for a@x, got nil")
	}
	if !hasFlag(d, "weekend-night-up") {
		t.Errorf("expected weekend-night-up flag, got %v", d.Flags)
	}
	// 100% current vs 0% baseline → should be alert (>= 50%)
	for _, f := range d.Flags {
		if f.Kind == "weekend-night-up" && f.Severity != "alert" {
			t.Errorf("expected alert severity at 100%%, got %s", f.Severity)
		}
	}
}

func TestComputeAuthorDrift_FixRatioUpFires(t *testing.T) {
	now := time.Date(2026, 4, 15, 10, 0, 0, 0, time.UTC) // Wednesday at 10am — non-off-hours
	// Baseline: all features
	base := mkAuthorCommits("a@x", "Alice", 24, "feat: ship X", 10, time.Wednesday, now.AddDate(0, 0, -180))
	// Current: mostly fixes
	var curr []types.CommitRecord
	weekStart := now.AddDate(0, 0, -28)
	curr = append(curr, mkAuthorCommits("a@x", "Alice", 8, "fix: bug Y", 10, time.Wednesday, weekStart)...)
	curr = append(curr, mkAuthorCommits("a@x", "Alice", 2, "feat: small Z", 10, time.Wednesday, weekStart)...)

	got := ComputeAuthorDrift(curr, base, Options{
		CurrentDays:  28,
		BaselineDays: 168,
		BugOptions:   defaultBugOpts,
	})
	d := findDrift(got, "a@x")
	if d == nil {
		t.Fatalf("expected drift entry for a@x, got nil")
	}
	if !hasFlag(d, "fix-ratio-up") {
		t.Errorf("expected fix-ratio-up flag, got %v", d.Flags)
	}
}

func TestComputeAuthorDrift_StableAuthorIsSilent(t *testing.T) {
	// Same cadence, same off-hours, same fix ratio in both windows
	now := time.Date(2026, 4, 15, 10, 0, 0, 0, time.UTC)
	base := mkAuthorCommits("a@x", "Alice", 24, "feat: x", 10, time.Wednesday, now.AddDate(0, 0, -180))
	curr := mkAuthorCommits("a@x", "Alice", 4, "feat: x", 10, time.Wednesday, now.AddDate(0, 0, -28))

	got := ComputeAuthorDrift(curr, base, Options{
		CurrentDays:  28,
		BaselineDays: 168,
		BugOptions:   defaultBugOpts,
	})
	if d := findDrift(got, "a@x"); d != nil {
		t.Errorf("expected no drift entry for stable author, got flags %v", d.Flags)
	}
}

func TestComputeAuthorDrift_AlertOutranksWatchInSort(t *testing.T) {
	now := time.Date(2026, 4, 15, 0, 0, 0, 0, time.UTC)

	// Author A: weekend-night-up alert (100% off-hours)
	baseA := mkAuthorCommits("a@x", "Alice", 24, "feat: x", 10, time.Wednesday, now.AddDate(0, 0, -180))
	currA := mkAuthorCommits("a@x", "Alice", 8, "feat: x", 20, time.Saturday, now.AddDate(0, 0, -28))

	// Author B: weekend-night-up watch (only 30% off-hours)
	baseB := mkAuthorCommits("b@x", "Bob", 30, "feat: y", 10, time.Wednesday, now.AddDate(0, 0, -180))
	currB := []types.CommitRecord{}
	currB = append(currB, mkAuthorCommits("b@x", "Bob", 7, "feat: y", 10, time.Wednesday, now.AddDate(0, 0, -28))...)
	currB = append(currB, mkAuthorCommits("b@x", "Bob", 3, "feat: y", 20, time.Saturday, now.AddDate(0, 0, -25))...)

	got := ComputeAuthorDrift(append(currA, currB...), append(baseA, baseB...), Options{
		CurrentDays:  28,
		BaselineDays: 168,
		BugOptions:   defaultBugOpts,
	})
	if len(got.Authors) < 2 {
		t.Fatalf("expected both authors to drift, got %d", len(got.Authors))
	}
	if got.Authors[0].Email != "a@x" {
		t.Errorf("expected alert author first, got %s", got.Authors[0].Email)
	}
}

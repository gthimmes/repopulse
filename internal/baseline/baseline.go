// Package baseline computes per-contributor drift between a current window
// and a longer historical baseline. Each contributor is compared only
// against *their own* prior behavior — never against team peers — and
// drift is surfaced as "Worth a 1:1" cards when deltas are both
// statistically meaningful and above absolute-noise floors.
//
// Plank 1 of the lens-not-scorecard direction. See ROADMAP.md.
package baseline

import (
	"fmt"
	"math"
	"sort"
	"time"

	"repopulse/internal/signals"
	"repopulse/internal/types"
)

// Options carries everything ComputeAuthorDrift needs to compare a
// current-window CommitRecord set against a baseline set.
type Options struct {
	CurrentDays    int
	BaselineDays   int
	BugOptions     signals.BugOptions
	NightStartHour int // optional override; 0 → default 20
	NightEndHour   int // optional override; 0 → default 7
	// Author commit-count gates. An author is only evaluated when they
	// have at least MinCommitsCurrent commits in the current window AND
	// MinCommitsBaseline in the baseline window.
	MinCommitsCurrent  int
	MinCommitsBaseline int
}

// Defaults applied when the corresponding Options field is zero.
const (
	defaultNightStart         = 20
	defaultNightEnd           = 7
	defaultMinCommitsCurrent  = 3
	defaultMinCommitsBaseline = 5
)

// ComputeAuthorDrift returns per-author drift between currentCommits and
// baselineCommits. Authors are only included when they have at least one
// flag; authors with insufficient signal are silently dropped.
func ComputeAuthorDrift(currentCommits, baselineCommits []types.CommitRecord, opts Options) types.AuthorDriftSignal {
	nightStart := opts.NightStartHour
	if nightStart == 0 {
		nightStart = defaultNightStart
	}
	nightEnd := opts.NightEndHour
	if nightEnd == 0 {
		nightEnd = defaultNightEnd
	}
	minCurrent := opts.MinCommitsCurrent
	if minCurrent == 0 {
		minCurrent = defaultMinCommitsCurrent
	}
	minBaseline := opts.MinCommitsBaseline
	if minBaseline == 0 {
		minBaseline = defaultMinCommitsBaseline
	}

	currentStats := aggregatePerAuthor(currentCommits, opts.BugOptions, nightStart, nightEnd)
	baselineStats := aggregatePerAuthor(baselineCommits, opts.BugOptions, nightStart, nightEnd)

	currentWeeks := math.Max(1, float64(opts.CurrentDays)/7.0)
	baselineWeeks := math.Max(1, float64(opts.BaselineDays)/7.0)

	out := make([]types.AuthorDrift, 0, len(currentStats))
	for email, cur := range currentStats {
		base, ok := baselineStats[email]
		if !ok {
			continue
		}
		if cur.commits < minCurrent || base.commits < minBaseline {
			continue
		}

		curRate := float64(cur.commits) / currentWeeks
		baseRate := float64(base.commits) / baselineWeeks

		curWN := pct(cur.weekendNight, cur.commits)
		baseWN := pct(base.weekendNight, base.commits)

		curFix := pct(cur.fixCommits, cur.commits)
		baseFix := pct(base.fixCommits, base.commits)

		drift := types.AuthorDrift{
			Name:                   cur.name,
			Email:                  email,
			CommitsCurrent:         cur.commits,
			CommitsPerWeekCurrent:  round1(curRate),
			CommitsPerWeekBaseline: round1(baseRate),
			CommitsDeltaPct:        round1(deltaPct(curRate, baseRate)),
			WeekendNightCurrent:    round1(curWN),
			WeekendNightBaseline:   round1(baseWN),
			WeekendNightDeltaPP:    round1(curWN - baseWN),
			FixRatioCurrent:        round1(curFix),
			FixRatioBaseline:       round1(baseFix),
			FixRatioDeltaPP:        round1(curFix - baseFix),
		}
		drift.Flags = computeFlags(drift)

		if len(drift.Flags) > 0 {
			out = append(out, drift)
		}
	}

	// Sort: alert > watch > info, then by current commit count desc
	severityRank := map[string]int{"alert": 3, "watch": 2, "info": 1}
	maxSeverity := func(d types.AuthorDrift) int {
		m := 0
		for _, f := range d.Flags {
			if r := severityRank[f.Severity]; r > m {
				m = r
			}
		}
		return m
	}
	sort.SliceStable(out, func(i, j int) bool {
		si, sj := maxSeverity(out[i]), maxSeverity(out[j])
		if si != sj {
			return si > sj
		}
		return out[i].CommitsCurrent > out[j].CommitsCurrent
	})

	return types.AuthorDriftSignal{
		Type:         "authorDrift",
		CurrentDays:  opts.CurrentDays,
		BaselineDays: opts.BaselineDays,
		Authors:      out,
	}
}

type authorAgg struct {
	name         string
	commits      int
	fixCommits   int
	weekendNight int
}

func aggregatePerAuthor(commits []types.CommitRecord, bugOpts signals.BugOptions, nightStart, nightEnd int) map[string]*authorAgg {
	out := map[string]*authorAgg{}
	for _, c := range commits {
		a, ok := out[c.AuthorEmail]
		if !ok {
			a = &authorAgg{name: c.AuthorName}
			out[c.AuthorEmail] = a
		}
		a.commits++
		tier := signals.ClassifyCommit(c.Message, c.IsRevert, bugOpts)
		if tier != signals.TierNone {
			a.fixCommits++
		}
		if isWeekendOrNight(c.AuthorDate, nightStart, nightEnd) {
			a.weekendNight++
		}
	}
	return out
}

// computeFlags applies the surfacing rules. Each rule has a relative-
// magnitude condition AND an absolute-floor condition so a 100% delta on
// a 1-commit baseline doesn't fire. Phrasing is deliberately about
// observation (something to discuss) rather than judgement.
func computeFlags(d types.AuthorDrift) []types.DriftFlag {
	var flags []types.DriftFlag

	// Cadence
	if d.CommitsDeltaPct >= 50 && d.CommitsPerWeekCurrent >= 5 {
		flags = append(flags, types.DriftFlag{
			Kind:     "cadence-up",
			Severity: severityForCadence(d.CommitsDeltaPct),
			Text: fmt.Sprintf("commit cadence up %.0f%% (%.1f/wk now vs %.1f/wk baseline) — crunch?",
				d.CommitsDeltaPct, d.CommitsPerWeekCurrent, d.CommitsPerWeekBaseline),
		})
	} else if d.CommitsDeltaPct <= -50 && d.CommitsPerWeekBaseline >= 3 {
		flags = append(flags, types.DriftFlag{
			Kind:     "cadence-down",
			Severity: severityForCadence(-d.CommitsDeltaPct),
			Text: fmt.Sprintf("commit cadence down %.0f%% (%.1f/wk now vs %.1f/wk baseline) — what changed?",
				-d.CommitsDeltaPct, d.CommitsPerWeekCurrent, d.CommitsPerWeekBaseline),
		})
	}

	// Weekend / night creep
	if d.WeekendNightDeltaPP >= 15 && d.WeekendNightCurrent >= 25 {
		sev := "watch"
		if d.WeekendNightCurrent >= 50 {
			sev = "alert"
		}
		flags = append(flags, types.DriftFlag{
			Kind:     "weekend-night-up",
			Severity: sev,
			Text: fmt.Sprintf("weekend/night commits up %.0fpp (%.0f%% now vs %.0f%% baseline) — load creeping off-hours",
				d.WeekendNightDeltaPP, d.WeekendNightCurrent, d.WeekendNightBaseline),
		})
	}

	// Fix-vs-feature shift
	if d.FixRatioDeltaPP >= 20 && d.FixRatioCurrent >= 30 {
		sev := "watch"
		if d.FixRatioCurrent >= 60 {
			sev = "alert"
		}
		flags = append(flags, types.DriftFlag{
			Kind:     "fix-ratio-up",
			Severity: sev,
			Text: fmt.Sprintf("fix-vs-feature mix up %.0fpp (%.0f%% bug-tier now vs %.0f%% baseline) — stuck firefighting?",
				d.FixRatioDeltaPP, d.FixRatioCurrent, d.FixRatioBaseline),
		})
	}

	return flags
}

func severityForCadence(absDeltaPct float64) string {
	switch {
	case absDeltaPct >= 150:
		return "alert"
	case absDeltaPct >= 75:
		return "watch"
	default:
		return "info"
	}
}

// isWeekendOrNight mirrors signals.isWeekendOrNight which is unexported.
// Duplicated here rather than expanding the signals package surface.
func isWeekendOrNight(d time.Time, nightStart, nightEnd int) bool {
	utc := d.UTC()
	day := int(utc.Weekday())
	if day == 0 || day == 6 {
		return true
	}
	h := utc.Hour()
	if h >= nightStart {
		return true
	}
	if h < nightEnd {
		return true
	}
	return false
}

func pct(part, whole int) float64 {
	if whole <= 0 {
		return 0
	}
	return float64(part) / float64(whole) * 100
}

func deltaPct(current, baseline float64) float64 {
	if baseline <= 0 {
		if current <= 0 {
			return 0
		}
		return 100 // infinite-ish; cap so the UI doesn't show "+Inf%"
	}
	return (current - baseline) / baseline * 100
}

func round1(x float64) float64 {
	return math.Round(x*10) / 10
}

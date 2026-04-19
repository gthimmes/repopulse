// Package narrative generates human-readable finding bullets and the
// rolling-timeline sparkline data.
package narrative

import (
	"fmt"
	"math"
	"sort"
	"strings"
	"time"

	"repopulse/internal/signals"
	"repopulse/internal/types"
)

// Generate produces 3-8 findings ordered alert > warn > info > good.
func Generate(data types.MoodResult, meta types.RepoMeta) []types.NarrativeBullet {
	bullets := []types.NarrativeBullet{}

	// Overall mood
	var moodKind string
	switch data.Mood {
	case types.MoodCalm:
		moodKind = "good"
	case types.MoodAnxious:
		moodKind = "warn"
	default:
		moodKind = "alert"
	}
	bullets = append(bullets, types.NarrativeBullet{
		Kind: moodKind,
		Text: fmt.Sprintf("Overall mood is %s (%d/100) across %d commits in the last %d days.",
			data.Mood, data.CompositeScore, meta.AnalyzedCommits, meta.WindowDays),
	})

	// Top 3 hot modules
	modules := data.Signals.Modules.Modules
	hotModules := []types.ModuleEntry{}
	for _, m := range modules {
		if m.Mood != types.MoodCalm {
			hotModules = append(hotModules, m)
		}
		if len(hotModules) == 3 {
			break
		}
	}
	if len(hotModules) > 0 {
		parts := make([]string, len(hotModules))
		for i, m := range hotModules {
			parts[i] = fmt.Sprintf("%s (%d/100)", m.Name, m.Score)
		}
		k := "warn"
		if hotModules[0].Mood == types.MoodChaotic {
			k = "alert"
		}
		bullets = append(bullets, types.NarrativeBullet{Kind: k, Text: "Hottest modules: " + strings.Join(parts, ", ") + "."})
	} else if len(modules) > 0 {
		bullets = append(bullets, types.NarrativeBullet{
			Kind: "good",
			Text: "No module is flagged — churn and bug activity are spread evenly.",
		})
	}

	// Top hotspot
	if len(data.Signals.Hotspots.Hotspots) > 0 {
		top := data.Signals.Hotspots.Hotspots[0]
		k := "warn"
		if top.ChaosTouches > 0 {
			k = "alert"
		}
		chaosNote := ""
		if top.ChaosTouches > 0 {
			chaosNote = fmt.Sprintf(" (%d chaos-tier)", top.ChaosTouches)
		}
		bullets = append(bullets, types.NarrativeBullet{
			Kind: k,
			Text: fmt.Sprintf("Top hotspot: %s — %d commits, %s bug-related%s.",
				top.Path, top.TotalCommits, trimTrailingZero(top.BugTouches), chaosNote),
		})
	}

	// Reverts
	if data.Signals.BugRatio.RevertedWithin7d > 0 {
		n := data.Signals.BugRatio.RevertedWithin7d
		noun := "commit was"
		if n > 1 {
			noun = "commits were"
		}
		bullets = append(bullets, types.NarrativeBullet{
			Kind: "alert",
			Text: fmt.Sprintf("%d %s reverted within a week of landing — a strong chaos signal.", n, noun),
		})
	}

	// Chaos-tier
	if data.Signals.BugRatio.ChaosCommitCount > 0 && data.Signals.BugRatio.TotalCommits > 0 {
		pct := int(math.Round(float64(data.Signals.BugRatio.ChaosCommitCount) / float64(data.Signals.BugRatio.TotalCommits) * 100))
		k := "warn"
		if pct >= 5 {
			k = "alert"
		}
		bullets = append(bullets, types.NarrativeBullet{
			Kind: k,
			Text: fmt.Sprintf("%d chaos-tier commits (revert/hotfix/broken) — %d%% of all commits.",
				data.Signals.BugRatio.ChaosCommitCount, pct),
		})
	}

	// Weekend/night
	if data.Signals.Authors.WeekendNightPct >= 15 {
		k := "warn"
		if data.Signals.Authors.WeekendNightPct >= 25 {
			k = "alert"
		}
		bullets = append(bullets, types.NarrativeBullet{
			Kind: k,
			Text: fmt.Sprintf("%s%% of commits land on weekends or outside 7am–8pm — possible burnout signal.",
				trimTrailingZero(data.Signals.Authors.WeekendNightPct)),
		})
	}

	// Bus factor
	if data.Signals.Authors.BusFactorTop1Pct >= 40 {
		k := "warn"
		if data.Signals.Authors.BusFactorTop1Pct >= 60 {
			k = "alert"
		}
		bullets = append(bullets, types.NarrativeBullet{
			Kind: k,
			Text: fmt.Sprintf("Single author produced %s%% of commits — bus-factor risk.",
				trimTrailingZero(data.Signals.Authors.BusFactorTop1Pct)),
		})
	}

	// New contributor share
	if data.Signals.Authors.NewContributorChurnPct >= 40 {
		bullets = append(bullets, types.NarrativeBullet{
			Kind: "warn",
			Text: fmt.Sprintf("%s%% of LOC changed by new contributors (first commit in window) — review load likely elevated.",
				trimTrailingZero(data.Signals.Authors.NewContributorChurnPct)),
		})
	}

	// Coverage
	if data.Signals.Coverage != nil {
		pct := data.Signals.Coverage.Percentage
		if pct < 50 {
			bullets = append(bullets, types.NarrativeBullet{
				Kind: "warn",
				Text: fmt.Sprintf("Test coverage at %.1f%% — low for this size of codebase.", pct),
			})
		} else if pct >= 80 {
			bullets = append(bullets, types.NarrativeBullet{
				Kind: "good",
				Text: fmt.Sprintf("Test coverage %.1f%% — healthy safety net.", pct),
			})
		}
	}

	// Rewritten files
	rewrittenCount := 0
	for _, f := range data.Signals.FileChurn.TopChurners {
		if f.Rewritten {
			rewrittenCount++
		}
	}
	if rewrittenCount > 0 {
		noun := "file was"
		if rewrittenCount > 1 {
			noun = "files were"
		}
		bullets = append(bullets, types.NarrativeBullet{
			Kind: "info",
			Text: fmt.Sprintf("%d top-churn %s effectively rewritten (churn > 5× current size).", rewrittenCount, noun),
		})
	}

	// Throughput
	if data.Signals.FileChurn.LinesPerDay >= 1000 {
		k := "info"
		if data.Signals.FileChurn.LinesPerDay >= 5000 {
			k = "warn"
		}
		bullets = append(bullets, types.NarrativeBullet{
			Kind: k,
			Text: fmt.Sprintf("%s lines changed per day — high throughput.",
				formatThousands(int(data.Signals.FileChurn.LinesPerDay))),
		})
	}

	order := map[string]int{"alert": 0, "warn": 1, "info": 2, "good": 3}
	sort.SliceStable(bullets, func(i, j int) bool {
		return order[bullets[i].Kind] < order[bullets[j].Kind]
	})
	if len(bullets) > 8 {
		bullets = bullets[:8]
	}
	return bullets
}

// ComputeRollingTimeline produces rolling 7-day composite score points.
func ComputeRollingTimeline(
	commits []types.CommitRecord,
	windowStart, windowEnd time.Time,
	bugOpts signals.BugOptions,
) []types.RollingPoint {
	oneDay := int64(24 * 60 * 60 * 1000)

	type tagged struct {
		commit types.CommitRecord
		tier   signals.BugTier
	}
	tgd := make([]tagged, len(commits))
	for i, c := range commits {
		tgd[i] = tagged{commit: c, tier: signals.ClassifyCommit(c.Message, c.IsRevert, bugOpts)}
	}

	startMs := windowStart.UnixNano() / 1e6
	endMs := windowEnd.UnixNano() / 1e6
	totalDays := int((endMs - startMs + oneDay - 1) / oneDay)
	if totalDays < 1 {
		totalDays = 1
	}

	points := []types.RollingPoint{}
	for i := 6; i < totalDays; i++ {
		rightMs := startMs + int64(i+1)*oneDay
		leftMs := rightMs - 7*oneDay

		n, bugs, chaos := 0, 0, 0
		for _, t := range tgd {
			m := t.commit.Date.UnixNano() / 1e6
			if m >= leftMs && m < rightMs {
				n++
				if t.tier != signals.TierNone {
					bugs++
				}
				if t.tier == signals.TierChaos {
					chaos++
				}
			}
		}
		bugPct := 0.0
		if n > 0 {
			bugPct = float64(bugs) / float64(n)
		}

		perDay := float64(n) / 7
		var volumeScore float64
		switch {
		case perDay < 2:
			volumeScore = perDay / 2 * 20
		case perDay < 10:
			volumeScore = 20 + ((perDay - 2) / 8 * 30)
		case perDay < 30:
			volumeScore = 50 + ((perDay - 10) / 20 * 25)
		default:
			volumeScore = math.Min(100, 75+((perDay-30)/30*25))
		}
		chaosPct := 0.0
		if n > 0 {
			chaosPct = float64(chaos) / float64(n) * 100
		}
		bugScore := math.Min(100, chaosPct*4+(bugPct*100-chaosPct)*1.2)
		dryPenalty := 0.0
		if n == 0 {
			dryPenalty = 30
		}
		composite := int(math.Min(100, math.Round(volumeScore*0.35+bugScore*0.55+dryPenalty)))

		rightDate := time.Unix(0, rightMs*int64(time.Millisecond)).UTC().Format("2006-01-02")
		points = append(points, types.RollingPoint{
			Date:    rightDate,
			Score:   composite,
			Commits: n,
			BugPct:  math.Round(bugPct*1000) / 1000,
		})
	}
	return points
}

func trimTrailingZero(x float64) string {
	s := strconvFormat(x)
	return s
}

func strconvFormat(x float64) string {
	// 1-decimal if non-integer, integer if integer
	if x == math.Trunc(x) {
		return fmt.Sprintf("%d", int(x))
	}
	return fmt.Sprintf("%g", math.Round(x*10)/10)
}

func formatThousands(n int) string {
	s := fmt.Sprintf("%d", n)
	if n < 1000 {
		return s
	}
	// insert commas from the right
	out := make([]byte, 0, len(s)+len(s)/3)
	rev := reverse(s)
	for i, c := range rev {
		if i > 0 && i%3 == 0 {
			out = append(out, ',')
		}
		out = append(out, byte(c))
	}
	return reverse(string(out))
}

func reverse(s string) string {
	r := []byte(s)
	for i, j := 0, len(r)-1; i < j; i, j = i+1, j-1 {
		r[i], r[j] = r[j], r[i]
	}
	return string(r)
}

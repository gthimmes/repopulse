// Package signals contains per-signal computations. Each function takes
// []CommitRecord + options and returns a *Signal struct.
package signals

import (
	"math"
	"regexp"
	"sort"
	"strings"
	"time"

	"repopulse/internal/types"
)

// BugTier is the classification of a commit's bug-ness.
type BugTier string

const (
	TierChaos   BugTier = "chaos"
	TierNormal  BugTier = "normal"
	TierRoutine BugTier = "routine"
	TierNone    BugTier = "none"
)

type BugOptions struct {
	ChaosKeywords   []string
	NormalKeywords  []string
	RoutineKeywords []string
}

// Max commits per tier retained for explainability drill-down.
const sampleCapPerTier = 20

// Tier weight applied to composite scoring.
var tierWeight = map[BugTier]float64{
	TierChaos:   1.0,
	TierNormal:  0.4,
	TierRoutine: 0.1,
}

// ClassifyCommit returns the tier of a commit subject.
func ClassifyCommit(message string, isRevert bool, opts BugOptions) BugTier {
	t, _ := ClassifyCommitWithKeyword(message, isRevert, opts)
	return t
}

// nonBugPrefixes are Conventional-Commit types that by definition aren't
// bug fixes. Subjects starting with these (optionally with a scope) short-
// circuit to TierNone so a `feat: add X to replace buggy Y` doesn't get
// caught by body-content keyword matches. `fix` and `revert` are absent
// on purpose — those remain classified by the regular keyword path.
var nonBugPrefixRE = regexp.MustCompile(`^\s*(feat|feature|chore|docs|doc|style|test|tests|refactor|ci|build|perf)(\([^)]*\))?\s*!?:`)

// ClassifyCommitWithKeyword returns the tier plus the keyword that matched
// (or "(revert)" for reverts, empty string for none).
func ClassifyCommitWithKeyword(message string, isRevert bool, opts BugOptions) (BugTier, string) {
	if isRevert {
		return TierChaos, "(revert)"
	}
	lower := strings.ToLower(message)

	// Conventional-commit prefix veto: feat/chore/docs/etc. never classify
	// as a bug regardless of body content.
	if firstLine := strings.SplitN(lower, "\n", 2)[0]; nonBugPrefixRE.MatchString(firstLine) {
		return TierNone, ""
	}

	if kw := findKeyword(lower, opts.ChaosKeywords); kw != "" {
		return TierChaos, kw
	}
	routineHit := findKeyword(lower, opts.RoutineKeywords)
	normalHit := findKeyword(lower, opts.NormalKeywords)
	if routineHit != "" {
		return TierRoutine, routineHit
	}
	if normalHit != "" {
		return TierNormal, normalHit
	}
	return TierNone, ""
}

func findKeyword(msg string, keywords []string) string {
	for _, kw := range keywords {
		re := regexp.MustCompile(`\b` + regexp.QuoteMeta(kw) + `\b`)
		if re.MatchString(msg) {
			return kw
		}
	}
	return ""
}

// ComputeBugRatio computes the bug signal with tiered weighting and revert detection.
func ComputeBugRatio(commits []types.CommitRecord, opts BugOptions) types.BugSignal {
	if len(commits) == 0 {
		return types.BugSignal{
			Type:              "bugRatio",
			BugCommitsByDay:   []types.DayBucket{},
			NormalCommitsByDay: []types.DayBucket{},
			ChaosCommitsByDay: []types.DayBucket{},
			ClassifiedSamples: types.BugClassifiedGroups{
				Chaos: []types.BugClassifiedCommit{}, Normal: []types.BugClassifiedCommit{}, Routine: []types.BugClassifiedCommit{},
			},
		}
	}

	type tagged struct {
		commit         types.CommitRecord
		tier           BugTier
		matchedKeyword string
	}
	allTagged := make([]tagged, len(commits))
	for i, c := range commits {
		t, kw := ClassifyCommitWithKeyword(c.Message, c.IsRevert, opts)
		allTagged[i] = tagged{c, t, kw}
	}

	chaosCount, normalCount, routineCount := 0, 0, 0
	for _, t := range allTagged {
		switch t.tier {
		case TierChaos:
			chaosCount++
		case TierNormal:
			normalCount++
		case TierRoutine:
			routineCount++
		}
	}
	bugCount := chaosCount + normalCount + routineCount

	// Reverted-within-7-days
	hashIndex := map[string]time.Time{}
	for _, c := range commits {
		hashIndex[shortHash(c.Hash)] = c.Date
	}
	revertedWithin7d := 0
	for _, c := range commits {
		if !c.IsRevert || c.RevertedHashShort == "" {
			continue
		}
		target, ok := hashIndex[shortHash(c.RevertedHashShort)]
		if !ok {
			continue
		}
		diffDays := math.Abs(c.Date.Sub(target).Hours()) / 24
		if diffDays <= 7 {
			revertedWithin7d++
		}
	}

	// Longest consecutive streak of any-tier bug commits
	longestStreak, currentStreak := 0, 0
	for _, t := range allTagged {
		if t.tier != TierNone {
			currentStreak++
			if currentStreak > longestStreak {
				longestStreak = currentStreak
			}
		} else {
			currentStreak = 0
		}
	}

	// Daily buckets
	bugByDay, normalByDay, chaosByDay := map[string]int{}, map[string]int{}, map[string]int{}
	for _, t := range allTagged {
		key := t.commit.Date.UTC().Format("2006-01-02")
		if t.tier == TierNone {
			normalByDay[key]++
		} else {
			bugByDay[key]++
			if t.tier == TierChaos {
				chaosByDay[key]++
			}
		}
	}
	sortedDates := mergeSortedDateKeys(bugByDay, normalByDay)
	bugCommitsByDay := datesToBuckets(sortedDates, bugByDay)
	normalCommitsByDay := datesToBuckets(sortedDates, normalByDay)
	chaosCommitsByDay := datesToBuckets(sortedDates, chaosByDay)

	// Weighted ratio
	weightedBugs := float64(chaosCount)*tierWeight[TierChaos] +
		float64(normalCount)*tierWeight[TierNormal] +
		float64(routineCount)*tierWeight[TierRoutine]
	weightedRatioPct := (weightedBugs / float64(len(commits))) * 100

	var base float64
	switch {
	case weightedRatioPct < 5:
		base = (weightedRatioPct / 5) * 20
	case weightedRatioPct < 15:
		base = 20 + ((weightedRatioPct - 5) / 10 * 35)
	case weightedRatioPct < 30:
		base = 55 + ((weightedRatioPct - 15) / 15 * 30)
	default:
		base = math.Min(100, 85+((weightedRatioPct-30)/30*15))
	}

	clusterBonus := 0.0
	switch {
	case longestStreak >= 7:
		clusterBonus = 12
	case longestStreak >= 4:
		clusterBonus = 6
	}
	revertBonus := math.Min(15, float64(revertedWithin7d)*4)

	score := int(math.Min(100, math.Round(base+clusterBonus+revertBonus)))

	// Build per-tier classified samples, newest first, capped
	buildSamples := func(tier BugTier) []types.BugClassifiedCommit {
		filtered := make([]tagged, 0)
		for _, t := range allTagged {
			if t.tier == tier {
				filtered = append(filtered, t)
			}
		}
		sort.SliceStable(filtered, func(i, j int) bool {
			return filtered[i].commit.Date.After(filtered[j].commit.Date)
		})
		if len(filtered) > sampleCapPerTier {
			filtered = filtered[:sampleCapPerTier]
		}
		out := make([]types.BugClassifiedCommit, len(filtered))
		for i, t := range filtered {
			out[i] = types.BugClassifiedCommit{
				Hash:           shortHash(t.commit.Hash),
				Date:           t.commit.Date.UTC().Format("2006-01-02"),
				Author:         t.commit.AuthorName,
				Message:        firstLine(t.commit.Message, 160),
				MatchedKeyword: t.matchedKeyword,
			}
		}
		return out
	}

	return types.BugSignal{
		Type:               "bugRatio",
		Score:              score,
		BugCommitCount:     bugCount,
		ChaosCommitCount:   chaosCount,
		RoutineFixCount:    routineCount,
		NormalFixCount:     normalCount,
		TotalCommits:       len(commits),
		Ratio:              round3(float64(bugCount) / float64(len(commits))),
		LongestFixStreak:   longestStreak,
		BugCommitsByDay:    bugCommitsByDay,
		NormalCommitsByDay: normalCommitsByDay,
		ChaosCommitsByDay:  chaosCommitsByDay,
		RevertedWithin7d:   revertedWithin7d,
		ClassifiedSamples: types.BugClassifiedGroups{
			Chaos:   buildSamples(TierChaos),
			Normal:  buildSamples(TierNormal),
			Routine: buildSamples(TierRoutine),
		},
	}
}

func shortHash(h string) string {
	if len(h) < 7 {
		return h
	}
	return h[:7]
}

func firstLine(msg string, maxLen int) string {
	if idx := strings.Index(msg, "\n"); idx >= 0 {
		msg = msg[:idx]
	}
	if len(msg) > maxLen {
		return msg[:maxLen]
	}
	return msg
}

func round3(x float64) float64 {
	return math.Round(x*1000) / 1000
}

// mergeSortedDateKeys merges keys from both maps and returns sorted ascending.
func mergeSortedDateKeys(a, b map[string]int) []string {
	set := map[string]struct{}{}
	for k := range a {
		set[k] = struct{}{}
	}
	for k := range b {
		set[k] = struct{}{}
	}
	dates := make([]string, 0, len(set))
	for k := range set {
		dates = append(dates, k)
	}
	sort.Strings(dates)
	return dates
}

func datesToBuckets(dates []string, counts map[string]int) []types.DayBucket {
	out := make([]types.DayBucket, len(dates))
	for i, d := range dates {
		out[i] = types.DayBucket{Date: d, Count: counts[d]}
	}
	return out
}

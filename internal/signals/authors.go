package signals

import (
	"math"
	"sort"
	"time"

	"repopulse/internal/types"
)

type AuthorOptions struct {
	IsExcluded            func(string) bool
	WindowStart           time.Time
	PreWindowAuthorEmails map[string]struct{}
	NightStartHour        int
	NightEndHour          int
}

// ComputeAuthors — weekend/night rate + bus factor + new contributor share.
func ComputeAuthors(commits []types.CommitRecord, opts AuthorOptions) types.AuthorSignal {
	isExcluded := opts.IsExcluded
	if isExcluded == nil {
		isExcluded = func(string) bool { return false }
	}
	nightStart := opts.NightStartHour
	if nightStart == 0 {
		nightStart = 20
	}
	nightEnd := opts.NightEndHour
	if nightEnd == 0 {
		nightEnd = 7
	}

	type fileTouch struct {
		commits, added, removed int
	}
	type agg struct {
		name                string
		email               string
		commitCount         int
		linesChanged        int
		weekendNightCommits int
		firstSeen           time.Time
		// per-file touches by this author for the drill-down panel
		files map[string]*fileTouch
	}
	byEmail := map[string]*agg{}
	totalLines := 0
	weekendNightTotal := 0

	for _, c := range commits {
		a, ok := byEmail[c.AuthorEmail]
		if !ok {
			a = &agg{
				name:      c.AuthorName,
				email:     c.AuthorEmail,
				firstSeen: c.AuthorDate,
				files:     map[string]*fileTouch{},
			}
			byEmail[c.AuthorEmail] = a
		}
		a.commitCount++
		if c.AuthorDate.Before(a.firstSeen) {
			a.firstSeen = c.AuthorDate
		}
		for _, f := range c.FilesChanged {
			if isExcluded(f.Path) {
				continue
			}
			lines := f.Added + f.Removed
			a.linesChanged += lines
			totalLines += lines

			ft, ok := a.files[f.Path]
			if !ok {
				ft = &fileTouch{}
				a.files[f.Path] = ft
			}
			ft.commits++
			ft.added += f.Added
			ft.removed += f.Removed
		}
		if isWeekendOrNight(c.AuthorDate, nightStart, nightEnd) {
			a.weekendNightCommits++
			weekendNightTotal++
		}
	}

	totalAuthors := len(byEmail)
	totalCommits := len(commits)
	if totalCommits < 1 {
		totalCommits = 1
	}

	ranked := make([]*agg, 0, totalAuthors)
	for _, a := range byEmail {
		ranked = append(ranked, a)
	}
	sort.SliceStable(ranked, func(i, j int) bool {
		return ranked[i].commitCount > ranked[j].commitCount
	})

	top1Pct := 0.0
	if len(ranked) > 0 {
		top1Pct = float64(ranked[0].commitCount) / float64(totalCommits) * 100
	}
	top3Sum := 0
	for i := 0; i < len(ranked) && i < 3; i++ {
		top3Sum += ranked[i].commitCount
	}
	top3Pct := float64(top3Sum) / float64(totalCommits) * 100

	isNew := func(a *agg) bool {
		if opts.PreWindowAuthorEmails != nil {
			_, inPre := opts.PreWindowAuthorEmails[a.email]
			return !inPre
		}
		return !a.firstSeen.Before(opts.WindowStart)
	}
	newLines := 0
	for _, a := range ranked {
		if isNew(a) {
			newLines += a.linesChanged
		}
	}
	newContribPct := 0.0
	if totalLines > 0 {
		newContribPct = float64(newLines) / float64(totalLines) * 100
	}

	weekendNightPct := float64(weekendNightTotal) / float64(totalCommits) * 100

	var weekendNightScore float64
	if weekendNightPct < 20 {
		weekendNightScore = weekendNightPct / 20 * 60
	} else {
		weekendNightScore = math.Min(100, 60+((weekendNightPct-20)/20*40))
	}

	var busFactorScore float64
	switch {
	case top1Pct < 30:
		busFactorScore = top1Pct / 30 * 30
	case top1Pct < 60:
		busFactorScore = 30 + ((top1Pct - 30) / 30 * 40)
	default:
		busFactorScore = math.Min(100, 70+((top1Pct-60)/40*30))
	}

	var newContribScore float64
	if newContribPct < 30 {
		newContribScore = newContribPct / 30 * 20
	} else {
		newContribScore = math.Min(40, 20+((newContribPct-30)/70*20))
	}

	score := int(math.Round(weekendNightScore*0.45 + busFactorScore*0.35 + newContribScore*0.20))

	// Build the full Contributors list, sorted by lines-changed desc.
	// This drives the bottom-of-report explorer; mini-stats above use
	// the same data for "Distinct authors" / "Top author share" / etc.
	contribOrder := append([]*agg(nil), ranked...)
	sort.SliceStable(contribOrder, func(i, j int) bool {
		if contribOrder[i].linesChanged != contribOrder[j].linesChanged {
			return contribOrder[i].linesChanged > contribOrder[j].linesChanged
		}
		return contribOrder[i].commitCount > contribOrder[j].commitCount
	})
	// Pull each author's top 5 most-touched files for their drill-down.
	pickTopFiles := func(files map[string]*fileTouch, n int) []types.AuthorFileTouch {
		type pair struct {
			path                    string
			commits, added, removed int
		}
		pairs := make([]pair, 0, len(files))
		for p, t := range files {
			pairs = append(pairs, pair{p, t.commits, t.added, t.removed})
		}
		sort.Slice(pairs, func(i, j int) bool {
			if pairs[i].commits != pairs[j].commits {
				return pairs[i].commits > pairs[j].commits
			}
			return (pairs[i].added + pairs[i].removed) > (pairs[j].added + pairs[j].removed)
		})
		if len(pairs) > n {
			pairs = pairs[:n]
		}
		out := make([]types.AuthorFileTouch, len(pairs))
		for i, p := range pairs {
			out[i] = types.AuthorFileTouch{Path: p.path, Commits: p.commits, Added: p.added, Removed: p.removed}
		}
		return out
	}

	contributors := make([]types.AuthorEntry, len(contribOrder))
	for i, a := range contribOrder {
		contributors[i] = types.AuthorEntry{
			Name:                a.name,
			Email:               a.email,
			Commits:             a.commitCount,
			LinesChanged:        a.linesChanged,
			WeekendNightCommits: a.weekendNightCommits,
			FirstSeen:           a.firstSeen.UTC().Format("2006-01-02"),
			IsNew:               isNew(a),
			TopFiles:            pickTopFiles(a.files, 5),
		}
	}

	return types.AuthorSignal{
		Type:                   "authors",
		Score:                  minInt(100, score),
		TotalAuthors:           totalAuthors,
		WeekendNightPct:        round1(weekendNightPct),
		BusFactorTop1Pct:       round1(top1Pct),
		BusFactorTop3Pct:       round1(top3Pct),
		NewContributorChurnPct: round1(newContribPct),
		Contributors:           contributors,
	}
}

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

func round1(x float64) float64 {
	return math.Round(x*10) / 10
}


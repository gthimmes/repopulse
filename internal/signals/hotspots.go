package signals

import (
	"math"
	"sort"
	"time"

	"repopulse/internal/codeowners"
	"repopulse/internal/types"
)

type HotspotOptions struct {
	IsExcluded           func(string) bool
	BugOptions           BugOptions
	Limit                int
	DrilldownCommitLimit int
	Codeowners           *codeowners.Codeowners
	WindowEnd            time.Time
	ChurnLookup          map[string]types.ChurnEntry
}

// ComputeHotspots — Feathers-style: churn * bug involvement. Only surfaces
// files that have actual bug involvement.
func ComputeHotspots(commits []types.CommitRecord, opts HotspotOptions) types.HotspotSignal {
	isExcluded := opts.IsExcluded
	if isExcluded == nil {
		isExcluded = func(string) bool { return false }
	}
	limit := opts.Limit
	if limit < 1 {
		limit = 15
	}
	drilldownLimit := opts.DrilldownCommitLimit
	if drilldownLimit < 1 {
		drilldownLimit = 10
	}

	type taggedBug struct {
		commit types.CommitRecord
		tier   BugTier
	}
	type fileAgg struct {
		churn          int
		bugTouches     float64
		chaosTouches   int
		totalCommits   int
		authors        map[string]struct{}
		authorCommits  map[string]int
		lastTouchedMs  int64
		bugCommits     []taggedBug
	}
	byFile := map[string]*fileAgg{}

	getOrNew := func(path string) *fileAgg {
		if f, ok := byFile[path]; ok {
			return f
		}
		f := &fileAgg{
			authors:       map[string]struct{}{},
			authorCommits: map[string]int{},
		}
		byFile[path] = f
		return f
	}

	for _, c := range commits {
		tier := ClassifyCommit(c.Message, c.IsRevert, opts.BugOptions)
		isBug := tier != TierNone
		isChaos := tier == TierChaos
		bugWeight := 0.0
		switch tier {
		case TierChaos:
			bugWeight = 1.0
		case TierNormal:
			bugWeight = 0.4
		case TierRoutine:
			bugWeight = 0.1
		}
		commitMs := c.Date.UnixNano() / 1e6

		for _, f := range c.FilesChanged {
			if isExcluded(f.Path) {
				continue
			}
			agg := getOrNew(f.Path)
			agg.churn += f.Added + f.Removed
			agg.totalCommits++
			agg.authors[c.AuthorEmail] = struct{}{}
			agg.authorCommits[c.AuthorName]++
			if commitMs > agg.lastTouchedMs {
				agg.lastTouchedMs = commitMs
			}
			if isBug {
				agg.bugTouches += bugWeight
				agg.bugCommits = append(agg.bugCommits, taggedBug{commit: c, tier: tier})
			}
			if isChaos {
				agg.chaosTouches++
			}
		}
	}

	// Rank by churn desc
	type scored struct {
		path string
		agg  *fileAgg
	}
	ranked := make([]scored, 0, len(byFile))
	for p, a := range byFile {
		ranked = append(ranked, scored{path: p, agg: a})
	}
	sort.Slice(ranked, func(i, j int) bool { return ranked[i].agg.churn > ranked[j].agg.churn })

	maxChurn := 1
	maxBugTouches := 1.0
	if len(ranked) > 0 {
		maxChurn = ranked[0].agg.churn
		if maxChurn < 1 {
			maxChurn = 1
		}
	}
	for _, r := range ranked {
		if r.agg.bugTouches > maxBugTouches {
			maxBugTouches = r.agg.bugTouches
		}
	}

	windowEnd := opts.WindowEnd
	if windowEnd.IsZero() {
		windowEnd = time.Now()
	}

	entries := make([]types.HotspotEntry, 0, len(ranked))
	for idx, r := range ranked {
		churnScore := float64(r.agg.churn) / float64(maxChurn) * 100
		bugScore := r.agg.bugTouches / maxBugTouches * 100
		hotspotScore := int(math.Round(churnScore*0.4 + bugScore*0.6))

		// Top 3 authors
		type kv struct {
			name string
			n    int
		}
		authors := make([]kv, 0, len(r.agg.authorCommits))
		for n, v := range r.agg.authorCommits {
			authors = append(authors, kv{n, v})
		}
		sort.Slice(authors, func(i, j int) bool { return authors[i].n > authors[j].n })
		if len(authors) > 3 {
			authors = authors[:3]
		}
		topAuthors := make([]types.HotspotFileAuthor, len(authors))
		for i, a := range authors {
			topAuthors[i] = types.HotspotFileAuthor{Name: a.name, Commits: a.n}
		}

		// Recent bug commits, newest first, capped
		bc := append([]taggedBug(nil), r.agg.bugCommits...)
		sort.SliceStable(bc, func(i, j int) bool {
			return bc[i].commit.Date.After(bc[j].commit.Date)
		})
		if len(bc) > drilldownLimit {
			bc = bc[:drilldownLimit]
		}
		recent := make([]types.HotspotCommit, len(bc))
		for i, b := range bc {
			recent[i] = types.HotspotCommit{
				Hash:    shortHash(b.commit.Hash),
				Date:    b.commit.Date.UTC().Format("2006-01-02"),
				Author:  b.commit.AuthorName,
				Message: firstLine(b.commit.Message, 140),
				Tier:    string(b.tier),
			}
		}

		owners := []string{}
		if opts.Codeowners != nil {
			owners = opts.Codeowners.OwnersFor(r.path)
			if owners == nil {
				owners = []string{}
			}
		}

		lastTouched := ""
		if r.agg.lastTouchedMs > 0 {
			lastTouched = time.Unix(0, r.agg.lastTouchedMs*1e6).UTC().Format("2006-01-02")
		}

		entry := types.HotspotEntry{
			Path:             r.path,
			ChurnRank:        idx + 1,
			BugTouches:       math.Round(r.agg.bugTouches*10) / 10,
			ChaosTouches:     r.agg.chaosTouches,
			TotalCommits:     r.agg.totalCommits,
			HotspotScore:     hotspotScore,
			Authors:          len(r.agg.authors),
			LastTouched:      lastTouched,
			TopAuthorsOfFile: topAuthors,
			RecentBugCommits: recent,
			Owners:           owners,
			Recommendations:  []types.HotspotRecommendation{},
		}

		var churnEntry *types.ChurnEntry
		if opts.ChurnLookup != nil {
			if ce, ok := opts.ChurnLookup[r.path]; ok {
				churnEntry = &ce
			}
		}
		entry.Recommendations = BuildRecommendations(entry, churnEntry, windowEnd, 3)

		entries = append(entries, entry)
	}

	// Filter to files with real bug involvement, sort by hotspot score, cap
	withBugs := make([]types.HotspotEntry, 0, len(entries))
	for _, e := range entries {
		if e.BugTouches > 0 {
			withBugs = append(withBugs, e)
		}
	}
	sort.Slice(withBugs, func(i, j int) bool { return withBugs[i].HotspotScore > withBugs[j].HotspotScore })
	if len(withBugs) > limit {
		withBugs = withBugs[:limit]
	}

	return types.HotspotSignal{Type: "hotspots", Hotspots: withBugs}
}

package signals

import (
	"math"
	"sort"
	"time"

	"repopulse/internal/types"
)

// ChurnOptions controls churn computation.
type ChurnOptions struct {
	IsExcluded func(string) bool
	WindowDays int
	// GetLineCount returns the current line count of a file at HEAD. Used
	// for the churn-to-size ratio.
	GetLineCount func(path string) int
	// BugOptions classifies the recent-commit drill-down samples per file
	// so the UI can color-tag them with their bug tier. When zero-valued
	// the classifier still runs and returns TierNone for everything.
	BugOptions BugOptions
}

// churnRecentCommitCap caps how many recent commits we keep per file
// for the drill-down panel.
const churnRecentCommitCap = 12

const (
	ratioCap                = 10.0
	rewrittenThreshold      = 5.0
	highChurnRatio          = 2.0
	minLOCForEligibility    = 20
	minChurnForEligibility  = 20
)

// ComputeChurn computes the file churn signal.
func ComputeChurn(commits []types.CommitRecord, opts ChurnOptions) types.ChurnSignal {
	isExcluded := opts.IsExcluded
	if isExcluded == nil {
		isExcluded = func(string) bool { return false }
	}
	windowDays := opts.WindowDays
	if windowDays < 1 {
		windowDays = 1
	}

	type pathStats struct {
		added, removed int
		// commits touching this file, newest first after sort below
		touchingCommits []types.CommitRecord
		authorCommits   map[string]int
	}
	fileStats := map[string]*pathStats{}
	for _, c := range commits {
		for _, f := range c.FilesChanged {
			if isExcluded(f.Path) {
				continue
			}
			s := fileStats[f.Path]
			if s == nil {
				s = &pathStats{authorCommits: map[string]int{}}
				fileStats[f.Path] = s
			}
			s.added += f.Added
			s.removed += f.Removed
			s.touchingCommits = append(s.touchingCommits, c)
			s.authorCommits[c.AuthorName]++
		}
	}

	totalFilesTouched := len(fileStats)
	if totalFilesTouched == 0 {
		return types.ChurnSignal{
			Type:        "fileChurn",
			TopChurners: []types.ChurnEntry{},
		}
	}

	// Sort by total churn desc
	type stat struct {
		path           string
		added, removed int
		total          int
	}
	sorted := make([]stat, 0, totalFilesTouched)
	for path, s := range fileStats {
		sorted = append(sorted, stat{path: path, added: s.added, removed: s.removed, total: s.added + s.removed})
	}
	sort.Slice(sorted, func(i, j int) bool { return sorted[i].total > sorted[j].total })

	// Resolve line counts for top 100
	topN := 100
	if len(sorted) < topN {
		topN = len(sorted)
	}
	resolved := make([]types.ChurnEntry, topN)
	for i := 0; i < topN; i++ {
		f := sorted[i]
		lineCount := 0
		if opts.GetLineCount != nil {
			lineCount = opts.GetLineCount(f.path)
		}
		denom := lineCount
		if denom < 1 {
			denom = 1
		}
		rawRatio := float64(f.total) / float64(denom)
		capped := math.Min(ratioCap, rawRatio)

		stats := fileStats[f.path]
		entry := types.ChurnEntry{
			Path:      f.path,
			Added:     f.added,
			Removed:   f.removed,
			Ratio:     math.Round(capped*100) / 100,
			Rewritten: rawRatio > rewrittenThreshold,
		}
		// Build drill-down data for the top 20 only (what the report
		// actually surfaces). Skipping it for the rest keeps the JSON
		// snapshot small on large repos.
		if i < 20 && stats != nil {
			entry.TotalCommits = len(stats.touchingCommits)
			entry.TopAuthorsOfFile = topAuthorsOfFile(stats.authorCommits)
			entry.RecentCommits = recentCommitsForFile(stats.touchingCommits, opts.BugOptions)
			entry.LastTouched = lastTouchedDate(stats.touchingCommits)
		}
		resolved[i] = entry
	}

	// Eligible: has real churn AND real size AND not rewritten
	eligible := 0
	highChurn := 0
	for _, f := range resolved {
		totalChurn := f.Added + f.Removed
		if totalChurn >= minChurnForEligibility && !f.Rewritten {
			eligible++
			if f.Ratio > highChurnRatio {
				highChurn++
			}
		}
	}
	eligibleDenom := eligible
	if eligibleDenom < 1 {
		eligibleDenom = 1
	}

	// Total across ALL files
	totalLinesChanged := 0
	for _, s := range sorted {
		totalLinesChanged += s.total
	}
	linesPerDay := float64(totalLinesChanged) / float64(windowDays)

	densityPct := float64(highChurn) / float64(eligibleDenom) * 100
	var densityScore float64
	switch {
	case densityPct < 5:
		densityScore = densityPct / 5 * 20
	case densityPct < 15:
		densityScore = 20 + ((densityPct - 5) / 10 * 30)
	case densityPct < 30:
		densityScore = 50 + ((densityPct - 15) / 15 * 30)
	default:
		densityScore = math.Min(100, 80+((densityPct-30)/30*20))
	}

	var throughputScore float64
	switch {
	case linesPerDay < 200:
		throughputScore = linesPerDay / 200 * 20
	case linesPerDay < 1000:
		throughputScore = 20 + ((linesPerDay - 200) / 800 * 30)
	case linesPerDay < 5000:
		throughputScore = 50 + ((linesPerDay - 1000) / 4000 * 25)
	default:
		throughputScore = math.Min(100, 75+((linesPerDay-5000)/10000*25))
	}

	score := int(math.Min(100, math.Round(densityScore*0.6+throughputScore*0.4)))

	top20 := resolved
	if len(top20) > 20 {
		top20 = top20[:20]
	}

	return types.ChurnSignal{
		Type:               "fileChurn",
		Score:              score,
		TopChurners:        top20,
		TotalFilesTouched:  totalFilesTouched,
		EligibleFileCount:  eligible,
		HighChurnFileCount: highChurn,
		TotalLinesChanged:  totalLinesChanged,
		LinesPerDay:        math.Round(linesPerDay),
	}
}

// topAuthorsOfFile returns the top-3 authors by commit count for a file,
// sorted descending. Mirrors the hotspot signal's per-file author block
// so the renderer can share UI.
func topAuthorsOfFile(authorCommits map[string]int) []types.HotspotFileAuthor {
	type kv struct {
		name string
		n    int
	}
	pairs := make([]kv, 0, len(authorCommits))
	for n, c := range authorCommits {
		pairs = append(pairs, kv{n, c})
	}
	sort.Slice(pairs, func(i, j int) bool { return pairs[i].n > pairs[j].n })
	if len(pairs) > 3 {
		pairs = pairs[:3]
	}
	out := make([]types.HotspotFileAuthor, len(pairs))
	for i, p := range pairs {
		out[i] = types.HotspotFileAuthor{Name: p.name, Commits: p.n}
	}
	return out
}

// recentCommitsForFile returns the newest-first capped sample of commits
// touching one file, classified with bug tier so the UI can color them.
func recentCommitsForFile(commits []types.CommitRecord, bugOpts BugOptions) []types.HotspotCommit {
	cs := append([]types.CommitRecord(nil), commits...)
	sort.SliceStable(cs, func(i, j int) bool {
		return cs[i].Date.After(cs[j].Date)
	})
	if len(cs) > churnRecentCommitCap {
		cs = cs[:churnRecentCommitCap]
	}
	out := make([]types.HotspotCommit, len(cs))
	for i, c := range cs {
		tier := ClassifyCommit(c.Message, c.IsRevert, bugOpts)
		out[i] = types.HotspotCommit{
			Hash:    shortHash(c.Hash),
			Date:    c.Date.UTC().Format("2006-01-02"),
			Author:  c.AuthorName,
			Message: firstLine(c.Message, 140),
			Tier:    string(tier),
		}
	}
	return out
}

func lastTouchedDate(commits []types.CommitRecord) string {
	var latest time.Time
	for _, c := range commits {
		if c.Date.After(latest) {
			latest = c.Date
		}
	}
	if latest.IsZero() {
		return ""
	}
	return latest.UTC().Format("2006-01-02")
}

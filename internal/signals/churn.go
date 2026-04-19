package signals

import (
	"math"
	"sort"

	"repopulse/internal/types"
)

// ChurnOptions controls churn computation.
type ChurnOptions struct {
	IsExcluded func(string) bool
	WindowDays int
	// GetLineCount returns the current line count of a file at HEAD. Used
	// for the churn-to-size ratio.
	GetLineCount func(path string) int
}

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
	}
	fileStats := map[string]*pathStats{}
	for _, c := range commits {
		for _, f := range c.FilesChanged {
			if isExcluded(f.Path) {
				continue
			}
			s := fileStats[f.Path]
			if s == nil {
				s = &pathStats{}
				fileStats[f.Path] = s
			}
			s.added += f.Added
			s.removed += f.Removed
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
		resolved[i] = types.ChurnEntry{
			Path:      f.path,
			Added:     f.added,
			Removed:   f.removed,
			Ratio:     math.Round(capped*100) / 100,
			Rewritten: rawRatio > rewrittenThreshold,
		}
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

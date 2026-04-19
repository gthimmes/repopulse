package signals

import (
	"math"
	"sort"
	"strings"

	"repopulse/internal/codeowners"
	"repopulse/internal/types"
)

type ModuleOptions struct {
	IsExcluded func(string) bool
	BugOptions BugOptions
	Depth      int
	MinCommits int
	Codeowners *codeowners.Codeowners
}

// ComputeModules groups by top-level directory and scores each module.
func ComputeModules(commits []types.CommitRecord, opts ModuleOptions) types.ModuleSignal {
	isExcluded := opts.IsExcluded
	if isExcluded == nil {
		isExcluded = func(string) bool { return false }
	}
	depth := opts.Depth
	if depth < 1 {
		depth = 1
	}
	minCommits := opts.MinCommits
	if minCommits < 1 {
		minCommits = 3
	}

	type moduleStat struct {
		commits      map[string]struct{}
		linesChanged int
		bugCommits   int
		chaosCommits int
		authors      map[string]struct{}
		fileChurn    map[string]int
	}
	stats := map[string]*moduleStat{}

	getOrNew := func(name string) *moduleStat {
		if s, ok := stats[name]; ok {
			return s
		}
		s := &moduleStat{
			commits:   map[string]struct{}{},
			authors:   map[string]struct{}{},
			fileChurn: map[string]int{},
		}
		stats[name] = s
		return s
	}

	for _, c := range commits {
		if len(c.FilesChanged) == 0 {
			continue
		}
		tier := ClassifyCommit(c.Message, c.IsRevert, opts.BugOptions)
		isBug := tier != TierNone
		isChaos := tier == TierChaos

		touched := map[string]struct{}{}
		for _, f := range c.FilesChanged {
			if isExcluded(f.Path) {
				continue
			}
			mod := moduleNameFor(f.Path, depth)
			touched[mod] = struct{}{}
		}
		for mod := range touched {
			s := getOrNew(mod)
			s.commits[c.Hash] = struct{}{}
			s.authors[c.AuthorEmail] = struct{}{}
			if isBug {
				s.bugCommits++
			}
			if isChaos {
				s.chaosCommits++
			}
			for _, f := range c.FilesChanged {
				if isExcluded(f.Path) {
					continue
				}
				if moduleNameFor(f.Path, depth) != mod {
					continue
				}
				total := f.Added + f.Removed
				s.linesChanged += total
				s.fileChurn[f.Path] += total
			}
		}
	}

	totalLinesChanged := 0
	for _, s := range stats {
		totalLinesChanged += s.linesChanged
	}

	modules := []types.ModuleEntry{}
	for name, s := range stats {
		if len(s.commits) < minCommits {
			continue
		}
		commits := len(s.commits)
		bugRatio := float64(s.bugCommits) / float64(commits)
		chaosRatio := float64(s.chaosCommits) / float64(commits)
		loc := s.linesChanged

		share := 0.0
		if totalLinesChanged > 0 {
			share = float64(loc) / float64(totalLinesChanged)
		}
		shareScore := math.Min(100, share*300)

		authorConcentrationScore := 0.0
		if len(s.authors) == 1 {
			authorConcentrationScore = 100
		} else {
			authorConcentrationScore = math.Max(0, 100-float64(len(s.authors)-1)*20)
		}

		weightedBug := (chaosRatio*1.0 + (bugRatio-chaosRatio)*0.4) * 100
		bugSubScore := math.Min(100, weightedBug*2)

		score := int(math.Round(
			shareScore*0.3 +
				bugSubScore*0.4 +
				authorConcentrationScore*0.15 +
				math.Min(100, float64(loc)/500)*0.15,
		))

		// Top churned file
		var topFile string
		topChurn := 0
		for path, churn := range s.fileChurn {
			if churn > topChurn {
				topChurn = churn
				topFile = path
			}
		}

		// Owner aggregation
		filePaths := make([]string, 0, len(s.fileChurn))
		for p := range s.fileChurn {
			filePaths = append(filePaths, p)
		}
		owners := codeowners.AggregateOwnersForModule(filePaths, opts.Codeowners, 2)
		if owners == nil {
			owners = []string{}
		}

		modules = append(modules, types.ModuleEntry{
			Name:         name,
			Score:        minInt(100, score),
			Mood:         moodFromScore(score),
			Commits:      commits,
			LinesChanged: loc,
			BugRatio:     round3(bugRatio),
			Authors:      len(s.authors),
			TopFile:      topFile,
			Owners:       owners,
		})
	}
	sort.Slice(modules, func(i, j int) bool { return modules[i].Score > modules[j].Score })
	return types.ModuleSignal{Type: "modules", Modules: modules}
}

func moduleNameFor(path string, depth int) string {
	normalized := strings.ReplaceAll(path, "\\", "/")
	parts := []string{}
	for _, p := range strings.Split(normalized, "/") {
		if p != "" {
			parts = append(parts, p)
		}
	}
	if len(parts) == 0 || len(parts) == 1 {
		return "(root)"
	}
	depthUsed := depth
	if depthUsed > len(parts)-1 {
		depthUsed = len(parts) - 1
	}
	return strings.Join(parts[:depthUsed], "/")
}

func moodFromScore(score int) types.MoodLabel {
	if score <= 40 {
		return types.MoodCalm
	}
	if score <= 70 {
		return types.MoodAnxious
	}
	return types.MoodChaotic
}

func minInt(a, b int) int {
	if a < b {
		return a
	}
	return b
}

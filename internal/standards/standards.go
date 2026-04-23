package standards

import (
	"math"
	"regexp"

	"repopulse/internal/types"
)

// Options controls per-signal knobs. Fields are all optional — any
// zero-value falls back to sensible defaults.
type Options struct {
	// CommitPattern is the effective regex for the compliance signal.
	// Pass nil to use the built-in Conventional Commits default.
	CommitPattern *regexp.Regexp
	// CommitPatternSource is the raw pattern string used for the UI
	// subtitle ("pattern: `<regex>`"). Useful when the team has
	// overridden the default. Empty string → UI shows the default-pattern copy.
	CommitPatternSource string
}

// Compute runs every Plank-2 deterministic standard against the inputs
// and returns the combined result. Adding a new standard means: define
// the result type in types.go, write the computation, and call it here.
func Compute(commits []types.CommitRecord, allFiles []string, opts Options) types.StandardsSignal {
	pattern := opts.CommitPattern
	if pattern == nil {
		// Lazy fallback so callers don't have to import config just to
		// get the default regex. Config is the canonical owner.
		pattern = regexp.MustCompile(defaultCommitPattern)
	}
	cc := ComputeCommitCompliance(commits, pattern)
	cc.Pattern = opts.CommitPatternSource
	return types.StandardsSignal{
		Type:                "standards",
		ConventionalCommits: cc,
		TestDensity:         ComputeTestDensity(allFiles),
	}
}

// Duplicated here (instead of importing internal/config) to keep the
// dependency graph flat — standards is below config in the logical
// layering; callers resolve the pattern before calling Compute.
const defaultCommitPattern = `^(?i)(feat|feature|fix|chore|docs?|style|tests?|refactor|ci|build|perf|revert)(\([^)]*\))?!?:\s+\S`

func pct(part, whole int) float64 {
	if whole <= 0 {
		return 0
	}
	return float64(part) / float64(whole) * 100
}

func round1(x float64) float64 {
	return math.Round(x*10) / 10
}

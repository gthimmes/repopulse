package standards

import (
	"math"

	"repopulse/internal/types"
)

// Compute runs every Plank-2 deterministic standard against the inputs
// and returns the combined result. Adding a new standard means: define
// the result type in types.go, write the computation, and call it here.
func Compute(commits []types.CommitRecord, allFiles []string) types.StandardsSignal {
	return types.StandardsSignal{
		Type:                "standards",
		ConventionalCommits: ComputeConventionalCommits(commits),
		TestDensity:         ComputeTestDensity(allFiles),
	}
}

func pct(part, whole int) float64 {
	if whole <= 0 {
		return 0
	}
	return float64(part) / float64(whole) * 100
}

func round1(x float64) float64 {
	return math.Round(x*10) / 10
}

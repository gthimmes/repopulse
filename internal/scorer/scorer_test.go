package scorer

import (
	"testing"

	"mood-ring/internal/types"
)

func makeFreq(score int) types.FrequencySignal {
	return types.FrequencySignal{Type: "commitFrequency", Score: score, DailyBuckets: []types.DayBucket{}, Mean: 1, StdDev: 0.5, LongestGapDays: 2}
}
func makeChurn(score int) types.ChurnSignal {
	return types.ChurnSignal{Type: "fileChurn", Score: score, TopChurners: []types.ChurnEntry{}, TotalFilesTouched: 10, EligibleFileCount: 8, HighChurnFileCount: 1, TotalLinesChanged: 100, LinesPerDay: 10}
}
func makeBug(score int) types.BugSignal {
	return types.BugSignal{
		Type: "bugRatio", Score: score,
		BugCommitCount: 2, ChaosCommitCount: 0, RoutineFixCount: 0, NormalFixCount: 2,
		TotalCommits: 20, Ratio: 0.1, LongestFixStreak: 1,
		BugCommitsByDay: []types.DayBucket{}, NormalCommitsByDay: []types.DayBucket{}, ChaosCommitsByDay: []types.DayBucket{},
		ClassifiedSamples: types.BugClassifiedGroups{
			Chaos: []types.BugClassifiedCommit{}, Normal: []types.BugClassifiedCommit{}, Routine: []types.BugClassifiedCommit{},
		},
	}
}
func makeModules() types.ModuleSignal { return types.ModuleSignal{Type: "modules", Modules: []types.ModuleEntry{}} }
func makeHotspots() types.HotspotSignal {
	return types.HotspotSignal{Type: "hotspots", Hotspots: []types.HotspotEntry{}}
}
func makeAuthors(score int) types.AuthorSignal {
	return types.AuthorSignal{
		Type: "authors", Score: score, TotalAuthors: 3,
		WeekendNightPct: 0, BusFactorTop1Pct: 33, BusFactorTop3Pct: 100,
		NewContributorChurnPct: 0, TopAuthors: []types.AuthorEntry{},
	}
}

func TestComputeMood_Calm(t *testing.T) {
	r := ComputeMood(Input{
		CommitFrequency: makeFreq(10), FileChurn: makeChurn(10), BugRatio: makeBug(10),
		Modules: makeModules(), Hotspots: makeHotspots(), Authors: makeAuthors(10),
	})
	if r.Mood != types.MoodCalm {
		t.Errorf("want calm, got %s", r.Mood)
	}
	if r.CompositeScore > 40 {
		t.Errorf("want composite <= 40, got %d", r.CompositeScore)
	}
}

func TestComputeMood_Chaotic(t *testing.T) {
	r := ComputeMood(Input{
		CommitFrequency: makeFreq(90), FileChurn: makeChurn(90), BugRatio: makeBug(90),
		Modules: makeModules(), Hotspots: makeHotspots(), Authors: makeAuthors(90),
	})
	if r.Mood != types.MoodChaotic {
		t.Errorf("want chaotic, got %s", r.Mood)
	}
	if r.CompositeScore <= 70 {
		t.Errorf("want composite > 70, got %d", r.CompositeScore)
	}
}

func TestWeightsSumToOne(t *testing.T) {
	total := Weights.CommitFrequency + Weights.FileChurn + Weights.BugRatio + Weights.Coverage + Weights.Authors
	if total < 0.999 || total > 1.001 {
		t.Errorf("weights should sum to 1.0, got %g", total)
	}
}

func TestCoverageRedistribution(t *testing.T) {
	cov := &types.CoverageSignal{Type: "coverage", Score: 50, Percentage: 60, Source: "istanbul"}
	withCov := ComputeMood(Input{
		CommitFrequency: makeFreq(50), FileChurn: makeChurn(50), BugRatio: makeBug(50),
		Coverage: cov, Modules: makeModules(), Hotspots: makeHotspots(), Authors: makeAuthors(50),
	})
	withoutCov := ComputeMood(Input{
		CommitFrequency: makeFreq(50), FileChurn: makeChurn(50), BugRatio: makeBug(50),
		Coverage: nil, Modules: makeModules(), Hotspots: makeHotspots(), Authors: makeAuthors(50),
	})
	if withCov.CompositeScore != withoutCov.CompositeScore {
		t.Errorf("equal sub-scores should produce equal composite: %d vs %d", withCov.CompositeScore, withoutCov.CompositeScore)
	}
}

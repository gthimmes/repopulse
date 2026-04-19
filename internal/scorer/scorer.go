// Package scorer combines signals into a MoodResult.
package scorer

import (
	"math"

	"repopulse/internal/types"
)

var Weights = struct {
	CommitFrequency float64
	FileChurn       float64
	BugRatio        float64
	Coverage        float64
	Authors         float64
}{
	CommitFrequency: 0.15,
	FileChurn:       0.25,
	BugRatio:        0.30,
	Coverage:        0.10,
	Authors:         0.20,
}

type Input struct {
	CommitFrequency types.FrequencySignal
	FileChurn       types.ChurnSignal
	BugRatio        types.BugSignal
	Coverage        *types.CoverageSignal
	Modules         types.ModuleSignal
	Hotspots        types.HotspotSignal
	Authors         types.AuthorSignal
	Narrative       []types.NarrativeBullet
	RollingTimeline []types.RollingPoint
}

func ComputeMood(in Input) types.MoodResult {
	wFreq := Weights.CommitFrequency
	wChurn := Weights.FileChurn
	wBug := Weights.BugRatio
	wCov := Weights.Coverage
	wAuth := Weights.Authors

	if in.Coverage == nil {
		wBug += wCov
		wCov = 0
	}

	var weighted float64
	weighted += float64(in.CommitFrequency.Score) * wFreq
	weighted += float64(in.FileChurn.Score) * wChurn
	weighted += float64(in.BugRatio.Score) * wBug
	weighted += float64(in.Authors.Score) * wAuth
	if in.Coverage != nil {
		weighted += float64(in.Coverage.Score) * wCov
	}
	totalWeight := wFreq + wChurn + wBug + wAuth
	if in.Coverage != nil {
		totalWeight += wCov
	}
	composite := int(math.Round(weighted / totalWeight))

	breakdown := types.MoodBreakdown{
		CommitFrequency: int(math.Round(float64(in.CommitFrequency.Score) * wFreq)),
		FileChurn:       int(math.Round(float64(in.FileChurn.Score) * wChurn)),
		BugRatio:        int(math.Round(float64(in.BugRatio.Score) * wBug)),
		Authors:         int(math.Round(float64(in.Authors.Score) * wAuth)),
	}
	if in.Coverage != nil {
		cov := int(math.Round(float64(in.Coverage.Score) * wCov))
		breakdown.Coverage = &cov
	}

	narrative := in.Narrative
	if narrative == nil {
		narrative = []types.NarrativeBullet{}
	}
	timeline := in.RollingTimeline
	if timeline == nil {
		timeline = []types.RollingPoint{}
	}

	return types.MoodResult{
		Mood:           moodFromScore(composite),
		CompositeScore: composite,
		Breakdown:      breakdown,
		Signals: types.Signals{
			CommitFrequency: in.CommitFrequency,
			FileChurn:       in.FileChurn,
			BugRatio:        in.BugRatio,
			Coverage:        in.Coverage,
			Modules:         in.Modules,
			Hotspots:        in.Hotspots,
			Authors:         in.Authors,
		},
		Narrative:       narrative,
		RollingTimeline: timeline,
	}
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

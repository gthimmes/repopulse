// Package compare handles JSON snapshots and delta computation.
package compare

import (
	"encoding/json"
	"os"

	"repopulse/internal/types"
)

type ReportSnapshot struct {
	GeneratedAt     string           `json:"generatedAt"`
	RepoName        string           `json:"repoName"`
	WindowDays      int              `json:"windowDays"`
	AnalyzedCommits int              `json:"analyzedCommits"`
	MoodResult      types.MoodResult `json:"moodResult"`
}

func LoadSnapshot(path string) (*ReportSnapshot, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	var s ReportSnapshot
	if err := json.Unmarshal(data, &s); err != nil {
		return nil, err
	}
	return &s, nil
}

func BuildDelta(current, previous types.MoodResult, previousAt string) types.MoodDelta {
	d := types.MoodDelta{
		Composite: current.CompositeScore - previous.CompositeScore,
		Breakdown: map[string]int{
			"commitFrequency": current.Breakdown.CommitFrequency - previous.Breakdown.CommitFrequency,
			"fileChurn":       current.Breakdown.FileChurn - previous.Breakdown.FileChurn,
			"bugRatio":        current.Breakdown.BugRatio - previous.Breakdown.BugRatio,
			"authors":         current.Breakdown.Authors - previous.Breakdown.Authors,
		},
		PreviousAt: previousAt,
	}
	if current.Breakdown.Coverage != nil && previous.Breakdown.Coverage != nil {
		d.Breakdown["coverage"] = *current.Breakdown.Coverage - *previous.Breakdown.Coverage
	}
	return d
}

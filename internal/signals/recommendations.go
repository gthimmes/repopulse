package signals

import (
	"fmt"
	"math"
	"sort"
	"strings"
	"time"

	"repopulse/internal/types"
)

// BuildRecommendations produces 0–N recommendations for a hotspot.
func BuildRecommendations(
	h types.HotspotEntry,
	churnEntry *types.ChurnEntry,
	windowEnd time.Time,
	maxRecommendations int,
) []types.HotspotRecommendation {
	if maxRecommendations < 1 {
		maxRecommendations = 3
	}
	recs := []types.HotspotRecommendation{}

	// 1. bus-factor
	if len(h.TopAuthorsOfFile) > 0 && h.TotalCommits > 0 {
		topPct := float64(h.TopAuthorsOfFile[0].Commits) / float64(h.TotalCommits) * 100
		if topPct >= 60 {
			sev := "warn"
			if topPct >= 80 {
				sev = "alert"
			}
			recs = append(recs, types.HotspotRecommendation{
				Kind:     "bus-factor",
				Severity: sev,
				Text: fmt.Sprintf(
					"Bus-factor risk: %s has %d%% of commits to this file. Pair a second engineer in before they go on vacation.",
					h.TopAuthorsOfFile[0].Name, int(math.Round(topPct)),
				),
			})
		}
	}

	// 2. chaos-repeat
	if h.ChaosTouches >= 3 {
		sev := "warn"
		if h.ChaosTouches >= 5 {
			sev = "alert"
		}
		recs = append(recs, types.HotspotRecommendation{
			Kind:     "chaos-repeat",
			Severity: sev,
			Text: fmt.Sprintf(
				"%d chaos-tier commits (reverts / hotfixes) in this window. Add a pre-merge integration test or put behind a feature flag before the next change.",
				h.ChaosTouches,
			),
		})
	}

	// 3. rewritten
	if churnEntry != nil && churnEntry.Rewritten {
		recs = append(recs, types.HotspotRecommendation{
			Kind:     "rewritten",
			Severity: "info",
			Text:     "File was effectively rewritten (churn > 5× current size). Confirm the new structure is documented and reviewed end-to-end, not just diffed.",
		})
	}

	// 4. unowned
	if len(h.Owners) == 0 {
		sev := "info"
		if h.ChaosTouches > 0 {
			sev = "warn"
		}
		recs = append(recs, types.HotspotRecommendation{
			Kind:     "unowned",
			Severity: sev,
			Text:     "No CODEOWNERS entry for this file. Assign a team so reviews don't fall through the cracks — especially given its bug history.",
		})
	}

	// 5. multi-owner
	if len(h.Owners) >= 2 {
		recs = append(recs, types.HotspotRecommendation{
			Kind:     "multi-owner",
			Severity: "info",
			Text: fmt.Sprintf(
				"Ownership split across %s. Pick a primary owner; shared ownership usually means nobody owns it.",
				strings.Join(h.Owners, ", "),
			),
		})
	}

	// 6. stale-buggy
	if h.LastTouched != "" && h.BugTouches > 0 {
		lastDate, err := time.Parse("2006-01-02", h.LastTouched)
		if err == nil {
			daysSince := windowEnd.Sub(lastDate).Hours() / 24
			if daysSince >= 30 {
				recs = append(recs, types.HotspotRecommendation{
					Kind:     "stale-buggy",
					Severity: "info",
					Text: fmt.Sprintf(
						"Last touched %d days ago but has a bug history. Schedule a maintenance pass before someone hits a latent issue.",
						int(math.Round(daysSince)),
					),
				})
			}
		}
	}

	// 7. bug-heavy
	if h.TotalCommits >= 5 {
		bugRatio := h.BugTouches / float64(h.TotalCommits)
		if bugRatio >= 0.4 {
			sev := "info"
			if bugRatio >= 0.6 {
				sev = "warn"
			}
			recs = append(recs, types.HotspotRecommendation{
				Kind:     "bug-heavy",
				Severity: sev,
				Text: fmt.Sprintf(
					"%d%% of recent commits are bug-related. This file is in firefighting mode — consider a targeted refactor rather than more patches.",
					int(math.Round(bugRatio*100)),
				),
			})
		}
	}

	order := map[string]int{"alert": 0, "warn": 1, "info": 2}
	sort.SliceStable(recs, func(i, j int) bool {
		return order[recs[i].Severity] < order[recs[j].Severity]
	})
	if len(recs) > maxRecommendations {
		recs = recs[:maxRecommendations]
	}
	return recs
}

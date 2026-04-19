package render

import (
	"strings"
	"testing"
	"time"

	"repopulse/internal/types"
)

func makeHotspot(overrides func(*types.HotspotEntry)) types.HotspotEntry {
	h := types.HotspotEntry{
		Path:             "src/x.ts",
		ChurnRank:        1,
		BugTouches:       3,
		ChaosTouches:     1,
		TotalCommits:     10,
		HotspotScore:     70,
		Authors:          2,
		LastTouched:      "2026-04-10",
		TopAuthorsOfFile: []types.HotspotFileAuthor{{Name: "Alice", Commits: 6}},
		RecentBugCommits: []types.HotspotCommit{},
		Owners:           []string{},
		Recommendations:  []types.HotspotRecommendation{},
	}
	if overrides != nil {
		overrides(&h)
	}
	return h
}

func baseMoodResult() types.MoodResult {
	return types.MoodResult{
		Mood:           types.MoodAnxious,
		CompositeScore: 57,
		Breakdown:      types.MoodBreakdown{CommitFrequency: 5, FileChurn: 18, BugRatio: 20, Authors: 14},
		Signals: types.Signals{
			CommitFrequency: types.FrequencySignal{Type: "commitFrequency", Score: 52, DailyBuckets: []types.DayBucket{}, Mean: 5, StdDev: 1, LongestGapDays: 1},
			FileChurn:       types.ChurnSignal{Type: "fileChurn", Score: 60, TopChurners: []types.ChurnEntry{}, TotalFilesTouched: 10, EligibleFileCount: 8, HighChurnFileCount: 1, TotalLinesChanged: 100, LinesPerDay: 10},
			BugRatio: types.BugSignal{
				Type: "bugRatio", Score: 50,
				BugCommitCount: 3, ChaosCommitCount: 1, RoutineFixCount: 0, NormalFixCount: 2,
				TotalCommits: 10, Ratio: 0.3, LongestFixStreak: 1,
				BugCommitsByDay: []types.DayBucket{}, NormalCommitsByDay: []types.DayBucket{}, ChaosCommitsByDay: []types.DayBucket{},
				ClassifiedSamples: types.BugClassifiedGroups{
					Chaos: []types.BugClassifiedCommit{}, Normal: []types.BugClassifiedCommit{}, Routine: []types.BugClassifiedCommit{},
				},
			},
			Modules:  types.ModuleSignal{Type: "modules", Modules: []types.ModuleEntry{}},
			Hotspots: types.HotspotSignal{Type: "hotspots", Hotspots: []types.HotspotEntry{}},
			Authors: types.AuthorSignal{
				Type: "authors", Score: 40, TotalAuthors: 3,
				WeekendNightPct: 0, BusFactorTop1Pct: 33, BusFactorTop3Pct: 100,
				NewContributorChurnPct: 0, Contributors: []types.AuthorEntry{},
			},
		},
		Narrative:       []types.NarrativeBullet{},
		RollingTimeline: []types.RollingPoint{},
	}
}

var testMeta = types.RepoMeta{
	RepoName:        "test-repo",
	RepoPath:        "/tmp/test",
	AnalyzedCommits: 120,
	WindowDays:      90,
	WindowStart:     time.Date(2026, 1, 15, 0, 0, 0, 0, time.UTC),
	WindowEnd:       time.Date(2026, 4, 15, 0, 0, 0, 0, time.UTC),
	GeneratedAt:     time.Date(2026, 4, 15, 12, 0, 0, 0, time.UTC),
}

func TestMarkdown_Header(t *testing.T) {
	md := RenderMarkdown(baseMoodResult(), testMeta, nil, MarkdownOptions{})
	if !strings.Contains(md, "## Repopulse: test-repo") {
		t.Error("header missing")
	}
	if !strings.Contains(md, "Anxious") {
		t.Error("mood label missing")
	}
}

func TestMarkdown_ScoreLine(t *testing.T) {
	md := RenderMarkdown(baseMoodResult(), testMeta, nil, MarkdownOptions{})
	for _, want := range []string{"**Score: 57/100**", "120 commits", "2026-01-15", "2026-04-15"} {
		if !strings.Contains(md, want) {
			t.Errorf("missing %q", want)
		}
	}
}

func TestMarkdown_DeltaIncluded(t *testing.T) {
	md := RenderMarkdown(baseMoodResult(), testMeta, &types.MoodDelta{Composite: 8}, MarkdownOptions{})
	if !strings.Contains(md, "+8") {
		t.Errorf("expected +8 in delta:\n%s", md)
	}
}

func TestMarkdown_FindingsRendered(t *testing.T) {
	data := baseMoodResult()
	data.Narrative = []types.NarrativeBullet{
		{Kind: "alert", Text: "Bad thing"},
		{Kind: "warn", Text: "Cautionary thing"},
	}
	md := RenderMarkdown(data, testMeta, nil, MarkdownOptions{})
	for _, want := range []string{"### Findings", "Bad thing", "Cautionary thing"} {
		if !strings.Contains(md, want) {
			t.Errorf("missing %q", want)
		}
	}
}

func TestMarkdown_HotspotsTable(t *testing.T) {
	data := baseMoodResult()
	data.Signals.Hotspots.Hotspots = []types.HotspotEntry{
		makeHotspot(func(h *types.HotspotEntry) {
			h.Path = "src/a.ts"
			h.Owners = []string{"@team/payments"}
			h.HotspotScore = 90
		}),
		makeHotspot(func(h *types.HotspotEntry) {
			h.Path = "src/b.ts"
			h.HotspotScore = 60
		}),
	}
	md := RenderMarkdown(data, testMeta, nil, MarkdownOptions{})
	for _, want := range []string{"### Hotspots", "| Score | File | Team | Chaos | Bug-touches |", "`src/a.ts`", "`@team/payments`", "_(unowned)_"} {
		if !strings.Contains(md, want) {
			t.Errorf("missing %q", want)
		}
	}
}

func TestMarkdown_HotspotsCapped(t *testing.T) {
	data := baseMoodResult()
	many := make([]types.HotspotEntry, 10)
	for i := 0; i < 10; i++ {
		p := "src/f" + string(rune('0'+i)) + ".ts"
		many[i] = makeHotspot(func(h *types.HotspotEntry) { h.Path = p })
	}
	data.Signals.Hotspots.Hotspots = many
	md := RenderMarkdown(data, testMeta, nil, MarkdownOptions{TopHotspots: 3})
	if !strings.Contains(md, "`src/f0.ts`") || !strings.Contains(md, "`src/f2.ts`") {
		t.Error("top 3 not present")
	}
	if strings.Contains(md, "`src/f3.ts`") {
		t.Error("beyond top 3 should be excluded")
	}
}

func TestMarkdown_TopRecommendations_SortedBySeverity(t *testing.T) {
	data := baseMoodResult()
	data.Signals.Hotspots.Hotspots = []types.HotspotEntry{
		makeHotspot(func(h *types.HotspotEntry) {
			h.Path = "src/a.ts"
			h.Recommendations = []types.HotspotRecommendation{{Kind: "bug-heavy", Severity: "info", Text: "info text here"}}
		}),
		makeHotspot(func(h *types.HotspotEntry) {
			h.Path = "src/b.ts"
			h.Recommendations = []types.HotspotRecommendation{{Kind: "chaos-repeat", Severity: "alert", Text: "chaos alert text"}}
		}),
	}
	md := RenderMarkdown(data, testMeta, nil, MarkdownOptions{TopRecommendations: 3})
	if !strings.Contains(md, "### Top recommendations") {
		t.Error("section missing")
	}
	if !strings.Contains(md, "**chaos-repeat**") {
		t.Error("chaos-repeat missing")
	}
	if !strings.Contains(md, "`src/b.ts`") {
		t.Error("path missing")
	}
	alertIdx := strings.Index(md, "chaos alert text")
	infoIdx := strings.Index(md, "info text here")
	if alertIdx < 0 || infoIdx < 0 {
		t.Fatal("both recs must be present")
	}
	if infoIdx < alertIdx {
		t.Error("alert must precede info in output")
	}
}

func TestMarkdown_OmitsEmptySections(t *testing.T) {
	md := RenderMarkdown(baseMoodResult(), testMeta, nil, MarkdownOptions{})
	for _, notWant := range []string{"### Findings", "### Hotspots", "### Top recommendations"} {
		if strings.Contains(md, notWant) {
			t.Errorf("should not contain %q", notWant)
		}
	}
	if !strings.Contains(md, "## Repopulse") || !strings.Contains(md, "Generated by repopulse") {
		t.Error("header or footer missing")
	}
}

func TestMarkdown_Footer(t *testing.T) {
	md := RenderMarkdown(baseMoodResult(), testMeta, nil, MarkdownOptions{})
	if !strings.Contains(md, "_Generated by repopulse on 2026-04-15_") {
		t.Error("footer date missing")
	}
}

func TestMarkdown_HTMLLink(t *testing.T) {
	md := RenderMarkdown(baseMoodResult(), testMeta, nil, MarkdownOptions{HTMLReportPath: "/tmp/report.html"})
	if !strings.Contains(md, "[full HTML report](/tmp/report.html)") {
		t.Error("HTML link missing")
	}
}

func TestMarkdown_OnlyHotModules(t *testing.T) {
	data := baseMoodResult()
	data.Signals.Modules.Modules = []types.ModuleEntry{
		{Name: "payments", Score: 75, Mood: types.MoodChaotic, Commits: 20, LinesChanged: 900, BugRatio: 0.4, Authors: 3, TopFile: "a", Owners: []string{"@org/pay"}},
		{Name: "utils", Score: 22, Mood: types.MoodCalm, Commits: 5, LinesChanged: 100, BugRatio: 0.05, Authors: 2, TopFile: "u", Owners: []string{}},
	}
	md := RenderMarkdown(data, testMeta, nil, MarkdownOptions{})
	if !strings.Contains(md, "payments") {
		t.Error("hot module missing")
	}
	if strings.Contains(md, "| utils |") {
		t.Error("calm module should be excluded")
	}
}

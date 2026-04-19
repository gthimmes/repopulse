package fixtures

import (
	"time"

	"repopulse/internal/types"
)

// UIHotspots — deterministic hotspot data used by the Playwright UI tests.
// Lives in Go so the fixture generator is language-homogeneous with the rest
// of the codebase. Any changes here must stay in sync with the assertions in
// tests/e2e/*.spec.ts (selectors, chip counts, row counts).
func UIHotspots() []types.HotspotEntry {
	return []types.HotspotEntry{
		{
			Path:         "src/payments/ledger.ts",
			ChurnRank:    1,
			BugTouches:   9.6,
			ChaosTouches: 3,
			TotalCommits: 22,
			HotspotScore: 88,
			Authors:      5,
			LastTouched:  "2026-04-12",
			Owners:       []string{"@org/payments-team"},
			TopAuthorsOfFile: []types.HotspotFileAuthor{
				{Name: "Alice Chen", Commits: 11},
				{Name: "Bob Martinez", Commits: 6},
				{Name: "Carol Park", Commits: 3},
			},
			RecentBugCommits: []types.HotspotCommit{
				{Hash: "a1b2c3d", Date: "2026-04-12", Author: "Alice Chen", Message: "fix: incorrect rounding when currency has 3 decimal places", Tier: "chaos"},
				{Hash: "e4f5g6h", Date: "2026-04-10", Author: "Bob Martinez", Message: `Revert "refactor: extract fee calculator"`, Tier: "chaos"},
				{Hash: "i7j8k9l", Date: "2026-04-07", Author: "Alice Chen", Message: "hotfix: race on concurrent ledger writes", Tier: "chaos"},
				{Hash: "m0n1o2p", Date: "2026-04-02", Author: "Carol Park", Message: "fix: handle null tax jurisdiction", Tier: "normal"},
				{Hash: "q3r4s5t", Date: "2026-03-28", Author: "Alice Chen", Message: "fix: precision loss in multi-currency aggregate", Tier: "normal"},
				{Hash: "u6v7w8x", Date: "2026-03-20", Author: "Bob Martinez", Message: "chore(fix): typo in ledger comment", Tier: "routine"},
			},
			Recommendations: []types.HotspotRecommendation{
				{Kind: "chaos-repeat", Severity: "warn", Text: "3 chaos-tier commits (reverts / hotfixes) in this window. Add a pre-merge integration test or put behind a feature flag before the next change."},
				{Kind: "bug-heavy", Severity: "warn", Text: "44% of recent commits are bug-related. This file is in firefighting mode — consider a targeted refactor rather than more patches."},
			},
		},
		{
			Path:         "src/auth/session.ts",
			ChurnRank:    4,
			BugTouches:   4.2,
			ChaosTouches: 1,
			TotalCommits: 14,
			HotspotScore: 62,
			Authors:      3,
			LastTouched:  "2026-04-15",
			Owners:       []string{"@org/security", "@org/platform"},
			TopAuthorsOfFile: []types.HotspotFileAuthor{
				{Name: "Dana Rivera", Commits: 9},
				{Name: "Alice Chen", Commits: 4},
			},
			RecentBugCommits: []types.HotspotCommit{
				{Hash: "aa11bb2", Date: "2026-04-15", Author: "Dana Rivera", Message: "fix: expired session not clearing cookie", Tier: "normal"},
				{Hash: "cc33dd4", Date: "2026-04-01", Author: "Dana Rivera", Message: "hotfix: CSRF token reuse across tabs", Tier: "chaos"},
				{Hash: "ee55ff6", Date: "2026-03-22", Author: "Alice Chen", Message: "fix: session refresh misses offset", Tier: "normal"},
			},
			Recommendations: []types.HotspotRecommendation{
				{Kind: "bus-factor", Severity: "warn", Text: "Bus-factor risk: Dana Rivera has 64% of commits to this file. Pair a second engineer in before they go on vacation."},
				{Kind: "multi-owner", Severity: "info", Text: "Ownership split across @org/security, @org/platform. Pick a primary owner; shared ownership usually means nobody owns it."},
			},
		},
		{
			Path:         "src/api/rate-limiter.ts",
			ChurnRank:    9,
			BugTouches:   1.1,
			ChaosTouches: 0,
			TotalCommits: 5,
			HotspotScore: 34,
			Authors:      2,
			LastTouched:  "2026-03-30",
			Owners:       []string{}, // unowned — exercises the fallback path
			TopAuthorsOfFile: []types.HotspotFileAuthor{
				{Name: "Evan Lee", Commits: 4},
				{Name: "Alice Chen", Commits: 1},
			},
			RecentBugCommits: []types.HotspotCommit{
				{Hash: "99aa88b", Date: "2026-03-30", Author: "Evan Lee", Message: "fix: off-by-one in window bucket rollover", Tier: "normal"},
			},
			Recommendations: []types.HotspotRecommendation{
				{Kind: "unowned", Severity: "info", Text: "No CODEOWNERS entry for this file. Assign a team so reviews don't fall through the cracks — especially given its bug history."},
			},
		},
	}
}

// UIMoodResult — deterministic MoodResult for the Playwright UI tests.
func UIMoodResult() types.MoodResult {
	hotspots := UIHotspots()
	return types.MoodResult{
		Mood:           types.MoodAnxious,
		CompositeScore: 57,
		Breakdown: types.MoodBreakdown{
			CommitFrequency: 8,
			FileChurn:       18,
			BugRatio:        19,
			Authors:         12,
		},
		Signals: types.Signals{
			CommitFrequency: types.FrequencySignal{
				Type:  "commitFrequency",
				Score: 52,
				DailyBuckets: []types.DayBucket{
					{Date: "2026-04-10", Count: 6},
					{Date: "2026-04-11", Count: 4},
					{Date: "2026-04-12", Count: 8},
					{Date: "2026-04-13", Count: 5},
					{Date: "2026-04-14", Count: 7},
				},
				Mean:           6,
				StdDev:         1.4,
				LongestGapDays: 1,
			},
			FileChurn: types.ChurnSignal{
				Type:  "fileChurn",
				Score: 60,
				TopChurners: []types.ChurnEntry{
					{Path: "src/payments/ledger.ts", Added: 420, Removed: 180, Ratio: 2.4, Rewritten: false},
					{Path: "src/auth/session.ts", Added: 220, Removed: 90, Ratio: 1.3, Rewritten: false},
				},
				TotalFilesTouched:  48,
				EligibleFileCount:  32,
				HighChurnFileCount: 3,
				TotalLinesChanged:  4800,
				LinesPerDay:        480,
			},
			BugRatio: types.BugSignal{
				Type:             "bugRatio",
				Score:            64,
				BugCommitCount:   38,
				ChaosCommitCount: 6,
				RoutineFixCount:  5,
				NormalFixCount:   27,
				TotalCommits:     120,
				Ratio:            0.317,
				LongestFixStreak: 4,
				BugCommitsByDay: []types.DayBucket{
					{Date: "2026-04-10", Count: 2},
					{Date: "2026-04-11", Count: 1},
					{Date: "2026-04-12", Count: 3},
				},
				NormalCommitsByDay: []types.DayBucket{
					{Date: "2026-04-10", Count: 4},
					{Date: "2026-04-11", Count: 3},
					{Date: "2026-04-12", Count: 5},
				},
				ChaosCommitsByDay: []types.DayBucket{
					{Date: "2026-04-12", Count: 1},
				},
				RevertedWithin7d: 1,
				ClassifiedSamples: types.BugClassifiedGroups{
					Chaos: []types.BugClassifiedCommit{
						{Hash: "a1b2c3d", Date: "2026-04-12", Author: "Alice Chen", Message: "hotfix: race on concurrent ledger writes", MatchedKeyword: "hotfix"},
						{Hash: "e4f5g6h", Date: "2026-04-10", Author: "Bob Martinez", Message: `Revert "refactor: extract fee calculator"`, MatchedKeyword: "(revert)"},
						{Hash: "cc33dd4", Date: "2026-04-01", Author: "Dana Rivera", Message: "hotfix: CSRF token reuse across tabs", MatchedKeyword: "hotfix"},
					},
					Normal: []types.BugClassifiedCommit{
						{Hash: "aa11bb2", Date: "2026-04-15", Author: "Dana Rivera", Message: "fix: expired session not clearing cookie", MatchedKeyword: "fix"},
						{Hash: "m0n1o2p", Date: "2026-04-02", Author: "Carol Park", Message: "fix: handle null tax jurisdiction", MatchedKeyword: "fix"},
						{Hash: "q3r4s5t", Date: "2026-03-28", Author: "Alice Chen", Message: "fix: precision loss in multi-currency aggregate", MatchedKeyword: "fix"},
						{Hash: "ee55ff6", Date: "2026-03-22", Author: "Alice Chen", Message: "bug: session refresh misses offset", MatchedKeyword: "bug"},
					},
					Routine: []types.BugClassifiedCommit{
						{Hash: "u6v7w8x", Date: "2026-03-20", Author: "Bob Martinez", Message: "chore(fix): typo in ledger comment", MatchedKeyword: "typo"},
						{Hash: "5h6g7f8", Date: "2026-03-15", Author: "Alice Chen", Message: "lint: unused import", MatchedKeyword: "lint"},
					},
				},
			},
			Coverage: nil,
			Modules: types.ModuleSignal{
				Type: "modules",
				Modules: []types.ModuleEntry{
					{Name: "payments", Score: 72, Mood: types.MoodChaotic, Commits: 34, LinesChanged: 1800, BugRatio: 0.42, Authors: 5, TopFile: "src/payments/ledger.ts", Owners: []string{"@org/payments-team"}},
					{Name: "auth", Score: 48, Mood: types.MoodAnxious, Commits: 18, LinesChanged: 520, BugRatio: 0.28, Authors: 3, TopFile: "src/auth/session.ts", Owners: []string{"@org/security"}},
				},
			},
			Hotspots: types.HotspotSignal{Type: "hotspots", Hotspots: hotspots},
			Authors: types.AuthorSignal{
				Type:                   "authors",
				Score:                  44,
				TotalAuthors:           7,
				WeekendNightPct:        12.5,
				BusFactorTop1Pct:       32,
				BusFactorTop3Pct:       71,
				NewContributorChurnPct: 8.3,
				TopAuthors: []types.AuthorEntry{
					{Name: "Alice Chen", Email: "alice@example.com", Commits: 28, LinesChanged: 1400, WeekendNightCommits: 4, FirstSeen: "2026-01-22", IsNew: false},
					{Name: "Bob Martinez", Email: "bob@example.com", Commits: 18, LinesChanged: 900, WeekendNightCommits: 1, FirstSeen: "2026-01-28", IsNew: false},
				},
			},
		},
		Narrative: []types.NarrativeBullet{
			{Kind: "alert", Text: "Top hotspot: src/payments/ledger.ts — 22 commits, 9.6 bug-weighted touches (3 chaos-tier)."},
			{Kind: "warn", Text: "Hottest modules: payments (72/100), auth (48/100)."},
			{Kind: "info", Text: "1 commit was reverted within a week of landing."},
		},
		RollingTimeline: []types.RollingPoint{
			{Date: "2026-04-10", Score: 55, Commits: 42, BugPct: 0.28},
			{Date: "2026-04-11", Score: 58, Commits: 45, BugPct: 0.30},
			{Date: "2026-04-12", Score: 62, Commits: 48, BugPct: 0.35},
		},
	}
}

// UIMeta — deterministic RepoMeta for the UI fixture.
func UIMeta() types.RepoMeta {
	return types.RepoMeta{
		RepoName:        "fixture-repo",
		RepoPath:        "/tmp/fixture-repo",
		AnalyzedCommits: 120,
		WindowDays:      90,
		WindowStart:     time.Date(2026, 1, 15, 0, 0, 0, 0, time.UTC),
		WindowEnd:       time.Date(2026, 4, 15, 0, 0, 0, 0, time.UTC),
		GeneratedAt:     time.Date(2026, 4, 15, 12, 0, 0, 0, time.UTC),
	}
}

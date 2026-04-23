// Package types holds all shared data shapes for the repopulse tool.
// This is a direct port of src/types.ts — field names and JSON tags must
// match the TS version so JSON snapshots and Playwright e2e tests stay
// compatible.
package types

import "time"

// MoodLabel is the top-level qualitative label.
type MoodLabel string

const (
	MoodCalm    MoodLabel = "calm"
	MoodAnxious MoodLabel = "anxious"
	MoodChaotic MoodLabel = "chaotic"
)

// --- Commit data ---

type CommitRecord struct {
	Hash               string
	Date               time.Time // committer date
	AuthorDate         time.Time
	AuthorName         string
	AuthorEmail        string
	Message            string
	FilesChanged       []FileChange
	IsRevert           bool
	RevertedHashShort  string // optional; empty if not a revert or no target found
}

type FileChange struct {
	Path    string
	Added   int
	Removed int
}

// --- Repo / run metadata ---

type RepoMeta struct {
	RepoName          string    `json:"repoName"`
	RepoPath          string    `json:"repoPath"`
	AnalyzedCommits   int       `json:"analyzedCommits"`
	WindowDays        int       `json:"windowDays"`
	WindowStart       time.Time `json:"windowStart"`
	WindowEnd         time.Time `json:"windowEnd"`
	GeneratedAt       time.Time `json:"generatedAt"`
	HasLimitedHistory bool      `json:"hasLimitedHistory"`
}

// --- Signal result types ---

type FrequencySignal struct {
	Type           string      `json:"type"` // "commitFrequency"
	Score          int         `json:"score"`
	DailyBuckets   []DayBucket `json:"dailyBuckets"`
	Mean           float64     `json:"mean"`
	StdDev         float64     `json:"stdDev"`
	LongestGapDays int         `json:"longestGapDays"`
}

type ChurnSignal struct {
	Type               string       `json:"type"` // "fileChurn"
	Score              int          `json:"score"`
	TopChurners        []ChurnEntry `json:"topChurners"`
	TotalFilesTouched  int          `json:"totalFilesTouched"`
	EligibleFileCount  int          `json:"eligibleFileCount"`
	HighChurnFileCount int          `json:"highChurnFileCount"`
	TotalLinesChanged  int          `json:"totalLinesChanged"`
	LinesPerDay        float64      `json:"linesPerDay"`
}

type ChurnEntry struct {
	Path             string              `json:"path"`
	Added            int                 `json:"added"`
	Removed          int                 `json:"removed"`
	Ratio            float64             `json:"ratio"`
	Rewritten        bool                `json:"rewritten"`
	TotalCommits     int                 `json:"totalCommits,omitempty"`
	LastTouched      string              `json:"lastTouched,omitempty"`
	TopAuthorsOfFile []HotspotFileAuthor `json:"topAuthorsOfFile,omitempty"`
	RecentCommits    []HotspotCommit     `json:"recentCommits,omitempty"`
}

type BugSignal struct {
	Type               string              `json:"type"` // "bugRatio"
	Score              int                 `json:"score"`
	BugCommitCount     int                 `json:"bugCommitCount"`
	ChaosCommitCount   int                 `json:"chaosCommitCount"`
	RoutineFixCount    int                 `json:"routineFixCount"`
	NormalFixCount     int                 `json:"normalFixCount"`
	TotalCommits       int                 `json:"totalCommits"`
	Ratio              float64             `json:"ratio"`
	LongestFixStreak   int                 `json:"longestFixStreak"`
	BugCommitsByDay    []DayBucket         `json:"bugCommitsByDay"`
	NormalCommitsByDay []DayBucket         `json:"normalCommitsByDay"`
	ChaosCommitsByDay  []DayBucket         `json:"chaosCommitsByDay"`
	RevertedWithin7d   int                 `json:"revertedWithin7d"`
	ClassifiedSamples  BugClassifiedGroups `json:"classifiedSamples"`
}

type BugClassifiedGroups struct {
	Chaos   []BugClassifiedCommit `json:"chaos"`
	Normal  []BugClassifiedCommit `json:"normal"`
	Routine []BugClassifiedCommit `json:"routine"`
}

type BugClassifiedCommit struct {
	Hash           string `json:"hash"`
	Date           string `json:"date"`
	Author         string `json:"author"`
	Message        string `json:"message"`
	MatchedKeyword string `json:"matchedKeyword"`
}

type CoverageSignal struct {
	Type       string  `json:"type"` // "coverage"
	Score      int     `json:"score"`
	Percentage float64 `json:"percentage"`
	Source     string  `json:"source"` // "istanbul" | "lcov"
}

type ModuleSignal struct {
	Type    string        `json:"type"` // "modules"
	Modules []ModuleEntry `json:"modules"`
}

type ModuleEntry struct {
	Name         string    `json:"name"`
	Score        int       `json:"score"`
	Mood         MoodLabel `json:"mood"`
	Commits      int       `json:"commits"`
	LinesChanged int       `json:"linesChanged"`
	BugRatio     float64   `json:"bugRatio"`
	Authors      int       `json:"authors"`
	TopFile      string    `json:"topFile,omitempty"`
	Owners       []string  `json:"owners"`
}

type HotspotSignal struct {
	Type     string         `json:"type"` // "hotspots"
	Hotspots []HotspotEntry `json:"hotspots"`
}

type HotspotEntry struct {
	Path             string                 `json:"path"`
	ChurnRank        int                    `json:"churnRank"`
	BugTouches       float64                `json:"bugTouches"`
	ChaosTouches     int                    `json:"chaosTouches"`
	TotalCommits     int                    `json:"totalCommits"`
	HotspotScore     int                    `json:"hotspotScore"`
	Authors          int                    `json:"authors"`
	LastTouched      string                 `json:"lastTouched"`
	TopAuthorsOfFile []HotspotFileAuthor    `json:"topAuthorsOfFile"`
	RecentBugCommits []HotspotCommit        `json:"recentBugCommits"`
	Owners           []string               `json:"owners"`
	Recommendations  []HotspotRecommendation `json:"recommendations"`
}

type HotspotFileAuthor struct {
	Name    string `json:"name"`
	Commits int    `json:"commits"`
}

type HotspotCommit struct {
	Hash    string `json:"hash"`
	Date    string `json:"date"`
	Author  string `json:"author"`
	Message string `json:"message"`
	Tier    string `json:"tier"` // "chaos" | "normal" | "routine" | "none"
}

type HotspotRecommendation struct {
	Kind     string `json:"kind"`     // "bus-factor" | "chaos-repeat" | etc.
	Severity string `json:"severity"` // "alert" | "warn" | "info"
	Text     string `json:"text"`
}

type AuthorSignal struct {
	Type                   string        `json:"type"` // "authors"
	Score                  int           `json:"score"`
	TotalAuthors           int           `json:"totalAuthors"`
	WeekendNightPct        float64       `json:"weekendNightPct"`
	BusFactorTop1Pct       float64       `json:"busFactorTop1Pct"`
	BusFactorTop3Pct       float64       `json:"busFactorTop3Pct"`
	NewContributorChurnPct float64       `json:"newContributorChurnPct"`
	// Contributors is the FULL list (every author who appears in the
	// window), sorted by lines-changed descending. Powers the bottom
	// Contributors explorer. Populated even if AuthorEntry.TopFiles
	// would be empty (e.g. the author touched only excluded paths).
	Contributors []AuthorEntry `json:"contributors"`
}

type AuthorEntry struct {
	Name                string             `json:"name"`
	Email               string             `json:"email"`
	Commits             int                `json:"commits"`
	LinesChanged        int                `json:"linesChanged"`
	WeekendNightCommits int                `json:"weekendNightCommits"`
	FirstSeen           string             `json:"firstSeen"`
	IsNew               bool               `json:"isNew"`
	TopFiles            []AuthorFileTouch  `json:"topFiles,omitempty"`
}

// AuthorFileTouch is one of an author's most-touched files in the window,
// surfaced inside their drill-down on the Contributors explorer.
type AuthorFileTouch struct {
	Path    string `json:"path"`
	Commits int    `json:"commits"`
	Added   int    `json:"added"`
	Removed int    `json:"removed"`
}

// --- Plank 1: per-author baseline drift ---

type AuthorDriftSignal struct {
	Type         string        `json:"type"` // "authorDrift"
	CurrentDays  int           `json:"currentDays"`
	BaselineDays int           `json:"baselineDays"`
	Authors      []AuthorDrift `json:"authors"`
}

type AuthorDrift struct {
	Name                   string      `json:"name"`
	Email                  string      `json:"email"`
	CommitsCurrent         int         `json:"commitsCurrent"`
	CommitsPerWeekCurrent  float64     `json:"commitsPerWeekCurrent"`
	CommitsPerWeekBaseline float64     `json:"commitsPerWeekBaseline"`
	CommitsDeltaPct        float64     `json:"commitsDeltaPct"`
	WeekendNightCurrent    float64     `json:"weekendNightCurrent"`  // %
	WeekendNightBaseline   float64     `json:"weekendNightBaseline"` // %
	WeekendNightDeltaPP    float64     `json:"weekendNightDeltaPP"`  // percentage points
	FixRatioCurrent        float64     `json:"fixRatioCurrent"`      // %
	FixRatioBaseline       float64     `json:"fixRatioBaseline"`     // %
	FixRatioDeltaPP        float64     `json:"fixRatioDeltaPP"`      // percentage points
	Flags                  []DriftFlag `json:"flags"`
}

type DriftFlag struct {
	Kind     string `json:"kind"`     // "cadence-up" | "cadence-down" | "weekend-night-up" | "fix-ratio-up"
	Severity string `json:"severity"` // "info" | "watch" | "alert"
	Text     string `json:"text"`
}

// --- Plank 2: deterministic standards-detection ---

type StandardsSignal struct {
	Type                string                    `json:"type"` // "standards"
	ConventionalCommits ConventionalCommitsResult `json:"conventionalCommits"`
	TestDensity         TestDensityResult         `json:"testDensity"`
}

type ConventionalCommitsResult struct {
	Total               int                     `json:"total"`
	Compliant           int                     `json:"compliant"`
	CompliancePct       float64                 `json:"compliancePct"`
	PerAuthor           []AuthorComplianceEntry `json:"perAuthor"`
	NonCompliantSamples []NonCompliantCommit    `json:"nonCompliantSamples"`
	// Pattern is the effective regex for this run — empty string means
	// the built-in Conventional Commits default was used, non-empty is
	// whatever the team declared in `.repopulserc`. Surfaced in the UI
	// subtitle so the user can see which pattern is in effect.
	Pattern string `json:"pattern,omitempty"`
}

type AuthorComplianceEntry struct {
	Name          string  `json:"name"`
	Email         string  `json:"email"`
	Total         int     `json:"total"`
	Compliant     int     `json:"compliant"`
	CompliancePct float64 `json:"compliancePct"`
}

type NonCompliantCommit struct {
	Hash    string `json:"hash"`
	Author  string `json:"author"`
	Subject string `json:"subject"`
}

// TestDensityResult measures test-to-source file ratio per language and
// per module. Replaced the earlier "test-file colocation" metric because
// that one required filename correspondence (FooController.kt ↔
// FooControllerTest.kt), which under-reports any team that splits tests
// by action, uses integration-test suites, or otherwise doesn't name
// tests after single source classes. Density asks the simpler question:
// "does this module have tests at all, and roughly how much?"
type TestDensityResult struct {
	Languages   []string             `json:"languages"` // detected source extensions, e.g. [".kt", ".ts"]
	SourceFiles int                  `json:"sourceFiles"`
	TestFiles   int                  `json:"testFiles"`
	// DensityPct = TestFiles / SourceFiles * 100. Can exceed 100% when a
	// codebase has more test files than source files (healthy!).
	DensityPct float64              `json:"densityPct"`
	PerModule  []ModuleDensityEntry `json:"perModule"`
}

type ModuleDensityEntry struct {
	Module      string  `json:"module"`
	SourceFiles int     `json:"sourceFiles"`
	TestFiles   int     `json:"testFiles"`
	DensityPct  float64 `json:"densityPct"`
}

// --- PR flow (GitHub PR metadata integration) ---

// PRFlowSignal carries every PR-derived signal surfaced in the report.
// Nil-able on MoodResult.Signals because the whole block is gated on a
// GitHub token being available; no token → no section.
type PRFlowSignal struct {
	Type         string          `json:"type"` // "prFlow"
	OwnerRepo    string          `json:"ownerRepo"`  // e.g. "anthropic/claude-code"
	WindowDays   int             `json:"windowDays"`
	TotalPRs     int             `json:"totalPrs"`
	MergedPRs    int             `json:"mergedPrs"`
	CycleHours   Percentiles     `json:"cycleHours"`   // merged-PR cycle time p50/p75/p95 in hours
	TTFRHours    Percentiles     `json:"ttfrHours"`    // time-to-first-review p50/p75 in hours
	Reviewers    []ReviewerEntry `json:"reviewers"`    // top-N reviewers by PR count
	RubberStamps []PRSample      `json:"rubberStamps"` // approved in <60s with 0 review comments
	RubberStampRate float64      `json:"rubberStampRate"` // %
	SelfMergeRate   float64      `json:"selfMergeRate"`   // % of merged PRs merged by their own author
	// CacheBanner surfaces when rate-limited so the report explains why
	// the data might be stale rather than silently serving cached
	// numbers. Empty string when no banner should render.
	CacheBanner string `json:"cacheBanner,omitempty"`
}

// Percentiles holds common distribution stats for a single metric.
type Percentiles struct {
	P50 float64 `json:"p50"`
	P75 float64 `json:"p75"`
	P95 float64 `json:"p95"`
}

type ReviewerEntry struct {
	Login       string  `json:"login"`
	ReviewCount int     `json:"reviewCount"` // # of PRs they reviewed
	SharePct    float64 `json:"sharePct"`    // their share of total review activity
}

type PRSample struct {
	Number     int    `json:"number"`
	Title      string `json:"title"`
	Author     string `json:"author"`
	MergedBy   string `json:"mergedBy,omitempty"`
	CycleHours float64 `json:"cycleHours"`
}

type DayBucket struct {
	Date  string `json:"date"`
	Count int    `json:"count"`
}

// --- Composite ---

type MoodResult struct {
	Mood            MoodLabel         `json:"mood"`
	CompositeScore  int               `json:"compositeScore"`
	Breakdown       MoodBreakdown     `json:"breakdown"`
	Signals         Signals           `json:"signals"`
	Narrative       []NarrativeBullet `json:"narrative"`
	RollingTimeline []RollingPoint    `json:"rollingTimeline"`
}

type MoodBreakdown struct {
	CommitFrequency int  `json:"commitFrequency"`
	FileChurn       int  `json:"fileChurn"`
	BugRatio        int  `json:"bugRatio"`
	Coverage        *int `json:"coverage"` // nil if no coverage data
	Authors         int  `json:"authors"`
}

type Signals struct {
	CommitFrequency FrequencySignal   `json:"commitFrequency"`
	FileChurn       ChurnSignal       `json:"fileChurn"`
	BugRatio        BugSignal         `json:"bugRatio"`
	Coverage        *CoverageSignal   `json:"coverage"`
	Modules         ModuleSignal      `json:"modules"`
	Hotspots        HotspotSignal     `json:"hotspots"`
	Authors         AuthorSignal      `json:"authors"`
	AuthorDrift     AuthorDriftSignal `json:"authorDrift"`
	Standards       StandardsSignal   `json:"standards"`
	PRFlow          *PRFlowSignal     `json:"prFlow,omitempty"`
}

type NarrativeBullet struct {
	Kind string `json:"kind"` // "info" | "warn" | "alert" | "good"
	Text string `json:"text"`
}

type RollingPoint struct {
	Date    string  `json:"date"`
	Score   int     `json:"score"`
	Commits int     `json:"commits"`
	BugPct  float64 `json:"bugPct"`
}

// --- CLI options ---

type CliOptions struct {
	Window       int
	Output       string
	Open         bool
	Since        string
	Ignore       []string
	JSON         string
	Compare      string
	Markdown     string
	NoSnapshot   bool
	GitHubToken  string // optional; falls back to GITHUB_TOKEN env var
	GitHubRepo   string // optional owner/name override when origin URL is ambiguous
}

// --- Delta (compare) ---

type MoodDelta struct {
	Composite  int            `json:"composite"`
	Breakdown  map[string]int `json:"breakdown"`
	PreviousAt string         `json:"previousAt,omitempty"`
}

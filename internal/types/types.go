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
	Path      string  `json:"path"`
	Added     int     `json:"added"`
	Removed   int     `json:"removed"`
	Ratio     float64 `json:"ratio"`
	Rewritten bool    `json:"rewritten"`
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
	TopAuthors             []AuthorEntry `json:"topAuthors"`
}

type AuthorEntry struct {
	Name                string `json:"name"`
	Email               string `json:"email"`
	Commits             int    `json:"commits"`
	LinesChanged        int    `json:"linesChanged"`
	WeekendNightCommits int    `json:"weekendNightCommits"`
	FirstSeen           string `json:"firstSeen"`
	IsNew               bool   `json:"isNew"`
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
	CommitFrequency FrequencySignal `json:"commitFrequency"`
	FileChurn       ChurnSignal     `json:"fileChurn"`
	BugRatio        BugSignal       `json:"bugRatio"`
	Coverage        *CoverageSignal `json:"coverage"`
	Modules         ModuleSignal    `json:"modules"`
	Hotspots        HotspotSignal   `json:"hotspots"`
	Authors         AuthorSignal    `json:"authors"`
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
	Window     int
	Output     string
	Open       bool
	Since      string
	Ignore     []string
	JSON       string
	Compare    string
	Markdown   string
	NoSnapshot bool
}

// --- Delta (compare) ---

type MoodDelta struct {
	Composite  int            `json:"composite"`
	Breakdown  map[string]int `json:"breakdown"`
	PreviousAt string         `json:"previousAt,omitempty"`
}

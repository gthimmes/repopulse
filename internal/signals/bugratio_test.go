package signals

import (
	"fmt"
	"regexp"
	"testing"
	"time"

	"repopulse/internal/config"
	"repopulse/internal/fixtures"
	"repopulse/internal/types"
)

var defaultBugOpts = BugOptions{
	ChaosKeywords:   config.DefaultBugKeywords.Chaos,
	NormalKeywords:  config.DefaultBugKeywords.Normal,
	RoutineKeywords: config.DefaultBugKeywords.Routine,
}

var revertMsgRE = regexp.MustCompile(`^(?:Revert\s+["']|revert\s*[:(])`)

func makeCommit(hash string, daysAgo int, message, author string) types.CommitRecord {
	base := time.Date(2026, 4, 15, 12, 0, 0, 0, time.UTC)
	d := base.AddDate(0, 0, -daysAgo)
	if author == "" {
		author = "Alice Chen"
	}
	return types.CommitRecord{
		Hash:        hash,
		Date:        d,
		AuthorDate:  d,
		AuthorName:  author,
		AuthorEmail: "alice@example.com",
		Message:     message,
		IsRevert:    revertMsgRE.MatchString(message),
	}
}

func TestComputeBugRatio_CalmFixture(t *testing.T) {
	r := ComputeBugRatio(fixtures.CalmFixture(), defaultBugOpts)
	if r.Type != "bugRatio" {
		t.Errorf("type mismatch: %s", r.Type)
	}
	if r.Score >= 20 {
		t.Errorf("calm score should be < 20, got %d", r.Score)
	}
	if r.Ratio >= 0.1 {
		t.Errorf("calm ratio should be < 0.1, got %g", r.Ratio)
	}
}

func TestComputeBugRatio_ChaoticFixture(t *testing.T) {
	r := ComputeBugRatio(fixtures.ChaoticFixture(), defaultBugOpts)
	if r.Score <= 50 {
		t.Errorf("chaotic score should be > 50, got %d", r.Score)
	}
}

func TestComputeBugRatio_DetectsFixStreaks(t *testing.T) {
	r := ComputeBugRatio(fixtures.ChaoticFixture(), defaultBugOpts)
	if r.LongestFixStreak < 5 {
		t.Errorf("expected streak >= 5, got %d", r.LongestFixStreak)
	}
}

func TestComputeBugRatio_Empty(t *testing.T) {
	r := ComputeBugRatio(nil, defaultBugOpts)
	if r.Score != 0 {
		t.Errorf("empty score should be 0, got %d", r.Score)
	}
	if r.Ratio != 0 {
		t.Errorf("empty ratio should be 0, got %g", r.Ratio)
	}
	if len(r.ClassifiedSamples.Chaos) != 0 || len(r.ClassifiedSamples.Normal) != 0 || len(r.ClassifiedSamples.Routine) != 0 {
		t.Error("empty should produce no samples")
	}
}

func TestClassifyCommitWithKeyword_ChaosKeyword(t *testing.T) {
	tier, kw := ClassifyCommitWithKeyword("hotfix: critical crash", false, defaultBugOpts)
	if tier != TierChaos || kw != "hotfix" {
		t.Errorf("want chaos+hotfix, got %s+%s", tier, kw)
	}
}

func TestClassifyCommitWithKeyword_Revert(t *testing.T) {
	tier, kw := ClassifyCommitWithKeyword(`Revert "feat: add foo"`, true, defaultBugOpts)
	if tier != TierChaos || kw != "(revert)" {
		t.Errorf("want chaos+(revert), got %s+%s", tier, kw)
	}
}

func TestClassifyCommitWithKeyword_RoutineBeforeNormal(t *testing.T) {
	tier, kw := ClassifyCommitWithKeyword("fix: typo in comment", false, defaultBugOpts)
	if tier != TierRoutine || kw != "typo" {
		t.Errorf("want routine+typo, got %s+%s", tier, kw)
	}
}

func TestClassifyCommitWithKeyword_NonBug(t *testing.T) {
	tier, kw := ClassifyCommitWithKeyword("feat: add user profile", false, defaultBugOpts)
	if tier != TierNone || kw != "" {
		t.Errorf("want none+'', got %s+%s", tier, kw)
	}
}

func TestClassifyCommitWithKeyword_ConventionalPrefixVetoesKeywordMatch(t *testing.T) {
	cases := []struct {
		name, msg string
	}{
		{"feat-with-fix-in-body", "feat: add X to replace the old buggy fix path"},
		{"feat-with-scope", "feat(auth): rework session flow to fix login"},
		{"feat-breaking", "feat!: remove deprecated patch endpoint"},
		{"feat-scope-breaking", "feat(api)!: delete the old hotfix route"},
		{"chore-with-patch", "chore: bump dep to patch CVE"},
		{"docs-with-typo", "docs(api): fix typo in examples"},
		{"style-with-formatting", "style: apply standard formatting across package"},
		{"test-with-bug", "test: add coverage for login bug repro"},
		{"refactor-with-fix", "refactor: extract helper; fix duplicate imports"},
		{"ci-with-broken", "ci: skip broken matrix entry until upstream patch"},
		{"build-with-regression", "build: revert webpack 5 regression bump"},
		{"perf-with-workaround", "perf: switch algorithm; removes old workaround"},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			tier, kw := ClassifyCommitWithKeyword(c.msg, false, defaultBugOpts)
			if tier != TierNone || kw != "" {
				t.Errorf("%q → want none+'', got %s+%s", c.msg, tier, kw)
			}
		})
	}
}

func TestClassifyCommitWithKeyword_FixPrefixStillClassifies(t *testing.T) {
	// `fix:` and `revert:` are deliberately NOT in the non-bug prefix list —
	// they remain subject to the regular keyword path so they keep landing
	// in normal/routine/chaos as appropriate.
	tier, kw := ClassifyCommitWithKeyword("fix(auth): handle nil token", false, defaultBugOpts)
	if tier != TierNormal || kw != "fix" {
		t.Errorf("want normal+fix, got %s+%s", tier, kw)
	}
	tier2, _ := ClassifyCommitWithKeyword("fix: cleanup whitespace", false, defaultBugOpts)
	if tier2 != TierRoutine {
		t.Errorf("want routine for fix+whitespace, got %s", tier2)
	}
}

func TestClassifiedSamples_GroupedByTier(t *testing.T) {
	commits := []types.CommitRecord{
		makeCommit("h001", 1, "hotfix: crash on startup", ""),
		makeCommit("h002", 2, "fix: login flow issue", ""),
		makeCommit("h003", 3, "fix: typo in readme", ""),
		makeCommit("h004", 4, "feat: new feature", ""),
		makeCommit("h005", 5, `Revert "bad merge"`, ""),
	}
	r := ComputeBugRatio(commits, defaultBugOpts)
	if got := len(r.ClassifiedSamples.Chaos); got != 2 {
		t.Errorf("chaos count: want 2, got %d", got)
	}
	if got := len(r.ClassifiedSamples.Normal); got != 1 {
		t.Errorf("normal count: want 1, got %d", got)
	}
	if got := len(r.ClassifiedSamples.Routine); got != 1 {
		t.Errorf("routine count: want 1, got %d", got)
	}
}

func TestClassifiedSamples_NewestFirst(t *testing.T) {
	commits := []types.CommitRecord{
		makeCommit("h001", 10, "fix: old bug", ""),
		makeCommit("h002", 2, "fix: new bug", ""),
		makeCommit("h003", 5, "fix: middle bug", ""),
	}
	r := ComputeBugRatio(commits, defaultBugOpts)
	dates := r.ClassifiedSamples.Normal
	for i := 1; i < len(dates); i++ {
		if dates[i-1].Date < dates[i].Date {
			t.Errorf("not newest-first at %d: %s vs %s", i, dates[i-1].Date, dates[i].Date)
		}
	}
}

func TestClassifiedSamples_Capped(t *testing.T) {
	commits := make([]types.CommitRecord, 35)
	for i := 0; i < 35; i++ {
		h := fmt.Sprintf("h%03d", i)
		commits[i] = makeCommit(h, i, "fix: issue "+h, "")
	}
	r := ComputeBugRatio(commits, defaultBugOpts)
	if len(r.ClassifiedSamples.Normal) != 20 {
		t.Errorf("expected cap of 20, got %d", len(r.ClassifiedSamples.Normal))
	}
	if r.ClassifiedSamples.Normal[0].Hash != "h000" {
		t.Errorf("newest first: expected h000, got %s", r.ClassifiedSamples.Normal[0].Hash)
	}
}

func TestClassifiedSamples_KeywordPerSample(t *testing.T) {
	commits := []types.CommitRecord{
		makeCommit("h001", 3, "hotfix: chaos", ""),
		makeCommit("h002", 2, "fix: regular bug", ""),
		makeCommit("h003", 1, "fix: typo", ""),
	}
	r := ComputeBugRatio(commits, defaultBugOpts)
	if r.ClassifiedSamples.Chaos[0].MatchedKeyword != "hotfix" {
		t.Errorf("chaos[0] keyword: got %s", r.ClassifiedSamples.Chaos[0].MatchedKeyword)
	}
	if r.ClassifiedSamples.Normal[0].MatchedKeyword != "fix" {
		t.Errorf("normal[0] keyword: got %s", r.ClassifiedSamples.Normal[0].MatchedKeyword)
	}
	if r.ClassifiedSamples.Routine[0].MatchedKeyword != "typo" {
		t.Errorf("routine[0] keyword: got %s", r.ClassifiedSamples.Routine[0].MatchedKeyword)
	}
}

func TestClassifiedSamples_HashPrefix7(t *testing.T) {
	commits := []types.CommitRecord{makeCommit("abcdef0123456", 1, "hotfix: bad", "Bob")}
	r := ComputeBugRatio(commits, defaultBugOpts)
	s := r.ClassifiedSamples.Chaos[0]
	if s.Hash != "abcdef0" || len(s.Hash) != 7 {
		t.Errorf("hash prefix: got %s", s.Hash)
	}
	if s.Date != "2026-04-14" {
		t.Errorf("date: got %s", s.Date)
	}
	if s.Author != "Bob" {
		t.Errorf("author: got %s", s.Author)
	}
	if s.Message != "hotfix: bad" {
		t.Errorf("message: got %s", s.Message)
	}
}


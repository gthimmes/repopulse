package config

import (
	"reflect"
	"sort"
	"testing"
)

func sortedCopy(s []string) []string {
	out := append([]string{}, s...)
	sort.Strings(out)
	return out
}

func TestResolvedBugKeywords_NoConfigReturnsDefaults(t *testing.T) {
	got := ResolvedBugKeywords(RepopulseConfig{})
	if !reflect.DeepEqual(sortedCopy(got.Chaos), sortedCopy(DefaultBugKeywords.Chaos)) {
		t.Errorf("chaos: want defaults, got %v", got.Chaos)
	}
	if !reflect.DeepEqual(sortedCopy(got.Normal), sortedCopy(DefaultBugKeywords.Normal)) {
		t.Errorf("normal: want defaults, got %v", got.Normal)
	}
}

func TestResolvedBugKeywords_AppendsRatherThanReplaces(t *testing.T) {
	cfg := RepopulseConfig{
		BugKeywords: &BugKeywords{
			Normal: []string{"defect", "incident"},
		},
	}
	got := ResolvedBugKeywords(cfg)
	// Must keep the default "fix" etc.
	hasFix, hasDefect := false, false
	for _, k := range got.Normal {
		if k == "fix" {
			hasFix = true
		}
		if k == "defect" {
			hasDefect = true
		}
	}
	if !hasFix {
		t.Errorf("merge mode lost default 'fix': %v", got.Normal)
	}
	if !hasDefect {
		t.Errorf("merge mode lost added 'defect': %v", got.Normal)
	}
}

func TestResolvedBugKeywords_BangPrefixRemovesDefault(t *testing.T) {
	cfg := RepopulseConfig{
		BugKeywords: &BugKeywords{
			Normal: []string{"!workaround", "defect"},
		},
	}
	got := ResolvedBugKeywords(cfg)
	for _, k := range got.Normal {
		if k == "workaround" {
			t.Errorf("expected !workaround to drop the default, got %v", got.Normal)
		}
	}
	found := false
	for _, k := range got.Normal {
		if k == "defect" {
			found = true
		}
	}
	if !found {
		t.Errorf("expected added 'defect', got %v", got.Normal)
	}
}

func TestResolvedBugKeywords_DuplicatesCollapseCaseInsensitive(t *testing.T) {
	cfg := RepopulseConfig{
		BugKeywords: &BugKeywords{
			Normal: []string{"FIX", "fix", "Bug"}, // "fix" already default; "Bug" already default
		},
	}
	got := ResolvedBugKeywords(cfg)
	count := 0
	for _, k := range got.Normal {
		if k == "fix" || k == "FIX" {
			count++
		}
	}
	if count != 1 {
		t.Errorf("want exactly one `fix`-like entry, got %d in %v", count, got.Normal)
	}
}

func TestResolvedCommitPattern_DefaultWhenAbsent(t *testing.T) {
	re, src, custom := ResolvedCommitPattern(RepopulseConfig{})
	if custom {
		t.Errorf("expected default-path when no CommitPattern set")
	}
	if src != DefaultCommitPattern {
		t.Errorf("expected default pattern string")
	}
	if !re.MatchString("feat: add foo") {
		t.Errorf("default regex should match a Conventional Commit")
	}
}

func TestResolvedCommitPattern_UsesCustom(t *testing.T) {
	cfg := RepopulseConfig{CommitPattern: `^\[[A-Z]+-\d+\] (Fix|Feature)\s`}
	re, src, custom := ResolvedCommitPattern(cfg)
	if !custom {
		t.Errorf("expected custom-path when CommitPattern set")
	}
	if src != cfg.CommitPattern {
		t.Errorf("source string should echo the custom pattern")
	}
	if !re.MatchString("[TICKET-1234] Fix login flow") {
		t.Errorf("custom regex should match team commits")
	}
	if re.MatchString("feat: add foo") {
		t.Errorf("custom regex should NOT match Conventional Commits style")
	}
}

func TestResolvedCommitPattern_FallsBackOnInvalid(t *testing.T) {
	cfg := RepopulseConfig{CommitPattern: `[unterminated group`}
	re, _, custom := ResolvedCommitPattern(cfg)
	if custom {
		t.Errorf("invalid regex should fall back to default, not be marked custom")
	}
	if !re.MatchString("feat: add foo") {
		t.Errorf("fallback regex should be the default Conventional Commits pattern")
	}
}

func TestResolvedBugKeywords_EmptyOverrideKeepsDefaults(t *testing.T) {
	cfg := RepopulseConfig{BugKeywords: &BugKeywords{}}
	got := ResolvedBugKeywords(cfg)
	if len(got.Chaos) != len(DefaultBugKeywords.Chaos) {
		t.Errorf("empty override should preserve chaos defaults, got %d", len(got.Chaos))
	}
}

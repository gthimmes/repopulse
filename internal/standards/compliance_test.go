package standards

import (
	"regexp"
	"testing"
	"time"

	"repopulse/internal/types"
)

// defaultPattern for tests — mirrors config.DefaultCommitPattern.
// Duplicated here rather than importing config to keep the standards
// test scope tight.
var defaultPattern = regexp.MustCompile(defaultCommitPattern)

func mkCommit(hash, email, name, message string) types.CommitRecord {
	return types.CommitRecord{
		Hash:        hash,
		Date:        time.Now(),
		AuthorDate:  time.Now(),
		AuthorEmail: email,
		AuthorName:  name,
		Message:     message,
	}
}

// call ComputeCommitCompliance with the default pattern — handy for
// tests that just want to exercise the built-in Conventional Commits
// behaviour.
func computeDefault(commits []types.CommitRecord) types.ConventionalCommitsResult {
	return ComputeCommitCompliance(commits, defaultPattern)
}

func TestConventionalCommits_AllCompliant(t *testing.T) {
	commits := []types.CommitRecord{
		mkCommit("h1", "a@x", "A", "feat: add foo"),
		mkCommit("h2", "a@x", "A", "fix(auth): null check"),
		mkCommit("h3", "a@x", "A", "chore: bump deps"),
		mkCommit("h4", "a@x", "A", "feat!: breaking change"),
		mkCommit("h5", "a@x", "A", "feat(api)!: bigger breaking"),
	}
	r := computeDefault(commits)
	if r.Compliant != 5 {
		t.Errorf("want 5 compliant, got %d", r.Compliant)
	}
	if r.CompliancePct != 100.0 {
		t.Errorf("want 100%%, got %v", r.CompliancePct)
	}
}

func TestConventionalCommits_RejectsBadFormats(t *testing.T) {
	commits := []types.CommitRecord{
		mkCommit("h1", "a@x", "A", "Add foo"),                              // missing type
		mkCommit("h2", "a@x", "A", "feat add foo"),                         // missing colon
		mkCommit("h3", "a@x", "A", "wip: something"),                       // unknown type
		mkCommit("h4", "a@x", "A", "Merge pull request #123 from foo/bar"), // merge commit
		mkCommit("h5", "a@x", "A", "feat:no-space"),                        // no space after colon
		mkCommit("h6", "a@x", "A", "feat: ok"),                             // valid baseline
	}
	r := computeDefault(commits)
	if r.Compliant != 1 {
		t.Errorf("want 1 compliant, got %d", r.Compliant)
	}
	if len(r.NonCompliantSamples) != 5 {
		t.Errorf("want 5 samples, got %d", len(r.NonCompliantSamples))
	}
}

func TestConventionalCommits_PerAuthorWorstFirst(t *testing.T) {
	commits := []types.CommitRecord{
		// Alice — 4 commits, all compliant (100%)
		mkCommit("a1", "a@x", "Alice", "feat: x"),
		mkCommit("a2", "a@x", "Alice", "feat: y"),
		mkCommit("a3", "a@x", "Alice", "fix: z"),
		mkCommit("a4", "a@x", "Alice", "chore: w"),
		// Bob — 4 commits, 1 compliant (25%)
		mkCommit("b1", "b@x", "Bob", "feat: ok"),
		mkCommit("b2", "b@x", "Bob", "wip thing"),
		mkCommit("b3", "b@x", "Bob", "another thing"),
		mkCommit("b4", "b@x", "Bob", "merge"),
		// Carol — 1 commit only, should sort below Bob even though 0%
		mkCommit("c1", "c@x", "Carol", "x"),
	}
	r := computeDefault(commits)
	if len(r.PerAuthor) < 2 {
		t.Fatalf("expected ≥2 authors, got %d", len(r.PerAuthor))
	}
	if r.PerAuthor[0].Email != "b@x" {
		t.Errorf("worst-first should put Bob first, got %s", r.PerAuthor[0].Email)
	}
}

func TestConventionalCommits_Empty(t *testing.T) {
	r := computeDefault(nil)
	if r.Total != 0 || r.Compliant != 0 {
		t.Errorf("empty input should be zeroed, got %+v", r)
	}
}

func TestCommitCompliance_CustomTicketPrefixPattern(t *testing.T) {
	// Team uses `[TICKET-NNNN] Verb …` convention instead of
	// Conventional Commits. The compliance check should count these as
	// compliant when given the team's pattern, and reject CC-style
	// subjects that don't match the team's shape.
	ticketRe := regexp.MustCompile(`^\[[A-Z]+-\d+\] (Fix|Feature|Chore|Refactor|Test)\s`)
	commits := []types.CommitRecord{
		mkCommit("h1", "a@x", "A", "[TICKET-1234] Fix login flow under low-memory"),
		mkCommit("h2", "a@x", "A", "[TICKET-5678] Feature add billing automation"),
		mkCommit("h3", "a@x", "A", "feat: add foo"),      // CC style, does NOT match ticket pattern
		mkCommit("h4", "a@x", "A", "Merge pull request #1"), // plain prose
	}
	r := ComputeCommitCompliance(commits, ticketRe)
	if r.Compliant != 2 {
		t.Errorf("want 2 compliant against ticket pattern, got %d", r.Compliant)
	}
}

func TestConventionalCommits_SamplesCappedAtMax(t *testing.T) {
	var commits []types.CommitRecord
	for i := 0; i < 50; i++ {
		commits = append(commits, mkCommit("h", "a@x", "A", "bad commit message"))
	}
	r := computeDefault(commits)
	if len(r.NonCompliantSamples) != MaxNonCompliantSamples {
		t.Errorf("samples should cap at %d, got %d", MaxNonCompliantSamples, len(r.NonCompliantSamples))
	}
}

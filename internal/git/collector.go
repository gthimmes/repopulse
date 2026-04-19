// Package git handles invoking git and parsing its output. All git I/O
// in the project goes through this package (mirrors the TS src/git/).
package git

import (
	"fmt"
	"os/exec"
	"strings"
	"time"

	"repopulse/internal/types"
)

// Unit separator between fields, record separator between commits.
// Non-printable so commit messages can't collide with them.
const (
	FieldSep  = "\x1f"
	RecordSep = "\x1e"
)

// GitFormat is the --format= string passed to `git log`. Record sep is at
// the START so splitting by it yields one record per commit with the
// numstat block attached to the correct header.
var GitFormat = RecordSep + strings.Join([]string{
	"%H",
	"%cI", // committer date ISO
	"%aI", // author date ISO
	"%an",
	"%ae",
	"%s",
}, FieldSep)

// CollectorOptions controls which commits get pulled.
type CollectorOptions struct {
	RepoPath   string
	Since      string // ISO date or relative string (e.g. "3 months ago")
	WindowDays int    // used if Since is empty
	Until      string // optional ISO date upper bound; powers baseline-window queries
}

// CollectCommits runs `git log --numstat` in the window and returns parsed records, newest first.
func CollectCommits(opts CollectorOptions) ([]types.CommitRecord, error) {
	if err := checkIsRepo(opts.RepoPath); err != nil {
		return nil, err
	}
	sinceStr := opts.Since
	if sinceStr == "" {
		days := opts.WindowDays
		if days <= 0 {
			days = 90
		}
		sinceStr = time.Now().AddDate(0, 0, -days).UTC().Format(time.RFC3339)
	}

	args := []string{
		"-C", opts.RepoPath,
		"log",
		"--numstat",
		"--format=" + GitFormat,
		"--since=" + sinceStr,
		"--no-merges",
	}
	if opts.Until != "" {
		args = append(args, "--until="+opts.Until)
	}
	cmd := exec.Command("git", args...)
	out, err := cmd.Output()
	if err != nil {
		if ee, ok := err.(*exec.ExitError); ok {
			stderr := string(ee.Stderr)
			if strings.Contains(stderr, "does not have any commits") {
				return nil, fmt.Errorf("repository has no commits")
			}
			return nil, fmt.Errorf("git log failed: %s", stderr)
		}
		return nil, err
	}
	return ParseGitLog(string(out))
}

// GetPreWindowAuthorEmails returns the set of author emails seen in commits
// BEFORE the given date. Used to decide which in-window authors are "new".
func GetPreWindowAuthorEmails(repoPath string, before time.Time) (map[string]struct{}, error) {
	cmd := exec.Command("git",
		"-C", repoPath,
		"log",
		"--format=%ae",
		"--before="+before.UTC().Format(time.RFC3339),
	)
	out, err := cmd.Output()
	if err != nil {
		// Quiet failure — return empty set, don't break the run
		return map[string]struct{}{}, nil
	}
	set := map[string]struct{}{}
	for _, line := range strings.Split(string(out), "\n") {
		e := strings.TrimSpace(line)
		if e != "" {
			set[e] = struct{}{}
		}
	}
	return set, nil
}

// ListFiles returns every file tracked at HEAD via `git ls-files`. Used
// by the standards signals (test density) to walk the working tree
// without falling back to OS-level globbing.
func ListFiles(repoPath string) ([]string, error) {
	cmd := exec.Command("git", "-C", repoPath, "ls-files")
	out, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("git ls-files failed: %w", err)
	}
	lines := strings.Split(strings.TrimRight(string(out), "\n"), "\n")
	if len(lines) == 1 && lines[0] == "" {
		return nil, nil
	}
	return lines, nil
}

// GetFileLineCount returns the current line count for a file at HEAD.
// Used by the churn signal to compute the churn/LOC ratio.
func GetFileLineCount(repoPath, filePath string) int {
	cmd := exec.Command("git", "-C", repoPath, "show", "HEAD:"+filePath)
	out, err := cmd.Output()
	if err != nil {
		return 0
	}
	return strings.Count(string(out), "\n") + 1
}

func checkIsRepo(repoPath string) error {
	cmd := exec.Command("git", "-C", repoPath, "rev-parse", "--is-inside-work-tree")
	out, err := cmd.Output()
	if err != nil {
		return fmt.Errorf("no git repository found at %s", repoPath)
	}
	if strings.TrimSpace(string(out)) != "true" {
		return fmt.Errorf("no git repository found at %s", repoPath)
	}
	return nil
}

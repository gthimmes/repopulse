// Package github handles the GitHub REST integration for PR metrics.
// Entry points used by the CLI: ParseOwnerRepo (from a git remote URL)
// and Fetcher (client + cache + signal assembly).
//
// Scope intentionally narrow — read-only PR/review fetches, GitHub only.
// GitLab/Bitbucket support and GraphQL are out of scope for v1.
package github

import (
	"fmt"
	"os/exec"
	"regexp"
	"strings"
)

// OwnerRepo identifies a GitHub repository (owner/name).
type OwnerRepo struct {
	Owner string
	Name  string
}

func (r OwnerRepo) String() string { return r.Owner + "/" + r.Name }

// sshRE matches `git@github.com:owner/repo(.git)?` style remotes.
var sshRE = regexp.MustCompile(`^git@github\.com:([^/]+)/([^/]+?)(?:\.git)?$`)

// httpsRE matches `https://github.com/owner/repo(.git)?(/?)$` style remotes.
// Accepts http(s), optional user@, optional trailing slash, optional .git.
var httpsRE = regexp.MustCompile(`^https?://(?:[^@]*@)?github\.com/([^/]+)/([^/]+?)(?:\.git)?/?$`)

// ParseRemoteURL parses a single remote URL into OwnerRepo. Returns
// ok=false for unrecognised forms (non-GitHub hosts, local paths, etc.)
// so the caller can fall back to CLI override or skip the PR section.
func ParseRemoteURL(remote string) (OwnerRepo, bool) {
	remote = strings.TrimSpace(remote)
	if m := sshRE.FindStringSubmatch(remote); m != nil {
		return OwnerRepo{Owner: m[1], Name: m[2]}, true
	}
	if m := httpsRE.FindStringSubmatch(remote); m != nil {
		return OwnerRepo{Owner: m[1], Name: m[2]}, true
	}
	return OwnerRepo{}, false
}

// DetectOwnerRepo runs `git -C <repoPath> remote get-url origin` and
// parses the result. Used when the user hasn't passed --github-repo.
func DetectOwnerRepo(repoPath string) (OwnerRepo, error) {
	cmd := exec.Command("git", "-C", repoPath, "remote", "get-url", "origin")
	out, err := cmd.Output()
	if err != nil {
		return OwnerRepo{}, fmt.Errorf("git remote get-url origin failed: %w", err)
	}
	ownerRepo, ok := ParseRemoteURL(strings.TrimSpace(string(out)))
	if !ok {
		return OwnerRepo{}, fmt.Errorf("origin URL is not a recognisable GitHub remote: %q", strings.TrimSpace(string(out)))
	}
	return ownerRepo, nil
}

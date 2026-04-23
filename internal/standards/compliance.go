// Package standards holds Plank-2 deterministic standards-detection
// signals: things you can compute purely from git data + the working
// tree, no AI required. AI enrichment lands in a later layer.
//
// First two signals: conventional-commit compliance (this file) and
// test density (colocation.go — filename retained for legacy). The
// framing throughout is "observation worth a conversation," not
// "score worth a ranking."
package standards

import (
	"regexp"
	"sort"
	"strings"

	"repopulse/internal/types"
)

// MaxNonCompliantSamples caps how many bad subjects we keep on hand for
// the report drill-down. Trimmed to the most recent N per the order
// commits arrive (newest first, per the existing collector contract).
const MaxNonCompliantSamples = 12

// ComputeCommitCompliance walks the commits and emits a compliance
// breakdown — overall %, per-author %, and a small sample of subjects
// that didn't match the pattern. `pattern` is the effective regex the
// team has chosen (Conventional Commits by default, or whatever they
// declared in `.repopulserc`). Merge commits and reverts are *included*
// in the denominator on purpose: a "Merge pull request #123" subject
// is itself a standards violation worth surfacing.
func ComputeCommitCompliance(commits []types.CommitRecord, pattern *regexp.Regexp) types.ConventionalCommitsResult {
	if len(commits) == 0 {
		return types.ConventionalCommitsResult{}
	}

	type authorAgg struct {
		name      string
		total     int
		compliant int
	}
	byEmail := map[string]*authorAgg{}

	totalCompliant := 0
	var nonCompliantSamples []types.NonCompliantCommit

	for _, c := range commits {
		subject := firstLine(c.Message)

		a, ok := byEmail[c.AuthorEmail]
		if !ok {
			a = &authorAgg{name: c.AuthorName}
			byEmail[c.AuthorEmail] = a
		}
		a.total++

		if pattern.MatchString(subject) {
			a.compliant++
			totalCompliant++
			continue
		}
		if len(nonCompliantSamples) < MaxNonCompliantSamples {
			nonCompliantSamples = append(nonCompliantSamples, types.NonCompliantCommit{
				Hash:    shortHash(c.Hash),
				Author:  c.AuthorName,
				Subject: subject,
			})
		}
	}

	perAuthor := make([]types.AuthorComplianceEntry, 0, len(byEmail))
	for email, a := range byEmail {
		perAuthor = append(perAuthor, types.AuthorComplianceEntry{
			Name:          a.name,
			Email:         email,
			Total:         a.total,
			Compliant:     a.compliant,
			CompliancePct: round1(pct(a.compliant, a.total)),
		})
	}
	// Surface the worst offenders first (lowest % at the top), but only
	// among authors with enough volume to be meaningful (>= 3 commits).
	// Authors below the volume floor get sorted by raw compliance count
	// so they don't crowd the drill-down.
	sort.SliceStable(perAuthor, func(i, j int) bool {
		ai, aj := perAuthor[i], perAuthor[j]
		minVolume := 3
		bothEnough := ai.Total >= minVolume && aj.Total >= minVolume
		if bothEnough {
			if ai.CompliancePct != aj.CompliancePct {
				return ai.CompliancePct < aj.CompliancePct
			}
			return ai.Total > aj.Total
		}
		return ai.Total > aj.Total
	})

	return types.ConventionalCommitsResult{
		Total:               len(commits),
		Compliant:           totalCompliant,
		CompliancePct:       round1(pct(totalCompliant, len(commits))),
		PerAuthor:           perAuthor,
		NonCompliantSamples: nonCompliantSamples,
	}
}

func firstLine(s string) string {
	if i := strings.IndexByte(s, '\n'); i >= 0 {
		return s[:i]
	}
	return s
}

func shortHash(h string) string {
	if len(h) > 7 {
		return h[:7]
	}
	return h
}

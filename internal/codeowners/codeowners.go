// Package codeowners parses GitHub-style CODEOWNERS files and maps
// paths to owning teams/users. Supports the subset used in the wild:
// comments, blank lines, multi-owner lines, anchored vs unanchored,
// directory patterns, last-match-wins.
package codeowners

import (
	"os"
	"path/filepath"
	"strings"

	"github.com/bmatcuk/doublestar/v4"
)

type Rule struct {
	OriginalPattern string
	Owners          []string
	globs           []string
}

func (r Rule) Match(path string) bool {
	n := normalizePath(path)
	for _, g := range r.globs {
		ok, err := doublestar.PathMatch(g, n)
		if err == nil && ok {
			return true
		}
	}
	return false
}

type Codeowners struct {
	Rules []Rule
}

// OwnersFor returns the owners for a path using last-match-wins.
func (c *Codeowners) OwnersFor(path string) []string {
	if c == nil {
		return nil
	}
	n := normalizePath(path)
	for i := len(c.Rules) - 1; i >= 0; i-- {
		for _, g := range c.Rules[i].globs {
			if ok, _ := doublestar.PathMatch(g, n); ok {
				return c.Rules[i].Owners
			}
		}
	}
	return nil
}

var candidatePaths = []string{"CODEOWNERS", ".github/CODEOWNERS", "docs/CODEOWNERS"}

// Load attempts to load a CODEOWNERS file from the repo. Returns nil if none found.
func Load(repoPath string) *Codeowners {
	for _, rel := range candidatePaths {
		full := filepath.Join(repoPath, rel)
		data, err := os.ReadFile(full)
		if err != nil {
			continue
		}
		return Parse(string(data))
	}
	return nil
}

// Parse parses CODEOWNERS text into a ruleset.
func Parse(text string) *Codeowners {
	rules := []Rule{}
	for _, raw := range strings.Split(text, "\n") {
		// Strip # comments
		if idx := strings.Index(raw, "#"); idx >= 0 {
			raw = raw[:idx]
		}
		line := strings.TrimSpace(raw)
		if line == "" {
			continue
		}
		parts := strings.Fields(line)
		if len(parts) < 2 {
			continue
		}
		pattern := parts[0]
		owners := parts[1:]
		rules = append(rules, Rule{
			OriginalPattern: pattern,
			Owners:          owners,
			globs:           patternToGlobs(pattern),
		})
	}
	return &Codeowners{Rules: rules}
}

// patternToGlobs converts a CODEOWNERS pattern into one or more doublestar globs.
// Matches the TS implementation for parity.
func patternToGlobs(p string) []string {
	anchored := strings.HasPrefix(p, "/")
	core := p
	if anchored {
		core = core[1:]
	}
	isDir := strings.HasSuffix(core, "/")
	if isDir {
		core = strings.TrimSuffix(core, "/")
	}

	var bases []string
	if anchored {
		bases = append(bases, core)
	} else {
		bases = append(bases, core)
		bases = append(bases, "**/"+core)
	}

	var out []string
	for _, b := range bases {
		if isDir {
			out = append(out, b+"/**")
		} else {
			out = append(out, b)
			out = append(out, b+"/**")
		}
	}
	return out
}

// AggregateOwnersForModule ranks owners in a module by file count and returns the top N.
func AggregateOwnersForModule(filePaths []string, co *Codeowners, topN int) []string {
	if co == nil {
		return []string{}
	}
	counts := map[string]int{}
	for _, p := range filePaths {
		for _, o := range co.OwnersFor(p) {
			counts[o]++
		}
	}
	type kv struct {
		k string
		v int
	}
	ranked := make([]kv, 0, len(counts))
	for k, v := range counts {
		ranked = append(ranked, kv{k, v})
	}
	// simple selection sort — map iteration isn't stable anyway
	for i := 0; i < len(ranked); i++ {
		maxIdx := i
		for j := i + 1; j < len(ranked); j++ {
			if ranked[j].v > ranked[maxIdx].v {
				maxIdx = j
			}
		}
		if maxIdx != i {
			ranked[i], ranked[maxIdx] = ranked[maxIdx], ranked[i]
		}
	}
	if topN > len(ranked) {
		topN = len(ranked)
	}
	out := make([]string, topN)
	for i := 0; i < topN; i++ {
		out[i] = ranked[i].k
	}
	return out
}

func normalizePath(p string) string {
	p = strings.ReplaceAll(p, "\\", "/")
	p = strings.TrimLeft(p, "/")
	return p
}

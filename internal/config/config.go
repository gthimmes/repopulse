// Package config holds default ignore patterns + bug keyword tiers and
// loads per-repo overrides from .repopulserc.
package config

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/bmatcuk/doublestar/v4"
)

// DefaultIgnorePatterns match paths that dominate churn but aren't hand-written code.
var DefaultIgnorePatterns = []string{
	// Lockfiles
	"**/package-lock.json",
	"**/yarn.lock",
	"**/pnpm-lock.yaml",
	"**/Cargo.lock",
	"**/Gemfile.lock",
	"**/poetry.lock",
	"**/composer.lock",
	"**/go.sum",
	"**/*.lock",

	// Build / dist / vendor
	"**/dist/**",
	"**/build/**",
	"**/out/**",
	"**/.next/**",
	"**/target/**",
	"**/node_modules/**",
	"**/vendor/**",
	"**/coverage/**",

	// Minified / sourcemap
	"**/*.min.js",
	"**/*.min.css",
	"**/*.map",

	// Schema dumps / generated typings
	"**/*.generated.*",
	"**/*.g.ts",
	"**/*.gen.ts",
	"**/schemas/*.graphql",
	"**/schema.graphql",
	"**/openapi.json",
	"**/openapi.yaml",
	"**/openapi.yml",
	"**/swagger.json",
	"**/swagger.yaml",
	"**/swagger.yml",
	"**/graphql.schema.json",

	// Ops dashboards / infra-as-data
	"**/grafana/**/*.json",
	"**/prometheus/**/*.json",

	// Binary-ish / data
	"**/*.min.map",
	"**/*.wasm",

	// Typical auto-generated metadata
	"**/auto_generated_metadata*.json",
	"**/__generated__/**",
}

// DefaultBugKeywords tiers.
var DefaultBugKeywords = BugKeywords{
	Chaos:   []string{"revert", "reverted", "rollback", "hotfix", "urgent", "regression", "broken", "broke", "critical", "emergency", "oops", "p0", "p1"},
	Normal:  []string{"fix", "fixes", "fixed", "bug", "patch", "workaround"},
	Routine: []string{"typo", "lint", "format", "formatting", "whitespace", "indent"},
}

type BugKeywords struct {
	Chaos   []string `json:"chaos,omitempty"`
	Normal  []string `json:"normal,omitempty"`
	Routine []string `json:"routine,omitempty"`
}

type RepopulseConfig struct {
	Ignore      []string     `json:"ignore,omitempty"`
	BugKeywords *BugKeywords `json:"bugKeywords,omitempty"`
}

// LoadConfig reads .repopulserc from the repo. Missing file → empty config.
func LoadConfig(repoPath string) RepopulseConfig {
	rcPath := filepath.Join(repoPath, ".repopulserc")
	data, err := os.ReadFile(rcPath)
	if err != nil {
		return RepopulseConfig{}
	}
	var cfg RepopulseConfig
	if err := json.Unmarshal(data, &cfg); err != nil {
		fmt.Fprintf(os.Stderr, "Warning: failed to parse .repopulserc: %v\n", err)
		return RepopulseConfig{}
	}
	return cfg
}

// BuildIgnorePredicate returns a func(path) bool that is true if the path
// matches any of the combined default + repo-config + CLI ignore patterns.
func BuildIgnorePredicate(cfg RepopulseConfig, cliIgnore []string) func(string) bool {
	patterns := append([]string{}, DefaultIgnorePatterns...)
	patterns = append(patterns, cfg.Ignore...)
	patterns = append(patterns, cliIgnore...)

	return func(p string) bool {
		n := normalizePath(p)
		for _, pat := range patterns {
			ok, err := doublestar.PathMatch(pat, n)
			if err == nil && ok {
				return true
			}
		}
		return false
	}
}

// ResolvedBugKeywords merges repo config with defaults. Per-tier arrays in
// `.repopulserc` are appended to the defaults rather than replacing them, so
// a team can add a single house keyword without having to restate the full
// list. To explicitly drop a default, prefix the entry with `!` (e.g.
// `"!fix"` removes "fix" from the normal tier). Duplicates are collapsed.
func ResolvedBugKeywords(cfg RepopulseConfig) BugKeywords {
	out := BugKeywords{
		Chaos:   append([]string{}, DefaultBugKeywords.Chaos...),
		Normal:  append([]string{}, DefaultBugKeywords.Normal...),
		Routine: append([]string{}, DefaultBugKeywords.Routine...),
	}
	if cfg.BugKeywords != nil {
		out.Chaos = mergeKeywords(out.Chaos, cfg.BugKeywords.Chaos)
		out.Normal = mergeKeywords(out.Normal, cfg.BugKeywords.Normal)
		out.Routine = mergeKeywords(out.Routine, cfg.BugKeywords.Routine)
	}
	return out
}

func normalizePath(p string) string {
	return strings.ReplaceAll(p, "\\", "/")
}

// mergeKeywords appends `additions` to `base`, honoring a `!foo` token as
// "drop foo from the result." Order is preserved; duplicates are collapsed.
func mergeKeywords(base, additions []string) []string {
	if len(additions) == 0 {
		return base
	}
	drop := map[string]bool{}
	var adds []string
	for _, a := range additions {
		if strings.HasPrefix(a, "!") {
			drop[strings.ToLower(strings.TrimPrefix(a, "!"))] = true
			continue
		}
		adds = append(adds, a)
	}
	seen := map[string]bool{}
	out := make([]string, 0, len(base)+len(adds))
	for _, k := range append(base, adds...) {
		low := strings.ToLower(k)
		if drop[low] || seen[low] {
			continue
		}
		seen[low] = true
		out = append(out, k)
	}
	return out
}

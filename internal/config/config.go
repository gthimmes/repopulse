// Package config holds default ignore patterns + bug keyword tiers and
// loads per-repo overrides from .moodringrc.
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

type MoodRingConfig struct {
	Ignore      []string     `json:"ignore,omitempty"`
	BugKeywords *BugKeywords `json:"bugKeywords,omitempty"`
}

// LoadConfig reads .moodringrc from the repo. Missing file → empty config.
func LoadConfig(repoPath string) MoodRingConfig {
	rcPath := filepath.Join(repoPath, ".moodringrc")
	data, err := os.ReadFile(rcPath)
	if err != nil {
		return MoodRingConfig{}
	}
	var cfg MoodRingConfig
	if err := json.Unmarshal(data, &cfg); err != nil {
		fmt.Fprintf(os.Stderr, "Warning: failed to parse .moodringrc: %v\n", err)
		return MoodRingConfig{}
	}
	return cfg
}

// BuildIgnorePredicate returns a func(path) bool that is true if the path
// matches any of the combined default + repo-config + CLI ignore patterns.
func BuildIgnorePredicate(cfg MoodRingConfig, cliIgnore []string) func(string) bool {
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

// ResolvedBugKeywords merges repo config with defaults.
func ResolvedBugKeywords(cfg MoodRingConfig) BugKeywords {
	out := BugKeywords{
		Chaos:   DefaultBugKeywords.Chaos,
		Normal:  DefaultBugKeywords.Normal,
		Routine: DefaultBugKeywords.Routine,
	}
	if cfg.BugKeywords != nil {
		if cfg.BugKeywords.Chaos != nil {
			out.Chaos = cfg.BugKeywords.Chaos
		}
		if cfg.BugKeywords.Normal != nil {
			out.Normal = cfg.BugKeywords.Normal
		}
		if cfg.BugKeywords.Routine != nil {
			out.Routine = cfg.BugKeywords.Routine
		}
	}
	return out
}

func normalizePath(p string) string {
	return strings.ReplaceAll(p, "\\", "/")
}

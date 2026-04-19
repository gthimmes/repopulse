package standards

import (
	"sort"
	"strings"

	"repopulse/internal/types"
)

// MaxModulesShown caps the per-module density breakdown.
const MaxModulesShown = 12

// langRule recognises files for one language. Each file under a
// matching language is classified as either a SOURCE or a TEST:
//
//	isTest(path) → true  : counted toward test files for the module
//	isTest(path) → false : counted toward source files
//
// Density is the ratio of test files to source files — a more forgiving
// signal than filename-matching colocation because many teams organise
// tests by action, behaviour, or integration scenario rather than one
// test file per source class.
type langRule struct {
	ext    string
	isTest func(path string) bool
}

func hasAnySubstr(p string, needles []string) bool {
	for _, n := range needles {
		if strings.Contains(p, n) {
			return true
		}
	}
	return false
}

var defaultRules = []langRule{
	{
		ext: ".go",
		isTest: func(p string) bool {
			return strings.HasSuffix(p, "_test.go")
		},
	},
	{
		ext: ".kt",
		isTest: func(p string) bool {
			// Filename-suffix shapes…
			if strings.HasSuffix(p, "Test.kt") ||
				strings.HasSuffix(p, "Tests.kt") ||
				strings.HasSuffix(p, "Spec.kt") ||
				strings.HasSuffix(p, "IT.kt") {
				return true
			}
			// …OR anything under a conventional test source root.
			// Catches helpers (TestHelper.kt, PtoFixtures.kt, etc.)
			// that live alongside test classes.
			return hasAnySubstr(p, []string{"/src/test/", "/src/androidTest/", "/src/integrationTest/", "/src/testFixtures/"})
		},
	},
	{
		ext: ".ts",
		isTest: func(p string) bool {
			if strings.HasSuffix(p, ".test.ts") || strings.HasSuffix(p, ".spec.ts") {
				return true
			}
			if strings.HasSuffix(p, ".d.ts") {
				// Type declaration file — skip entirely (not a test, not a source).
				return false
			}
			return hasAnySubstr(p, []string{"/__tests__/", "/tests/", "/test/"})
		},
	},
	{
		ext: ".tsx",
		isTest: func(p string) bool {
			if strings.HasSuffix(p, ".test.tsx") || strings.HasSuffix(p, ".spec.tsx") {
				return true
			}
			return hasAnySubstr(p, []string{"/__tests__/", "/tests/", "/test/"})
		},
	},
	{
		ext: ".js",
		isTest: func(p string) bool {
			if strings.HasSuffix(p, ".test.js") || strings.HasSuffix(p, ".spec.js") {
				return true
			}
			return hasAnySubstr(p, []string{"/__tests__/", "/tests/", "/test/"})
		},
	},
	{
		ext: ".jsx",
		isTest: func(p string) bool {
			if strings.HasSuffix(p, ".test.jsx") || strings.HasSuffix(p, ".spec.jsx") {
				return true
			}
			return hasAnySubstr(p, []string{"/__tests__/", "/tests/", "/test/"})
		},
	},
	{
		ext: ".py",
		isTest: func(p string) bool {
			base := lastSegment(p)
			if strings.HasPrefix(base, "test_") || strings.HasSuffix(p, "_test.py") {
				return true
			}
			return hasAnySubstr(p, []string{"/tests/", "/test/"})
		},
	},
}

// isSkippedExt excludes TypeScript declaration files — they're neither
// source nor test.
func isSkippedExt(p string) bool {
	return strings.HasSuffix(p, ".d.ts")
}

// ComputeTestDensity calculates the test-to-source file ratio per
// language and per module across the tracked file set. `allFiles` is
// typically the output of `git ls-files` at HEAD.
func ComputeTestDensity(allFiles []string) types.TestDensityResult {
	if len(allFiles) == 0 {
		return types.TestDensityResult{}
	}

	type modAgg struct {
		src, tests int
	}
	byModule := map[string]*modAgg{}
	languages := map[string]struct{}{}
	totalSrc, totalTests := 0, 0

	for _, f := range allFiles {
		if isSkippedExt(f) {
			continue
		}
		rule, ok := matchRule(f)
		if !ok {
			continue
		}
		languages[rule.ext] = struct{}{}

		mod := topLevelModule(f)
		ma, exists := byModule[mod]
		if !exists {
			ma = &modAgg{}
			byModule[mod] = ma
		}

		if rule.isTest(f) {
			ma.tests++
			totalTests++
		} else {
			ma.src++
			totalSrc++
		}
	}

	langs := make([]string, 0, len(languages))
	for l := range languages {
		langs = append(langs, l)
	}
	sort.Strings(langs)

	mods := make([]types.ModuleDensityEntry, 0, len(byModule))
	for name, m := range byModule {
		// Need at least some source files to have a meaningful ratio.
		// A module that's 100% tests (or 0 sources) gets hidden from
		// the per-module breakdown; it's probably a test-only module.
		if m.src < 5 {
			continue
		}
		mods = append(mods, types.ModuleDensityEntry{
			Module:      name,
			SourceFiles: m.src,
			TestFiles:   m.tests,
			DensityPct:  round1(pct(m.tests, m.src)),
		})
	}
	// Worst (lowest density) first; ties broken by source-file count desc.
	sort.SliceStable(mods, func(i, j int) bool {
		if mods[i].DensityPct != mods[j].DensityPct {
			return mods[i].DensityPct < mods[j].DensityPct
		}
		return mods[i].SourceFiles > mods[j].SourceFiles
	})
	if len(mods) > MaxModulesShown {
		mods = mods[:MaxModulesShown]
	}

	return types.TestDensityResult{
		Languages:   langs,
		SourceFiles: totalSrc,
		TestFiles:   totalTests,
		DensityPct:  round1(pct(totalTests, totalSrc)),
		PerModule:   mods,
	}
}

func matchRule(p string) (langRule, bool) {
	for _, r := range defaultRules {
		if strings.HasSuffix(p, r.ext) {
			return r, true
		}
	}
	return langRule{}, false
}

// topLevelModule returns the first path segment, which is what the
// existing module signal uses too — keeps the two views consistent.
func topLevelModule(p string) string {
	if i := strings.IndexAny(p, "/\\"); i > 0 {
		return p[:i]
	}
	return "(root)"
}

func lastSegment(p string) string {
	if i := strings.LastIndexAny(p, "/\\"); i >= 0 {
		return p[i+1:]
	}
	return p
}

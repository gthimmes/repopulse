package standards

import (
	"path"
	"sort"
	"strings"

	"repopulse/internal/types"
)

// MaxMissingSamples caps how many "source-without-test" examples we
// keep on hand for the report drill-down.
const MaxMissingSamples = 25

// MaxModulesShown caps the per-module breakdown — only the worst N
// modules by colocation gap show up by default.
const MaxModulesShown = 12

// langRule describes how a language locates its test sibling.
//
//	srcExt:    file extension that identifies a source file (".kt", ".ts", ...)
//	skipSubstr: don't classify as source if the path already contains any of
//	            these substrings (e.g. ".test.", "/test/", "_test.go") —
//	            keeps test files themselves out of the denominator.
//	hasTest:   given the source path AND the full file set, returns true if
//	            the test sibling exists.
type langRule struct {
	srcExt     string
	skipSubstr []string
	hasTest    func(srcPath string, all map[string]struct{}) bool
}

var defaultRules = []langRule{
	{
		srcExt:     ".go",
		skipSubstr: []string{"_test.go"},
		hasTest: func(p string, all map[string]struct{}) bool {
			testPath := strings.TrimSuffix(p, ".go") + "_test.go"
			_, ok := all[testPath]
			return ok
		},
	},
	{
		srcExt:     ".kt",
		skipSubstr: []string{"Test.kt", "Tests.kt", "/test/", "/androidTest/", "/integrationTest/"},
		hasTest: func(p string, all map[string]struct{}) bool {
			// Sibling: FooTest.kt next to Foo.kt
			base := strings.TrimSuffix(path.Base(p), ".kt")
			dir := path.Dir(p)
			if _, ok := all[path.Join(dir, base+"Test.kt")]; ok {
				return true
			}
			if _, ok := all[path.Join(dir, base+"Tests.kt")]; ok {
				return true
			}
			// Mirror layout: src/main/kotlin/x/y/Foo.kt → src/test/kotlin/x/y/FooTest.kt
			mirrored := strings.Replace(p, "/main/", "/test/", 1)
			mirrored = strings.TrimSuffix(mirrored, ".kt") + "Test.kt"
			if _, ok := all[mirrored]; ok {
				return true
			}
			return false
		},
	},
	{
		srcExt:     ".ts",
		skipSubstr: []string{".test.ts", ".spec.ts", ".d.ts", "/__tests__/", "/test/", "/tests/"},
		hasTest:    typeScriptHasTest,
	},
	{
		srcExt:     ".tsx",
		skipSubstr: []string{".test.tsx", ".spec.tsx", "/__tests__/", "/test/", "/tests/"},
		hasTest:    typeScriptHasTest,
	},
	{
		srcExt:     ".js",
		skipSubstr: []string{".test.js", ".spec.js", "/__tests__/", "/test/", "/tests/"},
		hasTest:    typeScriptHasTest,
	},
	{
		srcExt:     ".jsx",
		skipSubstr: []string{".test.jsx", ".spec.jsx", "/__tests__/", "/test/", "/tests/"},
		hasTest:    typeScriptHasTest,
	},
	{
		srcExt:     ".py",
		skipSubstr: []string{"test_", "_test.py", "/tests/", "/test/"},
		hasTest: func(p string, all map[string]struct{}) bool {
			base := strings.TrimSuffix(path.Base(p), ".py")
			dir := path.Dir(p)
			candidates := []string{
				path.Join(dir, "test_"+base+".py"),
				path.Join(dir, base+"_test.py"),
				path.Join(dir, "tests", "test_"+base+".py"),
			}
			for _, c := range candidates {
				if _, ok := all[c]; ok {
					return true
				}
			}
			return false
		},
	},
}

// typeScriptHasTest covers the JS/TS/TSX/JSX family — sibling
// foo.test.ts / foo.spec.ts / __tests__/foo.test.ts patterns.
func typeScriptHasTest(p string, all map[string]struct{}) bool {
	ext := path.Ext(p)
	base := strings.TrimSuffix(path.Base(p), ext)
	dir := path.Dir(p)
	candidates := []string{
		path.Join(dir, base+".test"+ext),
		path.Join(dir, base+".spec"+ext),
		path.Join(dir, "__tests__", base+".test"+ext),
		path.Join(dir, "__tests__", base+".spec"+ext),
	}
	for _, c := range candidates {
		if _, ok := all[c]; ok {
			return true
		}
	}
	return false
}

// ComputeTestColocation walks the tracked file set, classifies sources
// by language using defaultRules, and reports per-module + per-language
// colocation coverage.
func ComputeTestColocation(allFiles []string) types.TestColocationResult {
	if len(allFiles) == 0 {
		return types.TestColocationResult{}
	}

	all := make(map[string]struct{}, len(allFiles))
	for _, f := range allFiles {
		all[f] = struct{}{}
	}

	totalSrc, totalCoLoc := 0, 0
	languages := map[string]struct{}{}
	type modAgg struct {
		src, coloc int
	}
	byModule := map[string]*modAgg{}
	var missing []string

	for _, f := range allFiles {
		rule, ok := matchRule(f)
		if !ok {
			continue
		}

		// Classify whether this file IS a source file (not a test itself).
		if isExcludedAsTest(f, rule) {
			continue
		}

		languages[rule.srcExt] = struct{}{}
		totalSrc++
		mod := topLevelModule(f)
		ma, ok := byModule[mod]
		if !ok {
			ma = &modAgg{}
			byModule[mod] = ma
		}
		ma.src++

		if rule.hasTest(f, all) {
			totalCoLoc++
			ma.coloc++
		} else if len(missing) < MaxMissingSamples {
			missing = append(missing, f)
		}
	}

	langs := make([]string, 0, len(languages))
	for l := range languages {
		langs = append(langs, l)
	}
	sort.Strings(langs)

	mods := make([]types.ModuleColocationEntry, 0, len(byModule))
	for name, m := range byModule {
		// Only surface modules with ≥5 source files; below that the % swings
		// too much per file to mean anything.
		if m.src < 5 {
			continue
		}
		mods = append(mods, types.ModuleColocationEntry{
			Module:      name,
			SourceFiles: m.src,
			Colocated:   m.coloc,
			CoveragePct: round1(pct(m.coloc, m.src)),
		})
	}
	// Worst coverage first; ties by source-file count desc.
	sort.SliceStable(mods, func(i, j int) bool {
		if mods[i].CoveragePct != mods[j].CoveragePct {
			return mods[i].CoveragePct < mods[j].CoveragePct
		}
		return mods[i].SourceFiles > mods[j].SourceFiles
	})
	if len(mods) > MaxModulesShown {
		mods = mods[:MaxModulesShown]
	}

	return types.TestColocationResult{
		Languages:      langs,
		SourceFiles:    totalSrc,
		Colocated:      totalCoLoc,
		CoveragePct:    round1(pct(totalCoLoc, totalSrc)),
		PerModule:      mods,
		MissingSamples: missing,
	}
}

func matchRule(p string) (langRule, bool) {
	for _, r := range defaultRules {
		if strings.HasSuffix(p, r.srcExt) {
			return r, true
		}
	}
	return langRule{}, false
}

func isExcludedAsTest(p string, r langRule) bool {
	for _, s := range r.skipSubstr {
		if strings.Contains(p, s) {
			return true
		}
	}
	return false
}

// topLevelModule returns the first path segment, which is what the
// existing module signal uses too — keeps the two views consistent.
func topLevelModule(p string) string {
	if i := strings.IndexAny(p, "/\\"); i > 0 {
		return p[:i]
	}
	return "(root)"
}

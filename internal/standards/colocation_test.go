package standards

import "testing"

func findModule(mods []struct {
	Module      string
	SourceFiles int
	TestFiles   int
	DensityPct  float64
}, name string) bool {
	for _, m := range mods {
		if m.Module == name {
			return true
		}
	}
	return false
}

func TestDensity_GoCountsTestsAndSourcesSeparately(t *testing.T) {
	files := []string{
		"foo/bar.go",
		"foo/bar_test.go",
		"foo/baz.go",
	}
	r := ComputeTestDensity(files)
	if r.SourceFiles != 2 {
		t.Errorf("want 2 sources, got %d", r.SourceFiles)
	}
	if r.TestFiles != 1 {
		t.Errorf("want 1 test, got %d", r.TestFiles)
	}
	// 1 test / 2 sources = 50%
	if r.DensityPct != 50 {
		t.Errorf("want density 50%%, got %v", r.DensityPct)
	}
}

func TestDensity_KotlinTestInAnyLocationCounts(t *testing.T) {
	// Real-world case: tests live under src/test/ with a restructured
	// package path vs their source (e.g. source package is `api/foo`
	// but test package is `foo/api`). Filename-suffix match catches
	// them regardless of directory layout.
	files := []string{
		"svc/src/main/kotlin/com/example/api/foo/controller/FooController.kt",
		"svc/src/test/kotlin/com/example/foo/api/controller/FooControllerTest.kt",
	}
	r := ComputeTestDensity(files)
	if r.SourceFiles != 1 {
		t.Errorf("want 1 source, got %d", r.SourceFiles)
	}
	if r.TestFiles != 1 {
		t.Errorf("want 1 test, got %d", r.TestFiles)
	}
}

func TestDensity_KotlinActionSplitTestsCount(t *testing.T) {
	// A common real-world pattern: tests are split by action so no
	// one-to-one source↔test mapping exists. Density metric should
	// still count these as tests.
	files := []string{
		"svc/src/main/kotlin/com/example/FooService.kt",
		"svc/src/test/kotlin/com/example/FooServiceCreateTest.kt",
		"svc/src/test/kotlin/com/example/FooServiceDeleteTest.kt",
		"svc/src/test/kotlin/com/example/ApiScenarioTest.kt",
	}
	r := ComputeTestDensity(files)
	if r.SourceFiles != 1 {
		t.Errorf("want 1 source, got %d", r.SourceFiles)
	}
	if r.TestFiles != 3 {
		t.Errorf("want 3 tests, got %d", r.TestFiles)
	}
}

func TestDensity_KotlinTestHelperInTestDirCounts(t *testing.T) {
	// TestHelper classes in /src/test/ aren't strictly tests but they're
	// test-side code. Path-based detection catches them.
	files := []string{
		"m/src/main/kotlin/Foo.kt",
		"m/src/test/kotlin/PtoEntryTestHelper.kt",
	}
	r := ComputeTestDensity(files)
	if r.TestFiles != 1 {
		t.Errorf("want 1 test (helper in test dir), got %d", r.TestFiles)
	}
}

func TestDensity_TypeScriptSpecAndTest(t *testing.T) {
	files := []string{
		"src/foo.ts", "src/foo.test.ts",
		"src/bar.ts", "src/bar.spec.ts",
		"src/baz.ts",
	}
	r := ComputeTestDensity(files)
	if r.SourceFiles != 3 {
		t.Errorf("want 3 sources, got %d", r.SourceFiles)
	}
	if r.TestFiles != 2 {
		t.Errorf("want 2 tests, got %d", r.TestFiles)
	}
}

func TestDensity_TypeScript_SkipDtsEntirely(t *testing.T) {
	files := []string{
		"src/foo.d.ts", // neither source nor test
		"src/bar.ts",
	}
	r := ComputeTestDensity(files)
	if r.SourceFiles != 1 {
		t.Errorf(".d.ts files should be skipped; want 1 source, got %d", r.SourceFiles)
	}
	if r.TestFiles != 0 {
		t.Errorf("want 0 tests, got %d", r.TestFiles)
	}
}

func TestDensity_PerModuleSortedWorstFirst(t *testing.T) {
	// modA: 5 sources, 1 test (20%)
	// modB: 5 sources, 4 tests (80%)
	files := []string{
		"modA/a.go", "modA/b.go", "modA/c.go", "modA/d.go", "modA/e.go", "modA/a_test.go",
		"modB/p.go", "modB/q.go", "modB/r.go", "modB/s.go", "modB/t.go",
		"modB/p_test.go", "modB/q_test.go", "modB/r_test.go", "modB/s_test.go",
	}
	r := ComputeTestDensity(files)
	if len(r.PerModule) != 2 {
		t.Fatalf("want 2 modules, got %d", len(r.PerModule))
	}
	if r.PerModule[0].Module != "modA" {
		t.Errorf("worst-first should put modA first, got %s", r.PerModule[0].Module)
	}
}

func TestDensity_SmallModulesHiddenFromBreakdown(t *testing.T) {
	files := []string{"modA/a.go", "modA/b.go", "modA/c.go"}
	r := ComputeTestDensity(files)
	if r.SourceFiles != 3 {
		t.Errorf("global count should be 3, got %d", r.SourceFiles)
	}
	if len(r.PerModule) != 0 {
		t.Errorf("modules < 5 sources should be hidden from breakdown")
	}
}

func TestDensity_AllowsOver100Pct(t *testing.T) {
	// Test-heavy codebase: more tests than sources → density > 100%.
	files := []string{
		"m/a.go", "m/b.go", "m/c.go", "m/d.go", "m/e.go",
		"m/a_test.go", "m/b_test.go", "m/c_test.go", "m/d_test.go", "m/e_test.go",
		"m/integration_test.go", "m/bench_test.go",
	}
	r := ComputeTestDensity(files)
	// 7 tests / 5 sources = 140%
	if r.DensityPct != 140 {
		t.Errorf("want 140%%, got %v", r.DensityPct)
	}
}

func TestDensity_EmptyInput(t *testing.T) {
	r := ComputeTestDensity(nil)
	if r.SourceFiles != 0 || r.TestFiles != 0 {
		t.Errorf("empty should be zeroed, got %+v", r)
	}
}

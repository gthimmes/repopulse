package standards

import "testing"

func TestColocation_GoSiblingMatch(t *testing.T) {
	files := []string{
		"foo/bar.go",
		"foo/bar_test.go",
		"foo/baz.go", // missing test
	}
	r := ComputeTestColocation(files)
	if r.SourceFiles != 2 {
		t.Errorf("want 2 source files, got %d", r.SourceFiles)
	}
	if r.Colocated != 1 {
		t.Errorf("want 1 colocated, got %d", r.Colocated)
	}
}

func TestColocation_KotlinMirroredLayout(t *testing.T) {
	files := []string{
		"src/main/kotlin/com/foo/Bar.kt",
		"src/test/kotlin/com/foo/BarTest.kt",
		"src/main/kotlin/com/foo/Baz.kt", // missing test
	}
	r := ComputeTestColocation(files)
	if r.SourceFiles != 2 {
		t.Errorf("want 2 source, got %d", r.SourceFiles)
	}
	if r.Colocated != 1 {
		t.Errorf("want 1 colocated (mirrored), got %d", r.Colocated)
	}
}

func TestColocation_KotlinSiblingTest(t *testing.T) {
	files := []string{
		"app/Foo.kt",
		"app/FooTest.kt",
	}
	r := ComputeTestColocation(files)
	if r.Colocated != 1 {
		t.Errorf("want sibling FooTest.kt to count, got %d", r.Colocated)
	}
}

func TestColocation_TypeScriptSpecAndTest(t *testing.T) {
	files := []string{
		"src/foo.ts",
		"src/foo.test.ts",
		"src/bar.ts",
		"src/bar.spec.ts",
		"src/baz.ts", // missing
	}
	r := ComputeTestColocation(files)
	if r.SourceFiles != 3 {
		t.Errorf("want 3 source, got %d", r.SourceFiles)
	}
	if r.Colocated != 2 {
		t.Errorf("want 2 colocated, got %d", r.Colocated)
	}
}

func TestColocation_TypeScript_TestsThemselvesNotInDenominator(t *testing.T) {
	// .test.ts and .spec.ts files should be EXCLUDED from the source count
	files := []string{
		"src/foo.test.ts",
		"src/foo.spec.ts",
		"src/foo.d.ts",
	}
	r := ComputeTestColocation(files)
	if r.SourceFiles != 0 {
		t.Errorf("test/d files should not count as source, got %d", r.SourceFiles)
	}
}

func TestColocation_PerModuleSortedWorstFirst(t *testing.T) {
	// modA: 5 sources, 1 with test (20%)
	// modB: 5 sources, 4 with test (80%)
	files := []string{
		"modA/a.go", "modA/b.go", "modA/c.go", "modA/d.go", "modA/e.go", "modA/a_test.go",
		"modB/p.go", "modB/q.go", "modB/r.go", "modB/s.go", "modB/t.go",
		"modB/p_test.go", "modB/q_test.go", "modB/r_test.go", "modB/s_test.go",
	}
	r := ComputeTestColocation(files)
	if len(r.PerModule) != 2 {
		t.Fatalf("want 2 modules in breakdown, got %d", len(r.PerModule))
	}
	if r.PerModule[0].Module != "modA" {
		t.Errorf("worst-first should put modA first, got %s", r.PerModule[0].Module)
	}
}

func TestColocation_SmallModulesExcludedFromBreakdown(t *testing.T) {
	// Only 3 source files in modA — below the per-module threshold of 5
	files := []string{
		"modA/a.go", "modA/b.go", "modA/c.go",
	}
	r := ComputeTestColocation(files)
	if r.SourceFiles != 3 {
		t.Errorf("global source count should still be 3, got %d", r.SourceFiles)
	}
	if len(r.PerModule) != 0 {
		t.Errorf("modules below threshold should be hidden, got %d modules", len(r.PerModule))
	}
}

func TestColocation_LanguagesDetected(t *testing.T) {
	files := []string{"a.go", "b.kt", "c.ts", "d.py"}
	r := ComputeTestColocation(files)
	if len(r.Languages) != 4 {
		t.Errorf("want 4 languages, got %v", r.Languages)
	}
}

func TestColocation_EmptyInput(t *testing.T) {
	r := ComputeTestColocation(nil)
	if r.SourceFiles != 0 {
		t.Errorf("empty should be zeroed, got %+v", r)
	}
}

// fixture-gen writes a deterministic UI-fixture HTML report to the given path.
// Invoked by the Playwright e2e tests as their fixture builder — replaces the
// old TS `tests/e2e/fixtures.ts:writeFixtureReport`.
package main

import (
	"fmt"
	"os"
	"path/filepath"

	"repopulse/internal/fixtures"
	"repopulse/internal/render"
)

func main() {
	if len(os.Args) < 2 {
		fmt.Fprintln(os.Stderr, "Usage: fixture-gen <output-html-path>")
		os.Exit(2)
	}
	outPath, err := filepath.Abs(os.Args[1])
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
	if err := os.MkdirAll(filepath.Dir(outPath), 0755); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
	html := render.RenderHTML(fixtures.UIMoodResult(), fixtures.UIMeta(), nil, nil)
	if err := os.WriteFile(outPath, []byte(html), 0644); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
	fmt.Println(outPath)
}

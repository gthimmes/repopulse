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
	"repopulse/internal/types"
)

func main() {
	args := os.Args[1:]
	if len(args) < 1 {
		fmt.Fprintln(os.Stderr, "Usage: fixture-gen <output-html-path> [--enriched]")
		os.Exit(2)
	}
	outArg := args[0]
	enriched := false
	for _, a := range args[1:] {
		if a == "--enriched" {
			enriched = true
		}
	}

	outPath, err := filepath.Abs(outArg)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
	if err := os.MkdirAll(filepath.Dir(outPath), 0755); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}

	mood := fixtures.UIMoodResult()
	if enriched {
		mood.Enrichment = sampleEnrichment()
	}
	html := render.RenderHTML(mood, fixtures.UIMeta(), nil, nil)
	if err := os.WriteFile(outPath, []byte(html), 0644); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
	fmt.Println(outPath)
}

// sampleEnrichment is the Plank-2 Layer-B fixture used by the
// enrichment Playwright test. Mirrors what a real Mode-B/C run would
// produce for the deterministic UI fixture: one bullet per kind, a
// standards verdict, one drift reading anchored to a fixture email.
func sampleEnrichment() *types.EnrichmentResult {
	return &types.EnrichmentResult{
		Type:          "enrichment",
		SchemaVersion: 1,
		Source:        "claude-code-skill",
		Model:         "claude-fixture",
		GeneratedAt:   "2026-04-19T03:14:00Z",
		Narrative: []types.EnrichedNarrativeBullet{
			{Kind: "alert", Text: "Bug-tier ratio above the calm band, with one chaos commit reverted in the past week."},
			{Kind: "warn", Text: "Top 3 contributors carry 92% of LOC — bus-factor remains thin."},
			{Kind: "good", Text: "Commit compliance is at 100% across every contributor in the window."},
			{Kind: "info", Text: "Activity spikes around the 12th align with the ledger refactor; investigate whether load-shed before the next release."},
		},
		Standards: &types.StandardsVerdict{
			Headline:    "Standards holding; concentration is the open thread",
			Summary:     "Conventional-commit compliance is essentially uniform. Test density is healthy in payments/ but lower in auth/ — worth a check before the next big change there.",
			Suggestions: []string{"Pair Carol with one of the auth/ regulars on the next non-trivial change there."},
		},
		Drift: []types.DriftInterpretation{
			{
				Email:      "alice@example.com",
				Reading:    "Cadence and after-hours load both crept up this window — usually a signal of a tight delivery, worth checking in their next 1:1.",
				Suggestion: "ask whether the ledger work is bounded or open-ended",
			},
		},
	}
}

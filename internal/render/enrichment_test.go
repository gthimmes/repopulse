package render

import (
	"strings"
	"testing"

	"repopulse/internal/types"
)

func TestRenderEnrichmentReturnsEmptyWhenNil(t *testing.T) {
	if got := renderEnrichment(types.MoodResult{}); got != "" {
		t.Errorf("expected empty render when no enrichment, got %q", got)
	}
}

func TestRenderEnrichmentEmitsAITagAndContent(t *testing.T) {
	data := types.MoodResult{
		Enrichment: &types.EnrichmentResult{
			Source: "claude-code-skill",
			Model:  "claude-test",
			Narrative: []types.EnrichedNarrativeBullet{
				{Kind: "info", Text: "trend is steady"},
				{Kind: "warn", Text: "watch test density"},
			},
			Standards: &types.StandardsVerdict{
				Headline:    "compliance solid",
				Summary:     "team is consistent",
				Suggestions: []string{"keep doing what you're doing"},
			},
		},
	}
	got := renderEnrichment(data)
	mustContain := []string{
		"AI-GENERATED",
		"claude-code-skill",
		"claude-test",
		"trend is steady",
		"watch test density",
		"compliance solid",
		"keep doing what",
	}
	for _, want := range mustContain {
		if !strings.Contains(got, want) {
			t.Errorf("render missing %q in output:\n%s", want, got)
		}
	}
}

func TestRenderEnrichmentDriftMatchesContributorByEmail(t *testing.T) {
	data := types.MoodResult{
		Signals: types.Signals{
			Authors: types.AuthorSignal{
				Contributors: []types.AuthorEntry{
					{Name: "Alex Doe", Email: "alex@example.com"},
				},
			},
		},
		Enrichment: &types.EnrichmentResult{
			Drift: []types.DriftInterpretation{
				{
					Email:      "ALEX@example.com", // case-insensitive match
					Reading:    "load picked up sharply this window",
					Suggestion: "ask in their next 1:1",
				},
			},
		},
	}
	got := renderEnrichment(data)
	if !strings.Contains(got, "Alex Doe") {
		t.Errorf("drift entry should resolve email to display name, got:\n%s", got)
	}
	if !strings.Contains(got, "load picked up sharply") {
		t.Errorf("drift reading missing from output:\n%s", got)
	}
	if !strings.Contains(got, "ask in their next 1:1") {
		t.Errorf("drift suggestion missing from output:\n%s", got)
	}
}

func TestRenderEnrichmentReturnsEmptyForEmptyEnrichment(t *testing.T) {
	data := types.MoodResult{
		Enrichment: &types.EnrichmentResult{Source: "x"}, // no narrative, standards, drift
	}
	if got := renderEnrichment(data); got != "" {
		t.Errorf("expected empty render when enrichment has no usable content, got %q", got)
	}
}

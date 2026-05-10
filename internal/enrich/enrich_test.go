package enrich

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"repopulse/internal/types"
)

func sampleSnapshot() (types.MoodResult, types.RepoMeta) {
	cov := 70
	meta := types.RepoMeta{
		RepoName:        "demo",
		RepoPath:        "/tmp/demo",
		AnalyzedCommits: 42,
		WindowDays:      30,
		WindowStart:     time.Date(2026, 4, 1, 0, 0, 0, 0, time.UTC),
		WindowEnd:       time.Date(2026, 5, 1, 0, 0, 0, 0, time.UTC),
		GeneratedAt:     time.Date(2026, 5, 1, 12, 0, 0, 0, time.UTC),
	}
	mood := types.MoodResult{
		Mood:           types.MoodAnxious,
		CompositeScore: 55,
		Breakdown: types.MoodBreakdown{
			CommitFrequency: 30, FileChurn: 60, BugRatio: 50, Coverage: &cov, Authors: 40,
		},
		Signals: types.Signals{
			BugRatio: types.BugSignal{
				ChaosCommitCount: 1, NormalFixCount: 5, RoutineFixCount: 2,
				TotalCommits: 42, Ratio: 0.19,
			},
			FileChurn: types.ChurnSignal{
				TotalFilesTouched: 80, HighChurnFileCount: 4, TotalLinesChanged: 5000, LinesPerDay: 166,
			},
			Authors: types.AuthorSignal{
				TotalAuthors: 4, WeekendNightPct: 22, BusFactorTop1Pct: 60, BusFactorTop3Pct: 92,
				NewContributorChurnPct: 8,
			},
			Modules: types.ModuleSignal{Modules: []types.ModuleEntry{
				{Name: "internal", Score: 70, Mood: types.MoodAnxious, Commits: 30, LinesChanged: 2000, BugRatio: 0.10},
			}},
			Standards: types.StandardsSignal{
				ConventionalCommits: types.ConventionalCommitsResult{
					Total: 42, Compliant: 38, CompliancePct: 90.5,
				},
				TestDensity: types.TestDensityResult{
					SourceFiles: 60, TestFiles: 40, DensityPct: 66.7,
				},
			},
			AuthorDrift: types.AuthorDriftSignal{
				CurrentDays: 30, BaselineDays: 180,
				Authors: []types.AuthorDrift{
					{
						Name: "Alex", Email: "alex@example.com", CommitsCurrent: 12,
						CommitsPerWeekCurrent: 3, CommitsPerWeekBaseline: 1.5, CommitsDeltaPct: 100,
						WeekendNightCurrent: 40, WeekendNightBaseline: 15, WeekendNightDeltaPP: 25,
						FixRatioCurrent: 50, FixRatioBaseline: 20, FixRatioDeltaPP: 30,
						Flags: []types.DriftFlag{
							{Kind: "weekend-night-up", Severity: "watch", Text: "off-hours commit share doubled"},
						},
					},
				},
			},
		},
	}
	return mood, meta
}

func TestBuildPromptIncludesAggregateSignals(t *testing.T) {
	mood, meta := sampleSnapshot()
	prompt := BuildPrompt(mood, meta)
	required := []string{
		"demo",                          // repo name
		"55/100",                        // composite score
		"anxious",                       // mood band
		"alex@example.com",              // drift author
		"off-hours commit share doubled", // drift flag text
		"42",                            // analyzed commits
		"commit compliance",             // standards subhead
		"test density",                  // standards subhead
	}
	for _, want := range required {
		if !strings.Contains(strings.ToLower(prompt), strings.ToLower(want)) {
			t.Errorf("prompt missing expected fragment %q\nprompt:\n%s", want, prompt)
		}
	}
}

func TestBuildPromptOmitsDriftAuthorsWithoutFlags(t *testing.T) {
	mood, meta := sampleSnapshot()
	mood.Signals.AuthorDrift.Authors = append(mood.Signals.AuthorDrift.Authors, types.AuthorDrift{
		Name:  "Quiet",
		Email: "quiet@example.com",
		// No flags — should be filtered out.
	})
	prompt := BuildPrompt(mood, meta)
	if strings.Contains(prompt, "quiet@example.com") {
		t.Errorf("prompt should not mention drift authors with no flags")
	}
}

func TestParseModelJSONHandlesPlainAndFenced(t *testing.T) {
	cases := map[string]string{
		"plain": `{"narrative":[{"kind":"info","text":"all good"}]}`,
		"fenced": "```json\n{\"narrative\":[{\"kind\":\"info\",\"text\":\"all good\"}]}\n```",
		"prefaced": "Sure, here you go: {\"narrative\":[{\"kind\":\"info\",\"text\":\"all good\"}]}",
	}
	for name, input := range cases {
		t.Run(name, func(t *testing.T) {
			r, err := ParseModelJSON(input)
			if err != nil {
				t.Fatalf("parse failed: %v", err)
			}
			if len(r.Narrative) != 1 || r.Narrative[0].Text != "all good" {
				t.Fatalf("unexpected narrative: %+v", r.Narrative)
			}
			if r.Type != "enrichment" || r.SchemaVersion != SchemaVersion {
				t.Errorf("defaults not filled in: type=%q ver=%d", r.Type, r.SchemaVersion)
			}
		})
	}
}

func TestParseModelJSONRejectsNonJSON(t *testing.T) {
	if _, err := ParseModelJSON("not json at all"); err == nil {
		t.Fatal("expected error on non-JSON input")
	}
}

func TestInputHashChangesWhenSnapshotChanges(t *testing.T) {
	mood, meta := sampleSnapshot()
	h1 := InputHash(mood, meta, "claude-x")
	mood.CompositeScore = 99
	h2 := InputHash(mood, meta, "claude-x")
	if h1 == h2 {
		t.Errorf("hash should change when snapshot mutates")
	}
}

func TestInputHashChangesWithModel(t *testing.T) {
	mood, meta := sampleSnapshot()
	h1 := InputHash(mood, meta, "claude-x")
	h2 := InputHash(mood, meta, "claude-y")
	if h1 == h2 {
		t.Errorf("hash should change when model id changes — cache must not serve cross-model")
	}
}

func TestRunCachesSuccessfulResponse(t *testing.T) {
	mood, meta := sampleSnapshot()

	calls := 0
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		calls++
		body := map[string]any{
			"content": []map[string]any{
				{"type": "text", "text": `{"narrative":[{"kind":"good","text":"steady"}]}`},
			},
		}
		w.Header().Set("content-type", "application/json")
		_ = json.NewEncoder(w).Encode(body)
	}))
	defer srv.Close()

	cacheDir := filepath.Join(t.TempDir(), "enrichment-cache")

	first, err := Run(context.Background(), mood, meta, Options{
		APIKey:      "x",
		Model:       "test-model",
		MessagesURL: srv.URL,
		CacheDir:    cacheDir,
	})
	if err != nil {
		t.Fatalf("first run failed: %v", err)
	}
	if first.Source != "anthropic-api" {
		t.Errorf("source not set: %q", first.Source)
	}
	if first.Model != "test-model" {
		t.Errorf("model not set: %q", first.Model)
	}

	// Second call should hit the cache and not the server.
	second, err := Run(context.Background(), mood, meta, Options{
		APIKey:      "x",
		Model:       "test-model",
		MessagesURL: srv.URL,
		CacheDir:    cacheDir,
	})
	if err != nil {
		t.Fatalf("second run failed: %v", err)
	}
	if calls != 1 {
		t.Errorf("expected 1 server call, got %d — cache miss?", calls)
	}
	if second.InputHash != first.InputHash {
		t.Errorf("cached result should preserve input hash")
	}
}

func TestRunSurfacesAPIErrors(t *testing.T) {
	mood, meta := sampleSnapshot()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, `{"error":"forbidden"}`, http.StatusForbidden)
	}))
	defer srv.Close()
	_, err := Run(context.Background(), mood, meta, Options{
		APIKey:      "x",
		MessagesURL: srv.URL,
	})
	if err == nil || !strings.Contains(err.Error(), "403") {
		t.Errorf("expected 403 in error, got %v", err)
	}
}

func TestRunRequiresAPIKey(t *testing.T) {
	mood, meta := sampleSnapshot()
	if _, err := Run(context.Background(), mood, meta, Options{}); err == nil {
		t.Fatal("expected error when no API key")
	}
}

func TestLoadAndWriteRoundTrip(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "enriched.json")
	want := &types.EnrichmentResult{
		Type:          "enrichment",
		SchemaVersion: SchemaVersion,
		Source:        "claude-code-skill",
		Model:         "claude-test",
		GeneratedAt:   "2026-05-09T22:00:00Z",
		Narrative:     []types.EnrichedNarrativeBullet{{Kind: "info", Text: "hi"}},
	}
	if err := WriteToFile(path, want); err != nil {
		t.Fatal(err)
	}
	got, err := LoadFromFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if got.Source != want.Source || got.Model != want.Model {
		t.Errorf("round-trip lost data: got %+v", got)
	}
	if len(got.Narrative) != 1 || got.Narrative[0].Text != "hi" {
		t.Errorf("narrative lost in round-trip: %+v", got.Narrative)
	}
	if _, err := os.Stat(path); err != nil {
		t.Errorf("file should exist: %v", err)
	}
}

package snapshots

import (
	"encoding/json"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"testing"
	"time"

	"repopulse/internal/compare"
	"repopulse/internal/types"
)

func makeSnap(score int, when time.Time) compare.ReportSnapshot {
	return compare.ReportSnapshot{
		GeneratedAt:     when.UTC().Format(time.RFC3339Nano),
		RepoName:        "t",
		WindowDays:      30,
		AnalyzedCommits: 10,
		MoodResult: types.MoodResult{
			Mood:           types.MoodCalm,
			CompositeScore: score,
			Breakdown: types.MoodBreakdown{
				CommitFrequency: score,
				FileChurn:       score,
				BugRatio:        score,
				Authors:         score,
			},
		},
	}
}

func TestSaveCreatesDirAndFileAndGitignore(t *testing.T) {
	repo := t.TempDir()
	path, err := Save(repo, makeSnap(40, time.Now()))
	if err != nil {
		t.Fatalf("Save: %v", err)
	}
	if _, err := os.Stat(path); err != nil {
		t.Fatalf("snapshot file missing: %v", err)
	}
	if !strings.Contains(path, filepath.Join(".repopulse", "snapshots")) {
		t.Fatalf("unexpected snapshot path: %s", path)
	}
	gi := filepath.Join(repo, ".repopulse", ".gitignore")
	b, err := os.ReadFile(gi)
	if err != nil {
		t.Fatalf("missing .gitignore: %v", err)
	}
	if strings.TrimSpace(string(b)) != "*" {
		t.Fatalf("gitignore should be `*`, got %q", string(b))
	}
}

func TestLoadReturnsSortedSnapshots(t *testing.T) {
	repo := t.TempDir()
	// Write out of order (saved as different files so no collision)
	times := []time.Time{
		time.Date(2026, 1, 3, 10, 0, 0, 0, time.UTC),
		time.Date(2026, 1, 1, 10, 0, 0, 0, time.UTC),
		time.Date(2026, 1, 2, 10, 0, 0, 0, time.UTC),
	}
	for i, ts := range times {
		snap := makeSnap(i*10, ts)
		// bypass Save's "now" filename — write directly to mimic historical entries
		dir := filepath.Join(repo, Dir)
		_ = os.MkdirAll(dir, 0755)
		b, _ := json.MarshalIndent(snap, "", "  ")
		_ = os.WriteFile(filepath.Join(dir, ts.Format("2006-01-02T150405Z")+".json"), b, 0644)
	}
	got, err := Load(repo)
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if len(got) != 3 {
		t.Fatalf("want 3 snapshots, got %d", len(got))
	}
	// Ensure ascending by GeneratedAt
	for i := 1; i < len(got); i++ {
		if got[i-1].GeneratedAt > got[i].GeneratedAt {
			t.Fatalf("snapshots not sorted: %v", got)
		}
	}
}

func TestLoadMissingDirIsNotAnError(t *testing.T) {
	repo := t.TempDir()
	got, err := Load(repo)
	if err != nil {
		t.Fatalf("Load empty: %v", err)
	}
	if len(got) != 0 {
		t.Fatalf("want 0 snapshots, got %d", len(got))
	}
}

func TestPruneKeepsNewestN(t *testing.T) {
	repo := t.TempDir()
	dir := filepath.Join(repo, Dir)
	_ = os.MkdirAll(dir, 0755)
	// Fabricate 10 dated snapshots, each with a distinct filename timestamp
	for i := 0; i < 10; i++ {
		ts := time.Date(2026, 1, 1+i, 10, 0, 0, 0, time.UTC)
		snap := makeSnap(i, ts)
		b, _ := json.MarshalIndent(snap, "", "  ")
		_ = os.WriteFile(filepath.Join(dir, ts.Format("2006-01-02T150405Z")+".json"), b, 0644)
	}
	if err := prune(dir, 4); err != nil {
		t.Fatalf("prune: %v", err)
	}
	entries, _ := os.ReadDir(dir)
	var names []string
	for _, e := range entries {
		if strings.HasSuffix(e.Name(), ".json") {
			names = append(names, e.Name())
		}
	}
	if len(names) != 4 {
		t.Fatalf("want 4 after prune, got %d", len(names))
	}
	sort.Strings(names)
	// The 4 newest correspond to Jan 7..10 (indices 6..9)
	want := []string{
		"2026-01-07T100000Z.json",
		"2026-01-08T100000Z.json",
		"2026-01-09T100000Z.json",
		"2026-01-10T100000Z.json",
	}
	for i, w := range want {
		if names[i] != w {
			t.Fatalf("kept wrong files: got %v, want %v", names, want)
		}
	}
}

func TestSameSecondCollisionGetsSuffix(t *testing.T) {
	repo := t.TempDir()
	// Two back-to-back saves within the same second: the second should
	// pick up a -2 suffix rather than overwrite.
	p1, err := Save(repo, makeSnap(1, time.Now()))
	if err != nil {
		t.Fatalf("Save 1: %v", err)
	}
	p2, err := Save(repo, makeSnap(2, time.Now()))
	if err != nil {
		t.Fatalf("Save 2: %v", err)
	}
	if p1 == p2 {
		t.Fatalf("collision not resolved: both wrote to %s", p1)
	}
	if !strings.Contains(filepath.Base(p2), "-2.json") &&
		!strings.Contains(filepath.Base(p1), "-2.json") {
		// At least one of the two filenames must carry a numeric suffix;
		// strconv used only to prove the -N is a legit number.
		tail := strings.TrimSuffix(filepath.Base(p2), ".json")
		parts := strings.Split(tail, "-")
		if _, err := strconv.Atoi(parts[len(parts)-1]); err != nil {
			t.Fatalf("expected numeric suffix, got %s", filepath.Base(p2))
		}
	}
}

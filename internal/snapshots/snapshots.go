// Package snapshots manages the persistent `.repopulse/snapshots/` store
// inside the analyzed repository — a time series of dated JSON snapshots
// that powers the trend chart in the HTML report.
package snapshots

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"repopulse/internal/compare"
)

// Dir is the relative path inside the analyzed repo where snapshots live.
const Dir = ".repopulse/snapshots"

// MaxSnapshots is the default rolling cap. Older snapshots beyond this
// count are pruned at write time.
const MaxSnapshots = 365

// Save writes a snapshot for the given repo, prunes older entries past
// the cap, and returns the absolute path written. It also lays down a
// `.repopulse/.gitignore` (containing `*`) on first run so users do not
// have to remember to ignore the directory themselves.
func Save(repoPath string, snap compare.ReportSnapshot) (string, error) {
	dir := filepath.Join(repoPath, Dir)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return "", err
	}
	if err := ensureGitignore(filepath.Dir(dir)); err != nil {
		return "", err
	}

	ts := time.Now().UTC().Format("2006-01-02T150405Z")
	path := filepath.Join(dir, ts+".json")
	// Same-second collisions get a -N suffix.
	for i := 2; ; i++ {
		if _, err := os.Stat(path); os.IsNotExist(err) {
			break
		}
		path = filepath.Join(dir, fmt.Sprintf("%s-%d.json", ts, i))
		if i > 100 {
			return "", fmt.Errorf("too many same-second snapshot collisions")
		}
	}
	data, err := json.MarshalIndent(snap, "", "  ")
	if err != nil {
		return "", err
	}
	if err := os.WriteFile(path, data, 0644); err != nil {
		return "", err
	}
	if err := prune(dir, MaxSnapshots); err != nil {
		return path, err
	}
	return path, nil
}

// Load returns every snapshot under `<repoPath>/.repopulse/snapshots/`
// sorted ascending by GeneratedAt (then by filename as a stable tiebreak).
// Missing directory is not an error — returns an empty slice.
func Load(repoPath string) ([]compare.ReportSnapshot, error) {
	dir := filepath.Join(repoPath, Dir)
	entries, err := os.ReadDir(dir)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, err
	}
	var out []compare.ReportSnapshot
	for _, e := range entries {
		if e.IsDir() || !strings.HasSuffix(e.Name(), ".json") {
			continue
		}
		s, err := compare.LoadSnapshot(filepath.Join(dir, e.Name()))
		if err != nil {
			// Skip unreadable/corrupt files rather than failing the whole load.
			continue
		}
		out = append(out, *s)
	}
	sort.SliceStable(out, func(i, j int) bool {
		return out[i].GeneratedAt < out[j].GeneratedAt
	})
	return out, nil
}

func prune(dir string, keep int) error {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return err
	}
	var names []string
	for _, e := range entries {
		if e.IsDir() || !strings.HasSuffix(e.Name(), ".json") {
			continue
		}
		names = append(names, e.Name())
	}
	if len(names) <= keep {
		return nil
	}
	sort.Strings(names) // chronological since filenames are ISO timestamps
	excess := len(names) - keep
	for _, n := range names[:excess] {
		_ = os.Remove(filepath.Join(dir, n))
	}
	return nil
}

func ensureGitignore(moodDir string) error {
	path := filepath.Join(moodDir, ".gitignore")
	if _, err := os.Stat(path); err == nil {
		return nil
	}
	return os.WriteFile(path, []byte("*\n"), 0644)
}

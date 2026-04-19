package signals

import (
	"encoding/json"
	"math"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"mood-ring/internal/types"
)

// DetectCoverage tries to find + parse a coverage report. Returns nil if nothing found.
func DetectCoverage(repoPath string) *types.CoverageSignal {
	if p := filepath.Join(repoPath, "coverage", "coverage-summary.json"); fileExists(p) {
		if pct, ok := parseIstanbul(p); ok {
			return buildCoverage(pct, "istanbul")
		}
	}
	if p := filepath.Join(repoPath, "lcov.info"); fileExists(p) {
		if pct, ok := parseLcov(p); ok {
			return buildCoverage(pct, "lcov")
		}
	}
	if p := filepath.Join(repoPath, "coverage", "lcov.info"); fileExists(p) {
		if pct, ok := parseLcov(p); ok {
			return buildCoverage(pct, "lcov")
		}
	}
	return nil
}

func buildCoverage(pct float64, source string) *types.CoverageSignal {
	return &types.CoverageSignal{
		Type:       "coverage",
		Score:      scoreFromPercentage(pct),
		Percentage: math.Round(pct*100) / 100,
		Source:     source,
	}
}

func scoreFromPercentage(pct float64) int {
	if pct >= 80 {
		return 10
	}
	if pct >= 60 {
		return 35
	}
	if pct >= 40 {
		return 60
	}
	return 85
}

func parseIstanbul(path string) (float64, bool) {
	data, err := os.ReadFile(path)
	if err != nil {
		return 0, false
	}
	var parsed struct {
		Total struct {
			Lines struct {
				Pct float64 `json:"pct"`
			} `json:"lines"`
		} `json:"total"`
	}
	if err := json.Unmarshal(data, &parsed); err != nil {
		return 0, false
	}
	return parsed.Total.Lines.Pct, true
}

func parseLcov(path string) (float64, bool) {
	data, err := os.ReadFile(path)
	if err != nil {
		return 0, false
	}
	var lf, lh int
	for _, line := range strings.Split(string(data), "\n") {
		if strings.HasPrefix(line, "LF:") {
			n, _ := strconv.Atoi(strings.TrimSpace(line[3:]))
			lf += n
		}
		if strings.HasPrefix(line, "LH:") {
			n, _ := strconv.Atoi(strings.TrimSpace(line[3:]))
			lh += n
		}
	}
	if lf == 0 {
		return 0, false
	}
	return float64(lh) / float64(lf) * 100, true
}

func fileExists(p string) bool {
	_, err := os.Stat(p)
	return err == nil
}

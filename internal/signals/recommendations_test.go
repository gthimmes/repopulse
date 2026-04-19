package signals

import (
	"strings"
	"testing"
	"time"

	"mood-ring/internal/types"
)

func makeHS(overrides func(*types.HotspotEntry)) types.HotspotEntry {
	h := types.HotspotEntry{
		Path:         "src/x.ts",
		ChurnRank:    1,
		BugTouches:   0,
		ChaosTouches: 0,
		TotalCommits: 10,
		HotspotScore: 50,
		Authors:      3,
		LastTouched:  "2026-04-10",
		TopAuthorsOfFile: []types.HotspotFileAuthor{
			{Name: "Alice", Commits: 4},
			{Name: "Bob", Commits: 3},
			{Name: "Carol", Commits: 3},
		},
		RecentBugCommits: []types.HotspotCommit{},
		Owners:           []string{"@team/x"},
		Recommendations:  []types.HotspotRecommendation{},
	}
	if overrides != nil {
		overrides(&h)
	}
	return h
}

var windowEndApr15 = time.Date(2026, 4, 15, 0, 0, 0, 0, time.UTC)

func findRec(recs []types.HotspotRecommendation, kind string) *types.HotspotRecommendation {
	for i := range recs {
		if recs[i].Kind == kind {
			return &recs[i]
		}
	}
	return nil
}

func TestRec_BusFactor_Alert(t *testing.T) {
	h := makeHS(func(h *types.HotspotEntry) {
		h.TotalCommits = 10
		h.TopAuthorsOfFile = []types.HotspotFileAuthor{{Name: "Alice", Commits: 8}, {Name: "Bob", Commits: 2}}
	})
	recs := BuildRecommendations(h, nil, windowEndApr15, 3)
	r := findRec(recs, "bus-factor")
	if r == nil {
		t.Fatal("expected bus-factor rec")
	}
	if r.Severity != "alert" {
		t.Errorf("want alert, got %s", r.Severity)
	}
	if !strings.Contains(r.Text, "Alice") || !strings.Contains(r.Text, "80%") {
		t.Errorf("text: %s", r.Text)
	}
}

func TestRec_BusFactor_Warn(t *testing.T) {
	h := makeHS(func(h *types.HotspotEntry) {
		h.TotalCommits = 10
		h.TopAuthorsOfFile = []types.HotspotFileAuthor{{Name: "Alice", Commits: 7}, {Name: "Bob", Commits: 3}}
	})
	recs := BuildRecommendations(h, nil, windowEndApr15, 3)
	r := findRec(recs, "bus-factor")
	if r == nil || r.Severity != "warn" {
		t.Errorf("want warn, got %+v", r)
	}
}

func TestRec_BusFactor_BalancedOwnership(t *testing.T) {
	h := makeHS(nil) // Alice 4, Bob 3, Carol 3 of 10 — no 60%+ dominant
	recs := BuildRecommendations(h, nil, windowEndApr15, 3)
	if r := findRec(recs, "bus-factor"); r != nil {
		t.Errorf("did not expect bus-factor, got %+v", r)
	}
}

func TestRec_ChaosRepeat(t *testing.T) {
	h := makeHS(func(h *types.HotspotEntry) { h.ChaosTouches = 4 })
	recs := BuildRecommendations(h, nil, windowEndApr15, 3)
	if findRec(recs, "chaos-repeat") == nil {
		t.Error("expected chaos-repeat")
	}
}

func TestRec_ChaosRepeat_AlertAt5(t *testing.T) {
	h := makeHS(func(h *types.HotspotEntry) { h.ChaosTouches = 6 })
	recs := BuildRecommendations(h, nil, windowEndApr15, 3)
	r := findRec(recs, "chaos-repeat")
	if r == nil || r.Severity != "alert" {
		t.Errorf("want alert, got %+v", r)
	}
}

func TestRec_Rewritten(t *testing.T) {
	h := makeHS(nil)
	ce := &types.ChurnEntry{Path: "src/x.ts", Added: 100, Removed: 100, Ratio: 10, Rewritten: true}
	recs := BuildRecommendations(h, ce, windowEndApr15, 3)
	if findRec(recs, "rewritten") == nil {
		t.Error("expected rewritten")
	}
}

func TestRec_Unowned(t *testing.T) {
	h := makeHS(func(h *types.HotspotEntry) { h.Owners = []string{} })
	recs := BuildRecommendations(h, nil, windowEndApr15, 3)
	if findRec(recs, "unowned") == nil {
		t.Error("expected unowned")
	}
}

func TestRec_Unowned_WarnOnChaos(t *testing.T) {
	h := makeHS(func(h *types.HotspotEntry) {
		h.Owners = []string{}
		h.ChaosTouches = 1
	})
	recs := BuildRecommendations(h, nil, windowEndApr15, 3)
	r := findRec(recs, "unowned")
	if r == nil || r.Severity != "warn" {
		t.Errorf("want warn, got %+v", r)
	}
}

func TestRec_MultiOwner(t *testing.T) {
	h := makeHS(func(h *types.HotspotEntry) { h.Owners = []string{"@team/a", "@team/b"} })
	recs := BuildRecommendations(h, nil, windowEndApr15, 3)
	r := findRec(recs, "multi-owner")
	if r == nil {
		t.Fatal("expected multi-owner")
	}
	if !strings.Contains(r.Text, "@team/a") || !strings.Contains(r.Text, "@team/b") {
		t.Errorf("text: %s", r.Text)
	}
}

func TestRec_StaleBuggy(t *testing.T) {
	h := makeHS(func(h *types.HotspotEntry) {
		h.LastTouched = "2026-02-01"
		h.BugTouches = 3
	})
	recs := BuildRecommendations(h, nil, windowEndApr15, 3)
	if findRec(recs, "stale-buggy") == nil {
		t.Error("expected stale-buggy")
	}
}

func TestRec_NoStaleWithoutBugs(t *testing.T) {
	h := makeHS(func(h *types.HotspotEntry) {
		h.LastTouched = "2026-02-01"
		h.BugTouches = 0
	})
	recs := BuildRecommendations(h, nil, windowEndApr15, 3)
	if findRec(recs, "stale-buggy") != nil {
		t.Error("did not expect stale-buggy")
	}
}

func TestRec_BugHeavy(t *testing.T) {
	h := makeHS(func(h *types.HotspotEntry) {
		h.TotalCommits = 10
		h.BugTouches = 5
	})
	recs := BuildRecommendations(h, nil, windowEndApr15, 3)
	if findRec(recs, "bug-heavy") == nil {
		t.Error("expected bug-heavy")
	}
}

func TestRec_BugHeavy_WarnAt60(t *testing.T) {
	h := makeHS(func(h *types.HotspotEntry) {
		h.TotalCommits = 10
		h.BugTouches = 7
	})
	recs := BuildRecommendations(h, nil, windowEndApr15, 3)
	r := findRec(recs, "bug-heavy")
	if r == nil || r.Severity != "warn" {
		t.Errorf("want warn, got %+v", r)
	}
}

func TestRec_CapAndSort(t *testing.T) {
	h := makeHS(func(h *types.HotspotEntry) {
		h.TotalCommits = 10
		h.TopAuthorsOfFile = []types.HotspotFileAuthor{{Name: "Alice", Commits: 9}}
		h.ChaosTouches = 6
		h.BugTouches = 8
		h.Owners = []string{}
		h.LastTouched = "2026-01-01"
	})
	recs := BuildRecommendations(h, nil, windowEndApr15, 3)
	if len(recs) != 3 {
		t.Fatalf("want 3 recs, got %d", len(recs))
	}
	if recs[0].Severity != "alert" || recs[1].Severity != "alert" {
		t.Errorf("top 2 should be alerts, got %s/%s", recs[0].Severity, recs[1].Severity)
	}
}

func TestRec_CleanHotspot(t *testing.T) {
	h := makeHS(func(h *types.HotspotEntry) {
		h.TotalCommits = 5
		h.BugTouches = 1
		h.ChaosTouches = 0
		h.TopAuthorsOfFile = []types.HotspotFileAuthor{{Name: "A", Commits: 2}, {Name: "B", Commits: 2}, {Name: "C", Commits: 1}}
		h.Owners = []string{"@team/x"}
		h.LastTouched = "2026-04-14"
	})
	recs := BuildRecommendations(h, nil, windowEndApr15, 3)
	if len(recs) != 0 {
		t.Errorf("expected 0 recs, got %d: %+v", len(recs), recs)
	}
}

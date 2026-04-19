package signals

import (
	"fmt"
	"testing"
	"time"

	"repopulse/internal/types"
)

func mkc(hash string, daysAgo int, message string, files []types.FileChange, name string) types.CommitRecord {
	base := time.Date(2026, 4, 15, 12, 0, 0, 0, time.UTC)
	d := base.AddDate(0, 0, -daysAgo)
	if name == "" {
		name = "Alice Chen"
	}
	return types.CommitRecord{
		Hash:         hash,
		Date:         d,
		AuthorDate:   d,
		AuthorName:   name,
		AuthorEmail:  nameToEmail(name),
		Message:      message,
		FilesChanged: files,
		IsRevert:     revertMsgRE.MatchString(message),
	}
}

func nameToEmail(n string) string {
	return n + "@example.com"
}

var hsOpts = HotspotOptions{BugOptions: defaultBugOpts}

func TestHotspots_LastTouched(t *testing.T) {
	commits := []types.CommitRecord{
		mkc("h001", 30, "fix: bug A", []types.FileChange{{Path: "src/ledger.ts", Added: 10, Removed: 5}}, ""),
		mkc("h002", 10, "fix: bug B", []types.FileChange{{Path: "src/ledger.ts", Added: 8, Removed: 2}}, ""),
		mkc("h003", 20, "fix: bug C", []types.FileChange{{Path: "src/ledger.ts", Added: 5, Removed: 1}}, ""),
	}
	r := ComputeHotspots(commits, hsOpts)
	var ledger *types.HotspotEntry
	for i := range r.Hotspots {
		if r.Hotspots[i].Path == "src/ledger.ts" {
			ledger = &r.Hotspots[i]
		}
	}
	if ledger == nil {
		t.Fatal("ledger not found in hotspots")
	}
	// h002 at daysAgo=10 from 2026-04-15 → 2026-04-05
	if ledger.LastTouched != "2026-04-05" {
		t.Errorf("lastTouched: want 2026-04-05, got %s", ledger.LastTouched)
	}
}

func TestHotspots_Top3Authors(t *testing.T) {
	mkf := func(a, r int) []types.FileChange { return []types.FileChange{{Path: "src/pay.ts", Added: a, Removed: r}} }
	commits := []types.CommitRecord{
		mkc("h001", 30, "fix: a", mkf(10, 5), "Alice"),
		mkc("h002", 28, "fix: b", mkf(10, 5), "Alice"),
		mkc("h003", 26, "fix: c", mkf(10, 5), "Bob"),
		mkc("h004", 24, "fix: d", mkf(10, 5), "Bob"),
		mkc("h005", 22, "fix: e", mkf(10, 5), "Bob"),
		mkc("h006", 20, "fix: f", mkf(10, 5), "Carol"),
		mkc("h007", 18, "fix: g", mkf(10, 5), "Dave"),
	}
	r := ComputeHotspots(commits, hsOpts)
	pay := findHotspot(r.Hotspots, "src/pay.ts")
	if pay == nil {
		t.Fatal("pay not found")
	}
	if len(pay.TopAuthorsOfFile) != 3 {
		t.Fatalf("want 3 top authors, got %d", len(pay.TopAuthorsOfFile))
	}
	if pay.TopAuthorsOfFile[0].Name != "Bob" || pay.TopAuthorsOfFile[0].Commits != 3 {
		t.Errorf("top[0]: %+v", pay.TopAuthorsOfFile[0])
	}
	if pay.TopAuthorsOfFile[1].Name != "Alice" || pay.TopAuthorsOfFile[1].Commits != 2 {
		t.Errorf("top[1]: %+v", pay.TopAuthorsOfFile[1])
	}
	if pay.TopAuthorsOfFile[2].Commits > 1 {
		t.Errorf("top[2]: too many commits %d", pay.TopAuthorsOfFile[2].Commits)
	}
}

func TestHotspots_RecentBugCommits_Capped(t *testing.T) {
	commits := make([]types.CommitRecord, 15)
	for i := 0; i < 15; i++ {
		h := fmt.Sprintf("h%03d", i)
		commits[i] = mkc(h, i*2, "fix: issue "+h,
			[]types.FileChange{{Path: "src/pay.ts", Added: 10, Removed: 5}}, "")
	}
	opts := hsOpts
	opts.DrilldownCommitLimit = 10
	r := ComputeHotspots(commits, opts)
	pay := findHotspot(r.Hotspots, "src/pay.ts")
	if pay == nil {
		t.Fatal("pay not found")
	}
	if len(pay.RecentBugCommits) != 10 {
		t.Errorf("want 10 commits, got %d", len(pay.RecentBugCommits))
	}
	if pay.RecentBugCommits[0].Hash != "h000" {
		t.Errorf("first should be newest (h000), got %s", pay.RecentBugCommits[0].Hash)
	}
	for i := 1; i < len(pay.RecentBugCommits); i++ {
		if pay.RecentBugCommits[i-1].Date < pay.RecentBugCommits[i].Date {
			t.Errorf("not newest-first at %d", i)
		}
	}
}

func TestHotspots_TierClassification(t *testing.T) {
	mkf := []types.FileChange{{Path: "src/pay.ts", Added: 10, Removed: 5}}
	commits := []types.CommitRecord{
		mkc("h001", 1, "hotfix: critical chaos", mkf, ""),
		mkc("h002", 2, "fix: normal bug", mkf, ""),
		mkc("h003", 3, "fix: typo", mkf, ""),
		mkc("h004", 4, `Revert "broken thing"`, mkf, ""),
		mkc("h005", 5, "feat: add thing", mkf, ""),
	}
	r := ComputeHotspots(commits, hsOpts)
	pay := findHotspot(r.Hotspots, "src/pay.ts")
	if pay == nil {
		t.Fatal("pay not found")
	}
	tiers := make([]string, len(pay.RecentBugCommits))
	for i, c := range pay.RecentBugCommits {
		tiers[i] = c.Tier
	}
	// Newest first: hotfix(chaos), fix-normal(normal), typo(routine), Revert(chaos)
	expect := []string{"chaos", "normal", "routine", "chaos"}
	if fmt.Sprint(tiers) != fmt.Sprint(expect) {
		t.Errorf("tiers: want %v, got %v", expect, tiers)
	}
	for _, c := range pay.RecentBugCommits {
		if c.Message == "feat: add thing" {
			t.Error("feat commit should not appear")
		}
	}
}

func TestHotspots_MessageTruncation(t *testing.T) {
	commits := []types.CommitRecord{
		mkc("h001", 1, "fix: short first line\n\nlong body that should not appear",
			[]types.FileChange{{Path: "src/pay.ts", Added: 10, Removed: 5}}, ""),
	}
	r := ComputeHotspots(commits, hsOpts)
	pay := findHotspot(r.Hotspots, "src/pay.ts")
	if pay == nil {
		t.Fatal("pay not found")
	}
	if pay.RecentBugCommits[0].Message != "fix: short first line" {
		t.Errorf("message: got %q", pay.RecentBugCommits[0].Message)
	}
}

func TestHotspots_HashPrefix(t *testing.T) {
	commits := []types.CommitRecord{
		mkc("0123456789abcdef", 1, "fix: thing",
			[]types.FileChange{{Path: "src/pay.ts", Added: 10, Removed: 5}}, ""),
	}
	r := ComputeHotspots(commits, hsOpts)
	pay := findHotspot(r.Hotspots, "src/pay.ts")
	if pay == nil {
		t.Fatal("pay not found")
	}
	h := pay.RecentBugCommits[0].Hash
	if h != "0123456" || len(h) != 7 {
		t.Errorf("hash prefix: got %s (len %d)", h, len(h))
	}
}

func TestHotspots_ExcludesFilesWithNoBugInvolvement(t *testing.T) {
	commits := []types.CommitRecord{
		mkc("h001", 1, "feat: a", []types.FileChange{{Path: "src/clean.ts", Added: 10, Removed: 5}}, ""),
		mkc("h002", 2, "refactor: b", []types.FileChange{{Path: "src/clean.ts", Added: 10, Removed: 5}}, ""),
	}
	r := ComputeHotspots(commits, hsOpts)
	if findHotspot(r.Hotspots, "src/clean.ts") != nil {
		t.Error("src/clean.ts should not appear (no bug involvement)")
	}
}

func findHotspot(list []types.HotspotEntry, path string) *types.HotspotEntry {
	for i := range list {
		if list[i].Path == path {
			return &list[i]
		}
	}
	return nil
}

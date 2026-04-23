// repopulse — Go port of the TypeScript CLI. Produces a self-contained
// HTML dashboard showing the emotional state of a Git repository.
package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"math"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"time"

	"repopulse/internal/baseline"
	"repopulse/internal/codeowners"
	"repopulse/internal/compare"
	"repopulse/internal/config"
	"repopulse/internal/git"
	ghfetch "repopulse/internal/github"
	"repopulse/internal/narrative"
	"repopulse/internal/prmetrics"
	"repopulse/internal/render"
	"repopulse/internal/scorer"
	"repopulse/internal/signals"
	"repopulse/internal/snapshots"
	"repopulse/internal/standards"
	"repopulse/internal/types"
)

type stringsFlag []string

func (s *stringsFlag) String() string     { return strings.Join(*s, ",") }
func (s *stringsFlag) Set(v string) error { *s = append(*s, v); return nil }

func main() {
	fs := flag.NewFlagSet("repopulse", flag.ExitOnError)
	fs.Usage = func() {
		fmt.Fprintf(fs.Output(), "Usage: repopulse <repo-path> [options]\n\nOptions:\n")
		fs.PrintDefaults()
	}

	window := fs.Int("window", 90, "Analysis window in days")
	output := fs.String("output", "output/repopulse-report.html", "Output HTML file path")
	open := fs.Bool("open", false, "Open in browser after writing")
	since := fs.String("since", "", "Start from date (ISO or relative)")
	var ignorePats stringsFlag
	fs.Var(&ignorePats, "ignore", "Additional ignore glob pattern (repeatable)")
	jsonOut := fs.String("json", "", "Also write a JSON snapshot")
	cmpPath := fs.String("compare", "", "Previous JSON snapshot to diff against")
	markdownOut := fs.String("markdown", "", "Also write a markdown digest")
	noSnapshot := fs.Bool("no-snapshot", false, "Skip the automatic .repopulse/snapshots/ entry")
	ghToken := fs.String("github-token", "", "GitHub personal-access token for PR metrics (falls back to GITHUB_TOKEN env var)")
	ghRepo := fs.String("github-repo", "", "Override owner/name for PR metrics when origin URL is ambiguous")

	if len(os.Args) < 2 {
		fs.Usage()
		os.Exit(1)
	}
	if err := fs.Parse(os.Args[2:]); err != nil {
		os.Exit(1)
	}
	repoPathRaw := os.Args[1]
	opts := types.CliOptions{
		Window:     *window,
		Output:     *output,
		Open:       *open,
		Since:      *since,
		Ignore:     []string(ignorePats),
		JSON:       *jsonOut,
		Compare:    *cmpPath,
		Markdown:   *markdownOut,
		NoSnapshot:  *noSnapshot,
		GitHubToken: firstNonEmpty(*ghToken, os.Getenv("GITHUB_TOKEN")),
		GitHubRepo:  *ghRepo,
	}

	if err := run(repoPathRaw, opts); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
}

func run(repoPathRaw string, opts types.CliOptions) error {
	repoPath, err := filepath.Abs(repoPathRaw)
	if err != nil {
		return err
	}

	cfg := config.LoadConfig(repoPath)
	isExcluded := config.BuildIgnorePredicate(cfg, opts.Ignore)
	bugKW := config.ResolvedBugKeywords(cfg)
	bugOpts := signals.BugOptions{
		ChaosKeywords:   bugKW.Chaos,
		NormalKeywords:  bugKW.Normal,
		RoutineKeywords: bugKW.Routine,
	}

	commits, err := git.CollectCommits(git.CollectorOptions{
		RepoPath:   repoPath,
		WindowDays: opts.Window,
		Since:      opts.Since,
	})
	if err != nil {
		return err
	}
	if len(commits) == 0 {
		return fmt.Errorf("repository has no commits in the analysis window")
	}

	// Clip window to oldest commit date
	windowEnd := time.Now()
	oldest := commits[0].Date
	for _, c := range commits {
		if c.Date.Before(oldest) {
			oldest = c.Date
		}
	}
	requestedStart := time.Now().AddDate(0, 0, -opts.Window)
	windowStart := oldest
	if requestedStart.After(oldest) {
		windowStart = requestedStart
	}
	effectiveWindowDays := int(math.Max(1, math.Ceil(windowEnd.Sub(windowStart).Hours()/24)))

	// Signals
	freq := signals.ComputeFrequency(commits, effectiveWindowDays)
	churn := signals.ComputeChurn(commits, signals.ChurnOptions{
		IsExcluded:   isExcluded,
		WindowDays:   effectiveWindowDays,
		GetLineCount: func(p string) int { return git.GetFileLineCount(repoPath, p) },
		BugOptions:   bugOpts,
	})
	bug := signals.ComputeBugRatio(commits, bugOpts)
	cov := signals.DetectCoverage(repoPath)
	co := codeowners.Load(repoPath)
	modules := signals.ComputeModules(commits, signals.ModuleOptions{
		IsExcluded: isExcluded,
		BugOptions: bugOpts,
		Codeowners: co,
	})
	churnLookup := map[string]types.ChurnEntry{}
	for _, c := range churn.TopChurners {
		churnLookup[c.Path] = c
	}
	hotspots := signals.ComputeHotspots(commits, signals.HotspotOptions{
		IsExcluded:  isExcluded,
		BugOptions:  bugOpts,
		Codeowners:  co,
		ChurnLookup: churnLookup,
		WindowEnd:   windowEnd,
	})
	preEmails, _ := git.GetPreWindowAuthorEmails(repoPath, windowStart)
	authors := signals.ComputeAuthors(commits, signals.AuthorOptions{
		IsExcluded:            isExcluded,
		WindowStart:           windowStart,
		PreWindowAuthorEmails: preEmails,
	})
	rolling := narrative.ComputeRollingTimeline(commits, windowStart, windowEnd, bugOpts)

	// Plank 1 — per-author baseline drift. Pull a baseline window
	// 6× the current window, ending where the current window starts,
	// so each contributor is compared against their own prior pattern.
	baselineDays := effectiveWindowDays * 6
	baselineSince := windowStart.AddDate(0, 0, -baselineDays).UTC().Format(time.RFC3339)
	baselineUntil := windowStart.UTC().Format(time.RFC3339)
	baselineCommits, err := git.CollectCommits(git.CollectorOptions{
		RepoPath: repoPath,
		Since:    baselineSince,
		Until:    baselineUntil,
	})
	if err != nil {
		fmt.Fprintf(os.Stderr, "Warning: baseline window fetch failed: %v\n", err)
	}
	authorDrift := baseline.ComputeAuthorDrift(commits, baselineCommits, baseline.Options{
		CurrentDays:  effectiveWindowDays,
		BaselineDays: baselineDays,
		BugOptions:   bugOpts,
	})

	// Plank 2 — deterministic standards. Conventional-commit compliance
	// is computed over the current window's commits; test density over
	// the full HEAD file set so the result reflects the whole repo,
	// not just files changed in the window.
	allFiles, err := git.ListFiles(repoPath)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Warning: could not list files for test-density analysis: %v\n", err)
	}
	commitPattern, commitPatternSource, commitPatternCustom := config.ResolvedCommitPattern(cfg)
	// Only pass the pattern string to the renderer when it's custom;
	// the default-case UI reads "Conventional Commits" rather than
	// showing the regex itself.
	sourceForUI := ""
	if commitPatternCustom {
		sourceForUI = commitPatternSource
	}
	standardsSig := standards.Compute(commits, allFiles, standards.Options{
		CommitPattern:       commitPattern,
		CommitPatternSource: sourceForUI,
	})

	// Phase 3.1 — PR flow. Gated on a GitHub token being available.
	// No token → no PR section (the report still works for offline
	// analysis). Rate-limit hits fall back to cached PRs + banner.
	prFlow := computePRFlow(repoPath, opts, effectiveWindowDays, windowStart)

	base := scorer.ComputeMood(scorer.Input{
		CommitFrequency: freq,
		FileChurn:       churn,
		BugRatio:        bug,
		Coverage:        cov,
		Modules:         modules,
		Hotspots:        hotspots,
		Authors:         authors,
		RollingTimeline: rolling,
	})
	base.Signals.AuthorDrift = authorDrift
	base.Signals.Standards = standardsSig
	if prFlow != nil {
		base.Signals.PRFlow = prFlow
	}
	meta := types.RepoMeta{
		RepoName:          filepath.Base(repoPath),
		RepoPath:          repoPath,
		AnalyzedCommits:   len(commits),
		WindowDays:        effectiveWindowDays,
		WindowStart:       windowStart,
		WindowEnd:         windowEnd,
		GeneratedAt:       time.Now(),
		HasLimitedHistory: len(commits) < 10,
	}
	narr := narrative.Generate(base, meta)
	mood := base
	mood.Narrative = narr

	// Delta
	var delta *types.MoodDelta
	if opts.Compare != "" {
		prev, err := compare.LoadSnapshot(opts.Compare)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Warning: could not load --compare file: %v\n", err)
		} else {
			d := compare.BuildDelta(mood, prev.MoodResult, prev.GeneratedAt)
			delta = &d
		}
	}

	// Persistent snapshot store: write current run, then load the full
	// history so the trend chart includes every prior point.
	currentSnap := compare.ReportSnapshot{
		GeneratedAt:     meta.GeneratedAt.UTC().Format(time.RFC3339Nano),
		RepoName:        meta.RepoName,
		WindowDays:      meta.WindowDays,
		AnalyzedCommits: meta.AnalyzedCommits,
		MoodResult:      mood,
	}
	var trendSnaps []compare.ReportSnapshot
	var snapPath string
	if !opts.NoSnapshot {
		p, err := snapshots.Save(repoPath, currentSnap)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Warning: could not write snapshot: %v\n", err)
		} else {
			snapPath = p
		}
		loaded, err := snapshots.Load(repoPath)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Warning: could not load snapshot history: %v\n", err)
		} else {
			trendSnaps = loaded
		}
	}

	// HTML
	html := render.RenderHTML(mood, meta, delta, trendSnaps)
	outputPath, err := filepath.Abs(opts.Output)
	if err != nil {
		return err
	}
	if err := writeFileMkdir(outputPath, []byte(html)); err != nil {
		return err
	}

	// Optional JSON (explicit path; independent of the persistent store)
	if opts.JSON != "" {
		data, _ := json.MarshalIndent(currentSnap, "", "  ")
		jsonPath, _ := filepath.Abs(opts.JSON)
		if err := writeFileMkdir(jsonPath, data); err != nil {
			return err
		}
		fmt.Printf("  JSON snapshot:     %s\n", jsonPath)
	}

	// Optional markdown
	if opts.Markdown != "" {
		md := render.RenderMarkdown(mood, meta, delta, render.MarkdownOptions{
			HTMLReportPath: outputPath,
		})
		mdPath, _ := filepath.Abs(opts.Markdown)
		if err := writeFileMkdir(mdPath, []byte(md)); err != nil {
			return err
		}
		fmt.Printf("  Markdown digest:   %s\n", mdPath)
	}

	// Summary
	moodLabel := strings.ToUpper(string(mood.Mood)[:1]) + string(mood.Mood)[1:]
	emoji := render.MoodEmoji(string(mood.Mood))
	fmt.Printf("\n\u2713 Analyzed %d commits over %d days\n", len(commits), effectiveWindowDays)
	fmt.Printf("  Mood: %s %s (score: %d/100)\n", moodLabel, emoji, mood.CompositeScore)
	if delta != nil {
		sign := ""
		if delta.Composite >= 0 {
			sign = "+"
		}
		fmt.Printf("  \u0394 vs previous:   %s%d\n", sign, delta.Composite)
	}
	fmt.Printf("  HTML report:       %s\n", outputPath)
	if snapPath != "" {
		fmt.Printf("  Snapshot stored:   %s (%d total)\n", snapPath, len(trendSnaps))
	}
	if len(mood.Signals.Modules.Modules) > 0 {
		top := mood.Signals.Modules.Modules
		if len(top) > 3 {
			top = top[:3]
		}
		parts := make([]string, len(top))
		for i, m := range top {
			parts[i] = fmt.Sprintf("%s:%d", m.Name, m.Score)
		}
		fmt.Printf("  Hot modules:       %s\n", strings.Join(parts, ", "))
	}
	fmt.Println()

	if opts.Open {
		openInBrowser(outputPath)
	}
	return nil
}

func openInBrowser(path string) {
	var cmd *exec.Cmd
	switch runtime.GOOS {
	case "windows":
		cmd = exec.Command("cmd", "/c", "start", "", path)
	case "darwin":
		cmd = exec.Command("open", path)
	default:
		cmd = exec.Command("xdg-open", path)
	}
	_ = cmd.Start()
}

// writeFileMkdir writes data to path, creating parent directories as needed.
func writeFileMkdir(path string, data []byte) error {
	if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
		return err
	}
	return os.WriteFile(path, data, 0644)
}

func firstNonEmpty(a, b string) string {
	if strings.TrimSpace(a) != "" {
		return a
	}
	return b
}

// computePRFlow does the full PR flow pass: resolve owner/repo, fetch
// merged PRs in the window, compute metrics. Returns nil (and logs a
// warning) when the token is absent, the remote isn't GitHub, or the
// fetch fails irrecoverably.
func computePRFlow(repoPath string, opts types.CliOptions, windowDays int, windowStart time.Time) *types.PRFlowSignal {
	if opts.GitHubToken == "" && opts.GitHubRepo == "" {
		return nil // offline / no-token mode; silently skip
	}

	var ownerRepo ghfetch.OwnerRepo
	if opts.GitHubRepo != "" {
		parts := strings.SplitN(opts.GitHubRepo, "/", 2)
		if len(parts) != 2 {
			fmt.Fprintf(os.Stderr, "Warning: --github-repo must be owner/name, got %q\n", opts.GitHubRepo)
			return nil
		}
		ownerRepo = ghfetch.OwnerRepo{Owner: parts[0], Name: parts[1]}
	} else {
		detected, err := ghfetch.DetectOwnerRepo(repoPath)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Warning: could not detect GitHub owner/repo: %v\n", err)
			return nil
		}
		ownerRepo = detected
	}

	cacheDir := filepath.Join(repoPath, ".repopulse", "pr-cache")
	client := ghfetch.NewClient(opts.GitHubToken)
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
	defer cancel()

	res, err := ghfetch.FetchMergedPRs(ctx, client, ghfetch.FetchOptions{
		Owner:    ownerRepo.Owner,
		Repo:     ownerRepo.Name,
		Since:    windowStart,
		CacheDir: cacheDir,
		MaxPRs:   1000,
	})
	if err != nil {
		fmt.Fprintf(os.Stderr, "Warning: PR fetch failed: %v\n", err)
		return nil
	}

	sig := prmetrics.Compute(res.PRs, ownerRepo.String(), windowDays, prmetrics.Options{})
	sig.CacheBanner = res.CacheBanner
	return &sig
}

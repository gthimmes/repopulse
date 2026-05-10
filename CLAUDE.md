# CLAUDE.md — Codebase Repopulse

## What this project is

A Go CLI tool (`repopulse`) that analyzes a local Git repository and generates a **self-contained HTML file** + optional Markdown digest visualizing the "emotional state" of the codebase over time.

No AI. No external APIs. Pure git data + math + Chart.js output.

---

## Current state (read this first when resuming)

**Phase 1 complete. Direction pivoted to the "3-plank lens" model (see ROADMAP.md). Plank 1 + Plank 2 deterministic layer shipped. Codebase is 100% Go.**

- Application code + all signal math: `cmd/repopulse/main.go` + `internal/*`
- Persistent snapshot store: `internal/snapshots/` writes `<repo>/.repopulse/snapshots/<ts>.json` every run, capped at 365, gitignore auto-laid. Opt out with `-no-snapshot`.
- Trend chart in HTML report: `internal/render/trends.go` reads the store and renders a multi-series Chart.js line (composite shown by default, 5 per-signal series legend-toggleable).
- **Plank 1 — baseline drift:** `internal/baseline/` compares each contributor against their own 6×-window historical baseline on commit cadence, weekend/night %, and fix-vs-feature mix. Flagged deltas render in a "Worth a 1:1" card. No cross-author ranking — ever.
- **Plank 2 — standards detection (deterministic layer):** `internal/standards/` computes commit-pattern compliance (Conventional Commits by default, configurable via `.repopulserc`'s `commitPattern` regex) + test density (test-files-per-source ratio per module). Rendered in a "Standards" card.
- **Plank 2 Layer B — AI enrichment:** `internal/enrich/` produces an optional `EnrichmentResult` (narrative bullets, standards verdict, per-author drift readings) on top of the deterministic snapshot. Three modes: **A** (no AI, default — exactly the same report as before), **B** (`--enrich` flag, gated on `ANTHROPIC_API_KEY` env var; calls `api.anthropic.com/v1/messages` directly with stdlib HTTP, caches at `<repo>/.repopulse/enrichment-cache/` keyed by snapshot+model hash so identical inputs don't get re-billed), **C** (a Claude Code skill at `skills/repopulse-enrich.md` produces `enriched.json` in-conversation, fed back via `--enriched`; no API key required). Pipeline split also lives here: `--emit-json` writes the deterministic snapshot, `--from-json` renders from one without re-collecting. The renderer's `renderEnrichment()` paints a purple-bordered "AI read" card with an `AI-GENERATED` tag — visually distinct so a reader can always tell interpretation from measurement. Network/auth/parse failures only warn; the deterministic report still renders.
- **Phase 3.1 — PR Flow:** `internal/github/` (REST client + cache) + `internal/prmetrics/` (signal math). Gated on a GitHub token (`--github-token` flag or `GITHUB_TOKEN` env var). Produces cycle-time percentiles, time-to-first-review, reviewer concentration, rubber-stamp rate, self-merge rate. Cache lives at `<repo>/.repopulse/pr-cache/` and is incrementally refreshed (only refetch PRs whose `updated_at` changed). Rate-limit hits fall back to cached data with a banner.
- **Plank 3 — exploration:** Top Churned Files is a drillable `<details>` list (per-file authors + recent bug-tagged commits). **Contributors explorer** at the bottom of the report — full unbounded list of every contributor in the window, sorted by LOC desc, scrollable, drillable. Each row's expanded panel shows stats, baseline-drift detail (or "no flags this window"), conventional-commit compliance bar, and top-touched files. Drift-flagged contributors get an alert/watch pill in the Watch? column. Folds in what used to be the standalone Worth-a-1:1 card. See `renderContributorsSection` and `renderChurnDetail` in `internal/render/template.go`.
- **Badge redesign:** "REPO PRESSURE" headline + numeric score + band pill (Steady/Active/Volatile) + horizontal gradient bar with marker. The emoji + mood label are gone from the report; the same emoji is still emitted in the CLI summary line and markdown digest as a quick visual cue.
- Fixture generator for the Playwright e2e tests: `cmd/fixture-gen/main.go`
- Ships as a **4 MB static binary** (`repopulse.exe`) with a single runtime dependency (the `git` command on PATH)

**TypeScript is gone.** The only remaining Node footprint is the Playwright test harness — `package.json` declares `@playwright/test` as the sole devDependency, and `tests/e2e/*.spec.ts` are Playwright specs (Playwright has its own built-in TS handling, no `tsconfig.json` needed). This is test infrastructure, not product code, and it stays because Playwright is the industry-standard tool for browser automation and there's no Go-native equivalent.

See `ROADMAP.md` for the full picture of where we've been and where we're going. The current-state section at the top of that file is the fastest way to orient.

---

## Output target (most important constraint)

The final deliverable is always **one self-contained HTML file**:
- All CSS inlined
- Chart.js loaded from CDN (`https://cdn.jsdelivr.net/npm/chart.js`)
- All repo data embedded in the HTML — no runtime fetches
- No build step required to open it — just `open report.html`

Do NOT build a dev server, web app, or anything requiring a running process to view the report.

---

## Project structure

```
repopulse/
├── cmd/
│   ├── repopulse/main.go          # Primary CLI entry point
│   └── fixture-gen/main.go        # Test-only: writes the Playwright fixture HTML (--enriched optional)
├── skills/
│   └── repopulse-enrich.md        # Mode C: Claude Code skill producing enriched.json in-conversation
├── internal/
│   ├── types/          # Shared structs (incl. EnrichmentResult)
│   ├── git/            # git log invocation + parser
│   ├── config/         # ignore patterns + bug-keyword tiers
│   ├── codeowners/     # CODEOWNERS parser, path matcher
│   ├── snapshots/      # Phase 2.1 persistent store: save/load/prune `.repopulse/snapshots/`
│   ├── baseline/       # Plank 1: per-author drift vs their own historical baseline
│   ├── standards/      # Plank 2 deterministic: conventional-commit compliance + test density
│   ├── enrich/         # Plank 2 Layer B: Anthropic Messages client + prompt builder + cache
│   ├── github/         # Phase 3.1: REST client + PR fetcher + on-disk cache
│   ├── prmetrics/      # Phase 3.1: cycle-time / reviewer / rubber-stamp signal math
│   ├── signals/        # per-signal computations
│   │   ├── frequency.go    # Commit cadence
│   │   ├── churn.go        # Churn density + throughput
│   │   ├── bugratio.go     # Classified bug commits (chaos/normal/routine)
│   │   ├── modules.go      # Per-module mood
│   │   ├── hotspots.go     # Feathers-style hotspot detection
│   │   ├── authors.go      # Weekend/night %, bus factor, new contributors
│   │   ├── recommendations.go  # 7 heuristic rules per hotspot
│   │   └── coverage.go     # Istanbul / lcov parsing
│   ├── scorer/         # Weighted composite → mood label
│   ├── narrative/      # Finding bullets + rolling 7-day timeline
│   ├── render/         # HTML (template+charts+css) + Markdown
│   ├── compare/        # Snapshot loading + delta
│   └── fixtures/       # Test fixtures (commit sets + UI fixture MoodResult)
├── tests/e2e/                      # Playwright specs (test harness, language-agnostic)
├── output/                         # Generated reports land here (gitignored)
├── tools/go/                       # Portable Go toolchain (gitignored — install locally)
├── go.mod / go.sum                 # Go module definition
├── package.json                    # Playwright harness only — NOT application code
├── playwright.config.ts
├── ROADMAP.md                      # Where we're going
├── SPEC.md                         # Signal math + scoring spec
└── README.md                       # User-facing quick start
```

---

## Resuming on a new device

When the repo is cloned fresh:

1. **Install Go (1.23+)**. Two options:
   - System install: `winget install GoLang.Go` (Windows), `brew install go` (macOS), etc.
   - Portable: download from https://go.dev/dl/ and unzip to `./tools/go/`. Shell out via `./tools/go/bin/go` — nothing touches system PATH.

2. **Install Node (18+) + npm** for Playwright only.

3. **First-time dep sync**:
   ```bash
   go get github.com/bmatcuk/doublestar/v4@v4.7.1    # Go's only dep
   npm install                                         # Playwright
   npx playwright install chromium                     # Browser for e2e
   ```
   Note: `go mod tidy` fails if `tools/go/` is present — it scans the toolchain's `test/` directory. Use `go get <pkg>` directly instead.

4. **Build**:
   ```bash
   go build -o repopulse.exe ./cmd/repopulse       # Windows (or `repopulse` on unix)
   go build -o fixture-gen.exe ./cmd/fixture-gen   # Needed before running Playwright
   ```

5. **Verify everything works**:
   ```bash
   go test ./internal/...       # Go unit tests
   npx playwright test          # Playwright e2e tests
   ```

Both suites should be green before making changes.

---

## Running the tool

```bash
./repopulse.exe /path/to/repo                                    # writes output/repopulse-report.html
./repopulse.exe /path/to/repo -markdown output/digest.md         # HTML + markdown digest
./repopulse.exe /path/to/repo -json output/snap.json -open       # HTML + JSON snapshot + auto-open

# Plank 2 Layer B — AI enrichment:
./repopulse.exe /path/to/repo -enrich                            # Mode B: API key in $ANTHROPIC_API_KEY
./repopulse.exe /path/to/repo -emit-json output/det.json         # collect → write deterministic snapshot
./repopulse.exe -from-json output/det.json -enriched out/ai.json # render from deterministic + enriched
```

Full flag list: `./repopulse.exe --help`. Notable flags for the pipeline split:

- `-emit-json <path>` — write the deterministic snapshot post-collect, pre-render. Inputs to a Mode-C skill.
- `-from-json <path>` — skip collect entirely; render from a previously-emitted snapshot.
- `-enriched <path>` — layer an `enriched.json` (Plank-2 Layer-B output) on top of either path.
- `-enrich` — call the Anthropic API after collect (requires `ANTHROPIC_API_KEY` or `--anthropic-api-key`).
- `-anthropic-api-key <key>` — explicit override; falls back to `ANTHROPIC_API_KEY` env var.
- `-enrich-model <id>` — override the Claude model id (default: `claude-sonnet-4-6`).

---

## Signal weights (in `internal/scorer/scorer.go`)

```
commitFrequency: 15%
fileChurn:       25%
bugRatio:        30%
coverage:        10%  (redistributes to bugRatio if no coverage found)
authors:         20%
```

**Mood thresholds:**
```
0–40   → calm      😌
41–70  → anxious   😬
71–100 → chaotic   🔥
```

See `SPEC.md` for full signal math.

---

## Testing

Two suites, both green:

| Suite | Count | Command | Notes |
|-------|-------|---------|-------|
| Go unit | ~150 | `go test ./internal/...` | Pure math verification over deterministic fixtures (incl. enrich package) |
| Playwright e2e | 36 | `npx playwright test` | Drives a real browser against fixture + real-data reports (incl. enrichment AI-card spec) |

Playwright requires `fixture-gen.exe` to be built first — `tests/e2e/fixtures.ts` execs it to produce the fixture HTML. `playwright.config.ts` does not currently auto-build; add a `globalSetup` hook when this becomes friction.

Commit/signal fixtures: `internal/fixtures/commits.go` (calm + chaotic sets used by unit tests) and `internal/fixtures/ui.go` (deterministic MoodResult for the Playwright UI tests).

---

## Key implementation notes

### Git calls all go through `internal/git/`
Never shell out to `git` directly from signal or render packages. Isolates I/O so signals stay pure-function.

### CODEOWNERS semantics
GitHub-compatible subset: comments, blank lines, multi-owner lines, anchored (`/src/`) vs unanchored patterns, directory patterns (`docs/`), last-match-wins. See `internal/codeowners/codeowners.go` for supported/unsupported cases.

### Bug signal is tiered
Not a single regex. Chaos (`revert`, `hotfix`, `broken`, `p0`, …) weighted 1.0; normal (`fix`, `bug`, `patch`) weighted 0.4; routine (`typo`, `lint`, `format`) weighted 0.1. Revert commits always chaos. Routine wins over normal when both match (`fix: typo` → routine).

### Coverage detection waterfall
1. `coverage/coverage-summary.json` (Istanbul JSON → `total.lines.pct`)
2. `lcov.info` at root (sum `LF:` / `LH:` lines)
3. `coverage/lcov.info`

If none found, skip signal and redistribute its weight to bugRatio.

### Repos with sparse history
Fewer than 10 commits in window → still generate report with a warning banner; do not error out.

### Hotspot detail panels
Use `<details>`/`<summary>`. Zero-JS toggle, survives print-to-HTML and save-as.

### Chart.js function callbacks
Chart.js config objects are emitted as JS object-literal strings (not JSON) so callback functions like tooltip formatters are real functions, not strings. See `internal/render/charts.go`.

### Output paths
The CLI defaults all outputs under `output/` and auto-creates the parent directory via `writeFileMkdir`. Users can override with explicit `-output`, `-json`, `-markdown` paths pointing anywhere.

### Snapshot store (Phase 2.1)
Lives **inside the analyzed repo** at `<repoPath>/.repopulse/snapshots/`, not the cwd — keeps history portable with the repo so a teammate's report includes the same trend. First write lays down `.repopulse/.gitignore` with `*` so the store never accidentally gets committed. Filenames are UTC ISO timestamps (`2026-04-19T020807Z.json`) so lexical sort = chronological sort. Pruning is count-based (default 365 newest); replace with retention-policy logic if/when needed.

### Plank 1 baseline drift (internal/baseline)
Fetches a baseline window 6× the current window (non-overlapping, immediately prior) via a second `git.CollectCommits` call with `Until` set to `windowStart`. Each author's current stats (cadence, weekend-night %, fix-ratio) are compared to their own baseline. Flagging requires BOTH a relative-magnitude threshold AND an absolute-volume floor — prevents noise at small commit counts (e.g. 100% cadence increase on 1 commit/week doesn't fire). Only authors with ≥3 current + ≥5 baseline commits are evaluated; authors with no flags are silently dropped so the "Worth a 1:1" card only ever shows signal.

### Plank 2 standards detection (internal/standards)
Two deterministic signals today: conventional-commit compliance (regex over subject lines, `type(scope?)!?: subject`) and test density (in `colocation.go` — filename kept for legacy). Density classifies every tracked file under a known language as EITHER a test (by filename suffix like `*Test.kt`, `*_test.go`, `*.test.ts` — OR by presence under a conventional test-source path like `/src/test/`, `/__tests__/`) or a source, then emits `test_count / source_count * 100` per module. Intentionally ratio-based rather than filename-matching: catches "does this module have tests at all" without false negatives when a team splits tests by action (e.g. `FooServiceCreateTest.kt` for source `FooService.kt`) or uses integration-test suites. Density can exceed 100% for healthy test-heavy codebases. Walks `git ls-files` at HEAD so the result reflects the whole repo, not just files touched in the window. Module breakdown uses the same top-level-dir convention as the existing modules signal.

### Plank 2 Layer B AI enrichment (internal/enrich)
Adds optional AI-authored interpretation on top of the deterministic snapshot. The `EnrichmentResult` shape (in `internal/types`) carries up to four fields: narrative bullets, a standards verdict, per-author drift readings, and free-form notes — all optional, all rendered only when present. `enrich.Run()` is the API entry: it builds a compact text digest of aggregate signals (NEVER raw code or full commit lists), POSTs to `api.anthropic.com/v1/messages` via stdlib HTTP (no SDK dependency), parses the response (tolerates `\`\`\`json`-fenced output), and writes a cached copy keyed by `sha256(digest + model + schemaVersion)`. Cache hits skip the network call entirely so re-runs of an unchanged snapshot don't get re-billed. `enrich.BuildPrompt()` is exported so the Mode-C skill can use the same prompt body without going through the API. Failure modes (no key, network down, non-2xx, malformed JSON) all return errors that the caller logs as warnings and proceeds without enrichment — the deterministic report is the source of truth and must always render.

The renderer (`renderEnrichment` in `internal/render/template.go`) lays down a single dedicated card between Findings and the rest of the report. Card is purple-tinted with an `AI-GENERATED` tag and a one-line "interpretation only, deterministic numbers above are the source of truth" subtitle so a reader can never confuse this with measurement. Drift entries match back to contributors by case-insensitive email so the displayed name comes from the deterministic Authors signal, not from whatever the model invented.

Three modes are documented in `skills/repopulse-enrich.md` (Mode C — the Claude Code skill). The skill explains how to read the deterministic snapshot, produce JSON matching the `EnrichmentResult` shape, and feed it back via `--from-json` + `--enriched`. Mode B (`--enrich` with `ANTHROPIC_API_KEY`) is end-to-end automated; Mode A is just running the binary normally — the deterministic report renders unchanged.

---

## Dependencies

**Go runtime**: `github.com/bmatcuk/doublestar/v4` (glob matching). That's it.

**Node dev-only (Playwright)**: `@playwright/test`. That's it.

**Both**: `git` on PATH.

**In generated HTML only**: Chart.js loaded from `https://cdn.jsdelivr.net/npm/chart.js` — not a compile-time dependency.

---

## Do not

- Add runtime dependencies to Go without checking if stdlib can do it.
- Introduce a JavaScript/TypeScript build step for application code — the product is Go-only. Playwright specs stay in `.ts` because that's what Playwright uses internally, but anything under `cmd/` or `internal/` is Go.
- Create dev servers, Express/fiber apps, or anything requiring a running process to serve the report.
- Add new signals without updating `SPEC.md` and the scorer weight table.
- Commit generated files. Anything under `output/` is gitignored. Generated binaries (`repopulse.exe`, `fixture-gen.exe`) are too.

---

## Collaboration notes

The user prefers:
- Short, direct responses. No marketing copy.
- PM/QA/dev hat changes are explicit — when acting as PM reviewing my own work, I say so.
- Tests verify, screenshots confirm. I have Playwright + Chromium set up and should drive the browser to verify UI changes rather than just claiming they work.
- The user will commit; I should make sure docs are always current and nothing that shouldn't be committed is tracked.
- Honest status reports — if something's half-done, say so with specifics (file paths, function names, remaining steps).

# CLAUDE.md — Codebase Mood Ring

## What this project is

A Go CLI tool (`mood-ring`) that analyzes a local Git repository and generates a **self-contained HTML file** + optional Markdown digest visualizing the "emotional state" of the codebase over time.

No AI. No external APIs. Pure git data + math + Chart.js output.

---

## Current state (read this first when resuming)

**Phase 1 is complete. The codebase is 100% Go.**

- Application code + all signal math: `cmd/mood-ring/main.go` + `internal/*`
- Fixture generator for the Playwright e2e tests: `cmd/fixture-gen/main.go`
- Ships as a **4 MB static binary** (`mood-ring.exe`) with a single runtime dependency (the `git` command on PATH)

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
mood-ring/
├── cmd/
│   ├── mood-ring/main.go          # Primary CLI entry point
│   └── fixture-gen/main.go        # Test-only: writes the Playwright fixture HTML
├── internal/
│   ├── types/          # Shared structs
│   ├── git/            # git log invocation + parser
│   ├── config/         # ignore patterns + bug-keyword tiers
│   ├── codeowners/     # CODEOWNERS parser, path matcher
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
   go build -o mood-ring.exe ./cmd/mood-ring       # Windows (or `mood-ring` on unix)
   go build -o fixture-gen.exe ./cmd/fixture-gen   # Needed before running Playwright
   ```

5. **Verify everything works**:
   ```bash
   go test ./internal/...       # 68 tests
   npx playwright test          # 26 tests
   ```

Both suites should be green before making changes.

---

## Running the tool

```bash
./mood-ring.exe /path/to/repo                                    # writes output/mood-report.html
./mood-ring.exe /path/to/repo -markdown output/digest.md         # HTML + markdown digest
./mood-ring.exe /path/to/repo -json output/snap.json -open       # HTML + JSON snapshot + auto-open
```

Full flag list: `./mood-ring.exe --help`.

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
| Go unit | 68 | `go test ./internal/...` | Pure math verification over deterministic fixtures |
| Playwright e2e | 26 | `npx playwright test` | Drives a real browser against fixture + real-data reports |

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
- Commit generated files. Anything under `output/` is gitignored. Generated binaries (`mood-ring.exe`, `fixture-gen.exe`) are too.

---

## Collaboration notes

The user prefers:
- Short, direct responses. No marketing copy.
- PM/QA/dev hat changes are explicit — when acting as PM reviewing my own work, I say so.
- Tests verify, screenshots confirm. I have Playwright + Chromium set up and should drive the browser to verify UI changes rather than just claiming they work.
- The user will commit; I should make sure docs are always current and nothing that shouldn't be committed is tracked.
- Honest status reports — if something's half-done, say so with specifics (file paths, function names, remaining steps).

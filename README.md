# repopulse 💍

Visualize the "emotional state" of a Git repository as a self-contained HTML dashboard + shareable Markdown digest.

No AI. No external APIs. Just git history + math. Single 4 MB binary.

```bash
./repopulse.exe /path/to/repo -open
```

## What it produces

Two artifacts, written to `output/` by default:

- **HTML dashboard** — a single self-contained file you can open in any browser, email to a teammate, or drop in a wiki. Works offline.
- **Markdown digest** (optional via `-markdown <file>`) — a shareable summary suitable for Slack, PR descriptions, or standup docs.

## Mood states

- 😌 **Calm** — stable cadence, low churn, few fix commits
- 😬 **Anxious** — bursty activity, rising churn, some fire-fighting
- 🔥 **Chaotic** — lots of hotfixes/reverts, high churn density, irregular bursts

## What's in the report

- **Mood badge** — composite score 0–100 with a `0 = calm / 100 = chaotic / lower is better` scale hint
- **Findings** — 3–8 narrative bullets sorted by severity (alert/warn/info/good)
- **Score trend across snapshots** — composite + per-signal lines across every past run (snapshots auto-stored in `.repopulse/snapshots/`)
- **Standards** — deterministic compliance card: conventional-commit format % + test-file colocation % with per-author and per-module breakdowns
- **Stats row** — commits, files touched, bug %, commits/day
- **Score breakdown** — per-signal horizontal bars with bands
- **Module mood grid** — per top-level directory with team ownership from CODEOWNERS
- **Hotspots** — expandable rows showing recommendations, top authors, and recent bug-tier commits
- **Commit frequency** / **Mood timeline** / **Bug signal timeline**
- **Bug explainability** — collapsible panel with tier legend + classified sample commits per tier
- **Authors** — weekend/night %, bus factor, top-10 table
- **Worth a 1:1** — per-author baseline drift: cadence, off-hours, fix-vs-feature shifts vs each contributor's own history (not a peer ranking)
- **Top churned files** — sortable
- **Coverage** — if your repo generates an Istanbul or lcov report

## Usage

```
repopulse <repo-path> [options]

Options:
  -window <days>      Analysis window in days (default: 90)
  -output <file>      Output HTML file (default: output/repopulse-report.html)
  -open               Auto-open in browser after writing
  -since <date>       Start from date, e.g. "2024-01-01" or "3 months ago"
  -ignore <pattern>   Additional glob ignore pattern (repeatable)
  -json <file>        Also write a JSON snapshot (for later -compare)
  -compare <file>     Previous JSON snapshot to diff against
  -markdown <file>    Also write a Markdown digest
  -no-snapshot        Skip the automatic .repopulse/snapshots/ entry
```

## Configuration — `.repopulserc`

Drop a `.repopulserc` JSON file at the root of the analyzed repo to tune
ignore patterns and bug-tier keywords for your team's workflow.

```json
{
  "ignore": [
    "**/generated/**",
    "**/proto/**"
  ],
  "bugKeywords": {
    "chaos":   ["sev1", "sev2"],
    "normal":  ["defect", "!workaround"],
    "routine": ["rename", "cleanup"]
  }
}
```

- `ignore` — extra glob patterns, appended to the built-in default list
  (lockfiles, `dist/`, `node_modules/`, minified bundles, etc.)
- `bugKeywords` — per-tier lists that are **appended** to the defaults.
  Prefix an entry with `!` to remove a default (e.g. `"!workaround"` drops
  `workaround` from the normal tier). Built-in defaults live in
  `internal/config/config.go:DefaultBugKeywords`.

The classifier also respects Conventional-Commit prefixes: subjects
starting with `feat:`, `chore:`, `docs:`, `style:`, `test:`, `refactor:`,
`ci:`, `build:`, or `perf:` (with or without a `(scope)`) are **never**
classified as bugs, regardless of body content. `fix:` and `revert:` are
left alone and still flow through the keyword matcher.

## How mood is scored

| Signal | Weight | What triggers it |
|---|---|---|
| Commit frequency | 15% | Irregular bursts, long gaps |
| File churn | 25% | Eligible files with churn > 2× current size; volume per day |
| Bug ratio | 30% | Tiered — chaos (revert/hotfix) weighted 1.0, normal (fix/bug) 0.4, routine (typo/lint) 0.1 |
| Authors | 20% | Weekend/night %, bus factor, new-contributor LOC share |
| Coverage | 10% | Low coverage %; redistributes to bug if missing |

Composite 0–40 = Calm, 41–70 = Anxious, 71–100 = Chaotic.

Full signal math: see [SPEC.md](./SPEC.md).

## Requirements

- **Git** on PATH (the tool shells out to `git log`)
- **Go 1.23+** to build from source
- **Node.js 18+ and npm** — only for running the Playwright e2e suite (not for the tool itself)

## Build

```bash
# Primary binary
go build -o repopulse.exe ./cmd/repopulse         # Windows
go build -o repopulse ./cmd/repopulse             # macOS / Linux

# Test fixture generator (required before running Playwright)
go build -o fixture-gen.exe ./cmd/fixture-gen     # Windows
go build -o fixture-gen ./cmd/fixture-gen         # macOS / Linux
```

The product is 100% Go. The `package.json` is a minimal Playwright test harness — `@playwright/test` is its only dependency.

## Output

By default, reports go to `output/` (gitignored). The CLI auto-creates the parent directory. Override with `-output <path>` / `-json <path>` / `-markdown <path>` to write anywhere.

## Test

```bash
go test ./internal/...       # Go unit tests
npx playwright test          # Playwright end-to-end tests
```

Both suites are expected green. Playwright requires `fixture-gen` to be built first.

## Project status

- ✅ **Phase 1 complete** — drill-downs, bug explainability, CODEOWNERS teams, recommendations, markdown digest
- ✅ **Go port complete** — TypeScript fully removed; codebase is 100% Go
- ✅ **Snapshot store + trend chart** — persistent JSON history + multi-series Chart.js line in the report
- ✅ **Plank 1 — baseline drift** — per-author cadence / weekend-night / fix-ratio vs their own 6×-window baseline in a "Worth a 1:1" card
- ✅ **Plank 2 deterministic** — conventional-commit compliance + test-file colocation signal
- ⏳ **Plank 3** — filters, drill-down on Top Churned Files, personal-mirror view
- ⏳ **Plank 2 AI layer** — collect → enrich → render split + Anthropic API / Claude Code skill modes
- ⏳ **Go port polish** — `git show` parallelization, 1-commit off-by-one fix

See `ROADMAP.md` for the full direction.

Full context: see [ROADMAP.md](./ROADMAP.md) for where we've been and where we're going, and [CLAUDE.md](./CLAUDE.md) for developer-facing setup and architecture notes.

## License

MIT.

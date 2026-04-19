# mood-ring 💍

Visualize the "emotional state" of a Git repository as a self-contained HTML dashboard + shareable Markdown digest.

No AI. No external APIs. Just git history + math. Single 4 MB binary.

```bash
./mood-ring.exe /path/to/repo -open
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

- **Mood badge** — composite score 0–100 with mood label *(redesign pending — task #18)*
- **Findings** — 3–8 narrative bullets sorted by severity (alert/warn/info/good)
- **Score breakdown** — per-signal horizontal bars with bands
- **Module mood grid** — per top-level directory with team ownership from CODEOWNERS
- **Hotspots** — expandable rows showing recommendations, top authors, and recent bug-tier commits
- **Commit frequency** / **Mood timeline** / **Bug signal timeline**
- **Bug explainability** — collapsible panel showing which commits matched which keywords
- **Authors** — weekend/night %, bus factor, top-10 table
- **Top churned files** — sortable
- **Coverage** — if your repo generates an Istanbul or lcov report

## Usage

```
mood-ring <repo-path> [options]

Options:
  -window <days>      Analysis window in days (default: 90)
  -output <file>      Output HTML file (default: output/mood-report.html)
  -open               Auto-open in browser after writing
  -since <date>       Start from date, e.g. "2024-01-01" or "3 months ago"
  -ignore <pattern>   Additional glob ignore pattern (repeatable)
  -json <file>        Also write a JSON snapshot (for later -compare)
  -compare <file>     Previous JSON snapshot to diff against
  -markdown <file>    Also write a Markdown digest
```

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
go build -o mood-ring.exe ./cmd/mood-ring         # Windows
go build -o mood-ring ./cmd/mood-ring             # macOS / Linux

# Test fixture generator (required before running Playwright)
go build -o fixture-gen.exe ./cmd/fixture-gen     # Windows
go build -o fixture-gen ./cmd/fixture-gen         # macOS / Linux
```

The product is 100% Go. The `package.json` is a minimal Playwright test harness — `@playwright/test` is its only dependency.

## Output

By default, reports go to `output/` (gitignored). The CLI auto-creates the parent directory. Override with `-output <path>` / `-json <path>` / `-markdown <path>` to write anywhere.

## Test

```bash
go test ./internal/...       # 68 Go unit tests
npx playwright test          # 26 Playwright end-to-end tests
```

Both suites are expected green. Playwright requires `fixture-gen` to be built first.

## Project status

- ✅ **Phase 1 complete** — drill-downs, bug explainability, CODEOWNERS teams, recommendations, markdown digest
- ✅ **Go port complete** — TypeScript fully removed; codebase is 100% Go
- ⏳ **Go port polish** (task #19) — `git show` parallelization, 1-commit off-by-one fix
- 📋 **Phase 2 next** — snapshot store, trend charts, GitHub Action, threshold alerts

Full context: see [ROADMAP.md](./ROADMAP.md) for where we've been and where we're going, and [CLAUDE.md](./CLAUDE.md) for developer-facing setup and architecture notes.

## License

MIT.

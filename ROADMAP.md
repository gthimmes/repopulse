# mood-ring roadmap

## Current state

- ✅ **Phase 1 complete** (1.1–1.5): hotspot drill-downs, bug explainability, CODEOWNERS team tags, hotspot recommendations, markdown digest export.
- ✅ **Go port complete**: codebase is 100% Go. TypeScript fully removed. The fixture generator for Playwright now lives at `cmd/fixture-gen/main.go` with data in `internal/fixtures/ui.go`.
- ⏳ **Go port polish remaining** (task #19): parallelize `git show HEAD:<path>` calls (currently 100 sequential subprocess invocations make Go noticeably slower than it should be — clear parallelization opportunity); investigate a 1-commit off-by-one observed vs the prior TS implementation on a real-data run.
- 📋 **Mood-badge redesign deferred** (task #18): emoji badge replaced with a score-ring or mission-control readout.
- 📋 **Phase 2 not started.**

**Test coverage:** 68 Go unit tests + 26 Playwright e2e = **94 green, 0 failures.**

The only Node footprint that remains is `@playwright/test` — Playwright is the industry-standard browser automation tool and its specs are test infra, not product code. That's explicitly scoped and intentional.

---

## North star

Today the tool is a "check engine light" for a repo: a score, a narrative, some charts. To become load-bearing, it needs to serve three user lenses that the current report only partially touches:

- **Engineer**: "Is the file I'm about to touch on fire? What do I do about it?"
- **Eng leader**: "Is my org trending up or down? Where do I invest? Who's burning out?"
- **PM**: "Can I ship this feature area? Show me the risk."

The north star is moving from *interesting snapshot* to *daily signal in the systems where work already happens* (PRs, tickets, Slack, CI), validated against real incidents, sliced by team.

---

## Language migration (done)

We ported the codebase from TypeScript to Go at the Phase 1 → Phase 2 boundary. Rationale:

- Phase 2's GitHub Action wants a single static binary — Go's natural strength, Node's weakness.
- Phase 4's portfolio aggregator needs real parallelism across repos; Node's async-only model is awkward here.
- Distribution to a multi-stack audience (not just JS shops) is one `go install` / single-binary download, no runtime hell.
- TypeScript's "strong typing" is compile-time only — the runtime fragility we had (Chart.js function strings, stringly-typed commander options) would not have survived Phase 2's complexity.

**Result**: a 4 MB static binary at `./mood-ring.exe` (or `./mood-ring` on unix). Application code lives under `cmd/mood-ring/` and `internal/`. A second 3 MB test-only binary `fixture-gen` at `cmd/fixture-gen/` backs the Playwright UI fixtures. The TS application code is gone.

---

## How to read the time estimates

Each phase has an effort estimate expressed in **engineer-weeks** — roughly how long a single engineer working full-time on the tool would take to ship the phase. It is **not** calendar time and **not** a prediction of how long Claude will take in a session.

- A "1 engineer-week" item ≈ one solid session of focused work, or a day or two of context-switched work.
- Claude working end-to-end on a well-specified item typically compresses these significantly.

Use the estimates to compare phases against each other, not as deadlines.

---

## Phase 1 — Make the report load-bearing ✅

Shipped. Summary of what landed (TS first, then ported to Go):

- **1.1 Hotspot drill-downs**: `<details>`-based expandable rows showing churn rank, top 3 authors, recent bug-tier commits with tier tags.
- **1.2 Bug explainability**: collapsible "Why this score?" panel with per-tier samples + inline keyword highlighting.
- **1.3 Recommendations**: 7-rule heuristic engine (bus-factor, chaos-repeat, rewritten, unowned, multi-owner, stale-buggy, bug-heavy) rendered under each hotspot.
- **1.4 CODEOWNERS**: parser + path matcher with GitHub's last-match-wins semantics. Team chips on hotspots + modules.
- **1.5 Markdown export**: `--markdown <file>` writes a Slack/PR-ready digest.

---

## Phase 2 — Continuous signal, not ad-hoc (~2–4 engineer-weeks)

The snapshot model caps usefulness. Turn the tool into a time series in the systems engineers already watch. **Built in Go.**

### 2.1 Snapshot store
A `.mood-ring/snapshots/` directory of dated JSON snapshots, automatically pruned. Replaces the manual `--json` + `--compare` dance.

### 2.2 Trend charts
Composite score and per-signal trends over 3 / 6 / 12 months, with annotations (releases, big merges, known refactors). Answers "are we getting better or worse" which a single snapshot can't.

### 2.3 GitHub Action
Runs on every PR, posts a comment: *"this PR touches `src/payments/ledger.ts` (hotspot #2, 14 bug commits in 90d) — consider an extra reviewer."* Configurable thresholds.

**Why this is the single highest-leverage Phase 2 item**: it's the only thing on the roadmap that puts the tool in front of every engineer on every PR. A health tool that isn't seen daily doesn't change behavior.

### 2.4 Threshold alerts
`moodring.yml` declares max scores per signal; CI fails or posts to Slack when crossed. Optional, opt-in.

---

## Phase 3 — Connect to where work actually happens (~1–2 engineer-months)

Git-only signals are a proxy. Real work lives in PRs and tickets. This is also where the weakest signal (commit-message regex) gets replaced with ground truth.

### 3.1 GitHub PR metadata
Time-to-first-review, merge latency, reopen rate, stale-PR count, rubber-stamp detection (approved in <60s with no comments).

### 3.2 Issue tracker overlay (GitHub Issues / Jira / Linear)
Count actual bug tickets closed per module. Calibrate the commit-regex bug signal against labeled bugs. **Replace the 30% bugRatio weight with a validated signal.**

### 3.3 Optional incident feeds
Sentry, PagerDuty, or Datadog incident counts per module. "This hotspot caused 3 pages last quarter" is a different sentence than "this file churns a lot."

### 3.4 Test-to-code growth ratio
Is test code keeping up with app code, not just coverage %. Coverage % can stay flat while a codebase accretes untested complexity.

---

## Phase 4 — Portfolio + team view (~2–3 engineer-months)

Eng leaders with 10+ services don't want 10 HTML files.

### 4.1 Multi-repo aggregator
One dashboard, each repo a tile, drill into any one.

### 4.2 Team rollups
Via CODEOWNERS (from Phase 1.4): weekend/night %, bus factor, hotspot count per team. **Privacy defaults**: no individual call-outs by default; personal views are opt-in only.

### 4.3 Benchmarks
Cohort percentiles: "your bug ratio is p80 vs similar-size TS monorepos." Requires a dataset; can start with a hand-curated list of public OSS repos.

### 4.4 Goal setting
Declare quarterly targets per signal, track progress, surface drift in the report.

---

## Phase 5 — Deeper / exploratory (later, validate before committing)

Only pursue these if Phases 1–3 got traction. Each one is a research bet, not a feature with obvious ROI.

### 5.1 Complexity integration
Pipe in `eslintcc` / `lizard`: a 2000-LOC file with cyclomatic 40 is a very different hotspot than a 100-LOC config.

### 5.2 Dependency-graph health
Circular deps, longest import chain, layer-violation growth.

### 5.3 Per-author personal view
Opt-in, privacy-first. Personal weekend/night trend visible only to that engineer.

### 5.4 Incident correlation study
The key validation experiment: does the composite score predict incidents N weeks out? If yes, the tool is indispensable. If no, we're building vibes and should rescope.

---

## Immediate next items (pick one to resume)

From the active task list:

1. **Finish task #19 (Go port polish)** — parallelize `git show`, investigate the 1-commit off-by-one, then port fixture generation to Go so we can delete `src/`. This closes out the port cleanly.
2. **Task #18 (mood badge redesign)** — replace the emoji badge with a score ring. UI-only, no Go porting required, but should probably wait until after #19 to avoid thrashing.
3. **Start Phase 2.3 (GitHub Action)** — the single highest-leverage item on the whole roadmap. The Go binary is ready for it. Would skip over #18 and #19's polish.

**Recommended order**: finish #19 → ship #18 → start Phase 2.3. But any of the three is a reasonable next move depending on whether we want correctness, polish, or reach.

---

## Sequencing logic

- **Phase 1 pays for itself on the next run** — polish the existing signals already earned. ✅
- **Phase 2 is the inflection**: the GitHub Action turns this from "a thing you run once" into "a thing in every PR review."
- **Phase 3 is the accuracy win**: replaces the weakest 30%-weight signal with validated data. Do not build Phase 4/5 before Phase 3; they compound on top of a signal we haven't validated yet.
- **Phases 4 and 5 only pay off if 1–3 got traction.** Don't build them on spec.

## The one thing to build next if we could only build one

**The Phase 2.3 GitHub Action.** It's the only item on the whole roadmap that guarantees the tool is seen daily by every engineer on every PR — which is the only way a health tool changes behavior. Phase 1 exists largely to make that Action's output actionable enough to be worth reading, and the Go port exists to make shipping it as a single binary trivial.

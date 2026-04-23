# repopulse roadmap

## Direction

**repopulse is a lens, not a scorecard.** You open it when you want to understand what's going on in a codebase or with the people contributing to it — not because a dashboard told you to. Two lenses, equally weighted:

- **Codebase health** — what's on fire, what's drifting, where are the standards we said we'd hold, are we actually holding them?
- **Contributor health** — is someone's load shifting in a way worth a 1:1? Are patterns emerging that point to struggle, burnout, or a gap we can coach on?

Both lenses emit **things to look at**, not scores to rank against. The mood badge and composite score remain (they're useful at-a-glance), but the forward-looking work is about making the report *explorable* and the signals *personal without being performative*.

---

## Current state

- ✅ **Phase 1 complete** (1.1–1.5): hotspot drill-downs, bug explainability, CODEOWNERS team tags, hotspot recommendations, markdown digest export.
- ✅ **Go port complete**: codebase is 100% Go. Single 4 MB binary, `git` as the only runtime dep.
- ✅ **Snapshot store + trend chart shipped** (was "Phase 2.1 + 2.2"). Every run writes `<repo>/.repopulse/snapshots/<ts>.json`; the HTML report carries a multi-series trend line.
- ✅ **UX polish shipped** this cycle: header shows date range + "0 = calm / 100 = chaotic / lower is better," bug-tier explainer legend, conventional-commit prefix veto (`feat:`/`chore:`/etc. never classify as bugs), `.repopulserc` now appends keywords to defaults instead of replacing, full rebrand `mood-ring → repopulse`.
- ✅ **Plank 1 shipped** this cycle: per-author baseline drift. Each contributor compared against their own rolling 6×-window baseline on commit cadence, weekend/night %, and fix-vs-feature mix. Flagged deltas surface in a "Worth a 1:1" card with alert/watch/info severity. No cross-author ranking anywhere. See `internal/baseline/`.
- ✅ **Plank 2 shipped** this cycle (deterministic layer): conventional-commit compliance + test density (test-file-to-source ratio per module). Earlier attempt at filename-based colocation was replaced with a density ratio because filename matching required teams to name tests after individual source classes — too brittle across real-world test-organization conventions. Languages auto-detected from HEAD's tracked files. AI enrichment layer for plank 2 still pending. See `internal/standards/`.
- ✅ **Plank 3 shipped** this cycle: Top Churned Files now drillable (per-author chips + recent commits with bug-tier color coding inside `<details>` rows). New **Contributors explorer** at the bottom of the report — full unbounded list of every contributor in the window, sorted by LOC desc, scrollable, drillable. Each row's expanded panel shows: stats grid, baseline-drift detail (or "no flags this window"), conventional-commit compliance bar, top files they touched. Folds in what was previously the standalone "Worth a 1:1" card — flagged contributors get an inline alert/watch pill in the row's Watch? column. The earlier `--me` flag and personal-mirror banner were removed; an engineer just opens the report and finds themselves in the contributor list. See `renderContributorsSection` in `internal/render/template.go`.
- ✅ **Badge redesign shipped** this cycle: dropped the calm/anxious/chaotic emoji + mood label. Replaced with a "REPO PRESSURE" headline, large numeric score, color-coded band pill (Steady 0-40 / Active 41-70 / Volatile 71-100), and a horizontal gradient bar (green→amber→red) with a marker showing where this run lands. Same scoring math, more honest framing — the score isn't really mood, it's a weighted composite of churn density + bug-tier ratio + off-hours load + bus factor + coverage.
- ✅ **Phase 3.1 — PR Flow shipped** this cycle: optional GitHub PR metrics. Gated on a token (`--github-token` flag or `GITHUB_TOKEN` env). Pulls merged PRs in the window via REST, caches them at `<repo>/.repopulse/pr-cache/` for incremental refresh, and produces the "PR Flow" card: cycle-time percentiles (p50/p75/p95), time-to-first-review, top reviewers (workload concentration), rubber-stamp rate (approved <60s with 0 comments), self-merge rate. No token → section is skipped silently. Rate-limit hits fall back to cached data with a warning banner. Turns the tool from "code health only" into "code + team-flow health." See `internal/github/` and `internal/prmetrics/`.
- 📋 **Mood-badge redesign deferred**: emoji → score-ring. UI-only, low priority.
- ⏳ **Go port polish deferred**: parallelize `git show HEAD:<path>`, investigate 1-commit off-by-one.

**Test coverage:** 131 Go unit tests + 31 Playwright e2e = **162 green, 0 failures.**

---

## The three planks (active direction)

Forward work reorganizes around three planks. Each plank is a product capability, not a sprint.

### Plank 1 — Baseline-drift detection ("is someone struggling?")

Per-contributor signals compared against **that contributor's own rolling baseline**, not against a team average or a leaderboard. The output isn't "Alex commits less than median" — it's "Alex's weekend/night commits doubled vs their 6-month baseline and their fix-vs-feature ratio shifted 20 pp toward fix; worth a 1:1."

What we compute per author:

- **Commit cadence drift** — commits/week now vs over the baseline.
- **Weekend/night drift** — fraction of their commits outside business hours.
- **Fix-vs-feature drift** — classified bug-tier commits / total for this author.
- **Self-revert rate** *(future)* — how often their own commits got reverted.
- **PR cycle-time drift** *(future, needs PR metadata plank)* — time-to-review, time-to-merge on their PRs.
- **Review latency drift** *(future, needs PR metadata)* — how fast they respond to review requests.

Flagging rule: deltas are surfaced as **"Worth a 1:1"** cards only when the relative change is meaningful **and** the absolute numbers aren't noise (e.g., 100% increase on 1 commit/week doesn't fire). Framing in the UI is about things to discuss, not scores to act on.

Privacy: no cross-author ranking surface anywhere. The cards are anchored by author name/email from git, full stop — just like the existing authors panel.

### Plank 2 — Standards detection (deterministic core, AI enrichment optional)

Is the codebase following the standards we said we'd hold?

**Layer A — deterministic** (no AI, runs always):
- Commit message format (conventional-commits, subject length, body presence).
- PR description presence & length *(needs PR metadata)*.
- Test density per module (test files / source files). Ratio-based, not filename-based, so it catches "does this module have tests at all" without false negatives when tests are split by action or class-naming differs from source.
- Branch naming convention.
- File-size limits (flag > N lines / > N bytes).
- Lint-config presence & enforcement (delegate to the repo's existing linter; just surface violation trends).
- New-code-without-tests ratio.

**Layer B — AI-augmented** (opt-in, gated on API key OR Claude Code skill):
- Does this PR/diff follow the team's established patterns in neighboring files?
- Are new abstractions being introduced when existing ones would work?
- Are error-handling / logging / naming conventions consistent with the rest of the file/module?

Skill mode is uniquely good here — Claude already has the surrounding code in context, so we don't have to ship a huge diff over an API.

### Plank 3 — Exploration over scorecard

The current report is a read-once artifact. Move toward: open it, follow a thread, drill into what's interesting.

- Query CLI (`repopulse ask "PRs Alex reviewed in <60s last quarter"`), *eventually* natural-language powered by the skill.
- Deeper HTML drill-downs — filter authors, filter by module, filter by time range inside the same report.
- "Trace" mode — click a hotspot, see every commit touching it in the window, with author, tier, and linked PR.
- Personal-mirror view — engineer runs `repopulse --me me@example.com` and sees only their own drift signals, same data the coaching view would show their lead.

---

## Architecture shift to support the three planks

To make plank 2 work in both no-AI and AI-enriched modes, split the pipeline into three phases:

1. **Collect** — git + config → `deterministic.json` (what we have today, just named).
2. **Enrich** — optional: `deterministic.json` → `enriched.json` with AI-added narrative bullets, standards-adherence verdicts, drift interpretations. Gated on `ANTHROPIC_API_KEY` env or a Claude Code skill invocation.
3. **Render** — `deterministic.json` (+ optional `enriched.json`) → HTML / Markdown.

Add a `-from-json` CLI entry point so the binary can render an externally-enriched snapshot, enabling:
- **Mode A** (no AI): run normally.
- **Mode B** (API key): same run, but the enrich phase calls Claude.
- **Mode C** (Claude Code skill): a markdown skill that runs collect → lets Claude analyze → runs render. No API key needed because Claude is already there.

This refactor is cheap (mostly moving already-structured code) and unblocks plank 2's AI layer without making the core path depend on AI.

---

## Immediate next item

**Plank 1 — baseline-drift detection.** Chosen first because:

- It uses data we already collect (git commits — no new integrations needed).
- It delivers the "is someone struggling?" signal you explicitly asked for.
- It proves the lens-vs-scorecard thesis before we invest in AI plumbing for plank 2.
- The snapshot store we just built is well-suited to powering it (historical baselines).

---

## Parking lot (former roadmap items, still relevant but not active)

These were the pre-pivot Phase 2.3+ items. They're not dead — several fold naturally into the planks — but they're no longer the ordering constraint.

- **GitHub Action** (was 2.3) — PR comments with hotspot/plank-1/plank-2 context. Probably becomes the *delivery surface* for planks 1+2 once those are solid.
- **Threshold alerts** (was 2.4) — `repopulse.yml` with CI fail / Slack hook. Only useful after planks 1+2 produce signals worth alerting on.
- **GitHub PR metadata** (was 3.1) — time-to-first-review, merge latency, rubber-stamp detection. Feeds plank 1 (review latency drift) and plank 2 (PR description standards) — **likely the next integration after plank 1 lands.**
- **Issue-tracker overlay** (was 3.2) — Jira/Linear/GH Issues bug ground truth. Calibrates the bug-keyword signal but isn't urgent now.
- **Incident feeds** (was 3.3) — Sentry/PagerDuty per-module incident counts. Aspirational.
- **Test-to-code growth ratio** (was 3.4) — folds naturally into plank 2's "new code without tests" standard.
- **Multi-repo aggregator** (was 4.1) — one dashboard, many repos. Deferred until a single repo's lens is actually useful.
- **Team rollups** (was 4.2) — via CODEOWNERS. Becomes a natural grouping overlay on plank 1 once plank 1 is solid.
- **Benchmarks** (was 4.3) — "your ratio is p80 vs similar repos." Requires a dataset we don't have; deprioritized.
- **Goal setting** (was 4.4) — quarterly targets per signal. Contradicts the lens-not-scorecard framing; probably won't build as-specified.
- **Complexity integration** (was 5.1) — cyclomatic via lizard/eslintcc. Feeds plank 2 as a deterministic standard.
- **Dependency-graph health** (was 5.2) — layer violations. Feeds plank 2.
- **Per-author personal view** (was 5.3) — now explicitly plank 3's "personal mirror."
- **Incident correlation study** (was 5.4) — "does the composite score predict incidents?" The real validation experiment, still open.
- **Bus-factor deep dive** (was 5.5) — per-module + recency-weighted bus factor. Feeds plank 1 as a module-level signal.

---

## Non-goals (explicit)

- **Engineering productivity rankings.** No "engineers by commits" leaderboard, ever.
- **Performance-review-ready scorecards.** The signals are for conversation, not evaluation.
- **AI-required features on the happy path.** Plank 1 is pure Go; plank 2 has a deterministic layer that works without AI. AI enriches, never gatekeeps.
- **Online service.** The tool stays a local CLI producing a self-contained HTML file. No accounts, no sync, no backend.

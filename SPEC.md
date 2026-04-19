# SPEC.md — Technical specification

> **Scope**: this document describes the **current** signal math and scoring — as of the end of Phase 1 / post-Go-port. It supersedes the original single-regex bug signal, equal-weight scorer, and simpler set of signals that shipped in the initial TypeScript version (now removed).

## Overview

`repopulse` reads a local Git repository and outputs a self-contained HTML file plus optional Markdown digest visualizing the "emotional state" of the codebase over a configurable time window.

Mood is derived from **six signals**: commit frequency, file churn, bug ratio (tiered), authors, modules, hotspots. Coverage is a seventh optional signal.

Source of truth: `internal/signals/<signal>.go` + `internal/scorer/scorer.go`.

---

## Signal weights (composite scorer)

```
commitFrequency: 0.15
fileChurn:       0.25
bugRatio:        0.30
coverage:        0.10   (redistributes to bugRatio if no coverage found)
authors:         0.20
                 ----
                 1.00
```

Source of truth: `internal/scorer/scorer.go`.

---

## Mood thresholds

```
composite  0–40  → calm     😌
composite 41–70  → anxious  😬
composite 71–100 → chaotic  🔥
```

---

## 1. Commit frequency (15%)

**What it measures:** consistency of commit activity over the window.

**Compute:**
1. Bucket commits by calendar day in UTC for `windowDays` days ending at "end of today."
2. For each day in the window, record count (0 if no commits).
3. Mean μ, stdDev σ over the full bucket array (including zero days).
4. Coefficient of variation: `cv = σ / μ` (0 if μ == 0).
5. Longest consecutive zero-commit streak (`longestGapDays`).

**Score:**
```
cvNormalized:
  cv < 0.5  → 0–20
  cv < 1.0  → 20–40
  cv < 1.5  → 40–60
  cv < 2.5  → 60–80
  cv ≥ 2.5  → 80–100

gapScore = min(60, longestGap / windowDays * 100)

score = min(100, round(cvNormalized * 0.6 + gapScore * 0.4))
```

Empty-commit special case: `score = 60, longestGapDays = windowDays`.

---

## 2. File churn (25%)

**What it measures:** how much of the eligible codebase is being heavily rewritten.

**Compute:**
1. Aggregate `added + removed` per file path across the window (excluding files that match any default or user-supplied ignore pattern).
2. Sort by total churn descending. Resolve current line counts for the top 100 only (cost: one `git show HEAD:<path>` per file).
3. Per file: `rawRatio = total_churn / max(1, current_loc)`; `cappedRatio = min(10, rawRatio)`.
4. `rewritten = rawRatio > 5` (file was effectively replaced — excluded from eligibility).
5. **Eligible** = `total_churn ≥ 20` AND NOT `rewritten` (small or rewritten files aren't real hand-authored churn).

**Score — two sub-signals, weighted 60/40:**

```
densityPct = highChurnFiles / max(1, eligibleCount) * 100
  where highChurnFiles = files in eligible set with cappedRatio > 2.0

densityScore:
  < 5%  → 0–20
  < 15% → 20–50
  < 30% → 50–80
  ≥ 30% → 80–100

throughputScore (based on total LOC changed / windowDays):
  < 200    → 0–20
  < 1000   → 20–50
  < 5000   → 50–75
  ≥ 5000   → 75–100

score = min(100, round(densityScore * 0.6 + throughputScore * 0.4))
```

---

## 3. Bug ratio (30%) — tiered

**What it measures:** prevalence and severity of bug-related commits.

**Tiered classification** (weights applied to the ratio calculation):

| Tier | Weight | Keywords |
|---|---|---|
| chaos | 1.0 | `revert reverted rollback hotfix urgent regression broken broke critical emergency oops p0 p1` |
| normal | 0.4 | `fix fixes fixed bug patch workaround` |
| routine | 0.1 | `typo lint format formatting whitespace indent` |

Revert commits (detected via subject regex `^(?:Revert\s+["']|revert\s*[:(])`) always → chaos, with matched keyword `"(revert)"`.

**Precedence when a message matches multiple tiers:**
- If revert → chaos
- Else chaos wins over routine + normal
- Else routine wins over normal (so `"fix: typo"` → routine)

**Compute:**
```
weightedBugs = chaosCount * 1.0 + normalCount * 0.4 + routineCount * 0.1
weightedRatioPct = weightedBugs / totalCommits * 100

base:
  < 5%   → 0–20
  < 15%  → 20–55
  < 30%  → 55–85
  ≥ 30%  → 85–100

clusterBonus:
  longestFixStreak ≥ 7 → +12
  longestFixStreak ≥ 4 → +6

revertBonus = min(15, revertedWithin7d * 4)
  where revertedWithin7d = count of revert commits whose target commit is within 7 days

score = min(100, round(base + clusterBonus + revertBonus))
```

**Explainability**: up to 20 commits per tier are retained (newest first) in `classifiedSamples` for the "Why this score?" UI panel and the markdown digest.

---

## 4. Authors (20%)

**What it measures:** after-hours work pressure, ownership concentration, new-contributor load.

**Compute:**
1. Aggregate per-author (by email): commit count, total LOC changed, weekend/night commit count, first-seen date.
2. `weekendNightPct = (weekend_or_night_commits / total_commits) * 100`. "Weekend" = UTC Saturday/Sunday. "Night" = UTC hour ≥ 20 OR < 7.
3. `busFactorTop1Pct = topAuthor.commits / total_commits * 100`; same for top 3.
4. "New contributor" = author whose email is NOT in the pre-window email set (fetched separately via `git log --before=<windowStart> --format=%ae`). Fallback (if that set isn't available): `firstSeen >= windowStart`.
5. `newContributorChurnPct = new_contributor_LOC / total_LOC * 100`.

**Score — weighted combination:**

```
weekendNightScore:
  < 20%  → 0–60   linear
  ≥ 20%  → 60–100 linear (capped at 100)

busFactorScore (based on top1Pct):
  < 30%  → 0–30    linear
  < 60%  → 30–70   linear
  ≥ 60%  → 70–100  linear (capped at 100)

newContribScore:
  < 30%  → 0–20    linear
  ≥ 30%  → 20–40   linear (capped at 40)

score = round(weekendNightScore * 0.45 + busFactorScore * 0.35 + newContribScore * 0.20)
```

---

## 5. Modules (informational, per-module mood)

**What it measures:** which top-level directories are hot vs calm.

Grouping: by the first path segment (`src/auth/session.ts` → `src`), depth configurable (default 1). Single-file paths → `(root)`.

For each module with ≥ 3 commits:

```
shareScore = min(100, (module_LOC / total_LOC) * 300)
  (33% of total churn → 100)

authorConcentrationScore:
  1 author  → 100
  N authors → max(0, 100 - (N-1) * 20)

weightedBug = (chaosRatio * 1.0 + (bugRatio - chaosRatio) * 0.4) * 100
bugSubScore = min(100, weightedBug * 2)

score = round(
  shareScore * 0.30 +
  bugSubScore * 0.40 +
  authorConcentrationScore * 0.15 +
  min(100, module_LOC / 500) * 0.15
)
```

Module mood = same thresholds as composite (0–40 calm, 41–70 anxious, 71–100 chaotic).

Owners: aggregated from CODEOWNERS — ranked by file count, top 2.

---

## 6. Hotspots (informational, Feathers-style)

**What it measures:** specific files at the intersection of high churn AND bug involvement.

```
hotspotScore = round(
  (file_churn / max_churn_in_window) * 100 * 0.4 +
  (weighted_bug_touches / max_bug_touches) * 100 * 0.6
)

weighted_bug_touches =
  chaos_touches   * 1.0 +
  normal_touches  * 0.4 +
  routine_touches * 0.1
```

Files with `bugTouches == 0` are excluded (that's just top-churn, already covered by the churn signal). Top 15 by `hotspotScore` surfaced.

Each entry carries drill-down data: `lastTouched`, `topAuthorsOfFile` (top 3 by commit count), `recentBugCommits` (up to 10 newest bug-tier commits with tier tags), `owners`, `recommendations`.

---

## 7. Recommendations engine

Per-hotspot, up to 3 rules fire (sorted alert > warn > info):

| Rule | Trigger | Severity |
|---|---|---|
| bus-factor | top1 ≥ 60% of file commits | alert ≥ 80%, warn otherwise |
| chaos-repeat | chaosTouches ≥ 3 | alert ≥ 5, warn otherwise |
| rewritten | churn entry has `rewritten: true` | info |
| unowned | `owners` is empty | warn if chaosTouches > 0, info otherwise |
| multi-owner | `len(owners) ≥ 2` | info |
| stale-buggy | `lastTouched` ≥ 30 days before window end AND bugTouches > 0 | info |
| bug-heavy | `bugTouches / totalCommits ≥ 40%` (requires totalCommits ≥ 5) | warn ≥ 60%, info otherwise |

Source: `internal/signals/recommendations.go`.

---

## 8. Coverage (10%, optional)

**Detection waterfall:**
1. `coverage/coverage-summary.json` (Istanbul JSON — uses `total.lines.pct`)
2. `lcov.info` at root (sum `LF:` lines found, `LH:` lines hit → `LH/LF*100`)
3. `coverage/lcov.info`

If nothing found: signal is skipped, its 10% weight redistributes to bugRatio.

**Score** based on absolute percentage:
```
≥ 80%: score = 10   (healthy)
≥ 60%: score = 35
≥ 40%: score = 60
<  40%: score = 85  (very low)
```

---

## Composite scoring

```go
weighted :=
  CommitFrequency.Score * wFreq +
  FileChurn.Score       * wChurn +
  BugRatio.Score        * wBug +
  Authors.Score         * wAuth +
  Coverage.Score        * wCov   // only if Coverage != nil

totalWeight := wFreq + wChurn + wBug + wAuth + (wCov if coverage else 0)
composite := round(weighted / totalWeight)
```

When coverage is missing, `wBug := wBug + wCov` and `wCov := 0` (single-line change — see `scorer.go`).

**Breakdown in output** = each signal's contribution to composite: `round(signal.Score * weight)`.

---

## Rolling 7-day timeline

Separate from the composite — used for the mood timeline chart.

For each day in the window (starting day 7):
```
n     = commits in trailing 7 days
bugs  = bug-tier commits (any tier) in window
chaos = chaos-tier commits in window
perDay = n / 7
bugPct = bugs / n

volumeScore:
  perDay < 2   → 0–20
  perDay < 10  → 20–50
  perDay < 30  → 50–75
  perDay ≥ 30  → 75–100

chaosPct = chaos / n * 100
bugScore = min(100, chaosPct * 4 + (bugPct * 100 - chaosPct) * 1.2)

dryPenalty = 30 if n == 0, else 0

composite_for_day = min(100, round(volumeScore * 0.35 + bugScore * 0.55 + dryPenalty))
```

Source: `internal/narrative/narrative.go → ComputeRollingTimeline`.

---

## HTML output sections (in order)

1. **Limited-history banner** (if fewer than 10 commits in window)
2. **Mood badge** — emoji + label + composite score + window metadata. *(Slated for redesign — see task #18)*
3. **Findings** (3–8 narrative bullets, sorted alert > warn > info > good)
4. **Stats row** — commits, files touched, bug %, commits/day
5. **Score breakdown** (horizontal bar chart — one bar per signal, colored by score band)
6. **Module mood grid** — one glass card per module with score pill, meta, owner chips
7. **Hotspots** — expandable `<details>` rows with per-file drill-down (recommendations, authors, recent bug commits)
8. **Commit frequency histogram**
9. **Mood timeline + Bug signal timeline** (side by side)
10. **Bug signal explainability** — collapsible panel with per-tier samples + keyword highlighting
11. **Authors** — mini-stats + top-10 table
12. **Top churned files** — sortable
13. **Coverage panel** — only if coverage detected
14. **Footer** — generated-at + window dates

---

## Markdown digest sections

Order: header → score line → Findings → Hotspots table → Hottest modules table → Top recommendations → Footer (with optional link back to the HTML report).

Tables use GitHub-flavored markdown. Team ownership rendered as `` `@org/team-name` `` (inline code); unowned rows show `_(unowned)_`.

Empty sections are omitted. `Score line` and footer always render.

---

## Error handling

| Scenario | Behavior |
|---|---|
| Path is not a git repo | Exit 1: "No git repository found at ./path" |
| Repo has 0 commits in window | Exit 1: "Repository has no commits in the analysis window" |
| Repo has < 10 commits | Still generate, show banner, do not error |
| `.repopulserc` parse fails | Warn to stderr, use defaults, continue |
| Coverage file parse fails | Warn to stderr, skip coverage signal |
| `git show HEAD:<path>` fails (deleted file) | Use 0 for line count (ratio goes to cap) |
| CODEOWNERS missing | Silent — no chips rendered, no errors |

---

## Known gaps / caveats (Phase 1 exit)

1. **Churn signal runtime**: 100 sequential `git show HEAD:<path>` subprocess calls make the tool noticeably slower than necessary on large repos (measured on a ~1000-commit test run). Fix: parallelize with goroutines (task #19).
2. **1-commit off-by-one**: during the Go port, one top-level module's commit count came out 1 lower than the prior TS run on the same repo. Likely a timezone/window-boundary edge case. Under investigation (task #19).
3. **Mood badge still uses emoji**: slated for redesign as a score ring (task #18).

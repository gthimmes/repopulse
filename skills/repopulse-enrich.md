---
name: repopulse-enrich
description: Enrich a repopulse deterministic snapshot with AI-authored interpretation, then render the final HTML/Markdown report. Use when the user asks for an AI-augmented repopulse report, runs `/repopulse-enrich`, or wants Plank-2 Layer-B enrichment without setting up an Anthropic API key. Triggers on phrases like "enrich the repopulse report", "run repopulse with AI read", "add AI interpretation to repopulse".
---

# repopulse-enrich

This is **Mode C** of repopulse's Plank-2 Layer-B AI enrichment: a Claude Code skill produces the enrichment in-conversation, no Anthropic API key required.

## Workflow

1. **Collect** the deterministic snapshot.
   ```
   ./repopulse.exe <repo-path> -emit-json output/deterministic.json -no-snapshot
   ```
   This writes the full deterministic snapshot to disk and **does not** render the HTML yet. The user may still want a deterministic-only HTML alongside; that's optional.

2. **Read** `output/deterministic.json`. Pay attention to:
   - `moodResult.compositeScore` and `moodResult.mood` (the band)
   - `moodResult.signals.bugRatio` (chaos / normal / routine commit counts and ratio)
   - `moodResult.signals.fileChurn` (high-churn files, total LOC, LOC/day)
   - `moodResult.signals.modules.modules` (per-module pressure scores)
   - `moodResult.signals.hotspots.hotspots` (top files by hotspot score)
   - `moodResult.signals.authors` (bus factor, weekend/night %, new-contributor share)
   - `moodResult.signals.standards.conventionalCommits` and `.testDensity`
   - `moodResult.signals.authorDrift.authors` — **only contributors with a non-empty `flags` array matter for the drift section**
   - `moodResult.signals.prFlow` (optional; null when no GitHub token was available)
   - `moodResult.narrative` — the deterministic findings already shown to the user; **do not duplicate them word-for-word**

3. **Interpret**. Produce JSON matching this exact shape:
   ```json
   {
     "type": "enrichment",
     "schemaVersion": 1,
     "source": "claude-code-skill",
     "model": "<your model id, e.g. claude-opus-4-7>",
     "generatedAt": "<ISO 8601 UTC timestamp>",
     "narrative": [
       {"kind": "info" | "warn" | "alert" | "good", "text": "<one sentence ≤180 chars>"}
     ],
     "standards": {
       "headline": "<one short phrase>",
       "summary": "<two sentences max>",
       "suggestions": ["<one short suggestion>", "..."]
     },
     "drift": [
       {"email": "<exact email from authorDrift>", "reading": "<coaching-context sentence>", "suggestion": "<optional short follow-up>"}
     ],
     "notes": ["<optional caveats>"]
   }
   ```

   **Hard rules:**
   - This is a **lens, not a scorecard**. Never rank contributors against each other.
   - Drift readings sound like coaching context for a 1:1, not feedback. Stick to the `email` field exactly as it appears in the input — the renderer matches it back to the contributor row.
   - 3–6 narrative bullets MAX. Skip the section if you have nothing to add beyond the deterministic findings.
   - `standards` is optional; only include it when you can say something more useful than the numbers already say.
   - Only include `drift` entries for contributors who appeared in `authorDrift` with at least one flag. Skip anyone where you can't say something useful.
   - No code suggestions. No file-by-file analysis. You don't have the source.
   - No emoji.

4. **Write** the JSON to `output/enriched.json` (or wherever the user prefers).

5. **Render** the final report by feeding both files back to the binary:
   ```
   ./repopulse.exe -from-json output/deterministic.json -enriched output/enriched.json -output output/repopulse-report.html
   ```
   Optionally add `-markdown output/digest.md` for the Markdown digest.

## Failure handling

- If `output/deterministic.json` is missing, ask the user to run step 1 first; do not fabricate signals.
- If a deterministic field you'd want to interpret is empty (e.g., `prFlow` is null), skip that part of the enrichment rather than commenting on absent data.
- Network or environment problems (the user's binary won't run, etc.) are reported back to the user — don't try to render manually.

## Why this skill exists

Mode B (`-enrich` with `ANTHROPIC_API_KEY`) sends an aggregate digest to the API and pays per-call. Mode C (this skill) does the same job inside the existing Claude Code conversation — no extra API key, no per-run billing, and Claude already has surrounding context if the user wants follow-ups. The deterministic numbers remain the source of truth in either mode.

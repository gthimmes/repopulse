package render

import (
	"fmt"
	"html"
	"strings"

	"repopulse/internal/compare"
	"repopulse/internal/types"
)

// RenderHTML returns the full self-contained HTML report. `trends` is
// the full list of historical snapshots (may be empty or nil) used to
// render the trend-chart section; pass nil when none are available.
func RenderHTML(data types.MoodResult, meta types.RepoMeta, delta *types.MoodDelta, trends []compare.ReportSnapshot) string {
	avgPerDay := float64(meta.AnalyzedCommits) / float64(max1(meta.WindowDays))
	bugPct := int(data.Signals.BugRatio.Ratio*100 + 0.5)
	filesTouched := data.Signals.FileChurn.TotalFilesTouched

	freqJS := CommitFrequencyChart(data)
	timelineJS := MoodTimelineChart(data)
	bugJS := BugSignalChart(data)
	covJS := CoverageChart(data)
	breakdownJS := ScoreBreakdownChart(data)

	churnRows := renderChurnRows(data.Signals.FileChurn.TopChurners)
	narrativeHTML := renderNarrative(data.Narrative)
	modulesHTML := renderModules(data.Signals.Modules.Modules)
	hotspotsHTML := renderHotspots(data.Signals.Hotspots.Hotspots)
	deltaHTML := ""
	if delta != nil {
		deltaHTML = renderDelta(*delta)
	}
	bugWhyHTML := renderBugExplainability(data.Signals.BugRatio)
	standardsHTML := renderStandards(data.Signals.Standards)
	prFlowHTML := renderPRFlow(data.Signals.PRFlow)
	trendSectionHTML := TrendSection(trends)
	trendInit := ""
	if len(trends) > 1 {
		trendInit = fmt.Sprintf("initChart('trendChart', %s);", TrendChart(trends))
	}

	limitedBanner := ""
	if meta.HasLimitedHistory {
		limitedBanner = `<div class="limited-banner">&#x26A0;&#xFE0F; Limited history &mdash; fewer than 10 commits found. Signals may not be reliable.</div>`
	}

	coveragePanel := ""
	if data.Signals.Coverage != nil {
		cov := data.Signals.Coverage
		pctColor := "#f87171"
		if cov.Percentage >= 80 {
			pctColor = "#4ade80"
		} else if cov.Percentage >= 60 {
			pctColor = "#fbbf24"
		}
		source := "lcov.info"
		if cov.Source == "istanbul" {
			source = "Istanbul JSON"
		}
		coveragePanel = fmt.Sprintf(`<div class="card" style="margin-bottom:24px">
        <h2>Test Coverage</h2>
        <div style="display:flex;align-items:center;gap:32px">
          <div class="coverage-ring">
            <canvas id="coverageChart"></canvas>
            <div class="pct" style="color:%s">%.1f%%</div>
          </div>
          <div style="font-family:'JetBrains Mono',monospace;font-size:12px;color:var(--text-dim);letter-spacing:0.06em">
            <div>SOURCE: %s</div>
            <div style="margin-top:4px">SIGNAL: %d/100</div>
          </div>
        </div>
      </div>`, pctColor, cov.Percentage, source, cov.Score)
	}

	coverageInit := ""
	if data.Signals.Coverage != nil {
		coverageInit = fmt.Sprintf("initChart('coverageChart', %s);", covJS)
	}
	coverageHeight := "240px"
	if data.Signals.Coverage != nil {
		coverageHeight = "260px"
	}

	moodLabelUpper := strings.ToUpper(string(data.Mood))
	_ = moodLabelUpper

	var sb strings.Builder
	fmt.Fprintf(&sb, `<!DOCTYPE html>
<html lang="en">
<head>
  <meta charset="UTF-8">
  <meta name="viewport" content="width=device-width, initial-scale=1.0">
  <title>Repopulse — %s</title>
  <script src="https://cdn.jsdelivr.net/npm/chart.js"></script>
  <style>
%s
  </style>
</head>
<body>
  <div class="container">
    %s

    <!-- Pressure Badge -->
    <div class="mood-badge pressure-badge">
      <div class="pressure-headline">REPO PRESSURE</div>
      <div class="pressure-score">%d<span class="pressure-of">/ 100</span>%s</div>
      <div class="pressure-band band-%s">%s</div>
      <div class="pressure-bar">
        <div class="pressure-bar-track">
          <div class="pressure-bar-marker" style="left:%d%%"></div>
        </div>
        <div class="pressure-bar-labels">
          <span class="band-label band-steady">0 &middot; Steady &middot; 40</span>
          <span class="band-label band-active">41 &middot; Active &middot; 70</span>
          <span class="band-label band-volatile">71 &middot; Volatile &middot; 100</span>
        </div>
      </div>
      <div class="mood-meta">%s &middot; %dD (%s &rarr; %s) &middot; %d COMMITS</div>
    </div>

    <!-- Narrative findings -->
    <div class="card narrative" style="margin-bottom:24px">
      <h2>Findings</h2>
      %s
    </div>

    <!-- Optional AI enrichment (Plank 2 Layer B) -->
    %s

    <!-- Trends across snapshots -->
    %s

    <!-- Standards (Plank 2) -->
    %s

    <!-- PR Flow (Phase 3.1 — GitHub PR metrics, if token wired up) -->
    %s

    <!-- Stats Row -->
    <div class="stats-row">
      <div class="stat-card">
        <div class="stat-value">%d</div>
        <div class="stat-label">Commits<span class="sublabel">last %d days</span></div>
      </div>
      <div class="stat-card">
        <div class="stat-value">%s</div>
        <div class="stat-label">Files touched<span class="sublabel">%d eligible</span></div>
      </div>
      <div class="stat-card">
        <div class="stat-value">%d%%</div>
        <div class="stat-label">Bug commits<span class="sublabel">chaos · %d</span></div>
      </div>
      <div class="stat-card">
        <div class="stat-value">%.1f</div>
        <div class="stat-label">Commits/day<span class="sublabel">%d authors</span></div>
      </div>
    </div>

    <!-- Score Breakdown -->
    <div class="card" style="margin-bottom:24px">
      <div class="chart-container" style="height:%s">
        <canvas id="breakdownChart"></canvas>
      </div>
    </div>

    %s

    %s

    <!-- Commit Frequency -->
    <div class="card" style="margin-bottom:24px">
      <div class="chart-container">
        <canvas id="freqChart"></canvas>
      </div>
    </div>

    <!-- Timeline + Bug Signal -->
    <div class="side-by-side">
      <div class="card">
        <div class="chart-container">
          <canvas id="timelineChart"></canvas>
        </div>
      </div>
      <div class="card">
        <div class="chart-container">
          <canvas id="bugChart"></canvas>
        </div>
      </div>
    </div>

    <!-- Bug signal explainability -->
    <div class="card" style="margin-bottom:24px">
      %s
    </div>

    <!-- Top Churned Files (drillable, Plank 3) -->
    <div class="card" style="margin-bottom:24px">
      <h2>Top Churned Files <span class="sub">· click any row to drill in</span></h2>
      <div class="hotspot-list">
        %s
      </div>
    </div>

    %s

    <!-- Contributors explorer (bottom of report, drillable) -->
    %s

    <!-- Footer -->
    <div class="footer">
      Generated by <strong>repopulse</strong> on %s
      &middot; %s – %s
    </div>
  </div>

  <script>
    (function() {
      if (window.Chart) {
        Chart.defaults.color = '#94a3b8';
        Chart.defaults.borderColor = 'rgba(148, 163, 184, 0.10)';
        Chart.defaults.font.family = "'Inter', system-ui, sans-serif";
        Chart.defaults.plugins.tooltip.backgroundColor = 'rgba(15, 23, 42, 0.95)';
        Chart.defaults.plugins.tooltip.titleColor = '#e2e8f0';
        Chart.defaults.plugins.tooltip.bodyColor = '#cbd5e1';
        Chart.defaults.plugins.tooltip.borderColor = 'rgba(148, 163, 184, 0.25)';
        Chart.defaults.plugins.tooltip.borderWidth = 1;
        Chart.defaults.plugins.tooltip.padding = 10;
        Chart.defaults.plugins.tooltip.cornerRadius = 8;
        Chart.defaults.plugins.title.color = '#e2e8f0';
        Chart.defaults.plugins.legend.labels.color = '#cbd5e1';
      }
      function initChart(id, config) {
        var ctx = document.getElementById(id);
        if (!ctx) return;
        new Chart(ctx.getContext('2d'), config);
      }
      initChart('breakdownChart', %s);
      initChart('freqChart', %s);
      initChart('timelineChart', %s);
      initChart('bugChart', %s);
      %s
      %s
    })();
  </script>
</body>
</html>`,
		escapeHTML(meta.RepoName),
		buildCSS(),
		limitedBanner,
		data.CompositeScore, deltaHTML,
		bandClass(data.CompositeScore), bandLabel(data.CompositeScore),
		clampScore(data.CompositeScore),
		strings.ToUpper(escapeHTML(meta.RepoName)), meta.WindowDays,
		formatDateShort2(meta.WindowStart), formatDateShort2(meta.WindowEnd),
		meta.AnalyzedCommits,
		narrativeHTML,
		renderEnrichment(data),
		trendSectionHTML,
		standardsHTML,
		prFlowHTML,
		meta.AnalyzedCommits, meta.WindowDays,
		formatThousands(filesTouched), data.Signals.FileChurn.EligibleFileCount,
		bugPct, data.Signals.BugRatio.ChaosCommitCount,
		avgPerDay, data.Signals.Authors.TotalAuthors,
		coverageHeight,
		renderModulesSection(data.Signals.Modules.Modules, modulesHTML),
		renderHotspotsSection(data.Signals.Hotspots.Hotspots, hotspotsHTML),
		bugWhyHTML,
		churnRows,
		coveragePanel,
		renderContributorsSection(data.Signals.Authors, data.Signals.AuthorDrift, data.Signals.Standards.ConventionalCommits),
		formatDateLong(meta.GeneratedAt),
		formatDateShort2(meta.WindowStart), formatDateShort2(meta.WindowEnd),
		breakdownJS, freqJS, timelineJS, bugJS, coverageInit, trendInit,
	)
	return sb.String()
}

// MoodEmoji returns the emoji for a given mood string.
func MoodEmoji(mood string) string {
	switch mood {
	case "calm":
		return "\U0001F60C"
	case "anxious":
		return "\U0001F62C"
	case "chaotic":
		return "\U0001F525"
	}
	return "?"
}

// bandClass is the CSS suffix for the score's band (steady/active/volatile).
// Mirrors the same 0-40 / 41-70 / 71-100 thresholds as moodFromScore.
func bandClass(score int) string {
	switch {
	case score <= 40:
		return "steady"
	case score <= 70:
		return "active"
	default:
		return "volatile"
	}
}

// bandLabel is the human-readable band name shown next to the score.
func bandLabel(score int) string {
	switch bandClass(score) {
	case "steady":
		return "Steady"
	case "active":
		return "Active"
	default:
		return "Volatile"
	}
}

// clampScore keeps the bar marker inside the track on edge cases.
func clampScore(score int) int {
	if score < 0 {
		return 0
	}
	if score > 100 {
		return 100
	}
	return score
}

func renderNarrative(bullets []types.NarrativeBullet) string {
	if len(bullets) == 0 {
		return `<p style="color:#6b7280;font-size:13px">No notable findings.</p>`
	}
	iconFor := func(k string) string {
		switch k {
		case "alert":
			return "\U0001F525"
		case "warn":
			return "\u26A0\uFE0F"
		case "good":
			return "\u2705"
		default:
			return "\u2139\uFE0F"
		}
	}
	var sb strings.Builder
	sb.WriteString("<ul>")
	for _, b := range bullets {
		fmt.Fprintf(&sb, `
    <li class="%s"><span class="bullet-icon">%s</span><span>%s</span></li>
  `, b.Kind, iconFor(b.Kind), escapeHTML(b.Text))
	}
	sb.WriteString("</ul>")
	return sb.String()
}

func renderModules(modules []types.ModuleEntry) string {
	if len(modules) == 0 {
		return `<div style="color:var(--text-dim);font-size:13px">No modules meet the minimum commit threshold.</div>`
	}
	limit := 18
	if len(modules) < limit {
		limit = len(modules)
	}
	var sb strings.Builder
	for _, m := range modules[:limit] {
		pillClass := "pill-calm"
		if m.Mood == types.MoodAnxious {
			pillClass = "pill-anxious"
		} else if m.Mood == types.MoodChaotic {
			pillClass = "pill-chaotic"
		}
		topFileText := ""
		if m.TopFile != "" {
			topFileText = fmt.Sprintf("<br>top: <code>%s</code>", escapeHTML(shorten(m.TopFile, 40)))
		}
		ownerLine := ""
		if len(m.Owners) > 0 {
			var chips strings.Builder
			for _, o := range m.Owners {
				fmt.Fprintf(&chips, `<span class="owner-chip-inline" data-owner="%s">%s</span>`, escapeHTML(o), escapeHTML(o))
			}
			ownerLine = "<br>" + chips.String()
		}
		plural := "s"
		if m.Authors == 1 {
			plural = ""
		}
		fmt.Fprintf(&sb, `<div class="module-card">
      <span class="score-pill %s">%d</span>
      <span class="name">%s</span>
      <div class="meta">
        %d commits · %s LOC · %d author%s<br>
        bug ratio %d%%%s%s
      </div>
    </div>`, pillClass, m.Score, escapeHTML(m.Name),
			m.Commits, formatThousands(m.LinesChanged), m.Authors, plural,
			int(m.BugRatio*100+0.5), topFileText, ownerLine)
	}
	return sb.String()
}

func renderHotspots(hotspots []types.HotspotEntry) string {
	var sb strings.Builder
	sb.WriteString(`<div class="row head">
    <div></div>
    <div>File</div>
    <div style="text-align:right">Commits</div>
    <div style="text-align:right">Bug-touches</div>
    <div style="text-align:right">Chaos</div>
    <div style="text-align:right">Score</div>
  </div>`)
	for _, h := range hotspots {
		color := "linear-gradient(90deg, #22d3ee, #67e8f9)"
		if h.HotspotScore >= 70 {
			color = "linear-gradient(90deg, #f43f5e, #fb7185)"
		} else if h.HotspotScore >= 40 {
			color = "linear-gradient(90deg, #fbbf24, #fde68a)"
		}
		chaosColor := "var(--text-faint)"
		if h.ChaosTouches > 0 {
			chaosColor = "#fda4af"
		}
		var owners strings.Builder
		max := 2
		if len(h.Owners) < max {
			max = len(h.Owners)
		}
		for _, o := range h.Owners[:max] {
			fmt.Fprintf(&owners, `<span class="owner-chip" data-owner="%s">%s</span>`, escapeHTML(o), escapeHTML(o))
		}
		fmt.Fprintf(&sb, `<details class="hotspot-item">
      <summary class="row">
        <div class="chevron">&#x25B6;</div>
        <div class="path" title="%s">%s%s</div>
        <div style="text-align:right;font-family:'JetBrains Mono',monospace;color:var(--text-dim)">%d</div>
        <div style="text-align:right;font-family:'JetBrains Mono',monospace;color:var(--anxious)">%s</div>
        <div style="text-align:right;font-family:'JetBrains Mono',monospace;color:%s">%d</div>
        <div>
          <div style="display:flex;gap:8px;align-items:center;justify-content:flex-end">
            <span style="width:28px;font-family:'JetBrains Mono',monospace;color:var(--text);font-size:12px;text-align:right">%d</span>
            <div class="bar-wrap" style="width:80px"><div class="bar-fill" style="width:%d%%;background:%s"></div></div>
          </div>
        </div>
      </summary>
      %s
    </details>`, escapeHTML(h.Path), escapeHTML(h.Path), owners.String(),
			h.TotalCommits, fmt1(h.BugTouches), chaosColor, h.ChaosTouches,
			h.HotspotScore, h.HotspotScore, color, renderHotspotDetail(h))
	}
	return sb.String()
}

func renderHotspotDetail(h types.HotspotEntry) string {
	var sb strings.Builder
	fmt.Fprintf(&sb, `<div class="hotspot-detail"><div class="detail-meta">
    <span>Churn rank <b>#%d</b></span>
    <span>Distinct authors <b>%d</b></span>`, h.ChurnRank, h.Authors)
	if h.LastTouched != "" {
		fmt.Fprintf(&sb, `<span>Last touched <b>%s</b></span>`, h.LastTouched)
	}
	sb.WriteString("</div>")

	if len(h.Recommendations) > 0 {
		sb.WriteString(`<div class="detail-title">What to do next</div><div class="recommendations"><ul>`)
		for _, r := range h.Recommendations {
			fmt.Fprintf(&sb, `
         <li class="%s" data-rec-kind="%s">
           <span class="rec-kind">%s</span>
           %s
         </li>`, r.Severity, r.Kind, strings.ReplaceAll(r.Kind, "-", " "), escapeHTML(r.Text))
		}
		sb.WriteString("</ul></div>")
	}

	if len(h.TopAuthorsOfFile) > 0 {
		sb.WriteString(`<div class="detail-title">Top authors of this file</div><div class="detail-authors">`)
		for _, a := range h.TopAuthorsOfFile {
			fmt.Fprintf(&sb, `<span class="author-chip">%s<span class="count">%d</span></span>`, escapeHTML(a.Name), a.Commits)
		}
		sb.WriteString(`</div>`)
	}

	if len(h.RecentBugCommits) > 0 {
		ofText := ""
		if h.BugTouches > float64(len(h.RecentBugCommits)) {
			ofText = fmt.Sprintf(" of ~%d", int(h.BugTouches+0.5))
		}
		fmt.Fprintf(&sb, `<div class="detail-title">Recent bug-tier commits (%d%s)</div><ul class="commit-list">`, len(h.RecentBugCommits), ofText)
		for _, c := range h.RecentBugCommits {
			tierClass := ""
			switch c.Tier {
			case "chaos":
				tierClass = "tier-chaos"
			case "normal":
				tierClass = "tier-normal"
			case "routine":
				tierClass = "tier-routine"
			}
			fmt.Fprintf(&sb, `<li>
           <span class="commit-tier %s">%s</span>
           <span class="commit-hash">%s</span>
           <span class="commit-date">%s · <span class="commit-author" title="%s">%s</span></span>
           <span class="commit-message" title="%s">%s</span>
         </li>`, tierClass, c.Tier,
				escapeHTML(c.Hash),
				c.Date, escapeHTML(c.Author), escapeHTML(shorten(c.Author, 14)),
				escapeHTML(c.Message), escapeHTML(c.Message))
		}
		sb.WriteString("</ul>")
	} else {
		sb.WriteString(`<div style="color:var(--text-faint);font-size:12px">No bug-tier commits captured for drill-down.</div>`)
	}

	sb.WriteString(`</div>`)
	return sb.String()
}

// renderContributorsSection is the bottom-of-report Contributors explorer.
// One drillable row per contributor in the window, sorted by lines-changed
// desc, scrollable when the list grows long. Folds in:
//   - the mini-stats header (distinct authors, weekend-night %, top share, new LOC %)
//   - a per-row "Worth a 1:1" alert/watch pill when that contributor has any baseline drift
//   - per-contributor drill-down with stats, drift detail, conventional-commit %, top files
//
// Designed so a manager can scan the list to spot patterns AND an engineer
// can find themselves and click in for their own slice — replacing the
// removed --me banner with inline self-service.
func renderContributorsSection(authors types.AuthorSignal, drift types.AuthorDriftSignal, cc types.ConventionalCommitsResult) string {
	if len(authors.Contributors) == 0 {
		return ""
	}

	// Lookup tables so each row can pull its drift / cc entry in O(1).
	driftByEmail := map[string]types.AuthorDrift{}
	for _, d := range drift.Authors {
		driftByEmail[strings.ToLower(d.Email)] = d
	}
	ccByEmail := map[string]types.AuthorComplianceEntry{}
	for _, c := range cc.PerAuthor {
		ccByEmail[strings.ToLower(c.Email)] = c
	}

	miniStats := fmt.Sprintf(`<div class="mini-stats">
        <div><div class="val">%d</div><div class="lbl">Distinct</div></div>
        <div><div class="val">%s%%</div><div class="lbl">Weekend/night</div></div>
        <div><div class="val">%s%%</div><div class="lbl">Top author share</div></div>
        <div><div class="val">%s%%</div><div class="lbl">LOC by new</div></div>
      </div>`,
		authors.TotalAuthors,
		fmt1(authors.WeekendNightPct),
		fmt1(authors.BusFactorTop1Pct),
		fmt1(authors.NewContributorChurnPct))

	var rows strings.Builder
	for _, a := range authors.Contributors {
		emailLower := strings.ToLower(a.Email)
		d, hasDrift := driftByEmail[emailLower]
		c, hasCC := ccByEmail[emailLower]

		// Pill on the summary row indicates whether this person crossed
		// any drift threshold in the current window. Severity is the
		// max across all their flags.
		alertPill := ""
		if hasDrift {
			sev := highestDriftSeverity(d.Flags)
			alertPill = fmt.Sprintf(`<span class="drift-pill drift-%s">1:1</span>`, sev)
		}
		newTag := ""
		if a.IsNew {
			newTag = `<span class="new-tag">NEW</span>`
		}
		wnPct := 0.0
		if a.Commits > 0 {
			wnPct = float64(a.WeekendNightCommits) / float64(a.Commits) * 100
		}

		fmt.Fprintf(&rows, `<details class="contrib-item">
      <summary class="row contrib-row" data-email="%s">
        <div class="chevron">&#x25B6;</div>
        <div class="contrib-name-cell">
          <span class="contrib-name">%s</span>%s
          <span class="contrib-email">%s</span>
        </div>
        <div style="text-align:right;font-family:'JetBrains Mono',monospace;color:var(--text-dim)">%d</div>
        <div style="text-align:right;font-family:'JetBrains Mono',monospace;color:var(--text-dim)">%s</div>
        <div style="text-align:right;font-family:'JetBrains Mono',monospace;color:var(--text-faint)">%.0f%%</div>
        <div style="text-align:right">%s</div>
      </summary>
      %s
    </details>`,
			escapeHTML(emailLower),
			escapeHTML(a.Name), newTag, escapeHTML(a.Email),
			a.Commits, formatThousands(a.LinesChanged), wnPct,
			alertPill,
			renderContributorDetail(a, hasDrift, d, hasCC, c, drift.BaselineDays))
	}

	return fmt.Sprintf(`<div class="card" style="margin-bottom:24px">
      <h2>Contributors <span class="sub">&middot; click any row to drill in &middot; sorted by LOC</span></h2>
      %s
      <div class="row head contrib-head">
        <div></div>
        <div>Contributor</div>
        <div style="text-align:right">Commits</div>
        <div style="text-align:right">LOC</div>
        <div style="text-align:right">Wknd/night</div>
        <div style="text-align:right">Watch?</div>
      </div>
      <div class="contributors-list">
        %s
      </div>
    </div>`, miniStats, rows.String())
}

// renderContributorDetail builds the expanded panel for one contributor.
// Lays down: stats grid → drift detail (or no-flag note) → conventional-commit
// compliance bar → top files they touched.
func renderContributorDetail(a types.AuthorEntry, hasDrift bool, d types.AuthorDrift, hasCC bool, c types.AuthorComplianceEntry, baselineDays int) string {
	var sb strings.Builder
	wnPct := 0.0
	if a.Commits > 0 {
		wnPct = float64(a.WeekendNightCommits) / float64(a.Commits) * 100
	}

	fmt.Fprintf(&sb, `<div class="hotspot-detail"><div class="detail-meta">
    <span>Commits <b>%d</b></span>
    <span>LOC <b>%s</b></span>
    <span>Weekend/night <b>%d (%.0f%%)</b></span>
    <span>First seen <b>%s</b></span>
  </div>`,
		a.Commits, formatThousands(a.LinesChanged), a.WeekendNightCommits, wnPct, a.FirstSeen)

	// Drift card, if flagged.
	if hasDrift {
		fmt.Fprintf(&sb, `<div class="detail-title">Worth a 1:1 &middot; vs their %d-day baseline</div>`, baselineDays)
		sb.WriteString(`<div class="drift-card">`)
		fmt.Fprintf(&sb, `<div class="drift-stats" style="margin-bottom:8px">
        <span title="commits/week now vs baseline">%.1f / %.1f cmt/wk</span>
        <span title="weekend or night-hour share now vs baseline">%.0f%% / %.0f%% off-hours</span>
        <span title="bug-tier share of their commits now vs baseline">%.0f%% / %.0f%% fix-mix</span>
      </div>`,
			d.CommitsPerWeekCurrent, d.CommitsPerWeekBaseline,
			d.WeekendNightCurrent, d.WeekendNightBaseline,
			d.FixRatioCurrent, d.FixRatioBaseline)
		sb.WriteString(`<ul class="drift-flags">`)
		for _, f := range d.Flags {
			fmt.Fprintf(&sb, `<li class="drift-flag drift-%s"><span class="drift-pill drift-%s">%s</span><span>%s</span></li>`,
				f.Severity, f.Severity, escapeHTML(strings.ToUpper(f.Severity)), escapeHTML(f.Text))
		}
		sb.WriteString(`</ul></div>`)
	} else if baselineDays > 0 {
		sb.WriteString(`<div class="detail-title">Baseline drift</div><div style="color:var(--text-faint);font-size:12px">No flags this window — their cadence, off-hours load, and fix-vs-feature mix all stayed within their own baseline.</div>`)
	}

	// Conventional-commit compliance bar.
	if hasCC {
		barColor := pctColor2(c.CompliancePct)
		fmt.Fprintf(&sb, `<div class="detail-title">Conventional-commit compliance</div>
      <div class="contrib-cc-row">
        <span class="std-author-bar"><span style="width:%.0f%%;background:%s"></span></span>
        <span class="std-author-pct">%.0f%% <span class="dim">(%d/%d)</span></span>
      </div>`,
			c.CompliancePct, barColor,
			c.CompliancePct, c.Compliant, c.Total)
	}

	// Top files they touched.
	if len(a.TopFiles) > 0 {
		fmt.Fprintf(&sb, `<div class="detail-title">Most-touched files (%d shown)</div><ul class="contrib-files">`, len(a.TopFiles))
		for _, f := range a.TopFiles {
			fmt.Fprintf(&sb, `<li>
          <code title="%s">%s</code>
          <span class="contrib-file-stats">%d commits · <span class="added">+%d</span> <span class="removed">-%d</span></span>
        </li>`,
				escapeHTML(f.Path), escapeHTML(shorten(f.Path, 60)),
				f.Commits, f.Added, f.Removed)
		}
		sb.WriteString(`</ul>`)
	}

	sb.WriteString(`</div>`)
	return sb.String()
}

func highestDriftSeverity(flags []types.DriftFlag) string {
	rank := map[string]int{"alert": 3, "watch": 2, "info": 1}
	bestSev := ""
	best := 0
	for _, f := range flags {
		if r := rank[f.Severity]; r > best {
			best = r
			bestSev = f.Severity
		}
	}
	if bestSev == "" {
		return "info"
	}
	return bestSev
}

// renderPRFlow emits the "PR Flow" card surfacing GitHub PR metrics.
// Renders nothing when the signal is absent (no token configured or
// fetch failed); the rest of the report stays valid without it.
func renderPRFlow(p *types.PRFlowSignal) string {
	if p == nil || p.MergedPRs == 0 {
		return ""
	}

	// Banner when we served cached-only due to rate-limit.
	banner := ""
	if p.CacheBanner != "" {
		banner = fmt.Sprintf(`<div class="pr-banner">&#x26A0;&#xFE0F; %s</div>`, escapeHTML(p.CacheBanner))
	}

	// Top reviewers bar list.
	var revRows strings.Builder
	for _, r := range p.Reviewers {
		barColor := pctColor2(100 - r.SharePct) // higher share = warmer bar (concentration = risk)
		fmt.Fprintf(&revRows, `<li class="std-author-row">
        <span class="std-author-name" title="%s">%s</span>
        <span class="std-author-bar"><span style="width:%.0f%%;background:%s"></span></span>
        <span class="std-author-pct">%d reviews <span class="dim">(%.0f%%)</span></span>
      </li>`,
			escapeHTML(r.Login), escapeHTML(r.Login),
			r.SharePct, barColor,
			r.ReviewCount, r.SharePct)
	}

	// Rubber-stamp drill-down.
	rubberPanel := ""
	if len(p.RubberStamps) > 0 {
		var lis strings.Builder
		for _, s := range p.RubberStamps {
			fmt.Fprintf(&lis, `<li>
          <span class="pr-num">#%d</span>
          <span class="pr-author">%s</span>
          <span class="pr-cycle">%.1fh</span>
          <span class="pr-title" title="%s">%s</span>
        </li>`,
				s.Number,
				escapeHTML(s.Author),
				s.CycleHours,
				escapeHTML(s.Title),
				escapeHTML(shorten(s.Title, 90)))
		}
		rubberPanel = fmt.Sprintf(`<details class="std-samples">
        <summary><span class="chevron">&#x25B6;</span> rubber-stamped PRs (%d shown)</summary>
        <ul class="pr-samples">%s</ul>
      </details>`, len(p.RubberStamps), lis.String())
	}

	return fmt.Sprintf(`<div class="card" style="margin-bottom:24px">
      <h2>PR Flow <span class="section-sub">&middot; %s &middot; %d merged PRs in %dd</span></h2>
      %s
      <div class="pr-headlines">
        <div class="pr-headline">
          <div class="pr-num-big">%.1fh</div>
          <div class="pr-label">P50 cycle time<span class="sublabel">p75 %.1fh &middot; p95 %.1fh</span></div>
        </div>
        <div class="pr-headline">
          <div class="pr-num-big">%.1fh</div>
          <div class="pr-label">P50 time-to-first-review<span class="sublabel">p75 %.1fh</span></div>
        </div>
        <div class="pr-headline">
          <div class="pr-num-big" style="color:%s">%.0f%%</div>
          <div class="pr-label">Rubber-stamp rate<span class="sublabel">approved &lt;60s, no comments</span></div>
        </div>
        <div class="pr-headline">
          <div class="pr-num-big">%.0f%%</div>
          <div class="pr-label">Self-merge rate<span class="sublabel">author merged own PR</span></div>
        </div>
      </div>
      <div class="std-section-h">Top reviewers (workload concentration)</div>
      <ul class="std-author-list">%s</ul>
      %s
    </div>`,
		escapeHTML(p.OwnerRepo), p.MergedPRs, p.WindowDays,
		banner,
		p.CycleHours.P50, p.CycleHours.P75, p.CycleHours.P95,
		p.TTFRHours.P50, p.TTFRHours.P75,
		pctColor(100-p.RubberStampRate), p.RubberStampRate,
		p.SelfMergeRate,
		revRows.String(),
		rubberPanel,
	)
}

// renderStandards is the Plank-2 deterministic-standards card. Two
// sub-sections: Conventional-commit compliance and test density.
// Both render with the same coaching framing — observation, not score.
func renderStandards(s types.StandardsSignal) string {
	if s.Type == "" {
		return ""
	}
	cc := renderConventionalCommits(s.ConventionalCommits)
	tc := renderTestDensity(s.TestDensity)
	if cc == "" && tc == "" {
		return ""
	}
	return fmt.Sprintf(`<div class="card" style="margin-bottom:24px">
      <h2>Standards <span class="section-sub">&middot; deterministic checks, no AI</span></h2>
      <div class="standards-row">%s%s</div>
    </div>`, cc, tc)
}

func renderConventionalCommits(c types.ConventionalCommitsResult) string {
	if c.Total == 0 {
		return ""
	}
	pctColor := pctColor(c.CompliancePct)

	var perAuthor strings.Builder
	shown := 0
	for _, a := range c.PerAuthor {
		if a.Total < 3 {
			continue
		}
		if shown >= 8 {
			break
		}
		shown++
		barColor := pctColor2(a.CompliancePct)
		fmt.Fprintf(&perAuthor, `<li class="std-author-row">
      <span class="std-author-name" title="%s">%s</span>
      <span class="std-author-bar"><span style="width:%.0f%%;background:%s"></span></span>
      <span class="std-author-pct">%.0f%% <span class="dim">(%d/%d)</span></span>
    </li>`,
			escapeHTML(a.Email),
			escapeHTML(shorten(a.Name, 22)),
			a.CompliancePct, barColor,
			a.CompliancePct, a.Compliant, a.Total,
		)
	}

	var samples strings.Builder
	for _, s := range c.NonCompliantSamples {
		fmt.Fprintf(&samples, `<li>
      <span class="std-hash">%s</span>
      <span class="std-sample-author">%s</span>
      <span class="std-sample-msg" title="%s">%s</span>
    </li>`,
			escapeHTML(s.Hash),
			escapeHTML(shorten(s.Author, 14)),
			escapeHTML(s.Subject),
			escapeHTML(shorten(s.Subject, 90)),
		)
	}

	// Subtitle reflects whichever pattern is in effect. Default case
	// keeps the Conventional Commits phrasing (most users); custom
	// pattern surfaces the actual regex so there's no ambiguity
	// about what's being measured.
	patternLine := fmt.Sprintf(`%d of %d commits in this window match <code>type(scope?)!?: subject</code> (Conventional Commits).`, c.Compliant, c.Total)
	if c.Pattern != "" {
		patternLine = fmt.Sprintf(`%d of %d commits match your team's pattern <code>%s</code> (configured in <code>.repopulserc</code>).`,
			c.Compliant, c.Total, escapeHTML(c.Pattern))
	}

	return fmt.Sprintf(`<div class="standards-card">
      <div class="std-head">
        <div class="std-title">Commit compliance</div>
        <div class="std-pct" style="color:%s">%.1f%%</div>
      </div>
      <div class="std-sub">%s</div>
      <div class="std-section-h">Per-author (≥3 commits, sorted lowest first)</div>
      <ul class="std-author-list">%s</ul>
      %s
    </div>`,
		pctColor, c.CompliancePct,
		patternLine,
		perAuthor.String(),
		conditionalNonCompliantPanel(samples.String()),
	)
}

func conditionalNonCompliantPanel(samples string) string {
	if samples == "" {
		return ""
	}
	return fmt.Sprintf(`<details class="std-samples">
        <summary><span class="chevron">&#x25B6;</span> non-compliant samples (newest first)</summary>
        <ul>%s</ul>
      </details>`, samples)
}

func renderTestDensity(c types.TestDensityResult) string {
	if c.SourceFiles == 0 {
		return ""
	}
	pctColor := pctColor(c.DensityPct)

	var langPills strings.Builder
	for _, l := range c.Languages {
		fmt.Fprintf(&langPills, `<span class="lang-pill">%s</span>`, escapeHTML(l))
	}

	var perModule strings.Builder
	for _, m := range c.PerModule {
		barColor := pctColor2(m.DensityPct)
		// Bar width caps at 100%% visually; the numeric label shows the true value (can exceed 100%%).
		barWidth := m.DensityPct
		if barWidth > 100 {
			barWidth = 100
		}
		fmt.Fprintf(&perModule, `<li class="std-author-row">
      <span class="std-author-name">%s</span>
      <span class="std-author-bar"><span style="width:%.0f%%;background:%s"></span></span>
      <span class="std-author-pct">%.0f%% <span class="dim">(%d/%d)</span></span>
    </li>`,
			escapeHTML(m.Module),
			barWidth, barColor,
			m.DensityPct, m.TestFiles, m.SourceFiles,
		)
	}

	return fmt.Sprintf(`<div class="standards-card">
      <div class="std-head">
        <div class="std-title">Test density %s</div>
        <div class="std-pct" style="color:%s">%.1f%%</div>
      </div>
      <div class="std-sub">%d test files vs %d source files (ratio, not coverage). Catches "does this module have tests at all"; filename matching deliberately avoided.</div>
      <div class="std-section-h">Worst-covered modules (≥5 source files, sorted lowest first)</div>
      <ul class="std-author-list">%s</ul>
    </div>`,
		langPills.String(),
		pctColor, c.DensityPct,
		c.TestFiles, c.SourceFiles,
		perModule.String(),
	)
}

func pctColor(p float64) string {
	switch {
	case p >= 80:
		return "#4ade80"
	case p >= 50:
		return "#fbbf24"
	default:
		return "#f87171"
	}
}

func pctColor2(p float64) string {
	switch {
	case p >= 80:
		return "rgba(74, 222, 128, 0.55)"
	case p >= 50:
		return "rgba(251, 191, 36, 0.55)"
	default:
		return "rgba(244, 114, 114, 0.55)"
	}
}

func renderDelta(d types.MoodDelta) string {
	cls := "delta-flat"
	if d.Composite > 0 {
		cls = "delta-up"
	} else if d.Composite < 0 {
		cls = "delta-down"
	}
	sign := ""
	if d.Composite > 0 {
		sign = "+"
	}
	prev := d.PreviousAt
	if prev == "" {
		prev = "previous snapshot"
	}
	return fmt.Sprintf(` <span class="delta-pill %s" title="vs %s">Δ %s%d</span>`, cls, prev, sign, d.Composite)
}

func renderBugExplainability(bug types.BugSignal) string {
	s := bug.ClassifiedSamples
	total := len(s.Chaos) + len(s.Normal) + len(s.Routine)
	if total == 0 {
		return `<details class="why-panel">
      <summary>
        <span class="chevron">&#x25B6;</span>
        <span class="why-title">Why this score?</span>
        <span class="why-sub">&middot; bug signal breakdown</span>
      </summary>
      <div style="color:var(--text-faint);font-size:12.5px;padding:8px 0">
        No bug-tier commits classified in this window.
      </div>
    </details>`
	}
	chaosBlock := renderTierBlock("chaos", s.Chaos, bug.ChaosCommitCount)
	normalBlock := renderTierBlock("normal", s.Normal, bug.NormalFixCount)
	routineBlock := renderTierBlock("routine", s.Routine, bug.RoutineFixCount)
	return fmt.Sprintf(`<details class="why-panel">
    <summary>
      <span class="chevron">&#x25B6;</span>
      <span class="why-title">Why this score?</span>
      <span class="why-sub">&middot; bug signal breakdown &middot; score %d/100 &middot; up to 20 samples per tier</span>
    </summary>
    <div class="why-legend">
      <strong>chaos</strong> (weight 1.0): reverts, hotfixes, regressions, p0/p1.
      <strong>normal</strong> (weight 0.4): generic "fix" / "bug" / "patch".
      <strong>routine</strong> (weight 0.1): typo / lint / format / whitespace &mdash;
      <em>a "fix" that only touches cleanup words lands here, not under normal</em>.
    </div>
    <div>%s%s%s</div>
  </details>`, bug.Score, chaosBlock, normalBlock, routineBlock)
}

func renderTierBlock(tier string, samples []types.BugClassifiedCommit, totalInTier int) string {
	if totalInTier == 0 {
		return ""
	}
	plural := ""
	if totalInTier != 1 {
		plural = "s"
	}
	showing := ""
	if len(samples) < totalInTier {
		showing = fmt.Sprintf(` <b class="why-sampling">(showing %d newest of %d)</b>`, len(samples), totalInTier)
	}
	header := fmt.Sprintf(`<div class="why-tier-header %s">
    <span>%s</span>
    <span class="count">· %d commit%s</span>%s
  </div>`, tier, tier, totalInTier, plural, showing)

	if len(samples) == 0 {
		return fmt.Sprintf(`<div class="why-tier-block">%s<div style="color:var(--text-faint);font-size:12px">No samples captured.</div></div>`, header)
	}
	var rows strings.Builder
	for _, sm := range samples {
		kw := sm.MatchedKeyword
		if kw == "" {
			kw = tier
		}
		rows.WriteString(fmt.Sprintf(`<li>
    <span class="kw %s" title="Matched keyword">%s</span>
    <span class="hash">%s</span>
    <span class="dateauth">%s · %s</span>
    <span class="msg" title="%s">%s</span>
  </li>`, tier, escapeHTML(kw), escapeHTML(sm.Hash),
			sm.Date, escapeHTML(shorten(sm.Author, 14)),
			escapeHTML(sm.Message), highlightKeyword(sm.Message, sm.MatchedKeyword, tier)))
	}
	return fmt.Sprintf(`<div class="why-tier-block">%s<ul class="why-commit-list">%s</ul></div>`, header, rows.String())
}

func highlightKeyword(message, keyword, tier string) string {
	esc := escapeHTML(message)
	if keyword == "" || strings.HasPrefix(keyword, "(") {
		return esc
	}
	// Build a case-insensitive word-boundary match manually on the escaped text.
	// We lowercase-scan for matches and splice in <mark>.
	lowMsg := strings.ToLower(esc)
	lowKw := strings.ToLower(keyword)
	idx := 0
	var sb strings.Builder
	for {
		pos := strings.Index(lowMsg[idx:], lowKw)
		if pos < 0 {
			sb.WriteString(esc[idx:])
			return sb.String()
		}
		start := idx + pos
		end := start + len(lowKw)
		// word boundaries: preceding char non-alnum and following char non-alnum
		if isWordBoundary(esc, start) && isWordBoundary(esc, end) {
			sb.WriteString(esc[idx:start])
			fmt.Fprintf(&sb, `<mark class="%s">%s</mark>`, tier, esc[start:end])
			idx = end
		} else {
			sb.WriteString(esc[idx : start+1])
			idx = start + 1
		}
	}
}

func isWordBoundary(s string, pos int) bool {
	if pos <= 0 || pos >= len(s) {
		return true
	}
	prev := s[pos-1]
	return !isWord(prev) || (pos < len(s) && !isWord(s[pos]))
}

func isWord(b byte) bool {
	return (b >= 'a' && b <= 'z') || (b >= 'A' && b <= 'Z') || (b >= '0' && b <= '9') || b == '_'
}

func renderModulesSection(modules []types.ModuleEntry, inner string) string {
	if len(modules) == 0 {
		return ""
	}
	return fmt.Sprintf(`
    <div class="card" style="margin-bottom:24px">
      <h2>Module Mood <span class="sub">· per top-level directory</span></h2>
      <div class="module-grid">%s</div>
    </div>
    `, inner)
}

func renderHotspotsSection(hotspots []types.HotspotEntry, inner string) string {
	if len(hotspots) == 0 {
		return ""
	}
	return fmt.Sprintf(`
    <div class="card" style="margin-bottom:24px">
      <h2>Hotspots <span class="sub">· churn × bug involvement</span></h2>
      <div class="hotspot-list">%s</div>
    </div>
    `, inner)
}

// renderChurnRows produces the Top Churned Files list as a `<details>`
// drillable list (Plank 3): each file is collapsed by default, expand
// to see top authors and the most-recent commits touching it.
func renderChurnRows(entries []types.ChurnEntry) string {
	var sb strings.Builder
	sb.WriteString(`<div class="row head churn-head">
    <div></div>
    <div>File</div>
    <div style="text-align:right">Added</div>
    <div style="text-align:right">Removed</div>
    <div style="text-align:right">Total</div>
    <div style="text-align:center">Ratio</div>
  </div>`)
	for _, f := range entries {
		pillClass := "ratio-low"
		if f.Ratio > 2.0 {
			pillClass = "ratio-high"
		} else if f.Ratio >= 1.0 {
			pillClass = "ratio-mid"
		}
		truncPath := f.Path
		if len(truncPath) > 60 {
			truncPath = "…" + truncPath[len(truncPath)-57:]
		}
		rewrittenTag := ""
		if f.Rewritten {
			rewrittenTag = ` <span class="rewritten-tag" title="Raw churn ratio > 5× — file was effectively rewritten (or generated)">rewritten</span>`
		}
		fmt.Fprintf(&sb, `<details class="churn-item">
      <summary class="row churn-row">
        <div class="chevron">&#x25B6;</div>
        <div class="path" title="%s" style="font-family:'JetBrains Mono',monospace">%s%s</div>
        <div style="text-align:right"><span class="added">+%d</span></div>
        <div style="text-align:right"><span class="removed">-%d</span></div>
        <div style="text-align:right;font-family:'JetBrains Mono',monospace;color:var(--text-dim)">%d</div>
        <div style="text-align:center"><span class="ratio-pill %s">%.1f×</span></div>
      </summary>
      %s
    </details>`,
			escapeHTML(f.Path), escapeHTML(truncPath), rewrittenTag,
			f.Added, f.Removed, f.Added+f.Removed, pillClass, f.Ratio,
			renderChurnDetail(f))
	}
	return sb.String()
}

// renderChurnDetail builds the expanded panel for one churn row. Mirrors
// renderHotspotDetail but without the recommendations block (those are
// hotspot-specific) and shows ALL recent commits, not just bug-tier.
func renderChurnDetail(f types.ChurnEntry) string {
	var sb strings.Builder
	fmt.Fprintf(&sb, `<div class="hotspot-detail"><div class="detail-meta">
    <span>Total churn <b>%d</b></span>
    <span>Distinct authors <b>%d</b></span>`,
		f.Added+f.Removed, len(f.TopAuthorsOfFile))
	if f.TotalCommits > 0 {
		fmt.Fprintf(&sb, `<span>Touching commits <b>%d</b></span>`, f.TotalCommits)
	}
	if f.LastTouched != "" {
		fmt.Fprintf(&sb, `<span>Last touched <b>%s</b></span>`, f.LastTouched)
	}
	sb.WriteString("</div>")

	if len(f.TopAuthorsOfFile) > 0 {
		sb.WriteString(`<div class="detail-title">Top authors of this file</div><div class="detail-authors">`)
		for _, a := range f.TopAuthorsOfFile {
			fmt.Fprintf(&sb, `<span class="author-chip">%s<span class="count">%d</span></span>`, escapeHTML(a.Name), a.Commits)
		}
		sb.WriteString(`</div>`)
	}

	if len(f.RecentCommits) > 0 {
		fmt.Fprintf(&sb, `<div class="detail-title">Recent commits touching this file (%d newest)</div><ul class="commit-list">`, len(f.RecentCommits))
		for _, c := range f.RecentCommits {
			tierClass := ""
			tierLabel := c.Tier
			switch c.Tier {
			case "chaos":
				tierClass = "tier-chaos"
			case "normal":
				tierClass = "tier-normal"
			case "routine":
				tierClass = "tier-routine"
			default:
				tierClass = "tier-none"
				tierLabel = "—"
			}
			fmt.Fprintf(&sb, `<li>
         <span class="commit-tier %s">%s</span>
         <span class="commit-hash">%s</span>
         <span class="commit-date">%s · <span class="commit-author" title="%s">%s</span></span>
         <span class="commit-message" title="%s">%s</span>
       </li>`, tierClass, tierLabel,
				escapeHTML(c.Hash),
				c.Date, escapeHTML(c.Author), escapeHTML(shorten(c.Author, 14)),
				escapeHTML(c.Message), escapeHTML(c.Message))
		}
		sb.WriteString("</ul>")
	}

	sb.WriteString(`</div>`)
	return sb.String()
}

// --- helpers ---

func escapeHTML(s string) string {
	return html.EscapeString(s)
}

func capitalize(s string) string {
	if s == "" {
		return s
	}
	return strings.ToUpper(s[:1]) + s[1:]
}

func shorten(s string, n int) string {
	if len(s) <= n {
		return s
	}
	return "…" + s[len(s)-(n-1):]
}

func formatDateLong(t any) string {
	if tt, ok := t.(interface{ Format(string) string }); ok {
		return tt.Format("January 2, 2006")
	}
	return fmt.Sprintf("%v", t)
}

func formatDateShort2(t any) string {
	if tt, ok := t.(interface{ Format(string) string }); ok {
		return tt.Format("Jan 2, 2006")
	}
	return fmt.Sprintf("%v", t)
}

func formatThousands(n int) string {
	s := fmt.Sprintf("%d", n)
	if n < 1000 && n > -1000 {
		return s
	}
	neg := false
	if n < 0 {
		neg = true
		s = s[1:]
	}
	out := make([]byte, 0, len(s)+len(s)/3)
	// Insert commas from right to left
	for i := 0; i < len(s); i++ {
		if i > 0 && (len(s)-i)%3 == 0 {
			out = append(out, ',')
		}
		out = append(out, s[i])
	}
	if neg {
		return "-" + string(out)
	}
	return string(out)
}

func fmt1(x float64) string {
	if x == float64(int(x)) {
		return fmt.Sprintf("%d", int(x))
	}
	return fmt.Sprintf("%.1f", x)
}

func max1(n int) int {
	if n < 1 {
		return 1
	}
	return n
}

// renderEnrichment is the Plank-2 Layer-B AI-read card. Rendered only
// when data.Enrichment is non-nil. The deterministic report renders
// identically without it. Visually distinguished from the deterministic
// findings so a reader can always tell what came from numbers vs. what
// came from a language model.
func renderEnrichment(data types.MoodResult) string {
	er := data.Enrichment
	if er == nil {
		return ""
	}

	// Build narrative bullets if present. Reuse the same kind→icon
	// scheme as the deterministic findings so the visual language stays
	// consistent.
	iconFor := func(k string) string {
		switch k {
		case "alert":
			return "\U0001F525"
		case "warn":
			return "⚠️"
		case "good":
			return "✅"
		default:
			return "ℹ️"
		}
	}

	var narrativeHTML string
	if len(er.Narrative) > 0 {
		var sb strings.Builder
		sb.WriteString(`<ul class="enriched-narrative">`)
		for _, b := range er.Narrative {
			fmt.Fprintf(&sb, `<li class="%s"><span class="bullet-icon">%s</span><span>%s</span></li>`,
				escapeHTML(b.Kind), iconFor(b.Kind), escapeHTML(b.Text))
		}
		sb.WriteString(`</ul>`)
		narrativeHTML = sb.String()
	}

	var standardsHTML string
	if er.Standards != nil {
		var sb strings.Builder
		sb.WriteString(`<div class="enriched-section"><div class="enriched-section-h">Standards</div>`)
		if er.Standards.Headline != "" {
			fmt.Fprintf(&sb, `<div class="enriched-headline">%s</div>`, escapeHTML(er.Standards.Headline))
		}
		if er.Standards.Summary != "" {
			fmt.Fprintf(&sb, `<div class="enriched-summary">%s</div>`, escapeHTML(er.Standards.Summary))
		}
		if len(er.Standards.Suggestions) > 0 {
			sb.WriteString(`<ul class="enriched-suggestions">`)
			for _, s := range er.Standards.Suggestions {
				fmt.Fprintf(&sb, `<li>%s</li>`, escapeHTML(s))
			}
			sb.WriteString(`</ul>`)
		}
		sb.WriteString(`</div>`)
		standardsHTML = sb.String()
	}

	var driftHTML string
	if len(er.Drift) > 0 {
		// Look up display names for the emails so the reader gets a name,
		// not a raw email string.
		nameByEmail := map[string]string{}
		for _, c := range data.Signals.Authors.Contributors {
			nameByEmail[strings.ToLower(c.Email)] = c.Name
		}
		var sb strings.Builder
		sb.WriteString(`<div class="enriched-section"><div class="enriched-section-h">Per-contributor reading <span class="enriched-sub">&middot; coaching context, not evaluation</span></div><ul class="enriched-drift">`)
		for _, d := range er.Drift {
			name := nameByEmail[strings.ToLower(d.Email)]
			if name == "" {
				name = d.Email
			}
			fmt.Fprintf(&sb, `<li><div class="enriched-drift-name">%s</div><div class="enriched-drift-reading">%s</div>`,
				escapeHTML(name), escapeHTML(d.Reading))
			if d.Suggestion != "" {
				fmt.Fprintf(&sb, `<div class="enriched-drift-suggestion">&rarr; %s</div>`, escapeHTML(d.Suggestion))
			}
			sb.WriteString(`</li>`)
		}
		sb.WriteString(`</ul></div>`)
		driftHTML = sb.String()
	}

	if narrativeHTML == "" && standardsHTML == "" && driftHTML == "" {
		return ""
	}

	model := er.Model
	if model == "" {
		model = "language model"
	}
	source := er.Source
	if source == "" {
		source = "ai"
	}

	notesHTML := ""
	if len(er.Notes) > 0 {
		var sb strings.Builder
		sb.WriteString(`<div class="enriched-section"><div class="enriched-section-h">Notes</div><ul class="enriched-notes">`)
		for _, n := range er.Notes {
			fmt.Fprintf(&sb, `<li>%s</li>`, escapeHTML(n))
		}
		sb.WriteString(`</ul></div>`)
		notesHTML = sb.String()
	}

	return fmt.Sprintf(`<div class="card enriched-card" style="margin-bottom:24px">
      <div class="enriched-banner">
        <span class="enriched-tag">AI-GENERATED</span>
        <span class="enriched-meta">%s &middot; %s</span>
      </div>
      <h2>AI read <span class="section-sub">&middot; interpretation only, deterministic numbers above are the source of truth</span></h2>
      %s
      %s
      %s
      %s
    </div>`,
		escapeHTML(source), escapeHTML(model),
		narrativeHTML, standardsHTML, driftHTML, notesHTML)
}

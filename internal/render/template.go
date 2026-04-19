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
	mc := moodConfig(data.Mood)
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
	authorsHTML := renderAuthors(data.Signals.Authors.TopAuthors)
	deltaHTML := ""
	if delta != nil {
		deltaHTML = renderDelta(*delta)
	}
	bugWhyHTML := renderBugExplainability(data.Signals.BugRatio)
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

    <!-- Mood Badge -->
    <div class="mood-badge">
      <span class="mood-emoji">%s</span>
      <div class="mood-label">%s%s</div>
      <div class="mood-score">SCORE <b>%d</b> / 100</div>
      <div class="mood-scale">0 = calm &middot; 100 = chaotic &middot; lower is better</div>
      <div class="mood-meta">%s &middot; %dD (%s &rarr; %s) &middot; %d COMMITS</div>
    </div>

    <!-- Narrative findings -->
    <div class="card narrative" style="margin-bottom:24px">
      <h2>Findings</h2>
      %s
    </div>

    <!-- Trends across snapshots -->
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

    <!-- Authors panel -->
    <div class="card" style="margin-bottom:24px">
      <h2>Authors</h2>
      <div class="mini-stats">
        <div><div class="val">%d</div><div class="lbl">Distinct</div></div>
        <div><div class="val">%s%%</div><div class="lbl">Weekend/night</div></div>
        <div><div class="val">%s%%</div><div class="lbl">Top author share</div></div>
        <div><div class="val">%s%%</div><div class="lbl">LOC by new</div></div>
      </div>
      <div>%s</div>
    </div>

    <!-- Top Churned Files -->
    <div class="card" style="margin-bottom:24px">
      <h2>Top Churned Files</h2>
      <div style="overflow-x:auto">
        <table id="churnTable">
          <thead>
            <tr>
              <th onclick="sortTable(0)">File path &#x25B4;&#x25BE;</th>
              <th onclick="sortTable(1)" style="text-align:right">Added &#x25B4;&#x25BE;</th>
              <th onclick="sortTable(2)" style="text-align:right">Removed &#x25B4;&#x25BE;</th>
              <th onclick="sortTable(3)" style="text-align:right">Total &#x25B4;&#x25BE;</th>
              <th onclick="sortTable(4)" style="text-align:center">Ratio &#x25B4;&#x25BE;</th>
            </tr>
          </thead>
          <tbody>
            %s
          </tbody>
        </table>
      </div>
    </div>

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

    var sortState = {};
    function sortTable(colIndex) {
      var table = document.getElementById('churnTable');
      var tbody = table.querySelector('tbody');
      var rows = Array.from(tbody.querySelectorAll('tr'));
      var asc = sortState[colIndex] = !sortState[colIndex];
      rows.sort(function(a, b) {
        var aVal = a.children[colIndex].textContent.trim();
        var bVal = b.children[colIndex].textContent.trim();
        var aNum = parseFloat(aVal.replace(/[^0-9.-]/g, ''));
        var bNum = parseFloat(bVal.replace(/[^0-9.-]/g, ''));
        if (!isNaN(aNum) && !isNaN(bNum)) {
          return asc ? aNum - bNum : bNum - aNum;
        }
        return asc ? aVal.localeCompare(bVal) : bVal.localeCompare(aVal);
      });
      rows.forEach(function(row) {
        tbody.appendChild(row);
      });
    }
  </script>
</body>
</html>`,
		escapeHTML(meta.RepoName),
		buildCSS(mc),
		limitedBanner,
		mc.Emoji,
		capitalize(string(data.Mood)), deltaHTML,
		data.CompositeScore,
		strings.ToUpper(escapeHTML(meta.RepoName)), meta.WindowDays,
		formatDateShort2(meta.WindowStart), formatDateShort2(meta.WindowEnd),
		meta.AnalyzedCommits,
		narrativeHTML,
		trendSectionHTML,
		meta.AnalyzedCommits, meta.WindowDays,
		formatThousands(filesTouched), data.Signals.FileChurn.EligibleFileCount,
		bugPct, data.Signals.BugRatio.ChaosCommitCount,
		avgPerDay, data.Signals.Authors.TotalAuthors,
		coverageHeight,
		renderModulesSection(data.Signals.Modules.Modules, modulesHTML),
		renderHotspotsSection(data.Signals.Hotspots.Hotspots, hotspotsHTML),
		bugWhyHTML,
		data.Signals.Authors.TotalAuthors,
		fmt1(data.Signals.Authors.WeekendNightPct),
		fmt1(data.Signals.Authors.BusFactorTop1Pct),
		fmt1(data.Signals.Authors.NewContributorChurnPct),
		authorsHTML,
		churnRows,
		coveragePanel,
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

type moodConfigT struct {
	Accent, Glow, Emoji string
}

func moodConfig(m types.MoodLabel) moodConfigT {
	switch m {
	case types.MoodCalm:
		return moodConfigT{Accent: "#22d3ee", Glow: "rgba(34, 211, 238, 0.35)", Emoji: "\U0001F60C"}
	case types.MoodAnxious:
		return moodConfigT{Accent: "#fbbf24", Glow: "rgba(251, 191, 36, 0.32)", Emoji: "\U0001F62C"}
	default:
		return moodConfigT{Accent: "#f43f5e", Glow: "rgba(244, 63, 94, 0.38)", Emoji: "\U0001F525"}
	}
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

func renderAuthors(authors []types.AuthorEntry) string {
	if len(authors) == 0 {
		return `<div style="color:var(--text-dim);font-size:13px">No author data.</div>`
	}
	var sb strings.Builder
	sb.WriteString(`<div class="author-row head">
    <div>Author</div>
    <div style="text-align:right">Commits</div>
    <div style="text-align:right">LOC</div>
    <div style="text-align:right">Wknd/night</div>
  </div>`)
	for _, a := range authors {
		newTag := ""
		if a.IsNew {
			newTag = `<span class="new-tag">NEW</span>`
		}
		wnColor := "var(--text-faint)"
		if a.WeekendNightCommits > 0 {
			wnColor = "var(--anxious)"
		}
		fmt.Fprintf(&sb, `<div class="author-row">
      <div style="overflow:hidden;text-overflow:ellipsis;white-space:nowrap">
        <span style="color:var(--text)">%s</span>%s
        <span class="author-email">%s</span>
      </div>
      <div style="text-align:right;font-family:'JetBrains Mono',monospace;color:var(--text-dim)">%d</div>
      <div style="text-align:right;font-family:'JetBrains Mono',monospace;color:var(--text-dim)">%s</div>
      <div style="text-align:right;font-family:'JetBrains Mono',monospace;color:%s">%d</div>
    </div>`, escapeHTML(a.Name), newTag, escapeHTML(a.Email),
			a.Commits, formatThousands(a.LinesChanged),
			wnColor, a.WeekendNightCommits)
	}
	return sb.String()
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

func renderChurnRows(entries []types.ChurnEntry) string {
	var sb strings.Builder
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
		fmt.Fprintf(&sb, `<tr>
      <td style="font-family:'JetBrains Mono',monospace" title="%s">%s%s</td>
      <td style="text-align:right"><span class="added">+%d</span></td>
      <td style="text-align:right"><span class="removed">-%d</span></td>
      <td style="text-align:right;font-family:'JetBrains Mono',monospace">%d</td>
      <td style="text-align:center"><span class="ratio-pill %s">%.1f×</span></td>
    </tr>
`, escapeHTML(f.Path), escapeHTML(truncPath), rewrittenTag,
			f.Added, f.Removed, f.Added+f.Removed, pillClass, f.Ratio)
	}
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

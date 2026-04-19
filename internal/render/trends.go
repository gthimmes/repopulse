package render

import (
	"fmt"
	"strings"
	"time"

	"repopulse/internal/compare"
)

// TrendChart returns the Chart.js config for the multi-series trend
// line. Each snapshot is a point on the time axis; one dataset per
// signal plus the composite. Datasets are legend-toggleable so the
// user can isolate any series. The current run is appended by the
// caller before passing snaps in.
func TrendChart(snaps []compare.ReportSnapshot) string {
	labels := make([]string, len(snaps))
	composite := make([]int, len(snaps))
	freq := make([]int, len(snaps))
	churn := make([]int, len(snaps))
	bug := make([]int, len(snaps))
	authors := make([]int, len(snaps))
	cov := make([]any, len(snaps)) // nil-able; null gaps in the line

	for i, s := range snaps {
		labels[i] = trendLabel(s.GeneratedAt)
		composite[i] = s.MoodResult.CompositeScore
		composite_breakdown := s.MoodResult.Breakdown
		freq[i] = composite_breakdown.CommitFrequency
		churn[i] = composite_breakdown.FileChurn
		bug[i] = composite_breakdown.BugRatio
		authors[i] = composite_breakdown.Authors
		if composite_breakdown.Coverage != nil {
			cov[i] = *composite_breakdown.Coverage
		} else {
			cov[i] = nil
		}
	}

	dataset := func(label, color string, data any, hidden bool) string {
		hiddenStr := "false"
		if hidden {
			hiddenStr = "true"
		}
		return fmt.Sprintf(`{
        label: '%s',
        data: %s,
        borderColor: '%s',
        backgroundColor: '%s',
        borderWidth: 2,
        tension: 0.25,
        pointRadius: 3,
        pointHoverRadius: 5,
        spanGaps: true,
        hidden: %s,
      }`, label, jsonArr(data), color, color+"22", hiddenStr)
	}

	datasets := []string{
		dataset("Composite", "#e2e8f0", composite, false),
		dataset("Commit Frequency", "#22d3ee", freq, true),
		dataset("File Churn", "#fbbf24", churn, true),
		dataset("Bug Ratio", "#f43f5e", bug, true),
		dataset("Authors", "#a78bfa", authors, true),
		dataset("Coverage", "#4ade80", cov, true),
	}

	return fmt.Sprintf(`{
    type: 'line',
    data: {
      labels: %s,
      datasets: [%s],
    },
    options: {
      responsive: true,
      maintainAspectRatio: false,
      interaction: { mode: 'index', intersect: false },
      plugins: {
        title: { display: true, text: 'Score Trend Across Snapshots', font: { size: 16, weight: 'bold' } },
        legend: { position: 'bottom', labels: { boxWidth: 12, padding: 12 } },
        tooltip: {
          callbacks: {
            label: function(ctx) { return ctx.dataset.label + ': ' + (ctx.raw == null ? 'n/a' : ctx.raw + '/100'); }
          },
        },
      },
      scales: {
        x: { grid: { display: false }, ticks: { maxRotation: 45, maxTicksLimit: 12 } },
        y: { min: 0, max: 100, title: { display: true, text: 'Score (0 = calm, 100 = chaotic)' },
          grid: {
            color: function(ctx) { var v = ctx.tick.value; return v === 40 || v === 70 ? 'rgba(0,0,0,0.15)' : 'rgba(0,0,0,0.05)'; }
          },
        },
      },
    },
  }`, jsonArr(labels), strings.Join(datasets, ","))
}

// TrendSection returns the full HTML block for the trend card,
// including the empty-state copy when there is only a single snapshot
// (the current run).
func TrendSection(snaps []compare.ReportSnapshot) string {
	if len(snaps) <= 1 {
		return `<div class="card" style="margin-bottom:24px">
      <h2>Trends</h2>
      <p style="color:var(--text-dim);font-size:13px;margin:0">
        Only one snapshot recorded so far. Run repopulse again over time &mdash;
        each run is stored under <code>.repopulse/snapshots/</code> and the
        composite + per-signal scores will plot here.
      </p>
    </div>`
	}
	return fmt.Sprintf(`<div class="card" style="margin-bottom:24px">
      <div class="chart-container" style="height:320px">
        <canvas id="trendChart"></canvas>
      </div>
      <div style="color:var(--text-dim);font-size:11px;margin-top:8px;text-align:right">
        %d snapshots &middot; click any legend item to toggle a series
      </div>
    </div>`, len(snaps))
}

func trendLabel(generatedAt string) string {
	t, err := time.Parse(time.RFC3339Nano, generatedAt)
	if err != nil {
		t, err = time.Parse(time.RFC3339, generatedAt)
		if err != nil {
			return generatedAt
		}
	}
	return t.Local().Format("Jan 2 15:04")
}

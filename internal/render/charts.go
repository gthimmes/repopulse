// Package render builds chart configs + HTML + markdown output.
package render

import (
	"encoding/json"
	"fmt"
	"sort"
	"time"

	"mood-ring/internal/types"
)

// Each "Chart" function returns a JS object literal string ready to drop
// into a <script>new Chart(ctx, <here>)</script> block. We emit raw JS rather
// than round-tripping through JSON because Chart.js callbacks need to be
// actual functions, not strings.

func CommitFrequencyChart(data types.MoodResult) string {
	buckets := data.Signals.CommitFrequency.DailyBuckets
	mean := data.Signals.CommitFrequency.Mean
	sd := data.Signals.CommitFrequency.StdDev

	labels := make([]string, len(buckets))
	values := make([]int, len(buckets))
	colors := make([]string, len(buckets))
	for i, b := range buckets {
		labels[i] = formatLabel(b.Date)
		values[i] = b.Count
		switch {
		case b.Count == 0:
			colors[i] = "rgba(148, 163, 184, 0.10)"
		case float64(b.Count) > mean+sd:
			colors[i] = "#fbbf24"
		case float64(b.Count) > mean:
			colors[i] = "#22d3ee"
		default:
			colors[i] = "rgba(148, 163, 184, 0.45)"
		}
	}

	dataVals := make([]float64, len(values))
	for i, v := range values {
		if v == 0 {
			dataVals[i] = 0.1
		} else {
			dataVals[i] = float64(v)
		}
	}

	return fmt.Sprintf(`{
    type: 'bar',
    data: {
      labels: %s,
      datasets: [{
        label: 'Commits',
        data: %s,
        backgroundColor: %s,
        borderRadius: 2,
        borderSkipped: false,
      }],
    },
    options: {
      responsive: true,
      maintainAspectRatio: false,
      plugins: {
        title: { display: true, text: 'Commit Activity', font: { size: 16, weight: 'bold' } },
        legend: { display: false },
        tooltip: {
          callbacks: {
            label: function(ctx) { var v = ctx.raw; return v <= 0.1 ? '0 commits' : v + ' commits'; }
          },
        },
      },
      scales: {
        x: { grid: { display: false }, ticks: { maxRotation: 45, maxTicksLimit: 20 } },
        y: { beginAtZero: true, title: { display: true, text: 'Commits' } },
      },
    },
  }`, jsonArr(labels), jsonArr(dataVals), jsonArr(colors))
}

func MoodTimelineChart(data types.MoodResult) string {
	pts := data.RollingTimeline
	labels := make([]string, len(pts))
	scores := make([]int, len(pts))
	for i, p := range pts {
		labels[i] = formatLabel(p.Date)
		scores[i] = p.Score
	}
	ptsJSON, _ := json.Marshal(pts)

	return fmt.Sprintf(`{
    type: 'line',
    data: {
      labels: %s,
      datasets: [{
        label: 'Rolling mood score (7d)',
        data: %s,
        borderColor: '#818cf8',
        segment: {
          borderColor: function(ctx) { var v = ctx.p1.parsed.y; return v <= 40 ? '#22d3ee' : v <= 70 ? '#fbbf24' : '#f43f5e'; }
        },
        backgroundColor: 'rgba(129, 140, 248, 0.12)',
        fill: true,
        tension: 0.25,
        pointRadius: 0,
        borderWidth: 2,
      }],
    },
    options: {
      responsive: true,
      maintainAspectRatio: false,
      plugins: {
        title: { display: true, text: 'Mood Timeline (rolling 7-day)', font: { size: 14, weight: 'bold' } },
        legend: { display: false },
        tooltip: {
          callbacks: {
            afterLabel: function(ctx) { var p = %s[ctx.dataIndex]; return p ? p.commits + ' commits · ' + Math.round((p.bugPct||0)*100) + '%% bug' : ''; }
          },
        },
      },
      scales: {
        x: { grid: { display: false }, ticks: { maxTicksLimit: 10 } },
        y: { min: 0, max: 100, title: { display: true, text: 'Score' } },
      },
    },
  }`, jsonArr(labels), jsonArr(scores), string(ptsJSON))
}

func BugSignalChart(data types.MoodResult) string {
	bug := data.Signals.BugRatio
	bugMap := map[string]int{}
	normalMap := map[string]int{}
	chaosMap := map[string]int{}
	dates := map[string]struct{}{}
	for _, b := range bug.BugCommitsByDay {
		bugMap[b.Date] = b.Count
		dates[b.Date] = struct{}{}
	}
	for _, b := range bug.NormalCommitsByDay {
		normalMap[b.Date] = b.Count
		dates[b.Date] = struct{}{}
	}
	for _, b := range bug.ChaosCommitsByDay {
		chaosMap[b.Date] = b.Count
	}
	sortedDates := make([]string, 0, len(dates))
	for d := range dates {
		sortedDates = append(sortedDates, d)
	}
	sort.Strings(sortedDates)

	labels := make([]string, len(sortedDates))
	normalData := make([]int, len(sortedDates))
	bugFixData := make([]int, len(sortedDates))
	chaosData := make([]int, len(sortedDates))
	for i, d := range sortedDates {
		labels[i] = formatLabel(d)
		normalData[i] = normalMap[d]
		bugVal := bugMap[d] - chaosMap[d]
		if bugVal < 0 {
			bugVal = 0
		}
		bugFixData[i] = bugVal
		chaosData[i] = chaosMap[d]
	}

	return fmt.Sprintf(`{
    type: 'bar',
    data: {
      labels: %s,
      datasets: [
        { label: 'Normal',  data: %s, backgroundColor: '#22d3ee', borderRadius: 2 },
        { label: 'Bug/fix', data: %s, backgroundColor: '#fbbf24', borderRadius: 2 },
        { label: 'Chaos',   data: %s, backgroundColor: '#f43f5e', borderRadius: 2 },
      ],
    },
    options: {
      responsive: true,
      maintainAspectRatio: false,
      plugins: {
        title: { display: true, text: 'Bug Signal Timeline', font: { size: 14, weight: 'bold' } },
        legend: { position: 'bottom' },
      },
      scales: {
        x: { stacked: true, grid: { display: false }, ticks: { maxTicksLimit: 10 } },
        y: { stacked: true, beginAtZero: true, title: { display: true, text: 'Commits' } },
      },
    },
  }`, jsonArr(labels), jsonArr(normalData), jsonArr(bugFixData), jsonArr(chaosData))
}

func CoverageChart(data types.MoodResult) string {
	if data.Signals.Coverage == nil {
		return "null"
	}
	pct := data.Signals.Coverage.Percentage
	color := "#f87171"
	if pct >= 80 {
		color = "#4ade80"
	} else if pct >= 60 {
		color = "#fbbf24"
	}
	return fmt.Sprintf(`{
    type: 'doughnut',
    data: {
      labels: ['Covered', 'Uncovered'],
      datasets: [{
        data: [%g, %g],
        backgroundColor: ['%s', 'rgba(148, 163, 184, 0.15)'],
        borderWidth: 0,
      }],
    },
    options: {
      responsive: true,
      maintainAspectRatio: false,
      cutout: '70%%',
      plugins: { legend: { display: false } },
    },
  }`, pct, 100-pct, color)
}

func ScoreBreakdownChart(data types.MoodResult) string {
	type row struct{ label string; score int }
	rows := []row{
		{"Commit Frequency (15%)", data.Signals.CommitFrequency.Score},
		{"File Churn (25%)", data.Signals.FileChurn.Score},
		{"Bug Ratio (30%)", data.Signals.BugRatio.Score},
		{"Authors (20%)", data.Signals.Authors.Score},
	}
	if data.Signals.Coverage != nil {
		rows = append(rows, row{"Coverage (10%)", data.Signals.Coverage.Score})
	}

	labels := make([]string, len(rows))
	scores := make([]int, len(rows))
	colors := make([]string, len(rows))
	for i, r := range rows {
		labels[i] = r.label
		scores[i] = r.score
		switch {
		case r.score <= 40:
			colors[i] = "#22d3ee"
		case r.score <= 70:
			colors[i] = "#fbbf24"
		default:
			colors[i] = "#f43f5e"
		}
	}

	return fmt.Sprintf(`{
    type: 'bar',
    data: {
      labels: %s,
      datasets: [{
        label: 'Signal Score',
        data: %s,
        backgroundColor: %s,
        borderRadius: 4,
        barThickness: 24,
      }],
    },
    options: {
      indexAxis: 'y',
      responsive: true,
      maintainAspectRatio: false,
      plugins: {
        title: { display: true, text: 'Score Breakdown by Signal', font: { size: 16, weight: 'bold' } },
        legend: { display: false },
        tooltip: {
          callbacks: {
            label: function(ctx) { var v = ctx.raw; return v + '/100 — ' + (v <= 40 ? 'Calm' : v <= 70 ? 'Anxious' : 'Chaotic'); }
          },
        },
      },
      scales: {
        x: { min: 0, max: 100, title: { display: true, text: 'Score (0 = calm, 100 = chaotic)' },
          grid: {
            color: function(ctx) { var v = ctx.tick.value; return v === 40 || v === 70 ? 'rgba(0,0,0,0.15)' : 'rgba(0,0,0,0.05)'; }
          },
        },
        y: { grid: { display: false } },
      },
    },
  }`, jsonArr(labels), jsonArr(scores), jsonArr(colors))
}

func formatLabel(dateStr string) string {
	t, err := time.Parse("2006-01-02", dateStr)
	if err != nil {
		return dateStr
	}
	return t.Format("Jan 2")
}

func jsonArr(v any) string {
	b, _ := json.Marshal(v)
	return string(b)
}

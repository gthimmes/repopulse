package render

import (
	"fmt"
	"sort"
	"strings"

	"repopulse/internal/types"
)

type MarkdownOptions struct {
	TopHotspots        int
	TopRecommendations int
	HTMLReportPath     string
}

var kindIcon = map[string]string{
	"alert": "\U0001F6A8",
	"warn":  "\u26A0\uFE0F",
	"info":  "\u2139\uFE0F",
	"good":  "\u2705",
}

// RenderMarkdown produces the shareable digest.
func RenderMarkdown(data types.MoodResult, meta types.RepoMeta, delta *types.MoodDelta, opts MarkdownOptions) string {
	topHotspots := opts.TopHotspots
	if topHotspots <= 0 {
		topHotspots = 5
	}
	topRecs := opts.TopRecommendations
	if topRecs <= 0 {
		topRecs = 3
	}

	moodLabel := capitalize(string(data.Mood))
	emoji := MoodEmoji(string(data.Mood))
	deltaText := ""
	if delta != nil {
		sign := ""
		if delta.Composite >= 0 {
			sign = "+"
		}
		deltaText = fmt.Sprintf(" (\u0394 %s%d)", sign, delta.Composite)
	}

	header := fmt.Sprintf("## Repopulse: %s \u2014 %s %s", meta.RepoName, moodLabel, emoji)
	scoreLine := fmt.Sprintf("**Score: %d/100**%s \u00B7 %d commits \u00B7 %d days \u00B7 %s \u2192 %s",
		data.CompositeScore, deltaText, meta.AnalyzedCommits, meta.WindowDays,
		meta.WindowStart.UTC().Format("2006-01-02"),
		meta.WindowEnd.UTC().Format("2006-01-02"))

	blocks := []string{
		header,
		scoreLine,
		renderFindingsMD(data.Narrative),
		renderEnrichmentMD(data),
		renderHotspotsTableMD(data.Signals.Hotspots.Hotspots, topHotspots),
		renderModulesMD(data.Signals.Modules.Modules),
		renderTopRecommendationsMD(data.Signals.Hotspots.Hotspots, topRecs),
		renderFooterMD(meta, opts.HTMLReportPath),
	}
	clean := []string{}
	for _, b := range blocks {
		t := strings.TrimRight(b, "\n")
		if t != "" {
			clean = append(clean, t)
		}
	}
	return strings.Join(clean, "\n\n") + "\n"
}

func renderFindingsMD(bullets []types.NarrativeBullet) string {
	if len(bullets) == 0 {
		return ""
	}
	var sb strings.Builder
	sb.WriteString("### Findings\n\n")
	limit := 5
	if len(bullets) < limit {
		limit = len(bullets)
	}
	for _, b := range bullets[:limit] {
		icon := kindIcon[b.Kind]
		if icon == "" {
			icon = "\u2022"
		}
		fmt.Fprintf(&sb, "- %s %s\n", icon, b.Text)
	}
	return sb.String()
}

func renderHotspotsTableMD(hotspots []types.HotspotEntry, cap int) string {
	if len(hotspots) == 0 {
		return ""
	}
	limit := cap
	if len(hotspots) < limit {
		limit = len(hotspots)
	}
	var sb strings.Builder
	sb.WriteString("### Hotspots\n\n")
	sb.WriteString("| Score | File | Team | Chaos | Bug-touches |\n")
	sb.WriteString("| ---: | --- | --- | ---: | ---: |\n")
	for _, h := range hotspots[:limit] {
		teams := "_(unowned)_"
		if len(h.Owners) > 0 {
			parts := make([]string, len(h.Owners))
			for i, o := range h.Owners {
				parts[i] = "`" + o + "`"
			}
			teams = strings.Join(parts, ", ")
		}
		fmt.Fprintf(&sb, "| %d | `%s` | %s | %d | %s |\n",
			h.HotspotScore, h.Path, teams, h.ChaosTouches, fmt1(h.BugTouches))
	}
	return sb.String()
}

func renderModulesMD(modules []types.ModuleEntry) string {
	hot := []types.ModuleEntry{}
	for _, m := range modules {
		if m.Mood != types.MoodCalm {
			hot = append(hot, m)
		}
		if len(hot) == 5 {
			break
		}
	}
	if len(hot) == 0 {
		return ""
	}
	var sb strings.Builder
	sb.WriteString("### Hottest modules\n\n")
	sb.WriteString("| Score | Module | Team | Commits | Bug ratio |\n")
	sb.WriteString("| ---: | --- | --- | ---: | ---: |\n")
	for _, m := range hot {
		teams := "_(unowned)_"
		if len(m.Owners) > 0 {
			parts := make([]string, len(m.Owners))
			for i, o := range m.Owners {
				parts[i] = "`" + o + "`"
			}
			teams = strings.Join(parts, ", ")
		}
		fmt.Fprintf(&sb, "| %d | %s | %s | %d | %d%% |\n",
			m.Score, m.Name, teams, m.Commits, int(m.BugRatio*100+0.5))
	}
	return sb.String()
}

func renderTopRecommendationsMD(hotspots []types.HotspotEntry, cap int) string {
	type withPath struct {
		rec  types.HotspotRecommendation
		path string
	}
	all := []withPath{}
	for _, h := range hotspots {
		for _, r := range h.Recommendations {
			all = append(all, withPath{rec: r, path: h.Path})
		}
	}
	if len(all) == 0 {
		return ""
	}
	order := map[string]int{"alert": 0, "warn": 1, "info": 2}
	sort.SliceStable(all, func(i, j int) bool {
		return order[all[i].rec.Severity] < order[all[j].rec.Severity]
	})
	limit := cap
	if len(all) < limit {
		limit = len(all)
	}
	var sb strings.Builder
	sb.WriteString("### Top recommendations\n\n")
	for _, r := range all[:limit] {
		icon := kindIcon["info"]
		switch r.rec.Severity {
		case "alert":
			icon = kindIcon["alert"]
		case "warn":
			icon = kindIcon["warn"]
		}
		fmt.Fprintf(&sb, "- %s **%s** \u00B7 `%s`\n  %s\n", icon, r.rec.Kind, r.path, r.rec.Text)
	}
	return sb.String()
}

// renderEnrichmentMD adds the AI-read section to the markdown digest
// when enrichment is present. Marked clearly as AI-generated so the
// digest reader knows what's interpretation vs. measurement.
func renderEnrichmentMD(data types.MoodResult) string {
	er := data.Enrichment
	if er == nil {
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

	var sb strings.Builder
	sb.WriteString(fmt.Sprintf("### AI read &mdash; _%s &middot; %s_\n\n", source, model))
	sb.WriteString("> Interpretation only; the deterministic numbers above are the source of truth.\n")

	if len(er.Narrative) > 0 {
		sb.WriteString("\n")
		for _, b := range er.Narrative {
			icon := kindIcon[b.Kind]
			if icon == "" {
				icon = "•"
			}
			fmt.Fprintf(&sb, "- %s %s\n", icon, b.Text)
		}
	}

	if er.Standards != nil {
		sb.WriteString("\n**Standards:** ")
		if er.Standards.Headline != "" {
			fmt.Fprintf(&sb, "%s\n\n", er.Standards.Headline)
		} else {
			sb.WriteString("\n\n")
		}
		if er.Standards.Summary != "" {
			fmt.Fprintf(&sb, "%s\n", er.Standards.Summary)
		}
		if len(er.Standards.Suggestions) > 0 {
			sb.WriteString("\n")
			for _, s := range er.Standards.Suggestions {
				fmt.Fprintf(&sb, "- %s\n", s)
			}
		}
	}

	if len(er.Drift) > 0 {
		// Email → name lookup so the digest shows names, not emails.
		nameByEmail := map[string]string{}
		for _, c := range data.Signals.Authors.Contributors {
			nameByEmail[strings.ToLower(c.Email)] = c.Name
		}
		sb.WriteString("\n**Per-contributor reading** (coaching context, not evaluation):\n\n")
		for _, d := range er.Drift {
			name := nameByEmail[strings.ToLower(d.Email)]
			if name == "" {
				name = d.Email
			}
			fmt.Fprintf(&sb, "- **%s** &mdash; %s", name, d.Reading)
			if d.Suggestion != "" {
				fmt.Fprintf(&sb, " _(%s)_", d.Suggestion)
			}
			sb.WriteString("\n")
		}
	}

	return sb.String()
}

func renderFooterMD(meta types.RepoMeta, htmlReportPath string) string {
	when := meta.GeneratedAt.UTC().Format("2006-01-02")
	link := ""
	if htmlReportPath != "" {
		link = fmt.Sprintf(" \u00B7 [full HTML report](%s)", htmlReportPath)
	}
	return fmt.Sprintf("---\n_Generated by repopulse on %s%s_", when, link)
}

// Package enrich implements Plank-2 Layer-B: optional AI enrichment of
// a deterministic repopulse snapshot. It is purely additive — every
// enrichment failure (no key, network down, bad response) returns
// nil + an error to the caller, who is expected to log a warning and
// continue with the deterministic report.
//
// Two modes:
//   - Run(...) calls api.anthropic.com/v1/messages directly, given an
//     API key. Suitable for CLI use with ANTHROPIC_API_KEY in env.
//   - Skill mode is just "produce an EnrichmentResult by hand and feed
//     it through --enriched"; this package doesn't need to do anything
//     special for that, but BuildPrompt() is exported so the skill can
//     reuse the same prompt body.
package enrich

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"time"

	"repopulse/internal/types"
)

// SchemaVersion is bumped when the EnrichmentResult shape changes in a
// way that older renderers cannot tolerate.
const SchemaVersion = 1

// DefaultModel is the model used when the caller doesn't override it.
// Tracks the recommended Sonnet for routine analysis tasks.
const DefaultModel = "claude-sonnet-4-6"

// APIEndpoint is the Anthropic Messages endpoint. Override only via the
// MessagesURL Options field, e.g. for tests.
const APIEndpoint = "https://api.anthropic.com/v1/messages"

// AnthropicVersion is the API version header value the SDK currently
// requires. If the API version changes, bump this.
const AnthropicVersion = "2023-06-01"

// CacheDir is where successful enrichments are cached, keyed by input
// hash, inside the analyzed repo.
const CacheDir = ".repopulse/enrichment-cache"

// Options configures a Run(). All fields are optional except APIKey.
type Options struct {
	APIKey      string
	Model       string
	MessagesURL string // override API endpoint (tests)
	HTTPClient  *http.Client
	CacheDir    string // override cache directory; "" → no cache
	Timeout     time.Duration
}

// Run produces an EnrichmentResult for the given deterministic snapshot
// by calling the Anthropic Messages API. Caching is automatic when
// opts.CacheDir is non-empty: identical input hashes are served from
// the cache rather than re-billed.
//
// Errors are returned verbatim — callers should log + continue, never
// abort the run.
func Run(ctx context.Context, snap types.MoodResult, meta types.RepoMeta, opts Options) (*types.EnrichmentResult, error) {
	if opts.APIKey == "" {
		return nil, fmt.Errorf("no Anthropic API key supplied")
	}
	if opts.Model == "" {
		opts.Model = DefaultModel
	}
	if opts.MessagesURL == "" {
		opts.MessagesURL = APIEndpoint
	}
	if opts.HTTPClient == nil {
		opts.HTTPClient = &http.Client{Timeout: 60 * time.Second}
	}
	if opts.Timeout == 0 {
		opts.Timeout = 60 * time.Second
	}

	hash := InputHash(snap, meta, opts.Model)

	if opts.CacheDir != "" {
		if cached, err := loadCache(opts.CacheDir, hash); err == nil && cached != nil {
			return cached, nil
		}
	}

	prompt := BuildPrompt(snap, meta)
	reqBody := map[string]any{
		"model":      opts.Model,
		"max_tokens": 1500,
		"system":     systemPrompt,
		"messages": []map[string]any{
			{"role": "user", "content": prompt},
		},
	}
	body, err := json.Marshal(reqBody)
	if err != nil {
		return nil, err
	}
	callCtx, cancel := context.WithTimeout(ctx, opts.Timeout)
	defer cancel()
	req, err := http.NewRequestWithContext(callCtx, http.MethodPost, opts.MessagesURL, bytes.NewReader(body))
	if err != nil {
		return nil, err
	}
	req.Header.Set("content-type", "application/json")
	req.Header.Set("x-api-key", opts.APIKey)
	req.Header.Set("anthropic-version", AnthropicVersion)

	resp, err := opts.HTTPClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		// Surface the first 400 chars of the body so the user sees what
		// went wrong (auth, rate limit, model-not-found) without dumping
		// massive payloads into stderr.
		snippet := string(respBody)
		if len(snippet) > 400 {
			snippet = snippet[:400] + "..."
		}
		return nil, fmt.Errorf("anthropic API %d: %s", resp.StatusCode, snippet)
	}

	result, err := ParseAPIResponse(respBody)
	if err != nil {
		return nil, fmt.Errorf("parse anthropic response: %w", err)
	}
	result.Source = "anthropic-api"
	result.Model = opts.Model
	result.SchemaVersion = SchemaVersion
	result.Type = "enrichment"
	result.GeneratedAt = time.Now().UTC().Format(time.RFC3339)
	result.InputHash = hash

	if opts.CacheDir != "" {
		_ = saveCache(opts.CacheDir, hash, result) // cache failures aren't fatal
	}
	return result, nil
}

// InputHash is a stable hash of the inputs that influence the
// enrichment output. Used as the cache key. Changing the model or the
// snapshot content invalidates the cache.
func InputHash(snap types.MoodResult, meta types.RepoMeta, model string) string {
	h := sha256.New()
	fmt.Fprintf(h, "model=%s\n", model)
	fmt.Fprintf(h, "schema=%d\n", SchemaVersion)
	fmt.Fprintf(h, "repo=%s\n", meta.RepoName)
	fmt.Fprintf(h, "windowDays=%d\n", meta.WindowDays)
	fmt.Fprintf(h, "windowStart=%s\n", meta.WindowStart.UTC().Format(time.RFC3339))
	fmt.Fprintf(h, "windowEnd=%s\n", meta.WindowEnd.UTC().Format(time.RFC3339))
	fmt.Fprintf(h, "score=%d\n", snap.CompositeScore)
	fmt.Fprintf(h, "mood=%s\n", snap.Mood)
	// Hash the digest text rather than the whole struct — the digest is
	// what we send to Claude, so two snapshots producing the same digest
	// will produce the same enrichment.
	digest := buildDigest(snap, meta)
	h.Write([]byte(digest))
	return hex.EncodeToString(h.Sum(nil))
}

// systemPrompt frames the model's role. Kept short — most context goes
// in the user message digest.
const systemPrompt = `You are reviewing a deterministic snapshot of a Git repository's recent activity (commits, churn, bug-tier ratios, contributor patterns, standards adherence). Your job is to add brief human-readable interpretation on top of the numbers.

Hard rules:
- This is a lens, NOT a scorecard. Do NOT rank contributors against each other. Do NOT produce performance-review-style judgments.
- Per-author "drift" interpretations should sound like coaching context for a 1:1, not feedback.
- Be concrete and short. No marketing language. Plain English.
- Respond with a single JSON object, no surrounding prose, matching the schema in the user message.`

// BuildPrompt is exported so a Claude Code skill can reuse the same
// prompt body without going through the API client.
func BuildPrompt(snap types.MoodResult, meta types.RepoMeta) string {
	digest := buildDigest(snap, meta)
	return fmt.Sprintf(`I have a deterministic repopulse snapshot. Add interpretation.

%s

Respond with a single JSON object, no other text, with this shape:

{
  "narrative": [
    {"kind": "info"|"warn"|"alert"|"good", "text": "<one sentence, ≤180 chars>"}
  ],
  "standards": {
    "headline": "<one short phrase>",
    "summary": "<two sentences max>",
    "suggestions": ["<one short suggestion>", ...]
  },
  "drift": [
    {"email": "<exact email from input>", "reading": "<coaching-context sentence>", "suggestion": "<optional short follow-up>"}
  ],
  "notes": ["<optional caveats>"]
}

Constraints:
- 3–6 narrative bullets MAX. Skip the section entirely if nothing is worth adding beyond the deterministic findings.
- standards is optional; include it only if you can say something more useful than the numbers already say.
- drift only includes contributors who appeared in the AuthorDrift block. Use their exact email. Skip anyone you can't say anything useful about.
- No code suggestions. No file-by-file analysis. You don't have the source.
- No emoji.`, digest)
}

// buildDigest summarizes the snapshot into the compact text we send to
// Claude. Aggregate metrics only — never raw code or full commit lists.
func buildDigest(snap types.MoodResult, meta types.RepoMeta) string {
	var sb strings.Builder
	fmt.Fprintf(&sb, "Repo: %s\n", meta.RepoName)
	fmt.Fprintf(&sb, "Window: %d days (%s → %s)\n",
		meta.WindowDays,
		meta.WindowStart.UTC().Format("2006-01-02"),
		meta.WindowEnd.UTC().Format("2006-01-02"))
	fmt.Fprintf(&sb, "Composite pressure score: %d/100 (band: %s)\n", snap.CompositeScore, snap.Mood)
	fmt.Fprintf(&sb, "Analyzed commits: %d\n\n", meta.AnalyzedCommits)

	fmt.Fprintf(&sb, "## Score breakdown\n")
	fmt.Fprintf(&sb, "- commit frequency: %d\n- file churn: %d\n- bug ratio: %d\n- authors: %d\n",
		snap.Breakdown.CommitFrequency, snap.Breakdown.FileChurn,
		snap.Breakdown.BugRatio, snap.Breakdown.Authors)
	if snap.Breakdown.Coverage != nil {
		fmt.Fprintf(&sb, "- coverage: %d\n", *snap.Breakdown.Coverage)
	}
	sb.WriteString("\n")

	fmt.Fprintf(&sb, "## Bug ratio\n")
	fmt.Fprintf(&sb, "%.1f%% of commits classified as bug-tier (chaos=%d, normal=%d, routine=%d, total=%d).\n",
		snap.Signals.BugRatio.Ratio*100,
		snap.Signals.BugRatio.ChaosCommitCount,
		snap.Signals.BugRatio.NormalFixCount,
		snap.Signals.BugRatio.RoutineFixCount,
		snap.Signals.BugRatio.TotalCommits)
	if snap.Signals.BugRatio.RevertedWithin7d > 0 {
		fmt.Fprintf(&sb, "%d commits were reverted within 7 days.\n", snap.Signals.BugRatio.RevertedWithin7d)
	}
	sb.WriteString("\n")

	fmt.Fprintf(&sb, "## Churn\n")
	c := snap.Signals.FileChurn
	fmt.Fprintf(&sb, "%d files touched, %d high-churn, %d total LOC, %.1f LOC/day.\n",
		c.TotalFilesTouched, c.HighChurnFileCount, c.TotalLinesChanged, c.LinesPerDay)
	sb.WriteString("\n")

	fmt.Fprintf(&sb, "## Authors\n")
	a := snap.Signals.Authors
	fmt.Fprintf(&sb, "%d total authors. Bus factor top-1: %.0f%%, top-3: %.0f%%. Weekend/night commits: %.0f%%. New-contributor churn: %.0f%%.\n",
		a.TotalAuthors, a.BusFactorTop1Pct, a.BusFactorTop3Pct, a.WeekendNightPct, a.NewContributorChurnPct)
	sb.WriteString("\n")

	if len(snap.Signals.Modules.Modules) > 0 {
		fmt.Fprintf(&sb, "## Top modules by pressure\n")
		limit := 5
		if len(snap.Signals.Modules.Modules) < limit {
			limit = len(snap.Signals.Modules.Modules)
		}
		for _, m := range snap.Signals.Modules.Modules[:limit] {
			fmt.Fprintf(&sb, "- %s: score %d (%s), %d commits, %d LOC, bug ratio %.1f%%\n",
				m.Name, m.Score, m.Mood, m.Commits, m.LinesChanged, m.BugRatio*100)
		}
		sb.WriteString("\n")
	}

	if len(snap.Signals.Hotspots.Hotspots) > 0 {
		fmt.Fprintf(&sb, "## Top hotspots\n")
		limit := 5
		if len(snap.Signals.Hotspots.Hotspots) < limit {
			limit = len(snap.Signals.Hotspots.Hotspots)
		}
		for _, h := range snap.Signals.Hotspots.Hotspots[:limit] {
			fmt.Fprintf(&sb, "- %s: hotspot %d, %d commits, %.1f bug-touches, %d chaos\n",
				h.Path, h.HotspotScore, h.TotalCommits, h.BugTouches, h.ChaosTouches)
		}
		sb.WriteString("\n")
	}

	st := snap.Signals.Standards
	fmt.Fprintf(&sb, "## Standards (deterministic)\n")
	fmt.Fprintf(&sb, "- commit compliance: %.1f%% (%d/%d)\n",
		st.ConventionalCommits.CompliancePct,
		st.ConventionalCommits.Compliant,
		st.ConventionalCommits.Total)
	fmt.Fprintf(&sb, "- test density: %.1f%% (%d test / %d source)\n",
		st.TestDensity.DensityPct, st.TestDensity.TestFiles, st.TestDensity.SourceFiles)
	if len(st.TestDensity.PerModule) > 0 {
		limit := 3
		if len(st.TestDensity.PerModule) < limit {
			limit = len(st.TestDensity.PerModule)
		}
		for _, m := range st.TestDensity.PerModule[:limit] {
			fmt.Fprintf(&sb, "  - module %s: %.1f%% (%d/%d)\n",
				m.Module, m.DensityPct, m.TestFiles, m.SourceFiles)
		}
	}
	sb.WriteString("\n")

	flagged := []types.AuthorDrift{}
	for _, d := range snap.Signals.AuthorDrift.Authors {
		if len(d.Flags) > 0 {
			flagged = append(flagged, d)
		}
	}
	if len(flagged) > 0 {
		fmt.Fprintf(&sb, "## AuthorDrift (only contributors with at least one flag)\n")
		for _, d := range flagged {
			fmt.Fprintf(&sb, "- %s <%s>: %d commits this window\n",
				d.Name, d.Email, d.CommitsCurrent)
			fmt.Fprintf(&sb, "  cadence/wk: %.1f → %.1f (%+0.0f%%); weekend-night: %.0f%% → %.0f%% (%+0.0fpp); fix-ratio: %.0f%% → %.0f%% (%+0.0fpp)\n",
				d.CommitsPerWeekBaseline, d.CommitsPerWeekCurrent, d.CommitsDeltaPct,
				d.WeekendNightBaseline, d.WeekendNightCurrent, d.WeekendNightDeltaPP,
				d.FixRatioBaseline, d.FixRatioCurrent, d.FixRatioDeltaPP)
			for _, f := range d.Flags {
				fmt.Fprintf(&sb, "  flag: [%s] %s\n", f.Severity, f.Text)
			}
		}
		sb.WriteString("\n")
	}

	if snap.Signals.PRFlow != nil {
		pf := snap.Signals.PRFlow
		fmt.Fprintf(&sb, "## PR flow (%s)\n", pf.OwnerRepo)
		fmt.Fprintf(&sb, "%d merged PRs. Cycle p50/p75/p95: %.1f/%.1f/%.1f h. TTFR p50/p75: %.1f/%.1f h.\n",
			pf.MergedPRs, pf.CycleHours.P50, pf.CycleHours.P75, pf.CycleHours.P95,
			pf.TTFRHours.P50, pf.TTFRHours.P75)
		fmt.Fprintf(&sb, "Rubber-stamp rate: %.1f%%. Self-merge rate: %.1f%%.\n",
			pf.RubberStampRate, pf.SelfMergeRate)
		sb.WriteString("\n")
	}

	if len(snap.Narrative) > 0 {
		fmt.Fprintf(&sb, "## Existing deterministic findings (avoid duplicating these word-for-word)\n")
		for _, b := range snap.Narrative {
			fmt.Fprintf(&sb, "- [%s] %s\n", b.Kind, b.Text)
		}
	}

	return sb.String()
}

// ParseAPIResponse extracts the JSON object from an Anthropic Messages
// response and unmarshals it into an EnrichmentResult. Tolerates the
// model wrapping its JSON in ```json fences (it shouldn't, but does
// occasionally).
func ParseAPIResponse(body []byte) (*types.EnrichmentResult, error) {
	var resp struct {
		Content []struct {
			Type string `json:"type"`
			Text string `json:"text"`
		} `json:"content"`
	}
	if err := json.Unmarshal(body, &resp); err != nil {
		return nil, fmt.Errorf("decode envelope: %w", err)
	}
	var text string
	for _, b := range resp.Content {
		if b.Type == "text" {
			text += b.Text
		}
	}
	if text == "" {
		return nil, fmt.Errorf("no text content in response")
	}
	return ParseModelJSON(text)
}

// ParseModelJSON pulls the first JSON object out of `text` and decodes
// it into an EnrichmentResult. Used both by the API path and the skill
// path (which just hands a string to the renderer).
func ParseModelJSON(text string) (*types.EnrichmentResult, error) {
	cleaned := stripFences(text)
	// Find the first { and the matching }. Models sometimes preface
	// their JSON with a stray sentence even when told not to.
	start := strings.Index(cleaned, "{")
	end := strings.LastIndex(cleaned, "}")
	if start < 0 || end < 0 || end < start {
		return nil, fmt.Errorf("no JSON object found in model output")
	}
	body := cleaned[start : end+1]
	var out types.EnrichmentResult
	if err := json.Unmarshal([]byte(body), &out); err != nil {
		return nil, fmt.Errorf("decode model JSON: %w", err)
	}
	if out.Type == "" {
		out.Type = "enrichment"
	}
	if out.SchemaVersion == 0 {
		out.SchemaVersion = SchemaVersion
	}
	return &out, nil
}

var fenceRe = regexp.MustCompile("(?s)```(?:json)?\\s*(.*?)```")

func stripFences(s string) string {
	if m := fenceRe.FindStringSubmatch(s); len(m) == 2 {
		return m[1]
	}
	return s
}

func loadCache(dir, hash string) (*types.EnrichmentResult, error) {
	path := filepath.Join(dir, hash+".json")
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	var out types.EnrichmentResult
	if err := json.Unmarshal(data, &out); err != nil {
		return nil, err
	}
	return &out, nil
}

func saveCache(dir, hash string, r *types.EnrichmentResult) error {
	if err := os.MkdirAll(dir, 0755); err != nil {
		return err
	}
	data, err := json.MarshalIndent(r, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(filepath.Join(dir, hash+".json"), data, 0644)
}

// LoadFromFile reads an enriched.json from disk. Used by --enriched
// and by tests.
func LoadFromFile(path string) (*types.EnrichmentResult, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	var out types.EnrichmentResult
	if err := json.Unmarshal(data, &out); err != nil {
		return nil, err
	}
	return &out, nil
}

// WriteToFile saves an EnrichmentResult to disk in pretty JSON form.
func WriteToFile(path string, r *types.EnrichmentResult) error {
	if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
		return err
	}
	data, err := json.MarshalIndent(r, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(path, data, 0644)
}

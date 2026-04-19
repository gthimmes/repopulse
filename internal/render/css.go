package render

import "fmt"

// buildCSS returns the full <style> body. Keeps all CSS in one place so
// we can tweak the dark theme without editing the template function.
// mc provides mood-specific accent/glow colors (used in CSS custom props).
func buildCSS(mc moodConfigT) string {
	return fmt.Sprintf(cssTemplate, mc.Accent, mc.Glow)
}

// %s1 = mood accent, %s2 = mood glow
const cssTemplate = `    :root {
      --bg-0: #05070d;
      --bg-1: #0a0f1c;
      --bg-2: #111827;
      --panel: rgba(17, 24, 39, 0.55);
      --panel-hi: rgba(30, 41, 59, 0.55);
      --border: rgba(148, 163, 184, 0.12);
      --border-hi: rgba(148, 163, 184, 0.22);
      --text: #e2e8f0;
      --text-dim: #94a3b8;
      --text-faint: #64748b;
      --accent: #22d3ee;
      --accent-2: #818cf8;
      --accent-3: #f472b6;
      --mood-accent: %s;
      --mood-glow: %s;
      --calm: #22d3ee;
      --anxious: #fbbf24;
      --chaotic: #f43f5e;
      --good: #4ade80;
    }
    * { box-sizing: border-box; margin: 0; padding: 0; }
    html, body { min-height: 100%%; }
    body {
      font-family: 'Inter', system-ui, -apple-system, sans-serif;
      color: var(--text);
      line-height: 1.5;
      background:
        radial-gradient(1200px 800px at 15%% -10%%, rgba(129, 140, 248, 0.18), transparent 60%%),
        radial-gradient(900px 700px at 95%% 10%%, rgba(34, 211, 238, 0.14), transparent 60%%),
        radial-gradient(800px 900px at 50%% 120%%, rgba(244, 114, 182, 0.10), transparent 60%%),
        linear-gradient(180deg, #05070d 0%%, #090d18 100%%);
      background-attachment: fixed;
      letter-spacing: 0.005em;
      -webkit-font-smoothing: antialiased;
    }
    body::before {
      content: '';
      position: fixed;
      inset: 0;
      pointer-events: none;
      background-image:
        linear-gradient(rgba(148, 163, 184, 0.04) 1px, transparent 1px),
        linear-gradient(90deg, rgba(148, 163, 184, 0.04) 1px, transparent 1px);
      background-size: 48px 48px;
      mask-image: radial-gradient(1200px 800px at 50%% 0%%, black, transparent 70%%);
      z-index: 0;
    }
    .container { max-width: 1180px; margin: 0 auto; padding: 40px 20px 80px; position: relative; z-index: 1; }

    /* Glass card primitive */
    .card {
      position: relative;
      background: var(--panel);
      border: 1px solid var(--border);
      border-radius: 16px;
      padding: 24px;
      backdrop-filter: blur(20px) saturate(140%%);
      -webkit-backdrop-filter: blur(20px) saturate(140%%);
      box-shadow:
        0 1px 0 0 rgba(255, 255, 255, 0.04) inset,
        0 0 0 1px rgba(255, 255, 255, 0.01) inset,
        0 20px 40px -20px rgba(0, 0, 0, 0.6);
    }
    .card::before {
      content: '';
      position: absolute;
      inset: 0;
      border-radius: 16px;
      padding: 1px;
      background: linear-gradient(135deg, rgba(255,255,255,0.14), rgba(255,255,255,0) 40%%);
      -webkit-mask: linear-gradient(#000 0 0) content-box, linear-gradient(#000 0 0);
      -webkit-mask-composite: xor;
              mask-composite: exclude;
      pointer-events: none;
      opacity: 0.6;
    }

    /* Mood badge */
    .mood-badge {
      position: relative;
      text-align: center;
      padding: 44px 32px;
      border-radius: 20px;
      margin-bottom: 28px;
      background:
        radial-gradient(800px 200px at 50%% -40%%, var(--mood-glow), transparent 60%%),
        linear-gradient(180deg, rgba(30, 41, 59, 0.45) 0%%, rgba(15, 23, 42, 0.75) 100%%);
      border: 1px solid var(--border-hi);
      backdrop-filter: blur(20px) saturate(140%%);
      -webkit-backdrop-filter: blur(20px) saturate(140%%);
      box-shadow:
        0 0 0 1px rgba(255,255,255,0.04) inset,
        0 40px 80px -30px var(--mood-glow),
        0 20px 40px -20px rgba(0, 0, 0, 0.8);
      overflow: hidden;
    }
    .mood-badge::before {
      content: '';
      position: absolute;
      inset: 0;
      border-radius: 20px;
      padding: 1px;
      background: linear-gradient(135deg, var(--mood-accent) 0%%, rgba(255,255,255,0) 40%%, var(--mood-accent) 100%%);
      -webkit-mask: linear-gradient(#000 0 0) content-box, linear-gradient(#000 0 0);
      -webkit-mask-composite: xor;
              mask-composite: exclude;
      pointer-events: none;
      opacity: 0.45;
    }
    .mood-badge::after {
      content: '';
      position: absolute;
      inset: 0;
      background-image:
        linear-gradient(rgba(148, 163, 184, 0.06) 1px, transparent 1px),
        linear-gradient(90deg, rgba(148, 163, 184, 0.06) 1px, transparent 1px);
      background-size: 24px 24px;
      mask-image: radial-gradient(600px 200px at 50%% 50%%, black, transparent 70%%);
      pointer-events: none;
    }
    .mood-emoji { font-size: 88px; display: block; margin-bottom: 4px; filter: drop-shadow(0 0 30px var(--mood-glow)); position: relative; z-index: 1; }
    .mood-label { font-size: 30px; font-weight: 800; letter-spacing: 0.12em; text-transform: uppercase; color: var(--text); background: linear-gradient(180deg, #fff 0%%, var(--mood-accent) 120%%); -webkit-background-clip: text; background-clip: text; -webkit-text-fill-color: transparent; position: relative; z-index: 1; }
    .mood-score { font-size: 13px; color: var(--text-dim); margin-top: 10px; font-family: 'JetBrains Mono', ui-monospace, 'Menlo', monospace; letter-spacing: 0.1em; text-transform: uppercase; position: relative; z-index: 1; }
    .mood-score b { color: var(--mood-accent); font-weight: 700; }
    .mood-scale { font-size: 10.5px; color: var(--text-faint); margin-top: 4px; position: relative; z-index: 1; font-family: 'JetBrains Mono', ui-monospace, monospace; letter-spacing: 0.08em; text-transform: uppercase; opacity: 0.75; }
    .mood-meta { font-size: 12px; color: var(--text-faint); margin-top: 8px; position: relative; z-index: 1; font-family: 'JetBrains Mono', ui-monospace, monospace; letter-spacing: 0.06em; }

    .delta-pill { display: inline-block; margin-left: 10px; padding: 3px 10px; border-radius: 12px; font-size: 11px; font-weight: 700; font-family: 'JetBrains Mono', ui-monospace, monospace; letter-spacing: 0.08em; vertical-align: middle; border: 1px solid; }
    .delta-up   { background: rgba(244, 63, 94, 0.12);  color: #fda4af; border-color: rgba(244, 63, 94, 0.35); }
    .delta-down { background: rgba(74, 222, 128, 0.12); color: #86efac; border-color: rgba(74, 222, 128, 0.35); }
    .delta-flat { background: rgba(148, 163, 184, 0.10); color: var(--text-dim); border-color: var(--border-hi); }

    h2 { font-size: 11px; font-weight: 700; letter-spacing: 0.22em; text-transform: uppercase; margin: 0 0 18px; color: var(--text-dim); display: flex; align-items: center; gap: 12px; }
    h2::after { content: ''; flex: 1; height: 1px; background: linear-gradient(90deg, var(--border-hi), transparent); }
    h2 .sub { font-size: 11px; font-weight: 500; text-transform: none; letter-spacing: 0.04em; color: var(--text-faint); }

    .stats-row { display: grid; grid-template-columns: repeat(4, 1fr); gap: 14px; margin-bottom: 24px; }
    .stat-card { position: relative; background: var(--panel); border: 1px solid var(--border); border-radius: 14px; padding: 18px 20px; text-align: left; backdrop-filter: blur(16px) saturate(140%%); -webkit-backdrop-filter: blur(16px) saturate(140%%); overflow: hidden; }
    .stat-card::after { content: ''; position: absolute; left: 0; top: 0; bottom: 0; width: 3px; background: linear-gradient(180deg, var(--accent), var(--accent-2)); opacity: 0.6; }
    .stat-value { font-size: 30px; font-weight: 700; color: var(--text); font-family: 'JetBrains Mono', ui-monospace, monospace; letter-spacing: -0.02em; }
    .stat-label { font-size: 10px; color: var(--text-dim); margin-top: 6px; text-transform: uppercase; letter-spacing: 0.14em; font-weight: 600; }
    .stat-label .sublabel { display: block; margin-top: 2px; font-size: 10px; color: var(--text-faint); letter-spacing: 0.08em; text-transform: none; font-weight: 400; font-family: 'JetBrains Mono', ui-monospace, monospace; }

    .chart-container { position: relative; height: 300px; margin-bottom: 0; }
    .side-by-side { display: grid; grid-template-columns: 1fr 1fr; gap: 16px; margin-bottom: 24px; }
    .side-by-side .card { overflow: hidden; }
    .side-by-side .chart-container { height: 260px; }

    table { width: 100%%; border-collapse: collapse; }
    th { padding: 10px 12px; text-align: left; font-size: 10px; text-transform: uppercase; color: var(--text-dim); border-bottom: 1px solid var(--border-hi); cursor: pointer; user-select: none; letter-spacing: 0.14em; font-weight: 600; }
    th:hover { color: var(--accent); }
    td { border-bottom: 1px solid var(--border); padding: 10px 12px; font-size: 13px; color: var(--text); }
    td code { font-family: 'JetBrains Mono', ui-monospace, monospace; color: var(--text); }

    .narrative ul { list-style: none; padding: 0; }
    .narrative li { padding: 12px 16px; border-radius: 12px; margin-bottom: 10px; font-size: 13.5px; display: flex; gap: 12px; align-items: flex-start; border: 1px solid var(--border); background: var(--panel-hi); position: relative; overflow: hidden; color: var(--text); }
    .narrative li::before { content: ''; position: absolute; left: 0; top: 0; bottom: 0; width: 3px; }
    .narrative li.alert::before { background: linear-gradient(180deg, var(--chaotic), #ef4444); box-shadow: 0 0 12px var(--chaotic); }
    .narrative li.warn::before  { background: linear-gradient(180deg, var(--anxious), #f59e0b); box-shadow: 0 0 12px var(--anxious); }
    .narrative li.info::before  { background: linear-gradient(180deg, var(--accent), var(--accent-2)); box-shadow: 0 0 12px var(--accent); }
    .narrative li.good::before  { background: linear-gradient(180deg, var(--good), #22c55e); box-shadow: 0 0 12px var(--good); }
    .narrative li.alert { color: #fecdd3; }
    .narrative li.warn  { color: #fde68a; }
    .narrative li.info  { color: #a5f3fc; }
    .narrative li.good  { color: #bbf7d0; }
    .narrative .bullet-icon { font-size: 16px; line-height: 20px; flex-shrink: 0; }

    .module-grid { display: grid; grid-template-columns: repeat(auto-fit, minmax(220px, 1fr)); gap: 12px; }
    .module-card { position: relative; border: 1px solid var(--border); border-radius: 12px; padding: 14px 16px; background: var(--panel-hi); overflow: hidden; transition: transform 120ms ease, border-color 120ms ease; }
    .module-card:hover { transform: translateY(-1px); border-color: var(--border-hi); }
    .module-card .name { font-family: 'JetBrains Mono', ui-monospace, monospace; font-weight: 600; font-size: 13px; color: var(--text); letter-spacing: 0.02em; }
    .module-card .score-pill { float: right; padding: 3px 10px; border-radius: 999px; font-size: 11px; font-weight: 700; font-family: 'JetBrains Mono', ui-monospace, monospace; letter-spacing: 0.06em; border: 1px solid; }
    .module-card .meta { font-size: 11px; color: var(--text-dim); margin-top: 10px; line-height: 1.5; font-family: 'JetBrains Mono', ui-monospace, monospace; }
    .module-card .meta code { color: var(--text); }
    .pill-calm    { background: rgba(34, 211, 238, 0.12); color: #67e8f9; border-color: rgba(34, 211, 238, 0.35); }
    .pill-anxious { background: rgba(251, 191, 36, 0.12); color: #fde68a; border-color: rgba(251, 191, 36, 0.35); }
    .pill-chaotic { background: rgba(244, 63, 94, 0.12);  color: #fda4af; border-color: rgba(244, 63, 94, 0.45); box-shadow: 0 0 16px rgba(244, 63, 94, 0.18); }

    .hotspot-list .row { display: grid; grid-template-columns: 14px minmax(0,1fr) 70px 80px 60px 130px; gap: 12px; padding: 10px 12px; border-bottom: 1px solid var(--border); font-size: 13px; align-items: center; border-radius: 8px; }
    .hotspot-list .row:hover { background: rgba(148, 163, 184, 0.05); }
    .hotspot-list .row.head { font-size: 10px; text-transform: uppercase; letter-spacing: 0.14em; color: var(--text-dim); border-bottom: 1px solid var(--border-hi); padding: 8px 12px; }
    .hotspot-list .row.head:hover { background: none; }
    .hotspot-list .path { font-family: 'JetBrains Mono', ui-monospace, monospace; overflow: hidden; text-overflow: ellipsis; white-space: nowrap; color: var(--text); font-size: 12.5px; }

    details.hotspot-item { border-bottom: 1px solid var(--border); }
    details.hotspot-item:last-of-type { border-bottom: none; }
    details.hotspot-item > summary { list-style: none; cursor: pointer; user-select: none; }
    details.hotspot-item > summary::-webkit-details-marker { display: none; }
    details.hotspot-item > summary::marker { content: ''; }
    details.hotspot-item > summary .chevron { display: inline-block; color: var(--text-faint); transition: transform 150ms ease, color 150ms ease; font-size: 10px; text-align: center; }
    details.hotspot-item[open] > summary .chevron { transform: rotate(90deg); color: var(--accent); }
    details.hotspot-item[open] > summary { background: rgba(148, 163, 184, 0.04); }

    .hotspot-detail { padding: 16px 20px 20px 38px; background: rgba(5, 7, 13, 0.35); border-top: 1px solid var(--border); animation: hotspot-expand 160ms ease-out; }
    @keyframes hotspot-expand { from { opacity: 0; transform: translateY(-4px); } to { opacity: 1; transform: translateY(0); } }
    .hotspot-detail .detail-title { font-size: 10px; letter-spacing: 0.18em; text-transform: uppercase; color: var(--text-dim); margin: 0 0 8px; font-weight: 600; }
    .hotspot-detail .detail-meta { font-family: 'JetBrains Mono', ui-monospace, monospace; font-size: 11.5px; color: var(--text-dim); margin-bottom: 14px; display: flex; flex-wrap: wrap; gap: 18px; }
    .hotspot-detail .detail-meta b { color: var(--text); font-weight: 600; }
    .hotspot-detail .detail-authors { display: flex; flex-wrap: wrap; gap: 6px; margin-bottom: 16px; }
    .hotspot-detail .author-chip { background: rgba(129, 140, 248, 0.10); border: 1px solid rgba(129, 140, 248, 0.25); color: #c7d2fe; padding: 3px 10px; border-radius: 999px; font-size: 11px; font-family: 'JetBrains Mono', ui-monospace, monospace; }
    .hotspot-detail .author-chip .count { color: var(--text-faint); margin-left: 6px; }
    .hotspot-detail ul.commit-list { list-style: none; padding: 0; margin: 0; }
    .hotspot-detail ul.commit-list li { display: grid; grid-template-columns: 60px 68px 150px 1fr; gap: 12px; padding: 6px 0; border-bottom: 1px dashed var(--border); font-size: 12px; align-items: center; }
    .hotspot-detail ul.commit-list li:last-child { border-bottom: none; }
    .hotspot-detail .commit-tier { font-family: 'JetBrains Mono', ui-monospace, monospace; font-size: 9px; font-weight: 700; padding: 2px 7px; border-radius: 999px; text-align: center; letter-spacing: 0.14em; text-transform: uppercase; border: 1px solid; }
    .tier-chaos   { background: rgba(244, 63, 94, 0.14);  color: #fda4af; border-color: rgba(244, 63, 94, 0.40); }
    .tier-normal  { background: rgba(251, 191, 36, 0.12); color: #fde68a; border-color: rgba(251, 191, 36, 0.35); }
    .tier-routine { background: rgba(34, 211, 238, 0.10); color: #67e8f9; border-color: rgba(34, 211, 238, 0.30); }
    .commit-hash { font-family: 'JetBrains Mono', ui-monospace, monospace; color: var(--text-faint); font-size: 11px; }
    .commit-author { color: var(--text-dim); font-size: 11px; overflow: hidden; text-overflow: ellipsis; white-space: nowrap; }
    .commit-date { font-family: 'JetBrains Mono', ui-monospace, monospace; color: var(--text-faint); font-size: 11px; }
    .commit-message { color: var(--text); overflow: hidden; text-overflow: ellipsis; white-space: nowrap; }

    .hotspot-detail .recommendations { margin-bottom: 18px; }
    .hotspot-detail .recommendations ul { list-style: none; padding: 0; margin: 0; }
    .hotspot-detail .recommendations li { position: relative; padding: 10px 14px 10px 16px; margin-bottom: 8px; border-radius: 10px; border: 1px solid var(--border); background: var(--panel-hi); font-size: 12.5px; line-height: 1.45; overflow: hidden; }
    .hotspot-detail .recommendations li::before { content: ''; position: absolute; left: 0; top: 0; bottom: 0; width: 3px; }
    .hotspot-detail .recommendations li.alert::before { background: linear-gradient(180deg, var(--chaotic), #ef4444); box-shadow: 0 0 10px var(--chaotic); }
    .hotspot-detail .recommendations li.warn::before  { background: linear-gradient(180deg, var(--anxious), #f59e0b); box-shadow: 0 0 10px var(--anxious); }
    .hotspot-detail .recommendations li.info::before  { background: linear-gradient(180deg, var(--accent), var(--accent-2)); box-shadow: 0 0 8px var(--accent); }
    .hotspot-detail .recommendations li.alert { color: #fecdd3; }
    .hotspot-detail .recommendations li.warn  { color: #fde68a; }
    .hotspot-detail .recommendations li.info  { color: #bae6fd; }
    .hotspot-detail .rec-kind { display: inline-block; font-family: 'JetBrains Mono', ui-monospace, monospace; font-size: 9px; letter-spacing: 0.16em; text-transform: uppercase; margin-right: 8px; padding: 1px 7px; border-radius: 4px; background: rgba(255,255,255,0.06); border: 1px solid rgba(255,255,255,0.08); vertical-align: middle; white-space: nowrap; }

    .author-row { display: grid; grid-template-columns: minmax(0, 1fr) 70px 90px 100px; gap: 12px; padding: 8px 12px; align-items: center; font-size: 13px; border-bottom: 1px solid var(--border); border-radius: 8px; }
    .author-row:hover { background: rgba(148, 163, 184, 0.05); }
    .author-row.head { font-size: 10px; text-transform: uppercase; letter-spacing: 0.14em; color: var(--text-dim); border-bottom: 1px solid var(--border-hi); padding: 8px 12px; }
    .author-row.head:hover { background: none; }
    .new-tag { background: rgba(129, 140, 248, 0.15); color: #c7d2fe; font-size: 9px; padding: 2px 7px; border-radius: 999px; margin-left: 8px; font-weight: 700; letter-spacing: 0.12em; border: 1px solid rgba(129, 140, 248, 0.35); vertical-align: middle; }
    .author-email { color: var(--text-faint); font-size: 11px; margin-left: 6px; font-family: 'JetBrains Mono', ui-monospace, monospace; }

    .mini-stats { display: grid; grid-template-columns: repeat(auto-fit, minmax(170px, 1fr)); gap: 14px; margin-bottom: 18px; padding-bottom: 18px; border-bottom: 1px solid var(--border); }
    .mini-stats .val { font-size: 22px; font-weight: 700; color: var(--text); font-family: 'JetBrains Mono', ui-monospace, monospace; }
    .mini-stats .lbl { font-size: 10px; color: var(--text-dim); margin-top: 4px; text-transform: uppercase; letter-spacing: 0.14em; font-weight: 600; }

    .rewritten-tag { background: rgba(129, 140, 248, 0.15); color: #c7d2fe; padding: 2px 8px; border-radius: 999px; font-size: 9px; font-weight: 700; letter-spacing: 0.14em; margin-left: 6px; border: 1px solid rgba(129, 140, 248, 0.3); text-transform: uppercase; }

    .owner-chip { display: inline-block; background: rgba(129, 140, 248, 0.12); color: #c7d2fe; border: 1px solid rgba(129, 140, 248, 0.30); padding: 1px 8px; border-radius: 999px; font-family: 'JetBrains Mono', ui-monospace, monospace; font-size: 10px; letter-spacing: 0.06em; margin-left: 6px; vertical-align: middle; white-space: nowrap; }
    .owner-chip-inline { display: inline-block; background: rgba(129, 140, 248, 0.10); color: #a5b4fc; border: 1px solid rgba(129, 140, 248, 0.25); padding: 1px 6px; border-radius: 4px; font-family: 'JetBrains Mono', ui-monospace, monospace; font-size: 10px; margin-right: 4px; }

    details.why-panel > summary { list-style: none; cursor: pointer; user-select: none; display: flex; align-items: center; gap: 12px; padding: 2px 0 16px; }
    details.why-panel > summary::-webkit-details-marker { display: none; }
    details.why-panel > summary::marker { content: ''; }
    details.why-panel > summary .chevron { display: inline-block; color: var(--text-faint); transition: transform 150ms ease, color 150ms ease; font-size: 10px; }
    details.why-panel[open] > summary .chevron { transform: rotate(90deg); color: var(--accent); }
    details.why-panel > summary .why-title { font-size: 11px; font-weight: 700; letter-spacing: 0.22em; text-transform: uppercase; color: var(--text-dim); }
    details.why-panel > summary .why-sub { font-size: 11px; color: var(--text-faint); font-family: 'JetBrains Mono', ui-monospace, monospace; letter-spacing: 0.04em; }
    .why-legend { font-size: 12px; color: var(--text-dim); line-height: 1.55; padding: 10px 12px; margin: 4px 0 16px; border-left: 2px solid var(--border-hi); background: rgba(148, 163, 184, 0.04); border-radius: 4px; }
    .why-legend strong { color: var(--text); font-family: 'JetBrains Mono', ui-monospace, monospace; font-size: 11px; letter-spacing: 0.08em; text-transform: uppercase; }
    .why-legend em { color: var(--text-faint); font-style: normal; }
    .why-sampling { font-family: 'JetBrains Mono', ui-monospace, monospace; font-size: 10px; color: var(--text-dim); letter-spacing: 0.06em; text-transform: none; font-weight: 600; }
    .why-tier-block { margin-bottom: 16px; border-top: 1px solid var(--border); padding-top: 14px; }
    .why-tier-block:last-child { margin-bottom: 0; }
    .why-tier-header { display: flex; align-items: center; gap: 10px; margin-bottom: 10px; font-family: 'JetBrains Mono', ui-monospace, monospace; font-size: 11px; letter-spacing: 0.12em; text-transform: uppercase; }
    .why-tier-header .count { color: var(--text-faint); font-weight: 400; }
    .why-tier-header.chaos   { color: #fda4af; }
    .why-tier-header.normal  { color: #fde68a; }
    .why-tier-header.routine { color: #67e8f9; }
    ul.why-commit-list { list-style: none; padding: 0; margin: 0; }
    ul.why-commit-list li { display: grid; grid-template-columns: 68px 90px 130px 1fr; gap: 12px; padding: 5px 0; border-bottom: 1px dashed var(--border); font-size: 12px; align-items: center; }
    ul.why-commit-list li:last-child { border-bottom: none; }
    ul.why-commit-list .kw { font-family: 'JetBrains Mono', ui-monospace, monospace; font-size: 9.5px; font-weight: 700; padding: 2px 8px; border-radius: 999px; text-align: center; letter-spacing: 0.1em; text-transform: uppercase; border: 1px solid; white-space: nowrap; overflow: hidden; text-overflow: ellipsis; }
    ul.why-commit-list .kw.chaos   { background: rgba(244, 63, 94, 0.14);  color: #fda4af; border-color: rgba(244, 63, 94, 0.40); }
    ul.why-commit-list .kw.normal  { background: rgba(251, 191, 36, 0.12); color: #fde68a; border-color: rgba(251, 191, 36, 0.35); }
    ul.why-commit-list .kw.routine { background: rgba(34, 211, 238, 0.10); color: #67e8f9; border-color: rgba(34, 211, 238, 0.30); }
    ul.why-commit-list .hash { font-family: 'JetBrains Mono', ui-monospace, monospace; color: var(--text-faint); font-size: 11px; }
    ul.why-commit-list .dateauth { font-family: 'JetBrains Mono', ui-monospace, monospace; color: var(--text-faint); font-size: 11px; overflow: hidden; text-overflow: ellipsis; white-space: nowrap; }
    ul.why-commit-list .msg { color: var(--text); overflow: hidden; text-overflow: ellipsis; white-space: nowrap; }
    ul.why-commit-list .msg mark { background: rgba(251, 191, 36, 0.22); color: #fef3c7; padding: 1px 4px; border-radius: 3px; font-weight: 600; }
    ul.why-commit-list .msg mark.chaos   { background: rgba(244, 63, 94, 0.22);  color: #fecdd3; }
    ul.why-commit-list .msg mark.routine { background: rgba(34, 211, 238, 0.18); color: #cffafe; }

    .ratio-pill { padding: 3px 10px; border-radius: 999px; font-size: 11px; font-weight: 700; font-family: 'JetBrains Mono', ui-monospace, monospace; letter-spacing: 0.04em; border: 1px solid; }
    .ratio-low  { background: rgba(34, 211, 238, 0.10); color: #67e8f9; border-color: rgba(34, 211, 238, 0.30); }
    .ratio-mid  { background: rgba(251, 191, 36, 0.10); color: #fde68a; border-color: rgba(251, 191, 36, 0.30); }
    .ratio-high { background: rgba(244, 63, 94, 0.12);  color: #fda4af; border-color: rgba(244, 63, 94, 0.40); }
    .added   { color: #4ade80; font-family: 'JetBrains Mono', ui-monospace, monospace; }
    .removed { color: #f87171; font-family: 'JetBrains Mono', ui-monospace, monospace; }

    .limited-banner { background: rgba(251, 191, 36, 0.08); border: 1px solid rgba(251, 191, 36, 0.35); padding: 12px 16px; border-radius: 12px; margin-bottom: 24px; text-align: center; color: #fde68a; font-size: 13px; }
    .footer { text-align: center; font-size: 11px; color: var(--text-faint); margin-top: 40px; padding-top: 20px; border-top: 1px solid var(--border); font-family: 'JetBrains Mono', ui-monospace, monospace; letter-spacing: 0.08em; }
    .coverage-ring { width: 130px; height: 130px; position: relative; }
    .coverage-ring .pct { position: absolute; top: 50%%; left: 50%%; transform: translate(-50%%, -50%%); font-size: 24px; font-weight: 700; font-family: 'JetBrains Mono', ui-monospace, monospace; }

    @media (max-width: 900px) {
      .stats-row { grid-template-columns: repeat(2, 1fr); }
      .side-by-side { grid-template-columns: 1fr; }
      .mood-label { font-size: 24px; }
      .mood-emoji { font-size: 68px; }
    }
`

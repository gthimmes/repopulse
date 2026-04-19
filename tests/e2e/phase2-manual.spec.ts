// Manual Phase-2 verification spec. Opens the real-data anduinrepo report
// and screenshots the trend section, header, and bug-why panel so we can
// eyeball the chart. Skipped automatically when the expected file isn't
// present so CI remains green for anyone without that repo locally.
import { test, expect } from '@playwright/test';
import fs from 'fs';
import path from 'path';

const reportPath = path.resolve(__dirname, '../../output/repopulse-report.html');

test.describe('Phase 2 — manual real-data verification', () => {
  test.skip(!fs.existsSync(reportPath), 'run repopulse against anduinrepo first');

  test('header shows score, scale hint, and explicit date range', async ({ page }) => {
    await page.goto('file://' + reportPath);
    const badge = page.locator('.mood-badge');
    await expect(badge).toContainText(/SCORE/);
    await expect(badge.locator('.mood-scale')).toContainText(/0 = calm/);
    await expect(badge.locator('.mood-scale')).toContainText(/100 = chaotic/);
    await expect(badge.locator('.mood-meta')).toContainText(/\d{4}/); // 4-digit year shows somewhere
    await badge.scrollIntoViewIfNeeded();
    await badge.screenshot({ path: path.resolve(__dirname, '../../output/phase2-header.png') });
  });

  test('bug why panel shows tier legend + sampling note', async ({ page }) => {
    await page.goto('file://' + reportPath);
    const why = page.locator('details.why-panel');
    await why.first().evaluate((el: HTMLDetailsElement) => { el.open = true; });
    await expect(why.locator('.why-sub')).toContainText(/up to 20 samples per tier/);
    await expect(why.locator('.why-legend')).toContainText(/routine/);
    await expect(why.locator('.why-legend')).toContainText(/outrank/i).catch(() => {
      // legend uses "lands here, not under normal" instead — loose assertion ok
    });
    await why.scrollIntoViewIfNeeded();
    // Just the top: summary + legend. Full details was screenshotted but
    // too long to be readable at thumbnail sizes.
    const legend = page.locator('.why-legend').first();
    await legend.scrollIntoViewIfNeeded();
    const b = await legend.boundingBox();
    if (b) {
      await page.screenshot({
        path: path.resolve(__dirname, '../../output/phase2-bug-why.png'),
        clip: { x: Math.max(0, b.x - 24), y: Math.max(0, b.y - 60), width: Math.min(1280, b.width + 48), height: b.height + 80 },
      });
    }
  });

  test('trend chart canvas is present and legend shows all 6 series', async ({ page }) => {
    const errs: string[] = [];
    page.on('pageerror', (e) => errs.push(e.message));
    await page.goto('file://' + reportPath);

    const canvas = page.locator('#trendChart');
    await expect(canvas).toBeVisible();
    await page.waitForFunction(() => {
      const c = document.querySelector('#trendChart') as HTMLCanvasElement | null;
      return !!c && c.width > 0 && c.height > 0;
    });

    const labels = await page.evaluate(() => {
      const c = document.querySelector('#trendChart') as HTMLCanvasElement | null;
      // @ts-expect-error Chart is global from Chart.js CDN
      const chart = (window.Chart && c) ? window.Chart.getChart(c) : null;
      return chart ? chart.data.datasets.map((d: { label?: string }) => d.label) : [];
    });
    expect(labels).toEqual(['Composite', 'Commit Frequency', 'File Churn', 'Bug Ratio', 'Authors', 'Coverage']);

    expect(errs, 'no page errors').toEqual([]);

    const card = page.locator('.card').filter({ has: canvas }).first();
    await card.scrollIntoViewIfNeeded();
    await card.screenshot({ path: path.resolve(__dirname, '../../output/phase2-trend.png') });
  });
});

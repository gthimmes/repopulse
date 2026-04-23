// Manual verification spec — opens a real-data repopulse report and
// screenshots key surfaces. Auto-skipped when the report file isn't
// present so CI stays green for anyone without one locally.
import { test, expect } from '@playwright/test';
import fs from 'fs';
import path from 'path';

const reportPath = path.resolve(__dirname, '../../output/repopulse-report.html');

test.describe('Real-data manual verification', () => {
  test.skip(!fs.existsSync(reportPath), 'no real-data report present; run repopulse against a local repo first');

  test('badge: gradient pressure bar + band label, no emoji', async ({ page }) => {
    await page.goto('file://' + reportPath);
    const badge = page.locator('.pressure-badge');
    await expect(badge).toContainText(/REPO PRESSURE/);
    await expect(badge.locator('.pressure-band')).toBeVisible();
    await expect(badge.locator('.pressure-bar-marker')).toBeVisible();
    // Old emoji shouldn't render anywhere in the badge
    expect(await badge.locator('.mood-emoji').count()).toBe(0);
    await badge.scrollIntoViewIfNeeded();
    await badge.screenshot({ path: path.resolve(__dirname, '../../output/badge-pressure.png') });
  });

  test('Standards card shows compliance + test density', async ({ page }) => {
    await page.goto('file://' + reportPath);
    const card = page.locator('.card').filter({ hasText: 'Standards' }).first();
    await expect(card).toContainText('Commit compliance');
    await expect(card).toContainText('Test density');
  });

  test('Top Churned Files: drillable rows', async ({ page }) => {
    await page.goto('file://' + reportPath);
    const card = page.locator('.card').filter({ hasText: 'Top Churned Files' }).first();
    const rows = card.locator('details.churn-item');
    expect(await rows.count(), 'expected drill rows').toBeGreaterThan(0);
    await rows.first().evaluate((el: HTMLDetailsElement) => { el.open = true; });
    await expect(rows.first()).toContainText(/Top authors of this file/);
  });

  test('Contributors explorer at the bottom: scrollable + drillable', async ({ page }) => {
    await page.goto('file://' + reportPath);
    const card = page.locator('.card').filter({ hasText: 'Contributors' }).first();
    await expect(card).toContainText(/sorted by LOC/);
    const rows = card.locator('details.contrib-item');
    expect(await rows.count(), 'expected ≥1 contributor row').toBeGreaterThan(0);

    // Scroll container should be capped (overflow-y: auto with max-height)
    const list = card.locator('.contributors-list');
    const overflowY = await list.evaluate((el) => getComputedStyle(el).overflowY);
    expect(overflowY).toBe('auto');

    // Open the first contributor and verify the per-person panels render
    const first = rows.first();
    await first.evaluate((el: HTMLDetailsElement) => { el.open = true; });
    await expect(first).toContainText(/Most-touched files/);

    await card.scrollIntoViewIfNeeded();
    await card.screenshot({ path: path.resolve(__dirname, '../../output/contributors-bottom.png') });
  });

  test('trend chart canvas is present', async ({ page }) => {
    await page.goto('file://' + reportPath);
    const canvas = page.locator('#trendChart');
    if (await canvas.count() === 0) {
      test.skip(true, 'only one snapshot — trend chart not rendered');
    }
    await expect(canvas).toBeVisible();
    const labels = await page.evaluate(() => {
      const c = document.querySelector('#trendChart') as HTMLCanvasElement | null;
      // @ts-expect-error Chart is global from Chart.js CDN
      const chart = (window.Chart && c) ? window.Chart.getChart(c) : null;
      return chart ? chart.data.datasets.map((d: { label?: string }) => d.label) : [];
    });
    expect(labels).toEqual(['Composite', 'Commit Frequency', 'File Churn', 'Bug Ratio', 'Authors', 'Coverage']);
  });
});

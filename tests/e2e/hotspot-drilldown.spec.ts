import { test, expect } from '@playwright/test';
import { pathToFileURL } from 'node:url';
import { resolve } from 'node:path';
import { writeFixtureReport } from './fixtures.js';

const REPORT_PATH = resolve(process.cwd(), 'tests/screenshots/fixture-report.html');
const SCREENSHOT_DIR = resolve(process.cwd(), 'tests/screenshots');

test.beforeAll(() => {
  writeFixtureReport(REPORT_PATH);
});

test.describe('Phase 1.1 — Hotspot drill-downs', () => {
  test('report loads with no JS errors and hotspots visible', async ({ page }) => {
    const errors: string[] = [];
    page.on('pageerror', e => errors.push(e.message));
    page.on('console', msg => {
      if (msg.type() === 'error') errors.push(msg.text());
    });

    await page.goto(pathToFileURL(REPORT_PATH).href);
    await page.waitForLoadState('networkidle');

    const hotspots = page.locator('details.hotspot-item');
    await expect(hotspots).toHaveCount(3);

    await page.screenshot({
      path: `${SCREENSHOT_DIR}/01-initial-load.png`,
      fullPage: true,
    });

    expect(errors, `JS errors on load: ${errors.join(' | ')}`).toEqual([]);
  });

  test('hotspot rows are collapsed by default', async ({ page }) => {
    await page.goto(pathToFileURL(REPORT_PATH).href);
    const openCount = await page.locator('details.hotspot-item[open]').count();
    expect(openCount).toBe(0);
  });

  test('clicking a hotspot row expands it and shows detail panel', async ({ page }) => {
    await page.goto(pathToFileURL(REPORT_PATH).href);

    const firstHotspot = page.locator('details.hotspot-item').first();
    await expect(firstHotspot).toHaveAttribute('open', /.*/, { timeout: 1000 }).catch(() => {});

    // Click the summary to expand
    await firstHotspot.locator('summary').click();

    // details element now has `open` attribute
    await expect(firstHotspot).toHaveAttribute('open', '');

    // Detail panel is visible
    const detail = firstHotspot.locator('.hotspot-detail');
    await expect(detail).toBeVisible();

    // Detail meta shows churn rank, authors, last touched
    await expect(detail).toContainText('Churn rank');
    await expect(detail).toContainText('#1');
    await expect(detail).toContainText('Distinct authors');
    await expect(detail).toContainText('Last touched');
    await expect(detail).toContainText('2026-04-12');

    await page.screenshot({
      path: `${SCREENSHOT_DIR}/02-expanded-hotspot.png`,
      fullPage: true,
    });
  });

  test('detail panel renders top-author chips', async ({ page }) => {
    await page.goto(pathToFileURL(REPORT_PATH).href);
    const firstHotspot = page.locator('details.hotspot-item').first();
    await firstHotspot.locator('summary').click();

    const chips = firstHotspot.locator('.author-chip');
    await expect(chips).toHaveCount(3);
    await expect(chips.nth(0)).toContainText('Alice Chen');
    await expect(chips.nth(0)).toContainText('11');
    await expect(chips.nth(1)).toContainText('Bob Martinez');
    await expect(chips.nth(2)).toContainText('Carol Park');
  });

  test('detail panel lists recent bug commits with tier, hash, date, message', async ({ page }) => {
    await page.goto(pathToFileURL(REPORT_PATH).href);
    const firstHotspot = page.locator('details.hotspot-item').first();
    await firstHotspot.locator('summary').click();

    const commits = firstHotspot.locator('ul.commit-list li');
    await expect(commits).toHaveCount(6);

    // First commit should be newest: 2026-04-12, chaos tier
    const first = commits.nth(0);
    await expect(first.locator('.commit-tier')).toHaveText('chaos');
    await expect(first.locator('.commit-tier')).toHaveClass(/tier-chaos/);
    await expect(first.locator('.commit-hash')).toHaveText('a1b2c3d');
    await expect(first).toContainText('2026-04-12');
    await expect(first).toContainText('rounding');

    // Verify we have a mix of tiers
    await expect(firstHotspot.locator('.commit-tier.tier-chaos').first()).toBeVisible();
    await expect(firstHotspot.locator('.commit-tier.tier-normal').first()).toBeVisible();
    await expect(firstHotspot.locator('.commit-tier.tier-routine').first()).toBeVisible();
  });

  test('chevron rotates when expanded (visual state)', async ({ page }) => {
    await page.goto(pathToFileURL(REPORT_PATH).href);
    const firstHotspot = page.locator('details.hotspot-item').first();
    const chevron = firstHotspot.locator('.chevron');

    const transformBefore = await chevron.evaluate(el => getComputedStyle(el).transform);
    await firstHotspot.locator('summary').click();

    // CSS transition may take a moment
    await page.waitForTimeout(250);
    const transformAfter = await chevron.evaluate(el => getComputedStyle(el).transform);

    // Before: 'none' or identity; after: rotation matrix (rotate(90deg) => matrix(0, 1, -1, 0, 0, 0))
    expect(transformBefore).not.toBe(transformAfter);
    expect(transformAfter).toMatch(/matrix/);
  });

  test('multiple rows can be opened independently', async ({ page }) => {
    await page.goto(pathToFileURL(REPORT_PATH).href);
    const items = page.locator('details.hotspot-item');

    await items.nth(0).locator('summary').click();
    await items.nth(1).locator('summary').click();

    await expect(items.nth(0)).toHaveAttribute('open', '');
    await expect(items.nth(1)).toHaveAttribute('open', '');

    await page.screenshot({
      path: `${SCREENSHOT_DIR}/03-multiple-expanded.png`,
      fullPage: true,
    });
  });

  test('no horizontal overflow when rows are expanded', async ({ page }) => {
    await page.goto(pathToFileURL(REPORT_PATH).href);
    await page.locator('details.hotspot-item').nth(0).locator('summary').click();

    const hasHorizontalScroll = await page.evaluate(() => {
      return document.documentElement.scrollWidth > document.documentElement.clientWidth + 1;
    });
    expect(hasHorizontalScroll).toBe(false);
  });
});

import { test, expect } from '@playwright/test';
import { pathToFileURL } from 'node:url';
import { resolve } from 'node:path';
import { writeFixtureReport } from './fixtures.js';

const REPORT_PATH = resolve(process.cwd(), 'tests/screenshots/fixture-report.html');
const SCREENSHOT_DIR = resolve(process.cwd(), 'tests/screenshots');

test.beforeAll(() => {
  writeFixtureReport(REPORT_PATH);
});

test.describe('Phase 1.3 — Hotspot recommendations', () => {
  test('expanded hotspot shows "What to do next" recommendations', async ({ page }) => {
    await page.goto(pathToFileURL(REPORT_PATH).href);
    const firstHotspot = page.locator('details.hotspot-item').first();
    await firstHotspot.locator('summary').click();

    await expect(firstHotspot.locator('.hotspot-detail')).toContainText('What to do next');
    const recs = firstHotspot.locator('.recommendations li');
    await expect(recs).toHaveCount(2);  // fixture has 2 recs on payments/ledger.ts
  });

  test('recommendations carry severity classes for color coding', async ({ page }) => {
    await page.goto(pathToFileURL(REPORT_PATH).href);
    const firstHotspot = page.locator('details.hotspot-item').first();
    await firstHotspot.locator('summary').click();

    const recs = firstHotspot.locator('.recommendations li');
    await expect(recs.nth(0)).toHaveClass(/warn/);
    await expect(recs.nth(0)).toContainText('chaos-tier commits');
  });

  test('recommendations expose kind as a data attribute and rendered pill', async ({ page }) => {
    await page.goto(pathToFileURL(REPORT_PATH).href);
    const firstHotspot = page.locator('details.hotspot-item').first();
    await firstHotspot.locator('summary').click();

    const firstRec = firstHotspot.locator('.recommendations li').first();
    await expect(firstRec).toHaveAttribute('data-rec-kind', /chaos-repeat|bus-factor|bug-heavy|rewritten|unowned|multi-owner|stale-buggy/);
    await expect(firstRec.locator('.rec-kind')).toBeVisible();
  });

  test('unowned hotspot shows an unowned recommendation', async ({ page }) => {
    await page.goto(pathToFileURL(REPORT_PATH).href);
    const thirdHotspot = page.locator('details.hotspot-item').nth(2);  // rate-limiter: owners=[]
    await thirdHotspot.locator('summary').click();

    const recs = thirdHotspot.locator('.recommendations li');
    await expect(recs).toHaveCount(1);
    await expect(recs).toHaveAttribute('data-rec-kind', 'unowned');
    await expect(recs).toContainText('No CODEOWNERS entry');
  });

  test('screenshot: expanded hotspot with recommendations panel', async ({ page }) => {
    await page.goto(pathToFileURL(REPORT_PATH).href);
    await page.locator('details.hotspot-item').first().locator('summary').click();
    await page.screenshot({
      path: `${SCREENSHOT_DIR}/07-recommendations.png`,
      fullPage: true,
    });
  });
});

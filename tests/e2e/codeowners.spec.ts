import { test, expect } from '@playwright/test';
import { pathToFileURL } from 'node:url';
import { resolve } from 'node:path';
import { writeFixtureReport } from './fixtures.js';

const REPORT_PATH = resolve(process.cwd(), 'tests/screenshots/fixture-report.html');
const SCREENSHOT_DIR = resolve(process.cwd(), 'tests/screenshots');

test.beforeAll(() => {
  writeFixtureReport(REPORT_PATH);
});

test.describe('Phase 1.4 — CODEOWNERS team tags', () => {
  test('hotspot with single owner shows one chip', async ({ page }) => {
    await page.goto(pathToFileURL(REPORT_PATH).href);
    const firstHotspot = page.locator('details.hotspot-item').first();
    const chips = firstHotspot.locator('.owner-chip');
    await expect(chips).toHaveCount(1);
    await expect(chips).toHaveText('@org/payments-team');
  });

  test('hotspot with multiple owners shows up to 2 chips', async ({ page }) => {
    await page.goto(pathToFileURL(REPORT_PATH).href);
    const secondHotspot = page.locator('details.hotspot-item').nth(1);
    const chips = secondHotspot.locator('.owner-chip');
    await expect(chips).toHaveCount(2);
    await expect(chips.nth(0)).toHaveText('@org/security');
    await expect(chips.nth(1)).toHaveText('@org/platform');
  });

  test('unowned hotspot shows no chips', async ({ page }) => {
    await page.goto(pathToFileURL(REPORT_PATH).href);
    const thirdHotspot = page.locator('details.hotspot-item').nth(2);
    await expect(thirdHotspot.locator('.owner-chip')).toHaveCount(0);
  });

  test('module cards render owner chips', async ({ page }) => {
    await page.goto(pathToFileURL(REPORT_PATH).href);
    // Cards are in descending score order; the fixture has payments (72) then auth (48)
    const cards = page.locator('.module-card');
    await expect(cards).toHaveCount(2);

    const paymentsCard = cards.nth(0);
    await expect(paymentsCard.locator('.name')).toHaveText('payments');
    await expect(paymentsCard.locator('.owner-chip-inline')).toHaveText('@org/payments-team');

    const authCard = cards.nth(1);
    await expect(authCard.locator('.name')).toHaveText('auth');
    await expect(authCard.locator('.owner-chip-inline')).toHaveText('@org/security');
  });

  test('screenshot: hotspot + module owner chips visible together', async ({ page }) => {
    await page.goto(pathToFileURL(REPORT_PATH).href);
    await page.screenshot({
      path: `${SCREENSHOT_DIR}/06-codeowners-chips.png`,
      fullPage: true,
    });
  });
});

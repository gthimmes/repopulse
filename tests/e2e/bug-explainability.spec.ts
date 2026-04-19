import { test, expect } from '@playwright/test';
import { pathToFileURL } from 'node:url';
import { resolve } from 'node:path';
import { writeFixtureReport } from './fixtures.js';

const REPORT_PATH = resolve(process.cwd(), 'tests/screenshots/fixture-report.html');
const SCREENSHOT_DIR = resolve(process.cwd(), 'tests/screenshots');

test.beforeAll(() => {
  writeFixtureReport(REPORT_PATH);
});

test.describe('Phase 1.2 — Bug signal explainability', () => {
  test('explainability panel exists and is collapsed by default', async ({ page }) => {
    await page.goto(pathToFileURL(REPORT_PATH).href);
    const panel = page.locator('details.why-panel');
    await expect(panel).toHaveCount(1);
    await expect(panel).not.toHaveAttribute('open', '');
    await expect(panel.locator('.why-title')).toHaveText('Why this score?');
  });

  test('expanding panel reveals chaos / normal / routine tier blocks', async ({ page }) => {
    await page.goto(pathToFileURL(REPORT_PATH).href);
    const panel = page.locator('details.why-panel');
    await panel.locator('summary').click();
    await expect(panel).toHaveAttribute('open', '');

    await expect(panel.locator('.why-tier-header.chaos')).toHaveCount(1);
    await expect(panel.locator('.why-tier-header.normal')).toHaveCount(1);
    await expect(panel.locator('.why-tier-header.routine')).toHaveCount(1);

    await page.screenshot({
      path: `${SCREENSHOT_DIR}/04-bug-explainability-expanded.png`,
      fullPage: true,
    });
  });

  test('tier headers show total counts from the bug signal', async ({ page }) => {
    await page.goto(pathToFileURL(REPORT_PATH).href);
    const panel = page.locator('details.why-panel');
    await panel.locator('summary').click();

    // Fixture: chaosCommitCount=6, normalFixCount=27, routineFixCount=5
    await expect(panel.locator('.why-tier-header.chaos')).toContainText('6 commits');
    await expect(panel.locator('.why-tier-header.normal')).toContainText('27 commits');
    await expect(panel.locator('.why-tier-header.routine')).toContainText('5 commits');

    // Normal tier has 27 total but only 4 samples — should say "showing 4 newest"
    await expect(panel.locator('.why-tier-header.normal')).toContainText('showing 4 newest');
  });

  test('each commit row shows matched keyword, hash, date, author, message', async ({ page }) => {
    await page.goto(pathToFileURL(REPORT_PATH).href);
    const panel = page.locator('details.why-panel');
    await panel.locator('summary').click();

    const chaosBlock = panel.locator('.why-tier-block').filter({ has: page.locator('.why-tier-header.chaos') });
    const firstRow = chaosBlock.locator('ul.why-commit-list li').first();

    await expect(firstRow.locator('.kw')).toHaveText('hotfix');
    await expect(firstRow.locator('.kw')).toHaveClass(/chaos/);
    await expect(firstRow.locator('.hash')).toHaveText('a1b2c3d');
    await expect(firstRow.locator('.dateauth')).toContainText('2026-04-12');
    await expect(firstRow.locator('.msg')).toContainText('race on concurrent ledger writes');
  });

  test('matched keyword is highlighted inline in the message', async ({ page }) => {
    await page.goto(pathToFileURL(REPORT_PATH).href);
    const panel = page.locator('details.why-panel');
    await panel.locator('summary').click();

    // Normal tier first row: "fix: expired session..." with "fix" highlighted
    const normalBlock = panel.locator('.why-tier-block').filter({ has: page.locator('.why-tier-header.normal') });
    const firstMark = normalBlock.locator('ul.why-commit-list li').first().locator('.msg mark');

    await expect(firstMark).toHaveCount(1);
    await expect(firstMark).toHaveText(/^fix$/i);
  });

  test('revert commits show "(revert)" keyword pill and no inline highlight', async ({ page }) => {
    await page.goto(pathToFileURL(REPORT_PATH).href);
    const panel = page.locator('details.why-panel');
    await panel.locator('summary').click();

    const revertRow = panel.locator('ul.why-commit-list li').filter({ hasText: 'Revert' });
    await expect(revertRow.locator('.kw')).toHaveText('(revert)');
    // No mark in the message — "(revert)" starts with paren, so we skip highlighting
    await expect(revertRow.locator('.msg mark')).toHaveCount(0);
  });

  test('no JS errors after expanding panel', async ({ page }) => {
    const errors: string[] = [];
    page.on('pageerror', e => errors.push(e.message));
    page.on('console', msg => {
      if (msg.type() === 'error') errors.push(msg.text());
    });

    await page.goto(pathToFileURL(REPORT_PATH).href);
    await page.locator('details.why-panel summary').click();
    await page.waitForTimeout(100);

    expect(errors, `JS errors: ${errors.join(' | ')}`).toEqual([]);
  });
});

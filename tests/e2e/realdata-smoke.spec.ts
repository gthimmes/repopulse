import { test, expect } from '@playwright/test';
import { pathToFileURL } from 'node:url';
import { resolve } from 'node:path';
import { existsSync } from 'node:fs';

/**
 * Smoke test against a real-data report (if one has been generated).
 * Verifies hotspot drill-down + bug explainability work on real commit data,
 * not just the deterministic fixture.
 *
 * Skipped by default so CI without a local repo still passes. To enable:
 *   ./mood-ring.exe /path/to/any/local/repo -output output/mood-report.html
 *   npx playwright test tests/e2e/realdata-smoke.spec.ts
 *
 * The path tracks the CLI's default `-output`, so just running the binary
 * once before Playwright populates it automatically.
 */
const REAL_REPORT = resolve(process.cwd(), 'output/mood-report.html');
const SCREENSHOT_DIR = resolve(process.cwd(), 'tests/screenshots');

test.describe('Real-data smoke', () => {
  test.skip(!existsSync(REAL_REPORT), 'output/mood-report.html not generated; skipping real-data smoke');

  test('real-data report: hotspot drill-down + bug explainability render without errors', async ({ page }) => {
    const errors: string[] = [];
    page.on('pageerror', e => errors.push(e.message));
    page.on('console', msg => {
      if (msg.type() === 'error') errors.push(msg.text());
    });

    await page.goto(pathToFileURL(REAL_REPORT).href);
    await page.waitForLoadState('networkidle');

    // Hotspot drill-down
    const hotspots = page.locator('details.hotspot-item');
    const hotspotCount = await hotspots.count();
    expect(hotspotCount).toBeGreaterThan(0);

    await hotspots.first().locator('summary').click();
    await expect(hotspots.first().locator('.hotspot-detail')).toBeVisible();

    // Bug explainability
    const why = page.locator('details.why-panel');
    await expect(why).toHaveCount(1);
    await why.locator('summary').click();
    await expect(why).toHaveAttribute('open', '');

    // At least one of the three tier blocks should have rows
    const rowCount = await why.locator('ul.why-commit-list li').count();
    expect(rowCount).toBeGreaterThan(0);

    await page.screenshot({
      path: `${SCREENSHOT_DIR}/05-real-data.png`,
      fullPage: true,
    });

    expect(errors, `JS errors on real data: ${errors.join(' | ')}`).toEqual([]);
  });
});

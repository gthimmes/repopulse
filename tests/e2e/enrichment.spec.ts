// Plank-2 Layer-B AI-enrichment render: verify the AI-read card appears
// only when an EnrichmentResult is attached, with the right tag,
// content sections, and accessible structure.
import { test, expect } from '@playwright/test';
import { pathToFileURL } from 'node:url';
import { resolve } from 'node:path';
import { writeFixtureReport, writeEnrichedFixtureReport } from './fixtures.js';

const SCREENSHOT_DIR = resolve(process.cwd(), 'tests/screenshots');
const ENRICHED_PATH = resolve(SCREENSHOT_DIR, 'fixture-enriched.html');
const PLAIN_PATH = resolve(SCREENSHOT_DIR, 'fixture-plain.html');

test.beforeAll(() => {
  writeEnrichedFixtureReport(ENRICHED_PATH);
  writeFixtureReport(PLAIN_PATH);
});

test.describe('Plank-2 Layer-B — AI enrichment render', () => {
  test('plain (deterministic) report does NOT show the AI card', async ({ page }) => {
    await page.goto(pathToFileURL(PLAIN_PATH).href);
    await page.waitForLoadState('networkidle');
    await expect(page.locator('.enriched-card')).toHaveCount(0);
    await expect(page.locator('.enriched-tag')).toHaveCount(0);
  });

  test('enriched report shows AI-GENERATED tag with model + source', async ({ page }) => {
    await page.goto(pathToFileURL(ENRICHED_PATH).href);
    await page.waitForLoadState('networkidle');

    const card = page.locator('.enriched-card');
    await expect(card).toHaveCount(1);
    await expect(card.locator('.enriched-tag')).toHaveText('AI-GENERATED');
    await expect(card.locator('.enriched-meta')).toContainText('claude-code-skill');
    await expect(card.locator('.enriched-meta')).toContainText('claude-fixture');
  });

  test('AI card carries narrative bullets, standards verdict, and drift reading', async ({ page }) => {
    await page.goto(pathToFileURL(ENRICHED_PATH).href);
    const card = page.locator('.enriched-card');

    // Narrative — fixture has 4 bullets, one per kind
    const bullets = card.locator('ul.enriched-narrative > li');
    await expect(bullets).toHaveCount(4);
    await expect(card).toContainText('Bug-tier ratio above the calm band');

    // Standards verdict
    await expect(card).toContainText('Standards holding');
    await expect(card).toContainText('Pair Carol');

    // Drift reading: email resolves to display name, reading + suggestion render
    const drift = card.locator('ul.enriched-drift > li').first();
    await expect(drift.locator('.enriched-drift-name')).toContainText('Alice Chen');
    await expect(drift.locator('.enriched-drift-reading')).toContainText('after-hours load');
    await expect(drift.locator('.enriched-drift-suggestion')).toContainText('ledger work');
  });

  test('AI card sits between Findings and the rest of the report', async ({ page }) => {
    // Sanity: the visual order is Findings → AI read → Standards/PRFlow,
    // so a reader sees deterministic findings first and AI interpretation
    // is clearly demarcated below.
    await page.goto(pathToFileURL(ENRICHED_PATH).href);
    const findingsBox = await page.locator('.card.narrative').first().boundingBox();
    const aiBox = await page.locator('.enriched-card').first().boundingBox();
    expect(findingsBox && aiBox).toBeTruthy();
    expect(aiBox!.y).toBeGreaterThan(findingsBox!.y);
  });

  test('screenshot: enriched fixture full-page', async ({ page }) => {
    await page.goto(pathToFileURL(ENRICHED_PATH).href);
    await page.waitForLoadState('networkidle');
    await page.screenshot({
      path: `${SCREENSHOT_DIR}/enrichment-fullpage.png`,
      fullPage: true,
    });
  });
});

// Thin shim: delegates fixture HTML generation to the Go `fixture-gen` binary.
// The fixture data itself lives in internal/fixtures/ui.go — single source of
// truth, Go-homogeneous with the rest of the codebase.
//
// Build the binary once before running Playwright (CI does this automatically):
//   go build -o fixture-gen.exe ./cmd/fixture-gen    # Windows
//   go build -o fixture-gen ./cmd/fixture-gen        # macOS / Linux

import { execFileSync } from 'node:child_process';
import { resolve } from 'node:path';

const FIXTURE_GEN_BIN = resolve(
  process.cwd(),
  process.platform === 'win32' ? 'fixture-gen.exe' : 'fixture-gen'
);

export function writeFixtureReport(outputPath: string): string {
  const abs = resolve(outputPath);
  execFileSync(FIXTURE_GEN_BIN, [abs], { stdio: 'pipe' });
  return abs;
}

// Same fixture, but with the Plank-2 Layer-B enrichment block injected so
// the AI-read card renders. Used by the enrichment-render Playwright spec.
export function writeEnrichedFixtureReport(outputPath: string): string {
  const abs = resolve(outputPath);
  execFileSync(FIXTURE_GEN_BIN, [abs, '--enriched'], { stdio: 'pipe' });
  return abs;
}

import { readFileSync } from 'node:fs';
import { describe, expect, it } from 'vitest';

const source = readFileSync(`${process.cwd()}/src/app/(main)/settings/plugin/page.tsx`, 'utf8');

describe('extension pairing API paths', () => {
  it('uses the apiClient /v1 prefix for every pairing request', () => {
    expect(source).not.toMatch(/apiClient\.(?:get|post|delete)[^\n]*['`]\/auth\//);
    expect(source.match(/\/v1\/auth\/extension-/g)).toHaveLength(5);
  });

  it('re-enables Owner confirmation after the browser responds', () => {
    expect(source).toContain('setPairing(next); setBusy(false)');
    expect(source).toContain("if (!event.data.ok) { setBusy(false);");
  });
});

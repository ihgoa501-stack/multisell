import { describe, expect, it } from 'vitest';
import { materialBlockers, type MaterialAsset } from './MaterialAssetWorkspace';

function asset(overrides: Partial<MaterialAsset> = {}): MaterialAsset {
  return {
    id: 1, role: 'main', ordinal: 1, source_url: 'https://img.example/a.jpg', source_sha256: 'a'.repeat(64),
    mime_type: 'image/jpeg', media_type: 'image', byte_size: 100, processing_status: 'ready', rights_versions: [], renditions: [],
    latest_rights: { id: 2, version: 1, status: 'approved', effective_status: 'approved', license_scope: 'marketplace', countries: ['US'], channels: ['amazon'], licensor: 'supplier', source_uri: 'evidence://rights', observed_at: '2026-07-13T00:00:00Z' },
    ...overrides,
  };
}

describe('MaterialAssetWorkspace facts', () => {
  it('keeps video blocked while no processor exists even when rights are approved', () => {
    expect(materialBlockers(asset({ role: 'video', media_type: 'video', processing_status: 'blocked', blocker: 'video processor unavailable' }))).toContain('video processor unavailable');
  });

  it('blocks missing rights and SKU images without a canonical mapping', () => {
    expect(materialBlockers(asset({ role: 'sku', latest_rights: undefined, canonical_sku_mapping_id: undefined }))).toEqual(['缺少权利证据', 'SKU图未绑定canonical SKU']);
  });

  it('accepts only ready images with approved effective rights and required SKU binding', () => {
    expect(materialBlockers(asset({ role: 'sku', canonical_sku_mapping_id: 90 }))).toEqual([]);
  });
});

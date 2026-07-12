import { beforeEach, describe, expect, it, vi } from 'vitest';

vi.mock('@/lib/api-client', () => ({ default: { get: vi.fn(), getPage: vi.fn(), post: vi.fn(), upload: vi.fn() } }));

import apiClient from '@/lib/api-client';
import { createCostEntry, createFiveAxisReview, createImageJob, createRightsGrant, executeImageJob, getImageProcessorCapabilities, listImageJobs, uploadSourceImage } from './api';

describe('product images API', () => {
  beforeEach(() => vi.clearAllMocks());

  it('reads processor availability from the backend capability endpoint', async () => {
    const capabilities = [{ code: 'deterministic' as const, name: '凌镜标准处理', configured: false, operations: ['DETERMINISTIC_RESIZE'] }];
    vi.mocked(apiClient.get).mockResolvedValue({ code: 0, message: 'ok', data: capabilities });
    await expect(getImageProcessorCapabilities()).resolves.toEqual(capabilities);
    expect(apiClient.get).toHaveBeenCalledWith('/v1/product-images/capabilities');
  });

  it('rejects a capability response without data instead of assuming availability', async () => {
    vi.mocked(apiClient.get).mockResolvedValue({ code: 0, message: 'ok' });
    await expect(getImageProcessorCapabilities()).rejects.toThrow('能力状态接口没有返回数据');
  });

  it('uses the LingMirror product image task endpoint', async () => {
    vi.mocked(apiClient.getPage).mockResolvedValue({ code: 0, message: 'ok', data: [], total: 0, page: 1, size: 20 });
    await listImageJobs();
    expect(apiClient.getPage).toHaveBeenCalledWith('/v1/product-images/tasks');
  });

  it('uploads source bytes rather than a browser URL', async () => {
    const asset = { id: 1, owner_id: 1, blob_id: 'blob-1', filename: 'shoe.png', content_type: 'image/png', size_bytes: 3, sha256: 'abc', truth: 'actual' };
    vi.mocked(apiClient.upload).mockResolvedValue({ code: 0, message: 'ok', data: asset });
    await expect(uploadSourceImage(new File(['png'], 'shoe.png', { type: 'image/png' }))).resolves.toEqual(asset);
    const [, form] = vi.mocked(apiClient.upload).mock.calls[0];
    expect(apiClient.upload).toHaveBeenCalledWith('/v1/product-images/assets', expect.any(FormData));
    expect(form.get('file')).toBeInstanceOf(File);
  });

  it('creates and explicitly executes a deterministic task', async () => {
    const job = { id: 1, owner_id: 1, asset_id: 1, idempotency_key: 'create-key', manifest_hash: 'manifest', operation: 'DETERMINISTIC_RESIZE', width: 1200, height: 1200, format: 'png', status: 'pending' as const };
    vi.mocked(apiClient.post).mockResolvedValue({ code: 0, message: 'ok', data: job });
    await createImageJob({ asset_id: 1, idempotency_key: 'create-key', operation: 'DETERMINISTIC_RESIZE', width: 1200, height: 1200, format: 'png' });
    await executeImageJob(1);
    expect(apiClient.post).toHaveBeenNthCalledWith(1, '/v1/product-images/tasks', expect.objectContaining({ operation: 'DETERMINISTIC_RESIZE' }));
    expect(apiClient.post).toHaveBeenNthCalledWith(2, '/v1/product-images/tasks/1/executions', expect.objectContaining({ idempotency_key: expect.any(String) }));
  });

  it('writes rights, five-axis review and cost to the exact governance endpoints', async () => {
    vi.mocked(apiClient.post).mockImplementation(async (path) => ({ code: 0, message: 'ok', data: { id: 1, path } }));
    await createRightsGrant({ asset_sha256: 'a'.repeat(64), can_copy: true, can_modify: true, can_third_party_ai: false, can_cross_border: true, can_commercial_publish: true, can_platform_sublicense: true, trademark_cleared: true, likeness_cleared: true, purpose: 'listing_main', jurisdiction: '*', channel: 'ozon', provider: 'deterministic', region: 'local', grantor: 'owner', rights_chain: 'contract', evidence_sha256: 'b'.repeat(64), owner_verified: true, valid_from: new Date().toISOString(), idempotency_key: 'rights-1', expected_version: 1 });
    await createFiveAxisReview(7, { asset_sha256: 'a'.repeat(64), purpose: 'listing_main', channel: 'ozon', product_authenticity: 'passed', rights: 'passed', channel_rules: 'passed', claims_scene: 'passed', technical_visual: 'passed', evidence_sha256: 'c'.repeat(64), evidence_truth: 'quoted', idempotency_key: 'review-1', expected_version: 1 });
    await createCostEntry(7, { kind: 'estimated', category: 'provider', provider: 'deterministic', amount: '0', currency: 'USD', exchange_rate: '1', exchange_rate_source: 'owner', observed_at: new Date().toISOString(), billing_status: 'estimated', idempotency_key: 'cost-1', expected_version: 1 });
    expect(apiClient.post).toHaveBeenNthCalledWith(1, '/v1/product-images/rights-grants', expect.objectContaining({ asset_sha256: 'a'.repeat(64) }));
    expect(apiClient.post).toHaveBeenNthCalledWith(2, '/v1/product-images/tasks/7/reviews', expect.objectContaining({ product_authenticity: 'passed' }));
    expect(apiClient.post).toHaveBeenNthCalledWith(3, '/v1/product-images/tasks/7/costs', expect.objectContaining({ amount: '0' }));
  });
});

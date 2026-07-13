import { beforeEach, describe, expect, it, vi } from 'vitest';

vi.mock('@/lib/api-client', () => ({ default: { get: vi.fn(), getPage: vi.fn(), post: vi.fn(), upload: vi.fn() } }));

import apiClient from '@/lib/api-client';
import { approveImageExecution, cancelImageBudgetReservation, createCandidateFeedback, createCostEntry, createFiveAxisReview, createImageBudgetPolicy, createImageJob, createImageRuleSnapshot, createManualImport, createRightsGrant, decideImageSet, executeImageJob, getImageProcessorCapabilities, getImageReleaseAttestation, getRecipeSummary, issueImageReleaseAttestation, listImageBudgetPolicies, listImageBudgetReservations, listImageJobs, listManualImports, listSKUOptions, reconcileImageBudgetCharge, reconcileImageBudgetNoCharge, uploadSourceImage } from './api';

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
    await createImageJob({ asset_id: 1, sku_id: 1, recipe_key: 'recipe-1', recipe_version: 1, candidate_round: 1, recipe: { scene_structure: 'clean studio', model: 'deterministic', model_version: '1', parameters: {}, must_not_change: ['shape'] }, idempotency_key: 'create-key', operation: 'DETERMINISTIC_RESIZE', processor: 'deterministic', purpose: 'listing_main', channel: 'ozon', region: 'local', width: 1200, height: 1200, format: 'png' });
    await executeImageJob(1);
    expect(apiClient.post).toHaveBeenNthCalledWith(1, '/v1/product-images/tasks', expect.objectContaining({ operation: 'DETERMINISTIC_RESIZE', processor: 'deterministic', purpose: 'listing_main', channel: 'ozon', region: 'local' }));
    expect(apiClient.post).toHaveBeenNthCalledWith(2, '/v1/product-images/tasks/1/executions', expect.objectContaining({ idempotency_key: expect.any(String) }));
  });

  it('binds paid approval to the exact task version and reservation ceiling', async () => {
    const job = { id: 9, owner_id: 1, asset_id: 1, idempotency_key: 'openai', manifest_hash: 'a'.repeat(64), operation: 'OPENAI_IMAGE_EDIT' as const, processor: 'openai', max_cost: '0.20', currency: 'USD', version: 3, width: 1024, height: 1024, format: 'png', status: 'QUEUED' as const };
    vi.mocked(apiClient.post).mockResolvedValue({ code: 0, message: 'ok', data: { id: 4 } });
    await approveImageExecution(job);
    expect(apiClient.post).toHaveBeenCalledWith('/v1/product-images/tasks/9/execution-approvals', { processor: 'openai', max_cost: '0.20', currency: 'USD', expected_version: 3 });
  });

  it('reads exact SKU options and persists candidate feedback with recipe statistics', async () => {
    const sku = { id: 5, product_id: 2, code: 'SKU-5', spec_desc: 'blue' };
    const feedback = { id: 8, task_id: 7, outcome: 'rejected' as const, reason_codes: ['color' as const], review_seconds: 12, asset_sha256: 'a'.repeat(64), idempotency_key: 'feedback-1', expected_version: 1 };
    const summary = { recipe_key: 'recipe-1', sku_id: 5, purpose: 'scene_gallery', channel: 'ozon', latest_recipe_version: 1, candidates: 3, selected: 2, rejected: 1, rework_requested: 0, acceptance_rate: 2 / 3, review_seconds: 36, production_seconds: 60, rework_rounds: 0, actual_cost: '0.3000', currency: 'USD' };
    vi.mocked(apiClient.getPage).mockResolvedValue({ code: 0, message: 'ok', data: [sku], total: 1, page: 1, size: 100 });
    vi.mocked(apiClient.post).mockResolvedValue({ code: 0, message: 'ok', data: feedback });
    vi.mocked(apiClient.get).mockResolvedValue({ code: 0, message: 'ok', data: summary });
    await expect(listSKUOptions()).resolves.toEqual([sku]);
    await expect(createCandidateFeedback(7, feedback)).resolves.toEqual(feedback);
    await expect(getRecipeSummary('recipe-1')).resolves.toEqual(summary);
    expect(apiClient.getPage).toHaveBeenCalledWith('/v1/skus', { page: '1', size: '100' });
    expect(apiClient.post).toHaveBeenCalledWith('/v1/product-images/tasks/7/feedback', feedback);
    expect(apiClient.get).toHaveBeenCalledWith('/v1/product-images/recipes/recipe-1/summary');
  });

  it('uploads manual import bytes with immutable provenance and lists them', async () => {
    const imported = { id: 9, asset_id: 2, asset_sha256: 'b'.repeat(64), parent_asset_id: 1, parent_asset_sha256: 'a'.repeat(64), import_kind: 'manual_import' as const, tool: 'Photoshop', operation: 'retouch', fee_amount: '2.50', fee_currency: 'USD', model: 'unknown', model_version: 'unknown', channel_restriction: '*', source_observed_at: new Date().toISOString(), truth: 'unknown' as const, idempotency_key: 'manual-1' };
    vi.mocked(apiClient.upload).mockResolvedValue({ code: 0, message: 'ok', data: imported });
    vi.mocked(apiClient.getPage).mockResolvedValue({ code: 0, message: 'ok', data: [imported], total: 1, page: 1, size: 20 });
    await expect(createManualImport({ file: new File(['edited'], 'edited.png', { type: 'image/png' }), parent_asset_id: 1, parent_asset_sha256: 'a'.repeat(64), import_kind: 'manual_import', tool: 'Photoshop', operation: 'retouch', fee_amount: '2.50', fee_currency: 'USD', model: 'unknown', model_version: 'unknown', channel_restriction: '*', source_observed_at: imported.source_observed_at, idempotency_key: 'manual-1' })).resolves.toEqual(imported);
    const [, form] = vi.mocked(apiClient.upload).mock.calls[0];
    expect(apiClient.upload).toHaveBeenCalledWith('/v1/product-images/manual-imports', expect.any(FormData));
    expect(form.get('parent_asset_sha256')).toBe('a'.repeat(64));
    await expect(listManualImports()).resolves.toEqual([imported]);
    expect(apiClient.getPage).toHaveBeenCalledWith('/v1/product-images/manual-imports');
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

  it('uses the exact immutable release-proof endpoints and never exposes a consume action', async () => {
    const rule = { id: 11, channel: 'ozon', site: 'ru', locale: 'ru-RU', category_id: 42, version: 1, rules: {}, rules_sha256: 'a'.repeat(64), effective_at: new Date().toISOString() };
    const decision = { id: 12, image_set_id: 7, image_set_version: 2, set_manifest_sha256: 'b'.repeat(64), decision: 'approved' as const, reason: '已核对', decided_at: new Date().toISOString() };
    const attestation = { id: 13, listing_id: 5, image_set_id: 7, rule_snapshot_id: 11, status: 'issued' as const, items: [] };
    vi.mocked(apiClient.post).mockResolvedValueOnce({ code: 0, message: 'ok', data: rule }).mockResolvedValueOnce({ code: 0, message: 'ok', data: decision }).mockResolvedValueOnce({ code: 0, message: 'ok', data: attestation });
    vi.mocked(apiClient.get).mockResolvedValue({ code: 0, message: 'ok', data: attestation });
    await createImageRuleSnapshot({ channel: 'ozon', site: 'ru', locale: 'ru-RU', category_id: 42, rules: { minimum_width: 1200 }, effective_at: rule.effective_at, idempotency_key: 'rule-1' });
    await decideImageSet(7, { decision: 'approved', reason: '已核对', expected_version: 2, idempotency_key: 'decision-1' });
    await issueImageReleaseAttestation({ image_set_id: 7, rule_snapshot_id: 11, platform_account_id: 3, site: 'ru', ttl_seconds: 900, idempotency_key: 'attestation-1' });
    await getImageReleaseAttestation(13);
    expect(apiClient.post).toHaveBeenNthCalledWith(1, '/v1/product-images/rule-snapshots', expect.objectContaining({ category_id: 42 }));
    expect(apiClient.post).toHaveBeenNthCalledWith(2, '/v1/product-images/image-sets/7/decisions', expect.objectContaining({ expected_version: 2 }));
    expect(apiClient.post).toHaveBeenNthCalledWith(3, '/v1/product-images/release-attestations', expect.objectContaining({ platform_account_id: 3 }));
    expect(apiClient.get).toHaveBeenCalledWith('/v1/product-images/release-attestations/13');
    expect(vi.mocked(apiClient.post).mock.calls.flatMap(([path]) => [path])).not.toContain(expect.stringContaining('consume'));
  });

  it('creates and lists immutable budget policies', async () => {
    const policy = { id: 4, owner_id: 1, currency: 'USD', period_start: '2026-07-01T00:00:00Z', period_end: '2026-08-01T00:00:00Z', total_amount: '100.0000', idempotency_key: 'budget-1' };
    vi.mocked(apiClient.post).mockResolvedValue({ code: 0, message: 'ok', data: policy });
    vi.mocked(apiClient.get).mockResolvedValue({ code: 0, message: 'ok', data: [policy] });
    await expect(createImageBudgetPolicy({ currency: 'USD', period_start: policy.period_start, period_end: policy.period_end, total_amount: '100', idempotency_key: 'budget-1' })).resolves.toEqual(policy);
    await expect(listImageBudgetPolicies()).resolves.toEqual([policy]);
    expect(apiClient.post).toHaveBeenCalledWith('/v1/product-images/budget-policies', expect.objectContaining({ total_amount: '100' }));
    expect(apiClient.get).toHaveBeenCalledWith('/v1/product-images/budget-policies');
  });

  it('lists reservations and only cancels through the dedicated route', async () => {
    const reservation = { id: 8, owner_id: 1, policy_id: 4, approval_id: 2, task_id: 9, task_version: 1, manifest_hash: 'a'.repeat(64), provider: 'openai', currency: 'USD', reserved_amount: '2.50', state: 'reserved' as const };
    vi.mocked(apiClient.get).mockResolvedValue({ code: 0, message: 'ok', data: [reservation] });
    vi.mocked(apiClient.post).mockResolvedValue({ code: 0, message: 'ok', data: { ...reservation, state: 'released' } });
    await expect(listImageBudgetReservations()).resolves.toEqual([reservation]);
    await cancelImageBudgetReservation(8, '不再执行');
    expect(apiClient.get).toHaveBeenCalledWith('/v1/product-images/budget-reservations');
    expect(apiClient.post).toHaveBeenCalledWith('/v1/product-images/budget-reservations/8/cancel', { reason: '不再执行' });
  });

  it('reconciles a charged-without-output terminal result with explicit evidence', async () => {
    const input = { amount: '2.75', currency: 'USD', evidence_sha256: 'b'.repeat(64), observed_at: '2026-07-12T00:00:00Z', idempotency_key: 'charge-1', resolution: 'charged_no_output' as const };
    vi.mocked(apiClient.post).mockResolvedValue({ code: 0, message: 'ok', data: { id: 3, reservation_id: 8, ...input, delta_amount: '0.25', kind: 'provider_bill', over_budget: true } });
    await reconcileImageBudgetCharge(8, input);
    expect(apiClient.post).toHaveBeenCalledWith('/v1/product-images/budget-reservations/8/charges', input);
  });

  it('submits exact no-charge evidence through the dedicated reconciliation route', async () => {
    const input = { evidence_sha256: 'c'.repeat(64), observed_at: '2026-07-13T00:00:00Z', reason: 'Provider dashboard shows no request or charge', idempotency_key: 'no-charge-1' };
    vi.mocked(apiClient.post).mockResolvedValue({ code: 0, message: 'ok', data: { id: 4, reservation_id: 8, amount: '0', currency: 'USD', kind: 'no_charge', ...input } });
    await reconcileImageBudgetNoCharge(8, input);
    expect(apiClient.post).toHaveBeenCalledWith('/v1/product-images/budget-reservations/8/no-charge-reconciliations', input);
  });

  it('fails closed when budget mutation responses omit data', async () => {
    vi.mocked(apiClient.post).mockResolvedValue({ code: 0, message: 'ok' });
    await expect(createImageBudgetPolicy({ currency: 'USD', period_start: '2026-07-01T00:00:00Z', period_end: '2026-08-01T00:00:00Z', total_amount: '100', idempotency_key: 'x' })).rejects.toThrow('没有返回记录');
    await expect(cancelImageBudgetReservation(1, '取消')).rejects.toThrow('没有返回记录');
    await expect(reconcileImageBudgetCharge(1, { amount: '1', currency: 'USD', evidence_sha256: 'a'.repeat(64), observed_at: '2026-07-12T00:00:00Z', idempotency_key: 'x' })).rejects.toThrow('没有返回记录');
    await expect(reconcileImageBudgetNoCharge(1, { evidence_sha256: 'a'.repeat(64), observed_at: '2026-07-12T00:00:00Z', reason: 'none', idempotency_key: 'x' })).rejects.toThrow('没有返回记录');
  });
});

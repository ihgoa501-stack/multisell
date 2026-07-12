import { describe, expect, it } from 'vitest';
import { buildDraftPayload, buildPublishRequestPayload, buildReconcilePayload, buildSampleTransitionPayload, collectionCompleteness, editableDraftToForm, publishStatusMeta, sampleNextStatuses, taskLinkAvailableActions, taskLinkStatus, taskWorkflowPath } from './page';

const observedAt = '2026-07-12T03:00:00.000Z';

function formValues() {
  return {
    conversion_request_id: 'convert_12_once',
    platform_id: 2, category_id: 9, currency: 'CNY', title: '内部标题', description: '内部说明',
    platform_sku: 'MAIN-1', unit: '件', localized_title: 'Товар', localized_description: 'Описание', target_locale: 'ru-RU',
    localized_bullet_points: ['Пункт'], localized_keywords: ['слово'], localized_attributes: [{ name: 'цвет', value: 'белый' }],
    localization_rule_truth_status: 'quoted', localization_rule_source_uri: 'https://rules/localization', localization_rule_observed_at: observedAt,
    allowed_scripts: ['cyrillic'], min_title_length: 1, max_title_length: 200, min_bullet_points: 1, max_bullet_length: 100, min_keywords: 1, prohibited_words: [],
    sku_variants: [{ supplier_sku: 'SUP-1', internal_sku: 'INT-1', channel_sku: 'CH-1', color: 'white', size: 'M', material: 'cotton', packaging: 'bag', cost_price: 5, price: 12, weight: 0.2, truth_status: 'quoted', source_uri: 'https://1688/sku', observed_at: observedAt }],
    sku_rule_truth_status: 'quoted', sku_rule_source_uri: 'https://rules/sku', sku_rule_observed_at: observedAt,
    media: [{ processing_record_id: 7, source_url: 'https://1688/image.jpg', processed_url: '/processed/7', media_role: 'main', rights_evidence_uri: 'evidence://rights', rights_observed_at_exact: observedAt, channel_rule_uri: 'https://rules/image', content_sha256: 'abc', width: 1200, height: 1200, background: 'white', cropped: true, clarity_score: 1, no_watermark: true, no_chinese_text: true, no_brand_mark: true, verification_observed_at: observedAt }],
    image_rule_truth_status: 'quoted', image_rule_source_uri: 'https://rules/image', image_rule_observed_at: observedAt,
    allowed_backgrounds: ['white'], require_crop: true, min_clarity_score: 0.8, image_rule_min_images: 1, image_rule_max_images: 9,
    costs: [{ cost_type: 'purchase', amount: 5, currency: 'CNY', truth_status: 'quoted', source_uri: 'https://1688/price', observed_at: observedAt }],
    exchange_rates: [], revenue_amount: 12, revenue_currency: 'CNY', revenue_truth_status: 'estimated', revenue_source_uri: 'evidence://revenue', revenue_observed_at: observedAt,
    supplier_assessment: [{ check_type: 'identity', result: 'pass', truth_status: 'quoted', source_uri: 'https://1688/shop', observed_at: observedAt }],
    compliance_checks: [{ check_type: 'brand_ip', result: 'pass', truth_status: 'actual', source_uri: 'evidence://compliance', observed_at: observedAt }],
    category_schema_uri: 'https://rules/category', category_observed_at: observedAt, channel_rule_truth_status: 'quoted',
    category_attributes: [{ name: 'material', value: 'cotton' }], required_category_attributes: ['material'],
    variant_dimensions: ['color'], required_variant_dimensions: ['color'], allowed_variant_dimensions: ['color'],
    min_images: 1, max_images: 9, min_image_width: 1000, min_image_height: 1000, shipping_template_id: 'SHIP-1',
  };
}

describe('buildDraftPayload', () => {
  it('generates matching SKU, media, cost and channel validation snapshots without JSON input', () => {
    const payload = buildDraftPayload(formValues());

    expect(payload.sku_variants[0]).toMatchObject({
      supplier_sku: 'SUP-1', internal_sku: 'INT-1', channel_sku: 'CH-1',
      spec_values: { color: 'white', size: 'M', material: 'cotton', packaging: 'bag' },
    });
    expect(payload.validation.skus[0]).toMatchObject({ supplier_sku: 'SUP-1', internal_sku: 'INT-1', channel_sku: 'CH-1' });
    expect(payload.media[0]).toMatchObject({
      rights_status: 'verified', has_watermark: false, has_chinese_text: false, has_brand_mark: false,
      operations: [{ operation: 'center_crop' }, { operation: 'resize', width: 1200, height: 1200 }, { operation: 'white_background' }],
    });
    expect(payload.validation.images[0]).toMatchObject({ truth_status: 'actual', source_uri: 'sha256:abc' });
    expect(payload.validation.costs.costs[0]).toEqual({ type: 'purchase', amount: 5, currency: 'CNY', truth_status: 'quoted', source_uri: 'https://1688/price', observed_at: observedAt });
    expect(payload.listing_payload).toEqual({ attributes: { material: 'cotton' }, variant_dimensions: ['color'], shipping_template_id: 'SHIP-1' });
    expect(payload.validation.channel).toMatchObject({ platform_id: 2, category_id: '9', image_count: 1 });
  });

  it('round-trips the canonical editable payload without losing approval evidence or SKU/media facts', () => {
    const original = buildDraftPayload(formValues());
    const restored = buildDraftPayload(editableDraftToForm({ ...original, editable_version: 7, editable_sha256: 'a'.repeat(64) }));

    expect(restored).toEqual({ ...original, editable_version: 7, editable_sha256: 'a'.repeat(64) });
  });
});

describe('publish safety payloads', () => {
  it('freezes a complete SKU inventory map without carrying UI-only rows', () => {
    expect(buildPublishRequestPayload({
      platform_account_id: 12,
      idempotency_key: '  publish-once-1  ',
      reason: '  最小真实实验  ',
      inventory_rows: [{ sku_code: 'INT-1', quantity: 3 }, { sku_code: 'INT-2', quantity: 0 }],
    })).toEqual({
      platform_account_id: 12,
      idempotency_key: 'publish-once-1',
      reason: '最小真实实验',
      inventories: { 'INT-1': 3, 'INT-2': 0 },
    });
  });

  it('forces reconciliation truth to actual and structures the platform result', () => {
    expect(buildReconcilePayload({
      outcome: 'submitted', evidence_uri: ' evidence://platform ', observed_at: observedAt,
      platform_product_id: ' P-1 ', platform_sku: ' SKU-1 ', platform_url: ' https://platform/item/1 ',
      sync_message: ' accepted ', published_data: [{ name: 'review_status', value: 'pending' }],
    })).toEqual({
      outcome: 'submitted', evidence_uri: 'evidence://platform', observed_at: observedAt, truth_status: 'actual',
      platform_result: {
        platform_product_id: 'P-1', platform_sku: 'SKU-1', platform_url: 'https://platform/item/1',
        sync_message: 'accepted', published_data: { review_status: 'pending' },
      },
    });
  });

  it('does not describe submitted as confirmed live', () => {
    expect(publishStatusMeta.submitted.label).toBe('平台已接收请求');
    expect(publishStatusMeta.submitted.description).toContain('不等于商品已真实上线');
    expect(publishStatusMeta.reconcile_required.description).toContain('禁止重试');
  });
});

describe('task link status presentation', () => {
  it('builds an exact task-scoped route and rejects missing identities', () => {
    expect(taskWorkflowPath(12, 34, '/publish-requests/56/execute')).toBe('/v1/sourcing-1688/12/task-links/34/publish-requests/56/execute');
    expect(() => taskWorkflowPath(12, 0, 'draft')).toThrow('invalid sourcing task identity');
  });

  it('surfaces a blocker even when the persisted link status is still linked', () => {
    expect(taskLinkStatus({ status: 'linked', current_status: 'pending_review', current_blocker: '缺少受控采集快照' })).toEqual({
      color: 'red', label: '已阻塞', status: 'pending_review',
    });
  });

  it('uses the current workflow state when no blocker exists', () => {
    expect(taskLinkStatus({ status: 'linked', current_status: 'active' })).toEqual({
      color: 'processing', label: '进行中', status: 'active',
    });
  });

  it('uses the independent workflow state instead of the legacy primary status', () => {
    expect(taskLinkStatus({ status: 'linked', current_status: 'ready_for_draft', workflow_status: 'pending_approval' })).toEqual({
      color: 'orange', label: '草稿待审批', status: 'pending_approval',
    });
  });

  it('exposes only actions valid for the exact task draft', () => {
    expect(taskLinkAvailableActions({ workflow_status: 'ready_for_draft' })).toEqual({
      convert: true, edit: false, submitApproval: false, decideApproval: false, publish: false,
    });
    expect(taskLinkAvailableActions({ workflow_status: 'pending_approval', draft_id: 91 })).toEqual({
      convert: false, edit: false, submitApproval: false, decideApproval: true, publish: false,
    });
    expect(taskLinkAvailableActions({ workflow_status: 'approved_draft', draft_id: 91 })).toEqual({
      convert: false, edit: false, submitApproval: false, decideApproval: false, publish: true,
    });
  });

  it('suppresses all mutating actions when this task is blocked', () => {
    expect(taskLinkAvailableActions({ workflow_status: 'ready_for_draft', current_blocker: '市场决定已暂停' })).toEqual({
      convert: false, edit: false, submitApproval: false, decideApproval: false, publish: false,
    });
  });
});

describe('collection box completeness', () => {
  it('treats an explicitly confirmed no-SKU page as collected instead of inventing variants', () => {
    expect(collectionCompleteness({
      title: 'observed', price: 'observed', moq: 'observed', supplier: 'observed', images: 'observed', sku: 'no_sku',
    })).toEqual({ complete: 6, total: 6, missing: [], isComplete: true });
  });

  it('surfaces unknown and parser failures as named missing fields', () => {
    expect(collectionCompleteness({ title: 'observed', price: 'unknown', sku: 'parse_failed' })).toEqual({
      complete: 1,
      total: 6,
      missing: [
        { field: 'price', label: '价格', status: 'unknown' },
        { field: 'moq', label: '起订量', status: 'unknown' },
        { field: 'supplier', label: '供应商身份', status: 'unknown' },
        { field: 'images', label: '主图', status: 'unknown' },
        { field: 'sku', label: 'SKU', status: 'parse_failed' },
      ],
      isComplete: false,
    });
  });
});

describe('sample fact chain', () => {
  it('exposes only strict next states', () => {
    expect(sampleNextStatuses('request')).toEqual(['approved_to_order']);
    expect(sampleNextStatuses('approved_to_order')).toEqual(['ordered']);
    expect(sampleNextStatuses('ordered')).toEqual(['received']);
    expect(sampleNextStatuses('received')).toEqual(['evaluated']);
    expect(sampleNextStatuses('evaluated')).toEqual(['accepted', 'rejected']);
    expect(sampleNextStatuses('accepted')).toEqual([]);
  });

  it('does not turn Owner approval into an actual order', () => {
    expect(buildSampleTransitionPayload('approved_to_order', { note: '  预算内，可购买一个样品  ' })).toEqual({
      to_status: 'approved_to_order', note: '预算内，可购买一个样品',
    });
  });

  it('requires the order UI payload to carry actual proof and amount', () => {
    expect(buildSampleTransitionPayload('ordered', {
      note: ' 1688订单已支付 ', external_credential_uri: ' evidence://order/123 ', observed_at: observedAt,
      order_amount: 28.5, currency: ' cny ',
    })).toEqual({
      to_status: 'ordered', note: '1688订单已支付', truth_status: 'actual',
      external_credential_uri: 'evidence://order/123', observed_at: observedAt, order_amount: 28.5, currency: 'CNY',
    });
  });
});

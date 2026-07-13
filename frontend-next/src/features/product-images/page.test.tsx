import { QueryClient, QueryClientProvider } from '@tanstack/react-query';
import { fireEvent, render, screen, waitFor } from '@testing-library/react';
import userEvent from '@testing-library/user-event';
import { beforeEach, describe, expect, it, vi } from 'vitest';

vi.mock('next/navigation', () => ({ usePathname: () => '/product-images' }));
vi.mock('./api', () => ({
  getImageProcessorCapabilities: vi.fn(), listImageJobs: vi.fn(), uploadSourceImage: vi.fn(),
  createImageJob: vi.fn(), approveImageExecution: vi.fn(), executeImageJob: vi.fn(), fetchImageOutput: vi.fn(),
  createRightsGrant: vi.fn(), createFiveAxisReview: vi.fn(), createCostEntry: vi.fn(),
  createManualImport: vi.fn(), listManualImports: vi.fn(),
  createProductImageSet: vi.fn(), freezeProductImageSet: vi.fn(),
  createImageRuleSnapshot: vi.fn(), decideImageSet: vi.fn(), issueImageReleaseAttestation: vi.fn(), getImageReleaseAttestation: vi.fn(),
  createImageBudgetPolicy: vi.fn(), listImageBudgetPolicies: vi.fn(), listImageBudgetReservations: vi.fn(),
  cancelImageBudgetReservation: vi.fn(), reconcileImageBudgetCharge: vi.fn(), reconcileImageBudgetNoCharge: vi.fn(),
  listSKUOptions: vi.fn(), createCandidateFeedback: vi.fn(), getRecipeSummary: vi.fn(),
  newImageIdempotencyKey: vi.fn(() => 'test-idempotency-key'),
}));

import ProductImagesPage from '@/app/(main)/product-images/page';
import { approveImageExecution, createImageJob, createRightsGrant, executeImageJob, getImageProcessorCapabilities, getRecipeSummary, listImageBudgetPolicies, listImageBudgetReservations, listImageJobs, listManualImports, listSKUOptions, uploadSourceImage } from './api';

function renderPage() {
  const client = new QueryClient({ defaultOptions: { queries: { retry: false }, mutations: { retry: false } } });
  return render(<QueryClientProvider client={client}><ProductImagesPage /></QueryClientProvider>);
}

describe('product image workspace', () => {
  beforeEach(() => {
    vi.clearAllMocks();
    vi.mocked(listImageJobs).mockResolvedValue([]);
    vi.mocked(listManualImports).mockResolvedValue([]);
    vi.mocked(listSKUOptions).mockResolvedValue([{ id: 1, code: 'SKU-1', spec_desc: '默认规格', product_id: 1 }]);
    vi.mocked(listImageBudgetPolicies).mockResolvedValue([]);
    vi.mocked(listImageBudgetReservations).mockResolvedValue([]);
    vi.mocked(getImageProcessorCapabilities).mockResolvedValue([
      { code: 'deterministic', name: '凌镜标准处理', configured: true, operations: ['DETERMINISTIC_RESIZE'] },
      { code: 'photoroom', name: 'Photoroom', configured: false, operations: [] },
      { code: 'adobe', name: 'Adobe Firefly', configured: false, operations: [] },
      { code: 'openai', name: 'OpenAI Images', configured: false, operations: [] },
    ]);
  });

  it('shows the real deterministic path and marks an external provider unconfigured', async () => {
    renderPage();
    expect(screen.getByText('商品图片工作室')).toBeInTheDocument();
    expect(await screen.findByText('凌镜标准处理 · 可用')).toBeInTheDocument();
    expect(screen.getByText('Photoroom · sandbox canary 不可用：一次 canary 配额不可用')).toBeInTheDocument();
    expect(screen.getByText('OpenAI Images · 付费执行未启用：后端生产安全条件不完整')).toBeInTheDocument();
    expect(screen.getByRole('button', { name: '创建确定性图片任务' })).toBeDisabled();
  });

  it('fails closed while capability state is loading or unavailable', async () => {
    let rejectCapabilities!: (reason: Error) => void;
    vi.mocked(getImageProcessorCapabilities).mockReturnValue(new Promise((_, reject) => { rejectCapabilities = reject; }));
    renderPage();
    expect(screen.getByText('正在读取真实能力状态…')).toBeInTheDocument();
    expect(screen.getByRole('button', { name: '创建确定性图片任务' })).toBeDisabled();
    rejectCapabilities(new Error('能力接口不可达'));
    await screen.findByText('能力状态读取失败');
    expect(screen.getByText(/能力接口不可达/)).toBeInTheDocument();
    expect(screen.getByRole('button', { name: '创建确定性图片任务' })).toBeDisabled();
  });

  it('does not enable deterministic work when the backend reports it unconfigured', async () => {
    vi.mocked(getImageProcessorCapabilities).mockResolvedValue([
      { code: 'deterministic', name: '凌镜标准处理', configured: false, operations: ['DETERMINISTIC_RESIZE'], reason: 'Image Service 未连接' },
      { code: 'openai', name: 'OpenAI Images', configured: true, operations: ['IMAGE_GENERATION'] },
    ]);
    renderPage();
    expect(await screen.findByText('凌镜标准处理 · 未配置')).toBeInTheDocument();
    expect(screen.getByText('Image Service 未连接')).toBeInTheDocument();
    expect(screen.getByText('OpenAI Images · 付费执行未启用：后端生产安全条件不完整')).toBeInTheDocument();
    expect(screen.queryByText('OpenAI Images · 可用')).not.toBeInTheDocument();
    expect(screen.getByRole('button', { name: '创建确定性图片任务' })).toBeDisabled();
  });

  it('keeps Photoroom disabled unless every sandbox capability gate is explicit', async () => {
    vi.mocked(getImageProcessorCapabilities).mockResolvedValue([
      { code: 'deterministic', name: '凌镜标准处理', configured: true, availability: 'available', operations: ['DETERMINISTIC_RESIZE'] },
      { code: 'photoroom', name: 'Photoroom', configured: true, availability: 'available', operations: ['PHOTOROOM_REMOVE_BACKGROUND_SANDBOX'], safety_level: 'sandbox_only', provider_environment: 'sandbox', watermarked: true, non_publishable: true, quota_available: false, quota_remaining: 0, reason: 'canary 配额已耗尽' },
    ]);
    renderPage();
    expect(await screen.findByText('Photoroom · sandbox canary 不可用：canary 配额已耗尽')).toBeInTheDocument();
    await userEvent.click(screen.getByLabelText('处理器'));
    const option = await screen.findByText('Photoroom（sandbox-only）');
    expect(option.closest('.ant-select-item-option')).toHaveClass('ant-select-item-option-disabled');
    expect(screen.queryByText(/provider_environment=sandbox/)).not.toBeInTheDocument();
  });

  it('creates exactly one non-publishable Photoroom sandbox canary after all exact rights are recorded', async () => {
    const user = userEvent.setup();
    const asset = { id: 12, owner_id: 1, blob_id: 'blob-12', filename: 'bag.png', content_type: 'image/png', size_bytes: 3, sha256: 'a'.repeat(64), truth: 'actual' };
    const job = { id: 22, owner_id: 1, asset_id: 12, idempotency_key: 'photo-key', manifest_hash: 'manifest', operation: 'PHOTOROOM_AI_SHADOW_SANDBOX' as const, processor: 'photoroom', purpose: 'listing_main', channel: 'ozon', region: 'us', provider_environment: 'sandbox', max_cost: '0', currency: 'USD', sandbox: true, watermarked: true, non_publishable: true, width: 1200, height: 1200, format: 'png', status: 'pending' as const };
    vi.mocked(getImageProcessorCapabilities).mockResolvedValue([
      { code: 'deterministic', name: '凌镜标准处理', configured: true, availability: 'available', operations: ['DETERMINISTIC_RESIZE'] },
      { code: 'photoroom', name: 'Photoroom', configured: true, availability: 'available', operations: ['PHOTOROOM_REMOVE_BACKGROUND_SANDBOX', 'PHOTOROOM_WHITE_BACKGROUND_SANDBOX', 'PHOTOROOM_AI_SHADOW_SANDBOX'], safety_level: 'sandbox_only', provider_environment: 'sandbox', watermarked: true, non_publishable: true, quota_available: true, quota_remaining: 1 },
    ]);
    vi.mocked(uploadSourceImage).mockResolvedValue(asset);
    vi.mocked(createRightsGrant).mockResolvedValue({ id: 2, asset_id: 12, asset_sha256: asset.sha256, purpose: 'listing_main', jurisdiction: 'us', channel: 'ozon', provider: 'photoroom', region: 'us', grantor: 'owner', owner_verified: true, version: 1 });
    vi.mocked(createImageJob).mockResolvedValue(job);
    const { container } = renderPage();
    fireEvent.change(container.querySelector('input[type="file"]') as HTMLInputElement, { target: { files: [new File(['png'], 'bag.png', { type: 'image/png' })] } });
    expect(await screen.findByText('bag.png')).toBeInTheDocument();
    await user.click(screen.getByLabelText('处理器'));
    await user.click(await screen.findByText('Photoroom（sandbox-only）'));
    expect(await screen.findByText(/provider_environment=sandbox · region=US · max_cost=0 USD/)).toBeInTheDocument();
    await user.click(screen.getByLabelText('显式操作'));
    await user.click(await screen.findByText('添加 AI 阴影'));
    await user.type(screen.getByLabelText('用途'), 'listing_main');
    await user.type(screen.getByLabelText('销售渠道'), 'ozon');
    await user.type(screen.getByLabelText('授权人/权利人'), 'owner');
    await user.type(screen.getByLabelText('权利来源说明'), '原始拍摄文件与授权记录');
    await user.type(screen.getByLabelText('原图权利证据 SHA-256'), 'b'.repeat(64));
    await user.click(screen.getByRole('checkbox', { name: /can_copy/ }));
    await user.click(screen.getByRole('checkbox', { name: /can_modify/ }));
    await user.click(screen.getByRole('checkbox', { name: /can_third_party_ai/ }));
    await user.click(screen.getByRole('checkbox', { name: /can_cross_border/ }));
    await user.click(screen.getByRole('checkbox', { name: 'Owner 已核对精确证据' }));
    await user.click(screen.getByRole('button', { name: '保存原图处理权利' }));
    await waitFor(() => expect(createRightsGrant).toHaveBeenCalledWith(expect.objectContaining({ asset_id: 12, provider: 'photoroom', region: 'us', jurisdiction: 'us', can_copy: true, can_modify: true, can_third_party_ai: true, can_cross_border: true, can_commercial_publish: false })));
    await user.click(screen.getByRole('button', { name: '创建一次 Photoroom sandbox canary' }));
    await waitFor(() => expect(createImageJob).toHaveBeenCalledWith(expect.objectContaining({ operation: 'PHOTOROOM_AI_SHADOW_SANDBOX', processor: 'photoroom', region: 'us', max_cost: '0', currency: 'USD' })));
    expect(screen.getByText(/页面不接收 Provider 密钥/)).toBeInTheDocument();
  });

  it('never offers a ready Photoroom sandbox result to image sets or release', async () => {
    vi.mocked(listImageJobs).mockResolvedValue([{ id: 30, owner_id: 1, asset_id: 1, idempotency_key: 'sandbox-ready', manifest_hash: 'm', operation: 'PHOTOROOM_WHITE_BACKGROUND_SANDBOX', processor: 'photoroom', region: 'us', provider_environment: 'sandbox', max_cost: '0', currency: 'USD', sandbox: true, watermarked: true, non_publishable: true, version: 1, width: 1200, height: 1200, format: 'png', status: 'READY', output_blob_id: 'c'.repeat(64) }]);
    renderPage();
    expect(await screen.findByText(/Photoroom canary 永不进入图片集合或发布证明/)).toBeInTheDocument();
    expect(screen.queryByRole('checkbox', { name: '任务 30' })).not.toBeInTheDocument();
    expect(screen.queryByRole('button', { name: '创建图片集合' })).not.toBeInTheDocument();
  });

  it('creates an exact OpenAI scene task only after external-processing rights are recorded', async () => {
    const user = userEvent.setup();
    const asset = { id: 13, owner_id: 1, blob_id: 'blob-13', filename: 'shoe.png', content_type: 'image/png', size_bytes: 3, sha256: 'd'.repeat(64), truth: 'actual' };
    vi.mocked(getImageProcessorCapabilities).mockResolvedValue([
      { code: 'deterministic', name: '凌镜标准处理', configured: true, availability: 'available', operations: ['DETERMINISTIC_RESIZE'] },
      { code: 'openai', name: 'OpenAI Images', configured: true, availability: 'available', operations: ['OPENAI_IMAGE_EDIT'], safety_level: 'production_paid', provider_environment: 'production', region: 'us', watermarked: false, non_publishable: false },
    ]);
    vi.mocked(uploadSourceImage).mockResolvedValue(asset);
    vi.mocked(createRightsGrant).mockResolvedValue({ id: 3, asset_id: 13, asset_sha256: asset.sha256, purpose: 'scene_gallery', jurisdiction: 'us', channel: 'ozon', provider: 'openai', region: 'us', grantor: 'owner', owner_verified: true, version: 1 });
    vi.mocked(createImageJob).mockResolvedValue({ id: 23, owner_id: 1, asset_id: 13, sku_id: 1, recipe_key: 'recipe', recipe_version: 1, candidate_round: 1, idempotency_key: 'openai', manifest_hash: 'e'.repeat(64), operation: 'OPENAI_IMAGE_EDIT', processor: 'openai', purpose: 'scene_gallery', channel: 'ozon', region: 'us', provider_environment: 'production', max_cost: '0.20', currency: 'USD', version: 1, width: 1024, height: 1024, format: 'png', status: 'QUEUED' });
    const { container } = renderPage();
    fireEvent.change(container.querySelector('input[type="file"]') as HTMLInputElement, { target: { files: [new File(['png'], 'shoe.png', { type: 'image/png' })] } });
    expect(await screen.findByText('shoe.png')).toBeInTheDocument();
    await user.click(screen.getByLabelText('处理器'));
    await user.click(await screen.findByText('OpenAI Images（真实付费）'));
    await user.type(screen.getByLabelText('用途'), 'scene_gallery');
    await user.type(screen.getByLabelText('销售渠道'), 'ozon');
    await user.type(screen.getByLabelText('Provider 提示词（确定性处理可留空）'), 'Keep the exact shoe unchanged on a clean shelf');
    await user.type(screen.getByLabelText('授权人/权利人'), 'owner');
    await user.type(screen.getByLabelText('权利来源说明'), 'Owner original product photo');
    await user.type(screen.getByLabelText('原图权利证据 SHA-256'), 'f'.repeat(64));
    for (const name of [/can_copy/, /can_modify/, /can_third_party_ai/, /can_cross_border/]) await user.click(screen.getByRole('checkbox', { name }));
    await user.click(screen.getByRole('checkbox', { name: 'Owner 已核对精确证据' }));
    await user.click(screen.getByRole('button', { name: '保存原图处理权利' }));
    await waitFor(() => expect(createRightsGrant).toHaveBeenCalledWith(expect.objectContaining({ provider: 'openai', region: 'us', can_third_party_ai: true, can_cross_border: true })));
    await user.click(screen.getByRole('button', { name: '创建 OpenAI 场景候选' }));
    await waitFor(() => expect(createImageJob).toHaveBeenCalledWith(expect.objectContaining({ processor: 'openai', operation: 'OPENAI_IMAGE_EDIT', width: 1024, height: 1024, max_cost: '0.20', currency: 'USD', recipe: expect.objectContaining({ model: 'gpt-image-2', model_version: 'current', parameters: {} }) })));
  });

  it('requires exact input rights and sends the explicit task scope before execution', async () => {
    const user = userEvent.setup();
    const asset = { id: 1, owner_id: 1, blob_id: 'blob-1', filename: 'shoe.png', content_type: 'image/png', size_bytes: 3, sha256: 'a'.repeat(64), truth: 'actual' };
    const job = { id: 1, owner_id: 1, asset_id: 1, idempotency_key: 'create-key', manifest_hash: 'manifest', operation: 'DETERMINISTIC_RESIZE', processor: 'deterministic', purpose: 'listing_main', channel: 'ozon', region: 'local', width: 1200, height: 1200, format: 'png', status: 'QUEUED' as const };
    vi.mocked(uploadSourceImage).mockResolvedValue(asset);
    vi.mocked(createImageJob).mockResolvedValue(job);
    vi.mocked(createRightsGrant).mockResolvedValue({ id: 1, asset_id: 1, asset_sha256: asset.sha256, purpose: 'listing_main', jurisdiction: 'local', channel: 'ozon', provider: 'deterministic', region: 'local', grantor: 'owner', owner_verified: true, version: 1 });
    vi.mocked(executeImageJob).mockResolvedValue({ ...job, status: 'queued' });
    vi.mocked(listImageJobs).mockResolvedValueOnce([]).mockResolvedValue([job]);
    const { container } = renderPage();
    const input = container.querySelector('input[type="file"]') as HTMLInputElement;
    fireEvent.change(input, { target: { files: [new File(['png'], 'shoe.png', { type: 'image/png' })] } });
    expect(await screen.findByText('shoe.png')).toBeInTheDocument();
    expect(screen.getByRole('button', { name: '创建确定性图片任务' })).toBeDisabled();
    await user.type(screen.getByLabelText('用途'), 'listing_main');
    await user.type(screen.getByLabelText('销售渠道'), 'ozon');
    await user.type(screen.getByLabelText('授权人/权利人'), 'owner');
    await user.type(screen.getByLabelText('权利来源说明'), '原始拍摄文件与授权记录');
    await user.type(screen.getByLabelText('原图权利证据 SHA-256'), 'b'.repeat(64));
    await user.click(screen.getByRole('checkbox', { name: /can_copy/ }));
    await user.click(screen.getByRole('checkbox', { name: /can_modify/ }));
    await user.click(screen.getByRole('checkbox', { name: 'Owner 已核对精确证据' }));
    await user.click(screen.getByRole('button', { name: '保存原图处理权利' }));
    await waitFor(() => expect(createRightsGrant).toHaveBeenCalledWith(expect.objectContaining({ asset_id: 1, asset_sha256: asset.sha256, can_copy: true, can_modify: true, provider: 'deterministic', purpose: 'listing_main', channel: 'ozon', region: 'local' })));
    expect(screen.getByRole('button', { name: '创建确定性图片任务' })).toBeEnabled();
    await user.click(screen.getByRole('button', { name: '创建确定性图片任务' }));
    await waitFor(() => expect(createImageJob).toHaveBeenCalledWith(expect.objectContaining({ asset_id: 1, processor: 'deterministic', purpose: 'listing_main', channel: 'ozon', region: 'local' })));
    expect(await screen.findByText('1')).toBeInTheDocument();
    await user.click(screen.getByRole('button', { name: /执行/ }));
    expect(executeImageJob).toHaveBeenCalled();
    expect(vi.mocked(executeImageJob).mock.calls[0][0]).toBe(1);
  });

  it('surfaces backend failures instead of showing a mock success', async () => {
    vi.mocked(listImageJobs).mockRejectedValue(new Error('图片服务不可用'));
    renderPage();
    expect(await screen.findByText('图片服务不可用')).toBeInTheDocument();
  });

  it('shows fail-closed rights, five-axis review and cost controls for a ready output', async () => {
    vi.mocked(listImageJobs).mockResolvedValue([{ id: 9, owner_id: 1, asset_id: 1, idempotency_key: 'ready', manifest_hash: 'm', operation: 'DETERMINISTIC_RESIZE', processor: 'deterministic', version: 1, width: 1200, height: 1200, format: 'png', status: 'READY', output_blob_id: 'a'.repeat(64) }]);
    renderPage();
    expect(await screen.findByText('3. 权利、五类审核与成本')).toBeInTheDocument();
    expect(await screen.findByText('A. 精确图片权利授权')).toBeInTheDocument();
    expect(screen.getByText('B. 五类逐项审核')).toBeInTheDocument();
    expect(screen.getByText('C. 成本记录（确定性处理可选，外部付费处理必填）')).toBeInTheDocument();
    expect(screen.getByText(/必须绑定精确输出 SHA-256/)).toBeInTheDocument();
  });

  it('restores a saved recipe and shows its candidate feedback statistics', async () => {
    vi.mocked(listImageJobs).mockResolvedValue([{ id: 19, owner_id: 1, asset_id: 1, sku_id: 5, recipe_key: 'recipe-5', recipe_version: 2, candidate_round: 2, recipe_hash: 'd'.repeat(64), recipe_manifest: { scene_structure: 'bright kitchen', prompt: 'keep product unchanged', model: 'deterministic', model_version: '1', parameters: {}, must_not_change: ['color'] }, idempotency_key: 'ready-recipe', manifest_hash: 'm', operation: 'DETERMINISTIC_RESIZE', processor: 'deterministic', purpose: 'scene_gallery', channel: 'ozon', region: 'local', version: 1, width: 1200, height: 1200, format: 'png', status: 'READY', output_blob_id: 'a'.repeat(64) }]);
    vi.mocked(getRecipeSummary).mockResolvedValue({ recipe_key: 'recipe-5', sku_id: 5, purpose: 'scene_gallery', channel: 'ozon', latest_recipe_version: 2, candidates: 3, selected: 2, rejected: 1, rework_requested: 1, acceptance_rate: 2 / 3, review_seconds: 60, production_seconds: 120, rework_rounds: 1, actual_cost: '0.3000', currency: 'USD' });
    renderPage();
    expect(await screen.findByText('D. 候选选择、拒绝与返工')).toBeInTheDocument();
		expect(await screen.findByText('候选 3')).toBeInTheDocument();
		expect(getRecipeSummary).toHaveBeenCalledWith('recipe-5', 5);
		expect(screen.getByText('采用 2')).toBeInTheDocument();
    expect(screen.getByText('返工轮次 1')).toBeInTheDocument();
    expect(screen.getByRole('button', { name: '载入为返工配方' })).toBeEnabled();
  });

  it('explains release proof boundaries and does not offer attestation consumption', async () => {
    renderPage();
    expect(await screen.findByText('发布证明管理')).toBeInTheDocument();
    expect(screen.getByText('签发证明不等于已经发布')).toBeInTheDocument();
    expect(screen.getByRole('button', { name: '签发发布证明（不会发布）' })).toBeDisabled();
    expect(screen.queryByRole('button', { name: /消费证明/ })).not.toBeInTheDocument();
    expect(screen.getByText(/本页面不提供消费按钮/)).toBeInTheDocument();
  });

  it('explains budget truth levels and fails closed without an active policy', async () => {
    renderPage();
    expect(await screen.findByText('付费图片预算与预占')).toBeInTheDocument();
    expect(screen.getByText(/estimated 是事前估算/)).toBeInTheDocument();
    expect(screen.getByText(/spent 只能来自外部账单对账/)).toBeInTheDocument();
    expect(screen.getByText('当前没有生效中的预算政策')).toBeInTheDocument();
    expect(screen.getByRole('button', { name: '追加账单对账' })).toBeDisabled();
  });

  it('shows reserved exposure and remaining allowance before paid approval', async () => {
    vi.mocked(listImageBudgetPolicies).mockResolvedValue([{ id: 4, owner_id: 1, currency: 'USD', period_start: '2026-01-01T00:00:00Z', period_end: '2030-01-01T00:00:00Z', total_amount: '10.0000', idempotency_key: 'policy-1' }]);
    vi.mocked(listImageBudgetReservations).mockResolvedValue([{ id: 8, owner_id: 1, policy_id: 4, approval_id: 2, task_id: 9, task_version: 1, manifest_hash: 'a'.repeat(64), provider: 'openai', currency: 'USD', reserved_amount: '2.5000', state: 'reserved' }]);
    renderPage();
    expect(await screen.findByText(/reserved（已预占）2.5/)).toBeInTheDocument();
    expect(screen.getByText(/预占口径剩余 7.5 USD/)).toBeInTheDocument();
    expect(screen.getByText(/spent（已花费）须以外部账单对账记录为准/)).toBeInTheDocument();
  });

  it('does not allow cancellation of a claimed reservation', async () => {
    vi.mocked(listImageBudgetReservations).mockResolvedValue([{ id: 8, owner_id: 1, policy_id: 4, approval_id: 2, task_id: 9, task_version: 1, manifest_hash: 'a'.repeat(64), provider: 'openai', currency: 'USD', reserved_amount: '2.5000', state: 'claimed' }]);
    renderPage();
    const select = await screen.findByRole('combobox', { name: '选择预占' });
    await userEvent.click(select);
    await userEvent.click(await screen.findByText('#8 · claimed · 2.5000 USD'));
    expect(screen.getByRole('button', { name: '取消未claim预占' })).toBeDisabled();
  });

  it('shows the complete frozen recipe before a second explicit paid confirmation', async () => {
    const user = userEvent.setup();
    const job = { id: 31, owner_id: 1, asset_id: 13, sku_id: 5, recipe_key: 'shoe-scene', recipe_version: 2, recipe_hash: 'd'.repeat(64), recipe_manifest: { reference_asset_ids: [13], scene_structure: 'bright shelf at eye level', prompt: 'Keep the exact blue shoe unchanged on a bright shelf', negative_prompt: 'no extra accessories', model: 'gpt-image-2', model_version: 'current', parameters: {}, must_not_change: ['blue color', 'logo', 'sole shape'] }, candidate_round: 2, idempotency_key: 'openai-31', manifest_hash: 'e'.repeat(64), operation: 'OPENAI_IMAGE_EDIT', processor: 'openai', purpose: 'scene_gallery', channel: 'ozon', region: 'us', provider_environment: 'production', max_cost: '0.20', currency: 'USD', version: 3, width: 1024, height: 1024, format: 'png', status: 'QUEUED' as const };
    vi.mocked(getImageProcessorCapabilities).mockResolvedValue([{ code: 'deterministic', name: '凌镜标准处理', configured: true, operations: ['DETERMINISTIC_RESIZE'] }, { code: 'openai', name: 'OpenAI Images', configured: true, availability: 'available', operations: ['OPENAI_IMAGE_EDIT'], safety_level: 'production_paid', provider_environment: 'production', region: 'us', watermarked: false, non_publishable: false }]);
    vi.mocked(listImageJobs).mockResolvedValue([job]);
    vi.mocked(approveImageExecution).mockResolvedValue({ id: 1 });
    vi.mocked(executeImageJob).mockResolvedValue({ id: 'attempt', status: 'QUEUED' });
    renderPage();
    await user.click(await screen.findByRole('button', { name: /查看配方并批准付费执行/ }));
    expect(screen.getByRole('dialog')).toBeInTheDocument();
    expect(screen.getByText('确认真实付费执行')).toBeInTheDocument();
    expect(screen.getByText('Keep the exact blue shoe unchanged on a bright shelf')).toBeInTheDocument();
    expect(screen.getAllByText(/gpt-image-2 \/ current/)).not.toHaveLength(0);
    expect(screen.getAllByText(/blue color；logo；sole shape/)).not.toHaveLength(0);
    expect(approveImageExecution).not.toHaveBeenCalled();
    expect(executeImageJob).not.toHaveBeenCalled();
    await user.click(screen.getByRole('button', { name: '确认批准、预占并执行' }));
    await waitFor(() => expect(executeImageJob).toHaveBeenCalledWith(31));
    expect(vi.mocked(approveImageExecution).mock.invocationCallOrder[0]).toBeLessThan(vi.mocked(executeImageJob).mock.invocationCallOrder[0]);
  });

  it('blocks retries and explains exact recovery for uppercase reconciliation state', async () => {
    const user = userEvent.setup();
    vi.mocked(listImageJobs).mockResolvedValue([{ id: 40, owner_id: 1, asset_id: 1, sku_id: 5, recipe_key: 'r', recipe_version: 1, recipe_hash: 'd'.repeat(64), recipe_manifest: { reference_asset_ids: [1], scene_structure: 'shelf', prompt: 'keep exact product', model: 'gpt-image-2', model_version: 'current', parameters: {}, must_not_change: ['shape'] }, idempotency_key: 'reconcile', manifest_hash: 'e'.repeat(64), operation: 'OPENAI_IMAGE_EDIT', processor: 'openai', purpose: 'scene_gallery', channel: 'ozon', region: 'us', provider_environment: 'production', max_cost: '0.20', currency: 'USD', version: 1, width: 1024, height: 1024, format: 'png', status: 'RECONCILE_REQUIRED', error_code: 'INTERNAL_DISPATCH_OUTCOME_UNKNOWN' }]);
    vi.mocked(listImageBudgetReservations).mockResolvedValue([{ id: 12, owner_id: 1, policy_id: 4, approval_id: 2, task_id: 40, task_version: 1, manifest_hash: 'e'.repeat(64), provider: 'openai', currency: 'USD', reserved_amount: '0.20', state: 'claimed' }]);
    renderPage();
    expect(await screen.findByText('待对账')).toBeInTheDocument();
    expect(screen.getByText(/禁止重试/)).toBeInTheDocument();
    expect(screen.queryByRole('button', { name: /付费执行/ })).not.toBeInTheDocument();
    expect(screen.getByText('确认未扣费（只用于待对账任务）')).toBeInTheDocument();
    await user.click(screen.getByRole('combobox', { name: '选择预占' }));
    await user.click(await screen.findByText('#12 · claimed · 0.20 USD'));
    expect(screen.getByRole('checkbox', { name: /已扣费且没有可恢复输出/ })).toBeEnabled();
  });
});

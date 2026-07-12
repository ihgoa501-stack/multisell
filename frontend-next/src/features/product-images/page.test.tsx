import { QueryClient, QueryClientProvider } from '@tanstack/react-query';
import { fireEvent, render, screen, waitFor } from '@testing-library/react';
import userEvent from '@testing-library/user-event';
import { beforeEach, describe, expect, it, vi } from 'vitest';

vi.mock('next/navigation', () => ({ usePathname: () => '/product-images' }));
vi.mock('./api', () => ({
  getImageProcessorCapabilities: vi.fn(), listImageJobs: vi.fn(), uploadSourceImage: vi.fn(),
  createImageJob: vi.fn(), executeImageJob: vi.fn(), fetchImageOutput: vi.fn(),
  createRightsGrant: vi.fn(), createFiveAxisReview: vi.fn(), createCostEntry: vi.fn(),
  createProductImageSet: vi.fn(), freezeProductImageSet: vi.fn(),
  newImageIdempotencyKey: vi.fn(() => 'test-idempotency-key'),
}));

import ProductImagesPage from '@/app/(main)/product-images/page';
import { createImageJob, executeImageJob, getImageProcessorCapabilities, listImageJobs, uploadSourceImage } from './api';

function renderPage() {
  const client = new QueryClient({ defaultOptions: { queries: { retry: false }, mutations: { retry: false } } });
  return render(<QueryClientProvider client={client}><ProductImagesPage /></QueryClientProvider>);
}

describe('product image workspace', () => {
  beforeEach(() => {
    vi.clearAllMocks();
    vi.mocked(listImageJobs).mockResolvedValue([]);
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
    expect(screen.getByText('Photoroom · 未配置')).toBeInTheDocument();
    expect(screen.getByText('OpenAI Images · 适配器合同已实现，但付费执行未启用（需 Owner 批准门禁）')).toBeInTheDocument();
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
    expect(screen.getByText('OpenAI Images · 适配器合同已实现，但付费执行未启用（需 Owner 批准门禁）')).toBeInTheDocument();
    expect(screen.queryByText('OpenAI Images · 可用')).not.toBeInTheDocument();
    expect(screen.getByRole('button', { name: '创建确定性图片任务' })).toBeDisabled();
  });

  it('uploads an original, creates a task and then executes it', async () => {
    const user = userEvent.setup();
    const asset = { id: 1, owner_id: 1, blob_id: 'blob-1', filename: 'shoe.png', content_type: 'image/png', size_bytes: 3, sha256: 'hash-1', truth: 'actual' };
    const job = { id: 1, owner_id: 1, asset_id: 1, idempotency_key: 'create-key', manifest_hash: 'manifest', operation: 'DETERMINISTIC_RESIZE', width: 1200, height: 1200, format: 'png', status: 'pending' as const };
    vi.mocked(uploadSourceImage).mockResolvedValue(asset);
    vi.mocked(createImageJob).mockResolvedValue(job);
    vi.mocked(executeImageJob).mockResolvedValue({ ...job, status: 'queued' });
    vi.mocked(listImageJobs).mockResolvedValueOnce([]).mockResolvedValue([job]);
    const { container } = renderPage();
    const input = container.querySelector('input[type="file"]') as HTMLInputElement;
    fireEvent.change(input, { target: { files: [new File(['png'], 'shoe.png', { type: 'image/png' })] } });
    expect(await screen.findByText('shoe.png')).toBeInTheDocument();
    await user.click(screen.getByRole('button', { name: '创建确定性图片任务' }));
    await waitFor(() => expect(createImageJob).toHaveBeenCalledWith(expect.objectContaining({ asset_id: 1 })));
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
});

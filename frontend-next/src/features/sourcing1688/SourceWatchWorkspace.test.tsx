import { QueryClient, QueryClientProvider } from '@tanstack/react-query';
import { fireEvent, render, screen, waitFor } from '@testing-library/react';
import { App } from 'antd';
import { beforeEach, describe, expect, it, vi } from 'vitest';
import apiClient from '@/lib/api-client';
import SourceWatchWorkspace, { watchSeverity } from './SourceWatchWorkspace';

vi.mock('@/lib/api-client', () => ({ default: { get: vi.fn(), put: vi.fn(), post: vi.fn() } }));

function renderWorkspace() {
  const client = new QueryClient({ defaultOptions: { queries: { retry: false } } });
  return render(<QueryClientProvider client={client}><App><SourceWatchWorkspace sourceID={12} sourceURL="https://detail.1688.com/offer/9001.html" open onClose={vi.fn()} /></App></QueryClientProvider>);
}

describe('SourceWatchWorkspace', () => {
  beforeEach(() => {
    vi.clearAllMocks();
    vi.stubGlobal('crypto', { randomUUID: () => 'fixed-watch-request' });
	    vi.mocked(apiClient.get).mockImplementation(async (path: string) => {
	      if (path.endsWith('/watch')) return { code: 0, message: 'ok', data: { id: 1, enabled: true, updated_at: '2026-07-13T00:00:00Z' } } as never;
	      if (path.endsWith('/refresh-runs')) return { code: 0, message: 'ok', data: [] } as never;
      if (path.endsWith('/identity-history')) return { code: 0, message: 'ok', data: { snapshots: [{ id: 2, collected_at: '2026-07-13T02:00:00Z' }, { id: 1, collected_at: '2026-07-13T01:00:00Z' }] } } as never;
      return { code: 0, message: 'ok', data: [{ id: 9, change_type: 'offer_state', previous_snapshot_id: 1, current_snapshot_id: 2, before_value: 'online', after_value: 'delisted', content_hash: 'a'.repeat(64), created_at: '2026-07-13T02:00:00Z' }] } as never;
    });
    vi.mocked(apiClient.post).mockResolvedValue({ code: 0, message: 'ok', data: { id: 5, request_id: 'watch_12_fixed-watch-request', status: 'pending_browser', alert_count: 0, created_at: '2026-07-13T03:00:00Z' } } as never);
  });

  it('explains browser-required refresh and renders observation alerts without draft actions', async () => {
    renderWorkspace();
    expect(await screen.findByText('监控不会自动登录或抓取1688')).toBeInTheDocument();
    expect(screen.getByText('当前后端未提供已读/未读状态')).toBeInTheDocument();
    expect(await screen.findByText(/online.*delisted/)).toBeInTheDocument();
    expect(screen.getByText('高')).toBeInTheDocument();
    expect(screen.queryByText(/不会覆盖已审核草稿/)).not.toBeNull();
    expect(screen.queryByRole('button', { name: /改价|改库存|覆盖草稿/ })).not.toBeInTheDocument();
  });

	  it('creates a pending browser run only after explicit Owner action', async () => {
    renderWorkspace();
    const button = await screen.findByRole('button', { name: '请求浏览器补采' });
    await waitFor(() => expect(button).toBeEnabled());
    fireEvent.click(button);
    await waitFor(() => expect(apiClient.post).toHaveBeenCalledWith('/v1/sourcing-1688/12/watch/refresh-runs', { request_id: 'watch_12_fixed-watch-request' }));
    expect(await screen.findByText('等待Owner浏览器补采')).toBeInTheDocument();
	  });

	  it('restores a pending run and only offers snapshots created after that run', async () => {
	    vi.mocked(apiClient.get).mockImplementation(async (path: string) => {
	      if (path.endsWith('/watch')) return { code: 0, message: 'ok', data: { id: 1, enabled: true } } as never;
	      if (path.endsWith('/refresh-runs')) return { code: 0, message: 'ok', data: [{ id: 5, request_id: 'watch-5', status: 'pending_browser', alert_count: 0, created_at: '2026-07-13T01:30:00Z' }] } as never;
	      if (path.endsWith('/identity-history')) return { code: 0, message: 'ok', data: { snapshots: [{ id: 2, collected_at: '2026-07-13T02:00:00Z' }, { id: 1, collected_at: '2026-07-13T01:00:00Z' }] } } as never;
	      return { code: 0, message: 'ok', data: [] } as never;
	    });
	    renderWorkspace();
	    expect(await screen.findByText('等待Owner浏览器补采')).toBeInTheDocument();
	    fireEvent.mouseDown(screen.getByLabelText('较新快照'));
	    expect(await screen.findByText(/#2/)).toBeInTheDocument();
	    expect(screen.queryByText((text) => text.startsWith('#1 ·'))).not.toBeInTheDocument();
	  });

  it('assigns deterministic display severity without pretending it is backend unread state', () => {
    expect(watchSeverity('offer_state', 'delisted').label).toBe('高');
    expect(watchSeverity('supplier', 'new')).toEqual({ color: 'red', label: '高' });
    expect(watchSeverity('quoted_inventory', { sku: 0 }).label).toBe('中');
    expect(watchSeverity('price', 12).label).toBe('关注');
  });
});

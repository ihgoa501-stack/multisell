import { act, fireEvent, render, screen, waitFor } from '@testing-library/react';
import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest';

vi.mock('@/lib/api-client', () => ({
  default: { get: vi.fn(), post: vi.fn(), delete: vi.fn() },
}));

import apiClient from '@/lib/api-client';
import ExtensionPairingPage, { friendlyPairingError, PLUGIN_RESPONSE_TIMEOUT_MS } from './page';

const pairing = {
  pairing_id: 7,
  nonce: 'nonce-7',
  environment: 'production' as const,
  expires_at: '2026-07-13T08:00:00Z',
};

describe('1688采集助手连接页', () => {
  beforeEach(() => {
    vi.resetAllMocks();
    vi.mocked(apiClient.get).mockResolvedValue({ code: 0, message: 'ok', data: [] });
    vi.mocked(apiClient.post).mockResolvedValue({ code: 0, message: 'ok', data: pairing });
  });

  afterEach(() => {
    vi.useRealTimers();
  });

  it('把鉴权和网络错误转成Owner可操作的中文提示', () => {
    expect(friendlyPairingError({ status: 401, message: 'HTTP 401' })).toEqual({
      title: '登录已失效', description: '请重新登录凌镜，回到本页后再次连接。',
    });
    expect(friendlyPairingError(new Error('HTTP 403')).description).not.toContain('HTTP');
    expect(friendlyPairingError({ status: 0, message: 'Failed to fetch' })).toEqual({
      title: '无法联系凌镜服务', description: '请检查网络并刷新本页，然后重试；本次没有连接成功。',
    });
  });

  it('插件无响应时停止等待，并提供重新连接和取消', async () => {
    vi.useFakeTimers();
    render(<ExtensionPairingPage />);

    await act(async () => {
      fireEvent.click(screen.getByRole('button', { name: '连接1688采集助手' }));
      await Promise.resolve();
    });
    expect(screen.getByText('正在检测这台浏览器中的采集助手……')).toBeInTheDocument();

    act(() => { vi.advanceTimersByTime(PLUGIN_RESPONSE_TIMEOUT_MS); });
    expect(screen.getByText('没有检测到1688采集助手')).toBeInTheDocument();
    expect(screen.getByRole('button', { name: '重新连接' })).toBeInTheDocument();
    expect(screen.getByRole('button', { name: /取\s*消/ })).toBeInTheDocument();
  });

  it('插件响应后展示待确认设备、环境和权限', async () => {
    vi.mocked(apiClient.get)
      .mockResolvedValueOnce({ code: 0, message: 'ok', data: [] })
      .mockResolvedValueOnce({
        code: 0,
        message: 'ok',
        data: { pairing_id: 7, device_id: 'device-7', extension_id: 'ext-7', browser_label: 'Chrome · Mac' },
      });
    render(<ExtensionPairingPage />);
    fireEvent.click(screen.getByRole('button', { name: '连接1688采集助手' }));

    await waitFor(() => expect(apiClient.post).toHaveBeenCalledWith('/v1/auth/extension-pairings', { environment: 'development' }));
    await act(async () => {
      window.dispatchEvent(new MessageEvent('message', {
        source: window,
        origin: window.location.origin,
        data: { source: 'lingmirror-extension', type: 'LINGMIRROR_EXTENSION_PAIRING_RESULT', ok: true },
      }));
    });

    expect(await screen.findByText('Chrome · Mac')).toBeInTheDocument();
    expect(screen.getByText('生产')).toBeInTheDocument();
    expect(screen.getByText('保存1688商品到私人采集箱、读取本次保存结果')).toBeInTheDocument();
    expect(screen.getByRole('button', { name: '确认连接' })).toBeInTheDocument();
  });

  it('已连接列表使用中文环境和权限', async () => {
    vi.mocked(apiClient.get).mockResolvedValue({
      code: 0,
      message: 'ok',
      data: [{ device_id: 'device-9', browser_label: 'Chrome · Mac', environment: 'production', scope: 'sourcing1688.collect' }],
    });
    render(<ExtensionPairingPage />);

    expect(await screen.findByText('Chrome · Mac')).toBeInTheDocument();
    expect(screen.getByText('设备 device-9 · 权限：保存1688私人收藏')).toBeInTheDocument();
  });
});

import { beforeEach, describe, expect, it, vi } from 'vitest';

import { ApiClient } from '../api-client';

describe('ApiClient sourcing1688 errors', () => {
  beforeEach(() => {
    vi.restoreAllMocks();
  });

  it('surfaces the backend business error message', async () => {
    const body = { code: 502, message: 'TAB_NOT_FOUND: 请先打开完全相同的 1688 商品页面' };
    vi.stubGlobal('fetch', vi.fn().mockResolvedValue({
      ok: false,
      status: 502,
      statusText: 'Bad Gateway',
      clone: () => ({ json: () => Promise.resolve(body) }),
    }));

    const client = new ApiClient('http://test.api');
    await expect(client.post('/v1/sourcing-1688/fetch', {})).rejects.toThrow(body.message);
  });
});

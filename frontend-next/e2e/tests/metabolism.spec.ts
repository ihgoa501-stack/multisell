import { test, expect } from '@playwright/test';

/**
 * Metabolism (代谢管理) — E2E tests for the M1 scoring dashboard.
 *
 * Uses Playwright route interception to mock API responses,
 * so no running backend is needed. The AuthGuard is bypassed
 * by injecting a fake token in localStorage before navigation.
 */

const API_PREFIX = '**/api/v1/metabolism';
const AUTH_API = '**/api/v1/auth/login';
const RBAC_API = '**/api/v1/rbac/**';

/** Sample metabolism logs for a "has data" scenario. */
const SAMPLE_LOGS = [
  {
    id: 1,
    event_id: 101,
    source: 'order',
    total_score: 0.85,
    impact_score: 0.9,
    ref_score: 0.8,
    freshness_score: 0.85,
    semantic_score: 0.88,
    sem_skipped: false,
    excretable: true,
    reason: '低价值订单事件',
    created_at: '2026-06-26T10:00:00Z',
  },
  {
    id: 2,
    event_id: 102,
    source: 'inventory',
    total_score: 0.25,
    impact_score: 0.3,
    ref_score: 0.2,
    freshness_score: 0.15,
    semantic_score: 0.35,
    sem_skipped: false,
    excretable: false,
    reason: '活跃SKU，需要保留事件记录',
    created_at: '2026-06-26T09:00:00Z',
  },
  {
    id: 3,
    event_id: 103,
    source: 'order',
    total_score: 0.62,
    impact_score: 0.7,
    ref_score: 0.5,
    freshness_score: 0.55,
    semantic_score: 0.72,
    sem_skipped: false,
    excretable: false,
    reason: '中风险事件，需人工复核',
    created_at: '2026-06-25T18:00:00Z',
  },
];

/** Set auth tokens so AuthGuard passes, and stub RBAC + login endpoints. */
async function setupAuthStubs(page: import('@playwright/test').Page) {
  // Inject into localStorage BEFORE any page JS runs so AuthGuard finds a token.
  await page.addInitScript(() => {
    localStorage.setItem('token', 'e2e-test-token');
    localStorage.setItem('refresh_token', 'e2e-refresh-token');
    localStorage.setItem(
      'user',
      JSON.stringify({
        id: '1',
        email: 'e2e@lingmirror.test',
        name: 'E2E Test',
        roles: ['admin'],
      }),
    );
  });

  // Intercept RBAC permission check (called by AuthGuard on mount).
  await page.route(RBAC_API, async (route) => {
    await route.fulfill({
      status: 200,
      contentType: 'application/json',
      body: JSON.stringify({ code: 0, message: 'ok', data: { permissions: ['*'] } }),
    });
  });

  // Intercept login (used if navigation ever redirects to /login).
  await page.route(AUTH_API, async (route) => {
    await route.fulfill({
      status: 200,
      contentType: 'application/json',
      body: JSON.stringify({
        code: 0,
        message: 'ok',
        data: {
          access_token: 'e2e-test-token',
          refresh_token: 'e2e-refresh-token',
          user: { id: '1', email: 'e2e@lingmirror.test', name: 'E2E Test' },
        },
      }),
    });
  });
}

test.describe('代谢管理 (Metabolism Dashboard)', () => {
  test.beforeEach(async ({ page }) => {
    await setupAuthStubs(page);
  });

  test('renders stat cards, action bar, and log table with data', async ({
    page,
  }) => {
    // Mock the metabolism logs list endpoint.
    await page.route(`${API_PREFIX}`, async (route) => {
      if (route.request().method() !== 'GET') {
        await route.fallback();
        return;
      }
      await route.fulfill({
        status: 200,
        contentType: 'application/json',
        body: JSON.stringify({
          code: 0,
          message: 'ok',
          data: { data: SAMPLE_LOGS, total: SAMPLE_LOGS.length },
        }),
      });
    });

    await page.goto('/metabolism');
    await page.waitForLoadState('networkidle');

    // Page heading.
    await expect(page.getByText('代谢管理')).toBeVisible();
    await expect(
      page.getByText('M1 排泄系统 — 四维评分引擎状态监控'),
    ).toBeVisible();

    // --- Stat cards ---

    // 已评分事件 card shows total = 3.
    await expect(page.getByText('已评分事件')).toBeVisible();
    await expect(page.getByText('3').first()).toBeVisible();

    // 可排泄率 → 1/3 = 33.3%.
    await expect(page.getByText('可排泄率')).toBeVisible();
    await expect(page.getByText('33.3%')).toBeVisible();

    // 平均分 card.
    await expect(page.getByText('平均分')).toBeVisible();

    // 评分分布 stat shows low/mid/high counts (1 low, 1 mid, 1 high).
    await expect(page.getByText(/1\/1\/1/)).toBeVisible();

    // --- Action bar ---
    await expect(page.getByText('Dry-run 模式:')).toBeVisible();
    await expect(
      page.getByRole('button', { name: /立即触发评分/i }),
    ).toBeVisible();
    await expect(page.getByText('仅评分，不执行排泄')).toBeVisible();

    // --- Table ---
    const table = page.locator('.ant-table');
    await expect(table).toBeVisible();

    // Table contains the mock rows.
    await expect(page.getByText('order')).toBeVisible();
    await expect(page.getByText('inventory')).toBeVisible();
    await expect(page.getByText('低价值订单事件')).toBeVisible();
    await expect(
      page.getByText('活跃SKU，需要保留事件记录'),
    ).toBeVisible();
    await expect(page.getByText('中风险事件，需人工复核')).toBeVisible();
  });

  test('triggers M1 dry-run and shows success message', async ({ page }) => {
    // Mock GET (initial load) and POST (dry-run trigger).
    await page.route(`${API_PREFIX}`, async (route) => {
      const method = route.request().method();
      if (method === 'GET') {
        await route.fulfill({
          status: 200,
          contentType: 'application/json',
          body: JSON.stringify({
            code: 0,
            message: 'ok',
            data: { data: SAMPLE_LOGS, total: SAMPLE_LOGS.length },
          }),
        });
      } else if (method === 'POST') {
        await route.fulfill({
          status: 200,
          contentType: 'application/json',
          body: JSON.stringify({
            code: 0,
            message: 'ok',
            data: { message: 'dry-run completed' },
          }),
        });
      } else {
        await route.fallback();
      }
    });

    await page.goto('/metabolism');
    await page.waitForLoadState('networkidle');

    // Click the trigger button.
    await page.getByRole('button', { name: /立即触发评分/i }).click();

    // Ant Design message.success shows "M1 代谢评分已触发".
    await expect(page.getByText('M1 代谢评分已触发')).toBeVisible({
      timeout: 5000,
    });
  });

  test('shows error when dry-run API fails', async ({ page }) => {
    await page.route(`${API_PREFIX}`, async (route, request) => {
      if (request.method() === 'GET') {
        await route.fulfill({
          status: 200,
          contentType: 'application/json',
          body: JSON.stringify({
            code: 0,
            message: 'ok',
            data: { data: SAMPLE_LOGS, total: SAMPLE_LOGS.length },
          }),
        });
      } else {
        // POST returns server error.
        await route.fulfill({
          status: 500,
          contentType: 'application/json',
          body: JSON.stringify({ code: 500, message: '评分引擎内部错误' }),
        });
      }
    });

    await page.goto('/metabolism');
    await page.waitForLoadState('networkidle');

    await page.getByRole('button', { name: /立即触发评分/i }).click();

    // The mutation onError handler calls message.error('触发失败').
    await expect(page.getByText('触发失败')).toBeVisible({ timeout: 5000 });
  });

  test('dry-run switch toggles between dry-run and live mode labels', async ({
    page,
  }) => {
    await page.route(`${API_PREFIX}`, async (route) => {
      if (route.request().method() !== 'GET') {
        await route.fallback();
        return;
      }
      await route.fulfill({
        status: 200,
        contentType: 'application/json',
        body: JSON.stringify({
          code: 0,
          message: 'ok',
          data: { data: SAMPLE_LOGS, total: SAMPLE_LOGS.length },
        }),
      });
    });

    await page.goto('/metabolism');
    await page.waitForLoadState('networkidle');

    // Default: dry-run mode.
    await expect(page.getByText('仅评分，不执行排泄')).toBeVisible();

    // Toggle the Ant Design Switch.
    const toggle = page.locator('.ant-switch');
    await expect(toggle).toBeVisible();
    await toggle.click();

    // Now should show live mode label.
    await expect(page.getByText('评分 + 实际排泄')).toBeVisible();

    // Toggle back to dry-run.
    await toggle.click();
    await expect(page.getByText('仅评分，不执行排泄')).toBeVisible();
  });

  test('shows empty state when no metabolism logs exist', async ({ page }) => {
    await page.route(`${API_PREFIX}`, async (route) => {
      if (route.request().method() !== 'GET') {
        await route.fallback();
        return;
      }
      await route.fulfill({
        status: 200,
        contentType: 'application/json',
        body: JSON.stringify({
          code: 0,
          message: 'ok',
          data: { data: [], total: 0 },
        }),
      });
    });

    await page.goto('/metabolism');
    await page.waitForLoadState('networkidle');

    // Stats show 0.
    await expect(page.getByText('已评分事件')).toBeVisible();
    await expect(page.getByText('0').first()).toBeVisible();
    await expect(page.getByText('可排泄率')).toBeVisible();
    await expect(page.getByText('0.0%')).toBeVisible();

    // Ant Design empty-state indicator.
    await expect(page.locator('.ant-empty')).toBeVisible();
  });

  test('waste marketplace route returns 404 (not yet implemented)', async ({
    page,
  }) => {
    // Navigate to the unimplemented waste page.
    await page.goto('/metabolism/waste');

    // Next.js App Router's built-in 404 page.
    await expect(
      page
        .getByText('404')
        .or(page.getByText(/This page could not be found/i))
        .or(page.getByText(/page.*not.*found/i)),
    ).toBeVisible({ timeout: 10000 });
  });
});

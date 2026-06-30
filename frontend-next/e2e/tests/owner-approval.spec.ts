import { test, expect } from '@playwright/test';

/**
 * Owner Approval E2E tests.
 *
 * Covers the Owner (经营总控台) approval flow:
 *   1. Page renders with risk summary stat cards
 *   2. Agent suggestions table displays with correct status tags
 *   3. Approve a suggestion → modal → confirm → success message
 *
 * All tests use Playwright route interception for API mocking
 * and inject a fake JWT token to bypass AuthGuard.
 */

const FAKE_TOKEN = 'e2e-owner-test-token';

/** A sample suggestion that can be approved. */
const APPROVABLE_SUGGESTION = {
  id: 1,
  product_id: 101,
  product_title: 'Wireless Bluetooth Earbuds',
  agent_source: 'A8',
  suggestion: '建议上架',
  decision: 'list',
  reason: 'High demand, good profit margin, compliant with platform policies',
  confidence: 0.85,
  risk_level: 'low',
  created_at: '2026-06-29T10:00:00Z',
  listing_task_id: 42,
};

const CAUTIOUS_SUGGESTION = {
  id: 2,
  product_id: 102,
  product_title: 'Premium Phone Case',
  agent_source: 'A8',
  suggestion: '谨慎上架',
  decision: 'cautious',
  reason: 'Moderate demand, acceptable margin, some compliance risk',
  confidence: 0.65,
  risk_level: 'medium',
  created_at: '2026-06-28T10:00:00Z',
  listing_task_id: null,
};

const SKIP_SUGGESTION = {
  id: 3,
  product_id: 103,
  product_title: 'Unknown Brand Charger',
  agent_source: 'A8',
  suggestion: '不建议上架',
  decision: 'skip',
  reason: 'Low quality supplier, high return rate expected',
  confidence: 0.3,
  risk_level: 'high',
  created_at: '2026-06-27T10:00:00Z',
  listing_task_id: null,
};

test.describe('Owner Approval — 经营总控台', () => {
  test.beforeEach(async ({ page }) => {
    await page.addInitScript((token: string) => {
      localStorage.setItem('token', token);
      localStorage.setItem('refresh_token', token);
    }, FAKE_TOKEN);

    // Mock all API calls
    await page.route('**/api/**', async (route) => {
      const url = route.request().url();

      if (url.includes('/v1/rbac/')) {
        return route.fulfill({
          status: 200, contentType: 'application/json',
          body: JSON.stringify({ code: 0, message: 'ok', data: { permissions: ['*'] } }),
        });
      }

      if (url.includes('/v1/owner/risk-summary')) {
        return route.fulfill({
          status: 200, contentType: 'application/json',
          body: JSON.stringify({
            code: 0, message: 'ok',
            data: { pending_approvals: 1, low_profit_products: 3, missing_data_products: 5, sync_errors: 0, total_candidates: 20, total_recommendations: 15, list_ready_products: 5 },
          }),
        });
      }

      if (url.includes('/v1/owner/suggestions')) {
        return route.fulfill({
          status: 200, contentType: 'application/json',
          body: JSON.stringify({ code: 0, message: 'ok', data: [APPROVABLE_SUGGESTION, CAUTIOUS_SUGGESTION, SKIP_SUGGESTION] }),
        });
      }

      if (url.includes('/v1/owner/platform-sync')) {
        return route.fulfill({
          status: 200, contentType: 'application/json',
          body: JSON.stringify({ code: 0, message: 'ok', data: [{ platform_id: 1, platform_name: 'Ozon', mode: 'mock', orders_sync: 'success', products_sync: 'success', fees_sync: 'pending', settlements_sync: 'pending', last_sync_at: '2026-06-30T08:00:00Z' }] }),
        });
      }

      if (url.includes('/v1/listing-tasks/') && route.request().method() === 'PUT') {
        return route.fulfill({
          status: 200, contentType: 'application/json',
          body: JSON.stringify({ code: 0, message: 'ok', data: { id: 42, status: 'approved' } }),
        });
      }

      return route.fulfill({ status: 200, contentType: 'application/json', body: JSON.stringify({ code: 0, message: 'ok', data: {} }) });
    });
  });

  test('page renders with risk summary stat cards', async ({ page }) => {
    await page.goto('/owner');
    await expect(page.locator('h1')).toContainText('Owner', { timeout: 10000 });

    // Scope stat value assertions to specific cards to avoid ambiguity
    const pendingCard = page.locator('.ant-statistic').filter({ hasText: '待审批上架' });
    await expect(pendingCard).toBeVisible();
    await expect(pendingCard.locator('.ant-statistic-content-value-int')).toHaveText('1');

    await expect(page.locator('.ant-statistic').filter({ hasText: '低利润商品' }).locator('.ant-statistic-content-value-int')).toHaveText('3');
    await expect(page.locator('.ant-statistic').filter({ hasText: '资料不完整商品' }).locator('.ant-statistic-content-value-int')).toHaveText('5');
    await expect(page.locator('.ant-statistic').filter({ hasText: '同步异常' }).locator('.ant-statistic-content-value-int')).toHaveText('0');
    await expect(page.getByText('候选商品总数')).toBeVisible();
    await expect(page.getByText('评估建议总数')).toBeVisible();
    await expect(page.getByText('Mock / 模拟')).toBeVisible();
  });

  test('agent suggestions table shows correct status tags', async ({ page }) => {
    await page.goto('/owner');
    await expect(page.locator('h1')).toContainText('Owner', { timeout: 10000 });
    await expect(page.getByText('Agent 上架建议')).toBeVisible();

    const table = page.locator('.ant-table-tbody');
    await expect(table).toBeVisible();
    await expect(table).toContainText('推荐上架');
    await expect(table).toContainText('谨慎');
    await expect(table).toContainText('跳过');
    await expect(table).toContainText('Wireless Bluetooth Earbuds');
    await expect(table).toContainText('Premium Phone Case');
    await expect(table).toContainText('Unknown Brand Charger');
  });

  test('approve a suggestion through the modal', async ({ page }) => {
    await page.goto('/owner');
    await expect(page.locator('h1')).toContainText('Owner', { timeout: 10000 });

    // Find the row with our approvable suggestion
    const row = page.locator('.ant-table-tbody .ant-table-row').filter({ hasText: 'Wireless Bluetooth Earbuds' });
    await expect(row).toBeVisible();

    // The approve button uses Ant Design Button — use CSS + text as fallback
    const approveBtn = row.locator('button').filter({ hasText: /批准/ });
    await expect(approveBtn).toBeEnabled();
    await approveBtn.click();

    // Approval modal should appear — check title text
    await expect(page.getByText('确认批准上架')).toBeVisible({ timeout: 5000 });

    // Click the modal footer's primary button
    await page.locator('.ant-modal-footer .ant-btn-primary').click();

    // Wait for success toast — Ant Design message
    await expect(page.locator('.ant-message')).toContainText(/已批准上架/, { timeout: 10000 });
  });
});

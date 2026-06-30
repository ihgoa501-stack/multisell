import { test, expect } from '@playwright/test';

/**
 * Login Flow E2E tests.
 *
 * Covers the full login experience:
 *   1. Login page renders with brand and form
 *   2. Successful login → redirect to dashboard with stat cards
 *   3. Invalid credentials show error message
 *
 * All tests use Playwright route interception for API mocking,
 * so no running backend is needed.
 */

const FAKE_ACCESS_TOKEN = 'e2e-login-test-token';
const FAKE_REFRESH_TOKEN = 'e2e-login-test-refresh-token';
const MOCK_USER = { id: 1, username: 'e2e_test', email: 'e2e@lingmirror.test', name: 'E2E Tester', roles: ['admin'] };

test.describe('Login Flow', () => {
  test.beforeEach(async ({ page }) => {
    // Intercept all API calls with mock responses
    await page.route('**/api/**', async (route) => {
      const url = route.request().url();

      // --- Auth endpoints ---
      if (url.includes('/v1/auth/login') && route.request().method() === 'POST') {
        return route.fulfill({
          status: 200,
          contentType: 'application/json',
          body: JSON.stringify({
            code: 0,
            message: 'ok',
            data: { access_token: FAKE_ACCESS_TOKEN, refresh_token: FAKE_REFRESH_TOKEN, user: MOCK_USER },
          }),
        });
      }
      if (url.includes('/v1/auth/refresh')) {
        return route.fulfill({
          status: 200,
          contentType: 'application/json',
          body: JSON.stringify({
            code: 0, message: 'ok',
            data: { access_token: FAKE_ACCESS_TOKEN, refresh_token: FAKE_REFRESH_TOKEN },
          }),
        });
      }

      // --- RBAC permissions ---
      if (url.includes('/v1/rbac/')) {
        return route.fulfill({
          status: 200,
          contentType: 'application/json',
          body: JSON.stringify({
            code: 0, message: 'ok',
            data: { permissions: ['product.read', 'order.read', 'finance.read', 'agent.read', 'settings.read', 'rbac.manage', 'report.read', 'audit.read'] },
          }),
        });
      }

      // --- Dashboard endpoints ---
      if (url.includes('/v1/dashboard/overview')) {
        return route.fulfill({
          status: 200,
          contentType: 'application/json',
          body: JSON.stringify({
            code: 0, message: 'ok',
            data: {
              order_total: 42, order_revenue: 15000.50, order_profit: 3500.00, sku_total: 128,
              low_stock_count: 3, out_of_stock_count: 1, listing_active_count: 85,
              aftersales_pending_count: 2, exception_open_count: 5,
              month_revenue: 25000, month_cost: 18000,
              order_by_status: { pending: 10, paid: 15, shipped: 8, completed: 7, cancelled: 2 },
              platform_connections: [],
              agent_statuses: [],
            },
          }),
        });
      }
      if (url.includes('/v1/dashboard/exceptions')) {
        return route.fulfill({
          status: 200, contentType: 'application/json',
          body: JSON.stringify({ code: 0, message: 'ok', data: [] }),
        });
      }
      if (url.includes('/v1/dashboard/rejection-reasons')) {
        return route.fulfill({
          status: 200, contentType: 'application/json',
          body: JSON.stringify({ code: 0, message: 'ok', data: [] }),
        });
      }

      // --- User / profile endpoints ---
      if (url.includes('/v1/user/')) {
        return route.fulfill({
          status: 200, contentType: 'application/json',
          body: JSON.stringify({ code: 0, message: 'ok', data: MOCK_USER }),
        });
      }

      // Default: generic success for unhandled endpoints
      return route.fulfill({
        status: 200, contentType: 'application/json',
        body: JSON.stringify({ code: 0, message: 'ok', data: {} }),
      });
    });
  });

  test('renders login page with brand and form', async ({ page }) => {
    await page.goto('/login');

    // Brand elements
    await expect(page.getByText('凌镜')).toBeVisible();
    await expect(page.getByText(/跨境电商.*AI.*工作台/)).toBeVisible();

    // Form
    await expect(page.getByPlaceholder('用户名 / 邮箱')).toBeVisible();
    await expect(page.getByPlaceholder('密码')).toBeVisible();
    // Ant Design button renders accessible name as "登 录" with spacing
    await expect(page.getByRole('button', { name: /登/ })).toBeVisible();
  });

  test('submits login and redirects to dashboard with stat cards', async ({ page }) => {
    await page.goto('/login');

    // Fill and submit form
    await page.getByPlaceholder('用户名 / 邮箱').fill('e2e@lingmirror.test');
    await page.getByPlaceholder('密码').fill('test-password');
    await page.getByRole('button', { name: /登/ }).click();

    // Wait for success toast (use exact match to avoid source-code references)
    await expect(page.getByText('登录成功', { exact: true })).toBeVisible({ timeout: 10000 });

    // Should navigate to dashboard
    await page.waitForURL('**/dashboard', { timeout: 15000 });

    // Dashboard renders stat cards with mock data
    await expect(page.getByText('订单总数')).toBeVisible({ timeout: 10000 });
    await expect(page.getByText('42')).toBeVisible();
    await expect(page.getByText('SKU 总数')).toBeVisible();
    await expect(page.getByText('128')).toBeVisible();
    await expect(page.getByText('低库存')).toBeVisible();
    // The "3" low stock value inside the low-stock stat card
    await expect(
      page.locator('.ant-statistic').filter({ hasText: '低库存' }).locator('.ant-statistic-content-value-int')
    ).toHaveText('3');
  });

  test('shows error on invalid credentials', async ({ page }) => {
    // Return 400 (not 401, which triggers api-client redirect to /login)
    await page.route('**/api/v1/auth/login', async (route) => {
      if (route.request().method() === 'POST') {
        return route.fulfill({
          status: 400,
          contentType: 'application/json',
          body: JSON.stringify({ code: 400, message: 'Invalid credentials' }),
        });
      }
      return route.fallback();
    });

    await page.goto('/login');
    await page.getByPlaceholder('用户名 / 邮箱').fill('wrong@test.com');
    await page.getByPlaceholder('密码').fill('wrong-password');
    await page.getByRole('button', { name: /登/ }).click();

    // Should show an error message
    await expect(page.locator('.ant-message')).toContainText(/Bad Request/i, { timeout: 10000 });

    // Should remain on login page (no redirect)
    await expect(page).toHaveURL(/\/login/);
  });
});

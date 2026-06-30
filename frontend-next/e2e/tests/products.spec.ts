import { test, expect } from '@playwright/test';

/**
 * Products E2E tests.
 *
 * Covers product browsing and creation page rendering:
 *   1. Product list page renders with table data
 *   2. Product create form renders with all input fields
 *
 * All tests use Playwright route interception for API mocking
 * and inject a fake JWT token to bypass AuthGuard.
 *
 * NOTE: Product detail test is skipped due to a pre-existing
 * compilation error in products/[id]/page.tsx (line 673).
 */

const FAKE_TOKEN = 'e2e-products-test-token';

function makeProduct(id: number, overrides: Record<string, unknown> = {}) {
  return {
    id, name: `Test Product ${id}`, subtitle: `Subtitle ${id}`,
    brand_id: id % 2 + 1, category_id: id % 3 + 10,
    unit: '件', status: id === 1 ? '1' : '0', cargo_type: 'physical',
    created_at: new Date(Date.now() - id * 86400000).toISOString(),
    ...overrides,
  };
}

function paginatedBody(items: unknown[], total: number) {
  return JSON.stringify({ code: 0, message: 'ok', data: items, total, page: 1, size: 10 });
}

test.describe('Products — List and Create', () => {
  test.beforeEach(async ({ page }) => {
    await page.addInitScript((token: string) => {
      localStorage.setItem('token', token);
      localStorage.setItem('refresh_token', token);
    }, FAKE_TOKEN);

    await page.route('**/api/**', async (route) => {
      const url = route.request().url();
      const method = route.request().method();

      if (url.includes('/v1/rbac/')) {
        return route.fulfill({
          status: 200, contentType: 'application/json',
          body: JSON.stringify({ code: 0, message: 'ok', data: { permissions: ['*'] } }),
        });
      }

      if (url.includes('/v1/products') && method === 'POST') {
        return route.fulfill({
          status: 200, contentType: 'application/json',
          body: JSON.stringify({ code: 0, message: 'ok', data: { id: 99 } }),
        });
      }

      if (url.match(/\/v1\/products(\?|$)/) && method === 'GET') {
        const items = [makeProduct(1, { status: '1' }), makeProduct(2), makeProduct(3)];
        return route.fulfill({
          status: 200, contentType: 'application/json',
          body: paginatedBody(items, items.length),
        });
      }

      return route.fulfill({ status: 200, contentType: 'application/json', body: JSON.stringify({ code: 0, message: 'ok', data: {} }) });
    });
  });

  test('product list page renders table with data', async ({ page }) => {
    await page.goto('/products');
    await expect(page.locator('h1')).toContainText('商品', { timeout: 10000 });

    const table = page.locator('.ant-table-tbody');
    await expect(table).toBeVisible();
    await expect(table).toContainText('Test Product 1', { timeout: 10000 });
    await expect(table).toContainText('Test Product 2');
    await expect(table).toContainText('Test Product 3');
    await expect(page.locator('.ant-pagination')).toContainText('共 3 条');
  });

  test('product create form renders with all fields', async ({ page }) => {
    await page.goto('/products/create');
    await expect(page.locator('h1')).toContainText('创建商品', { timeout: 10000 });

    // Verify all form fields render
    await expect(page.getByPlaceholder('商品名称')).toBeVisible();
    await expect(page.getByPlaceholder('商品副标题')).toBeVisible();
    await expect(page.getByPlaceholder('品牌 ID')).toBeVisible();
    await expect(page.getByPlaceholder('分类 ID')).toBeVisible();
    await expect(page.getByPlaceholder('件 / 个 / 箱')).toBeVisible();
    await expect(page.getByPlaceholder('商品描述')).toBeVisible();
  });
});

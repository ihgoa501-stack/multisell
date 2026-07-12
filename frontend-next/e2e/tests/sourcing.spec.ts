import { test, expect } from '@playwright/test';

/**
 * A8 Sourcing Agent — Full Pipeline E2E test.
 *
 * Tests the complete sourcing flow:
 *   1. Load page with empty state
 *   2. Enter 1688 URL and submit
 *   3. See loading indicator while fetching
 *   4. Product appears in table with score, status, price
 *   5. Error states are handled gracefully
 *   6. Recommendations list and pagination
 *
 * All tests use Playwright route interception for API mocking,
 * so no running backend is needed. A fake JWT token is injected
 * into localStorage to bypass the AuthGuard.
 *
 * NOTE: Each test registers a single page.route('api glob', ...) that
 * handles ALL API calls (sourcing, RBAC, etc.) to avoid route override
 * ordering issues with Playwright's route handler chain.
 */

const FAKE_TOKEN = 'e2e-fake-token-for-testing';

/** Full response to the GET /recommendations endpoint. */
function recommendationsBody(items: unknown[], total: number) {
  return {
    code: 0,
    message: 'ok',
    data: { data: items, total, page: 1, size: 20 },
  };
}

/** Full response to the POST /fetch endpoint. */
function fetchBody(product_id: number, title: string, price: number, score: number, status: string) {
  return { code: 0, message: 'ok', data: { product_id, title, price, score, status } };
}

/** A single product recommendation. */
function sampleProduct(id: number, overrides: Record<string, unknown> = {}) {
  return {
    id,
    title: `Product ${id}`,
    source_url: `https://detail.1688.com/offer/${id}.html`,
    supplier_name: `Supplier ${id}`,
    price: 50 + id * 10,
    score: Math.min(id + 5, 10),
    status: id % 3 === 0 ? 'pending' : id % 3 === 1 ? 'recommended' : 'low_quality',
    created_at: new Date(Date.now() - id * 3600000).toISOString(),
    ...overrides,
  };
}

test.describe('A8 Sourcing Agent — Full Pipeline', () => {
  // ------------------------------------------------------------------
  // Hooks
  // ------------------------------------------------------------------

  test.beforeEach(async ({ page }) => {
    // Inject fake auth token before page scripts run.
    await page.addInitScript((token: string) => {
      localStorage.setItem('token', token);
      localStorage.setItem('refresh_token', token);
    }, FAKE_TOKEN);
  });

  // ------------------------------------------------------------------
  // Tests
  // ------------------------------------------------------------------

  test('shows empty state when no recommendations exist', async ({ page }) => {
    await page.route('**/api/**', async (route) => {
      const url = route.request().url();
      if (url.includes('/rbac/')) {
        await route.fulfill({ status: 200, contentType: 'application/json', body: JSON.stringify({ code: 0, message: 'ok', data: { permissions: [] } }) });
      } else {
        await route.fulfill({ status: 200, contentType: 'application/json', body: JSON.stringify(recommendationsBody([], 0)) });
      }
    });

    await page.goto('/sourcing');

    // Wait for page heading (confirms AuthGuard passed).
    await expect(page.locator('h1')).toContainText('AI 选品', { timeout: 10000 });

    // Empty indicator should be visible.
    await expect(page.locator('.ant-empty').first()).toBeVisible();
  });

  test('displays URL input and fetch button', async ({ page }) => {
    await page.route('**/api/**', async (route) => {
      const url = route.request().url();
      if (url.includes('/rbac/')) {
        await route.fulfill({ status: 200, contentType: 'application/json', body: JSON.stringify({ code: 0, message: 'ok', data: { permissions: [] } }) });
      } else {
        await route.fulfill({ status: 200, contentType: 'application/json', body: JSON.stringify(recommendationsBody([], 0)) });
      }
    });

    await page.goto('/sourcing');
    await expect(page.locator('h1')).toContainText('AI 选品', { timeout: 10000 });

    const urlInput = page.locator('input[placeholder*="1688"]');
    await expect(urlInput).toBeVisible();
    await expect(urlInput).toHaveAttribute('placeholder', /1688/);

    await expect(page.getByRole('button', { name: /采集分析/i })).toBeVisible();
  });

  test('shows loading spinner while fetching product', async ({ page }) => {
    // Route all requests; delay the fetch response.
    await page.route('**/api/**', async (route) => {
      const url = route.request().url();
      if (url.includes('/rbac/')) {
        await route.fulfill({ status: 200, contentType: 'application/json', body: JSON.stringify({ code: 0, message: 'ok', data: { permissions: [] } }) });
      } else if (url.includes('/fetch')) {
        await new Promise((r) => setTimeout(r, 800));
        await route.fulfill({ status: 200, contentType: 'application/json', body: JSON.stringify(fetchBody(1, 'Delayed', 99, 8, 'recommended')) });
      } else {
        await route.fulfill({ status: 200, contentType: 'application/json', body: JSON.stringify(recommendationsBody([], 0)) });
      }
    });

    await page.goto('/sourcing');
    await expect(page.locator('h1')).toContainText('AI 选品', { timeout: 10000 });

    await page.locator('input[placeholder*="1688"]').fill('https://detail.1688.com/offer/test.html');
    await page.getByRole('button', { name: /采集分析/i }).click();

    // Spin should appear during the delayed fetch.
    await expect(page.locator('.ant-spin').first()).toBeVisible({ timeout: 2000 });
  });

  test('displays fetched product in recommendations table', async ({ page }) => {
    let fetchCalled = false;

    await page.route('**/api/**', async (route) => {
      const url = route.request().url();
      if (url.includes('/rbac/')) {
        await route.fulfill({ status: 200, contentType: 'application/json', body: JSON.stringify({ code: 0, message: 'ok', data: { permissions: [] } }) });
      } else if (url.includes('/fetch')) {
        fetchCalled = true;
        await route.fulfill({ status: 200, contentType: 'application/json', body: JSON.stringify(fetchBody(42, 'Wireless Bluetooth Earbuds', 35.99, 9, 'recommended')) });
      } else {
        // After fetch, return populated list
        const items = fetchCalled
          ? [sampleProduct(42, { title: 'Wireless Bluetooth Earbuds', supplier_name: 'Shenzhen Audio Tech', price: 35.99, score: 9 })]
          : [];
        await route.fulfill({ status: 200, contentType: 'application/json', body: JSON.stringify(recommendationsBody(items, items.length)) });
      }
    });

    await page.goto('/sourcing');
    await expect(page.locator('h1')).toContainText('AI 选品', { timeout: 10000 });

    // Fill URL and click fetch.
    await page.locator('input[placeholder*="1688"]').fill('https://detail.1688.com/offer/test.html');
    await page.getByRole('button', { name: /采集分析/i }).click();

    // Wait for success message.
    await expect(page.locator('.ant-message')).toContainText(/采集成功|评分/i, { timeout: 5000 });

    // Table should contain the product.
    await expect(page.locator('.ant-table-tbody')).toContainText('Wireless Bluetooth Earbuds', { timeout: 5000 });
    await expect(page.locator('.ant-table-tbody')).toContainText('Shenzhen Audio Tech');
    await expect(page.locator('.ant-badge').first()).toBeVisible();
  });

  test('handles fetch error gracefully', async ({ page }) => {
    await page.route('**/api/**', async (route) => {
      const url = route.request().url();
      if (url.includes('/rbac/')) {
        await route.fulfill({ status: 200, contentType: 'application/json', body: JSON.stringify({ code: 0, message: 'ok', data: { permissions: [] } }) });
      } else if (url.includes('/fetch')) {
        await route.fulfill({ status: 500, contentType: 'application/json', body: JSON.stringify({ code: 500, message: 'bridge offline' }) });
      } else {
        await route.fulfill({ status: 200, contentType: 'application/json', body: JSON.stringify(recommendationsBody([], 0)) });
      }
    });

    await page.goto('/sourcing');
    await expect(page.locator('h1')).toContainText('AI 选品', { timeout: 10000 });

    await page.locator('input[placeholder*="1688"]').fill('https://detail.1688.com/offer/fail.html');
    await page.getByRole('button', { name: /采集分析/i }).click();

    await expect(page.locator('.ant-message')).toContainText(/失败|error/i, { timeout: 3000 });
  });

  test('full pipeline: submit URL, analyze, see result in table', async ({ page }) => {
    let fetchCalled = false;

    await page.route('**/api/**', async (route) => {
      const url = route.request().url();
      if (url.includes('/rbac/')) {
        await route.fulfill({ status: 200, contentType: 'application/json', body: JSON.stringify({ code: 0, message: 'ok', data: { permissions: [] } }) });
      } else if (url.includes('/fetch')) {
        fetchCalled = true;
        await route.fulfill({ status: 200, contentType: 'application/json', body: JSON.stringify(fetchBody(99, 'Winter Jacket Waterproof', 89.00, 7, 'recommended')) });
      } else {
        const items = fetchCalled
          ? [sampleProduct(99, { title: 'Winter Jacket Waterproof', supplier_name: 'Guangzhou Garments Co.', price: 89.00, score: 7, status: 'recommended' })]
          : [];
        await route.fulfill({ status: 200, contentType: 'application/json', body: JSON.stringify(recommendationsBody(items, items.length)) });
      }
    });

    await page.goto('/sourcing');
    await expect(page.locator('h1')).toContainText('AI 选品', { timeout: 10000 });

    await page.locator('input[placeholder*="1688"]').fill('https://detail.1688.com/offer/jacket-test.html');
    await page.getByRole('button', { name: /采集分析/i }).click();

    await expect(page.locator('.ant-message')).toContainText(/采集成功/i, { timeout: 5000 });
    await expect(page.locator('.ant-table-tbody')).toContainText('Winter Jacket Waterproof', { timeout: 5000 });
    await expect(page.locator('.ant-badge').first()).toContainText('7');
    await expect(page.locator('.ant-tag')).toContainText('推荐');
  });

  test('displays multiple recommendations', async ({ page }) => {
    const items = [
      sampleProduct(1, { title: 'Product Alpha', score: 9, status: 'recommended' }),
      sampleProduct(2, { title: 'Product Beta', score: 5, status: 'pending' }),
    ];

    await page.route('**/api/**', async (route) => {
      const url = route.request().url();
      if (url.includes('/rbac/')) {
        await route.fulfill({ status: 200, contentType: 'application/json', body: JSON.stringify({ code: 0, message: 'ok', data: { permissions: [] } }) });
      } else {
        await route.fulfill({ status: 200, contentType: 'application/json', body: JSON.stringify(recommendationsBody(items, items.length)) });
      }
    });

    await page.goto('/sourcing');
    await expect(page.locator('h1')).toContainText('AI 选品', { timeout: 10000 });

    const table = page.locator('.ant-table-tbody');
    await expect(table).toBeVisible();
    await expect(table).toContainText('Product Alpha');
    await expect(table).toContainText('Product Beta');
    await expect(page.locator('.ant-pagination')).toBeVisible();
  });

  test('shows correct status tags', async ({ page }) => {
    const items = [
      sampleProduct(1, { title: 'Top Pick', score: 9, status: 'recommended' }),
      sampleProduct(2, { title: 'Pending Item', score: 5, status: 'pending' }),
      sampleProduct(3, { title: 'Low Quality', score: 2, status: 'low_quality' }),
    ];

    await page.route('**/api/**', async (route) => {
      const url = route.request().url();
      if (url.includes('/rbac/')) {
        await route.fulfill({ status: 200, contentType: 'application/json', body: JSON.stringify({ code: 0, message: 'ok', data: { permissions: [] } }) });
      } else {
        await route.fulfill({ status: 200, contentType: 'application/json', body: JSON.stringify(recommendationsBody(items, items.length)) });
      }
    });

    await page.goto('/sourcing');
    await expect(page.locator('h1')).toContainText('AI 选品', { timeout: 10000 });

    const tags = page.locator('.ant-tag');
    await expect(tags.nth(0)).toContainText('推荐');
    await expect(tags.nth(1)).toContainText('待处理');
    await expect(tags.nth(2)).toContainText('低质量');
  });

  test('page layout renders correctly', async ({ page }) => {
    await page.setViewportSize({ width: 1280, height: 800 });

    await page.route('**/api/**', async (route) => {
      const url = route.request().url();
      if (url.includes('/rbac/')) {
        await route.fulfill({ status: 200, contentType: 'application/json', body: JSON.stringify({ code: 0, message: 'ok', data: { permissions: [] } }) });
      } else {
        await route.fulfill({ status: 200, contentType: 'application/json', body: JSON.stringify(recommendationsBody([], 0)) });
      }
    });

    await page.goto('/sourcing');
    await expect(page.locator('h1')).toContainText('AI 选品', { timeout: 10000 });

    const container = page.locator('div[style*="padding: 16px 20px"]').first();
    await expect(container).toBeVisible();
    await expect(page.locator('.ant-card').first()).toBeVisible();
  });
});

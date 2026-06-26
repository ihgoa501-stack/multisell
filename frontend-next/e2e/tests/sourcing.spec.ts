import { test, expect } from '@playwright/test';

/**
 * A8 Sourcing Agent — E2E sourcing panel test.
 *
 * Uses Playwright route interception to mock API responses,
 * so no running backend is needed.
 */

const SOURCING_URL = '/sourcing';
const API_PREFIX = '**/api/v1/sourcing';

test.describe('Sourcing Panel', () => {
  test.beforeEach(async ({ page }) => {
    await page.goto(SOURCING_URL);
  });

  test('shows empty state when no recommendations exist', async ({ page }) => {
    // Mock GET /recommendations to return empty list.
    await page.route(`${API_PREFIX}/recommendations`, async (route) => {
      await route.fulfill({
        status: 200,
        contentType: 'application/json',
        body: JSON.stringify({ code: 0, message: 'ok', data: { items: [], total: 0, page: 1, size: 20 } }),
      });
    });

    // Wait for the page to load and check for an empty state.
    await page.waitForLoadState('networkidle');
    // The table should show "no data" or empty indicator.
    await expect(page.locator('text=暂无数据').or(page.locator('.ant-empty')).or(page.locator('text=No Data'))).toBeVisible();
  });

  test('displays fetch button and URL input', async ({ page }) => {
    // Mock empty recommendations list.
    await page.route(`${API_PREFIX}/recommendations`, async (route) => {
      await route.fulfill({
        status: 200,
        contentType: 'application/json',
        body: JSON.stringify({ code: 0, message: 'ok', data: { items: [], total: 0, page: 1, size: 20 } }),
      });
    });

    await page.waitForLoadState('networkidle');

    // URL input should be visible.
    const urlInput = page.locator('input[placeholder*="1688"]').or(page.locator('textarea'));
    await expect(urlInput).toBeVisible();

    // Submit button should be visible.
    await expect(page.getByRole('button', { name: /采集|分析|fetch/i })).toBeVisible();
  });

  test('shows loading state while fetching product', async ({ page }) => {
    // Mock recommendations to empty, then intercept fetch call.
    await page.route(`${API_PREFIX}/recommendations`, async (route) => {
      await route.fulfill({
        status: 200,
        contentType: 'application/json',
        body: JSON.stringify({ code: 0, message: 'ok', data: { items: [], total: 0, page: 1, size: 20 } }),
      });
    });

    // Delay the fetch response to trigger loading state.
    await page.route(`${API_PREFIX}/fetch`, async (route) => {
      await new Promise((r) => setTimeout(r, 500));
      await route.fulfill({
        status: 200,
        contentType: 'application/json',
        body: JSON.stringify({
          code: 0, message: 'ok',
          data: { id: 1, title: 'Test Product', source_url: 'https://detail.1688.com/offer/test.html' },
        }),
      });
    });

    await page.waitForLoadState('networkidle');

    // Type a URL.
    const urlInput = page.locator('input').first();
    await urlInput.fill('https://detail.1688.com/offer/test.html');
    await page.getByRole('button', { name: /采集|分析|fetch/i }).click();

    // Expect loading indicator (Spin).
    await expect(page.locator('.ant-spin')).toBeVisible({ timeout: 2000 });
  });

  test('displays fetched product in recommendations list', async ({ page }) => {
    // Mock the initial empty list.
    await page.route(`${API_PREFIX}/recommendations`, async (route) => {
      const url = route.request().url();
      if (url.includes('fetch')) {
        await route.fallback();
        return;
      }
      await route.fulfill({
        status: 200,
        contentType: 'application/json',
        body: JSON.stringify({ code: 0, message: 'ok', data: { items: [], total: 0, page: 1, size: 20 } }),
      });
    });

    // Mock fetch endpoint.
    await page.route(`${API_PREFIX}/fetch`, async (route) => {
      await route.fulfill({
        status: 200,
        contentType: 'application/json',
        body: JSON.stringify({
          code: 0, message: 'ok',
          data: {
            id: 1,
            title: 'Test Product',
            source_url: 'https://detail.1688.com/offer/test.html',
            supplier_name: 'Test Supplier',
            price: 25.50,
            score: 8,
            status: 'pending',
          },
        }),
      });
    });

    // After fetch, mock the recommendations list to include the new item.
    await page.route(`${API_PREFIX}/recommendations`, async (route, request) => {
      if (request.url().includes('fetch')) {
        await route.fallback();
        return;
      }
      await route.fulfill({
        status: 200,
        contentType: 'application/json',
        body: JSON.stringify({
          code: 0, message: 'ok',
          data: {
            items: [
              {
                id: 1,
                title: 'Test Product',
                source_url: 'https://detail.1688.com/offer/test.html',
                supplier_name: 'Test Supplier',
                price: 25.50,
                score: 8,
                status: 'pending',
                created_at: '2026-06-26T12:00:00Z',
              },
            ],
            total: 1, page: 1, size: 20,
          },
        }),
      });
    });

    await page.waitForLoadState('networkidle');

    const urlInput = page.locator('input').first();
    await urlInput.fill('https://detail.1688.com/offer/test.html');
    await page.getByRole('button', { name: /采集|分析|fetch/i }).click();

    // Wait for the product to appear in the table.
    await expect(page.locator('text=Test Product')).toBeVisible({ timeout: 5000 });
    await expect(page.locator('text=Test Supplier')).toBeVisible();
  });

  test('handles fetch error gracefully', async ({ page }) => {
    // Mock empty recommendations.
    await page.route(`${API_PREFIX}/recommendations`, async (route) => {
      await route.fulfill({
        status: 200,
        contentType: 'application/json',
        body: JSON.stringify({ code: 0, message: 'ok', data: { items: [], total: 0, page: 1, size: 20 } }),
      });
    });

    // Mock fetch to return error.
    await page.route(`${API_PREFIX}/fetch`, async (route) => {
      await route.fulfill({
        status: 500,
        contentType: 'application/json',
        body: JSON.stringify({ code: 500, message: 'bridge offline' }),
      });
    });

    await page.waitForLoadState('networkidle');

    const urlInput = page.locator('input').first();
    await urlInput.fill('https://detail.1688.com/offer/fail.html');
    await page.getByRole('button', { name: /采集|分析|fetch/i }).click();

    // Should show error message.
    await expect(page.locator('text=error').or(page.locator('text=失败')).or(page.locator('text=fail'))).toBeVisible({ timeout: 3000 });
  });
});

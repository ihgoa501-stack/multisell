import { type Page, type Route } from '@playwright/test';

/**
 * A8 Sourcing Agent — mock API handlers for Playwright route interception.
 *
 * Import and call from test cases to mock the sourcing API endpoints without
 * requiring a running backend.
 *
 * Usage:
 *   import { mockEmptyRecommendations, mockFetchSuccess, API_PREFIX } from '../sourcing.mock';
 *
 *   test('my test', async ({ page }) => {
 *     await mockEmptyRecommendations(page);
 *     await mockFetchSuccess(page);
 *     await page.goto('/sourcing');
 *     // ...
 *   });
 */

export const API_PREFIX = '**/api/v1/sourcing';

// ---------------------------------------------------------------------------
// Mock data factories
// ---------------------------------------------------------------------------

export interface SourcingRecommendation {
  id: number;
  source_url: string;
  title: string;
  supplier_name: string;
  price: number;
  score: number;
  status: string;
  product_id_1688?: string;
  image_url?: string;
  recommend_reason?: string;
  created_at: string;
}

export function sampleRecommendation(overrides?: Partial<SourcingRecommendation>): SourcingRecommendation {
  return {
    id: 1,
    source_url: 'https://detail.1688.com/offer/test.html',
    title: 'Test Product',
    supplier_name: 'Test Supplier Co.',
    price: 25.50,
    score: 8,
    status: 'recommended',
    created_at: '2026-06-26T12:00:00Z',
    ...overrides,
  };
}

export function sampleRecommendations(count: number = 3): SourcingRecommendation[] {
  const products = [];
  for (let i = 1; i <= count; i++) {
    const statuses = ['recommended', 'pending', 'low_quality', 'pending', 'recommended'];
    const scores = [9, 5, 2, 6, 8];
    products.push(
      sampleRecommendation({
        id: i,
        title: `Product ${i}`,
        score: scores[(i - 1) % scores.length],
        status: statuses[(i - 1) % statuses.length],
        price: 15.0 + i * 10.5,
        created_at: new Date(Date.now() - i * 3600000).toISOString(),
      }),
    );
  }
  return products;
}

// ---------------------------------------------------------------------------
// Route interception helpers
// ---------------------------------------------------------------------------

/**
 * Mock GET /recommendations to return an empty list.
 */
export async function mockEmptyRecommendations(page: Page): Promise<void> {
  await page.route(`${API_PREFIX}/recommendations`, async (route: Route) => {
    await route.fulfill({
      status: 200,
      contentType: 'application/json',
      body: JSON.stringify(emptyRecommendationsResponse()),
    });
  });
}

/**
 * Mock GET /recommendations to return the given items.
 */
export async function mockRecommendations(
  page: Page,
  items: SourcingRecommendation[],
  total?: number,
): Promise<void> {
  const t = total ?? items.length;
  await page.route(`${API_PREFIX}/recommendations`, async (route: Route) => {
    await route.fulfill({
      status: 200,
      contentType: 'application/json',
      body: JSON.stringify(populatedRecommendationsResponse(items, t)),
    });
  });
}

/**
 * Mock POST /fetch to succeed and return product data.
 */
export async function mockFetchSuccess(
  page: Page,
  product?: Partial<{
    product_id: number;
    title: string;
    price: number;
    score: number;
    status: string;
  }>,
): Promise<void> {
  const data = {
    product_id: 1,
    title: 'Test Product',
    price: 25.5,
    score: 8,
    status: 'recommended',
    ...product,
  };
  await page.route(`${API_PREFIX}/fetch`, async (route: Route) => {
    await route.fulfill({
      status: 200,
      contentType: 'application/json',
      body: JSON.stringify(fetchResponse(data)),
    });
  });
}

/**
 * Mock POST /fetch to return an error.
 */
export async function mockFetchError(page: Page, statusCode: number = 500, message: string = 'bridge offline'): Promise<void> {
  await page.route(`${API_PREFIX}/fetch`, async (route: Route) => {
    await route.fulfill({
      status: statusCode,
      contentType: 'application/json',
      body: JSON.stringify({
        code: statusCode,
        message,
      }),
    });
  });
}

/**
 * Mock POST /fetch with a delay to trigger loading/loading state.
 */
export async function mockFetchLoading(
  page: Page,
  delayMs: number = 1500,
): Promise<void> {
  await page.route(`${API_PREFIX}/fetch`, async (route: Route) => {
    await new Promise((r) => setTimeout(r, delayMs));
    await route.fulfill({
      status: 200,
      contentType: 'application/json',
      body: JSON.stringify(
        fetchResponse({ product_id: 1, title: 'Delayed Product', price: 30, score: 7, status: 'pending' }),
      ),
    });
  });
}

/**
 * Mock BOTH endpoints at once with a full set of recommendations.
 * Useful for simpler tests that just need the page to render with data.
 */
export async function mockFullSourcingPage(page: Page, itemCount: number = 5): Promise<void> {
  const items = sampleRecommendations(itemCount);
  await mockRecommendations(page, items);
  await mockFetchSuccess(page);
}

// ---------------------------------------------------------------------------
// Response shape helpers
//
// The frontend sourcing page calls apiClient.get<PageData>('/v1/sourcing/...')
// and accesses res.data.data / res.data.total.  The backend uses
// response.Paginated() which returns {code, message, data: items[], total, page, size}.
// To make res.data.data return the items array, the mock wraps items inside
// an extra `data` layer:
//
//   {code:0, message:"ok", data: {data: [...items...], total: N, page: P, size: S}}
//
// The fetch endpoint returns {code:0, message:"ok", data: {product_id, ...}}.
// ---------------------------------------------------------------------------

function emptyRecommendationsResponse() {
  return {
    code: 0,
    message: 'ok',
    data: {
      data: [],
      total: 0,
      page: 1,
      size: 20,
    },
  };
}

function populatedRecommendationsResponse(items: SourcingRecommendation[], total: number) {
  return {
    code: 0,
    message: 'ok',
    data: {
      data: items,
      total,
      page: 1,
      size: 20,
    },
  };
}

function fetchResponse(data: {
  product_id: number;
  title: string;
  price: number;
  score: number;
  status: string;
}) {
  return {
    code: 0,
    message: 'ok',
    data,
  };
}

import { expect, test } from '@playwright/test';

const API_BASE = process.env.E2E_API_BASE || 'http://localhost:8080';
const TOKEN = 'business-loop-e2e-token';

test.describe('Owner self-use business loop', () => {
  test.beforeAll(async () => {
    const response = await fetch(`${API_BASE}/api/health`).catch(() => null);
    if (!response?.ok) throw new Error(`Backend not reachable at ${API_BASE}; acceptance must fail, not skip`);
  });

  test.beforeEach(async ({ page }) => {
    await page.addInitScript((token) => {
      localStorage.setItem('token', token);
      localStorage.setItem('refresh_token', token);
    }, TOKEN);
  });

  test('candidate research keeps incomplete evidence blocked', async ({ page }) => {
    let imported = false;
    await page.route('**/api/**', async (route) => {
      const url = route.request().url();
      if (url.includes('/rbac/')) return route.fulfill({ status: 200, contentType: 'application/json', body: JSON.stringify({ code: 0, data: { permissions: ['ai.action'] } }) });
      if (url.endsWith('/reviewed-market-permission-batch')) { imported = true; return route.fulfill({ status: 200, contentType: 'application/json', body: JSON.stringify({ code: 0, data: { imported: 1 } }) }); }
      if (url.includes('/v1/demand-cases?')) {
        const data = imported ? [{ id: 901, region: '测试地区', consumer: '测试消费者', need_scenario: '隔离场景', sales_channel: '只读渠道', status: 'evidence_missing', stop_condition: '关键证据缺失即停止' }] : [];
        return route.fulfill({ status: 200, contentType: 'application/json', body: JSON.stringify({ code: 0, data, total: data.length, page: 1, size: 100 }) });
      }
      return route.fulfill({ status: 200, contentType: 'application/json', body: JSON.stringify({ code: 0, data: {} }) });
    });
    await page.goto('/demand-cases');
    await expect(page.getByRole('heading', { name: '候选市场' })).toBeVisible();
    await page.getByRole('button', { name: '导入本轮已审阅研究' }).click();
    await expect(page.getByText('测试地区 × 测试消费者')).toBeVisible();
    await expect(page.getByText('证据不足')).toBeVisible();
  });

  test('experiment keeps final profit and cash recovery independent', async ({ page }) => {
    await page.route('**/api/**', async (route) => {
      const url = route.request().url();
      if (url.includes('/rbac/')) return route.fulfill({ status: 200, contentType: 'application/json', body: JSON.stringify({ code: 0, data: { permissions: [] } }) });
      if (url.includes('/v1/experiments?')) {
        const data = [{ experiment_id: 'exp_e2e_001', name: '隔离经营实验', stage: 'order', final_profit_status: 'final', cash_recovery_status: 'receivable', updated_at: '2026-07-12T00:00:00Z' }];
        return route.fulfill({ status: 200, contentType: 'application/json', body: JSON.stringify({ code: 0, data, total: 1, page: 1, size: 100 }) });
      }
      return route.fulfill({ status: 422, contentType: 'application/json', body: JSON.stringify({ code: 422, message: 'paid_order requires a paid linked order' }) });
    });
    await page.goto('/experiments');
    await expect(page.getByRole('heading', { name: '经营实验案卷' })).toBeVisible();
    await expect(page.getByText('隔离经营实验')).toBeVisible();
    await expect(page.getByText('final')).toBeVisible();
    await expect(page.getByText('receivable')).toBeVisible();
    await expect(page.getByText('recovered')).toHaveCount(0);
  });
});

import { test, expect } from '@playwright/test';

test('旧 Owner 模拟工作台跳转到平台真相', async ({ page }) => {
  await page.addInitScript(() => {
    localStorage.setItem('token', 'owner-route-retirement-test');
  });
  await page.route('**/api/v1/rbac/current/permissions', async (route) => {
    await route.fulfill({
      status: 200,
      contentType: 'application/json',
      body: JSON.stringify({ code: 0, data: { permissions: [] } }),
    });
  });

  await page.goto('/owner');
  await expect(page).toHaveURL(/\/platform-truth$/);
});

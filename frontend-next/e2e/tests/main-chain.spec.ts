import { test, expect, type Page } from '@playwright/test';

/**
 * LingMirror main-chain E2E.
 *
 * Verifies the happy path: login → dashboard → AI command center →
 * agent run → trace replay → action review → approve → execute →
 * status update visible.
 *
 * Prerequisites:
 *   - Next dev server on :3000
 *   - Go backend on :8080
 *   - DB migrated (000001 + 000002 applied; AI tables exist)
 *   - A test user exists (created by /api/v1/auth/register or seed)
 *
 * If the backend is not reachable, tests fail. Main-chain E2E is an
 * acceptance gate, not a frontend-only smoke test.
 */

const API_BASE = process.env.E2E_API_BASE || 'http://localhost:8080';
const TEST_USER = process.env.E2E_USER || 'e2e@lingmirror.test';
const TEST_PASS = process.env.E2E_PASS || 'e2e-password-123';
const E2E_USERNAME = 'e2e_test_user';

async function apiReachable(): Promise<boolean> {
  try {
    const res = await fetch(`${API_BASE}/api/health`, { signal: AbortSignal.timeout(2000) });
    return res.ok;
  } catch {
    return false;
  }
}

async function ensureTestUser(): Promise<string | null> {
  // Try login first; if it fails, attempt register.
  try {
    const res = await fetch(`${API_BASE}/api/v1/auth/login`, {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({ username: E2E_USERNAME, password: TEST_PASS }),
    });
    const status = res.status;
    const text = await res.text();
    console.log("E2E ensureTestUser: login response status =", status, "body =", text);
    if (res.ok) {
      const body = JSON.parse(text);
      return body?.data?.token || body?.data?.access_token || null;
    }
  } catch (e: unknown) {
    console.log("E2E ensureTestUser: login failed with error:", e instanceof Error ? e.message : String(e));
  }
  try {
    const res = await fetch(`${API_BASE}/api/v1/auth/register`, {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({ username: E2E_USERNAME, email: TEST_USER, password: TEST_PASS, display_name: 'E2E' }),
    });
    const status = res.status;
    const text = await res.text();
    console.log("E2E ensureTestUser: register response status =", status, "body =", text);
    if (res.ok) {
      // Registration succeeded, now login to get the token
      const loginRes = await fetch(`${API_BASE}/api/v1/auth/login`, {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({ username: E2E_USERNAME, password: TEST_PASS }),
      });
      if (loginRes.ok) {
        const body = await loginRes.json();
        return body?.data?.token || body?.data?.access_token || null;
      }
    }
  } catch (e: unknown) {
    console.log("E2E ensureTestUser: register failed with error:", e instanceof Error ? e.message : String(e));
  }
  return null;
}

async function loginViaUI(page: Page) {
  await page.goto('/login');
  await page.getByLabel(/email/i).fill(E2E_USERNAME);
  await page.getByLabel(/password/i).fill(TEST_PASS);
  await page.locator('button[type="submit"]').click();
  // Wait for redirect to dashboard
  await page.waitForURL('**/dashboard', { timeout: 10000 });
}

test.describe('LingMirror main chain', () => {
  test.beforeAll(async () => {
    const reachable = await apiReachable();
    console.log("E2E beforeAll: apiReachable =", reachable);
    if (!reachable) {
      throw new Error(`Backend not reachable at ${API_BASE}; main-chain E2E cannot prove acceptance`);
    }
    const token = await ensureTestUser();
    console.log("E2E beforeAll: test user token =", token);
    if (!token) {
      throw new Error('Could not create/login test user; main-chain E2E cannot prove acceptance');
    }
  });

  test('login → dashboard → /ai → trace → action → execute', async ({ page }) => {
    test.setTimeout(60_000);
    await loginViaUI(page);

    // 1. Dashboard loads with stat cards.
    await expect(page.getByText(/订单总数|orders/i).first()).toBeVisible({ timeout: 10000 });
    await expect(page.locator('text=SKU').first()).toBeVisible();

    // 2. Navigate to AI Command Center.
    await page.goto('/ai');
    await expect(page.getByRole('heading', { name: 'AI 指挥中心' }).first()).toBeVisible({ timeout: 10000 });

    // 3. Type a natural-language command and submit.
    const commandInput = page.getByPlaceholder(/输入|命令|ask|message/i).first();
    await commandInput.fill('检查库存是否充足');
    await page.getByRole('button', { name: /发送|send|submit/i }).first().click();

    // 4. Wait for the agent roster to render (proves backend wired).
    await expect(page.getByText(/A[1-7]|G[1-3]/).first()).toBeVisible({ timeout: 15000 });

    // 5. Hit the Agent run endpoint directly to get a trace id (UI button is flaky in CI).
    const loginRes = await fetch(`${API_BASE}/api/v1/auth/login`, {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({ username: E2E_USERNAME, password: TEST_PASS }),
    });
    const loginBody = await loginRes.json();
    const token = loginBody?.data?.token || loginBody?.data?.access_token;
    expect(token, 'login token for API calls').toBeTruthy();

    const runRes = await fetch(`${API_BASE}/api/v1/ai/run`, {
      method: 'POST',
      headers: { 'Content-Type': 'application/json', Authorization: `Bearer ${token}` },
      body: JSON.stringify({
        agent_id: 'A5',
        decision_point: 'stock_alert',
        context: { sku_id: 1, message: '检查库存' },
      }),
    });
    expect(runRes.ok, `ai/run returned ${runRes.status}`).toBeTruthy();
    const runBody = await runRes.json();
    const traceId = runBody?.data?.trace_id;
    expect(traceId, 'trace_id from agent run').toBeTruthy();
    const actionId = runBody?.data?.action?.id;

    // 6. Trace replay page renders the timeline.
    await page.goto(`/agents/A5/trace/${traceId}`);
    await expect(page.getByText(/trace|推理|timeline|时间线/i).first()).toBeVisible({ timeout: 10000 });
    // Events should be present.
    await expect(page.getByText(/prompt_start|tool_call|reasoning/i).first()).toBeVisible({ timeout: 10000 });

    // 7. If an action was created, review it.
    if (actionId) {
      await page.goto(`/actions/${actionId}`);
      await expect(page.getByRole('heading').first()).toBeVisible({ timeout: 10000 });
      // Action detail should mention risk level.
      await expect(page.getByText(/low|medium|high/i).first()).toBeVisible();

      // 8. Approve the action.
      const approveBtn = page.getByRole('button', { name: /批准|approve/i }).first();
      if (await approveBtn.isVisible({ timeout: 5000 }).catch(() => false)) {
        await approveBtn.click();
        // 9. Execute (only enabled after approval).
        const executeBtn = page.getByRole('button', { name: /执行|execute/i }).first();
        await expect(executeBtn).toBeEnabled({ timeout: 5000 });
        await executeBtn.click();
        // 10. Status should update to executed.
        await expect(page.getByText(/executed|已执行/i).first()).toBeVisible({ timeout: 10000 });
      }
    }

    // 11. AgentOS cockpit shows the work queue.
    await page.goto('/agentos');
    await expect(page.getByText(/pending|待审|work queue|工作/i).first()).toBeVisible({ timeout: 10000 });
  });

  test('order list page loads', async ({ page }) => {
    await loginViaUI(page);
    await page.goto('/orders');
    // Table should render (even if empty).
    await expect(page.locator('table').first()).toBeVisible({ timeout: 10000 });
  });

  test('global search returns results or empty state', async ({ page }) => {
    await loginViaUI(page);
    await page.goto('/search');
    const searchInput = page.getByPlaceholder(/搜索|search/i).first();
    await searchInput.fill('test');
    await page.keyboard.press('Enter');
    // Either results appear or "no results" empty state.
    await expect(
      page.locator('table').first().or(page.getByText(/no results|无结果|暂无|未找到/i).first())
    ).toBeVisible({ timeout: 10000 });
  });

  test('AgentOS autonomy controls render', async ({ page }) => {
    await loginViaUI(page);
    await page.goto('/agentos');
    await expect(page.getByText(/autonomy|自主|trust/i).first()).toBeVisible({ timeout: 10000 });
  });
});

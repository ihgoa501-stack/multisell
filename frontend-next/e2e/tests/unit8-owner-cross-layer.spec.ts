import { expect, test, type APIRequestContext } from '@playwright/test';

const enabled = process.env.E2E_UNIT8_LIVE === '1';
const apiBase = process.env.E2E_API_BASE ?? 'http://127.0.0.1:18080';
const ownerUsername = process.env.E2E_UNIT8_OWNER ?? 'unit8_owner';
const otherUsername = process.env.E2E_UNIT8_OTHER ?? 'unit8_other';
const password = process.env.E2E_UNIT8_PASSWORD ?? 'unit8-password-123';
const factID = Number(process.env.E2E_UNIT8_FACT_ID ?? '1');

type LoginData = { access_token: string; refresh_token: string; user: { id: number } };

async function login(request: APIRequestContext, username: string): Promise<LoginData> {
  const response = await request.post(`${apiBase}/api/v1/auth/login`, {
    data: { username, password },
  });
  expect(response.ok(), `login ${username}: ${response.status()} ${await response.text()}`).toBeTruthy();
  return (await response.json()).data as LoginData;
}

function auth(token: string, extra: Record<string, string> = {}) {
  return { Authorization: `Bearer ${token}`, ...extra };
}

test.describe('Unit 8 real PostgreSQL Owner cross-layer acceptance', () => {
  test.skip(!enabled, 'Set E2E_UNIT8_LIVE=1; this suite requires an isolated migrated PostgreSQL database.');

  test('JWT/RBAC → mock fact → approval → audit → browser refresh recovery', async ({ page, request }) => {
    test.setTimeout(90_000);
    const owner = await login(request, ownerUsername);
    const other = await login(request, otherUsername);
    const nonce = `${Date.now()}-${Math.random().toString(16).slice(2)}`;

    const create = await request.post(`${apiBase}/api/v1/business-decisions`, {
      headers: auth(owner.access_token, { 'Idempotency-Key': `unit8-case-${nonce}` }),
      data: {
        question: '是否对 MOCK 订单采取下一步行动？',
        target: '仅验证 Owner 决策安全链，不执行外部写',
        object_type: 'platform_order_ingest',
        object_id: factID,
        unknowns: ['真实渠道结果未知'],
        idempotency_key: `unit8-case-${nonce}`,
      },
    });
    expect(create.ok(), `${create.status()} ${await create.text()}`).toBeTruthy();
    const decisionCase = (await create.json()).data as { id: number; manifest_sha256: string; truth_status: string };
    expect(decisionCase.truth_status).toBe('mock');

    const recommend = await request.post(`${apiBase}/api/v1/business-decisions/${decisionCase.id}/ai-recommendations`, {
      headers: auth(owner.access_token, { 'Idempotency-Key': `unit8-ai-${nonce}` }),
      data: {
        recommendation: '先补充证据，不执行动作',
        rationale: '当前订单明确为本地 MOCK 事实',
        truth_status: 'inferred',
        unknowns: ['真实订单表现未知'],
        idempotency_key: `unit8-ai-${nonce}`,
      },
    });
    expect(recommend.ok()).toBeTruthy();

    const crossOwner = await request.get(`${apiBase}/api/v1/business-decisions/${decisionCase.id}`, {
      headers: auth(other.access_token),
    });
    expect(crossOwner.status()).toBe(404);

    const failedKey = `unit8-decision-failed-${nonce}`;
    const missingApproval = await request.post(`${apiBase}/api/v1/business-decisions/${decisionCase.id}/owner-decisions`, {
      headers: auth(owner.access_token, { 'Idempotency-Key': failedKey }),
      data: {
        decision: 'request_more_evidence',
        reason: 'mock 不得伪装为真实结果',
        manifest_sha256: decisionCase.manifest_sha256,
        idempotency_key: failedKey,
      },
    });
    expect(missingApproval.status()).toBe(403);

    const afterFailure = await request.get(`${apiBase}/api/v1/business-decisions/${decisionCase.id}`, {
      headers: auth(owner.access_token),
    });
    expect(((await afterFailure.json()).data.owner_decisions as unknown[])).toHaveLength(0);

    const approvalCreate = await request.post(`${apiBase}/api/v1/approval`, {
      headers: auth(owner.access_token, { 'Idempotency-Key': `unit8-approval-${nonce}` }),
      data: {
        product_id: factID,
        request_type: 'agent_action',
        reason: 'Unit8 Owner mock decision acceptance',
        target_type: 'business-decision',
        target_id: decisionCase.id,
        risk_level: 'high',
      },
    });
    expect(approvalCreate.ok()).toBeTruthy();
    const approvalID = (await approvalCreate.json()).data.id as number;

    const review = await request.put(`${apiBase}/api/v1/approval/${approvalID}/review`, {
      headers: auth(owner.access_token, { 'Idempotency-Key': `unit8-review-${nonce}` }),
      data: { action: 'approve', review_note: '只批准保存 mock 补证决定，不授权外部写' },
    });
    expect(review.ok()).toBeTruthy();

    const successKey = `unit8-decision-success-${nonce}`;
    const decided = await request.post(`${apiBase}/api/v1/business-decisions/${decisionCase.id}/owner-decisions`, {
      headers: auth(owner.access_token, {
        'X-Approval-ID': String(approvalID),
        'Idempotency-Key': successKey,
      }),
      data: {
        decision: 'request_more_evidence',
        reason: 'mock 不得伪装为真实结果',
        manifest_sha256: decisionCase.manifest_sha256,
        idempotency_key: successKey,
      },
    });
    expect(decided.ok(), `${decided.status()} ${await decided.text()}`).toBeTruthy();

    const replay = await request.post(`${apiBase}/api/v1/business-decisions/${decisionCase.id}/owner-decisions`, {
      headers: auth(owner.access_token, {
        'X-Approval-ID': String(approvalID),
        'Idempotency-Key': successKey,
      }),
      data: {
        decision: 'request_more_evidence',
        reason: 'mock 不得伪装为真实结果',
        manifest_sha256: decisionCase.manifest_sha256,
        idempotency_key: successKey,
      },
    });
    expect(replay.status()).toBe(409);

    const audit = await request.get(`${apiBase}/api/v1/operation-log`, {
      headers: auth(owner.access_token),
      params: { module: 'business-decisions', page: '1', size: '100' },
    });
    expect(audit.ok()).toBeTruthy();
    const logs = (await audit.json()).data as Array<{ result: string; content: string }>;
    expect(logs.some((row) => row.result === 'failure' && row.content.includes(failedKey))).toBeTruthy();
    expect(logs.some((row) => row.result === 'success' && row.content.includes(successKey))).toBeTruthy();

    await page.goto('/login');
    await page.evaluate(({ access, refresh, user }) => {
      localStorage.setItem('token', access);
      localStorage.setItem('refresh_token', refresh);
      localStorage.setItem('user', JSON.stringify(user));
    }, { access: owner.access_token, refresh: owner.refresh_token, user: { id: owner.user.id, username: ownerUsername, role: 'admin' } });
    await page.goto(`/business-decisions/${decisionCase.id}`);
    await expect(page.getByText('经营决策案卷 #' + decisionCase.id)).toBeVisible();
    await expect(page.getByText('模拟', { exact: true })).toBeVisible();
    await expect(page.getByText('mock 不得伪装为真实结果')).toBeVisible();
    await page.reload();
    await expect(page.getByText('mock 不得伪装为真实结果')).toBeVisible();

    const refreshLogin = await login(request, ownerUsername);
    await page.evaluate(({ refresh, user }) => {
      localStorage.setItem('token', 'deliberately-invalid-access-token');
      localStorage.setItem('refresh_token', refresh);
      localStorage.setItem('user', JSON.stringify(user));
    }, { refresh: refreshLogin.refresh_token, user: { id: owner.user.id, username: ownerUsername, role: 'admin' } });
    const refreshed = page.waitForResponse((response) => response.url().endsWith('/api/v1/auth/refresh') && response.status() === 200);
    await page.reload();
    await refreshed;
    await expect(page.getByText('mock 不得伪装为真实结果')).toBeVisible();
    expect(await page.evaluate(() => localStorage.getItem('token'))).not.toBe('deliberately-invalid-access-token');
  });
});

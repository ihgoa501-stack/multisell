import { readFileSync } from 'node:fs';
import { describe, expect, it } from 'vitest';

const source = readFileSync(`${process.cwd()}/src/app/(main)/approval/page.tsx`, 'utf8');

describe('Owner approval request workspace', () => {
  it('can create an exact pending approval before review and one-time execution', () => {
    expect(source).toMatch(/apiClient\.post<ApprovalRequest>\(["']\/v1\/approval["']/);
    expect(source).toContain('request_type');
    expect(source).toContain('target_type');
    expect(source).toContain('target_id');
    expect(source).toContain('申请一次性审批');
  });
});

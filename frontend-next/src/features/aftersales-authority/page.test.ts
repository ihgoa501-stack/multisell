import { readFileSync } from "node:fs";
import { describe, expect, it } from "vitest";
const source = readFileSync(
  `${process.cwd()}/src/app/(main)/aftersales/page.tsx`,
  "utf8",
);
describe("Owner aftersales authority workspace", () => {
  it("uses only the resolution authority write chain", () => {
    expect(source).toContain("/v1/aftersales/resolutions");
    expect(source).toContain("/decisions");
    expect(source).toContain("/executions");
    expect(source).toContain("/receipts");
    expect(source).not.toContain("/${id}/refund");
    expect(source).not.toContain("/${id}/auto-decide");
  });
  it("labels legacy shortcuts disabled and exposes evidence hashes", () => {
    expect(source).toContain("legacy disabled");
    expect(source).toContain("receipt_sha256");
    expect(source).toContain("不可变终态证据");
    expect(source).toContain("consequence_status");
  });
  it("gates actions by the authoritative state machine", () => {
    expect(source).toMatch(/c\.status\s*!==\s*["']requested["']/);
    expect(source).toMatch(/c\.status\s*!==\s*["']approved["']/);
    expect(source).toMatch(/c\.status\s*!==\s*["']execution_submitted["']/);
  });
  it("uses the one-time HTTP approval contract for every authoritative write", () => {
    expect(source.match(/postApproved/g)?.length).toBe(4);
    expect(source).toMatch(/body\.target_id\s*=\s*v\.order_id/);
    expect(source).toContain("approvalExecution");
    expect(source).toContain("approval_id");
    expect(source).toContain("idempotencyKey");
  });
});

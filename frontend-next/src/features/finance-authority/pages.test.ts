import { readFileSync } from "node:fs";
import { describe, expect, it } from "vitest";
import { normalizeFinalProfitVersion } from "@/app/(main)/profit/page";
const root = `${process.cwd()}/src/app/(main)`;
const settlement = readFileSync(`${root}/settlement/page.tsx`, "utf8");
const profit = readFileSync(`${root}/profit/page.tsx`, "utf8");
const finance = readFileSync(`${root}/finance/page.tsx`, "utf8");
describe("Unit 5 Owner authority workspaces", () => {
  it("uses immutable platform settlement APIs and minor units", () => {
    expect(settlement).toContain("/v1/settlement/platform-accounts/");
    expect(settlement).toContain("/v1/settlement/platform-facts/");
    expect(settlement).toContain("amount_minor");
    expect(settlement).toContain("原始凭证 JSON");
    expect(settlement).toContain("legacy");
  });
  it("binds exact cost versions and exposes append-only final profit versions", () => {
    expect(profit).toContain("/cost-allocations");
    expect(profit).toContain("/finalize");
    expect(profit).toContain("/final-versions");
    expect(profit).toContain("source_manifest_sha256");
    expect(profit).toContain("legacy");
  });
  it("normalizes the current Go wire shape without losing minor-unit facts", () => {
    expect(
      normalizeFinalProfitVersion({
        ID: 4,
        Version: 2,
        Currency: "USD",
        RevenueMinor: 1200,
        ProfitMinor: 300,
        SourceManifestSHA256: "a".repeat(64),
      }),
    ).toMatchObject({
      id: 4,
      version: 2,
      currency: "USD",
      revenue_minor: 1200,
      profit_minor: 300,
      source_manifest_sha256: "a".repeat(64),
    });
  });
  it("keeps cash observations separate and sends reconciliation through the approval protocol", () => {
    expect(finance).toContain("/v1/finance/cash-receipts");
    expect(finance).toMatch(/postApproved\([\s\S]*?\/v1\/finance\/cash-reconciliations/);
    expect(finance).toMatch(/target_id:\s*v\.cash_receipt_id/);
    expect(finance).toContain("approvalId");
    expect(finance).toContain("idempotencyKey");
    expect(finance).toContain("对账被阻断");
    expect(finance).toContain("raw_payload_sha256");
    expect(finance).toContain("legacy");
  });
});

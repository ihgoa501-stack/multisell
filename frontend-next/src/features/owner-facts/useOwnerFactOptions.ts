"use client";
import { useQuery } from "@tanstack/react-query";
import { apiClient } from "@/lib/api-client";

export type OwnerFactOptions = {
  accounts: Array<{ id: number; platform_code: string; store_name: string }>;
  orders: Array<{
    id: number;
    account_id: number;
    external_order_id: string;
    platform_code: string;
    currency: string;
    observed_at: string;
  }>;
  order_items: Array<{
    id: number;
    order_id: number;
    sku_id: number;
    sku_code: string;
    product_name: string;
    quantity: number;
  }>;
  cost_versions: Array<{
    id: number;
    internal_sku_id: number;
    version: number;
    total_minor: number;
    currency: string;
  }>;
  settlements: Array<{
    id: number;
    account_id: number;
    external_settlement_id: string;
    platform_code: string;
    currency: string;
    receivable_minor: number;
    observed_at: string;
  }>;
  cash_receipts: Array<{
    id: number;
    external_receipt_id: string;
    currency: string;
    amount_minor: number;
    reconciliation_status: string;
    observed_at: string;
  }>;
  finance_accounts: Array<{
    id: number;
    name: string;
    account_type: string;
    currency: string;
    status: string;
  }>;
  aftersales_cases: Array<{
    id: number;
    order_id: number;
    kind: string;
    status: string;
    requested_minor: number;
    currency: string;
  }>;
};
export function useOwnerFactOptions() {
  return useQuery({
    queryKey: ["owner-fact-options"],
    queryFn: async () =>
      (
        await apiClient.get<OwnerFactOptions>(
          "/v1/platform-integrations/owner-fact-options",
        )
      ).data!,
  });
}
export const orderLabel = (o: OwnerFactOptions["orders"][number]) =>
  `${o.platform_code} · ${o.external_order_id} · #${o.id}`;
export const accountLabel = (a: OwnerFactOptions["accounts"][number]) =>
  `${a.platform_code}${a.store_name ? ` · ${a.store_name}` : ""} · #${a.id}`;
export const settlementLabel = (s: OwnerFactOptions["settlements"][number]) =>
  `${s.platform_code} · ${s.external_settlement_id} · ${s.receivable_minor} ${s.currency} · #${s.id}`;
export const cashLabel = (c: OwnerFactOptions["cash_receipts"][number]) =>
  `${c.external_receipt_id} · ${c.amount_minor} ${c.currency} · #${c.id}`;

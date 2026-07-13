"use client";
import { useState } from "react";
import {
  Alert,
  Button,
  Card,
  Form,
  Select,
  Space,
  Table,
  Tag,
  Typography,
  message,
} from "antd";
import { useMutation, useQuery } from "@tanstack/react-query";
import apiClient from "@/lib/api-client";
import {
  orderLabel,
  useOwnerFactOptions,
} from "@/features/owner-facts/useOwnerFactOptions";
type Version = {
  id: number;
  version: number;
  currency: string;
  revenue_minor: number;
  product_cost_minor: number;
  settlement_fee_minor: number;
  fulfillment_fee_minor: number;
  refund_minor: number;
  total_cost_minor: number;
  profit_minor: number;
  source_manifest_sha256: string;
  finalized_at: string;
};
type VersionWire = Partial<Version> & {
  ID?: number;
  Version?: number;
  Currency?: string;
  RevenueMinor?: number;
  ProductCostMinor?: number;
  SettlementFeeMinor?: number;
  FulfillmentFeeMinor?: number;
  RefundMinor?: number;
  TotalCostMinor?: number;
  ProfitMinor?: number;
  SourceManifestSHA256?: string;
  FinalizedAt?: string;
};
export const normalizeFinalProfitVersion = (v: VersionWire): Version => ({
  id: v.id ?? v.ID ?? 0,
  version: v.version ?? v.Version ?? 0,
  currency: v.currency ?? v.Currency ?? "",
  revenue_minor: v.revenue_minor ?? v.RevenueMinor ?? 0,
  product_cost_minor: v.product_cost_minor ?? v.ProductCostMinor ?? 0,
  settlement_fee_minor: v.settlement_fee_minor ?? v.SettlementFeeMinor ?? 0,
  fulfillment_fee_minor: v.fulfillment_fee_minor ?? v.FulfillmentFeeMinor ?? 0,
  refund_minor: v.refund_minor ?? v.RefundMinor ?? 0,
  total_cost_minor: v.total_cost_minor ?? v.TotalCostMinor ?? 0,
  profit_minor: v.profit_minor ?? v.ProfitMinor ?? 0,
  source_manifest_sha256:
    v.source_manifest_sha256 ?? v.SourceManifestSHA256 ?? "",
  finalized_at: v.finalized_at ?? v.FinalizedAt ?? "",
});
const amount = (n: number, c: string) => `${c} ${(n / 100).toFixed(2)}`;
export default function ProfitPage() {
  const facts = useOwnerFactOptions();
  const [orderId, setOrderId] = useState<number>();
  const versions = useQuery({
    queryKey: ["final-profit-versions", orderId],
    queryFn: async () =>
      (
        (
          await apiClient.get<VersionWire[]>(
            `/v1/profit/order/${orderId}/final-versions`,
          )
        ).data ?? []
      ).map(normalizeFinalProfitVersion),
    enabled: !!orderId,
  });
  const allocate = useMutation({
    mutationFn: async (v: {
      order_item_id: number;
      sourcing_cost_version_id: number;
    }) =>
      (await apiClient.post(`/v1/profit/order/${orderId}/cost-allocations`, v))
        .data,
    onSuccess: () => message.success("精确成本版本已冻结到订单行"),
  });
  const finalize = useMutation({
    mutationFn: async () =>
      (await apiClient.post<Version>(`/v1/profit/order/${orderId}/finalize`))
        .data,
    onSuccess: () => {
      message.success("已生成新的不可变最终利润版本");
      versions.refetch();
    },
  });
  return (
    <main className="p-4 md:p-5">
      <Typography.Title level={1}>订单最终利润</Typography.Title>
      <Alert
        type="info"
        showIcon
        message="权威利润只来自同一订单的可信事实"
        description="必须先绑定每个订单行的 actual 成本版本，再由平台结算、履约费用和售后终局共同复算。旧商品利润汇总是 legacy 浮点估算，不是最终利润。"
      />
      <Card className="mt-4" title="选择订单">
        <Space wrap>
          <Select
            showSearch
            optionFilterProp="label"
            aria-label="选择权威订单"
            style={{ width: 360 }}
            value={orderId}
            options={(facts.data?.orders ?? []).map((o) => ({
              value: o.id,
              label: orderLabel(o),
            }))}
            onChange={setOrderId}
          />
          <Button onClick={() => versions.refetch()} disabled={!orderId}>
            读取版本
          </Button>
        </Space>
      </Card>
      <Card className="mt-4" title="冻结订单行成本">
        <Form layout="inline" onFinish={(v) => allocate.mutate(v)}>
          <Form.Item
            name="order_item_id"
            label="订单行"
            rules={[{ required: true }]}
          >
            <Select
              style={{ width: 300 }}
              options={(facts.data?.order_items ?? [])
                .filter((i) => i.order_id === orderId)
                .map((i) => ({
                  value: i.id,
                  label: `${i.product_name} · ${i.sku_code || `SKU #${i.sku_id}`} · ×${i.quantity}`,
                }))}
            />
          </Form.Item>
          <Form.Item
            name="sourcing_cost_version_id"
            label="actual 成本版本"
            rules={[{ required: true }]}
          >
            <Select
              style={{ width: 300 }}
              options={(facts.data?.cost_versions ?? []).map((v) => ({
                value: v.id,
                label: `SKU #${v.internal_sku_id} · v${v.version} · ${v.total_minor} ${v.currency}`,
              }))}
            />
          </Form.Item>
          <Button
            type="primary"
            htmlType="submit"
            disabled={!orderId}
            loading={allocate.isPending}
          >
            绑定精确成本
          </Button>
        </Form>
        {allocate.error && (
          <Alert
            className="mt-4"
            type="error"
            showIcon
            message="成本未绑定"
            description={allocate.error.message}
          />
        )}
      </Card>
      <Card
        className="mt-4"
        title="复算并冻结最终利润"
        extra={
          <Button
            type="primary"
            danger
            disabled={!orderId}
            loading={finalize.isPending}
            onClick={() => finalize.mutate()}
          >
            生成新版本
          </Button>
        }
      >
        {finalize.error && (
          <Alert
            type="warning"
            showIcon
            message="当前不能形成最终利润"
            description={finalize.error.message}
          />
        )}
        <Table
          className="mt-4"
          rowKey="id"
          loading={versions.isLoading}
          locale={{
            emptyText: "尚无最终利润版本；缺少任一权威事实时系统会阻断。",
          }}
          dataSource={versions.data ?? []}
          columns={[
            { title: "版本", dataIndex: "version" },
            {
              title: "收入",
              render: (_, r) => amount(r.revenue_minor, r.currency),
            },
            {
              title: "商品成本",
              render: (_, r) => amount(r.product_cost_minor, r.currency),
            },
            {
              title: "平台/履约/退款",
              render: (_, r) =>
                amount(
                  r.settlement_fee_minor +
                    r.fulfillment_fee_minor +
                    r.refund_minor,
                  r.currency,
                ),
            },
            {
              title: "最终利润",
              render: (_, r) => (
                <Tag
                  color={
                    r.profit_minor > 0
                      ? "success"
                      : r.profit_minor < 0
                        ? "error"
                        : "default"
                  }
                >
                  {amount(r.profit_minor, r.currency)}
                </Tag>
              ),
            },
            {
              title: "来源 manifest",
              render: (_, r) => (
                <Typography.Text copyable ellipsis style={{ maxWidth: 180 }}>
                  {r.source_manifest_sha256}
                </Typography.Text>
              ),
            },
            { title: "冻结时间", dataIndex: "finalized_at" },
          ]}
        />
      </Card>
    </main>
  );
}

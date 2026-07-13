"use client";
import { useState } from "react";
import {
  Alert,
  Button,
  Card,
  Descriptions,
  Form,
  Input,
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
  accountLabel,
  settlementLabel,
  useOwnerFactOptions,
} from "@/features/owner-facts/useOwnerFactOptions";

type FactDetail = {
  ingest: {
    id: number;
    account_id: number;
    platform_code: string;
    external_event_id: string;
    external_settlement_id: string;
    truth_status: string;
    currency: string;
    payload_sha256: string;
    content_sha256: string;
    observed_at: string;
  };
  lines: Array<{
    id: number;
    external_order_id: string;
    kind: string;
    fee_code: string;
    amount_minor: number;
    currency: string;
    external_transaction_id: string;
  }>;
};
const money = (minor: number, currency: string) =>
  `${currency} ${(minor / 100).toFixed(2)} (${minor} minor)`;
export default function SettlementPage() {
  const facts = useOwnerFactOptions();
  const [lookup, setLookup] = useState<number>();
  const [form] = Form.useForm();
  const fact = useQuery({
    queryKey: ["platform-settlement-fact", lookup],
    queryFn: async () =>
      (
        await apiClient.get<FactDetail>(
          `/v1/settlement/platform-facts/${lookup}`,
        )
      ).data!,
    enabled: !!lookup,
  });
  const ingest = useMutation({
    mutationFn: async (v: Record<string, unknown>) => {
      let lines, raw_payload;
      try {
        lines = JSON.parse(String(v.lines_json));
        raw_payload = JSON.parse(String(v.raw_payload_json));
      } catch {
        throw new Error("原始凭证与明细必须是有效 JSON");
      }
      return (
        await apiClient.post<{ ingest_id: number; replay: boolean }>(
          `/v1/settlement/platform-accounts/${v.account_id}/events`,
          {
            ...v,
            account_id: undefined,
            lines_json: undefined,
            raw_payload_json: undefined,
            lines,
            raw_payload,
          },
        )
      ).data!;
    },
    onSuccess: (r) => {
      setLookup(r.ingest_id);
      message.success(
        r.replay ? "已找到完全一致的既有凭证" : "结算事实已不可变保存",
      );
    },
  });
  return (
    <main className="p-4 md:p-5">
      <Typography.Title level={1}>平台结算事实</Typography.Title>
      <Alert
        showIcon
        type="info"
        message="权威入口"
        description="金额使用最小货币单位（minor，例如分），原始平台凭证与规范化明细同时保存并校验哈希。下方旧结算 CRUD 属于 legacy，不构成权威结算事实。"
      />
      <Card title="录入平台结算事件" className="mt-4">
        <Form
          form={form}
          layout="vertical"
          onFinish={(v) => ingest.mutate(v)}
          initialValues={{
            truth_status: "external_observed",
            currency: "USD",
            observed_at: new Date().toISOString(),
            raw_payload_json: "{}",
            lines_json: "[]",
          }}
        >
          <Space wrap align="start">
            <Form.Item
              name="account_id"
              label="Owner 平台账户"
              rules={[{ required: true }]}
            >
              <Select
                style={{ width: 300 }}
                loading={facts.isLoading}
                options={(facts.data?.accounts ?? []).map((a) => ({
                  value: a.id,
                  label: accountLabel(a),
                }))}
                onChange={(id) =>
                  form.setFieldValue(
                    "platform_code",
                    facts.data?.accounts.find((a) => a.id === id)
                      ?.platform_code,
                  )
                }
              />
            </Form.Item>
            <Form.Item
              name="platform_code"
              label="平台代码"
              rules={[{ required: true }]}
            >
              <Input aria-label="平台代码" readOnly />
            </Form.Item>
            <Form.Item
              name="external_event_id"
              label="外部事件 ID"
              rules={[{ required: true }]}
            >
              <Input />
            </Form.Item>
            <Form.Item
              name="external_settlement_id"
              label="外部结算 ID"
              rules={[{ required: true }]}
            >
              <Input />
            </Form.Item>
            <Form.Item
              name="currency"
              label="币种"
              rules={[{ required: true, len: 3 }]}
            >
              <Input maxLength={3} />
            </Form.Item>
            <Form.Item name="truth_status" label="事实等级">
              <Select
                style={{ width: 190 }}
                options={[
                  { value: "external_observed", label: "external_observed" },
                  { value: "mock", label: "mock（仅测试）" },
                ]}
              />
            </Form.Item>
          </Space>
          <Form.Item
            name="observed_at"
            label="平台观察时间（ISO 8601）"
            rules={[{ required: true }]}
          >
            <Input />
          </Form.Item>
          <Form.Item
            name="raw_payload_json"
            label="原始凭证 JSON"
            rules={[{ required: true }]}
          >
            <Input.TextArea rows={3} />
          </Form.Item>
          <Form.Item
            name="lines_json"
            label="规范化明细 JSON 数组"
            extra="字段：external_line_id、external_order_id、kind、fee_code、amount_minor、currency、external_transaction_id、occurred_at"
            rules={[{ required: true }]}
          >
            <Input.TextArea rows={6} />
          </Form.Item>
          {ingest.error && (
            <Alert
              className="mb-4"
              type="error"
              showIcon
              message="未保存"
              description={ingest.error.message}
            />
          )}
          <Button htmlType="submit" type="primary" loading={ingest.isPending}>
            保存不可变结算事实
          </Button>
        </Form>
      </Card>
      <Card
        title="查验结算事实"
        className="mt-4"
        extra={
          <Space>
            <Select
              aria-label="选择结算事实"
              showSearch
              optionFilterProp="label"
              style={{ width: 360 }}
              value={lookup}
              options={(facts.data?.settlements ?? []).map((s) => ({
                value: s.id,
                label: settlementLabel(s),
              }))}
              onChange={setLookup}
            />
            <Button onClick={() => fact.refetch()} disabled={!lookup}>
              查验
            </Button>
          </Space>
        }
      >
        {fact.error ? (
          <Alert
            type="error"
            showIcon
            message="无法读取"
            description={fact.error.message}
          />
        ) : fact.data ? (
          <>
            <Descriptions
              bordered
              size="small"
              column={{ xs: 1, md: 2 }}
              items={[
                { key: "id", label: "事实 ID", children: fact.data.ingest.id },
                {
                  key: "truth",
                  label: "事实等级",
                  children: <Tag>{fact.data.ingest.truth_status}</Tag>,
                },
                {
                  key: "account",
                  label: "Owner 平台账户",
                  children: fact.data.ingest.account_id,
                },
                {
                  key: "identity",
                  label: "外部结算",
                  children: fact.data.ingest.external_settlement_id,
                },
                {
                  key: "observed",
                  label: "观察时间",
                  children: fact.data.ingest.observed_at,
                },
                {
                  key: "hash",
                  label: "内容 SHA-256",
                  children: (
                    <Typography.Text copyable>
                      {fact.data.ingest.content_sha256}
                    </Typography.Text>
                  ),
                },
              ]}
            />
            <Table
              className="mt-4"
              rowKey="id"
              pagination={false}
              dataSource={fact.data.lines}
              columns={[
                { title: "外部订单", dataIndex: "external_order_id" },
                { title: "类型", dataIndex: "kind" },
                { title: "费用代码", dataIndex: "fee_code" },
                {
                  title: "金额",
                  render: (_, r) => money(r.amount_minor, r.currency),
                },
                { title: "外部交易", dataIndex: "external_transaction_id" },
              ]}
            />
          </>
        ) : (
          <Alert
            type="warning"
            showIcon
            message="尚未选择事实"
            description="录入事件或输入已知事实 ID 后查验；系统不会把旧结算单自动升级为权威事实。"
          />
        )}
      </Card>
    </main>
  );
}

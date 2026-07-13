"use client";
import { useState } from "react";
import {
  Alert,
  Button,
  Card,
  Descriptions,
  Form,
  Input,
  InputNumber,
  Select,
  Space,
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
import { OwnerApprovalSelect } from "@/features/owner-facts/OwnerApprovalSelect";

type Resolution = {
  id: number;
  order_id: number;
  platform_account_id: number;
  kind: string;
  requested_minor: number;
  currency: string;
  reason: string;
  request_source: string;
  request_evidence_id: string;
  request_observed_at: string;
  status: string;
  decision_reason?: string;
  decided_by?: number;
  decided_at?: string;
  external_request_id?: string;
  submitted_at?: string;
  consequence_status: string;
  created_at: string;
};
type Receipt = {
  id: number;
  outcome: string;
  source_type: string;
  evidence_id: string;
  external_receipt_id: string;
  observed_at: string;
  actual_minor: number;
  currency: string;
  failure_code?: string;
  receipt_payload: unknown;
  receipt_sha256: string;
  recorded_at: string;
};
type Detail = { case: Resolution; receipt?: Receipt };
const cash = (minor: number, currency: string) =>
  `${currency} ${(minor / 100).toFixed(2)} (${minor} minor)`;
async function sha256(value: string) {
  const bytes = new TextEncoder().encode(value);
  const digest = await crypto.subtle.digest("SHA-256", bytes);
  return Array.from(new Uint8Array(digest), (b) =>
    b.toString(16).padStart(2, "0"),
  ).join("");
}
function approvalExecution(values: Record<string, unknown>) {
  const approvalId = Number(values.approval_id);
  const idempotencyKey = String(values.idempotency_key ?? "").trim();
  const body = { ...values };
  delete body.approval_id;
  return { body, execution: { approvalId, idempotencyKey } };
}

export default function AftersalesPage() {
  const facts = useOwnerFactOptions();
  const [selected, setSelected] = useState<number>();
  const detail = useQuery({
    queryKey: ["aftersales-resolution", selected],
    queryFn: async () =>
      (await apiClient.get<Detail>(`/v1/aftersales/resolutions/${selected}`))
        .data!,
    enabled: !!selected,
  });
  const refresh = () => detail.refetch();
  const create = useMutation({
    mutationFn: async (v: Record<string, unknown>) => {
      const { body, execution } = approvalExecution(v);
      body.target_id = v.order_id;
      return (
        await apiClient.postApproved<Resolution>(
          "/v1/aftersales/resolutions",
          body,
          execution,
        )
      ).data!;
    },
    onSuccess: (r) => {
      setSelected(r.id);
      message.success("售后请求证据已保存，等待 Owner 决定");
    },
  });
  const decide = useMutation({
    mutationFn: async (v: Record<string, unknown>) => {
      const { body, execution } = approvalExecution(v);
      return (
        await apiClient.postApproved(
          `/v1/aftersales/resolutions/${selected}/decisions`,
          body,
          execution,
        )
      ).data;
    },
    onSuccess: () => {
      message.success("Owner 决定已保存");
      refresh();
    },
  });
  const execute = useMutation({
    mutationFn: async (v: Record<string, unknown>) => {
      const { body, execution } = approvalExecution(v);
      return (
        await apiClient.postApproved(
          `/v1/aftersales/resolutions/${selected}/executions`,
          body,
          execution,
        )
      ).data;
    },
    onSuccess: () => {
      message.success("外部请求提交凭证已登记");
      refresh();
    },
  });
  const receipt = useMutation({
    mutationFn: async (v: Record<string, unknown>) => {
      const raw = String(v.receipt_payload_json);
      let receipt_payload;
      try {
        receipt_payload = JSON.parse(raw);
      } catch {
        throw new Error("终态回执必须是有效 JSON");
      }
      const canonical = JSON.stringify(receipt_payload);
      const { body, execution } = approvalExecution(v);
      return (
        await apiClient.postApproved(
          `/v1/aftersales/resolutions/${selected}/receipts`,
          {
            ...body,
            receipt_payload_json: undefined,
            receipt_payload,
            receipt_sha256: await sha256(canonical),
          },
          execution,
        )
      ).data;
    },
    onSuccess: () => {
      message.success("可信终态回执已不可变保存");
      refresh();
    },
  });
  const c = detail.data?.case;
  return (
    <main className="p-4 md:p-5">
      <Typography.Title level={1}>Owner 售后处置</Typography.Title>
      <Alert
        type="info"
        showIcon
        message="权威路径：请求证据 → Owner 决定 → 外部提交凭证 → 可信终态回执"
        description="退款、退货和争议不会因手工点击直接完成。旧 /aftersales/:id/refund 与 disputes 自动决策入口属于 legacy disabled，不能写入本工作台的权威终态。"
      />
      <Card className="mt-4" title="创建售后处置案卷">
        <Alert
          className="mb-4"
          type="warning"
          showIcon
          message="售后写入需要一次性审批"
          description="审批必须绑定当前 Owner 和本次订单目标；每一步使用新的已批准审批与幂等键。"
        />
        <Form
          layout="vertical"
          initialValues={{
            kind: "refund",
            currency: "USD",
            request_source: "buyer_request",
            observed_at: new Date().toISOString(),
          }}
          onFinish={(v) => create.mutate(v)}
        >
          <Space wrap align="start">
            <Form.Item
              name="order_id"
              label="权威订单"
              rules={[{ required: true }]}
            >
              <Select
                showSearch
                optionFilterProp="label"
                loading={facts.isLoading}
                style={{ width: 320 }}
                options={(facts.data?.orders ?? []).map((o) => ({
                  value: o.id,
                  label: orderLabel(o),
                }))}
              />
            </Form.Item>
            <Form.Item
              name="platform_account_id"
              label="平台账户"
              rules={[{ required: true }]}
            >
              <Select
                style={{ width: 260 }}
                options={(facts.data?.orders ?? []).map((o) => ({
                  value: o.account_id,
                  label: `${o.platform_code} · #${o.account_id}`,
                }))}
              />
            </Form.Item>
            <Form.Item name="kind" label="处置类型">
              <Select
                style={{ width: 140 }}
                options={[
                  { value: "refund", label: "退款" },
                  { value: "return", label: "退货" },
                  { value: "dispute", label: "争议" },
                ]}
              />
            </Form.Item>
            <Form.Item
              name="requested_minor"
              label="申请金额（minor）"
              rules={[{ required: true }]}
            >
              <InputNumber min={1} />
            </Form.Item>
            <Form.Item
              name="currency"
              label="币种"
              rules={[{ required: true, len: 3 }]}
            >
              <Input maxLength={3} />
            </Form.Item>
            <Form.Item name="request_source" label="请求来源">
              <Select
                style={{ width: 170 }}
                options={[
                  { value: "buyer_request", label: "买家请求" },
                  { value: "platform_request", label: "平台请求" },
                ]}
              />
            </Form.Item>
            <Form.Item
              name="approval_id"
              label="已批准审批 ID"
              rules={[{ required: true }]}
            >
              <OwnerApprovalSelect />
            </Form.Item>
          </Space>
          <Form.Item
            name="reason"
            label="请求原因"
            rules={[{ required: true }]}
          >
            <Input.TextArea rows={2} />
          </Form.Item>
          <Form.Item
            name="request_evidence_id"
            label="外部请求证据 ID"
            rules={[{ required: true }]}
          >
            <Input />
          </Form.Item>
          <Form.Item
            name="observed_at"
            label="观察时间（ISO 8601）"
            rules={[{ required: true }]}
          >
            <Input />
          </Form.Item>
          <Form.Item
            name="idempotency_key"
            label="幂等键"
            rules={[{ required: true, min: 8 }]}
          >
            <Input />
          </Form.Item>
          {create.error && (
            <Alert
              className="mb-4"
              type="error"
              showIcon
              message="案卷未创建"
              description={create.error.message}
            />
          )}
          <Button type="primary" htmlType="submit" loading={create.isPending}>
            保存请求事实
          </Button>
        </Form>
      </Card>
      <Card
        className="mt-4"
        title="查验处置案卷"
        extra={
          <Space>
            <Select
              showSearch
              optionFilterProp="label"
              style={{ width: 320 }}
              value={selected}
              aria-label="选择处置案卷"
              options={(facts.data?.aftersales_cases ?? []).map((c) => ({
                value: c.id,
                label: `${c.kind} · 订单 #${c.order_id} · ${c.status} · #${c.id}`,
              }))}
              onChange={setSelected}
            />
            <Button disabled={!selected} onClick={refresh}>
              读取
            </Button>
          </Space>
        }
      >
        {detail.error ? (
          <Alert
            type="error"
            showIcon
            message="无法读取案卷"
            description={detail.error.message}
          />
        ) : c ? (
          <>
            <Descriptions
              bordered
              size="small"
              column={{ xs: 1, md: 2 }}
              items={[
                { key: "id", label: "案卷 ID", children: c.id },
                {
                  key: "order",
                  label: "订单 / 平台账户",
                  children: `${c.order_id} / ${c.platform_account_id}`,
                },
                {
                  key: "status",
                  label: "状态",
                  children: <Tag>{c.status}</Tag>,
                },
                {
                  key: "amount",
                  label: "申请金额",
                  children: cash(c.requested_minor, c.currency),
                },
                {
                  key: "evidence",
                  label: "请求证据",
                  children: `${c.request_source} · ${c.request_evidence_id}`,
                },
                {
                  key: "decision",
                  label: "Owner 决定理由",
                  children: c.decision_reason || "尚未决定",
                },
                {
                  key: "execution",
                  label: "外部请求 ID",
                  children: c.external_request_id || "尚未提交",
                },
                {
                  key: "consequence",
                  label: "后续账务影响",
                  children: <Tag>{c.consequence_status}</Tag>,
                },
              ]}
            />
            <Card size="small" className="mt-4" title="1. Owner 批准或拒绝">
              <Form layout="inline" onFinish={(v) => decide.mutate(v)}>
                <Form.Item
                  name="decision"
                  label="决定"
                  rules={[{ required: true }]}
                >
                  <Select
                    style={{ width: 130 }}
                    options={[
                      { value: "approved", label: "批准" },
                      { value: "rejected", label: "拒绝" },
                    ]}
                  />
                </Form.Item>
                <Form.Item
                  name="reason"
                  label="理由"
                  rules={[{ required: true }]}
                >
                  <Input />
                </Form.Item>
                <Form.Item
                  name="approval_id"
                  label="已批准审批 ID"
                  rules={[{ required: true }]}
                >
                  <OwnerApprovalSelect />
                </Form.Item>
                <Form.Item
                  name="idempotency_key"
                  label="幂等键"
                  rules={[{ required: true, min: 8 }]}
                >
                  <Input />
                </Form.Item>
                <Button
                  htmlType="submit"
                  type="primary"
                  disabled={c.status !== "requested"}
                  loading={decide.isPending}
                >
                  保存 Owner 决定
                </Button>
              </Form>
              {decide.error && (
                <Alert
                  className="mt-3"
                  type="error"
                  showIcon
                  message="决定未保存"
                  description={decide.error.message}
                />
              )}
            </Card>
            <Card size="small" className="mt-4" title="2. 登记外部请求提交">
              <Form layout="inline" onFinish={(v) => execute.mutate(v)}>
                <Form.Item
                  name="external_request_id"
                  label="平台请求 ID"
                  rules={[{ required: true }]}
                >
                  <Input />
                </Form.Item>
                <Form.Item
                  name="approval_id"
                  label="已批准审批 ID"
                  rules={[{ required: true }]}
                >
                  <OwnerApprovalSelect />
                </Form.Item>
                <Form.Item
                  name="idempotency_key"
                  label="幂等键"
                  rules={[{ required: true, min: 8 }]}
                >
                  <Input />
                </Form.Item>
                <Button
                  htmlType="submit"
                  type="primary"
                  disabled={c.status !== "approved"}
                  loading={execute.isPending}
                >
                  登记已提交
                </Button>
              </Form>
              {execute.error && (
                <Alert
                  className="mt-3"
                  type="error"
                  showIcon
                  message="提交凭证未登记"
                  description={execute.error.message}
                />
              )}
            </Card>
            <Card size="small" className="mt-4" title="3. 登记可信终态回执">
              <Form
                layout="vertical"
                initialValues={{
                  outcome: "succeeded",
                  source_type: "platform_receipt",
                  currency: c.currency,
                  actual_minor: c.requested_minor,
                  observed_at: new Date().toISOString(),
                  receipt_payload_json: "{}",
                }}
                onFinish={(v) => receipt.mutate(v)}
              >
                <Space wrap align="start">
                  <Form.Item name="outcome" label="结果">
                    <Select
                      style={{ width: 130 }}
                      options={[
                        { value: "succeeded", label: "成功" },
                        { value: "failed", label: "失败" },
                      ]}
                    />
                  </Form.Item>
                  <Form.Item name="source_type" label="可信来源">
                    <Select
                      style={{ width: 210 }}
                      options={[
                        { value: "platform_receipt", label: "平台回执" },
                        {
                          value: "controlled_reconciliation",
                          label: "受控对账",
                        },
                      ]}
                    />
                  </Form.Item>
                  <Form.Item
                    name="evidence_id"
                    label="证据 ID"
                    rules={[{ required: true }]}
                  >
                    <Input />
                  </Form.Item>
                  <Form.Item
                    name="external_receipt_id"
                    label="外部回执 ID"
                    rules={[{ required: true }]}
                  >
                    <Input />
                  </Form.Item>
                  <Form.Item
                    name="actual_minor"
                    label="实际金额（minor）"
                    rules={[{ required: true }]}
                  >
                    <InputNumber min={0} />
                  </Form.Item>
                  <Form.Item
                    name="currency"
                    label="币种"
                    rules={[{ required: true }]}
                  >
                    <Input maxLength={3} />
                  </Form.Item>
                  <Form.Item
                    name="approval_id"
                    label="已批准审批 ID"
                    rules={[{ required: true }]}
                  >
                    <OwnerApprovalSelect />
                  </Form.Item>
                </Space>
                <Form.Item
                  name="observed_at"
                  label="观察时间（ISO 8601）"
                  rules={[{ required: true }]}
                >
                  <Input />
                </Form.Item>
                <Form.Item
                  name="idempotency_key"
                  label="幂等键"
                  rules={[{ required: true, min: 8 }]}
                >
                  <Input />
                </Form.Item>
                <Form.Item name="failure_code" label="失败代码（失败时必填）">
                  <Input />
                </Form.Item>
                <Form.Item
                  name="receipt_payload_json"
                  label="原始终态回执 JSON（SHA-256 由浏览器按规范 JSON 计算）"
                  rules={[{ required: true }]}
                >
                  <Input.TextArea rows={4} />
                </Form.Item>
                <Button
                  htmlType="submit"
                  type="primary"
                  danger
                  disabled={c.status !== "execution_submitted"}
                  loading={receipt.isPending}
                >
                  保存不可变终态回执
                </Button>
              </Form>
              {receipt.error && (
                <Alert
                  className="mt-3"
                  type="error"
                  showIcon
                  message="终态未形成"
                  description={receipt.error.message}
                />
              )}
            </Card>
            {detail.data?.receipt && (
              <Card size="small" className="mt-4" title="不可变终态证据">
                <Descriptions
                  bordered
                  size="small"
                  column={{ xs: 1, md: 2 }}
                  items={[
                    {
                      key: "outcome",
                      label: "终态",
                      children: <Tag>{detail.data.receipt.outcome}</Tag>,
                    },
                    {
                      key: "amount",
                      label: "实际金额",
                      children: cash(
                        detail.data.receipt.actual_minor,
                        detail.data.receipt.currency,
                      ),
                    },
                    {
                      key: "source",
                      label: "可信来源",
                      children: detail.data.receipt.source_type,
                    },
                    {
                      key: "evidence",
                      label: "证据 / 外部回执",
                      children: `${detail.data.receipt.evidence_id} / ${detail.data.receipt.external_receipt_id}`,
                    },
                    {
                      key: "observed",
                      label: "观察时间",
                      children: detail.data.receipt.observed_at,
                    },
                    {
                      key: "hash",
                      label: "Payload SHA-256",
                      children: (
                        <Typography.Text copyable>
                          {detail.data.receipt.receipt_sha256}
                        </Typography.Text>
                      ),
                    },
                  ]}
                />
              </Card>
            )}
          </>
        ) : (
          <Alert
            type="warning"
            showIcon
            message="尚未选择处置案卷"
            description="创建新案卷或输入已知案卷 ID；旧售后记录不会自动升级为权威处置事实。"
          />
        )}
      </Card>
    </main>
  );
}

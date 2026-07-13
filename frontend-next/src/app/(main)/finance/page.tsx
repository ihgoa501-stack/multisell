"use client";
import {
  Alert,
  Button,
  Card,
  Form,
  Input,
  InputNumber,
  Select,
  Space,
  Table,
  Tag,
  Typography,
  message,
} from "antd";
import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import apiClient from "@/lib/api-client";
import {
  cashLabel,
  settlementLabel,
  useOwnerFactOptions,
} from "@/features/owner-facts/useOwnerFactOptions";
import { OwnerApprovalSelect } from "@/features/owner-facts/OwnerApprovalSelect";
type Receipt = {
  id: number;
  finance_account_id: number;
  source_type: string;
  external_receipt_id: string;
  amount_minor: number;
  currency: string;
  observed_at: string;
  raw_payload_sha256: string;
  truth_status: string;
  reconciliation_status: string;
};
type Reconciliation = {
  id: number;
  cash_receipt_id: number;
  platform_settlement_ingest_id: number;
  amount_minor: number;
  currency: string;
  expected_receivable_minor: number;
  status: string;
  conflict_reason?: string;
  reconciled_at?: string;
};
const money = (n: number, c: string) =>
  `${c} ${(n / 100).toFixed(2)} (${n} minor)`;
export default function FinancePage() {
  const facts = useOwnerFactOptions();
  const qc = useQueryClient();
  const receipts = useQuery({
    queryKey: ["cash-receipts"],
    queryFn: async () =>
      (await apiClient.get<Receipt[]>("/v1/finance/cash-receipts")).data!,
  });
  const recs = useQuery({
    queryKey: ["cash-reconciliations"],
    queryFn: async () =>
      (
        await apiClient.get<Reconciliation[]>(
          "/v1/finance/cash-reconciliations",
        )
      ).data!,
  });
  const createReceipt = useMutation({
    mutationFn: async (v: Record<string, unknown>) => {
      let raw_payload;
      try {
        raw_payload = JSON.parse(String(v.raw_payload_json));
      } catch {
        throw new Error("原始到账凭证必须是有效 JSON");
      }
      return (
        await apiClient.post("/v1/finance/cash-receipts", {
          ...v,
          raw_payload_json: undefined,
          raw_payload,
        })
      ).data;
    },
    onSuccess: () => {
      message.success("到账事实已不可变保存");
      qc.invalidateQueries({ queryKey: ["cash-receipts"] });
    },
  });
  const reconcile = useMutation({
    mutationFn: async (v: Record<string, unknown>) => {
      const approvalId = Number(v.approval_id);
      const idempotencyKey = String(v.idempotency_key ?? "").trim();
      const body: Record<string, unknown> = {
        ...v,
        target_id: v.cash_receipt_id,
      };
      delete body.approval_id;
      return (
        await apiClient.postApproved("/v1/finance/cash-reconciliations", body, {
          approvalId,
          idempotencyKey,
        })
      ).data;
    },
    onSuccess: () => {
      message.success("对账结果已由服务器计算");
      qc.invalidateQueries({ queryKey: ["cash-receipts"] });
      qc.invalidateQueries({ queryKey: ["cash-reconciliations"] });
    },
  });
  return (
    <main className="p-4 md:p-5">
      <Typography.Title level={1}>现金回收与对账</Typography.Title>
      <Alert
        type="info"
        showIcon
        message="到账不等于结算已对账"
        description="银行或支付机构到账先作为 external_observed 独立保存；只有 Owner、对象、币种和完整金额一致，服务器才标记 reconciled。旧账户 balance 与手工交易属于 legacy 浮点账，不是权威现金事实。"
      />
      <Card className="mt-4" title="录入银行 / Payment 到账">
        <Form
          layout="vertical"
          onFinish={(v) => createReceipt.mutate(v)}
          initialValues={{
            source_type: "bank",
            currency: "USD",
            observed_at: new Date().toISOString(),
            raw_payload_json: "{}",
          }}
        >
          <Space wrap align="start">
            <Form.Item
              name="finance_account_id"
              label="Owner 财务账户"
              rules={[{ required: true }]}
            >
              <Select
                style={{ width: 280 }}
                options={(facts.data?.finance_accounts ?? []).map((a) => ({
                  value: a.id,
                  label: `${a.name} · ${a.account_type} · ${a.currency}`,
                }))}
              />
            </Form.Item>
            <Form.Item name="source_type" label="来源">
              <Select
                style={{ width: 140 }}
                options={[
                  { value: "bank", label: "银行" },
                  { value: "payment", label: "支付机构" },
                ]}
              />
            </Form.Item>
            <Form.Item
              name="external_receipt_id"
              label="外部到账 ID"
              rules={[{ required: true }]}
            >
              <Input />
            </Form.Item>
            <Form.Item
              name="idempotency_key"
              label="幂等键"
              rules={[{ required: true }]}
            >
              <Input />
            </Form.Item>
            <Form.Item
              name="amount_minor"
              label="到账金额（minor）"
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
          </Space>
          <Form.Item
            name="observed_at"
            label="观察时间（ISO 8601）"
            rules={[{ required: true }]}
          >
            <Input />
          </Form.Item>
          <Form.Item
            name="raw_payload_json"
            label="原始到账凭证 JSON"
            rules={[{ required: true }]}
          >
            <Input.TextArea rows={3} />
          </Form.Item>
          {createReceipt.error && (
            <Alert
              className="mb-4"
              type="error"
              showIcon
              message="到账未保存"
              description={createReceipt.error.message}
            />
          )}
          <Button
            type="primary"
            htmlType="submit"
            loading={createReceipt.isPending}
          >
            保存到账事实
          </Button>
        </Form>
      </Card>
      <Card className="mt-4" title="到账事实">
        <Table
          rowKey="id"
          loading={receipts.isLoading}
          dataSource={receipts.data ?? []}
          locale={{ emptyText: "暂无银行或支付机构到账事实" }}
          columns={[
            { title: "到账 ID", dataIndex: "id" },
            { title: "账户", dataIndex: "finance_account_id" },
            { title: "外部凭证", dataIndex: "external_receipt_id" },
            {
              title: "金额",
              render: (_, r) => money(r.amount_minor, r.currency),
            },
            {
              title: "事实等级",
              render: (_, r) => <Tag>{r.truth_status}</Tag>,
            },
            {
              title: "对账状态",
              render: (_, r) => (
                <Tag
                  color={
                    r.reconciliation_status === "reconciled"
                      ? "success"
                      : "warning"
                  }
                >
                  {r.reconciliation_status}
                </Tag>
              ),
            },
            {
              title: "原始凭证 SHA-256",
              render: (_, r) => (
                <Typography.Text copyable ellipsis style={{ maxWidth: 180 }}>
                  {r.raw_payload_sha256}
                </Typography.Text>
              ),
            },
          ]}
        />
      </Card>
      <Card className="mt-4" title="将到账分配到权威平台结算">
        <Alert
          className="mb-4"
          type="warning"
          showIcon
          message="高风险确定性对账"
          description="必须使用绑定本次到账/结算对象及当前 Owner 的已批准一次性审批；审批只能消费一次。"
        />
        <Form layout="inline" onFinish={(v) => reconcile.mutate(v)}>
          <Form.Item
            name="cash_receipt_id"
            label="到账事实"
            rules={[{ required: true }]}
          >
            <Select
              style={{ width: 300 }}
              options={(facts.data?.cash_receipts ?? []).map((c) => ({
                value: c.id,
                label: cashLabel(c),
              }))}
            />
          </Form.Item>
          <Form.Item
            name="platform_settlement_ingest_id"
            label="平台结算事实"
            rules={[{ required: true }]}
          >
            <Select
              style={{ width: 360 }}
              options={(facts.data?.settlements ?? []).map((s) => ({
                value: s.id,
                label: settlementLabel(s),
              }))}
            />
          </Form.Item>
          <Form.Item
            name="amount_minor"
            label="分配金额（minor）"
            rules={[{ required: true }]}
          >
            <InputNumber min={1} />
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
            type="primary"
            htmlType="submit"
            loading={reconcile.isPending}
          >
            执行确定性对账
          </Button>
        </Form>
        {reconcile.error && (
          <Alert
            className="mt-4"
            type="warning"
            showIcon
            message="对账被阻断"
            description={reconcile.error.message}
          />
        )}
        <Table
          className="mt-4"
          rowKey="id"
          dataSource={recs.data ?? []}
          locale={{ emptyText: "暂无对账记录" }}
          columns={[
            { title: "到账 ID", dataIndex: "cash_receipt_id" },
            {
              title: "结算事实 ID",
              dataIndex: "platform_settlement_ingest_id",
            },
            {
              title: "本次分配",
              render: (_, r) => money(r.amount_minor, r.currency),
            },
            {
              title: "应收",
              render: (_, r) => money(r.expected_receivable_minor, r.currency),
            },
            {
              title: "状态",
              render: (_, r) => (
                <Tag
                  color={
                    r.status === "reconciled"
                      ? "success"
                      : r.status === "conflict"
                        ? "error"
                        : "warning"
                  }
                >
                  {r.status}
                </Tag>
              ),
            },
            {
              title: "阻断/冲突",
              dataIndex: "conflict_reason",
              render: (v) => v || "—",
            },
          ]}
        />
      </Card>
    </main>
  );
}

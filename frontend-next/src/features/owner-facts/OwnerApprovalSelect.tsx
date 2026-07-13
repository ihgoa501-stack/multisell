"use client";
import { Select } from "antd";
import { useQuery } from "@tanstack/react-query";
import apiClient from "@/lib/api-client";
type Approval = {
  id: number;
  request_type: string;
  target_type?: string;
  target_id?: number;
};
export function OwnerApprovalSelect({
  value,
  onChange,
}: {
  value?: number;
  onChange?: (value: number) => void;
}) {
  const query = useQuery({
    queryKey: ["owner-approved-approval-options"],
    queryFn: async () =>
      (
        await apiClient.getPage<Approval>("/v1/approval", {
          page: "1",
          size: "100",
          status: "approved",
        })
      ).data ?? [],
  });
  return (
    <Select
      showSearch
      optionFilterProp="label"
      loading={query.isLoading}
      value={value}
      onChange={onChange}
      style={{ width: 340 }}
      placeholder="选择当前 Owner 已批准的一次性审批"
      options={(query.data ?? []).map((a) => ({
        value: a.id,
        label: `${a.request_type} · ${a.target_type || "目标"} #${a.target_id || "—"} · 审批 #${a.id}`,
      }))}
    />
  );
}

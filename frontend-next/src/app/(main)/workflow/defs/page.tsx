'use client';

import { useState } from 'react';
import {
  Button, Card, Input, message, Modal, Space, Table, Tag, Typography,
} from 'antd';
import { PlusOutlined, PlayCircleOutlined, DeleteOutlined } from '@ant-design/icons';
import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query';
import apiClient from '@/lib/api-client';
import PageContainer from '@/components/ui/PageContainer';
import dayjs from 'dayjs';

const { TextArea } = Input;
const { Text } = Typography;

interface WorkflowDef {
  id: number;
  name: string;
  description: string;
  steps: string;
  created_at: string;
  updated_at: string;
}

export default function WorkflowDefsPage() {
  const queryClient = useQueryClient();
  const [createOpen, setCreateOpen] = useState(false);
  const [editOpen, setEditOpen] = useState(false);
  const [editDef, setEditDef] = useState<WorkflowDef | null>(null);

  const { data: defs, isLoading } = useQuery({
    queryKey: ['workflow-defs'],
    queryFn: async () => {
      const res = await apiClient.get<WorkflowDef[]>('/v1/workflow/defs');
      return res.data || [];
    },
  });

  const createMutation = useMutation({
    mutationFn: async (values: { name: string; steps: string; description?: string }) => {
      return apiClient.post('/v1/workflow/defs', values);
    },
    onSuccess: () => {
      message.success('创建成功');
      setCreateOpen(false);
      queryClient.invalidateQueries({ queryKey: ['workflow-defs'] });
    },
    onError: () => message.error('创建失败'),
  });

  const updateMutation = useMutation({
    mutationFn: async (values: WorkflowDef) => {
      return apiClient.put(`/v1/workflow/defs/${values.id}`, values);
    },
    onSuccess: () => {
      message.success('更新成功');
      setEditOpen(false);
      queryClient.invalidateQueries({ queryKey: ['workflow-defs'] });
    },
    onError: () => message.error('更新失败'),
  });

  const deleteMutation = useMutation({
    mutationFn: async (id: number) => {
      return apiClient.delete(`/v1/workflow/defs/${id}`);
    },
    onSuccess: () => {
      message.success('删除成功');
      queryClient.invalidateQueries({ queryKey: ['workflow-defs'] });
    },
    onError: () => message.error('删除失败'),
  });

  const startRunMutation = useMutation({
    mutationFn: async (defId: number) => {
      return apiClient.post(`/v1/workflow/defs/${defId}/start`, { context: {} });
    },
    onSuccess: () => message.success('工作流已启动'),
    onError: () => message.error('启动失败'),
  });

  const columns = [
    { title: 'ID', dataIndex: 'id', width: 60 },
    { title: '名称', dataIndex: 'name', ellipsis: true },
    {
      title: '描述', dataIndex: 'description', ellipsis: true, width: 200,
    },
    {
      title: '步骤数',
      dataIndex: 'steps',
      width: 80,
      render: (steps: string) => {
        try {
          const parsed = JSON.parse(steps);
          return Array.isArray(parsed) ? parsed.length : 0;
        } catch { return 0; }
      },
    },
    {
      title: '创建时间', dataIndex: 'created_at', width: 160,
      render: (v: string) => v ? dayjs(v).format('YYYY-MM-DD HH:mm') : '-',
    },
    {
      title: '操作', width: 180,
      render: (_: unknown, record: WorkflowDef) => (
        <Space size="small">
          <Button size="small" type="primary" icon={<PlayCircleOutlined />}
            onClick={() => startRunMutation.mutate(record.id)}>
            运行
          </Button>
          <Button size="small" onClick={() => { setEditDef(record); setEditOpen(true); }}>
            编辑
          </Button>
          <Button size="small" danger icon={<DeleteOutlined />}
            onClick={() => deleteMutation.mutate(record.id)} />
        </Space>
      ),
    },
  ];

  return (
    <PageContainer
      title="工作流定义"
      extra={
        <Button type="primary" icon={<PlusOutlined />} onClick={() => setCreateOpen(true)}>
          新建定义
        </Button>
      }
    >
      <Card size="small" styles={{ body: { padding: 0 } }}>
        <Table<WorkflowDef>
          rowKey="id"
          columns={columns}
          dataSource={defs}
          loading={isLoading}
          scroll={{ x: 700 }}
        />
      </Card>

      {/* Create modal */}
      <Modal
        title="新建工作流定义"
        open={createOpen}
        onCancel={() => setCreateOpen(false)}
        footer={null}
      >
        <CreateDefForm
          onFinish={(v) => createMutation.mutate(v)}
          loading={createMutation.isPending}
        />
      </Modal>

      {/* Edit modal */}
      {editDef && (
        <Modal
          title={`编辑 #${editDef.id}`}
          open={editOpen}
          onCancel={() => { setEditOpen(false); setEditDef(null); }}
          footer={null}
        >
          <CreateDefForm
            initial={editDef}
            onFinish={(v) => updateMutation.mutate({ ...editDef, ...v })}
            loading={updateMutation.isPending}
          />
        </Modal>
      )}
    </PageContainer>
  );
}

function CreateDefForm({
  initial, onFinish, loading,
}: {
  initial?: Partial<WorkflowDef>;
  onFinish: (v: { name: string; description: string; steps: string }) => void;
  loading: boolean;
}) {
  const [name, setName] = useState(initial?.name || '');
  const [description, setDescription] = useState(initial?.description || '');
  const [steps, setSteps] = useState(initial?.steps || '');

  const validJson = (() => {
    if (!steps) return false;
    try { JSON.parse(steps); return true; } catch { return false; }
  })();

  return (
    <Space direction="vertical" style={{ width: '100%' }}>
      <div>
        <Text type="secondary">名称</Text>
        <Input value={name} onChange={(e) => setName(e.target.value)} placeholder="工作流名称" />
      </div>
      <div>
        <Text type="secondary">描述</Text>
        <Input value={description} onChange={(e) => setDescription(e.target.value)} placeholder="可选描述" />
      </div>
      <div>
        <Text type="secondary">步骤定义 (JSON)</Text>
        <TextArea
          rows={8}
          value={steps}
          onChange={(e) => setSteps(e.target.value)}
          placeholder='[{"name":"step1","type":"agent","agent_id":"A1","timeout_seconds":60}]'
          style={{ fontFamily: 'monospace', fontSize: '0.85rem' }}
        />
        {steps && (
          <Text type={validJson ? 'success' : 'danger'} style={{ fontSize: '0.78rem' }}>
            {validJson ? 'JSON 格式正确' : 'JSON 格式错误'}
          </Text>
        )}
      </div>
      <Button type="primary" disabled={!name || !validJson} loading={loading}
        onClick={() => onFinish({ name, description, steps })}>
        {initial ? '保存' : '创建'}
      </Button>
    </Space>
  );
}

'use client';

import { useEffect, useMemo, useState } from 'react';
import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query';
import { Alert, App, Button, Card, Descriptions, Divider, Drawer, Select, Space, Table, Tag, Typography } from 'antd';
import apiClient from '@/lib/api-client';

const { Text } = Typography;

type Watch = { id: number; enabled: boolean; updated_at: string; disabled_at?: string };
type Run = { id: number; request_id: string; status: 'pending_browser' | 'evaluated' | 'failed'; previous_snapshot_id?: number; current_snapshot_id?: number; alert_count: number; created_at: string; completed_at?: string };
type Snapshot = { id: number; collected_at: string; driver?: string; observed_title?: string };
type History = { snapshots?: Snapshot[] };
type AlertRow = { id: number; change_type: string; previous_snapshot_id: number; current_snapshot_id: number; before_value: unknown; after_value: unknown; content_hash: string; created_at: string };

export type SourceWatchWorkspaceProps = { sourceID: number; sourceURL: string; open: boolean; onClose: () => void };

export function watchSeverity(type: string, after: unknown) {
  if (type === 'offer_state' && String(after).toLowerCase().includes('delist')) return { color: 'red', label: '高' };
  if (type === 'supplier') return { color: 'red', label: '高' };
  if (type === 'sku_set' || type === 'quoted_inventory') return { color: 'orange', label: '中' };
  return { color: 'gold', label: '关注' };
}

function displayValue(value: unknown) {
  if (value == null) return '未取得';
  return typeof value === 'string' ? value : JSON.stringify(value);
}

export default function SourceWatchWorkspace({ sourceID, sourceURL, open, onClose }: SourceWatchWorkspaceProps) {
  const { message } = App.useApp();
  const qc = useQueryClient();
  const base = `/v1/sourcing-1688/${sourceID}/watch`;
  const [run, setRun] = useState<Run | null>(null);
  const [beforeID, setBeforeID] = useState<number>();
  const [afterID, setAfterID] = useState<number>();

	  const watch = useQuery({ queryKey: ['sourcing-watch', sourceID], queryFn: () => apiClient.get<Watch>(base), enabled: open, retry: false });
	  const runs = useQuery({ queryKey: ['sourcing-watch-runs', sourceID], queryFn: () => apiClient.get<Run[]>(`${base}/refresh-runs`), enabled: open });
  const history = useQuery({ queryKey: ['sourcing-watch-history', sourceID], queryFn: () => apiClient.get<History>(`/v1/sourcing-1688/${sourceID}/identity-history`), enabled: open });
  const alerts = useQuery({ queryKey: ['sourcing-watch-alerts', sourceID], queryFn: () => apiClient.get<AlertRow[]>(`${base}/alerts`), enabled: open });
	  const snapshots = useMemo(() => [...(history.data?.data?.snapshots ?? [])].sort((a, b) => Date.parse(b.collected_at) - Date.parse(a.collected_at)), [history.data?.data?.snapshots]);
	  const enabled = watch.data?.data?.enabled === true;
	  useEffect(() => {
	    if (run || !runs.data?.data) return;
	    setRun(runs.data.data.find((item) => item.status === 'pending_browser') ?? null);
	  }, [run, runs.data?.data]);
	  const snapshotsAfterRun = run ? snapshots.filter((snapshot) => Date.parse(snapshot.collected_at) > Date.parse(run.created_at)) : [];

  const setEnabled = useMutation({
    mutationFn: (value: boolean) => apiClient.put<Watch>(base, { enabled: value }),
    onSuccess: async (_, value) => { await qc.invalidateQueries({ queryKey: ['sourcing-watch', sourceID] }); message.success(value ? '已关注该货源' : '已暂停货源关注'); },
    onError: (error: Error) => message.error(error.message),
  });
  const createRun = useMutation({
    mutationFn: () => apiClient.post<Run>(`${base}/refresh-runs`, { request_id: `watch_${sourceID}_${crypto.randomUUID()}` }),
	    onSuccess: async (res) => { setRun(res.data ?? null); await qc.invalidateQueries({ queryKey: ['sourcing-watch-runs', sourceID] }); message.info('刷新请求已创建，请打开1688详情页用凌镜插件补采'); },
    onError: (error: Error) => message.error(error.message),
  });
  const evaluate = useMutation({
    mutationFn: () => {
      if (!run || !beforeID || !afterID) throw new Error('请选择前后两个快照');
      return apiClient.post<Run>(`${base}/refresh-runs/${run.id}/evaluate`, { previous_snapshot_id: beforeID, current_snapshot_id: afterID });
    },
    onSuccess: async (res) => { setRun(res.data ?? null); await qc.invalidateQueries({ queryKey: ['sourcing-watch-alerts', sourceID] }); message.success('观察差异已生成提醒，草稿未被修改'); },
    onError: (error: Error) => message.error(error.message),
  });

  const safeURL = /^https:\/\/detail\.1688\.com\/offer\/\d+\.html(?:[?#].*)?$/i.test(sourceURL) ? sourceURL : undefined;
  return <Drawer title={`1688货源关注 · 收藏 #${sourceID}`} open={open} onClose={onClose} size={900} destroyOnHidden>
    <Alert type="warning" showIcon title="监控不会自动登录或抓取1688" description="刷新任务会停在“等待浏览器补采”。请由你打开真实1688详情页并点击凌镜采集；系统只比较不可变页面观察，不会覆盖已审核草稿、改价或改库存。" />
    <Card size="small" style={{ marginTop: 16 }} title="1. 关注状态" extra={<Button loading={setEnabled.isPending} onClick={() => setEnabled.mutate(!enabled)}>{enabled ? '暂停关注' : '开始关注'}</Button>}>
      {watch.isLoading ? <Text type="secondary">读取中…</Text> : enabled ? <Tag color="green">关注中</Tag> : <Tag>未关注或已暂停</Tag>}
    </Card>
    <Card size="small" style={{ marginTop: 16 }} title="2. 请求新的页面观察">
      <Space wrap>
        <Button type="primary" disabled={!enabled} loading={createRun.isPending} onClick={() => createRun.mutate()}>请求浏览器补采</Button>
	        {safeURL ? <Button href={safeURL} target="_blank" rel="noopener noreferrer">打开1688详情页</Button> : <Button disabled>详情链接不可用</Button>}
      </Space>
      {run && <Descriptions size="small" bordered column={2} style={{ marginTop: 12 }}>
        <Descriptions.Item label="刷新任务">#{run.id}</Descriptions.Item>
        <Descriptions.Item label="状态"><Tag color={run.status === 'pending_browser' ? 'gold' : run.status === 'evaluated' ? 'green' : 'red'}>{run.status === 'pending_browser' ? '等待Owner浏览器补采' : run.status === 'evaluated' ? '已评估' : '失败'}</Tag></Descriptions.Item>
        <Descriptions.Item label="提醒数">{run.alert_count}</Descriptions.Item><Descriptions.Item label="请求时间">{new Date(run.created_at).toLocaleString('zh-CN')}</Descriptions.Item>
      </Descriptions>}
    </Card>
    <Card size="small" style={{ marginTop: 16 }} title="3. 补采完成后选择两个快照评估">
      <Space wrap>
        <Select aria-label="较早快照" placeholder="较早快照" value={beforeID} onChange={setBeforeID} style={{ width: 280 }} options={snapshots.map((s) => ({ value: s.id, label: `#${s.id} · ${new Date(s.collected_at).toLocaleString('zh-CN')}` }))} />
	        <Select aria-label="较新快照" placeholder={run ? '仅显示本次请求后的快照' : '请先创建刷新任务'} value={afterID} onChange={setAfterID} style={{ width: 280 }} options={snapshotsAfterRun.map((s) => ({ value: s.id, label: `#${s.id} · ${new Date(s.collected_at).toLocaleString('zh-CN')}` }))} />
	        <Button disabled={!run || run.status !== 'pending_browser' || !beforeID || !afterID || beforeID === afterID} loading={evaluate.isPending} onClick={() => evaluate.mutate()}>生成变化提醒</Button>
	      </Space>
	      {run && snapshotsAfterRun.length === 0 && <Alert type="info" showIcon title="尚未发现本次请求后的新快照" description="完成详情页补采后刷新；旧快照不能冒充本次观察。" style={{ marginTop: 12 }} />}
    </Card>
    <Divider>变化提醒</Divider>
    <Alert type="info" showIcon title="当前后端未提供已读/未读状态" description="这里按变化类型显示建议严重性；它只是页面声明变化，不代表供应商事实已经独立核验。" style={{ marginBottom: 12 }} />
    <Table rowKey="id" pagination={false} loading={alerts.isLoading} dataSource={alerts.data?.data ?? []} columns={[
      { title: '严重性', render: (_, row) => { const meta = watchSeverity(row.change_type, row.after_value); return <Tag color={meta.color}>{meta.label}</Tag>; } },
      { title: '变化', dataIndex: 'change_type' },
      { title: '前 → 后', render: (_, row) => <Text>{displayValue(row.before_value)} → {displayValue(row.after_value)}</Text> },
      { title: '快照', render: (_, row) => `#${row.previous_snapshot_id} → #${row.current_snapshot_id}` },
      { title: '时间', render: (_, row) => new Date(row.created_at).toLocaleString('zh-CN') },
    ]} />
  </Drawer>;
}

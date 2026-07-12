'use client';

import { useEffect, useMemo, useRef, useState } from 'react';
import { Alert, Button, Card, Descriptions, List, Space, Tag, Typography, message } from 'antd';
import apiClient from '@/lib/api-client';

type Pairing = {
	pairing_id: number;
  nonce: string;
  environment: 'development' | 'acceptance' | 'production';
  expires_at: string;
  status?: string;
  device_id?: string;
  extension_id?: string;
  browser_label?: string;
};

type Device = {
  device_id: string;
  browser_label: string;
  environment: string;
  scope: string;
  revoked_at?: string;
  last_used_at?: string;
};

function currentEnvironment(): Pairing['environment'] {
  if (typeof window === 'undefined') return 'development';
  if (window.location.hostname === 'localhost' || window.location.hostname === '127.0.0.1') return 'development';
  return window.location.hostname.includes('accept') || window.location.hostname.includes('staging') ? 'acceptance' : 'production';
}

export default function ExtensionPairingPage() {
  const [pairing, setPairing] = useState<Pairing | null>(null);
  const [devices, setDevices] = useState<Device[]>([]);
  const [busy, setBusy] = useState(false);
	const pairingRef = useRef<Pairing | null>(null);
  const environment = useMemo(currentEnvironment, []);

  const loadDevices = async () => {
    const result = await apiClient.get<Device[]>('/auth/extension-devices');
    setDevices(result.data ?? []);
  };

  useEffect(() => { void loadDevices(); }, []);
  useEffect(() => {
    const listener = async (event: MessageEvent) => {
	  const activePairing = pairingRef.current;
      if (event.source !== window || event.origin !== window.location.origin || !activePairing) return;
      if (event.data?.source !== 'lingmirror-extension') return;
      if (event.data.type === 'LINGMIRROR_EXTENSION_PAIRING_RESULT') {
        if (!event.data.ok) { message.error(event.data.error || '插件没有接受配对'); return; }
		const result = await apiClient.get<Omit<Pairing, 'nonce' | 'environment'>>(`/auth/extension-pairings/${activePairing.pairing_id}`);
		const next = { ...activePairing, ...(result.data ?? {}) };
		pairingRef.current = next; setPairing(next);
      }
      if (event.data.type === 'LINGMIRROR_EXTENSION_PAIRING_FINISHED') {
        setBusy(false);
        if (!event.data.ok) { message.error(event.data.error || '插件凭证签发失败'); return; }
		message.success('此浏览器已经连接凌镜'); pairingRef.current = null; setPairing(null); await loadDevices();
      }
    };
    window.addEventListener('message', listener);
    return () => window.removeEventListener('message', listener);
  }, []);

  const beginPairing = async () => {
    setBusy(true);
    try {
      const result = await apiClient.post<Pairing>('/auth/extension-pairings', { environment });
      if (!result.data?.nonce) throw new Error('服务端没有返回一次性配对码');
	  pairingRef.current = result.data;
      setPairing(result.data);
	  window.postMessage({ source: 'lingmirror-web', type: 'LINGMIRROR_EXTENSION_PAIRING', nonce: result.data.nonce, environment: result.data.environment }, window.location.origin);
    } catch (error) {
      setBusy(false); message.error(error instanceof Error ? error.message : '无法开始配对');
    }
  };

  const confirmPairing = async () => {
    if (!pairing) return;
    setBusy(true);
    try {
      await apiClient.post(`/auth/extension-pairings/${pairing.pairing_id}/confirm`, {});
	  window.postMessage({ source: 'lingmirror-web', type: 'LINGMIRROR_EXTENSION_PAIRING_CONFIRMED', nonce: pairing.nonce, environment: pairing.environment }, window.location.origin);
    } catch (error) { setBusy(false); message.error(error instanceof Error ? error.message : '确认失败'); }
  };

  const revoke = async (deviceID: string) => {
    await apiClient.delete(`/auth/extension-devices/${encodeURIComponent(deviceID)}`);
    message.success('已断开此浏览器'); await loadDevices();
  };

  return <Space direction="vertical" size={16} style={{ width: '100%' }}>
    <div><Typography.Title level={2}>1688采集插件连接</Typography.Title>
      <Typography.Text type="secondary">网页密码和登录Token不会交给插件。每个浏览器使用独立、可撤销的采集凭证。</Typography.Text></div>
    <Alert type={environment === 'production' ? 'warning' : 'info'} showIcon
      message={`当前环境：${environment === 'production' ? '生产' : environment === 'acceptance' ? '验收' : '本地开发'}`}
      description="插件凭证只对当前环境有效，不能跨环境写入。" />
    <Card title="连接当前浏览器">
      {!pairing && <Button type="primary" loading={busy} onClick={beginPairing}>连接此浏览器</Button>}
      {pairing && !pairing.device_id && <Alert type="info" message="正在等待已安装的凌镜插件响应……" />}
      {pairing?.device_id && <Space direction="vertical" style={{ width: '100%' }}>
        <Alert type="warning" showIcon message="请确认下面确实是你正在使用的浏览器" />
        <Descriptions bordered size="small" column={1}>
          <Descriptions.Item label="浏览器">{pairing.browser_label}</Descriptions.Item>
          <Descriptions.Item label="设备编号">{pairing.device_id}</Descriptions.Item>
          <Descriptions.Item label="插件编号">{pairing.extension_id}</Descriptions.Item>
          <Descriptions.Item label="权限">仅采集1688私人收藏、读取本次保存结果</Descriptions.Item>
        </Descriptions>
		<Space><Button type="primary" loading={busy} onClick={confirmPairing}>确认连接</Button><Button onClick={() => { pairingRef.current = null; setPairing(null); setBusy(false); }}>取消</Button></Space>
      </Space>}
    </Card>
    <Card title="已经连接的浏览器">
      <List dataSource={devices} locale={{ emptyText: '还没有已连接浏览器' }} renderItem={(item) => <List.Item
        actions={!item.revoked_at ? [<Button danger key="revoke" onClick={() => void revoke(item.device_id)}>断开</Button>] : []}>
        <List.Item.Meta title={<Space>{item.browser_label}<Tag>{item.environment}</Tag>{item.revoked_at && <Tag color="red">已撤销</Tag>}</Space>}
          description={`设备 ${item.device_id} · 权限 ${item.scope}`} />
      </List.Item>} />
    </Card>
  </Space>;
}

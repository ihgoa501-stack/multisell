'use client';

import { useCallback, useEffect, useMemo, useRef, useState } from 'react';
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

type ConnectionIssue = { title: string; description: string };

export const PLUGIN_RESPONSE_TIMEOUT_MS = 8_000;

const environmentLabel: Record<string, string> = {
  development: '本地开发', acceptance: '验收', production: '生产',
};

export function friendlyPairingError(error: unknown): ConnectionIssue {
  const status = typeof error === 'object' && error !== null && 'status' in error
    ? Number((error as { status?: unknown }).status)
    : undefined;
  const raw = error instanceof Error ? error.message : String(error ?? '');
  const inferredStatus = status || Number(raw.match(/HTTP\s*(401|403)/i)?.[1]);
  if (inferredStatus === 401) return {
    title: '登录已失效', description: '请重新登录凌镜，回到本页后再次连接。',
  };
  if (inferredStatus === 403) return {
    title: '当前浏览器暂时无法连接', description: '请刷新本页后重试；如果仍然失败，说明凌镜的插件连接服务尚未就绪，本次没有连接成功。',
  };
  if (status === 0 || /network|failed to fetch|load failed|无法连接/i.test(raw)) return {
    title: '无法联系凌镜服务', description: '请检查网络并刷新本页，然后重试；本次没有连接成功。',
  };
  if (raw && !/^HTTP\s*\d+$/i.test(raw)) return {
    title: '连接没有完成', description: `${raw}。请按提示处理后重试。`,
  };
  return { title: '连接没有完成', description: '请刷新本页后重试；本次没有连接成功。' };
}

function currentEnvironment(): Pairing['environment'] {
  if (typeof window === 'undefined') return 'development';
  if (window.location.hostname === 'localhost' || window.location.hostname === '127.0.0.1') return 'development';
  return window.location.hostname.includes('accept') || window.location.hostname.includes('staging') ? 'acceptance' : 'production';
}

export default function ExtensionPairingPage() {
  const [pairing, setPairing] = useState<Pairing | null>(null);
  const [devices, setDevices] = useState<Device[]>([]);
  const [busy, setBusy] = useState(false);
  const [issue, setIssue] = useState<ConnectionIssue | null>(null);
  const pairingRef = useRef<Pairing | null>(null);
  const timeoutRef = useRef<number | null>(null);
  const environment = useMemo(() => currentEnvironment(), []);

  const clearPluginTimeout = useCallback(() => {
    if (timeoutRef.current !== null) window.clearTimeout(timeoutRef.current);
    timeoutRef.current = null;
  }, []);

  const cancelPairing = useCallback(() => {
    clearPluginTimeout();
    pairingRef.current = null;
    setPairing(null);
    setBusy(false);
  }, [clearPluginTimeout]);

  const armPluginTimeout = useCallback((pairingID: number) => {
    clearPluginTimeout();
    timeoutRef.current = window.setTimeout(() => {
      if (pairingRef.current?.pairing_id !== pairingID) return;
      pairingRef.current = null;
      setPairing(null);
      setBusy(false);
      setIssue({
        title: '没有检测到1688采集助手',
        description: '请确认Chrome扩展已安装并启用，然后点击“重新连接”；本次没有连接成功。',
      });
    }, PLUGIN_RESPONSE_TIMEOUT_MS);
  }, [clearPluginTimeout]);

  const loadDevices = useCallback(async () => {
    try {
      const result = await apiClient.get<Device[]>('/v1/auth/extension-devices');
      setDevices(result.data ?? []);
    } catch (error) {
      setIssue(friendlyPairingError(error));
    }
  }, []);

  useEffect(() => { void loadDevices(); }, [loadDevices]);
  useEffect(() => () => clearPluginTimeout(), [clearPluginTimeout]);
  useEffect(() => {
    const listener = async (event: MessageEvent) => {
      const activePairing = pairingRef.current;
      if (event.source !== window || event.origin !== window.location.origin || !activePairing) return;
      if (event.data?.source !== 'lingmirror-extension') return;
      if (event.data.type === 'LINGMIRROR_EXTENSION_PAIRING_RESULT') {
        clearPluginTimeout();
        if (!event.data.ok) {
          cancelPairing();
          setIssue(friendlyPairingError(new Error(event.data.error || '插件没有接受连接')));
          return;
        }
        try {
          const result = await apiClient.get<Omit<Pairing, 'nonce' | 'environment'>>(`/v1/auth/extension-pairings/${activePairing.pairing_id}`);
          const next = { ...activePairing, ...(result.data ?? {}) };
          pairingRef.current = next;
          setPairing(next);
          setBusy(false);
        } catch (error) {
          cancelPairing();
          setIssue(friendlyPairingError(error));
        }
      }
      if (event.data.type === 'LINGMIRROR_EXTENSION_PAIRING_FINISHED') {
        clearPluginTimeout();
        setBusy(false);
        if (!event.data.ok) {
          cancelPairing();
          setIssue(friendlyPairingError(new Error(event.data.error || '插件凭证签发失败')));
          return;
        }
        message.success('1688采集助手已连接');
        pairingRef.current = null;
        setPairing(null);
        setIssue(null);
        await loadDevices();
      }
    };
    window.addEventListener('message', listener);
    return () => window.removeEventListener('message', listener);
  }, [cancelPairing, clearPluginTimeout, loadDevices]);

  const beginPairing = async () => {
    setBusy(true);
    setIssue(null);
    try {
      const result = await apiClient.post<Pairing>('/v1/auth/extension-pairings', { environment });
      if (!result.data?.nonce) throw new Error('凌镜没有返回连接凭据');
      pairingRef.current = result.data;
      setPairing(result.data);
      window.postMessage({ source: 'lingmirror-web', type: 'LINGMIRROR_EXTENSION_PAIRING', nonce: result.data.nonce, environment: result.data.environment }, window.location.origin);
      armPluginTimeout(result.data.pairing_id);
    } catch (error) {
      cancelPairing();
      setIssue(friendlyPairingError(error));
    }
  };

  const confirmPairing = async () => {
    if (!pairing) return;
    setBusy(true);
    setIssue(null);
    try {
      await apiClient.post(`/v1/auth/extension-pairings/${pairing.pairing_id}/confirm`, {});
      window.postMessage({ source: 'lingmirror-web', type: 'LINGMIRROR_EXTENSION_PAIRING_CONFIRMED', nonce: pairing.nonce, environment: pairing.environment }, window.location.origin);
      armPluginTimeout(pairing.pairing_id);
    } catch (error) {
      setBusy(false);
      setIssue(friendlyPairingError(error));
    }
  };

  const revoke = async (deviceID: string) => {
    try {
      await apiClient.delete(`/v1/auth/extension-devices/${encodeURIComponent(deviceID)}`);
      message.success('已断开此浏览器');
      await loadDevices();
    } catch (error) {
      setIssue(friendlyPairingError(error));
    }
  };

  return <Space direction="vertical" size={16} style={{ width: '100%' }}>
    <div><Typography.Title level={2}>1688采集助手</Typography.Title>
      <Typography.Text type="secondary">连接后，你可以在1688页面把商品保存到凌镜私人采集箱。凌镜不会把网页密码或登录凭据交给插件。</Typography.Text></div>
    <Alert type={environment === 'production' ? 'warning' : 'info'} showIcon
      message={`当前连接：${environmentLabel[environment]}`}
      description="连接只在当前凌镜环境有效，可以随时在下方断开。" />
    {issue && <Alert type="error" showIcon message={issue.title} description={issue.description}
      action={<Space><Button size="small" type="primary" onClick={() => void beginPairing()}>重新连接</Button><Button size="small" onClick={() => { cancelPairing(); setIssue(null); }}>取消</Button></Space>} />}
    <Card title="连接这台浏览器">
      {!pairing && <Space direction="vertical">
        <Typography.Text>请确认Chrome中已经安装并启用“凌镜1688采集助手”。</Typography.Text>
        <Button type="primary" loading={busy} onClick={() => void beginPairing()}>连接1688采集助手</Button>
      </Space>}
      {pairing && !pairing.device_id && <Space direction="vertical" style={{ width: '100%' }}>
        <Alert type="info" showIcon message="正在检测这台浏览器中的采集助手……" description="通常几秒内完成；没有响应时页面会告诉你如何重试。" />
        <Button onClick={cancelPairing}>取消连接</Button>
      </Space>}
      {pairing?.device_id && <Space direction="vertical" style={{ width: '100%' }}>
        <Alert type="warning" showIcon message="请确认下面确实是你正在使用的浏览器" />
        <Descriptions bordered size="small" column={1}>
          <Descriptions.Item label="浏览器">{pairing.browser_label}</Descriptions.Item>
          <Descriptions.Item label="设备编号">{pairing.device_id}</Descriptions.Item>
          <Descriptions.Item label="运行环境">{environmentLabel[pairing.environment]}</Descriptions.Item>
          <Descriptions.Item label="允许的操作">保存1688商品到私人采集箱、读取本次保存结果</Descriptions.Item>
        </Descriptions>
        <Space><Button type="primary" loading={busy} onClick={() => void confirmPairing()}>确认连接</Button><Button onClick={cancelPairing}>取消</Button></Space>
      </Space>}
    </Card>
    <Card title="已连接的浏览器">
      <List dataSource={devices} locale={{ emptyText: '还没有已连接浏览器' }} renderItem={(item) => <List.Item
        actions={!item.revoked_at ? [<Button danger key="revoke" onClick={() => void revoke(item.device_id)}>断开</Button>] : []}>
        <List.Item.Meta title={<Space>{item.browser_label}<Tag>{environmentLabel[item.environment] || item.environment}</Tag>{item.revoked_at && <Tag color="red">已撤销</Tag>}</Space>}
          description={`设备 ${item.device_id} · 权限：${item.scope === 'sourcing1688.collect' ? '保存1688私人收藏' : item.scope}`} />
      </List.Item>} />
    </Card>
  </Space>;
}

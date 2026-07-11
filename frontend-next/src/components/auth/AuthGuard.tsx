'use client';

import { useEffect, useState } from 'react';
import { useRouter } from 'next/navigation';
import { Button, Result, Spin } from 'antd';
import { usePermissionStore } from '@/stores/permission-store';

export default function AuthGuard({ children }: { children: React.ReactNode }) {
  const router = useRouter();
  const fetchPermissions = usePermissionStore((s) => s.fetchPermissions);
  const clearPermissions = usePermissionStore((s) => s.clearPermissions);
  const permissionsFetched = usePermissionStore((s) => s.fetched);
  const [takingTooLong, setTakingTooLong] = useState(false);
  const isBrowser = typeof window !== 'undefined';

  useEffect(() => {
    if (!isBrowser) return;
    const token = localStorage.getItem('token');
    if (!token) {
      router.replace('/login');
    } else {
      fetchPermissions();
    }
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, []);

  useEffect(() => {
    if (permissionsFetched) {
      setTakingTooLong(false);
      return;
    }

    const timer = window.setTimeout(() => setTakingTooLong(true), 6000);
    return () => window.clearTimeout(timer);
  }, [permissionsFetched]);

  // During SSR: render nothing to avoid hydration mismatch
  if (!isBrowser) return null;

  if (!permissionsFetched) {
    if (takingTooLong) {
      return (
        <Result
          status="warning"
          title="权限信息加载超时"
          subTitle="网络或权限服务暂时没有响应。你可以重试，或者返回登录页重新验证身份。"
          extra={[
            <Button
              type="primary"
              key="retry"
              onClick={() => {
                setTakingTooLong(false);
                clearPermissions();
                void fetchPermissions();
              }}
            >
              重新加载权限
            </Button>,
            <Button key="login" onClick={() => router.replace('/login')}>
              返回登录
            </Button>,
          ]}
        />
      );
    }

    return (
      <div
        style={{
          display: 'flex',
          justifyContent: 'center',
          alignItems: 'center',
          height: '100vh',
        }}
      >
        <Spin size="large" />
      </div>
    );
  }

  return <>{children}</>;
}

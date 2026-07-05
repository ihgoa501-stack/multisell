'use client';

import { useEffect } from 'react';
import { useRouter } from 'next/navigation';
import { Spin } from 'antd';
import { usePermissionStore } from '@/stores/permission-store';

export default function AuthGuard({ children }: { children: React.ReactNode }) {
  const router = useRouter();
  const fetchPermissions = usePermissionStore((s) => s.fetchPermissions);
  const permissionsFetched = usePermissionStore((s) => s.fetched);
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

  // During SSR: render nothing to avoid hydration mismatch
  if (!isBrowser) return null;

  if (!permissionsFetched) {
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

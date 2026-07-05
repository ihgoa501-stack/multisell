'use client';

import { useEffect, useState } from 'react';
import { useRouter } from 'next/navigation';
import { Spin } from 'antd';
import { usePermissionStore } from '@/stores/permission-store';

export default function AuthGuard({ children }: { children: React.ReactNode }) {
  const router = useRouter();
  const [mounted, setMounted] = useState(false);
  const fetchPermissions = usePermissionStore((s) => s.fetchPermissions);
  const permissionsFetched = usePermissionStore((s) => s.fetched);

  useEffect(() => {
    setMounted(true);
    const token = localStorage.getItem('token');
    if (!token) {
      router.replace('/login');
    } else {
      fetchPermissions();
    }
  }, [router, fetchPermissions]);

  if (!mounted) return null;

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

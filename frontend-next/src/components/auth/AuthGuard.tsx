'use client';

import { useEffect, useState } from 'react';
import { useRouter } from 'next/navigation';
import { Spin } from 'antd';
import { usePermissionStore } from '@/stores/permission-store';

export default function AuthGuard({ children }: { children: React.ReactNode }) {
  const router = useRouter();
  // Initialize authChecked from localStorage — runs once before any effect
  // Guard against SSR: localStorage is only available in the browser
  const [authChecked] = useState(() => {
    if (typeof window === 'undefined') return false;
    return !!localStorage.getItem('token');
  });
  const fetchPermissions = usePermissionStore((s) => s.fetchPermissions);
  const permissionsFetched = usePermissionStore((s) => s.fetched);

  useEffect(() => {
    const token = localStorage.getItem('token');
    if (!token) {
      router.replace('/login');
    } else {
      fetchPermissions();
    }
  }, [router, fetchPermissions]);

  if (!authChecked) return null;

  // Show a brief loading indicator while permissions are being fetched
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

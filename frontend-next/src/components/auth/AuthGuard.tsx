'use client';

import { useEffect, useState } from 'react';
import { useRouter } from 'next/navigation';
import { Spin } from 'antd';
import { usePermissionStore } from '@/stores/permission-store';

export default function AuthGuard({ children }: { children: React.ReactNode }) {
  const router = useRouter();
  const [checked, setChecked] = useState(false);
  const fetchPermissions = usePermissionStore((s) => s.fetchPermissions);
  const permissionsFetched = usePermissionStore((s) => s.fetched);

  useEffect(() => {
    const token = localStorage.getItem('token');
    if (!token) {
      router.replace('/login');
    } else {
      // Fetch RBAC permissions after confirming auth
      fetchPermissions();
      setChecked(true);
    }
  }, [router, fetchPermissions]);

  if (!checked) return null;

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

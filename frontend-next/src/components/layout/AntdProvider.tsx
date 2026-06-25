'use client';

import { ConfigProvider } from 'antd';
import { QueryClientProvider } from '@tanstack/react-query';
import { getQueryClient } from '@/lib/query-client';

const theme = {
  token: {
    colorPrimary: '#1677ff',
    borderRadius: 6,
    fontFamily:
      '-apple-system, BlinkMacSystemFont, "Segoe UI", Roboto, sans-serif',
  },
};

export function AntdProvider({ children }: { children: React.ReactNode }) {
  const queryClient = getQueryClient();
  return (
    <QueryClientProvider client={queryClient}>
      <ConfigProvider theme={theme}>{children}</ConfigProvider>
    </QueryClientProvider>
  );
}

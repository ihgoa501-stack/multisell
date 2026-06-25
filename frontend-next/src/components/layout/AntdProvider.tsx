'use client';

import { useEffect, useState } from 'react';
import { ConfigProvider, theme as antdTheme } from 'antd';
import { QueryClientProvider } from '@tanstack/react-query';
import { getQueryClient } from '@/lib/query-client';

function getDesignTokens(isDark: boolean) {
  return {
    token: {
      colorPrimary: '#6366F1',
      colorPrimaryHover: '#818CF8',
      colorPrimaryActive: '#4F46E5',
      colorInfo: '#6366F1',
      colorSuccess: '#34D399',
      colorWarning: '#FBBF24',
      colorError: '#F87171',
      colorLink: '#818CF8',
      borderRadius: 6,
      fontFamily: "'DM Sans', sans-serif",
      fontSize: 14,
    },
    algorithm: isDark ? antdTheme.darkAlgorithm : antdTheme.defaultAlgorithm,
    components: {
      Layout: {
        bodyBg: 'var(--bg)',
        headerBg: 'var(--s1)',
        siderBg: 'var(--s1)',
        triggerBg: 'var(--s2)',
        triggerHeight: 40,
      },
      Menu: {
        itemBg: 'transparent',
        itemColor: 'var(--t2)',
        itemHoverBg: 'var(--s2)',
        itemHoverColor: 'var(--t1)',
        itemSelectedBg: 'var(--s2)',
        itemSelectedColor: 'var(--i4)',
        groupTitleColor: 'var(--t4)',
        groupTitleFontSize: 11,
        fontFamily: "'DM Sans', sans-serif",
      },
      Button: {
        primaryShadow: '0 2px 12px rgba(99,102,241,0.25)',
        fontFamily: "'DM Sans', sans-serif",
      },
      Typography: {
        fontFamilyCode: "'JetBrains Mono', monospace",
      },
      Table: {
        headerBg: 'var(--s1)',
        headerColor: 'var(--t3)',
        borderColor: 'var(--bd)',
        rowHoverBg: 'var(--s1)',
      },
    },
  };
}

export function AntdProvider({ children }: { children: React.ReactNode }) {
  const queryClient = getQueryClient();
  const [mounted, setMounted] = useState(false);

  useEffect(() => {
    setMounted(true);
  }, []);

  // Read initial theme from html element
  const isDark =
    mounted && document.documentElement.getAttribute('data-theme') !== 'light';

  // Prevent flash of wrong theme on first render
  if (!mounted) {
    return (
      <QueryClientProvider client={queryClient}>
        <ConfigProvider theme={getDesignTokens(true)}>
          {children}
        </ConfigProvider>
      </QueryClientProvider>
    );
  }

  return (
    <QueryClientProvider client={queryClient}>
      <ConfigProvider theme={getDesignTokens(isDark)}>
        {children}
      </ConfigProvider>
    </QueryClientProvider>
  );
}

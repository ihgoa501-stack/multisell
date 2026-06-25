import type { Metadata } from 'next';
import { AntdProvider } from '@/components/layout/AntdProvider';
import './globals.css';
import '@/styles/design-tokens.css';

export const metadata: Metadata = {
  title: '凌镜 LingMirror | AgentOS',
  description: 'Cross-border e-commerce AI AgentOS',
};

export default function RootLayout({
  children,
}: Readonly<{
  children: React.ReactNode;
}>) {
  return (
    <html lang="zh-CN" data-theme="dark" className="h-full antialiased">
      <body className="min-h-full flex flex-col" style={{ fontFamily: 'var(--body)' }}>
        <AntdProvider>{children}</AntdProvider>
      </body>
    </html>
  );
}

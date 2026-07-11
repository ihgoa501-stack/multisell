import { describe, it, expect, vi } from 'vitest';
import { render, screen, fireEvent } from '@testing-library/react';
import MainLayout from '@/app/(main)/layout';

vi.mock('@/components/layout/AppSidebar', () => ({
  default: () => <nav>测试导航</nav>,
}));
vi.mock('@/components/layout/AppHeader', () => ({ default: () => <div>页头</div> }));
vi.mock('@/components/layout/ToolPanel', () => ({ default: () => null }));
vi.mock('@/components/copilot/CopilotPanel', () => ({ default: () => <aside>AI 面板</aside> }));
vi.mock('@/components/layout/CommandPalette', () => ({ default: () => null }));
vi.mock('@/components/auth/AuthGuard', () => ({
  default: ({ children }: { children: React.ReactNode }) => children,
}));
vi.mock('@/stores/app-store', () => ({
  useAppStore: () => ({
    toolPanelOpen: false,
    panelMode: 'balanced',
    setPanelMode: vi.fn(),
    toggleToolPanel: vi.fn(),
  }),
}));

describe('MainLayout mobile navigation', () => {
  it('opens and closes the navigation drawer with an accessible control', () => {
    render(<MainLayout><div>页面内容</div></MainLayout>);

    const openButton = screen.getByRole('button', { name: '打开导航' });
    expect(openButton).toHaveAttribute('aria-expanded', 'false');

    fireEvent.click(openButton);
    expect(openButton).toHaveAttribute('aria-expanded', 'true');
    expect(screen.getByRole('button', { name: '关闭导航' })).toBeInTheDocument();

    fireEvent.click(screen.getByRole('button', { name: '关闭导航' }));
    expect(openButton).toHaveAttribute('aria-expanded', 'false');
  });
});

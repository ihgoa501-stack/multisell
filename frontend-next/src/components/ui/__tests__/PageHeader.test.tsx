import { describe, it, expect } from 'vitest';
import { render, screen } from '@testing-library/react';
import PageHeader from '@/components/ui/PageHeader';

// Breadcrumbs uses usePathname which needs mocking
vi.mock('next/navigation', () => ({
  usePathname: () => '/test',
  useRouter: () => ({ push: vi.fn() }),
}));

describe('PageHeader', () => {
  it('renders title', () => {
    render(<PageHeader title="测试页面" />);
    expect(screen.getByText('测试页面')).toBeInTheDocument();
  });

  it('renders subtitle when provided', () => {
    render(<PageHeader title="测试" subtitle="这是副标题" />);
    expect(screen.getByText('这是副标题')).toBeInTheDocument();
  });

  it('renders extra content', () => {
    render(<PageHeader title="测试" extra={<button>操作</button>} />);
    expect(screen.getByText('操作')).toBeInTheDocument();
  });

  it('hides breadcrumbs when showBreadcrumb=false', () => {
    render(<PageHeader title="测试" showBreadcrumb={false} />);
    // Breadcrumbs would render "Home" — it shouldn't be visible
    expect(screen.queryByText('Home')).not.toBeInTheDocument();
  });
});

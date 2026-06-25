import { describe, it, expect, vi } from 'vitest';
import { render, screen } from '@testing-library/react';
import EmptyState from '@/components/ui/EmptyState';

describe('EmptyState', () => {
  it('renders default description', () => {
    render(<EmptyState />);
    expect(screen.getByText('暂无数据')).toBeInTheDocument();
  });

  it('renders custom description', () => {
    render(<EmptyState description="自定义空状态" />);
    expect(screen.getByText('自定义空状态')).toBeInTheDocument();
  });

  it('renders subtitle when provided', () => {
    render(<EmptyState description="空" subtitle="请先创建一条记录" />);
    expect(screen.getByText('请先创建一条记录')).toBeInTheDocument();
  });

  it('renders action button with correct label', () => {
    const onClick = vi.fn();
    render(<EmptyState description="空" action={{ label: '新建', onClick }} />);
    expect(screen.getByText((content) => content.replace(/\s/g, '').includes('新建'))).toBeInTheDocument();
  });

  it('renders large variant', () => {
    render(<EmptyState description="空" size="large" />);
    expect(screen.getByText('空')).toBeInTheDocument();
  });

  it('renders children', () => {
    render(
      <EmptyState description="空">
        <div>extra content</div>
      </EmptyState>
    );
    expect(screen.getByText('extra content')).toBeInTheDocument();
  });
});

import { describe, it, expect, vi } from 'vitest';
import { render, screen } from '@testing-library/react';
import PageContainer from '@/components/ui/PageContainer';

describe('PageContainer', () => {
  // --- Normal state ---
  it('renders title', () => {
    render(<PageContainer title="Test Title">Content</PageContainer>);
    expect(screen.getByText('Test Title')).toBeInTheDocument();
  });

  it('renders children', () => {
    render(<PageContainer title="Title">Child Content</PageContainer>);
    expect(screen.getByText('Child Content')).toBeInTheDocument();
  });

  it('renders subtitle when provided', () => {
    render(<PageContainer title="Title" subtitle="Sub text">Content</PageContainer>);
    expect(screen.getByText('Sub text')).toBeInTheDocument();
  });

  it('renders extra action buttons', () => {
    render(<PageContainer title="Title" extra={<button>Action</button>}>Content</PageContainer>);
    expect(screen.getByText('Action')).toBeInTheDocument();
  });

  // --- Loading state ---
  it('shows skeleton when loading', () => {
    render(<PageContainer title="Title" loading>Content</PageContainer>);
    expect(screen.getByText('Title')).toBeInTheDocument();
    expect(screen.queryByText('Content')).not.toBeInTheDocument();
  });

  it('loading skeleton shows title and hides children', () => {
    render(<PageContainer title="Loading Title" loading loadingDesc="custom desc">Content</PageContainer>);
    expect(screen.getByText('Loading Title')).toBeInTheDocument();
    expect(screen.queryByText('Content')).not.toBeInTheDocument();
  });

  // --- Empty state ---
  it('shows empty state when empty=true', () => {
    render(<PageContainer title="Title" empty>Content</PageContainer>);
    expect(screen.getByText('暂无数据')).toBeInTheDocument();
    expect(screen.queryByText('Content')).not.toBeInTheDocument();
  });

  it('shows custom empty description', () => {
    render(<PageContainer title="Title" empty emptyDesc="还没有任何数据">Content</PageContainer>);
    expect(screen.getByText('还没有任何数据')).toBeInTheDocument();
  });

  it('shows empty action button when configured', () => {
    const onClick = vi.fn();
    render(
      <PageContainer title="Title" empty emptyAction={{ label: '新建', onClick }}>
        Content
      </PageContainer>
    );
    expect(screen.getByText((content) => content.replace(/\s/g, '').includes('新建'))).toBeInTheDocument();
  });

  // --- Error state ---
  it('shows error state when error=true', () => {
    render(<PageContainer title="Title" error>Content</PageContainer>);
    expect(screen.getByText('加载失败')).toBeInTheDocument();
    expect(screen.queryByText('Content')).not.toBeInTheDocument();
  });

  it('shows custom error message', () => {
    render(<PageContainer title="Title" error errorMsg="服务器错误">Content</PageContainer>);
    expect(screen.getByText('服务器错误')).toBeInTheDocument();
  });

  it('shows retry button when onRetry is provided', () => {
    const onRetry = vi.fn();
    render(<PageContainer title="Title" error onRetry={onRetry}>Content</PageContainer>);
    expect(screen.getByRole('button', { name: /重/ })).toBeInTheDocument();
  });

  // --- Priority order ---
  it('loading takes priority over empty', () => {
    render(
      <PageContainer title="Title" loading empty>
        Content
      </PageContainer>
    );
    expect(screen.getByText('Title')).toBeInTheDocument();
  });

  it('empty takes priority over error', () => {
    render(
      <PageContainer title="Title" empty error>
        Content
      </PageContainer>
    );
    expect(screen.getByText('暂无数据')).toBeInTheDocument();
    expect(screen.queryByText('加载失败')).not.toBeInTheDocument();
  });
});

import { describe, it, expect, vi } from 'vitest';
import { render, screen } from '@testing-library/react';
import DetailDrawer from '@/components/ui/DetailDrawer';

describe('DetailDrawer', () => {
  it('renders title', () => {
    render(<DetailDrawer open title="订单详情" onClose={vi.fn()} />);
    expect(screen.getByText('订单详情')).toBeInTheDocument();
  });

  it('renders fields as key-value pairs', () => {
    const fields = [
      { label: '订单号', value: 'ORD-001' },
      { label: '金额', value: '¥100.00' },
    ];
    render(<DetailDrawer open title="详情" fields={fields} onClose={vi.fn()} />);
    expect(screen.getByText('订单号')).toBeInTheDocument();
    expect(screen.getByText('ORD-001')).toBeInTheDocument();
    expect(screen.getByText('金额')).toBeInTheDocument();
  });

  it('shows loading spinner', () => {
    render(<DetailDrawer open title="详情" loading onClose={vi.fn()} />);
    expect(document.querySelector('.ant-spin')).toBeTruthy();
  });

  it('shows empty state', () => {
    render(<DetailDrawer open title="详情" empty onClose={vi.fn()} />);
    expect(screen.getByText('暂无详情')).toBeInTheDocument();
  });

  it('renders children instead of fields', () => {
    render(
      <DetailDrawer open title="详情" onClose={vi.fn()}>
        <div>自定义内容</div>
      </DetailDrawer>
    );
    expect(screen.getByText('自定义内容')).toBeInTheDocument();
  });

  it('renders footer', () => {
    render(
      <DetailDrawer
        open
        title="详情"
        onClose={vi.fn()}
        footer={<button>操作</button>}
      />
    );
    expect(screen.getByText('操作')).toBeInTheDocument();
  });
});

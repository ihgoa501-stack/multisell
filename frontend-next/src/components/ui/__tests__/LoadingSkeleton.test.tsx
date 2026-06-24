import { describe, it, expect } from 'vitest';
import { render } from '@testing-library/react';
import { SkeletonTable, SkeletonCard, SkeletonList } from '@/components/ui/LoadingSkeleton';

describe('SkeletonTable', () => {
  it('renders without crashing', () => {
    const { container } = render(<SkeletonTable />);
    expect(container.querySelector('.ant-table')).toBeTruthy();
  });

  it('renders custom number of rows', () => {
    const { container } = render(<SkeletonTable rows={3} />);
    const rows = container.querySelectorAll('.ant-table-tbody tr');
    expect(rows.length).toBe(3);
  });
});

describe('SkeletonCard', () => {
  it('renders without crashing', () => {
    const { container } = render(<SkeletonCard />);
    expect(container.querySelector('.ant-card')).toBeTruthy();
  });

  it('renders custom number of cards', () => {
    const { container } = render(<SkeletonCard count={2} />);
    const cards = container.querySelectorAll('.ant-card');
    expect(cards.length).toBe(2);
  });
});

describe('SkeletonList', () => {
  it('renders without crashing', () => {
    const { container } = render(<SkeletonList />);
    expect(container.querySelector('.ant-card')).toBeTruthy();
  });
});

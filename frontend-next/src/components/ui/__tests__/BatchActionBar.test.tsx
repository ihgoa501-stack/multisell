import { describe, it, expect, vi } from 'vitest';
import { render, screen, fireEvent } from '@testing-library/react';
import BatchActionBar from '@/components/ui/BatchActionBar';

describe('BatchActionBar', () => {
  it('renders nothing when selectedCount is 0', () => {
    const { container } = render(
      <BatchActionBar selectedCount={0} actions={[]} onClearSelection={vi.fn()} />
    );
    expect(container.firstChild).toBeNull();
  });

  it('shows selected count', () => {
    render(
      <BatchActionBar
        selectedCount={3}
        actions={[]}
        onClearSelection={vi.fn()}
      />
    );
    expect(screen.getByText('3')).toBeInTheDocument();
    expect(screen.getByText((content) => content.replace(/\s/g, '').includes('已选'))).toBeInTheDocument();
  });

  it('renders action buttons', () => {
    const onClick = vi.fn();
    render(
      <BatchActionBar
        selectedCount={2}
        actions={[{ key: 'delete', label: '批量删除', onClick }]}
        onClearSelection={vi.fn()}
      />
    );
    const btn = screen.getByText('批量删除');
    expect(btn).toBeInTheDocument();
    fireEvent.click(btn);
    expect(onClick).toHaveBeenCalled();
  });

  it('fires onClearSelection', () => {
    const onClear = vi.fn();
    render(
      <BatchActionBar
        selectedCount={2}
        actions={[]}
        onClearSelection={onClear}
      />
    );
    fireEvent.click(screen.getByText('清除选择'));
    expect(onClear).toHaveBeenCalled();
  });
});

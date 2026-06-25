import { describe, it, expect, vi } from 'vitest';
import { render, screen, fireEvent } from '@testing-library/react';
import FilterBar from '@/components/ui/FilterBar';

describe('FilterBar', () => {
  it('renders search input', () => {
    render(<FilterBar />);
    const input = screen.getByPlaceholderText('搜索...');
    expect(input).toBeInTheDocument();
  });

  it('fires onSearch when value is submitted', () => {
    const onSearch = vi.fn();
    render(<FilterBar search="" onSearch={onSearch} />);
    const input = screen.getByPlaceholderText('搜索...');
    fireEvent.change(input, { target: { value: 'test' } });
    fireEvent.keyDown(input, { key: 'Enter' });
    // onSearch may be called on change + enter
    expect(onSearch).toHaveBeenCalled();
  });

  it('renders filter selects', () => {
    const filters = [
      {
        key: 'status',
        label: '状态',
        options: [
          { label: '启用', value: 'active' },
          { label: '停用', value: 'inactive' },
        ],
      },
    ];
    render(<FilterBar filters={filters} />);
    expect(screen.getByText((content) => content.replace(/\s/g, '').includes('状态'))).toBeInTheDocument();
  });

  it('renders reset button when filters are active and onReset provided', () => {
    const onReset = vi.fn();
    render(<FilterBar search="test" onReset={onReset} />);
    expect(screen.getByText('重置')).toBeInTheDocument();
  });
});

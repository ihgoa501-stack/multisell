import { describe, it, expect } from 'vitest';
import { render, screen } from '@testing-library/react';
import StatusTag from '@/components/ui/StatusTag';

describe('StatusTag', () => {
  it('renders status text', () => {
    render(<StatusTag status="active" />);
    expect(screen.getByText('active')).toBeInTheDocument();
  });

  it('renders custom label over status', () => {
    render(<StatusTag status="active" label="启用" />);
    expect(screen.getByText('启用')).toBeInTheDocument();
    expect(screen.queryByText('active')).not.toBeInTheDocument();
  });

  it('applies green color for active status', () => {
    render(<StatusTag status="active" />);
    const tag = screen.getByText('active');
    // Ant Tag with color='green' applies the class
    expect(tag.closest('.ant-tag-green')).toBeTruthy();
  });

  it('applies red color for high risk', () => {
    render(<StatusTag status="high" />);
    const tag = screen.getByText('high');
    expect(tag.closest('.ant-tag-red')).toBeTruthy();
  });

  it('applies default color for unknown status', () => {
    render(<StatusTag status="unknown_status" />);
    const tag = screen.getByText('unknown_status');
    // Default tags don't have a specific color class
    expect(tag.closest('.ant-tag')).toBeTruthy();
  });

  it('custom color overrides built-in map', () => {
    render(<StatusTag status="active" color="purple" />);
    const tag = screen.getByText('active');
    expect(tag.closest('.ant-tag-purple')).toBeTruthy();
  });
});

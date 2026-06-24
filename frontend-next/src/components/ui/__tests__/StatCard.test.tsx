import { describe, it, expect } from 'vitest';
import { render, screen } from '@testing-library/react';
import StatCard from '@/components/ui/StatCard';

describe('StatCard', () => {
  it('renders title and value', () => {
    render(<StatCard title="订单数" value={42} />);
    expect(screen.getByText('订单数')).toBeInTheDocument();
    expect(screen.getByText('42')).toBeInTheDocument();
  });

  it('renders dash when value is null', () => {
    render(<StatCard title="收入" value={null} />);
    expect(screen.getByText('-')).toBeInTheDocument();
  });

  it('renders icon with background color', () => {
    render(
      <StatCard title="测试" value={100} icon={<span>💰</span>} iconBgColor="#fffbe6" />
    );
    expect(screen.getByText('💰')).toBeInTheDocument();
  });

  it('renders prefix and suffix', () => {
    render(<StatCard title="收入" value={100} prefix="¥" suffix="元" />);
    expect(screen.getByText('¥')).toBeInTheDocument();
    expect(screen.getByText('元')).toBeInTheDocument();
  });

  it('renders up trend with green color', () => {
    render(<StatCard title="测试" value={100} trend={{ value: 12.5 }} />);
    expect(screen.getByText('12.5%')).toBeInTheDocument();
  });

  it('renders down trend', () => {
    render(<StatCard title="测试" value={100} trend={{ value: -5.3 }} />);
    expect(screen.getByText('5.3%')).toBeInTheDocument();
  });

  it('renders trend with label', () => {
    render(<StatCard title="测试" value={100} trend={{ value: 5, label: '较上月' }} />);
    expect(screen.getByText('较上月')).toBeInTheDocument();
  });
});

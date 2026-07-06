import { describe, it, expect, vi } from 'vitest';
import { render, screen, fireEvent } from '@testing-library/react';
import RiskConfirmDialog from '@/components/ui/RiskConfirmDialog';

describe('RiskConfirmDialog', () => {
  const baseChanges = [
    { label: '价格', before: '¥100', after: '¥120' },
    { label: '库存', before: '50', after: '30' },
  ];
  const baseProps = {
    open: true,
    target: 'SKU-001',
    risk: 'high' as const,
    changes: baseChanges,
    onOk: vi.fn(),
    onCancel: vi.fn(),
  };

  it('renders the dialog with target object', () => {
    render(<RiskConfirmDialog {...baseProps} />);
    expect(screen.getByText('SKU-001')).toBeInTheDocument();
  });

  it('shows risk level tag', () => {
    render(<RiskConfirmDialog {...baseProps} risk="high" />);
    expect(screen.getByText('高风险')).toBeInTheDocument();
  });

  it('shows before/after values when provided', () => {
    render(<RiskConfirmDialog {...baseProps} />);
    expect(screen.getByText('¥100')).toBeInTheDocument();
    expect(screen.getByText('¥120')).toBeInTheDocument();
    expect(screen.getByText('50')).toBeInTheDocument();
    expect(screen.getByText('30')).toBeInTheDocument();
  });

  it('shows audit destination', () => {
    render(
      <RiskConfirmDialog
        {...baseProps}
        auditDestination="操作日志可追溯至 operation_log 表"
      />
    );
    expect(screen.getByText('操作日志可追溯至 operation_log 表')).toBeInTheDocument();
  });

  it('shows risk level for medium', () => {
    render(<RiskConfirmDialog {...baseProps} risk="medium" />);
    expect(screen.getByText('中风险')).toBeInTheDocument();
  });

  it('shows risk level for low', () => {
    render(<RiskConfirmDialog {...baseProps} risk="low" />);
    expect(screen.getByText('低风险')).toBeInTheDocument();
  });

  it('calls onCancel when cancel button is clicked', () => {
    const onCancel = vi.fn();
    render(<RiskConfirmDialog {...baseProps} onCancel={onCancel} />);
    fireEvent.click(screen.getByRole('button', { name: /取/ }));
    expect(onCancel).toHaveBeenCalled();
  });

  it('calls onOk when confirm button is clicked', () => {
    const onOk = vi.fn();
    render(<RiskConfirmDialog {...baseProps} onOk={onOk} />);
    fireEvent.click(screen.getByRole('button', { name: /确认执行/ }));
    expect(onOk).toHaveBeenCalled();
  });

  it('does not render when closed', () => {
    render(<RiskConfirmDialog {...baseProps} open={false} />);
    expect(screen.queryByText('SKU-001')).not.toBeInTheDocument();
  });

  it("renders without 'changes' array when empty", () => {
    // config: changes default value is an empty array
    render(<RiskConfirmDialog {...baseProps} changes={[]} />);
    expect(screen.queryByText('¥100')).not.toBeInTheDocument();
  });
});

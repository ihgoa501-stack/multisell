import { describe, it, expect, vi } from 'vitest';
import { render, screen, fireEvent } from '@testing-library/react';
import HighRiskConfirmDialog from '@/components/ui/HighRiskConfirmDialog';

describe('HighRiskConfirmDialog', () => {
  const baseProps = {
    open: true,
    actionName: 'execute',
    onConfirm: vi.fn(),
    onCancel: vi.fn(),
  };

  it('renders the dialog with action name', () => {
    render(<HighRiskConfirmDialog {...baseProps} />);
    expect(screen.getByText('高风险操作确认')).toBeInTheDocument();
    expect(screen.getByText(/execute/)).toBeInTheDocument();
  });

  it('shows risk level tag', () => {
    render(<HighRiskConfirmDialog {...baseProps} riskLevel="high" />);
    expect(screen.getByText('高风险')).toBeInTheDocument();
  });

  it('shows environment mode', () => {
    render(<HighRiskConfirmDialog {...baseProps} environmentMode="sandbox" />);
    expect(screen.getByText('Sandbox（沙箱）')).toBeInTheDocument();
  });

  it('shows before/after values when provided', () => {
    render(
      <HighRiskConfirmDialog
        {...baseProps}
        detail={{
          targetLabel: 'SKU-001',
          beforeValue: '¥100',
          afterValue: '¥120',
        }}
      />
    );
    expect(screen.getByText('SKU-001')).toBeInTheDocument();
    expect(screen.getByText('¥100')).toBeInTheDocument();
    expect(screen.getByText('¥120')).toBeInTheDocument();
  });

  it('shows audit destination', () => {
    render(
      <HighRiskConfirmDialog
        {...baseProps}
        auditDestination="操作日志可追溯至 operation_log 表"
      />
    );
    expect(screen.getByText('操作日志可追溯至 operation_log 表')).toBeInTheDocument();
  });

  it('shows rollback note', () => {
    render(
      <HighRiskConfirmDialog
        {...baseProps}
        rollbackNote="此操作不可回滚"
      />
    );
    expect(screen.getByText('此操作不可回滚')).toBeInTheDocument();
  });

  it('shows reason textarea when showReason is true', () => {
    render(<HighRiskConfirmDialog {...baseProps} showReason />);
    expect(screen.getByPlaceholderText('补充说明（选填）')).toBeInTheDocument();
  });

  it('calls onCancel when cancel button is clicked', () => {
    const onCancel = vi.fn();
    render(<HighRiskConfirmDialog {...baseProps} onCancel={onCancel} />);
    fireEvent.click(screen.getByText('取消'));
    expect(onCancel).toHaveBeenCalled();
  });

  it('does not render when closed', () => {
    render(<HighRiskConfirmDialog {...baseProps} open={false} />);
    expect(screen.queryByText('高风险操作确认')).not.toBeInTheDocument();
  });
});

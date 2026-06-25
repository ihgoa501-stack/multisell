import { describe, it, expect, vi } from 'vitest';
import { render, screen } from '@testing-library/react';
import ConfirmDialog from '@/components/ui/ConfirmDialog';

describe('ConfirmDialog', () => {
  it('renders title and content', () => {
    render(
      <ConfirmDialog open title="确认删除" content="确定要删除这条记录吗？" onOk={vi.fn()} onCancel={vi.fn()} />
    );
    expect(screen.getByText('确认删除')).toBeInTheDocument();
    expect(screen.getByText('确定要删除这条记录吗？')).toBeInTheDocument();
  });

  it('renders description when provided', () => {
    render(
      <ConfirmDialog
        open
        title="确认"
        content="操作不可逆"
        description="此操作无法撤销"
        onOk={vi.fn()}
        onCancel={vi.fn()}
      />
    );
    expect(screen.getByText('此操作无法撤销')).toBeInTheDocument();
  });

  it('shows reason textarea when showReason=true', () => {
    render(
      <ConfirmDialog
        open
        title="驳回"
        content="确定要驳回吗？"
        showReason
        onOk={vi.fn()}
        onCancel={vi.fn()}
      />
    );
    const textarea = screen.getByPlaceholderText('请输入原因（选填）');
    expect(textarea).toBeInTheDocument();
  });

  it('uses custom ok button text', () => {
    render(
      <ConfirmDialog open title="确认" content="确认操作" okText="确定" onOk={vi.fn()} onCancel={vi.fn()} />
    );
    expect(screen.getByText((content) => content.replace(/\s/g, '').includes('确定'))).toBeInTheDocument();
  });
});

import { useState } from 'react';
import { Modal, Input, Typography } from 'antd';
import { ExclamationCircleOutlined } from '@ant-design/icons';

const { Text } = Typography;
const { TextArea } = Input;

export interface ConfirmDialogProps {
  /** Whether the dialog is visible. */
  open: boolean;
  /** Dialog title. */
  title: string;
  /** Main content/message. */
  content: string;
  /** Detailed description (shown below content in smaller text). */
  description?: string;
  /** Label for the confirm button. @default '确认' */
  okText?: string;
  /** Label for the cancel button. @default '取消' */
  cancelText?: string;
  /** Confirm button type. @default 'primary' */
  okType?: 'primary' | 'danger' | 'default';
  /** Whether the confirm action is loading. */
  confirmLoading?: boolean;
  /** Whether to show a reason textarea. */
  showReason?: boolean;
  /** Placeholder for the reason textarea. */
  reasonPlaceholder?: string;
  /** Callback on confirm. */
  onOk: (reason?: string) => void;
  /** Callback on cancel. */
  onCancel: () => void;
  /** Risk level indicator. @default 'default' */
  risk?: 'low' | 'medium' | 'high';
}

const riskColors: Record<string, string> = {
  low: '#52c41a',
  medium: '#fa8c16',
  high: '#f5222d',
};

export default function ConfirmDialog({
  open,
  title,
  content,
  description,
  okText = '确认',
  cancelText = '取消',
  okType = 'primary',
  confirmLoading,
  showReason,
  reasonPlaceholder = '请输入原因（选填）',
  onOk,
  onCancel,
  risk,
}: ConfirmDialogProps) {
  const [reason, setReason] = useState('');

  const handleOk = () => {
    onOk(showReason ? reason : undefined);
    if (showReason) setReason('');
  };

  const handleCancel = () => {
    setReason('');
    onCancel();
  };

  return (
    <Modal
      open={open}
      title={
        <span>
          {risk && (
            <ExclamationCircleOutlined
              style={{ color: riskColors[risk], marginRight: 8 }}
            />
          )}
          {title}
        </span>
      }
      okText={okText}
      cancelText={cancelText}
      okButtonProps={{ danger: okType === 'danger', type: okType === 'danger' ? 'primary' : undefined }}
      confirmLoading={confirmLoading}
      onOk={handleOk}
      onCancel={handleCancel}
      destroyOnClose
      width={420}
    >
      <div style={{ marginTop: 8 }}>
        <Text>{content}</Text>
        {description && (
          <div style={{ marginTop: 8 }}>
            <Text type="secondary" style={{ fontSize: 13 }}>
              {description}
            </Text>
          </div>
        )}
        {showReason && (
          <div style={{ marginTop: 12 }}>
            <TextArea
              placeholder={reasonPlaceholder}
              value={reason}
              onChange={(e) => setReason(e.target.value)}
              rows={3}
            />
          </div>
        )}
      </div>
    </Modal>
  );
}

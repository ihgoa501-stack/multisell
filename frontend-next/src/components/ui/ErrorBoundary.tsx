'use client';

import { Component, type ReactNode, type ErrorInfo } from 'react';
import { Button, Typography } from 'antd';
import { ReloadOutlined } from '@ant-design/icons';

interface Props {
  children: ReactNode;
  fallback?: ReactNode;
}

interface State {
  hasError: boolean;
  error?: Error;
}

/**
 * Error boundary that catches render errors and shows a retry UI.
 * Usage: wrap page content or component tree.
 */
export class ErrorBoundary extends Component<Props, State> {
  state: State = { hasError: false };

  static getDerivedStateFromError(error: Error): State {
    return { hasError: true, error };
  }

  componentDidCatch(error: Error, info: ErrorInfo) {
    console.error('[ErrorBoundary]', error, info.componentStack);
  }

  handleRetry = () => {
    this.setState({ hasError: false, error: undefined });
  };

  render() {
    if (this.state.hasError) {
      if (this.props.fallback) return this.props.fallback;

      return (
        <div
          style={{
            padding: 'var(--space-2xl)',
            textAlign: 'center',
            color: 'var(--t2)',
          }}
        >
          <Typography.Text type="danger" style={{ fontSize: 'var(--text-h2)', display: 'block' }}>
            页面渲染异常
          </Typography.Text>
          <Typography.Paragraph style={{ color: 'var(--t3)', marginTop: 'var(--space-md)' }}>
            {this.state.error?.message || '未知错误'}
          </Typography.Paragraph>
          <Button icon={<ReloadOutlined />} onClick={this.handleRetry}>
            重试
          </Button>
        </div>
      );
    }

    return this.props.children;
  }
}

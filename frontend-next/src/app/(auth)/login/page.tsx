'use client';

import { Form, Input, Button, message, Typography } from 'antd';
import { UserOutlined, LockOutlined } from '@ant-design/icons';
import { useRouter } from 'next/navigation';
import { useMutation } from '@tanstack/react-query';
import apiClient from '@/lib/api-client';
import { setToken, setRefreshToken, setStoredUser } from '@/lib/auth';
import type { User } from '@/types/api';

const { Text } = Typography;

interface LoginResponse {
  access_token: string;
  refresh_token: string;
  user: User;
}

export default function LoginPage() {
  const router = useRouter();
  const [form] = Form.useForm();

  const loginMutation = useMutation({
    mutationFn: async (values: { username: string; password: string }) => {
      const res = await apiClient.post<LoginResponse>('/v1/auth/login', {
        username: values.username,
        password: values.password,
      });
      return res;
    },
    onSuccess: (res) => {
      if (res.data) {
        setToken(res.data.access_token);
        setRefreshToken(res.data.refresh_token);
        setStoredUser(res.data.user);
        message.success('登录成功');
        router.push('/dashboard');
      } else {
        message.error('登录返回数据为空');
      }
    },
    onError: (err: Error) => {
      message.error(err.message || '登录失败，请检查用户名和密码');
    },
  });

  const handleSubmit = async () => {
    try {
      const values = await form.validateFields();
      loginMutation.mutate(values);
    } catch {
      // Validation error
    }
  };

  return (
    <div
      style={{
        display: 'flex',
        justifyContent: 'center',
        alignItems: 'center',
        minHeight: '100vh',
        background: 'var(--bg)',
        position: 'relative',
        overflow: 'hidden',
        fontFamily: 'var(--body)',
      }}
    >
      {/* Subtle gradient orbs */}
      <div
        style={{
          position: 'absolute',
          width: 480,
          height: 480,
          borderRadius: '50%',
          background: 'radial-gradient(circle, rgba(99,102,241,0.08) 0%, transparent 70%)',
          top: '-20%',
          right: '-10%',
          pointerEvents: 'none',
        }}
      />
      <div
        style={{
          position: 'absolute',
          width: 400,
          height: 400,
          borderRadius: '50%',
          background: 'radial-gradient(circle, rgba(34,211,238,0.06) 0%, transparent 70%)',
          bottom: '-15%',
          left: '-10%',
          pointerEvents: 'none',
        }}
      />

      <div
        style={{
          width: 400,
          position: 'relative',
          zIndex: 1,
        }}
      >
        {/* Brand */}
        <div style={{ textAlign: 'center', marginBottom: 36 }}>
          <div
            style={{
              display: 'inline-flex',
              alignItems: 'center',
              justifyContent: 'center',
              width: 48,
              height: 48,
              borderRadius: 12,
              background: 'linear-gradient(135deg, var(--i5), var(--c5))',
              marginBottom: 16,
              fontSize: '1.2rem',
              color: 'white',
            }}
          >
            ✦
          </div>
          <h1
            style={{
              fontFamily: 'var(--ds)',
              fontWeight: 800,
              fontSize: '1.6rem',
              color: 'var(--t1)',
              letterSpacing: '-0.02em',
              margin: 0,
            }}
          >
            凌镜
          </h1>
          <Text
            style={{
              color: 'var(--t3)',
              display: 'block',
              marginTop: 4,
              fontSize: '0.85rem',
              fontFamily: 'var(--body)',
            }}
          >
            跨境电商 AI Agent 工作台
          </Text>
        </div>

        {/* Login card */}
        <div
          style={{
            background: 'var(--s1)',
            border: '1px solid var(--bd)',
            borderRadius: 12,
            padding: '32px 28px',
          }}
        >
          <Form
            form={form}
            layout="vertical"
            requiredMark={false}
            onFinish={handleSubmit}
          >
            <Form.Item
              name="username"
              label="用户名或邮箱"
              rules={[{ required: true, message: '请输入用户名/邮箱' }]}
              style={{ marginBottom: 20 }}
            >
              <Input
                aria-label="用户名或邮箱"
                prefix={<UserOutlined style={{ color: 'var(--t3)' }} />}
                placeholder="用户名 / 邮箱"
                size="large"
                style={{
                  background: 'var(--s2)',
                  border: '1px solid var(--bd)',
                  color: 'var(--t1)',
                  borderRadius: 8,
                  fontFamily: 'var(--body)',
                  height: 48,
                }}
              />
            </Form.Item>

            <Form.Item
              name="password"
              label="密码"
              rules={[{ required: true, message: '请输入密码' }]}
              style={{ marginBottom: 28 }}
            >
              <Input.Password
                aria-label="密码"
                prefix={<LockOutlined style={{ color: 'var(--t3)' }} />}
                placeholder="密码"
                size="large"
                style={{
                  background: 'var(--s2)',
                  border: '1px solid var(--bd)',
                  color: 'var(--t1)',
                  borderRadius: 8,
                  fontFamily: 'var(--body)',
                  height: 48,
                }}
              />
            </Form.Item>

            <Form.Item style={{ marginBottom: 0 }}>
              <Button
                type="primary"
                htmlType="submit"
                size="large"
                loading={loginMutation.isPending}
                style={{
                  width: '100%',
                  background: 'linear-gradient(135deg, var(--i5), var(--c5))',
                  border: 'none',
                  height: 44,
                  borderRadius: 8,
                  fontWeight: 600,
                  fontFamily: 'var(--body)',
                  fontSize: '0.9rem',
                  boxShadow: '0 2px 12px rgba(99,102,241,0.25)',
                }}
              >
                登录
              </Button>
            </Form.Item>
          </Form>
        </div>

        {/* Footer */}
        <div
          style={{
            textAlign: 'center',
            marginTop: 24,
            fontSize: '0.72rem',
            color: 'var(--t4)',
            fontFamily: 'var(--body)',
          }}
        >
          LingMirror v0.2.0 · 跨境电商 AI AgentOS
        </div>
      </div>
    </div>
  );
}

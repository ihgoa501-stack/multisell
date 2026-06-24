'use client';

import { useState } from 'react';
import { Form, Input, Button, Card, Typography, message } from 'antd';
import { UserOutlined, LockOutlined, ThunderboltOutlined } from '@ant-design/icons';
import { useRouter } from 'next/navigation';
import { useMutation } from '@tanstack/react-query';
import apiClient from '@/lib/api-client';
import { setToken, setStoredUser } from '@/lib/auth';
import type { User } from '@/types/api';

const { Title, Text } = Typography;

interface LoginResponse {
  access_token: string;
  refresh_token: string;
  user: User;
}

export default function LoginPage() {
  const router = useRouter();
  const [form] = Form.useForm();

  const loginMutation = useMutation({
    mutationFn: async (values: any) => {
      // The backend expects username and password
      const res = await apiClient.post<LoginResponse>('/v1/auth/login', {
        username: values.username,
        password: values.password,
      });
      return res;
    },
    onSuccess: (res) => {
      if (res.data) {
        setToken(res.data.access_token);
        setStoredUser(res.data.user);
        message.success('登录成功');
        router.push('/dashboard');
      } else {
        message.error('登录返回数据为空');
      }
    },
    onError: (err: any) => {
      message.error(err.message || '登录失败，请检查用户名和密码');
    },
  });

  const handleSubmit = async () => {
    try {
      const values = await form.validateFields();
      loginMutation.mutate(values);
    } catch (e) {
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
        background: 'radial-gradient(circle at 10% 20%, rgb(18, 16, 28) 0%, rgb(10, 8, 16) 90%)',
        position: 'relative',
        overflow: 'hidden',
        fontFamily: 'Inter, sans-serif',
      }}
    >
      {/* Decorative background design blobs */}
      <div
        style={{
          position: 'absolute',
          width: '500px',
          height: '500px',
          borderRadius: '50%',
          background: 'radial-gradient(circle, rgba(147, 51, 234, 0.15) 0%, rgba(147, 51, 234, 0) 70%)',
          top: '-10%',
          right: '-10%',
          filter: 'blur(40px)',
          zIndex: 0,
        }}
      />
      <div
        style={{
          position: 'absolute',
          width: '600px',
          height: '600px',
          borderRadius: '50%',
          background: 'radial-gradient(circle, rgba(59, 130, 246, 0.15) 0%, rgba(59, 130, 246, 0) 70%)',
          bottom: '-10%',
          left: '-10%',
          filter: 'blur(50px)',
          zIndex: 0,
        }}
      />

      <Card
        style={{
          width: 420,
          background: 'rgba(255, 255, 255, 0.03)',
          backdropFilter: 'blur(20px)',
          border: '1px solid rgba(255, 255, 255, 0.08)',
          borderRadius: '16px',
          boxShadow: '0 25px 50px -12px rgba(0, 0, 0, 0.5)',
          zIndex: 1,
        }}
        bodyStyle={{ padding: '40px 32px' }}
      >
        <div style={{ textAlign: 'center', marginBottom: 32 }}>
          <div
            style={{
              display: 'inline-flex',
              alignItems: 'center',
              justifyContent: 'center',
              width: 56,
              height: 56,
              borderRadius: '14px',
              background: 'linear-gradient(135deg, #3b82f6 0%, #8b5cf6 100%)',
              marginBottom: 16,
              boxShadow: '0 8px 16px rgba(139, 92, 246, 0.3)',
            }}
          >
            <ThunderboltOutlined style={{ fontSize: 28, color: '#fff' }} />
          </div>
          <Title level={2} style={{ color: '#fff', margin: 0, fontWeight: 700, letterSpacing: '-0.5px' }}>
            凌镜 LingMirror
          </Title>
          <Text style={{ color: 'rgba(255, 255, 255, 0.45)', display: 'block', marginTop: 8 }}>
            跨境电商智能 Agent 协同系统
          </Text>
        </div>

        <Form
          form={form}
          layout="vertical"
          requiredMark={false}
          onFinish={handleSubmit}
        >
          <Form.Item
            name="username"
            rules={[{ required: true, message: '请输入用户名/邮箱' }]}
          >
            <Input
              aria-label="Email"
              prefix={<UserOutlined style={{ color: 'rgba(255, 255, 255, 0.35)' }} />}
              placeholder="用户名 / 邮箱"
              size="large"
              style={{
                background: 'rgba(255, 255, 255, 0.05)',
                border: '1px solid rgba(255, 255, 255, 0.1)',
                color: '#fff',
                borderRadius: '8px',
              }}
            />
          </Form.Item>

          <Form.Item
            name="password"
            rules={[{ required: true, message: '请输入密码' }]}
            style={{ marginBottom: 32 }}
          >
            <Input.Password
              aria-label="Password"
              prefix={<LockOutlined style={{ color: 'rgba(255, 255, 255, 0.35)' }} />}
              placeholder="密码"
              size="large"
              style={{
                background: 'rgba(255, 255, 255, 0.05)',
                border: '1px solid rgba(255, 255, 255, 0.1)',
                color: '#fff',
                borderRadius: '8px',
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
                background: 'linear-gradient(135deg, #3b82f6 0%, #8b5cf6 100%)',
                border: 'none',
                height: '48px',
                borderRadius: '8px',
                fontWeight: 600,
                boxShadow: '0 4px 12px rgba(139, 92, 246, 0.25)',
              }}
            >
              登录
            </Button>
          </Form.Item>
        </Form>
      </Card>
    </div>
  );
}

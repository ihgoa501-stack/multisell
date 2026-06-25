'use client';

import { Switch } from 'antd';
import CrudListPage from '@/components/crud/CrudListPage';

export default function SupportTemplatesPage() {
  return (
    <CrudListPage
      resource="/support/templates"
      title="回复模板"
      singular="模板"
      searchPlaceholder="搜索模板名称 / 分类 / 内容..."
      columns={[
        { title: '模板名称', dataIndex: 'name', width: 180 },
        { title: '分类', dataIndex: 'category', width: 120 },
        { title: '内容', dataIndex: 'content', width: 360 },
        { title: '平台', dataIndex: 'platform', width: 120 },
        {
          title: '启用',
          dataIndex: 'enabled',
          width: 80,
          render: (v) => (
            <Switch checked={v === 1 || v === true} size="small" disabled />
          ),
        },
      ]}
      fields={[
        { name: 'name', label: '模板名称', required: true },
        {
          name: 'category',
          label: '分类',
          type: 'select',
          initialValue: 'general',
          options: [
            { label: '通用', value: 'general' },
            { label: '售后', value: 'aftersales' },
            { label: '物流', value: 'shipping' },
            { label: '商品咨询', value: 'product_inquiry' },
            { label: '其他', value: 'other' },
          ],
        },
        { name: 'content', label: '内容', type: 'textarea', required: true },
        { name: 'platform', label: '平台' },
        {
          name: 'enabled',
          label: '启用',
          type: 'select',
          initialValue: 1,
          options: [
            { label: '是', value: 1 },
            { label: '否', value: 0 },
          ],
        },
      ]}
      filters={[
        {
          key: 'category',
          label: '分类',
          options: [
            { label: '通用', value: 'general' },
            { label: '售后', value: 'aftersales' },
            { label: '物流', value: 'shipping' },
            { label: '商品咨询', value: 'product_inquiry' },
            { label: '其他', value: 'other' },
          ],
        },
        {
          key: 'enabled',
          label: '启用',
          options: [
            { label: '是', value: 1 },
            { label: '否', value: 0 },
          ],
        },
      ]}
    />
  );
}

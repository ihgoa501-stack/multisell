/**
 * 凌镜 LingMirror — Naive UI 主题配置
 *
 * 统一所有组件的视觉风格。
 * 使用方式: <n-config-provider :theme-overrides="themeOverrides">
 */
import type { GlobalThemeOverrides } from 'naive-ui'

export const themeOverrides: GlobalThemeOverrides = {
  common: {
    fontFamily: 'Inter, Noto Sans SC, system-ui, sans-serif',
    fontFamilyMono: 'JetBrains Mono, SF Mono, Cascadia Code, monospace',

    // 主色
    primaryColor: '#2962FF',
    primaryColorHover: '#1A4FDB',
    primaryColorPressed: '#173EB3',
    primaryColorSuppl: '#2962FF',

    // 信息
    infoColor: '#2962FF',
    infoColorHover: '#1A4FDB',

    // 成功
    successColor: '#0F973D',
    successColorHover: '#0B7A31',

    // 警告
    warningColor: '#B45309',
    warningColorHover: '#92400E',

    // 错误
    errorColor: '#B91C1C',
    errorColorHover: '#991B1B',

    // 中性色
    bodyColor: '#F6F7F9',
    cardColor: '#FFFFFF',
    modalColor: '#FFFFFF',
    popoverColor: '#FFFFFF',
    tableColor: '#FFFFFF',

    // 文字
    textColor1: '#1A1D23',
    textColor2: '#6B7280',
    textColor3: '#9CA3AF',

    // 边框
    borderColor: '#E8EAED',
    dividerColor: '#F0F1F3',

    // 圆角
    borderRadius: '8px',
    borderRadiusSmall: '6px',

    // 阴影
    boxShadow1: '0 1px 2px rgba(0,0,0,0.04)',
    boxShadow2: '0 1px 3px rgba(0,0,0,0.06), 0 1px 2px rgba(0,0,0,0.04)',
    boxShadow3: '0 4px 6px -1px rgba(0,0,0,0.06), 0 2px 4px -1px rgba(0,0,0,0.04)',
  },

  // 按钮
  Button: {
    fontSizeSmall: '12px',
    fontSizeMedium: '13px',
    fontSizeLarge: '14px',
    paddingSmall: '4px 12px',
    paddingMedium: '6px 16px',
    paddingLarge: '8px 20px',
    borderRadius: '6px',
    fontWeight: '500',
  },

  // 输入框
  Input: {
    fontSizeMedium: '13px',
    borderRadius: '6px',
    heightMedium: '34px',
    paddingMedium: '0 10px',
  },

  // 选择器
  Select: {
    fontSizeMedium: '13px',
    borderRadius: '6px',
    heightMedium: '34px',
  },

  // 表格
  DataTable: {
    fontSizeMedium: '13px',
    thColor: '#FFFFFF',
    thTextColor: '#6B7280',
    thFontWeight: '600',
    thPadding: '10px 16px',
    tdPadding: '10px 16px',
    borderColor: '#F0F1F3',
    tdColorHover: '#F8F9FA',
    borderRadius: '8px',
  },

  // 标签
  Tag: {
    fontSizeMedium: '11px',
    borderRadius: '6px',
    padding: '1px 8px',
  },

  // 卡片
  Card: {
    paddingMedium: '16px 20px',
    borderRadius: '8px',
    borderColor: '#F0F1F3',
    titleFontSizeMedium: '14px',
    titleFontWeight: '600',
  },

  // 步骤条
  Steps: {
    stepHeaderFontSizeMedium: '13px',
  },

  // 对话框
  Dialog: {
    borderRadius: '10px',
  },

  // 消息
  Message: {
    borderRadius: '8px',
  },

  // 通知
  Notification: {
    borderRadius: '8px',
  },

  // 菜单
  Menu: {
    itemHeight: '38px',
    borderRadius: '6px',
    fontSize: '13px',
  },

  // 分页
  Pagination: {
    itemSize: '30px',
    itemPadding: '0 8px',
    inputWidth: '60px',
  },
}

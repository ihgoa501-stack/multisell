/**
 * 凌镜 LingMirror — Ant Design Vue 主题配置
 *
 * AgentOS 品牌视觉系统。
 * 使用方式: <a-config-provider :theme="antdTheme">
 */
import type { ThemeConfig } from 'ant-design-vue/es/config-provider/context'

/**
 * AgentOS 品牌色板
 * - 主色 #2962FF：信任、科技、专业
 * - 辅色 #7C3AED：AI、智能、创新
 * - 成功 #059669：健康、增长
 * - 警告 #D97706：注意、待处理
 * - 危险 #DC2626：异常、紧急
 */
export const brandColors = {
  primary: '#2962FF',
  primaryHover: '#1A4FDB',
  primaryActive: '#173EB3',
  primaryBg: '#EBF0FF',

  secondary: '#7C3AED',
  secondaryHover: '#6D28D9',
  secondaryBg: '#F3EEFF',

  success: '#059669',
  successBg: '#ECFDF5',

  warning: '#D97706',
  warningBg: '#FFFBEB',

  danger: '#DC2626',
  dangerBg: '#FEF2F2',

  // 中性色
  textPrimary: '#111827',
  textSecondary: '#6B7280',
  textTertiary: '#9CA3AF',
  textDisabled: '#D1D5DB',

  border: '#E5E7EB',
  borderSecondary: '#F3F4F6',

  bgLayout: '#F9FAFB',
  bgContainer: '#FFFFFF',
  bgElevated: '#FFFFFF',

  // Agent 专用色
  agentOnline: '#059669',
  agentIdle: '#D97706',
  agentOffline: '#9CA3AF',
  agentThinking: '#7C3AED',
}

export const antdTheme: ThemeConfig = {
  token: {
    // 品牌色
    colorPrimary: brandColors.primary,
    colorSuccess: brandColors.success,
    colorWarning: brandColors.warning,
    colorError: brandColors.danger,
    colorInfo: brandColors.primary,

    // 文字
    colorText: brandColors.textPrimary,
    colorTextSecondary: brandColors.textSecondary,
    colorTextTertiary: brandColors.textTertiary,

    // 背景
    colorBgLayout: brandColors.bgLayout,
    colorBgContainer: brandColors.bgContainer,
    colorBgElevated: brandColors.bgElevated,

    // 边框
    colorBorder: brandColors.border,
    colorBorderSecondary: brandColors.borderSecondary,

    // 圆角
    borderRadius: 8,
    borderRadiusSM: 6,
    borderRadiusLG: 12,

    // 字体
    fontFamily: 'Inter, "Noto Sans SC", system-ui, -apple-system, sans-serif',
    // 等宽字体（通过 CSS 全局设置）
    fontSize: 14,
    fontSizeSM: 12,
    fontSizeLG: 16,

    // 阴影
    boxShadow: '0 1px 2px 0 rgba(0, 0, 0, 0.03), 0 1px 6px -1px rgba(0, 0, 0, 0.02), 0 2px 4px 0 rgba(0, 0, 0, 0.02)',
    boxShadowSecondary: '0 6px 16px 0 rgba(0, 0, 0, 0.08), 0 3px 6px -4px rgba(0, 0, 0, 0.12), 0 9px 28px 8px rgba(0, 0, 0, 0.05)',

    // 间距
    padding: 16,
    paddingSM: 12,
    paddingLG: 24,
    paddingXS: 8,
    margin: 16,
    marginSM: 12,
    marginLG: 24,
    marginXS: 8,

    // 控件高度
    controlHeight: 36,
    controlHeightSM: 28,
    controlHeightLG: 40,
  },

  components: {
    Button: {
      fontWeight: 500,
      borderRadius: 8,
      borderRadiusSM: 6,
      borderRadiusLG: 10,
    },
    Card: {
      paddingLG: 20,
    },
    Table: {
      headerBg: '#FAFAFA',
      headerColor: brandColors.textSecondary,
      headerSplitColor: 'transparent',
      rowHoverBg: '#F9FAFB',
      borderColor: brandColors.borderSecondary,
      cellPaddingBlock: 12,
      cellPaddingInline: 16,
      headerBorderRadius: 8,
    },
    Menu: {
      itemBorderRadius: 8,
      itemMarginInline: 8,
      itemPaddingInline: 12,
      itemHeight: 40,
      itemSelectedBg: brandColors.primaryBg,
      itemSelectedColor: brandColors.primary,
      iconSize: 18,
    },
    Tag: {
      borderRadiusSM: 6,
    },
    Input: {
      borderRadius: 8,
      controlHeight: 36,
    },
    Select: {
      borderRadius: 8,
      controlHeight: 36,
    },
    Drawer: {
      paddingLG: 24,
    },
    Modal: {
      borderRadiusLG: 16,
      paddingContentHorizontalLG: 24,
      titleFontSize: 18,
    },
    Tabs: {
      itemColor: brandColors.textSecondary,
      itemSelectedColor: brandColors.primary,
      inkBarColor: brandColors.primary,
    },
    Badge: {
      statusSize: 8,
    },
  } as Record<string, any>,
}

/** Agent 状态颜色映射 */
export const agentStatusColors: Record<string, string> = {
  online: brandColors.agentOnline,
  idle: brandColors.agentIdle,
  offline: brandColors.agentOffline,
  thinking: brandColors.agentThinking,
}

/** 风险等级颜色映射 */
export const riskLevelColors: Record<string, string> = {
  critical: brandColors.danger,
  high: brandColors.warning,
  medium: brandColors.primary,
  low: brandColors.textTertiary,
}

/** 自治等级颜色映射 */
export const autonomyColors: Record<string, string> = {
  observation: '#9CA3AF',
  suggestion: '#3B82F6',
  semi_autonomous: '#8B5CF6',
  full_autonomous: '#059669',
}

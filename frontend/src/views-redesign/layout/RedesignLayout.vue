<!-- ================================================
   FLOW: Global Layout & Navigation
   SCREEN 1 of 3: App Shell
   ------------------------------------------------
   ENTRY:  User login → default route
   EXIT:   Navigate to any business page
   BRANCH: ⌘K → Command Palette / Bell → Notification Drawer
   ================================================ -->
<template>
  <a-config-provider :theme="antdTheme">
    <a-layout class="redesign-layout" style="min-height: 100vh">
      <!-- ═══ Header ═══ -->
      <a-layout-header class="app-header">
        <div class="header-left">
          <div class="brand" @click="router.push('/redesign/dashboard')">
            <img src="/brand/lingmirror-icon.png" alt="凌镜" class="brand-icon" />
            <span v-if="!collapsed" class="brand-text">凌镜</span>
          </div>
          <a-button
            type="text"
            size="small"
            class="collapse-btn"
            @click="collapsed = !collapsed"
          >
            <template #icon>
              <component :is="collapsed ? MenuUnfoldOutlined : MenuFoldOutlined" />
            </template>
          </a-button>
        </div>

        <div class="header-center">
          <div class="cmd-trigger" @click="showCmdPalette = true">
            <SearchOutlined style="margin-right: 6px; opacity: 0.5" />
            <span class="cmd-placeholder">搜索页面、商品、Agent...</span>
            <span class="cmd-shortcut">⌘K</span>
          </div>
        </div>

        <div class="header-right">
          <a-badge :count="unreadCount" :overflow-count="99">
            <a-button type="text" class="header-btn" @click="showNotifDrawer = true">
              <template #icon><BellOutlined /></template>
            </a-button>
          </a-badge>
          <a-dropdown :trigger="['click']">
            <div class="user-area">
              <a-avatar :size="28" style="background-color: #2962FF">
                {{ userDisplayName.charAt(0) }}
              </a-avatar>
              <span class="user-name">{{ userDisplayName }}</span>
              <DownOutlined style="font-size: 10px; opacity: 0.5" />
            </div>
            <template #overlay>
              <a-menu @click="handleUserMenu">
                <a-menu-item key="profile"><UserOutlined /> 个人设置</a-menu-item>
                <a-menu-divider />
                <a-menu-item key="logout"><LogoutOutlined /> 退出登录</a-menu-item>
              </a-menu>
            </template>
          </a-dropdown>
        </div>
      </a-layout-header>

      <a-layout>
        <!-- ═══ Sidebar ═══ -->
        <a-layout-sider
          v-model:collapsed="collapsed"
          :trigger="null"
          collapsible
          :width="240"
          :collapsed-width="64"
          theme="dark"
          class="app-sidebar"
        >
          <a-menu
            v-model:selectedKeys="selectedKeys"
            v-model:openKeys="openKeys"
            mode="inline"
            theme="dark"
            class="sidebar-menu"
            @click="handleMenuClick"
          >
            <template v-for="group in mockMenuGroups" :key="group.key">
              <a-menu-item-group v-if="!collapsed" :title="group.label">
                <a-menu-item v-for="item in group.items" :key="item.key">
                  <template #icon>
                    <component :is="getIcon(item.icon)" />
                  </template>
                  <span>{{ item.label }}</span>
                  <a-badge
                    v-if="item.badge"
                    :count="item.badge"
                    :number-style="{ backgroundColor: '#DC2626', fontSize: '10px', minWidth: '16px', height: '16px', lineHeight: '16px' }"
                    style="margin-left: auto"
                  />
                </a-menu-item>
              </a-menu-item-group>
              <template v-else>
                <a-menu-item v-for="item in group.items" :key="item.key">
                  <template #icon>
                    <a-tooltip :title="item.label" placement="right">
                      <component :is="getIcon(item.icon)" />
                    </a-tooltip>
                  </template>
                </a-menu-item>
              </template>
            </template>
          </a-menu>
        </a-layout-sider>

        <!-- ═══ Content ═══ -->
        <a-layout-content class="app-content">
          <router-view />
        </a-layout-content>
      </a-layout>
    </a-layout>

    <!-- ═══ Command Palette ═══ -->
    <a-modal
      v-model:open="showCmdPalette"
      :footer="null"
      :closable="false"
      width="560"
      class="cmd-modal"
      @after-open-change="onCmdOpenChange"
    >
      <template #title>
        <div class="cmd-header">
          <SearchOutlined style="color: var(--ant-color-text-tertiary); font-size: 18px" />
          <input
            ref="cmdInputRef"
            v-model="cmdQuery"
            class="cmd-input"
            placeholder="搜索页面、商品、Agent 或输入命令..."
            @keyup.escape="showCmdPalette = false"
            @keyup.enter="executeFirstResult"
          />
        </div>
      </template>
      <div class="cmd-body">
        <div v-if="filteredCmdItems.length === 0" class="cmd-empty">
          <SearchOutlined style="font-size: 24px; color: var(--ant-color-text-quaternary)" />
          <p>没有找到匹配结果</p>
        </div>
        <div v-else>
          <template v-for="cat in cmdCategories" :key="cat.key">
            <div v-if="cat.items.length" class="cmd-category">
              <div class="cmd-cat-label">{{ cat.label }}</div>
              <div
                v-for="item in cat.items"
                :key="item.id"
                class="cmd-item"
                :class="{ 'cmd-item-active': item.id === activeCmdId }"
                @click="executeCmd(item)"
                @mouseenter="activeCmdId = item.id"
              >
                <component :is="getIcon(item.icon)" class="cmd-item-icon" />
                <div class="cmd-item-text">
                  <div class="cmd-item-label">{{ item.label }}</div>
                  <div v-if="item.description" class="cmd-item-desc">{{ item.description }}</div>
                </div>
                <span v-if="item.shortcut" class="cmd-item-shortcut">{{ item.shortcut }}</span>
                <ArrowRightOutlined class="cmd-item-arrow" />
              </div>
            </div>
          </template>
        </div>
      </div>
      <div class="cmd-footer">
        <span><kbd>↑↓</kbd> 导航</span>
        <span><kbd>↵</kbd> 执行</span>
        <span><kbd>esc</kbd> 关闭</span>
      </div>
    </a-modal>

    <!-- ═══ Notification Drawer ═══ -->
    <a-drawer
      v-model:open="showNotifDrawer"
      title="通知中心"
      placement="right"
      :width="400"
      :body-style="{ padding: '0' }"
    >
      <template #extra>
        <a-button type="link" size="small" @click="markAllRead">全部已读</a-button>
      </template>
      <a-tabs v-model:activeKey="notifTab" style="padding: 0 16px">
        <a-tab-pane key="all" tab="全部" />
        <a-tab-pane key="agent" tab="Agent" />
        <a-tab-pane key="alert" tab="告警" />
        <a-tab-pane key="system" tab="系统" />
      </a-tabs>
      <div class="notif-list">
        <a-empty v-if="filteredNotifs.length === 0" description="暂无通知" style="padding: 40px 0" />
        <div
          v-for="n in filteredNotifs"
          :key="n.id"
          class="notif-item"
          :class="{ 'notif-unread': !n.is_read }"
          @click="handleNotifClick(n)"
        >
          <div class="notif-icon">
            <a-tag :color="notifColor(n.severity)" :bordered="false" style="font-size: 11px">
              {{ notifTypeLabel(n.source_type) }}
            </a-tag>
          </div>
          <div class="notif-content">
            <div class="notif-title" :class="{ 'font-semibold': !n.is_read }">{{ n.title }}</div>
            <div class="notif-desc">{{ n.description }}</div>
            <div class="notif-time">{{ timeAgo(n.created_at) }}</div>
          </div>
          <div v-if="!n.is_read" class="notif-dot" />
        </div>
      </div>
    </a-drawer>
  </a-config-provider>
</template>

<script setup lang="ts">
/* ================================================
   FLOW: Global Layout & Navigation
   SCREEN 1 of 3: App Shell
   ================================================ */
import { ref, computed, onMounted, onUnmounted, nextTick, h } from 'vue'
import { useRouter, useRoute } from 'vue-router'
import { message } from 'ant-design-vue'
import {
  MenuFoldOutlined, MenuUnfoldOutlined, SearchOutlined, BellOutlined,
  DownOutlined, ArrowRightOutlined, UserOutlined, LogoutOutlined,
  DashboardOutlined, RobotOutlined, ShoppingOutlined, AppstoreOutlined,
  DatabaseOutlined, TagOutlined, TeamOutlined, GlobalOutlined,
  UploadOutlined, ExperimentOutlined, SendOutlined, AccountBookOutlined,
  MoneyCollectOutlined, CloudDownloadOutlined, SettingOutlined,
  BarChartOutlined, CloudServerOutlined,
} from '@ant-design/icons-vue'
import { antdTheme } from '@/config/antd-theme'
import type { MenuGroup, Notification, CommandPaletteItem } from '@/views-redesign/shared/types'
import {
  mockMenuGroups,
  mockNotifications,
  mockCommandItems,
  mockUser,
} from '@/views-redesign/shared/mock-data'

// ── State ──
const router = useRouter()
const route = useRoute()

const collapsed = ref(false)
const selectedKeys = ref<string[]>([route.path])
const openKeys = ref<string[]>(['command', 'product'])

// User
const user = ref(mockUser)
const userDisplayName = computed(() => user.value.display_name || user.value.username)

// Notifications
const notifications = ref<Notification[]>([...mockNotifications])
const showNotifDrawer = ref(false)
const notifTab = ref('all')
const unreadCount = computed(() => notifications.value.filter(n => !n.is_read).length)

const filteredNotifs = computed(() => {
  if (notifTab.value === 'all') return notifications.value
  if (notifTab.value === 'agent') return notifications.value.filter(n => n.source_type === 'agent')
  if (notifTab.value === 'alert') return notifications.value.filter(n => n.severity === 'error' || n.severity === 'critical' || n.severity === 'warning')
  return notifications.value.filter(n => n.source_type === 'system' || n.source_type === 'exception')
})

// Command Palette
const showCmdPalette = ref(false)
const cmdQuery = ref('')
const cmdInputRef = ref<HTMLInputElement | null>(null)
const activeCmdId = ref('')

const filteredCmdItems = computed(() => {
  const q = cmdQuery.value.toLowerCase().trim()
  if (!q) return mockCommandItems
  return mockCommandItems.filter(item =>
    item.label.toLowerCase().includes(q) ||
    (item.description?.toLowerCase().includes(q)) ||
    item.category.includes(q)
  )
})

const cmdCategories = computed(() => [
  { key: 'page', label: '页面', items: filteredCmdItems.value.filter(i => i.category === 'page') },
  { key: 'agent', label: 'Agent', items: filteredCmdItems.value.filter(i => i.category === 'agent') },
  { key: 'command', label: '命令', items: filteredCmdItems.value.filter(i => i.category === 'command') },
  { key: 'product', label: '商品', items: filteredCmdItems.value.filter(i => i.category === 'product') },
])

// ── Icon Registry ──
const iconRegistry: Record<string, any> = {
  DashboardOutlined, RobotOutlined, ShoppingOutlined, AppstoreOutlined,
  DatabaseOutlined, TagOutlined, TeamOutlined, GlobalOutlined,
  UploadOutlined, BellOutlined, ExperimentOutlined, SendOutlined,
  AccountBookOutlined, MoneyCollectOutlined, CloudDownloadOutlined,
  BarChartOutlined, CloudServerOutlined, SearchOutlined, SettingOutlined,
  UserOutlined,
}

function getIcon(name: string) {
  return iconRegistry[name] || DashboardOutlined
}

// ── Handlers ──
function handleMenuClick({ key }: { key: string }) {
  router.push(key)
}

function handleUserMenu({ key }: { key: string }) {
  if (key === 'logout') {
    localStorage.removeItem('token')
    localStorage.removeItem('user')
    window.location.href = '/login'
  }
}

function handleNotifClick(n: Notification) {
  if (!n.is_read) {
    n.is_read = true
  }
  if (n.link_url) {
    router.push(n.link_url)
    showNotifDrawer.value = false
  }
}

function markAllRead() {
  notifications.value.forEach(n => { n.is_read = true })
  message.success('已全部标记已读')
}

function notifColor(severity: string): string {
  const map: Record<string, string> = { info: 'blue', warning: 'orange', error: 'red', critical: 'red' }
  return map[severity] || 'default'
}

function notifTypeLabel(type: string): string {
  const map: Record<string, string> = { agent: 'Agent', system: '系统', exception: '异常', listing: '上架' }
  return map[type] || type
}

function timeAgo(t: string): string {
  if (!t) return ''
  const s = Math.floor((Date.now() - new Date(t).getTime()) / 1000)
  if (s < 60) return '刚刚'
  if (s < 3600) return `${Math.floor(s / 60)}分钟前`
  if (s < 86400) return `${Math.floor(s / 3600)}小时前`
  return `${Math.floor(s / 86400)}天前`
}

// Command palette
function onCmdOpenChange(open: boolean) {
  if (open) {
    nextTick(() => {
      cmdInputRef.value?.focus()
      if (filteredCmdItems.value.length) {
        activeCmdId.value = filteredCmdItems.value[0].id
      }
    })
  } else {
    cmdQuery.value = ''
  }
}

function executeCmd(item: CommandPaletteItem) {
  showCmdPalette.value = false
  if (item.action === 'toggle-theme') {
    message.info('主题切换功能开发中')
    return
  }
  router.push(item.action)
}

function executeFirstResult() {
  if (filteredCmdItems.value.length) {
    const active = filteredCmdItems.value.find(i => i.id === activeCmdId.value)
    executeCmd(active || filteredCmdItems.value[0])
  }
}

// Keyboard shortcuts
function handleKeydown(e: KeyboardEvent) {
  if ((e.metaKey || e.ctrlKey) && e.key === 'k') {
    e.preventDefault()
    showCmdPalette.value = !showCmdPalette.value
  }
}

onMounted(() => {
  document.addEventListener('keydown', handleKeydown)
})
onUnmounted(() => {
  document.removeEventListener('keydown', handleKeydown)
})
</script>

<style scoped>
.redesign-layout {
  background: var(--ant-color-bg-layout, #f5f5f5);
}

/* ═══ Header ═══ */
.app-header {
  height: 56px !important;
  line-height: 56px !important;
  padding: 0 16px !important;
  background: linear-gradient(135deg, #1a1d23 0%, #2c3040 100%) !important;
  display: flex !important;
  align-items: center !important;
  justify-content: space-between !important;
  border-bottom: 1px solid rgba(255, 255, 255, 0.06);
  position: sticky;
  top: 0;
  z-index: 100;
}

.header-left {
  display: flex;
  align-items: center;
  gap: 8px;
}

.brand {
  display: flex;
  align-items: center;
  gap: 8px;
  cursor: pointer;
  padding: 4px 8px;
  border-radius: 8px;
  transition: background 0.2s;
}
.brand:hover {
  background: rgba(255, 255, 255, 0.08);
}
.brand-icon {
  width: 28px;
  height: 28px;
  border-radius: 8px;
}
.brand-text {
  color: #fff;
  font-size: 16px;
  font-weight: 700;
  letter-spacing: 1px;
}

.collapse-btn {
  color: rgba(255, 255, 255, 0.45) !important;
}
.collapse-btn:hover {
  color: rgba(255, 255, 255, 0.85) !important;
}

.header-center {
  flex: 1;
  max-width: 480px;
  margin: 0 24px;
}

.cmd-trigger {
  display: flex;
  align-items: center;
  padding: 6px 14px;
  background: rgba(255, 255, 255, 0.08);
  border: 1px solid rgba(255, 255, 255, 0.08);
  border-radius: 8px;
  cursor: pointer;
  transition: all 0.2s;
  color: rgba(255, 255, 255, 0.45);
  font-size: 13px;
}
.cmd-trigger:hover {
  background: rgba(255, 255, 255, 0.12);
  border-color: rgba(255, 255, 255, 0.15);
  color: rgba(255, 255, 255, 0.65);
}
.cmd-placeholder {
  flex: 1;
}
.cmd-shortcut {
  padding: 2px 6px;
  background: rgba(255, 255, 255, 0.1);
  border-radius: 4px;
  font-size: 11px;
  font-family: monospace;
  color: rgba(255, 255, 255, 0.35);
}

.header-right {
  display: flex;
  align-items: center;
  gap: 4px;
}

.header-btn {
  color: rgba(255, 255, 255, 0.55) !important;
  font-size: 16px !important;
}
.header-btn:hover {
  color: rgba(255, 255, 255, 0.85) !important;
}

.user-area {
  display: flex;
  align-items: center;
  gap: 8px;
  padding: 4px 10px;
  border-radius: 8px;
  cursor: pointer;
  transition: background 0.2s;
}
.user-area:hover {
  background: rgba(255, 255, 255, 0.08);
}
.user-name {
  color: rgba(255, 255, 255, 0.75);
  font-size: 13px;
}

/* ═══ Sidebar ═══ */
.app-sidebar {
  background: #1a1d23 !important;
  border-right: 1px solid rgba(255, 255, 255, 0.04);
}
.app-sidebar :deep(.ant-layout-sider-children) {
  background: #1a1d23;
  display: flex;
  flex-direction: column;
}
.sidebar-menu {
  background: transparent !important;
  border-right: none !important;
  flex: 1;
  overflow-y: auto;
  padding: 8px 0;
}
.sidebar-menu :deep(.ant-menu-item-group-title) {
  color: rgba(255, 255, 255, 0.3) !important;
  font-size: 11px !important;
  font-weight: 600 !important;
  text-transform: uppercase !important;
  letter-spacing: 0.5px !important;
  padding: 20px 16px 8px !important;
  line-height: 1 !important;
  height: auto !important;
}
.sidebar-menu :deep(.ant-menu-item) {
  margin: 2px 8px !important;
  border-radius: 8px !important;
  height: 40px !important;
  line-height: 40px !important;
  font-size: 13px !important;
  display: flex !important;
  align-items: center !important;
}
.sidebar-menu :deep(.ant-menu-item-selected) {
  background: rgba(41, 98, 255, 0.15) !important;
  color: #6e9fff !important;
}
.sidebar-menu :deep(.ant-menu-item:not(.ant-menu-item-selected):hover) {
  background: rgba(255, 255, 255, 0.06) !important;
}

/* ═══ Content ═══ */
.app-content {
  padding: 24px;
  background: var(--ant-color-bg-layout, #f5f5f5);
  overflow-y: auto;
}

/* ═══ Command Palette ═══ */
.cmd-modal :deep(.ant-modal-content) {
  padding: 0 !important;
  border-radius: 16px !important;
  overflow: hidden;
}
.cmd-modal :deep(.ant-modal-header) {
  padding: 0 !important;
  margin: 0 !important;
  border-bottom: 1px solid var(--ant-color-border-secondary, #f0f0f0);
}
.cmd-modal :deep(.ant-modal-body) {
  padding: 0 !important;
  max-height: 400px;
  overflow-y: auto;
}
.cmd-header {
  display: flex;
  align-items: center;
  gap: 12px;
  padding: 16px 20px;
}
.cmd-input {
  flex: 1;
  border: none;
  outline: none;
  font-size: 16px;
  background: transparent;
  color: var(--ant-color-text, #111827);
}
.cmd-input::placeholder {
  color: var(--ant-color-text-tertiary, #9ca3af);
}
.cmd-body {
  min-height: 120px;
}
.cmd-empty {
  display: flex;
  flex-direction: column;
  align-items: center;
  justify-content: center;
  padding: 40px 0;
  gap: 12px;
  color: var(--ant-color-text-quaternary, #d1d5db);
}
.cmd-category {
  padding: 4px 0;
}
.cmd-cat-label {
  padding: 8px 20px 4px;
  font-size: 11px;
  font-weight: 600;
  color: var(--ant-color-text-tertiary, #9ca3af);
  text-transform: uppercase;
  letter-spacing: 0.5px;
}
.cmd-item {
  display: flex;
  align-items: center;
  gap: 12px;
  padding: 10px 20px;
  cursor: pointer;
  transition: background 0.15s;
}
.cmd-item:hover,
.cmd-item-active {
  background: var(--ant-color-primary-bg, #ebf0ff);
}
.cmd-item-icon {
  font-size: 18px;
  color: var(--ant-color-text-secondary, #6b7280);
  width: 24px;
  text-align: center;
}
.cmd-item-text {
  flex: 1;
}
.cmd-item-label {
  font-size: 14px;
  font-weight: 500;
  color: var(--ant-color-text, #111827);
}
.cmd-item-desc {
  font-size: 12px;
  color: var(--ant-color-text-tertiary, #9ca3af);
}
.cmd-item-shortcut {
  padding: 2px 8px;
  background: var(--ant-color-fill-secondary, #f5f5f5);
  border-radius: 4px;
  font-size: 11px;
  font-family: monospace;
  color: var(--ant-color-text-secondary, #6b7280);
}
.cmd-item-arrow {
  font-size: 12px;
  color: var(--ant-color-text-quaternary, #d1d5db);
}
.cmd-footer {
  display: flex;
  gap: 16px;
  padding: 10px 20px;
  border-top: 1px solid var(--ant-color-border-secondary, #f0f0f0);
  font-size: 11px;
  color: var(--ant-color-text-tertiary, #9ca3af);
}
.cmd-footer kbd {
  padding: 1px 5px;
  background: var(--ant-color-fill-secondary, #f5f5f5);
  border: 1px solid var(--ant-color-border, #e5e7eb);
  border-radius: 4px;
  font-family: monospace;
  font-size: 11px;
}

/* ═══ Notification Drawer ═══ */
.notif-list {
  padding: 0 16px 16px;
}
.notif-item {
  display: flex;
  gap: 12px;
  padding: 14px 12px;
  border-radius: 8px;
  cursor: pointer;
  transition: background 0.2s;
  position: relative;
  border-bottom: 1px solid var(--ant-color-border-secondary, #f3f4f6);
}
.notif-item:hover {
  background: var(--ant-color-bg-text-hover, #fafafa);
}
.notif-unread {
  background: var(--ant-color-primary-bg, #ebf0ff);
}
.notif-icon {
  flex-shrink: 0;
  padding-top: 2px;
}
.notif-content {
  flex: 1;
  min-width: 0;
}
.notif-title {
  font-size: 13px;
  color: var(--ant-color-text, #111827);
  margin-bottom: 4px;
  line-height: 1.4;
}
.notif-desc {
  font-size: 12px;
  color: var(--ant-color-text-secondary, #6b7280);
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}
.notif-time {
  font-size: 11px;
  color: var(--ant-color-text-tertiary, #9ca3af);
  margin-top: 4px;
}
.notif-dot {
  position: absolute;
  top: 18px;
  right: 12px;
  width: 8px;
  height: 8px;
  background: var(--ant-color-primary, #2962FF);
  border-radius: 50%;
}
</style>

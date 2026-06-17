<template>
  <n-config-provider :theme="theme">
    <n-layout position="absolute" style="height: 100vh">
      <n-layout-header bordered style="height: 48px; padding: 0 20px; display: flex; align-items: center; justify-content: space-between; background: #2c3e50;">
        <div style="display: flex; align-items: center; gap: 12px;">
          <div style="display: flex; align-items: center; gap: 8px;">
            <img
              src="/brand/lingmirror-icon.png"
              alt="凌镜"
              style="width: 24px; height: 24px; border-radius: 6px;"
            />
            <h2 style="margin: 0; color: #fff; font-size: 16px; letter-spacing: 1px;">凌镜</h2>
          </div>
        <n-auto-complete
          v-if="canSearch"
          v-model:value="searchQuery"
          :options="searchOptions"
          :input-props="{ placeholder: '搜索商品/SKU/供应商... (/)', style: 'width: 280px;' }"
          @select="handleSearchSelect"
          @keyup.enter="handleSearch"
          @update:value="onSearchInput"
          clearable
          size="small"
        />
      </div>
      <div style="display: flex; align-items: center; gap: 8px;">
        <n-popover trigger="click" placement="bottom-end" :width="360" @update:show="onNotifPopover">
          <template #trigger>
            <n-badge :value="unreadCount" :max="99" dot>
              <n-button size="tiny" quaternary style="color: rgba(255,255,255,0.7); position:relative;">
                <template #icon><n-icon><svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 512 512"><path d="M427.68 351.43C402 320 383.87 304 383.87 217.35c0-76.48-47.31-137.58-130.36-154.36V32H258.5v31c-83.05 16.78-130.36 77.88-130.36 154.36 0 86.66-18.14 102.62-43.81 134.08C77 365 72 370.09 77 376.38S88 384 96 384h320c8 0 14-2.53 19-7.62s0-11.38-7.31-24.95zM368 416H144a16 16 0 0 0 0 32h224a16 16 0 0 0 0-32z" fill="currentColor"/></svg></n-icon></template>
              </n-button>
            </n-badge>
          </template>
          <n-list v-if="recentNotifs.length" style="max-height:300px;overflow-y:auto;">
            <n-list-item v-for="n in recentNotifs" :key="n.id" clickable @click="goNotif(n)">
              <template #prefix>
                <n-tag size="tiny" :type="sevColor(n.severity)" round>{{ n.alert_type.split('_')[0] }}</n-tag>
              </template>
              <span :style="{fontWeight: n.is_read ? 'normal' : 'bold', fontSize:'13px'}">{{ n.title }}</span>
              <template #suffix><span style="font-size:11px;color:#999;white-space:nowrap;">{{ timeAgo(n.created_at) }}</span></template>
            </n-list-item>
          </n-list>
          <n-empty v-else description="暂无未读通知" style="padding:20px;" />
          <n-space justify="space-between" style="border-top:1px solid #eee;padding:8px 12px;margin-top:4px;">
            <n-button size="tiny" @click="markAllRead">全部已读</n-button>
            <n-button size="tiny" @click="router.push('/notifications')">查看全部</n-button>
          </n-space>
        </n-popover>
        <span style="color: rgba(255,255,255,0.7); font-size: 13px;">{{ userDisplayName }}</span>
        <n-button size="tiny" quaternary style="color: rgba(255,255,255,0.5);" @click="handleLogout">退出</n-button>
        <n-button size="tiny" quaternary style="color: rgba(255,255,255,0.5);" @click="toggleTheme">{{ themeIcon }}</n-button>
      </div>
    </n-layout-header>
    <n-layout has-sider position="absolute" style="top: 48px; bottom: 0;">
      <n-layout-sider bordered width="220" content-style="padding: 0;" :native-scrollbar="false">
        <n-menu :value="activeKey" :options="menuOptions" @update:value="handleMenuSelect" />
      </n-layout-sider>
      <n-layout content-style="padding: 20px; overflow-y: auto; background: #f5f7fa;">
        <router-view />
      </n-layout>
    </n-layout>
    </n-layout>
  </n-config-provider>
</template>

<script setup lang="ts">
import { computed, h, ref, onMounted, onUnmounted } from 'vue'
import { useRouter, useRoute } from 'vue-router'
import { NIcon, NLayout, NLayoutHeader, NLayoutSider, NMenu, NConfigProvider, NBadge, NPopover, NList, NListItem, NSpace, useMessage, darkTheme } from 'naive-ui'
import type { Component } from 'vue'
import http from '@/api/http'

// ========== 图标映射表 ==========
// 新模块在路由 meta.icon 中指定图标 key，Layout 会自动渲染
// 可用图标见下方 map。如需新增图标，在这里加一行 import + map 即可。
import {
  ListOutline,
  LayersOutline,
  ColorPaletteOutline,
  CashOutline,
  ArchiveOutline,
  PeopleOutline,
  PricetagOutline,
  HomeOutline,
  DocumentTextOutline,
  WarningOutline,
  GlobeOutline,
  CartOutline,
  BarChartOutline,
  CubeOutline,
  SettingsOutline,
  TrendingUpOutline,
  ShieldCheckmarkOutline,
  DownloadOutline,
  AnalyticsOutline,
  BusinessOutline,
  NotificationsOutline,
  CheckmarkCircleOutline,
} from '@vicons/ionicons5'

const iconMap: Record<string, Component> = {
  home: HomeOutline,
  list: ListOutline,
  layers: LayersOutline,
  palette: ColorPaletteOutline,
  cash: CashOutline,
  archive: ArchiveOutline,
  people: PeopleOutline,
  tag: PricetagOutline,
  'doc-text': DocumentTextOutline,
  warning: WarningOutline,
  globe: GlobeOutline,
  cart: CartOutline,
  chart: BarChartOutline,
  cube: CubeOutline,
  settings: SettingsOutline,
  trend: TrendingUpOutline,
  shield: ShieldCheckmarkOutline,
  download: DownloadOutline,
  'checkmark-circle': CheckmarkCircleOutline,
  warehouse: BusinessOutline,
  analytics: AnalyticsOutline,
  notification: NotificationsOutline,
}

function renderIcon(iconName: string) {
  const IconComp = iconMap[iconName]
  if (!IconComp) return undefined
  return () => h(NIcon, null, { default: () => h(IconComp) })
}

const router = useRouter()
const route = useRoute()

const activeKey = computed(() => route.path)

const message = useMessage()
const searchQuery = ref('')
const searchOptions = ref<any[]>([])
let searchTimer: any = null

// ── 通知 ──
const unreadCount = ref(0)
const recentNotifs = ref<any[]>([])
let notifTimer: any = null

function sevColor(sev: string): string {
  return { info: 'info', warning: 'warning', error: 'error', critical: 'error' }[sev] || 'info'
}

function timeAgo(t: string): string {
  if (!t) return ''
  const s = Math.floor((Date.now() - new Date(t).getTime()) / 1000)
  if (s < 60) return '刚刚'
  if (s < 3600) return `${Math.floor(s / 60)}分钟前`
  if (s < 86400) return `${Math.floor(s / 3600)}小时前`
  return `${Math.floor(s / 86400)}天前`
}

async function fetchNotifs() {
  try {
    const res: any = await http.get('/notifications?unread_only=true&page_size=5')
    const body = res.data
    recentNotifs.value = body?.data?.records ?? body?.records ?? []
    const countRes: any = await http.get('/notifications/unread-count')
    unreadCount.value = countRes.data?.data?.total ?? countRes.data?.total ?? 0
  } catch {}
}

function onNotifPopover(show: boolean) {
  if (show) fetchNotifs()
}

async function markAllRead() {
  try {
    await http.put('/notifications/read-all')
    unreadCount.value = 0
    recentNotifs.value = []
    message.success('已全部标记已读')
  } catch {}
}

function goNotif(n: any) {
  if (!n.is_read) {
    http.put(`/notifications/${n.id}/read`).catch(() => {})
    unreadCount.value = Math.max(0, unreadCount.value - 1)
  }
  if (n.link_url) router.push(n.link_url)
}

// 暗黑模式
const isDark = ref(localStorage.getItem('darkMode') === 'true')
const theme = computed(() => isDark.value ? darkTheme : null)
const themeIcon = computed(() => isDark.value ? '☀️' : '🌙')
function toggleTheme() {
  isDark.value = !isDark.value
  localStorage.setItem('darkMode', String(isDark.value))
}

// 用户信息
const user = ref<any>(JSON.parse(localStorage.getItem('user') || '{}'))
const userPermissions = ref<string[]>(user.value?.permissions || [])
const userDisplayName = computed(() => user.value?.display_name || user.value?.username || '未登录')
const canSearch = computed(() => user.value?.role === 'admin' || userPermissions.value.includes('search:view'))

function handleLogout() {
  localStorage.removeItem('token')
  localStorage.removeItem('user')
  window.location.href = '/login'
}

// 从 /auth/me 加载当前用户信息和权限
async function loadUserPermissions() {
  const token = localStorage.getItem('token')
  if (!token) return
  try {
    const res: any = await http.get('/auth/me')
    if (res.code === 200) {
      const me = res.data
      user.value = me
      localStorage.setItem('user', JSON.stringify(me))
      userPermissions.value = me.permissions || []
    }
  } catch {
    // ignore
  }
}

async function doSearch(q: string) {
  if (!canSearch.value) {
    searchOptions.value = []
    return
  }
  if (!q || q.length < 1) {
    searchOptions.value = []
    return
  }
  try {
    const res: any = await http.get(`/search?q=${encodeURIComponent(q)}&limit=5`)
    if (res.code === 200 && res.data) {
      const opts: any[] = []
      for (const p of res.data.products || []) {
        opts.push({ label: `📦 ${p.name}`, value: `product:${p.id}` })
      }
      for (const s of res.data.skus || []) {
        opts.push({ label: `🏷️ ${s.code || s.spec_desc}`, value: `sku:${s.product_id}` })
      }
      for (const s of res.data.suppliers || []) {
        opts.push({ label: `🤝 ${s.name}`, value: `supplier:${s.id}` })
      }
      searchOptions.value = opts
    }
  } catch {
    searchOptions.value = []
  }
}

function handleSearch() {
  if (searchQuery.value) router.push(`/products?name=${encodeURIComponent(searchQuery.value)}`)
  searchQuery.value = ''
  searchOptions.value = []
}

function handleSearchSelect(value: string) {
  const parts = value.split(':')
  if (parts[0] === 'product' || parts[0] === 'sku') {
    router.push(`/products/${parts[1]}`)
  } else if (parts[0] === 'supplier') {
    router.push(`/suppliers`)
  }
  searchQuery.value = ''
  searchOptions.value = []
}

function onSearchInput(val: string) {
  if (searchTimer) clearTimeout(searchTimer)
  searchTimer = setTimeout(() => doSearch(val), 300)
}

function handleKeydown(e: KeyboardEvent) {
  if (e.key === '/' && !['INPUT', 'TEXTAREA'].includes((e.target as HTMLElement)?.tagName || '')) {
    e.preventDefault()
    const input = document.querySelector('input[placeholder*="搜索"]') as HTMLInputElement
    input?.focus()
  }
}

onMounted(() => {
  document.addEventListener('keydown', handleKeydown)
  loadUserPermissions()
  fetchNotifs()
  notifTimer = setInterval(fetchNotifs, 30000)
})
onUnmounted(() => {
  document.removeEventListener('keydown', handleKeydown)
  if (notifTimer) clearInterval(notifTimer)
})

// ========== 从路由自动生成侧边菜单（带权限过滤） ==========
// 每个路由 meta 可设置:
//   menu: true              显示在侧边栏
//   icon: 'xxx'             图标 key
//   perm: 'module:action'   需要的权限码（可选）
// admin 用户看到所有菜单；普通用户只看到有权限的菜单
const menuOptions = computed(() => {
  const items: any[] = []
  const flatRoutes = router.getRoutes()
  const seen = new Set<string>()
  const perms = userPermissions.value
  const isAdmin = user.value?.role === 'admin'

  for (const r of flatRoutes) {
    const meta = r.meta as Record<string, any> | undefined
    if (meta?.menu === true && meta?.title && meta?.icon && !seen.has(r.path)) {
      seen.add(r.path)
      // 权限过滤：admin 或无 perm 要求的菜单始终显示
      if (!isAdmin && meta.perm && !perms.includes(meta.perm)) {
        continue
      }
      items.push({
        label: meta.title,
        key: r.path,
        icon: renderIcon(meta.icon as string),
      })
    }
  }
  return items
})

function handleMenuSelect(key: string) {
  router.push(key)
}
</script>

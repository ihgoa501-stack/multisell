<template>
  <div>
    <n-page-header subtitle="系统数据总览">
      <template #title>📊 首页仪表盘</template>
    </n-page-header>

    <!-- 统计卡片 -->
    <n-grid :cols="4" :x-gap="12" style="margin-top: 12px;">
      <n-grid-item>
        <n-card :bordered="true" hoverable>
          <n-statistic label="商品总数" :value="stats.products?.total || 0">
            <template #suffix>个</template>
          </n-statistic>
          <div style="margin-top: 8px; font-size: 13px; color: #888;">
            <n-tag size="tiny" type="success" style="margin-right: 4px;">上架 {{ stats.products?.on_shelf || 0 }}</n-tag>
            <n-tag size="tiny" type="warning">草稿 {{ stats.products?.draft || 0 }}</n-tag>
          </div>
        </n-card>
      </n-grid-item>
      <n-grid-item>
        <n-card :bordered="true" hoverable>
          <n-statistic label="SKU 总数" :value="stats.skus?.total || 0">
            <template #suffix>个</template>
          </n-statistic>
        </n-card>
      </n-grid-item>
      <n-grid-item>
        <n-card :bordered="true" hoverable>
          <n-statistic label="库存预警" :value="stats.inventory?.low_stock || 0" :style="lowStockStyle">
            <template #suffix>个SKU</template>
          </n-statistic>
        </n-card>
      </n-grid-item>
      <n-grid-item>
        <n-card :bordered="true" hoverable>
          <n-statistic label="品牌 / 供应商" :value="(stats.brands?.total || 0) + (stats.suppliers?.total || 0)">
            <template #suffix>
              <span style="font-size: 13px;">品牌 {{ stats.brands?.total || 0 }} / 供应商 {{ stats.suppliers?.total || 0 }}</span>
            </template>
          </n-statistic>
        </n-card>
      </n-grid-item>
    </n-grid>

    <n-grid :cols="2" :x-gap="12" style="margin-top: 12px;">
      <!-- 快捷操作 -->
      <n-grid-item>
        <n-card title="快捷操作" :bordered="true">
          <n-space>
            <n-button type="primary" @click="router.push('/products/create')">＋ 新增商品</n-button>
            <n-button @click="router.push('/products')">📋 商品列表</n-button>
            <n-button @click="router.push('/platforms')">🌐 平台管理</n-button>
            <n-button @click="router.push('/listings')">📤 发布管理</n-button>
          </n-space>
        </n-card>
      </n-grid-item>

      <!-- 平台发布概况 -->
      <n-grid-item>
        <n-card title="平台发布概况" :bordered="true">
          <n-empty v-if="!stats.platforms?.detail?.length" description="暂无平台发布" />
          <n-grid v-else :cols="stats.platforms?.detail?.length || 1" :x-gap="8">
            <n-grid-item v-for="p in stats.platforms?.detail || []" :key="p.code">
              <n-statistic :label="p.name" :value="p.count">
                <template #suffix>个商品</template>
              </n-statistic>
            </n-grid-item>
          </n-grid>
        </n-card>
      </n-grid-item>
    </n-grid>

    <n-grid :cols="2" :x-gap="12" style="margin-top: 12px;">
      <!-- 近期操作 -->
      <n-grid-item>
        <n-card title="近期操作" :bordered="true">
          <n-list v-if="stats.recent_logs?.items?.length" style="max-height: 240px; overflow-y: auto;">
            <n-list-item v-for="log in stats.recent_logs.items" :key="log.id">
              <template #prefix>
                <n-tag size="tiny" :type="log.action === '创建' ? 'success' : log.action === '删除' ? 'error' : 'info'">
                  {{ log.action }}
                </n-tag>
              </template>
              <span style="font-size: 13px;">{{ log.content || log.module }}</span>
              <template #suffix>
                <span style="font-size: 12px; color: #999;">{{ log.operator || '系统' }}</span>
              </template>
            </n-list-item>
          </n-list>
          <n-empty v-else description="暂无操作记录" />
        </n-card>
      </n-grid-item>

      <!-- 近期发布动态 -->
      <n-grid-item>
        <n-card title="近期发布动态" :bordered="true">
          <n-list v-if="stats.recent_listings?.length" style="max-height: 240px; overflow-y: auto;">
            <n-list-item v-for="item in stats.recent_listings" :key="item.product_name + item.platform_name">
              <template #prefix>
                <n-tag size="tiny" :color="{ color: item.platform_code === 'ozon' ? '#005bff' : '#ee4d2d' }">
                  {{ item.platform_name }}
                </n-tag>
              </template>
              <span style="font-size: 13px;">{{ item.product_name }}</span>
              <template #suffix>
                <n-tag size="tiny" :type="item.status === 'synced' ? 'success' : 'warning'">
                  {{ item.status === 'synced' ? '已发布' : item.status }}
                </n-tag>
              </template>
            </n-list-item>
          </n-list>
          <n-empty v-else description="暂无发布记录" />
        </n-card>
      </n-grid-item>
    </n-grid>
  </div>
</template>

<script setup lang="ts">
import { ref, computed, onMounted } from 'vue'
import { useRouter } from 'vue-router'
import { useMessage } from 'naive-ui'
import http from '@/api/http'

const router = useRouter()
const message = useMessage()
const stats = ref<any>({})

const lowStockStyle = computed(() => ({
  color: (stats.value?.inventory?.low_stock || 0) > 0 ? '#d03050' : undefined,
}))

async function fetchStats() {
  try {
    const res: any = await http.get('/dashboard/stats')
    stats.value = res.data || {}
  } catch (e: any) {
    message.error('加载统计数据失败')
  }
}

onMounted(fetchStats)
</script>

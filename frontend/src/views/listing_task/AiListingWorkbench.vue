<template>
  <div class="max-w-4xl mx-auto">
    <div class="flex items-center justify-between mb-6">
      <div>
        <h1 class="text-xl font-bold text-[var(--text-primary)]">铺货到平台</h1>
        <p class="text-sm text-[var(--text-tertiary)] mt-0.5">选择商品和目标平台，填写平台专属信息后发布</p>
      </div>
    </div>

    <!-- 选择商品 + 平台 -->
    <div class="bg-white border border-[var(--border-light)] rounded-lg p-4 mb-4 space-y-3">
      <div>
        <div class="text-[11px] font-semibold text-[var(--text-tertiary)] uppercase tracking-wide mb-2">商品</div>
        <n-select
          v-model:value="selectedProductId"
          :options="productOptions"
          placeholder="搜索并选择商品"
          filterable
          clearable
          size="small"
        />
      </div>
      <div>
        <div class="text-[11px] font-semibold text-[var(--text-tertiary)] uppercase tracking-wide mb-2">目标平台</div>
        <n-select
          v-model:value="selectedPlatformId"
          :options="platformOptions"
          placeholder="选择目标平台"
          filterable
          size="small"
        />
      </div>
    </div>

    <!-- 表单 -->
    <div v-if="!selectedProductId || !selectedPlatformId" class="bg-white border border-[var(--border-light)] rounded-lg p-12 text-center">
      <div class="text-4xl mb-3 opacity-30">📋</div>
      <div class="text-base font-medium text-[var(--text-secondary)] mb-1">请先选择商品和平台</div>
      <div class="text-sm text-[var(--text-tertiary)]">选择后即可编辑发布内容</div>
    </div>

    <div v-else class="space-y-4">
      <!-- 商品信息（只读） -->
      <div class="bg-white border border-[var(--border-light)] rounded-lg p-4">
        <div class="text-[11px] font-semibold text-[var(--text-tertiary)] uppercase tracking-wide mb-3">商品信息</div>
        <div class="flex gap-4">
          <div class="w-16 h-16 rounded-[6px] bg-[var(--bg-subtle)] flex items-center justify-center text-2xl shrink-0">{{ currentProduct?.emoji || '📦' }}</div>
          <div class="flex-1 min-w-0">
            <div class="text-[14px] font-medium text-[var(--text-primary)]">{{ currentProduct?.name }}</div>
            <div class="text-[12px] text-[var(--text-tertiary)] mt-0.5">SKU: {{ currentProduct?.skuCount }} 个 · 成本 ¥{{ currentProduct?.minPrice }}</div>
          </div>
        </div>
      </div>

      <!-- 平台表单 -->
      <div class="bg-white border border-[var(--border-light)] rounded-lg overflow-hidden">
        <div class="px-4 py-3 border-b border-[var(--border-light)] flex items-center justify-between">
          <span class="text-[11px] font-semibold text-[var(--text-tertiary)] uppercase tracking-wide">{{ currentPlatform?.name }} 发布信息</span>
          <n-button size="tiny" secondary :loading="generating" @click="handleGenerateAI">AI 帮我生成</n-button>
        </div>

        <div class="p-4 space-y-4">
          <!-- 公共字段：标题 -->
          <div>
            <div class="flex items-center justify-between mb-1">
              <label class="text-[13px] font-medium text-[var(--text-primary)]">标题 <span class="text-red-500">*</span></label>
              <span class="text-[11px] text-[var(--text-tertiary)]">{{ form.title.length }}/200</span>
            </div>
            <n-input v-model:value="form.title" placeholder="输入平台商品标题" :maxlength="200" size="small" />
          </div>

          <!-- 公共字段：描述 -->
          <div>
            <div class="flex items-center justify-between mb-1">
              <label class="text-[13px] font-medium text-[var(--text-primary)]">描述 <span class="text-red-500">*</span></label>
            </div>
            <n-input v-model:value="form.description" type="textarea" placeholder="输入平台商品描述" :rows="4" size="small" />
          </div>

          <!-- 平台专属字段 -->
          <template v-for="field in platformExtraFields" :key="field.key">
            <!-- 多行文本输入 -->
            <div v-if="field.type === 'textarea'" class="grid grid-cols-2 gap-4">
              <div v-for="(_, idx) in field.count" :key="idx">
                <div class="flex items-center justify-between mb-1">
                  <label class="text-[13px] font-medium text-[var(--text-primary)]">{{ field.label }} {{ idx + 1 }}</label>
                  <span class="text-[11px] text-[var(--text-tertiary)]">{{ (form.extra[field.key] || [])[idx]?.length || 0 }}/500</span>
                </div>
                <n-input
                  v-model:value="(form.extra[field.key] || [])[idx]"
                  type="textarea"
                  :placeholder="field.placeholder || `${field.label} ${idx + 1}`"
                  :rows="2"
                  size="small"
                  @update:value="ensureExtraArray(field.key, field.count || 1)"
                />
              </div>
            </div>

            <!-- 单行文本 -->
            <div v-else-if="field.type === 'text'">
              <div class="flex items-center justify-between mb-1">
                <label class="text-[13px] font-medium text-[var(--text-primary)]">{{ field.label }} <span v-if="field.required" class="text-red-500">*</span></label>
              </div>
              <n-input v-model:value="form.extra[field.key]" :placeholder="field.placeholder || `输入${field.label}`" size="small" />
            </div>

            <!-- 数字 -->
            <div v-else-if="field.type === 'number'" class="grid grid-cols-2 gap-4">
              <div v-for="(_, idx) in field.count || 1" :key="idx">
                <div class="flex items-center justify-between mb-1">
                  <label class="text-[13px] font-medium text-[var(--text-primary)]">{{ field.count && field.count > 1 ? `${field.label} ${idx + 1}` : field.label }} <span v-if="field.required" class="text-red-500">*</span></label>
                </div>
                <n-input-number
                  :value="field.count ? (form.extra[field.key] || [])[idx] : form.extra[field.key]"
                  @update:value="updateNum(field.key, field.count ? idx : null, $event)"
                  :placeholder="field.placeholder || '0'"
                  :min="0"
                  size="small"
                  class="w-full"
                />
              </div>
            </div>

            <!-- 下拉选择 -->
            <div v-else-if="field.type === 'select'">
              <div class="flex items-center justify-between mb-1">
                <label class="text-[13px] font-medium text-[var(--text-primary)]">{{ field.label }} <span v-if="field.required" class="text-red-500">*</span></label>
              </div>
              <n-select
                v-model:value="form.extra[field.key]"
                :options="field.options || []"
                :placeholder="field.placeholder || `选择${field.label}`"
                size="small"
              />
            </div>

            <!-- 标签 -->
            <div v-else-if="field.type === 'tags'">
              <div class="flex items-center justify-between mb-1">
                <label class="text-[13px] font-medium text-[var(--text-primary)]">{{ field.label }}</label>
              </div>
              <div class="flex flex-wrap gap-1 mb-2">
                <span
                  v-for="(tag, ti) in (form.extra[field.key] as string[])" :key="ti"
                  class="inline-flex items-center gap-1 text-[12px] px-2 py-0.5 rounded bg-[var(--accent-bg)] text-[var(--accent)]"
                >{{ tag }}<span class="cursor-pointer hover:text-red-500" @click="(form.extra[field.key] as string[]).splice(ti, 1)">×</span></span>
              </div>
              <div class="flex gap-1">
                <n-input v-model:value="newTag" :placeholder="field.placeholder || '添加'" size="small" @keyup.enter="addTag(field.key)" />
                <n-button size="small" secondary @click="addTag(field.key)">添加</n-button>
              </div>
            </div>

            <!-- 键值对 -->
            <div v-else-if="field.type === 'keyvalue'">
              <div class="flex items-center justify-between mb-1">
                <label class="text-[13px] font-medium text-[var(--text-primary)]">{{ field.label }}</label>
              </div>
              <div class="space-y-1.5">
                <div v-for="(kv, ki) in getKVArr(field.key)" :key="ki" class="flex gap-1.5 items-center">
                  <n-input v-model:value="kv.k" placeholder="属性名" size="small" style="width:120px" />
                  <n-input v-model:value="kv.v" placeholder="属性值" size="small" class="flex-1" />
                  <n-button size="tiny" quaternary @click="getAnyArr(field.key).splice(ki, 1)">×</n-button>
                </div>
                <n-button size="tiny" secondary @click="getAnyArr(field.key).push({k:'',v:''})">+ 添加</n-button>
              </div>
            </div>
          </template>

          <!-- 公共字段：价格 -->
          <div class="grid grid-cols-2 gap-4">
            <div>
              <div class="flex items-center justify-between mb-1">
                <label class="text-[13px] font-medium text-[var(--text-primary)]">售价 ({{ currentPlatform?.currency || '¥' }}) <span class="text-red-500">*</span></label>
              </div>
              <n-input-number v-model:value="form.salePrice" placeholder="0.00" :min="0" :precision="2" size="small" class="w-full" />
            </div>
            <div>
              <div class="flex items-center justify-between mb-1">
                <label class="text-[13px] font-medium text-[var(--text-primary)]">原价 ({{ currentPlatform?.currency || '¥' }})</label>
              </div>
              <n-input-number v-model:value="form.originalPrice" placeholder="0.00" :min="0" :precision="2" size="small" class="w-full" />
            </div>
          </div>

          <!-- 公共字段：类目 -->
          <div>
            <div class="flex items-center justify-between mb-1">
              <label class="text-[13px] font-medium text-[var(--text-primary)]">平台类目 <span class="text-red-500">*</span></label>
            </div>
            <n-select v-model:value="form.category" :options="categoryOptions" placeholder="选择平台类目" size="small" />
          </div>

          <!-- 公共字段：SEO关键词 -->
          <div>
            <div class="flex items-center justify-between mb-1">
              <label class="text-[13px] font-medium text-[var(--text-primary)]">SEO 关键词</label>
            </div>
            <div class="flex flex-wrap gap-1 mb-2">
              <span v-for="(kw, ki) in form.keywords" :key="ki" class="inline-flex items-center gap-1 text-[12px] px-2 py-0.5 rounded bg-[var(--accent-bg)] text-[var(--accent)]">
                {{ kw }}<span class="cursor-pointer hover:text-red-500" @click="form.keywords.splice(ki, 1)">×</span>
              </span>
            </div>
            <div class="flex gap-1">
              <n-input v-model:value="newKeyword" placeholder="添加关键词" size="small" @keyup.enter="addKeyword" />
              <n-button size="small" secondary @click="addKeyword">添加</n-button>
            </div>
          </div>
        </div>
      </div>

      <!-- 操作栏 -->
      <div class="bg-white border border-[var(--border-light)] rounded-lg p-4 flex items-center justify-between">
        <n-button size="small" @click="handleSaveDraft">存草稿</n-button>
        <n-button size="small" type="primary" :disabled="!formValid" @click="handlePublish">发布到 {{ currentPlatform?.name }}</n-button>
      </div>
    </div>
  </div>
</template>

<script setup lang="ts">
import { ref, computed, watch } from 'vue'
import { useMessage } from 'naive-ui'

const message = useMessage()

// ========== 平台定义 ==========
const platforms = [
  { id: 1, name: 'Amazon', code: 'amazon', currency: '$', region: '全球' },
  { id: 2, name: 'eBay', code: 'ebay', currency: '$', region: '全球' },
  { id: 3, name: 'AliExpress', code: 'aliexpress', currency: '$', region: '全球' },
  { id: 4, name: 'Temu', code: 'temu', currency: '$', region: '全球' },
  { id: 5, name: 'TikTok Shop', code: 'tiktok', currency: '$', region: '全球' },
  { id: 6, name: 'SHEIN', code: 'shein', currency: '$', region: '全球' },
  { id: 7, name: 'Ozon', code: 'ozon', currency: '₽', region: '俄罗斯' },
  { id: 8, name: 'Wildberries', code: 'wb', currency: '₽', region: '俄罗斯' },
  { id: 9, name: 'Yandex Market', code: 'yandex', currency: '₽', region: '俄罗斯' },
  { id: 10, name: 'Shopee', code: 'shopee', currency: '$', region: '东南亚' },
  { id: 11, name: 'Lazada', code: 'lazada', currency: '$', region: '东南亚' },
  { id: 12, name: 'Mercado Libre', code: 'mercadolibre', currency: '$', region: '拉美' },
  { id: 13, name: 'Walmart', code: 'walmart', currency: '$', region: '美国' },
  { id: 14, name: 'Allegro', code: 'allegro', currency: '€', region: '欧洲' },
]

// ========== 平台专属字段定义 ==========
interface ExtraField {
  key: string
  label: string
  type: 'text' | 'textarea' | 'number' | 'select' | 'tags' | 'keyvalue'
  required?: boolean
  count?: number
  placeholder?: string
  options?: { label: string, value: string }[]
}

const platformFieldMap: Record<string, ExtraField[]> = {
  amazon: [
    { key: 'bulletPoints', label: 'Bullet Point', type: 'textarea', count: 5, placeholder: '描述产品核心卖点' },
    { key: 'searchTerms', label: 'Search Terms', type: 'tags', placeholder: '添加后端搜索词' },
    { key: 'brand', label: 'Brand', type: 'text', required: true, placeholder: '品牌名称' },
    { key: 'gtin', label: 'UPC/EAN', type: 'text', placeholder: '全球贸易项目代码' },
    { key: 'itemType', label: 'Item Type', type: 'text', placeholder: '商品类型关键词' },
  ],
  ebay: [
    { key: 'condition', label: 'Condition', type: 'select', required: true, options: [
      { label: 'New', value: 'new' },
      { label: 'New without box', value: 'new_no_box' },
      { label: 'Used - Like New', value: 'used_like_new' },
      { label: 'Used - Good', value: 'used_good' },
    ]},
    { key: 'brand', label: 'Brand', type: 'text', placeholder: '品牌名称' },
    { key: 'mpn', label: 'MPN', type: 'text', placeholder: '制造商零件编号' },
    { key: 'listingType', label: 'Listing Type', type: 'select', options: [
      { label: 'Buy It Now', value: 'bin' },
      { label: 'Auction', value: 'auction' },
    ]},
  ],
  aliexpress: [
    { key: 'brand', label: 'Brand', type: 'text', placeholder: '品牌名称' },
    { key: 'packageWeight', label: 'Package Weight (g)', type: 'number', required: true },
    { key: 'packageSize', label: 'Package Size (cm)', type: 'number', count: 3 },
  ],
  temu: [
    { key: 'brand', label: 'Brand', type: 'text', placeholder: '品牌名称' },
    { key: 'packageWeight', label: 'Package Weight (g)', type: 'number', required: true },
    { key: 'packageSize', label: 'Package Size (cm)', type: 'number', count: 3 },
  ],
  tiktok: [
    { key: 'brand', label: 'Brand', type: 'text', required: true, placeholder: '品牌名称' },
    { key: 'video', label: 'Product Video URL', type: 'text', placeholder: '商品视频链接' },
    { key: 'warehouse', label: 'Warehouse', type: 'select', options: [
      { label: 'China Warehouse', value: 'cn' },
      { label: 'Local Warehouse', value: 'local' },
    ]},
  ],
  shein: [
    { key: 'brand', label: 'Brand', type: 'text', placeholder: '品牌名称' },
    { key: 'material', label: 'Material Composition', type: 'text', placeholder: '例如 100% Cotton' },
    { key: 'sizeChart', label: 'Size Chart URL', type: 'text', placeholder: '尺码表图片链接' },
  ],
  ozon: [
    { key: 'vat', label: 'VAT Rate', type: 'select', required: true, options: [
      { label: '0%', value: '0' },
      { label: '10%', value: '10' },
      { label: '20%', value: '20' },
    ]},
    { key: 'barcode', label: 'Баркод (Barcode)', type: 'text', placeholder: 'EAN-13 / UPC' },
    { key: 'attributes', label: 'Характеристики (Attributes)', type: 'keyvalue' },
  ],
  wb: [
    { key: 'characteristics', label: 'Характеристики (Characteristics)', type: 'keyvalue' },
    { key: 'brand', label: 'Бренд (Brand)', type: 'text', placeholder: '品牌名称' },
    { key: 'country', label: 'Страна производства', type: 'text', placeholder: '生产国' },
  ],
  yandex: [
    { key: 'manufacturer', label: 'Производитель', type: 'text', placeholder: '制造商' },
    { key: 'barcode', label: 'Штрихкод', type: 'text', placeholder: '条形码' },
    { key: 'warranty', label: 'Гарантия (дней)', type: 'number', placeholder: '保修天数' },
  ],
  shopee: [
    { key: 'brand', label: 'Brand', type: 'text', placeholder: '品牌名称' },
    { key: 'weight', label: 'Weight (kg)', type: 'number', required: true },
    { key: 'packageSize', label: 'Package Size (cm)', type: 'number', count: 3 },
    { key: 'daysToShip', label: 'Days to Ship', type: 'select', options: [
      { label: '1 Day', value: '1' },
      { label: '2 Days', value: '2' },
      { label: '3 Days', value: '3' },
      { label: '5 Days', value: '5' },
      { label: '7 Days', value: '7' },
    ]},
  ],
  lazada: [
    { key: 'brand', label: 'Brand', type: 'text', required: true, placeholder: '品牌名称' },
    { key: 'weight', label: 'Weight (kg)', type: 'number', required: true },
    { key: 'warranty', label: 'Warranty (months)', type: 'number', placeholder: '保修月数' },
  ],
  mercadolibre: [
    { key: 'condition', label: 'Condition', type: 'select', required: true, options: [
      { label: 'Nuevo', value: 'new' },
      { label: 'Usado', value: 'used' },
    ]},
    { key: 'brand', label: 'Marca (Brand)', type: 'text', required: true, placeholder: '品牌' },
    { key: 'attributes', label: 'Atributos', type: 'keyvalue' },
  ],
  walmart: [
    { key: 'brand', label: 'Brand', type: 'text', required: true, placeholder: '品牌名称' },
    { key: 'itemId', label: 'Item ID', type: 'text', placeholder: 'Walmart Item ID' },
    { key: 'shippingWeight', label: 'Shipping Weight (kg)', type: 'number', required: true },
  ],
  allegro: [
    { key: 'condition', label: 'Stan (Condition)', type: 'select', required: true, options: [
      { label: 'Nowy', value: 'new' },
      { label: 'Używany', value: 'used' },
    ]},
    { key: 'ean', label: 'EAN', type: 'text', placeholder: 'Europejski kod EAN' },
    { key: 'brand', label: 'Marka (Brand)', type: 'text', placeholder: '品牌' },
  ],
}

// ========== Mock 数据 ==========
const mockProducts = [
  { id: 1, name: '智能蓝牙耳机 Pro', emoji: '🎧', skuCount: 3, minPrice: 89 },
  { id: 2, name: '便携充电宝 20000mAh', emoji: '🔋', skuCount: 2, minPrice: 129 },
  { id: 3, name: '可折叠手机支架', emoji: '📱', skuCount: 4, minPrice: 25 },
  { id: 4, name: '无线鼠标 静音版', emoji: '🖱️', skuCount: 2, minPrice: 49 },
  { id: 5, name: 'USB-C 扩展坞 7合1', emoji: '🔌', skuCount: 2, minPrice: 159 },
  { id: 6, name: '桌面收纳盒 多功能', emoji: '📦', skuCount: 3, minPrice: 35 },
]

// ========== 类目映射 ==========
const categoryMap: Record<string, { label: string, value: string }[]> = {
  amazon: [
    { label: 'Electronics', value: 'electronics' },
    { label: 'Home & Kitchen', value: 'home_kitchen' },
    { label: 'Cell Phones & Accessories', value: 'phones' },
    { label: 'Computers & Accessories', value: 'computers' },
  ],
  ebay: [
    { label: 'Consumer Electronics', value: 'electronics' },
    { label: 'Home & Garden', value: 'home_garden' },
    { label: 'Clothing & Accessories', value: 'clothing' },
  ],
  aliexpress: [
    { label: 'Электроника', value: 'electronics' },
    { label: 'Дом и сад', value: 'home_garden' },
    { label: 'Одежда', value: 'clothing' },
  ],
  temu: [
    { label: 'Electronics', value: 'electronics' },
    { label: 'Home & Kitchen', value: 'home_kitchen' },
    { label: 'Fashion', value: 'fashion' },
  ],
  tiktok: [
    { label: 'Electronics', value: 'electronics' },
    { label: 'Fashion & Accessories', value: 'fashion' },
    { label: 'Beauty & Personal Care', value: 'beauty' },
  ],
  shein: [
    { label: 'Women Clothing', value: 'womens_clothing' },
    { label: 'Men Clothing', value: 'mens_clothing' },
    { label: 'Accessories', value: 'accessories' },
  ],
  ozon: [
    { label: 'Электроника', value: 'electronics' },
    { label: 'Телефоны и аксессуары', value: 'phones' },
    { label: 'Бытовая техника', value: 'appliances' },
  ],
  wb: [
    { label: 'Электроника', value: 'electronics' },
    { label: 'Аксессуары', value: 'accessories' },
    { label: 'Товары для дома', value: 'home' },
  ],
  yandex: [
    { label: 'Электроника', value: 'electronics' },
    { label: 'Бытовая техника', value: 'appliances' },
  ],
  shopee: [
    { label: 'Electronics', value: 'electronics' },
    { label: 'Home & Living', value: 'home' },
    { label: 'Fashion', value: 'fashion' },
    { label: 'Mobile & Gadgets', value: 'mobile' },
  ],
  lazada: [
    { label: 'Electronics', value: 'electronics' },
    { label: 'Home & Living', value: 'home' },
    { label: 'Fashion', value: 'fashion' },
  ],
  mercadolibre: [
    { label: 'Electrónica', value: 'electronics' },
    { label: 'Hogar y Muebles', value: 'home' },
    { label: 'Ropa y Accesorios', value: 'clothing' },
  ],
  walmart: [
    { label: 'Electronics', value: 'electronics' },
    { label: 'Home & Furniture', value: 'home' },
    { label: 'Health & Beauty', value: 'beauty' },
  ],
  allegro: [
    { label: 'Elektronika', value: 'electronics' },
    { label: 'Dom i Ogród', value: 'home_garden' },
    { label: 'Moda', value: 'fashion' },
  ],
}

// ========== 模板（AI生成用） ==========
const platformTemplates: Record<string, { title: (n: string) => string, desc: (n: string) => string, keywords: string[], price: number, extra: Record<string, any> }> = {
  amazon: {
    title: (n) => `${n}, Premium Quality, Fast Shipping from USA`,
    desc: (n) => `【${n}】Top quality, reliable performance.\n\n✅ Premium Materials\n✅ Certified Quality\n✅ Fast Free Shipping\n✅ Easy Returns\n\nOrder now with confidence!`,
    keywords: ['premium', 'quality', 'fast shipping'],
    price: 29,
    extra: {
      bulletPoints: ['Premium build quality for long-lasting use', 'Universal compatibility with all devices', 'Lightweight and portable design', 'Easy to use, no setup required', 'Backed by 12-month warranty'],
      searchTerms: ['premium', 'quality', 'best gift', 'new arrival'],
      brand: 'Generic',
      gtin: '',
      itemType: '',
    },
  },
  ebay: {
    title: (n) => `${n} - Brand New | Free Shipping | 30-Day Returns`,
    desc: (n) => `${n}\n\nCondition: Brand New\n✅ Fast Shipping\n✅ 30-Day Money Back\n✅ Top Rated Seller\n\nBuy with confidence!`,
    keywords: ['brand new', 'free shipping', 'best offer'],
    price: 27,
    extra: { condition: 'new', brand: 'Generic', mpn: '', listingType: 'bin' },
  },
  aliexpress: {
    title: (n) => `${n} Direct from Factory - Wholesale Price`,
    desc: (n) => `${n}\n\n✅ Factory Direct Price\n✅ Worldwide Free Shipping\n✅ 7-Day Returns\n✅ Order Tracking\n\nWholesale & Retail Welcome!`,
    keywords: ['factory price', 'free shipping', 'wholesale'],
    price: 22,
    extra: { brand: 'Generic', packageWeight: 200, packageSize: [15, 10, 5] },
  },
  temu: {
    title: (n) => `${n} Crazy Price | Fast Shipping`,
    desc: (n) => `🔥 UNBELIEVABLE DEAL 🔥 ${n}\n\n✅ Lowest Price Guarantee\n✅ Free Shipping\n✅ Easy Returns\n✅ 100% Quality Checked\n\nShop now - limited time offer!`,
    keywords: ['crazy deal', 'low price', 'clearance'],
    price: 12,
    extra: { brand: 'Generic', packageWeight: 200, packageSize: [15, 10, 5] },
  },
  tiktok: {
    title: (n) => `${n} | Must-Have! Shop Now ✨`,
    desc: (n) => `✨ Viral Find ✨ ${n}\n\n🔥 Everyone is talking about this!\n✅ Free Shipping\n✅ Limited Stock\n\n#fyp #musthave #viral`,
    keywords: ['viral', 'must have', 'trending', 'tiktok shop'],
    price: 25,
    extra: { brand: 'Generic', video: '', warehouse: 'cn' },
  },
  shein: {
    title: (n) => `${n} - Daily New Arrivals`,
    desc: (n) => `✨ ${n} ✨\n\nTrending now!\n✅ New Arrival\n✅ Free Shipping Over $29\n✅ Easy Returns\n\nShop the latest styles!`,
    keywords: ['new arrival', 'trending', 'fashion'],
    price: 18,
    extra: { brand: 'Generic', material: '', sizeChart: '' },
  },
  ozon: {
    title: (n) => `${n} — премиум качество, быстрая доставка по России`,
    desc: (n) => `【${n}】— высокое качество, доступная цена.\n\n✅ Премиум материалы\n✅ Сертифицировано\n✅ Быстрая доставка\n✅ Гарантия качества\n\nЗакажите прямо сейчас!`,
    keywords: ['купить', 'заказать', 'доставка'],
    price: 2990,
    extra: { vat: '20', barcode: '', attributes: [{ k: 'Цвет', v: 'Черный' }, { k: 'Материал', v: 'Пластик' }] },
  },
  wb: {
    title: (n) => `${n} — оригинал, сертификат, быстрая доставка`,
    desc: (n) => `${n}\n\n✅ Оригинальная продукция\n✅ Есть сертификаты\n✅ Быстрая доставка по РФ\n✅ Гарантия качества\n\nЦена указана за 1 шт.`,
    keywords: ['купить', 'цена', 'оригинал'],
    price: 2490,
    extra: { characteristics: [{ k: 'Цвет', v: 'Черный' }, { k: 'Размер', v: 'Standard' }], brand: 'Generic', country: 'China' },
  },
  yandex: {
    title: (n) => `${n} — высокое качество, быстрая доставка`,
    desc: (n) => `${n}\n\n✅ Гарантия качества\n✅ Быстрая доставка\n✅ Легкий возврат\n\nПокупайте с уверенностью!`,
    keywords: ['купить', 'доставка', 'гарантия'],
    price: 2690,
    extra: { manufacturer: 'Generic', barcode: '', warranty: 12 },
  },
  shopee: {
    title: (n) => `${n} | Free Shipping | Shopee Mall`,
    desc: (n) => `🔥 HOT SALE 🔥 ${n}\n\n✅ 100% Original\n✅ Free Shipping\n✅ Cash on Delivery\n✅ 7 Days Return\n\nOrder now! Limited stock!`,
    keywords: ['free shipping', 'best price', 'original'],
    price: 19,
    extra: { brand: 'Generic', weight: 0.3, daysToShip: '2', packageSize: [15, 10, 5] },
  },
  lazada: {
    title: (n) => `${n} | Free Shipping | Lazada`,
    desc: (n) => `🔥 ${n} 🔥\n\n✅ 100% Authentic\n✅ Free Shipping\n✅ Cash on Delivery\n✅ 15-Day Free Return\n\nShop now! Best price guaranteed!`,
    keywords: ['free shipping', 'authentic', 'best price'],
    price: 22,
    extra: { brand: 'Generic', weight: 0.3, warranty: 12 },
  },
  mercadolibre: {
    title: (n) => `${n} | Envío Gratis | Mercado Libre`,
    desc: (n) => `${n}\n\n✅ Envío Gratis\n✅ 12 Cuotas Sin Interés\n✅ Garantía\n✅ Devolución Gratis\n\nCompra con confianza!`,
    keywords: ['envío gratis', 'cuotas', 'garantía'],
    price: 1500,
    extra: { condition: 'new', brand: 'Generic', attributes: [{ k: 'Color', v: 'Negro' }] },
  },
  walmart: {
    title: (n) => `${n} - Everyday Low Price | Walmart`,
    desc: (n) => `${n}\n\n✅ Everyday Low Price\n✅ Free 2-Day Shipping\n✅ Easy Returns\n✅ Walmart Quality Promise\n\nOrder online for in-store pickup!`,
    keywords: ['low price', 'walmart', 'free shipping'],
    price: 28,
    extra: { brand: 'Generic', itemId: '', shippingWeight: 0.3 },
  },
  allegro: {
    title: (n) => `${n} - Najlepsza Jakość | Allegro`,
    desc: (n) => `${n}\n\n✅ Najwyższa jakość\n✅ Szybka wysyłka\n✅ 14 dni na zwrot\n✅ Bezpieczne zakupy\n\nKup teraz!`,
    keywords: ['najlepsza jakość', 'szybka wysyłka', 'okazja'],
    price: 129,
    extra: { condition: 'new', ean: '', brand: 'Generic' },
  },
}

// ========== 响应式状态 ==========
const generating = ref(false)
const selectedProductId = ref<number | null>(null)
const selectedPlatformId = ref<number | null>(null)
const newKeyword = ref('')
const newTag = ref('')

// ========== 计算属性 ==========
const productOptions = computed(() => mockProducts.map(p => ({ label: `${p.emoji} ${p.name}`, value: p.id })))

const platformOptions = computed(() => {
  const grouped: Record<string, { label: string, value: number }[]> = {}
  for (const p of platforms) {
    const g = p.region
    if (!grouped[g]) grouped[g] = []
    grouped[g].push({ label: p.name, value: p.id })
  }
  return Object.entries(grouped).map(([label, children]) => ({ label, children }))
})

const currentProduct = computed(() => mockProducts.find(p => p.id === selectedProductId.value) || null)
const currentPlatform = computed(() => platforms.find(p => p.id === selectedPlatformId.value) || null)

const platformExtraFields = computed(() => {
  if (!currentPlatform.value) return []
  return platformFieldMap[currentPlatform.value.code] || []
})

const categoryOptions = computed(() => {
  if (!currentPlatform.value) return []
  return categoryMap[currentPlatform.value.code] || categoryMap.amazon
})

interface FormState {
  title: string
  description: string
  keywords: string[]
  salePrice: number | null
  originalPrice: number | null
  category: string | null
  extra: Record<string, any>
}

const emptyForm = (): FormState => ({
  title: '',
  description: '',
  keywords: [],
  salePrice: null,
  originalPrice: null,
  category: null,
  extra: {},
})

const form = ref<FormState>(emptyForm())

const formValid = computed(() => {
  return form.value.title.trim() && form.value.description.trim() && form.value.salePrice !== null && form.value.salePrice > 0 && form.value.category
})

watch([selectedProductId, selectedPlatformId], () => {
  form.value = emptyForm()
})

// ========== 方法 ==========
function ensureExtraArray(key: string, count: number) {
  if (!Array.isArray(form.value.extra[key])) {
    form.value.extra[key] = Array(count).fill('')
  }
}

// Template-safe helpers to avoid `as` casts in templates (vue-tsc limitation)
function getStr(key: string): string {
  return (form.value.extra[key] ?? '') as string
}

function getStrArr(key: string): string[] {
  return (form.value.extra[key] ?? []) as string[]
}

function getKVArr(key: string): { k: string; v: string }[] {
  return (form.value.extra[key] ?? []) as { k: string; v: string }[]
}

function getAnyArr(key: string): any[] {
  return (form.value.extra[key] ?? []) as any[]
}

function updateNum(key: string, idx: number | null, val: number | null) {
  if (idx !== null) {
    const arr = getAnyArr(key)
    arr[idx] = val
    form.value.extra[key] = [...arr]
  } else {
    form.value.extra[key] = val
  }
}

function addKeyword() {
  const kw = newKeyword.value.trim()
  if (kw && !form.value.keywords.includes(kw)) {
    form.value.keywords.push(kw)
  }
  newKeyword.value = ''
}

function addTag(fieldKey: string) {
  const tag = newTag.value.trim()
  if (!tag) return
  if (!Array.isArray(form.value.extra[fieldKey])) {
    form.value.extra[fieldKey] = []
  }
  if (!(form.value.extra[fieldKey] as string[]).includes(tag)) {
    (form.value.extra[fieldKey] as string[]).push(tag)
  }
  newTag.value = ''
}

async function handleGenerateAI() {
  if (!currentProduct.value || !currentPlatform.value) return
  generating.value = true
  await new Promise(r => setTimeout(r, 600))
  const code = currentPlatform.value.code
  const tpl = platformTemplates[code] || platformTemplates.amazon
  const p = currentProduct.value
  form.value.title = tpl.title(p.name)
  form.value.description = tpl.desc(p.name)
  form.value.keywords = [...tpl.keywords]
  form.value.salePrice = tpl.price
  form.value.originalPrice = Math.round(tpl.price * 1.3)

  // Deep clone extra data
  const extra: Record<string, any> = {}
  for (const [k, v] of Object.entries(tpl.extra)) {
    extra[k] = Array.isArray(v) ? v.map((i: any) => typeof i === 'object' ? { ...i } : i) : v
  }
  form.value.extra = extra

  const cats = categoryOptions.value
  form.value.category = cats.length > 0 ? cats[0].value : null
  generating.value = false
  message.success('AI 内容已生成')
}

function handleSaveDraft() {
  message.success('已保存草稿')
}

function handlePublish() {
  if (!formValid.value) { message.error('请填写必填字段'); return }
  message.success(`已提交到 ${currentPlatform.value?.name} 发布队列`)
}
</script>

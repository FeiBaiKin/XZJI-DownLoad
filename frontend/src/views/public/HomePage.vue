<script setup lang="ts">
// 首页：双栏布局（主内容 + 侧边栏），对话式搜索，分类Tab精选资源，公告/购买弹窗
import { ref, computed, onMounted, type Component } from 'vue'
import { useRouter, useRoute } from 'vue-router'
import { NButton, NIcon, NModal, NTag, NEmpty, NSkeleton, NPagination, NAvatar } from 'naive-ui'
import {
  Search, Copy, ExternalLink, Lock, Crown, AlertCircle, Check, X,
  Clock, TrendingUp, ChevronRight, Send, Stars,
  Apps, Help, QuestionMark,
  User, Coin, Download, Heart,
} from '@vicons/tabler'
import api from '@/api'
import { contentApi, type PublicKeyword, type PaginatedKeywords, type Announcement, type Category } from '@/api/content'
import { useAuthStore } from '@/stores/auth'
import { useToastStore } from '@/stores/toast'
import { useSiteSettingsStore } from '@/stores/siteSettings'
import { useFavoritesStore } from '@/stores/favorites'
import { blockAccess } from '@/utils/blockAccess'
import type { KeywordBlock, KeywordAttribute } from '@/types/api'
import ResourceCard from '@/components/ResourceCard.vue'
import DOMPurify from 'dompurify'

const auth = useAuthStore()
const toast = useToastStore()
const router = useRouter()
const route = useRoute()
const settings = useSiteSettingsStore()
const favorites = useFavoritesStore()

// 支付返回检测
function checkPaymentReturn() {
  const q = route.query
  if (!q.from) return
  if (q.from === 'alipay' || q.from === 'epay') {
    if (q.trade_status === 'TRADE_SUCCESS' || q.trade_status === 'SUCCESS') {
      toast.success('支付成功！余额已到账')
    } else if (q.trade_status) {
      toast.error(`支付状态：${q.trade_status}`)
    } else {
      toast.info('正在处理支付结果，请查看余额确认')
    }
    router.replace({ query: {} })
  }
}

// ── 类型 ──
interface SearchResult {
  keywordId?: number
  keyword: string
  response: string | KeywordBlock[]
  icon?: string
  attributes?: string // JSON: [{label,value}]
  timestamp: string
  source?: string
}
interface PurchaseDialogState {
  keywordId: number
  blockIndex: number
  block: KeywordBlock
  cost: number
  access: 'balance' | 'membership'
  key: string
}

// ── 搜索状态 ──
const keyword = ref('')
const result = ref<SearchResult | null>(null)
const loading = ref(false)
const error = ref('')
const copiedIdx = ref<number | null>(null)
const purchaseDialog = ref<PurchaseDialogState | null>(null)
const purchaseLoading = ref(false)
const purchasedLinks = ref<Record<string, string>>({})
const purchasedCodes = ref<Record<string, string>>({})
const searchHistory = ref<string[]>([])

// 精选资源
const FEATURED_PAGE_SIZE = 8
const featuredItems = ref<PublicKeyword[]>([])
const featuredTotal = ref(0)
const featuredPage = ref(1)
const featuredLoading = ref(true)
const copiedKeywordId = ref<number | null>(null)

// 分类
const categories = ref<Category[]>([])
const selectedCategory = ref<number | null>(null)

// 侧边栏数据
const recentKeywords = ref<PublicKeyword[]>([])
const sidebarLoading = ref(true)
const downloadsToday = ref<number | null>(null)

// 公告
const visibleAnn = ref<Announcement | null>(null)

const inputRef = ref<HTMLInputElement | null>(null)

const isLoggedIn = computed(() => auth.isLoggedIn)

const isMember = computed(() => {
  const u = auth.user
  return !!u?.membership_level && !!u?.membership_expires_at && new Date(u.membership_expires_at) > new Date()
})

const currentBalance = computed(() => auth.user?.balance ?? 0)

// 搜索结果内容块
const resultBlocks = computed<KeywordBlock[] | null>(() =>
  Array.isArray(result.value?.response) ? (result.value!.response as KeywordBlock[]) : null
)

// 搜索结果的关键词属性（JSON 解析后）
const resultAttributes = computed<KeywordAttribute[]>(() => {
  const raw = result.value?.attributes
  if (!raw) return []
  try {
    const parsed = JSON.parse(raw)
    if (Array.isArray(parsed)) {
      return parsed.filter(
        (a: unknown): a is KeywordAttribute =>
          !!a && typeof (a as KeywordAttribute).label === 'string' && typeof (a as KeywordAttribute).value === 'string',
      )
    }
  } catch { /* 忽略损坏的 JSON */ }
  return []
})

// 公告正文
const annContent = computed(() => DOMPurify.sanitize(visibleAnn.value?.content || '暂无内容'))
const annTagType = computed(() => {
  const t = visibleAnn.value?.type
  return t === 'warning' ? 'warning' : t === 'success' ? 'success' : 'info'
})
const annTypeLabels: Record<string, string> = { info: '通知', warning: '重要', success: '活动' }

const featuredPageCount = computed(() =>
  Math.max(1, Math.ceil(featuredTotal.value / FEATURED_PAGE_SIZE)),
)

// 热门关键词（取前6个用于快捷搜索）
const hotKeywords = computed(() => {
  if (featuredItems.value.length > 0) {
    return featuredItems.value.slice(0, 6).map(k => k.keyword)
  }
  return searchHistory.value.slice(0, 6)
})

// ── 数据加载 ──
onMounted(async () => {
  checkPaymentReturn()
  // hydrate 收藏状态
  favorites.fetchIds()
  // 今日下载数（登录后获取）
  if (isLoggedIn.value) fetchDownloadsToday()
  // 从 localStorage 恢复搜索历史
  try {
    const saved = localStorage.getItem('search_history')
    if (saved) searchHistory.value = JSON.parse(saved)
  } catch { /* ignore */ }
  await Promise.all([
    fetchFeatured(),
    fetchCategories(),
    fetchRecentKeywords(),
    fetchAnnouncements(),
  ])
})

async function fetchFeatured() {
  featuredLoading.value = true
  try {
    let data: PaginatedKeywords
    if (selectedCategory.value) {
      data = await contentApi.public.keywords(selectedCategory.value, featuredPage.value, FEATURED_PAGE_SIZE)
    } else {
      data = await contentApi.public.featured(4, featuredPage.value, FEATURED_PAGE_SIZE)
    }
    featuredItems.value = data?.items ?? []
    featuredTotal.value = data?.total ?? 0
  } catch {
    featuredItems.value = []
    featuredTotal.value = 0
  } finally {
    featuredLoading.value = false
  }
}

async function fetchCategories() {
  try { categories.value = await contentApi.public.categories() } catch { categories.value = [] }
}

async function fetchRecentKeywords() {
  sidebarLoading.value = true
  try {
    const data = await contentApi.public.keywords(undefined, 1, 5)
    recentKeywords.value = data?.items ?? []
  } catch {
    recentKeywords.value = []
  } finally {
    sidebarLoading.value = false
  }
}

async function fetchDownloadsToday() {
  try {
    const res = await api.get('/user/downloads-today')
    downloadsToday.value = res.data?.downloads ?? null
  } catch { downloadsToday.value = null }
}

async function handleFeaturedPageChange(p: number) {
  featuredPage.value = p
  await fetchFeatured()
  document.getElementById('featured')?.scrollIntoView({ behavior: 'smooth', block: 'start' })
}

async function handleCategoryChange(catId: number | null) {
  selectedCategory.value = catId
  featuredPage.value = 1
  await fetchFeatured()
}

async function fetchAnnouncements() {
  try {
    const data = await contentApi.public.announcements()
    if (data.length > 0) {
      const dismissed = JSON.parse(localStorage.getItem('dismissed_announcements') || '{}')
      const firstUnseen = data.find((a: Announcement) => !dismissed[a.id])
      if (firstUnseen) visibleAnn.value = firstUnseen
    }
  } catch { /* ignore */ }
}

// ── 搜索 ──
async function handleSearch(query?: string) {
  const q = (query || keyword.value).trim()
  if (!q) { error.value = '请输入关键词'; return }
  loading.value = true
  error.value = ''
  result.value = null
  purchasedLinks.value = {}
  purchasedCodes.value = {}
  keyword.value = q
  // 更新搜索历史
  if (!searchHistory.value.includes(q)) {
    searchHistory.value = [q, ...searchHistory.value].slice(0, 10)
    localStorage.setItem('search_history', JSON.stringify(searchHistory.value))
  }
  try {
    const data = await contentApi.public.search(q)
    if (typeof data.response === 'string') {
      try { data.response = JSON.parse(data.response) } catch { /* 纯文本 */ }
    }
    result.value = data
    if (isLoggedIn.value && data.keywordId && Array.isArray(data.response)) {
      const paid: Record<string, string> = {}
      const paidCodes: Record<string, string> = {}
      for (let i = 0; i < data.response.length; i++) {
        const block = data.response[i]
        if (block.type !== 'link') continue
        const access = blockAccess(block)
        if (access === 'free') continue
        try {
          const sRes = await api.get('/user/purchase-status', { params: { keywordId: data.keywordId, blockIndex: i } })
          if (sRes.data?.purchased) {
            paid[`${data.keywordId}-${i}`] = sRes.data?.url || ''
            paidCodes[`${data.keywordId}-${i}`] = sRes.data?.code || ''
          }
        } catch { /* ignore */ }
      }
      purchasedLinks.value = paid
      purchasedCodes.value = paidCodes
    }
  } catch (err: unknown) {
    error.value = (err as Error)?.message || '搜索失败，请重试'
  } finally {
    loading.value = false
  }
}

// ── 相对时间：X分钟前 / X小时前 / X天前 / M-D（超过 7 天）
function formatRelative(ts?: string): string {
  if (!ts) return '-'
  const d = new Date(ts)
  const diff = Date.now() - d.getTime()
  if (diff < 60_000) return '刚刚'
  if (diff < 3_600_000) return `${Math.floor(diff / 60_000)}分钟前`
  if (diff < 86_400_000) return `${Math.floor(diff / 3_600_000)}小时前`
  if (diff < 604_800_000) return `${Math.floor(diff / 86_400_000)}天前`
  return `${d.getMonth() + 1}-${d.getDate()}`
}

// ── 复制提取码 ──
function handleCopyCode(code: string, idx: number) {
  navigator.clipboard.writeText(code).then(() => {
    copiedIdx.value = idx
    setTimeout(() => { copiedIdx.value = null }, 1500)
  })
}

// ── 精选卡片：复制关键词 ──
function handleKeywordCopy(kw: PublicKeyword) {
  copiedKeywordId.value = kw.id
  setTimeout(() => { copiedKeywordId.value = null }, 1200)
  inputRef.value?.focus()
}

// ── 链接访问控制 ──
function blockCode(block: KeywordBlock, i: number): string {
  if (blockAccess(block) === 'free') return block.code || ''
  const key = `${result.value?.keywordId}-${i}`
  return purchasedCodes.value[key] || ''
}

function linkIconOf(block: KeywordBlock): Component {
  const access = blockAccess(block)
  if (access === 'membership') return Crown
  if (access === 'balance') return Lock
  return ExternalLink
}

function handleLinkClick(e: Event, block: KeywordBlock, idx: number) {
  const access = blockAccess(block)
  if (access === 'free') return
  e.preventDefault()
  if (!isLoggedIn.value) { router.push('/login'); return }
  if (!result.value?.keywordId) return
  const key = `${result.value.keywordId}-${idx}`
  if (purchasedLinks.value[key]) { window.open(purchasedLinks.value[key], '_blank'); return }
  purchaseDialog.value = { keywordId: result.value.keywordId, blockIndex: idx, block, cost: block.cost || 0, access, key }
}

// ── 购买付费链接 ──
async function handlePurchase() {
  if (!purchaseDialog.value) return
  purchaseLoading.value = true
  const d = purchaseDialog.value
  try {
    const res = await api.post('/user/purchase-link', {
      keywordId: d.keywordId,
      blockIndex: d.blockIndex,
    })
    const data = res.data
    purchasedLinks.value = { ...purchasedLinks.value, [d.key]: data.url || '' }
    purchasedCodes.value = { ...purchasedCodes.value, [d.key]: data.code || '' }
    purchaseDialog.value = null
    window.open(data.url, '_blank')
    if (data.balance !== undefined && auth.user) auth.user.balance = data.balance
  } catch (err: unknown) {
    const data = (err as { response?: { data?: { error?: string } } })?.response?.data
    const errMsg = data?.error || '购买失败'
    toast.error(errMsg)
    if (d.access === 'membership' && errMsg.includes('会员')) {
      purchaseDialog.value = null
      router.push('/user/membership')
    }
  } finally {
    purchaseLoading.value = false
  }
}

function dismissAnnouncement() {
  if (visibleAnn.value) {
    const dismissed = JSON.parse(localStorage.getItem('dismissed_announcements') || '{}')
    dismissed[visibleAnn.value.id] = true
    localStorage.setItem('dismissed_announcements', JSON.stringify(dismissed))
  }
  visibleAnn.value = null
}

// ── 快捷入口点击 ──
function handleQuickAction(action: string) {
  switch (action) {
    case 'search':
      inputRef.value?.focus()
      break
    case 'categories':
      router.push('/categories')
      break
    case 'help':
      router.push('/page/help')
      break
    case 'about':
      router.push('/page/about')
      break
  }
}
</script>

<template>
  <div>
    <!-- ═══ 双栏区域 ═══ -->
    <div class="lg:grid lg:grid-cols-[1fr_320px] lg:gap-8 pt-14 sm:pt-20">
      <!-- ── 左栏：主内容 ── -->
      <div class="min-w-0">

        <!-- ① 搜索模块（对话式） -->
        <section class="overflow-hidden rounded-2xl border border-zinc-200 bg-white dark:border-zinc-800 dark:bg-zinc-900">
          <!-- 对话区域 -->
          <div class="px-5 pt-5 sm:px-6 sm:pt-6">
            <!-- 热门关键词快捷标签 -->
            <div v-if="hotKeywords.length > 0 && !result" class="mb-4">
              <p class="mb-2.5 flex items-center gap-1.5 text-xs text-zinc-400 dark:text-zinc-500">
                <NIcon :size="14"><TrendingUp /></NIcon>
                热门搜索
              </p>
              <div class="flex flex-wrap gap-2">
                <button
                  v-for="kw in hotKeywords"
                  :key="kw"
                  type="button"
                  class="rounded-full border border-zinc-200 px-3 py-1.5 text-[13px] text-zinc-600 motion-safe:transition-colors hover:border-emerald-400/60 hover:text-emerald-600 dark:border-zinc-700 dark:text-zinc-400 dark:hover:border-emerald-400/40 dark:hover:text-emerald-400"
                  @click="handleSearch(kw)"
                >
                  {{ kw }}
                </button>
              </div>
            </div>

            <!-- 搜索结果：对话式展示 -->
            <div v-if="result" class="space-y-4">
              <!-- 用户查询气泡 -->
              <div class="flex items-start justify-end gap-2.5">
                <div class="max-w-[85%] rounded-2xl rounded-br-md bg-emerald-600 px-4 py-2.5 dark:bg-emerald-500">
                  <p class="text-sm text-white">{{ result.keyword }}</p>
                </div>
                <div
                  class="grid size-8 shrink-0 place-items-center rounded-full bg-zinc-200 text-xs font-semibold text-zinc-500 dark:bg-zinc-700 dark:text-zinc-400"
                >
                  {{ (auth.user?.nickname || auth.user?.username || 'U')[0]?.toUpperCase() || '?' }}
                </div>
              </div>

              <!-- 系统响应区：重新设计 —— 图标 + 关键词 + 属性 tag + 内容块 -->
              <div class="flex items-start gap-2.5">
                <!-- 站点头像：有真实图标则用图标，否则用站点首字母 -->
                <div
                  class="grid size-10 shrink-0 place-items-center overflow-hidden rounded-xl text-xs font-bold text-white"
                  :class="result.icon ? '' : 'bg-emerald-600 dark:bg-emerald-400 dark:text-zinc-950'"
                >
                  <img v-if="result.icon" :src="result.icon" :alt="result.keyword" class="size-full object-cover" />
                  <span v-else>{{ (settings.siteName?.[0] || 'X').toUpperCase() }}</span>
                </div>

                <div class="min-w-0 flex-1 rounded-2xl rounded-bl-md border border-zinc-200 bg-zinc-50/80 px-4 py-3.5 dark:border-zinc-700 dark:bg-zinc-800/60">
                  <!-- 头部：关键词名称 + 属性 tags -->
                  <div class="mb-3 flex flex-wrap items-center gap-2">
                    <span class="text-base font-semibold tracking-tight text-zinc-900 dark:text-zinc-100">
                      {{ result.keyword }}
                    </span>
                    <span
                      v-for="(attr, ai) in resultAttributes"
                      :key="ai"
                      class="inline-flex items-center gap-1 rounded-md border border-emerald-200 bg-emerald-50/80 px-1.5 py-0.5 text-[11px] font-medium text-emerald-700 dark:border-emerald-400/25 dark:bg-emerald-400/10 dark:text-emerald-400"
                    >
                      <span class="opacity-70">{{ attr.label }}</span>
                      <span class="opacity-40">·</span>
                      <span>{{ attr.value }}</span>
                    </span>
                    <span class="ml-auto text-xs text-zinc-400 dark:text-zinc-500" :title="result.timestamp">
                      {{ formatRelative(result.timestamp) }}
                    </span>
                  </div>

                  <!-- 分隔线 -->
                  <div class="mb-3 border-t border-zinc-150 dark:border-zinc-700" />

                  <!-- 响应内容块 -->
                  <div v-if="resultBlocks" class="space-y-2.5">
                    <template v-for="(block, i) in resultBlocks" :key="i">
                      <p
                        v-if="block.type === 'text' && block.text"
                        class="text-[14px] leading-relaxed whitespace-pre-wrap text-zinc-700 dark:text-zinc-300"
                      >
                        {{ block.text }}
                      </p>

                      <img
                        v-else-if="block.type === 'image' && block.src"
                        :src="block.src"
                        :alt="block.alt || ''"
                        class="max-h-80 rounded-lg border border-zinc-100 dark:border-zinc-700"
                      />

                      <a
                        v-else-if="block.type === 'link'"
                        :href="blockAccess(block) === 'free' ? block.url : 'javascript:void(0)'"
                        target="_blank"
                        rel="noopener noreferrer"
                        class="group flex flex-wrap items-center gap-2.5 rounded-lg border px-3 py-2 motion-safe:transition-colors"
                        :class="blockAccess(block) !== 'free'
                          ? 'border-amber-200 bg-amber-50/60 hover:border-amber-300 dark:border-amber-400/20 dark:bg-amber-400/5 dark:hover:border-amber-400/40'
                          : 'border-zinc-200 hover:border-emerald-400 dark:border-zinc-600 dark:hover:border-emerald-400/60'"
                        @click="(e) => handleLinkClick(e, block, i)"
                      >
                        <span
                          class="grid size-7 shrink-0 place-items-center rounded-md"
                          :class="blockAccess(block) !== 'free'
                            ? 'bg-amber-100 text-amber-600 dark:bg-amber-400/10 dark:text-amber-400'
                            : 'bg-emerald-50 text-emerald-600 dark:bg-emerald-400/10 dark:text-emerald-400'"
                        >
                          <NIcon :size="15"><component :is="linkIconOf(block)" /></NIcon>
                        </span>
                        <span class="min-w-0 flex-1 truncate text-[13px] font-medium text-zinc-800 group-hover:text-zinc-950 dark:text-zinc-200 dark:group-hover:text-white">
                          {{ block.label || '链接' }}
                        </span>
                        <NTag
                          v-if="blockAccess(block) !== 'free' && !purchasedLinks[`${result.keywordId}-${i}`]"
                          :type="blockAccess(block) === 'membership' ? 'warning' : 'default'"
                          size="small"
                          round
                        >
                          {{ blockAccess(block) === 'membership' ? '会员免费' : `¥${block.cost || 0}` }}
                        </NTag>
                        <NTag
                          v-if="blockAccess(block) !== 'free' && purchasedLinks[`${result.keywordId}-${i}`]"
                          type="success"
                          size="small"
                          round
                        >已购买</NTag>
                        <button
                          v-if="blockCode(block, i)"
                          type="button"
                          class="flex items-center gap-1 rounded-md border border-dashed px-2 py-1 font-mono text-[11px] motion-safe:transition-colors"
                          :class="copiedIdx === i
                            ? 'border-emerald-400 text-emerald-600 dark:text-emerald-400'
                            : 'border-zinc-300 text-zinc-500 hover:border-zinc-400 hover:text-zinc-700 dark:border-zinc-600 dark:text-zinc-400 dark:hover:text-zinc-200'"
                          title="点击复制提取码"
                          @click.prevent.stop="handleCopyCode(blockCode(block, i), i)"
                        >
                          <NIcon :size="11"><Check v-if="copiedIdx === i" /><Copy v-else /></NIcon>
                          {{ copiedIdx === i ? '已复制' : `提取码 ${blockCode(block, i)}` }}
                        </button>
                      </a>
                    </template>
                  </div>
                  <p v-else class="whitespace-pre-wrap text-[14px] leading-relaxed text-zinc-700 dark:text-zinc-300">{{ result.response }}</p>
                </div>
              </div>
            </div>

            <!-- 搜索结果元信息 -->
            <div
              v-if="result"
              class="mt-3 flex flex-wrap items-center gap-x-3 gap-y-1 border-t border-zinc-100 pt-3 text-xs text-zinc-400 dark:border-zinc-800 dark:text-zinc-500"
            >
              <span v-if="result.source" class="flex items-center gap-1">
                <NIcon :size="12"><Clock /></NIcon>
                {{ result.source === 'database' ? '已收录关键词' : '系统默认回复' }}
              </span>
            </div>
          </div>

          <!-- 搜索输入框 -->
          <div class="border-t border-zinc-100 px-4 py-4 dark:border-zinc-800 sm:px-5">
            <div class="flex items-center gap-2.5">
              <div class="flex h-11 flex-1 items-center gap-2.5 rounded-xl border border-zinc-200 bg-zinc-50/60 px-3.5 motion-safe:transition-colors focus-within:border-emerald-500 focus-within:bg-white dark:border-zinc-700 dark:bg-zinc-800/50 dark:focus-within:border-emerald-400 dark:focus-within:bg-zinc-900">
                <NIcon :size="18" class="shrink-0 text-zinc-400 dark:text-zinc-500"><Search /></NIcon>
                <input
                  ref="inputRef"
                  v-model="keyword"
                  type="text"
                  class="h-full min-w-0 flex-1 bg-transparent text-[14px] text-zinc-900 outline-none placeholder:text-zinc-400 dark:text-zinc-100 dark:placeholder:text-zinc-500"
                  placeholder="输入关键词，如：Photoshop、Win11、ChatGPT..."
                  :disabled="loading"
                  autocomplete="off"
                  spellcheck="false"
                  @keypress.enter="handleSearch()"
                />
              </div>
              <NButton type="primary" circle size="large" :loading="loading" :disabled="!keyword.trim()" @click="handleSearch()">
                <template #icon><NIcon :size="18"><Send /></NIcon></template>
              </NButton>
            </div>
            <p v-if="error" class="mt-2 pl-1 text-xs text-red-500 dark:text-red-400">{{ error }}</p>
            <p v-else class="mt-2 pl-1 text-[11px] text-zinc-400 dark:text-zinc-500">
              提示：请输入软件名称、版本或功能关键词，系统将为你查找相关资源。
            </p>
          </div>
        </section>

        <!-- ② 精选资源模块 -->
        <section id="featured" class="mt-8 scroll-mt-20">
          <div class="mb-4 flex items-center justify-between">
            <div class="flex items-center gap-2">
              <NIcon :size="18" class="text-emerald-600 dark:text-emerald-400"><Stars /></NIcon>
              <h2 class="text-lg font-semibold tracking-tight text-zinc-900 dark:text-zinc-100">精选资源</h2>
            </div>
            <div class="flex items-center gap-3">
              <span v-if="!featuredLoading" class="font-mono text-xs text-zinc-400 dark:text-zinc-500">
                共 {{ featuredTotal }} 个
              </span>
              <router-link
                to="/categories"
                class="flex items-center gap-1 text-xs text-zinc-400 motion-safe:transition-colors hover:text-emerald-600 dark:text-zinc-500 dark:hover:text-emerald-400"
              >
                查看更多 <NIcon :size="13"><ChevronRight /></NIcon>
              </router-link>
            </div>
          </div>

          <!-- 分类 Tab -->
          <div v-if="categories.length > 0" class="mb-4 flex flex-wrap gap-1.5">
            <button
              type="button"
              class="rounded-full px-3.5 py-1.5 text-[13px] font-medium motion-safe:transition-colors"
              :class="selectedCategory === null
                ? 'bg-emerald-600 text-white dark:bg-emerald-400 dark:text-zinc-950'
                : 'border border-zinc-200 text-zinc-600 hover:border-emerald-400/60 hover:text-emerald-600 dark:border-zinc-700 dark:text-zinc-400 dark:hover:border-emerald-400/40 dark:hover:text-emerald-400'"
              @click="handleCategoryChange(null)"
            >全部</button>
            <button
              v-for="cat in categories"
              :key="cat.id"
              type="button"
              class="rounded-full px-3.5 py-1.5 text-[13px] font-medium motion-safe:transition-colors"
              :class="selectedCategory === cat.id
                ? 'bg-emerald-600 text-white dark:bg-emerald-400 dark:text-zinc-950'
                : 'border border-zinc-200 text-zinc-600 hover:border-emerald-400/60 hover:text-emerald-600 dark:border-zinc-700 dark:text-zinc-400 dark:hover:border-emerald-400/40 dark:hover:text-emerald-400'"
              @click="handleCategoryChange(cat.id)"
            >{{ cat.name }}</button>
          </div>

          <!-- 加载骨架 -->
          <div v-if="featuredLoading" class="grid gap-3 sm:grid-cols-2 sm:gap-4">
            <div
              v-for="i in 8"
              :key="i"
              class="flex flex-col rounded-2xl border border-zinc-200 bg-white p-4 dark:border-zinc-800 dark:bg-zinc-900"
            >
              <div class="flex items-center gap-2.5 pr-7">
                <NSkeleton width="40" height="40" :sharp="false" />
                <div class="flex-1 space-y-2">
                  <NSkeleton text width="85%" />
                </div>
              </div>
              <div class="mt-2.5 space-y-2">
                <NSkeleton text width="100%" />
                <NSkeleton text width="70%" />
              </div>
              <div class="mt-3 flex items-center justify-between">
                <NSkeleton text width="60px" height="20px" :sharp="false" />
                <NSkeleton text width="40px" height="20px" :sharp="false" />
              </div>
            </div>
          </div>

          <!-- 空状态 -->
          <div v-else-if="featuredItems.length === 0" class="rounded-2xl border border-dashed border-zinc-300 py-14 dark:border-zinc-700">
            <NEmpty description="暂无精选资源">
              <template #extra>
                <span class="text-xs text-zinc-400 dark:text-zinc-500">在管理后台添加关键词与分类后将在此展示</span>
              </template>
            </NEmpty>
          </div>

          <!-- 资源列表（两列网格卡片） -->
          <div v-else class="grid gap-3 sm:grid-cols-2 sm:gap-4">
            <ResourceCard
              v-for="kw in featuredItems"
              :key="kw.id"
              :keyword="kw"
              :category-name="categories.find((c: Category) => c.id === selectedCategory)?.name"
              :favorited="favorites.has(kw.id)"
              @copy="handleKeywordCopy"
              @toggle-favorite="favorites.toggle(kw.id)"
            />
          </div>

          <!-- 分页 -->
          <div v-if="!featuredLoading && featuredTotal > FEATURED_PAGE_SIZE" class="mt-6 flex justify-center">
            <NPagination
              :page="featuredPage"
              :page-count="featuredPageCount"
              :item-count="featuredTotal"
              show-quick-jumper
              @update:page="handleFeaturedPageChange"
            />
          </div>
        </section>
      </div>

      <!-- ── 右栏：侧边栏 ── -->
      <aside class="mt-8 space-y-5 lg:mt-0">
        <!-- ① 个人信息卡片 -->
        <div class="overflow-hidden rounded-2xl border border-zinc-200 bg-white dark:border-zinc-800 dark:bg-zinc-900">
          <!-- 顶部用户信息 -->
          <div class="px-5 pb-3 pt-5">
            <div class="flex items-center gap-3">
              <NAvatar
                v-if="isLoggedIn"
                :src="auth.user?.avatar || undefined"
                :size="48"
                round
                class="shrink-0"
              />
              <span
                v-else
                class="grid size-12 shrink-0 place-items-center rounded-full bg-zinc-200 text-base font-semibold text-zinc-500 dark:bg-zinc-800 dark:text-zinc-400"
              >
                <NIcon :size="22"><User /></NIcon>
              </span>
              <div class="min-w-0 flex-1">
                <div class="flex items-center gap-2">
                  <span class="truncate text-[15px] font-semibold text-zinc-900 dark:text-zinc-100">
                    {{ isLoggedIn ? (auth.user?.nickname || auth.user?.username) : '游客' }}
                  </span>
                  <span
                    v-if="isLoggedIn && auth.user?.level"
                    class="shrink-0 rounded-full bg-emerald-100 px-2 py-0.5 text-[11px] font-medium text-emerald-700 dark:bg-emerald-400/15 dark:text-emerald-400"
                  >Lv.{{ auth.user.level }}</span>
                </div>
                <p class="mt-0.5 text-xs text-zinc-400 dark:text-zinc-500">
                  {{ isLoggedIn ? (isMember ? '会员用户' : '普通用户') : '探索者' }}
                </p>
              </div>
            </div>
            <p v-if="!isLoggedIn" class="mt-2 text-xs text-zinc-400 dark:text-zinc-500">
              登录后享受更多功能
            </p>
            <p v-else class="mt-2 text-xs text-zinc-400 dark:text-zinc-500 line-clamp-1">
              简介：这个人很懒，什么也没有留下~
            </p>
          </div>

          <!-- 数据统计（3 列） -->
          <div class="grid grid-cols-3 border-t border-zinc-100 dark:border-zinc-800">
            <div class="flex flex-col items-center py-3">
              <span class="flex items-center gap-1 text-[11px] text-zinc-400 dark:text-zinc-500"><NIcon :size="12"><Coin /></NIcon>余额</span>
              <span class="mt-0.5 font-mono text-sm font-semibold text-zinc-800 dark:text-zinc-200">
                {{ isLoggedIn ? (currentBalance.toFixed(2)) : '-' }}
              </span>
            </div>
            <div class="flex flex-col items-center py-3">
              <span class="flex items-center gap-1 text-[11px] text-zinc-400 dark:text-zinc-500"><NIcon :size="12"><Download /></NIcon>下载</span>
              <span class="mt-0.5 font-mono text-sm font-semibold text-zinc-800 dark:text-zinc-200">
                {{ isLoggedIn ? (downloadsToday !== null ? downloadsToday : '-') : '-' }}
              </span>
            </div>
            <div class="flex flex-col items-center py-3">
              <span class="flex items-center gap-1 text-[11px] text-zinc-400 dark:text-zinc-500"><NIcon :size="12"><Heart /></NIcon>收藏</span>
              <span class="mt-0.5 font-mono text-sm font-semibold text-zinc-800 dark:text-zinc-200">
                {{ isLoggedIn ? favorites.favoritedIds.size : '-' }}
              </span>
            </div>
          </div>

          <!-- 未登录操作 -->
          <div v-if="!isLoggedIn" class="border-t border-zinc-100 px-5 pb-4 pt-3 dark:border-zinc-800">
            <div class="flex gap-2">
              <NButton type="primary" size="small" block @click="router.push('/login')">登录</NButton>
              <NButton secondary size="small" block @click="router.push('/register')">注册</NButton>
            </div>
          </div>
        </div>

        <!-- ② 最新更新 -->
        <div class="rounded-2xl border border-zinc-200 bg-white p-5 dark:border-zinc-800 dark:bg-zinc-900">
          <div class="mb-3 flex items-center justify-between">
            <div class="flex items-center gap-2">
              <NIcon :size="16" class="text-emerald-600 dark:text-emerald-400"><Clock /></NIcon>
              <h3 class="text-[15px] font-semibold text-zinc-900 dark:text-zinc-100">最新更新</h3>
            </div>
            <router-link
              to="/categories"
              class="flex items-center gap-1 text-[11px] text-zinc-400 motion-safe:transition-colors hover:text-emerald-600 dark:text-zinc-500 dark:hover:text-emerald-400"
            >
              更多 <NIcon :size="11"><ChevronRight /></NIcon>
            </router-link>
          </div>

          <div v-if="sidebarLoading" class="space-y-3">
            <div v-for="i in 4" :key="i" class="flex items-center gap-2.5">
              <NSkeleton width="6" height="6" round />
              <NSkeleton text width="70%" />
            </div>
          </div>
          <div v-else-if="recentKeywords.length === 0" class="py-2 text-center text-xs text-zinc-400 dark:text-zinc-500">
            暂无更新
          </div>
          <ul v-else class="space-y-2.5">
            <li v-for="kw in recentKeywords" :key="kw.id">
              <button
                type="button"
                class="flex w-full items-center gap-2 text-left motion-safe:transition-colors hover:text-emerald-600 dark:hover:text-emerald-400"
                @click="handleSearch(kw.keyword)"
              >
                <span class="min-w-0 flex-1 truncate text-[13px] text-zinc-700 dark:text-zinc-300">{{ kw.keyword }}</span>
                <span class="shrink-0 font-mono text-[11px] text-zinc-400 dark:text-zinc-500">{{ formatRelative(kw.updated_at || kw.created_at) }}</span>
              </button>
            </li>
          </ul>
        </div>

        <!-- ④ 快捷入口 -->
        <div class="rounded-2xl border border-zinc-200 bg-white p-5 dark:border-zinc-800 dark:bg-zinc-900">
          <div class="mb-3 flex items-center gap-2">
            <NIcon :size="16" class="text-emerald-600 dark:text-emerald-400"><Apps /></NIcon>
            <h3 class="text-[15px] font-semibold text-zinc-900 dark:text-zinc-100">快捷入口</h3>
          </div>
          <div class="grid grid-cols-2 gap-2.5">
            <button
              type="button"
              class="flex flex-col items-center gap-1.5 rounded-xl border border-zinc-200 py-3 motion-safe:transition-colors hover:border-emerald-400/60 hover:bg-emerald-50/50 dark:border-zinc-700 dark:hover:border-emerald-400/30 dark:hover:bg-emerald-400/5"
              @click="handleQuickAction('search')"
            >
              <NIcon :size="20" class="text-zinc-500 dark:text-zinc-400"><Search /></NIcon>
              <span class="text-[12px] text-zinc-600 dark:text-zinc-400">搜索资源</span>
            </button>
            <button
              type="button"
              class="flex flex-col items-center gap-1.5 rounded-xl border border-zinc-200 py-3 motion-safe:transition-colors hover:border-emerald-400/60 hover:bg-emerald-50/50 dark:border-zinc-700 dark:hover:border-emerald-400/30 dark:hover:bg-emerald-400/5"
              @click="handleQuickAction('categories')"
            >
              <NIcon :size="20" class="text-zinc-500 dark:text-zinc-400"><Apps /></NIcon>
              <span class="text-[12px] text-zinc-600 dark:text-zinc-400">分类浏览</span>
            </button>
            <button
              type="button"
              class="flex flex-col items-center gap-1.5 rounded-xl border border-zinc-200 py-3 motion-safe:transition-colors hover:border-emerald-400/60 hover:bg-emerald-50/50 dark:border-zinc-700 dark:hover:border-emerald-400/30 dark:hover:bg-emerald-400/5"
              @click="handleQuickAction('help')"
            >
              <NIcon :size="20" class="text-zinc-500 dark:text-zinc-400"><Help /></NIcon>
              <span class="text-[12px] text-zinc-600 dark:text-zinc-400">帮助中心</span>
            </button>
            <button
              type="button"
              class="flex flex-col items-center gap-1.5 rounded-xl border border-zinc-200 py-3 motion-safe:transition-colors hover:border-emerald-400/60 hover:bg-emerald-50/50 dark:border-zinc-700 dark:hover:border-emerald-400/30 dark:hover:bg-emerald-400/5"
              @click="handleQuickAction('about')"
            >
              <NIcon :size="20" class="text-zinc-500 dark:text-zinc-400"><QuestionMark /></NIcon>
              <span class="text-[12px] text-zinc-600 dark:text-zinc-400">关于本站</span>
            </button>
          </div>
        </div>
      </aside>
    </div>

    <!-- ═══ 公告弹窗 ═══ -->
    <NModal :show="!!visibleAnn" @update:show="(v: boolean) => { if (!v) dismissAnnouncement() }">
      <div v-if="visibleAnn" class="w-[92vw] max-w-lg rounded-2xl border border-zinc-200 bg-white shadow-xl dark:border-zinc-800 dark:bg-zinc-900">
        <div class="flex items-center justify-between border-b border-zinc-100 px-6 py-4 dark:border-zinc-800">
          <div class="flex items-center gap-2.5">
            <NTag :type="annTagType" size="small" round>{{ annTypeLabels[visibleAnn.type] || visibleAnn.type }}</NTag>
            <h3 class="text-base font-semibold text-zinc-900 dark:text-zinc-100">{{ visibleAnn.title }}</h3>
          </div>
          <button
            type="button"
            class="flex size-8 items-center justify-center rounded-full text-zinc-400 motion-safe:transition-colors hover:bg-zinc-100 hover:text-zinc-600 dark:hover:bg-zinc-800 dark:hover:text-zinc-300"
            aria-label="关闭"
            @click="dismissAnnouncement"
          >
            <NIcon :size="16"><X /></NIcon>
          </button>
        </div>
        <div class="rich-content max-h-[60vh] overflow-y-auto px-6 py-4 text-sm" v-html="annContent" />
        <div class="border-t border-zinc-100 px-6 py-4 dark:border-zinc-800">
          <NButton type="primary" block @click="dismissAnnouncement">我知道了</NButton>
        </div>
      </div>
    </NModal>

    <!-- ═══ 购买确认弹窗 ═══ -->
    <NModal :show="!!purchaseDialog" :mask-closable="!purchaseLoading" @update:show="(v: boolean) => { if (!v) purchaseDialog = null }">
      <div v-if="purchaseDialog" class="w-[92vw] max-w-sm rounded-2xl border border-zinc-200 bg-white shadow-xl dark:border-zinc-800 dark:bg-zinc-900">
        <div class="flex items-center justify-between border-b border-zinc-100 px-6 py-4 dark:border-zinc-800">
          <div class="flex items-center gap-2">
            <NIcon :size="18" :class="purchaseDialog.access === 'membership' ? 'text-amber-500' : 'text-emerald-600 dark:text-emerald-400'">
              <Crown v-if="purchaseDialog.access === 'membership'" /><Lock v-else />
            </NIcon>
            <h3 class="text-base font-semibold text-zinc-900 dark:text-zinc-100">
              {{ purchaseDialog.access === 'membership' ? '会员专属链接' : '付费链接' }}
            </h3>
          </div>
          <button
            type="button"
            class="flex size-8 items-center justify-center rounded-full text-zinc-400 motion-safe:transition-colors hover:bg-zinc-100 hover:text-zinc-600 dark:hover:bg-zinc-800 dark:hover:text-zinc-300"
            aria-label="关闭"
            @click="purchaseDialog = null"
          >
            <NIcon :size="16"><X /></NIcon>
          </button>
        </div>

        <div class="space-y-4 px-6 py-5">
          <div>
            <p class="text-xs text-zinc-400 dark:text-zinc-500">链接名称</p>
            <p class="mt-1 truncate text-sm font-medium text-zinc-800 dark:text-zinc-200">
              {{ purchaseDialog.block.label || purchaseDialog.block.url || '链接' }}
            </p>
          </div>

          <template v-if="purchaseDialog.access === 'membership'">
            <p class="rounded-lg bg-amber-50 px-3 py-2 text-xs leading-relaxed text-amber-700 dark:bg-amber-400/10 dark:text-amber-400">
              开通会员后免费下载，每日有次数限制
            </p>
            <div class="flex items-center justify-between text-sm">
              <span class="text-zinc-500 dark:text-zinc-400">非会员价格</span>
              <span class="font-mono font-medium text-zinc-900 dark:text-zinc-100">
                {{ purchaseDialog.cost > 0 ? `¥${purchaseDialog.cost}` : '不可购买' }}
              </span>
            </div>
          </template>
          <template v-else>
            <p
              v-if="isMember"
              class="flex items-start gap-1.5 rounded-lg bg-emerald-50 px-3 py-2 text-xs leading-relaxed text-emerald-700 dark:bg-emerald-400/10 dark:text-emerald-400"
            >
              <NIcon :size="14" class="mt-0.5 shrink-0"><Crown /></NIcon>
              会员折扣：购买付费链接将自动享受会员折扣价
            </p>
            <div class="flex items-center justify-between text-sm">
              <span class="text-zinc-500 dark:text-zinc-400">所需余额</span>
              <span class="font-mono text-base font-semibold text-zinc-900 dark:text-zinc-100">¥{{ purchaseDialog.cost }}</span>
            </div>
          </template>

          <div class="flex items-center justify-between border-t border-zinc-100 pt-3 text-sm dark:border-zinc-800">
            <span class="text-zinc-500 dark:text-zinc-400">当前余额</span>
            <span
              class="font-mono"
              :class="currentBalance < purchaseDialog.cost && purchaseDialog.access !== 'membership'
                ? 'text-red-500 dark:text-red-400'
                : 'text-zinc-700 dark:text-zinc-300'"
            >¥{{ currentBalance.toFixed(2) }}</span>
          </div>

          <p
            v-if="currentBalance < purchaseDialog.cost && purchaseDialog.access !== 'membership'"
            class="flex items-center gap-1.5 rounded-lg bg-red-50 px-3 py-2 text-xs text-red-600 dark:bg-red-500/10 dark:text-red-400"
          >
            <NIcon :size="14"><AlertCircle /></NIcon>
            余额不足，还需 <span class="font-mono font-semibold">¥{{ (purchaseDialog.cost - currentBalance).toFixed(2) }}</span>
          </p>

          <div class="space-y-2 pt-1">
            <template v-if="purchaseDialog.access === 'membership'">
              <NButton type="primary" block :loading="purchaseLoading" @click="handlePurchase">会员验证下载</NButton>
              <NButton block secondary @click="purchaseDialog = null; router.push('/user/membership')">开通会员</NButton>
            </template>
            <template v-else-if="currentBalance < purchaseDialog.cost">
              <NButton type="primary" block @click="purchaseDialog = null; router.push('/user')">去充值</NButton>
              <NButton quaternary block @click="purchaseDialog = null">取消</NButton>
            </template>
            <template v-else>
              <NButton type="primary" block :loading="purchaseLoading" @click="handlePurchase">确认购买</NButton>
              <NButton quaternary block @click="purchaseDialog = null">取消</NButton>
            </template>
          </div>
        </div>
      </div>
    </NModal>
  </div>
</template>

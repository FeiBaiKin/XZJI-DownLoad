<script setup lang="ts">
// 首页：command-bar 搜索 + 结果详情卡 + 精选资源 + 侧栏（用户卡强调）
import { ref, computed, onMounted, watch, type Component } from 'vue'
import { useRouter, useRoute } from 'vue-router'
import { NButton, NIcon, NModal, NTag, NEmpty, NSkeleton, NAvatar } from 'naive-ui'
import {
  PhMagnifyingGlass, PhCopy, PhArrowSquareOut, PhLock, PhCrown, PhWarningCircle, PhCheck, PhX,
  PhClock, PhTrendUp, PhCaretRight, PhSparkle,
  PhSquaresFour, PhLifebuoy, PhQuestion, PhCompass, PhArrowsClockwise, PhArrowUpRight, PhClockCounterClockwise,
  PhUser, PhCoin, PhDownloadSimple, PhHeart,
} from '@phosphor-icons/vue'
import api from '@/api'
import { contentApi, type PublicKeyword, type PaginatedKeywords, type Announcement, type Category } from '@/api/content'
import { economyApi } from '@/api/economy'
import { useAuthStore } from '@/stores/auth'
import { useAuthModalStore } from '@/stores/authModal'
import { useToastStore } from '@/stores/toast'
import { useFavoritesStore } from '@/stores/favorites'
import { blockAccess } from '@/utils/blockAccess'
import { TIER_FALLBACK_ICON } from '@/utils/tier'
import { scrollToElement } from '@/utils/motion'
import type { KeywordBlock, KeywordAttribute } from '@/types/api'
import ResourceCard from '@/components/ResourceCard.vue'
import AppPagination from '@/components/AppPagination.vue'
import DOMPurify from 'dompurify'

const auth = useAuthStore()
const authModal = useAuthModalStore()
const toast = useToastStore()
const router = useRouter()
const route = useRoute()
const favorites = useFavoritesStore()

// 支付返回检测：以自家订单状态为准。支付宝同步跳转不带 trade_status，
// 且充值与会员直付共用 payment_orders，需按 orderType 区分到账文案。
async function checkPaymentReturn() {
  const q = route.query
  if (q.from !== 'alipay' && q.from !== 'epay') return
  const orderNo = typeof q.out_trade_no === 'string' ? q.out_trade_no : ''
  const failedStatus = typeof q.trade_status === 'string' && q.trade_status !== 'TRADE_SUCCESS' && q.trade_status !== 'SUCCESS'
  router.replace({ query: {} })
  if (failedStatus) {
    toast.error(`支付状态：${q.trade_status}`)
    return
  }
  if (!orderNo) {
    toast.info('正在处理支付结果，请稍后查看账户状态')
    return
  }
  // 异步回调可能略晚于同步跳转，短轮询几次再下结论
  for (let i = 0; i < 4; i++) {
    try {
      const st = await economyApi.rechargeStatus(orderNo)
      if (st.status === 'paid') {
        await auth.fetchProfile()
        toast.success(st.orderType === 'membership' ? '支付成功！会员已开通' : '支付成功！余额已到账')
        return
      }
    } catch { break }
    await new Promise((resolve) => setTimeout(resolve, 1500))
  }
  toast.info('正在处理支付结果，请稍后查看账户状态')
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
const featuredPageSize = ref(8)
const featuredItems = ref<PublicKeyword[]>([])
const featuredTotal = ref(0)
const featuredPage = ref(1)
const featuredLoading = ref(true)

// 分类
const categories = ref<Category[]>([])
const selectedCategory = ref<number | null>(null)

// 侧边栏数据
const recentKeywords = ref<PublicKeyword[]>([])
const sidebarLoading = ref(true)
const downloadsToday = ref<number | null>(null)
const keywordTotal = ref(0)
const discoveryKeywords = ref<PublicKeyword[]>([])
const discoveryLoading = ref(true)
const discoveryBatch = ref(0)
let discoveryPage = 0

// 公告
const visibleAnn = ref<Announcement | null>(null)

const inputRef = ref<HTMLInputElement | null>(null)

const quickActions: { action: string; label: string; icon: Component }[] = [
  { action: 'search', label: '搜索资源', icon: PhMagnifyingGlass },
  { action: 'categories', label: '分类浏览', icon: PhSquaresFour },
  { action: 'help', label: '帮助中心', icon: PhLifebuoy },
  { action: 'about', label: '关于本站', icon: PhQuestion },
]

const isLoggedIn = computed(() => auth.isLoggedIn)

const isMember = computed(() => {
  const u = auth.user
  return !!u?.membership_level && !!u?.membership_expires_at && new Date(u.membership_expires_at) > new Date()
})

const currentBalance = computed(() => auth.user?.balance ?? 0)

// 等级徽章优先显示后台配置的等级名称，未配置时才回退 Lv.N
const levelBadge = computed(() => auth.user?.level_label?.trim() || `Lv.${auth.user?.level ?? 1}`)

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

// 结果卡属性最多展示 4 个，避免 chip 噪声
const visibleResultAttributes = computed(() => resultAttributes.value.slice(0, 4))

function clearResult() {
  result.value = null
  error.value = ''
  purchasedLinks.value = {}
  purchasedCodes.value = {}
}

// 公告正文
const annContent = computed(() => DOMPurify.sanitize(visibleAnn.value?.content || '暂无内容'))
const annTagType = computed(() => {
  const t = visibleAnn.value?.type
  return t === 'warning' ? 'warning' : t === 'success' ? 'success' : 'info'
})
const annTypeLabels: Record<string, string> = { info: '通知', warning: '重要', success: '活动' }


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

// 弹窗登录后补拉收藏与下载数
watch(isLoggedIn, (v) => {
  if (v) {
    favorites.fetchIds()
    fetchDownloadsToday()
  } else {
    favorites.clear()
    downloadsToday.value = null
  }
})

async function fetchFeatured() {
  featuredLoading.value = true
  try {
    let data: PaginatedKeywords
    if (selectedCategory.value) {
      data = await contentApi.public.keywords(selectedCategory.value, featuredPage.value, featuredPageSize.value)
    } else {
      data = await contentApi.public.featured(4, featuredPage.value, featuredPageSize.value)
    }
    featuredItems.value = data?.items ?? []
    featuredTotal.value = data?.total ?? 0
  } catch {
    featuredItems.value = []
    featuredTotal.value = 0
    toast.error('精选资源加载失败，请稍后重试')
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
    keywordTotal.value = data?.total ?? 0
  } catch {
    recentKeywords.value = []
    keywordTotal.value = 0
  } finally {
    sidebarLoading.value = false
  }
  await refreshDiscovery()
}

// 从整个关键词库中随机抽取一页，让侧边栏随内容规模增长而更有探索价值
async function refreshDiscovery() {
  discoveryLoading.value = true
  try {
    if (keywordTotal.value <= 0) {
      discoveryKeywords.value = []
      return
    }
    const pageSize = 3
    const maxPage = Math.max(1, Math.ceil(keywordTotal.value / pageSize))
    let nextPage = Math.floor(Math.random() * maxPage) + 1
    if (maxPage > 1 && nextPage === discoveryPage) nextPage = (nextPage % maxPage) + 1
    discoveryPage = nextPage
    const data = await contentApi.public.keywords(undefined, nextPage, pageSize)
    discoveryKeywords.value = data?.items ?? []
    discoveryBatch.value += 1
  } catch {
    if (discoveryKeywords.value.length === 0) {
      discoveryKeywords.value = recentKeywords.value.slice(0, 3)
      discoveryBatch.value += 1
    }
  } finally {
    discoveryLoading.value = false
  }
}

async function fetchDownloadsToday() {
  try {
    // 会话失效时 401 不跳登录页
    const res = await api.get('/user/downloads-today', { skipAuthRedirect: true })
    downloadsToday.value = res.data?.downloads ?? null
  } catch { downloadsToday.value = null }
}

async function handleFeaturedPageSizeChange(size: number) {
  featuredPageSize.value = size
  featuredPage.value = 1
  await fetchFeatured()
}

async function handleFeaturedPageChange(p: number) {
  featuredPage.value = p
  await fetchFeatured()
  scrollToElement('featured')
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

// 搜索请求代次，避免快速连搜时乱序覆盖
let searchSeq = 0

// ── 搜索 ──
async function handleSearch(query?: string) {
  const q = (query || keyword.value).trim()
  if (!q) { error.value = '请输入关键词'; return }
  const seq = ++searchSeq
  loading.value = true
  error.value = ''
  result.value = null
  purchasedLinks.value = {}
  purchasedCodes.value = {}
  keyword.value = q
  // 更新搜索历史：重复搜索也移动到最前，保持“最近”语义准确
  searchHistory.value = [q, ...searchHistory.value.filter(item => item !== q)].slice(0, 10)
  localStorage.setItem('search_history', JSON.stringify(searchHistory.value))
  try {
    const data = await contentApi.public.search(q)
    if (seq !== searchSeq) return
    if (typeof data.response === 'string') {
      try { data.response = JSON.parse(data.response) } catch { /* 纯文本 */ }
    }
    result.value = data
    if (isLoggedIn.value && data.keywordId && Array.isArray(data.response)) {
      const paid: Record<string, string> = {}
      const paidCodes: Record<string, string> = {}
      const tasks: Promise<void>[] = []
      for (let i = 0; i < data.response.length; i++) {
        const block = data.response[i]
        if (block.type !== 'link') continue
        const access = blockAccess(block)
        if (access === 'free') continue
        const idx = i
        tasks.push(
          api.get('/user/purchase-status', {
            params: { keywordId: data.keywordId, blockIndex: idx },
            skipAuthRedirect: true,
          }).then((sRes) => {
            if (sRes.data?.purchased) {
              paid[`${data.keywordId}-${idx}`] = sRes.data?.url || ''
              paidCodes[`${data.keywordId}-${idx}`] = sRes.data?.code || ''
            }
          }).catch(() => { /* ignore */ }),
        )
      }
      await Promise.all(tasks)
      if (seq !== searchSeq) return
      purchasedLinks.value = paid
      purchasedCodes.value = paidCodes
    }
  } catch (err: unknown) {
    if (seq !== searchSeq) return
    const { extractErrorMessage } = await import('@/types/error')
    error.value = extractErrorMessage(err, '搜索失败，请重试')
  } finally {
    if (seq === searchSeq) loading.value = false
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

// ── 精选卡片：复制关键词后回填搜索框，方便直接搜索 ──
function handleKeywordCopy(kw: PublicKeyword) {
  keyword.value = kw.keyword
  inputRef.value?.focus()
}

async function handleSidebarSearch(query: string) {
  scrollToElement('search-panel')
  await handleSearch(query)
}

function clearSearchHistory() {
  searchHistory.value = []
  localStorage.removeItem('search_history')
}

// ── 链接访问控制 ──
function blockCode(block: KeywordBlock, i: number): string {
  if (blockAccess(block) === 'free') return block.code || ''
  const key = `${result.value?.keywordId}-${i}`
  return purchasedCodes.value[key] || ''
}

function linkIconOf(block: KeywordBlock): Component {
  const access = blockAccess(block)
  if (access === 'membership') return TIER_FALLBACK_ICON
  if (access === 'balance') return PhLock
  return PhArrowSquareOut
}

function isSafeHttpUrl(url?: string): boolean {
  if (!url) return false
  try {
    const u = new URL(url, window.location.origin)
    return u.protocol === 'http:' || u.protocol === 'https:'
  } catch {
    return false
  }
}

function handleLinkClick(e: Event, block: KeywordBlock, idx: number) {
  const access = blockAccess(block)
  if (access === 'free') {
    if (!isSafeHttpUrl(block.url)) {
      e.preventDefault()
      toast.error('链接地址无效')
    }
    return
  }
  e.preventDefault()
  if (!isLoggedIn.value) { authModal.show('login'); return }
  if (!result.value?.keywordId) return
  const key = `${result.value.keywordId}-${idx}`
  if (purchasedLinks.value[key]) {
    const url = purchasedLinks.value[key]
    if (isSafeHttpUrl(url)) window.open(url, '_blank', 'noopener,noreferrer')
    return
  }
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
    if (isSafeHttpUrl(data.url)) {
      window.open(data.url, '_blank', 'noopener,noreferrer')
    } else {
      toast.success(data.message || '购买成功')
    }
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
      scrollToElement('search-panel')
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
    <div class="pt-8 sm:pt-12 lg:grid lg:grid-cols-[minmax(0,1fr)_320px] lg:gap-8">
      <!-- ── 左栏：主内容 ── -->
      <div class="min-w-0">

        <!-- ① Command-bar 搜索 -->
        <section id="search-panel" class="ui-enter scroll-mt-24">
          <div
            class="group flex h-14 items-center gap-3 rounded-2xl border border-zinc-200 bg-white pl-4 pr-2 shadow-[0_1px_0_rgb(0_0_0_/_0.02)] motion-safe:transition-all focus-within:border-emerald-500 focus-within:shadow-[0_0_0_4px_rgb(16_185_129_/_0.12)] dark:border-zinc-800 dark:bg-zinc-900 dark:focus-within:border-emerald-400 dark:focus-within:shadow-[0_0_0_4px_rgb(52_211_153_/_0.12)]"
          >
            <span
              class="grid size-9 shrink-0 place-items-center rounded-xl bg-emerald-500/10 text-emerald-600 dark:bg-emerald-400/10 dark:text-emerald-400"
              aria-hidden="true"
            >
              <NIcon :size="18"><PhMagnifyingGlass /></NIcon>
            </span>
            <input
              ref="inputRef"
              v-model="keyword"
              type="text"
              class="h-full min-w-0 flex-1 bg-transparent text-[15px] text-zinc-900 outline-none placeholder:text-zinc-400 dark:text-zinc-100 dark:placeholder:text-zinc-500"
              placeholder="搜索资源、工具或开源项目..."
              aria-label="输入搜索关键词"
              :disabled="loading"
              autocomplete="off"
              spellcheck="false"
              @keypress.enter="handleSearch()"
            />
            <button
              v-if="keyword && !loading"
              type="button"
              class="grid size-8 shrink-0 place-items-center rounded-lg text-zinc-400 motion-safe:transition-colors hover:bg-zinc-100 hover:text-zinc-700 focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-emerald-500/60 dark:hover:bg-zinc-800 dark:hover:text-zinc-200 dark:focus-visible:ring-emerald-400/60"
              aria-label="清除输入"
              @click="keyword = ''; clearResult()"
            >
              <NIcon :size="14"><PhX /></NIcon>
            </button>
            <span
              class="mr-1 hidden shrink-0 items-center gap-1 rounded-md border border-zinc-200 bg-zinc-50 px-1.5 py-0.5 font-mono text-[10px] text-zinc-500 sm:inline-flex dark:border-zinc-700 dark:bg-zinc-800/70 dark:text-zinc-400"
              aria-hidden="true"
            >Enter</span>
            <NButton
              type="primary"
              aria-label="搜索"
              :loading="loading"
              :disabled="!keyword.trim()"
              @click="handleSearch()"
            >搜索</NButton>
          </div>

          <!-- 热门 / 提示 / 错误：三选一，控制视觉密度 -->
          <div class="mt-3 flex min-h-[24px] items-center gap-2 pl-1">
            <p v-if="error" class="text-xs text-red-500 dark:text-red-400">{{ error }}</p>
            <template v-else-if="hotKeywords.length > 0 && !result">
              <span class="flex shrink-0 items-center gap-1 text-[11px] text-zinc-400 dark:text-zinc-500">
                <NIcon :size="12"><PhTrendUp weight="thin" /></NIcon>
                热门
              </span>
              <div class="flex min-w-0 flex-wrap gap-1.5">
                <button
                  v-for="kw in hotKeywords"
                  :key="kw"
                  type="button"
                  class="rounded-full border border-zinc-200 px-2.5 py-0.5 text-[12px] text-zinc-600 motion-safe:transition-colors hover:border-emerald-400/60 hover:text-emerald-600 focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-emerald-500/60 dark:border-zinc-800 dark:text-zinc-400 dark:hover:border-emerald-400/50 dark:hover:text-emerald-400 dark:focus-visible:ring-emerald-400/60"
                  @click="handleSearch(kw)"
                >{{ kw }}</button>
              </div>
            </template>
            <p v-else class="text-[11px] text-zinc-400 dark:text-zinc-500">
              请输入软件名称、版本或功能关键词，回车即可搜索。
            </p>
          </div>
        </section>

        <!-- ② 搜索结果详情卡 -->
        <Transition name="result-reveal" mode="out-in">
          <section
            v-if="result"
            :key="result.keyword"
            class="mt-6 overflow-hidden rounded-2xl border border-zinc-200 bg-white dark:border-zinc-800 dark:bg-zinc-900"
          >
            <div class="p-5 sm:p-6">
              <div class="flex items-start gap-4">
                <div
                  class="grid size-14 shrink-0 place-items-center overflow-hidden rounded-2xl bg-emerald-500/10 text-base font-bold text-emerald-700 dark:text-emerald-400"
                >
                  <img
                    v-if="result.icon"
                    :src="result.icon"
                    :alt="result.keyword"
                    class="size-full object-cover"
                    loading="lazy"
                  />
                  <span v-else>{{ (result.keyword?.[0] || 'K').toUpperCase() }}</span>
                </div>

                <div class="min-w-0 flex-1">
                  <div class="flex flex-wrap items-center gap-x-2 gap-y-1">
                    <h2 class="text-xl font-bold tracking-tight text-zinc-900 dark:text-zinc-50">
                      {{ result.keyword }}
                    </h2>
                    <span
                      v-if="result.source"
                      class="inline-flex items-center gap-1 rounded-full bg-emerald-50 px-2 py-0.5 text-[11px] font-medium text-emerald-700 dark:bg-emerald-400/10 dark:text-emerald-400"
                    >
                      <NIcon :size="10"><PhCheck /></NIcon>
                      {{ result.source === 'database' ? '已收录' : '默认回复' }}
                    </span>
                  </div>

                  <div
                    v-if="visibleResultAttributes.length > 0"
                    class="mt-2 flex flex-wrap gap-1.5"
                  >
                    <span
                      v-for="(attr, ai) in visibleResultAttributes"
                      :key="ai"
                      class="inline-flex items-center gap-1 rounded-md border border-zinc-200 bg-zinc-50/70 px-1.5 py-0.5 text-[11px] font-medium text-zinc-600 dark:border-zinc-700 dark:bg-zinc-800/60 dark:text-zinc-300"
                    >
                      <span class="text-zinc-400 dark:text-zinc-500">{{ attr.label }}</span>
                      <span>{{ attr.value }}</span>
                    </span>
                  </div>

                  <p
                    class="mt-2 font-mono text-[11px] text-zinc-400 dark:text-zinc-500"
                    :title="result.timestamp"
                  >{{ formatRelative(result.timestamp) }}</p>
                </div>

                <button
                  type="button"
                  class="grid size-8 shrink-0 place-items-center rounded-full text-zinc-400 motion-safe:transition-colors hover:bg-zinc-100 hover:text-zinc-700 focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-emerald-500/60 dark:hover:bg-zinc-800 dark:hover:text-zinc-200 dark:focus-visible:ring-emerald-400/60"
                  aria-label="关闭搜索结果"
                  title="关闭"
                  @click="clearResult"
                >
                  <NIcon :size="14"><PhX /></NIcon>
                </button>
              </div>

              <div class="my-5 border-t border-zinc-100 dark:border-zinc-800" />

              <div v-if="resultBlocks && resultBlocks.length > 0" class="space-y-3">
                <template v-for="(block, i) in resultBlocks" :key="i">
                  <p
                    v-if="block.type === 'text' && block.text"
                    class="text-[14px] leading-relaxed whitespace-pre-wrap text-zinc-700 dark:text-zinc-300"
                  >{{ block.text }}</p>

                  <img
                    v-else-if="block.type === 'image' && block.src"
                    :src="block.src"
                    :alt="block.alt || ''"
                    class="max-h-80 rounded-xl border border-zinc-100 dark:border-zinc-800"
                    loading="lazy"
                  />

                  <a
                    v-else-if="block.type === 'link'"
                    :href="blockAccess(block) === 'free' && isSafeHttpUrl(block.url) ? block.url : 'javascript:void(0)'"
                    target="_blank"
                    rel="noopener noreferrer"
                    class="group flex flex-wrap items-center gap-2.5 rounded-xl border px-3 py-2.5 motion-safe:transition-colors focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-emerald-500/60 dark:focus-visible:ring-emerald-400/60"
                    :class="blockAccess(block) === 'membership'
                      ? 'border-amber-200 bg-amber-50/50 hover:border-amber-300 dark:border-amber-400/20 dark:bg-amber-400/5 dark:hover:border-amber-400/40'
                      : blockAccess(block) === 'balance'
                        ? 'border-zinc-200 bg-white hover:border-emerald-400/70 dark:border-zinc-700 dark:bg-zinc-900 dark:hover:border-emerald-400/60'
                        : 'border-zinc-200 bg-white hover:border-emerald-400 hover:bg-emerald-50/40 dark:border-zinc-700 dark:bg-zinc-900 dark:hover:border-emerald-400/60 dark:hover:bg-emerald-400/5'"
                    @click="(e) => handleLinkClick(e, block, i)"
                  >
                    <span
                      class="grid size-8 shrink-0 place-items-center rounded-lg"
                      :class="blockAccess(block) === 'membership'
                        ? 'bg-amber-100 text-amber-600 dark:bg-amber-400/10 dark:text-amber-400'
                        : 'bg-emerald-50 text-emerald-600 dark:bg-emerald-400/10 dark:text-emerald-400'"
                    >
                      <NIcon :size="16"><component :is="linkIconOf(block)" /></NIcon>
                    </span>
                    <span class="min-w-0 flex-1 truncate text-[13.5px] font-medium text-zinc-800 group-hover:text-zinc-950 dark:text-zinc-200 dark:group-hover:text-white">
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
                      class="flex items-center gap-1 rounded-md border border-dashed px-2 py-1 font-mono text-[11px] motion-safe:transition-colors focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-emerald-500/60 dark:focus-visible:ring-emerald-400/60"
                      :class="copiedIdx === i
                        ? 'border-emerald-400 text-emerald-600 dark:text-emerald-400'
                        : 'border-zinc-300 text-zinc-500 hover:border-zinc-400 hover:text-zinc-700 dark:border-zinc-600 dark:text-zinc-400 dark:hover:text-zinc-200'"
                      title="点击复制提取码"
                      @click.prevent.stop="handleCopyCode(blockCode(block, i), i)"
                    >
                      <NIcon :size="11"><PhCheck v-if="copiedIdx === i" /><PhCopy v-else /></NIcon>
                      {{ copiedIdx === i ? '已复制' : `提取码 ${blockCode(block, i)}` }}
                    </button>
                  </a>
                </template>
              </div>
              <p
                v-else
                class="whitespace-pre-wrap text-[14px] leading-relaxed text-zinc-700 dark:text-zinc-300"
              >{{ result.response }}</p>
            </div>
          </section>
        </Transition>

        <!-- ③ 精选资源 -->
        <section
          id="featured"
          class="ui-enter mt-10 scroll-mt-20 [--enter-delay:120ms]"
        >
          <div class="mb-4 flex items-end justify-between gap-4">
            <div class="min-w-0">
              <h2 class="flex items-center gap-2 text-lg font-bold tracking-tight text-zinc-900 dark:text-zinc-50">
                <NIcon :size="18" class="text-emerald-600 dark:text-emerald-400"><PhSparkle /></NIcon>
                精选资源
              </h2>
              <p v-if="!featuredLoading" class="mt-1 font-mono text-[11px] text-zinc-500 dark:text-zinc-400">
                共 {{ featuredTotal }} 个
              </p>
            </div>
            <router-link
              to="/categories"
              class="inline-flex shrink-0 items-center gap-1 rounded-md px-2 py-1 text-xs text-zinc-500 motion-safe:transition-colors hover:text-emerald-600 focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-emerald-500/60 dark:text-zinc-400 dark:hover:text-emerald-400 dark:focus-visible:ring-emerald-400/60"
            >
              查看更多 <NIcon :size="13"><PhCaretRight /></NIcon>
            </router-link>
          </div>

          <!-- 分类 Tab：细边 ghost + 选中 solid -->
          <div v-if="categories.length > 0" class="mb-5 flex flex-wrap gap-1.5">
            <button
              type="button"
              class="rounded-full px-3 py-1 text-[12.5px] font-medium motion-safe:transition-colors focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-emerald-500/60 dark:focus-visible:ring-emerald-400/60"
              :class="selectedCategory === null
                ? 'bg-emerald-600 text-white shadow-sm dark:bg-emerald-400 dark:text-zinc-950'
                : 'border border-zinc-200 text-zinc-600 hover:border-emerald-400/60 hover:text-emerald-600 dark:border-zinc-800 dark:text-zinc-400 dark:hover:border-emerald-400/50 dark:hover:text-emerald-400'"
              @click="handleCategoryChange(null)"
            >全部</button>
            <button
              v-for="cat in categories"
              :key="cat.id"
              type="button"
              class="rounded-full px-3 py-1 text-[12.5px] font-medium motion-safe:transition-colors focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-emerald-500/60 dark:focus-visible:ring-emerald-400/60"
              :class="selectedCategory === cat.id
                ? 'bg-emerald-600 text-white shadow-sm dark:bg-emerald-400 dark:text-zinc-950'
                : 'border border-zinc-200 text-zinc-600 hover:border-emerald-400/60 hover:text-emerald-600 dark:border-zinc-800 dark:text-zinc-400 dark:hover:border-emerald-400/50 dark:hover:text-emerald-400'"
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
              <div class="flex items-center gap-2.5">
                <NSkeleton width="40" height="40" :sharp="false" />
                <NSkeleton text width="60%" />
              </div>
              <div class="mt-3 space-y-2">
                <NSkeleton text width="100%" />
                <NSkeleton text width="70%" />
              </div>
            </div>
          </div>

          <!-- 空状态 -->
          <div
            v-else-if="featuredItems.length === 0"
            class="rounded-2xl border border-dashed border-zinc-300 py-14 dark:border-zinc-700"
          >
            <NEmpty description="暂无精选资源">
              <template #extra>
                <span class="text-xs text-zinc-500 dark:text-zinc-400">
                  在管理后台添加关键词与分类后将在此展示
                </span>
              </template>
            </NEmpty>
          </div>

          <!-- 资源卡片网格 -->
          <TransitionGroup
            v-else
            name="resource-list"
            tag="div"
            class="grid gap-3 sm:grid-cols-2 sm:gap-4"
            appear
          >
            <ResourceCard
              v-for="(kw, index) in featuredItems"
              :key="kw.id"
              :keyword="kw"
              :category-name="categories.find((c: Category) => c.id === selectedCategory)?.name"
              :favorited="favorites.has(kw.id)"
              :style="{ '--item-index': index }"
              @copy="handleKeywordCopy"
              @toggle-favorite="favorites.toggle(kw.id)"
            />
          </TransitionGroup>

          <!-- 分页 -->
          <div v-if="!featuredLoading && featuredTotal > 0" class="mt-6">
            <AppPagination
              :page="featuredPage"
              :page-size="featuredPageSize"
              :item-count="featuredTotal"
              :page-sizes="[8, 12, 20, 50]"
              @update:page="handleFeaturedPageChange"
              @update:page-size="handleFeaturedPageSizeChange"
            />
          </div>
        </section>
      </div>

      <!-- ── 右栏：侧边栏 ── -->
      <aside class="sidebar-stack mt-10 space-y-4 lg:mt-0">
        <!-- ① 用户卡：emerald tint 强调 -->
        <div
          class="user-card overflow-hidden rounded-2xl border border-emerald-500/20 bg-gradient-to-br from-emerald-50/80 via-white to-white dark:border-emerald-400/20 dark:from-emerald-400/10 dark:via-zinc-900 dark:to-zinc-900"
        >
          <div class="px-5 pb-4 pt-5">
            <div class="flex items-center gap-3">
              <NAvatar
                v-if="isLoggedIn"
                :src="auth.user?.avatar || undefined"
                :size="48"
                round
                class="shrink-0 ring-2 ring-white dark:ring-zinc-900"
              />
              <span
                v-else
                class="grid size-12 shrink-0 place-items-center rounded-full bg-white text-zinc-500 shadow-sm ring-1 ring-emerald-500/20 dark:bg-zinc-900 dark:text-zinc-400 dark:ring-emerald-400/20"
              >
                <NIcon :size="22"><PhUser /></NIcon>
              </span>
              <div class="min-w-0 flex-1">
                <div class="flex items-center gap-2">
                  <span class="truncate text-[15px] font-semibold text-zinc-900 dark:text-zinc-50">
                    {{ isLoggedIn ? (auth.user?.nickname || auth.user?.username) : '游客' }}
                  </span>
                  <span
                    v-if="isLoggedIn && auth.user?.level"
                    class="max-w-24 shrink-0 truncate rounded-full bg-emerald-600/10 px-1.5 py-0.5 text-[10px] font-semibold text-emerald-700 dark:bg-emerald-400/15 dark:text-emerald-400"
                    :class="{ 'font-mono': !auth.user.level_label }"
                    :title="levelBadge"
                  >{{ levelBadge }}</span>
                </div>
                <p class="mt-0.5 text-xs text-zinc-500 dark:text-zinc-400">
                  {{ isLoggedIn ? (isMember ? '会员用户' : '普通用户') : '登录后可购买与下载资源' }}
                </p>
              </div>
            </div>
            <p
              v-if="isLoggedIn && isMember && auth.user?.membership_expires_at"
              class="mt-3 rounded-lg bg-amber-50 px-2.5 py-1.5 text-[11px] text-amber-700 dark:bg-amber-400/10 dark:text-amber-400"
            >
              会员有效期至 {{ new Date(auth.user.membership_expires_at).toLocaleDateString('zh-CN') }}
            </p>
          </div>

          <!-- 3 列统计 -->
          <div class="grid grid-cols-3 border-t border-emerald-500/15 bg-white/40 dark:border-emerald-400/15 dark:bg-zinc-900/40">
            <div class="flex flex-col items-center gap-0.5 py-3">
              <span class="flex items-center gap-1 text-[10px] uppercase tracking-wide text-zinc-500 dark:text-zinc-400">
                <NIcon :size="11"><PhCoin /></NIcon>余额
              </span>
              <span class="font-mono text-sm font-bold text-zinc-800 dark:text-zinc-100">
                {{ isLoggedIn ? currentBalance.toFixed(2) : '-' }}
              </span>
            </div>
            <div class="flex flex-col items-center gap-0.5 border-x border-emerald-500/15 py-3 dark:border-emerald-400/15">
              <span class="flex items-center gap-1 text-[10px] uppercase tracking-wide text-zinc-500 dark:text-zinc-400">
                <NIcon :size="11"><PhDownloadSimple /></NIcon>下载
              </span>
              <span class="font-mono text-sm font-bold text-zinc-800 dark:text-zinc-100">
                {{ isLoggedIn ? (downloadsToday !== null ? downloadsToday : '-') : '-' }}
              </span>
            </div>
            <div class="flex flex-col items-center gap-0.5 py-3">
              <span class="flex items-center gap-1 text-[10px] uppercase tracking-wide text-zinc-500 dark:text-zinc-400">
                <NIcon :size="11"><PhHeart /></NIcon>收藏
              </span>
              <span class="font-mono text-sm font-bold text-zinc-800 dark:text-zinc-100">
                {{ isLoggedIn ? favorites.favoritedIds.size : '-' }}
              </span>
            </div>
          </div>

          <!-- 未登录：登录/注册 -->
          <div
            v-if="!isLoggedIn"
            class="border-t border-emerald-500/15 bg-white/60 px-4 py-3 dark:border-emerald-400/15 dark:bg-zinc-900/60"
          >
            <div class="flex gap-2">
              <NButton type="primary" size="small" block @click="authModal.show('login')">登录</NButton>
              <NButton secondary size="small" block @click="authModal.show('register')">注册</NButton>
            </div>
          </div>
        </div>

        <!-- ② 灵感漫游 -->
        <div class="sidebar-card overflow-hidden rounded-2xl border border-zinc-200 bg-white dark:border-zinc-800 dark:bg-zinc-900">
          <div class="flex items-center justify-between px-4 py-3">
            <div class="flex items-center gap-2">
              <NIcon :size="14" class="text-emerald-600 dark:text-emerald-400"><PhCompass /></NIcon>
              <h3 class="text-[13.5px] font-semibold text-zinc-900 dark:text-zinc-100">灵感漫游</h3>
            </div>
            <button
              type="button"
              class="sidebar-action discovery-refresh grid size-7 shrink-0 place-items-center rounded-full text-zinc-500 focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-emerald-500/60 dark:text-zinc-400 dark:focus-visible:ring-emerald-400/60"
              :class="{ 'is-loading': discoveryLoading }"
              :disabled="discoveryLoading"
              title="换一组关键词"
              aria-label="换一组关键词"
              @click="refreshDiscovery"
            >
              <NIcon :size="13"><PhArrowsClockwise /></NIcon>
            </button>
          </div>

          <Transition name="discovery-swap" mode="out-in">
            <div
              v-if="discoveryLoading && discoveryKeywords.length === 0"
              key="loading"
              class="space-y-3 border-t border-zinc-100 px-4 py-3 dark:border-zinc-800"
            >
              <div v-for="i in 3" :key="i" class="flex items-center gap-3">
                <NSkeleton circle width="32px" height="32px" />
                <div class="flex-1 space-y-1.5">
                  <NSkeleton text width="62%" />
                  <NSkeleton text width="88%" />
                </div>
              </div>
            </div>
            <div
              v-else-if="discoveryKeywords.length > 0"
              :key="discoveryBatch"
              class="sidebar-list divide-y divide-zinc-100 border-t border-zinc-100 dark:divide-zinc-800 dark:border-zinc-800"
            >
              <button
                v-for="(kw, index) in discoveryKeywords"
                :key="kw.id"
                type="button"
                class="sidebar-row group flex w-full items-center gap-3 px-4 py-2.5 text-left focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-inset focus-visible:ring-emerald-500/60 dark:focus-visible:ring-emerald-400/60"
                :title="`搜索「${kw.keyword}」`"
                @click="handleSidebarSearch(kw.keyword)"
              >
                <span class="grid size-8 shrink-0 place-items-center overflow-hidden rounded-lg bg-zinc-100 font-mono text-[10px] font-semibold text-zinc-500 dark:bg-zinc-800 dark:text-zinc-400">
                  <img v-if="kw.icon" :src="kw.icon" alt="" class="size-full object-cover" loading="lazy" />
                  <span v-else>0{{ index + 1 }}</span>
                </span>
                <span class="min-w-0 flex-1">
                  <span class="block truncate text-[13px] font-medium text-zinc-800 dark:text-zinc-200">{{ kw.keyword }}</span>
                  <span class="mt-0.5 block truncate text-[11px] text-zinc-500 dark:text-zinc-400">{{ kw.description || '点击查看相关资源' }}</span>
                </span>
                <NIcon :size="13" class="sidebar-row__arrow shrink-0 text-zinc-300 dark:text-zinc-600">
                  <PhArrowUpRight weight="thin" />
                </NIcon>
              </button>
            </div>
            <div
              v-else
              key="empty"
              class="border-t border-zinc-100 px-4 py-5 text-center text-xs text-zinc-400 dark:border-zinc-800 dark:text-zinc-500"
            >
              暂无可探索的关键词
            </div>
          </Transition>
        </div>

        <!-- ③ 最近搜索 -->
        <div class="sidebar-card overflow-hidden rounded-2xl border border-zinc-200 bg-white dark:border-zinc-800 dark:bg-zinc-900">
          <div class="flex items-center justify-between px-4 py-3">
            <div class="flex items-center gap-2">
              <NIcon :size="14" class="text-emerald-600 dark:text-emerald-400"><PhClockCounterClockwise /></NIcon>
              <h3 class="text-[13.5px] font-semibold text-zinc-900 dark:text-zinc-100">最近搜索</h3>
            </div>
            <button
              v-if="searchHistory.length > 0"
              type="button"
              class="sidebar-action rounded-md px-2 py-1 text-[11px] text-zinc-500 focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-emerald-500/60 dark:text-zinc-400 dark:focus-visible:ring-emerald-400/60"
              @click="clearSearchHistory"
            >清除</button>
          </div>
          <TransitionGroup
            v-if="searchHistory.length > 0"
            name="history-list"
            tag="div"
            class="sidebar-list divide-y divide-zinc-100 border-t border-zinc-100 dark:divide-zinc-800 dark:border-zinc-800"
          >
            <button
              v-for="item in searchHistory.slice(0, 5)"
              :key="item"
              type="button"
              class="sidebar-row group flex w-full items-center gap-3 px-4 py-2.5 text-left focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-inset focus-visible:ring-emerald-500/60 dark:focus-visible:ring-emerald-400/60"
              @click="handleSidebarSearch(item)"
            >
              <span class="grid size-8 shrink-0 place-items-center rounded-lg bg-zinc-100 text-zinc-500 dark:bg-zinc-800 dark:text-zinc-400">
                <NIcon :size="13"><PhMagnifyingGlass /></NIcon>
              </span>
              <span class="min-w-0 flex-1 truncate text-[13px] font-medium text-zinc-800 dark:text-zinc-200">
                {{ item }}
              </span>
              <NIcon :size="13" class="sidebar-row__arrow shrink-0 text-zinc-300 dark:text-zinc-600">
                <PhArrowUpRight weight="thin" />
              </NIcon>
            </button>
          </TransitionGroup>
          <div
            v-else
            class="border-t border-zinc-100 px-4 py-5 text-center text-xs text-zinc-400 dark:border-zinc-800 dark:text-zinc-500"
          >
            搜索过的关键词会显示在这里
          </div>
        </div>

        <!-- ④ 最新更新 -->
        <div class="sidebar-card overflow-hidden rounded-2xl border border-zinc-200 bg-white dark:border-zinc-800 dark:bg-zinc-900">
          <div class="flex items-center justify-between px-4 py-3">
            <div class="flex items-center gap-2">
              <NIcon :size="14" class="text-emerald-600 dark:text-emerald-400"><PhClock /></NIcon>
              <h3 class="text-[13.5px] font-semibold text-zinc-900 dark:text-zinc-100">最新更新</h3>
            </div>
            <router-link
              to="/categories"
              class="sidebar-action inline-flex items-center gap-1 rounded-md px-2 py-1 text-[11px] text-zinc-500 focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-emerald-500/60 dark:text-zinc-400 dark:focus-visible:ring-emerald-400/60"
            >更多 <NIcon :size="11"><PhCaretRight /></NIcon></router-link>
          </div>

          <div
            v-if="sidebarLoading"
            class="space-y-3 border-t border-zinc-100 px-4 py-3 dark:border-zinc-800"
          >
            <div v-for="i in 4" :key="i" class="flex items-center gap-3">
              <NSkeleton circle width="32px" height="32px" />
              <div class="flex-1 space-y-1.5">
                <NSkeleton text width="62%" />
                <NSkeleton text width="34%" />
              </div>
            </div>
          </div>
          <div
            v-else-if="recentKeywords.length === 0"
            class="border-t border-zinc-100 px-4 py-5 text-center text-xs text-zinc-400 dark:border-zinc-800 dark:text-zinc-500"
          >
            暂无更新
          </div>
          <div
            v-else
            class="sidebar-list divide-y divide-zinc-100 border-t border-zinc-100 dark:divide-zinc-800 dark:border-zinc-800"
          >
            <button
              v-for="(kw, index) in recentKeywords"
              :key="kw.id"
              type="button"
              class="sidebar-row group flex w-full items-center gap-3 px-4 py-2.5 text-left focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-inset focus-visible:ring-emerald-500/60 dark:focus-visible:ring-emerald-400/60"
              @click="handleSidebarSearch(kw.keyword)"
            >
              <span class="grid size-8 shrink-0 place-items-center overflow-hidden rounded-lg bg-zinc-100 font-mono text-[10px] font-semibold text-zinc-500 dark:bg-zinc-800 dark:text-zinc-400">
                <img v-if="kw.icon" :src="kw.icon" alt="" class="size-full object-cover" loading="lazy" />
                <span v-else>0{{ index + 1 }}</span>
              </span>
              <span class="min-w-0 flex-1">
                <span class="block truncate text-[13px] font-medium text-zinc-800 dark:text-zinc-200">{{ kw.keyword }}</span>
                <span class="mt-0.5 block font-mono text-[11px] text-zinc-500 dark:text-zinc-400">
                  {{ formatRelative(kw.updated_at || kw.created_at) }}
                </span>
              </span>
              <NIcon :size="13" class="sidebar-row__arrow shrink-0 text-zinc-300 dark:text-zinc-600">
                <PhArrowUpRight weight="thin" />
              </NIcon>
            </button>
          </div>
        </div>

        <!-- ⑤ 快捷入口：2×2 -->
        <div class="sidebar-card overflow-hidden rounded-2xl border border-zinc-200 bg-white dark:border-zinc-800 dark:bg-zinc-900">
          <div class="flex items-center px-4 py-3">
            <div class="flex items-center gap-2">
              <NIcon :size="14" class="text-emerald-600 dark:text-emerald-400"><PhSquaresFour /></NIcon>
              <h3 class="text-[13.5px] font-semibold text-zinc-900 dark:text-zinc-100">快捷入口</h3>
            </div>
          </div>
          <div class="sidebar-quick-grid grid grid-cols-2 border-t border-zinc-100 dark:border-zinc-800">
            <button
              v-for="item in quickActions"
              :key="item.action"
              type="button"
              class="sidebar-quick-item flex items-center gap-2.5 px-4 py-3 text-left focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-inset focus-visible:ring-emerald-500/60 dark:focus-visible:ring-emerald-400/60"
              @click="handleQuickAction(item.action)"
            >
              <span class="grid size-8 shrink-0 place-items-center rounded-lg bg-zinc-100 text-zinc-500 dark:bg-zinc-800 dark:text-zinc-400">
                <NIcon :size="15"><component :is="item.icon" /></NIcon>
              </span>
              <span class="text-[12.5px] font-medium text-zinc-700 dark:text-zinc-300">{{ item.label }}</span>
            </button>
          </div>
        </div>
      </aside>
    </div>

    <!-- ═══ 公告弹窗 ═══ -->
    <NModal :show="!!visibleAnn" @update:show="(v: boolean) => { if (!v) dismissAnnouncement() }">
      <div v-if="visibleAnn" class="w-[92vw] max-w-lg rounded-2xl border border-zinc-200 bg-white shadow-xl dark:border-zinc-800 dark:bg-zinc-900">
        <div class="flex items-center justify-between gap-3 border-b border-zinc-100 px-6 py-4 dark:border-zinc-800">
          <div class="flex min-w-0 items-center gap-2.5">
            <NTag :type="annTagType" size="small" round class="shrink-0">{{ annTypeLabels[visibleAnn.type] || visibleAnn.type }}</NTag>
            <h3 class="truncate text-base font-semibold text-zinc-900 dark:text-zinc-100">{{ visibleAnn.title }}</h3>
          </div>
          <button
            type="button"
            class="flex size-8 shrink-0 items-center justify-center rounded-full text-zinc-400 motion-safe:transition-colors hover:bg-zinc-100 hover:text-zinc-600 focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-emerald-500/60 dark:hover:bg-zinc-800 dark:hover:text-zinc-300 dark:focus-visible:ring-emerald-400/60"
            aria-label="关闭"
            @click="dismissAnnouncement"
          >
            <NIcon :size="16"><PhX /></NIcon>
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
              <PhCrown weight="duotone" v-if="purchaseDialog.access === 'membership'" /><PhLock v-else />
            </NIcon>
            <h3 class="text-base font-semibold text-zinc-900 dark:text-zinc-100">
              {{ purchaseDialog.access === 'membership' ? '会员专属链接' : '付费链接' }}
            </h3>
          </div>
          <button
            type="button"
            class="flex size-8 items-center justify-center rounded-full text-zinc-400 motion-safe:transition-colors hover:bg-zinc-100 hover:text-zinc-600 focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-emerald-500/60 dark:hover:bg-zinc-800 dark:hover:text-zinc-300 dark:focus-visible:ring-emerald-400/60"
            aria-label="关闭"
            @click="purchaseDialog = null"
          >
            <NIcon :size="16"><PhX /></NIcon>
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
              <NIcon :size="14" class="mt-0.5 shrink-0"><PhCrown weight="duotone" /></NIcon>
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
            <NIcon :size="14"><PhWarningCircle /></NIcon>
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

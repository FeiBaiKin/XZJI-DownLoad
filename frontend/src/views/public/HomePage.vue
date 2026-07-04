<script setup lang="ts">
// 首页：搜索、分类浏览、内容块渲染（含付费链接购买）、公告弹窗、支付回跳提示
// 业务逻辑与后端契约保持不变；UI 重构为 Naive UI + Tailwind
import { ref, computed, onMounted, watch, type Component } from 'vue'
import { useRouter, useRoute } from 'vue-router'
import {
  NButton, NIcon, NDropdown, NModal, NTag, NEmpty, NSkeleton,
} from 'naive-ui'
import {
  Search, Sun, MoonStars, Copy, ExternalLink, Lock, Crown, ArrowRight, AlertCircle, Check, X,
} from '@vicons/tabler'
import api from '@/api'
import { contentApi, type Page, type Category, type Announcement } from '@/api/content'
import { useAuthStore } from '@/stores/auth'
import { useThemeStore } from '@/stores/theme'
import { useToastStore } from '@/stores/toast'
import { blockAccess } from '@/utils/blockAccess'
import type { KeywordBlock } from '@/types/api'
import DOMPurify from 'dompurify'

const auth = useAuthStore()
const theme = useThemeStore()
const toast = useToastStore()
const router = useRouter()
const route = useRoute()

// ── 支付返回检测 ──
function checkPaymentReturn() {
  const q = route.query
  if (!q.from) return
  // 支付宝 / 易支付 同步跳转返回
  if (q.from === 'alipay' || q.from === 'epay') {
    if (q.trade_status === 'TRADE_SUCCESS' || q.trade_status === 'SUCCESS') {
      toast.success('支付成功！余额已到账')
    } else if (q.trade_status) {
      toast.error(`支付状态：${q.trade_status}`)
    } else {
      toast.info('正在处理支付结果，请查看余额确认')
    }
    // 清除 URL 中的支付参数，避免刷新重复弹窗
    router.replace({ query: {} })
  }
}

// ── 状态 ──
interface SearchResult {
  keywordId?: number
  keyword: string
  response: string | KeywordBlock[]
  timestamp: string
  source?: string
}
interface PublicKeyword {
  id: number
  keyword: string
  description?: string
}
interface PurchaseDialogState {
  keywordId: number
  blockIndex: number
  block: KeywordBlock
  cost: number
  access: 'balance' | 'membership'
  key: string
}

const keyword = ref('')
const result = ref<SearchResult | null>(null)
const loading = ref(false)
const error = ref('')
const settings = ref({
  siteName: '',
  siteDescription: '',
  footerText: '',
  siteLogoType: 'text',
  siteLogoUrl: '',
})
const copiedIdx = ref<number | null>(null)
const purchaseDialog = ref<PurchaseDialogState | null>(null)
const purchaseLoading = ref(false)
const purchasedLinks = ref<Record<string, string>>({})
const purchasedCodes = ref<Record<string, string>>({})

// 关键词浏览
const keywords = ref<PublicKeyword[]>([])
const keywordsLoading = ref(true)
const categories = ref<Category[]>([])
const selectedCategory = ref<number | null>(null)
const pages = ref<Page[]>([])
const copiedKeywordId = ref<number | null>(null)

// 公告
const visibleAnn = ref<Announcement | null>(null)

const inputRef = ref<HTMLInputElement | null>(null)

const isLoggedIn = computed(() => auth.isLoggedIn)
const isAdmin = computed(() => auth.isAdmin)

// 当前用户是否为有效会员（用于购买弹窗折扣提示）
const isMember = computed(() => {
  const u = auth.user
  return !!u?.membership_level && !!u?.membership_expires_at && new Date(u.membership_expires_at) > new Date()
})

// 当前用户余额
const currentBalance = computed(() => auth.user?.balance ?? 0)

// Logo 首字母
const logoLetter = computed(() => (settings.value.siteName?.[0] || 'X').toUpperCase())

// 搜索结果内容块（数组形式时）
const resultBlocks = computed<KeywordBlock[] | null>(() =>
  Array.isArray(result.value?.response) ? (result.value!.response as KeywordBlock[]) : null
)

// 公告正文（消毒后）
const annContent = computed(() => DOMPurify.sanitize(visibleAnn.value?.content || '暂无内容'))
const annTagType = computed(() => {
  const t = visibleAnn.value?.type
  return t === 'warning' ? 'warning' : t === 'success' ? 'success' : 'info'
})
const annTypeLabels: Record<string, string> = { info: '通知', warning: '重要', success: '活动' }

// ── 数据加载 ──
onMounted(async () => {
  checkPaymentReturn()
  await Promise.all([
    fetchSettings(),
    fetchCategories(),
    fetchKeywords(selectedCategory.value),
    fetchPages(),
    fetchAnnouncements(),
  ])
})

async function fetchSettings() {
  try {
    const data = await contentApi.public.settings()
    if (data) settings.value = { ...settings.value, ...data }
  } catch { /* 忽略，使用默认 */ }
}
async function fetchCategories() {
  try { categories.value = await contentApi.public.categories() } catch { categories.value = [] }
}
async function fetchKeywords(categoryId: number | null) {
  keywordsLoading.value = true
  try { keywords.value = await contentApi.public.keywords(categoryId ?? undefined) } catch { keywords.value = [] }
  finally { keywordsLoading.value = false }
}
async function fetchPages() {
  try { pages.value = await contentApi.public.pages() } catch { pages.value = [] }
}
async function fetchAnnouncements() {
  try {
    const data = await contentApi.public.announcements()
    if (data.length > 0) {
      const dismissed = JSON.parse(localStorage.getItem('dismissed_announcements') || '{}')
      const firstUnseen = data.find((a: Announcement) => !dismissed[a.id])
      if (firstUnseen) {
        visibleAnn.value = firstUnseen
      }
    }
  } catch { /* 忽略 */ }
}

// 分类切换 → 重新拉取关键词
watch(selectedCategory, (id) => { fetchKeywords(id) })

// ── 用户下拉菜单 ──
const userMenuOptions = computed(() => {
  const opts: { label: string; key: string }[] = [{ label: '用户中心', key: 'user' }]
  if (isAdmin.value) opts.push({ label: '管理后台', key: 'admin' })
  opts.push({ label: '退出登录', key: 'logout' })
  return opts
})
function handleUserMenuSelect(key: string) {
  if (key === 'user') router.push('/user')
  else if (key === 'admin') router.push('/admin')
  else if (key === 'logout') {
    auth.logout()
    router.push('/')
  }
}

// ── 复制提取码 ──
function handleCopyCode(code: string, idx: number) {
  navigator.clipboard.writeText(code).then(() => {
    copiedIdx.value = idx
    setTimeout(() => { copiedIdx.value = null }, 1500)
  })
}

// ── 关键词卡片：复制关键词 ──
function handleKeywordClick(kw: PublicKeyword) {
  navigator.clipboard.writeText(kw.keyword).then(() => {
    copiedKeywordId.value = kw.id
    toast.success(`已复制「${kw.keyword}」，粘贴到搜索框获取资源`)
    setTimeout(() => { copiedKeywordId.value = null }, 1200)
    inputRef.value?.focus()
  }).catch(() => toast.error('复制失败，请手动复制'))
}

// ── 链接访问控制（收敛至 utils/blockAccess.ts，与后端 service/content.go 保持一致）──

// 取某块的提取码：免费链接用 response 中的 code；付费链接仅在已购买时用缓存的 code（脱敏后真实 code 不在 response 里）
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
  // 已购买：用缓存的真实 URL 打开（response 里的 url 已脱敏为空）
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
      balanceCost: d.cost,
      access: d.access,
    })
    const data = res.data
    // 购买成功后服务端下发真实 url/code（response 里的 url 已脱敏），缓存并打开
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

// ── 搜索 ──
async function handleSearch() {
  if (!keyword.value.trim()) { error.value = '请输入关键词'; return }

  loading.value = true
  error.value = ''
  result.value = null
  purchasedLinks.value = {}
  purchasedCodes.value = {}

  try {
    const data = await contentApi.public.search(keyword.value.trim())
    // 解析 response：后端返回 JSON 字符串（块数组或纯文本），前端解析以兼容块渲染与纯文本
    if (typeof data.response === 'string') {
      try { data.response = JSON.parse(data.response) } catch { /* 纯文本，保持字符串 */ }
    }
    result.value = data

    // 检查付费链接的购买状态（已购买的服务端下发真实 url/code，缓存以便直接打开）
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

// 时间戳格式化（结果元信息）
function formatTimestamp(ts: string) {
  return new Date(ts).toLocaleString('zh-CN', { hour: '2-digit', minute: '2-digit', second: '2-digit' })
}

// 关闭公告
function dismissAnnouncement() {
  if (visibleAnn.value) {
    const dismissed = JSON.parse(localStorage.getItem('dismissed_announcements') || '{}')
    dismissed[visibleAnn.value.id] = true
    localStorage.setItem('dismissed_announcements', JSON.stringify(dismissed))
  }
  visibleAnn.value = null
}

// 分类标题
const keywordsSectionTitle = computed(() => {
  if (selectedCategory.value) {
    return categories.value.find((c) => c.id === selectedCategory.value)?.name || '分类'
  }
  return '浏览资源'
})
</script>

<template>
  <div class="flex min-h-dvh flex-col">
    <!-- ── Header ── -->
    <header class="sticky top-0 z-30 border-b border-zinc-200/70 bg-white/80 backdrop-blur-md dark:border-zinc-800/70 dark:bg-zinc-950/80">
      <div class="mx-auto flex h-16 max-w-6xl items-center justify-between px-4 sm:px-6">
        <router-link to="/" class="flex items-center gap-2.5">
          <img
            v-if="settings.siteLogoType === 'image' && settings.siteLogoUrl"
            :src="settings.siteLogoUrl"
            :alt="settings.siteName || 'Logo'"
            class="h-8 w-auto"
          />
          <template v-else>
            <span class="grid size-8 place-items-center rounded-lg bg-emerald-600 text-sm font-bold text-white dark:bg-emerald-400 dark:text-zinc-950">
              {{ logoLetter }}
            </span>
            <span class="text-[15px] font-semibold tracking-tight text-zinc-900 dark:text-zinc-100">{{ settings.siteName }}</span>
          </template>
        </router-link>

        <div class="flex items-center gap-2">
          <!-- 主题切换 -->
          <button
            type="button"
            class="flex size-9 items-center justify-center rounded-full text-zinc-500 transition-colors hover:bg-zinc-100 hover:text-zinc-800 dark:text-zinc-400 dark:hover:bg-zinc-800 dark:hover:text-zinc-200"
            :title="theme.isDark ? '切换浅色模式' : '切换深色模式'"
            @click="theme.toggle()"
          >
            <NIcon :size="18">
              <Sun v-if="theme.isDark" />
              <MoonStars v-else />
            </NIcon>
          </button>

          <!-- 已登录：用户下拉 -->
          <NDropdown v-if="isLoggedIn" :options="userMenuOptions" trigger="click" @select="handleUserMenuSelect">
            <button type="button" class="flex items-center rounded-full outline-none focus-visible:ring-2 focus-visible:ring-emerald-500">
              <img
                v-if="auth.user?.avatar"
                :src="auth.user.avatar"
                alt=""
                class="size-9 rounded-full border border-zinc-200 object-cover dark:border-zinc-700"
              />
              <span
                v-else
                class="grid size-9 place-items-center rounded-full bg-zinc-200 text-sm font-semibold text-zinc-600 dark:bg-zinc-800 dark:text-zinc-300"
              >
                {{ (auth.user?.nickname || auth.user?.username || 'U')[0]!.toUpperCase() }}
              </span>
            </button>
          </NDropdown>

          <!-- 未登录：登录 / 注册 -->
          <template v-else>
            <NButton quaternary size="small" @click="router.push('/login')">登录</NButton>
            <NButton type="primary" size="small" @click="router.push('/register')">注册</NButton>
          </template>
        </div>
      </div>
    </header>

    <main class="mx-auto w-full max-w-6xl flex-1 px-4 sm:px-6">
      <!-- ── Hero + 搜索 ── -->
      <section class="pt-14 pb-4 sm:pt-20">
        <h1 class="text-4xl font-bold tracking-tight text-zinc-900 sm:text-5xl dark:text-zinc-50">
          {{ settings.siteName }}
        </h1>
        <p v-if="settings.siteDescription" class="mt-3 max-w-xl text-[15px] leading-relaxed text-zinc-500 dark:text-zinc-400">
          {{ settings.siteDescription }}
        </p>

        <!-- 搜索框 -->
        <div class="mt-8 max-w-2xl">
          <div
            class="flex h-14 items-center gap-3 rounded-2xl border border-zinc-300 bg-white pr-2 pl-4 shadow-sm transition-colors focus-within:border-emerald-500 hover:border-zinc-400 dark:border-zinc-700 dark:bg-zinc-900 dark:focus-within:border-emerald-400 dark:hover:border-zinc-600"
          >
            <NIcon :size="20" class="shrink-0 text-zinc-400"><Search /></NIcon>
            <input
              ref="inputRef"
              v-model="keyword"
              type="text"
              class="h-full min-w-0 flex-1 bg-transparent text-[15px] text-zinc-900 outline-none placeholder:text-zinc-400 dark:text-zinc-100 dark:placeholder:text-zinc-500"
              placeholder="输入关键词，获取对应资源…"
              :disabled="loading"
              autocomplete="off"
              spellcheck="false"
              @keypress.enter="handleSearch"
            />
            <NButton type="primary" size="large" round :loading="loading" :disabled="!keyword.trim()" @click="handleSearch">
              搜索
            </NButton>
          </div>
          <p v-if="error" class="mt-2 pl-1 text-sm text-red-500 dark:text-red-400">{{ error }}</p>
        </div>

        <!-- 分类胶囊 -->
        <div v-if="categories.length > 0" class="mt-6 flex flex-wrap items-center gap-2">
          <button
            type="button"
            class="rounded-full border px-3.5 py-1.5 text-[13px] font-medium transition-colors"
            :class="selectedCategory === null
              ? 'border-emerald-600 bg-emerald-600 text-white dark:border-emerald-400 dark:bg-emerald-400 dark:text-zinc-950'
              : 'border-zinc-300 text-zinc-600 hover:border-zinc-400 hover:text-zinc-900 dark:border-zinc-700 dark:text-zinc-400 dark:hover:border-zinc-500 dark:hover:text-zinc-200'"
            @click="selectedCategory = null"
          >全部</button>
          <button
            v-for="c in categories"
            :key="c.id"
            type="button"
            class="rounded-full border px-3.5 py-1.5 text-[13px] font-medium transition-colors"
            :class="selectedCategory === c.id
              ? 'border-emerald-600 bg-emerald-600 text-white dark:border-emerald-400 dark:bg-emerald-400 dark:text-zinc-950'
              : 'border-zinc-300 text-zinc-600 hover:border-zinc-400 hover:text-zinc-900 dark:border-zinc-700 dark:text-zinc-400 dark:hover:border-zinc-500 dark:hover:text-zinc-200'"
            :title="c.description || c.name"
            @click="selectedCategory = c.id"
          >{{ c.name }}</button>
        </div>
      </section>

      <!-- ── 搜索结果 ── -->
      <section v-if="result" :key="result.timestamp" class="mt-8">
        <div class="overflow-hidden rounded-2xl border border-zinc-200 bg-white dark:border-zinc-800 dark:bg-zinc-900">
          <div class="p-5 sm:p-6">
            <div v-if="resultBlocks" class="space-y-3">
              <template v-for="(block, i) in resultBlocks" :key="i">
                <!-- 文本块 -->
                <p v-if="block.type === 'text' && block.text" class="text-[15px] leading-relaxed whitespace-pre-wrap text-zinc-700 dark:text-zinc-300">
                  {{ block.text }}
                </p>

                <!-- 图片块 -->
                <img
                  v-else-if="block.type === 'image' && block.src"
                  :src="block.src"
                  :alt="block.alt || ''"
                  class="max-h-96 rounded-xl border border-zinc-100 dark:border-zinc-800"
                />

                <!-- 链接块 -->
                <a
                  v-else-if="block.type === 'link'"
                  :href="blockAccess(block) === 'free' ? block.url : 'javascript:void(0)'"
                  target="_blank"
                  rel="noopener noreferrer"
                  class="group flex flex-wrap items-center gap-3 rounded-xl border px-4 py-3 transition-colors"
                  :class="blockAccess(block) !== 'free'
                    ? 'border-amber-200 bg-amber-50/60 hover:border-amber-300 dark:border-amber-400/20 dark:bg-amber-400/5 dark:hover:border-amber-400/40'
                    : 'border-zinc-200 hover:border-emerald-400 dark:border-zinc-700 dark:hover:border-emerald-400/60'"
                  @click="(e) => handleLinkClick(e, block, i)"
                >
                  <span
                    class="grid size-9 shrink-0 place-items-center rounded-lg"
                    :class="blockAccess(block) !== 'free'
                      ? 'bg-amber-100 text-amber-600 dark:bg-amber-400/10 dark:text-amber-400'
                      : 'bg-emerald-50 text-emerald-600 dark:bg-emerald-400/10 dark:text-emerald-400'"
                  >
                    <NIcon :size="18"><component :is="linkIconOf(block)" /></NIcon>
                  </span>
                  <span class="min-w-0 flex-1 truncate text-sm font-medium text-zinc-800 group-hover:text-zinc-950 dark:text-zinc-200 dark:group-hover:text-white">
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
                    class="flex items-center gap-1.5 rounded-md border border-dashed px-2 py-1 font-mono text-xs transition-colors"
                    :class="copiedIdx === i
                      ? 'border-emerald-400 text-emerald-600 dark:text-emerald-400'
                      : 'border-zinc-300 text-zinc-500 hover:border-zinc-400 hover:text-zinc-700 dark:border-zinc-600 dark:text-zinc-400 dark:hover:text-zinc-200'"
                    title="点击复制提取码"
                    @click.prevent.stop="handleCopyCode(blockCode(block, i), i)"
                  >
                    <NIcon :size="12"><Check v-if="copiedIdx === i" /><Copy v-else /></NIcon>
                    {{ copiedIdx === i ? '已复制' : `提取码 ${blockCode(block, i)}` }}
                  </button>
                </a>
              </template>
            </div>
            <p v-else class="text-[15px] leading-relaxed whitespace-pre-wrap text-zinc-700 dark:text-zinc-300">{{ result.response }}</p>
          </div>

          <!-- 结果元信息 -->
          <div class="flex flex-wrap items-center gap-x-4 gap-y-1 border-t border-zinc-100 bg-zinc-50/60 px-5 py-2.5 text-xs text-zinc-400 sm:px-6 dark:border-zinc-800 dark:bg-zinc-950/40 dark:text-zinc-500">
            <span class="font-mono font-medium text-emerald-600 dark:text-emerald-400">{{ result.keyword }}</span>
            <span class="font-mono">{{ formatTimestamp(result.timestamp) }}</span>
            <span v-if="result.source" class="font-mono">{{ result.source }}</span>
          </div>
        </div>
      </section>

      <!-- ── 关键词浏览区 ── -->
      <section class="mt-14">
        <div class="mb-4 flex items-baseline justify-between">
          <h2 class="text-lg font-semibold tracking-tight text-zinc-900 dark:text-zinc-100">{{ keywordsSectionTitle }}</h2>
          <span v-if="!keywordsLoading" class="font-mono text-xs text-zinc-400 dark:text-zinc-500">{{ keywords.length }} 个关键词</span>
        </div>

        <!-- 加载骨架 -->
        <div v-if="keywordsLoading" class="grid gap-3 sm:grid-cols-2 lg:grid-cols-3">
          <div v-for="i in 6" :key="i" class="rounded-xl border border-zinc-200 bg-white p-4 dark:border-zinc-800 dark:bg-zinc-900">
            <NSkeleton text width="55%" />
            <NSkeleton text width="90%" class="mt-2" />
            <NSkeleton text width="35%" class="mt-2" />
          </div>
        </div>

        <!-- 空状态 -->
        <div v-else-if="keywords.length === 0" class="rounded-xl border border-dashed border-zinc-300 py-14 dark:border-zinc-700">
          <NEmpty :description="selectedCategory ? '该分类下暂无关键词' : '暂无关键词'">
            <template #extra>
              <span class="text-xs text-zinc-400 dark:text-zinc-500">
                {{ selectedCategory ? '试试其他分类，或在后台添加关键词' : '在管理后台添加关键词后将在此展示' }}
              </span>
            </template>
          </NEmpty>
        </div>

        <!-- 关键词卡片 -->
        <div v-else class="grid gap-3 sm:grid-cols-2 lg:grid-cols-3">
          <button
            v-for="kw in keywords"
            :key="kw.id"
            type="button"
            class="group rounded-xl border bg-white p-4 text-left transition-all dark:bg-zinc-900"
            :class="copiedKeywordId === kw.id
              ? 'border-emerald-500 ring-1 ring-emerald-500/40 dark:border-emerald-400'
              : 'border-zinc-200 hover:-translate-y-0.5 hover:border-emerald-400/70 hover:shadow-sm dark:border-zinc-800 dark:hover:border-emerald-400/50'"
            title="点击复制关键词"
            @click="handleKeywordClick(kw)"
          >
            <div class="flex items-center justify-between gap-2">
              <span class="truncate text-[15px] font-medium text-zinc-900 dark:text-zinc-100">{{ kw.keyword }}</span>
              <NIcon
                :size="15"
                class="shrink-0 transition-colors"
                :class="copiedKeywordId === kw.id ? 'text-emerald-500' : 'text-zinc-300 group-hover:text-emerald-500 dark:text-zinc-600'"
              >
                <Check v-if="copiedKeywordId === kw.id" /><Copy v-else />
              </NIcon>
            </div>
            <p v-if="kw.description" class="mt-1.5 line-clamp-2 text-[13px] leading-relaxed text-zinc-500 dark:text-zinc-400">{{ kw.description }}</p>
            <p class="mt-2 text-xs" :class="copiedKeywordId === kw.id ? 'text-emerald-600 dark:text-emerald-400' : 'text-zinc-400 dark:text-zinc-600'">
              {{ copiedKeywordId === kw.id ? '已复制' : '点击复制' }}
            </p>
          </button>
        </div>
      </section>

      <!-- ── 自定义页面区 ── -->
      <section v-if="pages.length > 0" class="mt-14 mb-4">
        <h2 class="mb-4 text-lg font-semibold tracking-tight text-zinc-900 dark:text-zinc-100">更多信息</h2>
        <div class="flex flex-wrap gap-2.5">
          <router-link
            v-for="p in pages"
            :key="p.id"
            :to="`/page/${p.slug}`"
            class="group inline-flex items-center gap-1.5 rounded-full border border-zinc-200 bg-white px-4 py-2 text-sm text-zinc-600 transition-colors hover:border-emerald-400/70 hover:text-zinc-900 dark:border-zinc-800 dark:bg-zinc-900 dark:text-zinc-400 dark:hover:border-emerald-400/50 dark:hover:text-zinc-100"
          >
            {{ p.title }}
            <NIcon :size="14" class="text-zinc-300 transition-transform group-hover:translate-x-0.5 group-hover:text-emerald-500 dark:text-zinc-600">
              <ArrowRight />
            </NIcon>
          </router-link>
        </div>
      </section>
    </main>

    <!-- ── Footer ── -->
    <footer class="mt-16 border-t border-zinc-200 py-8 dark:border-zinc-800/80">
      <div class="mx-auto flex max-w-6xl flex-wrap items-center gap-2 px-4 text-[13px] text-zinc-400 sm:px-6 dark:text-zinc-500">
        <span>{{ settings.footerText }}</span>
        <span v-if="settings.footerText" class="text-zinc-300 dark:text-zinc-700">/</span>
        <span>{{ settings.siteName || 'XZJI' }}</span>
      </div>
    </footer>

    <!-- ── 公告弹窗 ── -->
    <NModal :show="!!visibleAnn" @update:show="(v: boolean) => { if (!v) dismissAnnouncement() }">
      <div v-if="visibleAnn" class="w-[92vw] max-w-lg rounded-2xl border border-zinc-200 bg-white shadow-xl dark:border-zinc-800 dark:bg-zinc-900">
        <div class="flex items-center justify-between border-b border-zinc-100 px-6 py-4 dark:border-zinc-800">
          <div class="flex items-center gap-2.5">
            <NTag :type="annTagType" size="small" round>{{ annTypeLabels[visibleAnn.type] || visibleAnn.type }}</NTag>
            <h3 class="text-base font-semibold text-zinc-900 dark:text-zinc-100">{{ visibleAnn.title }}</h3>
          </div>
          <button
            type="button"
            class="flex size-8 items-center justify-center rounded-full text-zinc-400 transition-colors hover:bg-zinc-100 hover:text-zinc-600 dark:hover:bg-zinc-800 dark:hover:text-zinc-300"
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

    <!-- ── 购买确认弹窗 ── -->
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
            class="flex size-8 items-center justify-center rounded-full text-zinc-400 transition-colors hover:bg-zinc-100 hover:text-zinc-600 dark:hover:bg-zinc-800 dark:hover:text-zinc-300"
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
            <p v-if="isMember" class="flex items-start gap-1.5 rounded-lg bg-emerald-50 px-3 py-2 text-xs leading-relaxed text-emerald-700 dark:bg-emerald-400/10 dark:text-emerald-400">
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

          <!-- 操作 -->
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

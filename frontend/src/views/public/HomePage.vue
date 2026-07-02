<script setup lang="ts">
// 首页：1:1 移植自原项目 Node-XZJI/frontend/src/HomeApp.jsx
// 样式全部来自全局 styles/home.css + modal.css + toast.css（移植自原项目）
import { ref, computed, onMounted, onUnmounted, watch } from 'vue'
import { useRouter, useRoute } from 'vue-router'
import api from '@/api'
import { contentApi, type Page, type Category, type Announcement } from '@/api/content'
import { useAuthStore } from '@/stores/auth'
import { useThemeStore } from '@/stores/theme'
import { useToastStore } from '@/stores/toast'
import { blockAccess } from '@/utils/blockAccess'
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
const keyword = ref('')
const result = ref<any>(null)
const loading = ref(false)
const error = ref('')
const settings = ref({
  siteName: '',
  siteDescription: '',
  footerText: '',
  siteLogoType: 'text',
  siteLogoUrl: '',
})
const userDropdownOpen = ref(false)
const themeDropdownOpen = ref(false)
const copiedIdx = ref<number | null>(null)
const purchaseDialog = ref<any>(null)
const purchaseLoading = ref(false)
const purchasedLinks = ref<Record<string, string>>({})
const purchasedCodes = ref<Record<string, string>>({})

// 关键词浏览
const keywords = ref<any[]>([])
const keywordsLoading = ref(true)
const categories = ref<Category[]>([])
const selectedCategory = ref<number | null>(null)
const pages = ref<Page[]>([])
const copiedKeywordId = ref<number | null>(null)

// 公告
const announcements = ref<Announcement[]>([])
const visibleAnn = ref<Announcement | null>(null)

const inputRef = ref<HTMLInputElement | null>(null)
const userDropdownRef = ref<HTMLElement | null>(null)
const themeDropdownRef = ref<HTMLElement | null>(null)

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
const logoLetter = computed(() => (settings.value.siteName?.[0] || '').toUpperCase())

// 公告正文（消毒后）
const annContent = computed(() => DOMPurify.sanitize(visibleAnn.value?.content || '暂无内容'))
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
  attachClickOutside()
})

onUnmounted(() => detachClickOutside())

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
        announcements.value = data
        visibleAnn.value = firstUnseen
      }
    }
  } catch { /* 忽略 */ }
}

// 分类切换 → 重新拉取关键词
watch(selectedCategory, (id) => { fetchKeywords(id) })

// ── 主题下拉 ──
function applyTheme(t: 'dark' | 'light') {
  theme.setTheme(t)
  themeDropdownOpen.value = false
}

// ── 点击外部关闭下拉菜单 ──
let clickOutsideHandler: ((e: MouseEvent) => void) | null = null
function attachClickOutside() {
  clickOutsideHandler = (e: MouseEvent) => {
    if (userDropdownRef.value && !userDropdownRef.value.contains(e.target as Node)) userDropdownOpen.value = false
    if (themeDropdownRef.value && !themeDropdownRef.value.contains(e.target as Node)) themeDropdownOpen.value = false
  }
  document.addEventListener('mousedown', clickOutsideHandler)
}
function detachClickOutside() {
  if (clickOutsideHandler) document.removeEventListener('mousedown', clickOutsideHandler)
}

// ── 退出登录 ──
function handleLogout() {
  auth.logout()
  userDropdownOpen.value = false
  router.push('/')
}

// ── 复制提取码 ──
function handleCopyCode(code: string, idx: number) {
  navigator.clipboard.writeText(code).then(() => {
    copiedIdx.value = idx
    setTimeout(() => { copiedIdx.value = null }, 1500)
  })
}

// ── 关键词卡片：复制关键词 ──
function handleKeywordClick(kw: Keyword) {
  navigator.clipboard.writeText(kw.keyword).then(() => {
    copiedKeywordId.value = kw.id
    toast.success(`已复制「${kw.keyword}」，粘贴到搜索框获取资源`)
    setTimeout(() => { copiedKeywordId.value = null }, 1200)
    inputRef.value?.focus()
  }).catch(() => toast.error('复制失败，请手动复制'))
}

// ── 分类筛选 ──
function handleCategorySelect(id: number | null) { selectedCategory.value = id }

// ── 链接访问控制（收敛至 utils/blockAccess.ts，与后端 service/content.go:450 保持一致）──

// 取某块的提取码：免费链接用 response 中的 code；付费链接仅在已购买时用缓存的 code（脱敏后真实 code 不在 response 里）
function blockCode(block: KeywordBlock, i: number): string {
  if (blockAccess(block) === 'free') return block.code || ''
  const key = `${result.value?.keywordId}-${i}`
  return purchasedCodes.value[key] || ''
}

function handleLinkClick(e: Event, block: KeywordBlock, idx: number) {
  const access = blockAccess(block)
  if (access === 'free') return

  e.preventDefault()

  if (!isLoggedIn.value) { router.push('/login'); return }

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
  } catch (err: any) {
    const data = err?.response?.data
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
    // 解析 response：后端返回 JSON 字符串（块数组或纯文本），原版 Node 在后端 JSON.parse，
    // Go 版后端返回原始字符串，此处前端解析以兼容块渲染与纯文本。
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
  } catch (err: any) {
    error.value = err?.message || '搜索失败，请重试'
  } finally {
    loading.value = false
  }
}

function handleKeyPress(e: KeyboardEvent) {
  if (e.key === 'Enter') handleSearch()
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
  <div class="app">
    <!-- ── 公告弹窗 ── -->
    <div v-if="visibleAnn" class="modal-overlay" @click="dismissAnnouncement">
      <div class="modal-content announcement-modal" @click.stop>
        <div class="modal-header">
          <div style="display: flex; align-items: center; gap: 0.5rem;">
            <span class="announcement-type-badge" :class="visibleAnn.type || 'info'">
              {{ annTypeLabels[visibleAnn.type] || visibleAnn.type }}
            </span>
            <h3>{{ visibleAnn.title }}</h3>
          </div>
          <button class="modal-close" @click="dismissAnnouncement">X</button>
        </div>
        <div class="announcement-body">
          <div v-html="annContent" />
        </div>
        <div class="announcement-actions">
          <button class="action-button" @click="dismissAnnouncement">我知道了</button>
        </div>
      </div>
    </div>

    <!-- ── Header ── -->
    <header class="header">
      <div class="logo">
        <router-link v-if="settings.siteLogoType === 'image' && settings.siteLogoUrl" to="/">
          <img :src="settings.siteLogoUrl" :alt="settings.siteName || 'Logo'" class="logo-img" />
        </router-link>
        <template v-else>
          <span class="logo-mark">{{ logoLetter }}</span>
          <span class="logo-text">{{ settings.siteName }}</span>
        </template>
      </div>
      <div class="header-right">
        <!-- 主题切换下拉 -->
        <div class="theme-toggle" ref="themeDropdownRef">
          <button
            class="theme-toggle-btn"
            @click="themeDropdownOpen = !themeDropdownOpen"
            title="切换主题"
          >
            <svg v-if="!theme.isDark" width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round">
              <circle cx="12" cy="12" r="5" />
              <line x1="12" y1="1" x2="12" y2="3" /><line x1="12" y1="21" x2="12" y2="23" />
              <line x1="4.22" y1="4.22" x2="5.64" y2="5.64" /><line x1="18.36" y1="18.36" x2="19.78" y2="19.78" />
              <line x1="1" y1="12" x2="3" y2="12" /><line x1="21" y1="12" x2="23" y2="12" />
              <line x1="4.22" y1="19.78" x2="5.64" y2="18.36" /><line x1="18.36" y1="5.64" x2="19.78" y2="4.22" />
            </svg>
            <svg v-else width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round">
              <path d="M21 12.79A9 9 0 1 1 11.21 3 7 7 0 0 0 21 12.79z" />
            </svg>
          </button>

          <div v-if="themeDropdownOpen" class="dropdown-menu theme-dropdown-menu">
            <button
              class="dropdown-item"
              :class="{ 'theme-active': theme.isDark }"
              @click="applyTheme('dark')"
            >
              <svg width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round" class="theme-option-icon">
                <path d="M21 12.79A9 9 0 1 1 11.21 3 7 7 0 0 0 21 12.79z" />
              </svg>
              深色模式
            </button>
            <button
              class="dropdown-item"
              :class="{ 'theme-active': !theme.isDark }"
              @click="applyTheme('light')"
            >
              <svg width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round" class="theme-option-icon">
                <circle cx="12" cy="12" r="5" /><line x1="12" y1="1" x2="12" y2="3" /><line x1="12" y1="21" x2="12" y2="23" />
                <line x1="4.22" y1="4.22" x2="5.64" y2="5.64" /><line x1="18.36" y1="18.36" x2="19.78" y2="19.78" />
                <line x1="1" y1="12" x2="3" y2="12" /><line x1="21" y1="12" x2="23" y2="12" />
                <line x1="4.22" y1="19.78" x2="5.64" y2="18.36" /><line x1="18.36" y1="5.64" x2="19.78" y2="4.22" />
              </svg>
              浅色模式
            </button>
          </div>
        </div>

        <!-- 已登录：用户下拉 -->
        <div v-if="isLoggedIn" class="user-dropdown" ref="userDropdownRef">
          <button class="avatar-trigger" @click="userDropdownOpen = !userDropdownOpen">
            <img v-if="auth.user?.avatar" :src="auth.user.avatar" alt="" class="avatar-trigger-img" />
            <div v-else class="avatar-trigger-placeholder">
              {{ (auth.user?.nickname || auth.user?.username || 'U')[0].toUpperCase() }}
            </div>
          </button>

          <div v-if="userDropdownOpen" class="dropdown-menu">
            <div class="dropdown-header">
              <span class="dropdown-name">{{ auth.user?.nickname || auth.user?.username }}</span>
              <span class="dropdown-role">{{ auth.user?.role === 'admin' ? '管理员' : '用户' }}</span>
            </div>
            <div class="dropdown-divider" />
            <router-link to="/user" class="dropdown-item" @click="userDropdownOpen = false">用户中心</router-link>
            <router-link v-if="isAdmin" to="/admin" class="dropdown-item" @click="userDropdownOpen = false">管理后台</router-link>
            <button class="dropdown-item dropdown-item-danger" @click="handleLogout">退出登录</button>
          </div>
        </div>

        <!-- 未登录：登录/注册 -->
        <div v-else class="header-actions">
          <router-link to="/login" class="header-link">登录</router-link>
          <router-link to="/register" class="header-link header-link-primary">注册</router-link>
        </div>
      </div>
    </header>

    <main class="main">
      <!-- ── Hero + 搜索 ── -->
      <div class="hero">
        <p class="hero-tagline">关键词驱动的内容发现</p>
        <h1 class="title">{{ settings.siteName }}</h1>
        <p class="subtitle">{{ settings.siteDescription }}</p>
      </div>

      <!-- ── 搜索区 ── -->
      <div class="search-section">
        <div class="input-group">
          <label for="keyword-input" class="input-label">输入关键词搜索</label>
          <div class="input-wrapper">
            <input
              id="keyword-input"
              ref="inputRef"
              type="text"
              class="search-input"
              placeholder="hello, node, react..."
              v-model="keyword"
              @keypress="handleKeyPress"
              :disabled="loading"
              autocomplete="off"
              spellcheck="false"
            />
            <button class="search-button" @click="handleSearch" :disabled="loading || !keyword.trim()">
              <span v-if="loading" class="loading-dots"><span>.</span><span>.</span><span>.</span></span>
              <span v-else>搜索</span>
            </button>
          </div>
          <p v-if="error" class="error-text">{{ error }}</p>
        </div>
      </div>

      <!-- ── 分类胶囊 ── -->
      <div v-if="categories.length > 0" class="category-pills">
        <button
          class="category-pill"
          :class="{ active: selectedCategory === null }"
          @click="handleCategorySelect(null)"
        >全部</button>
        <button
          v-for="c in categories"
          :key="c.id"
          class="category-pill"
          :class="{ active: selectedCategory === c.id }"
          @click="handleCategorySelect(c.id)"
          :title="c.description || c.name"
        >{{ c.name }}</button>
      </div>

      <!-- ── 搜索结果 ── -->
      <div v-if="result" class="result-section" :key="result.timestamp">
        <div class="result-content">
          <div v-if="Array.isArray(result.response)" class="result-blocks">
            <template v-for="(block, i) in result.response" :key="i">
              <div v-if="block.type === 'text' && block.text" class="result-block-text">{{ block.text }}</div>
              <div v-else-if="block.type === 'image' && block.src" class="result-block-image">
                <img :src="block.src" :alt="block.alt || ''" />
              </div>
              <div
                v-else-if="block.type === 'link'"
                class="result-block-link-wrap"
                :class="{ 'paid-link': blockAccess(block) !== 'free' }"
              >
                <a
                  :href="blockAccess(block) === 'free' ? block.url : 'javascript:void(0)'"
                  target="_blank"
                  rel="noopener noreferrer"
                  class="result-block-link"
                  :class="{ 'has-cost': blockAccess(block) !== 'free' }"
                  @click="(e) => handleLinkClick(e, block, i)"
                >
                  <span class="result-block-link-icon">
                    {{ blockAccess(block) !== 'free' ? (blockAccess(block) === 'membership' ? 'M' : 'L') : '\u2197' }}
                  </span>
                  <span class="result-block-link-label">{{ block.label || '链接' }}</span>
                  <span
                    v-if="blockAccess(block) !== 'free' && !purchasedLinks[`${result.keywordId}-${i}`]"
                    class="result-block-link-cost"
                  >
                    {{ blockAccess(block) === 'membership' ? '会员免费' : `${block.cost || 0} 元` }}
                  </span>
                  <span
                    v-if="blockCode(block, i)"
                    class="result-block-link-code"
                    :class="{ copied: copiedIdx === i }"
                    @click="(e) => { e.preventDefault(); e.stopPropagation(); handleCopyCode(blockCode(block, i), i) }"
                    title="点击复制提取码"
                  >
                    {{ copiedIdx === i ? '已复制!' : `提取码: ${blockCode(block, i)}` }}
                  </span>
                </a>
              </div>
            </template>
          </div>
          <div v-else class="result-text-response">{{ result.response }}</div>
        </div>
        <div class="result-meta">
          <span class="result-meta-item result-meta-keyword">{{ result.keyword }}</span>
          <span class="result-meta-sep" />
          <span class="result-meta-item">{{ formatTimestamp(result.timestamp) }}</span>
          <span class="result-meta-sep" />
          <span class="result-meta-item result-meta-source">{{ result.source }}</span>
        </div>
      </div>

      <!-- ── 关键词浏览区 ── -->
      <div class="keywords-section">
        <div class="keywords-section-header">
          <h2 class="keywords-section-title">{{ keywordsSectionTitle }}</h2>
          <span v-if="!keywordsLoading" class="keywords-section-count">{{ keywords.length }} 个关键词</span>
        </div>

        <div v-if="keywordsLoading" class="keywords-grid">
          <div v-for="i in 6" :key="i" class="keyword-skeleton">
            <div class="skeleton-line" />
            <div class="skeleton-line" />
            <div class="skeleton-line" />
          </div>
        </div>
        <div v-else-if="keywords.length === 0" class="keywords-grid">
          <div class="keywords-empty">
            <p class="keywords-empty-text">{{ selectedCategory ? '该分类下暂无关键词' : '暂无关键词' }}</p>
            <p class="keywords-empty-hint">{{ selectedCategory ? '试试其他分类，或在后台添加关键词' : '在管理后台添加关键词后将在此展示' }}</p>
          </div>
        </div>
        <div v-else class="keywords-grid">
          <div
            v-for="kw in keywords"
            :key="kw.id"
            class="keyword-card"
            :class="{ 'copied-flash': copiedKeywordId === kw.id }"
            @click="handleKeywordClick(kw)"
            title="点击复制关键词"
          >
            <div class="keyword-card-header">
              <span class="keyword-card-name">{{ kw.keyword }}</span>
              <button class="keyword-card-copy-icon" @click.stop="handleKeywordClick(kw)" title="复制关键词">
                <svg width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round">
                  <rect x="9" y="9" width="13" height="13" rx="2" ry="2" />
                  <path d="M5 15H4a2 2 0 0 1-2-2V4a2 2 0 0 1 2-2h9a2 2 0 0 1 2 2v1" />
                </svg>
              </button>
            </div>
            <p v-if="kw.description" class="keyword-card-desc">{{ kw.description }}</p>
            <div class="keyword-card-meta"><span>点击复制</span></div>
          </div>
        </div>
      </div>

      <!-- ── 自定义页面区 ── -->
      <div v-if="pages.length > 0" class="pages-section">
        <div class="pages-section-header">
          <h2 class="pages-section-title">更多信息</h2>
        </div>
        <div class="pages-grid">
          <router-link
            v-for="p in pages"
            :key="p.id"
            :to="`/page/${p.slug}`"
            class="page-link-card"
          >
            <span>{{ p.title }}</span>
            <span class="page-link-card-arrow">
              <svg width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round">
                <path d="M5 12h14M12 5l7 7-7 7" />
              </svg>
            </span>
          </router-link>
        </div>
      </div>
    </main>

    <footer class="footer">
      <span class="footer-text">{{ settings.footerText }}</span>
      <span class="footer-dot">-</span>
      <span class="footer-text">XZJI</span>
    </footer>

    <!-- ── 购买确认弹窗 ── -->
    <div v-if="purchaseDialog" class="modal-overlay" @click="purchaseDialog = null">
      <div class="modal-content purchase-modal" @click.stop>
        <div class="modal-header">
          <h3>{{ purchaseDialog.access === 'membership' ? '会员免费链接' : '链接需要付费' }}</h3>
          <button class="modal-close" @click="purchaseDialog = null">X</button>
        </div>
        <div class="purchase-info">
          <p class="purchase-label">链接名称</p>
          <p class="purchase-value">{{ purchaseDialog.block.label || purchaseDialog.block.url }}</p>

          <template v-if="purchaseDialog.access === 'membership'">
            <p class="purchase-label">会员免费访问</p>
            <p class="purchase-value" style="color: var(--text-primary);">开通会员后免费下载，每日有次数限制</p>
            <p class="purchase-label">非会员价格</p>
            <p class="purchase-cost">{{ purchaseDialog.cost > 0 ? `${purchaseDialog.cost} 元` : '不可购买' }}</p>
          </template>
          <template v-else>
            <div v-if="isMember" class="purchase-discount-badge">
              <span class="purchase-discount-icon">M</span>
              <span>会员折扣：购买付费链接将自动享受会员折扣价</span>
            </div>
            <p class="purchase-label">所需余额</p>
            <p class="purchase-cost">{{ purchaseDialog.cost }} 元</p>
          </template>

          <p class="purchase-label">当前余额</p>
          <p class="purchase-balance" :class="{ insufficient: currentBalance < purchaseDialog.cost }">
            {{ currentBalance.toFixed(2) }} 元
          </p>

          <div
            v-if="currentBalance < purchaseDialog.cost && purchaseDialog.access !== 'membership'"
            class="purchase-warning"
          >
            <span class="purchase-warning-icon">
              <svg width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round">
                <circle cx="12" cy="12" r="10" /><line x1="12" y1="8" x2="12" y2="12" /><line x1="12" y1="16" x2="12.01" y2="16" />
              </svg>
            </span>
            <span>余额不足，还需 <strong>{{ (purchaseDialog.cost - currentBalance).toFixed(2) }}</strong> 元</span>
          </div>
        </div>

        <div class="purchase-actions">
          <template v-if="purchaseDialog.access === 'membership'">
            <button class="action-button" @click="handlePurchase" :disabled="purchaseLoading">
              {{ purchaseLoading ? '验证中...' : '会员验证' }}
            </button>
            <button class="action-button" @click="purchaseDialog = null; router.push('/user/membership')">开通会员</button>
          </template>
          <template v-else>
            <template v-if="currentBalance < purchaseDialog.cost">
              <button class="action-button purchase-recharge-btn" @click="purchaseDialog = null; router.push('/user')">去充值</button>
              <button class="action-button secondary" @click="purchaseDialog = null">取消</button>
            </template>
            <template v-else>
              <button class="action-button" @click="handlePurchase" :disabled="purchaseLoading">
                {{ purchaseLoading ? '购买中...' : '确认购买' }}
              </button>
              <button class="action-button secondary" @click="purchaseDialog = null">取消</button>
            </template>
          </template>
        </div>
      </div>
    </div>
  </div>
</template>

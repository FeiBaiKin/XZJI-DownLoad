<script setup lang="ts">
// 管理后台仪表盘：核心指标、近 7 日趋势、热门资源、最近订单
// 设计：沿用 zinc / emerald + rounded-2xl 卡片语言，数字用 mono，密度中等
import { ref, onMounted, computed } from 'vue'
import { useRouter } from 'vue-router'
import { NSkeleton, NButton, NTag, NIcon } from 'naive-ui'
import { PhArrowRight, PhCoin, PhHeart, PhLink as LinkIcon, PhFolder, PhFileText } from '@phosphor-icons/vue'
import api from '@/api'
import { formatMoney } from '@/utils/format'
import { methodLabels } from '@/constants/admin'
import { extractErrorMessage } from '@/types/error'
import { useToastStore } from '@/stores/toast'

const toast = useToastStore()
const router = useRouter()

interface MethodStat {
  payment_method: string
  count: number
  revenue: number
}
interface DayRev {
  date: string
  revenue: number
  orders: number
}
interface RecentOrder {
  id: number
  order_no: string
  user_id: number
  username: string
  amount: number
  payment_method: string
  paid_at?: string | null
}
interface TopKeyword {
  keyword_id: number
  keyword: string
  purchases: number
  status: string
}
interface Stats {
  siteName?: string
  users?: number
  newUsersToday?: number
  newUsersWeek?: number
  newUsersMonth?: number
  revenueTotal?: number | string
  revenueMonth?: number
  revenueToday?: number
  ordersTotal?: number
  ordersPaid?: number
  ordersPending?: number
  activeMemberships?: number
  expiringMemberships?: number
  keywords?: number
  keywordsPublished?: number
  keywordsDraft?: number
  checkinsToday?: number
  methodBreakdown?: MethodStat[]
  revenueTrend?: DayRev[]
  linkPurchasesTotal?: number
  linkPurchasesToday?: number
  favoritesTotal?: number
  categories?: number
  pagesPublished?: number
  recentOrders?: RecentOrder[]
  topKeywords?: TopKeyword[]
}

const stats = ref<Stats | null>(null)
const loading = ref(true)

onMounted(() => { fetchStats() })

async function fetchStats() {
  try {
    const res = await api.get('/admin/stats')
    stats.value = res.data
  } catch (err) {
    console.error('获取统计失败:', err)
    toast.error(extractErrorMessage(err, '加载失败，请刷新重试'))
  } finally {
    loading.value = false
  }
}

const totalRevenue = computed(() => Number(stats.value?.revenueTotal) || 0)
const methodList = computed(() => stats.value?.methodBreakdown || [])
const trend = computed(() => stats.value?.revenueTrend || [])
const maxTrendRevenue = computed(() => {
  const m = Math.max(0, ...trend.value.map((d) => Number(d.revenue) || 0))
  return m > 0 ? m : 1
})
const recentOrders = computed(() => stats.value?.recentOrders || [])
const topKeywords = computed(() => stats.value?.topKeywords || [])

function methodPct(revenue: number) {
  return totalRevenue.value > 0 ? Math.round((Number(revenue) / totalRevenue.value) * 100) : 0
}
function shortDate(iso: string) {
  // YYYY-MM-DD → MM/DD
  if (!iso || iso.length < 10) return iso
  return iso.slice(5).replace('-', '/')
}
function formatPaidAt(v?: string | null) {
  if (!v) return '-'
  try {
    const d = new Date(v)
    if (Number.isNaN(d.getTime())) return v
    const mm = String(d.getMonth() + 1).padStart(2, '0')
    const dd = String(d.getDate()).padStart(2, '0')
    const hh = String(d.getHours()).padStart(2, '0')
    const mi = String(d.getMinutes()).padStart(2, '0')
    return `${mm}-${dd} ${hh}:${mi}`
  } catch {
    return v
  }
}
</script>

<template>
  <div>
    <div v-if="loading" class="space-y-6">
      <NSkeleton text width="20%" size="large" />
      <div class="grid gap-4 sm:grid-cols-2 xl:grid-cols-4">
        <NSkeleton v-for="i in 4" :key="i" height="104px" :sharp="false" />
      </div>
      <NSkeleton height="180px" :sharp="false" />
      <div class="grid gap-4 lg:grid-cols-2">
        <NSkeleton height="240px" :sharp="false" />
        <NSkeleton height="240px" :sharp="false" />
      </div>
    </div>

    <template v-else>
      <p v-if="!stats" class="rounded-xl bg-zinc-50 py-12 text-center text-sm text-zinc-400 dark:bg-zinc-900 dark:text-zinc-500">
        暂无数据
      </p>

      <template v-else>
        <header class="mb-6 flex flex-wrap items-end justify-between gap-3">
          <div>
            <h1 class="text-xl font-bold tracking-tight text-zinc-900 dark:text-zinc-50">仪表盘</h1>
            <p class="mt-1 text-sm text-zinc-500 dark:text-zinc-400">
              {{ stats.siteName || '站点' }} · 经营与内容概览
            </p>
          </div>
          <NButton size="small" secondary @click="router.push('/admin/keywords')">
            管理关键词
            <template #icon><NIcon><PhArrowRight /></NIcon></template>
          </NButton>
        </header>

        <!-- 核心指标 -->
        <div class="grid gap-4 sm:grid-cols-2 xl:grid-cols-4">
          <div class="rounded-2xl border border-zinc-200 bg-white p-5 dark:border-zinc-800 dark:bg-zinc-900">
            <p class="text-[13px] text-zinc-500 dark:text-zinc-400">注册用户</p>
            <p class="mt-1.5 font-mono text-3xl font-semibold tracking-tight text-zinc-900 dark:text-zinc-50">
              {{ stats.users || 0 }}
            </p>
            <p class="mt-1.5 font-mono text-xs text-zinc-500 dark:text-zinc-400">
              本月 +{{ stats.newUsersMonth || 0 }} / 本周 +{{ stats.newUsersWeek || 0 }}
            </p>
          </div>
          <div class="rounded-2xl border border-zinc-200 bg-white p-5 dark:border-zinc-800 dark:bg-zinc-900">
            <p class="text-[13px] text-zinc-500 dark:text-zinc-400">总收入</p>
            <p class="mt-1.5 font-mono text-3xl font-semibold tracking-tight text-emerald-600 dark:text-emerald-400">
              {{ formatMoney(totalRevenue) }}
            </p>
            <p class="mt-1.5 font-mono text-xs text-zinc-500 dark:text-zinc-400">
              今日 {{ formatMoney(stats.revenueToday || 0) }} · 本月 {{ formatMoney(stats.revenueMonth || 0) }}
            </p>
          </div>
          <div class="rounded-2xl border border-zinc-200 bg-white p-5 dark:border-zinc-800 dark:bg-zinc-900">
            <p class="text-[13px] text-zinc-500 dark:text-zinc-400">订单总数</p>
            <p class="mt-1.5 font-mono text-3xl font-semibold tracking-tight text-zinc-900 dark:text-zinc-50">
              {{ stats.ordersTotal || 0 }}
            </p>
            <p class="mt-1.5 font-mono text-xs text-zinc-500 dark:text-zinc-400">
              已支付 {{ stats.ordersPaid || 0 }} / 待支付 {{ stats.ordersPending || 0 }}
            </p>
          </div>
          <div class="rounded-2xl border border-zinc-200 bg-white p-5 dark:border-zinc-800 dark:bg-zinc-900">
            <p class="text-[13px] text-zinc-500 dark:text-zinc-400">活跃会员</p>
            <p class="mt-1.5 font-mono text-3xl font-semibold tracking-tight text-zinc-900 dark:text-zinc-50">
              {{ stats.activeMemberships || 0 }}
            </p>
            <p
              class="mt-1.5 text-xs"
              :class="(stats.expiringMemberships || 0) > 0 ? 'text-amber-600 dark:text-amber-400' : 'text-zinc-500 dark:text-zinc-400'"
            >
              {{ (stats.expiringMemberships || 0) > 0 ? `${stats.expiringMemberships} 人 7 天内到期` : '近期无到期' }}
            </p>
          </div>
        </div>

        <!-- 近 7 日收入 + 支付方式 -->
        <section class="mt-8">
          <h3 class="mb-3 text-[15px] font-semibold text-zinc-900 dark:text-zinc-100">收入动态</h3>
          <div class="grid gap-4 lg:grid-cols-5">
            <div class="rounded-2xl border border-zinc-200 bg-white p-5 lg:col-span-3 dark:border-zinc-800 dark:bg-zinc-900">
              <div class="flex items-baseline justify-between gap-2">
                <p class="text-[13px] text-zinc-500 dark:text-zinc-400">近 7 日收入</p>
                <p class="font-mono text-xs text-zinc-400 dark:text-zinc-500">
                  峰值 {{ formatMoney(Math.max(0, ...trend.map((d) => Number(d.revenue) || 0))) }}
                </p>
              </div>
              <div v-if="trend.length === 0" class="mt-10 text-center text-sm text-zinc-400 dark:text-zinc-500">
                暂无趋势数据
              </div>
              <div v-else class="mt-5 flex h-36 items-end gap-2 sm:gap-3">
                <div
                  v-for="d in trend"
                  :key="d.date"
                  class="group flex min-w-0 flex-1 flex-col items-center gap-2"
                >
                  <span class="font-mono text-[10px] text-zinc-400 opacity-0 transition-opacity group-hover:opacity-100 dark:text-zinc-500">
                    {{ formatMoney(d.revenue) }}
                  </span>
                  <div class="flex w-full flex-1 items-end justify-center">
                    <div
                      class="w-full max-w-[36px] rounded-t-md bg-emerald-500/90 transition-all dark:bg-emerald-400/90"
                      :style="{ height: Math.max(4, (Number(d.revenue) / maxTrendRevenue) * 100) + '%' }"
                      :title="`${d.date} · ${formatMoney(d.revenue)} · ${d.orders} 笔`"
                    />
                  </div>
                  <span class="font-mono text-[10px] text-zinc-500 dark:text-zinc-400">{{ shortDate(d.date) }}</span>
                </div>
              </div>
            </div>

            <div class="rounded-2xl border border-zinc-200 bg-white p-5 lg:col-span-2 dark:border-zinc-800 dark:bg-zinc-900">
              <p class="text-[13px] text-zinc-500 dark:text-zinc-400">支付方式占比</p>
              <p v-if="methodList.length === 0" class="mt-8 text-center text-sm text-zinc-400 dark:text-zinc-500">
                暂无支付数据
              </p>
              <div v-else class="mt-4 space-y-3">
                <div v-for="m in methodList" :key="m.payment_method" class="flex items-center gap-3">
                  <span class="w-14 shrink-0 text-sm text-zinc-700 dark:text-zinc-300">
                    {{ methodLabels[m.payment_method] || m.payment_method }}
                  </span>
                  <span class="w-12 shrink-0 font-mono text-xs text-zinc-500 dark:text-zinc-400">{{ m.count }} 笔</span>
                  <div class="h-1.5 flex-1 overflow-hidden rounded-full bg-zinc-100 dark:bg-zinc-800">
                    <div
                      class="h-full rounded-full bg-emerald-500 dark:bg-emerald-400"
                      :style="{ width: methodPct(m.revenue) + '%' }"
                    />
                  </div>
                  <span class="w-10 shrink-0 text-right font-mono text-xs text-zinc-500 dark:text-zinc-400">
                    {{ methodPct(m.revenue) }}%
                  </span>
                </div>
              </div>
            </div>
          </div>
        </section>

        <!-- 内容与转化 -->
        <section class="mt-8">
          <h3 class="mb-3 text-[15px] font-semibold text-zinc-900 dark:text-zinc-100">内容与转化</h3>
          <div class="grid gap-4 sm:grid-cols-2 xl:grid-cols-4">
            <div class="rounded-2xl border border-zinc-200 bg-white p-5 dark:border-zinc-800 dark:bg-zinc-900">
              <div class="flex items-center gap-2 text-[13px] text-zinc-500 dark:text-zinc-400">
                <NIcon :size="15"><PhFolder /></NIcon>
                关键词资源
              </div>
              <p class="mt-2 font-mono text-2xl font-semibold text-zinc-900 dark:text-zinc-50">{{ stats.keywords || 0 }}</p>
              <p class="mt-1.5 font-mono text-xs text-zinc-500 dark:text-zinc-400">
                上架 {{ stats.keywordsPublished || 0 }} · 下架 {{ stats.keywordsDraft || 0 }}
              </p>
            </div>
            <div class="rounded-2xl border border-zinc-200 bg-white p-5 dark:border-zinc-800 dark:bg-zinc-900">
              <div class="flex items-center gap-2 text-[13px] text-zinc-500 dark:text-zinc-400">
                <NIcon :size="15"><LinkIcon /></NIcon>
                链接购买
              </div>
              <p class="mt-2 font-mono text-2xl font-semibold text-zinc-900 dark:text-zinc-50">
                {{ stats.linkPurchasesTotal || 0 }}
              </p>
              <p class="mt-1.5 font-mono text-xs text-zinc-500 dark:text-zinc-400">
                今日 +{{ stats.linkPurchasesToday || 0 }}
              </p>
            </div>
            <div class="rounded-2xl border border-zinc-200 bg-white p-5 dark:border-zinc-800 dark:bg-zinc-900">
              <div class="flex items-center gap-2 text-[13px] text-zinc-500 dark:text-zinc-400">
                <NIcon :size="15"><PhHeart /></NIcon>
                收藏总数
              </div>
              <p class="mt-2 font-mono text-2xl font-semibold text-zinc-900 dark:text-zinc-50">
                {{ stats.favoritesTotal || 0 }}
              </p>
              <p class="mt-1.5 font-mono text-xs text-zinc-500 dark:text-zinc-400">
                今日签到 {{ stats.checkinsToday || 0 }}
              </p>
            </div>
            <div class="rounded-2xl border border-zinc-200 bg-white p-5 dark:border-zinc-800 dark:bg-zinc-900">
              <div class="flex items-center gap-2 text-[13px] text-zinc-500 dark:text-zinc-400">
                <NIcon :size="15"><PhFileText /></NIcon>
                分类 / 页面
              </div>
              <p class="mt-2 font-mono text-2xl font-semibold text-zinc-900 dark:text-zinc-50">
                {{ stats.categories || 0 }}
                <span class="text-base font-normal text-zinc-400">/</span>
                {{ stats.pagesPublished || 0 }}
              </p>
              <p class="mt-1.5 text-xs text-zinc-500 dark:text-zinc-400">分类数 / 已发布页面</p>
            </div>
          </div>
        </section>

        <!-- 热门资源 + 最近订单 -->
        <section class="mt-8 grid gap-4 lg:grid-cols-2">
          <div class="rounded-2xl border border-zinc-200 bg-white p-5 dark:border-zinc-800 dark:bg-zinc-900">
            <div class="mb-4 flex items-center justify-between">
              <h3 class="text-[15px] font-semibold text-zinc-900 dark:text-zinc-100">热门资源</h3>
              <span class="text-xs text-zinc-400 dark:text-zinc-500">按链接购买次数</span>
            </div>
            <p v-if="topKeywords.length === 0" class="py-8 text-center text-sm text-zinc-400 dark:text-zinc-500">
              暂无购买数据
            </p>
            <ul v-else class="divide-y divide-zinc-100 dark:divide-zinc-800">
              <li
                v-for="(kw, idx) in topKeywords"
                :key="kw.keyword_id"
                class="flex items-center gap-3 py-2.5"
              >
                <span class="w-5 shrink-0 font-mono text-xs text-zinc-400 dark:text-zinc-500">
                  {{ String(idx + 1).padStart(2, '0') }}
                </span>
                <span class="min-w-0 flex-1 truncate text-sm font-medium text-zinc-800 dark:text-zinc-100">
                  {{ kw.keyword || `ID ${kw.keyword_id}` }}
                </span>
                <NTag
                  size="tiny"
                  :bordered="false"
                  :type="kw.status === 'draft' ? 'default' : 'success'"
                >
                  {{ kw.status === 'draft' ? '下架' : '上架' }}
                </NTag>
                <span class="shrink-0 font-mono text-xs text-zinc-500 dark:text-zinc-400">
                  {{ kw.purchases }} 次
                </span>
              </li>
            </ul>
          </div>

          <div class="rounded-2xl border border-zinc-200 bg-white p-5 dark:border-zinc-800 dark:bg-zinc-900">
            <div class="mb-4 flex items-center justify-between">
              <h3 class="text-[15px] font-semibold text-zinc-900 dark:text-zinc-100">最近订单</h3>
              <NButton size="tiny" quaternary type="primary" @click="router.push('/admin/orders')">
                全部订单
              </NButton>
            </div>
            <p v-if="recentOrders.length === 0" class="py-8 text-center text-sm text-zinc-400 dark:text-zinc-500">
              暂无已支付订单
            </p>
            <ul v-else class="divide-y divide-zinc-100 dark:divide-zinc-800">
              <li
                v-for="o in recentOrders"
                :key="o.id"
                class="flex flex-wrap items-center gap-x-3 gap-y-1 py-2.5"
              >
                <NIcon :size="14" class="shrink-0 text-emerald-500 dark:text-emerald-400"><PhCoin /></NIcon>
                <span class="min-w-0 flex-1 truncate font-mono text-xs text-zinc-600 dark:text-zinc-300">
                  {{ o.order_no }}
                </span>
                <span class="text-xs text-zinc-500 dark:text-zinc-400">
                  {{ o.username || `用户#${o.user_id}` }}
                </span>
                <span class="font-mono text-sm font-semibold text-zinc-900 dark:text-zinc-50">
                  {{ formatMoney(o.amount) }}
                </span>
                <span class="w-full font-mono text-[11px] text-zinc-400 dark:text-zinc-500 sm:w-auto">
                  {{ methodLabels[o.payment_method] || o.payment_method }} · {{ formatPaidAt(o.paid_at) }}
                </span>
              </li>
            </ul>
          </div>
        </section>

        <!-- 用户增长（保留简洁块） -->
        <section class="mt-8">
          <h3 class="mb-3 text-[15px] font-semibold text-zinc-900 dark:text-zinc-100">用户增长</h3>
          <div class="grid gap-4 sm:grid-cols-3">
            <div class="rounded-2xl border border-zinc-200 bg-white p-5 dark:border-zinc-800 dark:bg-zinc-900">
              <p class="font-mono text-2xl font-semibold text-zinc-900 dark:text-zinc-50">{{ stats.newUsersToday || 0 }}</p>
              <p class="mt-1 text-xs text-zinc-500 dark:text-zinc-400">今日新增</p>
            </div>
            <div class="rounded-2xl border border-zinc-200 bg-white p-5 dark:border-zinc-800 dark:bg-zinc-900">
              <p class="font-mono text-2xl font-semibold text-zinc-900 dark:text-zinc-50">{{ stats.newUsersWeek || 0 }}</p>
              <p class="mt-1 text-xs text-zinc-500 dark:text-zinc-400">本周新增</p>
            </div>
            <div class="rounded-2xl border border-zinc-200 bg-white p-5 dark:border-zinc-800 dark:bg-zinc-900">
              <p class="font-mono text-2xl font-semibold text-zinc-900 dark:text-zinc-50">{{ stats.newUsersMonth || 0 }}</p>
              <p class="mt-1 text-xs text-zinc-500 dark:text-zinc-400">本月新增</p>
            </div>
          </div>
        </section>
      </template>
    </template>
  </div>
</template>

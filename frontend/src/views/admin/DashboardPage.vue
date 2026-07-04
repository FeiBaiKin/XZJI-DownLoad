<script setup lang="ts">
// 管理后台仪表盘：核心指标、收入概览、支付方式占比、用户与内容统计
// 业务逻辑保持不变；UI 重构为 Naive UI + Tailwind
import { ref, onMounted, computed } from 'vue'
import { NSkeleton } from 'naive-ui'
import api from '@/api'
import { formatMoney } from '@/utils/format'
import { methodLabels } from '@/constants/admin'

interface MethodStat {
  payment_method: string
  count: number
  revenue: number
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
  checkinsToday?: number
  methodBreakdown?: MethodStat[]
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
  } finally {
    loading.value = false
  }
}

const totalRevenue = computed(() => Number(stats.value?.revenueTotal) || 0)
const methodList = computed(() => stats.value?.methodBreakdown || [])
function methodPct(revenue: number) {
  return totalRevenue.value > 0 ? Math.round((Number(revenue) / totalRevenue.value) * 100) : 0
}
</script>

<template>
  <div>
    <!-- 加载骨架 -->
    <div v-if="loading" class="space-y-6">
      <NSkeleton text width="20%" size="large" />
      <div class="grid gap-4 sm:grid-cols-2 xl:grid-cols-4">
        <NSkeleton v-for="i in 4" :key="i" height="104px" :sharp="false" />
      </div>
      <NSkeleton height="180px" :sharp="false" />
    </div>

    <template v-else>
      <p v-if="!stats" class="rounded-xl bg-zinc-50 py-12 text-center text-sm text-zinc-400 dark:bg-zinc-900 dark:text-zinc-500">暂无数据</p>

      <template v-else>
        <header class="mb-6">
          <h2 class="text-xl font-bold tracking-tight text-zinc-900 dark:text-zinc-50">仪表盘</h2>
          <p class="mt-1 text-sm text-zinc-500 dark:text-zinc-400">{{ stats.siteName || '站点' }} · 数据概览</p>
        </header>

        <!-- 核心指标 -->
        <div class="grid gap-4 sm:grid-cols-2 xl:grid-cols-4">
          <div class="rounded-2xl border border-zinc-200 bg-white p-5 dark:border-zinc-800 dark:bg-zinc-900">
            <p class="text-[13px] text-zinc-500 dark:text-zinc-400">注册用户</p>
            <p class="mt-1.5 font-mono text-3xl font-semibold tracking-tight text-zinc-900 dark:text-zinc-50">{{ stats.users || 0 }}</p>
            <p class="mt-1.5 font-mono text-xs text-zinc-400 dark:text-zinc-500">本月 +{{ stats.newUsersMonth || 0 }} / 本周 +{{ stats.newUsersWeek || 0 }}</p>
          </div>
          <div class="rounded-2xl border border-zinc-200 bg-white p-5 dark:border-zinc-800 dark:bg-zinc-900">
            <p class="text-[13px] text-zinc-500 dark:text-zinc-400">总收入</p>
            <p class="mt-1.5 font-mono text-3xl font-semibold tracking-tight text-emerald-600 dark:text-emerald-400">{{ formatMoney(totalRevenue) }}</p>
            <p class="mt-1.5 font-mono text-xs text-zinc-400 dark:text-zinc-500">本月 {{ formatMoney(stats.revenueMonth || 0) }}</p>
          </div>
          <div class="rounded-2xl border border-zinc-200 bg-white p-5 dark:border-zinc-800 dark:bg-zinc-900">
            <p class="text-[13px] text-zinc-500 dark:text-zinc-400">订单总数</p>
            <p class="mt-1.5 font-mono text-3xl font-semibold tracking-tight text-zinc-900 dark:text-zinc-50">{{ stats.ordersTotal || 0 }}</p>
            <p class="mt-1.5 font-mono text-xs text-zinc-400 dark:text-zinc-500">已支付 {{ stats.ordersPaid || 0 }} / 待支付 {{ stats.ordersPending || 0 }}</p>
          </div>
          <div class="rounded-2xl border border-zinc-200 bg-white p-5 dark:border-zinc-800 dark:bg-zinc-900">
            <p class="text-[13px] text-zinc-500 dark:text-zinc-400">活跃会员</p>
            <p class="mt-1.5 font-mono text-3xl font-semibold tracking-tight text-zinc-900 dark:text-zinc-50">{{ stats.activeMemberships || 0 }}</p>
            <p class="mt-1.5 text-xs" :class="(stats.expiringMemberships || 0) > 0 ? 'text-amber-600 dark:text-amber-400' : 'text-zinc-400 dark:text-zinc-500'">
              {{ (stats.expiringMemberships || 0) > 0 ? `${stats.expiringMemberships} 人即将到期` : '全部正常' }}
            </p>
          </div>
        </div>

        <!-- 收入概览 -->
        <section class="mt-8">
          <h3 class="mb-3 text-[15px] font-semibold text-zinc-900 dark:text-zinc-100">收入概览</h3>
          <div class="grid gap-4 lg:grid-cols-2">
            <div class="rounded-2xl border border-zinc-200 bg-white p-5 dark:border-zinc-800 dark:bg-zinc-900">
              <p class="text-[13px] text-zinc-500 dark:text-zinc-400">收入明细</p>
              <div class="mt-4 grid grid-cols-3 gap-4">
                <div>
                  <p class="font-mono text-xl font-semibold text-zinc-900 dark:text-zinc-50">{{ formatMoney(stats.revenueToday || 0) }}</p>
                  <p class="mt-1 text-xs text-zinc-400 dark:text-zinc-500">今日收入</p>
                </div>
                <div>
                  <p class="font-mono text-xl font-semibold text-zinc-900 dark:text-zinc-50">{{ formatMoney(stats.revenueMonth || 0) }}</p>
                  <p class="mt-1 text-xs text-zinc-400 dark:text-zinc-500">本月收入</p>
                </div>
                <div>
                  <p class="font-mono text-xl font-semibold text-zinc-900 dark:text-zinc-50">{{ formatMoney(totalRevenue) }}</p>
                  <p class="mt-1 text-xs text-zinc-400 dark:text-zinc-500">累计收入</p>
                </div>
              </div>
            </div>

            <div class="rounded-2xl border border-zinc-200 bg-white p-5 dark:border-zinc-800 dark:bg-zinc-900">
              <p class="text-[13px] text-zinc-500 dark:text-zinc-400">支付方式</p>
              <p v-if="methodList.length === 0" class="mt-6 text-center text-sm text-zinc-400 dark:text-zinc-500">暂无支付数据</p>
              <div v-else class="mt-4 space-y-3">
                <div v-for="m in methodList" :key="m.payment_method" class="flex items-center gap-3">
                  <span class="w-14 shrink-0 text-sm text-zinc-700 dark:text-zinc-300">{{ methodLabels[m.payment_method] || m.payment_method }}</span>
                  <span class="w-12 shrink-0 font-mono text-xs text-zinc-400 dark:text-zinc-500">{{ m.count }} 笔</span>
                  <div class="h-1.5 flex-1 overflow-hidden rounded-full bg-zinc-100 dark:bg-zinc-800">
                    <div class="h-full rounded-full bg-emerald-500 dark:bg-emerald-400" :style="{ width: methodPct(m.revenue) + '%' }" />
                  </div>
                  <span class="w-10 shrink-0 text-right font-mono text-xs text-zinc-500 dark:text-zinc-400">{{ methodPct(m.revenue) }}%</span>
                </div>
              </div>
            </div>
          </div>
        </section>

        <!-- 用户与内容 -->
        <section class="mt-8">
          <h3 class="mb-3 text-[15px] font-semibold text-zinc-900 dark:text-zinc-100">用户与内容</h3>
          <div class="grid gap-4 sm:grid-cols-3">
            <div class="rounded-2xl border border-zinc-200 bg-white p-5 dark:border-zinc-800 dark:bg-zinc-900">
              <p class="text-[13px] text-zinc-500 dark:text-zinc-400">用户增长</p>
              <dl class="mt-3 space-y-2.5">
                <div class="flex items-baseline justify-between">
                  <dd class="font-mono text-lg font-semibold text-zinc-900 dark:text-zinc-50">{{ stats.newUsersToday || 0 }}</dd>
                  <dt class="text-xs text-zinc-400 dark:text-zinc-500">今日新增</dt>
                </div>
                <div class="flex items-baseline justify-between">
                  <dd class="font-mono text-lg font-semibold text-zinc-900 dark:text-zinc-50">{{ stats.newUsersWeek || 0 }}</dd>
                  <dt class="text-xs text-zinc-400 dark:text-zinc-500">本周新增</dt>
                </div>
                <div class="flex items-baseline justify-between">
                  <dd class="font-mono text-lg font-semibold text-zinc-900 dark:text-zinc-50">{{ stats.newUsersMonth || 0 }}</dd>
                  <dt class="text-xs text-zinc-400 dark:text-zinc-500">本月新增</dt>
                </div>
              </dl>
            </div>
            <div class="rounded-2xl border border-zinc-200 bg-white p-5 dark:border-zinc-800 dark:bg-zinc-900">
              <p class="text-[13px] text-zinc-500 dark:text-zinc-400">内容数据</p>
              <dl class="mt-3 space-y-2.5">
                <div class="flex items-baseline justify-between">
                  <dd class="font-mono text-lg font-semibold text-zinc-900 dark:text-zinc-50">{{ stats.keywords || 0 }}</dd>
                  <dt class="text-xs text-zinc-400 dark:text-zinc-500">关键词总数</dt>
                </div>
                <div class="flex items-baseline justify-between">
                  <dd class="font-mono text-lg font-semibold text-zinc-900 dark:text-zinc-50">{{ stats.checkinsToday || 0 }}</dd>
                  <dt class="text-xs text-zinc-400 dark:text-zinc-500">今日签到</dt>
                </div>
              </dl>
            </div>
            <div class="rounded-2xl border border-zinc-200 bg-white p-5 dark:border-zinc-800 dark:bg-zinc-900">
              <p class="text-[13px] text-zinc-500 dark:text-zinc-400">会员状态</p>
              <dl class="mt-3 space-y-2.5">
                <div class="flex items-baseline justify-between">
                  <dd class="font-mono text-lg font-semibold text-zinc-900 dark:text-zinc-50">{{ stats.activeMemberships || 0 }}</dd>
                  <dt class="text-xs text-zinc-400 dark:text-zinc-500">活跃会员</dt>
                </div>
                <div class="flex items-baseline justify-between">
                  <dd class="font-mono text-lg font-semibold" :class="(stats.expiringMemberships || 0) > 0 ? 'text-amber-600 dark:text-amber-400' : 'text-zinc-900 dark:text-zinc-50'">
                    {{ stats.expiringMemberships || 0 }}
                  </dd>
                  <dt class="text-xs text-zinc-400 dark:text-zinc-500">7 天内到期</dt>
                </div>
              </dl>
            </div>
          </div>
        </section>
      </template>
    </template>
  </div>
</template>

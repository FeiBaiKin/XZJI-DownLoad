<script setup lang="ts">
// 会员中心（重新设计）：竖向定价卡 + 左对齐布局，避免 3 等大网格卡
import { ref, computed, onMounted } from 'vue'
import { NButton, NIcon, NTag, NSkeleton, NAlert } from 'naive-ui'
import { Check, Crown } from '@vicons/tabler'
import api from '@/api'
import { useAuthStore } from '@/stores/auth'
import type { MembershipConfig, UserBrief } from '@/types/api'
import MembershipModal from '@/components/MembershipModal.vue'

const auth = useAuthStore()

interface MembershipState {
  membership: { level: string; expiresAt: string } | null
  downloads: number
  dailyLimit: number
  configs?: MembershipConfig[]
}

const membership = ref<MembershipState | null>(null)
const profile = ref<UserBrief | null>(null)
const loading = ref(true)
const message = ref('')
const selectedPlan = ref<MembershipConfig | null>(null)

onMounted(() => { fetchMembership() })

async function fetchMembership() {
  try {
    const [mbRes, pfRes] = await Promise.all([
      api.get('/user/membership'),
      api.get('/user/profile'),
    ])
    membership.value = mbRes.data
    profile.value = pfRes.data
  } catch (err) {
    console.error(err)
  } finally {
    loading.value = false
  }
}

const hasMembership = computed(() => !!membership.value?.membership)
const daysLeft = computed(() => {
  if (!hasMembership.value) return 0
  return Math.max(0, Math.ceil((new Date(membership.value!.membership!.expiresAt).getTime() - Date.now()) / 86400000))
})
const limit = computed(() => membership.value?.dailyLimit)
const limitText = computed(() => (limit.value === -1 ? '无限' : `${limit.value} 次`))
const currentPlanColor = computed(() => {
  if (!hasMembership.value) return null
  return (membership.value?.configs || []).find((c) => c.level_name === membership.value!.membership!.level)?.color || null
})
const configs = computed(() => membership.value?.configs || [])

function isCurrentPlan(plan: MembershipConfig) {
  return hasMembership.value && membership.value!.membership!.level === plan.level_name
}
function handleBuyClick(plan: MembershipConfig) { selectedPlan.value = plan }

function handleBuySuccess(data: unknown) {
  const d = data as { membershipExpiresAt: string; balance: number }
  message.value = `购买成功，有效期至 ${new Date(d.membershipExpiresAt).toLocaleDateString('zh-CN')}`
  if (auth.user) {
    auth.user.balance = d.balance
    auth.user.membership_level = selectedPlan.value?.level_name ?? auth.user.membership_level
    auth.user.membership_expires_at = d.membershipExpiresAt
  }
  selectedPlan.value = null
  fetchMembership()
}

function periodText(days: number) {
  if (days >= 365) return `${Math.floor(days / 365)} 年`
  if (days >= 30) return `${Math.floor(days / 30)} 个月`
  return `${days} 天`
}
function discountPct(plan: MembershipConfig) {
  return plan.discount > 0 && plan.discount < 1 ? Math.round(plan.discount * 100) : null
}
function expiryShort() {
  return new Date(membership.value!.membership!.expiresAt).toLocaleDateString('zh-CN', { month: 'short', day: 'numeric' })
}
function recommendedIdx(total: number) {
  return total >= 2 ? 1 : 0
}
</script>

<template>
  <div>
    <!-- 加载骨架 -->
    <div v-if="loading" class="space-y-6">
      <NSkeleton text width="20%" size="large" />
      <NSkeleton height="100px" :sharp="false" />
      <div v-for="i in 3" :key="i" class="rounded-2xl border border-zinc-200 p-6 dark:border-zinc-800">
        <div class="flex flex-col gap-4 lg:flex-row lg:items-start">
          <div class="space-y-2 lg:w-2/5"><NSkeleton text width="30%" /><NSkeleton text width="50%" /></div>
          <div class="space-y-2 lg:w-3/5"><NSkeleton text :repeat="3" /></div>
        </div>
      </div>
    </div>

    <template v-else>
      <h2 class="text-xl font-bold tracking-tight text-zinc-900 dark:text-zinc-50">会员中心</h2>

      <NAlert
        v-if="message"
        :type="message.startsWith('购买成功') ? 'success' : 'error'"
        class="mt-4"
        closable
        @close="message = ''"
      >{{ message }}</NAlert>

      <!-- 状态条 -->
      <section
        class="mt-6 flex flex-wrap items-center gap-4 rounded-2xl border p-5"
        :class="hasMembership
          ? 'border-emerald-200 bg-emerald-50/40 dark:border-emerald-400/20 dark:bg-emerald-400/5'
          : 'border-zinc-200 bg-white dark:border-zinc-800 dark:bg-zinc-900'"
      >
        <!-- 图标 -->
        <span
          class="grid size-12 shrink-0 place-items-center rounded-full"
          :class="hasMembership ? 'bg-emerald-100 text-emerald-600 dark:bg-emerald-400/15 dark:text-emerald-400' : 'bg-zinc-100 text-zinc-400 dark:bg-zinc-800 dark:text-zinc-500'"
        >
          <NIcon :size="22"><Crown /></NIcon>
        </span>

        <!-- 状态容器 -->
        <div v-if="hasMembership" class="min-w-0 flex-1">
          <div class="flex flex-wrap items-center gap-2">
            <span class="text-lg font-bold tracking-tight" :style="currentPlanColor ? { color: currentPlanColor } : {}">
              {{ membership!.membership!.level }}
            </span>
            <NTag type="success" size="small" round>当前会员</NTag>
          </div>
          <div class="mt-2 flex flex-wrap gap-x-8 gap-y-1 text-sm">
            <span class="text-zinc-500 dark:text-zinc-400">剩余 <span class="font-mono font-semibold text-zinc-800 dark:text-zinc-200">{{ daysLeft }}</span> 天</span>
            <span class="text-zinc-500 dark:text-zinc-400">今日下载 <span class="font-mono font-semibold text-zinc-800 dark:text-zinc-200">{{ membership!.downloads }}/{{ limitText }}</span></span>
            <span class="text-zinc-500 dark:text-zinc-400">到期 <span class="font-mono font-semibold text-zinc-800 dark:text-zinc-200">{{ expiryShort() }}</span></span>
          </div>
        </div>
        <div v-else class="min-w-0 flex-1">
          <p class="text-base font-semibold text-zinc-700 dark:text-zinc-300">还不是会员</p>
          <p class="mt-0.5 text-sm text-zinc-500 dark:text-zinc-400">开通会员后，可免费下载付费链接内容，并享受专属折扣</p>
        </div>
      </section>

      <!-- 方案卡片列表（竖向堆叠，左对齐） -->
      <div class="mt-6 space-y-4">
        <div
          v-for="(plan, i) in configs"
          :key="plan.level_name"
          class="group overflow-hidden rounded-2xl border bg-white transition-shadow hover:shadow-md dark:bg-zinc-900"
          :class="i === recommendedIdx(configs.length)
            ? 'border-emerald-300/70 bg-emerald-50/30 dark:border-emerald-400/20 dark:bg-emerald-400/5'
            : 'border-zinc-200 dark:border-zinc-800'"
        >
          <!-- 顶部彩色条 -->
          <div class="h-1 w-full" :style="{ background: plan.color || '#10b981' }" />

          <div class="flex flex-col gap-6 p-6 lg:flex-row lg:gap-10">
            <!-- 左栏：套餐名 + 价格 -->
            <div class="shrink-0 lg:w-[220px]">
              <div class="flex items-center gap-2">
                <span class="text-lg font-bold tracking-tight text-zinc-900 dark:text-zinc-100" :style="{ color: plan.color || undefined }">
                  {{ plan.level_name }}
                </span>
                <NTag v-if="isCurrentPlan(plan)" type="success" size="small" round>当前</NTag>
                <NTag v-else-if="i === recommendedIdx(configs.length)" size="small" round>推荐</NTag>
              </div>
              <div class="mt-2">
                <span class="align-top text-sm text-zinc-400">¥</span>
                <span class="font-mono text-4xl font-bold tracking-tight text-zinc-900 dark:text-zinc-100">{{ plan.price }}</span>
                <span class="ml-1 text-sm text-zinc-400">/{{ periodText(plan.days) }}</span>
              </div>
            </div>

            <!-- 右栏：功能列表 + CTA -->
            <div class="flex min-w-0 flex-1 flex-col">
              <ul class="grid gap-x-6 gap-y-2 sm:grid-cols-2">
                <li class="flex items-center gap-2 text-sm text-zinc-600 dark:text-zinc-400">
                  <NIcon :size="16" class="text-emerald-500 dark:text-emerald-400 shrink-0"><Check /></NIcon>
                  每日下载 {{ plan.daily_download_limit === -1 ? '不限' : `${plan.daily_download_limit} 次` }}
                </li>
                <li class="flex items-center gap-2 text-sm text-zinc-600 dark:text-zinc-400">
                  <NIcon :size="16" class="text-emerald-500 dark:text-emerald-400 shrink-0"><Check /></NIcon>
                  <template v-if="discountPct(plan)">
                    付费链接 {{ discountPct(plan) }} 折
                  </template>
                  <template v-else>标准付费链接访问</template>
                </li>
                <li class="flex items-center gap-2 text-sm text-zinc-600 dark:text-zinc-400">
                  <NIcon :size="16" class="text-emerald-500 dark:text-emerald-400 shrink-0"><Check /></NIcon>
                  会员专属标识
                </li>
                <li class="flex items-center gap-2 text-sm text-zinc-600 dark:text-zinc-400">
                  <NIcon :size="16" class="text-emerald-500 dark:text-emerald-400 shrink-0"><Check /></NIcon>
                  {{ plan.daily_download_limit === -1 ? '无限制访问' : '专属资源访问' }}
                </li>
              </ul>
              <div class="mt-5">
                <NButton
                  type="primary"
                  size="large"
                  :secondary="isCurrentPlan(plan)"
                  :disabled="isCurrentPlan(plan)"
                  block
                  class="sm:w-auto"
                  @click="handleBuyClick(plan)"
                >
                  {{ isCurrentPlan(plan) ? '当前方案' : '立即开通' }}
                </NButton>
              </div>
            </div>
          </div>
        </div>
      </div>

      <!-- 底部提示 -->
      <p class="mt-4 text-xs text-zinc-400 dark:text-zinc-500">会员过期后购买新方案将自动延长有效期，重复购买同一方案会叠加天数</p>
    </template>

    <!-- 购买弹窗 -->
    <MembershipModal
      v-if="selectedPlan && profile"
      :plan="selectedPlan"
      :profile="profile"
      @close="selectedPlan = null"
      @success="handleBuySuccess"
    />
  </div>
</template>

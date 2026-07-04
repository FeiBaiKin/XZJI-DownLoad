<script setup lang="ts">
// 会员中心：当前会员状态 + 会员方案卡片（余额购买）
// 业务逻辑保持不变；UI 重构为 Naive UI + Tailwind
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
const totalDays = computed(() => {
  if (!hasMembership.value) return 30
  const d = new Date(membership.value!.membership!.expiresAt).getTime()
  const base = d - daysLeft.value * 86400000
  return Math.ceil((d - base) / 86400000) || 30
})
const progressPct = computed(() => {
  if (!hasMembership.value) return 0
  return Math.min(100, Math.max(0, Math.round((daysLeft.value / (totalDays.value || 30)) * 100)))
})
const limit = computed(() => membership.value?.dailyLimit)
const limitText = computed(() => (limit.value === -1 ? '无限' : `${limit.value} 次`))
const currentPlanColor = computed(() => {
  if (!hasMembership.value) return null
  return (membership.value?.configs || []).find((c) => c.level_name === membership.value!.membership!.level)?.color || null
})
const configs = computed(() => membership.value?.configs || [])
const recommendedIdx = computed(() => (configs.value.length >= 2 ? 1 : 0))

function isCurrentPlan(plan: MembershipConfig) {
  return hasMembership.value && membership.value!.membership!.level === plan.level_name
}

function handleBuyClick(plan: MembershipConfig) { selectedPlan.value = plan }

function handleBuySuccess(data: unknown) {
  const d = data as { membershipExpiresAt: string; balance: number }
  message.value = `购买成功！有效期至 ${new Date(d.membershipExpiresAt).toLocaleDateString('zh-CN')}`
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
</script>

<template>
  <div>
    <div v-if="loading" class="space-y-6">
      <NSkeleton text width="20%" size="large" />
      <NSkeleton height="120px" :sharp="false" />
      <div class="grid gap-4 md:grid-cols-3">
        <NSkeleton v-for="i in 3" :key="i" height="260px" :sharp="false" />
      </div>
    </div>

    <template v-else>
      <h2 class="text-xl font-bold tracking-tight text-zinc-900 dark:text-zinc-50">会员中心</h2>
      <p class="mt-1 text-sm text-zinc-500 dark:text-zinc-400">选择适合你的方案，解锁付费链接下载与专属权益</p>

      <NAlert
        v-if="message"
        :type="message.startsWith('购买成功') ? 'success' : 'error'"
        class="mt-4"
        closable
        @close="message = ''"
      >{{ message }}</NAlert>

      <!-- 当前状态 -->
      <section
        class="mt-6 rounded-2xl border p-6"
        :class="hasMembership
          ? 'border-emerald-200 bg-emerald-50/50 dark:border-emerald-400/25 dark:bg-emerald-400/5'
          : 'border-zinc-200 bg-white dark:border-zinc-800 dark:bg-zinc-900'"
      >
        <template v-if="hasMembership">
          <div class="flex flex-wrap items-center gap-3">
            <NTag type="success" size="small" round>当前会员</NTag>
            <span class="text-xl font-bold tracking-tight" :style="currentPlanColor ? { color: currentPlanColor } : {}">
              {{ membership!.membership!.level }}
            </span>
          </div>

          <div class="mt-5 flex flex-wrap gap-x-10 gap-y-4">
            <div>
              <p class="font-mono text-2xl font-semibold text-zinc-900 dark:text-zinc-50">{{ daysLeft }}</p>
              <p class="mt-0.5 text-xs text-zinc-500 dark:text-zinc-400">剩余天数</p>
            </div>
            <div>
              <p class="font-mono text-2xl font-semibold text-zinc-900 dark:text-zinc-50">
                {{ membership!.downloads }}<span class="text-sm text-zinc-400">/{{ limitText }}</span>
              </p>
              <p class="mt-0.5 text-xs text-zinc-500 dark:text-zinc-400">今日下载</p>
            </div>
            <div>
              <p class="font-mono text-2xl font-semibold text-zinc-900 dark:text-zinc-50">{{ expiryShort() }}</p>
              <p class="mt-0.5 text-xs text-zinc-500 dark:text-zinc-400">到期日期</p>
            </div>
          </div>

          <div class="mt-5 h-1.5 overflow-hidden rounded-full bg-zinc-200/70 dark:bg-zinc-800">
            <div
              class="h-full rounded-full transition-all"
              :style="{ width: progressPct + '%', background: currentPlanColor || 'var(--color-emerald-500)' }"
            />
          </div>
        </template>

        <template v-else>
          <div class="flex items-center gap-3">
            <span class="grid size-10 place-items-center rounded-full bg-zinc-100 text-zinc-400 dark:bg-zinc-800 dark:text-zinc-500">
              <NIcon :size="20"><Crown /></NIcon>
            </span>
            <div>
              <p class="text-base font-semibold text-zinc-700 dark:text-zinc-300">还不是会员</p>
              <p class="mt-0.5 text-sm text-zinc-500 dark:text-zinc-400">开通会员后，可免费下载付费链接内容，并享受专属折扣</p>
            </div>
          </div>
        </template>
      </section>

      <!-- 方案卡片 -->
      <div class="mt-6 grid gap-4 md:grid-cols-2 xl:grid-cols-3">
        <div
          v-for="(plan, i) in configs"
          :key="plan.level_name"
          class="relative flex flex-col overflow-hidden rounded-2xl border bg-white p-6 transition-shadow hover:shadow-md dark:bg-zinc-900"
          :class="i === recommendedIdx
            ? 'border-emerald-400/70 shadow-sm dark:border-emerald-400/50'
            : 'border-zinc-200 dark:border-zinc-800'"
        >
          <!-- 顶部色条：方案标识色 -->
          <span aria-hidden="true" class="absolute inset-x-0 top-0 h-1" :style="{ background: plan.color || 'var(--color-zinc-300)' }" />

          <div class="flex items-center justify-between">
            <span class="text-base font-semibold" :style="{ color: plan.color || undefined }">{{ plan.level_name }}</span>
            <NTag v-if="isCurrentPlan(plan)" size="small" round>当前</NTag>
            <NTag v-else-if="i === recommendedIdx" type="success" size="small" round>推荐</NTag>
          </div>

          <p class="mt-4">
            <span class="align-top text-base font-medium text-zinc-400">¥</span>
            <span class="font-mono text-4xl font-bold tracking-tight text-zinc-900 dark:text-zinc-50">{{ Number(plan.price).toFixed(0) }}</span>
            <span class="ml-1.5 text-sm text-zinc-400">/ {{ periodText(plan.days) }}</span>
          </p>

          <ul class="mt-5 flex-1 space-y-2.5 text-sm text-zinc-600 dark:text-zinc-300">
            <li class="flex items-center gap-2">
              <NIcon :size="15" class="shrink-0 text-emerald-500"><Check /></NIcon>
              每日 {{ plan.daily_download_limit === -1 ? '无限' : plan.daily_download_limit }} 次下载
            </li>
            <li v-if="discountPct(plan)" class="flex items-center gap-2">
              <NIcon :size="15" class="shrink-0 text-emerald-500"><Check /></NIcon>
              付费链接 {{ discountPct(plan) }} 折优惠
            </li>
            <li class="flex items-center gap-2">
              <NIcon :size="15" class="shrink-0 text-emerald-500"><Check /></NIcon>
              会员专属标识
            </li>
            <li v-if="plan.daily_download_limit === -1" class="flex items-center gap-2">
              <NIcon :size="15" class="shrink-0 text-emerald-500"><Check /></NIcon>
              无限制访问全部内容
            </li>
          </ul>

          <NButton
            class="mt-6"
            block
            size="large"
            :type="i === recommendedIdx ? 'primary' : 'default'"
            :secondary="i !== recommendedIdx"
            :disabled="isCurrentPlan(plan)"
            @click="handleBuyClick(plan)"
          >
            {{ isCurrentPlan(plan) ? '当前方案' : '立即开通' }}
          </NButton>
        </div>
      </div>

      <!-- 购买弹窗 -->
      <MembershipModal
        v-if="selectedPlan && profile"
        :plan="selectedPlan"
        :profile="profile"
        @close="selectedPlan = null"
        @success="handleBuySuccess"
      />
    </template>
  </div>
</template>

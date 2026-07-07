<script setup lang="ts">
// 会员中心：状态卡 + 方案定价区，避免等高网格 + 通用功能列表
import { ref, computed, onMounted } from 'vue'
import { NButton, NIcon, NTag, NSkeleton, NProgress } from 'naive-ui'
import { Crown, Infinity as InfinityIcon, Download, Discount } from '@vicons/tabler'
import api from '@/api'
import { useAuthStore } from '@/stores/auth'
import { useToastStore } from '@/stores/toast'
import type { MembershipConfig, UserBrief } from '@/types/api'
import MembershipModal from '@/components/MembershipModal.vue'

const auth = useAuthStore()
const toast = useToastStore()

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
    toast.error('加载失败，请刷新重试')
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
const limitText = computed(() => (limit.value === -1 || limit.value === 0 ? '无限' : `${limit.value} 次`))
const currentPlanColor = computed(() => {
  if (!hasMembership.value) return null
  return (membership.value?.configs || []).find((c) => c.level_name === membership.value!.membership!.level)?.color || null
})
const configs = computed(() => membership.value?.configs || [])

// 当前套餐剩余天数占套餐总天数的比例（用于进度环）
const expiryProgress = computed(() => {
  if (!hasMembership.value || !membership.value) return 0
  const currentCfg = configs.value.find((c) => c.level_name === membership.value!.membership!.level)
  if (!currentCfg || currentCfg.days <= 0) return 100
  return Math.min(100, Math.max(0, Math.round((daysLeft.value / currentCfg.days) * 100)))
})

function isCurrentPlan(plan: MembershipConfig) {
  return hasMembership.value && membership.value!.membership!.level === plan.level_name
}
function handleBuyClick(plan: MembershipConfig) {
  if (isCurrentPlan(plan)) return
  selectedPlan.value = plan
}

function handleBuySuccess(data: unknown) {
  const d = data as { membershipExpiresAt: string; balance: number }
  message.value = `开通成功，会员有效期至 ${new Date(d.membershipExpiresAt).toLocaleDateString('zh-CN')}`
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
// 年化单价（元/年），用于横向比较
function annualized(plan: MembershipConfig) {
  if (plan.days <= 0) return null
  const perYear = (plan.price / plan.days) * 365
  return perYear
}
function discountPct(plan: MembershipConfig) {
  return plan.discount > 0 && plan.discount < 1 ? Math.round(plan.discount * 100) : null
}
function expiryShort() {
  return new Date(membership.value!.membership!.expiresAt).toLocaleDateString('zh-CN', { year: 'numeric', month: 'short', day: 'numeric' })
}

// 推荐档位：取价格中间档；至少两档才有「推荐」
function recommendedIdx(total: number) {
  if (total < 2) return -1
  return Math.floor(total / 2)
}

// 每个方案的真实功能差异点（取自配置数据，不写假通用功能）
function planFeatures(plan: MembershipConfig) {
  const feats: { icon: typeof Download; text: string }[] = []
  const dl = plan.daily_download_limit
  feats.push({
    icon: Download,
    text: dl === -1 || dl === 0 ? '每日下载不限量' : `每日下载 ${dl} 次`,
  })
  const d = discountPct(plan)
  feats.push({
    icon: Discount,
    text: d ? `付费链接享 ${d} 折` : '付费链接标准访问',
  })
  feats.push({
    icon: InfinityIcon,
    text: plan.days >= 365 ? '长期有效 · 可续费叠加' : `有效期 ${plan.days} 天 · 可叠加续费`,
  })
  return feats
}

// 当前会员档位的折扣文本（状态卡）
const currentDiscountText = computed(() => {
  if (!hasMembership.value || !membership.value) return '标准价'
  const cfg = configs.value.find((c) => c.level_name === membership.value!.membership!.level)
  if (!cfg) return '标准价'
  const d = discountPct(cfg)
  return d ? `${d} 折` : '标准价'
})
</script>

<template>
  <div>
    <!-- 加载骨架 -->
    <div v-if="loading" class="space-y-6">
      <NSkeleton text width="22%" size="large" />
      <NSkeleton height="140px" :sharp="false" />
      <div class="grid gap-5 sm:grid-cols-2 xl:grid-cols-3">
        <NSkeleton v-for="i in (configs.length || 3)" :key="i" height="260px" :sharp="false" />
      </div>
    </div>

    <template v-else>
      <!-- 顶部页眉：左对齐，非居中 -->
      <header class="flex flex-col gap-1">
        <h1 class="text-2xl font-bold tracking-tight text-zinc-900 sm:text-[28px] dark:text-zinc-50">会员中心</h1>
        <p class="max-w-[52ch] text-sm text-zinc-500 dark:text-zinc-400">开通会员后可免费下载付费资源，并享受专属折扣。重复购买同档位会叠加有效期。</p>
      </header>

      <!-- 成功提示 -->
      <div
        v-if="message"
        class="mt-4 flex items-center justify-between gap-3 rounded-xl border border-emerald-200 bg-emerald-50 px-4 py-2.5 text-sm text-emerald-700 dark:border-emerald-400/20 dark:bg-emerald-400/10 dark:text-emerald-400"
      >
        <span>{{ message }}</span>
        <button type="button" class="shrink-0 text-emerald-600/70 hover:text-emerald-700 dark:text-emerald-400/70" @click="message = ''">关闭</button>
      </div>

      <!-- 状态卡：左对齐grade，会员时展示进度环与下载用量 -->
      <section
        class="mt-6 overflow-hidden rounded-2xl border"
        :class="hasMembership
          ? 'border-zinc-200 bg-white dark:border-zinc-800 dark:bg-zinc-900'
          : 'border-dashed border-zinc-300 bg-zinc-50/60 dark:border-zinc-700 dark:bg-zinc-900/40'"
      >
        <div class="flex flex-col gap-6 p-6 lg:flex-row lg:items-stretch lg:gap-8">
          <!-- 左：等级标识 -->
          <div class="flex items-center gap-4 lg:w-[260px] lg:shrink-0">
            <span
              class="grid size-14 shrink-0 place-items-center rounded-2xl text-white"
              :style="hasMembership ? { backgroundColor: currentPlanColor || '#10b981' } : {}"
              :class="hasMembership ? '' : 'bg-zinc-100 text-zinc-400 dark:bg-zinc-800 dark:text-zinc-500'"
            >
              <NIcon :size="28"><Crown /></NIcon>
            </span>
            <div class="min-w-0">
              <div v-if="hasMembership" class="flex items-center gap-2">
                <span class="truncate text-xl font-bold tracking-tight" :style="currentPlanColor ? { color: currentPlanColor } : {}">
                  {{ membership!.membership!.level }}
                </span>
                <NTag type="success" size="small" round :bordered="false">生效中</NTag>
              </div>
              <p v-else class="text-base font-semibold text-zinc-700 dark:text-zinc-300">尚未开通</p>
              <p class="mt-1 text-xs text-zinc-500 dark:text-zinc-400">
                {{ hasMembership ? `剩余 ${daysLeft} 天 · 到期 ${expiryShort()}` : '选择下方档位以开通权益' }}
              </p>
            </div>
          </div>

          <!-- 右：详情指标（仅会员时渲染） -->
          <div v-if="hasMembership" class="grid flex-1 grid-cols-1 gap-4 sm:grid-cols-3 sm:items-center">
            <!-- 有效期进度环 -->
            <div class="flex items-center gap-3">
              <div class="relative size-12 shrink-0">
                <NProgress
                  type="circle"
                  :percentage="expiryProgress"
                  :stroke-width="6"
                  :show-indicator="false"
                  :color="currentPlanColor || '#10b981'"
                  rail-color="var(--color-zinc-200)"
                />
                <span class="absolute inset-0 grid place-items-center font-mono text-xs font-semibold text-zinc-700 dark:text-zinc-200">
                  {{ daysLeft }}
                </span>
              </div>
              <div class="min-w-0">
                <p class="text-xs text-zinc-500 dark:text-zinc-400">剩余天数</p>
                <p class="truncate text-sm font-medium text-zinc-800 dark:text-zinc-200">{{ expiryShort() }} 到期</p>
              </div>
            </div>

            <!-- 今日下载用量 -->
            <div class="flex items-center gap-3">
              <span class="grid size-12 shrink-0 place-items-center rounded-xl bg-zinc-100 text-zinc-500 dark:bg-zinc-800 dark:text-zinc-300">
                <NIcon :size="22"><Download /></NIcon>
              </span>
              <div class="min-w-0">
                <p class="text-xs text-zinc-500 dark:text-zinc-400">今日下载</p>
                <p class="truncate text-sm font-medium text-zinc-800 dark:text-zinc-200">
                  <span class="font-mono">{{ membership!.downloads }}</span>
                  <span class="text-zinc-500 dark:text-zinc-400"> / {{ limitText }}</span>
                </p>
              </div>
            </div>

            <!-- 折扣权益 -->
            <div class="flex items-center gap-3">
              <span class="grid size-12 shrink-0 place-items-center rounded-xl bg-zinc-100 text-zinc-500 dark:bg-zinc-800 dark:text-zinc-300">
                <NIcon :size="22"><Discount /></NIcon>
              </span>
              <div class="min-w-0">
                <p class="text-xs text-zinc-500 dark:text-zinc-400">付费链接折扣</p>
                <p class="truncate text-sm font-medium text-zinc-800 dark:text-zinc-200">{{ currentDiscountText }}</p>
              </div>
            </div>
          </div>
          <div v-else class="flex flex-1 items-center lg:justify-end">
            <p class="text-sm text-zinc-500 dark:text-zinc-400 lg:max-w-xs">
              会员权益包含付费内容免费下载、专属折扣与不限量访问，按档位区分额度。
            </p>
          </div>
        </div>
      </section>

      <!-- 方案定价区：响应式卡片，差异真实化 -->
      <div class="mt-8 grid gap-5 sm:grid-cols-2 xl:grid-cols-3">
        <article
          v-for="(plan, i) in configs"
          :key="plan.level_name"
          class="relative flex flex-col overflow-hidden rounded-2xl border bg-white transition-all hover:-translate-y-0.5 hover:shadow-lg dark:bg-zinc-900"
          :class="[
            isCurrentPlan(plan)
              ? 'border-zinc-300 dark:border-zinc-700'
              : (i === recommendedIdx(configs.length)
                  ? 'border-emerald-300 bg-emerald-50/30 dark:border-emerald-400/25 dark:bg-emerald-400/5'
                  : 'border-zinc-200 dark:border-zinc-800')
          ]"
        >
          <!-- 顶部色条 -->
          <div class="h-1 w-full" :style="{ background: plan.color || '#10b981' }" />

          <div class="flex flex-1 flex-col p-6">
            <!-- 头部：名称 + 标记 -->
            <div class="flex items-center justify-between gap-2">
              <span
                class="truncate text-lg font-bold tracking-tight"
                :style="plan.color ? { color: plan.color } : {}"
                :class="!plan.color ? 'text-zinc-900 dark:text-zinc-100' : ''"
              >
                {{ plan.level_name }}
              </span>
              <NTag v-if="isCurrentPlan(plan)" size="small" round type="success" :bordered="false">当前</NTag>
              <NTag v-else-if="i === recommendedIdx(configs.length)" size="small" round :bordered="false" class="!bg-emerald-100 !text-emerald-700 dark:!bg-emerald-400/15 dark:!text-emerald-400">推荐</NTag>
            </div>

            <!-- 价格 -->
            <div class="mt-4 flex items-baseline gap-1">
              <span class="text-sm font-medium text-zinc-400 dark:text-zinc-500">¥</span>
              <span class="font-mono text-4xl font-bold tracking-tight text-zinc-900 dark:text-zinc-50">{{ plan.price }}</span>
              <span class="ml-1 text-sm text-zinc-400 dark:text-zinc-500">/ {{ periodText(plan.days) }}</span>
            </div>
            <p v-if="annualized(plan)" class="mt-1 text-xs text-zinc-500 dark:text-zinc-400">
              约 ¥{{ annualized(plan)!.toFixed(0) }} / 年
            </p>

            <!-- 功能差异点 -->
            <ul class="mt-5 space-y-2.5">
              <li
                v-for="(f, fi) in planFeatures(plan)"
                :key="fi"
                class="flex items-start gap-2 text-sm text-zinc-600 dark:text-zinc-400"
              >
                <NIcon :size="16" class="mt-0.5 shrink-0" :style="{ color: plan.color || '#10b981' }">
                  <component :is="f.icon" />
                </NIcon>
                <span>{{ f.text }}</span>
              </li>
            </ul>

            <!-- CTA -->
            <div class="mt-6 flex-1" />
            <div>
              <NButton
                :type="isCurrentPlan(plan) ? 'default' : 'primary'"
                size="large"
                block
                :disabled="isCurrentPlan(plan)"
                :secondary="isCurrentPlan(plan)"
                @click="handleBuyClick(plan)"
              >
                {{ isCurrentPlan(plan) ? '当前方案' : '立即开通' }}
              </NButton>
            </div>
          </div>
        </article>
      </div>

      <!-- 空状态 -->
      <div
        v-if="!loading && configs.length === 0"
        class="mt-8 rounded-2xl border border-dashed border-zinc-300 py-16 text-center dark:border-zinc-700"
      >
        <p class="text-sm text-zinc-500 dark:text-zinc-400">暂未配置会员档位，请稍后再来。</p>
      </div>

      <!-- 底部说明 -->
      <p v-if="configs.length > 0" class="mt-6 text-xs text-zinc-500 dark:text-zinc-400">
        会员到期前续费将自动延长有效期；不同档位之间切换不退还差价，请按需选择。
      </p>
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
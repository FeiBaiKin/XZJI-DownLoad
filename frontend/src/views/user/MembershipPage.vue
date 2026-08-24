<script setup lang="ts">
// 会员中心：深色会员卡横幅（会员 / 非会员双态）+ 档位定价卡 + 购买须知
// 业务逻辑与后端契约不变：GET /user/membership + /user/profile，POST /user/buy-membership
// 视觉体系：卡片 rounded-2xl / 控件 rounded-lg / 标签 rounded-full；emerald 为站点强调色，档位色为数据驱动语义色
import { ref, computed, onMounted } from 'vue'
import { NButton, NIcon, NTag, NSkeleton } from 'naive-ui'
import { PhCrown, PhCheck, PhCaretRight, PhDownloadSimple, PhPercent, PhCalendar, PhCalendarPlus, PhShieldCheck } from '@phosphor-icons/vue'
import api from '@/api'
import { useAuthStore } from '@/stores/auth'
import { useToastStore } from '@/stores/toast'
import type { MembershipConfig, UserBrief } from '@/types/api'
import { TIER_FALLBACK_ICON, tierIcon, tierColor, tierWash, readableOn, periodText, discountLabel, priceText, perDayText, badgeText } from '@/utils/tier'
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

const configs = computed(() => membership.value?.configs || [])
const hasMembership = computed(() => !!membership.value?.membership)
const daysLeft = computed(() => {
  if (!hasMembership.value) return 0
  return Math.max(0, Math.ceil((new Date(membership.value!.membership!.expiresAt).getTime() - Date.now()) / 86400000))
})
const limit = computed(() => membership.value?.dailyLimit ?? 0)
const isUnlimited = computed(() => limit.value === -1 || limit.value === 0)

const currentCfg = computed(() => {
  if (!hasMembership.value) return null
  return configs.value.find((c) => c.level_name === membership.value!.membership!.level) || null
})
const currentIdx = computed(() => {
  if (!hasMembership.value) return 0
  const i = configs.value.findIndex((c) => c.level_name === membership.value!.membership!.level)
  return i >= 0 ? i : 0
})
const heroColor = computed(() => tierColor(currentCfg.value?.color))

// 有效期剩余进度（相对购买档位总天数）
const expiryProgress = computed(() => {
  if (!currentCfg.value || currentCfg.value.days <= 0) return 100
  return Math.min(100, Math.max(0, Math.round((daysLeft.value / currentCfg.value.days) * 100)))
})
// 今日下载用量百分比（不限量时不渲染进度条）
const dlUsagePct = computed(() => {
  if (isUnlimited.value || !limit.value) return 0
  return Math.min(100, Math.max(0, Math.round(((membership.value?.downloads ?? 0) / limit.value) * 100)))
})
const currentDiscountText = computed(() => {
  if (!currentCfg.value) return '标准价'
  return discountLabel(currentCfg.value.discount) ?? '标准价'
})

function expiryShort() {
  return new Date(membership.value!.membership!.expiresAt).toLocaleDateString('zh-CN', { year: 'numeric', month: 'short', day: 'numeric' })
}

// ── 档位卡片 ──
const gridClass = computed(() => {
  const n = configs.value.length
  if (n === 1) return 'mx-auto max-w-sm'
  if (n === 2 || n === 4) return 'sm:grid-cols-2'
  return 'sm:grid-cols-2 lg:grid-cols-3'
})

function isCurrentPlan(plan: MembershipConfig) {
  return hasMembership.value && membership.value!.membership!.level === plan.level_name
}
function planBadge(plan: MembershipConfig) {
  return badgeText(plan.badge)
}
function dlBenefit(plan: MembershipConfig) {
  const d = plan.daily_download_limit
  return d === -1 || d === 0 ? '每日下载不限量' : `每日免费下载 ${d} 次`
}
function discountBenefit(plan: MembershipConfig) {
  const d = discountLabel(plan.discount)
  return d ? `付费链接享 ${d}` : '付费链接标准价'
}

function handleBuyClick(plan: MembershipConfig) {
  if (isCurrentPlan(plan)) return
  selectedPlan.value = plan
}

function handleBuySuccess(data: unknown) {
  // 余额开通返回 membershipExpiresAt；第三方直付由弹窗回读 profile 后传同名字段
  const d = data as { membershipExpiresAt?: string | null; balance?: number }
  message.value = d.membershipExpiresAt
    ? `开通成功，会员有效期至 ${new Date(d.membershipExpiresAt).toLocaleDateString('zh-CN')}`
    : '开通成功'
  if (auth.user) {
    if (typeof d.balance === 'number') auth.user.balance = d.balance
    auth.user.membership_level = selectedPlan.value?.level_name ?? auth.user.membership_level
    if (d.membershipExpiresAt) auth.user.membership_expires_at = d.membershipExpiresAt
  }
  selectedPlan.value = null
  fetchMembership()
}
</script>

<template>
  <div>
    <!-- 加载骨架：横幅 + 三卡 -->
    <div v-if="loading" class="space-y-6">
      <NSkeleton text width="22%" size="large" />
      <NSkeleton height="176px" :sharp="false" />
      <div class="grid gap-5 sm:grid-cols-2 lg:grid-cols-3">
        <NSkeleton v-for="i in 3" :key="i" height="300px" :sharp="false" />
      </div>
    </div>

    <template v-else>
      <!-- 页眉 -->
      <header class="flex flex-col gap-1">
        <h1 class="text-xl font-bold tracking-tight text-zinc-900 dark:text-zinc-50">会员中心</h1>
        <p class="max-w-[58ch] text-sm leading-relaxed text-zinc-500 dark:text-zinc-400">
          开通会员后可免费下载付费资源，购买付费链接享专属折扣。
        </p>
      </header>

      <!-- 开通成功提示 -->
      <div
        v-if="message"
        class="mt-5 flex items-center justify-between gap-3 rounded-2xl border border-emerald-200 bg-emerald-50 px-4 py-2.5 text-sm text-emerald-700 dark:border-emerald-400/20 dark:bg-emerald-400/10 dark:text-emerald-400"
      >
        <span class="flex items-center gap-2"><NIcon :size="16"><PhCheck /></NIcon>{{ message }}</span>
        <button
          type="button"
          class="shrink-0 rounded text-emerald-600/80 hover:text-emerald-700 focus-visible:outline-2 focus-visible:outline-offset-2 focus-visible:outline-emerald-500 dark:text-emerald-400/80 dark:hover:text-emerald-300"
          @click="message = ''"
        >关闭</button>
      </div>

      <!-- 会员卡横幅：刻意的深色卡片对象，两种主题下都是深色（像一张实体会员卡） -->
      <section class="relative mt-6 overflow-hidden rounded-2xl bg-zinc-950 dark:border dark:border-zinc-800">
        <!-- 档位色 / emerald 光晕 -->
        <div
          aria-hidden="true"
          class="pointer-events-none absolute -top-24 -right-16 size-72 rounded-full blur-3xl"
          :style="{ background: `radial-gradient(closest-side, color-mix(in srgb, ${hasMembership ? heroColor : '#10b981'} 42%, transparent), transparent)` }"
        />
        <div
          aria-hidden="true"
          class="pointer-events-none absolute -bottom-28 -left-20 size-72 rounded-full blur-3xl"
          :style="{ background: `radial-gradient(closest-side, color-mix(in srgb, ${hasMembership ? heroColor : '#10b981'} 22%, transparent), transparent)` }"
        />
        <!-- 底纹图标 -->
        <NIcon aria-hidden="true" :size="190" class="pointer-events-none absolute -right-8 -bottom-12 rotate-12 text-white/[0.05]">
          <component :is="hasMembership ? tierIcon(currentIdx) : TIER_FALLBACK_ICON" />
        </NIcon>

        <div class="relative p-6 sm:p-7">
          <!-- 会员态 -->
          <template v-if="hasMembership">
            <div class="flex flex-wrap items-center gap-4">
              <span
                class="grid size-12 shrink-0 place-items-center rounded-xl shadow-lg"
                :style="{ backgroundColor: heroColor, color: readableOn(heroColor), boxShadow: `0 8px 24px -8px color-mix(in srgb, ${heroColor} 60%, transparent)` }"
              >
                <NIcon :size="24"><component :is="tierIcon(currentIdx)" /></NIcon>
              </span>
              <div class="min-w-0 flex-1">
                <div class="flex flex-wrap items-center gap-2.5">
                  <h2 class="truncate text-lg font-bold tracking-tight text-white sm:text-xl">{{ membership!.membership!.level }}</h2>
                  <span class="rounded-full bg-emerald-400/15 px-2.5 py-1 text-[11px] font-medium leading-none text-emerald-300">生效中</span>
                </div>
                <p class="mt-1 flex items-center gap-1.5 text-xs text-zinc-400">
                  <NIcon :size="13"><PhCalendar /></NIcon>
                  {{ expiryShort() }} 到期 · 剩余 {{ daysLeft }} 天
                </p>
              </div>
            </div>

            <!-- 三项指标：玻璃质感信息块 -->
            <div class="mt-6 grid gap-3 sm:grid-cols-3">
              <div class="rounded-xl border border-white/10 bg-white/[0.06] px-4 py-3.5 backdrop-blur-sm">
                <p class="text-xs text-zinc-400">有效期剩余</p>
                <p class="mt-1.5 font-mono text-xl font-bold text-white">
                  {{ daysLeft }}<span class="ml-1 text-xs font-normal text-zinc-400">/ {{ currentCfg?.days ?? daysLeft }} 天</span>
                </p>
                <div v-if="currentCfg" class="mt-2.5 h-1.5 overflow-hidden rounded-full bg-white/10">
                  <div class="h-full rounded-full transition-all" :style="{ width: `${expiryProgress}%`, backgroundColor: heroColor }" />
                </div>
              </div>
              <div class="rounded-xl border border-white/10 bg-white/[0.06] px-4 py-3.5 backdrop-blur-sm">
                <p class="text-xs text-zinc-400">今日下载</p>
                <p class="mt-1.5 font-mono text-xl font-bold text-white">
                  {{ membership!.downloads }}<span class="ml-1 text-xs font-normal text-zinc-400">/ {{ isUnlimited ? '不限量' : `${limit} 次` }}</span>
                </p>
                <div v-if="!isUnlimited" class="mt-2.5 h-1.5 overflow-hidden rounded-full bg-white/10">
                  <div class="h-full rounded-full bg-emerald-400 transition-all" :style="{ width: `${dlUsagePct}%` }" />
                </div>
                <p v-else class="mt-2.5 text-[11px] text-zinc-500">额度不设上限</p>
              </div>
              <div class="rounded-xl border border-white/10 bg-white/[0.06] px-4 py-3.5 backdrop-blur-sm">
                <p class="text-xs text-zinc-400">付费链接折扣</p>
                <p class="mt-1.5 text-xl font-bold text-white">{{ currentDiscountText }}</p>
                <p class="mt-2.5 flex items-center gap-1 text-[11px] text-zinc-500">
                  <NIcon :size="12"><PhPercent /></NIcon>结算时自动应用
                </p>
              </div>
            </div>
          </template>

          <!-- 非会员态 -->
          <div v-else class="flex flex-col gap-6 py-2 sm:flex-row sm:items-center sm:justify-between sm:py-4">
            <div class="flex items-center gap-4">
              <span class="grid size-12 shrink-0 place-items-center rounded-xl border border-white/10 bg-white/[0.06] text-zinc-300">
                <NIcon :size="24"><PhCrown weight="duotone" /></NIcon>
              </span>
              <div>
                <h2 class="text-lg font-bold tracking-tight text-white sm:text-xl">尚未开通会员</h2>
                <p class="mt-1 text-sm text-zinc-400">开通后免费下载付费资源，购买付费链接享专属折扣。</p>
              </div>
            </div>
            <a
              href="#plans"
              class="inline-flex items-center gap-1.5 self-start rounded-lg bg-white px-4 py-2.5 text-sm font-semibold text-zinc-950 transition hover:bg-zinc-200 active:scale-[0.98] sm:self-auto focus-visible:outline-2 focus-visible:outline-offset-2 focus-visible:outline-emerald-400"
            >
              选择档位
              <NIcon :size="15"><PhCaretRight /></NIcon>
            </a>
          </div>
        </div>
      </section>

      <!-- 档位定价卡 -->
      <section id="plans" class="mt-10 scroll-mt-6">
        <div class="flex items-baseline justify-between gap-3">
          <h2 class="text-lg font-semibold tracking-tight text-zinc-900 dark:text-zinc-100">{{ hasMembership ? '续费或升级' : '选择适合你的档位' }}</h2>
          <p class="text-xs text-zinc-500 dark:text-zinc-400">共 {{ configs.length }} 个档位</p>
        </div>

        <!-- 空状态 -->
        <div
          v-if="configs.length === 0"
          class="mt-4 rounded-2xl border border-dashed border-zinc-300 bg-zinc-50/40 py-16 text-center dark:border-zinc-700 dark:bg-zinc-900/40"
        >
          <span class="mx-auto grid size-13 place-items-center rounded-xl bg-zinc-100 text-zinc-400 dark:bg-zinc-800 dark:text-zinc-500">
            <NIcon :size="26"><PhCrown weight="duotone" /></NIcon>
          </span>
          <p class="mt-4 text-sm font-medium text-zinc-600 dark:text-zinc-300">暂未开放会员档位</p>
          <p class="mt-1 text-xs text-zinc-500 dark:text-zinc-400">管理员尚未配置档位，请稍后再来看看。</p>
        </div>

        <div v-else class="mt-4 grid gap-5" :class="gridClass">
          <article
            v-for="(plan, i) in configs"
            :key="plan.level_name"
            class="group relative flex flex-col overflow-hidden rounded-2xl border bg-white transition duration-200 hover:-translate-y-0.5 dark:bg-zinc-900"
            :class="planBadge(plan) ? 'border-transparent' : 'border-zinc-200 hover:shadow-lg hover:shadow-zinc-900/5 dark:border-zinc-800 dark:hover:shadow-black/20'"
            :style="planBadge(plan)
              ? { boxShadow: `0 0 0 2px ${tierColor(plan.color)}, 0 16px 40px -16px color-mix(in srgb, ${tierColor(plan.color)} 45%, transparent)` }
              : undefined"
          >
            <!-- 渐染头部 -->
            <div class="p-5 pb-4" :style="{ background: tierWash(plan.color) }">
              <div class="flex items-start justify-between gap-2">
                <span
                  class="grid size-11 place-items-center rounded-xl"
                  :style="{ backgroundColor: tierColor(plan.color), color: readableOn(tierColor(plan.color)) }"
                >
                  <NIcon :size="22"><component :is="tierIcon(i)" /></NIcon>
                </span>
                <NTag v-if="isCurrentPlan(plan)" size="small" round type="success" :bordered="false">当前方案</NTag>
                <span
                  v-else-if="planBadge(plan)"
                  class="rounded-full px-2.5 py-1 text-[11px] font-semibold leading-none"
                  :style="{ backgroundColor: tierColor(plan.color), color: readableOn(tierColor(plan.color)) }"
                >{{ planBadge(plan) }}</span>
              </div>
              <h3 class="mt-3 truncate text-lg font-bold tracking-tight text-zinc-900 dark:text-zinc-50">{{ plan.level_name }}</h3>
            </div>

            <!-- 价格 + 权益 -->
            <div class="flex flex-1 flex-col p-5 pt-1">
              <div class="flex items-baseline gap-1">
                <span class="text-sm font-medium text-zinc-400 dark:text-zinc-500">¥</span>
                <span class="font-mono text-4xl font-bold tracking-tight text-zinc-900 dark:text-zinc-50">{{ priceText(plan.price) }}</span>
                <span class="ml-0.5 text-sm text-zinc-400 dark:text-zinc-500">/ {{ periodText(plan.days) }}</span>
              </div>
              <p class="mt-1 h-4 text-xs text-zinc-500 dark:text-zinc-400">{{ perDayText(plan.price, plan.days) ?? '' }}</p>

              <ul class="mt-4 space-y-2.5 border-t border-zinc-100 pt-4 text-sm text-zinc-600 dark:border-zinc-800 dark:text-zinc-300">
                <li class="flex items-center gap-2.5">
                  <NIcon :size="16" class="shrink-0" :style="{ color: tierColor(plan.color) }"><PhDownloadSimple /></NIcon>
                  {{ dlBenefit(plan) }}
                </li>
                <li class="flex items-center gap-2.5">
                  <NIcon :size="16" class="shrink-0" :style="{ color: tierColor(plan.color) }"><PhPercent /></NIcon>
                  {{ discountBenefit(plan) }}
                </li>
                <li class="flex items-center gap-2.5">
                  <NIcon :size="16" class="shrink-0" :style="{ color: tierColor(plan.color) }"><PhCalendarPlus /></NIcon>
                  有效期 {{ plan.days }} 天，续费叠加
                </li>
              </ul>

              <div class="mt-5 flex-1" />
              <NButton
                v-if="!isCurrentPlan(plan)"
                block
                size="large"
                :color="tierColor(plan.color)"
                :text-color="readableOn(tierColor(plan.color))"
                @click="handleBuyClick(plan)"
              >立即开通</NButton>
              <NButton v-else block size="large" secondary disabled>当前方案</NButton>
            </div>
          </article>
        </div>
      </section>

      <!-- 购买须知 -->
      <section v-if="configs.length > 0" class="mt-10 rounded-2xl border border-zinc-200 bg-white p-6 dark:border-zinc-800 dark:bg-zinc-900">
        <h3 class="flex items-center gap-2 text-sm font-semibold text-zinc-800 dark:text-zinc-200">
          <NIcon :size="17" class="text-emerald-600 dark:text-emerald-400"><PhShieldCheck /></NIcon>
          购买须知
        </h3>
        <ul class="mt-4 grid gap-x-8 gap-y-3 text-sm text-zinc-600 sm:grid-cols-2 dark:text-zinc-400">
          <li class="flex items-start gap-2">
            <NIcon :size="15" class="mt-0.5 shrink-0 text-emerald-600 dark:text-emerald-400"><PhCheck /></NIcon>
            每日额度内免费下载付费资源，超出后按原价购买。
          </li>
          <li class="flex items-start gap-2">
            <NIcon :size="15" class="mt-0.5 shrink-0 text-emerald-600 dark:text-emerald-400"><PhCheck /></NIcon>
            购买付费链接时自动应用档位折扣，无需手动领取。
          </li>
          <li class="flex items-start gap-2">
            <NIcon :size="15" class="mt-0.5 shrink-0 text-emerald-600 dark:text-emerald-400"><PhCheck /></NIcon>
            续费同档位从当前到期日起叠加，剩余天数不损失。
          </li>
          <li class="flex items-start gap-2">
            <NIcon :size="15" class="mt-0.5 shrink-0 text-emerald-600 dark:text-emerald-400"><PhCheck /></NIcon>
            切换不同档位不退还差价，请按需选择合适档位。
          </li>
        </ul>
      </section>
    </template>

    <!-- 购买弹窗 -->
    <MembershipModal
      v-if="selectedPlan && profile"
      :plan="selectedPlan"
      :profile="profile"
      :tier-index="configs.findIndex((c) => c.level_name === selectedPlan!.level_name)"
      @close="selectedPlan = null"
      @success="handleBuySuccess"
    />
  </div>
</template>

<script setup lang="ts">
// 会员购买弹窗：余额支付；有效期叠加续期提示
// 业务逻辑不变：POST /user/buy-membership { level } -> { message, balance, membershipLevel, membershipExpiresAt }
import { ref, computed } from 'vue'
import { NModal, NButton, NIcon } from 'naive-ui'
import { X, Calendar, Download, Discount, Wallet, InfoCircle } from '@vicons/tabler'
import api from '@/api'
import type { MembershipConfig, UserBrief } from '@/types/api'
import { tierIcon, tierColor, tierWash, readableOn, periodText, dlText, discountLabel, priceText } from '@/utils/tier'

const props = withDefaults(defineProps<{ plan: MembershipConfig; profile: UserBrief; tierIndex?: number }>(), { tierIndex: 0 })
const emit = defineEmits<{ close: []; success: [data: unknown] }>()

const loading = ref(false)
const error = ref('')

const balance = computed(() => Number(props.profile?.balance ?? 0))
const price = computed(() => Number(props.plan?.price ?? 0))
const canAfford = computed(() => balance.value >= price.value)
const planColor = computed(() => tierColor(props.plan?.color))
const iconIndex = computed(() => Math.max(0, props.tierIndex))
const hasMembership = computed(() => {
  return !!props.profile?.membership_level && !!props.profile?.membership_expires_at
    && new Date(props.profile.membership_expires_at) > new Date()
})
const isCurrentPlan = computed(() => hasMembership.value && props.profile?.membership_level === props.plan?.level_name)
const discountText = computed(() => discountLabel(props.plan?.discount ?? 0))

async function handleBuy() {
  loading.value = true
  error.value = ''
  try {
    const res = await api.post('/user/buy-membership', { level: props.plan.level_name })
    emit('success', res.data)
    emit('close')
  } catch (err: unknown) {
    const errorObj = err as { response?: { data?: { error?: string } } }
    error.value = errorObj?.response?.data?.error || '购买失败'
  } finally {
    loading.value = false
  }
}
</script>

<template>
  <NModal :show="true" :mask-closable="!loading" :close-on-esc="!loading" @update:show="(v: boolean) => { if (!v) emit('close') }">
    <div class="w-[92vw] max-w-md overflow-hidden rounded-2xl border border-zinc-200 bg-white shadow-2xl dark:border-zinc-800 dark:bg-zinc-900">
      <!-- 渐染头部 -->
      <div class="p-5 pb-4" :style="{ background: tierWash(planColor) }">
        <div class="flex items-start justify-between gap-3">
          <div class="flex items-center gap-3">
            <span
              class="grid size-11 shrink-0 place-items-center rounded-xl"
              :style="{ backgroundColor: planColor, color: readableOn(planColor) }"
            >
              <NIcon :size="22"><component :is="tierIcon(iconIndex)" /></NIcon>
            </span>
            <div class="min-w-0">
              <p class="text-xs text-zinc-500 dark:text-zinc-400">开通会员</p>
              <h3 class="truncate text-lg font-bold tracking-tight text-zinc-900 dark:text-zinc-50">{{ plan.level_name }}</h3>
            </div>
          </div>
          <button
            type="button"
            class="flex size-8 shrink-0 items-center justify-center rounded-full text-zinc-400 transition-colors hover:bg-zinc-900/5 hover:text-zinc-600 focus-visible:outline-2 focus-visible:outline-offset-2 focus-visible:outline-emerald-500 dark:hover:bg-white/10 dark:hover:text-zinc-300"
            aria-label="关闭"
            @click="emit('close')"
          >
            <NIcon :size="16"><X /></NIcon>
          </button>
        </div>
      </div>

      <div class="space-y-4 px-5 pb-5">
        <!-- 档位权益 -->
        <dl class="space-y-2.5 text-sm">
          <div class="flex items-center justify-between">
            <dt class="flex items-center gap-2 text-zinc-500 dark:text-zinc-400">
              <NIcon :size="15" :style="{ color: planColor }"><Calendar /></NIcon>有效期
            </dt>
            <dd class="font-medium text-zinc-800 dark:text-zinc-200">{{ plan.days }} 天（{{ periodText(plan.days) }}）</dd>
          </div>
          <div class="flex items-center justify-between">
            <dt class="flex items-center gap-2 text-zinc-500 dark:text-zinc-400">
              <NIcon :size="15" :style="{ color: planColor }"><Download /></NIcon>每日下载
            </dt>
            <dd class="font-medium text-zinc-800 dark:text-zinc-200">{{ dlText(plan.daily_download_limit) }}</dd>
          </div>
          <div class="flex items-center justify-between">
            <dt class="flex items-center gap-2 text-zinc-500 dark:text-zinc-400">
              <NIcon :size="15" :style="{ color: planColor }"><Discount /></NIcon>付费链接折扣
            </dt>
            <dd class="font-medium text-zinc-800 dark:text-zinc-200">{{ discountText ?? '标准价' }}</dd>
          </div>
        </dl>

        <!-- 价格区 -->
        <div class="space-y-2.5 rounded-xl bg-zinc-50 px-4 py-4 dark:bg-zinc-800/60">
          <div class="flex items-center justify-between">
            <span class="flex items-center gap-1.5 text-sm text-zinc-500 dark:text-zinc-400"><NIcon :size="14"><Wallet /></NIcon>支付金额</span>
            <span class="font-mono text-xl font-bold text-zinc-900 dark:text-zinc-100">¥{{ priceText(price) }}</span>
          </div>
          <div class="flex items-center justify-between">
            <span class="text-sm text-zinc-500 dark:text-zinc-400">当前余额</span>
            <span class="font-mono text-sm" :class="canAfford ? 'text-zinc-700 dark:text-zinc-300' : 'font-semibold text-red-500 dark:text-red-400'">
              ¥{{ balance.toFixed(2) }}
            </span>
          </div>
          <div v-if="!canAfford" class="flex items-center gap-1.5 rounded-lg bg-red-50 px-2.5 py-1.5 text-xs text-red-600 dark:bg-red-500/10 dark:text-red-400">
            <NIcon :size="13"><InfoCircle /></NIcon>余额不足，请先充值后再购买
          </div>
        </div>

        <!-- 续期 / 重复提示 -->
        <p
          v-if="hasMembership && !isCurrentPlan"
          class="flex items-start gap-1.5 rounded-lg bg-amber-50 px-3 py-2.5 text-xs text-amber-700 dark:bg-amber-400/10 dark:text-amber-400"
        >
          <NIcon :size="14" class="mt-0.5 shrink-0"><InfoCircle /></NIcon>
          <span>您当前是 {{ profile.membership_level }}，购买后有效期将从现有到期日延长。</span>
        </p>
        <p
          v-if="isCurrentPlan"
          class="flex items-start gap-1.5 rounded-lg bg-zinc-100 px-3 py-2.5 text-xs text-zinc-500 dark:bg-zinc-800 dark:text-zinc-400"
        >
          <NIcon :size="14" class="mt-0.5 shrink-0"><InfoCircle /></NIcon>
          <span>您当前已是 {{ plan.level_name }}，无需重复购买。</span>
        </p>

        <p v-if="error" class="rounded-lg bg-red-50 px-3 py-2 text-xs text-red-600 dark:bg-red-500/10 dark:text-red-400">{{ error }}</p>

        <!-- 操作 -->
        <div class="space-y-2.5 pt-1">
          <NButton type="primary" block size="large" :loading="loading" :disabled="!canAfford || isCurrentPlan" @click="handleBuy">
            {{ isCurrentPlan ? '当前方案' : `确认支付 ¥${priceText(price)}` }}
          </NButton>
          <NButton quaternary block @click="emit('close')">取消</NButton>
        </div>
      </div>
    </div>
  </NModal>
</template>

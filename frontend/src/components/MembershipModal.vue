<script setup lang="ts">
// 会员购买弹窗：余额支付；有效期叠加续期提示
// 业务逻辑不变：POST /user/buy-membership { level } -> { message, balance, membershipLevel, membershipExpiresAt }
import { ref, computed } from 'vue'
import { NModal, NButton, NIcon } from 'naive-ui'
import { X, Crown, Calendar, Download, Discount, Wallet, InfoCircle } from '@vicons/tabler'
import api from '@/api'
import type { MembershipConfig, UserBrief } from '@/types/api'

const props = defineProps<{ plan: MembershipConfig; profile: UserBrief }>()
const emit = defineEmits<{ close: []; success: [data: unknown] }>()

const loading = ref(false)
const error = ref('')

const balance = computed(() => Number(props.profile?.balance ?? 0))
const price = computed(() => Number(props.plan?.price ?? 0))
const canAfford = computed(() => balance.value >= price.value)
const planColor = computed(() => props.plan?.color || '#10b981')
const daysText = computed(() => {
  const d = props.plan?.days || 0
  if (d >= 365) return `${Math.floor(d / 365)} 年`
  if (d >= 30) return `${Math.floor(d / 30)} 个月`
  return `${d} 天`
})
const hasMembership = computed(() => {
  return !!props.profile?.membership_level && !!props.profile?.membership_expires_at
    && new Date(props.profile.membership_expires_at) > new Date()
})
const isCurrentPlan = computed(() => hasMembership.value && props.profile?.membership_level === props.plan?.level_name)
const discountPct = computed(() => {
  const d = props.plan?.discount
  return d && d > 0 && d < 1 ? Math.round(d * 100) : null
})
const dlText = computed(() => {
  const d = props.plan?.daily_download_limit
  return d === -1 || d === 0 ? '无限制' : `${d} 次/天`
})

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
      <!-- 顶部色条 + 头部 -->
      <div class="h-1.5 w-full" :style="{ background: planColor }" />
      <div class="flex items-center justify-between px-6 py-4">
        <div class="flex items-center gap-2.5">
          <span class="grid size-8 place-items-center rounded-lg text-white" :style="{ backgroundColor: planColor }">
            <NIcon :size="17"><Crown /></NIcon>
          </span>
          <h3 class="text-base font-semibold text-zinc-900 dark:text-zinc-100">开通会员</h3>
        </div>
        <button
          type="button"
          class="flex size-8 items-center justify-center rounded-full text-zinc-400 transition-colors hover:bg-zinc-100 hover:text-zinc-600 focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-emerald-500/60 dark:hover:bg-zinc-800 dark:hover:text-zinc-300 dark:focus-visible:ring-emerald-400/60"
          aria-label="关闭"
          @click="emit('close')"
        >
          <NIcon :size="16"><X /></NIcon>
        </button>
      </div>

      <div class="space-y-4 px-6 pb-6">
        <!-- 方案摘要 -->
        <div class="rounded-xl border border-zinc-200 px-4 py-4 dark:border-zinc-700">
          <div class="flex items-center gap-2">
            <span class="size-2.5 rounded-full" :style="{ backgroundColor: planColor }" />
            <span class="text-lg font-bold" :style="{ color: planColor }">{{ plan.level_name }}</span>
          </div>
          <dl class="mt-3 space-y-2.5 text-sm">
            <div class="flex items-center justify-between">
              <dt class="flex items-center gap-1.5 text-zinc-500 dark:text-zinc-400"><NIcon :size="14"><Calendar /></NIcon>有效期</dt>
              <dd class="font-medium text-zinc-800 dark:text-zinc-200">{{ plan.days }} 天（{{ daysText }}）</dd>
            </div>
            <div class="flex items-center justify-between">
              <dt class="flex items-center gap-1.5 text-zinc-500 dark:text-zinc-400"><NIcon :size="14"><Download /></NIcon>每日下载</dt>
              <dd class="font-medium text-zinc-800 dark:text-zinc-200">{{ dlText }}</dd>
            </div>
            <div v-if="discountPct" class="flex items-center justify-between">
              <dt class="flex items-center gap-1.5 text-zinc-500 dark:text-zinc-400"><NIcon :size="14"><Discount /></NIcon>付费链接折扣</dt>
              <dd class="font-medium text-zinc-800 dark:text-zinc-200">{{ discountPct }} 折</dd>
            </div>
          </dl>
        </div>

        <!-- 价格区 -->
        <div class="space-y-2.5 rounded-xl bg-zinc-50 px-4 py-4 dark:bg-zinc-800/60">
          <div class="flex items-center justify-between">
            <span class="flex items-center gap-1.5 text-sm text-zinc-500 dark:text-zinc-400"><NIcon :size="14"><Wallet /></NIcon>支付金额</span>
            <span class="font-mono text-xl font-bold text-zinc-900 dark:text-zinc-100">¥{{ price.toFixed(0) }}</span>
          </div>
          <div class="flex items-center justify-between">
            <span class="text-sm text-zinc-500 dark:text-zinc-400">当前余额</span>
            <span class="font-mono" :class="canAfford ? 'text-zinc-700 dark:text-zinc-300' : 'text-red-500 dark:text-red-400'">
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
          <NButton
            type="primary"
            block
            size="large"
            :loading="loading"
            :disabled="!canAfford || isCurrentPlan"
            class="!bg-emerald-600 hover:!bg-emerald-700 dark:!bg-emerald-500 dark:hover:!bg-emerald-400"
            @click="handleBuy"
          >
            {{ isCurrentPlan ? '当前方案' : `确认支付 ¥${price.toFixed(0)}` }}
          </NButton>
          <NButton quaternary block size="large" @click="emit('close')">取消</NButton>
        </div>
      </div>
    </div>
  </NModal>
</template>

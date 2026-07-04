<script setup lang="ts">
// 会员购买弹窗：余额支付；有效期叠加续期提示
// 业务逻辑不变，UI 重构为 Naive UI + Tailwind
import { ref, computed } from 'vue'
import { NModal, NButton, NIcon } from 'naive-ui'
import { X, Crown } from '@vicons/tabler'
import api from '@/api'
import type { MembershipConfig, UserBrief } from '@/types/api'

const props = defineProps<{ plan: MembershipConfig; profile: UserBrief }>()
const emit = defineEmits<{ close: []; success: [data: unknown] }>()

const loading = ref(false)
const error = ref('')

const balance = computed(() => Number(props.profile?.balance ?? 0))
const price = computed(() => Number(props.plan?.price ?? 0))
const canAfford = computed(() => balance.value >= price.value)
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
    <div class="w-[92vw] max-w-sm rounded-2xl border border-zinc-200 bg-white shadow-xl dark:border-zinc-800 dark:bg-zinc-900">
      <!-- 头部 -->
      <div class="flex items-center justify-between border-b border-zinc-100 px-6 py-4 dark:border-zinc-800">
        <div class="flex items-center gap-2">
          <NIcon :size="18" class="text-amber-500"><Crown /></NIcon>
          <h3 class="text-base font-semibold text-zinc-900 dark:text-zinc-100">开通会员</h3>
        </div>
        <button
          type="button"
          class="flex size-8 items-center justify-center rounded-full text-zinc-400 transition-colors hover:bg-zinc-100 hover:text-zinc-600 dark:hover:bg-zinc-800 dark:hover:text-zinc-300"
          aria-label="关闭"
          @click="emit('close')"
        >
          <NIcon :size="16"><X /></NIcon>
        </button>
      </div>

      <div class="space-y-4 px-6 py-5">
        <!-- 方案摘要 -->
        <div class="rounded-xl border border-zinc-200 px-4 py-3 dark:border-zinc-700">
          <p class="text-lg font-semibold" :style="plan.color ? { color: plan.color } : {}">{{ plan.level_name }}</p>
          <dl class="mt-2 space-y-1.5 text-sm">
            <div class="flex justify-between">
              <dt class="text-zinc-500 dark:text-zinc-400">有效期</dt>
              <dd class="text-zinc-800 dark:text-zinc-200">{{ plan.days }} 天（{{ daysText }}）</dd>
            </div>
            <div class="flex justify-between">
              <dt class="text-zinc-500 dark:text-zinc-400">每日下载</dt>
              <dd class="text-zinc-800 dark:text-zinc-200">{{ plan.daily_download_limit === -1 ? '无限制' : `${plan.daily_download_limit} 次` }}</dd>
            </div>
            <div v-if="plan.discount > 0 && plan.discount < 1" class="flex justify-between">
              <dt class="text-zinc-500 dark:text-zinc-400">付费链接折扣</dt>
              <dd class="text-zinc-800 dark:text-zinc-200">{{ Math.round(plan.discount * 100) }} 折</dd>
            </div>
          </dl>
        </div>

        <!-- 价格 -->
        <div class="space-y-2 rounded-xl bg-zinc-50 px-4 py-3 text-sm dark:bg-zinc-800/60">
          <div class="flex items-center justify-between">
            <span class="text-zinc-500 dark:text-zinc-400">支付金额</span>
            <span class="font-mono text-base font-semibold text-zinc-900 dark:text-zinc-100">¥{{ price.toFixed(0) }}</span>
          </div>
          <div class="flex items-center justify-between">
            <span class="text-zinc-500 dark:text-zinc-400">当前余额</span>
            <span class="font-mono" :class="canAfford ? 'text-zinc-700 dark:text-zinc-300' : 'text-red-500 dark:text-red-400'">
              ¥{{ balance.toFixed(2) }}
            </span>
          </div>
          <p v-if="!canAfford" class="text-xs text-red-500 dark:text-red-400">余额不足，请先充值后再购买</p>
        </div>

        <!-- 续期 / 重复提示 -->
        <p v-if="hasMembership && !isCurrentPlan" class="rounded-lg bg-amber-50 px-3 py-2 text-xs text-amber-700 dark:bg-amber-400/10 dark:text-amber-400">
          您当前是 {{ profile.membership_level }}，购买后有效期将从现有到期日延长
        </p>
        <p v-if="isCurrentPlan" class="rounded-lg bg-zinc-100 px-3 py-2 text-xs text-zinc-500 dark:bg-zinc-800 dark:text-zinc-400">
          您当前已是 {{ plan.level_name }}，无需重复购买
        </p>

        <p v-if="error" class="rounded-lg bg-red-50 px-3 py-2 text-xs text-red-600 dark:bg-red-500/10 dark:text-red-400">{{ error }}</p>

        <!-- 操作 -->
        <div class="space-y-3 pt-2">
          <NButton type="primary" block size="large" :loading="loading" :disabled="!canAfford || isCurrentPlan" @click="handleBuy">
            {{ isCurrentPlan ? '当前方案' : `确认支付 ¥${price.toFixed(0)}` }}
          </NButton>
          <NButton quaternary block size="large" @click="emit('close')">取消</NButton>
        </div>
      </div>
    </div>
  </NModal>
</template>

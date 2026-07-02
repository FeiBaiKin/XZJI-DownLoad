<script setup lang="ts">
// 会员购买弹窗：1:1 移植自原项目 Node-XZJI/frontend/src/MembershipModal.jsx
// 样式来自全局 styles/modal.css（.mm-* 类）
import { ref, computed, onMounted, onUnmounted } from 'vue'
import api from '@/api'
import type { MembershipConfig, UserBrief } from '@/types/api'

const props = defineProps<{ plan: MembershipConfig; profile: UserBrief }>()
const emit = defineEmits<{ close: []; success: [data: any] }>()

const loading = ref(false)
const error = ref('')
const modalRef = ref<HTMLElement | null>(null)

const balance = computed(() => parseFloat(props.profile?.balance || 0))
const canAfford = computed(() => balance.value >= parseFloat(props.plan?.price || 0))
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

function onKey(e: KeyboardEvent) { if (e.key === 'Escape') emit('close') }
onMounted(() => {
  modalRef.value?.focus()
  document.addEventListener('keydown', onKey)
})
onUnmounted(() => document.removeEventListener('keydown', onKey))

function handleOverlayClick(e: MouseEvent) {
  if (e.target === e.currentTarget) emit('close')
}

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
  <div class="modal-overlay" @click="handleOverlayClick">
    <div
      class="membership-modal"
      @click.stop
      ref="modalRef"
      role="dialog"
      aria-modal="true"
      aria-labelledby="mm-title"
      tabindex="-1"
    >
      <!-- Header -->
      <div class="mm-header">
        <div class="mm-header-left">
          <span class="mm-header-icon" aria-hidden="true">M</span>
          <h3 class="mm-title" id="mm-title">开通会员</h3>
        </div>
        <button class="mm-close" @click="emit('close')" aria-label="关闭">
          <svg width="14" height="14" viewBox="0 0 14 14" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" aria-hidden="true">
            <line x1="1" y1="1" x2="13" y2="13" /><line x1="13" y1="1" x2="1" y2="13" />
          </svg>
        </button>
      </div>

      <!-- Body -->
      <div class="mm-body">
        <!-- Plan summary -->
        <div class="mm-plan-summary">
          <div class="mm-plan-name" :style="plan.color ? { color: plan.color } : {}">{{ plan.level_name }}</div>
          <div class="mm-plan-meta">
            <div class="mm-meta-item">
              <span class="mm-meta-label">有效期</span>
              <span class="mm-meta-value">{{ plan.days }} 天 ({{ daysText }})</span>
            </div>
            <div class="mm-meta-item">
              <span class="mm-meta-label">每日下载</span>
              <span class="mm-meta-value">{{ plan.daily_download_limit === -1 ? '无限制' : `${plan.daily_download_limit} 次` }}</span>
            </div>
            <div v-if="plan.discount > 0 && plan.discount < 1" class="mm-meta-item">
              <span class="mm-meta-label">付费链接折扣</span>
              <span class="mm-meta-value">{{ Math.round(plan.discount * 100) }} 折</span>
            </div>
          </div>
        </div>

        <!-- Price -->
        <div class="mm-price-section">
          <div class="mm-price-row">
            <span class="mm-label">支付金额</span>
            <span class="mm-price-value">{{ parseFloat(plan.price).toFixed(0) }} 元</span>
          </div>
          <div class="mm-price-row">
            <span class="mm-label">当前余额</span>
            <span class="mm-balance-value" :class="{ insufficient: !canAfford }">{{ balance.toFixed(2) }} 元</span>
          </div>
          <div v-if="!canAfford" class="mm-insufficient-hint">余额不足，请先充值后再购买</div>
        </div>

        <!-- Extension note -->
        <div v-if="hasMembership && !isCurrentPlan" class="mm-extension-note">
          您当前是 {{ profile.membership_level }}，购买后有效期将从现有到期日延长
        </div>
        <div v-if="isCurrentPlan" class="mm-current-note">
          您当前已是 {{ plan.level_name }}，无需重复购买
        </div>

        <div v-if="error" class="mm-error">{{ error }}</div>

        <!-- Actions -->
        <div class="mm-actions">
          <button class="mm-submit-btn" @click="handleBuy" :disabled="loading || !canAfford || isCurrentPlan">
            {{ loading ? '处理中...' : isCurrentPlan ? '当前方案' : `确认支付 ${parseFloat(plan.price).toFixed(0)} 元` }}
          </button>
          <button class="mm-cancel-btn" @click="emit('close')">取消</button>
        </div>
      </div>
    </div>
  </div>
</template>

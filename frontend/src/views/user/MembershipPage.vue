<script setup lang="ts">
// 会员中心：1:1 移植自原项目 Node-XZJI/frontend/src/pages/UserCenter.jsx (MembershipPage)
// 样式来自全局 styles/membership-public.css
import { ref, computed, onMounted } from 'vue'
import api from '@/api'
import { useAuthStore } from '@/stores/auth'
import MembershipModal from '@/components/MembershipModal.vue'

const auth = useAuthStore()

const membership = ref<any>(null)
const profile = ref<any>(null)
const loading = ref(true)
const message = ref('')
const selectedPlan = ref<any>(null)

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
  return Math.max(0, Math.ceil((new Date(membership.value.membership.expiresAt).getTime() - Date.now()) / 86400000))
})
const totalDays = computed(() => {
  if (!hasMembership.value) return 30
  const d = new Date(membership.value.membership.expiresAt).getTime()
  const base = d - daysLeft.value * 86400000
  return Math.ceil((d - base) / 86400000) || 30
})
const progressPct = computed(() => {
  if (!hasMembership.value) return 0
  return Math.min(100, Math.max(0, Math.round((daysLeft.value / (totalDays.value || 30)) * 100)))
})
const limit = computed(() => membership.value?.dailyLimit)
const limitText = computed(() => limit.value === -1 ? '无限' : `${limit.value} 次`)
const currentPlanColor = computed(() => {
  if (!hasMembership.value) return null
  return (membership.value?.configs || []).find((c: any) => c.level_name === membership.value.membership.level)?.color || null
})
const configs = computed(() => membership.value?.configs || [])
const recommendedIdx = computed(() => configs.value.length >= 2 ? 1 : 0)

function handleBuyClick(plan: any) { selectedPlan.value = plan }

function handleBuySuccess(data: any) {
  message.value = `购买成功！有效期至 ${new Date(data.membershipExpiresAt).toLocaleDateString('zh-CN')}`
  if (auth.user) {
    auth.user.balance = data.balance
    auth.user.membership_level = selectedPlan.value?.level_name ?? auth.user.membership_level
    auth.user.membership_expires_at = data.membershipExpiresAt
  }
  selectedPlan.value = null
  fetchMembership()
}

function periodText(days: number) {
  if (days >= 365) return `${Math.floor(days / 365)} 年`
  if (days >= 30) return `${Math.floor(days / 30)} 个月`
  return `${days} 天`
}
function discountPct(plan: any) {
  return plan.discount > 0 && plan.discount < 1 ? Math.round(plan.discount * 100) : null
}
function expiryShort() {
  return new Date(membership.value.membership.expiresAt).toLocaleDateString('zh-CN', { month: 'short', day: 'numeric' })
}
</script>

<template>
  <div class="user-content">
    <div v-if="loading" class="loading-state">加载中...</div>
    <template v-else>
      <div class="content-header">
        <h2 class="content-title">会员中心</h2>
        <span class="content-sub">选择适合你的方案，解锁付费链接下载与专属权益</span>
      </div>

      <div v-if="message" :class="message.startsWith('购买成功') ? 'form-success' : 'form-error'" style="margin-bottom: 1.5rem;">
        {{ message }}
      </div>

      <!-- Hero Status Banner -->
      <div class="membership-hero">
        <div v-if="hasMembership" class="membership-hero-card active">
          <div class="membership-hero-tier-row">
            <span class="membership-hero-badge">当前会员</span>
            <span class="membership-hero-tier-name" :style="currentPlanColor ? { color: currentPlanColor } : {}">
              {{ membership.membership.level }}
            </span>
          </div>
          <div class="membership-hero-stats">
            <div class="membership-hero-stat">
              <span class="membership-hero-stat-value">{{ daysLeft }}</span>
              <span class="membership-hero-stat-label">剩余天数</span>
            </div>
            <div class="membership-hero-stat">
              <span class="membership-hero-stat-value">
                {{ membership.downloads }}<span class="membership-hero-stat-suffix">/{{ limitText }}</span>
              </span>
              <span class="membership-hero-stat-label">今日下载</span>
            </div>
            <div class="membership-hero-stat">
              <span class="membership-hero-stat-value">{{ expiryShort() }}</span>
              <span class="membership-hero-stat-label">到期日期</span>
            </div>
          </div>
          <div class="membership-hero-bar-track">
            <div class="membership-hero-bar-fill" :style="{ width: progressPct + '%', background: currentPlanColor || 'var(--text-primary)' }" />
          </div>
        </div>
        <div v-else class="membership-hero-card inactive">
          <span class="membership-hero-tier-name" style="color: var(--text-tertiary);">还不是会员</span>
          <p class="membership-hero-desc">开通会员后，可免费下载付费链接内容，并享受专属折扣</p>
        </div>
      </div>

      <!-- Plan Tiers -->
      <div class="membership-tiers">
        <div
          v-for="(plan, i) in configs"
          :key="plan.level_name"
          class="membership-tier-card"
          :class="{ recommended: i === recommendedIdx, current: hasMembership && membership.membership.level === plan.level_name }"
        >
          <div class="membership-tier-accent" :style="{ background: plan.color || 'var(--text-tertiary)' }" />
          <span v-if="i === recommendedIdx" class="membership-tier-recommend">推荐</span>
          <span v-if="hasMembership && membership.membership.level === plan.level_name" class="membership-tier-current">当前</span>
          <span class="membership-tier-name" :style="{ color: plan.color || 'var(--text-primary)' }">{{ plan.level_name }}</span>

          <div class="membership-tier-price-row">
            <span class="membership-tier-currency">¥</span>
            <span class="membership-tier-amount">{{ parseFloat(plan.price).toFixed(0) }}</span>
          </div>
          <span class="membership-tier-period">{{ periodText(plan.days) }}</span>

          <ul class="membership-tier-features">
            <li><span class="membership-tier-check">&#10003;</span>每日 {{ plan.daily_download_limit === -1 ? '无限' : plan.daily_download_limit }} 次下载</li>
            <li v-if="discountPct(plan)"><span class="membership-tier-check">&#10003;</span>付费链接 {{ discountPct(plan) }} 折优惠</li>
            <li><span class="membership-tier-check">&#10003;</span>会员专属标识</li>
            <li v-if="plan.daily_download_limit === -1"><span class="membership-tier-check">&#10003;</span>无限制访问全部内容</li>
          </ul>

          <button
            class="membership-tier-btn"
            :class="{ primary: i === recommendedIdx }"
            @click="handleBuyClick(plan)"
            :disabled="hasMembership && membership.membership.level === plan.level_name"
          >
            {{ hasMembership && membership.membership.level === plan.level_name ? '当前方案' : '立即开通' }}
          </button>
        </div>
      </div>

      <!-- Purchase Modal -->
      <MembershipModal
        v-if="selectedPlan"
        :plan="selectedPlan"
        :profile="profile"
        @close="selectedPlan = null"
        @success="handleBuySuccess"
      />
    </template>
  </div>
</template>

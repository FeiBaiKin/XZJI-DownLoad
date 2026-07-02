<script setup lang="ts">
// 充值弹窗：1:1 移植自原项目 Node-XZJI/frontend/src/RechargeModal.jsx
// 样式来自全局 styles/modal.css（.recharge-modal-v2 / .rm-*）
// 联动后台 payment_enabled_providers：拉取 /user/payment-methods 仅返回已开启的方式
import { ref, computed, onMounted, onBeforeUnmount } from 'vue'
import QRCode from 'qrcode'
import { economyApi, type PaymentMethod, type RechargeResult } from '@/api/economy'

const PRESET_AMOUNTS = [
  { label: '10 元', value: 10, exp: 20 },
  { label: '30 元', value: 30, exp: 60 },
  { label: '50 元', value: 50, exp: 100 },
  { label: '100 元', value: 100, exp: 200 },
  { label: '200 元', value: 200, exp: 400 },
  { label: '500 元', value: 500, exp: 1000 },
]

const emit = defineEmits<{ refresh: [] }>()

const show = ref(false)
const step = ref<'select' | 'paying' | 'done'>('select')
const amount = ref('')
const displayAmount = ref(0)
const error = ref('')
const loading = ref(false)
const result = ref<RechargeResult | null>(null)
const method = ref('')
const availableMethods = ref<PaymentMethod[]>([])
const methodsLoaded = ref(false)
const qrSrc = ref('')
let pollTimer: ReturnType<typeof setInterval> | null = null

const numAmount = computed(() => parseFloat(amount.value) || 0)
const bonusExp = computed(() => Math.floor(numAmount.value * 2))
const methodLabel = (name: string) => availableMethods.value.find((m) => m.name === name)?.label || name
const methodIcon = (name: string) => (name === 'wechat' ? 'W' : name === 'alipay' ? 'A' : 'E')
const methodBrief = (name: string) => (name === 'wechat' ? '扫码' : name === 'alipay' ? '网页' : name === 'epay' ? '聚合' : '')

// paying 步骤派生
const isWechat = computed(() => result.value?.type === 'wechat')
const isAlipay = computed(() => result.value?.type === 'alipay')
const isEpay = computed(() => result.value?.type === 'epay')
	const isAlipayPage = computed(() => isAlipay.value && (result.value?.scene === 'page' || result.value?.scene === 'wap'))
	const isAlipayPrecreate = computed(() => isAlipay.value && result.value?.scene === 'precreate')
	const isQrStyle = computed(() => isWechat.value || isAlipayPrecreate.value)
	const headerTitle = computed(() => {
	  if (isWechat.value) return '微信扫码支付'
	  if (isAlipayPage.value) return '支付宝网页支付'
	  if (isAlipayPrecreate.value) return '支付宝扫码支付'
	  if (isEpay.value) return '易支付收银台'
	  return '确认支付'
	})

function open() {
  show.value = true
  step.value = 'select'
  amount.value = ''
  displayAmount.value = 0
  error.value = ''
  loading.value = false
  result.value = null
  qrSrc.value = ''
  methodsLoaded.value = false
  fetchPaymentMethods()
}
function close() {
  if (pollTimer) { clearInterval(pollTimer); pollTimer = null }
  show.value = false
}
defineExpose({ open, close })

async function fetchPaymentMethods() {
  try {
    const methods = await economyApi.paymentMethods()
    if (Array.isArray(methods) && methods.length > 0) {
      availableMethods.value = methods
      method.value = methods[0].name
    } else {
      availableMethods.value = []
      method.value = ''
    }
  } catch { availableMethods.value = []; method.value = '' }
  finally { methodsLoaded.value = true }
}

const noMethods = computed(() => methodsLoaded.value && availableMethods.value.length === 0)

async function handleCreateOrder() {
  if (!method.value) { error.value = '未开启支付方式，请联系管理员'; return }
  if (!numAmount.value || numAmount.value <= 0) { error.value = '请输入有效金额'; return }
  displayAmount.value = numAmount.value
  loading.value = true
  error.value = ''
  try {
    const data = await economyApi.recharge(numAmount.value, method.value)
    if ((data as any).error) { error.value = (data as any).error; loading.value = false; return }
    result.value = data
    step.value = 'paying'
    // 微信 / 支付宝当面付 需要轮询订单状态
    if (data.type === 'wechat' || (data.type === 'alipay' && data.scene === 'precreate')) {
      if (data.qrCode) renderQR(data.qrCode)
      startPolling(data.orderNo, displayAmount.value, data.bonusExp || 0)
    }
  } catch (e: any) {
    error.value = e?.message || '创建订单失败'
  }
  loading.value = false
}

async function renderQR(url: string) {
  try {
    qrSrc.value = await QRCode.toDataURL(url, { width: 180, margin: 2, color: { dark: '#000000', light: '#ffffff' } })
  } catch { qrSrc.value = '' }
}

function startPolling(orderNo: string, orderAmount: number, exp: number) {
  if (pollTimer) clearInterval(pollTimer)
  pollTimer = setInterval(async () => {
    try {
      const data = await economyApi.rechargeStatus(orderNo)
      if (data.status === 'paid') {
        if (pollTimer) { clearInterval(pollTimer); pollTimer = null }
        step.value = 'done'
        result.value = { ...result.value, paid: true }
        emit('refresh')
      }
    } catch { /* 忽略，继续轮询 */ }
  }, 2000)
}

function handleAlipayPage() {
  if (result.value?.payUrl) {
    location.href = result.value.payUrl
  }
}
function handleEpayRedirect() {
  if (result.value?.payUrl) {
    location.href = result.value.payUrl
  }
}

function backToSelect() {
  if (pollTimer) { clearInterval(pollTimer); pollTimer = null }
  step.value = 'select'
  result.value = null
  qrSrc.value = ''
  error.value = ''
}

function onOverlayClick(e: MouseEvent) {
  if (e.target === e.currentTarget && step.value !== 'paying') close()
}

function onKeydown(e: KeyboardEvent) {
  if (e.key === 'Escape' && step.value !== 'paying') close()
}

onMounted(() => document.addEventListener('keydown', onKeydown))
onBeforeUnmount(() => {
  document.removeEventListener('keydown', onKeydown)
  if (pollTimer) clearInterval(pollTimer)
})
</script>

<template>
  <div v-if="show" class="modal-overlay" @click="onOverlayClick">
    <!-- ===== Step: Select ===== -->
    <div v-if="step === 'select'" class="recharge-modal-v2" role="dialog" aria-modal="true" aria-labelledby="rm-title" tabindex="-1">
      <div class="rm-header">
        <div class="rm-header-left">
          <span class="rm-header-icon" aria-hidden="true">¥</span>
          <h3 class="rm-title" id="rm-title">账户充值</h3>
        </div>
        <button class="rm-close" @click="close" aria-label="关闭">
          <svg width="14" height="14" viewBox="0 0 14 14" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" aria-hidden="true">
            <line x1="1" y1="1" x2="13" y2="13" /><line x1="13" y1="1" x2="1" y2="13" />
          </svg>
        </button>
      </div>

      <div class="rm-body">
        <!-- 后台未开启任何支付方式 -->
        <div v-if="noMethods" class="rm-no-methods">
          <p class="rm-qr-hint">未开启支付方式，请联系管理员</p>
        </div>

        <template v-else>
        <!-- 支付方式（多于一种时展示） -->
        <div v-if="availableMethods.length > 1" class="rm-section">
          <span class="rm-label">支付方式</span>
          <div class="rm-method-list">
            <button
              v-for="m in availableMethods"
              :key="m.name"
              type="button"
              class="rm-method-card"
              :class="{ active: method === m.name }"
              @click="method = m.name"
            >
              <span class="rm-method-icon">{{ methodIcon(m.name) }}</span>
              <span class="rm-method-label">{{ m.label }}</span>
              <span class="rm-method-brief">{{ methodBrief(m.name) }}</span>
            </button>
          </div>
        </div>

        <!-- 预设金额 -->
        <div class="rm-section">
          <span class="rm-label">选择金额</span>
          <div class="rm-amount-grid">
            <button
              v-for="item in PRESET_AMOUNTS"
              :key="item.value"
              type="button"
              class="rm-amount-chip"
              :class="{ active: numAmount === item.value }"
              @click="amount = String(item.value); error = ''"
            >
              <span class="rm-amount-value">{{ item.label }}</span>
              <span class="rm-amount-unit">+{{ item.exp }} EXP</span>
            </button>
          </div>
        </div>

        <!-- 自定义金额 -->
        <div class="rm-section">
          <span class="rm-label">自定义金额</span>
          <div class="rm-amount-custom">
            <input
              type="number"
              class="rm-custom-input"
              v-model="amount"
              placeholder="输入金额"
              min="0.01"
              max="50000"
              step="0.01"
              @input="error = ''"
            />
          </div>
          <div v-if="numAmount > 0" class="rm-bonus-hint">
            <span>赠送</span>
            <span class="rm-bonus-value">{{ bonusExp }} 经验</span>
          </div>
        </div>

        <div v-if="error" class="rm-error">{{ error }}</div>

        <button
          type="button"
          class="rm-submit-btn"
          :disabled="loading || numAmount <= 0"
          @click="handleCreateOrder"
        >
          {{ loading ? '创建订单中...' : numAmount > 0 ? `确认充值 ¥${numAmount}` : '确认充值' }}
        </button>
        </template>
      </div>
    </div>

    <!-- ===== Step: Paying ===== -->
    <div v-else-if="step === 'paying'" class="recharge-modal-v2" role="dialog" aria-modal="true" aria-labelledby="rm-title" tabindex="-1">
      <div class="rm-header">
        <div class="rm-header-left">
          <span class="rm-header-icon" aria-hidden="true">¥</span>
          <h3 class="rm-title" id="rm-title">{{ headerTitle }}</h3>
        </div>
        <button class="rm-close" @click="close" aria-label="关闭">
          <svg width="14" height="14" viewBox="0 0 14 14" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" aria-hidden="true">
            <line x1="1" y1="1" x2="13" y2="13" /><line x1="13" y1="1" x2="1" y2="13" />
          </svg>
        </button>
      </div>

      <div class="rm-body">
        <div class="rm-summary">
          <div class="rm-summary-row"><span>订单号</span><span class="rm-order-no">{{ result?.orderNo }}</span></div>
          <div class="rm-summary-row"><span>金额</span><span class="rm-summary-value">¥{{ displayAmount }}</span></div>
          <div class="rm-summary-row"><span>方式</span><span class="rm-summary-value">{{ methodLabel(method) }}</span></div>
          <div v-if="result?.bonusExp > 0" class="rm-summary-row"><span>赠送经验</span><span class="rm-summary-value">+{{ result.bonusExp }} EXP</span></div>
        </div>

        <!-- 二维码：微信 Native / 支付宝当面付 -->
        <div v-if="isQrStyle" class="rm-qr-section">
          <div v-if="qrSrc" class="rm-qr-frame">
            <img :src="qrSrc" alt="支付二维码" width="180" height="180" style="display:block;image-rendering:pixelated;" />
          </div>
          <div v-else class="rm-qr-frame" style="min-height:180px;"></div>
          <p class="rm-qr-hint">{{ isWechat ? '请使用微信扫描二维码完成支付' : '请使用支付宝扫描二维码完成支付' }}</p>
          <p class="rm-qr-sub">支付完成后将自动确认到账</p>
        </div>

        <!-- 支付宝网页支付：跳转 -->
	        <div v-else-if="isAlipayPage" class="rm-redirect-section">
	          <p class="rm-qr-hint">点击下方按钮，前往支付宝完成支付</p>
	          <p class="rm-qr-sub">支付完成后将自动跳转回本站</p>
	          <button type="button" class="rm-submit-btn" @click="handleAlipayPage">前往支付宝支付</button>
	        </div>

        <!-- 易支付：跳转收银台 -->
        <div v-else-if="isEpay" class="rm-redirect-section">
          <p class="rm-qr-hint">点击下方按钮，跳转到易支付收银台完成支付</p>
          <p class="rm-qr-sub">支持微信、支付宝等多种方式</p>
          <button type="button" class="rm-submit-btn" @click="handleEpayRedirect">前往易支付收银台</button>
        </div>

        <div v-if="error" class="rm-error">{{ error }}</div>
        <button type="button" class="rm-back-btn" @click="backToSelect">返回重选</button>
      </div>
    </div>

    <!-- ===== Step: Done ===== -->
    <div v-else class="recharge-modal-v2" role="dialog" aria-modal="true" aria-labelledby="rm-title" tabindex="-1">
      <div class="rm-header">
        <div class="rm-header-left">
          <span class="rm-header-icon" aria-hidden="true">✓</span>
          <h3 class="rm-title" id="rm-title">充值成功</h3>
        </div>
        <button class="rm-close" @click="close" aria-label="关闭">
          <svg width="14" height="14" viewBox="0 0 14 14" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" aria-hidden="true">
            <line x1="1" y1="1" x2="13" y2="13" /><line x1="13" y1="1" x2="1" y2="13" />
          </svg>
        </button>
      </div>
      <div class="rm-body rm-done">
        <svg class="rm-done-icon" width="32" height="32" viewBox="0 0 32 32" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" aria-hidden="true">
          <circle cx="16" cy="16" r="14" /><path d="M9 16l5 5 9-9" />
        </svg>
        <p class="rm-done-text">¥{{ displayAmount }} 已到账</p>
        <p v-if="result?.bonusExp > 0" class="rm-done-sub">+{{ result.bonusExp }} 经验</p>
        <button type="button" class="rm-submit-btn" style="margin-top:0.5rem;" @click="close">完成</button>
      </div>
    </div>
  </div>
</template>

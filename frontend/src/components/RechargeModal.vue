<script setup lang="ts">
// 充值弹窗：三步流程 select → paying → done
// 业务逻辑保持不变（支付方式联动后台 payment_enabled_providers；扫码类 2s 轮询订单状态）
import { ref, computed, onBeforeUnmount } from 'vue'
import { NModal, NButton, NInput, NIcon } from 'naive-ui'
import { X, Qrcode, Wallet, CreditCard, Check, ArrowLeft, ExternalLink } from '@vicons/tabler'
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

type RechargeState = RechargeResult & { paid?: boolean }

const emit = defineEmits<{ refresh: [] }>()

const show = ref(false)
const step = ref<'select' | 'paying' | 'done'>('select')
const amount = ref('')
const displayAmount = ref(0)
const error = ref('')
const loading = ref(false)
const result = ref<RechargeState | null>(null)
const method = ref('')
const availableMethods = ref<PaymentMethod[]>([])
const methodsLoaded = ref(false)
const qrSrc = ref('')
let pollTimer: ReturnType<typeof setInterval> | null = null

const numAmount = computed(() => parseFloat(amount.value) || 0)
const bonusExp = computed(() => Math.floor(numAmount.value * 2))
const resultBonusExp = computed(() => result.value?.bonusExp ?? 0)
const methodLabel = (name: string) => availableMethods.value.find((m) => m.name === name)?.label || name
const methodIconOf = (name: string) => (name === 'wechat' ? Qrcode : name === 'alipay' ? Wallet : CreditCard)
const methodBrief = (name: string) => (name === 'wechat' ? '扫码支付' : name === 'alipay' ? '网页支付' : name === 'epay' ? '聚合收银台' : '')

// paying 步骤派生
const isWechat = computed(() => result.value?.type === 'wechat')
const isAlipay = computed(() => result.value?.type === 'alipay')
const isEpay = computed(() => result.value?.type === 'epay')
const isAlipayPage = computed(() => isAlipay.value && (result.value?.scene === 'page' || result.value?.scene === 'wap'))
const isAlipayPrecreate = computed(() => isAlipay.value && result.value?.scene === 'precreate')
const isQrStyle = computed(() => isWechat.value || isAlipayPrecreate.value)
const headerTitle = computed(() => {
  if (step.value === 'done') return '充值成功'
  if (step.value === 'select') return '账户充值'
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
      method.value = methods[0]!.name
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
    if ((data as { error?: string }).error) {
      error.value = (data as { error?: string }).error || '创建订单失败'
      loading.value = false
      return
    }
    result.value = data
    step.value = 'paying'
    // 微信 / 支付宝当面付 需要轮询订单状态
    if (data.type === 'wechat' || (data.type === 'alipay' && data.scene === 'precreate')) {
      if (data.qrCode) renderQR(data.qrCode)
      startPolling(data.orderNo)
    }
  } catch (e: unknown) {
    error.value = (e as Error)?.message || '创建订单失败'
  }
  loading.value = false
}

async function renderQR(url: string) {
  try {
    qrSrc.value = await QRCode.toDataURL(url, { width: 180, margin: 2, color: { dark: '#000000', light: '#ffffff' } })
  } catch { qrSrc.value = '' }
}

function startPolling(orderNo: string) {
  if (pollTimer) clearInterval(pollTimer)
  pollTimer = setInterval(async () => {
    try {
      const data = await economyApi.rechargeStatus(orderNo)
      if (data.status === 'paid') {
        if (pollTimer) { clearInterval(pollTimer); pollTimer = null }
        step.value = 'done'
        if (result.value) result.value = { ...result.value, paid: true }
        emit('refresh')
      }
    } catch { /* 忽略，继续轮询 */ }
  }, 2000)
}

function handleRedirect() {
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

onBeforeUnmount(() => {
  if (pollTimer) clearInterval(pollTimer)
})
</script>

<template>
  <NModal
    :show="show"
    :mask-closable="step !== 'paying'"
    :close-on-esc="step !== 'paying'"
    @update:show="(v: boolean) => { if (!v) close() }"
  >
    <div class="w-[92vw] max-w-md rounded-2xl border border-zinc-200 bg-white shadow-xl dark:border-zinc-800 dark:bg-zinc-900">
      <!-- 头部 -->
      <div class="flex items-center justify-between border-b border-zinc-100 px-6 py-4 dark:border-zinc-800">
        <h3 class="text-base font-semibold text-zinc-900 dark:text-zinc-100">{{ headerTitle }}</h3>
        <button
          type="button"
          class="flex size-8 items-center justify-center rounded-full text-zinc-400 transition-colors hover:bg-zinc-100 hover:text-zinc-600 dark:hover:bg-zinc-800 dark:hover:text-zinc-300"
          aria-label="关闭"
          @click="close"
        >
          <NIcon :size="16"><X /></NIcon>
        </button>
      </div>

      <!-- ===== Step: Select ===== -->
      <div v-if="step === 'select'" class="space-y-5 px-6 py-5">
        <div v-if="noMethods" class="rounded-lg bg-zinc-50 py-10 text-center text-sm text-zinc-500 dark:bg-zinc-800/60 dark:text-zinc-400">
          未开启支付方式，请联系管理员
        </div>

        <template v-else>
          <!-- 支付方式（多于一种时展示） -->
          <div v-if="availableMethods.length > 1">
            <p class="mb-2 text-xs font-medium text-zinc-500 dark:text-zinc-400">支付方式</p>
            <div class="grid gap-2" :class="availableMethods.length >= 3 ? 'grid-cols-3' : 'grid-cols-2'">
              <button
                v-for="m in availableMethods"
                :key="m.name"
                type="button"
                class="flex flex-col items-center gap-1.5 rounded-xl border px-2 py-3 transition-colors"
                :class="method === m.name
                  ? 'border-emerald-500 bg-emerald-50 dark:border-emerald-400 dark:bg-emerald-400/10'
                  : 'border-zinc-200 hover:border-zinc-300 dark:border-zinc-700 dark:hover:border-zinc-600'"
                @click="method = m.name"
              >
                <NIcon :size="22" :class="method === m.name ? 'text-emerald-600 dark:text-emerald-400' : 'text-zinc-400'">
                  <component :is="methodIconOf(m.name)" />
                </NIcon>
                <span class="text-xs font-medium text-zinc-800 dark:text-zinc-200">{{ m.label }}</span>
                <span class="text-[10px] text-zinc-400 dark:text-zinc-500">{{ methodBrief(m.name) }}</span>
              </button>
            </div>
          </div>

          <!-- 预设金额 -->
          <div>
            <p class="mb-2 text-xs font-medium text-zinc-500 dark:text-zinc-400">选择金额</p>
            <div class="grid grid-cols-3 gap-2">
              <button
                v-for="item in PRESET_AMOUNTS"
                :key="item.value"
                type="button"
                class="rounded-xl border py-2.5 transition-colors"
                :class="numAmount === item.value
                  ? 'border-emerald-500 bg-emerald-50 dark:border-emerald-400 dark:bg-emerald-400/10'
                  : 'border-zinc-200 hover:border-zinc-300 dark:border-zinc-700 dark:hover:border-zinc-600'"
                @click="amount = String(item.value); error = ''"
              >
                <span class="block text-sm font-semibold text-zinc-900 dark:text-zinc-100">{{ item.label }}</span>
                <span class="mt-0.5 block font-mono text-[10px] text-zinc-400 dark:text-zinc-500">+{{ item.exp }} EXP</span>
              </button>
            </div>
          </div>

          <!-- 自定义金额 -->
          <div>
            <p class="mb-2 text-xs font-medium text-zinc-500 dark:text-zinc-400">自定义金额</p>
            <NInput
              v-model:value="amount"
              placeholder="输入金额（0.01 - 50000）"
              :input-props="{ inputmode: 'decimal' }"
              @input="error = ''"
            >
              <template #prefix><span class="text-zinc-400">¥</span></template>
            </NInput>
            <p v-if="numAmount > 0" class="mt-2 text-xs text-zinc-500 dark:text-zinc-400">
              到账后赠送 <span class="font-mono font-medium text-emerald-600 dark:text-emerald-400">{{ bonusExp }} 经验</span>
            </p>
          </div>

          <p v-if="error" class="rounded-lg bg-red-50 px-3 py-2 text-xs text-red-600 dark:bg-red-500/10 dark:text-red-400">{{ error }}</p>

          <NButton type="primary" block size="large" :loading="loading" :disabled="numAmount <= 0" @click="handleCreateOrder">
            {{ numAmount > 0 ? `确认充值 ¥${numAmount}` : '确认充值' }}
          </NButton>
        </template>
      </div>

      <!-- ===== Step: Paying ===== -->
      <div v-else-if="step === 'paying'" class="space-y-5 px-6 py-5">
        <!-- 订单摘要 -->
        <dl class="space-y-2 rounded-xl bg-zinc-50 px-4 py-3 text-sm dark:bg-zinc-800/60">
          <div class="flex items-center justify-between gap-4">
            <dt class="shrink-0 text-zinc-500 dark:text-zinc-400">订单号</dt>
            <dd class="truncate font-mono text-xs text-zinc-700 dark:text-zinc-300">{{ result?.orderNo }}</dd>
          </div>
          <div class="flex items-center justify-between">
            <dt class="text-zinc-500 dark:text-zinc-400">金额</dt>
            <dd class="font-mono font-semibold text-zinc-900 dark:text-zinc-100">¥{{ displayAmount }}</dd>
          </div>
          <div class="flex items-center justify-between">
            <dt class="text-zinc-500 dark:text-zinc-400">方式</dt>
            <dd class="text-zinc-700 dark:text-zinc-300">{{ methodLabel(method) }}</dd>
          </div>
          <div v-if="resultBonusExp > 0" class="flex items-center justify-between">
            <dt class="text-zinc-500 dark:text-zinc-400">赠送经验</dt>
            <dd class="font-mono text-emerald-600 dark:text-emerald-400">+{{ resultBonusExp }} EXP</dd>
          </div>
        </dl>

        <!-- 二维码：微信 Native / 支付宝当面付 -->
        <div v-if="isQrStyle" class="flex flex-col items-center gap-3">
          <div class="rounded-xl border border-zinc-200 bg-white p-3 dark:border-zinc-700">
            <img v-if="qrSrc" :src="qrSrc" alt="支付二维码" width="180" height="180" class="block [image-rendering:pixelated]" />
            <div v-else class="size-45 animate-pulse rounded bg-zinc-100"></div>
          </div>
          <p class="text-sm text-zinc-700 dark:text-zinc-300">
            {{ isWechat ? '请使用微信扫描二维码完成支付' : '请使用支付宝扫描二维码完成支付' }}
          </p>
          <p class="flex items-center gap-2 text-xs text-zinc-400 dark:text-zinc-500">
            <span class="inline-block size-1.5 animate-pulse rounded-full bg-emerald-500"></span>
            支付完成后将自动确认到账
          </p>
        </div>

        <!-- 跳转类：支付宝网页 / 易支付收银台 -->
        <div v-else class="flex flex-col items-center gap-2 py-2 text-center">
          <p class="text-sm text-zinc-700 dark:text-zinc-300">
            {{ isEpay ? '点击下方按钮，跳转到易支付收银台完成支付' : '点击下方按钮，前往支付宝完成支付' }}
          </p>
          <p class="text-xs text-zinc-400 dark:text-zinc-500">
            {{ isEpay ? '支持微信、支付宝等多种方式' : '支付完成后将自动跳转回本站' }}
          </p>
          <NButton type="primary" size="large" class="mt-3 w-full" @click="handleRedirect">
            <template #icon><NIcon><ExternalLink /></NIcon></template>
            {{ isEpay ? '前往易支付收银台' : '前往支付宝支付' }}
          </NButton>
        </div>

        <p v-if="error" class="rounded-lg bg-red-50 px-3 py-2 text-xs text-red-600 dark:bg-red-500/10 dark:text-red-400">{{ error }}</p>

        <NButton quaternary block @click="backToSelect">
          <template #icon><NIcon><ArrowLeft /></NIcon></template>
          返回重选
        </NButton>
      </div>

      <!-- ===== Step: Done ===== -->
      <div v-else class="flex flex-col items-center gap-2 px-6 py-8">
        <div class="flex size-14 items-center justify-center rounded-full bg-emerald-100 text-emerald-600 dark:bg-emerald-400/10 dark:text-emerald-400">
          <NIcon :size="28"><Check /></NIcon>
        </div>
        <p class="mt-2 font-mono text-xl font-semibold text-zinc-900 dark:text-zinc-100">¥{{ displayAmount }} 已到账</p>
        <p v-if="resultBonusExp > 0" class="font-mono text-sm text-emerald-600 dark:text-emerald-400">+{{ resultBonusExp }} 经验</p>
        <NButton type="primary" block size="large" class="mt-4" @click="close">完成</NButton>
      </div>
    </div>
  </NModal>
</template>

<script setup lang="ts">
// 会员购买弹窗：余额扣费 或 后台已开启的第三方支付直付（扫码 / 跳转 + 订单轮询）
// 契约：POST /user/buy-membership { level, method? }
//   - 不传 method（余额）→ { message, balance, membershipLevel, membershipExpiresAt }
//   - 传 method（直付）  → { pay:true, type, scene, payUrl, qrCode, orderNo, amount, bonusExp }
//     直付订单由支付回调开通会员，前端轮询 GET /user/recharge-status/:orderNo 等 paid
import { ref, computed, onMounted, onBeforeUnmount } from 'vue'
import { NModal, NButton } from 'naive-ui'
import { ArrowLeft, Calendar, Coins, CreditCard, Download, ExternalLink, Info, Percent, QrCode, Wallet, X } from '@lucide/vue'
import AppIcon from './AppIcon.vue'
import QRCode from 'qrcode'
import { economyApi, type PaymentMethod, type MembershipPayResult } from '@/api/economy'
import { authApi } from '@/api/auth'
import type { MembershipConfig, UserBrief } from '@/types/api'
import { tierIcon, tierColor, tierWash, readableOn, periodText, dlText, discountLabel, priceText } from '@/utils/tier'

const BALANCE = 'balance'

const props = withDefaults(defineProps<{ plan: MembershipConfig; profile: UserBrief; tierIndex?: number }>(), { tierIndex: 0 })
const emit = defineEmits<{ close: []; success: [data: unknown] }>()

const loading = ref(false)
const error = ref('')
const step = ref<'select' | 'paying'>('select')
const methods = ref<PaymentMethod[]>([])
const payMethod = ref(BALANCE)
const order = ref<MembershipPayResult | null>(null)
const qrSrc = ref('')
let pollTimer: ReturnType<typeof setInterval> | null = null

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

// ── 支付方式 ──
const methodIconOf = (name: string) =>
  name === BALANCE ? Coins : name === 'wechat' ? QrCode : name === 'alipay' ? Wallet : CreditCard
const methodBriefOf = (name: string) =>
  name === 'wechat' ? '扫码支付' : name === 'alipay' ? '网页支付' : name === 'epay' ? '聚合收银台' : ''

// 免费档位没有可下单的金额，只保留余额开通
const payOptions = computed(() => {
  const list = [{ name: BALANCE, label: '余额支付', brief: `可用 ¥${balance.value.toFixed(2)}` }]
  if (price.value > 0) {
    for (const m of methods.value) list.push({ name: m.name, label: m.label, brief: methodBriefOf(m.name) })
  }
  return list
})
const hasThirdParty = computed(() => payOptions.value.length > 1)
const usingBalance = computed(() => payMethod.value === BALANCE)
const methodLabelOf = (name: string) => payOptions.value.find((o) => o.name === name)?.label || name
// 余额不足只拦余额支付，第三方直付不受影响
const blocked = computed(() => isCurrentPlan.value || (usingBalance.value && !canAfford.value))

onMounted(fetchPaymentMethods)
onBeforeUnmount(stopPolling)

async function fetchPaymentMethods() {
  try {
    const list = await economyApi.paymentMethods()
    methods.value = Array.isArray(list) ? list : []
  } catch {
    methods.value = []
  }
  // 余额不足时默认选中第一种可用的第三方支付，省去用户再点一次
  if (price.value > 0 && !canAfford.value && methods.value.length > 0) {
    payMethod.value = methods.value[0]!.name
  }
}

// ── 付款中的派生状态 ──
const isWechat = computed(() => order.value?.type === 'wechat')
const isAlipayPrecreate = computed(() => order.value?.type === 'alipay' && order.value?.scene === 'precreate')
const isEpay = computed(() => order.value?.type === 'epay')
const isQrStyle = computed(() => isWechat.value || isAlipayPrecreate.value)
const bonusExp = computed(() => order.value?.bonusExp ?? 0)

async function handleBuy() {
  if (blocked.value) return
  loading.value = true
  error.value = ''
  try {
    const res = await economyApi.buyMembership(props.plan.level_name, usingBalance.value ? undefined : payMethod.value)
    if ('pay' in res && res.pay) {
      order.value = res
      step.value = 'paying'
      if (isQrStyle.value) {
        if (res.qrCode) renderQR(res.qrCode)
        startPolling(res.orderNo)
      }
    } else {
      emit('success', res)
      emit('close')
    }
  } catch (err: unknown) {
    const errorObj = err as { response?: { data?: { error?: string } } }
    error.value = errorObj?.response?.data?.error || '购买失败'
  } finally {
    loading.value = false
  }
}

async function renderQR(url: string) {
  try {
    qrSrc.value = await QRCode.toDataURL(url, { width: 180, margin: 2, color: { dark: '#000000', light: '#ffffff' } })
  } catch { qrSrc.value = '' }
}

function startPolling(orderNo: string) {
  stopPolling()
  let attempts = 0
  const maxAttempts = 90 // 约 3 分钟（2s 间隔）
  pollTimer = setInterval(async () => {
    attempts += 1
    if (attempts > maxAttempts) { stopPolling(); return }
    try {
      const data = await economyApi.rechargeStatus(orderNo)
      if (data.status === 'paid') handlePaid()
    } catch { /* 忽略，继续轮询 */ }
  }, 2000)
}

function stopPolling() {
  if (pollTimer) { clearInterval(pollTimer); pollTimer = null }
}

// 直付到账：会员由支付回调开通，这里回读 profile 拿到最新有效期后交给父页面提示
async function handlePaid() {
  stopPolling()
  let payload: { balance: number; membershipExpiresAt: string | null } = { balance: balance.value, membershipExpiresAt: null }
  try {
    const u = await authApi.profile()
    payload = { balance: u.balance, membershipExpiresAt: u.membership_expires_at }
  } catch { /* profile 失败不阻断，父页面会重新拉会员状态 */ }
  emit('success', payload)
  emit('close')
}

function handleRedirect() {
  if (order.value?.payUrl) location.href = order.value.payUrl
}

function backToSelect() {
  stopPolling()
  step.value = 'select'
  order.value = null
  qrSrc.value = ''
  error.value = ''
}
</script>

<template>
  <NModal
    :show="true"
    :mask-closable="step !== 'paying' && !loading"
    :close-on-esc="step !== 'paying' && !loading"
    @update:show="(v: boolean) => { if (!v) emit('close') }"
  >
    <div class="w-[92vw] max-w-md overflow-hidden rounded-2xl border border-zinc-200 bg-white shadow-2xl dark:border-zinc-800 dark:bg-zinc-900">
      <!-- 渐染头部 -->
      <div class="p-5 pb-4" :style="{ background: tierWash(planColor) }">
        <div class="flex items-start justify-between gap-3">
          <div class="flex min-w-0 items-center gap-3">
            <span
              class="grid size-11 shrink-0 place-items-center rounded-xl"
              :style="{ backgroundColor: planColor, color: readableOn(planColor) }"
            >
              <AppIcon :icon="tierIcon(iconIndex)" scale="md" />
            </span>
            <div class="min-w-0">
              <p class="text-xs text-zinc-500 dark:text-zinc-400">{{ step === 'paying' ? '等待支付' : '开通会员' }}</p>
              <h3 class="truncate text-lg font-bold tracking-tight text-zinc-900 dark:text-zinc-50">{{ plan.level_name }}</h3>
            </div>
          </div>
          <button
            type="button"
            class="flex size-8 shrink-0 items-center justify-center rounded-full text-zinc-400 transition-colors hover:bg-zinc-900/5 hover:text-zinc-600 focus-visible:outline-2 focus-visible:outline-offset-2 focus-visible:outline-emerald-500 dark:hover:bg-white/10 dark:hover:text-zinc-300"
            aria-label="关闭"
            @click="emit('close')"
          >
            <AppIcon :icon="X" scale="sm" />
          </button>
        </div>
      </div>

      <!-- ===== Step: Select ===== -->
      <div v-if="step === 'select'" class="space-y-4 px-5 pb-5">
        <!-- 档位权益 -->
        <dl class="space-y-2.5 text-sm">
          <div class="flex items-center justify-between">
            <dt class="flex items-center gap-2 text-zinc-500 dark:text-zinc-400">
              <AppIcon :icon="Calendar" scale="sm" :style="{ color: planColor }" />有效期
            </dt>
            <dd class="font-medium text-zinc-800 dark:text-zinc-200">{{ plan.days }} 天（{{ periodText(plan.days) }}）</dd>
          </div>
          <div class="flex items-center justify-between">
            <dt class="flex items-center gap-2 text-zinc-500 dark:text-zinc-400">
              <AppIcon :icon="Download" scale="sm" :style="{ color: planColor }" />每日下载
            </dt>
            <dd class="font-medium text-zinc-800 dark:text-zinc-200">{{ dlText(plan.daily_download_limit) }}</dd>
          </div>
          <div class="flex items-center justify-between">
            <dt class="flex items-center gap-2 text-zinc-500 dark:text-zinc-400">
              <AppIcon :icon="Percent" scale="sm" :style="{ color: planColor }" />付费链接折扣
            </dt>
            <dd class="font-medium text-zinc-800 dark:text-zinc-200">{{ discountText ?? '标准价' }}</dd>
          </div>
        </dl>

        <!-- 支付方式：仅在后台开启了第三方支付时展示（否则只有余额一种，无需选择） -->
        <div v-if="hasThirdParty">
          <p class="mb-2 text-xs font-medium text-zinc-500 dark:text-zinc-400">支付方式</p>
          <div class="grid gap-2" :class="payOptions.length >= 3 ? 'grid-cols-2 sm:grid-cols-3' : 'grid-cols-2'">
            <button
              v-for="opt in payOptions"
              :key="opt.name"
              type="button"
              class="flex flex-col items-center gap-1 rounded-xl border px-2 py-2.5 transition-colors focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-emerald-500/60 dark:focus-visible:ring-emerald-400/60"
              :class="payMethod === opt.name
                ? 'border-emerald-500 bg-emerald-50 dark:border-emerald-400 dark:bg-emerald-400/10'
                : 'border-zinc-200 hover:border-zinc-300 dark:border-zinc-700 dark:hover:border-zinc-600'"
              :aria-pressed="payMethod === opt.name"
              @click="payMethod = opt.name; error = ''"
            >
              <AppIcon
                :icon="methodIconOf(opt.name)"
                scale="md"
                :class="payMethod === opt.name ? 'text-emerald-600 dark:text-emerald-400' : 'text-zinc-500 dark:text-zinc-400'"
              />
              <span class="text-xs font-medium text-zinc-800 dark:text-zinc-200">{{ opt.label }}</span>
              <span class="text-[10px] text-zinc-500 dark:text-zinc-400">{{ opt.brief }}</span>
            </button>
          </div>
        </div>

        <!-- 价格区 -->
        <div class="space-y-2.5 rounded-xl bg-zinc-50 px-4 py-4 dark:bg-zinc-800/60">
          <div class="flex items-center justify-between">
            <span class="flex items-center gap-1.5 text-sm text-zinc-500 dark:text-zinc-400"><AppIcon :icon="Wallet" scale="xs" />支付金额</span>
            <span class="font-mono text-xl font-bold text-zinc-900 dark:text-zinc-100">¥{{ priceText(price) }}</span>
          </div>
          <div v-if="usingBalance" class="flex items-center justify-between">
            <span class="text-sm text-zinc-500 dark:text-zinc-400">当前余额</span>
            <span class="font-mono text-sm" :class="canAfford ? 'text-zinc-700 dark:text-zinc-300' : 'font-semibold text-red-500 dark:text-red-400'">
              ¥{{ balance.toFixed(2) }}
            </span>
          </div>
          <div v-else class="flex items-center justify-between">
            <span class="text-sm text-zinc-500 dark:text-zinc-400">支付方式</span>
            <span class="text-sm text-zinc-700 dark:text-zinc-300">{{ methodLabelOf(payMethod) }}</span>
          </div>
          <div v-if="usingBalance && !canAfford" class="flex items-start gap-1.5 rounded-lg bg-red-50 px-2.5 py-1.5 text-xs text-red-600 dark:bg-red-500/10 dark:text-red-400">
            <AppIcon :icon="Info" scale="xs" class="mt-px shrink-0" />
            <span>{{ hasThirdParty ? '余额不足，可改用上方支付方式直接开通，无需先充值' : '余额不足，请先充值后再购买' }}</span>
          </div>
        </div>

        <!-- 续期 / 重复提示 -->
        <p
          v-if="hasMembership && !isCurrentPlan"
          class="flex items-start gap-1.5 rounded-lg bg-amber-50 px-3 py-2.5 text-xs text-amber-700 dark:bg-amber-400/10 dark:text-amber-400"
        >
          <AppIcon :icon="Info" scale="xs" class="mt-0.5 shrink-0" />
          <span>您当前是 {{ profile.membership_level }}，购买后有效期将从现有到期日延长。</span>
        </p>
        <p
          v-if="isCurrentPlan"
          class="flex items-start gap-1.5 rounded-lg bg-zinc-100 px-3 py-2.5 text-xs text-zinc-500 dark:bg-zinc-800 dark:text-zinc-400"
        >
          <AppIcon :icon="Info" scale="xs" class="mt-0.5 shrink-0" />
          <span>您当前已是 {{ plan.level_name }}，无需重复购买。</span>
        </p>

        <p v-if="error" class="rounded-lg bg-red-50 px-3 py-2 text-xs text-red-600 dark:bg-red-500/10 dark:text-red-400">{{ error }}</p>

        <!-- 操作 -->
        <div class="space-y-2.5 pt-1">
          <NButton type="primary" block size="large" :loading="loading" :disabled="blocked" @click="handleBuy">
            {{ isCurrentPlan ? '当前方案' : `确认支付 ¥${priceText(price)}` }}
          </NButton>
          <NButton quaternary block @click="emit('close')">取消</NButton>
        </div>
      </div>

      <!-- ===== Step: Paying ===== -->
      <div v-else class="space-y-4 px-5 pb-5">
        <!-- 订单摘要 -->
        <dl class="space-y-2 rounded-xl bg-zinc-50 px-4 py-3 text-sm dark:bg-zinc-800/60">
          <div class="flex items-center justify-between gap-4">
            <dt class="shrink-0 text-zinc-500 dark:text-zinc-400">订单号</dt>
            <dd class="truncate font-mono text-xs text-zinc-700 dark:text-zinc-300">{{ order?.orderNo }}</dd>
          </div>
          <div class="flex items-center justify-between">
            <dt class="text-zinc-500 dark:text-zinc-400">金额</dt>
            <dd class="font-mono font-semibold text-zinc-900 dark:text-zinc-100">¥{{ priceText(price) }}</dd>
          </div>
          <div class="flex items-center justify-between">
            <dt class="text-zinc-500 dark:text-zinc-400">方式</dt>
            <dd class="text-zinc-700 dark:text-zinc-300">{{ methodLabelOf(payMethod) }}</dd>
          </div>
          <div v-if="bonusExp > 0" class="flex items-center justify-between">
            <dt class="text-zinc-500 dark:text-zinc-400">赠送经验</dt>
            <dd class="font-mono text-emerald-600 dark:text-emerald-400">+{{ bonusExp }} EXP</dd>
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
          <p class="flex items-center gap-2 text-xs text-zinc-500 dark:text-zinc-400">
            <span class="inline-block size-1.5 animate-pulse rounded-full bg-emerald-500"></span>
            支付完成后自动开通会员
          </p>
        </div>

        <!-- 跳转类：支付宝网页 / 易支付收银台 -->
        <div v-else class="flex flex-col items-center gap-2 py-2 text-center">
          <p class="text-sm text-zinc-700 dark:text-zinc-300">
            {{ isEpay ? '点击下方按钮，跳转到易支付收银台完成支付' : '点击下方按钮，前往支付宝完成支付' }}
          </p>
          <p class="text-xs text-zinc-500 dark:text-zinc-400">支付成功后会员自动开通，无需手动扣余额</p>
          <NButton type="primary" size="large" class="mt-3 w-full" @click="handleRedirect">
            <template #icon><AppIcon :icon="ExternalLink" scale="sm" /></template>
            {{ isEpay ? '前往易支付收银台' : '前往支付宝支付' }}
          </NButton>
        </div>

        <NButton quaternary block @click="backToSelect">
          <template #icon><AppIcon :icon="ArrowLeft" scale="sm" /></template>
          返回重选
        </NButton>
      </div>
    </div>
  </NModal>
</template>

<script setup lang="ts">
// 支付配置：站点地址 + 微信/支付宝/易支付三家参数（payment_* 键直读直写）
// 业务逻辑保持不变；UI 重构为 Naive UI + Tailwind
import { ref, computed, onMounted } from 'vue'
import { NButton, NInput, NAlert, NSwitch, NSkeleton, NRadioGroup, NRadioButton, NTag } from 'naive-ui'
import { ChevronDown, CreditCard, QrCode, Wallet } from '@lucide/vue'
import AppIcon from '../../components/AppIcon.vue'
import api from '@/api'
import { PROVIDERS } from '@/constants/admin'

interface PaymentConfigData {
  payment_site_url: string
  payment_enabled_providers: string
  payment_wx_appid: string
  payment_wx_mchid: string
  payment_wx_serial_no: string
  payment_wx_api_v3_key: string
  payment_wx_private_key: string
  payment_wx_notify_url: string
  payment_ali_appid: string
  payment_ali_private_key: string
  payment_ali_public_key: string
  payment_ali_scene: string
  payment_ali_notify_url: string
  payment_ali_return_url: string
  payment_epay_gateway: string
  payment_epay_pid: string
  payment_epay_key: string
  payment_epay_notify_url: string
  payment_epay_return_url: string
  [key: string]: string
}

const defaults: PaymentConfigData = {
  payment_site_url: '', payment_enabled_providers: '[]', payment_wx_appid: '',
  payment_wx_mchid: '', payment_wx_serial_no: '', payment_wx_api_v3_key: '',
  payment_wx_private_key: '', payment_wx_notify_url: '', payment_ali_appid: '',
  payment_ali_private_key: '', payment_ali_public_key: '', payment_ali_scene: 'page',
  payment_ali_notify_url: '', payment_ali_return_url: '', payment_epay_gateway: '',
  payment_epay_pid: '', payment_epay_key: '', payment_epay_notify_url: '', payment_epay_return_url: '',
}
const config = ref<PaymentConfigData>({ ...defaults })
const loading = ref(true)
const saving = ref(false)
const message = ref('')
const error = ref('')
const activeProvider = ref<string | null>(null)

const providerIcons: Record<string, typeof QrCode> = {
  wechat: QrCode,
  alipay: Wallet,
  epay: CreditCard,
}

onMounted(() => { fetchConfig() })

async function fetchConfig() {
  try {
    const res = await api.get('/admin/payment-config')
    // GO 返回 payment_* 键，直接合并
    config.value = { ...defaults, ...res.data }
  } catch (err) {
    console.error('获取支付配置异常:', err)
    error.value = '获取配置失败'
  } finally {
    loading.value = false
  }
}

async function handleSave() {
  saving.value = true; message.value = ''; error.value = ''
  try {
    await api.put('/admin/payment-config', config.value)
    message.value = '支付配置已保存'
    await fetchConfig()
  } catch (err: unknown) {
    const errorObj = err as { response?: { data?: { error?: string } } }
    error.value = errorObj?.response?.data?.error || '保存失败'
  } finally {
    saving.value = false
  }
}

function toggleProvider(provider: string) {
  let providers: string[] = []
  try { providers = JSON.parse(config.value.payment_enabled_providers) } catch { /* 保持空数组 */ }
  if (!Array.isArray(providers)) providers = []
  if (providers.includes(provider)) providers = providers.filter((p) => p !== provider)
  else providers.push(provider)
  config.value.payment_enabled_providers = JSON.stringify(providers)
  if (activeProvider.value === provider && !providers.includes(provider)) activeProvider.value = null
}

const enabledProviders = computed<string[]>(() => {
  try { const p = JSON.parse(config.value.payment_enabled_providers); return Array.isArray(p) ? p : [] } catch { return [] }
})
const siteUrl = computed(() => (config.value.payment_site_url ? config.value.payment_site_url.replace(/\/+$/, '') : ''))
</script>

<template>
  <div>
    <div v-if="loading" class="space-y-4">
      <NSkeleton text width="20%" size="large" />
      <NSkeleton height="140px" :sharp="false" />
      <NSkeleton v-for="i in 3" :key="i" height="64px" :sharp="false" />
    </div>

    <template v-else>
      <header class="mb-6">
        <h1 class="text-xl font-bold tracking-tight text-zinc-900 dark:text-zinc-50">支付配置</h1>
        <p class="mt-1 text-sm text-zinc-500 dark:text-zinc-400">管理可用支付方式及第三方支付参数</p>
      </header>

      <form class="space-y-5" @submit.prevent="handleSave">
        <!-- 站点地址 + 回调预览 -->
        <section class="rounded-2xl border border-zinc-200 bg-white p-5 dark:border-zinc-800 dark:bg-zinc-900">
          <div class="max-w-lg">
            <label class="mb-1.5 block text-sm text-zinc-600 dark:text-zinc-400">站点地址</label>
            <NInput v-model:value="config.payment_site_url" placeholder="https://your-domain.com" />
          </div>
          <div v-if="siteUrl" class="mt-3 space-y-1">
            <p v-for="prov in ['wechat', 'alipay', 'epay']" :key="prov" class="font-mono text-xs text-zinc-500 dark:text-zinc-400">
              {{ siteUrl }}/api/payment/callback/{{ prov }}
            </p>
          </div>
        </section>

        <!-- 支付方式开关 + 参数 -->
        <div class="space-y-3">
          <div
            v-for="p in PROVIDERS"
            :key="p.key"
            class="overflow-hidden rounded-2xl border bg-white transition-colors dark:bg-zinc-900"
            :class="enabledProviders.includes(p.key)
              ? 'border-zinc-200 dark:border-zinc-700'
              : 'border-zinc-200/70 opacity-70 dark:border-zinc-800'"
          >
            <div class="flex items-center gap-3 px-4 py-3.5">
              <button
                type="button"
                class="flex min-w-0 flex-1 items-center gap-3 rounded-lg text-left focus-visible:outline-2 focus-visible:outline-offset-2 focus-visible:outline-emerald-500"
                :disabled="!enabledProviders.includes(p.key)"
                :aria-expanded="activeProvider === p.key"
                @click="enabledProviders.includes(p.key) && (activeProvider = activeProvider === p.key ? null : p.key)"
              >
                <span
                  class="grid size-9 shrink-0 place-items-center rounded-lg"
                  :class="enabledProviders.includes(p.key)
                    ? 'bg-emerald-50 text-emerald-600 dark:bg-emerald-400/10 dark:text-emerald-400'
                    : 'bg-zinc-100 text-zinc-500 dark:bg-zinc-800 dark:text-zinc-400'"
                >
                  <AppIcon :icon="providerIcons[p.key]" scale="md" />
                </span>
                <span class="min-w-0 flex-1">
                  <span class="block text-sm font-medium text-zinc-900 dark:text-zinc-100">{{ p.label }}</span>
                  <span class="mt-0.5 block truncate text-xs text-zinc-500 dark:text-zinc-400">{{ p.brief }}</span>
                </span>
                <AppIcon :icon="ChevronDown" scale="sm" v-if="enabledProviders.includes(p.key)"
                  
                  class="shrink-0 text-zinc-400 transition-transform"
                  :class="activeProvider === p.key ? 'rotate-180' : ''" />
              </button>
              <NSwitch
                :value="enabledProviders.includes(p.key)"
                size="small"
                :aria-label="`${enabledProviders.includes(p.key) ? '禁用' : '启用'} ${p.label}`"
                @update:value="() => toggleProvider(p.key)"
              />
            </div>

            <!-- 微信参数 -->
            <div
              v-if="p.key === 'wechat' && enabledProviders.includes('wechat') && activeProvider === 'wechat'"
              class="grid gap-4 border-t border-zinc-100 p-4 sm:grid-cols-2 dark:border-zinc-800"
            >
              <div class="sm:col-span-2 -mb-1 flex items-center gap-2">
                <span class="text-sm font-semibold text-zinc-900 dark:text-zinc-100">微信支付 V3 API</span>
              </div>
              <div>
                <label class="mb-1.5 block text-sm text-zinc-600 dark:text-zinc-400">AppID</label>
                <NInput v-model:value="config.payment_wx_appid" placeholder="wx..." />
              </div>
              <div>
                <label class="mb-1.5 block text-sm text-zinc-600 dark:text-zinc-400">商户号</label>
                <NInput v-model:value="config.payment_wx_mchid" placeholder="商户号" />
              </div>
              <div>
                <label class="mb-1.5 block text-sm text-zinc-600 dark:text-zinc-400">证书序列号</label>
                <NInput v-model:value="config.payment_wx_serial_no" placeholder="证书序列号" />
              </div>
              <div>
                <label class="mb-1.5 block text-sm text-zinc-600 dark:text-zinc-400">API V3 密钥</label>
                <NInput v-model:value="config.payment_wx_api_v3_key" type="password" show-password-on="click" placeholder="32 位密钥" />
              </div>
              <div class="sm:col-span-2">
                <label class="mb-1.5 block text-sm text-zinc-600 dark:text-zinc-400">商户 API 私钥 (PEM)</label>
                <NInput v-model:value="config.payment_wx_private_key" type="textarea" :rows="5" class="font-mono" placeholder="-----BEGIN PRIVATE KEY-----" />
              </div>
              <div class="sm:col-span-2">
                <label class="mb-1.5 block text-sm text-zinc-600 dark:text-zinc-400">回调通知地址</label>
                <NInput v-model:value="config.payment_wx_notify_url" :placeholder="siteUrl ? `${siteUrl}/api/payment/callback/wechat` : 'Notify URL'" />
                <p v-if="siteUrl && !config.payment_wx_notify_url" class="mt-1 text-xs text-zinc-500 dark:text-zinc-400">留空将自动使用以上的站点地址拼接</p>
              </div>
            </div>

            <!-- 支付宝参数 -->
            <div
              v-if="p.key === 'alipay' && enabledProviders.includes('alipay') && activeProvider === 'alipay'"
              class="grid gap-4 border-t border-zinc-100 p-4 sm:grid-cols-2 dark:border-zinc-800"
            >
              <div class="sm:col-span-2 -mb-1 flex flex-wrap items-center justify-between gap-2">
                <span class="text-sm font-semibold text-zinc-900 dark:text-zinc-100">支付宝</span>
                <NRadioGroup
                  size="small"
                  :value="config.payment_ali_scene"
                  @update:value="(v: string) => config.payment_ali_scene = v"
                >
                  <NRadioButton value="page">跳转支付</NRadioButton>
                  <NRadioButton value="precreate">扫码支付</NRadioButton>
                </NRadioGroup>
              </div>
              <div>
                <label class="mb-1.5 block text-sm text-zinc-600 dark:text-zinc-400">AppID</label>
                <NInput v-model:value="config.payment_ali_appid" placeholder="2021..." />
              </div>
              <div class="sm:col-span-2">
                <label class="mb-1.5 block text-sm text-zinc-600 dark:text-zinc-400">应用私钥 (RSA2)</label>
                <NInput v-model:value="config.payment_ali_private_key" type="textarea" :rows="5" class="font-mono" placeholder="-----BEGIN RSA PRIVATE KEY-----" />
              </div>
              <div class="sm:col-span-2">
                <label class="mb-1.5 block text-sm text-zinc-600 dark:text-zinc-400">支付宝公钥</label>
                <NInput v-model:value="config.payment_ali_public_key" type="textarea" :rows="5" class="font-mono" placeholder="-----BEGIN PUBLIC KEY-----" />
              </div>
              <div class="sm:col-span-2">
                <label class="mb-1.5 block text-sm text-zinc-600 dark:text-zinc-400">异步通知地址</label>
                <NInput v-model:value="config.payment_ali_notify_url" :placeholder="siteUrl ? `${siteUrl}/api/payment/callback/alipay` : 'Notify URL'" />
              </div>
              <div class="sm:col-span-2">
                <label class="mb-1.5 block text-sm text-zinc-600 dark:text-zinc-400">同步跳转地址</label>
                <NInput v-model:value="config.payment_ali_return_url" :placeholder="siteUrl ? `${siteUrl}/?from=alipay` : 'Return URL（可选）'" />
              </div>
            </div>

            <!-- 易支付参数 -->
            <div
              v-if="p.key === 'epay' && enabledProviders.includes('epay') && activeProvider === 'epay'"
              class="grid gap-4 border-t border-zinc-100 p-4 sm:grid-cols-2 dark:border-zinc-800"
            >
              <div class="sm:col-span-2 -mb-1 flex items-center gap-2">
                <span class="text-sm font-semibold text-zinc-900 dark:text-zinc-100">彩虹易支付</span>
                <NTag size="small" :bordered="false">API V1</NTag>
              </div>
              <div class="sm:col-span-2">
                <label class="mb-1.5 block text-sm text-zinc-600 dark:text-zinc-400">网关地址</label>
                <NInput v-model:value="config.payment_epay_gateway" placeholder="https://pay.example.com" />
                <p class="mt-1 text-xs text-zinc-500 dark:text-zinc-400">自动追加 /submit.php</p>
              </div>
              <div>
                <label class="mb-1.5 block text-sm text-zinc-600 dark:text-zinc-400">商户 ID (PID)</label>
                <NInput v-model:value="config.payment_epay_pid" placeholder="1001" />
              </div>
              <div>
                <label class="mb-1.5 block text-sm text-zinc-600 dark:text-zinc-400">商户密钥 (KEY)</label>
                <NInput v-model:value="config.payment_epay_key" type="password" show-password-on="click" placeholder="MD5 签名密钥" />
              </div>
              <div class="sm:col-span-2">
                <label class="mb-1.5 block text-sm text-zinc-600 dark:text-zinc-400">异步通知地址</label>
                <NInput v-model:value="config.payment_epay_notify_url" :placeholder="siteUrl ? `${siteUrl}/api/payment/callback/epay` : 'Notify URL'" />
              </div>
              <div class="sm:col-span-2">
                <label class="mb-1.5 block text-sm text-zinc-600 dark:text-zinc-400">同步跳转地址</label>
                <NInput v-model:value="config.payment_epay_return_url" :placeholder="siteUrl ? `${siteUrl}/?from=epay` : 'Return URL（可选）'" />
              </div>
            </div>
          </div>
        </div>

        <NAlert v-if="message" type="success" closable @close="message = ''">{{ message }}</NAlert>
        <NAlert v-if="error" type="error" closable @close="error = ''">{{ error }}</NAlert>

        <NButton attr-type="submit" type="primary" :loading="saving">保存配置</NButton>
      </form>
    </template>
  </div>
</template>

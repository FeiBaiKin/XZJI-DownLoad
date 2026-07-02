<script setup lang="ts">
// 支付配置：1:1 移植自原项目 Node-XZJI/frontend/src/pages/admin/PaymentConfig.jsx
// 样式来自全局 styles/admin-payment.css + admin-email.css
// 契约适配：GO GET/PUT 均使用 payment_* 键（原版后端用 camelCase），前端直接读写 payment_*
import { ref, computed, onMounted } from 'vue'
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

onMounted(() => { fetchConfig() })

async function fetchConfig() {
  try {
    const res = await api.get('/admin/payment-config')
    // GO 返回 payment_* 键，直接合并（原版返回 camelCase 并映射）
    config.value = { ...defaults, ...res.data }
  } catch (err) {
    console.error('获取支付配置异常:', err)
    error.value = '获取配置失败'
  } finally {
    loading.value = false
  }
}

function updateConfig(key: string, value: string) {
  config.value[key] = value
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
  try { providers = JSON.parse(config.value.payment_enabled_providers) } catch {}
  if (!Array.isArray(providers)) providers = []
  if (providers.includes(provider)) providers = providers.filter((p) => p !== provider)
  else providers.push(provider)
  config.value.payment_enabled_providers = JSON.stringify(providers)
  if (activeProvider.value === provider && !providers.includes(provider)) activeProvider.value = null
}

const enabledProviders = computed<string[]>(() => {
  try { const p = JSON.parse(config.value.payment_enabled_providers); return Array.isArray(p) ? p : [] } catch { return [] }
})
const siteUrl = computed(() => config.value.payment_site_url ? config.value.payment_site_url.replace(/\/+$/, '') : '')

const callbackPath = (prov: string) => `callback/${prov}`
</script>

<template>
  <div class="admin-content">
    <div v-if="loading" class="loading-state">加载中...</div>
    <template v-else>
      <div class="content-header">
        <h2 class="content-title">支付配置</h2>
        <span class="content-sub">管理可用支付方式及第三方支付参数</span>
      </div>
      <form @submit.prevent="handleSave">
        <div class="pay-global">
          <div class="pay-global-header">
            <div class="pay-global-field">
              <span class="pay-global-label">站点地址</span>
              <input type="text" class="pay-global-input" v-model="config.payment_site_url" placeholder="https://your-domain.com" />
            </div>
          </div>
          <div v-if="siteUrl" class="pay-callback-preview">
            <div class="pay-callback-list">
              <span v-for="prov in ['wechat','alipay','epay']" :key="prov" class="pay-callback-item">
                <span class="pay-callback-dot" /><span>{{ siteUrl }}/api/payment/{{ callbackPath(prov) }}</span>
              </span>
            </div>
          </div>
        </div>

        <div class="pay-providers-v2">
          <div v-for="p in PROVIDERS" :key="p.key" class="pay-prov-row" :class="{ active: activeProvider === p.key, disabled: !enabledProviders.includes(p.key) }">
            <div class="pay-prov-row-main" @click="enabledProviders.includes(p.key) && (activeProvider = activeProvider === p.key ? null : p.key)">
              <span class="pay-prov-icon">{{ p.icon }}</span>
              <div class="pay-prov-row-info">
                <span class="pay-prov-row-label">{{ p.label }}</span>
                <span class="pay-prov-row-brief">{{ p.brief }}</span>
              </div>
              <button type="button" class="email-toggle-switch" :class="{ on: enabledProviders.includes(p.key) }" :aria-checked="enabledProviders.includes(p.key)" :aria-label="`${enabledProviders.includes(p.key) ? '禁用' : '启用'} ${p.label}`" role="switch" @click.stop="toggleProvider(p.key)">
                <span class="email-toggle-knob" />
              </button>
            </div>
          </div>
        </div>

        <!-- 微信 -->
        <div v-if="enabledProviders.includes('wechat') && activeProvider === 'wechat'" class="pay-prov-config">
          <div class="pay-prov-config-header"><span class="pay-prov-config-title">微信支付 V3 API</span></div>
          <div class="pay-fields">
            <div class="pay-field"><span class="pay-field-label">AppID</span><input type="text" class="pay-field-input" placeholder="wx..." v-model="config.payment_wx_appid" /></div>
            <div class="pay-field"><span class="pay-field-label">商户号</span><input type="text" class="pay-field-input" placeholder="商户号" v-model="config.payment_wx_mchid" /></div>
            <div class="pay-field"><span class="pay-field-label">证书序列号</span><input type="text" class="pay-field-input" placeholder="证书序列号" v-model="config.payment_wx_serial_no" /></div>
            <div class="pay-field"><span class="pay-field-label">API V3 密钥</span><input type="password" class="pay-field-input" placeholder="32位密钥" v-model="config.payment_wx_api_v3_key" /></div>
            <div class="pay-field" style="grid-column: 1 / -1;"><span class="pay-field-label">商户 API 私钥 (PEM)</span><textarea class="pay-field-input pay-field-textarea" placeholder="-----BEGIN PRIVATE KEY-----" rows="5" v-model="config.payment_wx_private_key" /></div>
            <div class="pay-field" style="grid-column: 1 / -1;"><span class="pay-field-label">回调通知地址</span><input type="text" class="pay-field-input" :placeholder="siteUrl ? `${siteUrl}/api/payment/callback/wechat` : 'Notify URL'" v-model="config.payment_wx_notify_url" /><span v-if="siteUrl && !config.payment_wx_notify_url" class="pay-field-hint">留空将自动使用以上的站点地址拼接</span></div>
          </div>
        </div>

        <!-- 支付宝 -->
        <div v-if="enabledProviders.includes('alipay') && activeProvider === 'alipay'" class="pay-prov-config">
          <div class="pay-prov-config-header">
            <span class="pay-prov-config-title">支付宝</span>
            <div class="pay-scene-toggle">
              <button type="button" class="pay-scene-btn" :class="{ active: config.payment_ali_scene === 'page' }" @click="updateConfig('payment_ali_scene', 'page')">跳转支付</button>
              <button type="button" class="pay-scene-btn" :class="{ active: config.payment_ali_scene === 'precreate' }" @click="updateConfig('payment_ali_scene', 'precreate')">扫码支付</button>
            </div>
          </div>
          <div class="pay-fields">
            <div class="pay-field"><span class="pay-field-label">AppID</span><input type="text" class="pay-field-input" placeholder="2021..." v-model="config.payment_ali_appid" /></div>
            <div class="pay-field" />
            <div class="pay-field" style="grid-column: 1 / -1;"><span class="pay-field-label">应用私钥 (RSA2)</span><textarea class="pay-field-input pay-field-textarea" placeholder="-----BEGIN RSA PRIVATE KEY-----" rows="5" v-model="config.payment_ali_private_key" /></div>
            <div class="pay-field" style="grid-column: 1 / -1;"><span class="pay-field-label">支付宝公钥</span><textarea class="pay-field-input pay-field-textarea" placeholder="-----BEGIN PUBLIC KEY-----" rows="5" v-model="config.payment_ali_public_key" /></div>
            <div class="pay-field" style="grid-column: 1 / -1;"><span class="pay-field-label">异步通知地址</span><input type="text" class="pay-field-input" :placeholder="siteUrl ? `${siteUrl}/api/payment/callback/alipay` : 'Notify URL'" v-model="config.payment_ali_notify_url" /></div>
            <div class="pay-field" style="grid-column: 1 / -1;"><span class="pay-field-label">同步跳转地址</span><input type="text" class="pay-field-input" :placeholder="siteUrl ? `${siteUrl}/?from=alipay` : 'Return URL（可选）'" v-model="config.payment_ali_return_url" /></div>
          </div>
        </div>

        <!-- 易支付 -->
        <div v-if="enabledProviders.includes('epay') && activeProvider === 'epay'" class="pay-prov-config">
          <div class="pay-prov-config-header">
            <span class="pay-prov-config-title">彩虹易支付</span>
            <span class="pay-field-label" style="padding: 0.25rem 0.75rem; border: 1px solid var(--border-default);">API V1</span>
          </div>
          <div class="pay-fields">
            <div class="pay-field" style="grid-column: 1 / -1;"><span class="pay-field-label">网关地址</span><input type="text" class="pay-field-input" placeholder="https://pay.example.com" v-model="config.payment_epay_gateway" /><span class="pay-field-hint">自动追加 /submit.php</span></div>
            <div class="pay-field"><span class="pay-field-label">商户 ID (PID)</span><input type="text" class="pay-field-input" placeholder="1001" v-model="config.payment_epay_pid" /></div>
            <div class="pay-field"><span class="pay-field-label">商户密钥 (KEY)</span><input type="password" class="pay-field-input" placeholder="MD5 签名密钥" v-model="config.payment_epay_key" /></div>
            <div class="pay-field" style="grid-column: 1 / -1;"><span class="pay-field-label">异步通知地址</span><input type="text" class="pay-field-input" :placeholder="siteUrl ? `${siteUrl}/api/payment/callback/epay` : 'Notify URL'" v-model="config.payment_epay_notify_url" /></div>
            <div class="pay-field" style="grid-column: 1 / -1;"><span class="pay-field-label">同步跳转地址</span><input type="text" class="pay-field-input" :placeholder="siteUrl ? `${siteUrl}/?from=epay` : 'Return URL（可选）'" v-model="config.payment_epay_return_url" /></div>
          </div>
        </div>

        <div v-if="message" class="form-success" style="margin-bottom: 1rem;">{{ message }}</div>
        <div v-if="error" class="form-error" style="margin-bottom: 1rem;">{{ error }}</div>
        <div class="form-actions"><button type="submit" class="action-button" :disabled="saving">{{ saving ? '保存中...' : '保存配置' }}</button></div>
      </form>
    </template>
  </div>
</template>

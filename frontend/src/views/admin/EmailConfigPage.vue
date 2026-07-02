<script setup lang="ts">
// 邮箱配置：1:1 移植自原项目 Node-XZJI/frontend/src/pages/admin/EmailConfig.jsx
// 样式来自全局 styles/admin-email.css + admin-payment.css
import { ref, computed, onMounted } from 'vue'
import api from '@/api'

interface EmailConfigData {
  email_smtp_host: string
  email_smtp_port: string
  email_smtp_user: string
  email_smtp_pass: string
  email_from_name: string
  email_force_verify: string
  email_enable_passwordless: string
  [key: string]: string
}

const config = ref<EmailConfigData>({
  email_smtp_host: '', email_smtp_port: '465', email_smtp_user: '', email_smtp_pass: '',
  email_from_name: '', email_force_verify: '0', email_enable_passwordless: '0',
})
const loading = ref(true)
const saving = ref(false)
const message = ref('')
const error = ref('')
const testing = ref(false)
const testTo = ref('')
const testResult = ref('')

onMounted(() => { fetchConfig() })

async function fetchConfig() {
  try {
    const res = await api.get('/admin/email-config')
    const data = res.data
    config.value = {
      email_smtp_host: data.smtpHost || '',
      email_smtp_port: String(data.smtpPort || '465'),
      email_smtp_user: data.smtpUser || '',
      email_smtp_pass: data.smtpPass || '',
      email_from_name: data.fromName || '',
      email_force_verify: data.forceVerify ? '1' : '0',
      email_enable_passwordless: data.enablePasswordless ? '1' : '0',
    }
  } catch (err) {
    console.error('获取邮箱配置异常:', err)
    error.value = '获取配置失败'
  } finally {
    loading.value = false
  }
}

function updateConfig(key: string, value: string) { config.value[key] = value }

async function handleSave() {
  saving.value = true; message.value = ''; error.value = ''
  try {
    await api.put('/admin/email-config', config.value)
    message.value = '邮箱配置已保存'
    await fetchConfig()
  } catch (err: unknown) {
    const errorObj = err as { response?: { data?: { error?: string } } }
    error.value = errorObj?.response?.data?.error || '保存失败'
  } finally {
    saving.value = false
  }
}

async function handleTestEmail() {
  if (!testTo.value.trim()) { testResult.value = '请输入测试收件邮箱'; return }
  testing.value = true; testResult.value = ''
  try {
    const res = await api.post('/admin/email-test', { to: testTo.value.trim() })
    testResult.value = res.data.message || '发送成功'
  } catch (err: unknown) {
    const errorObj = err as { response?: { data?: { error?: string } } }
    testResult.value = errorObj?.response?.data?.error || '发送失败'
  } finally {
    testing.value = false
  }
}

const forceVerify = computed(() => config.value.email_force_verify === '1')
const enablePasswordless = computed(() => config.value.email_enable_passwordless === '1')
</script>

<template>
  <div class="admin-content">
    <div v-if="loading">
      <div class="content-header"><h2 class="content-title">邮箱配置</h2></div>
      <div class="loading-state">加载中...</div>
    </div>
    <template v-else>
      <div class="content-header">
        <h2 class="content-title">邮箱配置</h2>
        <span class="content-sub">SMTP 发件服务器、验证码策略与测试</span>
      </div>
      <form @submit.prevent="handleSave">
        <div class="form-card">
          <div class="form-card-title">SMTP 服务器</div>
          <div class="pay-fields">
            <div class="pay-field"><span class="pay-field-label">SMTP 主机</span><input type="text" class="pay-field-input" placeholder="smtp.example.com" v-model="config.email_smtp_host" /></div>
            <div class="pay-field"><span class="pay-field-label">端口</span><input type="text" class="pay-field-input" placeholder="465" v-model="config.email_smtp_port" /></div>
            <div class="pay-field"><span class="pay-field-label">发件账号</span><input type="text" class="pay-field-input" placeholder="noreply@example.com" v-model="config.email_smtp_user" /></div>
            <div class="pay-field"><span class="pay-field-label">授权码 / 密码</span><input type="password" class="pay-field-input" placeholder="SMTP 授权码" v-model="config.email_smtp_pass" /></div>
            <div class="pay-field" style="grid-column: 1 / -1;"><span class="pay-field-label">发件人名称</span><input type="text" class="pay-field-input" placeholder="系统通知" v-model="config.email_from_name" /><span class="pay-field-hint">显示在邮件「发件人」字段</span></div>
          </div>
        </div>

        <div class="form-card">
          <div class="form-card-title">功能开关</div>
          <div class="email-toggles">
            <div class="email-toggle-row">
              <div class="email-toggle-info"><span class="email-toggle-label">注册强制邮箱验证</span><span class="email-toggle-desc">开启后，注册时必须验证邮箱</span></div>
              <button type="button" class="email-toggle-switch" :class="{ on: forceVerify }" :aria-checked="forceVerify" role="switch" @click="updateConfig('email_force_verify', forceVerify ? '0' : '1')"><span class="email-toggle-knob" /></button>
            </div>
            <div class="email-toggle-row">
              <div class="email-toggle-info"><span class="email-toggle-label">启用邮箱验证码登录</span><span class="email-toggle-desc">用户可使用邮箱验证码免密登录</span></div>
              <button type="button" class="email-toggle-switch" :class="{ on: enablePasswordless }" :aria-checked="enablePasswordless" role="switch" @click="updateConfig('email_enable_passwordless', enablePasswordless ? '0' : '1')"><span class="email-toggle-knob" /></button>
            </div>
          </div>
          <div v-if="forceVerify || enablePasswordless" class="email-toggle-note"><span class="pay-callback-dot" />验证码有效期 10 分钟，同一邮箱 60 秒内不可重复发送</div>
        </div>

        <div class="form-card">
          <div class="form-card-title">发送测试</div>
          <div class="email-test-row">
            <input type="email" class="pay-field-input" style="flex: 1;" placeholder="输入测试收件邮箱" v-model="testTo" />
            <button type="button" class="action-button secondary" :disabled="testing" @click="handleTestEmail">{{ testing ? '发送中...' : '发送测试邮件' }}</button>
          </div>
          <div v-if="testResult" class="email-test-result" :class="{ error: !testResult.includes('成功') }">{{ testResult }}</div>
        </div>

        <div v-if="message" class="form-success" style="margin-bottom: 1rem;">{{ message }}</div>
        <div v-if="error" class="form-error" style="margin-bottom: 1rem;">{{ error }}</div>
        <div class="form-actions"><button type="submit" class="action-button" :disabled="saving">{{ saving ? '保存中...' : '保存配置' }}</button></div>
      </form>
    </template>
  </div>
</template>

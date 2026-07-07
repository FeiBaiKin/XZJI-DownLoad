<script setup lang="ts">
// 邮箱配置：SMTP 发件服务器、验证码策略开关、发送测试
// 业务逻辑保持不变（config 键值均为字符串 '0'/'1'）；UI 重构为 Naive UI + Tailwind
import { ref, computed, onMounted } from 'vue'
import { NButton, NInput, NAlert, NSwitch, NSkeleton } from 'naive-ui'
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
  <div>
    <div v-if="loading" class="space-y-4">
      <NSkeleton text width="20%" size="large" />
      <NSkeleton height="240px" :sharp="false" />
      <NSkeleton height="160px" :sharp="false" />
    </div>

    <template v-else>
      <header class="mb-6">
        <h2 class="text-xl font-bold tracking-tight text-zinc-900 dark:text-zinc-50">邮箱配置</h2>
        <p class="mt-1 text-sm text-zinc-500 dark:text-zinc-400">SMTP 发件服务器、验证码策略与测试</p>
      </header>

      <form class="space-y-5" @submit.prevent="handleSave">
        <!-- SMTP 服务器 -->
        <section class="rounded-2xl border border-zinc-200 bg-white p-5 dark:border-zinc-800 dark:bg-zinc-900">
          <h3 class="mb-4 text-[15px] font-semibold text-zinc-900 dark:text-zinc-100">SMTP 服务器</h3>
          <div class="grid gap-4 sm:grid-cols-2">
            <div>
              <label class="mb-1.5 block text-sm text-zinc-600 dark:text-zinc-400">SMTP 主机</label>
              <NInput v-model:value="config.email_smtp_host" placeholder="smtp.example.com" />
            </div>
            <div>
              <label class="mb-1.5 block text-sm text-zinc-600 dark:text-zinc-400">端口</label>
              <NInput v-model:value="config.email_smtp_port" placeholder="465" :input-props="{ inputmode: 'numeric' }" />
            </div>
            <div>
              <label class="mb-1.5 block text-sm text-zinc-600 dark:text-zinc-400">发件账号</label>
              <NInput v-model:value="config.email_smtp_user" placeholder="noreply@example.com" />
            </div>
            <div>
              <label class="mb-1.5 block text-sm text-zinc-600 dark:text-zinc-400">授权码 / 密码</label>
              <NInput v-model:value="config.email_smtp_pass" type="password" show-password-on="click" placeholder="SMTP 授权码" />
            </div>
            <div class="sm:col-span-2">
              <label class="mb-1.5 block text-sm text-zinc-600 dark:text-zinc-400">发件人名称</label>
              <NInput v-model:value="config.email_from_name" placeholder="系统通知" />
              <p class="mt-1 text-xs text-zinc-500 dark:text-zinc-400">显示在邮件「发件人」字段</p>
            </div>
          </div>
        </section>

        <!-- 功能开关 -->
        <section class="rounded-2xl border border-zinc-200 bg-white p-5 dark:border-zinc-800 dark:bg-zinc-900">
          <h3 class="mb-4 text-[15px] font-semibold text-zinc-900 dark:text-zinc-100">功能开关</h3>
          <div class="space-y-3">
            <div class="flex flex-col items-start gap-3 rounded-xl bg-zinc-50 px-4 py-3 sm:flex-row sm:items-center sm:justify-between dark:bg-zinc-800/60">
              <div>
                <p class="text-sm font-medium text-zinc-800 dark:text-zinc-200">注册强制邮箱验证</p>
                <p class="mt-0.5 text-xs text-zinc-500 dark:text-zinc-400">开启后，注册时必须验证邮箱</p>
              </div>
              <NSwitch
                :value="forceVerify"
                size="small"
                @update:value="(v: boolean) => updateConfig('email_force_verify', v ? '1' : '0')"
              />
            </div>
            <div class="flex flex-col items-start gap-3 rounded-xl bg-zinc-50 px-4 py-3 sm:flex-row sm:items-center sm:justify-between dark:bg-zinc-800/60">
              <div>
                <p class="text-sm font-medium text-zinc-800 dark:text-zinc-200">启用邮箱验证码登录</p>
                <p class="mt-0.5 text-xs text-zinc-500 dark:text-zinc-400">用户可使用邮箱验证码免密登录</p>
              </div>
              <NSwitch
                :value="enablePasswordless"
                size="small"
                @update:value="(v: boolean) => updateConfig('email_enable_passwordless', v ? '1' : '0')"
              />
            </div>
          </div>
          <p v-if="forceVerify || enablePasswordless" class="mt-3 text-xs text-zinc-500 dark:text-zinc-400">
            验证码有效期 10 分钟，同一邮箱 60 秒内不可重复发送
          </p>
        </section>

        <!-- 发送测试 -->
        <section class="rounded-2xl border border-zinc-200 bg-white p-5 dark:border-zinc-800 dark:bg-zinc-900">
          <h3 class="mb-4 text-[15px] font-semibold text-zinc-900 dark:text-zinc-100">发送测试</h3>
          <div class="flex max-w-lg flex-col gap-2 sm:flex-row">
            <NInput v-model:value="testTo" class="flex-1" placeholder="输入测试收件邮箱" :input-props="{ type: 'email' }" />
            <NButton secondary :loading="testing" @click="handleTestEmail">发送测试邮件</NButton>
          </div>
          <p
            v-if="testResult"
            class="mt-3 rounded-lg px-3 py-2 text-sm"
            :class="testResult.includes('成功')
              ? 'bg-emerald-50 text-emerald-700 dark:bg-emerald-400/10 dark:text-emerald-400'
              : 'bg-red-50 text-red-600 dark:bg-red-500/10 dark:text-red-400'"
          >{{ testResult }}</p>
        </section>

        <NAlert v-if="message" type="success" closable @close="message = ''">{{ message }}</NAlert>
        <NAlert v-if="error" type="error" closable @close="error = ''">{{ error }}</NAlert>

        <NButton attr-type="submit" type="primary" :loading="saving">保存配置</NButton>
      </form>
    </template>
  </div>
</template>

<script setup lang="ts">
// 忘记密码页：三步流程 request(发码) → verify(验码) → reset(重置)
// 业务逻辑保持不变；UI 重构为 AuthShell + Naive UI（步骤条指示进度）
import { ref, computed, onUnmounted, watch } from 'vue'
import { useRouter } from 'vue-router'
import { NButton, NInput, NSteps, NStep } from 'naive-ui'
import { authApi } from '@/api/auth'
import AuthShell from '@/components/AuthShell.vue'

const router = useRouter()

const step = ref<'request' | 'verify' | 'reset'>('request')
const email = ref('')
const code = ref('')
const password = ref('')
const confirmPassword = ref('')
const loading = ref(false)
const error = ref('')
const message = ref('')
const sendingCode = ref(false)
const cooldown = ref(0)

const stepIndex = computed(() => (step.value === 'request' ? 1 : step.value === 'verify' ? 2 : 3))
const subtitle = computed(() => {
  if (step.value === 'request') return '输入注册邮箱，获取验证码'
  if (step.value === 'verify') return `验证码已发送至 ${email.value}`
  return '请输入您的新密码'
})

// 倒计时
let timer: ReturnType<typeof setInterval> | null = null
watch(cooldown, (v) => {
  if (v <= 0) { if (timer) { clearInterval(timer); timer = null }; return }
  if (!timer) {
    timer = setInterval(() => { cooldown.value = cooldown.value <= 1 ? 0 : cooldown.value - 1 }, 1000)
  }
})
onUnmounted(() => { if (timer) clearInterval(timer) })

// 步骤1：发送验证码
async function handleSendCode() {
  error.value = ''
  message.value = ''
  if (!email.value.trim()) { error.value = '请输入注册邮箱'; return }
  if (!/^[^\s@]+@[^\s@]+\.[^\s@]+$/.test(email.value.trim())) { error.value = '邮箱格式不正确'; return }

  sendingCode.value = true
  try {
    await authApi.sendCode(email.value.trim(), 'forgot-password')
    message.value = '验证码已发送，请查收邮箱'
    cooldown.value = 60
    step.value = 'verify'
  } catch (err: unknown) {
    error.value = (err as { response?: { data?: { error?: string } } })?.response?.data?.error || '发送失败'
  } finally {
    sendingCode.value = false
  }
}

// 步骤2：验证验证码
async function handleVerifyCode() {
  error.value = ''
  message.value = ''
  if (!code.value.trim()) { error.value = '请输入验证码'; return }
  if (code.value.trim().length !== 6) { error.value = '验证码为6位数字'; return }

  loading.value = true
  try {
    await authApi.verifyCode(email.value.trim(), code.value.trim(), 'forgot-password')
    message.value = ''
    step.value = 'reset'
  } catch (err: unknown) {
    error.value = (err as { response?: { data?: { error?: string } } })?.response?.data?.error || '验证失败'
  } finally {
    loading.value = false
  }
}

// 步骤3：重置密码
async function handleResetPassword() {
  error.value = ''
  message.value = ''
  if (!password.value) { error.value = '请输入新密码'; return }
  if (password.value.length < 6) { error.value = '密码至少需要6个字符'; return }
  if (password.value !== confirmPassword.value) { error.value = '两次输入的密码不一致'; return }

  loading.value = true
  try {
    await authApi.resetPassword(email.value.trim(), code.value.trim(), password.value)
    message.value = '密码重置成功，即将跳转到登录页...'
    setTimeout(() => router.push('/login'), 1500)
  } catch (err: unknown) {
    error.value = (err as { response?: { data?: { error?: string } } })?.response?.data?.error || '重置失败'
  } finally {
    loading.value = false
  }
}
</script>

<template>
  <AuthShell
    :title="step === 'reset' ? '重置密码' : '找回密码'"
    :subtitle="subtitle"
    back-to="/login"
    back-label="返回登录"
  >
    <NSteps :current="stepIndex" size="small" class="mb-8">
      <NStep title="获取验证码" />
      <NStep title="验证邮箱" />
      <NStep title="重置密码" />
    </NSteps>

    <!-- 步骤1：发送验证码 -->
    <form v-if="step === 'request'" class="space-y-5" @submit.prevent="handleSendCode">
      <div>
        <label for="email" class="mb-1.5 block text-sm font-medium text-zinc-700 dark:text-zinc-300">注册邮箱</label>
        <NInput
          id="email"
          v-model:value="email"
          size="large"
          placeholder="输入您的注册邮箱"
          :disabled="sendingCode"
          :input-props="{ type: 'email', autocomplete: 'email' }"
        />
      </div>

      <p v-if="error" class="rounded-lg bg-red-50 px-3 py-2 text-sm text-red-600 dark:bg-red-500/10 dark:text-red-400">{{ error }}</p>
      <p v-if="message" class="rounded-lg bg-emerald-50 px-3 py-2 text-sm text-emerald-700 dark:bg-emerald-400/10 dark:text-emerald-400">{{ message }}</p>

      <NButton attr-type="submit" type="primary" size="large" block :loading="sendingCode">发送验证码</NButton>
    </form>

    <!-- 步骤2：验证验证码 -->
    <form v-else-if="step === 'verify'" class="space-y-5" @submit.prevent="handleVerifyCode">
      <div>
        <label for="code" class="mb-1.5 block text-sm font-medium text-zinc-700 dark:text-zinc-300">验证码</label>
        <div class="flex gap-2">
          <NInput
            id="code"
            v-model:value="code"
            size="large"
            class="flex-1 text-center font-mono tracking-[0.3em]"
            placeholder="6 位数字验证码"
            maxlength="6"
            :disabled="loading"
            :input-props="{ autocomplete: 'one-time-code', inputmode: 'numeric' }"
          />
          <NButton
            size="large"
            secondary
            :disabled="sendingCode || cooldown > 0 || loading"
            :loading="sendingCode"
            @click="handleSendCode"
          >
            {{ cooldown > 0 ? `${cooldown}s` : '重新发送' }}
          </NButton>
        </div>
      </div>

      <p v-if="error" class="rounded-lg bg-red-50 px-3 py-2 text-sm text-red-600 dark:bg-red-500/10 dark:text-red-400">{{ error }}</p>
      <p v-if="message" class="rounded-lg bg-emerald-50 px-3 py-2 text-sm text-emerald-700 dark:bg-emerald-400/10 dark:text-emerald-400">{{ message }}</p>

      <NButton attr-type="submit" type="primary" size="large" block :loading="loading">验证</NButton>
    </form>

    <!-- 步骤3：重置密码 -->
    <form v-else class="space-y-5" @submit.prevent="handleResetPassword">
      <div>
        <label for="password" class="mb-1.5 block text-sm font-medium text-zinc-700 dark:text-zinc-300">新密码</label>
        <NInput
          id="password"
          v-model:value="password"
          type="password"
          show-password-on="click"
          size="large"
          placeholder="至少 6 个字符"
          :disabled="loading"
          :input-props="{ autocomplete: 'new-password' }"
        />
      </div>

      <div>
        <label for="confirmPassword" class="mb-1.5 block text-sm font-medium text-zinc-700 dark:text-zinc-300">确认新密码</label>
        <NInput
          id="confirmPassword"
          v-model:value="confirmPassword"
          type="password"
          show-password-on="click"
          size="large"
          placeholder="再次输入新密码"
          :disabled="loading"
          :input-props="{ autocomplete: 'new-password' }"
        />
      </div>

      <p v-if="error" class="rounded-lg bg-red-50 px-3 py-2 text-sm text-red-600 dark:bg-red-500/10 dark:text-red-400">{{ error }}</p>
      <p v-if="message" class="rounded-lg bg-emerald-50 px-3 py-2 text-sm text-emerald-700 dark:bg-emerald-400/10 dark:text-emerald-400">{{ message }}</p>

      <NButton attr-type="submit" type="primary" size="large" block :loading="loading">重置密码</NButton>
    </form>

    <template #footer>
      <router-link to="/login" class="font-medium text-emerald-600 hover:underline dark:text-emerald-400">返回登录</router-link>
    </template>
  </AuthShell>
</template>

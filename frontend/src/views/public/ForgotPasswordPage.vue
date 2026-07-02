<script setup lang="ts">
// 忘记密码页：1:1 移植自原项目 Node-XZJI/frontend/src/pages/ForgotPasswordPage.jsx
// 三步：request(发码) → verify(验码) → reset(重置)
// 样式来自全局 styles/auth.css + verification.css
import { ref, onUnmounted, watch } from 'vue'
import { useRouter } from 'vue-router'
import { authApi } from '@/api/auth'

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
  } catch (err: any) {
    error.value = err?.response?.data?.error || '发送失败'
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
  } catch (err: any) {
    error.value = err?.response?.data?.error || '验证失败'
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
  } catch (err: any) {
    error.value = err?.response?.data?.error || '重置失败'
  } finally {
    loading.value = false
  }
}
</script>

<template>
  <div class="auth-page">
    <div class="auth-container">
      <div class="auth-header">
        <router-link to="/login" class="auth-back">
          <span class="auth-back-icon">&#8592;</span>
          <span>返回登录</span>
        </router-link>
        <h1 class="auth-title">{{ step === 'reset' ? '重置密码' : '找回密码' }}</h1>
        <p class="auth-subtitle">
          <template v-if="step === 'request'">输入注册邮箱，获取验证码</template>
          <template v-else-if="step === 'verify'">验证码已发送至 {{ email }}</template>
          <template v-else>请输入您的新密码</template>
        </p>
      </div>

      <!-- 步骤1：发送验证码 -->
      <form v-if="step === 'request'" class="auth-form" @submit.prevent="handleSendCode">
        <div class="form-group">
          <label for="email" class="form-label">注册邮箱</label>
          <input id="email" type="email" class="form-input" placeholder="输入您的注册邮箱" v-model="email" :disabled="sendingCode" autocomplete="email" />
        </div>

        <div v-if="error" class="form-error">{{ error }}</div>
        <div v-if="message" class="form-success">{{ message }}</div>

        <button type="submit" class="auth-button" :disabled="sendingCode">
          {{ sendingCode ? '发送中...' : '发送验证码' }}
        </button>
      </form>

      <!-- 步骤2：验证验证码 -->
      <form v-else-if="step === 'verify'" class="auth-form" @submit.prevent="handleVerifyCode">
        <div class="form-group">
          <label for="code" class="form-label">验证码</label>
          <input
            id="code"
            type="text"
            class="form-input"
            placeholder="6位数字验证码"
            maxlength="6"
            v-model="code"
            :disabled="loading"
            autocomplete="one-time-code"
            style="letter-spacing: 0.2em; text-align: center; font-size: 1.1rem;"
          />
        </div>

        <div style="display: flex; gap: 0.5rem;">
          <button
            type="button"
            class="code-send-btn"
            @click="handleSendCode"
            :disabled="sendingCode || cooldown > 0 || loading"
            style="flex: 0 0 auto;"
          >
            {{ sendingCode ? '发送中...' : cooldown > 0 ? `${cooldown}s` : '重新发送' }}
          </button>
        </div>

        <div v-if="error" class="form-error">{{ error }}</div>
        <div v-if="message" class="form-success">{{ message }}</div>

        <button type="submit" class="auth-button" :disabled="loading">
          {{ loading ? '验证中...' : '验证' }}
        </button>
      </form>

      <!-- 步骤3：重置密码 -->
      <form v-else class="auth-form" @submit.prevent="handleResetPassword">
        <div class="form-group">
          <label for="password" class="form-label">新密码</label>
          <input id="password" type="password" class="form-input" placeholder="至少6个字符" v-model="password" :disabled="loading" autocomplete="new-password" />
        </div>

        <div class="form-group">
          <label for="confirmPassword" class="form-label">确认新密码</label>
          <input id="confirmPassword" type="password" class="form-input" placeholder="再次输入新密码" v-model="confirmPassword" :disabled="loading" autocomplete="new-password" />
        </div>

        <div v-if="error" class="form-error">{{ error }}</div>
        <div v-if="message" class="form-success">{{ message }}</div>

        <button type="submit" class="auth-button" :disabled="loading">
          {{ loading ? '重置中...' : '重置密码' }}
        </button>
      </form>

      <div class="auth-footer">
        <p class="auth-switch">
          <router-link to="/login" class="auth-link">返回登录</router-link>
        </p>
      </div>
    </div>
  </div>
</template>

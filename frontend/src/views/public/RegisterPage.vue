<script setup lang="ts">
// 注册页：1:1 移植自原项目 Node-XZJI/frontend/src/pages/RegisterPage.jsx
// 样式来自全局 styles/auth.css + verification.css
import { ref, onMounted, onUnmounted, watch } from 'vue'
import { useRouter } from 'vue-router'
import { useAuthStore } from '@/stores/auth'
import { authApi } from '@/api/auth'
import { contentApi } from '@/api/content'

const router = useRouter()
const auth = useAuthStore()

const username = ref('')
const password = ref('')
const confirmPassword = ref('')
const email = ref('')
const code = ref('')
const loading = ref(false)
const error = ref('')
const success = ref('')
const sendingCode = ref(false)
const cooldown = ref(0)
const forceVerify = ref(false)

onMounted(async () => {
  try {
    const cfg = await contentApi.public.siteConfig()
    forceVerify.value = !!cfg.forceVerify
  } catch { /* 忽略 */ }
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

async function handleSendCode() {
  if (!email.value.trim()) { error.value = '请先填写邮箱地址'; return }
  if (!/^[^\s@]+@[^\s@]+\.[^\s@]+$/.test(email.value.trim())) { error.value = '邮箱格式不正确'; return }

  sendingCode.value = true
  error.value = ''
  try {
    await authApi.sendCode(email.value.trim(), 'register')
    success.value = '验证码已发送，请查收邮箱'
    cooldown.value = 60
  } catch (err: any) {
    error.value = err?.response?.data?.error || '发送失败'
  } finally {
    sendingCode.value = false
  }
}

async function handleSubmit() {
  error.value = ''
  success.value = ''

  if (!username.value.trim() || !password.value) { error.value = '请填写用户名和密码'; return }
  if (username.value.trim().length < 3) { error.value = '用户名至少需要3个字符'; return }
  if (password.value.length < 6) { error.value = '密码至少需要6个字符'; return }
  if (password.value !== confirmPassword.value) { error.value = '两次输入的密码不一致'; return }

  if (forceVerify.value) {
    if (!email.value.trim()) { error.value = '请填写邮箱地址'; return }
    if (!code.value.trim()) { error.value = '请输入邮箱验证码'; return }
  }

  loading.value = true
  try {
    const body: { username: string; password: string; email?: string; code?: string } = {
      username: username.value.trim(),
      password: password.value,
      email: email.value.trim() || undefined,
    }
    if (forceVerify.value && code.value.trim()) body.code = code.value.trim()

    await auth.register(body)
    success.value = '注册成功，正在跳转...'
    setTimeout(() => router.push('/user'), 800)
  } catch (err: any) {
    error.value = err?.response?.data?.error || '注册失败'
  } finally {
    loading.value = false
  }
}
</script>

<template>
  <div class="auth-page">
    <div class="auth-container">
      <div class="auth-header">
        <router-link to="/" class="auth-back">
          <span class="auth-back-icon">&#8592;</span>
          <span>返回首页</span>
        </router-link>
        <h1 class="auth-title">创建账户</h1>
        <p class="auth-subtitle">注册一个新账户以使用全部功能</p>
      </div>

      <form class="auth-form" @submit.prevent="handleSubmit">
        <div class="form-group">
          <label for="username" class="form-label">用户名</label>
          <input id="username" type="text" class="form-input" placeholder="至少3个字符" v-model="username" :disabled="loading" autocomplete="username" />
        </div>

        <div class="form-group">
          <label for="email" class="form-label">
            邮箱
            <span v-if="!forceVerify" class="form-label-optional">选填</span>
            <span v-if="forceVerify" style="color: var(--danger);"> *</span>
          </label>
          <div style="display: flex; gap: 0.5rem;">
            <input
              id="email"
              type="email"
              class="form-input"
              style="flex: 1;"
              :placeholder="forceVerify ? '必填，用于接收验证码' : '用于找回密码'"
              v-model="email"
              :disabled="loading"
              autocomplete="email"
            />
            <button
              v-if="forceVerify"
              type="button"
              class="code-send-btn"
              @click="handleSendCode"
              :disabled="sendingCode || cooldown > 0 || loading"
            >
              {{ sendingCode ? '发送中...' : cooldown > 0 ? `${cooldown}s` : '发送验证码' }}
            </button>
          </div>
        </div>

        <div v-if="forceVerify" class="form-group">
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

        <div class="form-group">
          <label for="password" class="form-label">密码</label>
          <input id="password" type="password" class="form-input" placeholder="至少6个字符" v-model="password" :disabled="loading" autocomplete="new-password" />
        </div>

        <div class="form-group">
          <label for="confirmPassword" class="form-label">确认密码</label>
          <input id="confirmPassword" type="password" class="form-input" placeholder="再次输入密码" v-model="confirmPassword" :disabled="loading" autocomplete="new-password" />
        </div>

        <div v-if="error" class="form-error">{{ error }}</div>
        <div v-if="success" class="form-success">{{ success }}</div>

        <button type="submit" class="auth-button" :disabled="loading">
          {{ loading ? '注册中...' : '注册' }}
        </button>
      </form>

      <div class="auth-footer">
        <p class="auth-switch">
          已有账户？<router-link to="/login" class="auth-link">立即登录</router-link>
        </p>
      </div>
    </div>
  </div>
</template>

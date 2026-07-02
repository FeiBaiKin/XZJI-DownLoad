<script setup lang="ts">
// 登录页：1:1 移植自原项目 Node-XZJI/frontend/src/pages/LoginPage.jsx
// 样式来自全局 styles/auth.css + verification.css
import { ref, onMounted, onUnmounted, watch } from 'vue'
import { useRouter, useRoute } from 'vue-router'
import { useAuthStore } from '@/stores/auth'
import { authApi } from '@/api/auth'
import { contentApi } from '@/api/content'

const router = useRouter()
const route = useRoute()
const auth = useAuthStore()

const loginMode = ref<'password' | 'code'>('password')
const username = ref('')
const password = ref('')
const email = ref('')
const code = ref('')
const loading = ref(false)
const error = ref('')
const sendingCode = ref(false)
const cooldown = ref(0)
const enablePasswordless = ref(false)

const redirect = (route.query.redirect as string) || '/user'

// 获取站点配置（是否开启免密登录）
onMounted(async () => {
  try {
    const cfg = await contentApi.public.siteConfig()
    enablePasswordless.value = !!cfg.enablePasswordless
  } catch { /* 忽略 */ }
})

// 倒计时
let timer: ReturnType<typeof setInterval> | null = null
watch(cooldown, (v) => {
  if (v <= 0) {
    if (timer) { clearInterval(timer); timer = null }
    return
  }
  if (!timer) {
    timer = setInterval(() => {
      cooldown.value = cooldown.value <= 1 ? 0 : cooldown.value - 1
    }, 1000)
  }
})
onUnmounted(() => { if (timer) clearInterval(timer) })

async function handleSendCode() {
  if (!email.value.trim()) { error.value = '请先填写邮箱地址'; return }
  if (!/^[^\s@]+@[^\s@]+\.[^\s@]+$/.test(email.value.trim())) { error.value = '邮箱格式不正确'; return }

  sendingCode.value = true
  error.value = ''
  try {
    await authApi.sendCode(email.value.trim(), 'login')
    cooldown.value = 60
  } catch (err: any) {
    error.value = err?.response?.data?.error || '发送失败'
  } finally {
    sendingCode.value = false
  }
}

async function handlePasswordSubmit() {
  if (!username.value.trim() || !password.value.trim()) { error.value = '请填写用户名和密码'; return }
  loading.value = true
  error.value = ''
  try {
    await auth.login(username.value.trim(), password.value)
    router.push(redirect)
  } catch (err: any) {
    error.value = err?.response?.data?.error || '登录失败'
  } finally {
    loading.value = false
  }
}

async function handleCodeSubmit() {
  if (!email.value.trim() || !code.value.trim()) { error.value = '请填写邮箱和验证码'; return }
  loading.value = true
  error.value = ''
  try {
    const res = await authApi.passwordlessLogin(email.value.trim(), code.value.trim())
    auth.setAuth(res.token, res.user)
    router.push(redirect)
  } catch (err: any) {
    error.value = err?.response?.data?.error || '登录失败'
  } finally {
    loading.value = false
  }
}

function switchMode(mode: 'password' | 'code') {
  loginMode.value = mode
  error.value = ''
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
        <h1 class="auth-title">登录账户</h1>
        <p class="auth-subtitle">登录以使用全部功能</p>
      </div>

      <!-- 登录方式切换 -->
      <div v-if="enablePasswordless" class="login-mode-tabs">
        <button
          type="button"
          class="login-mode-tab"
          :class="{ active: loginMode === 'password' }"
          @click="switchMode('password')"
        >密码登录</button>
        <button
          type="button"
          class="login-mode-tab"
          :class="{ active: loginMode === 'code' }"
          @click="switchMode('code')"
        >验证码登录</button>
      </div>

      <!-- 密码登录 -->
      <form v-if="loginMode === 'password'" class="auth-form" @submit.prevent="handlePasswordSubmit">
        <div class="form-group">
          <label for="username" class="form-label">用户名</label>
          <input
            id="username"
            type="text"
            class="form-input"
            placeholder="输入用户名"
            v-model="username"
            :disabled="loading"
            autocomplete="username"
          />
        </div>

        <div class="form-group">
          <div class="form-label-row">
            <label for="password" class="form-label">密码</label>
            <router-link to="/forgot-password" class="form-label-link">忘记密码？</router-link>
          </div>
          <input
            id="password"
            type="password"
            class="form-input"
            placeholder="输入密码"
            v-model="password"
            :disabled="loading"
            autocomplete="current-password"
          />
        </div>

        <div v-if="error" class="form-error">{{ error }}</div>

        <button type="submit" class="auth-button" :disabled="loading">
          {{ loading ? '登录中...' : '登录' }}
        </button>
      </form>

      <!-- 验证码登录 -->
      <form v-else class="auth-form" @submit.prevent="handleCodeSubmit">
        <div class="form-group">
          <label for="email" class="form-label">邮箱</label>
          <div style="display: flex; gap: 0.5rem;">
            <input
              id="email"
              type="email"
              class="form-input"
              style="flex: 1;"
              placeholder="输入注册邮箱"
              v-model="email"
              :disabled="loading"
              autocomplete="email"
            />
            <button
              type="button"
              class="code-send-btn"
              @click="handleSendCode"
              :disabled="sendingCode || cooldown > 0 || loading"
            >
              {{ sendingCode ? '发送中...' : cooldown > 0 ? `${cooldown}s` : '发送验证码' }}
            </button>
          </div>
        </div>

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

        <div v-if="error" class="form-error">{{ error }}</div>

        <button type="submit" class="auth-button" :disabled="loading">
          {{ loading ? '登录中...' : '登录' }}
        </button>
      </form>

      <div class="auth-footer">
        <p class="auth-switch">
          没有账户？<router-link to="/register" class="auth-link">立即注册</router-link>
        </p>
      </div>
    </div>
  </div>
</template>

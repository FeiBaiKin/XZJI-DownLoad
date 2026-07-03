<script setup lang="ts">
// 管理后台独立登录页：与前台 /login 分离，登录成功直接进入 /admin。
// 仅密码登录（后台入口不提供验证码/注册通道）；非管理员账户登录会被拒绝并清除会话。
// 样式：全局 styles/admin-login.css，沿用站点 Brutalist 设计系统（CSS 变量双主题）。
import { ref, onMounted } from 'vue'
import { useRouter, useRoute } from 'vue-router'
import { useAuthStore } from '@/stores/auth'

const router = useRouter()
const route = useRoute()
const auth = useAuthStore()

const username = ref('')
const password = ref('')
const loading = ref(false)
const error = ref('')
const usernameInput = ref<HTMLInputElement | null>(null)

const redirect = (route.query.redirect as string) || '/admin'

onMounted(() => {
  // 已是管理员会话则直接进入后台
  if (auth.isLoggedIn && auth.isAdmin) {
    router.replace(redirect)
    return
  }
  usernameInput.value?.focus()
})

async function handleSubmit() {
  if (!username.value.trim() || !password.value.trim()) {
    error.value = '请填写账号和密码'
    return
  }
  loading.value = true
  error.value = ''
  try {
    await auth.login(username.value.trim(), password.value)
    if (!auth.isAdmin) {
      // 非管理员走错了门：拒绝进入并清除刚建立的会话
      await auth.logout()
      error.value = '该账户没有管理员权限'
      return
    }
    router.push(redirect)
  } catch (err: any) {
    error.value = err?.response?.data?.error || '登录失败'
  } finally {
    loading.value = false
  }
}
</script>

<template>
  <div class="admin-login-page">
    <main class="admin-login-panel">
      <header class="admin-login-head">
        <div class="admin-login-brand">
          <svg width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.8" stroke-linecap="round" aria-hidden="true">
            <rect x="3" y="11" width="18" height="11" rx="0" />
            <path d="M7 11V7a5 5 0 0110 0v4" />
          </svg>
          <span class="admin-login-brand-text">XZJI 管理控制台</span>
        </div>
        <h1 class="admin-login-title">管理员登录</h1>
        <p class="admin-login-subtitle">使用管理员账户进入后台</p>
      </header>

      <form class="admin-login-form" @submit.prevent="handleSubmit">
        <div class="admin-login-field">
          <label for="admin-username" class="admin-login-label">账号</label>
          <input
            id="admin-username"
            ref="usernameInput"
            type="text"
            class="admin-login-input"
            v-model="username"
            :disabled="loading"
            autocomplete="username"
            spellcheck="false"
          />
        </div>

        <div class="admin-login-field">
          <label for="admin-password" class="admin-login-label">密码</label>
          <input
            id="admin-password"
            type="password"
            class="admin-login-input"
            v-model="password"
            :disabled="loading"
            autocomplete="current-password"
          />
        </div>

        <div v-if="error" class="admin-login-error" role="alert">{{ error }}</div>

        <button type="submit" class="admin-login-submit" :disabled="loading">
          {{ loading ? '验证中...' : '进入后台' }}
        </button>
      </form>

      <footer class="admin-login-footer">
        <router-link to="/" class="admin-login-link">返回前台</router-link>
        <router-link to="/forgot-password" class="admin-login-link">忘记密码？</router-link>
      </footer>
    </main>
  </div>
</template>

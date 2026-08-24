<script setup lang="ts">
// 前台登录/注册弹窗：首页引导与 Header 等入口复用
// 设计：zinc + emerald 居中卡片；Teleport 到 body 后需自带 NConfigProvider 才能跟站点深浅色
import { ref, watch, onUnmounted, computed } from 'vue'
import {
  NButton, NInput, NTabs, NTabPane, NIcon, NConfigProvider, darkTheme, zhCN, dateZhCN,
} from 'naive-ui'
import { PhX, PhArrowLeft } from '@phosphor-icons/vue'
import { useAuthStore } from '@/stores/auth'
import { useAuthModalStore, type AuthModalTab } from '@/stores/authModal'
import { useSiteSettingsStore } from '@/stores/siteSettings'
import { useThemeStore } from '@/stores/theme'
import { lightThemeOverrides, darkThemeOverrides } from '@/theme'
import { authApi } from '@/api/auth'
import { contentApi } from '@/api/content'
import { extractErrorMessage } from '@/types/error'
import { useToastStore } from '@/stores/toast'

const auth = useAuthStore()
const modal = useAuthModalStore()
const site = useSiteSettingsStore()
const theme = useThemeStore()
const toast = useToastStore()

const logoLetter = computed(() => (site.siteName?.[0] || 'X').toUpperCase())
// Teleport 脱离 App 根 NConfigProvider，必须在弹窗内再包一层
const naiveTheme = computed(() => (theme.isDark ? darkTheme : null))
const naiveOverrides = computed(() => (theme.isDark ? darkThemeOverrides : lightThemeOverrides))
const themeAttr = computed(() => (theme.isDark ? 'dark' : 'light'))

// ── 站点开关（缓存，避免每次打开都请求） ──
const enablePasswordless = ref(false)
const forceVerify = ref(false)
let siteConfigLoaded = false

async function ensureSiteConfig() {
  if (siteConfigLoaded) return
  try {
    const cfg = await contentApi.public.siteConfig()
    enablePasswordless.value = !!cfg.enablePasswordless
    forceVerify.value = !!cfg.forceVerify
    siteConfigLoaded = true
  } catch { /* ignore */ }
}

// ── 登录 ──
const loginMode = ref<'password' | 'code'>('password')
const username = ref('')
const password = ref('')
const email = ref('')
const code = ref('')
const loading = ref(false)
const error = ref('')
const success = ref('')
const sendingCode = ref(false)
const cooldown = ref(0)

// ── 注册 ──
const regUsername = ref('')
const regPassword = ref('')
const regConfirm = ref('')
const regEmail = ref('')
const regCode = ref('')

// ── 找回密码 ──
const forgotStep = ref<'request' | 'verify' | 'reset'>('request')
const forgotEmail = ref('')
const forgotCode = ref('')
const forgotPassword = ref('')
const forgotConfirm = ref('')

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
function resetForms() {
  error.value = ''
  success.value = ''
  loginMode.value = 'password'
  username.value = ''
  password.value = ''
  email.value = ''
  code.value = ''
  regUsername.value = ''
  regPassword.value = ''
  regConfirm.value = ''
  regEmail.value = ''
  regCode.value = ''
  forgotStep.value = 'request'
  forgotEmail.value = ''
  forgotCode.value = ''
  forgotPassword.value = ''
  forgotConfirm.value = ''
  if (timer) { clearInterval(timer); timer = null }
  cooldown.value = 0
}

// 打开时再清表单 + 拉配置；关闭不做重计算，避免叉号卡顿
watch(() => modal.open, (v) => {
  if (!v) return
  resetForms()
  void ensureSiteConfig()
})

function switchTab(t: AuthModalTab) {
  modal.tab = t
  error.value = ''
  success.value = ''
}

/** 关闭：只改 open，sessionStorage 延后写，避免主线程卡顿 */
function close() {
  if (!modal.open) return
  modal.hide()
  // 标记本会话不再强弹（空闲时写入）
  const mark = () => {
    try { sessionStorage.setItem('xzji_auth_prompt_dismissed', '1') } catch { /* ignore */ }
  }
  if (typeof requestIdleCallback === 'function') {
    requestIdleCallback(mark, { timeout: 500 })
  } else {
    setTimeout(mark, 0)
  }
}

function onKeydown(e: KeyboardEvent) {
  if (e.key === 'Escape' && modal.open) close()
}

watch(() => modal.open, (v) => {
  if (v) {
    document.addEventListener('keydown', onKeydown)
    document.body.style.overflow = 'hidden'
  } else {
    document.removeEventListener('keydown', onKeydown)
    document.body.style.overflow = ''
  }
})

onUnmounted(() => {
  if (timer) clearInterval(timer)
  document.removeEventListener('keydown', onKeydown)
  document.body.style.overflow = ''
})

async function handleSendCode(purpose: 'login' | 'register' | 'forgot-password', targetEmail: string) {
  if (!targetEmail.trim()) { error.value = '请先填写邮箱地址'; return }
  if (!/^[^\s@]+@[^\s@]+\.[^\s@]+$/.test(targetEmail.trim())) { error.value = '邮箱格式不正确'; return }
  sendingCode.value = true
  error.value = ''
  try {
    await authApi.sendCode(targetEmail.trim(), purpose)
    cooldown.value = 60
    if (purpose === 'forgot-password') {
      success.value = '验证码已发送，请查收邮箱'
      forgotStep.value = 'verify'
    } else if (purpose === 'register') {
      success.value = '验证码已发送，请查收邮箱'
    }
  } catch (err) {
    error.value = extractErrorMessage(err, '发送失败')
  } finally {
    sendingCode.value = false
  }
}

async function handlePasswordLogin() {
  if (!username.value.trim() || !password.value.trim()) {
    error.value = '请填写用户名和密码'
    return
  }
  loading.value = true
  error.value = ''
  try {
    await auth.login(username.value.trim(), password.value)
    // auth.login 内已 fetchProfile 以种 CSRF
    toast.success('登录成功')
    modal.hide()
  } catch (err) {
    error.value = extractErrorMessage(err, '登录失败')
  } finally {
    loading.value = false
  }
}

async function handleCodeLogin() {
  if (!email.value.trim() || !code.value.trim()) {
    error.value = '请填写邮箱和验证码'
    return
  }
  loading.value = true
  error.value = ''
  try {
    const res = await authApi.passwordlessLogin(email.value.trim(), code.value.trim())
    auth.setAuth(res.token, res.user)
    try { await auth.fetchProfile() } catch { /* CSRF / 会话补全 */ }
    toast.success('登录成功')
    modal.hide()
  } catch (err) {
    error.value = extractErrorMessage(err, '登录失败')
  } finally {
    loading.value = false
  }
}

async function handleRegister() {
  error.value = ''
  success.value = ''
  if (!regUsername.value.trim() || !regPassword.value) {
    error.value = '请填写用户名和密码'
    return
  }
  if (regUsername.value.trim().length < 3) {
    error.value = '用户名至少需要3个字符'
    return
  }
  if (!/^[A-Za-z0-9_]+$/.test(regUsername.value.trim())) {
    error.value = '用户名仅支持英文字母、数字和下划线'
    return
  }
  if (regPassword.value.length < 6) {
    error.value = '密码至少需要6个字符'
    return
  }
  if (regPassword.value !== regConfirm.value) {
    error.value = '两次输入的密码不一致'
    return
  }
  if (forceVerify.value) {
    if (!regEmail.value.trim()) { error.value = '请填写邮箱地址'; return }
    if (!regCode.value.trim()) { error.value = '请输入邮箱验证码'; return }
  }
  loading.value = true
  try {
    const body: { username: string; password: string; email?: string; code?: string } = {
      username: regUsername.value.trim(),
      password: regPassword.value,
      email: regEmail.value.trim() || undefined,
    }
    if (forceVerify.value && regCode.value.trim()) body.code = regCode.value.trim()
    await auth.register(body)
    toast.success('注册成功')
    modal.hide()
  } catch (err) {
    error.value = extractErrorMessage(err, '注册失败')
  } finally {
    loading.value = false
  }
}

async function handleForgotRequest() {
  error.value = ''
  success.value = ''
  await handleSendCode('forgot-password', forgotEmail.value)
}

async function handleForgotVerify() {
  error.value = ''
  success.value = ''
  if (!forgotCode.value.trim()) { error.value = '请输入验证码'; return }
  if (forgotCode.value.trim().length !== 6) { error.value = '验证码为6位数字'; return }
  loading.value = true
  try {
    await authApi.verifyCode(forgotEmail.value.trim(), forgotCode.value.trim(), 'forgot-password')
    forgotStep.value = 'reset'
  } catch (err) {
    error.value = extractErrorMessage(err, '验证失败')
  } finally {
    loading.value = false
  }
}

async function handleForgotReset() {
  error.value = ''
  success.value = ''
  if (!forgotPassword.value) { error.value = '请输入新密码'; return }
  if (forgotPassword.value.length < 6) { error.value = '密码至少需要6个字符'; return }
  if (forgotPassword.value !== forgotConfirm.value) {
    error.value = '两次输入的密码不一致'
    return
  }
  loading.value = true
  try {
    await authApi.resetPassword(forgotEmail.value.trim(), forgotCode.value.trim(), forgotPassword.value)
    success.value = '密码已重置，请登录'
    toast.success('密码重置成功')
    switchTab('login')
    forgotStep.value = 'request'
  } catch (err) {
    error.value = extractErrorMessage(err, '重置失败')
  } finally {
    loading.value = false
  }
}

const title = computed(() => {
  if (modal.tab === 'register') return '注册'
  if (modal.tab === 'forgot') {
    if (forgotStep.value === 'reset') return '设置新密码'
    return '找回密码'
  }
  return '登录'
})

const subtitle = computed(() => {
  if (modal.tab === 'register') return '创建账户后即可搜索、收藏与购买资源'
  if (modal.tab === 'forgot') {
    if (forgotStep.value === 'request') return '输入注册邮箱，获取验证码'
    if (forgotStep.value === 'verify') return `验证码已发送至 ${forgotEmail.value}`
    return '请输入您的新密码'
  }
  return '登录后继续使用完整功能'
})
</script>

<template>
  <!-- 轻量 Teleport：脱离 App 根树，需自带 NConfigProvider + data-theme 才能深浅色一致 -->
  <Teleport to="body">
    <NConfigProvider
      v-if="modal.open"
      :theme="naiveTheme"
      :theme-overrides="naiveOverrides"
      :locale="zhCN"
      :date-locale="dateZhCN"
    >
      <div
        class="auth-overlay fixed inset-0 z-[2000] flex items-center justify-center p-4"
        role="presentation"
        :data-theme="themeAttr"
      >
        <div
          class="absolute inset-0 bg-zinc-900/40 dark:bg-black/65"
          aria-hidden="true"
          @click="close"
        />
        <div
          class="auth-modal-panel relative z-10 w-[min(100vw-2rem,420px)] max-h-[min(90dvh,720px)] overflow-y-auto overscroll-contain rounded-2xl border border-zinc-200/90 bg-white text-zinc-900 shadow-[0_1px_2px_rgb(24_24_27/0.06),0_16px_48px_-12px_rgb(24_24_27/0.2)] dark:border-zinc-800 dark:bg-zinc-900 dark:text-zinc-100 dark:shadow-[0_1px_2px_rgb(0_0_0/0.4),0_20px_56px_-16px_rgb(0_0_0/0.65)]"
          role="dialog"
          aria-modal="true"
          :aria-label="title"
        >
      <div class="relative px-6 pb-6 pt-5 sm:px-7 sm:pb-7 sm:pt-6">
        <!-- 顶栏 -->
        <div class="mb-5 flex items-start justify-between gap-3">
          <div class="flex min-w-0 items-center gap-2.5">
            <img
              v-if="site.siteLogoType === 'image' && site.siteLogoUrl"
              :src="site.siteLogoUrl"
              :alt="site.siteName"
              class="h-8 w-auto"
            />
            <span
              v-else
              class="grid size-8 shrink-0 place-items-center rounded-lg bg-emerald-600 text-sm font-bold text-white dark:bg-emerald-400 dark:text-zinc-950"
            >{{ logoLetter }}</span>
            <div class="min-w-0">
              <p class="truncate text-sm font-semibold tracking-tight text-zinc-800 dark:text-zinc-100">
                {{ site.siteName || 'XZJI' }}
              </p>
              <p class="text-[11px] text-zinc-500 dark:text-zinc-400">欢迎回来</p>
            </div>
          </div>
          <button
            type="button"
            class="grid size-8 shrink-0 place-items-center rounded-lg text-zinc-400 transition-colors hover:bg-zinc-100 hover:text-zinc-800 focus-visible:outline-2 focus-visible:outline-offset-2 focus-visible:outline-emerald-500 dark:text-zinc-500 dark:hover:bg-zinc-800 dark:hover:text-zinc-100"
            aria-label="关闭"
            @click="close"
          >
            <NIcon :size="18"><PhX /></NIcon>
          </button>
        </div>

        <h2 class="text-xl font-bold tracking-tight text-zinc-900 dark:text-zinc-50">{{ title }}</h2>
        <p class="mt-1 text-sm leading-relaxed text-zinc-500 dark:text-zinc-400">{{ subtitle }}</p>

        <!-- 登录 / 注册切换（找回密码时隐藏） -->
        <div v-if="modal.tab !== 'forgot'" class="mt-5">
          <div class="flex rounded-xl border border-zinc-200 bg-zinc-50 p-1 dark:border-zinc-700 dark:bg-zinc-950/60">
            <button
              type="button"
              class="flex-1 rounded-lg px-3 py-2 text-sm font-medium transition-colors focus-visible:outline-2 focus-visible:outline-offset-2 focus-visible:outline-emerald-500"
              :class="modal.tab === 'login'
                ? 'bg-white text-zinc-900 shadow-sm dark:bg-zinc-800 dark:text-zinc-50 dark:shadow-none'
                : 'text-zinc-500 hover:text-zinc-800 dark:text-zinc-400 dark:hover:text-zinc-200'"
              @click="switchTab('login')"
            >
              登录
            </button>
            <button
              type="button"
              class="flex-1 rounded-lg px-3 py-2 text-sm font-medium transition-colors focus-visible:outline-2 focus-visible:outline-offset-2 focus-visible:outline-emerald-500"
              :class="modal.tab === 'register'
                ? 'bg-white text-zinc-900 shadow-sm dark:bg-zinc-800 dark:text-zinc-50 dark:shadow-none'
                : 'text-zinc-500 hover:text-zinc-800 dark:text-zinc-400 dark:hover:text-zinc-200'"
              @click="switchTab('register')"
            >
              注册
            </button>
          </div>
        </div>
        <button
          v-else
          type="button"
          class="mt-4 inline-flex items-center gap-1 text-sm text-zinc-500 transition-colors hover:text-zinc-800 focus-visible:outline-2 focus-visible:outline-offset-2 focus-visible:outline-emerald-500 dark:text-zinc-400 dark:hover:text-zinc-200"
          @click="switchTab('login')"
        >
          <NIcon :size="15"><PhArrowLeft /></NIcon>
          返回登录
        </button>

        <div class="mt-5 max-h-[min(60dvh,480px)] overflow-y-auto overscroll-contain pr-0.5">
          <!-- ════ 登录 ════ -->
          <template v-if="modal.tab === 'login'">
            <NTabs
              v-if="enablePasswordless"
              :value="loginMode"
              type="segment"
              animated
              class="mb-4"
              @update:value="(v: string | number) => { loginMode = v as 'password' | 'code'; error = '' }"
            >
              <NTabPane name="password" tab="密码登录" />
              <NTabPane name="code" tab="验证码登录" />
            </NTabs>

            <form v-if="loginMode === 'password'" class="space-y-4" @submit.prevent="handlePasswordLogin">
              <div>
                <label class="mb-1.5 block text-sm font-medium text-zinc-700 dark:text-zinc-300">用户名</label>
                <NInput
                  v-model:value="username"
                  size="large"
                  placeholder="输入用户名"
                  :disabled="loading"
                  :input-props="{ autocomplete: 'username' }"
                />
              </div>
              <div>
                <div class="mb-1.5 flex items-center justify-between">
                  <label class="text-sm font-medium text-zinc-700 dark:text-zinc-300">密码</label>
                  <button
                    type="button"
                    class="text-xs text-emerald-600 hover:underline focus-visible:underline focus-visible:outline-none dark:text-emerald-400"
                    @click="switchTab('forgot')"
                  >
                    忘记密码？
                  </button>
                </div>
                <NInput
                  v-model:value="password"
                  type="password"
                  show-password-on="click"
                  size="large"
                  placeholder="输入密码"
                  :disabled="loading"
                  :input-props="{ autocomplete: 'current-password' }"
                />
              </div>
              <p v-if="error" class="rounded-lg bg-red-50 px-3 py-2 text-sm text-red-600 dark:bg-red-500/10 dark:text-red-400">{{ error }}</p>
              <NButton attr-type="submit" type="primary" size="large" block :loading="loading">登录</NButton>
            </form>

            <form v-else class="space-y-4" @submit.prevent="handleCodeLogin">
              <div>
                <label class="mb-1.5 block text-sm font-medium text-zinc-700 dark:text-zinc-300">邮箱</label>
                <div class="flex flex-col gap-2 sm:flex-row">
                  <NInput
                    v-model:value="email"
                    size="large"
                    class="flex-1"
                    placeholder="注册邮箱"
                    :disabled="loading"
                    :input-props="{ type: 'email', autocomplete: 'email' }"
                  />
                  <NButton
                    size="large"
                    secondary
                    class="sm:shrink-0"
                    :disabled="sendingCode || cooldown > 0 || loading"
                    :loading="sendingCode"
                    @click="handleSendCode('login', email)"
                  >
                    {{ cooldown > 0 ? `${cooldown}s` : '发送验证码' }}
                  </NButton>
                </div>
              </div>
              <div>
                <label class="mb-1.5 block text-sm font-medium text-zinc-700 dark:text-zinc-300">验证码</label>
                <NInput
                  v-model:value="code"
                  size="large"
                  placeholder="6 位数字验证码"
                  maxlength="6"
                  :disabled="loading"
                  class="text-center font-mono tracking-[0.3em]"
                  :input-props="{ autocomplete: 'one-time-code', inputmode: 'numeric' }"
                />
              </div>
              <p v-if="error" class="rounded-lg bg-red-50 px-3 py-2 text-sm text-red-600 dark:bg-red-500/10 dark:text-red-400">{{ error }}</p>
              <NButton attr-type="submit" type="primary" size="large" block :loading="loading">登录</NButton>
            </form>
          </template>

          <!-- ════ 注册 ════ -->
          <form v-else-if="modal.tab === 'register'" class="space-y-4" @submit.prevent="handleRegister">
            <div>
              <label class="mb-1.5 block text-sm font-medium text-zinc-700 dark:text-zinc-300">用户名</label>
              <NInput
                v-model:value="regUsername"
                size="large"
                placeholder="至少 3 个字符，英文/数字/下划线"
                :disabled="loading"
                :input-props="{ autocomplete: 'username' }"
              />
            </div>
            <div>
              <label class="mb-1.5 block text-sm font-medium text-zinc-700 dark:text-zinc-300">
                邮箱
                <span v-if="forceVerify" class="text-red-500">*</span>
                <span v-else class="ml-1 text-xs font-normal text-zinc-500">选填</span>
              </label>
              <div class="flex flex-col gap-2 sm:flex-row">
                <NInput
                  v-model:value="regEmail"
                  size="large"
                  class="flex-1"
                  :placeholder="forceVerify ? '必填，用于接收验证码' : '用于找回密码'"
                  :disabled="loading"
                  :input-props="{ type: 'email', autocomplete: 'email' }"
                />
                <NButton
                  v-if="forceVerify"
                  size="large"
                  secondary
                  class="sm:shrink-0"
                  :disabled="sendingCode || cooldown > 0 || loading"
                  :loading="sendingCode"
                  @click="handleSendCode('register', regEmail)"
                >
                  {{ cooldown > 0 ? `${cooldown}s` : '发送验证码' }}
                </NButton>
              </div>
            </div>
            <div v-if="forceVerify">
              <label class="mb-1.5 block text-sm font-medium text-zinc-700 dark:text-zinc-300">验证码</label>
              <NInput
                v-model:value="regCode"
                size="large"
                placeholder="6 位数字验证码"
                maxlength="6"
                :disabled="loading"
                class="text-center font-mono tracking-[0.3em]"
                :input-props="{ autocomplete: 'one-time-code', inputmode: 'numeric' }"
              />
            </div>
            <div>
              <label class="mb-1.5 block text-sm font-medium text-zinc-700 dark:text-zinc-300">密码</label>
              <NInput
                v-model:value="regPassword"
                type="password"
                show-password-on="click"
                size="large"
                placeholder="至少 6 个字符"
                :disabled="loading"
                :input-props="{ autocomplete: 'new-password' }"
              />
            </div>
            <div>
              <label class="mb-1.5 block text-sm font-medium text-zinc-700 dark:text-zinc-300">确认密码</label>
              <NInput
                v-model:value="regConfirm"
                type="password"
                show-password-on="click"
                size="large"
                placeholder="再次输入密码"
                :disabled="loading"
                :input-props="{ autocomplete: 'new-password' }"
              />
            </div>
            <p v-if="error" class="rounded-lg bg-red-50 px-3 py-2 text-sm text-red-600 dark:bg-red-500/10 dark:text-red-400">{{ error }}</p>
            <p v-if="success" class="rounded-lg bg-emerald-50 px-3 py-2 text-sm text-emerald-700 dark:bg-emerald-400/10 dark:text-emerald-400">{{ success }}</p>
            <NButton attr-type="submit" type="primary" size="large" block :loading="loading">注册</NButton>
          </form>

          <!-- ════ 找回密码 ════ -->
          <template v-else>
            <form v-if="forgotStep === 'request'" class="space-y-4" @submit.prevent="handleForgotRequest">
              <div>
                <label class="mb-1.5 block text-sm font-medium text-zinc-700 dark:text-zinc-300">注册邮箱</label>
                <NInput
                  v-model:value="forgotEmail"
                  size="large"
                  placeholder="输入注册邮箱"
                  :disabled="sendingCode"
                  :input-props="{ type: 'email', autocomplete: 'email' }"
                />
              </div>
              <p v-if="error" class="rounded-lg bg-red-50 px-3 py-2 text-sm text-red-600 dark:bg-red-500/10 dark:text-red-400">{{ error }}</p>
              <NButton attr-type="submit" type="primary" size="large" block :loading="sendingCode">发送验证码</NButton>
            </form>

            <form v-else-if="forgotStep === 'verify'" class="space-y-4" @submit.prevent="handleForgotVerify">
              <div>
                <label class="mb-1.5 block text-sm font-medium text-zinc-700 dark:text-zinc-300">验证码</label>
                <div class="flex flex-col gap-2 sm:flex-row">
                  <NInput
                    v-model:value="forgotCode"
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
                    class="sm:shrink-0"
                    :disabled="sendingCode || cooldown > 0 || loading"
                    :loading="sendingCode"
                    @click="handleSendCode('forgot-password', forgotEmail)"
                  >
                    {{ cooldown > 0 ? `${cooldown}s` : '重新发送' }}
                  </NButton>
                </div>
              </div>
              <p v-if="error" class="rounded-lg bg-red-50 px-3 py-2 text-sm text-red-600 dark:bg-red-500/10 dark:text-red-400">{{ error }}</p>
              <NButton attr-type="submit" type="primary" size="large" block :loading="loading">验证</NButton>
            </form>

            <form v-else class="space-y-4" @submit.prevent="handleForgotReset">
              <div>
                <label class="mb-1.5 block text-sm font-medium text-zinc-700 dark:text-zinc-300">新密码</label>
                <NInput
                  v-model:value="forgotPassword"
                  type="password"
                  show-password-on="click"
                  size="large"
                  placeholder="至少 6 个字符"
                  :disabled="loading"
                  :input-props="{ autocomplete: 'new-password' }"
                />
              </div>
              <div>
                <label class="mb-1.5 block text-sm font-medium text-zinc-700 dark:text-zinc-300">确认新密码</label>
                <NInput
                  v-model:value="forgotConfirm"
                  type="password"
                  show-password-on="click"
                  size="large"
                  placeholder="再次输入新密码"
                  :disabled="loading"
                  :input-props="{ autocomplete: 'new-password' }"
                />
              </div>
              <p v-if="error" class="rounded-lg bg-red-50 px-3 py-2 text-sm text-red-600 dark:bg-red-500/10 dark:text-red-400">{{ error }}</p>
              <NButton attr-type="submit" type="primary" size="large" block :loading="loading">重置密码</NButton>
            </form>
          </template>
        </div>

        <p class="mt-5 text-center text-[11px] text-zinc-400 dark:text-zinc-500">
          可关闭弹窗继续浏览，需要时再登录
        </p>
      </div>
        </div>
      </div>
    </NConfigProvider>
  </Teleport>
</template>

<style scoped>
/* 仅入场 opacity；关闭时 v-if 直接卸载 */
.auth-modal-panel {
  animation: auth-modal-in 0.2s ease-out both;
  color-scheme: light;
}

.auth-overlay[data-theme='dark'] .auth-modal-panel {
  color-scheme: dark;
}

@keyframes auth-modal-in {
  from { opacity: 0; }
  to { opacity: 1; }
}

@media (prefers-reduced-motion: reduce) {
  .auth-modal-panel {
    animation: none;
  }
}
</style>

<script setup lang="ts">
// 登录页：密码登录 + 邮箱验证码免密登录（受后台开关控制）
// 业务逻辑保持不变；UI 重构为 AuthShell + Naive UI
import { ref, onMounted, onUnmounted, watch } from 'vue'
import { useRouter, useRoute } from 'vue-router'
import { NButton, NInput, NTabs, NTabPane } from 'naive-ui'
import { useAuthStore } from '@/stores/auth'
import { authApi } from '@/api/auth'
import { contentApi } from '@/api/content'
import AuthShell from '@/components/AuthShell.vue'
import { safeRedirectPath } from '@/utils/safeRedirect'

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

const redirect = safeRedirectPath(route.query.redirect, '/user')

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
  } catch (err: unknown) {
    error.value = (err as { response?: { data?: { error?: string } } })?.response?.data?.error || '发送失败'
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
  } catch (err: unknown) {
    error.value = (err as { response?: { data?: { error?: string } } })?.response?.data?.error || '登录失败'
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
  } catch (err: unknown) {
    error.value = (err as { response?: { data?: { error?: string } } })?.response?.data?.error || '登录失败'
  } finally {
    loading.value = false
  }
}

function switchMode(mode: string | number) {
  loginMode.value = mode as 'password' | 'code'
  error.value = ''
}
</script>

<template>
  <AuthShell title="登录" subtitle="使用账户继续访问资源与会员权益">
    <!-- 登录方式切换 -->
    <NTabs
      v-if="enablePasswordless"
      :value="loginMode"
      type="segment"
      animated
      class="mb-6"
      @update:value="switchMode"
    >
      <NTabPane name="password" tab="密码登录" />
      <NTabPane name="code" tab="验证码登录" />
    </NTabs>

    <!-- 密码登录 -->
    <form v-if="loginMode === 'password'" class="space-y-5" @submit.prevent="handlePasswordSubmit">
      <div>
        <label for="username" class="mb-1.5 block text-sm font-medium text-zinc-700 dark:text-zinc-300">用户名</label>
        <NInput
          id="username"
          v-model:value="username"
          size="large"
          placeholder="输入用户名"
          :disabled="loading"
          :input-props="{ autocomplete: 'username' }"
        />
      </div>

      <div>
        <div class="mb-1.5 flex items-center justify-between">
          <label for="password" class="text-sm font-medium text-zinc-700 dark:text-zinc-300">密码</label>
          <router-link to="/forgot-password" class="text-xs text-emerald-600 hover:underline focus-visible:underline focus-visible:outline-none dark:text-emerald-400">忘记密码？</router-link>
        </div>
        <NInput
          id="password"
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

    <!-- 验证码登录 -->
    <form v-else class="space-y-5" @submit.prevent="handleCodeSubmit">
      <div>
        <label for="email" class="mb-1.5 block text-sm font-medium text-zinc-700 dark:text-zinc-300">邮箱</label>
        <div class="flex flex-col gap-2 sm:flex-row">
          <NInput
            id="email"
            v-model:value="email"
            size="large"
            class="flex-1"
            placeholder="输入注册邮箱"
            :disabled="loading"
            :input-props="{ type: 'email', autocomplete: 'email' }"
          />
          <NButton
            size="large"
            secondary
            class="sm:shrink-0"
            :disabled="sendingCode || cooldown > 0 || loading"
            :loading="sendingCode"
            @click="handleSendCode"
          >
            {{ cooldown > 0 ? `${cooldown}s` : '发送验证码' }}
          </NButton>
        </div>
      </div>

      <div>
        <label for="code" class="mb-1.5 block text-sm font-medium text-zinc-700 dark:text-zinc-300">验证码</label>
        <NInput
          id="code"
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

    <template #footer>
      没有账户？<router-link to="/register" class="font-medium text-emerald-600 hover:underline focus-visible:underline focus-visible:outline-none dark:text-emerald-400">立即注册</router-link>
    </template>
  </AuthShell>
</template>

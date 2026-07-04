<script setup lang="ts">
// 注册页：用户名/密码 + 邮箱（后台开启强制验证时需验证码）
// 业务逻辑保持不变；UI 重构为 AuthShell + Naive UI
import { ref, onMounted, onUnmounted, watch } from 'vue'
import { useRouter } from 'vue-router'
import { NButton, NInput } from 'naive-ui'
import { useAuthStore } from '@/stores/auth'
import { authApi } from '@/api/auth'
import { contentApi } from '@/api/content'
import AuthShell from '@/components/AuthShell.vue'

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
  } catch (err: unknown) {
    error.value = (err as { response?: { data?: { error?: string } } })?.response?.data?.error || '发送失败'
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
  } catch (err: unknown) {
    error.value = (err as { response?: { data?: { error?: string } } })?.response?.data?.error || '注册失败'
  } finally {
    loading.value = false
  }
}
</script>

<template>
  <AuthShell title="创建账户" subtitle="注册一个新账户以使用全部功能">
    <form class="space-y-5" @submit.prevent="handleSubmit">
      <div>
        <label for="username" class="mb-1.5 block text-sm font-medium text-zinc-700 dark:text-zinc-300">用户名</label>
        <NInput
          id="username"
          v-model:value="username"
          size="large"
          placeholder="至少 3 个字符"
          :disabled="loading"
          :input-props="{ autocomplete: 'username' }"
        />
      </div>

      <div>
        <label for="email" class="mb-1.5 block text-sm font-medium text-zinc-700 dark:text-zinc-300">
          邮箱
          <span v-if="forceVerify" class="text-red-500">*</span>
          <span v-else class="ml-1 text-xs font-normal text-zinc-400 dark:text-zinc-500">选填</span>
        </label>
        <div class="flex gap-2">
          <NInput
            id="email"
            v-model:value="email"
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
            :disabled="sendingCode || cooldown > 0 || loading"
            :loading="sendingCode"
            @click="handleSendCode"
          >
            {{ cooldown > 0 ? `${cooldown}s` : '发送验证码' }}
          </NButton>
        </div>
      </div>

      <div v-if="forceVerify">
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

      <div>
        <label for="password" class="mb-1.5 block text-sm font-medium text-zinc-700 dark:text-zinc-300">密码</label>
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
        <label for="confirmPassword" class="mb-1.5 block text-sm font-medium text-zinc-700 dark:text-zinc-300">确认密码</label>
        <NInput
          id="confirmPassword"
          v-model:value="confirmPassword"
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

    <template #footer>
      已有账户？<router-link to="/login" class="font-medium text-emerald-600 hover:underline dark:text-emerald-400">立即登录</router-link>
    </template>
  </AuthShell>
</template>

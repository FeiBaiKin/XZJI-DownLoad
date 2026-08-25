<script setup lang="ts">
// 管理后台独立登录页：与前台 /login 分离，登录成功直接进入 /admin。
// 仅密码登录（后台入口不提供验证码/注册通道）；非管理员账户登录会被拒绝并清除会话。
import { ref, onMounted } from 'vue'
import { useRouter, useRoute } from 'vue-router'
import { NButton, NInput } from 'naive-ui'
import { KeyRound } from '@lucide/vue'
import AppIcon from '../../components/AppIcon.vue'
import { useAuthStore } from '@/stores/auth'
import { safeRedirectPath } from '@/utils/safeRedirect'

const router = useRouter()
const route = useRoute()
const auth = useAuthStore()

const username = ref('')
const password = ref('')
const loading = ref(false)
const error = ref('')
const usernameInput = ref<InstanceType<typeof NInput> | null>(null)

const redirect = safeRedirectPath(route.query.redirect, '/admin')

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
  } catch (err: unknown) {
    error.value = (err as { response?: { data?: { error?: string } } })?.response?.data?.error || '登录失败'
  } finally {
    loading.value = false
  }
}
</script>

<template>
  <div class="relative flex min-h-dvh items-center justify-center overflow-hidden px-6 py-12">
    <!-- 背景光晕：低调的品牌色渲染 -->
    <div aria-hidden="true" class="pointer-events-none absolute -top-48 left-1/2 size-[560px] -translate-x-1/2 rounded-full bg-emerald-500/10 blur-3xl dark:bg-emerald-400/8"></div>

    <main class="relative w-full max-w-sm">
      <div class="rounded-2xl border border-zinc-200 bg-white p-8 shadow-sm dark:border-zinc-800 dark:bg-zinc-900">
        <div class="flex items-center gap-2 text-zinc-400 dark:text-zinc-500">
          <AppIcon :icon="KeyRound" scale="sm" />
          <span class="text-xs font-medium tracking-wide">管理控制台</span>
        </div>
        <h1 class="mt-3 text-2xl font-bold tracking-tight text-zinc-900 dark:text-zinc-50">管理员登录</h1>
        <p class="mt-1 text-sm text-zinc-500 dark:text-zinc-400">使用管理员账户进入后台</p>

        <form class="mt-7 space-y-4" @submit.prevent="handleSubmit">
          <div>
            <label for="admin-username" class="mb-1.5 block text-sm font-medium text-zinc-700 dark:text-zinc-300">账号</label>
            <NInput
              id="admin-username"
              ref="usernameInput"
              v-model:value="username"
              size="large"
              placeholder="管理员账号"
              :disabled="loading"
              :input-props="{ autocomplete: 'username', spellcheck: false }"
            />
          </div>

          <div>
            <label for="admin-password" class="mb-1.5 block text-sm font-medium text-zinc-700 dark:text-zinc-300">密码</label>
            <NInput
              id="admin-password"
              v-model:value="password"
              type="password"
              show-password-on="click"
              size="large"
              placeholder="密码"
              :disabled="loading"
              :input-props="{ autocomplete: 'current-password' }"
            />
          </div>

          <p v-if="error" role="alert" class="rounded-lg bg-red-50 px-3 py-2 text-sm text-red-600 dark:bg-red-500/10 dark:text-red-400">
            {{ error }}
          </p>

          <NButton attr-type="submit" type="primary" size="large" block :loading="loading">
            {{ loading ? '验证中...' : '进入后台' }}
          </NButton>
        </form>
      </div>

      <div class="mt-5 flex items-center justify-center gap-5 text-sm">
        <router-link to="/" class="rounded text-zinc-500 transition-colors hover:text-zinc-900 focus-visible:underline focus-visible:outline-none dark:text-zinc-400 dark:hover:text-zinc-100">返回前台</router-link>
        <span class="text-zinc-300 dark:text-zinc-700">/</span>
        <router-link to="/forgot-password" class="rounded text-zinc-500 transition-colors hover:text-zinc-900 focus-visible:underline focus-visible:outline-none dark:text-zinc-400 dark:hover:text-zinc-100">忘记密码？</router-link>
      </div>
    </main>
  </div>
</template>

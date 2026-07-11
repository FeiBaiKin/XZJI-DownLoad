<script setup lang="ts">
// Web 安装向导：站点信息 / 管理员账号 / 数据库配置 → POST /install
// 业务逻辑保持不变；UI 重构为 Naive UI + Tailwind
import { ref } from 'vue'
import { NButton, NInput, NInputNumber, NAlert } from 'naive-ui'
import api from '@/api'

const step = ref<'form' | 'loading' | 'error'>('form')
const errorMsg = ref('')
const form = ref({
  siteName: '', siteDescription: '', siteURL: '', footerText: '',
  adminUsername: '', adminPassword: '', adminEmail: '',
  dbHost: '127.0.0.1', dbPort: 5432, dbUser: 'postgres', dbPassword: '', dbName: 'xzji', dbSSLMode: 'disable', dbTimeZone: 'Asia/Shanghai',
})

async function submit() {
  step.value = 'loading'
  try {
    const s = await api.get('/install/status')
    if (s.data.installed) { errorMsg.value = '系统已安装'; step.value = 'error'; return }
    await api.post('/install', form.value)
    step.value = 'error'; errorMsg.value = '安装成功！请重启服务后访问首页。'
  } catch (e: unknown) {
    errorMsg.value = (e as { response?: { data?: { error?: string } } })?.response?.data?.error || '安装失败'
    step.value = 'error'
  }
}
</script>

<template>
  <div class="min-h-dvh py-12">
    <div class="mx-auto max-w-xl px-4 sm:px-6">
      <div class="flex items-center gap-3">
        <span class="grid size-10 place-items-center rounded-xl bg-emerald-600 font-bold text-white dark:bg-emerald-400 dark:text-zinc-950">X</span>
        <div>
          <h1 class="text-2xl font-bold tracking-tight text-zinc-900 dark:text-zinc-50">系统安装</h1>
          <p class="text-sm text-zinc-500 dark:text-zinc-400">配置站点信息与数据库连接</p>
        </div>
      </div>

      <form class="mt-8 space-y-5" @submit.prevent="submit">
        <!-- 网站信息 -->
        <section class="rounded-2xl border border-zinc-200 bg-white p-6 dark:border-zinc-800 dark:bg-zinc-900">
          <h2 class="mb-4 text-sm font-semibold text-zinc-900 dark:text-zinc-100">网站信息</h2>
          <div class="space-y-4">
            <div>
              <label class="mb-1.5 block text-sm text-zinc-600 dark:text-zinc-400">网站名称</label>
              <NInput v-model:value="form.siteName" placeholder="如：关键词搜索平台" />
            </div>
            <div>
              <label class="mb-1.5 block text-sm text-zinc-600 dark:text-zinc-400">站点地址 <span class="text-red-500">*</span></label>
              <NInput v-model:value="form.siteURL" placeholder="https://your-domain.com" />
              <p class="mt-1.5 text-xs text-zinc-500 dark:text-zinc-400">用于 CORS 安全策略，必须以 http:// 或 https:// 开头</p>
            </div>
            <div>
              <label class="mb-1.5 block text-sm text-zinc-600 dark:text-zinc-400">网站描述</label>
              <NInput v-model:value="form.siteDescription" placeholder="输入关键词，发现后端配置的响应内容" />
            </div>
            <div>
              <label class="mb-1.5 block text-sm text-zinc-600 dark:text-zinc-400">页脚文本</label>
              <NInput v-model:value="form.footerText" placeholder="Keyword Search App" />
            </div>
          </div>
        </section>

        <!-- 管理员账号 -->
        <section class="rounded-2xl border border-zinc-200 bg-white p-6 dark:border-zinc-800 dark:bg-zinc-900">
          <h2 class="mb-4 text-sm font-semibold text-zinc-900 dark:text-zinc-100">管理员账号</h2>
          <div class="space-y-4">
            <div>
              <label class="mb-1.5 block text-sm text-zinc-600 dark:text-zinc-400">用户名</label>
              <NInput v-model:value="form.adminUsername" placeholder="至少 3 个字符" :input-props="{ autocomplete: 'username' }" />
            </div>
            <div>
              <label class="mb-1.5 block text-sm text-zinc-600 dark:text-zinc-400">密码</label>
              <NInput v-model:value="form.adminPassword" type="password" show-password-on="click" placeholder="至少 6 个字符" :input-props="{ autocomplete: 'new-password' }" />
            </div>
            <div>
              <label class="mb-1.5 block text-sm text-zinc-600 dark:text-zinc-400">邮箱</label>
              <NInput v-model:value="form.adminEmail" placeholder="选填" :input-props="{ type: 'email' }" />
            </div>
          </div>
        </section>

        <!-- 数据库配置 -->
        <section class="rounded-2xl border border-zinc-200 bg-white p-6 dark:border-zinc-800 dark:bg-zinc-900">
          <h2 class="mb-4 text-sm font-semibold text-zinc-900 dark:text-zinc-100">数据库配置</h2>
          <div class="space-y-4">
            <div class="grid grid-cols-[1fr_140px] gap-3">
              <div>
                <label class="mb-1.5 block text-sm text-zinc-600 dark:text-zinc-400">主机</label>
                <NInput v-model:value="form.dbHost" />
              </div>
              <div>
                <label class="mb-1.5 block text-sm text-zinc-600 dark:text-zinc-400">端口</label>
                <NInputNumber
                  :value="form.dbPort"
                  :show-button="false"
                  class="w-full"
                  @update:value="(v: number | null) => form.dbPort = v ?? 5432"
                />
              </div>
            </div>
            <div>
              <label class="mb-1.5 block text-sm text-zinc-600 dark:text-zinc-400">用户名</label>
              <NInput v-model:value="form.dbUser" />
            </div>
            <div>
              <label class="mb-1.5 block text-sm text-zinc-600 dark:text-zinc-400">密码</label>
              <NInput v-model:value="form.dbPassword" type="password" show-password-on="click" />
            </div>
            <div>
              <label class="mb-1.5 block text-sm text-zinc-600 dark:text-zinc-400">数据库名</label>
              <NInput v-model:value="form.dbName" />
            </div>
            <div>
              <label class="mb-1.5 block text-sm text-zinc-600 dark:text-zinc-400">SSL 模式</label>
              <NInput v-model:value="form.dbSSLMode" placeholder="disable / require / verify-full" />
              <p class="mt-1.5 text-xs text-zinc-500 dark:text-zinc-400">本机可用 disable；远程生产数据库建议 require 或 verify-full</p>
            </div>
          </div>
        </section>

        <NAlert v-if="errorMsg" :type="errorMsg.includes('成功') ? 'success' : 'error'" :show-icon="true">
          {{ errorMsg }}
        </NAlert>

        <NButton attr-type="submit" type="primary" size="large" block :loading="step === 'loading'">
          {{ step === 'loading' ? '正在安装...' : '开始安装' }}
        </NButton>
      </form>
    </div>
  </div>
</template>

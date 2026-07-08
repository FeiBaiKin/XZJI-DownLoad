<script setup lang="ts">
// 安全设置 / 修改密码
// 业务逻辑保持不变；UI 重构为 Naive UI + Tailwind
import { ref } from 'vue'
import { NButton, NInput, NAlert } from 'naive-ui'
import api from '@/api'

const formData = ref({ currentPassword: '', newPassword: '', confirmPassword: '' })
const loading = ref(false)
const message = ref('')
const error = ref('')

async function handleSubmit() {
  error.value = ''
  message.value = ''

  if (!formData.value.currentPassword || !formData.value.newPassword) {
    error.value = '请填写当前密码和新密码'
    return
  }
  if (formData.value.newPassword.length < 6) {
    error.value = '新密码至少需要6个字符'
    return
  }
  if (formData.value.newPassword !== formData.value.confirmPassword) {
    error.value = '两次输入的新密码不一致'
    return
  }

  loading.value = true
  try {
    await api.put('/user/password', {
      currentPassword: formData.value.currentPassword,
      newPassword: formData.value.newPassword,
    })
    message.value = '密码修改成功'
    formData.value = { currentPassword: '', newPassword: '', confirmPassword: '' }
  } catch (err: unknown) {
    error.value = (err as { response?: { data?: { error?: string } } })?.response?.data?.error || '修改失败'
  } finally {
    loading.value = false
  }
}
</script>

<template>
  <div>
    <h1 class="text-xl font-bold tracking-tight text-zinc-900 dark:text-zinc-50">安全设置</h1>

    <section class="mt-6 rounded-2xl border border-zinc-200 bg-white p-5 sm:p-6 dark:border-zinc-800 dark:bg-zinc-900">
      <h3 class="text-[15px] font-semibold text-zinc-900 dark:text-zinc-100">修改密码</h3>
      <p class="mt-1 text-xs text-zinc-500 dark:text-zinc-400">修改成功后，下次登录请使用新密码</p>

      <NAlert v-if="message" type="success" class="mt-4" closable @close="message = ''">{{ message }}</NAlert>
      <NAlert v-if="error" type="error" class="mt-4" closable @close="error = ''">{{ error }}</NAlert>

      <form class="mt-5 max-w-md space-y-4" @submit.prevent="handleSubmit">
        <div>
          <label for="currentPassword" class="mb-1.5 block text-sm text-zinc-600 dark:text-zinc-400">当前密码</label>
          <NInput
            id="currentPassword"
            v-model:value="formData.currentPassword"
            type="password"
            show-password-on="click"
            :disabled="loading"
            :input-props="{ autocomplete: 'current-password' }"
          />
        </div>
        <div>
          <label for="newPassword" class="mb-1.5 block text-sm text-zinc-600 dark:text-zinc-400">新密码</label>
          <NInput
            id="newPassword"
            v-model:value="formData.newPassword"
            type="password"
            show-password-on="click"
            placeholder="至少 6 个字符"
            :disabled="loading"
            :input-props="{ autocomplete: 'new-password' }"
          />
        </div>
        <div>
          <label for="confirmPassword" class="mb-1.5 block text-sm text-zinc-600 dark:text-zinc-400">确认新密码</label>
          <NInput
            id="confirmPassword"
            v-model:value="formData.confirmPassword"
            type="password"
            show-password-on="click"
            placeholder="再次输入新密码"
            :disabled="loading"
            :input-props="{ autocomplete: 'new-password' }"
          />
        </div>
        <div class="pt-1">
          <NButton attr-type="submit" type="primary" :loading="loading">修改密码</NButton>
        </div>
      </form>
    </section>
  </div>
</template>

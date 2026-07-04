<script setup lang="ts">
// 个人资料：头像上传（≤2MB）+ 基本信息查看/编辑
// 业务逻辑保持不变；UI 重构为 Naive UI + Tailwind
import { ref, onMounted } from 'vue'
import { NButton, NInput, NAlert, NSkeleton, NTag } from 'naive-ui'
import api from '@/api'
import { useAuthStore } from '@/stores/auth'
import type { UserBrief } from '@/types/api'

const auth = useAuthStore()

const profile = ref<UserBrief | null>(null)
const loading = ref(true)
const editing = ref(false)
const saving = ref(false)
const message = ref('')
const error = ref('')
const formData = ref({ nickname: '', email: '' })
const fileInputRef = ref<HTMLInputElement | null>(null)

onMounted(() => { fetchProfile() })

async function fetchProfile() {
  try {
    const res = await api.get('/user/profile')
    profile.value = res.data
    formData.value = { nickname: res.data.nickname || '', email: res.data.email || '' }
  } catch (err) {
    console.error('获取资料失败:', err)
  } finally {
    loading.value = false
  }
}

async function handleSaveProfile() {
  saving.value = true
  error.value = ''
  message.value = ''
  try {
    const res = await api.put('/user/profile', formData.value)
    profile.value = res.data
    editing.value = false
    message.value = '资料更新成功'
    if (auth.user) auth.user.nickname = res.data.nickname
  } catch (err: unknown) {
    error.value = (err as { response?: { data?: { error?: string } } })?.response?.data?.error || '保存失败'
  } finally {
    saving.value = false
  }
}

async function handleAvatarUpload(e: Event) {
  const file = (e.target as HTMLInputElement).files?.[0]
  if (!file) return

  if (file.size > 2 * 1024 * 1024) {
    error.value = '文件大小不能超过 2MB'
    return
  }

  error.value = ''
  message.value = ''

  const fd = new FormData()
  fd.append('avatar', file)
  try {
    const res = await api.post('/user/avatar', fd)
    if (profile.value) profile.value = { ...profile.value, avatar: res.data.avatar }
    message.value = '头像更新成功'
    if (auth.user) auth.user.avatar = res.data.avatar
  } catch (err: unknown) {
    error.value = (err as { response?: { data?: { error?: string } } })?.response?.data?.error || '上传失败'
  }
}

function startEdit() {
  formData.value = { nickname: profile.value?.nickname || '', email: profile.value?.email || '' }
  editing.value = true
  error.value = ''
}
function cancelEdit() {
  editing.value = false
  formData.value = { nickname: profile.value?.nickname || '', email: profile.value?.email || '' }
  error.value = ''
}
</script>

<template>
  <div>
    <div v-if="loading" class="space-y-6">
      <NSkeleton text width="20%" size="large" />
      <div class="flex items-center gap-4">
        <NSkeleton circle width="72px" />
        <NSkeleton text width="30%" />
      </div>
      <NSkeleton text :repeat="6" />
    </div>

    <template v-else>
      <p v-if="!profile" class="rounded-xl bg-zinc-50 py-12 text-center text-sm text-zinc-400 dark:bg-zinc-900 dark:text-zinc-500">
        无法加载用户信息
      </p>

      <template v-else>
        <h2 class="text-xl font-bold tracking-tight text-zinc-900 dark:text-zinc-50">个人资料</h2>

        <!-- 头像区域 -->
        <section class="mt-6 flex items-center gap-5 rounded-2xl border border-zinc-200 bg-white p-5 dark:border-zinc-800 dark:bg-zinc-900">
          <button
            type="button"
            class="group relative size-18 shrink-0 overflow-hidden rounded-full outline-none focus-visible:ring-2 focus-visible:ring-emerald-500"
            title="更换头像"
            @click="fileInputRef?.click()"
          >
            <img v-if="profile.avatar" :src="profile.avatar" alt="头像" class="size-full object-cover" />
            <span v-else class="grid size-full place-items-center bg-emerald-600/90 text-2xl font-bold text-white dark:bg-emerald-400 dark:text-zinc-950">
              {{ (profile.nickname || profile.username || 'U')[0]!.toUpperCase() }}
            </span>
            <span class="absolute inset-0 grid place-items-center bg-black/45 text-xs font-medium text-white opacity-0 transition-opacity group-hover:opacity-100">
              更换头像
            </span>
          </button>
          <input
            ref="fileInputRef"
            type="file"
            accept="image/jpeg,image/png,image/gif,image/webp"
            class="hidden"
            @change="handleAvatarUpload"
          />
          <div class="min-w-0">
            <p class="truncate text-base font-semibold text-zinc-900 dark:text-zinc-100">{{ profile.nickname || profile.username }}</p>
            <p class="mt-1 text-xs text-zinc-400 dark:text-zinc-500">支持 JPG、PNG、GIF、WebP，最大 2MB</p>
          </div>
        </section>

        <!-- 基本信息 -->
        <section class="mt-6 rounded-2xl border border-zinc-200 bg-white p-5 dark:border-zinc-800 dark:bg-zinc-900">
          <div class="flex items-center justify-between">
            <h3 class="text-[15px] font-semibold text-zinc-900 dark:text-zinc-100">基本信息</h3>
            <NButton v-if="!editing" size="small" quaternary type="primary" @click="startEdit">编辑</NButton>
          </div>

          <NAlert v-if="message" type="success" class="mt-4" closable @close="message = ''">{{ message }}</NAlert>
          <NAlert v-if="error" type="error" class="mt-4" closable @close="error = ''">{{ error }}</NAlert>

          <!-- 编辑模式 -->
          <form v-if="editing" class="mt-5 max-w-md space-y-4" @submit.prevent="handleSaveProfile">
            <div>
              <label class="mb-1.5 block text-sm text-zinc-600 dark:text-zinc-400">用户名</label>
              <NInput :value="profile.username" disabled />
              <p class="mt-1 text-xs text-zinc-400 dark:text-zinc-500">用户名不可修改</p>
            </div>
            <div>
              <label class="mb-1.5 block text-sm text-zinc-600 dark:text-zinc-400">昵称</label>
              <NInput v-model:value="formData.nickname" placeholder="设置昵称" />
            </div>
            <div>
              <label class="mb-1.5 block text-sm text-zinc-600 dark:text-zinc-400">邮箱</label>
              <NInput v-model:value="formData.email" placeholder="用于找回密码" :input-props="{ type: 'email' }" />
            </div>
            <div class="flex gap-2 pt-1">
              <NButton attr-type="submit" type="primary" :loading="saving">保存</NButton>
              <NButton quaternary @click="cancelEdit">取消</NButton>
            </div>
          </form>

          <!-- 查看模式 -->
          <dl v-else class="mt-3 divide-y divide-zinc-100 dark:divide-zinc-800">
            <div class="flex items-center justify-between gap-4 py-3">
              <dt class="text-sm text-zinc-500 dark:text-zinc-400">用户名</dt>
              <dd class="text-sm font-medium text-zinc-900 dark:text-zinc-100">{{ profile.username }}</dd>
            </div>
            <div class="flex items-center justify-between gap-4 py-3">
              <dt class="text-sm text-zinc-500 dark:text-zinc-400">昵称</dt>
              <dd class="text-sm text-zinc-800 dark:text-zinc-200">{{ profile.nickname || '未设置' }}</dd>
            </div>
            <div class="flex items-center justify-between gap-4 py-3">
              <dt class="text-sm text-zinc-500 dark:text-zinc-400">邮箱</dt>
              <dd class="truncate text-sm text-zinc-800 dark:text-zinc-200">{{ profile.email || '未设置' }}</dd>
            </div>
            <div class="flex items-center justify-between gap-4 py-3">
              <dt class="text-sm text-zinc-500 dark:text-zinc-400">等级</dt>
              <dd class="font-mono text-sm text-zinc-800 dark:text-zinc-200">Lv.{{ profile.level || 1 }}</dd>
            </div>
            <div class="flex items-center justify-between gap-4 py-3">
              <dt class="text-sm text-zinc-500 dark:text-zinc-400">经验</dt>
              <dd class="font-mono text-sm text-zinc-800 dark:text-zinc-200">{{ profile.exp || 0 }} EXP</dd>
            </div>
            <div class="flex items-center justify-between gap-4 py-3">
              <dt class="text-sm text-zinc-500 dark:text-zinc-400">余额</dt>
              <dd class="font-mono text-sm text-zinc-800 dark:text-zinc-200">{{ (profile.balance || 0).toFixed(2) }} 元</dd>
            </div>
            <div class="flex items-center justify-between gap-4 py-3">
              <dt class="text-sm text-zinc-500 dark:text-zinc-400">会员</dt>
              <dd class="text-sm text-zinc-800 dark:text-zinc-200">
                {{ profile.membership_level
                  ? `${profile.membership_level}（${profile.membership_expires_at ? new Date(profile.membership_expires_at).toLocaleDateString('zh-CN') : ''}）`
                  : '无' }}
              </dd>
            </div>
            <div class="flex items-center justify-between gap-4 py-3">
              <dt class="text-sm text-zinc-500 dark:text-zinc-400">角色</dt>
              <dd><NTag :type="profile.role === 'admin' ? 'warning' : 'default'" size="small" round>{{ profile.role === 'admin' ? '管理员' : '普通用户' }}</NTag></dd>
            </div>
            <div class="flex items-center justify-between gap-4 py-3">
              <dt class="text-sm text-zinc-500 dark:text-zinc-400">注册时间</dt>
              <dd class="font-mono text-sm text-zinc-500 dark:text-zinc-400">{{ new Date(profile.created_at).toLocaleString('zh-CN') }}</dd>
            </div>
          </dl>
        </section>
      </template>
    </template>
  </div>
</template>

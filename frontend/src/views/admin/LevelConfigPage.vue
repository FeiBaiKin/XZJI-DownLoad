<script setup lang="ts">
// 等级配置：等级名称与所需经验（PUT 纯数组，适配 GO 契约）
// 业务逻辑保持不变；UI 重构为 Naive UI + Tailwind
import { ref, onMounted } from 'vue'
import { NButton, NInput, NInputNumber, NAlert, NSkeleton } from 'naive-ui'
import { Plus, X } from '@lucide/vue'
import AppIcon from '../../components/AppIcon.vue'
import api from '@/api'
import { useToastStore } from '@/stores/toast'
import type { LevelConfig } from '@/types/api'

const toast = useToastStore()
const levels = ref<LevelConfig[]>([])
const loading = ref(true)
const saving = ref(false)
const message = ref('')

onMounted(() => { fetchLevels() })

async function fetchLevels() {
  try {
    const res = await api.get('/admin/level-config')
    levels.value = res.data
  } catch (err) {
    console.error(err)
    toast.error('加载失败，请刷新重试')
  }
  finally { loading.value = false }
}

function updateLevel(idx: number, field: string, value: string | number) {
  levels.value[idx] = { ...levels.value[idx]!, [field]: value }
}

function addLevel() {
  const maxLevel = levels.value.length > 0 ? Math.max(...levels.value.map((l) => l.level)) : 0
  const maxExp = levels.value.length > 0 ? Math.max(...levels.value.map((l) => l.exp_required)) : 0
  levels.value.push({ level: maxLevel + 1, exp_required: maxExp + 1000, label: '' })
}

function removeLevel(idx: number) {
  levels.value = levels.value.filter((_, i) => i !== idx)
}

async function handleSave() {
  saving.value = true
  message.value = ''
  try {
    // 后端 level-config PUT 接收纯数组
    await api.put('/admin/level-config', levels.value)
    message.value = '保存成功'
    fetchLevels()
  } catch { message.value = '保存失败' }
  finally { saving.value = false }
}
</script>

<template>
  <div>
    <div v-if="loading" class="space-y-4">
      <NSkeleton text width="20%" size="large" />
      <NSkeleton height="360px" :sharp="false" />
    </div>

    <template v-else>
      <header class="mb-6 flex flex-wrap items-center justify-between gap-3">
        <div>
          <h1 class="text-xl font-bold tracking-tight text-zinc-900 dark:text-zinc-50">等级配置</h1>
          <p class="mt-1 text-sm text-zinc-500 dark:text-zinc-400">经验来自充值赠送与每日签到，达标自动升级</p>
        </div>
        <NButton secondary @click="addLevel">
          <template #icon><AppIcon :icon="Plus" scale="sm" /></template>
          添加等级
        </NButton>
      </header>

      <section class="rounded-2xl border border-zinc-200 bg-white p-5 dark:border-zinc-800 dark:bg-zinc-900">
        <form @submit.prevent="handleSave">
          <div
            v-if="levels.length === 0"
            class="rounded-xl border border-dashed border-zinc-300 py-12 text-center dark:border-zinc-700"
          >
            <p class="text-sm font-medium text-zinc-600 dark:text-zinc-300">暂无等级</p>
            <p class="mt-1 text-xs text-zinc-500 dark:text-zinc-400">点击右上角「添加等级」创建第一个等级</p>
          </div>
          <div v-else class="space-y-2.5">
            <div
              v-for="(lv, idx) in levels"
              :key="idx"
              class="grid grid-cols-1 gap-2 rounded-xl border border-zinc-200 px-3.5 py-2.5 sm:flex sm:items-center sm:gap-3 dark:border-zinc-700/80"
            >
              <span class="shrink-0 font-mono text-sm font-semibold text-emerald-600 sm:w-14 dark:text-emerald-400">Lv.{{ lv.level }}</span>
              <NInput
                size="small"
                class="flex-1"
                placeholder="等级名称"
                :value="lv.label || ''"
                @update:value="(v: string) => updateLevel(idx, 'label', v)"
              />
              <NInputNumber
                size="small"
                class="w-full sm:w-36"
                :min="0"
                placeholder="所需经验"
                :show-button="false"
                :value="lv.exp_required"
                @update:value="(v: number | null) => updateLevel(idx, 'exp_required', v ?? 0)"
              >
                <template #suffix><span class="text-[11px] text-zinc-500 dark:text-zinc-400">EXP</span></template>
              </NInputNumber>
              <button
                type="button"
                class="flex size-7 shrink-0 items-center justify-center justify-self-end rounded-lg text-zinc-400 transition-colors hover:bg-red-50 hover:text-red-500 focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-emerald-500/60 sm:justify-self-auto dark:hover:bg-red-500/10 dark:focus-visible:ring-emerald-400/60"
                title="删除此等级"
                aria-label="删除此等级"
                @click="removeLevel(idx)"
              >
                <AppIcon :icon="X" scale="xs" />
              </button>
            </div>
          </div>

          <NAlert v-if="message" :type="message === '保存成功' ? 'success' : 'error'" class="mt-4" closable @close="message = ''">
            {{ message }}
          </NAlert>

          <div class="mt-5">
            <NButton attr-type="submit" type="primary" :loading="saving">保存配置</NButton>
          </div>
        </form>
      </section>
    </template>
  </div>
</template>

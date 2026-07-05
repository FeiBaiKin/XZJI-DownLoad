<script setup lang="ts">
// 签到配置：每日余额/经验、连续签到天数与额外奖励
// 业务逻辑保持不变；UI 重构为 Naive UI + Tailwind
import { ref, onMounted } from 'vue'
import { NButton, NInputNumber, NAlert, NSkeleton } from 'naive-ui'
import api from '@/api'

interface CheckinConfig {
  daily_balance: number
  consecutive_days: number
  consecutive_bonus_balance: number
  daily_exp: number
}

const config = ref<CheckinConfig>({ daily_balance: 0.5, consecutive_days: 7, consecutive_bonus_balance: 2.0, daily_exp: 10 })
const loading = ref(true)
const saving = ref(false)
const message = ref('')

onMounted(() => { fetchConfig() })

async function fetchConfig() {
  try {
    const res = await api.get('/admin/checkin-config')
    config.value = res.data
  } catch (err) { console.error(err) }
  finally { loading.value = false }
}

async function handleSave() {
  saving.value = true
  message.value = ''
  try {
    await api.put('/admin/checkin-config', config.value)
    message.value = '保存成功'
  } catch { message.value = '保存失败' }
  finally { saving.value = false }
}
</script>

<template>
  <div>
    <div v-if="loading" class="space-y-4">
      <NSkeleton text width="20%" size="large" />
      <NSkeleton height="300px" :sharp="false" />
    </div>

    <template v-else>
      <header class="mb-6">
        <h2 class="text-xl font-bold tracking-tight text-zinc-900 dark:text-zinc-50">签到配置</h2>
        <p class="mt-1 text-sm text-zinc-500 dark:text-zinc-400">每日签到奖励与连续签到激励</p>
      </header>

      <section class="rounded-2xl border border-zinc-200 bg-white p-5 sm:p-6 dark:border-zinc-800 dark:bg-zinc-900">
        <form class="space-y-4" @submit.prevent="handleSave">
          <div>
            <label class="mb-1.5 block text-sm text-zinc-600 dark:text-zinc-400">每日签到余额（元）</label>
            <NInputNumber
              :value="config.daily_balance"
              :min="0"
              :step="0.01"
              class="w-full"
              @update:value="(v: number | null) => config.daily_balance = v ?? 0"
            />
          </div>
          <div>
            <label class="mb-1.5 block text-sm text-zinc-600 dark:text-zinc-400">每日签到经验</label>
            <NInputNumber
              :value="config.daily_exp"
              :min="1"
              class="w-full"
              @update:value="(v: number | null) => config.daily_exp = v ?? 1"
            />
          </div>
          <div>
            <label class="mb-1.5 block text-sm text-zinc-600 dark:text-zinc-400">连续签到天数（触发奖励）</label>
            <NInputNumber
              :value="config.consecutive_days"
              :min="2"
              class="w-full"
              @update:value="(v: number | null) => config.consecutive_days = v ?? 2"
            />
          </div>
          <div>
            <label class="mb-1.5 block text-sm text-zinc-600 dark:text-zinc-400">连续签到额外余额（元）</label>
            <NInputNumber
              :value="config.consecutive_bonus_balance"
              :min="0"
              :step="0.01"
              class="w-full"
              @update:value="(v: number | null) => config.consecutive_bonus_balance = v ?? 0"
            />
          </div>

          <NAlert v-if="message" :type="message === '保存成功' ? 'success' : 'error'" closable @close="message = ''">{{ message }}</NAlert>

          <NButton attr-type="submit" type="primary" :loading="saving">保存配置</NButton>
        </form>
      </section>
    </template>
  </div>
</template>

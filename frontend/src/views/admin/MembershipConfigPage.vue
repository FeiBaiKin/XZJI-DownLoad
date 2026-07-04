<script setup lang="ts">
// 会员配置：会员档位（名称/天数/价格/折扣/每日下载/标识色），PUT 纯数组
// 业务逻辑保持不变；UI 重构为 Naive UI + Tailwind（折叠行编辑）
import { ref, onMounted } from 'vue'
import { NButton, NInput, NInputNumber, NAlert, NSkeleton, NIcon, NColorPicker } from 'naive-ui'
import { Plus, X, ChevronDown } from '@vicons/tabler'
import api from '@/api'
import type { MembershipConfig } from '@/types/api'

const levels = ref<MembershipConfig[]>([])
const loading = ref(true)
const saving = ref(false)
const message = ref('')
const expandedIdx = ref<number | null>(null)

onMounted(() => { fetchLevels() })

async function fetchLevels() {
  try {
    const res = await api.get('/admin/membership-config')
    levels.value = res.data
  } catch (err) { console.error(err) }
  finally { loading.value = false }
}

function updateLevel(idx: number, field: string, value: string | number) {
  levels.value[idx] = { ...levels.value[idx]!, [field]: value }
}

function addLevel() {
  levels.value.push({ level_name: '', days: 30, price: 0, daily_download_limit: 20, discount: 0, color: '#888888' })
  expandedIdx.value = levels.value.length - 1
}

function removeLevel(idx: number) {
  levels.value = levels.value.filter((_, i) => i !== idx)
  if (expandedIdx.value === idx) expandedIdx.value = null
  else if (expandedIdx.value !== null && expandedIdx.value > idx) expandedIdx.value -= 1
}

async function handleSave() {
  saving.value = true
  message.value = ''
  try {
    // 后端 membership-config PUT 接收纯数组
    await api.put('/admin/membership-config', levels.value)
    message.value = '保存成功'
    fetchLevels()
  } catch (err: unknown) {
    const errorObj = err as { response?: { data?: { error?: string } } }
    message.value = errorObj?.response?.data?.error || '保存失败'
  }
  finally { saving.value = false }
}

function discountText(lv: MembershipConfig) {
  return lv.discount > 0 && lv.discount < 1 ? `${Math.round(lv.discount * 100)}折` : '原价'
}
</script>

<template>
  <div>
    <div v-if="loading" class="space-y-4">
      <NSkeleton text width="20%" size="large" />
      <NSkeleton v-for="i in 3" :key="i" height="64px" :sharp="false" />
    </div>

    <template v-else>
      <header class="mb-6 flex flex-wrap items-center justify-between gap-3">
        <div>
          <h2 class="text-xl font-bold tracking-tight text-zinc-900 dark:text-zinc-50">会员配置</h2>
          <p class="mt-1 text-sm text-zinc-500 dark:text-zinc-400">会员档位价格、时长、下载限额与折扣</p>
        </div>
        <NButton secondary @click="addLevel">
          <template #icon><NIcon><Plus /></NIcon></template>
          添加等级
        </NButton>
      </header>

      <form @submit.prevent="handleSave">
        <p
          v-if="levels.length === 0"
          class="rounded-xl border border-dashed border-zinc-300 py-12 text-center text-sm text-zinc-400 dark:border-zinc-700 dark:text-zinc-500"
        >
          暂无会员等级，点击右上角按钮添加
        </p>

        <div v-else class="space-y-3">
          <div
            v-for="(lv, idx) in levels"
            :key="idx"
            class="overflow-hidden rounded-xl border border-zinc-200 bg-white dark:border-zinc-800 dark:bg-zinc-900"
          >
            <!-- 摘要行 -->
            <div
              class="flex cursor-pointer items-center gap-3 px-4 py-3 select-none"
              :style="{ borderLeft: `3px solid ${lv.color || 'transparent'}` }"
              @click="expandedIdx = expandedIdx === idx ? null : idx"
            >
              <NIcon :size="15" class="shrink-0 text-zinc-400 transition-transform" :class="expandedIdx === idx ? 'rotate-180' : ''">
                <ChevronDown />
              </NIcon>
              <span class="min-w-0 flex-1 truncate font-medium text-zinc-900 dark:text-zinc-100" :style="lv.color ? { color: lv.color } : {}">
                {{ lv.level_name || '未命名' }}
              </span>

              <span class="hidden items-baseline gap-1 sm:flex">
                <span class="text-xs text-zinc-400 dark:text-zinc-500">价格</span>
                <span class="font-mono text-sm text-zinc-800 dark:text-zinc-200">{{ lv.price > 0 ? `${lv.price}元` : '免费' }}</span>
              </span>
              <span class="hidden items-baseline gap-1 sm:flex">
                <span class="text-xs text-zinc-400 dark:text-zinc-500">天数</span>
                <span class="font-mono text-sm text-zinc-800 dark:text-zinc-200">{{ lv.days }}天</span>
              </span>
              <span class="hidden items-baseline gap-1 md:flex">
                <span class="text-xs text-zinc-400 dark:text-zinc-500">折扣</span>
                <span class="font-mono text-sm text-zinc-800 dark:text-zinc-200">{{ lv.discount > 0 && lv.discount < 1 ? discountText(lv) : '-' }}</span>
              </span>
              <span class="hidden items-baseline gap-1 md:flex">
                <span class="text-xs text-zinc-400 dark:text-zinc-500">下载/天</span>
                <span class="font-mono text-sm text-zinc-800 dark:text-zinc-200">{{ lv.daily_download_limit > 0 ? lv.daily_download_limit : '无限' }}</span>
              </span>

              <span class="size-3 shrink-0 rounded-full border border-black/10 dark:border-white/15" :style="{ backgroundColor: lv.color || '#8888' }" :title="lv.color || '未设置'" />
              <button
                type="button"
                class="flex size-7 shrink-0 items-center justify-center rounded-lg text-zinc-400 transition-colors hover:bg-red-50 hover:text-red-500 dark:hover:bg-red-500/10"
                title="删除此档位"
                @click.stop="removeLevel(idx)"
              >
                <NIcon :size="14"><X /></NIcon>
              </button>
            </div>

            <!-- 展开编辑 -->
            <div v-if="expandedIdx === idx" class="grid gap-4 border-t border-zinc-100 p-4 sm:grid-cols-2 lg:grid-cols-3 dark:border-zinc-800">
              <div>
                <label class="mb-1.5 block text-sm text-zinc-600 dark:text-zinc-400">等级名称</label>
                <NInput
                  placeholder="等级名称"
                  :value="lv.level_name"
                  @update:value="(v: string) => updateLevel(idx, 'level_name', v)"
                />
              </div>
              <div>
                <label class="mb-1.5 block text-sm text-zinc-600 dark:text-zinc-400">有效天数</label>
                <NInputNumber
                  :min="1"
                  class="w-full"
                  :value="lv.days"
                  @update:value="(v: number | null) => updateLevel(idx, 'days', v ?? 1)"
                />
              </div>
              <div>
                <label class="mb-1.5 block text-sm text-zinc-600 dark:text-zinc-400">价格（元）</label>
                <NInputNumber
                  :min="0"
                  :step="0.01"
                  class="w-full"
                  :value="lv.price"
                  @update:value="(v: number | null) => updateLevel(idx, 'price', v ?? 0)"
                />
              </div>
              <div>
                <label class="mb-1.5 block text-sm text-zinc-600 dark:text-zinc-400">折扣率（0-1，0 为不打折）</label>
                <div class="flex items-center gap-2">
                  <NInputNumber
                    :min="0"
                    :max="1"
                    :step="0.01"
                    class="w-full"
                    :value="lv.discount || 0"
                    @update:value="(v: number | null) => updateLevel(idx, 'discount', v ?? 0)"
                  />
                  <span class="shrink-0 rounded-md bg-zinc-100 px-2 py-1 font-mono text-xs text-zinc-500 dark:bg-zinc-800 dark:text-zinc-400">
                    {{ discountText(lv) }}
                  </span>
                </div>
              </div>
              <div>
                <label class="mb-1.5 block text-sm text-zinc-600 dark:text-zinc-400">每日下载（0 为无限）</label>
                <NInputNumber
                  :min="0"
                  class="w-full"
                  :value="lv.daily_download_limit"
                  @update:value="(v: number | null) => updateLevel(idx, 'daily_download_limit', v ?? 0)"
                />
              </div>
              <div>
                <label class="mb-1.5 block text-sm text-zinc-600 dark:text-zinc-400">标识颜色</label>
                <NColorPicker
                  :modes="['hex']"
                  :show-alpha="false"
                  :value="lv.color || '#888888'"
                  @update:value="(v: string) => updateLevel(idx, 'color', v)"
                />
              </div>
            </div>
          </div>
        </div>

        <NAlert v-if="message" :type="message === '保存成功' ? 'success' : 'error'" class="mt-4" closable @close="message = ''">
          {{ message }}
        </NAlert>

        <div class="mt-5">
          <NButton attr-type="submit" type="primary" :loading="saving">保存配置</NButton>
        </div>
      </form>
    </template>
  </div>
</template>

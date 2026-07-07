<script setup lang="ts">
// 会员配置：卡片式档位编辑器，保留全部 6 字段与后端契约
// GET /admin/membership-config 返回纯数组，PUT 接收纯数组，按 level_name 全量同步
import { ref, computed, onMounted } from 'vue'
import { NButton, NInput, NInputNumber, NIcon, NSkeleton, NColorPicker } from 'naive-ui'
import { Plus, Trash, Crown, Calendar, Download, Discount, Coin } from '@vicons/tabler'
import api from '@/api'
import type { MembershipConfig } from '@/types/api'
import { useToastStore } from '@/stores/toast'

const levels = ref<MembershipConfig[]>([])
const loading = ref(true)
const saving = ref(false)
const message = ref<{ type: 'success' | 'error'; text: string } | null>(null)
const dirty = ref(false)
const toast = useToastStore()

onMounted(() => { fetchLevels() })

async function fetchLevels() {
  try {
    const res = await api.get('/admin/membership-config')
    levels.value = (res.data || []).map(normalize)
  } catch (err) {
    console.error(err)
    toast.error('加载失败，请刷新重试')
  }
  finally { loading.value = false }
  dirty.value = false
}

function normalize(lv: Partial<MembershipConfig>): MembershipConfig {
  return {
    level_name: lv.level_name ?? '',
    days: Number(lv.days ?? 30) || 0,
    price: Number(lv.price ?? 0) || 0,
    daily_download_limit: Number(lv.daily_download_limit ?? 0) || 0,
    discount: Number(lv.discount ?? 0) || 0,
    color: lv.color ?? '',
  }
}

function update(idx: number, field: keyof MembershipConfig, value: string | number) {
  levels.value[idx] = { ...levels.value[idx]!, [field]: value }
  dirty.value = true
}

function addLevel() {
  levels.value.push({ level_name: '', days: 30, price: 0, daily_download_limit: 20, discount: 0, color: '#10b981' })
  dirty.value = true
}

function removeLevel(idx: number) {
  levels.value = levels.value.filter((_, i) => i !== idx)
  dirty.value = true
}

async function handleSave() {
  const names = levels.value.map((l) => l.level_name.trim())
  if (names.some((n) => !n)) {
    message.value = { type: 'error', text: '每档等级名称不能为空' }
    return
  }
  if (new Set(names).size !== names.length) {
    message.value = { type: 'error', text: '存在重复的等级名称' }
    return
  }
  saving.value = true
  message.value = null
  try {
    await api.put('/admin/membership-config', levels.value)
    message.value = { type: 'success', text: '保存成功，前台已生效' }
    await fetchLevels()
  } catch (err: unknown) {
    const errorObj = err as { response?: { data?: { error?: string } } }
    message.value = { type: 'error', text: errorObj?.response?.data?.error || '保存失败' }
  }
  finally { saving.value = false }
}

// 展示用辅助函数
function discountText(lv: MembershipConfig) {
  return lv.discount > 0 && lv.discount < 1 ? `${Math.round(lv.discount * 100)} 折` : '原价'
}
function periodText(days: number) {
  if (!days) return '-'
  if (days >= 365) return `${Math.floor(days / 365)} 年`
  if (days >= 30) return `${Math.floor(days / 30)} 个月`
  return `${days} 天`
}
function dlText(lv: MembershipConfig) {
  const d = lv.daily_download_limit
  if (d === -1 || d === 0) return '不限'
  return `${d} 次/天`
}
function planColor(lv: MembershipConfig) {
  return lv.color || '#10b981'
}

const orderedLevels = computed(() => [...levels.value])
</script>

<template>
  <div>
    <!-- 加载骨架 -->
    <div v-if="loading" class="space-y-4">
      <NSkeleton text width="18%" size="large" />
      <div class="grid gap-4 sm:grid-cols-2">
        <NSkeleton v-for="i in 2" :key="i" height="200px" :sharp="false" />
      </div>
    </div>

    <template v-else>
      <!-- 页眉 -->
      <header class="flex flex-wrap items-end justify-between gap-4">
        <div>
          <h1 class="text-[26px] font-bold leading-tight tracking-tight text-zinc-900 sm:text-3xl dark:text-zinc-50">会员配置</h1>
          <p class="mt-1.5 max-w-[60ch] text-sm leading-relaxed text-zinc-500 dark:text-zinc-400">配置会员档位的名称、时长、价格、下载额度、折扣率与标识色。保存后立即对前台生效。</p>
        </div>
        <NButton
          type="primary"
          class="!bg-emerald-600 !text-white hover:!bg-emerald-700 dark:!bg-emerald-500 dark:hover:!bg-emerald-400"
          @click="addLevel"
        >
          <template #icon><NIcon><Plus /></NIcon></template>
          添加档位
        </NButton>
      </header>

      <!-- 提示条 -->
      <div
        v-if="message"
        class="mt-4 flex flex-col items-start gap-3 rounded-xl border px-4 py-2.5 text-sm sm:flex-row sm:items-center sm:justify-between"
        :class="message.type === 'success'
          ? 'border-emerald-200 bg-emerald-50 text-emerald-700 dark:border-emerald-400/20 dark:bg-emerald-400/10 dark:text-emerald-400'
          : 'border-red-200 bg-red-50 text-red-600 dark:border-red-400/20 dark:bg-red-400/10 dark:text-red-400'"
      >
        <span>{{ message.text }}</span>
        <button type="button" class="shrink-0 rounded opacity-70 hover:opacity-100 focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-emerald-500/60 dark:focus-visible:ring-emerald-400/60" @click="message = null">关闭</button>
      </div>

      <!-- 空状态 -->
      <div
        v-if="levels.length === 0"
        class="mt-6 rounded-2xl border border-dashed border-zinc-300 bg-zinc-50/40 py-20 text-center dark:border-zinc-700 dark:bg-zinc-900/40"
      >
        <span class="mx-auto grid size-14 place-items-center rounded-2xl bg-zinc-100 text-zinc-400 dark:bg-zinc-800 dark:text-zinc-600">
          <NIcon :size="28"><Crown /></NIcon>
        </span>
        <p class="mt-4 text-sm font-medium text-zinc-600 dark:text-zinc-300">暂无会员档位</p>
        <p class="mt-1 text-xs text-zinc-500 dark:text-zinc-400">点击右上角「添加档位」开始配置</p>
      </div>

      <!-- 档位卡片网格 -->
      <div v-else class="mt-6 grid gap-5 lg:grid-cols-2">
        <article
          v-for="(lv, idx) in orderedLevels"
          :key="idx"
          class="flex overflow-hidden rounded-2xl border border-zinc-200 bg-white dark:border-zinc-800 dark:bg-zinc-900"
        >
          <!-- 左侧色条 -->
          <div class="w-1.5 shrink-0" :style="{ backgroundColor: planColor(lv) }" />

          <div class="flex-1 p-5">
            <!-- 卡片头：色块 + 名称 + 删除 -->
            <div class="flex items-center justify-between gap-3">
              <div class="flex min-w-0 flex-1 items-center gap-2.5">
                <NColorPicker
                  :modes="['hex']"
                  :show-alpha="false"
                  size="small"
                  :value="lv.color || '#10b981'"
                  :render-label="() => null"
                  class="!w-8 shrink-0"
                  @update:value="(v: string) => update(idx, 'color', v)"
                />
                <NInput
                  :value="lv.level_name"
                  placeholder="等级名称，如 普通 / 高级 / 永久"
                  size="medium"
                  class="!w-full"
                  @update:value="(v: string) => update(idx, 'level_name', v)"
                />
              </div>
              <button
                type="button"
                class="flex size-8 shrink-0 items-center justify-center rounded-lg text-zinc-400 transition-colors hover:bg-red-50 hover:text-red-500 focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-emerald-500/60 dark:hover:bg-red-500/10 dark:hover:text-red-400 dark:focus-visible:ring-emerald-400/60"
                title="删除档位"
                aria-label="删除档位"
                @click="removeLevel(idx)"
              >
                <NIcon :size="16"><Trash /></NIcon>
              </button>
            </div>

            <!-- 字段分组：计费 -->
            <div class="mt-4 grid grid-cols-2 gap-3">
              <div>
                <label class="mb-1.5 flex items-center gap-1.5 text-xs font-medium text-zinc-500 dark:text-zinc-400">
                  <NIcon :size="13"><Coin /></NIcon>价格（元）
                </label>
                <NInputNumber
                  :min="0"
                  :step="0.01"
                  :value="lv.price"
                  size="medium"
                  class="!w-full"
                  placeholder="¥0"
                  @update:value="(v: number | null) => update(idx, 'price', v ?? 0)"
                />
              </div>
              <div>
                <label class="mb-1.5 flex items-center gap-1.5 text-xs font-medium text-zinc-500 dark:text-zinc-400">
                  <NIcon :size="13"><Calendar /></NIcon>有效天数
                </label>
                <NInputNumber
                  :min="1"
                  :value="lv.days"
                  size="medium"
                  class="!w-full"
                  placeholder="30"
                  @update:value="(v: number | null) => update(idx, 'days', v ?? 1)"
                />
              </div>
            </div>

            <!-- 字段分组：额度 -->
            <div class="mt-3 grid grid-cols-2 gap-3">
              <div>
                <label class="mb-1.5 flex items-center gap-1.5 text-xs font-medium text-zinc-500 dark:text-zinc-400">
                  <NIcon :size="13"><Download /></NIcon>每日下载（0 不限）
                </label>
                <NInputNumber
                  :min="0"
                  :value="lv.daily_download_limit"
                  size="medium"
                  class="!w-full"
                  placeholder="20"
                  @update:value="(v: number | null) => update(idx, 'daily_download_limit', v ?? 0)"
                />
              </div>
              <div>
                <label class="mb-1.5 flex items-center gap-1.5 text-xs font-medium text-zinc-500 dark:text-zinc-400">
                  <NIcon :size="13"><Discount /></NIcon>折扣（0 不打折）
                </label>
                <NInputNumber
                  :min="0"
                  :max="1"
                  :step="0.05"
                  :value="lv.discount || 0"
                  size="medium"
                  class="!w-full"
                  placeholder="0.8"
                  :format="(v: number | null) => v ? `${Math.round(v * 100)} 折` : '原价'"
                  :parse="(s: string) => { const m = s.match(/(\d+(?:\.\d+)?)/); return m ? Math.min(1, Number(m[1]!) / 100) : 0 }"
                  @update:value="(v: number | null) => update(idx, 'discount', v ?? 0)"
                />
              </div>
            </div>

            <!-- 实时预览条 -->
            <div class="mt-4 flex items-center gap-2 rounded-lg bg-zinc-50 px-3 py-2 text-xs dark:bg-zinc-800/60">
              <span class="size-2 rounded-full" :style="{ backgroundColor: planColor(lv) }" />
              <span class="font-medium text-zinc-700 dark:text-zinc-300">{{ lv.level_name || '未命名' }}</span>
              <span class="text-zinc-400 dark:text-zinc-500">·</span>
              <span class="text-zinc-500 dark:text-zinc-400">¥{{ lv.price }} · {{ periodText(lv.days) }} · {{ dlText(lv) }} · {{ discountText(lv) }}</span>
            </div>
          </div>
        </article>
      </div>

      <!-- 底部保存 -->
      <div v-if="levels.length > 0" class="mt-6 flex flex-col items-stretch gap-3 border-t border-zinc-100 pt-5 sm:flex-row sm:items-center sm:justify-between dark:border-zinc-800">
        <p class="flex items-center gap-1.5 text-xs">
          <span v-if="dirty" class="flex items-center gap-1 text-amber-600 dark:text-amber-400">
            <span class="size-1.5 rounded-full bg-amber-500" />有未保存的修改
          </span>
          <span v-else class="text-zinc-500 dark:text-zinc-400">保存后立即对前台生效</span>
        </p>
        <NButton
          type="primary"
          size="large"
          :loading="saving"
          :disabled="!dirty && levels.length > 0"
          class="!bg-emerald-600 hover:!bg-emerald-700 dark:!bg-emerald-500 dark:hover:!bg-emerald-400"
          @click="handleSave"
        >保存配置</NButton>
      </div>
    </template>
  </div>
</template>

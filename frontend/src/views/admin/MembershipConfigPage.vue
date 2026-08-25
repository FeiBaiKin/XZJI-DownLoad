<script setup lang="ts">
// 会员配置：行式内联编辑器 + 前台展示效果实时预览 + 悬浮保存栏
// 后端契约不变：GET /admin/membership-config 返回纯数组，PUT 接收纯数组，按 level_name 全量同步
// 视觉体系：面板 rounded-2xl / 控件与图标章 rounded-lg；emerald 为操作强调色，档位色为数据驱动语义色
import { ref, onMounted } from 'vue'
import { NButton, NInput, NInputNumber, NSkeleton, NColorPicker, NPopconfirm } from 'naive-ui'
import { Copy, Crown, Eye, Plus, Save, Trash2 } from '@lucide/vue'
import AppIcon from '../../components/AppIcon.vue'
import api from '@/api'
import type { MembershipConfig } from '@/types/api'
import { useToastStore } from '@/stores/toast'
import { tierIcon, tierColor, tierWash, readableOn, periodText, dlText, discountLabel, priceText, badgeText, NEW_TIER_COLORS } from '@/utils/tier'

const levels = ref<MembershipConfig[]>([])
const loading = ref(true)
const saving = ref(false)
const dirty = ref(false)
const toast = useToastStore()

onMounted(() => { fetchLevels() })

async function fetchLevels(opts: { silent?: boolean } = {}) {
  if (!opts.silent) loading.value = true
  try {
    const res = await api.get('/admin/membership-config')
    levels.value = (res.data || []).map(normalize)
    dirty.value = false
  } catch (err) {
    console.error(err)
    toast.error('加载失败，请刷新重试')
  } finally {
    loading.value = false
  }
}

function normalize(lv: Partial<MembershipConfig>): MembershipConfig {
  return {
    level_name: lv.level_name ?? '',
    days: Number(lv.days ?? 30) || 0,
    price: Number(lv.price ?? 0) || 0,
    daily_download_limit: Number(lv.daily_download_limit ?? 0) || 0,
    discount: Number(lv.discount ?? 0) || 0,
    color: lv.color ?? '',
    badge: lv.badge ?? '',
  }
}

function update(idx: number, field: keyof MembershipConfig, value: string | number) {
  levels.value[idx] = { ...levels.value[idx]!, [field]: value }
  dirty.value = true
}

function addLevel() {
  levels.value.push({
    level_name: '',
    days: 30,
    price: 0,
    daily_download_limit: 20,
    discount: 0,
    color: NEW_TIER_COLORS[levels.value.length % NEW_TIER_COLORS.length]!,
    badge: '',
  })
  dirty.value = true
}

function duplicateLevel(idx: number) {
  const src = levels.value[idx]!
  const names = new Set(levels.value.map((l) => l.level_name))
  let name = `${src.level_name || '未命名'} 副本`
  let n = 2
  while (names.has(name)) name = `${src.level_name || '未命名'} 副本${n++}`
  levels.value.splice(idx + 1, 0, { ...src, level_name: name })
  dirty.value = true
}

function removeLevel(idx: number) {
  levels.value = levels.value.filter((_, i) => i !== idx)
  dirty.value = true
}

async function handleSave() {
  const names = levels.value.map((l) => l.level_name.trim())
  if (names.some((n) => !n)) {
    toast.error('每档等级名称不能为空')
    return
  }
  if (new Set(names).size !== names.length) {
    toast.error('存在重复的等级名称，请修改后再保存')
    return
  }
  saving.value = true
  try {
    await api.put('/admin/membership-config', levels.value)
    toast.success('保存成功，前台已生效')
    await fetchLevels({ silent: true })
  } catch (err: unknown) {
    const errorObj = err as { response?: { data?: { error?: string } } }
    toast.error(errorObj?.response?.data?.error || '保存失败')
  } finally {
    saving.value = false
  }
}

async function discardChanges() {
  await fetchLevels({ silent: true })
}
</script>

<template>
  <div>
    <!-- 加载骨架：页眉 + 编辑面板 + 预览 -->
    <div v-if="loading" class="space-y-5">
      <NSkeleton text width="18%" size="large" />
      <NSkeleton height="280px" :sharp="false" />
      <NSkeleton height="150px" :sharp="false" />
    </div>

    <template v-else>
      <!-- 页眉 -->
      <header class="flex flex-wrap items-end justify-between gap-4">
        <div>
          <h1 class="text-xl font-bold tracking-tight text-zinc-900 dark:text-zinc-50">会员配置</h1>
          <p class="mt-1 max-w-[60ch] text-sm leading-relaxed text-zinc-500 dark:text-zinc-400">
            配置会员档位的名称、时长、价格、下载额度、折扣、标识色与推广角标，保存后立即对前台生效。
          </p>
        </div>
        <NButton type="primary" @click="addLevel">
          <template #icon><AppIcon :icon="Plus" scale="sm" /></template>
          添加档位
        </NButton>
      </header>

      <!-- 空状态 -->
      <div
        v-if="levels.length === 0"
        class="mt-6 rounded-2xl border border-dashed border-zinc-300 bg-zinc-50/40 py-20 text-center dark:border-zinc-700 dark:bg-zinc-900/40"
      >
        <span class="mx-auto grid size-14 place-items-center rounded-xl bg-zinc-100 text-zinc-400 dark:bg-zinc-800 dark:text-zinc-600">
          <AppIcon :icon="Crown" scale="lg" />
        </span>
        <p class="mt-4 text-sm font-medium text-zinc-600 dark:text-zinc-300">暂无会员档位</p>
        <p class="mt-1 text-xs text-zinc-500 dark:text-zinc-400">创建第一个档位后，前台会员中心即可展示并售卖。</p>
        <NButton type="primary" size="small" class="mt-5" @click="addLevel">
          <template #icon><AppIcon :icon="Plus" scale="sm" /></template>
          创建第一个档位
        </NButton>
      </div>

      <!-- 档位编辑器 -->
      <section v-else class="mt-6 overflow-hidden rounded-2xl border border-zinc-200 bg-white dark:border-zinc-800 dark:bg-zinc-900">
        <!-- 列头（宽屏） -->
        <div class="hidden border-b border-zinc-100 px-5 py-3 text-xs font-medium text-zinc-500 xl:grid xl:grid-cols-[minmax(200px,1.5fr)_repeat(4,minmax(104px,1fr))_minmax(96px,0.9fr)_96px] xl:items-end xl:gap-3 dark:border-zinc-800 dark:text-zinc-400">
          <span>档位名称与标识色</span>
          <span>价格（元）</span>
          <span>有效天数</span>
          <span>每日下载<span class="ml-1 font-normal text-zinc-400 dark:text-zinc-500">0 不限</span></span>
          <span>付费折扣<span class="ml-1 font-normal text-zinc-400 dark:text-zinc-500">0 原价</span></span>
          <span>角标<span class="ml-1 font-normal text-zinc-400 dark:text-zinc-500">选填</span></span>
          <span class="text-right">操作</span>
        </div>

        <!-- 行（窄屏折叠为带标签的卡式块） -->
        <div class="divide-y divide-zinc-100 dark:divide-zinc-800">
          <div
            v-for="(lv, idx) in levels"
            :key="idx"
            class="grid grid-cols-2 gap-3 px-4 py-4 transition-colors hover:bg-zinc-50/60 sm:px-5 xl:grid-cols-[minmax(200px,1.5fr)_repeat(4,minmax(104px,1fr))_minmax(96px,0.9fr)_96px] xl:items-center dark:hover:bg-zinc-800/30"
          >
            <!-- 档位：图标章即取色器 + 名称 -->
            <div class="col-span-2 flex items-center gap-2.5 xl:col-span-1">
              <div
                class="relative size-9 shrink-0 rounded-lg focus-within:outline-2 focus-within:outline-offset-2 focus-within:outline-emerald-500"
                title="点击修改标识色"
              >
                <span
                  class="pointer-events-none absolute inset-0 grid place-items-center rounded-lg"
                  :style="{ backgroundColor: tierColor(lv.color), color: readableOn(tierColor(lv.color)) }"
                >
                  <AppIcon :icon="tierIcon(idx)" scale="sm" />
                </span>
                <NColorPicker
                  class="tier-color-overlay"
                  :modes="['hex']"
                  :show-alpha="false"
                  :value="tierColor(lv.color)"
                  :render-label="() => null"
                  @update:value="(v: string) => update(idx, 'color', v)"
                />
              </div>
              <NInput
                :value="lv.level_name"
                placeholder="等级名称，如：高级会员"
                class="!w-full"
                @update:value="(v: string) => update(idx, 'level_name', v)"
              />
              <!-- 窄屏操作按钮 -->
              <div class="flex shrink-0 items-center gap-1 xl:hidden">
                <button
                  type="button"
                  class="flex size-8 items-center justify-center rounded-lg text-zinc-400 transition-colors hover:bg-zinc-100 hover:text-zinc-600 focus-visible:outline-2 focus-visible:outline-offset-2 focus-visible:outline-emerald-500 dark:hover:bg-zinc-800 dark:hover:text-zinc-300"
                  title="复制档位"
                  aria-label="复制档位"
                  @click="duplicateLevel(idx)"
                >
                  <AppIcon :icon="Copy" scale="sm" />
                </button>
                <NPopconfirm @positive-click="removeLevel(idx)">
                  <template #trigger>
                    <button
                      type="button"
                      class="flex size-8 items-center justify-center rounded-lg text-zinc-400 transition-colors hover:bg-red-50 hover:text-red-500 focus-visible:outline-2 focus-visible:outline-offset-2 focus-visible:outline-emerald-500 dark:hover:bg-red-500/10 dark:hover:text-red-400"
                      title="删除档位"
                      aria-label="删除档位"
                    >
                      <AppIcon :icon="Trash2" scale="sm" />
                    </button>
                  </template>
                  确认移除「{{ lv.level_name || '未命名' }}」？保存后生效。
                </NPopconfirm>
              </div>
            </div>

            <!-- 价格 -->
            <div>
              <label class="mb-1 block text-xs font-medium text-zinc-500 xl:hidden dark:text-zinc-400">价格（元）</label>
              <NInputNumber
                :min="0"
                :step="0.01"
                :value="lv.price"
                class="!w-full"
                placeholder="0"
                @update:value="(v: number | null) => update(idx, 'price', v ?? 0)"
              />
            </div>
            <!-- 有效天数 -->
            <div>
              <label class="mb-1 block text-xs font-medium text-zinc-500 xl:hidden dark:text-zinc-400">有效天数</label>
              <NInputNumber
                :min="1"
                :value="lv.days"
                class="!w-full"
                placeholder="30"
                @update:value="(v: number | null) => update(idx, 'days', v ?? 1)"
              />
            </div>
            <!-- 每日下载 -->
            <div>
              <label class="mb-1 block text-xs font-medium text-zinc-500 xl:hidden dark:text-zinc-400">每日下载（0 不限）</label>
              <NInputNumber
                :min="0"
                :value="lv.daily_download_limit"
                class="!w-full"
                placeholder="20"
                @update:value="(v: number | null) => update(idx, 'daily_download_limit', v ?? 0)"
              />
            </div>
            <!-- 折扣 -->
            <div>
              <label class="mb-1 block text-xs font-medium text-zinc-500 xl:hidden dark:text-zinc-400">付费折扣（0 原价）</label>
              <NInputNumber
                :min="0"
                :max="1"
                :step="0.05"
                :value="lv.discount || 0"
                class="!w-full"
                placeholder="0.8"
                :format="(v: number | null) => v ? `${Math.round(v * 100)} 折` : '原价'"
                :parse="(s: string) => { const m = s.match(/(\d+(?:\.\d+)?)/); return m ? Math.min(1, Number(m[1]!) / 100) : 0 }"
                @update:value="(v: number | null) => update(idx, 'discount', v ?? 0)"
              />
            </div>
            <!-- 角标 -->
            <div>
              <label class="mb-1 block text-xs font-medium text-zinc-500 xl:hidden dark:text-zinc-400">角标（选填）</label>
              <NInput
                :value="lv.badge"
                :maxlength="8"
                placeholder="如：推荐"
                class="!w-full"
                @update:value="(v: string) => update(idx, 'badge', v)"
              />
            </div>

            <!-- 宽屏操作 -->
            <div class="hidden items-center justify-end gap-1 xl:flex">
              <button
                type="button"
                class="flex size-8 items-center justify-center rounded-lg text-zinc-400 transition-colors hover:bg-zinc-100 hover:text-zinc-600 focus-visible:outline-2 focus-visible:outline-offset-2 focus-visible:outline-emerald-500 dark:hover:bg-zinc-800 dark:hover:text-zinc-300"
                title="复制档位"
                aria-label="复制档位"
                @click="duplicateLevel(idx)"
              >
                <AppIcon :icon="Copy" scale="sm" />
              </button>
              <NPopconfirm @positive-click="removeLevel(idx)">
                <template #trigger>
                  <button
                    type="button"
                    class="flex size-8 items-center justify-center rounded-lg text-zinc-400 transition-colors hover:bg-red-50 hover:text-red-500 focus-visible:outline-2 focus-visible:outline-offset-2 focus-visible:outline-emerald-500 dark:hover:bg-red-500/10 dark:hover:text-red-400"
                    title="删除档位"
                    aria-label="删除档位"
                  >
                    <AppIcon :icon="Trash2" scale="sm" />
                  </button>
                </template>
                确认移除「{{ lv.level_name || '未命名' }}」？保存后生效。
              </NPopconfirm>
            </div>
          </div>
        </div>
      </section>

      <!-- 前台展示效果预览 -->
      <section v-if="levels.length > 0" class="mt-5 rounded-2xl border border-zinc-200 bg-zinc-50/60 p-5 dark:border-zinc-800 dark:bg-zinc-900/40">
        <div class="flex flex-wrap items-baseline justify-between gap-2">
          <h2 class="flex items-center gap-2 text-sm font-semibold text-zinc-800 dark:text-zinc-200">
            <AppIcon :icon="Eye" scale="sm" class="text-emerald-600 dark:text-emerald-400" />
            前台展示效果
          </h2>
          <p class="text-xs text-zinc-500 dark:text-zinc-400">按上方顺序展示，填写角标的档位会突出标注，随编辑实时更新</p>
        </div>

        <div class="mt-4 grid gap-4 sm:grid-cols-2 xl:grid-cols-3">
          <article
            v-for="(lv, idx) in levels"
            :key="idx"
            class="overflow-hidden rounded-xl border border-zinc-200 bg-white dark:border-zinc-800 dark:bg-zinc-900"
          >
            <div class="p-4 pb-3" :style="{ background: tierWash(lv.color) }">
              <div class="flex items-center gap-2.5">
                <span
                  class="grid size-9 shrink-0 place-items-center rounded-lg"
                  :style="{ backgroundColor: tierColor(lv.color), color: readableOn(tierColor(lv.color)) }"
                >
                  <AppIcon :icon="tierIcon(idx)" scale="sm" />
                </span>
                <p class="min-w-0 truncate text-sm font-bold text-zinc-900 dark:text-zinc-50">{{ lv.level_name || '未命名档位' }}</p>
                <span
                  v-if="badgeText(lv.badge)"
                  class="shrink-0 rounded-full px-2 py-0.5 text-[10px] font-semibold leading-none"
                  :style="{ backgroundColor: tierColor(lv.color), color: readableOn(tierColor(lv.color)) }"
                >{{ badgeText(lv.badge) }}</span>
              </div>
            </div>
            <div class="px-4 pb-3.5">
              <div class="flex items-baseline gap-1">
                <span class="text-xs text-zinc-400 dark:text-zinc-500">¥</span>
                <span class="font-mono text-xl font-bold tracking-tight text-zinc-900 dark:text-zinc-50">{{ priceText(lv.price) }}</span>
                <span class="text-xs text-zinc-400 dark:text-zinc-500">/ {{ periodText(lv.days) }}</span>
              </div>
              <p class="mt-1 text-xs text-zinc-500 dark:text-zinc-400">{{ dlText(lv.daily_download_limit) }} · {{ discountLabel(lv.discount) ?? '标准价' }}</p>
            </div>
          </article>
        </div>
      </section>

      <!-- 悬浮保存栏 -->
      <div
        v-if="levels.length > 0"
        class="sticky bottom-4 z-10 mt-5 flex flex-col gap-3 rounded-2xl border border-zinc-200 bg-white/95 p-4 shadow-lg shadow-zinc-900/5 backdrop-blur sm:flex-row sm:items-center sm:justify-between dark:border-zinc-800 dark:bg-zinc-900/95 dark:shadow-black/30"
      >
        <p class="text-xs">
          <span v-if="dirty" class="inline-flex items-center gap-1.5 font-medium text-amber-600 dark:text-amber-400">
            <span class="size-1.5 rounded-full bg-current" />有未保存的修改
          </span>
          <span v-else class="text-zinc-500 dark:text-zinc-400">保存后立即对前台生效</span>
        </p>
        <div class="flex items-center gap-2.5">
          <NPopconfirm v-if="dirty" @positive-click="discardChanges">
            <template #trigger>
              <NButton quaternary :disabled="saving">放弃修改</NButton>
            </template>
            放弃所有未保存的修改？
          </NPopconfirm>
          <NButton type="primary" :loading="saving" :disabled="!dirty" @click="handleSave">
            <template #icon><AppIcon :icon="Save" scale="sm" /></template>
            保存配置
          </NButton>
        </div>
      </div>
    </template>
  </div>
</template>

<style scoped>
/* 图标章即取色器：透明的 NColorPicker 铺满整个图标章作为点击区 */
.tier-color-overlay {
  position: absolute !important;
  inset: 0;
  width: 100% !important;
  height: 100% !important;
  opacity: 0;
}
.tier-color-overlay :deep(.n-color-picker-trigger) {
  height: 100%;
}
</style>

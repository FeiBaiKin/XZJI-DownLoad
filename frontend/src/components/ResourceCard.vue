<script setup lang="ts">
// 可复用资源卡片：关键词名 + 描述 + 分类标签 + 日期 + 复制交互
import { ref, computed } from 'vue'
import { NIcon } from 'naive-ui'
import { Copy, Check, Heart } from '@vicons/tabler'
import type { PublicKeyword } from '@/api/content'

const props = defineProps<{
  keyword: PublicKeyword
  categoryName?: string
  createdAt?: string
  favorited?: boolean
}>()

const emit = defineEmits<{
  copy: [keyword: PublicKeyword]
  toggleFavorite: []
}>()

const copied = ref(false)
let copyTimer: ReturnType<typeof setTimeout> | null = null

// 根据关键词名生成首字母
const firstLetter = computed(() => {
  const c = props.keyword.keyword?.charAt(0)
  return c ? c.toUpperCase() : '?'
})

// 根据分类名分配颜色（hash 到预设颜色池）
const CATEGORY_COLORS = [
  { bg: 'bg-emerald-100 text-emerald-700 dark:bg-emerald-400/15 dark:text-emerald-400', icon: 'bg-emerald-500' },
  { bg: 'bg-blue-100 text-blue-700 dark:bg-blue-400/15 dark:text-blue-400', icon: 'bg-blue-500' },
  { bg: 'bg-purple-100 text-purple-700 dark:bg-purple-400/15 dark:text-purple-400', icon: 'bg-purple-500' },
  { bg: 'bg-amber-100 text-amber-700 dark:bg-amber-400/15 dark:text-amber-400', icon: 'bg-amber-500' },
  { bg: 'bg-rose-100 text-rose-700 dark:bg-rose-400/15 dark:text-rose-400', icon: 'bg-rose-500' },
  { bg: 'bg-cyan-100 text-cyan-700 dark:bg-cyan-400/15 dark:text-cyan-400', icon: 'bg-cyan-500' },
  { bg: 'bg-indigo-100 text-indigo-700 dark:bg-indigo-400/15 dark:text-indigo-400', icon: 'bg-indigo-500' },
  { bg: 'bg-teal-100 text-teal-700 dark:bg-teal-400/15 dark:text-teal-400', icon: 'bg-teal-500' },
]

function categoryColor(name?: string) {
  if (!name) return CATEGORY_COLORS[0]
  let hash = 0
  for (let i = 0; i < name.length; i++) hash = (hash * 31 + name.charCodeAt(i)) | 0
  return CATEGORY_COLORS[Math.abs(hash) % CATEGORY_COLORS.length]
}

function formatDate(ts?: string) {
  if (!ts) return ''
  const d = new Date(ts)
  const now = new Date()
  const diff = now.getTime() - d.getTime()
  if (diff < 3600000) return `${Math.floor(diff / 60000)}分钟前`
  if (diff < 86400000) return `${Math.floor(diff / 3600000)}小时前`
  if (diff < 604800000) return `${Math.floor(diff / 86400000)}天前`
  return `${d.getFullYear()}-${String(d.getMonth() + 1).padStart(2, '0')}-${String(d.getDate()).padStart(2, '0')}`
}

function handleCopy() {
  navigator.clipboard.writeText(props.keyword.keyword).then(() => {
    copied.value = true
    if (copyTimer) clearTimeout(copyTimer)
    copyTimer = setTimeout(() => { copied.value = false }, 1200)
    emit('copy', props.keyword)
  }).catch(() => { /* ignore */ })
}

</script>

<template>
  <button
    type="button"
    class="group flex w-full items-center gap-3 rounded-xl border bg-white p-3.5 text-left motion-safe:transition-all dark:bg-zinc-900"
    :class="copied
      ? 'border-emerald-500 ring-1 ring-emerald-500/40 dark:border-emerald-400'
      : 'border-zinc-200 hover:border-emerald-400/70 hover:shadow-sm dark:border-zinc-800 dark:hover:border-emerald-400/50'"
    title="点击复制关键词"
    @click="handleCopy"
  >
    <!-- 彩色方形图标 -->
    <span
      class="grid size-10 shrink-0 place-items-center rounded-lg text-sm font-bold"
      :class="CATEGORY_COLORS[0].bg"
    >
      {{ firstLetter }}
    </span>

    <!-- 中间：名称 + 描述 + 标签和日期 -->
    <div class="min-w-0 flex-1">
      <div class="flex items-center gap-2">
        <span class="truncate text-[15px] font-medium text-zinc-900 dark:text-zinc-100">
          {{ keyword.keyword }}
        </span>
      </div>
      <p class="mt-0.5 line-clamp-1 text-[13px] leading-relaxed text-zinc-500 dark:text-zinc-400">
        {{ keyword.description || '暂无描述' }}
      </p>
      <div class="mt-1.5 flex flex-wrap items-center gap-2 text-xs text-zinc-400 dark:text-zinc-500">
        <span
          v-if="categoryName"
          class="rounded-full px-2 py-0.5 text-[11px] font-medium"
          :class="categoryColor(categoryName).bg"
        >{{ categoryName }}</span>
        <span v-if="createdAt" class="font-mono">{{ formatDate(createdAt) }}</span>
      </div>
    </div>

    <!-- 右侧：收藏 + 复制 -->
    <span class="flex shrink-0 flex-col items-center gap-1">
      <!-- 心形收藏 -->
      <button
        type="button"
        class="grid size-7 place-items-center rounded-full motion-safe:transition-colors"
        :class="favorited
          ? 'text-rose-500 hover:text-rose-600 dark:text-rose-400 dark:hover:text-rose-300'
          : 'text-zinc-300 hover:text-rose-400 dark:text-zinc-600 dark:hover:text-rose-400'"
        :title="favorited ? '取消收藏' : '收藏'"
        @click.stop.prevent="emit('toggleFavorite')"
      >
        <NIcon :size="16"><Heart /></NIcon>
      </button>
      <span
        class="grid size-8 place-items-center rounded-lg text-xs font-medium motion-safe:transition-colors"
        :class="copied
          ? 'bg-emerald-50 text-emerald-600 dark:bg-emerald-400/10 dark:text-emerald-400'
          : 'bg-zinc-100 text-zinc-500 group-hover:bg-emerald-50 group-hover:text-emerald-600 dark:bg-zinc-800 dark:text-zinc-400 dark:group-hover:bg-emerald-400/10 dark:group-hover:text-emerald-400'"
      >
        <NIcon :size="15"><Check v-if="copied" /><Copy v-else /></NIcon>
      </span>
      <span
        class="text-[10px]"
        :class="copied ? 'text-emerald-600 dark:text-emerald-400' : 'text-zinc-400 dark:text-zinc-600'"
      >{{ copied ? '已复制' : '复制' }}</span>
    </span>
  </button>
</template>

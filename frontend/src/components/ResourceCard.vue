<script setup lang="ts">
// 可复用资源卡片：竖向网格卡片，适配两列布局
// 整卡可点击复制关键词；右上角收藏按钮为真实按钮
import { ref, computed } from 'vue'
import { NIcon } from 'naive-ui'
import { Copy, Check, Heart } from '@vicons/tabler'
import { useToastStore } from '@/stores/toast'
import type { PublicKeyword } from '@/api/content'
import type { KeywordAttribute } from '@/types/api'

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

const toast = useToastStore()

const copied = ref(false)
let copyTimer: ReturnType<typeof setTimeout> | null = null

// 根据关键词名生成首字母
const firstLetter = computed(() => {
  const c = props.keyword.keyword?.charAt(0)
  return c ? c.toUpperCase() : '?'
})

// 根据分类名分配颜色（hash 到预设颜色池）
const CATEGORY_COLORS = [
  'bg-emerald-100 text-emerald-700 dark:bg-emerald-400/15 dark:text-emerald-400',
  'bg-blue-100 text-blue-700 dark:bg-blue-400/15 dark:text-blue-400',
  'bg-purple-100 text-purple-700 dark:bg-purple-400/15 dark:text-purple-400',
  'bg-amber-100 text-amber-700 dark:bg-amber-400/15 dark:text-amber-400',
  'bg-rose-100 text-rose-700 dark:bg-rose-400/15 dark:text-rose-400',
  'bg-cyan-100 text-cyan-700 dark:bg-cyan-400/15 dark:text-cyan-400',
  'bg-indigo-100 text-indigo-700 dark:bg-indigo-400/15 dark:text-indigo-400',
  'bg-teal-100 text-teal-700 dark:bg-teal-400/15 dark:text-teal-400',
]

function categoryColor(name?: string) {
  if (!name) return CATEGORY_COLORS[0]
  let hash = 0
  for (let i = 0; i < name.length; i++) hash = (hash * 31 + name.charCodeAt(i)) | 0
  return CATEGORY_COLORS[Math.abs(hash) % CATEGORY_COLORS.length]
}

const avatarColor = computed(() => categoryColor(props.categoryName))

// 解析 properties.attributes JSON 字符串为 [{label,value}]
const parsedAttributes = computed<KeywordAttribute[]>(() => {
  const raw = props.keyword.attributes
  if (!raw) return []
  try {
    const parsed = JSON.parse(raw)
    if (Array.isArray(parsed)) {
      return parsed.filter(
        (a: unknown): a is KeywordAttribute =>
          !!a && typeof (a as KeywordAttribute).label === 'string' && typeof (a as KeywordAttribute).value === 'string',
      )
    }
  } catch { /* ignore */ }
  return []
})

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
  }).catch((err) => {
    console.error('复制失败', err)
    toast.error('复制失败，请手动复制')
  })
}

function handleKeydown(e: KeyboardEvent) {
  if (e.key === 'Enter' || e.key === ' ') {
    e.preventDefault()
    handleCopy()
  }
}
</script>

<template>
  <div
    role="button"
    tabindex="0"
    class="group relative flex h-full cursor-pointer flex-col rounded-2xl border bg-white p-4 text-left motion-safe:transition-all focus:outline-none focus-visible:ring-2 focus-visible:ring-emerald-500/60 dark:bg-zinc-900"
    :class="copied
      ? 'border-emerald-500 ring-1 ring-emerald-500/40 dark:border-emerald-400'
      : 'border-zinc-200 hover:-translate-y-0.5 hover:border-emerald-400/70 hover:shadow-md dark:border-zinc-800 dark:hover:border-emerald-400/50'"
    title="点击复制关键词"
    @click="handleCopy"
    @keydown="handleKeydown"
  >
    <!-- 头部：图标 + 标题 + 右上角收藏 -->
    <div class="flex items-start gap-2.5 pr-7">
      <!-- 真实图标 / 首字母回退 -->
      <div
        class="grid size-10 shrink-0 place-items-center overflow-hidden rounded-xl text-sm font-bold motion-safe:transition-transform group-hover:scale-105"
        :class="keyword.icon ? '' : avatarColor"
      >
        <img v-if="keyword.icon" :src="keyword.icon" :alt="keyword.keyword" class="size-full object-cover" loading="lazy" />
        <span v-else>{{ firstLetter }}</span>
      </div>
      <h3 class="min-w-0 flex-1 break-words pt-0.5 text-[15px] font-semibold leading-snug text-zinc-900 dark:text-zinc-100">
        {{ keyword.keyword }}
      </h3>
    </div>

    <!-- 右上角收藏按钮（真实按钮，避免与卡片可点击区冲突） -->
    <button
      type="button"
      class="absolute right-3 top-3 grid size-7 place-items-center rounded-full motion-safe:transition-colors focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-emerald-500/60 dark:focus-visible:ring-emerald-400/60"
      :class="favorited
        ? 'text-rose-500 hover:text-rose-600 dark:text-rose-400 dark:hover:text-rose-300'
        : 'text-zinc-300 hover:text-rose-400 dark:text-zinc-600 dark:hover:text-rose-400'"
      :title="favorited ? '取消收藏' : '收藏'"
      :aria-label="favorited ? '取消收藏' : '收藏'"
      @click.stop.prevent="emit('toggleFavorite')"
    >
      <NIcon :size="16"><Heart /></NIcon>
    </button>

    <!-- 描述：两行截断，撑开卡片高度 -->
    <p class="mt-2.5 line-clamp-2 min-h-[2.5rem] text-[13px] leading-relaxed text-zinc-500 dark:text-zinc-400">
      {{ keyword.description || '暂无描述' }}
    </p>

    <!-- 属性标签（键值对，例如 大小=56MB / 平台=安卓） -->
    <div v-if="parsedAttributes.length > 0" class="mt-3 flex flex-wrap gap-1.5">
      <span
        v-for="(attr, idx) in parsedAttributes"
        :key="idx"
        class="inline-flex items-center gap-1 rounded-md border border-emerald-200 bg-emerald-50/70 px-1.5 py-0.5 text-[11px] font-medium text-emerald-700 dark:border-emerald-400/20 dark:bg-emerald-400/10 dark:text-emerald-400"
      >
        <span class="text-emerald-600/80 dark:text-emerald-400/80">{{ attr.label }}</span>
        <span class="opacity-40">·</span>
        <span class="text-zinc-600 dark:text-zinc-300">{{ attr.value }}</span>
      </span>
    </div>

    <!-- 底部：分类标签 + 日期 + 复制动作 -->
    <div class="mt-3 flex items-center justify-between gap-2">
      <div class="flex min-w-0 flex-1 flex-wrap items-center gap-2 text-xs text-zinc-500 dark:text-zinc-400">
        <span
          v-if="categoryName"
          class="inline-flex items-center rounded-full px-2 py-0.5 text-[11px] font-medium"
          :class="avatarColor"
        >
          {{ categoryName }}
        </span>
        <span v-if="createdAt" class="font-mono text-[11px]">{{ formatDate(createdAt) }}</span>
      </div>

      <span
        class="flex shrink-0 items-center gap-1 rounded-lg px-2 py-1 text-[11px] font-medium motion-safe:transition-colors"
        :class="copied
          ? 'bg-emerald-50 text-emerald-600 dark:bg-emerald-400/10 dark:text-emerald-400'
          : 'text-zinc-500 group-hover:bg-emerald-50 group-hover:text-emerald-600 dark:text-zinc-400 dark:group-hover:bg-emerald-400/10 dark:group-hover:text-emerald-400'"
      >
        <NIcon :size="13"><Check v-if="copied" /><Copy v-else /></NIcon>
        <span>{{ copied ? '已复制' : '复制' }}</span>
      </span>
    </div>
  </div>
</template>
<script setup lang="ts">
// 资源卡片：图标 + 标题 + 两行描述 + 单一访问 chip + 收藏
// 整卡点击复制关键词；收藏为独立按钮
import { ref, computed } from 'vue'
import { NIcon } from 'naive-ui'
import { PhCopy, PhCheck, PhHeart } from '@phosphor-icons/vue'
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

const firstLetter = computed(() => {
  const c = props.keyword.keyword?.charAt(0)
  return c ? c.toUpperCase() : '?'
})

// 分类色仅用于无图标时的首字母底，不铺到标签上造成噪声
const FALLBACK_TONES = [
  'bg-emerald-100 text-emerald-700 dark:bg-emerald-400/15 dark:text-emerald-400',
  'bg-zinc-100 text-zinc-600 dark:bg-zinc-800 dark:text-zinc-300',
  'bg-sky-100 text-sky-700 dark:bg-sky-400/15 dark:text-sky-400',
  'bg-amber-100 text-amber-700 dark:bg-amber-400/15 dark:text-amber-400',
  'bg-violet-100 text-violet-700 dark:bg-violet-400/15 dark:text-violet-400',
  'bg-rose-100 text-rose-700 dark:bg-rose-400/15 dark:text-rose-400',
]

function toneOf(name?: string) {
  if (!name) return FALLBACK_TONES[0]
  let hash = 0
  for (let i = 0; i < name.length; i++) hash = (hash * 31 + name.charCodeAt(i)) | 0
  return FALLBACK_TONES[Math.abs(hash) % FALLBACK_TONES.length]
}

const avatarTone = computed(() => toneOf(props.categoryName || props.keyword.keyword))

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

// 卡片只留一个访问方式 chip，优先「访问/下载/权限」类属性
const accessChip = computed(() => {
  const attrs = parsedAttributes.value
  if (attrs.length === 0) return null
  const hit = attrs.find((a) => /访问|下载|权限|会员|付费|免费/.test(`${a.label}${a.value}`))
  return hit || null
})

const accessTone = computed(() => {
  const v = `${accessChip.value?.label || ''}${accessChip.value?.value || ''}`
  if (/会员|专享|VIP/i.test(v)) {
    return 'border-amber-200/80 bg-amber-50 text-amber-700 dark:border-amber-400/25 dark:bg-amber-400/10 dark:text-amber-400'
  }
  if (/余额|付费|积分|金币|¥|￥|\d+(\.\d+)?\s*元/.test(v)) {
    return 'border-zinc-200 bg-zinc-50 text-zinc-600 dark:border-zinc-700 dark:bg-zinc-800/80 dark:text-zinc-300'
  }
  return 'border-emerald-200/80 bg-emerald-50 text-emerald-700 dark:border-emerald-400/25 dark:bg-emerald-400/10 dark:text-emerald-400'
})

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
    class="resource-card group relative flex h-full cursor-pointer flex-col rounded-2xl border bg-white p-4 text-left focus:outline-none focus-visible:ring-2 focus-visible:ring-emerald-500/60 active:scale-[0.995] dark:bg-zinc-900/80"
    :class="copied
      ? 'border-emerald-500 ring-1 ring-emerald-500/40 dark:border-emerald-400'
      : 'border-zinc-200/90 hover:-translate-y-0.5 hover:border-emerald-400/60 hover:shadow-[0_8px_24px_-12px_rgba(16,185,129,0.25)] dark:border-zinc-800 dark:hover:border-emerald-400/40 dark:hover:shadow-[0_8px_28px_-14px_rgba(52,211,153,0.18)]'"
    title="点击复制关键词"
    @click="handleCopy"
    @keydown="handleKeydown"
  >
    <!-- 收藏 -->
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
      <NIcon :size="16"><PhHeart :weight="favorited ? 'fill' : 'regular'" /></NIcon>
    </button>

    <!-- 主体：图标 + 文案 -->
    <div class="flex items-start gap-3 pr-7">
      <div
        class="resource-card__icon grid size-11 shrink-0 place-items-center overflow-hidden rounded-xl text-sm font-bold group-hover:scale-[1.04]"
        :class="keyword.icon ? 'bg-zinc-100 dark:bg-zinc-800' : avatarTone"
      >
        <img v-if="keyword.icon" :src="keyword.icon" :alt="keyword.keyword" class="size-full object-cover" loading="lazy" />
        <span v-else>{{ firstLetter }}</span>
      </div>
      <div class="min-w-0 flex-1">
        <h3 class="break-words text-[15px] font-semibold leading-snug tracking-tight text-zinc-900 dark:text-zinc-50">
          {{ keyword.keyword }}
        </h3>
        <p class="mt-1 line-clamp-2 text-[13px] leading-relaxed text-zinc-500 dark:text-zinc-400">
          {{ keyword.description || '暂无描述' }}
        </p>
      </div>
    </div>

    <!-- 底栏：访问 chip + 复制反馈 -->
    <div class="mt-3.5 flex items-center justify-between gap-2">
      <span
        v-if="accessChip"
        class="inline-flex max-w-[70%] items-center truncate rounded-md border px-2 py-0.5 text-[11px] font-medium"
        :class="accessTone"
      >
        {{ accessChip.value }}
      </span>
      <span
        v-else-if="categoryName"
        class="inline-flex max-w-[70%] items-center truncate rounded-md border border-zinc-200 px-2 py-0.5 text-[11px] font-medium text-zinc-500 dark:border-zinc-700 dark:text-zinc-400"
      >
        {{ categoryName }}
      </span>
      <span v-else class="flex-1" />

      <span
        class="flex shrink-0 items-center gap-1 rounded-lg px-1.5 py-0.5 text-[11px] font-medium motion-safe:transition-colors"
        :class="copied
          ? 'text-emerald-600 dark:text-emerald-400'
          : 'text-zinc-400 opacity-0 group-hover:opacity-100 group-focus-visible:opacity-100 dark:text-zinc-500'"
      >
        <NIcon :size="12"><PhCheck v-if="copied" /><PhCopy v-else /></NIcon>
        <span>{{ copied ? '已复制' : '复制' }}</span>
      </span>
    </div>
  </div>
</template>

<script setup lang="ts">
// 媒体库可复用管理组件：拖拽上传 / 网络 URL 上传 / 缩略图网格 / 删除 / 翻页
// 用法：
//   1) 作为管理页内联使用：父组件直接渲染 <MediaPicker />
//   2) 作为选择器：父组件监听 @select 事件，拿到的参数是 MediaFile
import { ref, computed, onMounted, onBeforeUnmount } from 'vue'
import { NIcon, NButton, NInput, NPagination, NSkeleton, NEmpty, NPopconfirm } from 'naive-ui'
import { Upload, Link as LinkIcon, Trash, Search } from '@vicons/tabler'
import { contentApi, type MediaFile } from '@/api/content'
import { useToastStore } from '@/stores/toast'

const props = withDefaults(defineProps<{
  /** 选择模式：true 时点击图片会 emit select / enter；管理模式下点击无反应 */
  selectable?: boolean
  /** 已被使用的 url 集合（用于 disable 选中态高亮，可选） */
  usedUrls?: string[]
  pageSize?: number
}>(), {
  selectable: false,
  usedUrls: () => [],
  pageSize: 24,
})

const emit = defineEmits<{
  select: [file: MediaFile]
  enter: [file: MediaFile]
}>()

const toast = useToastStore()

const PAGE_SIZE = computed(() => props.pageSize || 24)

const items = ref<MediaFile[]>([])
const total = ref(0)
const page = ref(1)
const loading = ref(true)
const q = ref('')

const urlInput = ref('')
const urlUploading = ref(false)

const isDragging = ref(false)
const fileInput = ref<HTMLInputElement | null>(null)
const uploadError = ref('')
const uploadHint = ref('')

const pageCount = computed(() => Math.max(1, Math.ceil(total.value / PAGE_SIZE.value)))

onMounted(() => fetchMedia())
onBeforeUnmount(() => { if (dragTimer) clearTimeout(dragTimer) })

async function fetchMedia() {
  loading.value = true
  uploadError.value = ''
  try {
    const data = await contentApi.media.list(page.value, PAGE_SIZE.value, q.value || undefined)
    items.value = data?.items ?? []
    total.value = data?.total ?? 0
  } catch (err) {
    console.error('加载媒体失败:', err)
    items.value = []
    total.value = 0
    toast.error('加载失败，请刷新重试')
  } finally {
    loading.value = false
  }
}

function handleSearch() {
  page.value = 1
  fetchMedia()
}

function handlePageChange(p: number) {
  page.value = p
  fetchMedia()
}

// 本地拖拽上传
function openFilePicker() { fileInput.value?.click() }
function handleFileInput(e: Event) {
  const files = (e.target as HTMLInputElement).files
  if (!files || files.length === 0) return
  void uploadFiles(Array.from(files))
  ;(e.target as HTMLInputElement).value = ''
}

let dragTimer: ReturnType<typeof setTimeout> | null = null

function onDropEnter(e: DragEvent) {
  e.preventDefault()
  if (dragTimer) clearTimeout(dragTimer)
  isDragging.value = true
}
function onDropOver(e: DragEvent) {
  e.preventDefault()
  if (e.dataTransfer) e.dataTransfer.dropEffect = 'copy'
}
function onDropLeave(e: DragEvent) {
  e.preventDefault()
  // 用 timer 避免子元素切换时频繁触发 leave
  if (dragTimer) clearTimeout(dragTimer)
  dragTimer = setTimeout(() => { isDragging.value = false }, 80)
}
function onDrop(e: DragEvent) {
  e.preventDefault()
  isDragging.value = false
  if (dragTimer) clearTimeout(dragTimer)
  const files = e.dataTransfer?.files
  if (!files || files.length === 0) return
  void uploadFiles(Array.from(files))
}

async function uploadFiles(files: File[]) {
  const pattern = /\.(jpe?g|png|gif|webp)$/i
  const filtered = files.filter((f) => pattern.test(f.name) || f.type.startsWith('image/'))
  if (filtered.length === 0) {
    uploadError.value = '仅支持 JPG / PNG / GIF / WebP 图片'
    return
  }
  uploadError.value = ''
  uploadHint.value = `正在上传 ${filtered.length} 张...`
  const fd = new FormData()
  filtered.forEach((f) => fd.append('file', f))
  try {
    const res = await contentApi.media.upload(fd)
    if (res?.items?.length) {
      // 上传后回到第一页，确保刚上传的在视野中
      page.value = 1
      await fetchMedia()
      uploadHint.value = `已上传 ${res.items.length} 张`
      setTimeout(() => { uploadHint.value = '' }, 2000)
    }
  } catch (err: unknown) {
    const e = err as { response?: { data?: { error?: string } } }
    uploadError.value = e?.response?.data?.error || '上传失败'
  }
}

// 网络 URL 上传
async function uploadByUrl() {
  const url = urlInput.value.trim()
  if (!url) return
  if (!/^https?:\/\//i.test(url)) {
    uploadError.value = 'URL 必须以 http(s):// 开头'
    return
  }
  urlUploading.value = true
  uploadError.value = ''
  try {
    await contentApi.media.uploadByURL(url)
    urlInput.value = ''
    page.value = 1
    await fetchMedia()
  } catch (err: unknown) {
    const e = err as { response?: { data?: { error?: string } } }
    uploadError.value = e?.response?.data?.error || '拉取网络图片失败'
  } finally {
    urlUploading.value = false
  }
}

// 删除（二次确认由缩略图上的 NPopconfirm 触发）
const deletingIds = ref<Set<number>>(new Set())
async function handleDelete(file: MediaFile) {
  deletingIds.value.add(file.id)
  try {
    await contentApi.media.delete(file.id)
    await fetchMedia()
  } catch (err: unknown) {
    const errorObj = err as { response?: { data?: { error?: string } } }
    uploadError.value = errorObj?.response?.data?.error || '删除失败'
  } finally {
    deletingIds.value.delete(file.id)
  }
}

// 选中（selectable 时）
function handleClickFile(file: MediaFile) {
  if (!props.selectable) return
  emit('select', file)
}
function handleFileEnter(file: MediaFile) {
  if (!props.selectable) return
  emit('enter', file)
  emit('select', file)
}

const usedSet = computed(() => new Set(props.usedUrls))

function fmtSize(s: number) {
  if (!s) return '-'
  if (s < 1024) return `${s} B`
  if (s < 1024 * 1024) return `${(s / 1024).toFixed(1)} KB`
  return `${(s / 1024 / 1024).toFixed(2)} MB`
}
</script>

<template>
  <div class="space-y-5">
    <!-- 上传区 -->
    <div class="grid gap-3 md:grid-cols-2">
      <!-- 拖拽区 -->
      <div
        class="flex cursor-pointer flex-col items-center justify-center gap-2 rounded-2xl border-2 border-dashed p-6 transition-colors focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-emerald-500/60 dark:focus-visible:ring-emerald-400/60"
        :class="isDragging
          ? 'border-emerald-500 bg-emerald-50/60 dark:bg-emerald-400/10'
          : 'border-zinc-300 hover:border-emerald-400/70 hover:bg-zinc-50/40 dark:border-zinc-700 dark:hover:border-emerald-400/40 dark:hover:bg-zinc-900/40'"
        role="button"
        tabindex="0"
        @click="openFilePicker"
        @keydown.enter.prevent="openFilePicker"
        @dragenter="onDropEnter"
        @dragover="onDropOver"
        @dragleave="onDropLeave"
        @drop="onDrop"
      >
        <NIcon :size="28" class="text-emerald-500 dark:text-emerald-400"><Upload /></NIcon>
        <p class="text-sm font-medium text-zinc-700 dark:text-zinc-200">点击或拖拽图片到此处</p>
        <p class="text-xs text-zinc-500 dark:text-zinc-400">支持 JPG / PNG / GIF / WebP，单个文件最大 5MB</p>
        <input
          ref="fileInput"
          type="file"
          accept="image/jpeg,image/png,image/gif,image/webp"
          multiple
          class="hidden"
          @change="handleFileInput"
        />
      </div>

      <!-- 网络 URL -->
      <div class="flex flex-col justify-between gap-3 rounded-2xl border border-zinc-200 p-5 dark:border-zinc-800">
        <div class="flex items-center gap-2">
          <NIcon :size="16" class="text-emerald-500"><LinkIcon /></NIcon>
          <span class="text-sm font-medium text-zinc-700 dark:text-zinc-200">从网络 URL 导入</span>
        </div>
        <div class="flex flex-col gap-2 sm:flex-row">
          <NInput
            v-model:value="urlInput"
            size="medium"
            placeholder="https://example.com/image.jpg"
            :disabled="urlUploading"
            @keydown.enter.prevent="uploadByUrl"
          />
          <NButton
            type="primary"
            secondary
            :loading="urlUploading"
            :disabled="!urlInput.trim()"
            @click="uploadByUrl"
          >导入</NButton>
        </div>
        <p class="text-xs text-zinc-500 dark:text-zinc-400">后端将拉取并存入媒体库本地存储</p>
      </div>
    </div>

    <!-- 提示条 -->
    <div
      v-if="uploadError || uploadHint"
      class="flex items-center justify-between rounded-xl border px-4 py-2.5 text-sm"
      :class="uploadError
        ? 'border-red-200 bg-red-50 text-red-600 dark:border-red-400/20 dark:bg-red-400/10 dark:text-red-400'
        : 'border-emerald-200 bg-emerald-50 text-emerald-700 dark:border-emerald-400/20 dark:bg-emerald-400/10 dark:text-emerald-400'"
    >
      <span>{{ uploadError || uploadHint }}</span>
      <button v-if="uploadError" type="button" class="rounded opacity-70 hover:opacity-100 focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-emerald-500/60 dark:focus-visible:ring-emerald-400/60" @click="uploadError = ''">关闭</button>
    </div>

    <!-- 搜索 + 计数 -->
    <div class="flex flex-col items-stretch gap-3 sm:flex-row sm:items-center sm:justify-between">
      <div class="flex items-center gap-2">
        <NInput
          v-model:value="q"
          size="small"
          clearable
          placeholder="搜索文件名"
          class="!w-full sm:!w-48"
          @keydown.enter.prevent="handleSearch"
          @clear="handleSearch"
        />
        <NButton size="small" quaternary :disabled="loading" @click="handleSearch">
          <template #icon><NIcon><Search /></NIcon></template>
        </NButton>
      </div>
      <span v-if="!loading" class="font-mono text-xs text-zinc-500 dark:text-zinc-400">共 {{ total }} 个</span>
    </div>

    <!-- 加载骨架 -->
    <div v-if="loading" class="grid grid-cols-2 gap-3 sm:grid-cols-3 lg:grid-cols-4 xl:grid-cols-6">
      <div
        v-for="i in 12"
        :key="i"
        class="overflow-hidden rounded-xl border border-zinc-200 bg-white dark:border-zinc-800 dark:bg-zinc-900"
      >
        <NSkeleton height="120px" :sharp="false" />
        <div class="space-y-1 p-2">
          <NSkeleton text width="70%" />
          <NSkeleton text width="40%" />
        </div>
      </div>
    </div>

    <!-- 空状态 -->
    <div
      v-else-if="items.length === 0"
      class="rounded-2xl border border-dashed border-zinc-300 py-14 dark:border-zinc-700"
    >
      <NEmpty description="媒体库为空">
        <template #extra>
          <span class="text-xs text-zinc-500 dark:text-zinc-400">拖拽图片到上面区域，或填入网络图片 URL 开始添加</span>
        </template>
      </NEmpty>
    </div>

    <!-- 缩略图网格 -->
    <div v-else class="grid grid-cols-2 gap-3 sm:grid-cols-3 lg:grid-cols-4 xl:grid-cols-6">
      <div
        v-for="file in items"
        :key="file.id"
        class="group relative flex cursor-pointer flex-col overflow-hidden rounded-xl border bg-white transition-all hover:-translate-y-0.5 hover:shadow-md focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-emerald-500/60 dark:bg-zinc-900 dark:focus-visible:ring-emerald-400/60"
        :class="props.selectable && usedSet.has(file.url)
          ? 'border-emerald-400 ring-1 ring-emerald-400/40 dark:border-emerald-400'
          : 'border-zinc-200 hover:border-emerald-400/60 dark:border-zinc-800 dark:hover:border-emerald-400/40'"
        role="button"
        tabindex="0"
        :title="file.origin_name || file.filename"
        @click="handleClickFile(file)"
        @keydown.enter.prevent="handleFileEnter(file)"
        @dblclick="handleFileEnter(file)"
      >
        <!-- 缩略图 -->
        <div class="relative aspect-square w-full overflow-hidden bg-zinc-50 dark:bg-zinc-800">
          <img
            :src="file.url"
            :alt="file.origin_name || file.filename"
            class="size-full object-cover transition-transform group-hover:scale-[1.04]"
            loading="lazy"
          />
          <!-- 删除浮层（带二次确认，删除后不可恢复） -->
          <NPopconfirm @positive-click="handleDelete(file)">
            <template #trigger>
              <button
                type="button"
                class="absolute right-1.5 top-1.5 grid size-7 place-items-center rounded-full bg-white/90 text-zinc-500 opacity-0 shadow-sm transition-all hover:text-red-500 focus-visible:opacity-100 focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-emerald-500/60 group-hover:opacity-100 dark:bg-zinc-900/80 dark:text-zinc-400 dark:focus-visible:ring-emerald-400/60"
                title="删除"
                :aria-label="`删除 ${file.filename}`"
                :disabled="deletingIds.has(file.id)"
                @click.stop
              >
                <NIcon :size="14"><Trash /></NIcon>
              </button>
            </template>
            确认删除该图片？已引用它的内容将无法显示。
          </NPopconfirm>
          <!-- 来源标记 -->
          <span
            v-if="file.source === 'remote'"
            class="absolute bottom-1.5 left-1.5 rounded-md bg-zinc-900/70 px-1.5 py-0.5 text-[10px] font-medium text-zinc-100"
          >网络</span>
        </div>
        <!-- 元数据 -->
        <div class="flex flex-col gap-0.5 p-2">
          <p class="truncate text-xs font-medium text-zinc-700 dark:text-zinc-200">{{ file.origin_name || file.filename }}</p>
          <div class="flex items-center gap-2 text-[10px] text-zinc-500 dark:text-zinc-400">
            <span class="font-mono">{{ fmtSize(file.size) }}</span>
            <span v-if="file.width && file.height" class="font-mono">{{ file.width }}×{{ file.height }}</span>
          </div>
        </div>
      </div>
    </div>

    <!-- 分页 -->
    <div v-if="!loading && total > PAGE_SIZE" class="flex justify-center">
      <NPagination
        :page="page"
        :page-count="pageCount"
        :item-count="total"
        show-quick-jumper
        @update:page="handlePageChange"
      />
    </div>
  </div>
</template>
